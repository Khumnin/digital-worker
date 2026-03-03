# 🤖 Digital Worker — AI Agent Squad

An AI-powered agentic project where **5 role-based agents** collaborate to design, plan, architect, implement, and test a production-grade **Multi-Tenant Authentication System**.

---

## 📋 Project Overview

This project uses Claude-based AI agents, each playing a specific professional role, to produce a complete software delivery pipeline — from product requirements through implementation guides and quality assurance test plans.

### 🎯 Current Project: Multi-Tenant Authentication System

A cloud-native, API-first authentication service supporting multi-tenant isolation with enterprise-grade security.

**Key Constraints (Non-Negotiable)**
| Decision | Detail |
|----------|--------|
| ADR-001 | Schema-per-tenant PostgreSQL |
| ADR-002 | One user = one tenant |
| ADR-003 | API-only, no hosted UI |
| ADR-004 | JWT RS256 (15min) + opaque refresh tokens |

---

## 🧑‍💼 Agent Squad

Agents live in `.claude/agents/` and are invoked to generate and maintain project documentation.

| Agent | Role | Methodology |
|-------|------|-------------|
| **Product Owner** | Requirements & Backlog | INVEST stories, MoSCoW, Gherkin, Agile ceremonies |
| **Project Manager** | Planning & Coordination | Hybrid methodology, WBS, RACI, Risk Register, Budget |
| **Solution Architect** | Architecture & Design | Clean Architecture, C4 Diagrams, DDD, Cloud-Native |
| **Developer** | Implementation Guide | Next.js + Go/Gin, Handler→Service→Repository pattern |
| **Tester** | Quality Assurance | ISTQB, ISO 25010, Playwright, Quality Gates |

---

## 🏗️ Tech Stack

```
Backend       Go + Gin
Frontend      Next.js (TypeScript)
Database      PostgreSQL (schema-per-tenant)
Cache/Queue   Redis
Auth Library  ory/fosite
Secrets       HashiCorp Vault
Email         Resend
Deployment    Fly.io
```

---

## 📁 Project Structure

```
digital-worker/
├── .claude/
│   └── agents/
│       ├── product-owner/
│       ├── project-manager/
│       ├── solution-architect/
│       ├── developer/
│       └── tester/
├── docs/
│   └── auth-system/
│       ├── prd.md                      # Product Requirements Document (42KB)
│       ├── project-management-plan.md  # Project Management Plan (100KB)
│       ├── solution-architecture.md    # Solution Architecture (131KB)
│       ├── implementation-guide.md     # Developer Implementation Guide (197KB)
│       └── test-plan.md                # QA Test Plan (235KB)
├── CLAUDE.md                           # Agent squad context & pipeline status
└── README.md
```

---

## ✅ Pipeline Status

| Phase | Agent | Status | Output |
|-------|-------|--------|--------|
| 1. Requirements | Product Owner | ✅ Complete | `docs/auth-system/prd.md` |
| 2. Planning | Project Manager | ✅ Complete | `docs/auth-system/project-management-plan.md` |
| 3. Architecture | Solution Architect | ✅ Complete | `docs/auth-system/solution-architecture.md` |
| 4. Implementation | Developer | ✅ Complete | `docs/auth-system/implementation-guide.md` |
| 5. Testing | Tester | ✅ Complete | `docs/auth-system/test-plan.md` |

---

## 🧪 Test Results

### Sprint 1 — Core Auth Flows

**Unit Tests** · `go test ./...` · 67/67 passed

| Package | Tests | Status |
|---------|------:|--------|
| `internal/domain` | 33 | ✅ Pass |
| `pkg/apierror` | 7 | ✅ Pass |
| `pkg/crypto` | 14 | ✅ Pass |
| `pkg/jwtutil` | 13 | ✅ Pass |
| `pkg/validator` | 15 | ✅ Pass |

**E2E API Tests** · Playwright · 15/15 passed

| Test | Story | Status |
|------|-------|--------|
| 201 on valid registration | US-01 | ✅ |
| 409 on duplicate email | US-01 | ✅ |
| 422 on weak password | US-01 | ✅ |
| 422 on invalid email format | US-01 | ✅ |
| 401 on wrong credentials — ambiguous error message | US-03 | ✅ |
| Response contains no sensitive data on failure | US-03 | ✅ |
| 401 on invalid refresh token | US-04 | ✅ |
| 401 on empty refresh token | US-04 | ✅ |
| 401 logout without auth token | US-05 | ✅ |
| 200 for registered email (anti-enumeration) | US-06 | ✅ |
| 200 for unknown email (anti-enumeration) | US-06 | ✅ |
| Response body identical for both | US-06 | ✅ |
| Tenant A token cannot list tenant B users | US-08b | ✅ |
| No tenant B data in tenant A response | US-08b | ✅ |
| Cross-tenant token swap returns 401/403 — never 200 | US-08b | ✅ |

### Sprint 2 — Sessions, Security & Audit

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| US-04 | Token rotation + family revocation on reuse detection | ✅ |
| US-05 | Single-device and all-devices logout | ✅ |
| US-14 | Redis sliding-window rate limiting + account lockout (5 attempts / 15 min) | ✅ |
| US-15 | Audit log — 19 event types wired, filterable paginated read API | ✅ |
| M15 | Secure headers — HSTS, CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy | ✅ |

**Audit events wired (19 types)**

`USER_REGISTERED` · `EMAIL_VERIFICATION_SENT` · `EMAIL_VERIFIED` · `LOGIN_SUCCESS` · `LOGIN_FAILURE` · `ACCOUNT_LOCKED` · `LOGOUT` · `LOGOUT_ALL` · `TOKEN_REFRESHED` · `SUSPICIOUS_TOKEN_REUSE` · `PASSWORD_RESET_REQUESTED` · `PASSWORD_RESET_COMPLETED` · `PASSWORD_CHANGED` · `USER_INVITED` · `USER_DISABLED` · `USER_DELETED` · `ROLE_ASSIGNED` · `ROLE_UNASSIGNED` · `OAUTH_CLIENT_CREATED`

**All Sprint 1 E2E tests remain green — 15/15 passed after Sprint 2 changes.**

### Sprint 3 — Multi-Tenancy Foundation

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| US-07a | Tenant provisioning — schema migration runner + admin invite email | ✅ |
| US-07b | Per-tenant migration runner (golang-migrate, `file://migrations/tenant`) | ✅ |
| US-07c | Tenant API credentials — generate & rotate M2M client_id/client_secret | ✅ |
| US-08a | Schema-per-tenant routing — `search_path` set per request via `X-Tenant-ID` | ✅ |
| US-08b | Cross-tenant isolation — 12 test cases, all passing | ✅ |

**E2E API Tests** · Playwright · 24/24 passed (12 new US-08b isolation tests)

| Test | Story | Status |
|------|-------|--------|
| TC-08b-01: tenant A token cannot list tenant B users | US-08b | ✅ |
| TC-08b-02: no tenant B data appears in tenant A response | US-08b | ✅ |
| TC-08b-03: cross-tenant token swap returns 401/403 — never 200 | US-08b | ✅ |
| TC-08b-04: request without X-Tenant-ID header is rejected | US-08b | ✅ |
| TC-08b-05: unknown tenant ID returns 4xx — never leaks data | US-08b | ✅ |
| TC-08b-06: response headers do not expose tenant schema | US-08b | ✅ |
| TC-08b-07: error body for tenant A does not contain tenant B identifiers | US-08b | ✅ |
| TC-08b-08: password reset response identical regardless of tenant membership | US-08b | ✅ |
| TC-08b-09: refresh token issued for tenant A cannot be replayed against tenant B | US-08b | ✅ |
| TC-08b-10: registering in tenant A does not create a user visible in tenant B | US-08b | ✅ |
| TC-08b-11: login failure message identical for both tenants — no enumeration | US-08b | ✅ |
| TC-08b-12: malformed tenant ID (SQL injection attempt) is rejected | US-08b | ✅ |

**All previous E2E tests remain green — 24/24 passed after Sprint 3 changes.**

### Sprint 4 — RBAC, JWT Claims & Token Introspection

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| US-09 | Assign / unassign tenant-scoped roles — audit events wired | ✅ |
| US-10 | JWT now carries real per-user `roles[]` claim (fetched from DB at login) | ✅ |
| M16 | Admin user management — invite, disable, delete, list users | ✅ |
| M17 | Token introspection — RFC 7662 `POST /oauth/introspect` | ✅ |
| M18 | JWKS endpoint — RFC 7517 `GET /.well-known/jwks.json` | ✅ |

**Notes**
- US-10: login now calls `roleRepo.GetUserRoles()` per request; falls back to `["user"]` on error; logs a warning if a user holds > 20 roles (JWT size concern per DoD)
- M17: verifies the bearer token signature + claims; returns `{"active": false}` for any invalid/expired token — no information leakage; accepts both JSON and form-encoded bodies (RFC 7662 §2.1)
- US-09, M16, M18 were already fully scaffolded from Sprint 1/2 and required no new code

**All E2E tests remain green — 24/24 passed after Sprint 4 changes.**

### Sprint 5 — OAuth 2.0 Authorization Code + PKCE

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| US-11a | OAuth client registration — `POST /admin/oauth/clients` | ✅ |
| US-11b | Authorization endpoint — `GET /oauth/authorize` with mandatory state + PKCE S256 | ✅ |
| US-11c | Token exchange — `POST /oauth/token` with PKCE S256 verification, single-use codes, session issuance | ✅ |

**Notes**
- No external OAuth library — Authorization Code + PKCE S256 flow implemented directly using existing jwt/crypto utilities
- US-11c security: code reuse detection (mark-used before token issuance), PKCE S256 mandatory (`plain` method rejected with 400), single-use codes enforced at DB level
- API-only per ADR-003 — `/authorize` returns `{"code", "state"}` JSON; consuming app handles redirect
- Token endpoint returns RFC 6749 §5.1 format: `access_token`, `refresh_token`, `token_type`, `expires_in`, `scope`
- Audit events wired: `OAUTH_CLIENT_CREATED`, `OAUTH_CODE_ISSUED`, `OAUTH_TOKEN_ISSUED`

**All E2E tests remain green — 24/24 passed after Sprint 5 changes.**

### Sprint 6 — OAuth M2M and Google Social Login

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| US-12 | Client Credentials Grant (M2M) — `POST /oauth/token` with `grant_type=client_credentials`; scope-bound JWT, no refresh token (RFC 6749 §4.4.3) | ✅ |
| US-13 | Google Social Login (OIDC) — `POST /auth/oauth/google` returns auth URL; `GET /auth/oauth/google/callback` exchanges code for tokens | ✅ |

**Notes**
- US-12: `client_id` + `client_secret` (SHA-256 verified); `client_credentials` must be in client's `grant_types`; JWT `sub = client_id`; no user context; audit `OAUTH_TOKEN_ISSUED` with `grant_type: client_credentials`
- US-13: full OIDC flow via standard `net/http` — no external OAuth library added; state token stored in Redis (one-time-use, 10 min TTL); ID token verified via Google tokeninfo endpoint
- US-13 account linking (ADR-006): if Google email matches existing password user → password verification required before link; if no password (social-only) → auto-link; new users auto-verified
- API-only per ADR-003: Initiate returns `{"auth_url": "..."}` (client handles redirect); Callback returns `{"access_token", "refresh_token", "token_type", "expires_in", "is_new_user"}`
- New audit events wired: `GOOGLE_LOGIN`, `GOOGLE_ACCOUNT_LINKED`
- Per-tenant Google credentials supported via `TenantConfig.GoogleClientID/Secret`; falls back to global `OAUTH_GOOGLE_CLIENT_ID/SECRET`

**New files added**
- `internal/domain/social_account.go` — `SocialAccount` struct + `SocialAccountRepository` interface
- `internal/repository/postgres/social_account_repo.go` — queries existing `oauth_social_accounts` table (migration 000009)
- `internal/service/google_service.go` — `GoogleService` interface + full OIDC implementation

**All unit tests remain green — 67/67 passed after Sprint 6 changes.**

### Sprint 7 — MFA and User Profile

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| S1 | TOTP MFA setup — enroll via QR code, confirm first code, receive 8 backup codes | ✅ |
| S1 | TOTP MFA verification on login — step-up after password; 202 when code missing | ✅ |
| S2 | Per-tenant MFA enforcement — admin toggle; unenrolled users blocked with 403 | ✅ |
| S5 | Self-service User Profile API — update name, change password, request email change | ✅ |

**Notes**
- S1: RFC 6238, SHA1, 6-digit, 30s window, ±1 window clock-drift tolerance (`pquerna/otp`); backup codes stored as SHA-256 hashes, consumed atomically
- S1 login step-up: returns **HTTP 202** `{"mfa_required": true}` when MFA enabled but no `totp_code` supplied; backup codes accepted as fallback
- S2: `PUT /admin/tenant/mfa` toggles `mfa_required`; login enforces enrollment — `MFA_ENROLLMENT_REQUIRED` (403) if user unenrolled on required tenant
- S5: `GET /users/me` now returns full profile (name, email, MFA status); `PUT /users/me` dispatches by fields — password change revokes all sessions
- New audit events wired: `MFA_ENABLED`, `MFA_DISABLED`, `MFA_VERIFIED`, `MFA_FAILED`, `MFA_ENFORCEMENT_CHANGED`, `PROFILE_UPDATED`

**New endpoints**
| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/users/me/mfa/generate` | Returns `otpauth://` URL + base32 secret for QR code |
| `POST` | `/users/me/mfa/confirm` | Verifies first TOTP code, enables MFA, returns backup codes (once) |
| `DELETE` | `/users/me/mfa` | Disables MFA after password verification |
| `PUT` | `/admin/tenant/mfa` | Admin toggle for per-tenant MFA enforcement |

**New files added**
- `internal/domain/mfa.go` — `MFABackupCode` struct + `MFARepository` interface
- `internal/repository/postgres/mfa_repo.go` — backup code CRUD against `mfa_backup_codes` table
- `internal/service/mfa_service.go` — TOTP enroll/confirm/verify/disable
- `internal/service/profile_service.go` — GetProfile, UpdateProfile, ChangePassword, RequestEmailChange
- `internal/handler/mfa_handler.go` — Generate, Confirm, Disable endpoints
- `migrations/tenant/000013_add_mfa_fields` — `mfa_totp_secret` column on users
- `migrations/tenant/000014_add_mfa_backup_codes` — `mfa_backup_codes` table

**New dependency:** `github.com/pquerna/otp v1.4.0`

**All unit tests remain green — 67/67 passed after Sprint 7 changes.**

### Sprint 8 — Hardening and Compliance

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| COMP-01 | GDPR right-to-erasure — self-service `DELETE /users/me` + admin erase; full 9-step PII wipe | ✅ |
| Pentest | TOTP brute-force protection — Redis rate limit 5 attempts / 15 min; 429 on breach | ✅ |
| OWASP | Request body size limit — 64 KB cap via `MaxBodySize` middleware | ✅ |
| OWASP | Password change invalidates outstanding OAuth codes | ✅ |
| OWASP | Generic 500 fallback — no internal detail leaks in error responses | ✅ |
| PERF-04 | Performance indexes — 5 partial/compound indexes on hot query paths | ✅ |
| AVAIL-04 | Dependency-aware health check — `GET /health` pings PostgreSQL + Redis; 503 on degraded | ✅ |

**GDPR erasure sequence (irreversible)**
1. Revoke all sessions
2. Delete MFA backup codes
3. Delete social account links (`oauth_social_accounts`)
4. Delete outstanding OAuth authorization codes
5. Anonymize user PII → tombstone values
6. Soft-delete user record
7. Anonymize audit log `actor_id` → nil UUID tombstone
8. Write `USER_ERASED` audit event

Self-service requires password confirmation. Admin erase requires `admin` role.

**Security hardening applied**
- `ErrTOTPRateLimited` → 429 on Login and MFA Confirm endpoints
- `middleware.MaxBodySize(64KB)` applied globally before all routes
- `codeRepo.DeleteByUserID` called on password change (OAuth codes invalidated)
- Health check actively validates DB + Redis connectivity; returns 503 if degraded

**New files added**
- `internal/handler/health_handler.go` — dependency-aware health check
- `internal/middleware/request_size.go` — 64 KB body size cap
- `migrations/tenant/000015_performance_indexes` — 5 indexes (email, session hash, audit log, OAuth codes)

**All unit tests remain green — 67/67 passed after Sprint 8 changes.**

### Sprint 9 — Launch Preparation and UAT

**Deliverables completed**

| Story | Description | Status |
|-------|-------------|--------|
| E2E regression | Playwright API test suites for all Sprints 4-8 flows | ✅ |
| Smoke tests | `00-smoke.spec.ts` — full happy-path skeleton for go/no-go gate | ✅ |
| Incident runbook | `docs/runbook.md` — SEV-1/2/3 playbooks + rollback procedure | ✅ |
| Go/no-go checklist | `docs/go-no-go.md` — 13 critical + 8 important gates + sign-off table | ✅ |

**New E2E test files (Playwright API project)**

| File | Sprint | Tests |
|------|--------|-------|
| `00-smoke.spec.ts` | 9 | Full happy-path: health, register, login, JWKS, introspect, headers |
| `04-rbac.spec.ts` | 4 | Admin role gates, token introspection RFC 7662, JWKS RFC 7517 |
| `05-oauth-pkce.spec.ts` | 5 | Client registration ACL, PKCE S256, code reuse, plain method rejection |
| `06-m2m.spec.ts` | 6 | Client credentials grant, no refresh_token in response, bad credentials |
| `07-mfa.spec.ts` | 7 | MFA generate/confirm auth gates, profile CRUD, password change |
| `09-gdpr.spec.ts` | 8 | Self-erasure auth + wrong password, health check postgres+redis |

**New documentation**
- `docs/runbook.md` — SEV-1 (service down), SEV-2 (Redis/Vault/email degraded), SEV-3 (perf), rollback commands
- `docs/go-no-go.md` — launch gate checklist; 13 Critical blockers + sign-off table

**Go/no-go Critical gates**
`go build` · 67 unit tests · Playwright E2E · `/health` 200 in prod · zero Critical/High pentest findings · GDPR erasure verified · Vault-sourced JWT keys · all migrations applied · rate limiting active · secure headers · cross-tenant isolation · audit log writing

### Overall: 67 unit tests · 0 failures · 6 E2E suites ready for live environment

---

## 🚀 Getting Started

### 1. Start Local Infrastructure
```bash
docker compose up -d
```
This brings up: **PostgreSQL** · **Redis** · **HashiCorp Vault** · **Mailhog**

### 2. Run Migrations
```bash
cd auth-system/backend
go run ./cmd/migrate/main.go -scope=global -direction=up
```

### 3. Start the API
```bash
go run ./cmd/api/main.go
```

### 4. Run Tests
```bash
# Unit tests
go test ./...

# E2E tests
cd auth-system/tests && npx playwright test --project=api
```

---

## 📖 Documentation

| Document | Description |
|----------|-------------|
| [PRD](docs/auth-system/prd.md) | Product requirements, user stories, acceptance criteria |
| [Project Plan](docs/auth-system/project-management-plan.md) | WBS, RACI, risk register, timeline, budget |
| [Architecture](docs/auth-system/solution-architecture.md) | C4 diagrams, ADRs, data models, API contracts |
| [Implementation Guide](docs/auth-system/implementation-guide.md) | Code scaffold, patterns, Go/Next.js setup |
| [Test Plan](docs/auth-system/test-plan.md) | Test strategy, cases, quality gates, Playwright setup |

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.
