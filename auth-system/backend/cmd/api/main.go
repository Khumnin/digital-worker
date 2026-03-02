// cmd/api/main.go
package main

import (
	"context"
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

	vaultClient, err := vault.NewClient(cfg.Vault)
	if err != nil {
		return fmt.Errorf("connect to vault: %w", err)
	}

	signingKey, kid, err := vaultClient.LoadCurrentSigningKey(ctx)
	if err != nil {
		return fmt.Errorf("load signing key from vault: %w", err)
	}

	keyStore := jwtutil.NewKeyStore(cfg.JWT.Issuer, cfg.JWT.Audience, kid, signingKey)

	keyRotationWatcher := vault.NewKeyRotationWatcher(vaultClient, keyStore, cfg.Vault.KeyRotationPollInterval)
	go keyRotationWatcher.Watch(ctx)

	slog.Info("connected to vault", "kid", kid)

	emailChannel := make(chan service.EmailTask, cfg.Email.ChannelBufferSize)
	emailClient := emailinfra.NewClient(cfg.Email)
	emailWorker := emailinfra.NewWorker(emailClient, cfg.Email)
	go emailWorker.Start(ctx, emailChannel)

	slog.Info("email worker started", "workers", cfg.Email.WorkerConcurrency)

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

	rateLimiter := redisRepo.NewRedisRateLimiter(redisClient)

	authSvcCfg := service.AuthServiceConfig{
		AccessTokenTTL:         cfg.JWT.AccessTokenTTL,
		SessionDefaultTTL:      cfg.Session.DefaultTTL,
		VerificationTokenTTL:   cfg.Email.VerificationTokenTTL,
		LockoutThreshold:       cfg.RateLimit.LockoutThreshold,
		LockoutDurationSeconds: cfg.RateLimit.LockoutDurationSeconds,
	}

	authSvc := service.NewAuthService(
		userRepo, sessionRepo, tokenRepo, auditRepo, tenantRepo, roleRepo,
		keyStore, emailChannel, authSvcCfg,
	)
	emailVerificationSvc := service.NewEmailVerificationService(
		userRepo, tokenRepo, auditRepo, emailChannel, cfg.Email.VerificationTokenTTL,
	)
	passwordSvc := service.NewPasswordService(
		userRepo, sessionRepo, tokenRepo, auditRepo, emailChannel, cfg.Email.PasswordResetTokenTTL,
	)
	sessionSvc := service.NewSessionService(
		userRepo, sessionRepo, auditRepo, keyStore, cfg.JWT.AccessTokenTTL,
	)
	tenantSvc := service.NewTenantService(
		tenantRepo, credRepo, pool, cfg.Database.URL,
		"file://migrations/tenant", emailChannel,
	)
	rbacSvc := service.NewRBACService(roleRepo, userRepo, auditRepo)
	adminSvc := service.NewAdminService(userRepo, sessionRepo, roleRepo, auditRepo, emailChannel)
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

	deps := router.Dependencies{
		Config:           cfg,
		AuthHandler:      handler.NewAuthHandler(authSvc),
		EmailHandler:     handler.NewEmailHandler(emailVerificationSvc),
		PasswordHandler:  handler.NewPasswordHandler(passwordSvc),
		SessionHandler:   handler.NewSessionHandler(sessionSvc),
		UserHandler:      handler.NewUserHandler(authSvc),
		AdminHandler:     handler.NewAdminHandler(adminSvc),
		TenantHandler:    handler.NewTenantHandler(tenantSvc),
		RoleHandler:      handler.NewRoleHandler(rbacSvc),
		AuditHandler:     handler.NewAuditHandler(auditSvc),
		WellKnownHandler: handler.NewWellKnownHandler(jwtSvc),
		OAuthHandler:     handler.NewOAuthHandler(keyStore, oauthSvc),
		GoogleHandler:    handler.NewGoogleHandler(googleSvc, tenantSvc),
		JWTKeyStore:      keyStore,
		TenantCache:      tenantCache,
		RateLimiter:      rateLimiter,
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
