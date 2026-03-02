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

### Overall: 91 tests · 0 failures

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
