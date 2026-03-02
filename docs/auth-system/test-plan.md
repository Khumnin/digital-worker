# Authentication System — Test Plan

**Title:** Authentication System Test Plan
**Version:** 1.0
**Date:** 2026-02-28
**Status:** Draft
**Author:** QA Engineering / Test Architecture Team
**Project:** Auth System Pilot
**Stack:** Go + Gin + PostgreSQL (schema-per-tenant) + Redis + ory/fosite + JWT RS256

---

# Section 1: Test Strategy Overview

## 1.1 Testing Scope

### In Scope

| Area | Coverage |
|---|---|
| User Registration and Email Verification | Full functional + security coverage |
| User Login (all paths, brute force, lockout) | Full functional + security coverage |
| Session Management (JWT issuance, refresh rotation, reuse detection) | Full functional + security coverage |
| Password Reset (anti-enumeration, timing parity) | Full functional + security + timing coverage |
| Logout (single device, all devices, idempotency) | Full functional coverage |
| Multi-Tenancy — Schema-per-Tenant Provisioning | Full functional + isolation security coverage |
| Cross-Tenant Isolation (ADR-001 highest risk) | Full functional + security + penetration-style coverage |
| RBAC + JWT Claims | Full functional + security coverage |
| OAuth 2.0 Authorization Server (PKCE, client credentials, M2M) | Full functional + security coverage |
| Social Login — Google OIDC | Full functional + security (CSRF) coverage |
| Audit Log (append-only, tenant-scoped) | Full functional + integrity coverage |
| Rate Limiting (Redis sliding window) | Full functional + boundary coverage |
| API contract validation (request/response shapes, HTTP status codes) | Full functional coverage |
| Token cryptographic properties (RS256, expiry, claims) | Full security coverage |
| HashiCorp Vault secret injection | Integration-level smoke tests |
| Resend email delivery (verification, password reset) | Integration-level smoke tests (mock in CI) |

### Out of Scope

| Area | Reason |
|---|---|
| Frontend UI testing | ADR-003: API-only, no hosted UI |
| Browser-rendered login flows | ADR-003: no hosted login UI exists |
| Native mobile app testing | Not in scope for this release |
| Load / performance benchmarking (beyond rate-limit thresholds) | Separate performance test plan required |
| Infrastructure provisioning (Terraform, Fly.io config) | DevOps ownership |
| Third-party OAuth provider internals (Google OIDC IdP) | External dependency; mocked at integration boundary |
| Resend deliverability / spam testing | Vendor responsibility |
| SOC 2 audit evidence collection | Post-launch (ADR-008) |
| Penetration testing (full red team) | Separate engagement; Sprint 8 |

---

## 1.2 ISTQB Test Process Activities Mapped to Project Phases

| ISTQB Activity | Description | Project Phase | Artifacts Produced |
|---|---|---|---|
| **Planning** | Define scope, strategy, resource requirements, schedule, entry/exit criteria, risk register | Sprint 0 (Architecture Spike) | This test plan, risk register, test environment spec, QA schedule in WBS |
| **Monitoring and Control** | Track test execution progress, defect burn-down, coverage metrics, gate decisions | Sprint 1 — Sprint 9 (continuous) | Weekly QA status report, defect aging dashboard, coverage delta per sprint |
| **Analysis** | Derive testable conditions from requirements (prd.md, US stories, ADRs) | Sprint 0 — Sprint 1 kickoff | Traceability matrix (story → test condition → TC-ID), risk-based priority list |
| **Design** | Select ISTQB techniques, design test cases, define data sets, design oracles | Sprint 0 — per sprint, 1 sprint ahead | Test case repository (this document Section 3), test data catalogue, mock/stub design |
| **Implementation** | Write automated tests, configure test environments, seed test data, set up CI gates | Sprint 1 — Sprint 8 (1 sprint behind dev) | Go test suites, Playwright API scripts, Docker Compose test profile, CI YAML |
| **Execution** | Run tests, log results, raise defects, regression cycles | Sprint 1 — Sprint 9 (per story DoD) | Test execution reports, defect tickets, Allure/HTML test reports |
| **Completion** | Final regression, coverage report, quality gate sign-off, lessons learned | Sprint 9 (Launch Prep + UAT) | Final test summary report, sign-off matrix, retrospective findings |

---

## 1.3 Test Types Coverage Matrix

| Test Type | Sub-Type | Tools | Scope | Timing |
|---|---|---|---|---|
| **Functional** | Unit | `go test` + `testify` + `gomock` | Handler, Service, Repository layers in isolation | Every commit (CI gate) |
| **Functional** | Integration | `go test` + `testcontainers-go` (Postgres + Redis) | Handler→Service→Repository full chain, real DB schemas | Per story completion |
| **Functional** | API E2E | Playwright (API mode, no browser) | Full HTTP request flows across tenant boundaries | Per sprint regression |
| **Non-Functional** | Security | `go test` (negative paths), Playwright (token manipulation, header injection), OWASP ZAP (Sprint 8) | Auth bypass, token forgery, cross-tenant data leak, CSRF | Per story (security cases) + Sprint 8 full scan |
| **Non-Functional** | Rate Limit / Resilience | `go test` (counter boundary tests), Redis integration tests | Sliding window thresholds, lockout states | Sprint 2 + Sprint 8 |
| **Non-Functional** | Timing / Anti-Enumeration | `go test` with elapsed-time assertions | Password reset timing parity (known vs. unknown email) | Sprint 1 |
| **Structural** | Code Coverage | `go test -cover`, `govulncheck` | 80% line coverage threshold on service layer | CI gate (every PR) |
| **Structural** | Static Analysis | `golangci-lint`, `gosec` | Secret exposure, SQL injection patterns, unsafe JWT handling | CI gate (every PR) |
| **Change-Related** | Regression | Full automated suite (unit + integration + E2E) | All previously passing test cases | Sprint release cut + hotfix |
| **Change-Related** | Confirmation | Targeted re-execution of defect-linked test cases | Defect fix verification | Per defect closure |
| **Change-Related** | Tenant Migration Regression | `go test` (migration idempotency suite) | New schema gets all migrations, re-run is safe | Every schema migration PR |

---

## 1.4 ISO 25010 Quality Characteristics Priority Table

| Characteristic | Sub-Characteristic | Priority | Justification |
|---|---|---|---|
| **Functional Suitability** | Functional Correctness | **Critical** | Core auth flows (login, token issuance, session management) must be exactly correct — any defect is a security incident or system outage |
| **Functional Suitability** | Functional Completeness | **High** | All 20 Must Have stories must be implemented; gaps leave tenants without required auth capabilities |
| **Functional Suitability** | Functional Appropriateness | **Medium** | API contracts must match prd.md spec; inappropriate responses (e.g., leaking user existence on password reset) are security defects |
| **Security** | Confidentiality | **Critical** | Cross-tenant data isolation (ADR-001) is the #1 risk; any tenant A accessing tenant B data is a catastrophic breach |
| **Security** | Integrity | **Critical** | JWT RS256 signature verification, opaque refresh token hashing (ADR-004), audit log append-only enforcement — corruption of any means full auth compromise |
| **Security** | Non-repudiation | **High** | Audit log must provide irrefutable evidence of auth events (ADR-007); required for SOC 2 (ADR-008) |
| **Security** | Authenticity | **Critical** | RS256 key pair management, OAuth PKCE code verifier, Google OIDC state validation — identity forgery must be impossible |
| **Security** | Accountability | **High** | All auth events linked to tenant + user + timestamp; necessary for incident response and compliance |
| **Reliability** | Availability | **High** | Auth is the entry gate to the entire product; downtime blocks all tenants |
| **Reliability** | Fault Tolerance | **High** | Redis failure must not break login (graceful degradation for rate limiting); Vault unavailability handling must be defined |
| **Reliability** | Recoverability | **Medium** | Failed tenant schema migration must be isolated and rolled back without affecting other tenants |
| **Performance Efficiency** | Time Behaviour | **High** | Token validation must be sub-10ms; brute force protection via rate limiting must not introduce unacceptable latency for legitimate users |
| **Performance Efficiency** | Resource Utilisation | **Medium** | Schema-per-tenant model has connection pool implications at scale; baseline benchmarks needed pre-launch |
| **Maintainability** | Modularity | **Medium** | Handler→Service→Repository layering (as per developer agent constraints) must be enforced via architecture tests |
| **Maintainability** | Testability | **High** | Each layer must be independently testable via interfaces/mocks; tight coupling blocks regression speed |
| **Compatibility** | Interoperability | **Medium** | OAuth 2.0 endpoints must be RFC 6749 / RFC 7636 compliant so third-party clients work without custom integration |
| **Usability** | Not applicable (API-only) | **Low** | ADR-003: no hosted UI; usability of API error messages is covered under Functional Appropriateness |
| **Portability** | Adaptability | **Low** | Fly.io deployment is the single target for this pilot; portability is a post-launch concern |

---

## 1.5 Risk-Based Test Priorities

| Rank | Risk | Sprint | Test Focus Area | Mitigation via Testing |
|---|---|---|---|---|
| 1 | **Cross-tenant data leak** — schema routing bug exposes tenant A data to tenant B queries | Sprint 3 | US-08a, US-08b: schema-routing middleware, all cross-tenant negative paths, JWT tenant_id claim validation | Dedicated isolation test suite; every DB call in integration tests asserts `search_path` is set correctly; Playwright cross-tenant token swap attacks |
| 2 | **Refresh token reuse / session hijacking** — stolen refresh token replayed, no revocation cascade | Sprint 2 | US-04: family revocation on reuse detection, audit event SUSPICIOUS_TOKEN_REUSE, all-device logout | State machine coverage of all rotation transitions; negative: replay after rotation; assert full family revoked |
| 3 | **OAuth PKCE bypass / authorization code interception** — missing or weak code_verifier accepted | Sprint 5 | US-11b, US-11c: plain method rejected, wrong verifier rejected, code single-use TTL enforced | Boundary testing of all PKCE parameter combinations; code reuse asserts token revocation cascade |
| 4 | **User enumeration via timing / error message differences** — password reset or login reveals user existence | Sprint 1 | US-06: identical response body + timing for known/unknown email; US-03: ambiguous error on wrong password | Timing assertion tests (elapsed time within acceptable delta); response body diff assertions across known/unknown paths |
| 5 | **Tenant migration failure blast radius** — failed migration for one tenant corrupts or blocks others | Sprint 3 | US-07b: migration runner isolation, idempotency, rollback | Inject migration failure mid-run; assert other tenant schemas untouched; re-run asserts idempotency |

---

## 1.6 Test Environments

| Environment | Purpose | Infrastructure | Test Data Strategy | Tools |
|---|---|---|---|---|
| **Local Dev** | Developer unit + integration tests during active development | Docker Compose: `postgres:16`, `redis:7`, Vault dev mode, mock Resend webhook | Ephemeral — seeded per test via `testcontainers-go`; wiped after each test run | `go test`, `testify`, `gomock`, `testcontainers-go` |
| **CI (GitHub Actions)** | Automated gate on every PR and merge to main | GitHub Actions services: `postgres:16`, `redis:7`; Vault dev mode container; Resend calls intercepted by mock HTTP server | Ephemeral — fixture SQL scripts applied per job; isolated schemas per test tenant | `go test -race -cover`, `golangci-lint`, `gosec`, `govulncheck`, Playwright (API mode, headless) |
| **Staging** | Sprint-end regression, cross-sprint integration validation, UAT | Fly.io staging app; dedicated staging PostgreSQL cluster (schema-per-tenant mirroring prod topology); Redis staging instance; real Vault (staging mount); Resend sandbox mode | Semi-persistent — baseline tenant fixtures restored before each sprint regression cycle; test tenants prefixed `test_` | Full Playwright E2E suite, manual exploratory, OWASP ZAP (Sprint 8) |
| **Production Smoke** | Post-deploy health check only — narrow, non-destructive | Live production environment | Dedicated smoke-test tenant (`smoke_tenant`) with synthetic users; no real PII; all smoke test accounts flagged `is_test=true` | Playwright smoke suite (subset of critical path E2E — login, token refresh, logout only); automated on deploy pipeline |

---

## 1.7 Entry and Exit Criteria by Phase

### Sprint 0 — Architecture Spike

| | Criteria |
|---|---|
| **Entry** | Solution architecture document finalised and merged; tech stack confirmed (Go + Gin + PostgreSQL + Redis + ory/fosite) |
| **Exit** | Test plan v1.0 approved; CI pipeline skeleton green (lint + build + empty test suite passes); Docker Compose local env boots successfully; testcontainers-go proof-of-concept passes against real Postgres schema; test data catalogue v1.0 drafted |

### Sprint 1 — Core Auth (US-01 through US-06)

| | Criteria |
|---|---|
| **Entry** | Sprint 0 exit criteria met; Go module scaffolded; database migration tooling operational; Resend mock server configured in CI |
| **Exit** | All TC-01-xx through TC-06-xx automated and green in CI; 80%+ line coverage on `auth` service package; zero `gosec` HIGH findings; no P1/P2 defects open; anti-enumeration timing tests pass within 50ms delta |

### Sprint 2 — Sessions, Rate Limiting, Audit Log

| | Criteria |
|---|---|
| **Entry** | Sprint 1 exit criteria met; Redis integration confirmed in CI; audit log schema migrated |
| **Exit** | TC-04-xx (session), TC-14-xx (rate limit), TC-15-xx (audit log) automated and green; Redis sliding window boundary tests pass; family revocation test passes; no P1/P2 defects open |

### Sprint 3 — Multi-Tenancy + Isolation

| | Criteria |
|---|---|
| **Entry** | Sprint 2 exit criteria met; schema provisioning implemented; migration runner operational |
| **Exit** | TC-07a-xx, TC-07b-xx, TC-07c-xx, TC-08a-xx, TC-08b-xx automated and green; **cross-tenant isolation suite 100% pass rate** (non-negotiable gate); migration idempotency confirmed; no P1/P2 defects open; staging environment provisioned with 3+ test tenants |

### Sprint 4 — RBAC, JWT Claims, Admin API (US-09, US-10)

| | Criteria |
|---|---|
| **Entry** | Sprint 3 exit criteria met; RBAC schema migrated; JWT claims service implemented |
| **Exit** | TC-09-xx, TC-10-xx automated and green; JWT decode assertions confirm correct claims structure; no P1/P2 defects open |

### Sprint 5 — OAuth 2.0 Authorization Server (US-11a, US-11b, US-11c)

| | Criteria |
|---|---|
| **Entry** | Sprint 4 exit criteria met; ory/fosite integrated; PKCE implementation complete |
| **Exit** | TC-11a-xx, TC-11b-xx, TC-11c-xx automated and green; PKCE security boundary tests pass (plain method rejected, wrong verifier rejected); no P1/P2 defects open |

### Sprint 6 — OAuth M2M + Social Login (US-12, US-13)

| | Criteria |
|---|---|
| **Entry** | Sprint 5 exit criteria met; Google OIDC mock server configured; M2M client credential flow implemented |
| **Exit** | TC-12-xx, TC-13-xx automated and green; CSRF state validation tests pass; Google OIDC mock exchange confirmed; no P1/P2 defects open |

### Sprint 7 — MFA + User Profile

| | Criteria |
|---|---|
| **Entry** | Sprint 6 exit criteria met; TOTP library integrated |
| **Exit** | MFA test cases automated and green; regression suite for Sprint 1–6 stories passes; no P1/P2 defects open |

### Sprint 8 — Hardening + Compliance + Pentest Remediation

| | Criteria |
|---|---|
| **Entry** | Sprint 7 exit criteria met; OWASP ZAP configured against staging |
| **Exit** | OWASP ZAP scan zero HIGH/CRITICAL findings; `govulncheck` zero known CVEs in dependency tree; all Sprint 1–7 pentest findings remediated and confirmed; full regression suite green on staging |

### Sprint 9 — Launch Prep + UAT

| | Criteria |
|---|---|
| **Entry** | Sprint 8 exit criteria met; staging data baseline restored; UAT participants briefed |
| **Exit** | UAT sign-off from product owner; production smoke suite green post-deploy; final test summary report published; all P1/P2 defects closed; P3 defects accepted/deferred with product owner sign-off; test coverage report archived |

---

# Section 2: ISTQB Test Design Techniques

## 2.1 Technique Selection by Feature Area

### Registration + Email Verification (US-01, US-02)

**Primary Technique:** Equivalence Partitioning (EP) + Boundary Value Analysis (BVA)

**Rationale:** The registration input domain divides cleanly into valid and invalid equivalence classes — valid email format vs. invalid, password meeting policy vs. below minimum length vs. above maximum. BVA is applied at password length boundaries (e.g., 7 chars = reject, 8 chars = accept, 72 chars = accept, 73 chars = reject if bcrypt max is enforced). Email verification adds a state transition dimension: token in states {unissued, valid, expired, used} drives a secondary State Transition technique for the verification token lifecycle.

### Login — Including Brute Force and Lockout (US-03, US-14)

**Primary Technique:** Decision Table Testing + State Transition Testing

**Rationale:** Login correctness is determined by the combination of multiple independent conditions: email verified (yes/no), password correct (yes/no), account locked (yes/no), tenant match (yes/no). A decision table systematically covers all meaningful combinations without combinatorial explosion. The lockout state machine (attempts 1-4 = allowed, attempt 5 = locked, locked + time elapsed = unlocked) is a textbook State Transition problem requiring full transition coverage including invalid transitions (attempting login on a locked account).

### Session Management — Refresh, Rotation, Suspicious Reuse Detection (US-04, US-05)

**Primary Technique:** State Transition Testing

**Rationale:** Refresh token rotation is a finite state machine with states {issued, rotated/invalidated, expired, family-revoked}. The key security property — that reusing a rotated token triggers revocation of the entire family — is expressible only as a specific state transition (rotated → all-family-revoked). Full transition coverage including the suspicious-reuse transition is mandatory. All-device logout adds the "all tokens in family forcibly expired" transition, which must be verified independently.

### Password Reset — Anti-Enumeration Timing (US-06)

**Primary Technique:** Equivalence Partitioning + Pairwise / Timing Analysis (custom security technique)

**Rationale:** The functional paths (known email, unknown email, valid token, expired token, reused token) form clean equivalence partitions. The critical security property — that the response for a known email must be observationally identical (body, headers, and elapsed time) to an unknown email — requires a timing assertion layer not covered by standard EP alone. Tests explicitly measure response time across both partitions and assert the delta is within an acceptable jitter window (target: within 50ms).

### Multi-Tenancy / Schema Isolation (US-07a, US-07b, US-07c, US-08a, US-08b)

**Primary Technique:** Error Guessing (security-focused) + Boundary Value Analysis

**Rationale:** Schema isolation defects are the #1 risk and do not always manifest through standard positive/negative EP. Error guessing — informed by known PostgreSQL search_path misconfiguration patterns, connection pool leakage, and JWT tenant_id claim bypass scenarios — is the most effective technique here. This is supplemented by BVA on tenant identifiers (valid UUID, malformed UUID, SQL injection string, another tenant's valid UUID) and explicit cross-tenant attack scenario tests (swap JWT tenant_id, call endpoints with another tenant's credentials).

### RBAC + JWT Claims (US-09, US-10)

**Primary Technique:** Decision Table Testing

**Rationale:** RBAC authorization decisions depend on the combination of: authenticated (yes/no), tenant match (yes/no), role present in JWT claims (yes/no), operation permitted for role (yes/no). A decision table efficiently covers all combinations. JWT claims structure validation uses Specification-Based Testing against the defined claims schema (roles[], tenant_id, exp = now+15min, no sensitive fields).

### OAuth 2.0 — PKCE, Client Credentials (US-11a, US-11b, US-11c, US-12)

**Primary Technique:** Decision Table Testing + Boundary Value Analysis

**Rationale:** OAuth 2.0 PKCE has a complex parameter space — presence/absence of code_challenge, code_challenge_method (S256 vs. plain), code_verifier correctness, code single-use TTL, client_id validity. Decision tables cover the correctness combinations; BVA covers the 10-minute code TTL boundary (9:59 = valid, 10:01 = expired). Security-focused error guessing covers the code-reuse-triggers-revocation invariant. For client credentials (M2M), the claims-absence property (no sub, no roles) requires explicit specification-based assertion.

### Social Login — Google OIDC (US-13)

**Primary Technique:** Use Case Testing + Error Guessing (CSRF-focused)

**Rationale:** Google OIDC follows a multi-step OAuth flow best modelled as a use case (sequence of interactions with actors: client, auth server, Google IdP). Happy path and two key alternate flows (new user creation vs. existing user account linking requiring password verification) are use case extensions. CSRF protection testing uses error guessing: missing state, mismatched state, replayed state — drawn from known OAuth CSRF attack patterns.

### Audit Log — Append-Only, Tenant-Scoped (US-15)

**Primary Technique:** Specification-Based Testing + Error Guessing

**Rationale:** The audit log specification defines: exact event fields (event_type, user_id, tenant_id, ip_address, timestamp, metadata), append-only constraint (no UPDATE/DELETE operations permitted on audit table), and tenant scoping (admin API returns only own tenant's events). Specification-based testing verifies field presence and correctness after each triggering action. Error guessing covers the append-only invariant (attempt direct DELETE via SQL; attempt DELETE via API) and the tenant scoping invariant (admin A queries audit log with tenant B token).

### Rate Limiting — Redis Sliding Window (US-14)

**Primary Technique:** Boundary Value Analysis + State Transition Testing

**Rationale:** The sliding window rate limiter has a clear numerical threshold (5 failures → lockout) with defined boundary conditions (attempts 4 = allowed, attempt 5 = locked). BVA at the exact threshold boundary and just above/below is the primary technique. The lockout state itself (locked → window expires → counter resets → unlocked) is a state transition. IP-level rate limiting adds a second independent sliding window requiring the same BVA treatment.

---

# Section 3: Test Cases

### US-01 — User Registration (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-01-01 | Successful registration with valid inputs | EP — valid class | No existing user with that email; tenant exists and is active | POST `/auth/register` with `{"email":"alice@example.com","password":"Str0ngPass!","tenant_id":"<valid-uuid>"}` | HTTP 201; response body `{"message":"Verification email sent"}`; user row created in tenant schema with `email_verified=false`; bcrypt hash stored (not plaintext); Resend mock receives exactly one email to `alice@example.com` | Critical | Functional |
| TC-01-02 | Registration rejected for duplicate email within same tenant | EP — invalid class | User `alice@example.com` already exists in tenant schema | POST `/auth/register` with same email and tenant_id | HTTP 409; error body does not reveal whether it is email or password that conflicts (generic "registration failed" or "email already in use" is acceptable but must be consistent); no new user row created | Critical | Functional / Security |
| TC-01-03 | Registration rejected for password below minimum length | BVA — lower boundary | No existing user; tenant active | POST `/auth/register` with password `"Short1!"` (7 chars, assuming 8-char minimum) | HTTP 422; body contains field-level validation error referencing password policy; no user row created | High | Functional |
| TC-01-04 | Registration accepted at exact minimum password length | BVA — on-boundary | No existing user; tenant active | POST `/auth/register` with password of exactly 8 characters meeting complexity rules | HTTP 201; user created successfully | High | Functional |
| TC-01-05 | Registration rejected for invalid email format | EP — invalid class | Tenant active | POST `/auth/register` with `email: "not-an-email"` | HTTP 422; validation error on email field; no user row created | High | Functional |
| TC-01-06 | Registration rejected for missing required fields | EP — invalid class | Tenant active | POST `/auth/register` with empty body `{}` | HTTP 400 or 422; error lists missing fields; no user row created | Medium | Functional |
| TC-01-07 | Email uniqueness is per-tenant, not global | EP — valid class (cross-tenant) | `alice@example.com` exists in `tenant_a` schema | POST `/auth/register` with same email but `tenant_id` for `tenant_b` | HTTP 201; user created in `tenant_b` schema; `tenant_a` user record untouched; both users have separate UUIDs | Critical | Security / Functional |
| TC-01-08 | Password stored as bcrypt hash, not plaintext | Structural / Security — error guessing | User registered successfully (TC-01-01) | Query tenant schema `users` table directly; inspect `password_hash` column | Column value is a bcrypt hash starting with `$2a$` or `$2b$`; plaintext password string is not present anywhere in the row | Critical | Security |
| TC-01-09 | SQL injection string in email field is rejected | Error guessing — injection | Tenant active | POST `/auth/register` with `email: "'; DROP TABLE users; --"` | HTTP 422 (validation rejects before DB layer); no schema modification; DB connection stable | Critical | Security |

---

### US-02 — Email Verification (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-02-01 | Valid token verifies email successfully | EP — valid class | User registered; verification email received; token extracted from Resend mock | GET `/auth/verify-email?token=<valid-token>` | HTTP 200; `email_verified=true` in user row; token marked as used/deleted in DB; subsequent login no longer rejected for unverified account | Critical | Functional |
| TC-02-02 | Expired token is rejected | EP — invalid class (time) | Verification token generated; token TTL manually expired in DB or via time-travel in test | GET `/auth/verify-email?token=<expired-token>` | HTTP 400 or 410; error message indicates token expired; `email_verified` remains `false`; no state change in DB | High | Functional |
| TC-02-03 | Reused (already-consumed) token is rejected | State Transition — used state | TC-02-01 completed; same token presented again | GET `/auth/verify-email?token=<already-used-token>` | HTTP 400 or 410; error indicates token already used or invalid; `email_verified` still `true` (no regression); no error in server logs indicating panic or DB error | High | Security / Functional |
| TC-02-04 | Non-existent / fabricated token is rejected | EP — invalid class + error guessing | No matching token in DB | GET `/auth/verify-email?token=totally-fabricated-token-xyz` | HTTP 400 or 404; error does not reveal whether a user with that email exists; response time comparable to valid-token path (no timing oracle) | High | Security |
| TC-02-05 | Resend verification email — new token invalidates old token | State Transition | User registered; first token issued; resend requested | POST `/auth/resend-verification` with `{"email":"alice@example.com","tenant_id":"<uuid>"}`; then attempt GET with old token | POST returns HTTP 200; Resend mock receives second email with new token; old token returns HTTP 400/410; new token verifies successfully | High | Functional |
| TC-02-06 | Resend for already-verified account is handled gracefully | EP — invalid class | User already email-verified | POST `/auth/resend-verification` with verified user's email | HTTP 400 or 200 (consistent behaviour per spec); if 200, no new email sent (Resend mock receives zero calls); no server error | Medium | Functional |
| TC-02-07 | Resend for unknown email does not reveal user existence | Error guessing — enumeration | No user with that email in tenant | POST `/auth/resend-verification` with non-existent email | HTTP 200 with identical body to the known-email case; response time within 50ms of the known-email path | Critical | Security |

---

### US-03 — User Login (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-03-01 | Successful login returns JWT access token and refresh token | EP — valid class | User registered, email verified, account not locked | POST `/auth/login` with correct `{email, password, tenant_id}` | HTTP 200; response contains `access_token` (JWT RS256, exp = now+15min) and `refresh_token` (opaque string); JWT decoded claims contain `sub`, `tenant_id`, `roles[]`, `exp`; no sensitive fields (password hash, internal IDs) in JWT payload | Critical | Functional / Security |
| TC-03-02 | JWT access token claims structure is correct | Specification-based | TC-03-01 completed; access_token obtained | Decode access_token (Base64, no signature verification needed in this test); inspect payload | `tenant_id` matches the login request tenant; `exp` = current Unix time + 900 seconds (±5s tolerance); `roles` is an array; `sub` is user UUID; `iss` is configured issuer; no `password`, `password_hash`, `refresh_token`, or PII fields present | Critical | Security / Functional |
| TC-03-03 | Login rejected with wrong password — ambiguous error | EP — invalid class | User registered, email verified | POST `/auth/login` with correct email, wrong password | HTTP 401; error body is generic (e.g., `"Invalid credentials"`) — must NOT state "wrong password" or "user not found" separately; response body is byte-for-byte identical to the user-not-found error case | Critical | Security |
| TC-03-04 | Login rejected for unverified email account | Decision Table — email_verified=false | User registered but has not completed email verification | POST `/auth/login` with correct credentials for unverified user | HTTP 403; error indicates email not verified; no token issued | High | Functional |
| TC-03-05 | Login rejected for correct credentials but wrong tenant_id | Decision Table — tenant mismatch | User exists in `tenant_a`; login attempted with `tenant_b` tenant_id | POST `/auth/login` with `tenant_id` for `tenant_b` but user exists only in `tenant_a` | HTTP 401; generic "Invalid credentials" error; no token issued; no indication that user exists in another tenant | Critical | Security |
| TC-03-06 | Login with token from tenant_a rejected on tenant_b endpoint | Error guessing — JWT swap | Two tenants and two users; user_a has valid JWT for tenant_a | Use tenant_a JWT as Authorization header on an endpoint that requires tenant_b context | HTTP 401 or 403; request rejected; tenant_b data not accessible; error does not confirm tenant_a token was valid | Critical | Security |
| TC-03-07 | Login for non-existent user returns same response as wrong password | EP — user-not-found | No user with email `nobody@example.com` in tenant | POST `/auth/login` with `nobody@example.com` and any password | HTTP 401; response body identical to TC-03-03 wrong-password case; response time within 100ms of TC-03-03 (constant-time comparison enforced) | Critical | Security |

---

### US-04 — Session Refresh (3 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-04-01 | Valid refresh token returns new access token and rotated refresh token | State Transition — issued → rotated | User logged in; valid refresh_token held | POST `/auth/token/refresh` with `{"refresh_token":"<valid-opaque-token>"}` | HTTP 200; new `access_token` issued (new exp); new `refresh_token` issued (new opaque value); old refresh_token now invalidated in DB; old refresh_token hash no longer present in valid-tokens store | Critical | Functional |
| TC-04-02 | Expired refresh token is rejected | State Transition — expired | Refresh token TTL has elapsed (manipulate DB timestamp or use short-TTL test token) | POST `/auth/token/refresh` with expired token | HTTP 401; no new tokens issued; DB token record unchanged (or already purged) | High | Functional |
| TC-04-03 | Reused (already-rotated) refresh token triggers full family revocation | State Transition — rotated → family-revoked | TC-04-01 completed; old (now-rotated) token retained by attacker | POST `/auth/token/refresh` with the old (already-rotated) refresh_token | HTTP 401; all refresh tokens in the family (including any legitimately issued ones) revoked in DB; `SUSPICIOUS_TOKEN_REUSE` audit event written with correct user_id, tenant_id, ip, timestamp | Critical | Security |
| TC-04-04 | Family revocation on reuse — subsequent use of valid child token also fails | State Transition — post-revocation | TC-04-03 completed; a newer token from same family was held by legitimate user | Attempt POST `/auth/token/refresh` with the newer (legitimate) refresh_token from the same family | HTTP 401; family remains fully revoked; legitimate user must re-authenticate via full login | Critical | Security |
| TC-04-05 | Refresh token is stored as hash, not plaintext | Structural / Security | TC-04-01 completed | Query `refresh_tokens` table in tenant schema directly | Stored value is a cryptographic hash (e.g., SHA-256 hex); plaintext opaque token string is not present | Critical | Security |
| TC-04-06 | Malformed / garbage refresh token is rejected safely | EP — invalid class | No precondition | POST `/auth/token/refresh` with `{"refresh_token":"not-a-real-token-aaaa"}` | HTTP 401; no server error (no 500); no stack trace in response | High | Functional |

---

### US-05 — Logout (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-05-01 | Single-device logout invalidates current refresh token only | EP — valid class | User logged in on two sessions (two refresh tokens issued) | POST `/auth/logout` with valid access_token (Bearer) and `{"refresh_token":"<session-1-token>"}` | HTTP 200; session-1 refresh_token revoked in DB; session-2 refresh_token still valid; session-2 can still refresh successfully | High | Functional |
| TC-05-02 | All-devices logout invalidates all refresh tokens for user | EP — valid class (all-devices) | User logged in on three sessions | POST `/auth/logout/all` with valid access_token (Bearer) | HTTP 200; all three refresh_tokens for that user in that tenant are revoked; none can be refreshed; audit event `LOGOUT_ALL` written | High | Functional |
| TC-05-03 | Logout with already-expired refresh token is idempotent | EP — already-expired | Valid access_token; refresh_token already expired by TTL | POST `/auth/logout` with expired refresh_token | HTTP 200 (idempotent — logout of an already-expired session must not return an error to the client); no server error | Medium | Functional |
| TC-05-04 | Logout with already-revoked refresh token is idempotent | EP — already-revoked | TC-05-01 completed; re-submit same token | POST `/auth/logout` with already-revoked refresh_token | HTTP 200; no state change; no error | Medium | Functional |
| TC-05-05 | Logout requires valid access token (authentication enforced) | Error guessing | No precondition | POST `/auth/logout` with no Authorization header or invalid JWT | HTTP 401; no logout action taken | High | Security |
| TC-05-06 | Logout does not invalidate access token (JWT is stateless) | Specification-based | User has valid access_token and has logged out (TC-05-01) | Use the access_token (pre-logout) to call a protected endpoint within the 15-min TTL | HTTP 200 (access_token remains valid until expiry — this is expected and by design per ADR-004); test documents this as known behaviour, recommending short TTL and client-side token discard | High | Security / Non-Functional |

---

### US-06 — Password Reset (3 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-06-01 | Password reset request for known email sends reset email | EP — valid class | User `alice@example.com` exists and is verified in tenant | POST `/auth/password/reset-request` with `{"email":"alice@example.com","tenant_id":"<uuid>"}` | HTTP 200; Resend mock receives exactly one reset email to alice; reset token stored (hashed) in DB with expiry | Critical | Functional |
| TC-06-02 | Password reset request for UNKNOWN email returns identical response and timing | EP — invalid class + Timing | No user with `nobody@example.com` in tenant | POST `/auth/password/reset-request` with `{"email":"nobody@example.com","tenant_id":"<uuid>"}` | HTTP 200; response body byte-for-byte identical to TC-06-01; Resend mock receives zero emails; elapsed time within 50ms of TC-06-01 (artificial delay applied server-side) | Critical | Security |
| TC-06-03 | Valid reset token allows password change and revokes all sessions | EP — valid class | TC-06-01 completed; user has 2 active refresh_token sessions | POST `/auth/password/reset` with `{"token":"<valid-token>","new_password":"NewStr0ng!"}` | HTTP 200; user's password_hash updated in DB; all existing refresh_tokens for user revoked; audit event `PASSWORD_RESET` written; old password no longer works for login | Critical | Functional / Security |
| TC-06-04 | Expired reset token is rejected | State Transition — expired | Reset token TTL elapsed | POST `/auth/password/reset` with expired token | HTTP 400 or 410; password not changed; error indicates token expired | High | Functional |
| TC-06-05 | Reused reset token is rejected | State Transition — used | TC-06-03 completed; same token re-submitted | POST `/auth/password/reset` with already-used token | HTTP 400 or 410; password not changed again; token cannot be reused | High | Security |
| TC-06-06 | Weak new password rejected during reset | BVA — lower boundary | Valid reset token available | POST `/auth/password/reset` with valid token and weak password `"weak"` | HTTP 422; password not changed; reset token not consumed (still usable); error references password policy | Medium | Functional |
| TC-06-07 | Reset token is single-use — second use after successful reset fails | State Transition | TC-06-03 completed | POST `/auth/password/reset` with the same token used in TC-06-03 | HTTP 400 or 410; token marked used; no state change | High | Security |

---

### US-07a — Tenant Provisioning (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-07a-01 | Successful tenant provisioning creates schema, default roles, and returns tenant_id | EP — valid class | Super-admin authenticated; tenant name `acme-corp` not taken | POST `/admin/tenants` with `{"name":"acme-corp","plan":"starter"}` (super-admin Bearer token) | HTTP 201; response contains `tenant_id` (UUID), `name`, `created_at`; PostgreSQL schema `tenant_<uuid>` created; default roles (`admin`, `member`) exist in schema's `roles` table; all baseline migrations applied to new schema | Critical | Functional |
| TC-07a-02 | Duplicate tenant name returns 409 | EP — invalid class | Tenant `acme-corp` already exists | POST `/admin/tenants` with `{"name":"acme-corp"}` | HTTP 409; error body indicates name conflict; no new schema created; existing tenant untouched | High | Functional |
| TC-07a-03 | Non-super-admin user cannot create tenant | Decision Table — authorization | Regular `admin` role user (not super-admin) authenticated | POST `/admin/tenants` with valid body using regular admin Bearer token | HTTP 403; body indicates insufficient privileges; no schema created; no DB state change | Critical | Security |
| TC-07a-04 | Unauthenticated request to create tenant is rejected | Error guessing | No Authorization header | POST `/admin/tenants` with no Bearer token | HTTP 401; no tenant created | High | Security |
| TC-07a-05 | Tenant schema is isolated — new schema has no cross-schema foreign keys | Structural | TC-07a-01 completed | Inspect `information_schema.table_constraints` and `information_schema.referential_constraints` for the new schema | No foreign key references from new tenant schema to `public` schema or any other tenant schema | Critical | Security |
| TC-07a-06 | Tenant name with SQL special characters is rejected or safely handled | Error guessing — injection | Super-admin authenticated | POST `/admin/tenants` with `{"name":"tenant'; DROP SCHEMA public; --"}` | HTTP 422 (validation rejects) or, if name reaches DB, it is parameterised safely; schema name is not executable SQL; public schema intact | Critical | Security |

---

### US-07b — Migration Runner (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-07b-01 | New tenant schema receives all current migrations in order | Specification-based | Fresh tenant provisioned; 5 migrations exist in migrations directory | Inspect new tenant schema after provisioning | All 5 migration steps applied in sequence; `schema_migrations` tracking table shows all 5 as applied; all expected tables and columns present | Critical | Functional |
| TC-07b-02 | Failed migration for one tenant does not affect other tenants | Error guessing — fault injection | Two tenants exist; second tenant provisioning intentionally triggers a migration error (inject bad SQL in migration file for test) | Trigger provisioning for `tenant_b` with bad migration; inspect `tenant_a` schema | `tenant_b` provisioning returns HTTP 500 or meaningful error; `tenant_a` schema untouched — all tables intact; `tenant_a` users can still log in; `tenant_b` schema rolled back or left in failed state (not partially applied) | Critical | Reliability |
| TC-07b-03 | Migration re-run is idempotent | EP — idempotency | TC-07b-01 completed; all migrations already applied to tenant | Manually invoke migration runner against same tenant schema | No errors; `schema_migrations` table unchanged; no duplicate table creation errors; data in existing tables preserved | High | Functional |
| TC-07b-04 | New migration applied to existing tenants on upgrade | Specification-based | 3 existing tenants; migration v6 added | Run upgrade migration script across all tenants | All 3 tenants receive migration v6; `schema_migrations` updated for all; no cross-tenant interference | High | Functional |
| TC-07b-05 | Migration runner handles concurrent provisioning safely | Error guessing — concurrency | Migration runner does not hold global lock (or does — verify) | Trigger provisioning for 3 tenants simultaneously (parallel HTTP requests) | All 3 schemas created successfully; no deadlocks; no partial migrations; each schema is complete and independent | High | Reliability |

---

### US-07c — Tenant API Credentials (3 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-07c-01 | API credential generation returns client_id and client_secret once | Specification-based | Tenant exists; super-admin authenticated | POST `/admin/tenants/<tenant_id>/credentials` | HTTP 201; response contains `client_id` (stable, non-secret) and `client_secret` (random, high-entropy); response includes warning that secret cannot be retrieved again | Critical | Functional |
| TC-07c-02 | Client secret is stored as hash, not plaintext | Structural / Security | TC-07c-01 completed | Query `tenant_api_credentials` table (or equivalent) in DB | `client_secret` column contains a hash (not the raw secret returned in TC-07c-01); raw secret cannot be recovered from DB | Critical | Security |
| TC-07c-03 | Client secret cannot be retrieved after initial creation | Specification-based | TC-07c-01 completed | GET `/admin/tenants/<tenant_id>/credentials` or repeat GET of same credential | HTTP 200 returns `client_id` only; `client_secret` field absent or masked (`***`); no endpoint returns the original plaintext secret | Critical | Security |
| TC-07c-04 | Credential rotation generates new secret, old secret immediately invalid | State Transition | TC-07c-01 completed; credentials in use | POST `/admin/tenants/<tenant_id>/credentials/rotate` | HTTP 201; new `client_secret` returned (once); old secret can no longer authenticate (attempt M2M token with old secret → 401); new secret works for M2M | High | Functional / Security |
| TC-07c-05 | Duplicate credential generation — only one active credential set at a time | BVA — boundary | TC-07c-01 completed | POST `/admin/tenants/<tenant_id>/credentials` again without rotation | HTTP 409 or rotation is required; system does not issue two simultaneous active credential sets | Medium | Functional |

---

### US-08a — Schema-Routing Middleware (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-08a-01 | Valid X-Tenant-ID header routes request to correct tenant schema | Specification-based (ADR-010) | Two tenants with distinct schemas; both have a user named `alice` | GET `/users/me` with valid Bearer JWT for `tenant_a` and `X-Tenant-ID: <tenant_a_uuid>` | HTTP 200; response contains `tenant_a`'s `alice` data; DB query was executed against `tenant_a` schema (verify via query log or test instrumentation) | Critical | Functional |
| TC-08a-02 | Invalid (non-existent) tenant_id in X-Tenant-ID is rejected | EP — invalid class | No tenant with UUID `00000000-0000-0000-0000-000000000000` | Any authenticated request with `X-Tenant-ID: 00000000-0000-0000-0000-000000000000` | HTTP 404 or 400; error indicates tenant not found; no DB query executed against any tenant schema; no data returned | Critical | Security |
| TC-08a-03 | Malformed tenant_id (not a UUID) is rejected at middleware | BVA — invalid format | No precondition | Any request with `X-Tenant-ID: not-a-uuid` | HTTP 400; validation error at middleware layer (before any service logic); no DB query executed | High | Functional |
| TC-08a-04 | Unauthenticated request uses global schema only — no tenant schema exposed | Specification-based | No Bearer token; global schema exists | GET `/public/health` or similar unprotected endpoint with `X-Tenant-ID: <valid_tenant_uuid>` | HTTP 200 for health check; no tenant-specific data returned; middleware does not set tenant search_path for unauthenticated paths | High | Security |
| TC-08a-05 | JWT tenant_id claim mismatch with X-Tenant-ID header is rejected | Error guessing — claim mismatch | User has valid JWT for `tenant_a`; attempts to use `tenant_b` header | Request with `Authorization: Bearer <tenant_a_jwt>` and `X-Tenant-ID: <tenant_b_uuid>` | HTTP 403; middleware detects tenant_id claim in JWT does not match X-Tenant-ID header; request rejected; no tenant_b data accessed | Critical | Security |
| TC-08a-06 | search_path is reset between requests (connection pool safety) | Structural / Error guessing — pool leak | Two sequential requests: first for `tenant_a`, second for `tenant_b` using same connection from pool | Send request for `tenant_a`; send request for `tenant_b`; verify search_path on second request | Second request operates under `tenant_b` search_path, not `tenant_a`; no data bleed; test instrumentation confirms search_path value at query execution time | Critical | Security |

---

### US-08b — Cross-Tenant Isolation (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-08b-01 | User from tenant_a cannot read user data from tenant_b | Error guessing — direct object reference | `user_a` in `tenant_a`; `user_b` in `tenant_b`; `user_b`'s UUID is known to attacker | GET `/users/<user_b_uuid>` with `user_a`'s JWT and `X-Tenant-ID: tenant_a` | HTTP 404 (object not found in tenant_a schema); tenant_b user data not returned; no indication user_b exists in system | Critical | Security |
| TC-08b-02 | Tenant_a admin cannot list users of tenant_b | Error guessing — privilege escalation | `admin_a` has admin role in `tenant_a` | GET `/admin/users` with `admin_a` JWT and `X-Tenant-ID: tenant_b` | HTTP 403 (JWT tenant_id does not match requested tenant) or middleware blocks before query; no tenant_b user list returned | Critical | Security |
| TC-08b-03 | tenant_a JWT is rejected on tenant_b-scoped endpoints | Decision Table — tenant mismatch | `user_a` has valid JWT with `tenant_id: tenant_a` | POST `/auth/logout` with `user_a` JWT and `X-Tenant-ID: tenant_b` | HTTP 403; request rejected by middleware JWT+header mismatch check; no action taken in tenant_b | Critical | Security |
| TC-08b-04 | Error responses do not leak cross-tenant data | Error guessing — information disclosure | `user_b` exists in `tenant_b` | Trigger a 404 or 403 on a tenant_b-scoped resource using tenant_a credentials | Error body contains only a generic error message and error code; no UUIDs, emails, timestamps, or any field values from tenant_b schema appear in the response body or headers | Critical | Security |
| TC-08b-05 | Response headers do not leak tenant information | Error guessing — header leakage | Any cross-tenant request | Inspect all response headers on any rejected cross-tenant request | No `X-Tenant-*`, `X-DB-Schema`, or any custom headers that reveal internal schema names or tenant UUIDs; standard headers only | High | Security |
| TC-08b-06 | Sequential requests — tenant context does not leak to next request via middleware state | Structural / Error guessing | Tenant_a request followed immediately by tenant_b request on same server process | Issue request for tenant_a (gets 200); immediately issue request for tenant_b (authenticated for tenant_b) | Tenant_b request returns tenant_b data correctly; no tenant_a data bleeds into tenant_b response; test repeated 100 times to surface race conditions | Critical | Security / Reliability |

---

### US-09 — Role Assignment (3 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-09-01 | Admin can assign a valid role to a user within same tenant | EP — valid class | Admin user authenticated in tenant; target user exists in same tenant; role `editor` exists | POST `/admin/users/<user_uuid>/roles` with `{"role":"editor"}` (admin Bearer token) | HTTP 200; user now has `editor` role in tenant schema `user_roles` table; audit event `ROLE_ASSIGNED` written with admin_id, user_id, role, tenant_id, timestamp | High | Functional |
| TC-09-02 | Admin can unassign a role from a user | EP — valid class | User has `editor` role | DELETE `/admin/users/<user_uuid>/roles/editor` (admin Bearer token) | HTTP 200; role removed from `user_roles` table; audit event `ROLE_UNASSIGNED` written; user's next JWT (on refresh) does not contain `editor` role | High | Functional |
| TC-09-03 | Non-existent role name is rejected | EP — invalid class | Admin authenticated | POST `/admin/users/<user_uuid>/roles` with `{"role":"superuser-god-mode"}` | HTTP 400 or 422; error indicates role does not exist; no DB change | Medium | Functional |
| TC-09-04 | Admin cannot assign roles in a different tenant | Error guessing — cross-tenant | Admin authenticated in `tenant_a`; target user UUID belongs to `tenant_b` | POST `/admin/users/<tenant_b_user_uuid>/roles` with admin_a JWT and `X-Tenant-ID: tenant_a` | HTTP 404 (user not found in tenant_a); no role assigned to tenant_b user | Critical | Security |
| TC-09-05 | Regular member user cannot assign roles | Decision Table — insufficient privilege | Member-role user authenticated | POST `/admin/users/<user_uuid>/roles` with member Bearer token | HTTP 403; no role assigned; audit event optionally written for unauthorised RBAC attempt | High | Security |
| TC-09-06 | Audit event is written for role assignment | Specification-based | TC-09-01 completed | Query audit_logs table for event_type `ROLE_ASSIGNED` | Record exists with: `event_type=ROLE_ASSIGNED`, `actor_id=<admin_uuid>`, `target_user_id=<user_uuid>`, `tenant_id=<tenant_uuid>`, `role=editor`, `timestamp` within 1 second of TC-09-01 execution | High | Functional |

---

### US-10 — JWT Claims (2 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-10-01 | JWT contains roles array with all assigned roles | Specification-based | User has roles `[admin, editor]` in tenant | POST `/auth/login` with valid credentials | Decode access_token; `roles` claim is an array containing exactly `["admin","editor"]`; no extra roles; no missing roles | Critical | Functional / Security |
| TC-10-02 | JWT tenant_id claim matches the login tenant | Specification-based | User in `tenant_a` | POST `/auth/login` for `tenant_a` | Decode access_token; `tenant_id` claim equals `tenant_a`'s UUID exactly | Critical | Security |
| TC-10-03 | JWT exp claim equals now + 15 minutes | BVA — exact expiry | Fresh login | POST `/auth/login`; record server timestamp T | Decode access_token; `exp` = T + 900 seconds (±5 seconds tolerance for clock skew) | Critical | Functional / Security |
| TC-10-04 | JWT does not contain sensitive fields | Specification-based — negative | Fresh login | Decode access_token payload (Base64) | Payload does NOT contain any of: `password`, `password_hash`, `refresh_token`, `client_secret`, `vault_token`, `ssn`, `credit_card`; only expected claims present | Critical | Security |
| TC-10-05 | JWT signature algorithm is RS256, not HS256 | Specification-based | Fresh login | Inspect JWT header (first Base64 segment) | `alg` field equals `"RS256"`; `typ` field equals `"JWT"`; algorithm is not `HS256`, `none`, or any symmetric algorithm | Critical | Security |
| TC-10-06 | JWT with forged tenant_id claim is rejected by validation middleware | Error guessing — claim forgery | Valid JWT for tenant_a; attacker modifies payload, re-encodes, submits | Manually construct JWT with modified `tenant_id` pointing to tenant_b (invalid signature) | HTTP 401; signature validation fails; forged JWT rejected; no tenant_b data accessible | Critical | Security |

---

### US-11a — OAuth Client Registration (3 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-11a-01 | Valid OAuth client registration returns client_id and client_secret | EP — valid class | Tenant admin authenticated; valid redirect URI provided | POST `/oauth/clients` with `{"client_name":"My App","redirect_uris":["https://app.example.com/callback"],"grant_types":["authorization_code"],"scope":"openid profile"}` | HTTP 201; response contains `client_id` (stable identifier) and `client_secret` (high-entropy, returned once); client stored in ory/fosite client store for tenant | Critical | Functional |
| TC-11a-02 | Duplicate client_id is rejected | EP — invalid class | TC-11a-01 completed; client_id known | Attempt to register a new client with an explicitly provided duplicate `client_id` (if API allows specifying client_id) | HTTP 409; error indicates client_id conflict; original client record untouched | High | Functional |
| TC-11a-03 | Wildcard redirect URI is rejected | Error guessing — security | Admin authenticated | POST `/oauth/clients` with `{"redirect_uris":["https://*.evil.com/callback"]}` | HTTP 422 or 400; error indicates wildcard redirect URIs not permitted; client not registered; this prevents open-redirect attacks | Critical | Security |
| TC-11a-04 | HTTP (non-HTTPS) redirect URI is rejected for production clients | Specification-based | Admin authenticated | POST `/oauth/clients` with `{"redirect_uris":["http://insecure.example.com/callback"]}` | HTTP 422; error indicates only HTTPS redirect URIs are permitted (localhost exception may apply for dev); client not registered | High | Security |
| TC-11a-05 | Non-admin user cannot register OAuth clients | Decision Table — privilege | Member-role user authenticated | POST `/oauth/clients` with member Bearer token | HTTP 403; client not registered | High | Security |

---

### US-11b — /oauth/authorize (3 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-11b-01 | Valid PKCE authorization request returns authorization code | EP — valid class | Registered OAuth client; user authenticated; S256 code_challenge generated | GET `/oauth/authorize?response_type=code&client_id=<id>&redirect_uri=<uri>&scope=openid&state=<random>&code_challenge=<S256-hash>&code_challenge_method=S256` | HTTP 302 redirect to `redirect_uri?code=<auth_code>&state=<same-state>`; `state` in response equals request `state`; `code` is a single-use opaque string; code stored with 10-min TTL and code_challenge hash | Critical | Functional |
| TC-11b-02 | Missing state parameter is rejected | EP — invalid class | Registered OAuth client | GET `/oauth/authorize` without `state` parameter | HTTP 400 or redirect with `error=invalid_request`; no authorization code issued; error_description mentions missing state | High | Security |
| TC-11b-03 | Invalid client_id returns error — does NOT redirect | Error guessing — open redirect prevention | No registered client with that ID | GET `/oauth/authorize?client_id=nonexistent&redirect_uri=https://attacker.com/steal` | HTTP 400 rendered directly (not a redirect); error indicates invalid client; no redirect to `redirect_uri` (prevents open redirect with unvalidated client) | Critical | Security |
| TC-11b-04 | Authorization code is single-use and expires after 10 minutes | BVA — time boundary + State Transition | Valid auth code issued | Attempt to use code at 9:59 (success); attempt to use same code at 10:01 (expired) | Code at 9:59: successful exchange; code at 10:01: HTTP 400 `error=invalid_grant`; code already-used on second call: HTTP 400 | High | Security |
| TC-11b-05 | Redirect URI mismatch is rejected | Decision Table | Registered client with `redirect_uri=https://legit.example.com/cb` | GET `/oauth/authorize` with `redirect_uri=https://attacker.com/steal` | HTTP 400 rendered directly; no redirect; error indicates redirect_uri mismatch | Critical | Security |

---

### US-11c — /oauth/token PKCE Exchange (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-11c-01 | Valid PKCE token exchange returns access token and refresh token | EP — valid class | Authorization code issued via TC-11b-01; correct code_verifier retained | POST `/oauth/token` with `grant_type=authorization_code&code=<code>&code_verifier=<original-verifier>&client_id=<id>&redirect_uri=<uri>` | HTTP 200; JSON response with `access_token`, `token_type=bearer`, `expires_in=900`, `refresh_token`, optionally `id_token`; authorization code consumed (marked used) | Critical | Functional |
| TC-11c-02 | Wrong code_verifier is rejected | Decision Table | Valid authorization code | POST `/oauth/token` with correct code but `code_verifier=wrong-verifier-string` | HTTP 400; `error=invalid_grant`; no tokens issued; authorization code remains unused (or consumed to prevent further guessing attempts — verify behaviour per spec) | Critical | Security |
| TC-11c-03 | plain code_challenge_method is rejected | Specification-based | Attempt to initiate flow with plain method | GET `/oauth/authorize?code_challenge_method=plain&code_challenge=<raw-verifier>` | HTTP 400 or `error=invalid_request`; plain method not accepted; only S256 permitted | Critical | Security |
| TC-11c-04 | Authorization code reuse triggers HTTP 400 and revokes all issued tokens | State Transition — code-reuse attack | TC-11c-01 completed; access_token and refresh_token issued | POST `/oauth/token` with same authorization code again | HTTP 400; `error=invalid_grant`; both the access_token and refresh_token issued in TC-11c-01 are immediately revoked (attempt to use either returns 401); this is RFC 6749 security requirement | Critical | Security |
| TC-11c-05 | Token exchange with mismatched redirect_uri is rejected | Decision Table | Valid auth code with registered redirect_uri | POST `/oauth/token` with different `redirect_uri` than used in `/oauth/authorize` | HTTP 400; `error=invalid_grant`; no tokens issued | High | Security |
| TC-11c-06 | Token exchange after code expiry (>10 min) is rejected | BVA — time boundary | Auth code issued; 10-minute TTL elapsed | POST `/oauth/token` with expired code and correct code_verifier | HTTP 400; `error=invalid_grant`; error_description indicates code expired | High | Functional |

---

### US-12 — Client Credentials M2M (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-12-01 | Valid M2M token request returns access token with correct claims | EP — valid class | Tenant has valid client_id and client_secret from US-07c; requested scope is registered | POST `/oauth/token` with `grant_type=client_credentials&client_id=<id>&client_secret=<secret>&scope=read:users` | HTTP 200; access_token issued; decoded claims contain `iss`, `aud`, `exp`, `scope=read:users`, `tenant_id`; `sub` claim is absent or set to client_id (not a user UUID); `roles` claim is absent | Critical | Functional / Security |
| TC-12-02 | Invalid client_secret returns 401 | EP — invalid class | Valid client_id; wrong secret | POST `/oauth/token` with correct client_id but `client_secret=wrong-secret` | HTTP 401; `error=invalid_client`; no token issued | Critical | Security |
| TC-12-03 | Invalid scope returns 400 | EP — invalid class | Valid credentials | POST `/oauth/token` with `scope=delete:everything` (unregistered scope) | HTTP 400; `error=invalid_scope`; no token issued | High | Functional |
| TC-12-04 | M2M token does not contain user-specific claims | Specification-based | TC-12-01 completed | Decode access_token payload | `sub` is absent or is client_id (machine identity only); `roles` claim is absent; `email` is absent; no user UUID present in any claim | Critical | Security |
| TC-12-05 | Rotated / revoked client_secret is rejected | State Transition | TC-07c-04 (rotation) completed; old secret known | POST `/oauth/token` with old (revoked) client_secret | HTTP 401; `error=invalid_client`; new secret works; old does not | High | Security |
| TC-12-06 | M2M token is tenant-scoped — cannot be used on other tenants | Error guessing — cross-tenant | tenant_a M2M token issued | Use tenant_a M2M access_token with `X-Tenant-ID: tenant_b` | HTTP 401 or 403; tenant_b resources not accessible; JWT tenant_id claim mismatch detected | Critical | Security |

---

### US-13 — Google Social Login (8 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-13-01 | New user authenticated via Google OIDC is created as verified | Use Case — main flow | Google OIDC mock server configured; no existing user with Google account's email | GET `/auth/social/google?tenant_id=<uuid>` (initiates flow); mock Google returns valid id_token with `email_verified=true`; complete callback | User created in tenant schema with `email_verified=true` (inherited from Google); access_token and refresh_token issued; no separate email verification step required | Critical | Functional |
| TC-13-02 | Existing user with same email prompted to verify password before linking | Use Case — alternate flow | User `alice@example.com` already exists in tenant with password-based account | Google OIDC flow completes with Google account for `alice@example.com` | System does NOT auto-link; HTTP 409 or redirect with `action=link_account`; user must supply current password to confirm identity before linking; protects against account takeover | Critical | Security |
| TC-13-03 | Account linking completes after password verification | Use Case — extension | TC-13-02 initiated; Alice has correct password | POST `/auth/social/link` with `{"provider":"google","google_id_token":"<token>","current_password":"<correct>"}` | HTTP 200; Google account linked to existing Alice account; subsequent Google OIDC login for `alice@example.com` succeeds without password; audit event `SOCIAL_ACCOUNT_LINKED` written | High | Functional |
| TC-13-04 | Missing state parameter is rejected — CSRF protection | Error guessing — CSRF | OIDC flow initiated; state stored server-side | Simulate callback without state parameter: GET `/auth/social/google/callback?code=<code>` (no state) | HTTP 400; error indicates missing state; OIDC flow aborted; no user created or logged in | Critical | Security |
| TC-13-05 | Mismatched state parameter is rejected — CSRF protection | Error guessing — CSRF | OIDC flow initiated with `state=abc123` | Simulate callback with tampered state: GET `/auth/social/google/callback?code=<code>&state=tampered999` | HTTP 400; state mismatch detected; flow aborted; no user created or logged in | Critical | Security |
| TC-13-06 | Google id_token with email_verified=false is rejected or requires manual verification | Decision Table | Google mock returns id_token with `email_verified=false` | Complete OIDC callback with unverified-email id_token | HTTP 400 or user created with `email_verified=false` and verification email sent (per product spec); user cannot log in until email verified; no auto-login granted for unverified Google account | High | Security |
| TC-13-07 | Replay of Google authorization code is rejected | Error guessing — code replay | Valid OIDC code exchanged in TC-13-01; same code re-submitted | POST `/auth/social/google/callback` with already-used Google code | HTTP 400; Google mock server responds with `invalid_grant`; no user action taken | High | Security |

---

### US-14 — Rate Limiting (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-14-01 | 5 failed login attempts lock the account | BVA — on-threshold | Valid user account; Redis available | POST `/auth/login` with wrong password 5 times for same email | Attempts 1–4: HTTP 401 (generic invalid credentials); attempt 5: HTTP 429 or 423; account locked; Redis counter for user = 5 | Critical | Security / Functional |
| TC-14-02 | 4 failed attempts — account not yet locked | BVA — below threshold | Same setup as TC-14-01 | POST `/auth/login` with wrong password exactly 4 times | All 4 return HTTP 401; account not locked; 5th correct-password attempt succeeds (HTTP 200) | High | Functional |
| TC-14-03 | Locked account response does not reveal lockout reason | Error guessing — information disclosure | Account locked (TC-14-01) | POST `/auth/login` with correct credentials while locked | HTTP 429 or 423; response body is generic (e.g., `"Too many requests"` or `"Account temporarily locked"`); does NOT state "account locked due to too many failed attempts" in a way distinguishable from rate limiting; must not confirm the account exists | High | Security |
| TC-14-04 | IP-level rate limit triggers after threshold | BVA — IP threshold | Multiple accounts accessible; Redis available | POST `/auth/login` with different user emails from same IP, all with wrong passwords — exceed IP-level threshold (e.g., 20 attempts/minute per spec) | HTTP 429 with `Retry-After` header; IP blocked for remainder of window regardless of target email | High | Security |
| TC-14-05 | Redis counter resets after sliding window expiry | BVA — window boundary | Account locked from TC-14-01 | Wait for sliding window to expire (manipulate Redis TTL in test); attempt login again | HTTP 401 (not 429); Redis counter reset to 0 + 1 for this new failed attempt; account unlocked | High | Functional |
| TC-14-06 | Rate limit counters are tenant-scoped — tenant_a lockout does not affect tenant_b | Error guessing — cross-tenant | Same email `alice@example.com` exists in both tenant_a and tenant_b | Lock `alice@example.com` in tenant_a (5 failures); attempt login for same email in tenant_b | Tenant_a login locked; tenant_b login succeeds (if credentials correct); counters are scoped by `tenant_id + email`, not just email | High | Security / Functional |
| TC-14-07 | Successful login resets failed attempt counter | State Transition — reset on success | 3 failed attempts recorded | POST `/auth/login` with correct password (success) | HTTP 200; Redis counter for user reset to 0; subsequent 5 failures from fresh start are needed to lock | Medium | Functional |

---

### US-15 — Audit Log (5 pts)

| TC-ID | Title | ISTQB Technique | Preconditions | Steps | Expected Result | Priority | Type |
|---|---|---|---|---|---|---|---|
| TC-15-01 | LOGIN_SUCCESS event written with correct fields | Specification-based | User logs in successfully (TC-03-01) | Query `audit_logs` table in tenant schema after login | Record exists with: `event_type=LOGIN_SUCCESS`, `user_id=<correct-uuid>`, `tenant_id=<correct-uuid>`, `ip_address=<request-IP>`, `user_agent=<request-UA>`, `timestamp` within 1 second of login; `metadata` may contain token_family_id | Critical | Functional |
| TC-15-02 | LOGIN_FAILURE event written on wrong password | Specification-based | Failed login attempt (TC-03-03) | Query `audit_logs` for event after failed login | Record exists with: `event_type=LOGIN_FAILURE`, `email=<attempted-email>` (not user_id since user may not exist), `tenant_id`, `ip_address`, `timestamp`; failure reason NOT logged in metadata (avoid logging "wrong password" verbatim to prevent log enumeration) | High | Functional |
| TC-15-03 | SUSPICIOUS_TOKEN_REUSE event written on refresh token replay | Specification-based | TC-04-03 completed | Query `audit_logs` for event | Record exists with: `event_type=SUSPICIOUS_TOKEN_REUSE`, `user_id`, `tenant_id`, `ip_address`, `token_family_id`, `timestamp`; event written within 1 second of detection | Critical | Security / Functional |
| TC-15-04 | Audit log is append-only — DELETE is not permitted | Error guessing — integrity | Audit records exist; DB user used by application is service account | Attempt `DELETE FROM audit_logs WHERE ...` using the application's DB service account credentials | Permission denied (SQLSTATE 42501); no records deleted; DB role for application does not have DELETE or UPDATE privilege on `audit_logs` table | Critical | Security |
| TC-15-05 | Audit log is append-only — UPDATE is not permitted | Error guessing — integrity | Same as TC-15-04 | Attempt `UPDATE audit_logs SET event_type='ALTERED' WHERE ...` via application service account | Permission denied; no records modified | Critical | Security |
| TC-15-06 | Admin API returns only own tenant's audit events | Specification-based | Two tenants each with audit events; admin_a authenticated for tenant_a | GET `/admin/audit-logs` with admin_a Bearer and `X-Tenant-ID: tenant_a` | HTTP 200; response contains only tenant_a audit events; no tenant_b events present; total count matches tenant_a-only query against DB | Critical | Security |
| TC-15-07 | Admin from tenant_a cannot query tenant_b audit log | Error guessing — cross-tenant | admin_a JWT for tenant_a | GET `/admin/audit-logs` with admin_a JWT and `X-Tenant-ID: tenant_b` | HTTP 403; no audit events returned; middleware blocks JWT/header tenant mismatch | Critical | Security |
| TC-15-08 | Audit events include all required fields per specification | Specification-based | Multiple different event types triggered | Query audit_logs for records of types: LOGIN_SUCCESS, LOGOUT, ROLE_ASSIGNED, PASSWORD_RESET | Every record contains non-null values for: `id`, `event_type`, `tenant_id`, `timestamp`; context-appropriate fields populated (user_id for user-actions, ip_address for all network events) | High | Functional |
| TC-15-09 | Audit log retention — records older than 1 year not purged by application | Specification-based (ADR-007) | Audit record with timestamp > 1 year ago manually inserted | Run application cleanup job (if any); query for old record | Record still present; application does not delete records within 1-year retention window; deletion only permitted via controlled archival process outside application service account | High | Functional |

---

*End of Sections 1, 2, and 3.*

---

# Section 4: Test Scripts

## 4a. Go Unit Tests — Auth Service

File: `internal/service/auth_service_test.go`

```go
package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/yourorg/auth/internal/domain"
	"github.com/yourorg/auth/internal/service"
)

// ---------------------------------------------------------------------------
// Mock: UserRepository
// ---------------------------------------------------------------------------

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, tenantID, email string) (*domain.User, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) FindByID(ctx context.Context, tenantID, userID string) (*domain.User, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) IncrementFailedAttempts(ctx context.Context, tenantID, userID string) (int, error) {
	args := m.Called(ctx, tenantID, userID)
	return args.Int(0), args.Error(1)
}

func (m *MockUserRepository) ResetFailedAttempts(ctx context.Context, tenantID, userID string) error {
	args := m.Called(ctx, tenantID, userID)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Mock: SessionRepository
// ---------------------------------------------------------------------------

type MockSessionRepository struct {
	mock.Mock
}

func (m *MockSessionRepository) Create(ctx context.Context, session *domain.Session) error {
	args := m.Called(ctx, session)
	return args.Error(0)
}

func (m *MockSessionRepository) FindByRefreshTokenHash(ctx context.Context, tenantID, hash string) (*domain.Session, error) {
	args := m.Called(ctx, tenantID, hash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Session), args.Error(1)
}

func (m *MockSessionRepository) RevokeByID(ctx context.Context, tenantID, sessionID string) error {
	args := m.Called(ctx, tenantID, sessionID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeAllForUser(ctx context.Context, tenantID, userID string) error {
	args := m.Called(ctx, tenantID, userID)
	return args.Error(0)
}

func (m *MockSessionRepository) RevokeFamilyByID(ctx context.Context, tenantID, familyID string) error {
	args := m.Called(ctx, tenantID, familyID)
	return args.Error(0)
}

func (m *MockSessionRepository) ListActiveByUser(ctx context.Context, tenantID, userID string) ([]*domain.Session, error) {
	args := m.Called(ctx, tenantID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Session), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock: AuditRepository
// ---------------------------------------------------------------------------

type MockAuditRepository struct {
	mock.Mock
}

func (m *MockAuditRepository) Log(ctx context.Context, entry *domain.AuditEntry) error {
	args := m.Called(ctx, entry)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Mock: TokenService
// ---------------------------------------------------------------------------

type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) IssueAccessToken(ctx context.Context, user *domain.User, tenantID string) (string, error) {
	args := m.Called(ctx, user, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) IssueRefreshToken(ctx context.Context, sessionID, familyID string) (string, string, error) {
	args := m.Called(ctx, sessionID, familyID)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockTokenService) HashToken(token string) string {
	args := m.Called(token)
	return args.String(0)
}

func (m *MockTokenService) ValidateAccessToken(ctx context.Context, tokenStr string) (*domain.Claims, error) {
	args := m.Called(ctx, tokenStr)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Claims), args.Error(1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newAuthService(
	userRepo *MockUserRepository,
	sessionRepo *MockSessionRepository,
	auditRepo *MockAuditRepository,
	tokenSvc *MockTokenService,
) service.AuthService {
	return service.NewAuthService(userRepo, sessionRepo, auditRepo, tokenSvc)
}

func mustHashPassword(plain string) string {
	hash, err := domain.HashPassword(plain)
	if err != nil {
		panic(err)
	}
	return hash
}

// ---------------------------------------------------------------------------
// TestAuthService_Login
// ---------------------------------------------------------------------------

func TestAuthService_Login(t *testing.T) {
	const (
		tenantID = "tenant-abc"
		email    = "user@example.com"
		password = "SuperSecret123!"
	)

	type setupFn func(
		userRepo *MockUserRepository,
		sessionRepo *MockSessionRepository,
		auditRepo *MockAuditRepository,
		tokenSvc *MockTokenService,
	)

	tests := []struct {
		name        string
		email       string
		password    string
		setup       setupFn
		wantErr     error
		wantAccess  bool
		wantRefresh bool
	}{
		{
			name:     "success",
			email:    email,
			password: password,
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				user := &domain.User{
					ID:             "user-001",
					TenantID:       tenantID,
					Email:          email,
					PasswordHash:   mustHashPassword(password),
					Status:         domain.UserStatusActive,
					FailedAttempts: 0,
					Roles:          []string{"user"},
				}
				userRepo.On("FindByEmail", mock.Anything, tenantID, email).Return(user, nil)
				userRepo.On("ResetFailedAttempts", mock.Anything, tenantID, user.ID).Return(nil)
				sessionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Session")).Return(nil)
				tokenSvc.On("IssueAccessToken", mock.Anything, user, tenantID).Return("access-token-xyz", nil)
				tokenSvc.On("IssueRefreshToken", mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return("refresh-token-abc", "hash-abc", nil)
				tokenSvc.On("HashToken", "refresh-token-abc").Return("hash-abc")
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventLoginSuccess && e.TenantID == tenantID
				})).Return(nil)
			},
			wantAccess:  true,
			wantRefresh: true,
		},
		{
			name:     "wrong_password",
			email:    email,
			password: "WrongPass999!",
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				user := &domain.User{
					ID:           "user-001",
					TenantID:     tenantID,
					Email:        email,
					PasswordHash: mustHashPassword(password),
					Status:       domain.UserStatusActive,
				}
				userRepo.On("FindByEmail", mock.Anything, tenantID, email).Return(user, nil)
				userRepo.On("IncrementFailedAttempts", mock.Anything, tenantID, user.ID).Return(1, nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventLoginFailed
				})).Return(nil)
			},
			wantErr: service.ErrInvalidCredentials,
		},
		{
			name:     "unverified_account",
			email:    email,
			password: password,
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				user := &domain.User{
					ID:           "user-001",
					TenantID:     tenantID,
					Email:        email,
					PasswordHash: mustHashPassword(password),
					Status:       domain.UserStatusUnverified,
				}
				userRepo.On("FindByEmail", mock.Anything, tenantID, email).Return(user, nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventLoginFailed
				})).Return(nil)
			},
			wantErr: service.ErrAccountNotVerified,
		},
		{
			name:     "locked_account",
			email:    email,
			password: password,
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				user := &domain.User{
					ID:             "user-001",
					TenantID:       tenantID,
					Email:          email,
					PasswordHash:   mustHashPassword(password),
					Status:         domain.UserStatusLocked,
					FailedAttempts: 5,
				}
				userRepo.On("FindByEmail", mock.Anything, tenantID, email).Return(user, nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventLoginFailed
				})).Return(nil)
			},
			wantErr: service.ErrAccountLocked,
		},
		{
			name:     "user_not_found",
			email:    "ghost@example.com",
			password: password,
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				userRepo.On("FindByEmail", mock.Anything, tenantID, "ghost@example.com").Return(nil, domain.ErrNotFound)
			},
			wantErr: service.ErrInvalidCredentials,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := new(MockUserRepository)
			sessionRepo := new(MockSessionRepository)
			auditRepo := new(MockAuditRepository)
			tokenSvc := new(MockTokenService)

			tc.setup(userRepo, sessionRepo, auditRepo, tokenSvc)

			svc := newAuthService(userRepo, sessionRepo, auditRepo, tokenSvc)
			result, err := svc.Login(context.Background(), service.LoginInput{
				TenantID: tenantID,
				Email:    tc.email,
				Password: tc.password,
			})

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tc.wantAccess {
					assert.NotEmpty(t, result.AccessToken)
				}
				if tc.wantRefresh {
					assert.NotEmpty(t, result.RefreshToken)
				}
			}

			mock.AssertExpectations(t, userRepo, sessionRepo, auditRepo, tokenSvc)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthService_Register
// ---------------------------------------------------------------------------

func TestAuthService_Register(t *testing.T) {
	const tenantID = "tenant-abc"

	tests := []struct {
		name    string
		input   service.RegisterInput
		setup   func(userRepo *MockUserRepository, auditRepo *MockAuditRepository)
		wantErr error
	}{
		{
			name: "success",
			input: service.RegisterInput{
				TenantID: tenantID,
				Email:    "newuser@example.com",
				Password: "ValidPass123!",
			},
			setup: func(userRepo *MockUserRepository, auditRepo *MockAuditRepository) {
				userRepo.On("FindByEmail", mock.Anything, tenantID, "newuser@example.com").Return(nil, domain.ErrNotFound)
				userRepo.On("Create", mock.Anything, mock.MatchedBy(func(u *domain.User) bool {
					return u.Email == "newuser@example.com" &&
						u.TenantID == tenantID &&
						u.Status == domain.UserStatusUnverified &&
						u.PasswordHash != ""
				})).Return(nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventUserRegistered
				})).Return(nil)
			},
		},
		{
			name: "duplicate_email",
			input: service.RegisterInput{
				TenantID: tenantID,
				Email:    "existing@example.com",
				Password: "ValidPass123!",
			},
			setup: func(userRepo *MockUserRepository, auditRepo *MockAuditRepository) {
				userRepo.On("FindByEmail", mock.Anything, tenantID, "existing@example.com").Return(&domain.User{
					ID:     "user-existing",
					Email:  "existing@example.com",
					Status: domain.UserStatusActive,
				}, nil)
			},
			wantErr: service.ErrEmailAlreadyExists,
		},
		{
			name: "password_too_short",
			input: service.RegisterInput{
				TenantID: tenantID,
				Email:    "user@example.com",
				Password: "short",
			},
			setup: func(userRepo *MockUserRepository, auditRepo *MockAuditRepository) {},
			wantErr: service.ErrPasswordTooShort,
		},
		{
			name: "invalid_email",
			input: service.RegisterInput{
				TenantID: tenantID,
				Email:    "not-an-email",
				Password: "ValidPass123!",
			},
			setup: func(userRepo *MockUserRepository, auditRepo *MockAuditRepository) {},
			wantErr: service.ErrInvalidEmail,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := new(MockUserRepository)
			sessionRepo := new(MockSessionRepository)
			auditRepo := new(MockAuditRepository)
			tokenSvc := new(MockTokenService)

			tc.setup(userRepo, auditRepo)

			svc := newAuthService(userRepo, sessionRepo, auditRepo, tokenSvc)
			user, err := svc.Register(context.Background(), tc.input)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, user)
			} else {
				require.NoError(t, err)
				require.NotNil(t, user)
				assert.Equal(t, tc.input.Email, user.Email)
				assert.Equal(t, domain.UserStatusUnverified, user.Status)
				assert.Empty(t, user.PasswordHash, "password hash must not be returned in response")
			}

			mock.AssertExpectations(t, userRepo, sessionRepo, auditRepo, tokenSvc)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthService_RefreshToken
// ---------------------------------------------------------------------------

func TestAuthService_RefreshToken(t *testing.T) {
	const (
		tenantID = "tenant-abc"
		userID   = "user-001"
		familyID = "family-xyz"
	)

	tests := []struct {
		name         string
		refreshToken string
		setup        func(
			userRepo *MockUserRepository,
			sessionRepo *MockSessionRepository,
			auditRepo *MockAuditRepository,
			tokenSvc *MockTokenService,
		)
		wantErr     error
		wantNewPair bool
	}{
		{
			name:         "success_with_rotation",
			refreshToken: "valid-refresh-token",
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				tokenHash := "hash-of-valid-refresh-token"
				oldSession := &domain.Session{
					ID:               "session-001",
					TenantID:         tenantID,
					UserID:           userID,
					FamilyID:         familyID,
					RefreshTokenHash: tokenHash,
					ExpiresAt:        time.Now().Add(24 * time.Hour),
					Revoked:          false,
				}
				user := &domain.User{
					ID:       userID,
					TenantID: tenantID,
					Email:    "user@example.com",
					Status:   domain.UserStatusActive,
					Roles:    []string{"user"},
				}

				tokenSvc.On("HashToken", "valid-refresh-token").Return(tokenHash)
				sessionRepo.On("FindByRefreshTokenHash", mock.Anything, tenantID, tokenHash).Return(oldSession, nil)
				userRepo.On("FindByID", mock.Anything, tenantID, userID).Return(user, nil)
				sessionRepo.On("RevokeByID", mock.Anything, tenantID, "session-001").Return(nil)
				sessionRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.Session")).Return(nil)
				tokenSvc.On("IssueAccessToken", mock.Anything, user, tenantID).Return("new-access-token", nil)
				tokenSvc.On("IssueRefreshToken", mock.Anything, mock.AnythingOfType("string"), familyID).Return("new-refresh-token", "new-hash", nil)
				tokenSvc.On("HashToken", "new-refresh-token").Return("new-hash")
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventTokenRefreshed
				})).Return(nil)
			},
			wantNewPair: true,
		},
		{
			name:         "expired_token",
			refreshToken: "expired-refresh-token",
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				tokenHash := "hash-of-expired-token"
				expiredSession := &domain.Session{
					ID:               "session-expired",
					TenantID:         tenantID,
					UserID:           userID,
					FamilyID:         familyID,
					RefreshTokenHash: tokenHash,
					ExpiresAt:        time.Now().Add(-1 * time.Hour),
					Revoked:          false,
				}
				tokenSvc.On("HashToken", "expired-refresh-token").Return(tokenHash)
				sessionRepo.On("FindByRefreshTokenHash", mock.Anything, tenantID, tokenHash).Return(expiredSession, nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventTokenRefreshFailed
				})).Return(nil)
			},
			wantErr: service.ErrRefreshTokenExpired,
		},
		{
			name:         "reused_rotated_token_triggers_family_revocation",
			refreshToken: "rotated-old-token",
			setup: func(userRepo *MockUserRepository, sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				tokenHash := "hash-of-rotated-old-token"
				revokedSession := &domain.Session{
					ID:               "session-revoked",
					TenantID:         tenantID,
					UserID:           userID,
					FamilyID:         familyID,
					RefreshTokenHash: tokenHash,
					ExpiresAt:        time.Now().Add(24 * time.Hour),
					Revoked:          true,
				}
				tokenSvc.On("HashToken", "rotated-old-token").Return(tokenHash)
				sessionRepo.On("FindByRefreshTokenHash", mock.Anything, tenantID, tokenHash).Return(revokedSession, nil)
				sessionRepo.On("RevokeFamilyByID", mock.Anything, tenantID, familyID).Return(nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventSuspiciousTokenReuse &&
						e.TenantID == tenantID &&
						e.UserID == userID
				})).Return(nil)
			},
			wantErr: service.ErrRefreshTokenReused,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := new(MockUserRepository)
			sessionRepo := new(MockSessionRepository)
			auditRepo := new(MockAuditRepository)
			tokenSvc := new(MockTokenService)

			tc.setup(userRepo, sessionRepo, auditRepo, tokenSvc)

			svc := newAuthService(userRepo, sessionRepo, auditRepo, tokenSvc)
			result, err := svc.RefreshToken(context.Background(), service.RefreshTokenInput{
				TenantID:     tenantID,
				RefreshToken: tc.refreshToken,
			})

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				if tc.wantNewPair {
					assert.NotEmpty(t, result.AccessToken)
					assert.NotEmpty(t, result.RefreshToken)
					assert.NotEqual(t, "valid-refresh-token", result.RefreshToken, "token must be rotated")
				}
			}

			mock.AssertExpectations(t, userRepo, sessionRepo, auditRepo, tokenSvc)
		})
	}
}

// ---------------------------------------------------------------------------
// TestAuthService_Logout
// ---------------------------------------------------------------------------

func TestAuthService_Logout(t *testing.T) {
	const (
		tenantID     = "tenant-abc"
		userID       = "user-001"
		refreshToken = "current-refresh-token"
		tokenHash    = "hash-of-current-token"
		sessionID    = "session-001"
	)

	tests := []struct {
		name    string
		allDevices bool
		setup   func(
			sessionRepo *MockSessionRepository,
			auditRepo *MockAuditRepository,
			tokenSvc *MockTokenService,
		)
		wantErr error
	}{
		{
			name:       "single_device",
			allDevices: false,
			setup: func(sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				session := &domain.Session{
					ID:               sessionID,
					TenantID:         tenantID,
					UserID:           userID,
					RefreshTokenHash: tokenHash,
					Revoked:          false,
				}
				tokenSvc.On("HashToken", refreshToken).Return(tokenHash)
				sessionRepo.On("FindByRefreshTokenHash", mock.Anything, tenantID, tokenHash).Return(session, nil)
				sessionRepo.On("RevokeByID", mock.Anything, tenantID, sessionID).Return(nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventLogout && e.UserID == userID
				})).Return(nil)
			},
		},
		{
			name:       "all_devices",
			allDevices: true,
			setup: func(sessionRepo *MockSessionRepository, auditRepo *MockAuditRepository, tokenSvc *MockTokenService) {
				session := &domain.Session{
					ID:               sessionID,
					TenantID:         tenantID,
					UserID:           userID,
					RefreshTokenHash: tokenHash,
					Revoked:          false,
				}
				tokenSvc.On("HashToken", refreshToken).Return(tokenHash)
				sessionRepo.On("FindByRefreshTokenHash", mock.Anything, tenantID, tokenHash).Return(session, nil)
				sessionRepo.On("RevokeAllForUser", mock.Anything, tenantID, userID).Return(nil)
				auditRepo.On("Log", mock.Anything, mock.MatchedBy(func(e *domain.AuditEntry) bool {
					return e.EventType == domain.AuditEventLogoutAll && e.UserID == userID
				})).Return(nil)
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			userRepo := new(MockUserRepository)
			sessionRepo := new(MockSessionRepository)
			auditRepo := new(MockAuditRepository)
			tokenSvc := new(MockTokenService)

			tc.setup(sessionRepo, auditRepo, tokenSvc)

			svc := newAuthService(userRepo, sessionRepo, auditRepo, tokenSvc)
			err := svc.Logout(context.Background(), service.LogoutInput{
				TenantID:     tenantID,
				RefreshToken: refreshToken,
				AllDevices:   tc.allDevices,
			})

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}

			mock.AssertExpectations(t, userRepo, sessionRepo, auditRepo, tokenSvc)
		})
	}
}
```

---

## 4b. Go Integration Tests — Auth API

File: `test/integration/auth_api_test.go`

```go
package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourorg/auth/internal/bootstrap"
	"github.com/yourorg/auth/internal/migrate"
)

// ---------------------------------------------------------------------------
// Package-level state
// ---------------------------------------------------------------------------

var (
	testDB     *pgxpool.Pool
	testRouter *gin.Engine
)

// ---------------------------------------------------------------------------
// TestMain
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/auth_test?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to test database: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err = testDB.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "test database not reachable: %v\n", err)
		os.Exit(1)
	}

	if err = migrate.RunGlobalSchema(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "global schema migration failed: %v\n", err)
		os.Exit(1)
	}

	if err = seedGlobalSchema(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "global schema seed failed: %v\n", err)
		os.Exit(1)
	}

	testRouter = bootstrap.NewRouter(testDB)

	code := m.Run()

	os.Exit(code)
}

func seedGlobalSchema(ctx context.Context, db *pgxpool.Pool) error {
	_, err := db.Exec(ctx, `
		INSERT INTO global.plans (id, name, max_users, max_api_clients)
		VALUES ('plan-free', 'Free', 5, 2)
		ON CONFLICT (id) DO NOTHING
	`)
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type Tenant struct {
	ID   string
	Slug string
}

type AuthTokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

func createTestTenant(t *testing.T, slug string) Tenant {
	t.Helper()

	ctx := context.Background()
	tenantID := fmt.Sprintf("tenant-%s", slug)

	_, err := testDB.Exec(ctx, `
		INSERT INTO global.tenants (id, slug, plan_id, status)
		VALUES ($1, $2, 'plan-free', 'active')
		ON CONFLICT (slug) DO NOTHING
	`, tenantID, slug)
	require.NoError(t, err, "create tenant record")

	_, err = testDB.Exec(ctx, fmt.Sprintf(`
		CREATE SCHEMA IF NOT EXISTS "tenant_%s";
	`, slug))
	require.NoError(t, err, "create tenant schema")

	err = migrate.RunTenantSchema(ctx, testDB, slug)
	require.NoError(t, err, "run tenant schema migrations")

	t.Cleanup(func() {
		cleanupTenant(t, slug, tenantID)
	})

	return Tenant{ID: tenantID, Slug: slug}
}

func cleanupTenant(t *testing.T, slug, tenantID string) {
	t.Helper()
	ctx := context.Background()

	_, _ = testDB.Exec(ctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS "tenant_%s" CASCADE`, slug))
	_, _ = testDB.Exec(ctx, `DELETE FROM global.tenants WHERE id = $1`, tenantID)
}

func createTestUser(t *testing.T, tenantSlug, email, password string) map[string]interface{} {
	t.Helper()

	body := map[string]string{
		"email":    email,
		"password": password,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantSlug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code, "create test user: unexpected status: %s", w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

func verifyTestUser(t *testing.T, tenantSlug, userID string) {
	t.Helper()
	ctx := context.Background()

	_, err := testDB.Exec(ctx, fmt.Sprintf(`
		UPDATE "tenant_%s".users SET status = 'active' WHERE id = $1
	`, tenantSlug), userID)
	require.NoError(t, err)
}

func loginUser(t *testing.T, tenantSlug, email, password string) AuthTokens {
	t.Helper()

	body := map[string]string{
		"email":    email,
		"password": password,
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantSlug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "login user: unexpected status: %s", w.Body.String())

	var tokens AuthTokens
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &tokens))
	require.NotEmpty(t, tokens.AccessToken)
	require.NotEmpty(t, tokens.RefreshToken)
	return tokens
}

func doRequest(t *testing.T, method, path string, body interface{}, tenantSlug, bearerToken string) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if tenantSlug != "" {
		req.Header.Set("X-Tenant-ID", tenantSlug)
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Integration Tests
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("reg-success-%d", time.Now().UnixNano()))

	body := map[string]string{
		"email":    "newuser@example.com",
		"password": "ValidPass123!",
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.Slug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.NotEmpty(t, resp["user_id"])

	// Verify user exists in DB with status=unverified
	userID := resp["user_id"].(string)
	var status string
	err = testDB.QueryRow(context.Background(), fmt.Sprintf(`
		SELECT status FROM "tenant_%s".users WHERE id = $1
	`, tenant.Slug), userID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "unverified", status)
}

func TestRegister_DuplicateEmail(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("reg-dup-%d", time.Now().UnixNano()))
	email := "dup@example.com"

	createTestUser(t, tenant.Slug, email, "ValidPass123!")

	// Second registration with same email
	body := map[string]string{
		"email":    email,
		"password": "AnotherPass456!",
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.Slug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// Error body must NOT reveal whether the account is verified, locked, etc.
	body2, err := json.Marshal(resp)
	require.NoError(t, err)
	respStr := string(body2)
	assert.NotContains(t, respStr, "verified")
	assert.NotContains(t, respStr, "locked")
	assert.NotContains(t, respStr, "active")
	assert.NotContains(t, respStr, "unverified")
}

func TestLogin_Success(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("login-ok-%d", time.Now().UnixNano()))
	email := "loginok@example.com"
	password := "ValidPass123!"

	userResp := createTestUser(t, tenant.Slug, email, password)
	verifyTestUser(t, tenant.Slug, userResp["user_id"].(string))

	tokens := loginUser(t, tenant.Slug, email, password)

	assert.NotEmpty(t, tokens.AccessToken)
	assert.NotEmpty(t, tokens.RefreshToken)
	assert.Equal(t, "Bearer", tokens.TokenType)

	// Validate JWT claims
	claims := parseJWTClaims(t, tokens.AccessToken)

	assert.Equal(t, tenant.ID, claims["tenant_id"], "tenant_id claim must match")
	roles, ok := claims["roles"].([]interface{})
	require.True(t, ok, "roles claim must be an array")
	assert.NotEmpty(t, roles)

	// Access token must expire in approximately 15 minutes (±30 seconds)
	exp, ok := claims["exp"].(float64)
	require.True(t, ok, "exp claim must be present")
	iat, ok := claims["iat"].(float64)
	require.True(t, ok, "iat claim must be present")
	ttlSeconds := exp - iat
	assert.InDelta(t, 900, ttlSeconds, 30, "access token TTL must be ~15min")
}

func TestLogin_WrongPassword(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("login-bad-%d", time.Now().UnixNano()))
	email := "loginbad@example.com"

	userResp := createTestUser(t, tenant.Slug, email, "CorrectPass123!")
	verifyTestUser(t, tenant.Slug, userResp["user_id"].(string))

	body := map[string]string{
		"email":    email,
		"password": "WrongPassword999!",
	}
	bodyBytes, err := json.Marshal(body)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenant.Slug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	// Error message must be ambiguous — must not reveal "wrong password" vs "no account"
	msg, _ := resp["error"].(string)
	assert.NotContains(t, msg, "password")
	assert.NotContains(t, msg, "not found")
	assert.NotContains(t, msg, "does not exist")
}

func TestRefreshToken_Rotation(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("refresh-rotate-%d", time.Now().UnixNano()))
	email := "refreshtest@example.com"
	password := "ValidPass123!"

	userResp := createTestUser(t, tenant.Slug, email, password)
	verifyTestUser(t, tenant.Slug, userResp["user_id"].(string))
	tokens := loginUser(t, tenant.Slug, email, password)

	// First refresh
	w := doRequest(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, tenant.Slug, "")

	assert.Equal(t, http.StatusOK, w.Code, "first refresh must succeed: %s", w.Body.String())

	var newTokens AuthTokens
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newTokens))
	assert.NotEmpty(t, newTokens.AccessToken)
	assert.NotEmpty(t, newTokens.RefreshToken)
	assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken, "refresh token must be rotated")

	// Old token must be rejected
	w2 := doRequest(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, tenant.Slug, "")

	assert.Equal(t, http.StatusUnauthorized, w2.Code, "old refresh token must be rejected after rotation")
}

func TestRefreshToken_ReuseDetection(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("refresh-reuse-%d", time.Now().UnixNano()))
	email := "reusetest@example.com"
	password := "ValidPass123!"

	userResp := createTestUser(t, tenant.Slug, email, password)
	userID := userResp["user_id"].(string)
	verifyTestUser(t, tenant.Slug, userID)
	tokens := loginUser(t, tenant.Slug, email, password)

	// Use the token once to rotate it
	w := doRequest(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, tenant.Slug, "")
	require.Equal(t, http.StatusOK, w.Code, "initial refresh must succeed")

	var newTokens AuthTokens
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newTokens))

	// Reuse the old (now rotated) token — must detect reuse
	w2 := doRequest(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, tenant.Slug, "")
	assert.Equal(t, http.StatusUnauthorized, w2.Code, "reused token must be rejected")

	// Both the old and new tokens must now be revoked
	w3 := doRequest(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": newTokens.RefreshToken,
	}, tenant.Slug, "")
	assert.Equal(t, http.StatusUnauthorized, w3.Code, "new token from compromised family must also be revoked")

	// Verify SUSPICIOUS_TOKEN_REUSE in audit log
	var eventType string
	err := testDB.QueryRow(context.Background(), fmt.Sprintf(`
		SELECT event_type FROM "tenant_%s".audit_log
		WHERE user_id = $1 AND event_type = 'SUSPICIOUS_TOKEN_REUSE'
		ORDER BY created_at DESC LIMIT 1
	`, tenant.Slug), userID).Scan(&eventType)
	require.NoError(t, err, "SUSPICIOUS_TOKEN_REUSE audit entry must exist")
	assert.Equal(t, "SUSPICIOUS_TOKEN_REUSE", eventType)
}

func TestLogout_AllDevices(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("logout-all-%d", time.Now().UnixNano()))
	email := "logoutall@example.com"
	password := "ValidPass123!"

	userResp := createTestUser(t, tenant.Slug, email, password)
	verifyTestUser(t, tenant.Slug, userResp["user_id"].(string))

	// Create multiple sessions
	session1 := loginUser(t, tenant.Slug, email, password)
	session2 := loginUser(t, tenant.Slug, email, password)
	session3 := loginUser(t, tenant.Slug, email, password)

	// Logout from all devices using session1's access token
	w := doRequest(t, http.MethodPost, "/auth/logout/all", map[string]string{
		"refresh_token": session1.RefreshToken,
	}, tenant.Slug, session1.AccessToken)
	assert.Equal(t, http.StatusNoContent, w.Code, "logout/all must succeed: %s", w.Body.String())

	// All refresh tokens must now be rejected
	for i, tok := range []string{session1.RefreshToken, session2.RefreshToken, session3.RefreshToken} {
		w := doRequest(t, http.MethodPost, "/auth/refresh", map[string]string{
			"refresh_token": tok,
		}, tenant.Slug, "")
		assert.Equal(t, http.StatusUnauthorized, w.Code, "session %d must be revoked after logout/all", i+1)
	}
}

func TestForgotPassword_Timing(t *testing.T) {
	tenant := createTestTenant(t, fmt.Sprintf("timing-%d", time.Now().UnixNano()))
	knownEmail := "known@example.com"

	userResp := createTestUser(t, tenant.Slug, knownEmail, "ValidPass123!")
	verifyTestUser(t, tenant.Slug, userResp["user_id"].(string))

	const iterations = 10
	knownDurations := make([]time.Duration, 0, iterations)
	unknownDurations := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		// Known email
		start := time.Now()
		w := doRequest(t, http.MethodPost, "/auth/forgot-password", map[string]string{
			"email": knownEmail,
		}, tenant.Slug, "")
		knownDurations = append(knownDurations, time.Since(start))
		assert.Equal(t, http.StatusAccepted, w.Code)

		// Unknown email
		start = time.Now()
		w = doRequest(t, http.MethodPost, "/auth/forgot-password", map[string]string{
			"email": fmt.Sprintf("ghost%d@example.com", i),
		}, tenant.Slug, "")
		unknownDurations = append(unknownDurations, time.Since(start))
		assert.Equal(t, http.StatusAccepted, w.Code)
	}

	avgKnown := avgDuration(knownDurations)
	avgUnknown := avgDuration(unknownDurations)

	diff := avgKnown - avgUnknown
	if diff < 0 {
		diff = -diff
	}
	assert.Less(t, diff.Milliseconds(), int64(100),
		"timing difference between known and unknown email must be <100ms (got %dms)", diff.Milliseconds())
}

// ---------------------------------------------------------------------------
// JWT helper
// ---------------------------------------------------------------------------

func parseJWTClaims(t *testing.T, tokenStr string) map[string]interface{} {
	t.Helper()

	parts := bytes.Split([]byte(tokenStr), []byte("."))
	require.Len(t, parts, 3, "JWT must have 3 parts")

	// Decode payload (base64url, no padding)
	payload := make([]byte, base64URLDecodedLen(len(parts[1])))
	n, err := base64URLDecode(payload, parts[1])
	require.NoError(t, err)

	var claims map[string]interface{}
	require.NoError(t, json.Unmarshal(payload[:n], &claims))
	return claims
}

func base64URLDecodedLen(n int) int {
	return (n*3 + 3) / 4
}

func base64URLDecode(dst, src []byte) (int, error) {
	import "encoding/base64"
	return base64.RawURLEncoding.Decode(dst, src)
}

func avgDuration(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}
	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// math import is used for math.Abs equivalent; using explicit abs above instead
var _ = math.Abs
```

---

## 4c. Cross-Tenant Isolation Suite

File: `test/isolation/tenant_isolation_test.go`

```go
package isolation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourorg/auth/internal/bootstrap"
	"github.com/yourorg/auth/internal/migrate"
)

// ---------------------------------------------------------------------------
// Package-level fixtures
// ---------------------------------------------------------------------------

type TenantFixture struct {
	ID           string
	Slug         string
	AdminToken   string
	AdminUserID  string
	RegularToken string
	RegularUserID string
}

var (
	testDB        *pgxpool.Pool
	testRouter    *gin.Engine
	tenantAlpha   TenantFixture
	tenantBeta    TenantFixture
)

// ---------------------------------------------------------------------------
// TestMain
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/auth_test_isolation?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to isolation test database: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err = migrate.RunGlobalSchema(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "global schema migration failed: %v\n", err)
		os.Exit(1)
	}

	testRouter = bootstrap.NewRouter(testDB)

	tenantAlpha, err = setupTenantFixture(ctx, "alpha")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup tenant-alpha fixture: %v\n", err)
		os.Exit(1)
	}

	tenantBeta, err = setupTenantFixture(ctx, "beta")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to setup tenant-beta fixture: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()

	teardownTenantFixture(ctx, "alpha")
	teardownTenantFixture(ctx, "beta")

	os.Exit(code)
}

func setupTenantFixture(ctx context.Context, name string) (TenantFixture, error) {
	slug := fmt.Sprintf("isolation-%s", name)
	tenantID := fmt.Sprintf("tenant-%s", slug)

	if _, err := testDB.Exec(ctx, `
		INSERT INTO global.tenants (id, slug, plan_id, status)
		VALUES ($1, $2, 'plan-free', 'active')
		ON CONFLICT (slug) DO NOTHING
	`, tenantID, slug); err != nil {
		return TenantFixture{}, fmt.Errorf("insert tenant: %w", err)
	}

	if _, err := testDB.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "tenant_%s"`, slug)); err != nil {
		return TenantFixture{}, fmt.Errorf("create schema: %w", err)
	}

	if err := migrate.RunTenantSchema(ctx, testDB, slug); err != nil {
		return TenantFixture{}, fmt.Errorf("run tenant migrations: %w", err)
	}

	// Register and verify admin user
	adminEmail := fmt.Sprintf("admin-%s@example.com", name)
	adminPassword := "IsolationAdminPass123!"
	adminID, err := registerAndVerifyUser(ctx, slug, adminEmail, adminPassword)
	if err != nil {
		return TenantFixture{}, fmt.Errorf("register admin: %w", err)
	}

	if err = assignRole(ctx, slug, adminID, "admin"); err != nil {
		return TenantFixture{}, fmt.Errorf("assign admin role: %w", err)
	}

	adminToken, err := loginAndGetAccessToken(slug, adminEmail, adminPassword)
	if err != nil {
		return TenantFixture{}, fmt.Errorf("login admin: %w", err)
	}

	// Register and verify regular user
	regularEmail := fmt.Sprintf("user-%s@example.com", name)
	regularPassword := "IsolationUserPass123!"
	regularID, err := registerAndVerifyUser(ctx, slug, regularEmail, regularPassword)
	if err != nil {
		return TenantFixture{}, fmt.Errorf("register regular user: %w", err)
	}

	regularToken, err := loginAndGetAccessToken(slug, regularEmail, regularPassword)
	if err != nil {
		return TenantFixture{}, fmt.Errorf("login regular user: %w", err)
	}

	return TenantFixture{
		ID:            tenantID,
		Slug:          slug,
		AdminToken:    adminToken,
		AdminUserID:   adminID,
		RegularToken:  regularToken,
		RegularUserID: regularID,
	}, nil
}

func teardownTenantFixture(ctx context.Context, name string) {
	slug := fmt.Sprintf("isolation-%s", name)
	tenantID := fmt.Sprintf("tenant-%s", slug)
	testDB.Exec(ctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS "tenant_%s" CASCADE`, slug))
	testDB.Exec(ctx, `DELETE FROM global.tenants WHERE id = $1`, tenantID)
}

// ---------------------------------------------------------------------------
// Isolation Test Helpers
// ---------------------------------------------------------------------------

func registerAndVerifyUser(ctx context.Context, tenantSlug, email, password string) (string, error) {
	body := map[string]string{"email": email, "password": password}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantSlug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		return "", fmt.Errorf("register failed: status %d body %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		return "", err
	}
	userID, _ := resp["user_id"].(string)

	// Directly set user as active in DB
	_, err := testDB.Exec(ctx, fmt.Sprintf(`
		UPDATE "tenant_%s".users SET status = 'active' WHERE id = $1
	`, tenantSlug), userID)
	return userID, err
}

func assignRole(ctx context.Context, tenantSlug, userID, role string) error {
	_, err := testDB.Exec(ctx, fmt.Sprintf(`
		INSERT INTO "tenant_%s".user_roles (user_id, role)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
	`, tenantSlug), userID, role)
	return err
}

func loginAndGetAccessToken(tenantSlug, email, password string) (string, error) {
	body := map[string]string{"email": email, "password": password}
	b, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantSlug)

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		return "", fmt.Errorf("login failed: status %d body %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		return "", err
	}
	token, _ := resp["access_token"].(string)
	if token == "" {
		return "", fmt.Errorf("no access_token in login response")
	}
	return token, nil
}

func doIsolationRequest(t *testing.T, method, path, tenantSlug, bearerToken string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if tenantSlug != "" {
		req.Header.Set("X-Tenant-ID", tenantSlug)
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

// assertNoCrossTenantData scans the JSON response body for any field value
// that contains or equals the forbidden tenant identifier. Any match causes
// an immediate test failure.
func assertNoCrossTenantData(t *testing.T, body []byte, forbiddenTenantID string) {
	t.Helper()

	if len(body) == 0 {
		return
	}

	// String scan — catches tenant IDs embedded in any value
	if strings.Contains(string(body), forbiddenTenantID) {
		t.Fatalf("ISOLATION BREACH: response body contains forbidden tenant data %q\nBody: %s",
			forbiddenTenantID, string(body))
	}

	// Deep JSON scan of every string value in the response tree
	var raw interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return // Non-JSON body, string scan above is sufficient
	}
	scanJSONValues(t, raw, forbiddenTenantID)
}

func scanJSONValues(t *testing.T, v interface{}, forbidden string) {
	t.Helper()
	switch val := v.(type) {
	case string:
		if strings.Contains(val, forbidden) {
			t.Fatalf("ISOLATION BREACH: JSON value %q contains forbidden tenant data %q", val, forbidden)
		}
	case map[string]interface{}:
		for _, child := range val {
			scanJSONValues(t, child, forbidden)
		}
	case []interface{}:
		for _, item := range val {
			scanJSONValues(t, item, forbidden)
		}
	}
}

// ---------------------------------------------------------------------------
// Isolation Test Cases
// ---------------------------------------------------------------------------

func TestTenantIsolation(t *testing.T) {
	t.Run("TC-ISO-01_UserWithAlphaJWT_CannotAccessBetaAdminUsers", func(t *testing.T) {
		// tenant-alpha regular user tries to list tenant-beta admin users endpoint
		w := doIsolationRequest(t, http.MethodGet, "/admin/users",
			tenantBeta.Slug, tenantAlpha.RegularToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"alpha JWT must be rejected on beta admin endpoint, got: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
	})

	t.Run("TC-ISO-02_UserWithAlphaJWT_CannotAccessBetaAuditLog", func(t *testing.T) {
		w := doIsolationRequest(t, http.MethodGet, "/admin/audit-log",
			tenantBeta.Slug, tenantAlpha.AdminToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"alpha JWT must be rejected on beta audit-log endpoint, got: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
	})

	t.Run("TC-ISO-03_AlphaAdmin_CannotListBetaUsers", func(t *testing.T) {
		w := doIsolationRequest(t, http.MethodGet, "/admin/users",
			tenantBeta.Slug, tenantAlpha.AdminToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"alpha admin JWT must be rejected on beta users list, got: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.AdminUserID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.RegularUserID)
	})

	t.Run("TC-ISO-04_AlphaAdmin_CannotAssignRolesToBetaUsers", func(t *testing.T) {
		payload := map[string]string{
			"user_id": tenantBeta.RegularUserID,
			"role":    "admin",
		}
		w := doIsolationRequest(t, http.MethodPost, "/admin/users/roles",
			tenantBeta.Slug, tenantAlpha.AdminToken, payload)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"alpha admin must not be able to assign roles in beta tenant, got: %s", w.Body.String())

		// Verify the role was NOT assigned in beta tenant's database
		var count int
		err := testDB.QueryRow(context.Background(), fmt.Sprintf(`
			SELECT COUNT(*) FROM "tenant_%s".user_roles
			WHERE user_id = $1 AND role = 'admin'
		`, tenantBeta.Slug), tenantBeta.RegularUserID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "role must not have been assigned in beta tenant")
	})

	t.Run("TC-ISO-05_AlphaJWT_CannotAccessBetaOAuthClients", func(t *testing.T) {
		w := doIsolationRequest(t, http.MethodGet, "/oauth/clients",
			tenantBeta.Slug, tenantAlpha.AdminToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"alpha JWT must be rejected on beta OAuth clients endpoint, got: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
	})

	t.Run("TC-ISO-06_AlphaUser_CannotSeeBetaSessions", func(t *testing.T) {
		w := doIsolationRequest(t, http.MethodGet, "/auth/sessions",
			tenantBeta.Slug, tenantAlpha.RegularToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"alpha user must not be able to list beta sessions, got: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.RegularUserID)
	})

	t.Run("TC-ISO-07_ForgedTenantIDClaim_ValidSignatureWrongTenant_Returns403", func(t *testing.T) {
		// tenantAlpha.AdminToken has valid RS256 signature but tenant_id=tenantAlpha.ID
		// When presented to the tenantBeta endpoint, the middleware must detect the mismatch
		w := doIsolationRequest(t, http.MethodGet, "/admin/users",
			tenantBeta.Slug, tenantAlpha.AdminToken, nil)

		assert.Equal(t, http.StatusForbidden, w.Code,
			"valid JWT with wrong tenant_id claim must return 403, got: %s", w.Body.String())

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		// Error must not leak which tenant the token actually belongs to
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantAlpha.ID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
	})

	t.Run("TC-ISO-08_BetaData_NeverAppearsInAlphaResponses", func(t *testing.T) {
		// List users as alpha admin — verify no beta data leaks
		w := doIsolationRequest(t, http.MethodGet, "/admin/users",
			tenantAlpha.Slug, tenantAlpha.AdminToken, nil)

		assert.Equal(t, http.StatusOK, w.Code, "alpha admin users list must succeed: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.Slug)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.AdminUserID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.RegularUserID)

		// Audit log
		w2 := doIsolationRequest(t, http.MethodGet, "/admin/audit-log",
			tenantAlpha.Slug, tenantAlpha.AdminToken, nil)
		assert.Equal(t, http.StatusOK, w2.Code)
		assertNoCrossTenantData(t, w2.Body.Bytes(), tenantBeta.ID)
		assertNoCrossTenantData(t, w2.Body.Bytes(), tenantBeta.AdminUserID)
	})

	t.Run("TC-ISO-09_CrossTenantErrorMessages_ContainNoBetaData", func(t *testing.T) {
		endpoints := []struct {
			method string
			path   string
			body   interface{}
		}{
			{http.MethodGet, "/admin/users", nil},
			{http.MethodGet, "/admin/audit-log", nil},
			{http.MethodGet, "/oauth/clients", nil},
		}

		for _, ep := range endpoints {
			w := doIsolationRequest(t, ep.method, ep.path,
				tenantBeta.Slug, tenantAlpha.AdminToken, ep.body)

			// Must be a 4xx response
			assert.GreaterOrEqual(t, w.Code, 400)
			assert.Less(t, w.Code, 500)

			// Error message must not contain beta tenant data
			assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
			assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.Slug)
			assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.AdminUserID)
		}
	})

	t.Run("TC-ISO-10_CrossTenantPasswordReset_LeaksNoInformation", func(t *testing.T) {
		// Alpha user attempts to reset a beta user's password via beta tenant endpoint
		betaUserEmail := fmt.Sprintf("admin-%s@example.com", "beta")

		w := doIsolationRequest(t, http.MethodPost, "/auth/forgot-password",
			tenantBeta.Slug, "", map[string]string{"email": betaUserEmail})

		// Response must be 202 Accepted regardless (timing-safe)
		assert.Equal(t, http.StatusAccepted, w.Code)

		// Trying with alpha token on beta endpoint must give no information
		w2 := doIsolationRequest(t, http.MethodPost, "/auth/forgot-password",
			tenantBeta.Slug, tenantAlpha.RegularToken, map[string]string{"email": betaUserEmail})

		// Body must be identical or similarly uninformative in both cases
		assert.Equal(t, http.StatusAccepted, w2.Code)
		assertNoCrossTenantData(t, w2.Body.Bytes(), tenantBeta.ID)
		assertNoCrossTenantData(t, w2.Body.Bytes(), tenantBeta.AdminUserID)
	})

	t.Run("TC-ISO-11_AlphaOAuthClient_CannotAuthorize_BetaUsers", func(t *testing.T) {
		// Register an OAuth client in tenant-alpha
		clientPayload := map[string]interface{}{
			"name":          "alpha-app",
			"redirect_uris": []string{"https://alpha-app.example.com/callback"},
			"grant_types":   []string{"authorization_code"},
			"scopes":        []string{"openid", "profile"},
		}
		wClient := doIsolationRequest(t, http.MethodPost, "/oauth/clients",
			tenantAlpha.Slug, tenantAlpha.AdminToken, clientPayload)
		require.Equal(t, http.StatusCreated, wClient.Code, "create alpha OAuth client: %s", wClient.Body.String())

		var clientResp map[string]interface{}
		require.NoError(t, json.Unmarshal(wClient.Body.Bytes(), &clientResp))
		clientID, _ := clientResp["client_id"].(string)
		require.NotEmpty(t, clientID)

		// Attempt to use alpha OAuth client to authorize against beta tenant
		authURL := fmt.Sprintf(
			"/oauth/authorize?response_type=code&client_id=%s&redirect_uri=%s&scope=openid&state=test123",
			clientID, "https%3A%2F%2Falpha-app.example.com%2Fcallback",
		)
		w := doIsolationRequest(t, http.MethodGet, authURL,
			tenantBeta.Slug, tenantBeta.RegularToken, nil)

		// Must reject — client belongs to alpha, not beta
		assert.Equal(t, http.StatusBadRequest, w.Code,
			"alpha OAuth client must be rejected by beta tenant authorization endpoint, got: %s", w.Body.String())
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantAlpha.ID)
	})

	t.Run("TC-ISO-12_AuditLogQuery_NeverReturns_CrossTenantEvents", func(t *testing.T) {
		// Trigger a login event in tenant-alpha to generate an audit entry
		_ = doIsolationRequest(t, http.MethodPost, "/auth/login",
			tenantAlpha.Slug, "", map[string]string{
				"email":    fmt.Sprintf("admin-%s@example.com", "alpha"),
				"password": "IsolationAdminPass123!",
			})

		// Query alpha's audit log as alpha admin
		w := doIsolationRequest(t, http.MethodGet, "/admin/audit-log",
			tenantAlpha.Slug, tenantAlpha.AdminToken, nil)

		require.Equal(t, http.StatusOK, w.Code, "audit log query must succeed: %s", w.Body.String())

		// Verify no beta tenant events appear
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.ID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.Slug)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.AdminUserID)
		assertNoCrossTenantData(t, w.Body.Bytes(), tenantBeta.RegularUserID)

		// Also verify via direct DB query that the alpha schema contains no beta tenant_id values
		var crossTenantCount int
		err := testDB.QueryRow(context.Background(), fmt.Sprintf(`
			SELECT COUNT(*) FROM "tenant_%s".audit_log
			WHERE tenant_id = $1
		`, tenantAlpha.Slug), tenantBeta.ID).Scan(&crossTenantCount)
		require.NoError(t, err)
		assert.Equal(t, 0, crossTenantCount,
			"alpha audit_log schema must contain zero events with beta tenant_id")
	})
}
```

---

## 4d. Playwright API Tests — OAuth 2.0 Flow

File: `test/e2e/oauth_flow.spec.ts`

```typescript
import { test, expect, APIRequestContext, request } from "@playwright/test";
import * as crypto from "crypto";

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const BASE_URL = process.env.TEST_BASE_URL ?? "http://localhost:8080";
const TENANT_ALPHA = process.env.TEST_TENANT_SLUG ?? "e2e-oauth-alpha";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

interface AuthTokens {
  access_token: string;
  refresh_token?: string;
  token_type: string;
  expires_in: number;
}

interface OAuthClient {
  client_id: string;
  client_secret: string;
  name: string;
  redirect_uris: string[];
  grant_types: string[];
  scopes: string[];
}

interface JWTClaims {
  sub?: string;
  tenant_id?: string;
  roles?: string[];
  scope?: string;
  iat?: number;
  exp?: number;
  iss?: string;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// PKCE helpers
// ---------------------------------------------------------------------------

function generateCodeVerifier(): string {
  return crypto.randomBytes(32).toString("base64url");
}

function generateCodeChallenge(verifier: string): string {
  return crypto.createHash("sha256").update(verifier).digest("base64url");
}

function generateState(): string {
  return crypto.randomBytes(16).toString("hex");
}

// ---------------------------------------------------------------------------
// JWT decode (no verification — signature verified by API, we inspect claims)
// ---------------------------------------------------------------------------

function decodeJWTClaims(token: string): JWTClaims {
  const parts = token.split(".");
  if (parts.length !== 3) {
    throw new Error(`Invalid JWT: expected 3 parts, got ${parts.length}`);
  }
  const payload = Buffer.from(parts[1], "base64url").toString("utf-8");
  return JSON.parse(payload) as JWTClaims;
}

// ---------------------------------------------------------------------------
// Helper: register + verify user, return access token
// ---------------------------------------------------------------------------

async function registerAndVerifyUser(
  apiCtx: APIRequestContext,
  tenantSlug: string,
  email: string,
  password: string
): Promise<AuthTokens> {
  const registerResp = await apiCtx.post(`${BASE_URL}/auth/register`, {
    headers: { "X-Tenant-ID": tenantSlug, "Content-Type": "application/json" },
    data: { email, password },
  });
  expect(registerResp.status()).toBe(201);
  const regBody = await registerResp.json();
  const userID: string = regBody.user_id;
  expect(userID).toBeTruthy();

  // Verify the user via the verification endpoint using the token from the
  // email — in test mode the API exposes a shortcut endpoint.
  const verifyResp = await apiCtx.post(`${BASE_URL}/auth/verify-email/bypass`, {
    headers: { "X-Tenant-ID": tenantSlug, "Content-Type": "application/json" },
    data: { user_id: userID },
  });
  expect(verifyResp.status()).toBe(200);

  return loginUser(apiCtx, tenantSlug, email, password);
}

// ---------------------------------------------------------------------------
// Helper: login and return tokens
// ---------------------------------------------------------------------------

async function loginUser(
  apiCtx: APIRequestContext,
  tenantSlug: string,
  email: string,
  password: string
): Promise<AuthTokens> {
  const resp = await apiCtx.post(`${BASE_URL}/auth/login`, {
    headers: { "X-Tenant-ID": tenantSlug, "Content-Type": "application/json" },
    data: { email, password },
  });
  expect(resp.status()).toBe(200);
  const body = await resp.json();
  expect(body.access_token).toBeTruthy();
  return body as AuthTokens;
}

// ---------------------------------------------------------------------------
// Helper: register OAuth client
// ---------------------------------------------------------------------------

async function registerOAuthClient(
  apiCtx: APIRequestContext,
  tenantSlug: string,
  adminToken: string,
  overrides: Partial<{
    name: string;
    redirect_uris: string[];
    grant_types: string[];
    scopes: string[];
  }> = {}
): Promise<OAuthClient> {
  const payload = {
    name: overrides.name ?? `test-client-${Date.now()}`,
    redirect_uris: overrides.redirect_uris ?? [
      "https://app.example.com/callback",
    ],
    grant_types: overrides.grant_types ?? ["authorization_code", "refresh_token"],
    scopes: overrides.scopes ?? ["openid", "profile", "email"],
  };

  const resp = await apiCtx.post(`${BASE_URL}/oauth/clients`, {
    headers: {
      "X-Tenant-ID": tenantSlug,
      "Content-Type": "application/json",
      Authorization: `Bearer ${adminToken}`,
    },
    data: payload,
  });
  expect(resp.status()).toBe(201);
  return (await resp.json()) as OAuthClient;
}

// ---------------------------------------------------------------------------
// Test Suite
// ---------------------------------------------------------------------------

test.describe("OAuth 2.0 Flow", () => {
  let apiCtx: APIRequestContext;
  let adminTokens: AuthTokens;

  test.beforeAll(async () => {
    apiCtx = await request.newContext({ baseURL: BASE_URL });

    adminTokens = await registerAndVerifyUser(
      apiCtx,
      TENANT_ALPHA,
      `oauth-admin-${Date.now()}@example.com`,
      "OAuthAdminPass123!"
    );
  });

  test.afterAll(async () => {
    await apiCtx.dispose();
  });

  // -------------------------------------------------------------------------
  // OAuth Client Registration
  // -------------------------------------------------------------------------

  test.describe("OAuth Client Registration", () => {
    test("POST /oauth/clients → 201 with valid payload", async () => {
      const resp = await apiCtx.post(`${BASE_URL}/oauth/clients`, {
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          "Content-Type": "application/json",
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        data: {
          name: `reg-test-${Date.now()}`,
          redirect_uris: ["https://myapp.example.com/callback"],
          grant_types: ["authorization_code"],
          scopes: ["openid", "profile"],
        },
      });

      expect(resp.status()).toBe(201);
      const body = await resp.json();
      expect(body.client_id).toBeTruthy();
      expect(body.client_secret).toBeTruthy();
      expect(body.redirect_uris).toContain("https://myapp.example.com/callback");
    });

    test("Duplicate client name registration is rejected with 409", async () => {
      const name = `dup-client-${Date.now()}`;

      const first = await apiCtx.post(`${BASE_URL}/oauth/clients`, {
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          "Content-Type": "application/json",
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        data: {
          name,
          redirect_uris: ["https://first.example.com/callback"],
          grant_types: ["authorization_code"],
          scopes: ["openid"],
        },
      });
      expect(first.status()).toBe(201);

      const second = await apiCtx.post(`${BASE_URL}/oauth/clients`, {
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          "Content-Type": "application/json",
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        data: {
          name,
          redirect_uris: ["https://second.example.com/callback"],
          grant_types: ["authorization_code"],
          scopes: ["openid"],
        },
      });
      expect(second.status()).toBe(409);
    });

    test("Wildcard redirect_uri is rejected with 422", async () => {
      const resp = await apiCtx.post(`${BASE_URL}/oauth/clients`, {
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          "Content-Type": "application/json",
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        data: {
          name: `wildcard-test-${Date.now()}`,
          redirect_uris: ["https://app.example.com/*"],
          grant_types: ["authorization_code"],
          scopes: ["openid"],
        },
      });

      expect(resp.status()).toBe(422);
      const body = await resp.json();
      expect(JSON.stringify(body)).toContain("redirect_uri");
    });
  });

  // -------------------------------------------------------------------------
  // Authorization Code + PKCE
  // -------------------------------------------------------------------------

  test.describe("Authorization Code + PKCE (S256)", () => {
    test("Full PKCE S256 flow returns valid JWT with correct claims", async () => {
      const userTokens = await registerAndVerifyUser(
        apiCtx,
        TENANT_ALPHA,
        `pkce-user-${Date.now()}@example.com`,
        "PKCEUserPass123!"
      );

      const client = await registerOAuthClient(
        apiCtx,
        TENANT_ALPHA,
        adminTokens.access_token
      );

      const verifier = generateCodeVerifier();
      const challenge = generateCodeChallenge(verifier);
      const state = generateState();

      // Step 1: Authorization request
      const authorizeResp = await apiCtx.get(`${BASE_URL}/oauth/authorize`, {
        params: {
          response_type: "code",
          client_id: client.client_id,
          redirect_uri: client.redirect_uris[0],
          scope: "openid profile",
          state,
          code_challenge: challenge,
          code_challenge_method: "S256",
        },
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          Authorization: `Bearer ${userTokens.access_token}`,
        },
        maxRedirects: 0,
      });

      // Should redirect to redirect_uri with code
      expect([302, 303]).toContain(authorizeResp.status());
      const locationHeader = authorizeResp.headers()["location"];
      expect(locationHeader).toBeTruthy();

      const redirectURL = new URL(locationHeader);
      const code = redirectURL.searchParams.get("code");
      const returnedState = redirectURL.searchParams.get("state");

      expect(code).toBeTruthy();
      expect(returnedState).toBe(state);

      // Step 2: Token exchange with code_verifier
      const tokenResp = await apiCtx.post(`${BASE_URL}/oauth/token`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
        },
        form: {
          grant_type: "authorization_code",
          code: code!,
          redirect_uri: client.redirect_uris[0],
          client_id: client.client_id,
          client_secret: client.client_secret,
          code_verifier: verifier,
        },
      });

      expect(tokenResp.status()).toBe(200);
      const tokenBody = await tokenResp.json();
      expect(tokenBody.access_token).toBeTruthy();
      expect(tokenBody.token_type).toBe("Bearer");

      // Validate access token JWT claims
      const claims = decodeJWTClaims(tokenBody.access_token);
      expect(claims.tenant_id).toBeTruthy();
      expect(claims.sub).toBeTruthy();
      expect(claims.iss).toBeTruthy();
      expect(claims.exp).toBeGreaterThan(Math.floor(Date.now() / 1000));
      expect(claims.scope).toContain("openid");
    });

    test("PKCE plain method is rejected with 400", async () => {
      const client = await registerOAuthClient(
        apiCtx,
        TENANT_ALPHA,
        adminTokens.access_token
      );

      const verifier = generateCodeVerifier();
      const state = generateState();

      const resp = await apiCtx.get(`${BASE_URL}/oauth/authorize`, {
        params: {
          response_type: "code",
          client_id: client.client_id,
          redirect_uri: client.redirect_uris[0],
          scope: "openid",
          state,
          code_challenge: verifier, // plain: challenge == verifier
          code_challenge_method: "plain",
        },
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        maxRedirects: 0,
      });

      expect(resp.status()).toBe(400);
      const body = await resp.json();
      expect(JSON.stringify(body)).toMatch(/plain|code_challenge_method/i);
    });

    test("Authorization code reuse returns 400 and revokes previously issued tokens", async () => {
      const userTokens = await registerAndVerifyUser(
        apiCtx,
        TENANT_ALPHA,
        `code-reuse-${Date.now()}@example.com`,
        "CodeReusePass123!"
      );

      const client = await registerOAuthClient(
        apiCtx,
        TENANT_ALPHA,
        adminTokens.access_token
      );

      const verifier = generateCodeVerifier();
      const challenge = generateCodeChallenge(verifier);

      const authorizeResp = await apiCtx.get(`${BASE_URL}/oauth/authorize`, {
        params: {
          response_type: "code",
          client_id: client.client_id,
          redirect_uri: client.redirect_uris[0],
          scope: "openid",
          state: generateState(),
          code_challenge: challenge,
          code_challenge_method: "S256",
        },
        headers: {
          "X-Tenant-ID": TENANT_ALPHA,
          Authorization: `Bearer ${userTokens.access_token}`,
        },
        maxRedirects: 0,
      });
      expect([302, 303]).toContain(authorizeResp.status());

      const location = new URL(authorizeResp.headers()["location"]);
      const code = location.searchParams.get("code")!;
      expect(code).toBeTruthy();

      const exchangePayload = {
        grant_type: "authorization_code",
        code,
        redirect_uri: client.redirect_uris[0],
        client_id: client.client_id,
        client_secret: client.client_secret,
        code_verifier: verifier,
      };

      // First exchange — must succeed
      const first = await apiCtx.post(`${BASE_URL}/oauth/token`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
        },
        form: exchangePayload,
      });
      expect(first.status()).toBe(200);
      const firstTokens = await first.json();
      expect(firstTokens.access_token).toBeTruthy();

      // Second exchange with same code — must fail
      const second = await apiCtx.post(`${BASE_URL}/oauth/token`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
        },
        form: exchangePayload,
      });
      expect(second.status()).toBe(400);

      // Verify the previously issued access token is now revoked
      const introspectResp = await apiCtx.post(`${BASE_URL}/oauth/introspect`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        form: { token: firstTokens.access_token },
      });
      expect(introspectResp.status()).toBe(200);
      const introspectBody = await introspectResp.json();
      expect(introspectBody.active).toBe(false);
    });
  });

  // -------------------------------------------------------------------------
  // Client Credentials
  // -------------------------------------------------------------------------

  test.describe("Client Credentials Grant", () => {
    test("client_credentials grant returns token without sub claim", async () => {
      const client = await registerOAuthClient(
        apiCtx,
        TENANT_ALPHA,
        adminTokens.access_token,
        { grant_types: ["client_credentials"], scopes: ["api:read", "api:write"] }
      );

      const tokenResp = await apiCtx.post(`${BASE_URL}/oauth/token`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
        },
        form: {
          grant_type: "client_credentials",
          client_id: client.client_id,
          client_secret: client.client_secret,
          scope: "api:read",
        },
      });

      expect(tokenResp.status()).toBe(200);
      const tokenBody = await tokenResp.json();
      expect(tokenBody.access_token).toBeTruthy();

      const claims = decodeJWTClaims(tokenBody.access_token);
      expect(claims.sub).toBeUndefined();
      expect(claims.tenant_id).toBeTruthy();
      expect(claims.scope).toContain("api:read");
      expect(claims.exp).toBeGreaterThan(Math.floor(Date.now() / 1000));
    });
  });

  // -------------------------------------------------------------------------
  // JWKS Endpoint
  // -------------------------------------------------------------------------

  test.describe("JWKS Endpoint", () => {
    test("GET /.well-known/jwks.json returns valid RS256 JWK set", async () => {
      const resp = await apiCtx.get(
        `${BASE_URL}/.well-known/jwks.json`,
        { headers: { "X-Tenant-ID": TENANT_ALPHA } }
      );

      expect(resp.status()).toBe(200);
      const body = await resp.json();

      expect(body.keys).toBeDefined();
      expect(Array.isArray(body.keys)).toBe(true);
      expect(body.keys.length).toBeGreaterThan(0);

      const key = body.keys[0];
      expect(key.kty).toBe("RSA");
      expect(key.use).toBe("sig");
      expect(key.alg).toBe("RS256");
      expect(key.n).toBeTruthy();
      expect(key.e).toBeTruthy();
      expect(key.kid).toBeTruthy();

      // Must not expose private key material
      expect(key.d).toBeUndefined();
      expect(key.p).toBeUndefined();
      expect(key.q).toBeUndefined();
    });
  });

  // -------------------------------------------------------------------------
  // Token Introspection
  // -------------------------------------------------------------------------

  test.describe("Token Introspection", () => {
    test("Valid token returns active=true with claims", async () => {
      const client = await registerOAuthClient(
        apiCtx,
        TENANT_ALPHA,
        adminTokens.access_token,
        { grant_types: ["client_credentials"] }
      );

      const tokenResp = await apiCtx.post(`${BASE_URL}/oauth/token`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
        },
        form: {
          grant_type: "client_credentials",
          client_id: client.client_id,
          client_secret: client.client_secret,
          scope: "api:read",
        },
      });
      expect(tokenResp.status()).toBe(200);
      const { access_token } = await tokenResp.json();

      const introspectResp = await apiCtx.post(`${BASE_URL}/oauth/introspect`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        form: { token: access_token },
      });

      expect(introspectResp.status()).toBe(200);
      const body = await introspectResp.json();
      expect(body.active).toBe(true);
      expect(body.tenant_id).toBeTruthy();
      expect(body.exp).toBeGreaterThan(Math.floor(Date.now() / 1000));
    });

    test("Expired or revoked token returns active=false", async () => {
      // Use a syntactically valid but fabricated token that the server won't find
      const fakeToken =
        "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9." +
        Buffer.from(
          JSON.stringify({
            sub: "fake-user",
            tenant_id: TENANT_ALPHA,
            exp: Math.floor(Date.now() / 1000) - 3600,
            iat: Math.floor(Date.now() / 1000) - 7200,
          })
        ).toString("base64url") +
        ".invalidsignature";

      const introspectResp = await apiCtx.post(`${BASE_URL}/oauth/introspect`, {
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          "X-Tenant-ID": TENANT_ALPHA,
          Authorization: `Bearer ${adminTokens.access_token}`,
        },
        form: { token: fakeToken },
      });

      expect(introspectResp.status()).toBe(200);
      const body = await introspectResp.json();
      expect(body.active).toBe(false);
    });
  });
});
```

---

## 4e. Security Tests

File: `test/security/auth_security_test.go`

```go
package security_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yourorg/auth/internal/bootstrap"
	"github.com/yourorg/auth/internal/migrate"
)

// ---------------------------------------------------------------------------
// Package-level state
// ---------------------------------------------------------------------------

var (
	testDB     *pgxpool.Pool
	testRouter *gin.Engine
)

// ---------------------------------------------------------------------------
// TestMain
// ---------------------------------------------------------------------------

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/auth_test_security?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error
	testDB, err = pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to security test database: %v\n", err)
		os.Exit(1)
	}
	defer testDB.Close()

	if err = migrate.RunGlobalSchema(ctx, testDB); err != nil {
		fmt.Fprintf(os.Stderr, "global schema migration failed: %v\n", err)
		os.Exit(1)
	}

	testRouter = bootstrap.NewRouter(testDB)

	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type securityTenant struct {
	ID   string
	Slug string
}

func setupSecurityTenant(t *testing.T, name string) securityTenant {
	t.Helper()
	slug := fmt.Sprintf("sec-%s-%d", name, time.Now().UnixNano())
	tenantID := fmt.Sprintf("tenant-%s", slug)

	ctx := context.Background()
	_, err := testDB.Exec(ctx, `
		INSERT INTO global.tenants (id, slug, plan_id, status)
		VALUES ($1, $2, 'plan-free', 'active')
	`, tenantID, slug)
	require.NoError(t, err)

	_, err = testDB.Exec(ctx, fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "tenant_%s"`, slug))
	require.NoError(t, err)
	require.NoError(t, migrate.RunTenantSchema(ctx, testDB, slug))

	t.Cleanup(func() {
		testDB.Exec(ctx, fmt.Sprintf(`DROP SCHEMA IF EXISTS "tenant_%s" CASCADE`, slug))
		testDB.Exec(ctx, `DELETE FROM global.tenants WHERE id = $1`, tenantID)
	})

	return securityTenant{ID: tenantID, Slug: slug}
}

func doReq(t *testing.T, method, path string, body interface{}, tenantSlug, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var br *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(t, err)
		br = bytes.NewReader(b)
	} else {
		br = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, br)
	req.Header.Set("Content-Type", "application/json")
	if tenantSlug != "" {
		req.Header.Set("X-Tenant-ID", tenantSlug)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)
	return w
}

func registerAndActivateUser(t *testing.T, slug, email, password string) string {
	t.Helper()
	w := doReq(t, http.MethodPost, "/auth/register", map[string]string{
		"email": email, "password": password,
	}, slug, "")
	require.Equal(t, http.StatusCreated, w.Code, "register: %s", w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	userID := resp["user_id"].(string)

	_, err := testDB.Exec(context.Background(), fmt.Sprintf(`
		UPDATE "tenant_%s".users SET status = 'active' WHERE id = $1
	`, slug), userID)
	require.NoError(t, err)
	return userID
}

func loginGetTokens(t *testing.T, slug, email, password string) (accessToken, refreshToken string) {
	t.Helper()
	w := doReq(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": password,
	}, slug, "")
	require.Equal(t, http.StatusOK, w.Code, "login: %s", w.Body.String())

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp["access_token"].(string), resp["refresh_token"].(string)
}

func stddevMillis(durations []time.Duration) float64 {
	if len(durations) == 0 {
		return 0
	}
	var sum float64
	for _, d := range durations {
		sum += float64(d.Milliseconds())
	}
	mean := sum / float64(len(durations))
	var variance float64
	for _, d := range durations {
		diff := float64(d.Milliseconds()) - mean
		variance += diff * diff
	}
	variance /= float64(len(durations))
	return math.Sqrt(variance)
}

// ---------------------------------------------------------------------------
// Security Tests
// ---------------------------------------------------------------------------

func TestBruteForce_AccountLockout(t *testing.T) {
	tenant := setupSecurityTenant(t, "lockout")
	email := "lockout@example.com"
	password := "CorrectPass123!"
	userID := registerAndActivateUser(t, tenant.Slug, email, password)

	const maxAttempts = 5

	// Submit maxAttempts wrong-password requests
	for i := 0; i < maxAttempts; i++ {
		w := doReq(t, http.MethodPost, "/auth/login", map[string]string{
			"email": email, "password": "WrongPass999!",
		}, tenant.Slug, "")
		assert.Equal(t, http.StatusUnauthorized, w.Code, "attempt %d must return 401", i+1)
	}

	// The (maxAttempts+1)th attempt must return locked — not wrong password
	w := doReq(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": "WrongPass999!",
	}, tenant.Slug, "")
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	// Verify account is now locked in the database
	var status string
	err := testDB.QueryRow(context.Background(), fmt.Sprintf(`
		SELECT status FROM "tenant_%s".users WHERE id = $1
	`, tenant.Slug), userID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "locked", status, "user status must be locked after %d failed attempts", maxAttempts)

	// Correct password on locked account must also return locked error code
	wCorrect := doReq(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": password,
	}, tenant.Slug, "")
	assert.Equal(t, http.StatusUnauthorized, wCorrect.Code)

	var respBody map[string]interface{}
	require.NoError(t, json.Unmarshal(wCorrect.Body.Bytes(), &respBody))
	errorCode, _ := respBody["code"].(string)
	assert.Equal(t, "ACCOUNT_LOCKED", errorCode, "locked account must return ACCOUNT_LOCKED code")

	// Verify ACCOUNT_LOCKED audit entry exists
	var auditEvent string
	err = testDB.QueryRow(context.Background(), fmt.Sprintf(`
		SELECT event_type FROM "tenant_%s".audit_log
		WHERE user_id = $1 AND event_type = 'ACCOUNT_LOCKED'
		ORDER BY created_at DESC LIMIT 1
	`, tenant.Slug), userID).Scan(&auditEvent)
	require.NoError(t, err, "ACCOUNT_LOCKED audit entry must exist")
	assert.Equal(t, "ACCOUNT_LOCKED", auditEvent)
}

func TestBruteForce_ResponseConsistency(t *testing.T) {
	tenant := setupSecurityTenant(t, "consistency")
	email := "victim@example.com"
	password := "CorrectPass123!"
	registerAndActivateUser(t, tenant.Slug, email, password)

	// Lock the account
	for i := 0; i < 5; i++ {
		doReq(t, http.MethodPost, "/auth/login", map[string]string{
			"email": email, "password": "WrongPass!",
		}, tenant.Slug, "")
	}

	// Wrong-password response
	wWrong := doReq(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": "StillWrong!",
	}, tenant.Slug, "")

	// Locked-account response (correct password but account locked)
	wLocked := doReq(t, http.MethodPost, "/auth/login", map[string]string{
		"email": email, "password": password,
	}, tenant.Slug, "")

	// Both must be 401
	assert.Equal(t, http.StatusUnauthorized, wWrong.Code)
	assert.Equal(t, http.StatusUnauthorized, wLocked.Code)

	var wrongResp, lockedResp map[string]interface{}
	require.NoError(t, json.Unmarshal(wWrong.Body.Bytes(), &wrongResp))
	require.NoError(t, json.Unmarshal(wLocked.Body.Bytes(), &lockedResp))

	// The top-level "message" field must be identical — no information leak
	wrongMsg, _ := wrongResp["message"].(string)
	lockedMsg, _ := lockedResp["message"].(string)
	assert.Equal(t, wrongMsg, lockedMsg,
		"wrong-password and locked-account message must be identical to prevent enumeration")

	// Neither body should contain words that reveal the account state
	for _, body := range []string{wWrong.Body.String(), wLocked.Body.String()} {
		assert.NotContains(t, strings.ToLower(body), "locked",
			"response body must not reveal lock status")
	}
}

func TestRefreshToken_FamilyRevocation(t *testing.T) {
	tenant := setupSecurityTenant(t, "family-revoke")
	email := "family@example.com"
	password := "FamilyPass123!"
	userID := registerAndActivateUser(t, tenant.Slug, email, password)

	_, refreshToken := loginGetTokens(t, tenant.Slug, email, password)

	// Rotate the token once to produce a new token in the same family
	w := doReq(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, tenant.Slug, "")
	require.Equal(t, http.StatusOK, w.Code, "initial refresh must succeed")

	var newTokens map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &newTokens))
	newRefreshToken := newTokens["refresh_token"].(string)

	// Now reuse the old (rotated) token — must trigger family revocation
	w2 := doReq(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, tenant.Slug, "")
	assert.Equal(t, http.StatusUnauthorized, w2.Code, "reused token must be rejected")

	// The new token issued in the same family must also be revoked
	w3 := doReq(t, http.MethodPost, "/auth/refresh", map[string]string{
		"refresh_token": newRefreshToken,
	}, tenant.Slug, "")
	assert.Equal(t, http.StatusUnauthorized, w3.Code,
		"sibling token in compromised family must be revoked")

	// SUSPICIOUS_TOKEN_REUSE must be logged with correct tenant_id and user_id
	var loggedTenantID, loggedUserID string
	err := testDB.QueryRow(context.Background(), fmt.Sprintf(`
		SELECT tenant_id, user_id FROM "tenant_%s".audit_log
		WHERE event_type = 'SUSPICIOUS_TOKEN_REUSE' AND user_id = $1
		ORDER BY created_at DESC LIMIT 1
	`, tenant.Slug), userID).Scan(&loggedTenantID, &loggedUserID)
	require.NoError(t, err, "SUSPICIOUS_TOKEN_REUSE audit entry must exist")
	assert.Equal(t, tenant.ID, loggedTenantID)
	assert.Equal(t, userID, loggedUserID)
}

func TestPasswordReset_TimingConsistency(t *testing.T) {
	tenant := setupSecurityTenant(t, "timing")
	knownEmail := "known-timing@example.com"
	registerAndActivateUser(t, tenant.Slug, knownEmail, "TimingPass123!")

	const iterations = 20
	durations := make([]time.Duration, 0, iterations)

	for i := 0; i < iterations; i++ {
		var email string
		if i%2 == 0 {
			email = knownEmail
		} else {
			email = fmt.Sprintf("ghost%d@nowhere.example.com", i)
		}

		start := time.Now()
		w := doReq(t, http.MethodPost, "/auth/forgot-password", map[string]string{
			"email": email,
		}, tenant.Slug, "")
		elapsed := time.Since(start)
		durations = append(durations, elapsed)

		assert.Equal(t, http.StatusAccepted, w.Code,
			"forgot-password must always return 202 (email: %s)", email)
	}

	sd := stddevMillis(durations)
	assert.Less(t, sd, float64(50),
		"standard deviation of forgot-password response times must be <50ms (got %.2fms)", sd)
}

func TestJWT_TamperRejected(t *testing.T) {
	tenant := setupSecurityTenant(t, "jwt-tamper")
	email := "tamper@example.com"
	password := "TamperPass123!"
	registerAndActivateUser(t, tenant.Slug, email, password)
	accessToken, _ := loginGetTokens(t, tenant.Slug, email, password)

	// Decode the JWT payload
	parts := strings.Split(accessToken, ".")
	require.Len(t, parts, 3, "access token must be a 3-part JWT")

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]interface{}
	require.NoError(t, json.Unmarshal(payloadBytes, &claims))

	// Tamper: elevate roles to ["admin", "super_admin"]
	claims["roles"] = []string{"admin", "super_admin"}

	tamperedPayload, err := json.Marshal(claims)
	require.NoError(t, err)

	tamperedToken := parts[0] + "." +
		base64.RawURLEncoding.EncodeToString(tamperedPayload) + "." +
		parts[2] // Keep original signature — must fail verification

	// Use tampered token to call a protected endpoint
	w := doReq(t, http.MethodGet, "/admin/users", tenant.Slug, tamperedToken, nil)
	// Must reject — signature no longer matches the tampered payload
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"tampered JWT must be rejected (signature mismatch), got: %s", w.Body.String())
}

func TestJWT_ExpiredRejected(t *testing.T) {
	tenant := setupSecurityTenant(t, "jwt-expired")
	email := "expired@example.com"
	password := "ExpiredPass123!"
	registerAndActivateUser(t, tenant.Slug, email, password)
	accessToken, _ := loginGetTokens(t, tenant.Slug, email, password)

	// Decode the JWT and set exp to the past
	parts := strings.Split(accessToken, ".")
	require.Len(t, parts, 3)

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)

	var claims map[string]interface{}
	require.NoError(t, json.Unmarshal(payloadBytes, &claims))

	// Set expiry to 1 hour ago
	claims["exp"] = float64(time.Now().Add(-1 * time.Hour).Unix())

	tamperedPayload, err := json.Marshal(claims)
	require.NoError(t, err)

	expiredToken := parts[0] + "." +
		base64.RawURLEncoding.EncodeToString(tamperedPayload) + "." +
		parts[2]

	w := doReq(t, http.MethodGet, "/auth/sessions", tenant.Slug, expiredToken, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code,
		"expired JWT must be rejected, got: %s", w.Body.String())
}

func TestSQLInjection_EmailField(t *testing.T) {
	tenant := setupSecurityTenant(t, "sqli")

	injectionPayloads := []string{
		"'; DROP TABLE users; --",
		"' OR '1'='1",
		"' UNION SELECT null, null, null--",
		`admin'--`,
		`" OR ""="`,
		"1; SELECT * FROM users WHERE 't'='t",
	}

	for _, payload := range injectionPayloads {
		payload := payload
		t.Run(fmt.Sprintf("payload_%s", sanitizeTestName(payload)), func(t *testing.T) {
			w := doReq(t, http.MethodPost, "/auth/login", map[string]string{
				"email":    payload,
				"password": "SomePass123!",
			}, tenant.Slug, "")

			// Must return 422 (validation) or 401 (invalid credentials) — not 500
			assert.Contains(t, []int{http.StatusUnprocessableEntity, http.StatusUnauthorized}, w.Code,
				"SQL injection payload must not cause 5xx: email=%q status=%d body=%s",
				payload, w.Code, w.Body.String())

			// Verify the users table still exists and is intact
			var count int
			err := testDB.QueryRow(context.Background(), fmt.Sprintf(`
				SELECT COUNT(*) FROM "tenant_%s".users
			`, tenant.Slug)).Scan(&count)
			require.NoError(t, err, "users table must still exist after SQL injection attempt with payload: %s", payload)
		})
	}
}

func TestHTTPHeaders_Security(t *testing.T) {
	tenant := setupSecurityTenant(t, "headers")

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/health"},
		{http.MethodPost, "/auth/login"},
		{http.MethodPost, "/auth/register"},
	}

	requiredHeaders := map[string]string{
		"Strict-Transport-Security": "",          // must be present, any value
		"X-Frame-Options":           "DENY",
		"X-Content-Type-Options":    "nosniff",
	}

	for _, ep := range endpoints {
		ep := ep
		t.Run(fmt.Sprintf("%s_%s", ep.method, strings.ReplaceAll(ep.path, "/", "_")), func(t *testing.T) {
			w := doReq(t, ep.method, ep.path, nil, tenant.Slug, "")

			for header, expectedValue := range requiredHeaders {
				actual := w.Header().Get(header)
				require.NotEmpty(t, actual,
					"security header %q must be present on %s %s", header, ep.method, ep.path)
				if expectedValue != "" {
					assert.Equal(t, expectedValue, actual,
						"header %q must have value %q on %s %s", header, expectedValue, ep.method, ep.path)
				}
			}

			// Verify HSTS max-age is at least 1 year
			hsts := w.Header().Get("Strict-Transport-Security")
			assert.Contains(t, hsts, "max-age=",
				"HSTS header must contain max-age directive")
			assert.NotContains(t, strings.ToLower(hsts), "max-age=0",
				"HSTS max-age must not be 0")
		})
	}
}

func TestCrossTenantJWT_Rejected(t *testing.T) {
	tenantAlpha := setupSecurityTenant(t, "xtenant-alpha")
	tenantBeta := setupSecurityTenant(t, "xtenant-beta")

	alphaEmail := "alpha-user@example.com"
	alphaPassword := "AlphaPass123!"
	registerAndActivateUser(t, tenantAlpha.Slug, alphaEmail, alphaPassword)
	alphaToken, _ := loginGetTokens(t, tenantAlpha.Slug, alphaEmail, alphaPassword)

	// Use tenant-alpha's valid JWT against tenant-beta's endpoint
	w := doReq(t, http.MethodGet, "/auth/sessions",
		tenantBeta.Slug, alphaToken, nil)

	assert.Equal(t, http.StatusForbidden, w.Code,
		"valid JWT for tenant-alpha must be rejected with 403 on tenant-beta endpoint, got: %s",
		w.Body.String())

	// Error body must not leak which tenant the token belongs to
	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	respStr := string(w.Body.Bytes())
	assert.NotContains(t, respStr, tenantAlpha.ID,
		"403 response must not reveal the token's actual tenant")
	assert.NotContains(t, respStr, tenantAlpha.Slug,
		"403 response must not reveal the token's actual tenant slug")
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func sanitizeTestName(s string) string {
	replacer := strings.NewReplacer(
		" ", "_", "'", "", ";", "", "-", "_",
		"*", "", "=", "_", `"`, "", "/", "_",
	)
	return replacer.Replace(s)
}
```

---

# Section 5: Quality Gates Per Sprint

---

### Sprint 0 — Architecture Spike

**Entry Criteria**
- Project repository initialized with Go module and directory structure
- PostgreSQL and Redis instances available in local Docker Compose
- HashiCorp Vault dev instance reachable

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G0-01 | All 12 ADRs reviewed, signed off, and committed to `docs/adr/` | Yes |
| G0-02 | ERD reviewed by SA, covering `global` schema + at least one `tenant_*` schema | Yes |
| G0-03 | Migration POC executes `CREATE SCHEMA tenant_test` + all tables without error | Yes |
| G0-04 | Docker Compose `docker compose up` starts all services without manual intervention | Yes |
| G0-05 | `go build ./...` produces zero errors on CI | Yes |
| G0-06 | `golangci-lint run` exits 0 with default ruleset on all existing Go files | Yes |
| G0-07 | Vault secret read/write POC script succeeds in CI environment | Yes |
| G0-08 | RS256 key pair generation + JWT sign/verify POC passes as a Go test | Yes |
| G0-09 | `ory/fosite` dependency imported and compiles cleanly | Yes |
| G0-10 | No open Critical or High defects | Yes |

**Defect Threshold before Sprint 1 begins:** Critical = 0, High = 0

---

### Sprint 1 — Core Auth (Register, Login, Email Verify, Password Reset)

**Entry Criteria**
- Sprint 0 all exit gates passed
- `global` schema migration applied to staging DB
- RS256 key pair stored in Vault, accessible to application

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G1-01 | Unit test coverage on `internal/service/auth_service.go` ≥ 80% (measured by `go test -cover`) | Yes |
| G1-02 | All `TestAuthService_Login` table-driven cases pass | Yes |
| G1-03 | All `TestAuthService_Register` table-driven cases pass | Yes |
| G1-04 | `TestRegister_Success` integration test passes: status 201, `status=unverified` in DB | Yes |
| G1-05 | `TestRegister_DuplicateEmail` returns 409 with no account-status disclosure in body | Yes |
| G1-06 | `TestLogin_Success` returns 200, JWT contains `tenant_id` + `roles[]` claims, `exp - iat ≈ 900s` | Yes |
| G1-07 | `TestLogin_WrongPassword` returns 401 with ambiguous error message | Yes |
| G1-08 | Email verification token stored hashed in DB; plaintext never persisted | Yes |
| G1-09 | Password reset token expires in ≤ 1 hour (verified by integration test) | Yes |
| G1-10 | `TestForgotPassword_Timing` timing delta between known/unknown email < 100ms | Yes |
| G1-11 | All integration tests isolated: each test uses its own tenant schema, cleaned up in `t.Cleanup` | Yes |
| G1-12 | `go vet ./...` and `golangci-lint run` exit 0 | Yes |
| G1-13 | No open Critical defects; High ≤ 2 with mitigation plan | Yes |

**Defect Threshold before Sprint 2 begins:** Critical = 0, High ≤ 2

---

### Sprint 2 — Sessions, Rate Limiting, Audit Log

**Entry Criteria**
- Sprint 1 all exit gates passed
- Refresh token schema migrated in all tenant schemas
- Redis connected and reachable from the Go service

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G2-01 | `TestAuthService_RefreshToken` — success with rotation case passes | Yes |
| G2-02 | `TestAuthService_RefreshToken` — expired token case passes | Yes |
| G2-03 | `TestAuthService_RefreshToken` — reused rotated token triggers family revocation | Yes |
| G2-04 | `TestRefreshToken_Rotation` integration test: old token rejected after rotation | Yes |
| G2-05 | `TestRefreshToken_ReuseDetection`: reused token returns 401, both family tokens revoked, `SUSPICIOUS_TOKEN_REUSE` in audit log | Yes |
| G2-06 | `TestAuthService_Logout` — single device and all-devices cases pass | Yes |
| G2-07 | `TestLogout_AllDevices` integration test: all three sessions rejected after logout/all | Yes |
| G2-08 | `TestRefreshToken_FamilyRevocation` security test passes: correct `tenant_id` + `user_id` in audit entry | Yes |
| G2-09 | Rate limiter triggers after configured threshold; returns 429 with `Retry-After` header | Yes |
| G2-10 | Audit log entries include: `event_type`, `tenant_id`, `user_id`, `ip_address`, `user_agent`, `created_at` | Yes |
| G2-11 | Absolute session expiry enforced at 24h (configurable per tenant via tenant config table) | Yes |
| G2-12 | Refresh tokens stored as SHA-256 hash in DB; plaintext never persisted | Yes |
| G2-13 | Unit coverage on session service ≥ 80% | Yes |
| G2-14 | `TestPasswordReset_TimingConsistency` stddev < 50ms across 20 requests | Yes |
| G2-15 | No open Critical defects; High ≤ 2 | Yes |

**Defect Threshold before Sprint 3 begins:** Critical = 0, High ≤ 2

---

### Sprint 3 — Multi-Tenancy + Isolation Tests

**Entry Criteria**
- Sprint 2 all exit gates passed
- `tenant_*` schema migration automation complete (triggered on tenant creation)
- CI pipeline runs isolation suite as a separate blocking job

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G3-01 | **ALL 12 isolation test cases in `TestTenantIsolation` pass with zero failures** | **Yes — hard block** |
| G3-02 | TC-ISO-01: Alpha JWT rejected on beta `/admin/users` → 403 | Yes |
| G3-03 | TC-ISO-02: Alpha JWT rejected on beta `/admin/audit-log` → 403 | Yes |
| G3-04 | TC-ISO-03: Alpha admin cannot list beta users | Yes |
| G3-05 | TC-ISO-04: Alpha admin cannot assign roles to beta users; DB verified unchanged | Yes |
| G3-06 | TC-ISO-05: Alpha JWT rejected on beta OAuth clients endpoint | Yes |
| G3-07 | TC-ISO-06: Alpha user cannot see beta sessions | Yes |
| G3-08 | TC-ISO-07: Valid JWT with forged `tenant_id` (alpha token on beta endpoint) → 403 | Yes |
| G3-09 | TC-ISO-08: `assertNoCrossTenantData` finds zero beta identifiers in any alpha response | Yes |
| G3-10 | TC-ISO-09: All cross-tenant error messages contain zero tenant-beta data | Yes |
| G3-11 | TC-ISO-10: Cross-tenant password reset leaks no information; responses identical in shape | Yes |
| G3-12 | TC-ISO-11: Alpha OAuth client rejected by beta authorization endpoint | Yes |
| G3-13 | TC-ISO-12: Alpha audit log DB query returns zero rows with `tenant_id = beta` | Yes |
| G3-14 | Schema-level DB query confirms no cross-schema foreign key violations | Yes |
| G3-15 | `TestCrossTenantJWT_Rejected` security test passes: 403, no tenant-alpha data in body | Yes |
| G3-16 | Tenant schema creation + migration executes in < 2 seconds (measured in CI) | No |
| G3-17 | No open Critical defects; High = 0 | Yes |

**Defect Threshold before Sprint 4 begins:** Critical = 0, High = 0 (isolation is non-negotiable)

---

### Sprint 4 — RBAC, JWT Claims, Admin API

**Entry Criteria**
- Sprint 3 all exit gates passed (including all 12 isolation tests)
- Role schema defined in tenant migrations
- Admin API routes registered in Gin router

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G4-01 | JWT `roles[]` claim reflects assigned roles; verified by `TestLogin_Success` | Yes |
| G4-02 | Role assignment persisted in DB and reflected in next login's JWT | Yes |
| G4-03 | RBAC middleware rejects requests from users missing required role → 403 | Yes |
| G4-04 | `TestJWT_TamperRejected`: modified `roles` claim with original signature → 401 | Yes |
| G4-05 | `TestJWT_ExpiredRejected`: `exp` in past → 401 | Yes |
| G4-06 | Admin endpoints require `admin` role; regular user receives 403 | Yes |
| G4-07 | Role assignment action logged in audit log with `actor_id` + `target_user_id` | Yes |
| G4-08 | TC-ISO-04 continues to pass (alpha admin cannot assign roles in beta) | Yes |
| G4-09 | Unit coverage on RBAC middleware ≥ 85% | Yes |
| G4-10 | Admin user-list endpoint returns paginated results; page size enforced ≤ 100 | No |
| G4-11 | No open Critical defects; High ≤ 1 | Yes |

**Defect Threshold before Sprint 5 begins:** Critical = 0, High ≤ 1

---

### Sprint 5 — OAuth 2.0 Authorization Server

**Entry Criteria**
- Sprint 4 all exit gates passed
- `ory/fosite` store implementations complete for authorization codes, access tokens, refresh tokens
- OAuth clients table migrated in tenant schemas

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G5-01 | Playwright: `POST /oauth/clients` → 201 with `client_id` + `client_secret` | Yes |
| G5-02 | Playwright: Duplicate client name → 409 | Yes |
| G5-03 | Playwright: Wildcard `redirect_uri` rejected → 422 | Yes |
| G5-04 | Playwright: Full PKCE S256 flow completes: authorize → code → token → valid JWT claims | Yes |
| G5-05 | Playwright: PKCE `plain` method rejected → 400 | Yes |
| G5-06 | Playwright: Authorization code reuse → 400, previously issued tokens revoked (introspect returns `active=false`) | Yes |
| G5-07 | Playwright: JWKS endpoint returns RSA key with `kty=RSA`, `use=sig`, `alg=RS256`; no private key fields present | Yes |
| G5-08 | Playwright: Token introspection returns `active=true` with claims for valid token | Yes |
| G5-09 | Playwright: Token introspection returns `active=false` for expired/revoked token | Yes |
| G5-10 | TC-ISO-05 and TC-ISO-11 continue to pass (OAuth clients isolated per tenant) | Yes |
| G5-11 | Authorization codes expire in ≤ 10 minutes (verified by integration test attempting late exchange) | Yes |
| G5-12 | No open Critical defects; High ≤ 1 | Yes |

**Defect Threshold before Sprint 6 begins:** Critical = 0, High ≤ 1

---

### Sprint 6 — OAuth M2M + Google Social Login

**Entry Criteria**
- Sprint 5 all exit gates passed
- Google OAuth app credentials available in Vault (test tenant)
- Client credentials grant type enabled in fosite config

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G6-01 | Playwright: `client_credentials` grant returns token without `sub` claim | Yes |
| G6-02 | Playwright: `client_credentials` token has `tenant_id` + `scope` claims | Yes |
| G6-03 | `client_credentials` token cannot access user-scoped endpoints (e.g., `/auth/sessions`) → 403 | Yes |
| G6-04 | Google social login callback creates user with `status=active` on first login | Yes |
| G6-05 | Google social login for existing email: ADR-006 enforced — password verification required before account linking | Yes |
| G6-06 | Social login account linking audit event logged: `SOCIAL_ACCOUNT_LINKED` with provider name | Yes |
| G6-07 | Social login with already-linked account performs login (no duplicate linking) | Yes |
| G6-08 | TC-ISO-11 continues to pass with M2M clients | Yes |
| G6-09 | No open Critical defects; High ≤ 1 | Yes |

**Defect Threshold before Sprint 7 begins:** Critical = 0, High ≤ 1

---

### Sprint 7 — MFA + User Profile

**Entry Criteria**
- Sprint 6 all exit gates passed
- TOTP library integrated (`pquerna/otp` or equivalent)
- MFA enforcement config available per tenant

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G7-01 | TOTP enrollment: `POST /auth/mfa/totp/enroll` returns QR code URI and backup codes | Yes |
| G7-02 | TOTP verification: valid TOTP code completes login; invalid code returns 401 | Yes |
| G7-03 | TOTP replay prevention: same TOTP code rejected within the same 30-second window | Yes |
| G7-04 | Backup code: each code usable exactly once; second use returns 401 | Yes |
| G7-05 | MFA-enforced tenant: login without MFA step returns 403 with `MFA_REQUIRED` code | Yes |
| G7-06 | TOTP secret stored encrypted in DB (not plaintext) | Yes |
| G7-07 | MFA enrollment and usage logged in audit log | Yes |
| G7-08 | User profile update (display name, avatar URL) persisted and returned in `/auth/me` | No |
| G7-09 | Unit coverage on MFA service ≥ 80% | Yes |
| G7-10 | No open Critical defects; High ≤ 2 | Yes |

**Defect Threshold before Sprint 8 begins:** Critical = 0, High ≤ 2

---

### Sprint 8 — Hardening + Compliance + Pentest Remediation

**Entry Criteria**
- Sprint 7 all exit gates passed
- External penetration test report received
- Performance test scripts written and baseline established

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G8-01 | `TestBruteForce_AccountLockout`: account locked after 5 failed attempts, `ACCOUNT_LOCKED` in audit log | Yes |
| G8-02 | `TestBruteForce_ResponseConsistency`: locked + wrong-password message fields identical | Yes |
| G8-03 | `TestHTTPHeaders_Security`: `Strict-Transport-Security`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff` present on all endpoints | Yes |
| G8-04 | `TestSQLInjection_EmailField`: all 6 injection payloads return 422 or 401; DB intact | Yes |
| G8-05 | Pentest findings: **0 Critical** unresolved, **0 High** unresolved | **Yes — hard block** |
| G8-06 | p95 login latency (POST `/auth/login`) < 300ms under 50 concurrent users (k6 or `hey` report) | Yes |
| G8-07 | p95 token refresh latency < 150ms under 50 concurrent users | Yes |
| G8-08 | p95 JWT validation middleware latency < 10ms (measured in unit benchmark) | Yes |
| G8-09 | Audit log retention policy enforced: entries older than 1 year moved to cold archive table (verified by DB query) | Yes |
| G8-10 | All 12 isolation tests still pass after hardening changes | Yes |
| G8-11 | `Content-Security-Policy` header present on any HTML-adjacent endpoints | No |
| G8-12 | Dependency vulnerability scan (`govulncheck ./...`) exits 0 | Yes |
| G8-13 | No open Critical defects; High = 0 | Yes |

**Defect Threshold before Sprint 9 begins:** Critical = 0, High = 0

---

### Sprint 9 — Launch Prep + UAT

**Entry Criteria**
- Sprint 8 all exit gates passed
- Staging environment deployed to Fly.io and stable for ≥ 48 hours
- UAT participants identified and access provisioned

**Exit Criteria**

| Gate | Criterion | Blocking? |
|------|-----------|-----------|
| G9-01 | 100% of Must-Have acceptance criteria from `prd.md` covered by at least one passing test | Yes |
| G9-02 | Staging smoke test suite (all critical paths) passes: register → verify → login → refresh → logout | Yes |
| G9-03 | Staging smoke test suite passes: full PKCE OAuth flow end-to-end | Yes |
| G9-04 | Staging smoke test suite passes: tenant isolation verified between two staging tenants | Yes |
| G9-05 | UAT sign-off: Product Owner has reviewed and approved all Sprint 1–7 Must-Have stories | Yes |
| G9-06 | Rollback plan documented and tested: `fly deploy --image <previous>` restores service in < 5 minutes | Yes |
| G9-07 | DB migration rollback script tested on staging without data loss | Yes |
| G9-08 | Runbook complete: on-call playbook covers top 5 incident types (account locked, token reuse, tenant creation failure, Vault unreachable, Redis down) | Yes |
| G9-09 | Monitoring alerts configured: error rate > 1%, p95 latency > 500ms, failed login rate > 20/min | Yes |
| G9-10 | Final `govulncheck ./...` clean on production build SHA | Yes |
| G9-11 | All 12 isolation tests pass on staging environment (not just local) | Yes |
| G9-12 | Go/No-Go meeting completed; sign-off obtained from PO, PM, and SA | **Yes — release gate** |
| G9-13 | No open Critical defects; High = 0; Medium ≤ 3 with accepted risk | Yes |

**Defect Threshold for release:** Critical = 0, High = 0, Medium ≤ 3 (accepted risk documented)

---

### Cumulative Quality Gate Summary

| Sprint | Theme | Unit Coverage Target | Integration Gate | Isolation Gate | Pentest Gate | Perf Gate |
|--------|-------|---------------------|-----------------|----------------|--------------|-----------|
| 0 | Architecture Spike | n/a | Build + lint pass | n/a | n/a | n/a |
| 1 | Core Auth | auth service ≥ 80% | Register + Login pass | n/a | n/a | Timing delta < 100ms |
| 2 | Sessions + Rate Limiting | session service ≥ 80% | Rotation + reuse pass | n/a | n/a | Timing stddev < 50ms |
| 3 | Multi-Tenancy | n/a | n/a | **All 12 TC-ISO pass** | n/a | n/a |
| 4 | RBAC + JWT | RBAC middleware ≥ 85% | Claims + tamper pass | TC-ISO-04 re-verified | n/a | n/a |
| 5 | OAuth 2.0 | n/a | Playwright PKCE suite | TC-ISO-05, 11 re-verified | n/a | n/a |
| 6 | OAuth M2M + Social | n/a | M2M + social pass | TC-ISO-11 re-verified | n/a | n/a |
| 7 | MFA + Profile | MFA service ≥ 80% | TOTP suite pass | n/a | n/a | n/a |
| 8 | Hardening + Pentest | n/a | Security suite pass | All 12 re-verified | **0 Critical/High** | p95 login < 300ms |
| 9 | Launch + UAT | n/a | Staging smoke suite | All 12 on staging | n/a | Alerts configured |

## Section 6: ISO 25010 Test Coverage Matrix

---

### 1. Functional Suitability — Priority: Critical

**Justification:** An authentication system that is functionally incorrect is not merely degraded — it is a security liability. Incorrect JWT claims grant unauthorized access. Missing token rotation enables replay attacks. Incomplete story coverage means user journeys fail in production. Every functional gap in an auth system directly maps to either a broken user experience or an exploitable vulnerability.

**Test Approach:**
- Map all 20 Must Have stories from the PRD to explicit test cases; no story ships without at least one passing integration test covering its acceptance criteria.
- Validate JWT payload claims (`sub`, `tenant_id`, `roles`, `exp`, `iat`, `jti`) against the contract defined in `solution-architecture.md` for every token issuance path.
- Verify token rotation by asserting that the previous refresh token is invalidated immediately upon use (token family revocation).
- Validate OAuth 2.0 flows against RFC 6749 (Authorization Code + Client Credentials), RFC 7636 (PKCE), and RFC 7662 (Token Introspection) using conformance test scripts.
- Verify audit events are emitted with correct `event_type`, `outcome`, `user_id`, `tenant_id`, `ip_address`, and `timestamp` for every auth action.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-FS-001 | Register with valid payload → 201, user record created in correct tenant schema |
| TC-FS-002 | Login with correct credentials → 200, JWT issued with correct `sub`, `tenant_id`, `roles`, `exp` (15 min) |
| TC-FS-003 | Login with incorrect password → 401, no token issued, LOGIN_FAILED audit event written |
| TC-FS-004 | Refresh token rotation: use refresh token → new pair issued, old refresh token rejected on reuse → 401 + family revoked |
| TC-FS-005 | Email verification: unverified user cannot log in; verified user can |
| TC-FS-006 | Password reset: token single-use, expires in 1 hour, password updated in correct tenant schema |
| TC-FS-007 | RBAC: user with role `viewer` calling admin endpoint → 403 |
| TC-FS-008 | OAuth Authorization Code + PKCE full flow → access token issued, introspection returns correct `active: true` |
| TC-FS-009 | Token introspection on expired token → `active: false` |
| TC-FS-010 | Audit log completeness: after each of 10 auth actions, assert exactly one corresponding audit record exists with all required fields |
| TC-FS-011 | OIDC discovery endpoint (`/.well-known/openid-configuration`) returns correct `issuer`, `jwks_uri`, `token_endpoint` |
| TC-FS-012 | JWKS endpoint returns valid RSA public key matching the key used to sign issued JWTs |
| TC-FS-013 | All 20 Must Have stories: one integration test per story asserting acceptance criteria from Gherkin scenarios |

**Pass Criteria:**
- 100% of Must Have stories have at least one passing automated test covering the primary acceptance criterion.
- All JWT claims match the documented contract with zero field deviations.
- Token rotation correctly invalidates prior tokens in 100% of test executions.
- All OAuth flows produce RFC-conformant responses validated against specification.
- Zero audit events missing for any covered auth action in the integration test suite.

**Tools:**
- `go test ./internal/...` — service and handler unit tests
- Custom Go integration test harness with real PostgreSQL + Redis (Docker Compose)
- `oauth2-conformance` or manual RFC-driven curl scripts for RFC 6749/7636/7662 validation
- Postman/Newman collection for contract assertion
- SQL assertions directly on tenant schema for audit log verification

---

### 2. Performance Efficiency — Priority: High

**Justification:** The system targets 1,000 concurrent sessions with strict latency NFRs. Authentication is a synchronous hot path — every API call in downstream services depends on token validation. Latency regression or connection pool exhaustion under load directly causes user-facing failures. Resource leaks (connection pool saturation, Redis memory growth) compound over time and cause availability incidents that violate the 99.9% uptime SLA.

**Test Approach:**
- Run load tests at p95 targeting the three NFR thresholds: login < 300ms, token issuance < 100ms, token introspection < 50ms.
- Simulate 1,000 concurrent sessions with realistic auth traffic patterns (70% token validation, 20% login, 10% refresh).
- Monitor PostgreSQL connection pool utilization (pgBouncer or `pg_stat_activity`) and Redis memory (`INFO memory`) during and after load tests.
- Run soak tests (4–8 hours) to detect goroutine leaks (`pprof` goroutine endpoint) and memory growth.
- Verify N+1 query absence in integration tests by asserting query count per request.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-PE-001 | Login endpoint: k6 ramp to 1,000 VUs, measure p95 latency — must be < 300ms |
| TC-PE-002 | Token issuance (OAuth token endpoint): p95 < 100ms at 1,000 concurrent |
| TC-PE-003 | Token introspection: p95 < 50ms at 1,000 concurrent — Redis cache hit path |
| TC-PE-004 | Spike test: ramp from 100 to 2,000 VUs in 30 seconds — system must return 429 or 503, not hang or crash |
| TC-PE-005 | PostgreSQL connection pool: at 1,000 concurrent, pool does not exceed configured max; requests queue rather than fail |
| TC-PE-006 | Redis memory: after 1,000-session load test, Redis `used_memory` is within expected bounds (no unbounded growth) |
| TC-PE-007 | Soak test: 500 VUs for 4 hours — goroutine count stable (no leak), heap growth < 10% from baseline |
| TC-PE-008 | Introspection cache hit: second call for same token within TTL returns from Redis in < 10ms |
| TC-PE-009 | N+1 query check: login flow executes ≤ 3 SQL queries (user lookup, schema set, audit insert) |

**Pass Criteria:**
- Login p95 < 300ms at 1,000 concurrent VUs.
- Token issuance p95 < 100ms at 1,000 concurrent VUs.
- Token introspection p95 < 50ms at 1,000 concurrent VUs.
- Zero connection pool exhaustion errors during sustained load.
- Goroutine count stable (delta < 5%) after 4-hour soak.
- Redis memory growth < 20% above baseline after load test.
- Error rate < 0.1% during normal load (excluding intentional 429 responses).

**Tools:**
- k6 (load and spike tests, p95 measurement)
- `pprof` (goroutine and heap profiling during soak)
- Prometheus + Grafana (real-time connection pool and Redis metrics)
- `pg_stat_activity` and `pg_stat_statements` (query count and plan validation)
- Redis `INFO memory` and `INFO stats` (memory and hit rate)

---

### 3. Security — Priority: Critical

**Justification:** This system is the authentication perimeter for all tenants. A single security failure — credential exposure, cross-tenant data leak, JWT bypass, or audit log gap — is a product-ending event. Security testing is not optional and not sampled: every auth endpoint, every data path, and every trust boundary must be verified. The SOC2 Type II goal (ADR-008) requires demonstrable, documented security test evidence.

**Test Approach:**
- **Static analysis:** run `gosec` and `semgrep` on all Go source; `gitleaks` on every commit.
- **Secrets verification:** assert passwords are stored as Argon2id hashes; assert refresh tokens are stored as SHA-256 hashes; assert no raw credentials appear in API responses, logs, or error messages.
- **JWT integrity:** fuzz JWT headers (`alg: none`, `alg: HS256` with public key, tampered payload) and assert all rejected with 401.
- **RBAC enforcement:** for every protected endpoint, test with a token bearing insufficient role — assert 403.
- **Cross-tenant isolation:** tenant A's token must never access tenant B's data — assert 403 on all cross-tenant attempts across every data endpoint.
- **Audit log integrity:** assert no UPDATE or DELETE path exists on audit records at the database layer; assert append-only constraint.
- **OWASP Top 10** relevant categories: A01 (RBAC), A02 (cryptographic failures), A03 (injection via SQL parameterization), A07 (auth failures).
- **Rate limiting:** verify brute force protection engages after threshold and Redis unavailability does not bypass it.
- **Penetration testing:** manual pentest in Sprint 8 covering OWASP ASVS Level 2.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-SEC-001 | Password storage: after registration, assert `password_hash` column contains Argon2id hash, not plaintext |
| TC-SEC-002 | Refresh token storage: assert stored value is SHA-256 hash, not raw token |
| TC-SEC-003 | JWT `alg: none` attack: send token with `alg: none` → 401 |
| TC-SEC-004 | JWT algorithm confusion: sign with HMAC using public RSA key → 401 |
| TC-SEC-005 | Tampered JWT payload (role escalation): modify `roles` claim, re-encode without re-signing → 401 |
| TC-SEC-006 | Expired JWT: send token where `exp` is in the past → 401 |
| TC-SEC-007 | Cross-tenant isolation: tenant A user calls `/api/v1/users` with tenant B's `X-Tenant-ID` → 403 |
| TC-SEC-008 | Cross-tenant isolation: tenant A's JWT cannot retrieve tenant B's audit logs |
| TC-SEC-009 | RBAC: `viewer` role calling `DELETE /api/v1/users/{id}` → 403 |
| TC-SEC-010 | Brute force: 6th consecutive failed login → 429 with `Retry-After` header |
| TC-SEC-011 | Redis down during rate limit check: request is rejected (fail closed), not passed through |
| TC-SEC-012 | No credentials in error responses: failed login returns `401` with generic message, no password hint, no user existence oracle |
| TC-SEC-013 | No sensitive data in logs: after login attempt, assert log lines contain no `password`, no raw JWT, no refresh token |
| TC-SEC-014 | SQL injection: send `' OR '1'='1` in email field → 400, no SQL error in response, parameterized query confirmed |
| TC-SEC-015 | Audit log append-only: attempt direct SQL `UPDATE` or `DELETE` on `audit_log` table → permission denied |
| TC-SEC-016 | Audit log completeness: LOGIN_SUCCESS, LOGIN_FAILED, TOKEN_REFRESH, LOGOUT, PASSWORD_RESET, ROLE_ASSIGNED events all present after triggering each action |
| TC-SEC-017 | HTTP security headers: all responses include `X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`, `Content-Security-Policy` |
| TC-SEC-018 | PKCE required: OAuth Authorization Code flow without `code_challenge` → 400 |
| TC-SEC-019 | Social login account linking: attempt linking Google account without password verification → rejected |
| TC-SEC-020 | `gosec` scan: zero High or Critical findings in CI |

**Pass Criteria:**
- Zero Critical security vulnerabilities open at launch (product requirement).
- All JWT tamper/bypass tests return 401.
- Zero cross-tenant data access succeeds in the isolation test suite.
- All 20 auth action types produce correct audit log entries.
- `gosec` and `semgrep` report zero High/Critical findings in CI.
- Rate limiting engages correctly and fails closed when Redis is unavailable.
- No credentials or raw tokens present in any log line or error response.

**Tools:**
- `gosec` (Go static security analysis)
- `semgrep` with Go security ruleset
- `gitleaks` (secret scanning in CI)
- OWASP ZAP (active scan in staging)
- Manual penetration test (Sprint 8) against OWASP ASVS Level 2
- Custom Go test suite for JWT attack vectors
- SQL assertions for audit log append-only constraint
- `curl` scripts for RBAC and cross-tenant boundary testing

---

### 4. Reliability — Priority: High

**Justification:** A 99.9% uptime SLA allows approximately 8.7 hours of downtime per year. Authentication is a critical dependency of every other service — auth unavailability cascades. Reliability failures must be graceful: Redis unavailability must not bypass security controls, database saturation must return 503 not hang indefinitely, and no goroutine or connection leak may accumulate to cause eventual process death.

**Test Approach:**
- **Fault injection:** use Docker network rules to simulate Redis unavailability, PostgreSQL unavailability, and Vault unavailability; assert system behavior at each failure boundary.
- **Connection pool exhaustion:** cap the pool below load and verify 503 responses, not indefinite hangs.
- **Circuit breaker behavior:** verify timeout and fallback behavior for all external dependencies.
- **Chaos testing:** kill individual service replicas during load test; assert remaining replicas absorb traffic without cascading failure.
- **Memory/goroutine leak detection:** `pprof` during soak tests.
- **Recovery testing:** restore Redis/PostgreSQL after failure; assert system recovers without restart.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-REL-001 | Redis unavailable: login attempt proceeds but rate limiting fails closed (request rejected, not passed) |
| TC-REL-002 | Redis unavailable: JWT validation continues (stateless path) — introspection cache miss, falls back to DB check |
| TC-REL-003 | PostgreSQL connection pool exhausted: new requests return 503 with `Retry-After`; no request hangs > 5 seconds |
| TC-REL-004 | PostgreSQL unavailable: service returns 503 within 2 seconds (not timeout after 30s) |
| TC-REL-005 | Vault unavailable at startup: service fails fast with clear error (does not start with cached secrets silently) |
| TC-REL-006 | Soak test (4h, 500 VUs): goroutine count delta < 5% from baseline (no leak) |
| TC-REL-007 | Soak test (4h, 500 VUs): heap allocation stable (no unbounded growth) |
| TC-REL-008 | Replica kill during load: traffic redistributed within 5 seconds, no 5xx spike lasting > 10 seconds |
| TC-REL-009 | Redis recovery: after restoring Redis following outage, rate limiting resumes correctly within 30 seconds |
| TC-REL-010 | Database migration idempotency: running migrations twice on same DB produces no errors and no data loss |
| TC-REL-011 | Graceful shutdown: in-flight requests complete (or return 503 with `Connection: close`) within shutdown timeout |

**Pass Criteria:**
- Redis unavailability never bypasses security controls (rate limiting fails closed).
- All dependency failures return appropriate HTTP error codes within 5 seconds; zero indefinite hangs.
- Goroutine count and heap are stable across 4-hour soak test.
- System achieves 99.9% uptime in staging chaos tests (measured over a 72-hour window).
- Recovery from Redis and PostgreSQL outages is automatic and requires no process restart.

**Tools:**
- Docker network partitioning (`docker network disconnect`) for fault injection
- `toxiproxy` (latency injection, connection drop simulation)
- `pprof` (goroutine and heap profiling)
- k6 (load during chaos)
- Prometheus + Grafana (uptime and error rate measurement)
- `go test` with mock/stub dependencies for unit-level fault path coverage

---

### 5. Compatibility — Priority: Medium

**Justification:** The system must interoperate correctly with OAuth 2.0 clients, OIDC-aware consumers, and other internal services. API contract stability is essential — breaking changes break tenant integrations. The `X-Tenant-ID` header (ADR-010) and standard error envelopes must be consistent across all versions. RFC compliance determines whether standard OAuth libraries will work against this system without custom workarounds.

**Test Approach:**
- **RFC conformance:** execute RFC 6749, 7636, and 7662 conformance scenarios using standard OAuth 2.0 client libraries (e.g., `golang.org/x/oauth2`), not just curl, to verify real-world interoperability.
- **OIDC discovery:** validate `/.well-known/openid-configuration` against OIDC Core 1.0 specification field requirements.
- **Contract regression:** run the full API contract test suite on every PR; any schema change that breaks the contract must be a deliberate version bump.
- **Header interoperability:** verify `X-Tenant-ID` is correctly threaded through all middleware without stripping or modification by any proxy layer.
- **Error format consistency:** assert all error responses (400, 401, 403, 404, 429, 500) use the same `{"error": "code", "message": "..."}` envelope.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-COMP-001 | OAuth Authorization Code + PKCE flow using `golang.org/x/oauth2` client library — full flow succeeds |
| TC-COMP-002 | Client Credentials grant using standard OAuth client — access token issued and introspectable |
| TC-COMP-003 | OIDC discovery document: all required fields present (`issuer`, `authorization_endpoint`, `token_endpoint`, `jwks_uri`, `response_types_supported`) |
| TC-COMP-004 | JWKS endpoint: public key returned is valid RSA JWK; verify signature of issued JWT using this key |
| TC-COMP-005 | Error envelope consistency: trigger 400, 401, 403, 404, 429, 500 — all return `{"error": "...", "message": "..."}` with correct HTTP status |
| TC-COMP-006 | API versioning: `v1` endpoints remain accessible after `v2` is introduced; no breaking change without version bump |
| TC-COMP-007 | `X-Tenant-ID` header: missing header → 400 with `tenant_id_required` error code |
| TC-COMP-008 | `X-Tenant-ID` header: unknown tenant ID → 404 with `tenant_not_found` error code |
| TC-COMP-009 | Co-existence: auth service runs alongside a stub downstream service on Docker Compose; no port conflicts, no shared state |

**Pass Criteria:**
- All RFC 6749/7636/7662 conformance scenarios pass using real OAuth client libraries.
- OIDC discovery document passes validation against OIDC Core 1.0 specification.
- Zero breaking contract changes introduced without a version bump.
- Error envelope format is 100% consistent across all endpoint error responses.
- `X-Tenant-ID` is correctly processed on 100% of requests.

**Tools:**
- `golang.org/x/oauth2` and standard OIDC client libraries for interoperability tests
- Postman/Newman contract test collection (run in CI)
- `jwt.io` and Go JWT library (`github.com/golang-jwt/jwt`) for JWKS validation
- `openid-configuration` validator (custom Go script against OIDC Core 1.0 spec)
- Docker Compose multi-service test environment

---

### 6. Usability — Priority: Low

**Justification:** This is an API-only system with no hosted UI (ADR-003). The "user" is a developer integrating the API, not an end user clicking a form. Usability failures manifest as ambiguous error codes, inconsistent error messages, or non-actionable responses that force developers to read source code to debug integrations. The Priya persona (developer integrating the system) must be able to complete a basic integration in under half a day.

**Test Approach:**
- **Error message review:** manually inspect all error responses for actionability — does the message tell the developer what to fix?
- **Error code consistency:** automated assertion that all errors use documented error code strings (no ad-hoc messages).
- **Developer experience test:** timed walkthrough — a developer following the README from scratch achieves their first successful API call (register, login, access protected endpoint) in < 30 minutes.
- **API documentation accuracy:** verify that every example in the API documentation produces the documented response when executed against staging.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-USE-001 | Send request with missing required field → 400 response identifies the exact missing field name in `message` |
| TC-USE-002 | Send expired JWT → 401 with `error: "token_expired"` and `message` indicating expiry (not generic "unauthorized") |
| TC-USE-003 | All error codes in API responses are from the documented error code registry (no undocumented strings) |
| TC-USE-004 | Developer experience: follow README from `git clone` to first successful authenticated API call in < 30 minutes |
| TC-USE-005 | `docker compose up` brings all services healthy in < 5 minutes on a standard developer machine |
| TC-USE-006 | Every documented API example in solution-architecture.md produces the documented response when run against staging |

**Pass Criteria:**
- All 400-series error responses identify the specific error cause, not a generic message.
- Zero undocumented error code strings appear in API responses.
- Developer experience timed test completes in < 30 minutes.
- `docker compose up` ready state in < 5 minutes.
- 100% of API documentation examples produce the documented response.

**Tools:**
- Manual API inspection (QA engineer as developer persona)
- Newman/Postman collection asserting error code registry compliance
- Timer-based developer experience test (manual, documented)
- Docker Compose startup time measurement (`docker compose up --wait` with timing)

---

### 7. Maintainability — Priority: High

**Justification:** Authentication systems are never "done" — vulnerabilities are discovered, OAuth specs evolve, and new tenant requirements emerge. A maintainable codebase allows security patches to be applied quickly (Mean Time to Fix Critical < 24h requires this). Interface-based dependencies and layered architecture are only valuable if enforced and tested. Coverage targets (> 80% line, > 90% branch on critical paths) are a proxy for test quality and refactor safety.

**Test Approach:**
- **Coverage measurement:** `go test -coverprofile=coverage.out ./internal/...` on every CI run; coverage report published as CI artifact.
- **Architecture linting:** use `go-arch-lint` or `depguard` to assert no Handler imports Repository directly (layer isolation enforced).
- **Structured log verification:** integration tests assert that every log line contains `request_id`, `tenant_id`, and `user_id` where applicable.
- **Static analysis:** `golangci-lint` with enabled rules for complexity, duplication, and naming; enforced in CI as a blocking gate.
- **Interface compliance:** assert all service and repository types implement their declared interfaces via compile-time checks (`var _ ServiceInterface = (*ConcreteService)(nil)`).

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-MAINT-001 | CI gate: `go test -cover` reports ≥ 80% line coverage on all packages in `internal/` |
| TC-MAINT-002 | CI gate: branch coverage ≥ 90% on critical path packages (`auth`, `token`, `tenant`) — measured and reviewed manually against coverage HTML |
| TC-MAINT-003 | `golangci-lint` passes with zero errors on every PR |
| TC-MAINT-004 | Architecture assertion: no direct import from `handler` package to `repository` package (enforced by `depguard` or test) |
| TC-MAINT-005 | Log completeness: integration test triggers login, asserts log output contains `request_id`, `tenant_id`, `user_id` on every line produced by that request |
| TC-MAINT-006 | Interface compliance: compile-time assertion for every service and repository interface exists in `_test` package |
| TC-MAINT-007 | Cyclomatic complexity: no function exceeds complexity of 15 (enforced by `golangci-lint gocyclo` rule) |

**Pass Criteria:**
- Line coverage ≥ 80% on all `internal/` packages in every CI run (sprint exit blocking).
- Branch coverage ≥ 90% on `auth`, `token`, and `tenant` packages.
- `golangci-lint` passes with zero errors on every PR — blocking gate.
- No direct Handler-to-Repository imports detected.
- 100% of structured log lines in integration tests contain required correlation fields.

**Tools:**
- `go test -coverprofile` + `go tool cover -html` (coverage reporting)
- `golangci-lint` (static analysis, complexity, naming)
- `depguard` or custom `go-arch-lint` rules (architecture enforcement)
- `gocyclo` (cyclomatic complexity)
- CI artifact publishing (coverage HTML report per sprint)

---

### 8. Portability — Priority: Medium

**Justification:** The system targets Fly.io as primary deployment but must not be so tightly coupled that migration is impossible. `docker compose up` is the developer onboarding path — it must work on macOS, Linux, and Windows (WSL2) without manual steps. The pluggable secrets manager interface (ADR-009) ensures Vault can be replaced with AWS Secrets Manager or GCP Secret Manager without rewriting the application layer.

**Test Approach:**
- **Docker Compose smoke test:** run `docker compose up` on a clean environment and assert all health checks pass within 5 minutes; run on at least two platforms (Linux CI runner and local Windows/WSL2).
- **Environment variable configuration:** assert the application starts correctly with all supported environment variable combinations; assert it fails fast with a clear error when required variables are missing.
- **Secrets manager interface:** unit tests for the `SecretsManager` interface are run against both the Vault adapter and a test stub; the interface contract is verified, not the implementation.
- **Container image portability:** built Docker image runs on `linux/amd64` and `linux/arm64` without modification.

**Key Test Cases:**

| TC-ID | Description |
|-------|-------------|
| TC-PORT-001 | `docker compose up` on Linux CI runner: all services healthy within 5 minutes |
| TC-PORT-002 | `docker compose up` on Windows WSL2 (developer machine): all services healthy within 5 minutes |
| TC-PORT-003 | Application starts with minimal required env vars set; all optional vars default correctly |
| TC-PORT-004 | Application fails fast (exit code non-zero, clear error message) when `DATABASE_URL` is missing |
| TC-PORT-005 | `SecretsManager` interface: Vault adapter and test stub both pass the same interface contract test suite |
| TC-PORT-006 | Docker image: `docker buildx build --platform linux/amd64,linux/arm64` succeeds without errors |
| TC-PORT-007 | Fly.io deployment: `fly deploy` in staging environment succeeds and health check passes within 3 minutes |
| TC-PORT-008 | Database migration portability: migrations run successfully on PostgreSQL 14, 15, and 16 |

**Pass Criteria:**
- `docker compose up` reaches healthy state in < 5 minutes on both Linux CI and Windows WSL2.
- Application fails fast (< 3 seconds) with a clear error message for any missing required configuration.
- `SecretsManager` interface contract tests pass for all registered adapters.
- Multi-platform Docker image builds succeed without modification.
- Fly.io staging deployment succeeds and passes health checks in < 3 minutes.

**Tools:**
- Docker Compose (`docker compose up --wait` with timeout measurement)
- GitHub Actions matrix (test across platforms and PostgreSQL versions)
- `docker buildx` (multi-platform image build)
- `fly deploy` + Fly.io health check API (staging deployment verification)
- Go interface contract test suite (custom, against `SecretsManager` interface)

---

## Section 7: Bug Report Template

---

### Bug Report Template

```markdown
---
## Bug Report

**Bug ID:** BUG-YYYY-NNN
*(Format: BUG-{year}-{sequential number within year}. Example: BUG-2026-042)*

**Title:** [Imperative verb phrase describing the defect, < 80 characters]
*(Example: "Login endpoint returns 200 when password is incorrect")*

---

### Classification

**Severity:** [Critical / High / Medium / Low]

| Level | Definition for Authentication System |
|-------|--------------------------------------|
| **Critical** | Security vulnerability, authentication bypass, cross-tenant data leak, credential exposure in response or log, data breach risk, audit log tampered or missing for a security event |
| **High** | Auth flow broken for any user segment, JWT not issued when it should be or rejected when it should not be, token rotation failure, audit log missing events (but no security breach), MFA bypass, RBAC not enforced |
| **Medium** | Error message leaks existence oracle (user/email enumeration), performance outside NFR targets, rate limit bypassable under specific conditions, incorrect HTTP status code returned, wrong JWT claim value |
| **Low** | Cosmetic issue in error message wording, non-blocking documentation inaccuracy, minor inconsistency in error field naming, unused log field |

**Priority:** [P1 / P2 / P3 / P4]

| Level | Definition |
|-------|------------|
| **P1 — Fix Now** | Stop-ship issue; block sprint release; requires immediate developer assignment and fix within 24 hours. All Critical severity bugs are P1. |
| **P2 — Fix This Sprint** | Must be resolved before sprint sign-off. Typically High severity bugs. |
| **P3 — Fix Next Sprint** | Scheduled for the immediately following sprint. Typically Medium severity. |
| **P4 — Backlog** | Track and accept, or fix when capacity allows. Typically Low severity. |

**Status:** [New / Assigned / In Progress / Fixed / Verified / Closed / Won't Fix]

---

### People & Timeline

| Field | Value |
|-------|-------|
| **Reporter** | |
| **Assignee** | |
| **Sprint Found** | Sprint [N] — [Theme] |
| **Sprint Fixed** | Sprint [N] — [Theme] *(filled when resolved)* |
| **Date Reported** | YYYY-MM-DD |
| **Date Fixed** | YYYY-MM-DD *(filled when resolved)* |

---

### Affected Component

**Component:** [Select one]
- [ ] `auth-service` — Core registration, login, logout, password flows
- [ ] `oauth` — Authorization Code, Client Credentials, PKCE, introspection
- [ ] `multi-tenancy` — Schema routing, X-Tenant-ID handling, tenant isolation
- [ ] `rate-limiting` — Brute force protection, Redis-backed rate limiter
- [ ] `audit-log` — Event emission, append-only enforcement, log completeness
- [ ] `session` — Refresh token lifecycle, token family revocation, expiry
- [ ] `rbac` — Role assignment, claims in JWT, endpoint authorization

---

### Environment

| Field | Value |
|-------|-------|
| **Environment** | [ ] Local  [ ] CI  [ ] Staging |
| **Go Version** | go1.XX.X |
| **PostgreSQL Version** | 1X.X |
| **Redis Version** | X.X.X |
| **Git Commit / Build** | `abc1234` |
| **Branch** | `feature/...` or `main` |
| **Docker Compose Version** | X.XX.X *(if applicable)* |

---

### Auth Context

*Complete all fields that are relevant to this defect. These fields are mandatory for Security and Multi-Tenancy components.*

| Field | Value |
|-------|-------|
| **tenant_id** | *(Which tenant schema was affected, e.g., `tenant_abc123`)* |
| **user_id** | *(Affected user UUID, if applicable)* |
| **request_id** | *(Value of `X-Request-ID` header from the failing request)* |
| **Endpoint** | `METHOD /api/v1/path` — e.g., `POST /api/v1/auth/login` |
| **HTTP Status Received** | `4XX` / `5XX` |
| **HTTP Status Expected** | `2XX` / `4XX` |
| **JWT Claims (if relevant)** | `sub: ..., roles: [...], exp: ..., tenant_id: ...` *(redact sensitive values)* |
| **OAuth Client ID (if relevant)** | |
| **Audit Event Expected** | e.g., `LOGIN_FAILED` |
| **Audit Event Observed** | e.g., `(none)` or `LOGIN_SUCCESS` |

---

### Defect Detail

#### Steps to Reproduce
*(Numbered steps. Include exact curl command or request body where possible.)*

1. 
2. 
3. 

#### Expected Behavior
*(What should happen according to the specification, PRD, or Gherkin acceptance criteria.)*

#### Actual Behavior
*(What actually happens. Be specific about the incorrect status code, wrong field value, or missing event.)*

---

### Evidence

**Reproducing curl Command:**
```bash
curl -X POST https://staging.auth.example.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant_abc123" \
  -H "X-Request-ID: req_reproduce_001" \
  -d '{"email": "user@example.com", "password": "wrong_password"}'
```

**Actual Response:**
```json
{
  "error": "...",
  "message": "..."
}
```

**Relevant Log Snippet:**
```
[paste structured log lines here — remove any actual secrets before pasting]
```

**Audit Log Entry (if applicable):**
```json
{
  "event_type": "...",
  "user_id": "...",
  "tenant_id": "...",
  "ip_address": "...",
  "timestamp": "...",
  "outcome": "..."
}
```

**Additional Evidence:**
*(Screenshots, Prometheus metrics, pprof output, k6 summary — attach as files if large.)*

---

### Resolution

*(Filled by the developer assigned to this bug.)*

**Root Cause:**
*(What was the underlying cause? e.g., "Rate limiter middleware returned `nil` error when Redis connection timed out, causing it to proceed rather than reject.")*

**Fix Description:**
*(What was changed? Reference PR number.)*
- PR: #
- Files changed:

**Regression Test Added?**
- [ ] Yes — TC-ID: `TC-XXX-NNN`
- [ ] No — Reason:

---

### Verification

*(Filled by QA after fix is deployed to staging.)*

**Verification Steps:**
1. Deploy fix to staging.
2. Re-execute reproducing curl command from Evidence section.
3. Assert response matches Expected Behavior.
4. Assert audit log event is correct (if applicable).
5. Run regression test `TC-XXX-NNN` — assert pass.

**Verification Result:** [ ] PASS  [ ] FAIL

**Verified By:** _______________  **Date:** YYYY-MM-DD

---
```

---

## Section 8: Definition of Done — Per Story Checklist

---

### Global Definition of Done

Every story must satisfy all applicable checklist items before QA signs off. Items marked `(ALL)` apply to every story. Items marked with a scope condition apply only when that condition is true for the story being reviewed.

---

#### Code Quality `(ALL)`

- [ ] Code reviewed and approved by at least one peer before merge.
- [ ] `golangci-lint run ./...` passes with zero errors or warnings.
- [ ] `go vet ./...` passes with zero warnings.
- [ ] `gitleaks detect --source .` passes — no secrets committed.
- [ ] No hardcoded credentials, connection strings, or private keys in source code.
- [ ] No `TODO` or `FIXME` comments introduced without a linked issue.

---

#### Unit Tests `(ALL)`

- [ ] Unit tests written for all new service-layer functions introduced by this story.
- [ ] Unit test coverage on new code is at least 80% (measured with `go test -cover ./internal/...`).
- [ ] All unit tests pass: `go test ./internal/... -count=1`.
- [ ] Table-driven tests (`[]struct{ name, input, expected }`) used for all functions that handle multiple input variations.
- [ ] Unit tests use interface mocks (not real database or Redis connections).
- [ ] No test uses `time.Sleep` — use channels or mock clocks for timing-dependent logic.

---

#### Integration Tests `(ALL)`

- [ ] Integration test covers the primary happy path for this story's endpoint(s) against real PostgreSQL and Redis (Docker Compose test environment).
- [ ] Integration tests cover at least two distinct negative/error paths (e.g., missing field, unauthorized, resource not found).
- [ ] All integration tests pass: `go test ./integration/... -tags=integration -count=1`.
- [ ] API response shape (field names, field types, HTTP status code) matches the contract documented in `solution-architecture.md`.
- [ ] No integration test asserts against implementation internals — tests assert only the HTTP API contract.

---

#### Cross-Tenant Isolation *(Required for all stories that read or write user data, sessions, audit logs, roles, or any tenant-scoped resource)*

- [ ] `make test-isolation` passes with zero failures.
- [ ] The new endpoint(s) introduced by this story are covered by at least one isolation test (tenant A token cannot access tenant B data via this endpoint).
- [ ] No new code bypasses the tenant schema routing middleware.
- [ ] A cross-tenant access attempt for this story's resource returns `403` with `{"error": "forbidden"}`.
- [ ] Tenant schema is set via middleware before any SQL query executes — verified by integration test asserting correct schema name in query logs.

---

#### Security `(ALL)`

- [ ] All user-supplied input is validated before use (field presence, type, length, format).
- [ ] No raw user input is concatenated into SQL queries — parameterized queries or ORM used exclusively.
- [ ] Auth middleware is applied to all new protected endpoints — verified by integration test asserting `401` when no JWT is provided.
- [ ] No passwords, raw refresh tokens, private keys, or full JWTs appear in any log line or error response — verified by log assertion in integration test.
- [ ] HTTP security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Strict-Transport-Security`) are present on all responses from new endpoints.
- [ ] `gosec` reports no new High or Critical findings introduced by this story's code.

---

#### Audit Log *(Required for all stories that perform an auth action: login, logout, register, password reset, token refresh, role assignment, MFA, social link, admin action)*

- [ ] The correct `event_type` is written to the audit log for this story's action (e.g., `LOGIN_SUCCESS`, `ROLE_ASSIGNED`, `MFA_ENABLED`).
- [ ] The audit event record contains all required fields: `event_type`, `user_id`, `tenant_id`, `ip_address`, `timestamp`, `outcome`.
- [ ] Audit event is written on both success and failure paths for this action.
- [ ] Audit log is append-only: no `UPDATE` or `DELETE` path exists for audit records introduced or touched by this story.
- [ ] Audit log write is verified by integration test — test asserts the record exists in the database after the action completes.

---

#### Performance *(Required for all stories that introduce or modify an API endpoint)*

- [ ] Endpoint p95 response time is within NFR targets when measured against staging with representative load (login < 300ms, token issuance < 100ms, introspection < 50ms).
- [ ] No N+1 query pattern introduced — integration test asserts query count per request is within the expected bound.
- [ ] All new query patterns use appropriate database indexes — confirmed by `EXPLAIN ANALYZE` output reviewed in PR.
- [ ] No synchronous blocking call to an external service (Vault, Resend, Redis) is made inside a request handler without a timeout.

---

#### API Contract `(ALL)`

- [ ] Success response matches the documented schema in `solution-architecture.md` (field names, types, HTTP status code).
- [ ] All documented error conditions return the correct HTTP status code and `{"error": "code", "message": "human-readable"}` envelope.
- [ ] No new error code strings introduced that are not in the documented error code registry.
- [ ] If this story changes the API contract, `solution-architecture.md` is updated in the same PR.
- [ ] Newman/Postman contract test collection updated if new endpoints are introduced.

---

#### Deployment `(ALL)`

- [ ] Code merged to `main` branch and CI pipeline passes all stages (lint, test, security scan, build).
- [ ] Deployed to staging environment successfully (`fly deploy` or equivalent).
- [ ] All database migrations applied to staging without errors.
- [ ] No regressions in the existing test suite after deployment (all previously passing tests still pass).
- [ ] QA engineer has verified the story in the staging environment, not only in local or CI.

---

#### Release Gate *(Sprint exit — applies when signing off the full sprint, not individual stories)*

- [ ] Zero Critical severity defects open.
- [ ] Zero High severity defects open — or each open High defect has explicit stakeholder sign-off and a documented acceptance decision in the bug tracker.
- [ ] Test coverage report generated (`make test-coverage`) and confirms line coverage ≥ 80% across `internal/`, branch coverage ≥ 90% on `auth`, `token`, and `tenant` packages.
- [ ] Security checklist signed off by Engineering Lead.
- [ ] Sprint test report completed and linked in the sprint retrospective notes.
- [ ] All stories in the sprint are in `Closed` or `Won't Fix` status — no stories in `In Progress` or `Fixed` (awaiting verification).

---

## Section 9: Test Metrics and Reporting

---

### KPIs Per Sprint

| Metric | Target | Measurement Method |
|--------|--------|--------------------|
| Unit test coverage — line | >= 80% across all `internal/` packages | `go test -coverprofile=coverage.out ./internal/...` + `go tool cover -func` |
| Unit test coverage — branch, critical paths | >= 90% on `auth`, `token`, `tenant` packages | `go test -coverprofile` + manual review of HTML report for uncovered branches |
| Integration test pass rate | 100% — any failure blocks sprint sign-off | CI pipeline (`go test ./integration/... -tags=integration`) |
| Cross-tenant isolation pass rate | 100% — hard blocking gate, no exceptions | `make test-isolation` |
| Defect detection rate (pre-production) | >= 95% of defects found before reaching production | Defect log: (bugs found pre-prod) / (total bugs reported incl. post-prod) * 100 |
| Critical defects open at sprint end | 0 — absolute; sprint cannot close with any open Critical | Bug tracker severity filter |
| High defects open at sprint end | <= 2 (Sprints 1–7); 0 (Sprints 8–9) | Bug tracker severity filter |
| Build success rate | >= 95% of CI pipeline runs succeed | GitHub Actions run history |
| Mean time to fix Critical defect | < 24 hours from report to verified fix in staging | Bug tracker: `date_fixed` minus `date_reported` |
| NFR compliance — login p95 | < 300ms at 1,000 concurrent VUs | k6 report: `http_req_duration{p(95)}` on login endpoint |
| NFR compliance — token issuance p95 | < 100ms at 1,000 concurrent VUs | k6 report on token endpoint |
| NFR compliance — introspection p95 | < 50ms at 1,000 concurrent VUs | k6 report on introspection endpoint |
| Regression rate | < 5% of fixed bugs reintroduced in the same or following sprint | Defect log: count of bugs where `regression: true` |

---

### Sprint Test Report Template

```markdown
---
## Sprint [N] Test Report

**Sprint:** [N] — [Theme, e.g., "Core Auth — Register, Login, Email Verify, Password Reset"]
**Period:** YYYY-MM-DD to YYYY-MM-DD
**QA Engineer:** [Name]
**Report Date:** YYYY-MM-DD
**Overall Status:** PASS / CONDITIONAL PASS / FAIL

> **PASS** — All exit criteria met. Sprint can be closed.
> **CONDITIONAL PASS** — Minor open items accepted with documented risk. Stakeholder sign-off obtained.
> **FAIL** — One or more blocking exit criteria not met. Sprint cannot be closed until resolved.

---

### Test Execution Summary

| Test Type | Total Executed | Pass | Fail | Skip | Coverage / Notes |
|-----------|---------------|------|------|------|-----------------|
| Unit | | | | | Line coverage: XX% |
| Integration | | | | | |
| Isolation (cross-tenant) | | | | | |
| Security (static + dynamic) | | | | | `gosec` findings: X High, X Critical |
| Performance (k6) | | | | | p95 login: XXXms, token: XXms, introspection: XXms |
| E2E / Playwright | | | | | |
| **Total** | | | | | |

---

### Defect Summary

| Severity | New This Sprint | Fixed This Sprint | Open (End of Sprint) | Accepted (Documented) |
|----------|----------------|-------------------|----------------------|-----------------------|
| Critical | | | | |
| High | | | | |
| Medium | | | | |
| Low | | | | |
| **Total** | | | | |

**Open defects carried forward:**

| Bug ID | Title | Severity | Priority | Acceptance Decision |
|--------|-------|----------|----------|---------------------|
| | | | | |

---

### Quality Gate Status

| Gate | Status | Threshold | Actual | Notes |
|------|--------|-----------|--------|-------|
| Unit line coverage >= 80% | PASS / FAIL | 80% | XX% | |
| Branch coverage >= 90% (critical paths) | PASS / FAIL | 90% | XX% | |
| Integration tests 100% pass | PASS / FAIL | 100% | XX% | |
| Isolation suite 100% pass | PASS / FAIL | 100% | XX% | |
| 0 Critical defects open | PASS / FAIL | 0 | N | |
| High defects within threshold | PASS / FAIL | <=2 (Sprints 1-7) / 0 (8-9) | N | |
| NFR login p95 < 300ms | PASS / FAIL / N/A | 300ms | XXXms | N/A if perf test not run this sprint |
| NFR token p95 < 100ms | PASS / FAIL / N/A | 100ms | XXms | |
| NFR introspection p95 < 50ms | PASS / FAIL / N/A | 50ms | XXms | |
| `gosec` 0 High/Critical findings | PASS / FAIL | 0 | N | |
| Build success rate >= 95% | PASS / FAIL | 95% | XX% | |

---

### Stories Tested This Sprint

| Story ID | Story Title | Test Result | TC-IDs Executed | Open Defects |
|----------|-------------|-------------|-----------------|--------------|
| US-XXX | | PASS / FAIL | TC-XXX-001, ... | BUG-2026-NNN |

---

### Key Findings

*(Bullet points — notable defects discovered, patterns observed, unexpected behavior, test blockers.)*

- 
- 
- 

---

### Risks Carried Forward

*(Quality risks or open questions moving into the next sprint. Include mitigation plan for each.)*

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| | | | |

---

### Test Environment Notes

*(Any environment issues that affected test execution or results this sprint.)*

- 

---

### QA Sign-off

- [ ] All exit criteria met (or conditional acceptance documented above).
- [ ] Engineering Lead notified of sprint test report.
- [ ] All defects logged in bug tracker with correct severity, priority, and component.
- [ ] Coverage report generated and archived at `coverage/sprint-[N]/index.html`.
- [ ] Sprint retrospective notes updated with quality observations.

**Signed:** __________________________________ **Date:** YYYY-MM-DD

**Engineering Lead Sign-off (required for Sprints 8 and 9):**
**Signed:** __________________________________ **Date:** YYYY-MM-DD

---
```

---

### Defect Severity Distribution Target at Launch

The following thresholds define the minimum acceptable defect state before any production release is authorized. These are not aspirational targets — they are hard gates.

| Severity | Maximum Open at Launch | Rationale |
|----------|----------------------|-----------|
| Critical | **0 — absolute** | Any open Critical is a security vulnerability, auth bypass, or data breach risk. Launch is blocked unconditionally. |
| High | **0 — absolute** | Open High defects indicate broken auth flows or missing audit events. Launch is blocked unconditionally. |
| Medium | **<= 5** | Each open Medium must be individually accepted with documented justification, risk owner, and remediation sprint target. |
| Low | **<= 15** | Tracked in backlog. Grouped for review post-launch. No individual justification required but aggregate reviewed by Engineering Lead. |

---

### How to Interpret Coverage Reports

**Generating the report:**

```bash
make test-coverage
# Equivalent to:
go test -coverprofile=coverage.out -covermode=atomic ./internal/...
go tool cover -html=coverage.out -o coverage/index.html
go tool cover -func=coverage.out | tail -1   # prints total line coverage
```

The HTML report is published as a CI artifact on every main branch merge and on every sprint sign-off run.

**Reading the HTML report:**

Red-highlighted lines are uncovered. Not all red lines require immediate action — apply the following decision process:

| Coverage Level | Action |
|---------------|--------|
| < 80% total line coverage | Sprint sign-off is **blocked**. Add tests before any stories are marked Done. |
| 80%–90% total line coverage | **Acceptable**. Document which uncovered lines are excluded and the justification (e.g., `main()` startup code, OS signal handlers, third-party adapter boilerplate). Add justification as a comment in `coverage/exclusions.md`. |
| > 90% total line coverage | **Excellent**. Do not pursue 100% — testing behavior, not lines, is the goal. Marginal test additions for cosmetic coverage metrics reduce test quality. |
| Branch coverage < 90% on `auth`, `token`, `tenant` packages | **Blocking for those packages**. Branch gaps in auth code are the most dangerous — they represent error handling paths most likely to contain security defects. |

**Prioritizing uncovered branches:**

When branch coverage is below target, prioritize uncovered branches in this order:

1. Error return paths in the `auth` and `token` packages — these are the paths most likely to contain security-relevant behavior (e.g., what happens when JWT signing fails, when Argon2id returns an error, when the refresh token lookup finds nothing).
2. Redis and database fallback paths — failure handling under dependency unavailability.
3. Tenant schema routing edge cases — incorrect schema set, unknown tenant.
4. All other uncovered branches.

Do not write tests purely to increase a coverage number. Each test must assert a specific, documented behavior. A test that executes a line but asserts nothing is worse than no test — it creates false confidence.

---

*End of Test Plan — Authentication System v1.0*
*All open questions resolved. Pipeline complete: PO → PM → SA → Dev → Tester.*