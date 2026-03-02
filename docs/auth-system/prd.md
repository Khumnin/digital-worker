# Product Requirements Document
# Authentication System — Pilot Project

**Version:** 1.1 (Decisions Finalized)
**Date:** 2026-02-27
**Status:** ✅ Approved for Development
**Author:** Product Owner Agent
**Intended Consumers:** Project Manager, Solution Architect, Developer, Tester agents

---

## Revision History

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-02-27 | Initial brainstorm draft |
| 1.1 | 2026-02-27 | All open questions resolved; backlog refined; estimates updated |

---

## 1. Problem Statement

Teams lack a self-hosted, standards-compliant authentication platform that scales across multiple tenants and integrates with external identity providers — resulting in duplicated effort, security gaps, and poor user experience across products.

Off-the-shelf solutions (Auth0, Okta, Cognito) create vendor lock-in, unpredictable per-MAU pricing at scale, and loss of control over sensitive identity data. This system replaces that dependency with an owned, extensible authentication service.

### Who Is Affected

| Stakeholder | Pain Today |
|---|---|
| End Users | Must create separate credentials per product; no SSO; poor password reset UX |
| Tenant Admins | No visibility into who is logged in; cannot revoke sessions; no exportable audit logs |
| Developers | Must re-implement auth in every new service; inconsistent security guarantees |
| Security / Compliance Teams | No central audit log; difficulty demonstrating GDPR/SOC2 compliance |
| Business Leaders | Vendor lock-in risk; unpredictable auth costs at scale |

---

## 2. Goals & Success Metrics

### Business Goals
- Eliminate per-MAU licensing costs associated with third-party auth vendors
- Reduce time-to-auth-integration for new internal services from weeks to days
- Achieve compliance posture sufficient for SOC2 Type II and GDPR audits

### Success Metrics

| Metric | Target (6 months post-launch) |
|---|---|
| New service integration time | ≤ 2 business days |
| Login latency (p95) | < 300ms |
| Authentication error rate (excluding user error) | < 0.1% |
| MFA adoption (admin users) | ≥ 60% |
| Security incidents from auth layer | 0 |
| Audit log completeness | 100% of auth events logged |
| Tenant onboarding time | < 30 minutes self-service |
| Developer satisfaction (survey, 1–5) | ≥ 4.2 |

---

## 3. User Personas

### Maya — End User
Regular user of applications protected by this system. Non-technical.
- **Needs:** Fast login, social OAuth, self-service password reset, clear error messages, optional MFA
- **Success:** Logs in without friction; never blocked from legitimate access

### Carlos — Tenant Administrator
Manages user access for his organization (one tenant). Moderate technical.
- **Needs:** User invite/disable/remove, role assignment, audit log access, MFA enforcement for his org, emergency lockout
- **Success:** Onboards a new employee in < 5 minutes; can prove to auditors who accessed what and when

### Priya — Developer / API Consumer
Builds services that delegate authentication to this system. High technical.
- **Needs:** Well-documented OAuth 2.0 endpoints, standard JWT tokens with configurable claims, clear error codes, sandbox tenant for testing
- **Success:** Integrates a new service using the docs alone in < half a day

### Jordan — Platform Super-Admin
Owns and operates the auth platform itself. High technical.
- **Needs:** Tenant provisioning API, cross-tenant audit logs, system health visibility, incident response tools
- **Success:** Provisions a new tenant in < 10 minutes; responds to security incidents with full audit data

---

## 4. Architectural Decisions (Resolved)

All open questions are now resolved. These decisions are **final** and must be treated as constraints by the Solution Architect and Developer agents.

### ADR-001: Tenant Isolation Strategy — Schema-per-Tenant
**Decision:** Each tenant gets its own PostgreSQL schema (e.g., `tenant_acme`, `tenant_beta`).
**Rationale:** Provides strong DB-level isolation with no risk of query-level data leakage, clean per-tenant backup and restore, and a clear compliance boundary for audits. Chosen over RLS (weaker isolation guarantee) and DB-per-tenant (too expensive at pilot scale).
**Consequences:**
- ✅ Isolation is architectural — impossible to leak cross-tenant data via a missing WHERE clause
- ✅ Easy to export, backup, or delete a single tenant's data (GDPR compliance)
- ✅ Clean audit boundary for SOC2
- ⚠️ All database migrations must run against every tenant schema — requires a migration runner
- ⚠️ DB connections must dynamically set `search_path` based on the `tenant_id` in the JWT
- ⚠️ Schema count grows with tenants — monitor PostgreSQL schema limits at scale
**Impact on backlog:** US-07 gains a schema creation task and migration runner subtask. US-08 is simpler (isolation is architectural, not query-level).

### ADR-002: Cross-Tenant User Identity — Single Tenant Only
**Decision:** One user belongs to exactly one tenant. There is no cross-tenant identity.
**Rationale:** Simplifies the data model significantly for the pilot. A `User` table lives entirely within the tenant's schema. No global identity table needed.
**Consequences:**
- ✅ Simple data model — no UserIdentity + TenantMembership split
- ✅ Reduces US-08 complexity
- ⚠️ If cross-tenant is needed in the future, it requires a data model migration (not retrofittable easily)
- ⚠️ A user who belongs to two real-world organizations must have two separate accounts

### ADR-003: Interface — API Only (No Hosted Login UI)
**Decision:** This system exposes only REST API endpoints. No hosted login page is provided. Consuming applications build their own UI.
**Rationale:** Keeps scope focused for the pilot. All four personas are technical enough to integrate via API.
**Consequences:**
- ✅ Removes 2–4 sprints of frontend UI scope
- ✅ Forces clean API-first design
- ⚠️ OAuth 2.0 Authorization Code flow still requires a `/oauth/authorize` redirect response and a minimal consent acknowledgment mechanism — implement as an API response (JSON), not a rendered HTML page
- ⚠️ Consuming applications are responsible for building their own login forms

### ADR-004: Token Strategy — JWT Access + Opaque Refresh
**Decision:** Short-lived JWT access tokens (15 min, RS256/ES256) + long-lived opaque refresh tokens stored hashed.
**Consequences:** Resource servers can validate access tokens locally. Refresh tokens support family-based revocation on reuse detection.

### ADR-005: Session Expiry — Configurable per Tenant, Default Absolute 24h
**Decision:** Session expiry is configurable per tenant. Default is absolute expiry at 24 hours maximum regardless of activity. Sliding expiry is opt-in.

### ADR-006: Social Login Account Linking — Verify Password Before Link
**Decision:** If a Google-authenticated email matches an existing password account, prompt the user to verify their password before linking the accounts.

### ADR-007: Audit Log Retention — 1 Year Hot + Cold Archive
**Decision:** 1-year hot retention in the primary database, then archive to cold storage (S3/equivalent). Satisfies SOC2 Type II evidence requirements.

### ADR-008: SOC2 — Post-Launch Goal
**Decision:** SOC2 Type II certification is a post-launch goal. Evidence collection starts at launch. A compliance consultant will be engaged by Sprint 2.

### ADR-009: Secrets Management — Dedicated Secrets Manager
**Decision:** All signing keys, client secrets, and database credentials are stored in a dedicated secrets manager (HashiCorp Vault or cloud equivalent). Never in environment variables, code, or committed config files.

---

## 5. Feature List — MoSCoW (Final)

### Must Have — 18 Features (MVP)

| # | Feature | Notes |
|---|---|---|
| M1 | User Registration (email + password) | Per-tenant password complexity config |
| M2 | Email Verification Flow | Token-based, time-limited, single-use, with resend |
| M3 | Login (email + password) | Argon2id hashing; returns JWT + opaque refresh token |
| M4 | Password Reset Flow | Email-based, time-limited, single-use token |
| M5 | Session Management | JWT (15 min) + refresh token rotation; configurable TTL per tenant |
| M6 | Multi-Tenant Data Isolation | Schema-per-tenant; schema created at provisioning |
| M7 | Tenant Provisioning API | Creates schema, default roles, and API credentials |
| M8 | RBAC — Core | Define roles; assign roles to users within a tenant |
| M9 | OAuth 2.0 — Authorization Code + PKCE | Auth server; PKCE S256 mandatory; API response (no hosted UI) |
| M10 | OAuth 2.0 — Client Credentials (M2M) | Service-to-service tokens; scope-bound |
| M11 | OAuth 2.0 — External IdP (Google) | Social login; verify-before-link for existing accounts |
| M12 | Logout / Session Revocation | Single-device and all-devices; opaque refresh token denylist |
| M13 | Audit Log | Append-only; per-tenant; tenant admin read-only access |
| M14 | Rate Limiting + Brute Force Protection | Per-IP and per-user; Redis-backed; configurable per tenant |
| M15 | HTTPS + Secure Headers | HSTS, CSP, X-Frame-Options, X-Content-Type-Options |
| M16 | Admin User Management API | Invite, disable, delete users within a tenant |
| M17 | Token Introspection Endpoint | RFC 7662 — for resource servers |
| M18 | JWKS Endpoint | Public key publication for JWT verification |

### Should Have

| # | Feature |
|---|---|
| S1 | MFA — TOTP (Authenticator app) |
| S2 | Per-Tenant MFA Enforcement (admin can require MFA for all users) |
| S3 | OAuth 2.0 — Microsoft/Entra ID as external IdP |
| S4 | RBAC — Permission-level granularity (roles map to fine-grained permissions) |
| S5 | Self-Service User Profile (update name, email, password, MFA) |
| S6 | Tenant Admin Dashboard (minimal API client, not a full UI) |
| S7 | Developer Documentation Site |
| S8 | Webhook Notifications (auth events pushed to tenant-configured URLs) |
| S9 | Session Activity API (user can list and revoke active sessions) |
| S10 | Account Lockout with Admin Unlock |

### Could Have

| # | Feature |
|---|---|
| C1 | MFA — SMS/OTP |
| C2 | Magic Link / Passwordless Login |
| C3 | OAuth 2.0 — Device Authorization Grant (CLI/IoT) |
| C4 | Additional IdPs (GitHub, SAML, Okta) |
| C5 | Custom Email Templates per Tenant |
| C6 | IP Allowlist/Denylist per Tenant |
| C7 | Admin Impersonation with Audit Trail |
| C8 | Self-Service Tenant Onboarding UI |
| C9 | Passkey / WebAuthn Support |
| C10 | SDK Libraries (Go, TypeScript) |
| C11 | SCIM 2.0 Provisioning |

### Won't Do (This Pilot)

| # | Feature | Rationale |
|---|---|---|
| W1 | SAML 2.0 | Significant spec complexity; enterprise phase only |
| W2 | Full IAM / Policy Engine (OPA, Casbin) | RBAC is sufficient for pilot |
| W3 | Billing / Subscription Management | Out of auth domain |
| W4 | User Behavior Analytics / Risk-Based Auth | Separate data platform concern |
| W5 | Hosted Login Pages | API-only per ADR-003 |
| W6 | Multi-Region Active-Active | Single region for pilot |
| W7 | Native Mobile SDKs | Post-MVP |
| W8 | LDAP / Kerberos / WS-Federation | Legacy protocol; out of scope |
| W9 | Cross-Tenant User Identities | Single-tenant users per ADR-002 |

---

## 6. User Stories — Final Backlog

All stories are INVEST-validated. Story points use Fibonacci scale. Estimates reflect final architectural decisions (schema-per-tenant, API-only, single-tenant users).

---

### Epic 1: User Registration & Email Verification

**US-01 — User Self-Registration** `5 pts`
```
As Maya,
I want to register a new account with my email and password,
So that I can access the application for my tenant.
```
- Password complexity is configurable per tenant (min length, uppercase, number, special char)
- Account status on creation: `unverified`
- Triggers email verification (US-02)
- Response must never contain hashed password or any credential data

---

**US-02 — Email Verification** `5 pts` *(includes resend)*
```
As Maya,
I want to receive a verification email after registration,
So that my account is confirmed and I can log in.
```
- Token: single-use, time-limited (24h default)
- Expired token response must offer resend option
- Resend invalidates the previous token
- Resend endpoint returns 200 regardless of whether email exists (no enumeration)

---

### Epic 2: Authentication

**US-03 — User Login** `5 pts`
```
As Maya,
I want to log in with my email and password,
So that I receive a JWT access token and refresh token.
```
- Returns: `access_token` (JWT, RS256, 15 min) + `refresh_token` (opaque, hashed in DB)
- JWT claims: `sub`, `iss`, `aud`, `exp`, `iat`, `tenant_id`, `roles[]`
- Tenant routing: login request scoped to tenant (via subdomain, header, or request body field — SA to decide)
- Must increment failed login counter (feeds US-14)
- Error message must NOT distinguish "wrong email" vs "wrong password"
- Writes `LOGIN_SUCCESS` or `LOGIN_FAILURE` to audit log

---

**US-04 — Session Refresh** `3 pts`
```
As Maya,
I want my session transparently refreshed via a refresh token,
So that I am not forced to re-login during active use.
```
- Each use of a refresh token issues a new one and invalidates the old (rotation)
- If a previously rotated (old) token is presented: revoke entire token family + write `SUSPICIOUS_TOKEN_REUSE` to audit log
- Configurable per tenant: absolute expiry (default 24h max) or sliding

---

**US-05 — Logout / Session Revocation** `5 pts`
```
As Maya,
I want to log out from my current device or all devices,
So that my session is immediately invalidated.
```
- Single-device: invalidate the specific refresh token presented
- All-devices: invalidate all refresh tokens for the user
- Access tokens remain technically valid until 15-min TTL expiry (documented trade-off)
- Idempotent: logout with already-expired token returns 200
- Writes `LOGOUT` or `LOGOUT_ALL` to audit log

---

**US-06 — Password Reset Flow** `3 pts`
```
As Maya,
I want to request a password reset email when I forget my password,
So that I can regain access without contacting support.
```
- `/forgot-password` response is ALWAYS identical regardless of whether email exists
- Response timing must be statistically indistinguishable for known vs unknown emails
- Reset token: single-use, 1-hour TTL
- On successful reset: all existing sessions (refresh tokens) revoked
- Writes `PASSWORD_RESET_REQUESTED` and `PASSWORD_RESET_COMPLETED` to audit log

---

### Epic 3: Multi-Tenant Support

**US-07a — Tenant Provisioning API** `5 pts`
```
As Jordan,
I want to provision a new tenant via API,
So that a new organization gets an isolated schema and default configuration.
```
- Creates: tenant record in global registry + dedicated PostgreSQL schema
- Creates default roles within tenant schema: `admin`, `user`
- Sends invitation email to the provided admin email address
- Rejects duplicate tenant names with 409
- Only callable by platform super-admins

---

**US-07b — Per-Tenant Migration Runner** `5 pts`
```
As the platform,
I want all tenant schemas to stay in sync with the application schema version,
So that new features are available to all tenants without manual intervention.
```
- A migration runner applies pending migrations to every tenant schema on deployment
- Maintains a `schema_migrations` table within each tenant schema
- Migrations are idempotent and transactional
- Failed migration for one tenant does not block others (log + alert)
- Developer tooling: CLI command to run migrations for a specific tenant or all tenants

---

**US-07c — Tenant API Credentials** `3 pts`
```
As Jordan,
I want each tenant to receive a client_id and client_secret at provisioning,
So that the tenant's services can authenticate with the platform for M2M flows.
```
- `client_id` is public; `client_secret` is shown once and stored hashed
- Secret rotation endpoint available
- Scoped to the tenant's schema

---

**US-08a — Schema-Routing Middleware** `5 pts`
```
As the platform,
I want every authenticated database query automatically routed to the correct tenant schema,
So that cross-tenant data leakage is architecturally impossible.
```
- Go middleware extracts `tenant_id` from verified JWT
- Sets PostgreSQL `search_path` to the tenant's schema for the duration of the request
- Unauthenticated endpoints use a global schema only
- Logs and rejects any request where `tenant_id` cannot be resolved to a valid schema

---

**US-08b — Automated Cross-Tenant Isolation Test Suite** `5 pts`
```
As the platform,
I want an automated test suite that attempts cross-tenant data access,
So that any regression in tenant isolation is caught before it reaches production.
```
- Tests attempt to access tenant B's data using tenant A's valid credentials
- Tests cover: users, roles, audit logs, OAuth clients, session tokens
- Suite runs in CI on every pull request
- Any cross-tenant data returned in a response = immediate CI failure
- Tests are part of the Definition of Done for every data-access story

---

### Epic 4: Role-Based Access Control (RBAC)

**US-09 — Assign Role to User** `3 pts`
```
As Carlos,
I want to assign and unassign predefined roles to users in my tenant via API,
So that I can control what each user is authorized to do.
```
- Roles are tenant-scoped strings (e.g., `admin`, `user`, custom roles)
- A user can hold multiple roles within a tenant
- Role assignment/unassignment writes to audit log
- The auth system **issues** role claims in tokens; it does not **enforce** permissions on resources (downstream APIs own enforcement)

---

**US-10 — Role and Tenant Claims in JWT** `2 pts`
```
As Priya,
I want the JWT to include tenant_id and roles[] as standard claims,
So that my service can enforce authorization without additional API calls.
```
- Standard claims: `sub`, `iss`, `aud`, `exp`, `iat`, `tenant_id`, `roles[]`
- Token size: flag to engineering if user has > 20 roles (potential size issue)
- Custom claims per tenant: Could Have (not this story)

---

### Epic 5: OAuth 2.0 — Authorization Server

**US-11a — OAuth Client Registration API** `3 pts`
```
As Priya,
I want to register an OAuth 2.0 client for my application,
So that I can integrate using the Authorization Code flow.
```
- Registers: `client_id`, `client_secret` (hashed), allowed `redirect_uris[]`, allowed `scopes[]`
- Redirect URI strict matching (no wildcard, no partial match)
- Scoped to a tenant (a client belongs to one tenant)

---

**US-11b — `/oauth/authorize` Endpoint** `3 pts` *(API-only, no UI)*
```
As Priya,
I want my application to redirect users to the /oauth/authorize endpoint,
So that users are authenticated and an authorization code is returned.
```
- Validates: `client_id`, `redirect_uri`, `response_type=code`, `state`, `code_challenge` (PKCE)
- On valid request: returns JSON response with authorization code (consuming app handles redirect)
- `state` parameter is mandatory (CSRF prevention)
- Authorization code: single-use, 10-minute TTL
- On invalid `redirect_uri` or `client_id`: return error to caller, NOT redirect (prevents open redirect)

---

**US-11c — `/oauth/token` Endpoint — Code Exchange + PKCE** `5 pts`
```
As Priya,
I want to exchange an authorization code for an access token,
So that my application can act on behalf of the authenticated user.
```
- Validates: `code`, `code_verifier` (PKCE S256 — plain method rejected), `client_id`, `redirect_uri`
- Code must be single-use; second use returns 400 and revokes any tokens issued from that code
- Returns: `access_token` (JWT), `refresh_token` (opaque), `token_type`, `expires_in`, `scope`
- PKCE S256 is mandatory even for confidential clients

---

**US-12 — Client Credentials Grant (M2M)** `5 pts`
```
As Priya,
I want my backend service to obtain tokens using its client credentials,
So that service-to-service calls are authenticated without a user context.
```
- Token does not contain `sub` or user roles — represents a service identity
- Token contains `client_id`, `scope`, `tenant_id`
- Client secret stored hashed; never returned after initial registration
- Invalid scope rejected with 400

---

### Epic 6: External Identity Provider

**US-13 — Social Login via Google** `8 pts`
```
As Maya,
I want to log in with my Google account,
So that I do not need to create or remember a separate password.
```
- Per-tenant Google client ID/secret configuration (Jordan/Carlos sets this up)
- OIDC discovery endpoint used for IdP metadata
- `state` parameter mandatory (CSRF prevention)
- Account linking: if Google email matches existing password account → prompt user to verify password before linking (ADR-006)
- New user via Google: account created with `verified` status, no password set
- Returns same JWT + refresh token as standard login

---

### Epic 7: Security & Reliability

**US-14 — Rate Limiting + Account Lockout** `5 pts`
```
As the platform,
I want to lock out accounts and rate-limit IPs after repeated failed login attempts,
So that brute force and credential stuffing attacks are mitigated.
```
- Per-tenant configurable: N failed attempts before lockout, lockout duration
- Counters: per user+tenant (account lockout) AND per IP (rate limit) — both required
- Redis-backed distributed counter
- Locked account response does NOT reveal the lockout reason to the attacker
- Writes `ACCOUNT_LOCKED` to audit log

---

**US-15 — Audit Log** `5 pts`
```
As Carlos,
I want every authentication event for my tenant logged with timestamp, user, IP, and outcome,
So that I can investigate incidents and satisfy compliance auditors.
```
- Events logged: `LOGIN_SUCCESS`, `LOGIN_FAILURE`, `LOGOUT`, `LOGOUT_ALL`, `PASSWORD_RESET_REQUESTED`, `PASSWORD_RESET_COMPLETED`, `PASSWORD_CHANGED`, `EMAIL_VERIFIED`, `ACCOUNT_LOCKED`, `TOKEN_REFRESHED`, `SUSPICIOUS_TOKEN_REUSE`, `ROLE_ASSIGNED`, `ROLE_UNASSIGNED`, `USER_INVITED`, `USER_DISABLED`
- Append-only: no UPDATE or DELETE at the application layer
- Tenant admin: read-only access, scoped to own tenant only
- Retention: 1-year hot, then cold archive (ADR-007)

---

### Backlog Summary

| Story | Description | Points |
|---|---|---|
| US-01 | User Registration | 5 |
| US-02 | Email Verification + Resend | 5 |
| US-03 | User Login | 5 |
| US-04 | Session Refresh + Rotation | 3 |
| US-05 | Logout / Session Revocation | 5 |
| US-06 | Password Reset Flow | 3 |
| US-07a | Tenant Provisioning API + Schema Creation | 5 |
| US-07b | Per-Tenant Migration Runner | 5 |
| US-07c | Tenant API Credentials | 3 |
| US-08a | Schema-Routing Middleware | 5 |
| US-08b | Cross-Tenant Isolation Test Suite | 5 |
| US-09 | Assign / Unassign Roles | 3 |
| US-10 | Role + Tenant Claims in JWT | 2 |
| US-11a | OAuth Client Registration API | 3 |
| US-11b | /oauth/authorize Endpoint | 3 |
| US-11c | /oauth/token Code Exchange + PKCE | 5 |
| US-12 | Client Credentials Grant (M2M) | 5 |
| US-13 | Social Login via Google | 8 |
| US-14 | Rate Limiting + Account Lockout | 5 |
| US-15 | Audit Log | 5 |
| **Total Must Have** | | **~89 pts** |

---

## 7. Acceptance Criteria (Key Stories)

### US-01: User Registration
```gherkin
Feature: User Registration

  Scenario: Successful registration
    Given I am a new user on tenant "acme-corp"
    When I POST /auth/register with valid email and password meeting tenant complexity rules
    Then I receive 201 Created
    And my account has status "unverified"
    And a verification email is sent
    And the response body contains no password hash or credential data

  Scenario: Duplicate email rejected
    Given "maya@example.com" already exists in tenant "acme-corp"
    When I register with "maya@example.com"
    Then I receive 409 Conflict
    And the response does not reveal the account status (active, locked, etc.)

  Scenario: Weak password rejected
    When I register with a password that fails tenant complexity rules
    Then I receive 422 Unprocessable Entity
    And the error describes which specific requirements were not met

  Scenario: Invalid email format rejected
    When I register with "not-an-email"
    Then I receive 422 Unprocessable Entity
```

### US-03: User Login
```gherkin
Feature: User Login

  Scenario: Successful login
    Given I have an active verified account in tenant "acme-corp"
    When I POST /auth/login with correct email and password
    Then I receive 200 OK
    And the response contains "access_token" (signed JWT, RS256, 15 min expiry)
    And the response contains "refresh_token" (opaque string)
    And the JWT payload contains: sub, iss, aud, exp, iat, tenant_id, roles[]
    And a LOGIN_SUCCESS event is written to the audit log

  Scenario: Wrong password returns ambiguous error
    When I POST /auth/login with an incorrect password
    Then I receive 401 Unauthorized
    And the error message does NOT distinguish "wrong email" vs "wrong password"
    And a LOGIN_FAILURE event is written to the audit log

  Scenario: Cross-tenant login rejected
    Given "maya@example.com" exists in tenant "acme-corp" but not in "beta-corp"
    When I POST /auth/login with tenant "beta-corp"
    Then I receive 401 Unauthorized and no tokens are issued

  Scenario: Unverified account cannot log in
    Given my account has status "unverified"
    When I POST /auth/login with correct credentials
    Then I receive 403 Forbidden indicating email is not verified
```

### US-04: Session Refresh
```gherkin
Feature: Token Refresh

  Scenario: Valid refresh token issues new token pair
    Given I have a valid refresh token "rt_001"
    When I POST /auth/token/refresh with "rt_001"
    Then I receive 200 OK with new access_token and new refresh_token "rt_002"
    And "rt_001" is immediately invalidated

  Scenario: Rotated token reuse triggers family revocation
    Given I used "rt_001" and received "rt_002"
    When I POST /auth/token/refresh with old token "rt_001"
    Then I receive 401 Unauthorized
    And both "rt_001" and "rt_002" are revoked
    And a SUSPICIOUS_TOKEN_REUSE event is logged
```

### US-06: Password Reset (Anti-Enumeration)
```gherkin
Feature: Password Reset

  Scenario: Reset request always returns identical response
    When I POST /auth/forgot-password with any email address (registered or not)
    Then I always receive 200 with "If this email is registered, you will receive a reset link"
    And the response time is statistically indistinguishable regardless of whether the email exists

  Scenario: Successful reset revokes all sessions
    Given I have a valid reset token
    When I POST /auth/reset-password with the valid token and a new strong password
    Then I receive 200 OK
    And all existing refresh tokens for my account are revoked
    And the reset token is single-use and immediately invalidated

  Scenario: Expired or reused token rejected
    When I POST /auth/reset-password with an expired or already-used token
    Then I receive 400 Bad Request
    And my password is not changed
```

### US-07a: Tenant Provisioning
```gherkin
Feature: Tenant Provisioning

  Scenario: Super-admin provisions a new tenant
    Given I am authenticated as a platform super-admin
    When I POST /admin/tenants with name "acme-corp" and admin_email "carlos@acme.com"
    Then I receive 201 Created
    And a new PostgreSQL schema "tenant_acme_corp" is created
    And default migrations are applied to the new schema
    And default roles "admin" and "user" are created within the schema
    And client_id and client_secret are returned (secret shown once)
    And an invitation email is sent to "carlos@acme.com"

  Scenario: Duplicate tenant name rejected
    Given tenant "acme-corp" already exists
    When I POST /admin/tenants with name "acme-corp"
    Then I receive 409 Conflict and no schema is created

  Scenario: Non-super-admin cannot provision
    Given I am a tenant admin (not super-admin)
    When I POST /admin/tenants
    Then I receive 403 Forbidden
```

### US-08b: Cross-Tenant Isolation
```gherkin
Feature: Multi-Tenant Data Isolation

  Scenario: User cannot access another tenant's data
    Given Maya has a valid JWT for tenant "acme-corp"
    When Maya calls any endpoint with a valid "acme-corp" token
    Then she can never receive data belonging to "beta-corp" in any response field

  Scenario: Admin cannot query another tenant's users
    Given Carlos is a tenant admin for "acme-corp"
    When Carlos calls GET /admin/users with his "acme-corp" token
    Then he receives only users from "acme-corp"
    And the query is automatically scoped to the "tenant_acme_corp" schema

  Scenario: Automated isolation test suite
    Given the CI cross-tenant isolation suite is executed
    Then every attempt to access tenant B data using tenant A credentials returns 403
    And no tenant B data appears in any response body, error message, or header
```

### US-11c: OAuth PKCE Token Exchange
```gherkin
Feature: OAuth 2.0 Token Exchange

  Scenario: Valid code exchange with PKCE
    Given a valid authorization code "code_abc" with code_challenge based on verifier "xyz"
    When I POST /oauth/token with code "code_abc" and code_verifier "xyz"
    Then I receive 200 with access_token, refresh_token, token_type, expires_in, scope

  Scenario: Invalid code_verifier rejected
    When I POST /oauth/token with correct code but wrong code_verifier
    Then I receive 400 Bad Request — PKCE validation failure

  Scenario: Authorization code cannot be reused
    Given I already exchanged code "code_abc" for tokens
    When I attempt to exchange "code_abc" again
    Then I receive 400 Bad Request
    And any tokens issued from that code are revoked

  Scenario: Plain PKCE method rejected
    When I initiate an authorization request with code_challenge_method=plain
    Then I receive 400 Bad Request — only S256 is accepted
```

---

## 8. Non-Functional Requirements

### Security

| ID | Requirement |
|---|---|
| SEC-01 | Passwords hashed with Argon2id (preferred) or bcrypt cost ≥ 12. MD5/SHA-1/plaintext never acceptable. |
| SEC-02 | JWT signed with RS256 or ES256. HS256 not acceptable in multi-tenant contexts. Keys rotatable without restart. |
| SEC-03 | Access token TTL: 15 minutes maximum. |
| SEC-04 | Refresh tokens: opaque, stored hashed, rotated on every use, revocable by family. |
| SEC-05 | TLS 1.2 minimum, TLS 1.3 preferred. HSTS min-age 31,536,000s. No HTTP fallback in production. |
| SEC-06 | All responses include: `Strict-Transport-Security`, `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `Content-Security-Policy`, `Referrer-Policy: no-referrer`. |
| SEC-07 | All secrets in dedicated secrets manager (ADR-009). Never in env vars, code, or committed config. |
| SEC-08 | `state` parameter mandatory on all OAuth 2.0 flows. `SameSite=Strict` on session cookies. |
| SEC-09 | All inputs validated server-side. Parameterized queries only (no string-concatenated SQL). |
| SEC-10 | PKCE S256 mandatory for all Authorization Code flows. Plain method rejected. |

### Performance

| ID | Requirement |
|---|---|
| PERF-01 | Login endpoint: p95 < 300ms at 1,000 concurrent sessions |
| PERF-02 | Token issuance (all flows): p95 < 100ms |
| PERF-03 | Token introspection: p95 < 50ms |
| PERF-04 | Database queries: no full table scans in hot paths; indexes required on `email`, `tenant_id`, `refresh_token_hash` |
| PERF-05 | Rate limiting check: p95 < 5ms (Redis pipeline) |

### Availability & Reliability

| ID | Requirement |
|---|---|
| AVAIL-01 | Uptime: 99.9% (≤ 8.7h downtime/year) |
| AVAIL-02 | Graceful degradation: if Redis unavailable, fail closed on rate limiting (block, don't bypass) |
| AVAIL-03 | Database connection pooling; pool exhaustion must return 503 not hang |
| AVAIL-04 | Alerting on: error rate > 1%, p95 latency > 500ms, failed login rate spike > 10x baseline |

### Compliance

| ID | Requirement |
|---|---|
| COMP-01 | GDPR: right-to-erasure API endpoint — deletes user PII across tenant schema; audit log entries anonymized (user ID replaced with tombstone) |
| COMP-02 | SOC2 CC6.2: all authentication events logged with user, timestamp, IP, outcome |
| COMP-03 | SOC2 CC6.3: access revocation (logout, disable) takes effect within one access token TTL (15 min) |
| COMP-04 | Audit logs: append-only at application layer; 1-year retention (ADR-007) |

---

## 9. Out of Scope (Final)

| # | Item | Rationale |
|---|---|---|
| OS-01 | SAML 2.0 | Enterprise phase only |
| OS-02 | Fine-grained Policy Engine (OPA/Casbin) | RBAC sufficient for pilot |
| OS-03 | Billing / Subscription Management | Outside auth domain |
| OS-04 | User Behavior Analytics / Risk-Based Auth | Separate data platform |
| OS-05 | Hosted Login Pages | API-only per ADR-003 |
| OS-06 | Multi-Region Active-Active | Single region for pilot |
| OS-07 | Native Mobile SDKs | Post-MVP |
| OS-08 | LDAP / Kerberos / WS-Federation | Legacy protocols |
| OS-09 | Cross-Tenant User Identities | Single-tenant per ADR-002 |
| OS-10 | HSM Integration | Secrets Manager sufficient for pilot |
| OS-11 | SCIM 2.0 Provisioning | Post-MVP |
| OS-12 | White-labeled / Branded Login Pages | API-only per ADR-003 |

---

## 10. Sprint Plan (Refined)

**Team:** 3 backend engineers, 0.5 frontend (OAuth consent minimal), 1 QA, 0.5 DevOps (shared)
**Velocity:** ~28–32 points/sprint | **Sprint:** 2 weeks

---

### Sprint 0 — Architecture Spike (Weeks 1–2)
*No production code. All blocking decisions made here.*

| Task | Owner |
|---|---|
| Design global registry schema + tenant schema template (User, Role, Session, AuditLog, OAuthClient) | Architect |
| Design schema-routing middleware approach for Go | Engineering Lead |
| Design migration runner strategy for multi-schema PostgreSQL | Engineering Lead |
| Provision PostgreSQL + Redis in staging | DevOps |
| Set up secrets manager (ADR-009) | DevOps + Infra |
| Set up CI/CD pipeline, branch strategy, code repo | DevOps |
| Set up Go monorepo structure (backend) + Next.js minimal shell (frontend) | Engineering Lead |
| Write ADRs 001–009 (all resolved — just needs formal documentation) | Architect |
| Agree on API versioning strategy and error response format | Engineering Lead |
| Schedule external penetration test (book now for Sprint 6/7 window) | PM |

**Deliverable:** ERD, migration runner POC, CI/CD green, staging environment ready, all ADRs documented.

---

### Sprint 1 — Core Auth Foundation (Weeks 3–4)
*Users can register, verify email, log in, reset password.*

| Story | Points |
|---|---|
| US-01: User Registration | 5 |
| US-02: Email Verification + Resend | 5 |
| US-03: User Login (JWT + refresh token) | 5 |
| US-06: Password Reset Flow | 3 |
| Email service integration (async, transactional) | 3 |
| **Sprint Total** | **21** |

**Risk:** Email send infrastructure must exist. Password complexity config requires a tenant config namespace — use a global default for Sprint 1; per-tenant config delivered with US-07.

---

### Sprint 2 — Sessions, Security, Audit (Weeks 5–6)
*Sessions are secure; brute force protection live; audit log running.*

| Story | Points |
|---|---|
| US-04: Session Refresh + Rotation | 3 |
| US-05: Logout / Session Revocation (single + all devices) | 5 |
| US-14: Rate Limiting + Account Lockout (Redis) | 5 |
| US-15: Audit Log (event capture + read API) | 5 |
| M15: HTTPS Enforcement + Secure Headers | 2 |
| **Sprint Total** | **20** |

**Risk:** Redis must be available in staging. Default lockout thresholds require product sign-off before sprint starts.

---

### Sprint 3 — Multi-Tenancy Foundation (Weeks 7–8)
*Tenant isolation is architectural and tested.*

| Story | Points |
|---|---|
| US-07a: Tenant Provisioning API + Schema Creation | 5 |
| US-07b: Per-Tenant Migration Runner | 5 |
| US-07c: Tenant API Credentials | 3 |
| US-08a: Schema-Routing Middleware | 5 |
| US-08b: Cross-Tenant Isolation Test Suite | 5 |
| **Sprint Total** | **23** |

**Risk:** Highest-risk sprint. If Sprint 0 migration runner POC is incomplete, this sprint slips. US-08b (isolation test suite) is **non-negotiable** — it is the safety net for everything that follows. Do not defer it.

---

### Sprint 4 — RBAC, Claims, Admin API (Weeks 9–10)
*Roles assigned; tokens carry correct claims; admins can manage users.*

| Story | Points |
|---|---|
| US-09: Assign / Unassign Roles | 3 |
| US-10: Role + Tenant Claims in JWT | 2 |
| M16: Admin User Management API (invite, disable, delete) | 5 |
| M17: Token Introspection Endpoint (RFC 7662) | 3 |
| M18: JWKS Endpoint | 2 |
| **Sprint Total** | **15** |

**Note:** Lower point total is intentional — provides buffer for Sprint 3 overflow and time for end-to-end integration testing of the full auth flow.

---

### Sprint 5 — OAuth 2.0 Authorization Server (Weeks 11–12)
*Applications can integrate using Authorization Code + PKCE.*

| Story | Points |
|---|---|
| US-11a: OAuth Client Registration API | 3 |
| US-11b: /oauth/authorize Endpoint | 3 |
| US-11c: /oauth/token Exchange + PKCE Validation | 5 |
| **Sprint Total** | **11** |

**Note:** Reduced from original estimate (from 13 to 11) because no hosted consent UI is required (API-only, ADR-003). **Strong recommendation:** use a battle-tested OAuth 2.0 library as the foundation — do not implement the spec from scratch. Timebox strictly; OAuth edge cases are a consistent source of overrun.

---

### Sprint 6 — OAuth M2M + Google Social Login (Weeks 13–14)
*M2M tokens work; users can log in with Google.*

| Story | Points |
|---|---|
| US-12: Client Credentials Grant (M2M) | 5 |
| US-13: Social Login via Google | 8 |
| **Sprint Total** | **13** |

**Risk:** Google OAuth integration requires a working Google Cloud project with credentials — set this up before Sprint 6 starts. Account linking (verify password before linking) adds ~3 points if complex; included in the 8-point estimate.

---

### Sprint 7 — Should Haves: MFA (Weeks 15–16)
*TOTP MFA available; tenant admins can enforce it.*

| Story | Points |
|---|---|
| S1: MFA — TOTP Setup and Verification | 8 |
| S2: Per-Tenant MFA Enforcement | 3 |
| S5: Self-Service User Profile API | 5 |
| **Sprint Total** | **16** |

**Note:** Admin Dashboard UI (S6) deferred — API-only scope means this is lower priority. S1 + S2 (MFA) are strongly recommended for production readiness and should be protected from deferral.

---

### Sprint 8 — Hardening + Compliance (Weeks 17–18)
*System is production-ready: pentest clean, GDPR endpoint, performance validated.*

| Work Item | Points |
|---|---|
| GDPR Right-to-Erasure API endpoint (COMP-01) | 3 |
| Pentest finding remediation (findings from Sprint 6/7 test) | 8 |
| OWASP Top 10 review + gap remediation | 3 |
| Performance testing against PERF-01–PERF-05; remediation | 5 |
| Monitoring, alerting, dashboards (AVAIL-04 thresholds) | 3 |
| Developer integration documentation | 3 |
| **Sprint Total** | **25** |

**Risk:** Pentest findings are unknown — 8 points for remediation is an estimate. If Critical findings are discovered, this sprint may extend. **Do not wait until Sprint 8 to engage the penetration tester — book them in Sprint 0 for a Sprint 6/7 window.**

---

### Sprint 9 — Launch Preparation (Weeks 19–20)
*UAT signed off; system validated end-to-end in production-equivalent environment.*

| Work Item | Points |
|---|---|
| End-to-end integration testing (all Must Have flows) | 5 |
| Staging → production-equivalent environment validation | 3 |
| Incident response runbook | 2 |
| Go/no-go criteria defined and validated | 2 |
| Smoke tests on production environment | 3 |
| **Sprint Total** | **15** |

---

### Capacity Summary

| Sprint | Theme | Points | Cumulative |
|---|---|---|---|
| 0 | Architecture Spike | — | — |
| 1 | Core Auth Foundation | 21 | 21 |
| 2 | Sessions, Security, Audit | 20 | 41 |
| 3 | Multi-Tenancy Foundation | 23 | 64 |
| 4 | RBAC, Claims, Admin API | 15 | 79 |
| 5 | OAuth 2.0 Authorization Server | 11 | 90 |
| 6 | OAuth M2M + Google Login | 13 | 103 |
| 7 | MFA + User Profile | 16 | 119 |
| 8 | Hardening + Compliance | 25 | 144 |
| 9 | Launch Prep + UAT | 15 | 159 |
| **Total** | | **~159 pts** | **~20 weeks / 5 months** |

---

## 11. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Migration runner more complex than estimated (Sprint 3) | Medium | High | Sprint 0 spike must produce a working POC, not just a design |
| OAuth 2.0 edge cases cause Sprint 5 overrun | High | Medium | Use battle-tested OAuth library; strict timebox |
| Cross-tenant data leak discovered in Sprint 3 | Low | Critical | US-08b isolation test suite is non-negotiable DoD item |
| Pentest uncovers Critical findings | Medium | High | Book pentest now; target Sprint 6/7 window; do not wait for Sprint 8 |
| "Can we add SAML?" scope creep request | High | Medium | Reference OS-01 explicitly; formal scope change with re-estimation required |
| Google OAuth API changes during development | Low | Medium | Pin to OIDC discovery; monitor Google changelog |
| Refresh token rotation security edge cases underestimated | Medium | High | Designate auth security champion; require RFC 6749, 7009, 7519, 7636 reading |
| Schema-per-tenant migration tooling underestimated | Medium | High | Sprint 0 POC is mandatory; do not skip |

---

## 12. Definition of Done (Global)

All stories must satisfy these criteria before being considered complete:

- [ ] Code reviewed and approved by ≥ 1 peer
- [ ] Unit tests written and passing (≥ 80% coverage on new code)
- [ ] Integration tests written and passing
- [ ] US-08b cross-tenant isolation test suite passes (for any data-access story)
- [ ] Security checklist reviewed (no hardcoded secrets, inputs validated, SQL parameterized)
- [ ] Deployed to staging successfully
- [ ] QA sign-off in staging environment
- [ ] API documentation updated
- [ ] Audit log event written and verified (for any auth action story)

---

*End of Document — Authentication System PRD v1.1*
*All open questions resolved. Ready for handoff to: Project Manager, Solution Architect, Developer, Tester agents.*
