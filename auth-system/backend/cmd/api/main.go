// cmd/api/main.go
package main

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tigersoft/auth-system/internal/config"
	"tigersoft/auth-system/internal/handler"
	emailinfra "tigersoft/auth-system/internal/infrastructure/email"
	pginfra "tigersoft/auth-system/internal/infrastructure/postgres"
	redisinfra "tigersoft/auth-system/internal/infrastructure/redis"
	"tigersoft/auth-system/internal/infrastructure/vault"
	"tigersoft/auth-system/internal/middleware"
	pgRepo "tigersoft/auth-system/internal/repository/postgres"
	redisRepo "tigersoft/auth-system/internal/repository/redis"
	"tigersoft/auth-system/internal/router"
	"tigersoft/auth-system/internal/service"
	"tigersoft/auth-system/pkg/jwtutil"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.MessageKey {
				a.Key = "message"
			}
			return a
		},
	})))

	if err := run(); err != nil {
		slog.Error("fatal startup error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := pginfra.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	redisClient, err := redisinfra.NewClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer redisClient.Close()

	// -- Signing key: Vault (dev/self-hosted) or env var (Fly Secrets / CI) ---
	signingKey, kid, err := resolveSigningKey(ctx, cfg)
	if err != nil {
		return fmt.Errorf("resolve signing key: %w", err)
	}

	keyStore := jwtutil.NewKeyStore(cfg.JWT.Issuer, cfg.JWT.Audience, kid, signingKey)

	// Key rotation watcher only runs when Vault is configured.
	if cfg.Vault.Addr != "" {
		vaultClient, err := vault.NewClient(cfg.Vault)
		if err != nil {
			return fmt.Errorf("connect to vault for rotation watcher: %w", err)
		}
		keyRotationWatcher := vault.NewKeyRotationWatcher(vaultClient, keyStore, cfg.Vault.KeyRotationPollInterval)
		go keyRotationWatcher.Watch(ctx)
		slog.Info("vault key rotation watcher started")
	}

	emailChannel := make(chan service.EmailTask, cfg.Email.ChannelBufferSize)
	emailClient := emailinfra.NewClient(cfg.Email)
	emailWorker := emailinfra.NewWorker(emailClient, cfg.Email)
	go emailWorker.Start(ctx, emailChannel)

	slog.Info("email worker started", "workers", cfg.Email.WorkerConcurrency)

	// -- Repositories ----------------------------------------------------------
	userRepo := pgRepo.NewPostgresUserRepo(pool)
	sessionRepo := pgRepo.NewPostgresSessionRepo(pool)
	tokenRepo := pgRepo.NewPostgresTokenRepo(pool)
	auditRepo := pgRepo.NewPostgresAuditRepo(pool)
	tenantRepo := pgRepo.NewPostgresTenantRepo(pool)
	roleRepo := pgRepo.NewPostgresRoleRepo(pool)
	credRepo := pgRepo.NewPostgresCredentialRepo(pool)
	oauthClientRepo := pgRepo.NewPostgresOAuthClientRepo(pool)
	oauthCodeRepo := pgRepo.NewPostgresAuthCodeRepo(pool)
	socialAccountRepo := pgRepo.NewPostgresSocialAccountRepo(pool)
	mfaRepo := pgRepo.NewPostgresMFARepo(pool)

	rateLimiter := redisRepo.NewRedisRateLimiter(redisClient)

	// -- Services --------------------------------------------------------------

	// MFAService now requires a RateLimiter for TOTP brute-force protection.
	mfaSvc := service.NewMFAService(userRepo, mfaRepo, auditRepo, rateLimiter, cfg.JWT.Issuer)

	// ProfileService now requires mfaRepo, socialAccountRepo, and codeRepo for
	// GDPR self-erasure and OAuth code invalidation on password change.
	profileSvc := service.NewProfileService(
		userRepo, sessionRepo, mfaRepo, socialAccountRepo, oauthCodeRepo,
		auditRepo, emailChannel, cfg.Email.VerificationTokenTTL,
	)

	authSvcCfg := service.AuthServiceConfig{
		AccessTokenTTL:         cfg.JWT.AccessTokenTTL,
		SessionDefaultTTL:      cfg.Session.DefaultTTL,
		VerificationTokenTTL:   cfg.Email.VerificationTokenTTL,
		LockoutThreshold:       cfg.RateLimit.LockoutThreshold,
		LockoutDurationSeconds: cfg.RateLimit.LockoutDurationSeconds,
	}

	authSvc := service.NewAuthService(
		userRepo, sessionRepo, tokenRepo, auditRepo, tenantRepo, roleRepo,
		keyStore, emailChannel, mfaSvc, authSvcCfg,
	)
	emailVerificationSvc := service.NewEmailVerificationService(
		userRepo, tokenRepo, auditRepo, emailChannel, cfg.Email.VerificationTokenTTL,
	)
	passwordSvc := service.NewPasswordService(
		userRepo, sessionRepo, tokenRepo, auditRepo, emailChannel, cfg.Email.PasswordResetTokenTTL,
	)
	sessionSvc := service.NewSessionService(
		userRepo, sessionRepo, auditRepo, roleRepo, keyStore, cfg.JWT.AccessTokenTTL,
	)
	tenantSvc := service.NewTenantService(
		tenantRepo, credRepo, pool, cfg.Database.URL,
		"file://migrations/tenant", emailChannel,
	)
	rbacSvc := service.NewRBACService(roleRepo, userRepo, auditRepo)

	// AdminService now requires mfaRepo, socialAccountRepo, and codeRepo to
	// perform full GDPR erasure (EraseUser) instead of simple soft-delete.
	adminSvc := service.NewAdminService(
		userRepo, sessionRepo, roleRepo, auditRepo,
		mfaRepo, socialAccountRepo, oauthCodeRepo, emailChannel,
	)
	adminSvc = service.WithTokenRepo(adminSvc, tokenRepo)

	auditSvc := service.NewAuditService(auditRepo)
	jwtSvc := service.NewJWTService(keyStore)
	oauthSvc := service.NewOAuthService(
		oauthClientRepo, oauthCodeRepo, sessionRepo, auditRepo,
		keyStore, cfg.JWT.AccessTokenTTL, cfg.Session.DefaultTTL,
	)
	googleSvc := service.NewGoogleService(
		userRepo, sessionRepo, socialAccountRepo, auditRepo, roleRepo,
		redisClient,
		keyStore,
		cfg.OAuth, cfg.JWT.AccessTokenTTL, cfg.Session.DefaultTTL,
	)

	tenantCache := middleware.NewInMemoryTenantCache(tenantRepo, cfg.Tenant.CacheTTL)

	// -- Handlers --------------------------------------------------------------
	deps := router.Dependencies{
		Config:           cfg,
		AuthHandler:      handler.NewAuthHandler(authSvc),
		EmailHandler:     handler.NewEmailHandler(emailVerificationSvc),
		PasswordHandler:  handler.NewPasswordHandler(passwordSvc),
		SessionHandler:   handler.NewSessionHandler(sessionSvc),
		UserHandler:      handler.NewUserHandler(profileSvc, mfaSvc),
		MFAHandler:       handler.NewMFAHandler(mfaSvc),
		AdminHandler:     handler.NewAdminHandler(adminSvc),
		TenantHandler:    handler.NewTenantHandler(tenantSvc, adminSvc),
		RoleHandler:      handler.NewRoleHandler(rbacSvc),
		AuditHandler:     handler.NewAuditHandler(auditSvc),
		WellKnownHandler: handler.NewWellKnownHandler(jwtSvc),
		OAuthHandler:     handler.NewOAuthHandler(keyStore, oauthSvc),
		GoogleHandler:    handler.NewGoogleHandler(googleSvc, tenantSvc),
		// Sprint 8: dependency-aware health check.
		HealthHandler: handler.NewHealthHandler(pool, redisClient),
		JWTKeyStore:   keyStore,
		TenantCache:   tenantCache,
		RateLimiter:   rateLimiter,
	}

	ginRouter := router.New(deps)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      ginRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr, "env", cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	slog.Info("shutting down server gracefully", "timeout", cfg.Server.GracefulShutdownTimeout)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.GracefulShutdownTimeout)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	cancel()

	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer drainCancel()
	emailWorker.Drain(drainCtx)

	slog.Info("server shutdown complete")
	return nil
}

// resolveSigningKey loads the RSA signing key from Vault (when VAULT_ADDR is set)
// or from environment variables (JWT_PRIVATE_KEY_PATH or JWT_RSA_PRIVATE_KEY_PEM)
// for deployments that do not use HashiCorp Vault (e.g. Fly.io with Fly Secrets).
func resolveSigningKey(ctx context.Context, cfg *config.Config) (*rsa.PrivateKey, string, error) {
	if cfg.Vault.Addr != "" {
		vaultClient, err := vault.NewClient(cfg.Vault)
		if err != nil {
			return nil, "", fmt.Errorf("connect to vault: %w", err)
		}
		key, kid, err := vaultClient.LoadCurrentSigningKey(ctx)
		if err != nil {
			return nil, "", err
		}
		slog.Info("loaded signing key from vault", "kid", kid)
		return key, kid, nil
	}

	// Fly Secrets / env-var mode — no Vault dependency.
	// Priority 1: file path (existing dev behaviour, unchanged).
	if path := os.Getenv("JWT_PRIVATE_KEY_PATH"); path != "" {
		pemBytes, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read JWT key file: %w", err)
		}
		key, err := parseRSAPrivateKey(pemBytes)
		if err != nil {
			return nil, "", fmt.Errorf("parse JWT key file: %w", err)
		}
		kid := os.Getenv("JWT_KEY_ID")
		if kid == "" {
			kid = "key-dev-local"
		}
		slog.Info("loaded signing key from file", "kid", kid)
		return key, kid, nil
	}

	// Priority 2: PEM content stored directly as env var (Fly Secrets).
	pemContent := os.Getenv("JWT_RSA_PRIVATE_KEY_PEM")
	if pemContent == "" {
		return nil, "", fmt.Errorf(
			"no JWT signing key configured: set VAULT_ADDR, JWT_PRIVATE_KEY_PATH, or JWT_RSA_PRIVATE_KEY_PEM",
		)
	}
	key, err := parseRSAPrivateKey([]byte(pemContent))
	if err != nil {
		return nil, "", fmt.Errorf("parse JWT_RSA_PRIVATE_KEY_PEM: %w", err)
	}
	kid := os.Getenv("JWT_KEY_ID")
	if kid == "" {
		kid = "key-prod-1"
	}
	slog.Info("loaded signing key from environment", "kid", kid)
	return key, kid, nil
}

// parseRSAPrivateKey decodes a PEM block and parses an RSA private key (PKCS8 or PKCS1).
func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("PKCS8 key is not RSA")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
