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
		scope        = flag.String("scope", "all", "Migration scope: global | tenant | all")
		tenant       = flag.String("tenant", "", "Tenant slug (required when scope=tenant)")
		direction    = flag.String("direction", "up", "Migration direction: up | down")
		steps        = flag.Int("steps", 0, "Number of steps for down migration (0 = all)")
		seedPlatform = flag.Bool("seed-platform", false, "Bootstrap platform tenant and super_admin user (reads PLATFORM_ADMIN_EMAIL and PLATFORM_ADMIN_PASSWORD env vars)")
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

		if *seedPlatform {
			if err := bootstrapPlatform(ctx, runner); err != nil {
				slog.Error("platform bootstrap failed", "error", err)
				os.Exit(1)
			}
		}

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
		if *seedPlatform {
			if err := bootstrapPlatform(ctx, runner); err != nil {
				slog.Error("platform bootstrap failed", "error", err)
				os.Exit(1)
			}
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

// bootstrapPlatform reads credentials from the environment and calls BootstrapPlatform.
func bootstrapPlatform(ctx context.Context, runner *migrations.MigrationRunner) error {
	email := os.Getenv("PLATFORM_ADMIN_EMAIL")
	password := os.Getenv("PLATFORM_ADMIN_PASSWORD")
	if email == "" || password == "" {
		return fmt.Errorf("PLATFORM_ADMIN_EMAIL and PLATFORM_ADMIN_PASSWORD must be set when -seed-platform=true")
	}
	return runner.BootstrapPlatform(ctx, email, password)
}
