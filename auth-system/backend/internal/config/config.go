// internal/config/config.go
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	Vault     VaultConfig
	JWT       JWTConfig
	Session   SessionConfig
	Email     EmailConfig
	OAuth     OAuthConfig
	RateLimit RateLimitConfig
	Tenant    TenantConfig
	CORS      CORSConfig
}

type ServerConfig struct {
	Port                    string
	Env                     string
	LogLevel                string
	APIVersion              string
	BaseURL                 string
	GracefulShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnIdleTime   time.Duration
	MaxConnLifetime   time.Duration
	ConnectTimeout    time.Duration
	HealthCheckPeriod time.Duration
}

type RedisConfig struct {
	URL          string
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
}

type VaultConfig struct {
	Addr                    string
	Token                   string
	RoleID                  string
	SecretID                string
	Mount                   string
	KeyRotationPollInterval time.Duration
}

type JWTConfig struct {
	PrivateKeyPath string
	PublicKeyPath  string
	KeyID          string
	AccessTokenTTL time.Duration
	Issuer         string
	Audience       []string
}

type SessionConfig struct {
	DefaultTTL time.Duration
}

type EmailConfig struct {
	ResendAPIKey          string
	From                  string
	FromName              string
	WorkerConcurrency     int
	ChannelBufferSize     int
	MaxRetries            int
	RetryBackoffBase      time.Duration
	SMTPHost              string
	SMTPPort              int
	VerificationTokenTTL  time.Duration
	PasswordResetTokenTTL time.Duration
}

type OAuthConfig struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleDiscoveryURL string
	StateTokenTTL      time.Duration
}

type RateLimitConfig struct {
	LoginIPPerMinute          int
	LoginUserPerMinute        int
	RegisterIPPerMinute       int
	ForgotPasswordIPPerMinute int
	ForgotPasswordUserPerHour int
	TokenRefreshIPPerMinute   int
	TokenRefreshUserPerMinute int
	VerifyEmailIPPerMinute    int
	LockoutThreshold          int
	LockoutDurationSeconds    int
}

type TenantConfig struct {
	CacheTTL time.Duration
}

type CORSConfig struct {
	AllowedOrigins []string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:                    getEnvDefault("PORT", "8080"),
			Env:                     getEnvDefault("ENV", "development"),
			LogLevel:                getEnvDefault("LOG_LEVEL", "info"),
			APIVersion:              getEnvDefault("API_VERSION", "v1"),
			BaseURL:                 mustGetEnv("AUTH_SERVICE_BASE_URL"),
			GracefulShutdownTimeout: time.Duration(mustGetEnvInt("GRACEFUL_SHUTDOWN_TIMEOUT_SECONDS", 30)) * time.Second,
		},
		Database: DatabaseConfig{
			URL:               mustGetEnv("DATABASE_URL"),
			MaxConns:          int32(mustGetEnvInt("DB_MAX_CONNS", 20)),
			MinConns:          int32(mustGetEnvInt("DB_MIN_CONNS", 5)),
			MaxConnIdleTime:   time.Duration(mustGetEnvInt("DB_MAX_CONN_IDLE_TIME_MINUTES", 10)) * time.Minute,
			MaxConnLifetime:   time.Duration(mustGetEnvInt("DB_MAX_CONN_LIFETIME_MINUTES", 60)) * time.Minute,
			ConnectTimeout:    time.Duration(mustGetEnvInt("DB_CONNECT_TIMEOUT_SECONDS", 10)) * time.Second,
			HealthCheckPeriod: time.Duration(mustGetEnvInt("DB_HEALTH_CHECK_PERIOD_SECONDS", 60)) * time.Second,
		},
		Redis: RedisConfig{
			URL:          mustGetEnv("REDIS_URL"),
			MaxRetries:   mustGetEnvInt("REDIS_MAX_RETRIES", 3),
			DialTimeout:  time.Duration(mustGetEnvInt("REDIS_DIAL_TIMEOUT_SECONDS", 5)) * time.Second,
			ReadTimeout:  time.Duration(mustGetEnvInt("REDIS_READ_TIMEOUT_SECONDS", 3)) * time.Second,
			WriteTimeout: time.Duration(mustGetEnvInt("REDIS_WRITE_TIMEOUT_SECONDS", 3)) * time.Second,
			PoolSize:     mustGetEnvInt("REDIS_POOL_SIZE", 20),
		},
		Vault: VaultConfig{
			Addr:                    mustGetEnv("VAULT_ADDR"),
			Token:                   os.Getenv("VAULT_TOKEN"),
			RoleID:                  os.Getenv("VAULT_ROLE_ID"),
			SecretID:                os.Getenv("VAULT_SECRET_ID"),
			Mount:                   getEnvDefault("VAULT_MOUNT", "secret"),
			KeyRotationPollInterval: time.Duration(mustGetEnvInt("VAULT_KEY_ROTATION_POLL_INTERVAL_SECONDS", 300)) * time.Second,
		},
		JWT: JWTConfig{
			PrivateKeyPath: os.Getenv("JWT_PRIVATE_KEY_PATH"),
			PublicKeyPath:  os.Getenv("JWT_PUBLIC_KEY_PATH"),
			KeyID:          mustGetEnv("JWT_KEY_ID"),
			AccessTokenTTL: time.Duration(mustGetEnvInt("JWT_ACCESS_TOKEN_TTL_SECONDS", 900)) * time.Second,
			Issuer:         mustGetEnv("JWT_ISSUER"),
			Audience:       strings.Split(mustGetEnv("JWT_AUDIENCE"), ","),
		},
		Session: SessionConfig{
			DefaultTTL: time.Duration(mustGetEnvInt("SESSION_DEFAULT_TTL_SECONDS", 86400)) * time.Second,
		},
		Email: EmailConfig{
			ResendAPIKey:          os.Getenv("RESEND_API_KEY"),
			From:                  mustGetEnv("EMAIL_FROM"),
			FromName:              getEnvDefault("EMAIL_FROM_NAME", "Auth System"),
			WorkerConcurrency:     mustGetEnvInt("EMAIL_WORKER_CONCURRENCY", 4),
			ChannelBufferSize:     mustGetEnvInt("EMAIL_CHANNEL_BUFFER_SIZE", 100),
			MaxRetries:            mustGetEnvInt("EMAIL_MAX_RETRIES", 3),
			RetryBackoffBase:      time.Duration(mustGetEnvInt("EMAIL_RETRY_BACKOFF_BASE_SECONDS", 2)) * time.Second,
			SMTPHost:              getEnvDefault("SMTP_HOST", "localhost"),
			SMTPPort:              mustGetEnvInt("SMTP_PORT", 1025),
			VerificationTokenTTL:  time.Duration(mustGetEnvInt("EMAIL_VERIFICATION_TOKEN_TTL_HOURS", 24)) * time.Hour,
			PasswordResetTokenTTL: time.Duration(mustGetEnvInt("PASSWORD_RESET_TOKEN_TTL_HOURS", 1)) * time.Hour,
		},
		OAuth: OAuthConfig{
			GoogleClientID:     os.Getenv("OAUTH_GOOGLE_CLIENT_ID"),
			GoogleClientSecret: os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET"),
			GoogleDiscoveryURL: getEnvDefault("OAUTH_GOOGLE_DISCOVERY_URL", "https://accounts.google.com/.well-known/openid-configuration"),
			StateTokenTTL:      time.Duration(mustGetEnvInt("OAUTH_STATE_TTL_SECONDS", 600)) * time.Second,
		},
		RateLimit: RateLimitConfig{
			LoginIPPerMinute:          mustGetEnvInt("RATE_LIMIT_LOGIN_IP_PER_MINUTE", 20),
			LoginUserPerMinute:        mustGetEnvInt("RATE_LIMIT_LOGIN_USER_PER_MINUTE", 5),
			RegisterIPPerMinute:       mustGetEnvInt("RATE_LIMIT_REGISTER_IP_PER_MINUTE", 10),
			ForgotPasswordIPPerMinute: mustGetEnvInt("RATE_LIMIT_FORGOT_PASSWORD_IP_PER_MINUTE", 5),
			ForgotPasswordUserPerHour: mustGetEnvInt("RATE_LIMIT_FORGOT_PASSWORD_USER_PER_HOUR", 3),
			TokenRefreshIPPerMinute:   mustGetEnvInt("RATE_LIMIT_TOKEN_REFRESH_IP_PER_MINUTE", 60),
			TokenRefreshUserPerMinute: mustGetEnvInt("RATE_LIMIT_TOKEN_REFRESH_USER_PER_MINUTE", 30),
			VerifyEmailIPPerMinute:    mustGetEnvInt("RATE_LIMIT_VERIFY_EMAIL_IP_PER_MINUTE", 10),
			LockoutThreshold:          mustGetEnvInt("LOCKOUT_THRESHOLD", 5),
			LockoutDurationSeconds:    mustGetEnvInt("LOCKOUT_DURATION_SECONDS", 900),
		},
		Tenant: TenantConfig{
			CacheTTL: time.Duration(mustGetEnvInt("TENANT_CACHE_TTL_SECONDS", 60)) * time.Second,
		},
		CORS: CORSConfig{
			AllowedOrigins: strings.Split(getEnvDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000"), ","),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	slog.Info("config loaded", "env", cfg.Server.Env, "port", cfg.Server.Port)
	return cfg, nil
}

func (c *Config) validate() error {
	if c.JWT.AccessTokenTTL > 900*time.Second {
		return fmt.Errorf("JWT_ACCESS_TOKEN_TTL_SECONDS must not exceed 900 (SEC-03 compliance)")
	}

	isProd := c.Server.Env == "production"

	if isProd && c.Vault.Token != "" && c.Vault.RoleID == "" {
		return fmt.Errorf("production: VAULT_TOKEN (static) must not be used — configure VAULT_ROLE_ID + VAULT_SECRET_ID instead")
	}

	if c.Database.MaxConns < c.Database.MinConns {
		return fmt.Errorf("DB_MAX_CONNS (%d) must be >= DB_MIN_CONNS (%d)", c.Database.MaxConns, c.Database.MinConns)
	}

	if c.Email.WorkerConcurrency < 1 {
		return fmt.Errorf("EMAIL_WORKER_CONCURRENCY must be >= 1")
	}

	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return val
}

func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func mustGetEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		panic(fmt.Sprintf("environment variable %q must be an integer, got: %q", key, val))
	}
	return n
}
