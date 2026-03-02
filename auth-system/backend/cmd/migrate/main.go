// cmd/migrate/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"tigersoft/auth-system/internal/config"
	pginfra "tigersoft/auth-system/internal/infrastructure/postgres"
	"tigersoft/auth-system/internal/infrastructure/migrations"
	"tigersoft/auth-system/internal/repository/postgres"
)

func main() {
	var (
		scope     = flag.String("scope", "all", "Migration scope: global | tenant | all")
		tenant    = flag.String("tenant", "", "Tenant slug (required when scope=tenant)")
		direction = flag.String("direction", "up", "Migration direction: up | down")
		steps     = flag.Int("steps", 0, "Number of steps for down migration (0 = all)")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	pool, err := pginfra.NewPool(ctx, cfg.Database)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	tenantRepo := postgres.NewPostgresTenantRepo(pool)

	runner := migrations.NewMigrationRunner(
		pool,
		cfg.Database.URL,
		"file://migrations/global",
		"file://migrations/tenant",
		tenantRepo,
	)

	switch *scope {
	case "global":
		slog.Info("running global migrations", "direction", *direction)
		if err := runner.RunGlobal(ctx, *direction); err != nil {
			slog.Error("global migration failed", "error", err)
			os.Exit(1)
		}
		slog.Info("global migrations complete")

	case "tenant":
		if *tenant == "" {
			fmt.Fprintln(os.Stderr, "error: -tenant flag is required when -scope=tenant")
			os.Exit(1)
		}
		slog.Info("running tenant migration", "tenant", *tenant, "direction", *direction)
		if err := runner.RunTenant(ctx, *tenant, *direction, *steps); err != nil {
			slog.Error("tenant migration failed", "tenant", *tenant, "error", err)
			os.Exit(1)
		}
		slog.Info("tenant migration complete", "tenant", *tenant)

	case "all":
		slog.Info("running global migrations")
		if err := runner.RunGlobal(ctx, "up"); err != nil {
			slog.Error("global migration failed", "error", err)
			os.Exit(1)
		}
		slog.Info("running all tenant migrations")
		if err := runner.RunAllTenants(ctx); err != nil {
			slog.Error("one or more tenant migrations failed", "error", err)
			os.Exit(1)
		}
		slog.Info("all migrations complete")

	default:
		fmt.Fprintf(os.Stderr, "error: unknown scope %q — use global | tenant | all\n", *scope)
		os.Exit(1)
	}
}
