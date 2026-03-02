// internal/infrastructure/postgres/db.go
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tigersoft/auth-system/internal/config"
)

func NewPool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	const maxAttempts = 5
	var pool *pgxpool.Pool

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
		if err != nil {
			slog.Warn("postgres connect attempt failed",
				"attempt", attempt,
				"max", maxAttempts,
				"error", err,
			)
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
			continue
		}

		if pingErr := pool.Ping(ctx); pingErr != nil {
			pool.Close()
			pool = nil
			slog.Warn("postgres ping failed",
				"attempt", attempt,
				"max", maxAttempts,
				"error", pingErr,
			)
			if attempt < maxAttempts {
				time.Sleep(time.Duration(attempt*2) * time.Second)
			}
			continue
		}

		slog.Info("connected to postgres",
			"max_conns", cfg.MaxConns,
			"attempt", attempt,
		)
		return pool, nil
	}

	return nil, fmt.Errorf("failed to connect to postgres after %d attempts: %w", maxAttempts, err)
}

func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire: %w", err)
	}
	defer conn.Release()

	if err := conn.Conn().Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	return nil
}
