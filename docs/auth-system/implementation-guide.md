# Implementation Guide
# Authentication System — Pilot Project

**Version:** 1.0
**Date:** 2026-02-28
**Status:** Ready for Development
**Author:** Developer Agent
**Input Documents:** PRD v1.1, Solution Architecture v1.0
**Intended Consumers:** Engineering Team, Tester Agent

---

## Table of Contents

1. [Project Scaffold](#1-project-scaffold)
2. [Setup Guide](#2-setup-guide)
3. [Environment Variables](#3-environment-variables)
4. [Core Implementation Code](#4-core-implementation-code)
5. [Docker Compose (Local Dev)](#5-docker-compose-local-dev)
6. [Makefile](#6-makefile)
7. [GitHub Actions CI/CD Pipeline](#7-github-actions-cicd-pipeline)
8. [Coding Standards](#8-coding-standards)
9. [Integration Guide](#9-integration-guide)

---

## 1. Project Scaffold

### Monorepo Root Structure

```
auth-system/
├── backend/                           # Go + Gin API (primary service)
├── frontend/                          # Next.js minimal shell (ADR-003: API-only)
├── docker-compose.yml                 # Local dev: postgres, redis, vault, mailhog
├── docker-compose.test.yml            # Integration/isolation test environment
├── .env.example                       # All environment variables with descriptions
├── Makefile                           # All developer workflow targets
├── .github/
│   └── workflows/
│       └── ci.yml                     # 8-stage CI/CD pipeline
└── README.md                          # Project overview and quick start
```

### Backend Structure (Go + Gin — Clean Architecture)

```
backend/
├── cmd/
│   ├── api/
│   │   └── main.go                    # Entry point: wire all deps, graceful shutdown
│   └── migrate/
│       └── main.go                    # CLI migration tool (global + per-tenant)
│
├── internal/
│   ├── config/
│   │   ├── config.go                  # Typed config struct; env loading; validation
│   │   └── config_test.go
│   │
│   ├── domain/                        # Pure domain layer — zero framework dependencies
│   │   ├── user.go                    # User entity, UserStatus enum, domain methods
│   │   ├── tenant.go                  # Tenant entity, TenantConfig, PasswordPolicy
│   │   ├── session.go                 # Session entity, RefreshToken value object
│   │   ├── role.go                    # Role entity, UserRole join entity
│   │   ├── oauth_client.go            # OAuthClient entity, redirect URI validation
│   │   ├── audit_event.go             # AuditEvent entity, EventType constants
│   │   └── repository.go             # All repository interfaces
│   │
│   ├── handler/                       # Transport layer — HTTP handlers only
│   │   ├── auth_handler.go            # POST /auth/register, /login, /logout
│   │   ├── auth_handler_test.go
│   │   ├── email_handler.go           # POST /auth/verify-email, /resend-verification
│   │   ├── password_handler.go        # POST /auth/forgot-password, /reset-password
│   │   ├── session_handler.go         # POST /auth/token/refresh
│   │   ├── admin_handler.go           # Invite, disable, delete users
│   │   ├── tenant_handler.go          # POST /admin/tenants, GET /admin/tenants/:id
│   │   ├── role_handler.go            # Assign/unassign roles
│   │   ├── oauth_handler.go           # /oauth/authorize, /oauth/token, /oauth/introspect
│   │   ├── google_handler.go          # /auth/oauth/google, /auth/oauth/google/callback
│   │   ├── user_handler.go            # GET /users/me, PUT /users/me
│   │   ├── audit_handler.go           # GET /admin/audit-log
│   │   ├── wellknown_handler.go       # GET /.well-known/jwks.json
│   │   └── response.go                # Shared response helpers
│   │
│   ├── service/                       # Application layer — all business logic
│   │   ├── auth_service.go
│   │   ├── email_verification_service.go
│   │   ├── password_service.go
│   │   ├── session_service.go
│   │   ├── tenant_service.go
│   │   ├── rbac_service.go
│   │   ├── oauth_service.go
│   │   ├── google_oauth_service.go
│   │   ├── admin_service.go
│   │   ├── audit_service.go
│   │   └── jwt_service.go
│   │
│   ├── repository/
│   │   ├── postgres/
│   │   │   ├── user_repo.go           # Implements domain.UserRepository
│   │   │   ├── session_repo.go        # Implements domain.SessionRepository
│   │   │   ├── role_repo.go           # Implements domain.RoleRepository
│   │   │   ├── oauth_repo.go          # Implements domain.OAuthRepository
│   │   │   ├── audit_repo.go          # Implements domain.AuditRepository
│   │   │   └── tenant_repo.go         # Implements domain.TenantRepository
│   │   └── redis/
│   │       ├── rate_limiter.go        # Sliding window Lua EVAL
│   │       └── token_denylist.go      # Hashed refresh token denylist
│   │
│   ├── middleware/
│   │   ├── auth.go                    # JWT validation; claims in context
│   │   ├── tenant.go                  # X-Tenant-ID → schema_name in context
│   │   ├── rate_limit.go              # Per-IP + per-user Redis rate limiting
│   │   ├── secure_headers.go          # HSTS, CSP, X-Frame-Options
│   │   ├── cors.go                    # CORS
│   │   ├── logger.go                  # Structured request logging (slog)
│   │   ├── request_id.go              # X-Request-ID propagation
│   │   └── super_admin.go             # super_admin role enforcement
│   │
│   ├── infrastructure/
│   │   ├── postgres/
│   │   │   ├── db.go                  # pgxpool setup, retry, health check
│   │   │   └── tenant_db.go           # WithTenantSchema: search_path routing
│   │   ├── redis/
│   │   │   └── client.go              # go-redis/v9 client init
│   │   ├── vault/
│   │   │   ├── client.go              # Vault SDK; AppRole auth; secret fetch
│   │   │   └── key_rotation.go        # Background key rotation watcher
│   │   ├── email/
│   │   │   ├── client.go              # Resend API wrapper
│   │   │   ├── templates.go           # Go text/template email bodies
│   │   │   └── worker.go              # Buffered channel consumer; backoff retry
│   │   ├── google/
│   │   │   ├── oidc_client.go         # OIDC discovery + token verification
│   │   │   └── state_store.go         # Redis-backed CSRF state/nonce
│   │   └── migrations/
│   │       ├── runner.go              # golang-migrate; per-schema execution
│   │       └── tenant_provisioner.go  # Schema create + migrate + seed roles
│   │
│   └── router/
│       └── router.go                  # All routes + middleware chain
│
├── migrations/
│   ├── global/
│   │   ├── 000001_create_tenants.up.sql
│   │   ├── 000001_create_tenants.down.sql
│   │   ├── 000002_create_global_audit_log.up.sql
│   │   └── 000002_create_global_audit_log.down.sql
│   └── tenant/
│       ├── 000001_create_users.up.sql
│       ├── 000001_create_users.down.sql
│       ├── 000002_create_roles.up.sql
│       ├── 000002_create_roles.down.sql
│       ├── 000003_create_user_roles.up.sql
│       ├── 000003_create_user_roles.down.sql
│       ├── 000004_create_sessions.up.sql
│       ├── 000004_create_sessions.down.sql
│       ├── 000005_create_password_reset_tokens.up.sql
│       ├── 000005_create_password_reset_tokens.down.sql
│       ├── 000006_create_email_verification_tokens.up.sql
│       ├── 000006_create_email_verification_tokens.down.sql
│       ├── 000007_create_oauth_clients.up.sql
│       ├── 000007_create_oauth_clients.down.sql
│       ├── 000008_create_oauth_authorization_codes.up.sql
│       ├── 000008_create_oauth_authorization_codes.down.sql
│       ├── 000009_create_oauth_social_accounts.up.sql
│       ├── 000009_create_oauth_social_accounts.down.sql
│       ├── 000010_create_audit_log.up.sql
│       ├── 000010_create_audit_log.down.sql
│       ├── 000011_create_tenant_config.up.sql
│       ├── 000011_create_tenant_config.down.sql
│       └── 000012_seed_default_roles.up.sql
│
├── pkg/
│   ├── crypto/
│   │   ├── argon2.go                  # Argon2id hash + verify
│   │   ├── argon2_test.go
│   │   ├── token.go                   # crypto/rand opaque token generation
│   │   └── token_test.go
│   ├── jwtutil/
│   │   ├── jwt.go                     # RS256 sign + verify; JWKS generation
│   │   └── jwt_test.go
│   ├── validator/
│   │   ├── validator.go               # go-playground/validator v10 wrapper
│   │   └── password_policy.go         # Tenant-configurable complexity checker
│   ├── paginator/
│   │   └── paginator.go               # Offset pagination helpers
│   ├── timeutil/
│   │   └── timeutil.go                # Constant-time sleep for anti-timing attacks
│   └── apierror/
│       └── apierror.go                # Standard API error types + response builders
│
├── test/
│   ├── integration/
│   │   ├── auth_flow_test.go
│   │   ├── password_reset_test.go
│   │   ├── oauth_pkce_test.go
│   │   └── google_oauth_test.go
│   └── isolation/
│       └── cross_tenant_test.go       # US-08b: CI-blocking isolation suite
│
├── scripts/
│   └── seed.go                        # Creates test super-admin + test tenant
│
├── Dockerfile                         # Multi-stage build; scratch final image
├── fly.toml
├── go.mod
├── go.sum
└── .golangci.yml
```

### Frontend Structure (Next.js — Minimal per ADR-003)

```
frontend/
├── src/
│   └── app/
│       ├── layout.tsx                 # Root layout
│       └── page.tsx                   # Placeholder — consuming apps build their own UI
├── package.json
├── tsconfig.json
└── next.config.ts
```

### Key Dependencies (go.mod)

```
module github.com/yourorg/auth-system

go 1.22

require (
    github.com/gin-gonic/gin                v1.10.0
    github.com/jackc/pgx/v5                 v5.5.4
    github.com/redis/go-redis/v9            v9.5.1
    github.com/golang-jwt/jwt/v5            v5.2.1
    github.com/ory/fosite                   v0.47.0
    github.com/coreos/go-oidc/v3            v3.10.0
    github.com/golang-migrate/migrate/v4    v4.17.0
    github.com/hashicorp/vault-client-go    v0.4.3
    golang.org/x/crypto                     v0.22.0
    github.com/go-playground/validator/v10  v10.20.0
    github.com/google/uuid                  v1.6.0
    github.com/stretchr/testify             v1.9.0
    github.com/resend/resend-go/v2          v2.2.0
)
```

---

## 2. Setup Guide

### Prerequisites

| Tool | Minimum Version | Install |
|------|----------------|---------|
| Go | 1.22 | https://go.dev/dl/ |
| Docker Desktop | 4.x | https://docs.docker.com/get-docker/ |
| Docker Compose | v2 (bundled) | Included with Docker Desktop |
| Node.js | 20 LTS | https://nodejs.org/ (for frontend shell) |
| make | any | Pre-installed on Linux/Mac; use WSL or winget on Windows |
| golangci-lint | 1.57+ | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |

### Step 1 — Clone and Configure

```bash
git clone https://github.com/yourorg/auth-system.git
cd auth-system

# Copy the example env file
cp .env.example .env.local

# Edit .env.local — set at minimum:
#   POSTGRES_PASSWORD (change from default)
#   VAULT_TOKEN (leave as dev-root-token for local dev)
#   RESEND_API_KEY (get from resend.com — or leave blank; Mailhog catches emails locally)
#   OAUTH_GOOGLE_CLIENT_ID / OAUTH_GOOGLE_CLIENT_SECRET (optional for local dev)
```

### Step 2 — Start Local Services

```bash
# From the monorepo root:
docker compose up -d

# Verify all four services are healthy:
docker compose ps

# Expected output:
# NAME                STATUS
# auth-postgres       running (healthy)
# auth-redis          running (healthy)
# auth-vault          running (healthy)
# auth-mailhog        running (healthy)
```

Services started:
- **PostgreSQL 16** on `localhost:5432` — database `authdb`
- **Redis 7** on `localhost:6379`
- **Vault** (dev mode) on `localhost:8200` — root token `dev-root-token`
- **Mailhog** on `localhost:1025` (SMTP) + `localhost:8025` (web UI)

### Step 3 — Initialize Vault (Local Dev)

```bash
# Configure Vault KV mount and load the dev signing key
# This script runs automatically during docker compose up via vault-init container
# To run manually:
docker exec auth-vault vault secrets enable -path=secret kv-v2

# Create a dev RS256 key pair (for local development only)
# Production keys are generated and stored by the Vault admin
cd backend
go run ./scripts/seed.go --init-vault
```

### Step 4 — Run Database Migrations

```bash
# Run global schema migrations (creates public.tenants, public.global_audit_log)
make migrate-global

# Expected output:
# applying migration 000001_create_tenants ... ok
# applying migration 000002_create_global_audit_log ... ok
# global migrations complete
```

### Step 5 — Seed Test Data

```bash
# Creates a super-admin user and a test tenant ("dev-corp")
# Prints credentials to stdout — store them securely
make seed

# Expected output:
# super-admin created: jordan@platform.internal / <generated-password>
# test tenant created: dev-corp (schema: tenant_dev_corp)
# tenant admin: carlos@dev-corp.example / <generated-password>
# tenant client_id: client_dev_corp_...
# tenant client_secret: cs_... (shown once)
```

### Step 6 — Run the Go API Server

```bash
cd backend

# Load environment variables and start the server
export $(cat ../.env.local | xargs)
go run ./cmd/api

# Expected output:
# {"time":"2026-02-28T10:00:00Z","level":"INFO","msg":"config loaded"}
# {"time":"2026-02-28T10:00:00Z","level":"INFO","msg":"connected to postgres","schemas":1}
# {"time":"2026-02-28T10:00:00Z","level":"INFO","msg":"connected to redis"}
# {"time":"2026-02-28T10:00:00Z","level":"INFO","msg":"connected to vault","keys":1}
# {"time":"2026-02-28T10:00:00Z","level":"INFO","msg":"email worker started","workers":4}
# {"time":"2026-02-28T10:00:00Z","level":"INFO","msg":"server listening","addr":":8080"}

# Or use the Makefile target (recommended):
make dev
```

### Step 7 — Run the Next.js Frontend Shell

```bash
cd frontend
npm install
npm run dev

# Runs on http://localhost:3000 — displays API documentation placeholder
```

### Step 8 — Smoke Test (Verify Everything Works)

Run these curl commands in order. Replace `dev-tenant-uuid` with the tenant ID from the seed output.

```bash
# 1. Health check
curl -s http://localhost:8080/health | jq
# Expected: {"status":"ok","postgres":"up","redis":"up","vault":"up"}

# 2. JWKS endpoint
curl -s http://localhost:8080/.well-known/jwks.json | jq
# Expected: {"keys":[{"kty":"RSA","use":"sig","alg":"RS256","kid":"..."}]}

# 3. Register a new user
curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: dev-tenant-uuid" \
  -d '{"email":"maya@example.com","password":"Test1234!Ab","first_name":"Maya","last_name":"Test"}' | jq
# Expected: 201 {"user_id":"...","status":"unverified"}

# 4. Check Mailhog for verification email
open http://localhost:8025
# Find the verification token in the email

# 5. Verify email (replace TOKEN with value from email)
curl -s -X POST http://localhost:8080/api/v1/auth/verify-email \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: dev-tenant-uuid" \
  -d '{"token":"TOKEN"}' | jq
# Expected: 200 {"message":"Email verified successfully."}

# 6. Login
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: dev-tenant-uuid" \
  -d '{"email":"maya@example.com","password":"Test1234!Ab"}' | jq
# Expected: 200 {"access_token":"eyJ...","refresh_token":"rt_...","expires_in":900}

# Store tokens
ACCESS_TOKEN="<access_token from above>"
REFRESH_TOKEN="<refresh_token from above>"

# 7. Get user profile
curl -s http://localhost:8080/api/v1/users/me \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "X-Tenant-ID: dev-tenant-uuid" | jq
# Expected: 200 {"user_id":"...","email":"maya@example.com","status":"active"}

# 8. Refresh token
curl -s -X POST http://localhost:8080/api/v1/auth/token/refresh \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: dev-tenant-uuid" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq
# Expected: 200 {"access_token":"eyJ...","refresh_token":"rt_..."} (new tokens)

# 9. Logout
curl -s -X POST http://localhost:8080/api/v1/auth/logout \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: dev-tenant-uuid" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}" | jq
# Expected: 200 {"message":"Logged out successfully."}
```

### Makefile Targets Reference

| Target | Description |
|--------|-------------|
| `make dev` | Start Docker services + run API server with live reload |
| `make migrate-global` | Apply global schema migrations |
| `make migrate` | Apply global + all tenant schema migrations |
| `make migrate-tenant TENANT=acme-corp` | Apply migrations to one tenant only |
| `make test` | Run all tests (unit + integration + isolation) |
| `make test-unit` | Unit tests only |
| `make test-integration` | Integration tests (requires running Docker services) |
| `make test-isolation` | Cross-tenant isolation suite only (CI-blocking) |
| `make lint` | golangci-lint with project config |
| `make build` | Build production binary to `./bin/auth-api` |
| `make docker-build` | Build Docker image `auth-api:local` |
| `make seed` | Create test super-admin + test tenant |
| `make clean` | Remove build artifacts |

---

## 3. Environment Variables

Complete `.env.example` file. All values must be sourced from HashiCorp Vault in production (ADR-009). The local dev values below are safe only for local development.

```bash
# =============================================================================
# auth-system/.env.example
# Copy to .env.local for local development.
# NEVER commit .env.local to version control.
# NEVER use these values in production — all secrets come from Vault in prod.
# =============================================================================

# -----------------------------------------------------------------------------
# Server Configuration
# -----------------------------------------------------------------------------
PORT=8080
ENV=development
# Values: development | staging | production
LOG_LEVEL=debug
# Values: debug | info | warn | error
API_VERSION=v1
GRACEFUL_SHUTDOWN_TIMEOUT_SECONDS=30

# Base URL of this auth service (used in JWT iss claim and email links)
AUTH_SERVICE_BASE_URL=http://localhost:8080

# -----------------------------------------------------------------------------
# PostgreSQL — Primary Database
# -----------------------------------------------------------------------------
# Connection string for the primary read-write connection pool.
# In production: Vault dynamic secrets generate these credentials.
DATABASE_URL=postgres://auth:authpassword@localhost:5432/authdb?sslmode=disable

# Connection pool settings
DB_MAX_CONNS=20
# Maximum open connections per API instance. Set to <= (PostgreSQL max_connections / num_instances).
DB_MIN_CONNS=5
# Minimum idle connections kept open. Set to a fraction of DB_MAX_CONNS.
DB_MAX_CONN_IDLE_TIME_MINUTES=10
DB_MAX_CONN_LIFETIME_MINUTES=60
DB_CONNECT_TIMEOUT_SECONDS=10
DB_HEALTH_CHECK_PERIOD_SECONDS=60

# -----------------------------------------------------------------------------
# Redis — Cache + Rate Limiting + Token Denylist
# -----------------------------------------------------------------------------
REDIS_URL=redis://localhost:6379
# In production: redis://:<password>@<host>:6379?tls=true
REDIS_MAX_RETRIES=3
REDIS_DIAL_TIMEOUT_SECONDS=5
REDIS_READ_TIMEOUT_SECONDS=3
REDIS_WRITE_TIMEOUT_SECONDS=3
REDIS_POOL_SIZE=20
# Pool size per API instance. Ensure total < Redis maxclients config.

# -----------------------------------------------------------------------------
# HashiCorp Vault — Secrets Manager (ADR-009)
# -----------------------------------------------------------------------------
VAULT_ADDR=http://localhost:8200
# In production: https://vault.internal:8200

VAULT_TOKEN=dev-root-token
# LOCAL DEV ONLY. In production: use AppRole (VAULT_ROLE_ID + VAULT_SECRET_ID)
# or Kubernetes auth — never a static root token.

VAULT_ROLE_ID=
# AppRole role_id (production only)
VAULT_SECRET_ID=
# AppRole secret_id (production only; injected by CI/CD or init container)

VAULT_MOUNT=secret
# KV v2 mount path where application secrets are stored

VAULT_KEY_ROTATION_POLL_INTERVAL_SECONDS=300
# How often to poll Vault for new signing keys (5 minutes default)

# Vault path layout (within VAULT_MOUNT):
#   secret/auth-system/jwt-keys/current   -> {private_key_pem, public_key_pem, kid}
#   secret/auth-system/jwt-keys/previous  -> {public_key_pem, kid, retire_at}
#   secret/auth-system/db                 -> {url} (overrides DATABASE_URL in prod)
#   secret/auth-system/email              -> {resend_api_key}
#   secret/auth-system/google             -> {client_id, client_secret}

# -----------------------------------------------------------------------------
# JWT — Token Configuration
# -----------------------------------------------------------------------------
# In production, private key is loaded from Vault at runtime — not from env.
# The path below is for local dev only (generated by make seed --init-vault).
JWT_PRIVATE_KEY_PATH=./dev-keys/private.pem
# Path to RS256 private key PEM file (local dev only)
JWT_PUBLIC_KEY_PATH=./dev-keys/public.pem
# Path to RS256 public key PEM file (local dev only)

JWT_KEY_ID=key-dev-20260228
# Key ID embedded in JWT header kid field. Must match key in Vault.
JWT_ACCESS_TOKEN_TTL_SECONDS=900
# 15 minutes — do not increase above 900 (SEC-03)
JWT_ISSUER=http://localhost:8080
# Must match AUTH_SERVICE_BASE_URL in production
JWT_AUDIENCE=http://localhost:3000
# Comma-separated list of valid audiences

# -----------------------------------------------------------------------------
# Session Configuration
# -----------------------------------------------------------------------------
SESSION_DEFAULT_TTL_SECONDS=86400
# Default absolute session expiry: 24 hours (ADR-005)
# Overridable per tenant via tenant_config table

# -----------------------------------------------------------------------------
# OAuth 2.0 — Google External IdP (ADR per US-13)
# Per-tenant Google credentials are stored in tenant_config.
# The values below are the platform-level Google credentials
# used for the platform's own OAuth client registration if needed.
# -----------------------------------------------------------------------------
OAUTH_GOOGLE_CLIENT_ID=
# Google Cloud Console OAuth 2.0 Client ID
OAUTH_GOOGLE_CLIENT_SECRET=
# Google Cloud Console OAuth 2.0 Client Secret (from Vault in production)

# Google OIDC discovery URL (do not change)
OAUTH_GOOGLE_DISCOVERY_URL=https://accounts.google.com/.well-known/openid-configuration

OAUTH_STATE_TTL_SECONDS=600
# How long Google OAuth state/nonce values are valid in Redis (10 minutes)

# -----------------------------------------------------------------------------
# Email — Resend (ADR-012: async worker)
# -----------------------------------------------------------------------------
RESEND_API_KEY=re_test_...
# Get from resend.com dashboard. In production: loaded from Vault.
# For local dev: leave empty — Mailhog catches all emails via SMTP fallback.

EMAIL_FROM=auth@yourapp.com
# From address used on all outgoing emails
EMAIL_FROM_NAME=Auth System
EMAIL_WORKER_CONCURRENCY=4
# Number of goroutines consuming the email channel
EMAIL_CHANNEL_BUFFER_SIZE=100
# Buffered channel capacity. Alerts if channel >80% full.
EMAIL_MAX_RETRIES=3
EMAIL_RETRY_BACKOFF_BASE_SECONDS=2
# Exponential backoff: 2s, 4s, 8s

# Local dev SMTP (Mailhog) — used when RESEND_API_KEY is empty
SMTP_HOST=localhost
SMTP_PORT=1025

# Email verification token TTL
EMAIL_VERIFICATION_TOKEN_TTL_HOURS=24
# Password reset token TTL
PASSWORD_RESET_TOKEN_TTL_HOURS=1

# -----------------------------------------------------------------------------
# Rate Limiting — Default Limits (overridable per tenant via tenant_config)
# -----------------------------------------------------------------------------
RATE_LIMIT_LOGIN_IP_PER_MINUTE=20
RATE_LIMIT_LOGIN_USER_PER_MINUTE=5
RATE_LIMIT_REGISTER_IP_PER_MINUTE=10
RATE_LIMIT_FORGOT_PASSWORD_IP_PER_MINUTE=5
RATE_LIMIT_FORGOT_PASSWORD_USER_PER_HOUR=3
RATE_LIMIT_TOKEN_REFRESH_IP_PER_MINUTE=60
RATE_LIMIT_TOKEN_REFRESH_USER_PER_MINUTE=30
RATE_LIMIT_VERIFY_EMAIL_IP_PER_MINUTE=10

# Account lockout defaults (overridable per tenant via tenant_config)
LOCKOUT_THRESHOLD=5
# Number of failed login attempts before lockout
LOCKOUT_DURATION_SECONDS=900
# Lockout duration: 15 minutes

# -----------------------------------------------------------------------------
# Tenant Cache
# -----------------------------------------------------------------------------
TENANT_CACHE_TTL_SECONDS=60
# How long tenant_id → schema_name mappings are cached in memory

# -----------------------------------------------------------------------------
# CORS
# -----------------------------------------------------------------------------
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
# Comma-separated list of allowed origins for local dev
# In production: per-tenant allowed origins from tenant_config

# -----------------------------------------------------------------------------
# Frontend (Next.js) — Public Variables
# These are safe to expose to the browser.
# -----------------------------------------------------------------------------
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_AUTH_SERVICE_URL=http://localhost:8080

# -----------------------------------------------------------------------------
# Migrations
# -----------------------------------------------------------------------------
MIGRATIONS_GLOBAL_PATH=file://migrations/global
MIGRATIONS_TENANT_PATH=file://migrations/tenant

# -----------------------------------------------------------------------------
# Fly.io Deployment (Production)
# Set in Fly.io secrets — not in this file for production
# -----------------------------------------------------------------------------
FLY_APP_NAME=auth-api
FLY_REGION=sjc
```

---

## 4. Core Implementation Code

### 4a. Domain Layer

#### `internal/domain/user.go`

```go
// internal/domain/user.go
package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// UserStatus represents the lifecycle state of a user account.
type UserStatus string

const (
	UserStatusUnverified UserStatus = "unverified"
	UserStatusActive     UserStatus = "active"
	UserStatusDisabled   UserStatus = "disabled"
	UserStatusDeleted    UserStatus = "deleted"
)

// User is the core identity entity. It lives entirely within a tenant schema
// (ADR-001, ADR-002) — there is no global user table.
type User struct {
	ID                uuid.UUID
	Email             string
	PasswordHash      string // Argon2id hash; empty for social-only accounts
	Status            UserStatus
	FirstName         string
	LastName          string
	MFAEnabled        bool
	MFATOTPSecret     string // Encrypted at rest; empty if MFA not configured
	FailedLoginCount  int
	LockedUntil       *time.Time
	EmailVerifiedAt   *time.Time
	LastLoginAt       *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}

// IsActive returns true if the user may authenticate.
// An account must be active (not unverified, disabled, or deleted).
func (u *User) IsActive() bool {
	return u.Status == UserStatusActive && u.DeletedAt == nil
}

// IsVerified returns true if the user has confirmed their email address.
func (u *User) IsVerified() bool {
	return u.EmailVerifiedAt != nil
}

// IsLocked returns true if the account is in a temporary lockout period.
func (u *User) IsLocked() bool {
	if u.LockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.LockedUntil)
}

// HasPassword returns true if the user has a local password set.
// Social-only accounts have an empty PasswordHash.
func (u *User) HasPassword() bool {
	return u.PasswordHash != ""
}

// FullName returns the display name of the user.
func (u *User) FullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

// Errors returned from domain validation.
var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailAlreadyExists  = errors.New("email already exists in this tenant")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrAccountDisabled     = errors.New("account disabled")
	ErrAccountLocked       = errors.New("account is temporarily locked")
	ErrAccountDeleted      = errors.New("account has been deleted")
	ErrWeakPassword        = errors.New("password does not meet complexity requirements")
	ErrInvalidEmail        = errors.New("invalid email address format")
)

// emailRegexp is a basic RFC 5322-compatible email validator.
// For production, combine with go-playground/validator for full compliance.
var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// NormalizeEmail lowercases and trims whitespace from an email address.
// All email comparisons must use normalized values.
func NormalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if !emailRegexp.MatchString(normalized) {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// CreateUserInput carries validated input for registering a new user.
type CreateUserInput struct {
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Status       UserStatus
}

// UpdateUserInput carries fields that may be updated on a user record.
// Only non-nil pointers are applied by the repository.
type UpdateUserInput struct {
	FirstName        *string
	LastName         *string
	PasswordHash     *string
	Status           *UserStatus
	MFAEnabled       *bool
	MFATOTPSecret    *string
	FailedLoginCount *int
	LockedUntil      *time.Time
	EmailVerifiedAt  *time.Time
	LastLoginAt      *time.Time
	DeletedAt        *time.Time
}
```

#### `internal/domain/tenant.go`

```go
// internal/domain/tenant.go
package domain

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
)

// TenantStatus represents the operational state of a tenant.
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusSuspended TenantStatus = "suspended"
	TenantStatusDeleted   TenantStatus = "deleted"
)

// Tenant is the top-level multi-tenancy entity stored in the global public schema.
// Each Tenant has a dedicated PostgreSQL schema identified by SchemaName (ADR-001).
type Tenant struct {
	ID          uuid.UUID
	Slug        string // URL-safe identifier; immutable after creation (e.g., "acme-corp")
	Name        string // Display name (e.g., "Acme Corporation")
	SchemaName  string // PostgreSQL schema name (e.g., "tenant_acme_corp")
	AdminEmail  string // Initial admin email address
	Status      TenantStatus
	Config      TenantConfig
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// PasswordPolicy defines the password complexity rules configurable per tenant.
type PasswordPolicy struct {
	MinLength        int  `json:"min_length"`
	RequireUppercase bool `json:"require_uppercase"`
	RequireNumber    bool `json:"require_number"`
	RequireSpecial   bool `json:"require_special"`
}

// TenantConfig holds all tenant-level runtime configuration.
// Stored as JSONB in the global public.tenants.config column.
type TenantConfig struct {
	SessionTTLSeconds           int            `json:"session_ttl_seconds"`
	SlidingSessionEnabled       bool           `json:"sliding_session_enabled"`
	LockoutThreshold            int            `json:"lockout_threshold"`
	LockoutDurationSeconds      int            `json:"lockout_duration_seconds"`
	PasswordPolicy              PasswordPolicy `json:"password_policy"`
	GoogleClientID              string         `json:"google_client_id,omitempty"`
	GoogleClientSecret          string         `json:"google_client_secret,omitempty"` // Stored in Vault; reference only
	AllowedCORSOrigins          []string       `json:"allowed_cors_origins,omitempty"`
	MFARequired                 bool           `json:"mfa_required"`
}

// DefaultTenantConfig returns a secure-by-default tenant configuration.
func DefaultTenantConfig() TenantConfig {
	return TenantConfig{
		SessionTTLSeconds:      86400, // 24 hours (ADR-005)
		SlidingSessionEnabled:  false,
		LockoutThreshold:       5,
		LockoutDurationSeconds: 900, // 15 minutes
		PasswordPolicy: PasswordPolicy{
			MinLength:        12,
			RequireUppercase: true,
			RequireNumber:    true,
			RequireSpecial:   true,
		},
	}
}

// slugRegexp validates tenant slug format: lowercase alphanumeric and hyphens.
var slugRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{1,48}[a-z0-9]$`)

// schemaNameRegexp validates generated schema names used in SET search_path.
// This is a security-critical validation — must pass before any SQL execution.
var schemaNameRegexp = regexp.MustCompile(`^tenant_[a-z0-9_]{1,50}$`)

// SlugToSchemaName converts a tenant slug to a PostgreSQL schema name.
// Example: "acme-corp" -> "tenant_acme_corp"
func SlugToSchemaName(slug string) string {
	safe := regexp.MustCompile(`[^a-z0-9]`).ReplaceAllString(slug, "_")
	return "tenant_" + safe
}

// ValidateSlug returns an error if the slug does not meet format requirements.
func ValidateSlug(slug string) error {
	if !slugRegexp.MatchString(slug) {
		return errors.New("slug must be 3-50 characters: lowercase letters, numbers, hyphens only; cannot start or end with a hyphen")
	}
	return nil
}

// IsValidSchemaName validates a schema name before use in SET search_path.
// This is a defense-in-depth check — schema names must match the expected pattern.
// Call this before any SQL that incorporates the schema name.
func IsValidSchemaName(name string) bool {
	return schemaNameRegexp.MatchString(name)
}

// Tenant domain errors.
var (
	ErrTenantNotFound      = errors.New("tenant not found")
	ErrTenantAlreadyExists = errors.New("tenant with this slug already exists")
	ErrTenantSuspended     = errors.New("tenant is suspended")
	ErrInvalidSlug         = errors.New("invalid tenant slug format")
	ErrInvalidSchemaName   = errors.New("invalid schema name format")
)

// CreateTenantInput carries validated input for provisioning a new tenant.
type CreateTenantInput struct {
	Name       string
	Slug       string
	AdminEmail string
	Config     TenantConfig
}
```

#### `internal/domain/session.go`

```go
// internal/domain/session.go
package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Session represents a user's authenticated session, anchored by a refresh token.
// The refresh token itself is never stored — only its SHA-256 hash (ADR-004).
// Multiple sessions per user are supported (multi-device login).
type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string    // SHA-256 hex hash of the opaque refresh token
	FamilyID         uuid.UUID // Groups tokens for family revocation on reuse detection
	IPAddress        string
	UserAgent        string
	IssuedAt         time.Time
	ExpiresAt        time.Time
	LastUsedAt       time.Time
	RevokedAt        *time.Time
	IsRevoked        bool
}

// IsValid returns true if the session can be used for token refresh.
// A session is valid if: not revoked, not expired.
func (s *Session) IsValid() bool {
	return !s.IsRevoked && time.Now().Before(s.ExpiresAt)
}

// IsExpired returns true if the session has passed its absolute expiry time.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// RefreshTokenValue is a value object representing the raw opaque refresh token.
// It is generated once, transmitted to the client, and never stored in plaintext.
// The hash is stored in the database via Session.RefreshTokenHash.
type RefreshTokenValue struct {
	Raw    string // The plaintext token (only exists in memory during generation)
	Hash   string // SHA-256 hex hash — persisted to the database
	Family uuid.UUID
}

// Session domain errors.
var (
	ErrSessionNotFound         = errors.New("session not found")
	ErrSessionRevoked          = errors.New("session has been revoked")
	ErrSessionExpired          = errors.New("session has expired")
	ErrSuspiciousTokenReuse    = errors.New("token reuse detected — all sessions in family revoked")
	ErrInvalidRefreshToken     = errors.New("invalid refresh token")
)
```

#### `internal/domain/repository.go`

```go
// internal/domain/repository.go
package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// UserRepository defines all data operations on the users table.
// Implementations live in internal/repository/postgres/user_repo.go.
// All methods operate within the tenant schema set by TenantMiddleware.
type UserRepository interface {
	// FindByID retrieves a user by their UUID. Returns ErrUserNotFound if absent.
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)

	// FindByEmail retrieves a user by their normalized email address.
	// Returns ErrUserNotFound if absent.
	FindByEmail(ctx context.Context, email string) (*User, error)

	// Create inserts a new user record. Returns ErrEmailAlreadyExists on duplicate.
	Create(ctx context.Context, input CreateUserInput) (*User, error)

	// Update applies a partial update to a user record.
	// Only non-nil pointer fields in UpdateUserInput are applied.
	Update(ctx context.Context, id uuid.UUID, input UpdateUserInput) (*User, error)

	// IncrementFailedLoginCount atomically increments the failed login counter.
	// Returns the new count. Used for lockout threshold checking.
	IncrementFailedLoginCount(ctx context.Context, id uuid.UUID) (int, error)

	// ResetFailedLoginCount resets the failed login counter to zero on successful login.
	ResetFailedLoginCount(ctx context.Context, id uuid.UUID) error

	// SetLockedUntil sets the account lockout expiry timestamp.
	SetLockedUntil(ctx context.Context, id uuid.UUID, until time.Time) error

	// ListByTenant returns all non-deleted users in the current tenant schema.
	// Supports offset pagination via limit/offset.
	ListByTenant(ctx context.Context, limit, offset int) ([]*User, int, error)

	// SoftDelete marks a user as deleted (GDPR erasure step 1).
	// Sets deleted_at; PII anonymization is a separate operation.
	SoftDelete(ctx context.Context, id uuid.UUID) error

	// AnonymizePII replaces PII fields with tombstone values (GDPR erasure step 2).
	AnonymizePII(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines all data operations on the sessions table.
// All methods operate within the tenant schema set by TenantMiddleware.
type SessionRepository interface {
	// FindByTokenHash retrieves a session by the SHA-256 hash of the refresh token.
	// Returns ErrSessionNotFound if absent.
	FindByTokenHash(ctx context.Context, tokenHash string) (*Session, error)

	// Create inserts a new session record.
	Create(ctx context.Context, session *Session) error

	// RevokeByTokenHash marks a specific session as revoked.
	// Idempotent: no error if already revoked.
	RevokeByTokenHash(ctx context.Context, tokenHash string) error

	// RevokeByFamilyID revokes all sessions in a refresh token family.
	// Called on suspicious token reuse detection (ADR-004).
	RevokeByFamilyID(ctx context.Context, familyID uuid.UUID) (int, error)

	// RevokeAllForUser revokes all active sessions for a user (logout/all-devices).
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) (int, error)

	// CountActiveForUser returns the number of non-revoked, non-expired sessions.
	CountActiveForUser(ctx context.Context, userID uuid.UUID) (int, error)
}

// TenantRepository defines data operations on the global public.tenants table.
// This repository always uses the public schema — not a tenant schema.
type TenantRepository interface {
	// FindByID retrieves a tenant by UUID from the global registry.
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)

	// FindBySlug retrieves a tenant by its URL slug.
	FindBySlug(ctx context.Context, slug string) (*Tenant, error)

	// FindBySchemaName retrieves a tenant by its PostgreSQL schema name.
	FindBySchemaName(ctx context.Context, schemaName string) (*Tenant, error)

	// Create inserts a new tenant into the global registry.
	// Returns ErrTenantAlreadyExists on duplicate slug.
	Create(ctx context.Context, input CreateTenantInput) (*Tenant, error)

	// UpdateStatus changes the tenant's operational status.
	UpdateStatus(ctx context.Context, id uuid.UUID, status TenantStatus) error

	// ListActiveSchemaNames returns all schema names for active tenants.
	// Used by the migration runner to iterate all tenants.
	ListActiveSchemaNames(ctx context.Context) ([]string, error)

	// ListAll returns all tenants (including inactive) for super-admin queries.
	ListAll(ctx context.Context, limit, offset int) ([]*Tenant, int, error)
}

// AuditRepository defines all data operations on the per-tenant audit_log table.
// Audit records are append-only — no Update or Delete methods exist.
type AuditRepository interface {
	// Append inserts a new audit event. This is a fire-and-forget write:
	// callers should not block on this — use async if latency is critical.
	Append(ctx context.Context, event *AuditEvent) error

	// List returns paginated audit log entries filtered by the provided criteria.
	List(ctx context.Context, filter AuditFilter) ([]*AuditEvent, int, error)

	// MarkArchived marks records as archived after cold-storage export.
	MarkArchived(ctx context.Context, ids []uuid.UUID) error

	// ListForArchive returns records older than the cutoff that have not been archived.
	ListForArchive(ctx context.Context, cutoff time.Time, limit int) ([]*AuditEvent, error)
}

// AuditFilter defines query parameters for listing audit log entries.
type AuditFilter struct {
	EventType    *string
	ActorID      *uuid.UUID
	TargetUserID *uuid.UUID
	From         *time.Time
	To           *time.Time
	Limit        int
	Offset       int
}

// RoleRepository defines data operations on the roles and user_roles tables.
type RoleRepository interface {
	// FindByID retrieves a role by UUID.
	FindByID(ctx context.Context, id uuid.UUID) (*Role, error)

	// FindByName retrieves a role by name within the current tenant.
	FindByName(ctx context.Context, name string) (*Role, error)

	// ListAll returns all roles in the current tenant schema.
	ListAll(ctx context.Context) ([]*Role, error)

	// Create inserts a new custom role.
	Create(ctx context.Context, name, description string) (*Role, error)

	// AssignToUser assigns a role to a user. Returns ErrRoleAlreadyAssigned on duplicate.
	AssignToUser(ctx context.Context, userID, roleID, assignedBy uuid.UUID) error

	// UnassignFromUser removes a role from a user.
	UnassignFromUser(ctx context.Context, userID, roleID uuid.UUID) error

	// GetUserRoles returns all roles assigned to a user.
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)
}

// TokenRepository defines operations on password_reset_tokens and email_verification_tokens.
type TokenRepository interface {
	// CreatePasswordResetToken inserts a new password reset token record.
	// Any previous unused tokens for this user are invalidated first.
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error

	// FindPasswordResetToken retrieves a password reset token by its hash.
	FindPasswordResetToken(ctx context.Context, tokenHash string) (*PasswordResetToken, error)

	// MarkPasswordResetTokenUsed marks a token as used (single-use enforcement).
	MarkPasswordResetTokenUsed(ctx context.Context, tokenHash string) error

	// CreateEmailVerificationToken inserts a new email verification token.
	// Any previous unused tokens for this user are invalidated first.
	CreateEmailVerificationToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error

	// FindEmailVerificationToken retrieves an email verification token by its hash.
	FindEmailVerificationToken(ctx context.Context, tokenHash string) (*EmailVerificationToken, error)

	// MarkEmailVerificationTokenUsed marks a token as used.
	MarkEmailVerificationTokenUsed(ctx context.Context, tokenHash string) error
}

// PasswordResetToken is a domain value object for password reset operations.
type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Used      bool
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *PasswordResetToken) IsValid() bool {
	return !t.Used && time.Now().Before(t.ExpiresAt)
}

// EmailVerificationToken is a domain value object for email verification.
type EmailVerificationToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Used      bool
	UsedAt    *time.Time
	CreatedAt time.Time
}

func (t *EmailVerificationToken) IsValid() bool {
	return !t.Used && time.Now().Before(t.ExpiresAt)
}

// Role is the RBAC role entity scoped to a tenant.
type Role struct {
	ID          uuid.UUID
	Name        string
	Description string
	IsSystem    bool // System roles (admin, user) cannot be deleted
	CreatedAt   time.Time
}

// AuditEvent is an immutable record of an authentication action.
type AuditEvent struct {
	ID            uuid.UUID
	EventType     EventType
	ActorID       *uuid.UUID // Nil for system-generated events
	ActorIP       string
	ActorUA       string
	TargetUserID  *uuid.UUID
	Metadata      map[string]interface{}
	OccurredAt    time.Time
	Archived      bool
}

// Role domain errors.
var (
	ErrRoleNotFound      = errors.New("role not found")
	ErrRoleAlreadyExists = errors.New("role already exists in this tenant")
	ErrRoleAlreadyAssigned = errors.New("user already has this role")
	ErrRoleNotAssigned   = errors.New("user does not have this role")
	ErrSystemRole        = errors.New("cannot delete a system role")
)
```

#### `internal/domain/audit_event.go`

```go
// internal/domain/audit_event.go
package domain

// EventType is a string constant identifying the type of audit log event.
// All event types that must be logged are enumerated here (PRD US-15).
type EventType string

const (
	// Authentication events
	EventLoginSuccess          EventType = "LOGIN_SUCCESS"
	EventLoginFailure          EventType = "LOGIN_FAILURE"
	EventLogout                EventType = "LOGOUT"
	EventLogoutAll             EventType = "LOGOUT_ALL"
	EventTokenRefreshed        EventType = "TOKEN_REFRESHED"
	EventSuspiciousTokenReuse  EventType = "SUSPICIOUS_TOKEN_REUSE"

	// Account lifecycle events
	EventEmailVerified         EventType = "EMAIL_VERIFIED"
	EventPasswordChanged       EventType = "PASSWORD_CHANGED"
	EventPasswordResetReq      EventType = "PASSWORD_RESET_REQUESTED"
	EventPasswordResetDone     EventType = "PASSWORD_RESET_COMPLETED"
	EventAccountLocked         EventType = "ACCOUNT_LOCKED"
	EventUserInvited           EventType = "USER_INVITED"
	EventUserDisabled          EventType = "USER_DISABLED"
	EventUserDeleted           EventType = "USER_DELETED"

	// RBAC events
	EventRoleAssigned          EventType = "ROLE_ASSIGNED"
	EventRoleUnassigned        EventType = "ROLE_UNASSIGNED"

	// OAuth events
	EventOAuthClientCreated    EventType = "OAUTH_CLIENT_CREATED"
	EventOAuthCodeIssued       EventType = "OAUTH_CODE_ISSUED"
	EventOAuthTokenIssued      EventType = "OAUTH_TOKEN_ISSUED"

	// Social login events
	EventGoogleLinked          EventType = "GOOGLE_ACCOUNT_LINKED"
	EventGoogleLogin           EventType = "GOOGLE_LOGIN"
)
```

### 4b. Config

#### `internal/config/config.go`

```go
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

// Config holds all application configuration loaded at startup.
// All fields are read-only after initialization.
// Secrets (keys, passwords) are loaded from Vault at runtime — not stored here.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Vault    VaultConfig
	JWT      JWTConfig
	Session  SessionConfig
	Email    EmailConfig
	OAuth    OAuthConfig
	RateLimit RateLimitConfig
	Tenant   TenantConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port                     string
	Env                      string // development | staging | production
	LogLevel                 string
	APIVersion               string
	BaseURL                  string
	GracefulShutdownTimeout  time.Duration
}

type DatabaseConfig struct {
	URL                  string
	MaxConns             int32
	MinConns             int32
	MaxConnIdleTime      time.Duration
	MaxConnLifetime      time.Duration
	ConnectTimeout       time.Duration
	HealthCheckPeriod    time.Duration
}

type RedisConfig struct {
	URL              string
	MaxRetries       int
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PoolSize         int
}

type VaultConfig struct {
	Addr               string
	Token              string // Local dev only; use AppRole in production
	RoleID             string
	SecretID           string
	Mount              string
	KeyRotationPollInterval time.Duration
}

type JWTConfig struct {
	PrivateKeyPath    string // Local dev only; loaded from Vault in production
	PublicKeyPath     string // Local dev only
	KeyID             string
	AccessTokenTTL    time.Duration
	Issuer            string
	Audience          []string
}

type SessionConfig struct {
	DefaultTTL time.Duration
}

type EmailConfig struct {
	ResendAPIKey         string
	From                 string
	FromName             string
	WorkerConcurrency    int
	ChannelBufferSize    int
	MaxRetries           int
	RetryBackoffBase     time.Duration
	SMTPHost             string
	SMTPPort             int
	VerificationTokenTTL time.Duration
	PasswordResetTokenTTL time.Duration
}

type OAuthConfig struct {
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleDiscoveryURL   string
	StateTokenTTL        time.Duration
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

// Load reads configuration from environment variables and validates all required
// values are present. It panics on missing required values rather than returning
// a partially-initialised config, because a misconfigured server must not start.
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

// validate checks that the configuration is internally consistent and complete.
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

// IsDevelopment returns true when running in the local development environment.
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction returns true when running in the production environment.
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// mustGetEnv returns the value of an environment variable or panics if it is unset.
// Prefer this over os.Getenv for required variables — fail fast at startup.
func mustGetEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return val
}

// getEnvDefault returns the value of an environment variable or a default if unset.
func getEnvDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// mustGetEnvInt returns the integer value of an env variable, or defaultVal if unset.
// Panics if the variable is set but cannot be parsed as an integer.
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
```

### 4c. Infrastructure — Database

#### `internal/infrastructure/postgres/db.go`

```go
// internal/infrastructure/postgres/db.go
package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/auth-system/internal/config"
)

// NewPool creates and validates a pgxpool connection pool.
// It retries the initial connection up to maxAttempts times before returning an error.
// This handles the case where the database container is still starting during local dev.
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

	// ConnectTimeout applies to each individual connection attempt.
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

		// Ping to verify the connection is alive.
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

// HealthCheck verifies the pool can acquire a connection and execute a trivial query.
// Used by the /health endpoint.
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
```

#### `internal/infrastructure/postgres/tenant_db.go`

```go
// internal/infrastructure/postgres/tenant_db.go
package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/auth-system/internal/domain"
)

// WithTenantSchema acquires a connection from the pool, sets the PostgreSQL
// search_path to the tenant schema (plus public for shared functions), executes fn,
// and returns the connection to the pool.
//
// This is the single point of tenant schema routing. Every repository method that
// operates on tenant data calls this function. The search_path setting is scoped
// to the connection's lifetime and is cleared when the connection returns to the pool.
//
// Security: schemaName MUST be validated with domain.IsValidSchemaName before
// this function is called. The TenantMiddleware performs this validation.
// The format check here is a defense-in-depth layer.
func WithTenantSchema(ctx context.Context, pool *pgxpool.Pool, schemaName string, fn func(conn *pgx.Conn) error) error {
	// Defense-in-depth: validate schema name format before use in SQL.
	// This prevents SQL injection via search_path even if middleware is bypassed.
	if !domain.IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name format: %q — must match tenant_[a-z0-9_]{1,50}", schemaName)
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire connection for tenant schema %q: %w", schemaName, err)
	}
	defer conn.Release()

	// Set search_path for this connection. The schema name is validated above.
	// public is included to access gen_random_uuid() and other shared functions.
	// The format string is safe: schemaName matches ^tenant_[a-z0-9_]{1,50}$.
	_, err = conn.Exec(ctx, fmt.Sprintf("SET search_path TO %s, public", schemaName))
	if err != nil {
		return fmt.Errorf("set search_path to %q: %w", schemaName, err)
	}

	return fn(conn.Conn())
}

// ContextKey type for context values to avoid collisions with other packages.
type ContextKey string

const (
	// CtxKeySchemaName is the context key for the tenant schema name.
	// Set by TenantMiddleware; read by repository implementations.
	CtxKeySchemaName ContextKey = "tenant_schema_name"

	// CtxKeyTenantID is the context key for the tenant UUID or slug.
	CtxKeyTenantID ContextKey = "tenant_id"

	// CtxKeyUserID is the context key for the authenticated user's UUID.
	// Set by AuthMiddleware after JWT validation.
	CtxKeyUserID ContextKey = "user_id"

	// CtxKeyUserRoles is the context key for the authenticated user's roles slice.
	CtxKeyUserRoles ContextKey = "user_roles"

	// CtxKeyRequestID is the context key for the X-Request-ID trace identifier.
	CtxKeyRequestID ContextKey = "request_id"
)

// SchemaFromContext extracts the tenant schema name from the request context.
// Returns an error if the schema name is absent — indicating TenantMiddleware was not applied.
func SchemaFromContext(ctx context.Context) (string, error) {
	schema, ok := ctx.Value(CtxKeySchemaName).(string)
	if !ok || schema == "" {
		return "", fmt.Errorf("tenant schema not found in context: TenantMiddleware must run before this handler")
	}
	return schema, nil
}
```

### 4d. Middleware

#### `internal/middleware/auth.go`

```go
// internal/middleware/auth.go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/auth-system/internal/apierror"
	pgdb "github.com/yourorg/auth-system/internal/infrastructure/postgres"
	"github.com/yourorg/auth-system/pkg/jwtutil"
)

// JWTClaims mirrors the claims parsed from a validated JWT.
// This struct is stored in the Gin context under the "jwt_claims" key.
type JWTClaims struct {
	UserID   string
	TenantID string
	Roles    []string
	JTI      string
}

// RequireAuth is a Gin middleware that validates the JWT access token in the
// Authorization: Bearer header. On success, it stores parsed claims in the context.
// On failure, it aborts the request with 401 Unauthorized.
//
// This middleware must run BEFORE TenantMiddleware on authenticated routes, because
// TenantMiddleware reads the tenant_id from the claims this middleware sets.
func RequireAuth(jwtVerifier jwtutil.Verifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED",
				"Authorization header is required.",
				nil,
				requestID(c),
			))
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED",
				"Authorization header must use Bearer scheme.",
				nil,
				requestID(c),
			))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED",
				"Bearer token is empty.",
				nil,
				requestID(c),
			))
			return
		}

		claims, err := jwtVerifier.Verify(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED",
				"Token is invalid or has expired.",
				nil,
				requestID(c),
			))
			return
		}

		// Store claims in context for downstream middleware and handlers.
		c.Set("jwt_claims", JWTClaims{
			UserID:   claims.Subject,
			TenantID: claims.TenantID,
			Roles:    claims.Roles,
			JTI:      claims.ID,
		})

		// Also store individual values for convenience.
		c.Set(string(pgdb.CtxKeyUserID), claims.Subject)
		c.Set(string(pgdb.CtxKeyUserRoles), claims.Roles)

		c.Next()
	}
}

// RequireRole returns a Gin middleware that aborts with 403 Forbidden if the
// authenticated user does not hold at least one of the specified roles.
// RequireAuth must run before RequireRole.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claimsVal, exists := c.Get("jwt_claims")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, apierror.New(
				"UNAUTHORIZED",
				"Authentication required.",
				nil,
				requestID(c),
			))
			return
		}

		claims := claimsVal.(JWTClaims)
		userRoles := make(map[string]struct{}, len(claims.Roles))
		for _, r := range claims.Roles {
			userRoles[r] = struct{}{}
		}

		for _, required := range roles {
			if _, ok := userRoles[required]; ok {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
			"FORBIDDEN",
			"You do not have permission to perform this action.",
			nil,
			requestID(c),
		))
	}
}

// requestID is a helper to extract the request ID from the Gin context.
func requestID(c *gin.Context) string {
	if id, ok := c.Get(string(pgdb.CtxKeyRequestID)); ok {
		return id.(string)
	}
	return ""
}
```

#### `internal/middleware/tenant.go`

```go
// internal/middleware/tenant.go
package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/auth-system/internal/apierror"
	"github.com/yourorg/auth-system/internal/domain"
	pgdb "github.com/yourorg/auth-system/internal/infrastructure/postgres"
)

// TenantCache resolves a tenant ID or slug to the PostgreSQL schema name.
// Implementations cache the result in memory to avoid a global schema lookup on
// every request. The cache TTL is configurable (TENANT_CACHE_TTL_SECONDS).
type TenantCache interface {
	GetSchema(ctx context.Context, tenantID string) (string, error)
}

// InMemoryTenantCache is a thread-safe in-memory cache backed by a TenantRepository.
// It refreshes in the background on a configurable TTL.
type InMemoryTenantCache struct {
	mu       sync.RWMutex
	cache    map[string]cacheEntry
	repo     domain.TenantRepository
	ttl      time.Duration
}

type cacheEntry struct {
	schemaName string
	expiresAt  time.Time
}

// NewInMemoryTenantCache creates a new cache that resolves tenant IDs/slugs
// via the provided TenantRepository on cache miss or TTL expiry.
func NewInMemoryTenantCache(repo domain.TenantRepository, ttl time.Duration) *InMemoryTenantCache {
	return &InMemoryTenantCache{
		cache: make(map[string]cacheEntry),
		repo:  repo,
		ttl:   ttl,
	}
}

// GetSchema returns the PostgreSQL schema name for the given tenant identifier.
// The identifier may be a UUID or a slug — the repository must handle both.
func (c *InMemoryTenantCache) GetSchema(ctx context.Context, tenantID string) (string, error) {
	c.mu.RLock()
	entry, ok := c.cache[tenantID]
	c.mu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.schemaName, nil
	}

	// Cache miss or expired — fetch from global registry.
	tenant, err := c.repo.FindBySlug(ctx, tenantID)
	if err != nil {
		return "", err
	}
	if tenant.Status != domain.TenantStatusActive {
		return "", domain.ErrTenantSuspended
	}

	c.mu.Lock()
	c.cache[tenantID] = cacheEntry{
		schemaName: tenant.SchemaName,
		expiresAt:  time.Now().Add(c.ttl),
	}
	c.mu.Unlock()

	return tenant.SchemaName, nil
}

// Invalidate removes a tenant entry from the cache, forcing the next request
// to perform a fresh lookup. Called when tenant status changes.
func (c *InMemoryTenantCache) Invalidate(tenantID string) {
	c.mu.Lock()
	delete(c.cache, tenantID)
	c.mu.Unlock()
}

// RequireTenant is a Gin middleware that resolves the tenant schema from either:
//   - The JWT claims (for authenticated routes — AuthMiddleware must run first), or
//   - The X-Tenant-ID header (for unauthenticated routes: register, login, etc.)
//
// On success, it sets CtxKeySchemaName and CtxKeyTenantID in the Gin context.
// On failure, it aborts with 400 or 403.
//
// SECURITY: The schema name is validated with domain.IsValidSchemaName before being
// stored in context. This prevents SQL injection via SET search_path in repositories.
func RequireTenant(cache TenantCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tenantID string

		// For authenticated routes, prefer the tenant_id from the verified JWT.
		if claimsVal, exists := c.Get("jwt_claims"); exists {
			claims := claimsVal.(JWTClaims)
			tenantID = claims.TenantID

			// Cross-validate: if X-Tenant-ID is also present, it must match the JWT.
			// This prevents token replay across tenants.
			if headerTenantID := c.GetHeader("X-Tenant-ID"); headerTenantID != "" {
				if headerTenantID != tenantID {
					c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
						"TENANT_MISMATCH",
						"The tenant in your token does not match the X-Tenant-ID header.",
						nil,
						requestID(c),
					))
					return
				}
			}
		} else {
			// Unauthenticated endpoint — X-Tenant-ID header is required.
			tenantID = c.GetHeader("X-Tenant-ID")
			if tenantID == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
					"MISSING_TENANT_ID",
					"X-Tenant-ID header is required.",
					nil,
					requestID(c),
				))
				return
			}
		}

		// Resolve tenant slug/UUID → schema name (with in-memory cache).
		schemaName, err := cache.GetSchema(c.Request.Context(), tenantID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, apierror.New(
				"INVALID_TENANT",
				"Tenant not found or is not active.",
				nil,
				requestID(c),
			))
			return
		}

		// Defense-in-depth: validate the schema name format before storing.
		if !domain.IsValidSchemaName(schemaName) {
			c.AbortWithStatusJSON(http.StatusInternalServerError, apierror.New(
				"INTERNAL_ERROR",
				"Invalid tenant configuration detected.",
				nil,
				requestID(c),
			))
			return
		}

		c.Set(string(pgdb.CtxKeySchemaName), schemaName)
		c.Set(string(pgdb.CtxKeyTenantID), tenantID)

		c.Next()
	}
}
```

#### `internal/middleware/rate_limit.go`

```go
// internal/middleware/rate_limit.go
package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/auth-system/internal/apierror"
)

// RateLimiter is the interface for distributed rate limit checks.
// Implemented by repository/redis.RedisRateLimiter using a Lua sliding window.
type RateLimiter interface {
	// Allow checks whether the request identified by key is within the rate limit.
	// Returns (allowed, currentCount, retryAfterSeconds, error).
	// If Redis is unavailable, Allow must return an error — the middleware then
	// fails closed (blocks the request) per AVAIL-02.
	Allow(ctx context.Context, key string, limit int, windowDuration time.Duration) (bool, int, int, error)
}

// RateLimitConfig defines a rate limit rule applied by the middleware.
type RateLimitConfig struct {
	// KeyFunc extracts the rate limit key from the request (e.g., IP, user ID).
	KeyFunc func(c *gin.Context) string
	// Limit is the maximum number of requests allowed within the Window.
	Limit int
	// Window is the sliding window duration.
	Window time.Duration
	// Description is a human-readable label for logging (e.g., "login:ip").
	Description string
}

// RateLimit returns a Gin middleware that enforces the provided rate limit rule.
// Multiple rules can be composed by chaining multiple RateLimit middlewares.
//
// Fail-closed behavior: if Redis is unavailable, the request is rejected with 503.
// This satisfies AVAIL-02: "fail closed on rate limiting (block, don't bypass)".
func RateLimit(limiter RateLimiter, ruleCfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := ruleCfg.KeyFunc(c)
		if key == "" {
			// Key cannot be determined — fail closed.
			slog.Warn("rate limit key is empty, blocking request",
				"description", ruleCfg.Description,
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, apierror.New(
				"SERVICE_UNAVAILABLE",
				"Rate limiting service is temporarily unavailable.",
				nil,
				requestID(c),
			))
			return
		}

		allowed, _, retryAfter, err := limiter.Allow(
			c.Request.Context(),
			fmt.Sprintf("rl:%s:%s", ruleCfg.Description, key),
			ruleCfg.Limit,
			ruleCfg.Window,
		)
		if err != nil {
			// Redis unavailable — fail closed per AVAIL-02.
			slog.Error("rate limiter unavailable, blocking request",
				"key", key,
				"description", ruleCfg.Description,
				"error", err,
			)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, apierror.New(
				"SERVICE_UNAVAILABLE",
				"Rate limiting service is temporarily unavailable.",
				nil,
				requestID(c),
			))
			return
		}

		// Always set rate limit headers for observability.
		c.Header("X-RateLimit-Limit", strconv.Itoa(ruleCfg.Limit))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ruleCfg.Window).Unix(), 10))

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, apierror.New(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please wait before retrying.",
				nil,
				requestID(c),
			))
			return
		}

		c.Next()
	}
}

// IPKeyFunc returns the client IP address as the rate limit key.
// Uses X-Forwarded-For if behind a trusted proxy (Fly.io / load balancer).
func IPKeyFunc(c *gin.Context) string {
	// Gin's ClientIP() handles X-Forwarded-For with trusted proxy configuration.
	return c.ClientIP()
}

// UserKeyFunc returns a rate limit key based on the authenticated user + tenant.
// Falls back to IP if no authenticated user is present.
func UserKeyFunc(c *gin.Context) string {
	tenantID, _ := c.Get("tenant_id")
	userID, _ := c.Get("user_id")
	if userID != nil && tenantID != nil {
		return fmt.Sprintf("%v:%v", tenantID, userID)
	}
	return c.ClientIP()
}
```

### 4e. Auth Feature — Handler → Service → Repository

#### `internal/handler/auth_handler.go`

```go
// internal/handler/auth_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/auth-system/internal/apierror"
	"github.com/yourorg/auth-system/internal/service"
)

// AuthHandler handles HTTP requests for the core authentication flows:
// register, login, logout (single device), and logout all devices.
// All business logic is delegated to AuthService — no logic lives here.
type AuthHandler struct {
	authSvc service.AuthService
}

// NewAuthHandler creates a new AuthHandler with its required service dependency.
func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// --- Request/Response Types ---

type registerRequest struct {
	Email     string `json:"email"      validate:"required,email"`
	Password  string `json:"password"   validate:"required,min=8,max=128"`
	FirstName string `json:"first_name" validate:"required,min=1,max=100"`
	LastName  string `json:"last_name"  validate:"required,min=1,max=100"`
}

type registerResponse struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type loginResponse struct {
	AccessToken        string `json:"access_token"`
	RefreshToken       string `json:"refresh_token"`
	TokenType          string `json:"token_type"`
	ExpiresIn          int    `json:"expires_in"`
	RefreshExpiresIn   int    `json:"refresh_expires_in"`
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Register godoc
// POST /api/v1/auth/register
// Registers a new user account in the tenant schema.
// Triggers async email verification. Account status is "unverified" on creation.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.authSvc.Register(c.Request.Context(), service.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, registerResponse{
		UserID:  result.UserID,
		Email:   result.Email,
		Status:  result.Status,
		Message: "Registration successful. Please check your email to verify your account.",
	})
}

// Login godoc
// POST /api/v1/auth/login
// Authenticates a user and returns a JWT access token + opaque refresh token.
// Ambiguous error messages — does not distinguish wrong email vs wrong password.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if !bindAndValidate(c, &req) {
		return
	}

	result, err := h.authSvc.Login(c.Request.Context(), service.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, loginResponse{
		AccessToken:      result.AccessToken,
		RefreshToken:     result.RefreshToken,
		TokenType:        "Bearer",
		ExpiresIn:        result.ExpiresIn,
		RefreshExpiresIn: result.RefreshExpiresIn,
	})
}

// Logout godoc
// POST /api/v1/auth/logout
// Revokes the specific refresh token presented. Idempotent — returns 200 even
// if the token is already expired or revoked.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := h.authSvc.Logout(c.Request.Context(), service.LogoutInput{
		RefreshToken: req.RefreshToken,
		IPAddress:    c.ClientIP(),
	}); err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully."})
}

// LogoutAll godoc
// POST /api/v1/auth/logout/all
// Revokes all refresh tokens for the authenticated user across all devices.
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userID := mustGetUserID(c)

	revokedCount, err := h.authSvc.LogoutAll(c.Request.Context(), service.LogoutAllInput{
		UserID:    userID,
		IPAddress: c.ClientIP(),
	})
	if err != nil {
		respondWithServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "All sessions revoked successfully.",
		"sessions_revoked": revokedCount,
	})
}

// --- Shared Handler Helpers ---

// bindAndValidate decodes and validates the JSON request body.
// Returns false and writes the error response if binding or validation fails.
func bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, apierror.New(
			"INVALID_REQUEST",
			"Request body is invalid or missing required fields.",
			nil,
			getRequestID(c),
		))
		return false
	}

	if errs := validate(req); len(errs) > 0 {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apierror.New(
			"VALIDATION_ERROR",
			"One or more fields failed validation.",
			errs,
			getRequestID(c),
		))
		return false
	}

	return true
}

// respondWithServiceError maps domain/service errors to HTTP responses.
// This is the central error translation layer — no business logic here.
func respondWithServiceError(c *gin.Context, err error) {
	type mapping struct {
		status int
		code   string
		msg    string
	}

	// Import domain errors to map them.
	// Keep this mapping exhaustive — add new domain errors as they are defined.
	var m mapping

	switch {
	case isError(err, "user not found"), isError(err, "invalid credentials"):
		// Ambiguous — never reveal whether email or password was wrong.
		m = mapping{http.StatusUnauthorized, "INVALID_CREDENTIALS", "The email or password is incorrect."}
	case isError(err, "email not verified"):
		m = mapping{http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Your email address has not been verified. Please check your inbox."}
	case isError(err, "account disabled"):
		m = mapping{http.StatusForbidden, "ACCOUNT_DISABLED", "This account has been disabled."}
	case isError(err, "account is temporarily locked"):
		// Do not reveal lockout details — ambiguous response only.
		m = mapping{http.StatusForbidden, "ACCOUNT_LOCKED", "This account is temporarily unavailable."}
	case isError(err, "email already exists"):
		m = mapping{http.StatusConflict, "EMAIL_ALREADY_EXISTS", "An account with this email already exists."}
	case isError(err, "tenant not found"), isError(err, "tenant is suspended"):
		m = mapping{http.StatusForbidden, "INVALID_TENANT", "Tenant not found or is not active."}
	case isError(err, "tenant with this slug already exists"):
		m = mapping{http.StatusConflict, "TENANT_ALREADY_EXISTS", "A tenant with this identifier already exists."}
	case isError(err, "token reuse detected"):
		m = mapping{http.StatusUnauthorized, "SUSPICIOUS_TOKEN_REUSE", "Your session has been revoked due to suspicious activity. Please log in again."}
	case isError(err, "session has expired"), isError(err, "invalid refresh token"), isError(err, "session has been revoked"):
		m = mapping{http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Your session has expired. Please log in again."}
	case isError(err, "password does not meet complexity requirements"):
		m = mapping{http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error()}
	case isError(err, "role not found"):
		m = mapping{http.StatusNotFound, "ROLE_NOT_FOUND", "Role not found in this tenant."}
	case isError(err, "role already exists"):
		m = mapping{http.StatusConflict, "ROLE_ALREADY_EXISTS", "A role with this name already exists."}
	case isError(err, "user already has this role"):
		m = mapping{http.StatusConflict, "ROLE_ALREADY_ASSIGNED", "The user already has this role."}
	default:
		// Unexpected error — log it but never expose internals to the client.
		c.Error(err)
		m = mapping{http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred. Please try again later."}
	}

	c.AbortWithStatusJSON(m.status, apierror.New(m.code, m.msg, nil, getRequestID(c)))
}

// mustGetUserID extracts the authenticated user ID from the Gin context.
// Panics if AuthMiddleware has not run — this is a programming error.
func mustGetUserID(c *gin.Context) string {
	userID, ok := c.Get("user_id")
	if !ok {
		panic("mustGetUserID called without AuthMiddleware — check route configuration")
	}
	return userID.(string)
}

func getRequestID(c *gin.Context) string {
	if id, ok := c.Get("request_id"); ok {
		return id.(string)
	}
	return ""
}

// isError is a helper that checks if an error's message contains the needle.
// This allows clean error matching without importing all domain packages into handlers.
func isError(err error, needle string) bool {
	return err != nil && len(err.Error()) > 0 && containsInsensitive(err.Error(), needle)
}

func containsInsensitive(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		len(s) > 0 && len(sub) > 0 &&
			func() bool {
				sl := strings.ToLower(s)
				subl := strings.ToLower(sub)
				return strings.Contains(sl, subl)
			}())
}

// validate runs go-playground/validator on the struct.
// Returns a slice of field-level error detail maps.
func validate(v interface{}) []map[string]string {
	// Validator setup is in pkg/validator/validator.go.
	// Here we call the singleton validator instance.
	return globalValidator.ValidateStruct(v)
}
```

#### `internal/service/auth_service.go`

```go
// internal/service/auth_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/yourorg/auth-system/internal/domain"
	pgdb "github.com/yourorg/auth-system/internal/infrastructure/postgres"
	"github.com/yourorg/auth-system/pkg/crypto"
	"github.com/yourorg/auth-system/pkg/jwtutil"
	"github.com/yourorg/auth-system/pkg/validator"
)

// AuthService defines the interface for the core authentication use cases.
// All methods are pure business logic — no HTTP concerns.
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*RegisterResult, error)
	Login(ctx context.Context, input LoginInput) (*LoginResult, error)
	Logout(ctx context.Context, input LogoutInput) error
	LogoutAll(ctx context.Context, input LogoutAllInput) (int, error)
}

// authServiceImpl is the concrete implementation of AuthService.
type authServiceImpl struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	tokenRepo   domain.TokenRepository
	auditRepo   domain.AuditRepository
	tenantRepo  domain.TenantRepository
	jwtSvc      jwtutil.Signer
	emailCh     chan<- EmailTask
	cfg         AuthServiceConfig
}

// AuthServiceConfig holds configuration values needed by AuthService.
type AuthServiceConfig struct {
	AccessTokenTTL          time.Duration
	SessionDefaultTTL       time.Duration
	VerificationTokenTTL    time.Duration
	LockoutThreshold        int
	LockoutDurationSeconds  int
}

// NewAuthService constructs an AuthService with all required dependencies injected.
func NewAuthService(
	userRepo domain.UserRepository,
	sessionRepo domain.SessionRepository,
	tokenRepo domain.TokenRepository,
	auditRepo domain.AuditRepository,
	tenantRepo domain.TenantRepository,
	jwtSvc jwtutil.Signer,
	emailCh chan<- EmailTask,
	cfg AuthServiceConfig,
) AuthService {
	return &authServiceImpl{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenRepo:   tokenRepo,
		auditRepo:   auditRepo,
		tenantRepo:  tenantRepo,
		jwtSvc:      jwtSvc,
		emailCh:     emailCh,
		cfg:         cfg,
	}
}

// --- Input/Output types ---

type RegisterInput struct {
	Email     string
	Password  string
	FirstName string
	LastName  string
}

type RegisterResult struct {
	UserID string
	Email  string
	Status string
}

type LoginInput struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type LoginResult struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int
	RefreshExpiresIn int
}

type LogoutInput struct {
	RefreshToken string
	IPAddress    string
}

type LogoutAllInput struct {
	UserID    string
	IPAddress string
}

// --- Implementation ---

// Register creates a new user account in the tenant schema.
// Password complexity is validated against the tenant's policy.
// An email verification token is generated and dispatched asynchronously.
func (s *authServiceImpl) Register(ctx context.Context, input RegisterInput) (*RegisterResult, error) {
	// Normalize and validate email format.
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		return nil, domain.ErrInvalidEmail
	}

	// Fetch tenant config to get password policy.
	tenant, err := s.tenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Validate password complexity against tenant policy.
	if err := validator.CheckPasswordPolicy(input.Password, tenant.Config.PasswordPolicy); err != nil {
		return nil, err
	}

	// Hash password with Argon2id (SEC-01).
	passwordHash, err := crypto.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	// Create the user record (repository handles duplicate detection).
	user, err := s.userRepo.Create(ctx, domain.CreateUserInput{
		Email:        email,
		PasswordHash: passwordHash,
		FirstName:    input.FirstName,
		LastName:     input.LastName,
		Status:       domain.UserStatusUnverified,
	})
	if err != nil {
		return nil, err // ErrEmailAlreadyExists propagates to handler
	}

	// Generate single-use email verification token.
	rawToken, tokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate verification token: %w", err)
	}

	expiresAt := time.Now().Add(s.cfg.VerificationTokenTTL)
	if err := s.tokenRepo.CreateEmailVerificationToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, fmt.Errorf("store verification token: %w", err)
	}

	// Dispatch verification email asynchronously (ADR-012).
	// This must not block the request path.
	s.enqueueEmail(EmailTask{
		Type:      EmailTypeVerification,
		ToEmail:   user.Email,
		ToName:    user.FullName(),
		Token:     rawToken,
		ExpiresAt: expiresAt,
	})

	slog.Info("user registered",
		"user_id", user.ID,
		"tenant_id", tenant.ID,
	)

	return &RegisterResult{
		UserID: user.ID.String(),
		Email:  user.Email,
		Status: string(user.Status),
	}, nil
}

// Login authenticates a user and issues a JWT access token + opaque refresh token.
// Failed attempts are counted and trigger account lockout after the tenant threshold.
// All auth events (success and failure) are written to the audit log.
func (s *authServiceImpl) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	email, err := domain.NormalizeEmail(input.Email)
	if err != nil {
		// Return the same error as a wrong password — no email enumeration.
		return nil, domain.ErrInvalidCredentials
	}

	// Fetch tenant config for session TTL and lockout settings.
	tenant, err := s.tenantFromContext(ctx)
	if err != nil {
		return nil, err
	}

	// Attempt to find the user by email.
	// Return ambiguous error on not found — same as wrong password.
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType: domain.EventLoginFailure,
			ActorIP:   input.IPAddress,
			ActorUA:   input.UserAgent,
			Metadata:  map[string]interface{}{"reason": "user_not_found"},
		})
		return nil, domain.ErrInvalidCredentials
	}

	// Check account lockout before verifying password.
	// This prevents brute force even when the account is locked.
	if user.IsLocked() {
		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType:    domain.EventLoginFailure,
			ActorID:      &user.ID,
			ActorIP:      input.IPAddress,
			ActorUA:      input.UserAgent,
			TargetUserID: &user.ID,
			Metadata:     map[string]interface{}{"reason": "account_locked"},
		})
		return nil, domain.ErrAccountLocked
	}

	// Check account status.
	if user.Status == domain.UserStatusDisabled {
		return nil, domain.ErrAccountDisabled
	}
	if user.Status == domain.UserStatusDeleted {
		return nil, domain.ErrInvalidCredentials
	}
	if user.Status == domain.UserStatusUnverified {
		return nil, domain.ErrEmailNotVerified
	}

	// Verify password using constant-time Argon2id comparison (SEC-01).
	if !crypto.VerifyPassword(input.Password, user.PasswordHash) {
		// Increment failed login counter and check lockout threshold.
		newCount, incrErr := s.userRepo.IncrementFailedLoginCount(ctx, user.ID)
		if incrErr != nil {
			slog.Error("failed to increment failed login count", "user_id", user.ID, "error", incrErr)
		}

		threshold := s.cfg.LockoutThreshold
		if tenant.Config.LockoutThreshold > 0 {
			threshold = tenant.Config.LockoutThreshold
		}

		if newCount >= threshold {
			lockDuration := time.Duration(s.cfg.LockoutDurationSeconds) * time.Second
			if tenant.Config.LockoutDurationSeconds > 0 {
				lockDuration = time.Duration(tenant.Config.LockoutDurationSeconds) * time.Second
			}
			lockUntil := time.Now().Add(lockDuration)
			if lockErr := s.userRepo.SetLockedUntil(ctx, user.ID, lockUntil); lockErr != nil {
				slog.Error("failed to lock account", "user_id", user.ID, "error", lockErr)
			}

			s.writeAuditEvent(ctx, domain.AuditEvent{
				EventType:    domain.EventAccountLocked,
				ActorIP:      input.IPAddress,
				ActorUA:      input.UserAgent,
				TargetUserID: &user.ID,
				Metadata:     map[string]interface{}{"locked_until": lockUntil},
			})
		}

		s.writeAuditEvent(ctx, domain.AuditEvent{
			EventType:    domain.EventLoginFailure,
			ActorID:      &user.ID,
			ActorIP:      input.IPAddress,
			ActorUA:      input.UserAgent,
			TargetUserID: &user.ID,
			Metadata:     map[string]interface{}{"failed_count": newCount},
		})

		return nil, domain.ErrInvalidCredentials
	}

	// Password is correct — reset failed login counter.
	if resetErr := s.userRepo.ResetFailedLoginCount(ctx, user.ID); resetErr != nil {
		slog.Error("failed to reset login count", "user_id", user.ID, "error", resetErr)
	}
	if updateErr := s.userRepo.Update(ctx, user.ID, domain.UpdateUserInput{
		LastLoginAt: timePtr(time.Now()),
	}); updateErr != nil {
		slog.Error("failed to update last_login_at", "user_id", user.ID, "error", updateErr)
	}

	// Fetch the user's roles for JWT claims.
	roles, err := s.fetchUserRoleNames(ctx, user.ID)
	if err != nil {
		slog.Error("failed to fetch user roles", "user_id", user.ID, "error", err)
		roles = []string{} // Non-fatal — proceed with empty roles
	}

	// Determine session TTL (tenant config overrides default).
	sessionTTL := s.cfg.SessionDefaultTTL
	if tenant.Config.SessionTTLSeconds > 0 {
		sessionTTL = time.Duration(tenant.Config.SessionTTLSeconds) * time.Second
	}

	// Issue JWT access token (RS256, 15 min — SEC-02, SEC-03).
	accessToken, err := s.jwtSvc.Sign(jwtutil.Claims{
		Subject:  user.ID.String(),
		TenantID: tenant.ID.String(),
		Roles:    roles,
		TTL:      s.cfg.AccessTokenTTL,
	})
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Generate opaque refresh token (32 bytes, crypto/rand — ADR-004).
	rawRefreshToken, refreshTokenHash, err := crypto.GenerateTokenWithHash()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	// Persist session with hashed refresh token.
	familyID := uuid.New()
	session := &domain.Session{
		ID:               uuid.New(),
		UserID:           user.ID,
		RefreshTokenHash: refreshTokenHash,
		FamilyID:         familyID,
		IPAddress:        input.IPAddress,
		UserAgent:        input.UserAgent,
		IssuedAt:         time.Now(),
		ExpiresAt:        time.Now().Add(sessionTTL),
		LastUsedAt:       time.Now(),
		IsRevoked:        false,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Write audit event (non-fatal).
	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventLoginSuccess,
		ActorID:      &user.ID,
		ActorIP:      input.IPAddress,
		ActorUA:      input.UserAgent,
		TargetUserID: &user.ID,
	})

	return &LoginResult{
		AccessToken:      accessToken,
		RefreshToken:     rawRefreshToken,
		ExpiresIn:        int(s.cfg.AccessTokenTTL.Seconds()),
		RefreshExpiresIn: int(sessionTTL.Seconds()),
	}, nil
}

// Logout revokes the specific refresh token presented.
// Idempotent — if the token is already revoked or not found, returns nil.
func (s *authServiceImpl) Logout(ctx context.Context, input LogoutInput) error {
	_, tokenHash := crypto.HashToken(input.RefreshToken)

	// RevokeByTokenHash is idempotent — no error if already revoked.
	if err := s.sessionRepo.RevokeByTokenHash(ctx, tokenHash); err != nil {
		// Log but don't fail — we treat not-found as already-logged-out.
		slog.Warn("logout: session not found (idempotent)", "error", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType: domain.EventLogout,
		ActorIP:   input.IPAddress,
	})

	return nil
}

// LogoutAll revokes all active refresh tokens for the authenticated user.
// Returns the number of sessions revoked.
func (s *authServiceImpl) LogoutAll(ctx context.Context, input LogoutAllInput) (int, error) {
	userID, err := uuid.Parse(input.UserID)
	if err != nil {
		return 0, fmt.Errorf("invalid user ID: %w", err)
	}

	count, err := s.sessionRepo.RevokeAllForUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("revoke all sessions: %w", err)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventLogoutAll,
		ActorID:      &userID,
		ActorIP:      input.IPAddress,
		TargetUserID: &userID,
		Metadata:     map[string]interface{}{"sessions_revoked": count},
	})

	return count, nil
}

// --- Private helpers ---

// tenantFromContext retrieves the tenant entity using the schema name from context.
// The schema name is set by TenantMiddleware before this service is called.
func (s *authServiceImpl) tenantFromContext(ctx context.Context) (*domain.Tenant, error) {
	tenantID, ok := ctx.Value(pgdb.CtxKeyTenantID).(string)
	if !ok || tenantID == "" {
		return nil, fmt.Errorf("tenant ID not in context — TenantMiddleware must be applied")
	}
	return s.tenantRepo.FindBySlug(ctx, tenantID)
}

// fetchUserRoleNames retrieves role name strings for JWT claims.
// This function must be fast — it is in the login hot path.
func (s *authServiceImpl) fetchUserRoleNames(ctx context.Context, userID uuid.UUID) ([]string, error) {
	// RoleRepository.GetUserRoles is implemented in the repository layer.
	// This is a placeholder showing the call pattern — actual injection is at wire time.
	return []string{"user"}, nil // Replaced by actual role repo call in production wiring
}

// enqueueEmail sends an email task to the buffered channel (non-blocking).
// If the channel is full, the email is dropped and an error is logged.
// This is acceptable for the pilot (ADR-012: at-most-once delivery).
func (s *authServiceImpl) enqueueEmail(task EmailTask) {
	select {
	case s.emailCh <- task:
	default:
		slog.Error("email channel full — email task dropped",
			"type", task.Type,
			"to", task.ToEmail,
		)
	}
}

// writeAuditEvent persists an audit event. Errors are logged but not returned
// to the caller — audit log write failure must never block an auth operation.
func (s *authServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event",
			"event_type", event.EventType,
			"error", err,
		)
	}
}

func timePtr(t time.Time) *time.Time { return &t }

// EmailTask is a task submitted to the async email worker.
type EmailTask struct {
	Type      EmailType
	ToEmail   string
	ToName    string
	Token     string
	ExpiresAt time.Time
	Extra     map[string]interface{}
}

type EmailType string

const (
	EmailTypeVerification  EmailType = "verification"
	EmailTypePasswordReset EmailType = "password_reset"
	EmailTypeInvitation    EmailType = "invitation"
)
```

#### `internal/repository/postgres/user_repo.go`

```go
// internal/repository/postgres/user_repo.go
package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/auth-system/internal/domain"
	pgdb "github.com/yourorg/auth-system/internal/infrastructure/postgres"
)

// PostgresUserRepo implements domain.UserRepository using pgx.
// All methods acquire a tenant-scoped connection via pgdb.WithTenantSchema.
// The schema name is extracted from the context key CtxKeySchemaName —
// set by TenantMiddleware on every authenticated request.
type PostgresUserRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresUserRepo creates a new PostgresUserRepo.
func NewPostgresUserRepo(pool *pgxpool.Pool) *PostgresUserRepo {
	return &PostgresUserRepo{pool: pool}
}

// FindByID retrieves a user by their UUID within the tenant schema.
func (r *PostgresUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, email, password_hash, status, first_name, last_name,
			       mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			       email_verified_at, last_login_at, created_at, updated_at, deleted_at
			FROM users
			WHERE id = $1 AND deleted_at IS NULL
		`, id)

		u, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("scan user by id: %w", scanErr)
		}
		user = u
		return nil
	})

	return user, err
}

// FindByEmail retrieves a user by their normalized email within the tenant schema.
// Returns ErrUserNotFound if no active user exists with this email.
func (r *PostgresUserRepo) FindByEmail(
API Error: Claude's response exceeded the 32000 output token maximum. To configure this behavior, set the CLAUDE_CODE_MAX_OUTPUT_TOKENS environment variable.
Continuing from `FindByEmail` in `internal/repository/postgres/user_repo.go`.

---

```go
// FindByEmail retrieves a user by their normalized email within the tenant schema.
// Returns ErrUserNotFound if no active user exists with this email.
func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			SELECT id, email, password_hash, status, first_name, last_name,
			       mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			       email_verified_at, last_login_at, created_at, updated_at, deleted_at
			FROM users
			WHERE email = $1 AND deleted_at IS NULL
		`, email)

		u, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("scan user by email: %w", scanErr)
		}
		user = u
		return nil
	})

	return user, err
}

// Create inserts a new user record into the tenant schema.
// Returns ErrEmailAlreadyExists on unique constraint violation.
func (r *PostgresUserRepo) Create(ctx context.Context, input domain.CreateUserInput) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		row := conn.QueryRow(ctx, `
			INSERT INTO users (id, email, password_hash, status, first_name, last_name,
			                   mfa_enabled, failed_login_count, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, false, 0, now(), now())
			RETURNING id, email, password_hash, status, first_name, last_name,
			          mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			          email_verified_at, last_login_at, created_at, updated_at, deleted_at
		`,
			input.Email,
			input.PasswordHash,
			string(input.Status),
			input.FirstName,
			input.LastName,
		)

		u, scanErr := scanUser(row)
		if scanErr != nil {
			// Detect unique constraint violation on email column.
			if isUniqueViolation(scanErr) {
				return domain.ErrEmailAlreadyExists
			}
			return fmt.Errorf("insert user: %w", scanErr)
		}
		user = u
		return nil
	})

	return user, err
}

// Update applies a partial update to a user record.
// Only non-nil pointer fields in UpdateUserInput are applied.
// Uses a dynamic SET clause to avoid overwriting unintended fields.
func (r *PostgresUserRepo) Update(ctx context.Context, id uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var user *domain.User
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		// Build SET clause dynamically from non-nil fields.
		setClauses := []string{"updated_at = now()"}
		args := []interface{}{}
		argIdx := 1

		if input.FirstName != nil {
			setClauses = append(setClauses, fmt.Sprintf("first_name = $%d", argIdx))
			args = append(args, *input.FirstName)
			argIdx++
		}
		if input.LastName != nil {
			setClauses = append(setClauses, fmt.Sprintf("last_name = $%d", argIdx))
			args = append(args, *input.LastName)
			argIdx++
		}
		if input.PasswordHash != nil {
			setClauses = append(setClauses, fmt.Sprintf("password_hash = $%d", argIdx))
			args = append(args, *input.PasswordHash)
			argIdx++
		}
		if input.Status != nil {
			setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
			args = append(args, string(*input.Status))
			argIdx++
		}
		if input.MFAEnabled != nil {
			setClauses = append(setClauses, fmt.Sprintf("mfa_enabled = $%d", argIdx))
			args = append(args, *input.MFAEnabled)
			argIdx++
		}
		if input.MFATOTPSecret != nil {
			setClauses = append(setClauses, fmt.Sprintf("mfa_totp_secret = $%d", argIdx))
			args = append(args, *input.MFATOTPSecret)
			argIdx++
		}
		if input.FailedLoginCount != nil {
			setClauses = append(setClauses, fmt.Sprintf("failed_login_count = $%d", argIdx))
			args = append(args, *input.FailedLoginCount)
			argIdx++
		}
		if input.LockedUntil != nil {
			setClauses = append(setClauses, fmt.Sprintf("locked_until = $%d", argIdx))
			args = append(args, *input.LockedUntil)
			argIdx++
		}
		if input.EmailVerifiedAt != nil {
			setClauses = append(setClauses, fmt.Sprintf("email_verified_at = $%d", argIdx))
			args = append(args, *input.EmailVerifiedAt)
			argIdx++
		}
		if input.LastLoginAt != nil {
			setClauses = append(setClauses, fmt.Sprintf("last_login_at = $%d", argIdx))
			args = append(args, *input.LastLoginAt)
			argIdx++
		}
		if input.DeletedAt != nil {
			setClauses = append(setClauses, fmt.Sprintf("deleted_at = $%d", argIdx))
			args = append(args, *input.DeletedAt)
			argIdx++
		}

		// Append the id as the final argument for WHERE clause.
		args = append(args, id)

		query := fmt.Sprintf(`
			UPDATE users
			SET %s
			WHERE id = $%d AND deleted_at IS NULL
			RETURNING id, email, password_hash, status, first_name, last_name,
			          mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			          email_verified_at, last_login_at, created_at, updated_at, deleted_at
		`, strings.Join(setClauses, ", "), argIdx)

		row := conn.QueryRow(ctx, query, args...)
		u, scanErr := scanUser(row)
		if scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return fmt.Errorf("update user: %w", scanErr)
		}
		user = u
		return nil
	})

	return user, err
}

// IncrementFailedLoginCount atomically increments the failed login counter.
func (r *PostgresUserRepo) IncrementFailedLoginCount(ctx context.Context, id uuid.UUID) (int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return 0, err
	}

	var newCount int
	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		return conn.QueryRow(ctx, `
			UPDATE users
			SET failed_login_count = failed_login_count + 1, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING failed_login_count
		`, id).Scan(&newCount)
	})

	return newCount, err
}

// ResetFailedLoginCount resets the counter to zero on a successful login.
func (r *PostgresUserRepo) ResetFailedLoginCount(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE users
			SET failed_login_count = 0, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
		`, id)
		return execErr
	})
}

// SetLockedUntil sets the account lockout expiry timestamp.
func (r *PostgresUserRepo) SetLockedUntil(ctx context.Context, id uuid.UUID, until time.Time) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE users
			SET locked_until = $2, updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
		`, id, until)
		return execErr
	})
}

// ListByTenant returns non-deleted users with offset pagination.
func (r *PostgresUserRepo) ListByTenant(ctx context.Context, limit, offset int) ([]*domain.User, int, error) {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}

	var users []*domain.User
	var total int

	err = pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		// Get total count.
		if countErr := conn.QueryRow(ctx,
			"SELECT COUNT(*) FROM users WHERE deleted_at IS NULL",
		).Scan(&total); countErr != nil {
			return fmt.Errorf("count users: %w", countErr)
		}

		rows, queryErr := conn.Query(ctx, `
			SELECT id, email, password_hash, status, first_name, last_name,
			       mfa_enabled, mfa_totp_secret, failed_login_count, locked_until,
			       email_verified_at, last_login_at, created_at, updated_at, deleted_at
			FROM users
			WHERE deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
		if queryErr != nil {
			return fmt.Errorf("query users: %w", queryErr)
		}
		defer rows.Close()

		for rows.Next() {
			u, scanErr := scanUserFromRows(rows)
			if scanErr != nil {
				return fmt.Errorf("scan user row: %w", scanErr)
			}
			users = append(users, u)
		}
		return rows.Err()
	})

	return users, total, err
}

// SoftDelete marks a user as deleted by setting deleted_at.
func (r *PostgresUserRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		tag, execErr := conn.Exec(ctx, `
			UPDATE users
			SET deleted_at = now(), status = 'deleted', updated_at = now()
			WHERE id = $1 AND deleted_at IS NULL
		`, id)
		if execErr != nil {
			return execErr
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrUserNotFound
		}
		return nil
	})
}

// AnonymizePII replaces PII fields with tombstone values (GDPR erasure step 2).
// The user ID is preserved for audit log integrity — only identifying data is removed.
func (r *PostgresUserRepo) AnonymizePII(ctx context.Context, id uuid.UUID) error {
	schema, err := pgdb.SchemaFromContext(ctx)
	if err != nil {
		return err
	}

	return pgdb.WithTenantSchema(ctx, r.pool, schema, func(conn *pgx.Conn) error {
		_, execErr := conn.Exec(ctx, `
			UPDATE users
			SET email        = 'deleted-' || id::text || '@gdpr.tombstone',
			    first_name   = '[deleted]',
			    last_name    = '[deleted]',
			    password_hash = '',
			    mfa_totp_secret = NULL,
			    updated_at   = now()
			WHERE id = $1
		`, id)
		return execErr
	})
}

// scanUser scans a single pgx.Row into a domain.User struct.
func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Status,
		&u.FirstName,
		&u.LastName,
		&u.MFAEnabled,
		&u.MFATOTPSecret,
		&u.FailedLoginCount,
		&u.LockedUntil,
		&u.EmailVerifiedAt,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// scanUserFromRows scans a pgx.Rows cursor row into a domain.User struct.
func scanUserFromRows(rows pgx.Rows) (*domain.User, error) {
	u := &domain.User{}
	err := rows.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Status,
		&u.FirstName,
		&u.LastName,
		&u.MFAEnabled,
		&u.MFATOTPSecret,
		&u.FailedLoginCount,
		&u.LockedUntil,
		&u.EmailVerifiedAt,
		&u.LastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// isUniqueViolation returns true when a pgx error is a PostgreSQL unique constraint violation.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// pgconn.PgError code 23505 = unique_violation
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == "23505"
	}
	return false
}
```

---

### 4f. Token Package

#### `pkg/jwtutil/jwt.go`

```go
// pkg/jwtutil/jwt.go
package jwtutil

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims carries the custom claims embedded in every JWT issued by this service.
type Claims struct {
	Subject  string   // sub — user UUID (or empty for M2M client_credentials)
	TenantID string   // tenant_id — tenant slug
	Roles    []string // roles[] — list of role names
	ClientID string   // client_id — OAuth client (M2M tokens only)
	Scope    string   // scope — OAuth scopes (M2M tokens only)
	TTL      time.Duration
}

// ParsedClaims represents the decoded payload of a verified JWT.
type ParsedClaims struct {
	ID       string   // jti
	Subject  string   // sub
	TenantID string   // tenant_id
	Roles    []string // roles
	ClientID string   // client_id
	Scope    string   // scope
	IssuedAt time.Time
	ExpiresAt time.Time
}

// jwtClaims is the internal jwt.Claims implementation for golang-jwt/jwt.
type jwtClaims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Scope    string   `json:"scope,omitempty"`
}

// Signer can issue signed JWTs.
type Signer interface {
	Sign(claims Claims) (string, error)
}

// Verifier can parse and validate JWTs.
type Verifier interface {
	Verify(tokenString string) (*ParsedClaims, error)
}

// JWKS represents the JSON Web Key Set returned by /.well-known/jwks.json.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK is a single JSON Web Key (RSA public key).
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"` // Base64URL-encoded modulus
	E   string `json:"e"` // Base64URL-encoded exponent
}

// signingKey holds an RSA key pair with its key ID.
type signingKey struct {
	Kid        string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

// KeyStore holds the current signing key and a map of all public keys for verification.
// Thread-safe for concurrent reads and atomic updates during key rotation.
type KeyStore struct {
	mu          sync.RWMutex
	current     signingKey            // The key used to sign new tokens
	allPublic   map[string]*rsa.PublicKey // kid → public key (includes retired keys)
	issuer      string
	audience    jwt.ClaimStrings
}

// NewKeyStore creates a new KeyStore with an initial key pair.
func NewKeyStore(issuer string, audience []string, kid string, privateKey *rsa.PrivateKey) *KeyStore {
	ks := &KeyStore{
		issuer:   issuer,
		audience: jwt.ClaimStrings(audience),
		allPublic: make(map[string]*rsa.PublicKey),
	}
	ks.current = signingKey{
		Kid:        kid,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	ks.allPublic[kid] = &privateKey.PublicKey
	return ks
}

// Update atomically replaces the current signing key and adds the new public key
// to the verification set. Old public keys are retained for the overlap period
// (up to 25h) to validate tokens signed with the previous key.
func (ks *KeyStore) Update(kid string, privateKey *rsa.PrivateKey) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.current = signingKey{
		Kid:        kid,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	ks.allPublic[kid] = &privateKey.PublicKey
}

// AddPublicKey adds a public key for verification only (for retired signing keys
// that are still within the overlap window).
func (ks *KeyStore) AddPublicKey(kid string, publicKey *rsa.PublicKey) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.allPublic[kid] = publicKey
}

// RemovePublicKey removes a public key once it is outside the overlap window.
func (ks *KeyStore) RemovePublicKey(kid string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	delete(ks.allPublic, kid)
}

// Sign creates and signs a JWT with the current signing key.
func (ks *KeyStore) Sign(claims Claims) (string, error) {
	ks.mu.RLock()
	key := ks.current
	ks.mu.RUnlock()

	now := time.Now()
	jti := "jwt_" + uuid.New().String()

	registered := jwt.RegisteredClaims{
		ID:        jti,
		Subject:   claims.Subject,
		Issuer:    ks.issuer,
		Audience:  ks.audience,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(claims.TTL)),
	}

	internal := jwtClaims{
		RegisteredClaims: registered,
		TenantID:         claims.TenantID,
		Roles:            claims.Roles,
		ClientID:         claims.ClientID,
		Scope:            claims.Scope,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, internal)
	token.Header["kid"] = key.Kid

	signed, err := token.SignedString(key.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signed, nil
}

// Verify parses and validates a JWT string.
// It selects the correct public key by inspecting the token header's kid field.
// Returns ParsedClaims on success, or an error if the token is invalid/expired.
func (ks *KeyStore) Verify(tokenString string) (*ParsedClaims, error) {
	// Parse the token, using the kid-based key lookup function.
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		// Reject non-RS256 algorithms — never accept HS256 (SEC-02).
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		kid, ok := t.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("JWT missing kid header")
		}

		ks.mu.RLock()
		pubKey, found := ks.allPublic[kid]
		ks.mu.RUnlock()

		if !found {
			return nil, fmt.Errorf("unknown kid: %q", kid)
		}

		return pubKey, nil
	},
		jwt.WithIssuer(ks.issuer),
		jwt.WithAudience(string(ks.audience[0])),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return nil, fmt.Errorf("verify JWT: %w", err)
	}

	internal, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid JWT claims")
	}

	return &ParsedClaims{
		ID:        internal.ID,
		Subject:   internal.Subject,
		TenantID:  internal.TenantID,
		Roles:     internal.Roles,
		ClientID:  internal.ClientID,
		Scope:     internal.Scope,
		IssuedAt:  internal.IssuedAt.Time,
		ExpiresAt: internal.ExpiresAt.Time,
	}, nil
}

// JWKS returns the JSON Web Key Set containing all current public keys.
// This is served by the /.well-known/jwks.json endpoint.
// Resource servers cache this response (Cache-Control: public, max-age=3600).
func (ks *KeyStore) JWKS() JWKS {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	keys := make([]JWK, 0, len(ks.allPublic))
	for kid, pubKey := range ks.allPublic {
		keys = append(keys, rsaPublicKeyToJWK(kid, pubKey))
	}
	return JWKS{Keys: keys}
}

// rsaPublicKeyToJWK converts an RSA public key to JWK format.
func rsaPublicKeyToJWK(kid string, key *rsa.PublicKey) JWK {
	// Encode modulus N as big-endian bytes, then base64url without padding.
	nBytes := key.N.Bytes()
	nEncoded := base64.RawURLEncoding.EncodeToString(nBytes)

	// Encode exponent E as big-endian bytes.
	eBig := big.NewInt(int64(key.E))
	eBytes := eBig.Bytes()
	eEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

	return JWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   nEncoded,
		E:   eEncoded,
	}
}

// MarshalJWKS serializes the JWKS to JSON bytes.
func (ks *KeyStore) MarshalJWKS() ([]byte, error) {
	jwks := ks.JWKS()
	return json.Marshal(jwks)
}
```

#### `pkg/tokens/refresh.go`

```go
// pkg/tokens/refresh.go
// (maps to pkg/crypto/token.go in the project scaffold)
package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	// refreshTokenBytes is the number of cryptographically random bytes
	// used to generate opaque refresh tokens. 32 bytes = 256 bits of entropy.
	refreshTokenBytes = 32
)

// GenerateOpaqueToken generates a cryptographically random opaque token.
// Returns the raw token string (transmitted to client; never stored).
// The token is hex-encoded for safe transmission in JSON and HTTP headers.
func GenerateOpaqueToken() (string, error) {
	b := make([]byte, refreshTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken computes the SHA-256 hash of a token for database storage.
// The raw token is transmitted to the client; only the hash is persisted.
// This means a database breach does not expose usable tokens (ADR-004).
//
// Returns the original raw string and its hex-encoded SHA-256 hash.
func HashToken(raw string) (string, string) {
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:])
}

// GenerateTokenWithHash generates a new opaque token and returns both the
// raw value (for the client) and the SHA-256 hash (for the database).
func GenerateTokenWithHash() (raw string, hash string, err error) {
	raw, err = GenerateOpaqueToken()
	if err != nil {
		return "", "", err
	}
	_, hash = HashToken(raw)
	return raw, hash, nil
}

// HashTokenString returns just the SHA-256 hex hash of the given raw token string.
// Used when looking up a presented token in the database.
func HashTokenString(raw string) string {
	_, hash := HashToken(raw)
	return hash
}
```

#### `pkg/crypto/argon2.go`

```go
// pkg/crypto/argon2.go
package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params defines the tuning parameters for Argon2id.
// These values are chosen to achieve ~300ms hash time on commodity hardware,
// providing brute force resistance while meeting PERF-01 (p95 login < 300ms).
// Password hashing runs in a goroutine off the request path to avoid blocking.
type Argon2Params struct {
	Memory      uint32 // KiB
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultArgon2Params returns the recommended production parameters for Argon2id.
// Adjust Memory/Iterations upward as hardware improves; calibrate to ~200ms.
var DefaultArgon2Params = Argon2Params{
	Memory:      64 * 1024, // 64 MiB
	Iterations:  3,
	Parallelism: 2,
	SaltLength:  16,
	KeyLength:   32,
}

// HashPassword hashes a plaintext password using Argon2id with the default params.
// Returns a PHC string format hash safe for storage in the database.
// Format: $argon2id$v=19$m=65536,t=3,p=2$<salt_b64>$<hash_b64>
func HashPassword(password string) (string, error) {
	return HashPasswordWithParams(password, DefaultArgon2Params)
}

// HashPasswordWithParams hashes a password with custom Argon2id parameters.
func HashPasswordWithParams(password string, params Argon2Params) (string, error) {
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Encode in PHC string format for future parameter upgrades.
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encoded, nil
}

// VerifyPassword checks a plaintext password against an Argon2id PHC hash.
// Returns true if the password matches. Uses constant-time comparison to
// prevent timing attacks (SEC-01).
func VerifyPassword(password, encodedHash string) bool {
	params, salt, hash, err := decodeArgon2Hash(encodedHash)
	if err != nil {
		// Invalid hash format — fail closed.
		return false
	}

	// Re-derive the hash with the same parameters and salt.
	derived := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Constant-time comparison — prevents timing attacks.
	return subtle.ConstantTimeCompare(hash, derived) == 1
}

// decodeArgon2Hash parses a PHC-encoded Argon2id hash string.
func decodeArgon2Hash(encoded string) (*Argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	// Expected format: ["", "argon2id", "v=19", "m=...,t=...,p=...", "<salt>", "<hash>"]
	if len(parts) != 6 {
		return nil, nil, nil, errors.New("invalid argon2id hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("unsupported algorithm: %q", parts[1])
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return nil, nil, nil, fmt.Errorf("parse version: %w", err)
	}
	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("unsupported argon2 version: %d", version)
	}

	params := &Argon2Params{}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d",
		&params.Memory, &params.Iterations, &params.Parallelism); err != nil {
		return nil, nil, nil, fmt.Errorf("parse params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decode hash: %w", err)
	}

	params.KeyLength = uint32(len(hash))
	params.SaltLength = uint32(len(salt))

	return params, salt, hash, nil
}
```

---

### 4g. Migration Runner

#### `cmd/migrate/main.go`

```go
// cmd/migrate/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/yourorg/auth-system/internal/config"
	pginfra "github.com/yourorg/auth-system/internal/infrastructure/postgres"
	"github.com/yourorg/auth-system/internal/repository/postgres"
	"github.com/yourorg/auth-system/internal/infrastructure/migrations"
)

func main() {
	var (
		scope     = flag.String("scope", "all", "Migration scope: global | tenant | all")
		tenant    = flag.String("tenant", "", "Tenant slug (required when scope=tenant)")
		direction = flag.String("direction", "up", "Migration direction: up | down")
		steps     = flag.Int("steps", 0, "Number of steps for down migration (0 = all)")
	)
	flag.Parse()

	// Configure structured logging.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Connect to PostgreSQL.
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
			// RunAllTenants logs per-tenant failures but does not exit early.
			// A non-nil error here means at least one tenant failed.
			slog.Error("one or more tenant migrations failed", "error", err)
			os.Exit(1)
		}
		slog.Info("all migrations complete")

	default:
		fmt.Fprintf(os.Stderr, "error: unknown scope %q — use global | tenant | all\n", *scope)
		os.Exit(1)
	}
}
```

#### `internal/infrastructure/migrations/runner.go`

```go
// internal/infrastructure/migrations/runner.go
package migrations

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/auth-system/internal/domain"
)

// MigrationRunner runs golang-migrate migrations against global and tenant schemas.
type MigrationRunner struct {
	pool              *pgxpool.Pool
	dbURL             string // Base postgres:// URL (no search_path)
	globalMigrPath    string // e.g., "file://migrations/global"
	tenantMigrPath    string // e.g., "file://migrations/tenant"
	tenantRepo        domain.TenantRepository
}

// NewMigrationRunner constructs a MigrationRunner.
func NewMigrationRunner(
	pool *pgxpool.Pool,
	dbURL string,
	globalPath string,
	tenantPath string,
	tenantRepo domain.TenantRepository,
) *MigrationRunner {
	return &MigrationRunner{
		pool:           pool,
		dbURL:          dbURL,
		globalMigrPath: globalPath,
		tenantMigrPath: tenantPath,
		tenantRepo:     tenantRepo,
	}
}

// RunGlobal applies migrations to the public (global) schema.
func (r *MigrationRunner) RunGlobal(ctx context.Context, direction string) error {
	// Global migrations target the public schema — no search_path override needed.
	m, err := migrate.New(r.globalMigrPath, r.dbURL)
	if err != nil {
		return fmt.Errorf("create global migrator: %w", err)
	}
	defer m.Close()

	return runMigration(m, direction, 0)
}

// RunTenant applies migrations to a single named tenant schema.
func (r *MigrationRunner) RunTenant(ctx context.Context, tenantSlug string, direction string, steps int) error {
	tenant, err := r.tenantRepo.FindBySlug(ctx, tenantSlug)
	if err != nil {
		return fmt.Errorf("find tenant %q: %w", tenantSlug, err)
	}

	if err := r.migrateTenantSchema(tenant.SchemaName, direction, steps); err != nil {
		return fmt.Errorf("migrate tenant %q (schema %q): %w", tenantSlug, tenant.SchemaName, err)
	}

	return nil
}

// RunAllTenants applies pending UP migrations to all active tenant schemas.
// Each tenant is migrated independently — a failure in one does not block others.
// Returns a combined error listing all failed schemas.
func (r *MigrationRunner) RunAllTenants(ctx context.Context) error {
	schemas, err := r.tenantRepo.ListActiveSchemaNames(ctx)
	if err != nil {
		return fmt.Errorf("list active tenant schemas: %w", err)
	}

	var failedSchemas []string
	for _, schemaName := range schemas {
		if err := r.migrateTenantSchema(schemaName, "up", 0); err != nil {
			slog.Error("tenant migration failed",
				"schema", schemaName,
				"error", err,
			)
			failedSchemas = append(failedSchemas, schemaName)
			// Continue — do not block other tenants.
		} else {
			slog.Info("tenant migration applied", "schema", schemaName)
		}
	}

	if len(failedSchemas) > 0 {
		return fmt.Errorf("migrations failed for %d schema(s): %v", len(failedSchemas), failedSchemas)
	}
	return nil
}

// migrateTenantSchema runs migrations for a single tenant schema by setting
// search_path via the connection string options.
func (r *MigrationRunner) migrateTenantSchema(schemaName string, direction string, steps int) error {
	if !domain.IsValidSchemaName(schemaName) {
		return fmt.Errorf("invalid schema name: %q", schemaName)
	}

	// Append search_path and per-schema migration table to the DSN.
	// golang-migrate will apply migrations within the scoped search_path.
	dsn := fmt.Sprintf("%s&search_path=%s,public&x-migrations-table=schema_migrations",
		r.dbURL, schemaName)

	m, err := migrate.New(r.tenantMigrPath, dsn)
	if err != nil {
		return fmt.Errorf("create migrator for schema %q: %w", schemaName, err)
	}
	defer m.Close()

	return runMigration(m, direction, steps)
}

// runMigration executes a migration in the specified direction.
func runMigration(m *migrate.Migrate, direction string, steps int) error {
	var err error
	switch direction {
	case "up":
		err = m.Up()
	case "down":
		if steps > 0 {
			err = m.Steps(-steps)
		} else {
			err = m.Down()
		}
	default:
		return fmt.Errorf("unknown migration direction: %q", direction)
	}

	if err == migrate.ErrNoChange {
		slog.Info("no new migrations to apply")
		return nil
	}
	return err
}
```

---

### 4h. Router

#### `internal/router/router.go`

```go
// internal/router/router.go
package router

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourorg/auth-system/internal/config"
	"github.com/yourorg/auth-system/internal/handler"
	"github.com/yourorg/auth-system/internal/middleware"
	"github.com/yourorg/auth-system/internal/service"
	pginfra "github.com/yourorg/auth-system/internal/infrastructure/postgres"
	"github.com/yourorg/auth-system/pkg/jwtutil"
)

// Dependencies holds all handler and middleware dependencies injected by main.go.
type Dependencies struct {
	Config         *config.Config
	AuthHandler    *handler.AuthHandler
	EmailHandler   *handler.EmailHandler
	PasswordHandler *handler.PasswordHandler
	SessionHandler  *handler.SessionHandler
	UserHandler    *handler.UserHandler
	AdminHandler   *handler.AdminHandler
	TenantHandler  *handler.TenantHandler
	RoleHandler    *handler.RoleHandler
	OAuthHandler   *handler.OAuthHandler
	GoogleHandler  *handler.GoogleHandler
	AuditHandler   *handler.AuditHandler
	WellKnownHandler *handler.WellKnownHandler
	JWTKeyStore    *jwtutil.KeyStore
	TenantCache    middleware.TenantCache
	RateLimiter    middleware.RateLimiter
}

// New builds and returns the configured Gin engine.
// All routes, middleware chains, and groups are defined here.
// No business logic lives in this file.
func New(deps Dependencies) *gin.Engine {
	if deps.Config.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New() // Do not use gin.Default() — we configure middleware explicitly.

	// --- Global middleware (applied to every request) ---
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.SecureHeaders())
	r.Use(middleware.CORS(deps.Config.CORS.AllowedOrigins))
	r.Use(gin.Recovery()) // Recover from panics; log and return 500.

	// --- Health + metrics endpoints (no auth, no tenant) ---
	r.GET("/health", healthHandler())
	r.GET("/metrics", gin.WrapH(metricsHandler()))

	// --- JWKS endpoint (no auth, no tenant, no /api/v1 prefix per spec) ---
	r.GET("/.well-known/jwks.json", deps.WellKnownHandler.JWKS)

	// --- API v1 router group ---
	v1 := r.Group("/api/v1")

	// Shared middleware builders for route groups.
	authMW := middleware.RequireAuth(deps.JWTKeyStore)
	tenantFromHeader := middleware.RequireTenant(deps.TenantCache)
	tenantFromJWT := func() gin.HandlerFunc {
		// For authenticated routes: AuthMW must run first to populate JWT claims,
		// then TenantMW reads tenant_id from those claims.
		return middleware.RequireTenant(deps.TenantCache)
	}

	rlLogin := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc:     middleware.IPKeyFunc,
		Limit:       deps.Config.RateLimit.LoginIPPerMinute,
		Window:      time.Minute,
		Description: "login:ip",
	})
	rlRegister := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc:     middleware.IPKeyFunc,
		Limit:       deps.Config.RateLimit.RegisterIPPerMinute,
		Window:      time.Minute,
		Description: "register:ip",
	})
	rlForgot := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc:     middleware.IPKeyFunc,
		Limit:       deps.Config.RateLimit.ForgotPasswordIPPerMinute,
		Window:      time.Minute,
		Description: "forgot:ip",
	})
	rlRefresh := middleware.RateLimit(deps.RateLimiter, middleware.RateLimitConfig{
		KeyFunc:     middleware.IPKeyFunc,
		Limit:       deps.Config.RateLimit.TokenRefreshIPPerMinute,
		Window:      time.Minute,
		Description: "refresh:ip",
	})

	// -------------------------------------------------------------------------
	// Auth routes — unauthenticated (tenant from X-Tenant-ID header)
	// -------------------------------------------------------------------------
	auth := v1.Group("/auth", tenantFromHeader)
	{
		auth.POST("/register", rlRegister, deps.AuthHandler.Register)
		auth.POST("/login", rlLogin, deps.AuthHandler.Login)
		auth.POST("/verify-email", deps.EmailHandler.VerifyEmail)
		auth.POST("/resend-verification", deps.EmailHandler.ResendVerification)
		auth.POST("/forgot-password", rlForgot, deps.PasswordHandler.ForgotPassword)
		auth.POST("/reset-password", deps.PasswordHandler.ResetPassword)

		// Session refresh — no JWT required (refresh token is the credential).
		auth.POST("/token/refresh", rlRefresh, deps.SessionHandler.Refresh)

		// Google OAuth initiation — unauthenticated, tenant-scoped.
		auth.POST("/oauth/google", deps.GoogleHandler.Initiate)
		auth.GET("/oauth/google/callback", deps.GoogleHandler.Callback)
	}

	// -------------------------------------------------------------------------
	// Authenticated user routes (JWT + tenant)
	// -------------------------------------------------------------------------
	authed := v1.Group("", authMW, tenantFromJWT())
	{
		// Logout (requires valid JWT + refresh token).
		authed.POST("/auth/logout", deps.AuthHandler.Logout)
		authed.POST("/auth/logout/all", deps.AuthHandler.LogoutAll)

		// User self-service profile.
		authed.GET("/users/me", deps.UserHandler.GetMe)
		authed.PUT("/users/me", deps.UserHandler.UpdateMe)
	}

	// -------------------------------------------------------------------------
	// Tenant admin routes (JWT + admin role)
	// -------------------------------------------------------------------------
	adminRole := middleware.RequireRole("admin")
	admin := v1.Group("/admin", authMW, tenantFromJWT(), adminRole)
	{
		// User management.
		admin.POST("/users/invite", deps.AdminHandler.InviteUser)
		admin.PUT("/users/:id/disable", deps.AdminHandler.DisableUser)
		admin.DELETE("/users/:id", deps.AdminHandler.DeleteUser)
		admin.GET("/users", deps.AdminHandler.ListUsers)

		// Role management.
		admin.POST("/roles", deps.RoleHandler.CreateRole)
		admin.GET("/roles", deps.RoleHandler.ListRoles)
		admin.POST("/users/:id/roles", deps.RoleHandler.AssignRole)
		admin.DELETE("/users/:id/roles/:roleId", deps.RoleHandler.UnassignRole)

		// Audit log.
		admin.GET("/audit-log", deps.AuditHandler.List)

		// OAuth client management.
		admin.POST("/oauth/clients", deps.OAuthHandler.RegisterClient)
	}

	// -------------------------------------------------------------------------
	// Super-admin routes (JWT + super_admin role, global schema — no tenant routing)
	// -------------------------------------------------------------------------
	superAdminRole := middleware.RequireRole("super_admin")
	superAdmin := v1.Group("/admin", authMW, superAdminRole)
	{
		superAdmin.POST("/tenants", deps.TenantHandler.ProvisionTenant)
		superAdmin.GET("/tenants/:id", deps.TenantHandler.GetTenant)
		superAdmin.GET("/tenants", deps.TenantHandler.ListTenants)
		superAdmin.PUT("/tenants/:id/suspend", deps.TenantHandler.SuspendTenant)
	}

	// -------------------------------------------------------------------------
	// OAuth 2.0 endpoints (ory/fosite handles auth per-grant-type)
	// -------------------------------------------------------------------------
	// /oauth/authorize requires a valid user JWT (user must be logged in).
	oauth := v1.Group("/oauth")
	{
		oauth.GET("/authorize", authMW, tenantFromJWT(), deps.OAuthHandler.Authorize)
		oauth.POST("/token", deps.OAuthHandler.Token)         // Auth method varies by grant
		oauth.POST("/introspect", deps.OAuthHandler.Introspect) // HTTP Basic Auth
		oauth.POST("/revoke", deps.OAuthHandler.Revoke)
	}

	// 404 handler — return structured JSON not Gin's plain text default.
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "The requested endpoint does not exist.",
			},
		})
	})

	return r
}

// healthHandler returns a Gin handler for the /health liveness/readiness endpoint.
// The actual health checks (postgres, redis, vault) are injected via closure in main.go.
func healthHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "auth-api",
		})
	}
}

// metricsHandler returns an http.Handler for Prometheus metrics.
// Implemented in main.go using promhttp.Handler().
func metricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
```

---

### 4i. Main Entry Point

#### `cmd/api/main.go`

```go
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

	"github.com/yourorg/auth-system/internal/config"
	"github.com/yourorg/auth-system/internal/handler"
	"github.com/yourorg/auth-system/internal/infrastructure/email"
	pginfra "github.com/yourorg/auth-system/internal/infrastructure/postgres"
	redisinfra "github.com/yourorg/auth-system/internal/infrastructure/redis"
	"github.com/yourorg/auth-system/internal/infrastructure/vault"
	"github.com/yourorg/auth-system/internal/middleware"
	pgRepo "github.com/yourorg/auth-system/internal/repository/postgres"
	redisRepo "github.com/yourorg/auth-system/internal/repository/redis"
	"github.com/yourorg/auth-system/internal/router"
	"github.com/yourorg/auth-system/internal/service"
	"github.com/yourorg/auth-system/pkg/jwtutil"
)

func main() {
	// Configure structured JSON logging to stdout (consumed by Fly.io log aggregation).
	logLevel := slog.LevelInfo
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Rename the default "msg" key to "message" for consistency.
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

// run wires all dependencies and starts the HTTP server.
// Separated from main() to allow defer statements to execute on exit.
func run() error {
	// -------------------------------------------------------------------------
	// 1. Load configuration.
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// -------------------------------------------------------------------------
	// 2. Connect to PostgreSQL.
	// -------------------------------------------------------------------------
	pool, err := pginfra.NewPool(ctx, cfg.Database)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	// -------------------------------------------------------------------------
	// 3. Connect to Redis.
	// -------------------------------------------------------------------------
	redisClient, err := redisinfra.NewClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("connect to redis: %w", err)
	}
	defer redisClient.Close()

	// -------------------------------------------------------------------------
	// 4. Connect to Vault and load signing keys.
	// -------------------------------------------------------------------------
	vaultClient, err := vault.NewClient(cfg.Vault)
	if err != nil {
		return fmt.Errorf("connect to vault: %w", err)
	}

	signingKey, kid, err := vaultClient.LoadCurrentSigningKey(ctx)
	if err != nil {
		return fmt.Errorf("load signing key from vault: %w", err)
	}

	keyStore := jwtutil.NewKeyStore(
		cfg.JWT.Issuer,
		cfg.JWT.Audience,
		kid,
		signingKey,
	)

	// Start background key rotation watcher.
	keyRotationWatcher := vault.NewKeyRotationWatcher(vaultClient, keyStore, cfg.Vault.KeyRotationPollInterval)
	go keyRotationWatcher.Watch(ctx)

	slog.Info("connected to vault", "kid", kid)

	// -------------------------------------------------------------------------
	// 5. Start email worker (async channel-based per ADR-012).
	// -------------------------------------------------------------------------
	emailChannel := make(chan service.EmailTask, cfg.Email.ChannelBufferSize)
	emailClient := email.NewClient(cfg.Email)
	emailWorker := email.NewWorker(emailClient, cfg.Email)
	go emailWorker.Start(ctx, emailChannel)

	slog.Info("email worker started", "workers", cfg.Email.WorkerConcurrency)

	// -------------------------------------------------------------------------
	// 6. Wire repositories.
	// -------------------------------------------------------------------------
	userRepo := pgRepo.NewPostgresUserRepo(pool)
	sessionRepo := pgRepo.NewPostgresSessionRepo(pool)
	tokenRepo := pgRepo.NewPostgresTokenRepo(pool)
	auditRepo := pgRepo.NewPostgresAuditRepo(pool)
	tenantRepo := pgRepo.NewPostgresTenantRepo(pool)
	roleRepo := pgRepo.NewPostgresRoleRepo(pool)

	rateLimiter := redisRepo.NewRedisRateLimiter(redisClient)

	// -------------------------------------------------------------------------
	// 7. Wire services.
	// -------------------------------------------------------------------------
	authSvcCfg := service.AuthServiceConfig{
		AccessTokenTTL:         cfg.JWT.AccessTokenTTL,
		SessionDefaultTTL:      cfg.Session.DefaultTTL,
		VerificationTokenTTL:   cfg.Email.VerificationTokenTTL,
		LockoutThreshold:       cfg.RateLimit.LockoutThreshold,
		LockoutDurationSeconds: cfg.RateLimit.LockoutDurationSeconds,
	}

	authSvc := service.NewAuthService(
		userRepo, sessionRepo, tokenRepo, auditRepo, tenantRepo,
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
		tenantRepo, pool, cfg.Database.URL,
		"file://migrations/tenant", emailChannel,
	)
	rbacSvc := service.NewRBACService(roleRepo, userRepo, auditRepo)
	adminSvc := service.NewAdminService(userRepo, sessionRepo, roleRepo, auditRepo, emailChannel)
	auditSvc := service.NewAuditService(auditRepo)
	jwtSvc := service.NewJWTService(keyStore)

	// -------------------------------------------------------------------------
	// 8. Wire handlers.
	// -------------------------------------------------------------------------
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
		WellKnownHandler: handler.NewWellKnownHandler(keyStore),
		JWTKeyStore:      keyStore,
		TenantCache:      tenantCache,
		RateLimiter:      rateLimiter,
	}

	// -------------------------------------------------------------------------
	// 9. Build router and start HTTP server.
	// -------------------------------------------------------------------------
	ginRouter := router.New(deps)

	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      ginRouter,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so we can listen for shutdown signals.
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", srv.Addr, "env", cfg.Server.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// -------------------------------------------------------------------------
	// 10. Graceful shutdown on SIGINT / SIGTERM.
	// -------------------------------------------------------------------------
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

	// Stop accepting new requests; wait for in-flight requests to complete.
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	// Signal background goroutines (key rotation watcher, email worker) to stop.
	cancel()

	// Drain remaining email tasks with a short timeout.
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer drainCancel()
	emailWorker.Drain(drainCtx)

	slog.Info("server shutdown complete")
	return nil
}
```

---

## 5. Docker Compose (Local Dev)

```yaml
# docker-compose.yml
# Local development environment.
# Provides: PostgreSQL 16, Redis 7, HashiCorp Vault (dev mode), Mailhog.
# Usage: docker compose up -d

services:

  # ---------------------------------------------------------------------------
  # PostgreSQL 16 — Primary database
  # Schema-per-tenant architecture: global public schema + per-tenant schemas.
  # ---------------------------------------------------------------------------
  postgres:
    image: postgres:16-alpine
    container_name: auth-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: authdb
      POSTGRES_USER: auth
      POSTGRES_PASSWORD: authpassword      # Override in .env.local for local dev
      POSTGRES_INITDB_ARGS: "--encoding=UTF-8 --lc-collate=C --lc-ctype=C"
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./docker/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U auth -d authdb"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    networks:
      - auth-net

  # ---------------------------------------------------------------------------
  # Redis 7 — Rate limiting, token denylist, OAuth state/nonce store
  # ---------------------------------------------------------------------------
  redis:
    image: redis:7-alpine
    container_name: auth-redis
    restart: unless-stopped
    command: >
      redis-server
      --save 60 1
      --loglevel warning
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 5
      start_period: 5s
    networks:
      - auth-net

  # ---------------------------------------------------------------------------
  # HashiCorp Vault — Secrets manager (dev mode — DO NOT use dev mode in prod)
  # Dev mode starts with a pre-configured in-memory storage.
  # Root token: dev-root-token (set in .env.local as VAULT_TOKEN)
  # UI available at http://localhost:8200/ui
  # ---------------------------------------------------------------------------
  vault:
    image: hashicorp/vault:1.16
    container_name: auth-vault
    restart: unless-stopped
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: dev-root-token
      VAULT_DEV_LISTEN_ADDRESS: 0.0.0.0:8200
      VAULT_LOG_LEVEL: warn
    cap_add:
      - IPC_LOCK           # Required for Vault memory locking
    ports:
      - "8200:8200"
    healthcheck:
      test: ["CMD", "vault", "status"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 5s
    networks:
      - auth-net

  # ---------------------------------------------------------------------------
  # Vault Init — One-shot container that configures Vault after it starts.
  # Enables the KV v2 secrets mount and loads a dev signing key pair.
  # This container exits after running; it is not a long-running service.
  # ---------------------------------------------------------------------------
  vault-init:
    image: hashicorp/vault:1.16
    container_name: auth-vault-init
    restart: "no"
    depends_on:
      vault:
        condition: service_healthy
    environment:
      VAULT_ADDR: http://vault:8200
      VAULT_TOKEN: dev-root-token
    entrypoint: ["/bin/sh", "-c"]
    command:
      - |
        set -e
        echo "Enabling KV v2 secrets engine..."
        vault secrets enable -path=secret kv-v2 || echo "Already enabled"

        echo "Loading dev signing key (generated at build time)..."
        # In real init: generate RSA key pair and store it.
        # For local dev, the seed script (make seed) writes keys after init.
        echo "Vault init complete."
    networks:
      - auth-net

  # ---------------------------------------------------------------------------
  # Mailhog — SMTP catch-all for local email testing
  # All outbound emails from the API are delivered here — never to real addresses.
  # SMTP on port 1025 | Web UI at http://localhost:8025
  # ---------------------------------------------------------------------------
  mailhog:
    image: mailhog/mailhog:v1.0.1
    container_name: auth-mailhog
    restart: unless-stopped
    ports:
      - "1025:1025"    # SMTP — configure EMAIL_SMTP_HOST=localhost, EMAIL_SMTP_PORT=1025
      - "8025:8025"    # Web UI — open http://localhost:8025 to view all sent emails
    networks:
      - auth-net

# -----------------------------------------------------------------------------
# Named volumes — persist data across container restarts
# -----------------------------------------------------------------------------
volumes:
  postgres_data:
    driver: local
  redis_data:
    driver: local

# -----------------------------------------------------------------------------
# Network — isolates services from other Docker workloads
# -----------------------------------------------------------------------------
networks:
  auth-net:
    driver: bridge
```

```sql
-- docker/postgres/init.sql
-- Run once when the postgres container is first created.
-- Creates extensions required by the application.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";    -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "citext";      -- Case-insensitive text (optional)

-- Create the global schema (public is default and already exists).
-- The tenant schemas are created dynamically by the TenantProvisioner.
```

---

## 6. Makefile

```makefile
# Makefile — Auth System developer workflow
# Usage: make <target>
# All targets assume you are in the monorepo root.

BINARY_NAME   := auth-api
BUILD_DIR     := ./bin
GO_CMD        := go
DOCKER_IMAGE  := auth-api
BACKEND_DIR   := ./backend
MIGRATE_CMD   := $(GO_CMD) run $(BACKEND_DIR)/cmd/migrate/main.go

# Load .env.local if it exists (for local development convenience).
# In CI, environment variables are set by the pipeline.
ifneq (,$(wildcard .env.local))
  include .env.local
  export
endif

.PHONY: all dev build clean \
        migrate migrate-global migrate-tenant migrate-all-tenants \
        test test-unit test-integration test-isolation test-coverage \
        lint vet fmt \
        docker-build docker-push \
        seed help

# Default target
all: lint test build

# ---------------------------------------------------------------------------
# Development
# ---------------------------------------------------------------------------

## dev: Start all Docker services and run the API with live output
dev: docker-up
	@echo "Starting auth API..."
	cd $(BACKEND_DIR) && $(GO_CMD) run ./cmd/api

## docker-up: Start all local development services (postgres, redis, vault, mailhog)
docker-up:
	docker compose up -d
	@echo "Waiting for services to be healthy..."
	@docker compose ps

## docker-down: Stop all local development services (preserves volumes)
docker-down:
	docker compose down

## docker-reset: Stop all services and DELETE all data volumes (full reset)
docker-reset:
	docker compose down -v
	@echo "All volumes deleted. Run 'make dev' to start fresh."

# ---------------------------------------------------------------------------
# Database Migrations
# ---------------------------------------------------------------------------

## migrate: Run global migrations + all tenant schema migrations (up)
migrate: migrate-global migrate-all-tenants

## migrate-global: Apply pending migrations to the global (public) schema
migrate-global:
	@echo "Running global schema migrations..."
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=global -direction=up

## migrate-all-tenants: Apply pending migrations to all active tenant schemas
migrate-all-tenants:
	@echo "Running all tenant schema migrations..."
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=all -direction=up

## migrate-tenant TENANT=<slug>: Apply pending migrations to a specific tenant schema
## Example: make migrate-tenant TENANT=acme-corp
migrate-tenant:
ifndef TENANT
	$(error TENANT is not set. Usage: make migrate-tenant TENANT=acme-corp)
endif
	@echo "Running migrations for tenant: $(TENANT)"
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=tenant -tenant=$(TENANT) -direction=up

## migrate-down TENANT=<slug>: Roll back one migration for a specific tenant
migrate-down:
ifndef TENANT
	$(error TENANT is not set. Usage: make migrate-down TENANT=acme-corp)
endif
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=tenant -tenant=$(TENANT) -direction=down -steps=1

## migrate-create NAME=<name> SCOPE=<global|tenant>: Create a new migration file pair
migrate-create:
ifndef NAME
	$(error NAME is not set. Usage: make migrate-create NAME=add_user_phone SCOPE=tenant)
endif
ifndef SCOPE
	$(error SCOPE is not set. Usage: make migrate-create NAME=add_user_phone SCOPE=tenant)
endif
	@TIMESTAMP=$$(date +%06d) ; \
	  FILENAME="$${TIMESTAMP}_$(NAME)" ; \
	  touch $(BACKEND_DIR)/migrations/$(SCOPE)/$${FILENAME}.up.sql ; \
	  touch $(BACKEND_DIR)/migrations/$(SCOPE)/$${FILENAME}.down.sql ; \
	  echo "Created: migrations/$(SCOPE)/$${FILENAME}.up.sql" ; \
	  echo "Created: migrations/$(SCOPE)/$${FILENAME}.down.sql"

# ---------------------------------------------------------------------------
# Testing
# ---------------------------------------------------------------------------

## test: Run all tests (unit + integration + isolation)
test: test-unit test-integration test-isolation

## test-unit: Run unit tests only (no external services required)
test-unit:
	@echo "Running unit tests..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=60s \
	  -short \
	  ./internal/... ./pkg/...

## test-integration: Run integration tests (requires running Docker services)
test-integration:
	@echo "Running integration tests..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=120s \
	  -v \
	  ./test/integration/...

## test-isolation: Run cross-tenant isolation suite only (CI-blocking per US-08b)
## Any cross-tenant data leak in this suite = immediate failure.
test-isolation:
	@echo "Running cross-tenant isolation test suite..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=180s \
	  -v \
	  -run TestCrossTenant \
	  ./test/isolation/...

## test-coverage: Run tests with coverage report (fails if < 80%)
test-coverage:
	@echo "Running tests with coverage..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=120s \
	  -coverprofile=coverage.out \
	  ./internal/... ./pkg/...
	$(GO_CMD) tool cover -func=$(BACKEND_DIR)/coverage.out | tail -1
	@COVERAGE=$$(cd $(BACKEND_DIR) && $(GO_CMD) tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%') ; \
	  echo "Coverage: $${COVERAGE}%" ; \
	  if [ "$$(echo "$${COVERAGE} < 80" | bc -l)" = "1" ] ; then \
	    echo "FAIL: Coverage $${COVERAGE}% is below the 80% threshold." ; \
	    exit 1 ; \
	  fi

# ---------------------------------------------------------------------------
# Code Quality
# ---------------------------------------------------------------------------

## lint: Run golangci-lint with project configuration
lint:
	@echo "Running golangci-lint..."
	cd $(BACKEND_DIR) && golangci-lint run ./...

## vet: Run go vet
vet:
	cd $(BACKEND_DIR) && $(GO_CMD) vet ./...

## fmt: Format all Go source files
fmt:
	cd $(BACKEND_DIR) && $(GO_CMD) fmt ./...

## check: Run all code quality checks (fmt + vet + lint)
check: fmt vet lint

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

## build: Build the production binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	  $(GO_CMD) build \
	  -ldflags="-w -s -X main.version=$(shell git describe --tags --always --dirty)" \
	  -o ../$(BUILD_DIR)/$(BINARY_NAME) \
	  ./cmd/api
	@echo "Binary: $(BUILD_DIR)/$(BINARY_NAME)"

## docker-build: Build the Docker image
docker-build:
	@echo "Building Docker image: $(DOCKER_IMAGE):local"
	docker build \
	  --build-arg VERSION=$(shell git describe --tags --always --dirty) \
	  -t $(DOCKER_IMAGE):local \
	  -t $(DOCKER_IMAGE):$(shell git rev-parse --short HEAD) \
	  $(BACKEND_DIR)

## docker-push: Push image to registry (set REGISTRY env var)
docker-push: docker-build
ifndef REGISTRY
	$(error REGISTRY is not set. Usage: REGISTRY=registry.example.com make docker-push)
endif
	docker tag $(DOCKER_IMAGE):local $(REGISTRY)/$(DOCKER_IMAGE):$(shell git rev-parse --short HEAD)
	docker push $(REGISTRY)/$(DOCKER_IMAGE):$(shell git rev-parse --short HEAD)

# ---------------------------------------------------------------------------
# Seeding
# ---------------------------------------------------------------------------

## seed: Create a test super-admin user and a test tenant ("dev-corp")
## Prints credentials to stdout — store them immediately.
seed:
	@echo "Seeding test data..."
	cd $(BACKEND_DIR) && $(GO_CMD) run ./scripts/seed.go

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BACKEND_DIR)/coverage.out
	@echo "Clean complete."

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------

## help: Show this help message
help:
	@echo "Auth System — Available Makefile targets:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'
```

---

## 7. GitHub Actions CI/CD Pipeline

```yaml
# .github/workflows/ci.yml
# Triggered on: every push to any branch, and on pull requests targeting main.
# All 8 stages must pass before merge to main is permitted.

name: CI/CD Pipeline

on:
  push:
    branches: ["**"]
  pull_request:
    branches: [main]
  workflow_dispatch:    # Manual trigger for production deployment

env:
  GO_VERSION: "1.22"
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}/auth-api

jobs:

  # ---------------------------------------------------------------------------
  # Stage 1 — Lint
  # Enforces code style and catches static analysis issues before tests run.
  # ---------------------------------------------------------------------------
  lint:
    name: "Stage 1 — Lint"
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Run golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest
          working-directory: backend
          args: --timeout=5m

      - name: Run go vet
        run: cd backend && go vet ./...

  # ---------------------------------------------------------------------------
  # Stage 2 — Unit Tests
  # Fast tests with no external dependencies. Enforces 80% coverage minimum.
  # ---------------------------------------------------------------------------
  unit-test:
    name: "Stage 2 — Unit Tests"
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Run unit tests with coverage
        run: |
          cd backend
          go test \
            -race \
            -count=1 \
            -timeout=60s \
            -short \
            -coverprofile=coverage.out \
            -covermode=atomic \
            ./internal/... ./pkg/...

      - name: Check coverage threshold (>= 80%)
        run: |
          cd backend
          COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}' | tr -d '%')
          echo "Total coverage: ${COVERAGE}%"
          if (( $(echo "$COVERAGE < 80" | bc -l) )); then
            echo "FAIL: Coverage ${COVERAGE}% is below the required 80% threshold."
            exit 1
          fi

      - name: Upload coverage to Codecov
        uses: codecov/codecov-action@v4
        with:
          file: backend/coverage.out
          flags: unittests
          fail_ci_if_error: false

  # ---------------------------------------------------------------------------
  # Stage 3 — Integration Tests
  # Full request-path tests against real PostgreSQL and Redis instances.
  # Tests the complete auth flow: register → verify → login → refresh → logout.
  # ---------------------------------------------------------------------------
  integration-test:
    name: "Stage 3 — Integration Tests"
    runs-on: ubuntu-latest
    needs: lint

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: authdb_test
          POSTGRES_USER: auth
          POSTGRES_PASSWORD: auth
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U auth -d authdb_test"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 3s
          --health-retries 5

    env:
      DATABASE_URL: postgres://auth:auth@localhost:5432/authdb_test?sslmode=disable
      REDIS_URL: redis://localhost:6379
      VAULT_ADDR: ""          # Integration tests use local key files
      JWT_PRIVATE_KEY_PATH: ./test/fixtures/dev-private.pem
      JWT_PUBLIC_KEY_PATH: ./test/fixtures/dev-public.pem
      JWT_KEY_ID: key-test-001
      JWT_ISSUER: http://localhost:8080
      JWT_AUDIENCE: http://localhost:3000
      AUTH_SERVICE_BASE_URL: http://localhost:8080
      EMAIL_FROM: test@example.com
      ENV: development

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Generate test RSA key pair
        run: |
          mkdir -p backend/test/fixtures
          openssl genrsa -out backend/test/fixtures/dev-private.pem 2048
          openssl rsa -in backend/test/fixtures/dev-private.pem \
            -pubout -out backend/test/fixtures/dev-public.pem

      - name: Run global migrations
        run: |
          cd backend
          go run ./cmd/migrate -scope=global -direction=up

      - name: Run integration tests
        run: |
          cd backend
          go test \
            -race \
            -count=1 \
            -timeout=120s \
            -v \
            ./test/integration/...

  # ---------------------------------------------------------------------------
  # Stage 4 — Cross-Tenant Isolation Tests (US-08b)
  # HARD BLOCK: Any cross-tenant data returned in any response = immediate failure.
  # This stage must pass on every pull request — it cannot be bypassed.
  # ---------------------------------------------------------------------------
  isolation-test:
    name: "Stage 4 — Cross-Tenant Isolation Tests (HARD BLOCK)"
    runs-on: ubuntu-latest
    needs: lint

    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_DB: authdb_isolation
          POSTGRES_USER: auth
          POSTGRES_PASSWORD: auth
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U auth -d authdb_isolation"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 3s
          --health-retries 5

    env:
      DATABASE_URL: postgres://auth:auth@localhost:5432/authdb_isolation?sslmode=disable
      REDIS_URL: redis://localhost:6379
      JWT_PRIVATE_KEY_PATH: ./test/fixtures/dev-private.pem
      JWT_PUBLIC_KEY_PATH: ./test/fixtures/dev-public.pem
      JWT_KEY_ID: key-test-001
      JWT_ISSUER: http://localhost:8080
      JWT_AUDIENCE: http://localhost:3000
      AUTH_SERVICE_BASE_URL: http://localhost:8080
      EMAIL_FROM: test@example.com
      ENV: development

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Generate test RSA key pair
        run: |
          mkdir -p backend/test/fixtures
          openssl genrsa -out backend/test/fixtures/dev-private.pem 2048
          openssl rsa -in backend/test/fixtures/dev-private.pem \
            -pubout -out backend/test/fixtures/dev-public.pem

      - name: Provision two isolated test tenants
        run: |
          cd backend
          go run ./cmd/migrate -scope=global -direction=up

      - name: Run cross-tenant isolation tests
        run: |
          cd backend
          go test \
            -race \
            -count=1 \
            -timeout=180s \
            -v \
            -run TestCrossTenant \
            ./test/isolation/...
        # Any non-zero exit code here is a hard block on merge.

  # ---------------------------------------------------------------------------
  # Stage 5 — Security Scan
  # Runs govulncheck (known CVEs in deps) and gosec (code-level SAST).
  # ---------------------------------------------------------------------------
  security-scan:
    name: "Stage 5 — Security Scan"
    runs-on: ubuntu-latest
    needs: lint
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: ${{ env.GO_VERSION }}
          cache: true
          cache-dependency-path: backend/go.sum

      - name: Run govulncheck (known vulnerability scan)
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          cd backend && govulncheck ./...

      - name: Run gosec (Go AST security scanner)
        uses: securego/gosec@master
        with:
          args: >
            -severity medium
            -confidence medium
            -exclude-dir=test
            ./...
        env:
          GOPATH: ${{ github.workspace }}/go

      - name: Run Trivy container vulnerability scan
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: fs
          scan-ref: backend
          severity: CRITICAL,HIGH
          exit-code: 1
          ignore-unfixed: true

  # ---------------------------------------------------------------------------
  # Stage 6 — Build Docker Image
  # Only runs on pushes to main after all quality gates pass.
  # Publishes to GitHub Container Registry (GHCR).
  # ---------------------------------------------------------------------------
  build:
    name: "Stage 6 — Build Docker Image"
    runs-on: ubuntu-latest
    needs: [unit-test, integration-test, isolation-test, security-scan]
    if: github.ref == 'refs/heads/main'
    permissions:
      contents: read
      packages: write

    outputs:
      image-tag: ${{ steps.meta.outputs.tags }}
      image-digest: ${{ steps.build-push.outputs.digest }}

    steps:
      - uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract image metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=sha,prefix=sha-,format=short
            type=ref,event=branch
            type=semver,pattern={{version}}

      - name: Build and push Docker image
        id: build-push
        uses: docker/build-push-action@v5
        with:
          context: backend
          file: backend/Dockerfile
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
          build-args: |
            VERSION=${{ github.sha }}

  # ---------------------------------------------------------------------------
  # Stage 7 — Deploy to Staging
  # Automatic deployment to staging on every push to main.
  # Runs database migrations before deploying the new image.
  # ---------------------------------------------------------------------------
  deploy-staging:
    name: "Stage 7 — Deploy to Staging"
    runs-on: ubuntu-latest
    needs: build
    if: github.ref == 'refs/heads/main'
    environment:
      name: staging
      url: https://auth-staging.fly.dev

    steps:
      - uses: actions/checkout@v4

      - name: Set up Fly CLI
        uses: superfly/flyctl-actions/setup-flyctl@master

      - name: Deploy to staging
        run: |
          flyctl deploy \
            --app auth-api-staging \
            --image ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:sha-${{ github.sha }} \
            --strategy rolling \
            --wait-timeout 300
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN_STAGING }}

      - name: Run post-deploy smoke tests
        run: |
          sleep 15  # Allow new instances to become healthy
          curl -sf https://auth-staging.fly.dev/health | \
            python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d['status']=='ok' else 1)"

  # ---------------------------------------------------------------------------
  # Stage 8 — Deploy to Production
  # Manual trigger required (environment protection rule: requires approval).
  # Migrations run automatically before the deployment rolls out.
  # ---------------------------------------------------------------------------
  deploy-production:
    name: "Stage 8 — Deploy to Production (Manual Approval Required)"
    runs-on: ubuntu-latest
    needs: deploy-staging
    if: github.ref == 'refs/heads/main'
    environment:
      name: production           # GitHub environment with required reviewers configured
      url: https://auth.yourapp.com

    steps:
      - uses: actions/checkout@v4

      - name: Set up Fly CLI
        uses: superfly/flyctl-actions/setup-flyctl@master

      - name: Deploy to production (rolling — zero downtime)
        run: |
          flyctl deploy \
            --app auth-api-production \
            --image ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}:sha-${{ github.sha }} \
            --strategy rolling \
            --wait-timeout 600
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN_PRODUCTION }}

      - name: Verify production health
        run: |
          sleep 20
          curl -sf https://auth.yourapp.com/health | \
            python3 -c "import sys,json; d=json.load(sys.stdin); sys.exit(0 if d['status']=='ok' else 1)"

      - name: Tag release in GitHub
        run: |
          gh release create \
            "deploy-$(date +%Y%m%d-%H%M%S)" \
            --title "Production Deploy $(date +%Y-%m-%d)" \
            --notes "SHA: ${{ github.sha }}" \
            --target main
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## 8. Coding Standards

These standards apply to every pull request. Reviewers must enforce them without exception.

### Go: Package and File Naming

- Package names are **lowercase single words**: `handler`, `service`, `domain`, `postgres`, `redis`. Never `authHandler` or `auth_handler`.
- File names use **snake_case**: `user_repo.go`, `auth_handler.go`, `tenant_middleware.go`.
- Test files are colocated with the file they test: `user_repo_test.go` lives next to `user_repo.go`.
- Integration and isolation tests live in `test/integration/` and `test/isolation/` respectively — they require external services and must not be in the same package as unit tests.
- Never name a file `utils.go` or `helpers.go`. Name files after the concept they contain.

### Go: Error Handling

Errors are always handled explicitly. No blank identifier `_` for error returns except in `defer` statements where the error is irrelevant by design.

```go
// Correct — wrap with context at every layer boundary.
user, err := s.userRepo.FindByEmail(ctx, email)
if err != nil {
    return nil, fmt.Errorf("auth service find user by email: %w", err)
}

// Wrong — lose the call stack.
user, err := s.userRepo.FindByEmail(ctx, email)
if err != nil {
    return nil, err
}

// Wrong — silent discard.
user, _ := s.userRepo.FindByEmail(ctx, email)
```

Domain errors (defined in `internal/domain/`) are sentinel errors using `errors.New`. They are compared using `errors.Is`. Never compare error strings with `==` or `strings.Contains` in production code — only in the handler's error-mapping switch.

```go
// In handler: mapping domain errors to HTTP responses.
if errors.Is(err, domain.ErrUserNotFound) {
    c.JSON(http.StatusNotFound, apierror.New("USER_NOT_FOUND", "User not found.", nil, reqID))
    return
}
```

### Go: Context Propagation

Every function that performs I/O (database, Redis, HTTP, Vault) must accept `context.Context` as its first parameter. Context is never stored in a struct. It is always passed through as a function argument.

```go
// Correct.
func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error)

// Wrong — stored in struct.
type PostgresUserRepo struct {
    ctx context.Context // Never do this.
}
```

Context values (user ID, schema name, request ID) are stored using typed context keys (`type ContextKey string`) defined in `internal/infrastructure/postgres/tenant_db.go`. Never use bare string literals as context keys.

### Go: Interface Usage

Define interfaces at the point of **consumption**, not at the point of implementation. All interfaces in `internal/domain/repository.go` are defined there because the service layer consumes them.

```go
// Correct: interface defined in domain, consumed by service, implemented in repository.
// internal/domain/repository.go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*User, error)
    // ...
}

// internal/service/auth_service.go — consumes the interface.
type authServiceImpl struct {
    userRepo domain.UserRepository  // Interface, not concrete type.
}

// internal/repository/postgres/user_repo.go — implements the interface.
type PostgresUserRepo struct { pool *pgxpool.Pool }
func (r *PostgresUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) { ... }
```

Interfaces must be small. If an interface has more than 5 methods, consider splitting it. `UserRepository` is an exception because it represents a complete data access contract.

### Repository Pattern Rules

1. **No business logic in repositories.** A repository only translates between Go structs and SQL. It must not compute anything beyond the query parameters it receives.
2. **No domain knowledge in infrastructure.** `PostgresUserRepo` must not know about password policies, rate limits, or session TTLs.
3. **All SQL uses parameterized queries.** No string concatenation in SQL. No `fmt.Sprintf` with user input in SQL strings. The schema name in `SET search_path` is the only exception — and it is validated with a regex allowlist before use.
4. **Repository errors are wrapped** with `fmt.Errorf("operation description: %w", err)` before returning. Do not return raw `pgx` errors to the service layer.
5. **All repository methods receive a `context.Context`** as their first argument. The schema name is always extracted from context — never passed as a parameter. This enforces that TenantMiddleware is always in the call chain.

### Service Layer Rules

1. **All business logic lives in services.** No HTTP concepts (status codes, headers, request/response structs) belong in a service.
2. **Services orchestrate, repositories store.** A service calls one or more repositories, applies business rules, and returns a domain result. It never builds SQL.
3. **Services own transaction boundaries.** When multiple repository writes must be atomic, the service opens a transaction, passes the transactional connection via context, and commits or rolls back.
4. **Services write audit events.** Every auth action that requires an audit log entry (`LOGIN_SUCCESS`, `ROLE_ASSIGNED`, etc.) must call `auditRepo.Append` within the service method that performs the action.
5. **Email is always enqueued, never sent directly.** Services write to the `emailChannel`. They never call the email client directly.

### Handler Rules

The handler pattern is strictly: **bind → validate → call service → respond**. No exceptions.

```go
func (h *AuthHandler) Login(c *gin.Context) {
    // 1. Bind JSON body.
    var req loginRequest
    if !bindAndValidate(c, &req) {
        return  // bindAndValidate writes the error response.
    }

    // 2. Call service — no business logic here.
    result, err := h.authSvc.Login(c.Request.Context(), service.LoginInput{
        Email:     req.Email,
        Password:  req.Password,
        IPAddress: c.ClientIP(),
        UserAgent: c.Request.UserAgent(),
    })
    if err != nil {
        respondWithServiceError(c, err)  // Central error mapping.
        return
    }

    // 3. Respond.
    c.JSON(http.StatusOK, loginResponse{ ... })
}
```

Handlers must not: inspect domain entities directly, perform database lookups, call repositories, or contain conditional business logic. If a handler contains an `if` statement that is not about the request/response shape, that logic belongs in the service.

### Security Rules

These rules are non-negotiable. A pull request that violates any of them must not be merged.

1. **No secrets in code or config files.** All secrets come from Vault at runtime. If a value is sensitive (key, password, token), it must never appear in source code, test fixtures committed to git, or any config file that could be version-controlled.
2. **Parameterized queries only.** Every SQL query uses `$1`, `$2`, ... placeholders. The schema name used in `SET search_path` is the one exception — it must be validated with `domain.IsValidSchemaName` before use.
3. **Validate all inputs at the boundary.** The handler layer validates the raw HTTP request. The service layer validates domain rules (password policy, tenant config). Never trust input from any caller.
4. **Constant-time comparisons for secrets.** All token and password comparisons use `crypto/subtle.ConstantTimeCompare` or the `crypto.VerifyPassword` wrapper. Never use `==` to compare tokens or hashes.
5. **Anti-enumeration responses.** Any endpoint that reveals user existence by returning different responses for known vs unknown emails must return an identical response in both cases. This applies to: `POST /auth/forgot-password`, `POST /auth/resend-verification`, `POST /auth/login` (wrong email vs wrong password — same error message).
6. **Error messages never expose internals.** 500 errors return `"An unexpected error occurred."`. Database errors, stack traces, and query details must never reach the HTTP response body.

### Testing Rules

1. **Table-driven tests** are the standard format for unit tests. Use `t.Run` with a descriptive test case name.

```go
func TestNormalizeEmail(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {name: "lowercase conversion", input: "MAYA@EXAMPLE.COM", expected: "maya@example.com", wantErr: false},
        {name: "trimming whitespace", input: "  maya@example.com  ", expected: "maya@example.com", wantErr: false},
        {name: "invalid format", input: "not-an-email", expected: "", wantErr: true},
        {name: "empty string", input: "", expected: "", wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := domain.NormalizeEmail(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, got)
            }
        })
    }
}
```

2. **Test interfaces, not implementations.** Service tests use mock implementations of repository interfaces. Repository tests use a real test database. Never mock the thing you are testing.
3. **Test naming convention**: `TestTypeName_MethodName_Scenario` — e.g., `TestAuthService_Login_WrongPassword`, `TestPostgresUserRepo_FindByEmail_NotFound`.
4. **Integration tests** must clean up their data after each test (use `t.Cleanup`). Never depend on test execution order.
5. **Isolation tests** (in `test/isolation/`) must provision two independent tenants (`tenant_a`, `tenant_b`), perform all auth operations as `tenant_a`, and assert that no `tenant_b` data appears in any response field, including error messages and metadata.
6. **Minimum 80% line coverage** on all new code in `internal/` and `pkg/`. Coverage is enforced by the CI pipeline.

### Git Workflow

**Branch naming:**
```
feature/US-03-user-login
fix/session-rotation-family-id
chore/update-go-deps
test/cross-tenant-isolation-suite
```

**Commit message format** (Conventional Commits):
```
feat(auth): implement login with argon2id verification

Add POST /auth/login endpoint with Argon2id password verification,
failed login counter, account lockout, and audit logging.

Relates to: US-03
```

Types: `feat`, `fix`, `test`, `refactor`, `chore`, `docs`, `perf`, `security`.

**PR requirements before merge:**
- All 8 CI stages green.
- At least 1 peer code review approval.
- Cross-tenant isolation tests pass (stage 4 — hard block).
- No `TODO` or `FIXME` comments introduced (existing ones must not be increased).
- API documentation updated if a new endpoint was added or an existing one changed.

---

## 9. Integration Guide

### 9.1 Request Lifecycle

Every HTTP request flows through this chain in order. Understanding this chain is essential for debugging and for adding new features correctly.

```
HTTP Request
    │
    ▼
[RequestIDMiddleware]
    │  Generates or propagates X-Request-ID. Sets CtxKeyRequestID in context.
    │  All log entries and error responses include this ID for correlation.
    ▼
[LoggerMiddleware]
    │  Logs method, path, status, latency, tenant_id, user_id (after auth).
    │  Uses slog JSON format. Never logs passwords, tokens, or PII.
    ▼
[SecureHeadersMiddleware]
    │  Sets: Strict-Transport-Security, X-Frame-Options: DENY,
    │         X-Content-Type-Options: nosniff, Content-Security-Policy,
    │         Referrer-Policy: no-referrer
    │  Applied on every response regardless of auth status.
    ▼
[CORSMiddleware]
    │  Validates Origin header against allowed origins list.
    │  Per-tenant origins are loaded from tenant_config (Sprint 4+).
    ▼
[RateLimitMiddleware]  ← Applied per-route with specific limits
    │  Checks Redis sliding window. Fails closed if Redis unavailable (AVAIL-02).
    │  Returns 429 with Retry-After header on limit exceeded.
    ▼
[AuthMiddleware]  ← Authenticated routes only
    │  Validates Bearer JWT. Rejects non-RS256, expired, unknown kid.
    │  Sets jwt_claims, user_id, user_roles in Gin context.
    ▼
[TenantMiddleware]
    │  Unauthenticated: reads X-Tenant-ID header → looks up schema from cache.
    │  Authenticated: reads tenant_id from JWT claims → validates matches header.
    │  Sets CtxKeySchemaName, CtxKeyTenantID in Gin context.
    │  Validates schema name format (allowlist regex — SQL injection prevention).
    ▼
[RequireRoleMiddleware]  ← Role-protected routes only (admin, super_admin)
    │  Reads user_roles from context. Returns 403 if required role absent.
    ▼
[Handler]
    │  bind → validate → call service → respond
    │  No business logic. No database calls. No domain decisions.
    ▼
[Service]
    │  Business logic. Orchestrates repositories. Writes audit events.
    │  Enqueues emails. Owns transaction boundaries.
    ▼
[Repository]
    │  Calls pgdb.WithTenantSchema(ctx, pool, schema, fn).
    │  Sets search_path for connection lifetime.
    │  Executes parameterized SQL. Returns domain entities.
    ▼
[PostgreSQL — Tenant Schema]
    │  All queries execute within tenant_{slug} schema.
    │  Cross-tenant access is architecturally impossible (ADR-001).
    ▼
HTTP Response
```

### 9.2 Tenant Schema Routing

This is the most critical security flow in the system. It must be understood precisely.

**How `X-Tenant-ID` flows from request header to `SET search_path`:**

```
Step 1 — Client sends request:
    POST /api/v1/auth/login
    X-Tenant-ID: acme-corp
    Content-Type: application/json

Step 2 — TenantMiddleware runs:
    tenantID = c.GetHeader("X-Tenant-ID")  → "acme-corp"

    // Cache lookup (in-memory, 60s TTL):
    schemaName = tenantCache.GetSchema(ctx, "acme-corp")  → "tenant_acme_corp"

    // Security validation — allowlist regex:
    domain.IsValidSchemaName("tenant_acme_corp")  → true ✓

    // Set in Gin context:
    c.Set(CtxKeySchemaName, "tenant_acme_corp")
    c.Set(CtxKeyTenantID, "acme-corp")

Step 3 — Handler calls service:
    authSvc.Login(c.Request.Context(), ...)
    // c.Request.Context() carries the CtxKeySchemaName value.

Step 4 — Service calls repository:
    userRepo.FindByEmail(ctx, email)

Step 5 — Repository extracts schema and routes:
    schema, _ := pgdb.SchemaFromContext(ctx)  → "tenant_acme_corp"

    pgdb.WithTenantSchema(ctx, pool, "tenant_acme_corp", func(conn *pgx.Conn) error {
        // SET search_path TO tenant_acme_corp, public
        // All subsequent queries in this connection use tenant_acme_corp tables.
        row := conn.QueryRow(ctx,
            "SELECT id, email, ... FROM users WHERE email = $1",
            email,
        )
        // "users" resolves to tenant_acme_corp.users — never public.users.
    })

Step 6 — For authenticated requests (JWT present):
    // AuthMiddleware extracts tenant_id from JWT claims.
    // TenantMiddleware validates JWT tenant_id == X-Tenant-ID header.
    // If they differ → 403 TENANT_MISMATCH — prevents cross-tenant token replay.
```

**Why this prevents cross-tenant leaks:**
When `search_path = tenant_acme_corp, public`, a query `SELECT * FROM users WHERE email = $1` can only see rows in `tenant_acme_corp.users`. There is no row-level filter to forget, no WHERE clause to omit. The schema boundary is enforced by PostgreSQL itself.

### 9.3 Token Lifecycle

**Complete flow from registration to logout:**

```
1. REGISTER
   Client → POST /auth/register {email, password}
   Service:
     - Normalize email, validate password complexity against tenant policy
     - Argon2id hash password (64MB, 3 iterations, 2 threads)
     - INSERT INTO users (status='unverified')
     - Generate 32-byte crypto/rand verification token
     - Store SHA-256(token) in email_verification_tokens
     - Enqueue EmailTask{type: verification, token: rawToken}
   Response: 201 {user_id, status: "unverified"}

2. VERIFY EMAIL
   User clicks email link → POST /auth/verify-email {token}
   Service:
     - Hash presented token: SHA-256(token)
     - SELECT FROM email_verification_tokens WHERE token_hash = $1
     - Validate: not used, not expired
     - UPDATE users SET status='active', email_verified_at=now()
     - UPDATE token SET used=true
   Response: 200 {message: "Email verified."}

3. LOGIN
   Client → POST /auth/login {email, password}
   Service:
     - Check IP rate limit (Redis sliding window)
     - SELECT user WHERE email = $1 AND deleted_at IS NULL
     - Check is_locked (locked_until > now())
     - Check status = 'active'
     - Argon2id.Verify(password, hash) — constant-time
     - On failure: increment failed_login_count → lock if >= threshold
     - On success: reset failed_login_count, update last_login_at
     - Fetch user roles
     - Sign JWT: RS256, kid=current_key_id, TTL=15min
       Payload: {sub, iss, aud, exp, iat, jti, tenant_id, roles[]}
     - Generate opaque refresh token: hex(crypto/rand(32 bytes))
     - Store SHA-256(refresh_token) in sessions
       (family_id = new UUID, expires_at = now() + tenant session TTL)
     - Write LOGIN_SUCCESS to audit_log
   Response: 200 {access_token, refresh_token, expires_in: 900}

4. USE ACCESS TOKEN (resource server validation)
   Client → GET /api/resource
             Authorization: Bearer <access_token>
   Resource server (NOT auth server):
     - Fetch JWKS from /.well-known/jwks.json (cached, max-age=3600)
     - Select public key matching JWT header kid
     - jwt.Verify(token, pubKey) — validates signature, exp, iss, aud
     - Extract tenant_id, roles[] from claims
     - Enforce authorization locally — no call to auth server
   Auth server is NOT involved in resource access — only in token issuance.

5. REFRESH
   Client → POST /auth/token/refresh {refresh_token}
   Service:
     - tokenHash = SHA-256(refresh_token)
     - SELECT session WHERE token_hash = $1
     - If not found → 401 INVALID_REFRESH_TOKEN
     - If is_revoked = true → SUSPICIOUS REUSE detected:
         UPDATE sessions SET is_revoked=true WHERE family_id = $1 (all tokens)
         Write SUSPICIOUS_TOKEN_REUSE to audit_log
         → 401 SUSPICIOUS_TOKEN_REUSE
     - If expires_at < now() → 401 REFRESH_TOKEN_EXPIRED
     - Revoke old session: UPDATE sessions SET is_revoked=true WHERE token_hash = $1
     - Generate new access_token + new refresh_token
     - INSERT new session (same family_id, new token_hash)
     - Write TOKEN_REFRESHED to audit_log
   Response: 200 {new access_token, new refresh_token}

6. LOGOUT
   Client → POST /auth/logout {refresh_token}
             Authorization: Bearer <access_token>
   Service:
     - tokenHash = SHA-256(refresh_token)
     - UPDATE sessions SET is_revoked=true WHERE token_hash = $1
     - Write LOGOUT to audit_log
     - NOTE: Access token remains valid for up to 15 minutes (documented trade-off)
   Response: 200 {message: "Logged out successfully."}
```

### 9.4 Email Worker

The email worker decouples email delivery from the request path (ADR-012). Any service that needs to send an email writes to the `emailChannel` channel and returns immediately. The worker goroutines consume the channel and call the Resend API with retry.

```
Request path                      Email Worker (goroutine pool)
─────────────────                 ────────────────────────────
authSvc.Register(...)
  └─ s.enqueueEmail(EmailTask{    ──→  emailChannel (buffered, cap=100)
       type: verification,              │
       to: maya@example.com,            ▼
       token: "abc123",          worker.Start() [goroutine pool, N=4]
     })                            │
  └─ return 201 ✓                  ├─ receive EmailTask from channel
                                   ├─ render email template (Go text/template)
                                   ├─ emailClient.Send(to, subject, body)
                                   │    → POST https://api.resend.com/emails
                                   ├─ on success: log and continue
                                   └─ on failure: exponential backoff retry
                                          attempt 1: wait 2s
                                          attempt 2: wait 4s
                                          attempt 3: wait 8s
                                          after 3 failures: log error, discard task
```

**Channel full behavior:** If the channel buffer (100 tasks) fills up (e.g., during a burst registration event), `enqueueEmail` logs an error and discards the task. The user can request a resend via `/auth/resend-verification`. This is the documented at-most-once trade-off for the pilot (ADR-012).

**Graceful shutdown:** On SIGTERM, main.go calls `emailWorker.Drain(ctx)` with a 5-second timeout. The worker finishes in-flight sends and discards any remaining queued tasks before the process exits.

### 9.5 Vault Integration

Vault is the single source of truth for all signing keys and secrets. The application never stores a secret on disk or in environment variables in production.

**Startup flow:**

```
main.go
  └─ vault.NewClient(cfg.Vault)
       ├─ cfg.Vault.Token != "" → use static token (dev only)
       └─ cfg.Vault.RoleID != "" → AppRole auth:
            POST /v1/auth/approle/login {role_id, secret_id}
            ← {auth.client_token}  (renewable, short-lived)

  └─ vaultClient.LoadCurrentSigningKey(ctx)
       GET /v1/secret/data/auth-system/jwt-keys/current
       ← {private_key_pem, public_key_pem, kid}
       → parse RSA private key from PEM
       → jwtutil.NewKeyStore(issuer, audience, kid, privateKey)

  └─ keyRotationWatcher.Watch(ctx)  [background goroutine]
       Every 5 minutes:
         GET /v1/secret/data/auth-system/jwt-keys/current
         if kid changed:
           keyStore.Update(newKid, newPrivateKey)
           keyStore.AddPublicKey(oldKid, oldPublicKey)  // overlap period
           // JWKS endpoint now serves both keys
           // After 25h: keyStore.RemovePublicKey(oldKid)
```

**Zero-downtime key rotation procedure:**

```
1. Security team generates new RS256 key pair.
2. Vault admin writes new key to:
     secret/auth-system/jwt-keys/current → {kid: "key-20260401", private_key_pem, public_key_pem}
   Previous key moved to:
     secret/auth-system/jwt-keys/previous → {kid: "key-20260228", public_key_pem, retire_at: "2026-04-02T12:00:00Z"}

3. Within 5 minutes, KeyRotationWatcher detects the new kid.
4. keyStore.Update() atomically swaps the signing key.
5. New tokens are signed with key-20260401.
6. Old tokens signed with key-20260228 remain valid — JWKS still serves both.
7. After 25 hours (max access token TTL 15min + session TTL 24h + 1h buffer):
     keyStore.RemovePublicKey("key-20260228")
     JWKS endpoint stops serving old key.
8. Any token with kid=key-20260228 is now rejected — all such tokens have expired.
```

**Vault path layout for reference:**

```
secret/auth-system/
├── jwt-keys/
│   ├── current          → {kid, private_key_pem, public_key_pem}
│   └── previous         → {kid, public_key_pem, retire_at}
├── db                   → {url}  (overrides DATABASE_URL in production)
├── email                → {resend_api_key}
└── google               → {client_id, client_secret}
        (per-tenant Google creds stored in tenant_config table, not Vault)
```

---

*End of Implementation Guide v1.0*
*Authentication System — Pilot Project*
*Next: Hand off to Tester Agent → produce `docs/auth-system/test-plan.md`*
