# Product Requirements Document
# TGX Auth Platform — Version 2.0

**Version:** 2.0
**Date:** 2026-03-03
**Status:** 🔵 In Review
**Author:** Product Owner
**Supersedes:** prd.md v1.1 (2026-02-27)
**Consumers:** Engineering, Design, QA, Recruitment Team

---

## Revision History

| Version | Date | Change |
|---------|------|--------|
| 1.0 | 2026-02-27 | Initial brainstorm draft |
| 1.1 | 2026-02-27 | All open questions resolved; backlog refined |
| 2.0 | 2026-03-03 | Post-deployment update: Auth Admin UI added; Recruitment integration planned; bootstrap bug fixed; TGX design system applied |

---

## Executive Summary

The TGX Auth Platform is Tigersoft's self-hosted, multi-tenant authentication and identity service — the SSO backbone for all V4 modules (Recruitment, Payroll, Performance Management, etc.).

**Phase 1 (Auth API)** is **complete and live** at `https://auth.tgstack.dev`.

**Phase 2 (Auth Admin UI)** is the next build: a web console for platform operators and tenant administrators to manage tenants, users, roles, settings, and audit logs — without touching an API or database.

**Phase 3 (Recruitment Integration)** establishes the contract between the Auth platform and the Recruitment module — the first V4 consumer of the SSO system.

---

## 1. Problem Statement

### 1.1 What was solved (Phase 1 — Auth API ✅)

Before the Auth system, every Tigersoft product managed its own credentials, session state, and audit trail. This produced security gaps, duplicated effort, and no path to PDPA compliance.

The Auth API resolves this by providing:
- Standards-compliant OAuth 2.0 + RS256 JWT with PKCE
- Schema-per-tenant PostgreSQL isolation
- RBAC, MFA (TOTP), GDPR-compliant erasure
- Machine-to-machine (M2M) client credentials
- Full audit trail for every identity event

### 1.2 What remains unsolved (Phase 2 — Admin UI ⬜)

All tenant and user management currently requires direct API calls or database access. Platform operators cannot:
- Provision a new client company from a UI
- See which tenants exist, their status, or their module assignments
- Invite, disable, or reassign roles to users without writing `curl` commands
- View audit logs without querying the database

There is no self-service path for tenant administrators to manage their own users.

### 1.3 New requirement (Phase 3 — Recruitment Integration ⬜)

The Recruitment module is the first V4 module to consume the Auth platform. It introduces two new identity patterns that the current Auth system must explicitly support:
- **Internal HR users** — authenticate per company (per-tenant JWT)
- **Applicants** — centralized identity shared across all Tigersoft client companies (platform-tenant JWT)

---

## 2. Goals & Success Metrics

### Business goals

| Goal | Measure |
|------|---------|
| Zero manual provisioning steps | A new Recruitment client can be fully onboarded from the Admin UI alone, in < 15 minutes |
| Recruitment team unblocked | Recruitment backend can verify Auth JWTs and call Auth APIs from Day 1 of development |
| Compliance | Full PDPA audit trail for all identity events; applicant data deletion on request |
| Scalability | System supports 50+ concurrent tenants without configuration changes |
| Brand consistency | Admin UI matches TGX design system exactly |

### Success metrics

| Metric | Target |
|--------|--------|
| Auth API login p95 latency | < 300ms ✅ (live) |
| Admin UI — provision a new tenant | < 3 minutes end-to-end |
| Admin UI — invite a user | < 1 minute |
| Recruitment integration time | < 2 business days with integration guide |
| Auth Admin UI availability | 99.9% (shared EKS SLA) |
| PDPA: audit log completeness | 100% of auth events logged |

---

## 3. User Personas

### Maya — End User / Applicant
External job seeker. Non-technical. Registers once via the platform tenant and applies to multiple companies powered by Tigersoft Recruitment.
- **Needs:** Simple registration (email + password), password reset, profile data portability, right to deletion
- **Auth system interaction:** `POST /api/v1/auth/register` and `/login` with `X-Tenant-ID: platform`
- **Success:** Registers once; applies to multiple companies without re-entering data

### Carlos — Tenant Admin (HR Manager)
Manages users within his company's tenant (e.g., `acme`). Moderate technical. Will use the Admin UI daily.
- **Needs:** Invite HR colleagues, assign recruiter/hiring_manager roles, disable former employees, view audit trail
- **Auth system interaction:** Auth Admin UI — Users, Roles, Audit Log sections
- **Success:** Onboards a new HR colleague in < 3 minutes; never needs to touch an API

### Priya — Recruitment Developer
Builds the Recruitment module backend and frontend. High technical.
- **Needs:** Clear JWT verification guide, M2M credentials, JWKS endpoint, documented role names, sandbox tenant for testing
- **Auth system interaction:** JWKS endpoint, M2M token endpoint, RBAC API (to seed Recruitment roles)
- **Success:** Integrates Auth as SSO backbone for Recruitment in < 2 business days using the integration guide

### Jordan — Platform Super-Admin
Tigersoft platform operator. High technical. Manages all tenants across all clients.
- **Needs:** Create/suspend/view all tenants, assign modules, generate M2M credentials, cross-tenant audit visibility
- **Auth system interaction:** Auth Admin UI — full access; all sections
- **Success:** Provisions a new Recruitment client (tenant + roles + M2M credentials) in < 15 minutes from the Admin UI

---

## 4. Scope Overview

| Phase | Description | Status | Target |
|-------|-------------|--------|--------|
| **1 — Auth API** | 37 endpoints: auth, OAuth 2.0, RBAC, MFA, GDPR, multi-tenant | ✅ Live at auth.tgstack.dev | Complete |
| **1.1 — Bootstrap fix** | Automatic platform tenant + super_admin seeding on every deploy | ✅ Live | Complete |
| **1.2 — Recruitment prereqs** | `applicant` system role; `enabled_modules` in TenantConfig | ✅ Committed | Complete |
| **2 — Auth Admin UI** | Web console: tenant mgmt, user mgmt, audit log, settings | 🔵 Building | Next |
| **3 — Recruitment Integration** | SSO contract, JWKS verification guide, onboarding wizard | 🟡 Planned | After Admin UI |

---

## 5. Phase 1 — Auth API (Complete)

### 5.1 What is live

**Deployment**
- URL: `https://auth.tgstack.dev`
- Infrastructure: EKS `henderson` namespace, AWS ap-southeast-7 (Bangkok)
- Routing: Cloudflare Tunnel → Kong API Gateway → ClusterIP service
- Image: ECR `855407392262.dkr.ecr.ap-southeast-7.amazonaws.com/tigersoft-auth`

**Core capabilities**

| Feature | Endpoints | Notes |
|---------|-----------|-------|
| Authentication | Register, Login, Logout, Logout All | Email/password, RS256 JWT |
| Token management | Refresh (with rotation) | 15-min access / 24h refresh |
| Email verification | Verify, Resend | Resend email via Resend API |
| Password reset | Forgot, Reset | Time-limited token |
| Social login | Google OAuth (PKCE) | Per-tenant Google client config |
| User profile | Get me, Update, GDPR erase | Self-service profile management |
| MFA / TOTP | Generate, Confirm, Disable | Google Authenticator compatible |
| RBAC | Create role, List, Assign, Unassign | Per-tenant custom roles |
| Admin users | Invite, List, Disable, GDPR erase | Tenant-scoped admin operations |
| Audit log | Query with filters | 21 event types, date range, actor |
| Tenant lifecycle | Provision, Get, List, Suspend | Schema-per-tenant isolation |
| M2M credentials | Generate, Rotate | client_credentials OAuth grant |
| OAuth 2.0 server | PKCE authorize, token, introspect | RFC 6749/7662 compliant |
| Health + JWKS | `/health`, `/.well-known/jwks.json` | Dependency health, public keys |

**Multi-tenancy model**
- Each client company = one PostgreSQL schema (`tenant_<slug>`)
- Global `public` schema holds `tenants` registry
- Tenant resolved from `X-Tenant-ID` header (slug) or JWT `tenant_id` claim
- Platform tenant (`platform` / `tenant_platform`) is the super-admin and applicant identity store

**System roles (seeded in every tenant)**

| Role | Description |
|------|-------------|
| `user` | Standard authenticated user |
| `admin` | Tenant administrator — manages users/roles within their tenant |
| `super_admin` | Platform operator — manages all tenants |
| `applicant` | External job seeker — cross-tenant identity (platform tenant) |

**Known constraints**
- `POST /api/v1/oauth/revoke` — not yet implemented (returns 501)
- No mobile-native SDK yet
- No webhook/event streaming for external consumers

---

## 6. Phase 2 — Auth Admin UI

### 6.1 Overview

A web-based management console deployed at `auth-admin.tgstack.dev`. Serves two personas simultaneously via role-based routing:
- **Jordan (super_admin)** — sees full platform console including Tenants section
- **Carlos (tenant admin)** — sees users, roles, settings, and audit log scoped to their tenant only

### 6.2 Technical decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| Stack | Next.js 15 + Tailwind CSS + shadcn/ui | Server-side rendering, file-based routing, composable components |
| Design system | TGX (Tiger OpenSpace) | Matches Tigersoft V4 brand; bilingual TH/EN |
| Repository | Separate repo: `digital-worker/auth-admin-ui` | Independent deployments and PRs |
| Hosting | EKS `henderson` namespace | Consistent with Auth API infra |
| Domain | `auth-admin.tgstack.dev` | Via existing Cloudflare Tunnel → Kong |

### 6.3 TGX Design System — applied tokens

The Admin UI must match the TGX design system extracted from Figma `[Prototype] TGX - 20/80`.

**Colors**
| Token | Hex | Usage |
|-------|-----|-------|
| Tiger Red | `#C10016` | Active nav, selected states, primary checkboxes, breadcrumb active |
| Semi Black | `#3A3A3A` | All primary text |
| Page Background | `#FAFAFA` | Content area |
| Card / Panel | `#FFFFFF` | All cards, sidebar |
| Input Background | `#F0F0F0` | All form inputs |
| Primary Blue | `#2563EB` | CTA / action buttons |

**Typography**
- English: `Poppins` (400/500/600)
- Thai: `Noto Sans Thai` (matching weights)
- Button labels: `Inter`
- Scale: 20px titles → 16px subheadings → 14px body → 12px nav labels

**Layout (1440px desktop-first)**
- Sidebar: 60px collapsed (icon-only) / 298px expanded (icon + label)
- Active sidebar item: Tiger Red tint
- Top header: breadcrumb left + profile icons (40px circles) right
- Cards: `bg-white rounded-[10px] p-[10px]`
- Page background: `#FAFAFA`
- Inputs: `bg-[#f0f0f0] border-[#f0f0f0] rounded-[10px] p-[12px]`
- Buttons: `bg-[#2563eb] text-white rounded-[1000px]`

**Bilingual requirement**
- All screens must support TH and EN via language toggle (top-right header)
- Font stacks always include both Poppins and Noto Sans Thai

### 6.4 App structure

```
auth-admin-ui/
├── src/
│   ├── app/
│   │   ├── (auth)/
│   │   │   └── login/page.tsx          # Login form (email + password + tenant slug)
│   │   └── (dashboard)/
│   │       ├── layout.tsx              # TGX sidebar + header shell
│   │       ├── page.tsx                # Dashboard — summary stats
│   │       ├── tenants/
│   │       │   ├── page.tsx            # Tenant list (super_admin only)
│   │       │   ├── new/page.tsx        # Provision tenant form
│   │       │   └── [id]/
│   │       │       ├── page.tsx        # Tenant detail + M2M credentials
│   │       │       └── setup/
│   │       │           └── recruitment/page.tsx  # Recruitment onboarding wizard
│   │       ├── users/
│   │       │   ├── page.tsx            # User list + invite button
│   │       │   └── [id]/page.tsx       # User detail + role assignment
│   │       ├── roles/page.tsx          # Role list + create custom role
│   │       ├── audit/page.tsx          # Filterable audit log table
│   │       └── settings/
│   │           ├── page.tsx            # MFA enforcement toggle
│   │           └── oauth/page.tsx      # OAuth client registration
│   ├── lib/
│   │   ├── api.ts                      # Typed fetch client → auth.tgstack.dev
│   │   └── auth.ts                     # Token store + silent refresh
│   └── components/
│       ├── layout/
│       │   ├── Sidebar.tsx             # TGX collapsible sidebar
│       │   └── Header.tsx              # Breadcrumb + profile icons
│       └── ui/                         # shadcn/ui components
├── Dockerfile                          # node:20-alpine, standalone output
└── k8s/
    ├── deployment.yaml                 # 2 replicas, henderson namespace
    ├── service.yaml                    # ClusterIP port 3000
    └── configmap.yaml                  # NEXT_PUBLIC_API_URL etc.
```

### 6.5 Authentication flow

```
1. User visits auth-admin.tgstack.dev
2. Not authenticated → redirect to /login
3. /login form: email + password + tenant-slug (default: "platform")
4. POST auth.tgstack.dev/api/v1/auth/login  (X-Tenant-ID: <slug>)
5. On success: store access_token in memory, refresh_token in httpOnly cookie
6. Decode JWT → extract roles[], tenant_id
7. Role-based routing:
   super_admin → /dashboard (all sections visible)
   admin only  → /users (Tenants section hidden)
8. Silent token refresh before expiry via /api/v1/auth/token/refresh
```

### 6.6 Feature specifications

#### F1 — Login Page
- Email + password + tenant slug input (slug defaults to `platform`)
- Tiger Red branding, bilingual TH/EN
- Error states: invalid credentials, account disabled, MFA required (step-up flow)
- TOTP code entry step-up when `MFA_REQUIRED` response received

#### F2 — Dashboard
- Stat cards: Total tenants, Total users (in current tenant), Active sessions (if available)
- Super admin sees platform-wide; tenant admin sees own tenant only

#### F3 — Tenant List (super_admin only)
- Table: Name, Slug, Status badge (active / suspended), Module badges (RECRUITMENT etc.), Created date
- Actions: Provision new tenant, Suspend, View detail
- Pagination: TGX pagination component
- Search/filter by name or status

#### F4 — Provision Tenant Form
- Fields: Company name, Slug (auto-generate from name, editable), Admin email
- Module selector (multi-select): `Recruitment`, `Payroll` (disabled, future), `Performance Management` (disabled, future)
- On submit: `POST /api/v1/admin/tenants` then navigate to tenant detail
- If Recruitment selected: auto-navigate to Recruitment onboarding wizard

#### F5 — Tenant Detail
- Info panel: name, slug, schema name, status, created date
- Module badges
- M2M Credentials panel:
  - Show existing client_id (masked secret — shown once at generation)
  - Actions: Generate credentials, Rotate credentials
  - Copy-to-clipboard for each value
- Integration Panel (see F9)
- Danger zone: Suspend tenant (confirmation modal)

#### F6 — User List
- Table: Name, Email, Status badge, Roles (chips), Last login, MFA status indicator
- Filter: All | Active | Disabled | role filter dropdown
- Platform tenant: additional filter — All | HR Users | Applicants (by `applicant` role)
- Actions per row: View, Disable, Erase (with confirmation)
- Invite User button → modal form (email + first name + last name)

#### F7 — User Detail
- Profile: avatar initials, email, name, status, created date, last login, MFA enabled badge
- Roles panel: current roles (chips) + Assign role (dropdown) + Remove role (×)
- Sessions: list of active sessions with revoke option
- Danger zone: Disable user, GDPR Erase (double-confirm with email typed)

#### F8 — Roles Page
- Table: Name, Description, Type (System / Custom), User count
- Create Custom Role: name + description form
- System roles (user, admin, super_admin, applicant): shown, cannot be deleted
- Custom roles: can be deleted if no users assigned

#### F9 — Audit Log
- Table: Timestamp, Event Type, Actor (email), Target (email/tenant), IP address
- Filter bar: Event Type (dropdown), Actor email, Target user email, Date range (from/to)
- Pagination
- Export: CSV download button (generates filtered results)

#### F10 — Settings
- **MFA policy tab:** Toggle "Require MFA for all users in this tenant" → `PUT /api/v1/admin/tenant/mfa`
- **OAuth Clients tab:** List registered OAuth clients; Register new client form (name + redirect URIs)
- **Password policy tab:** Future (not Phase 1)

#### F11 — Recruitment Onboarding Wizard (`/tenants/[id]/setup/recruitment`)
Multi-step wizard shown after provisioning a tenant with Recruitment module selected:

**Step 1 — Seed Recruitment roles**
Automatically calls `POST /api/v1/admin/roles` (via M2M) to create:
- `recruiter` — HR staff managing the ATS
- `hiring_manager` — Department heads who approve requisitions
- `interviewer` — Senior staff who participate in interviews

Shows progress spinner → success confirmation.

**Step 2 — Generate M2M Credentials**
Calls `POST /api/v1/admin/tenants/:id/credentials`.
Displays `client_id` and `client_secret` (shown once — copy required).

**Step 3 — Integration Details Panel**
Copy-paste block with all env vars the Recruitment backend needs:
```bash
AUTH_API_URL=https://auth.tgstack.dev
AUTH_JWKS_URL=https://auth.tgstack.dev/.well-known/jwks.json
AUTH_ISSUER=https://auth.tgstack.dev
AUTH_AUDIENCE=tigersoft-auth
AUTH_TENANT_SLUG=<tenant-slug>
AUTH_PLATFORM_TENANT_SLUG=platform
AUTH_CLIENT_ID=<generated>
AUTH_CLIENT_SECRET=<shown once — copy now>
```
Each line has a copy-to-clipboard button.

**Step 4 — Done**
Checklist summary + link to Recruitment Integration Guide.

### 6.7 K8s additions required

```yaml
# New: auth-admin-ui Deployment (henderson namespace)
replicas: 2
image: <ECR>/tigersoft-auth-admin:<sha>
port: 3000

# New: auth-admin-ui Service (ClusterIP)
port: 3000

# New: auth-admin-ui ConfigMap
NEXT_PUBLIC_API_URL: "https://auth.tgstack.dev"
NEXT_PUBLIC_APP_NAME: "TGX Auth Console"

# Existing Kong: new route
host: auth-admin.tgstack.dev → auth-admin.henderson.svc.cluster.local:3000

# Existing Cloudflare Tunnel (Tiger Cluster): new ingress rule
auth-admin.tgstack.dev → http://auth-admin.henderson.svc.cluster.local:3000
```

### 6.8 Out of scope (Phase 2)

- Mobile-responsive layout (desktop-first at 1440px; responsive is Phase 3)
- Dark mode
- Notification emails triggered from Admin UI events
- Bulk user import (CSV)
- Analytics/reporting dashboard

---

## 7. Phase 3 — Recruitment System Integration

### 7.1 Overview

The Recruitment module is the **first V4 module** to consume the Auth platform as its SSO backbone. Every Tigersoft Recruitment client = one Auth tenant. The Auth platform serves two distinct identity types for Recruitment:

| Identity type | Auth tenant | Who | JWT payload |
|---------------|------------|-----|-------------|
| HR user (internal) | Client tenant (e.g. `acme`) | Recruiter, Hiring Manager, Interviewer | `tenant_id=acme-id, roles=["recruiter"]` |
| Applicant (external) | `platform` tenant | Job seeker | `tenant_id=platform-id, roles=["applicant"]` |

### 7.2 Auth system prerequisites (complete)

| Item | Status | Detail |
|------|--------|--------|
| `applicant` system role | ✅ Done (migration 000016) | Seeded in all tenants; `is_system=true` |
| `enabled_modules` in TenantConfig | ✅ Done (domain change) | `[]string` in existing config JSONB |
| JWKS endpoint | ✅ Live | `GET //.well-known/jwks.json` |
| OAuth 2.0 PKCE | ✅ Live | `/oauth/authorize` + `/oauth/token` |
| M2M client_credentials | ✅ Live | `/oauth/token` with client_credentials |
| RBAC API | ✅ Live | `POST /admin/roles`, assign/unassign |

### 7.3 Integration architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Tigersoft V4 Platform                        │
│                                                                  │
│   ┌──────────────────────┐         ┌────────────────────────┐   │
│   │   Auth System        │◀──JWT───│   Recruitment Module   │   │
│   │   auth.tgstack.dev   │         │   recruit.tgstack.dev  │   │
│   │                      │──M2M───▶│                        │   │
│   │  platform tenant     │  creds  │  verifies RS256 JWT    │   │
│   │  acme tenant         │         │  via JWKS endpoint     │   │
│   └──────────────────────┘         └────────────────────────┘   │
│                                                                  │
│   Applicants → platform tenant    (shared cross-client identity) │
│   HR Users   → company tenant     (isolated per client)          │
└─────────────────────────────────────────────────────────────────┘
```

### 7.4 HR user authentication flow (OAuth 2.0 PKCE)

```
Recruitment Frontend
  │
  ├─ 1. Generate PKCE code_verifier + code_challenge
  │
  ├─ 2. GET auth.tgstack.dev/api/v1/oauth/authorize
  │       ?client_id=<recruitment-client-id>
  │       &redirect_uri=https://recruit.tgstack.dev/auth/callback
  │       &response_type=code
  │       &code_challenge=<sha256(verifier)>
  │       &code_challenge_method=S256
  │       X-Tenant-ID: acme         ← company's tenant slug
  │
  ├─ 3. User logs in at Auth (email + password, optional MFA)
  │
  ├─ 4. Auth redirects to redirect_uri with ?code=<auth_code>
  │
  ├─ 5. Recruitment backend: POST /api/v1/oauth/token
  │       grant_type=authorization_code
  │       code=<auth_code>
  │       code_verifier=<original verifier>
  │       client_id=<recruitment-client-id>
  │
  └─ 6. Auth returns access_token (RS256 JWT)
         {
           "iss": "https://auth.tgstack.dev",
           "sub": "<user-id>",
           "aud": ["tigersoft-auth"],
           "tenant_id": "<acme-tenant-id>",
           "roles": ["recruiter"],
           "exp": ..., "iat": ...
         }
```

### 7.5 Applicant authentication flow (direct login)

```
Applicant Portal
  │
  ├─ Register: POST auth.tgstack.dev/api/v1/auth/register
  │              X-Tenant-ID: platform
  │              { email, password, first_name, last_name }
  │
  ├─ Login:    POST auth.tgstack.dev/api/v1/auth/login
  │              X-Tenant-ID: platform
  │              { email, password }
  │
  └─ Receives JWT:
       { tenant_id: <platform-id>, roles: ["user", "applicant"], sub: <user-id> }

       → Send as Bearer token to Recruitment API on every request
       → Recruitment backend: if tenant_id == platform_id → applicant context
                              if tenant_id == client_id  → HR user context
```

### 7.6 JWT verification in Recruitment backend

```go
// No Auth SDK needed — standard JWT verification
//
// 1. Fetch JWKS at startup (cache with rotation)
GET https://auth.tgstack.dev/.well-known/jwks.json
// Returns RSA public keys (kid, n, e)

// 2. On every request:
//    a. Parse JWT header → get kid
//    b. Find matching key in JWKS
//    c. Verify RS256 signature
//    d. Check: iss == "https://auth.tgstack.dev"
//    e. Check: aud contains "tigersoft-auth"
//    f. Check: exp > now()

// 3. Extract claims:
//    sub       → Auth user ID (stable across sessions)
//    tenant_id → Auth tenant UUID
//    roles[]   → ["recruiter"] / ["hiring_manager"] / ["applicant"] etc.

// 4. Context routing:
//    if tenant_id == PLATFORM_TENANT_ID → applicant context
//    else                               → HR user context (company tenant)
```

### 7.7 Recruitment-specific roles

These roles are **not seeded by Auth migrations** — Auth stays module-agnostic. The Recruitment backend creates them via the RBAC API using M2M credentials at client onboarding.

| Role | Created by | Purpose |
|------|-----------|---------|
| `recruiter` | Recruitment backend (RBAC API) | HR staff — full ATS access |
| `hiring_manager` | Recruitment backend (RBAC API) | Approve requisitions, review shortlists |
| `interviewer` | Recruitment backend (RBAC API) | Evaluate candidates in interview stage |
| `applicant` | Auth migration 000016 | External job seeker (platform tenant only) |

**Seeding at client onboarding:**
```bash
# Recruitment backend calls Auth using M2M credentials
POST https://auth.tgstack.dev/api/v1/admin/roles
Authorization: Bearer <M2M access token>
X-Tenant-ID: <client-tenant-slug>
{ "name": "recruiter", "description": "HR staff — full ATS access" }

# Repeat for hiring_manager and interviewer
```

### 7.8 M2M for Recruitment backend → Auth API

```bash
# 1. Get M2M access token
POST https://auth.tgstack.dev/api/v1/oauth/token
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials
&client_id=<from Auth Admin UI>
&client_secret=<from Auth Admin UI>

# 2. Use token to call Auth API (e.g. invite a user)
POST https://auth.tgstack.dev/api/v1/admin/users/invite
Authorization: Bearer <M2M token>
X-Tenant-ID: <client-slug>
{ "email": "hr@acme.com", "first_name": "Nok", "last_name": "Suphan" }
```

### 7.9 Environment variables for Recruitment backend

```bash
# Auth integration
AUTH_API_URL=https://auth.tgstack.dev
AUTH_JWKS_URL=https://auth.tgstack.dev/.well-known/jwks.json
AUTH_ISSUER=https://auth.tgstack.dev
AUTH_AUDIENCE=tigersoft-auth
AUTH_PLATFORM_TENANT_ID=25f56289-829b-42ea-903c-df8f6f664357  # platform tenant UUID

# Per-client (one set per Recruitment client company)
AUTH_CLIENT_TENANT_SLUG=acme       # company's tenant slug
AUTH_M2M_CLIENT_ID=<generated>     # from Auth Admin UI credentials panel
AUTH_M2M_CLIENT_SECRET=<generated> # from Auth Admin UI credentials panel (shown once)
```

### 7.10 PDPA compliance via Auth audit log

Every identity event is automatically recorded in Auth's audit log. The Recruitment module can query this for compliance reports:

```bash
GET https://auth.tgstack.dev/api/v1/admin/audit-log
Authorization: Bearer <M2M token>
X-Tenant-ID: <client-slug>

?event_type=REGISTER       # applicant registrations
?event_type=LOGIN          # all logins
?event_type=GDPR_ERASE     # deletion requests
?from=2026-01-01T00:00:00Z
?to=2026-03-31T23:59:59Z
```

Audit event types relevant to PDPA:
- `REGISTER` — user registration with consent timestamp
- `LOGIN` / `LOGOUT` — access records
- `EMAIL_VERIFIED` — verification timestamp
- `PASSWORD_RESET` — password changes
- `GDPR_ERASE` — deletion completion record

---

## 8. Constraints & Non-Functional Requirements

### 8.1 Security

| Requirement | Implementation |
|-------------|---------------|
| JWT signing | RS256 with 2048-bit RSA key; key ID in header for rotation |
| Password hashing | Argon2id (m=65536, t=3, p=2, salt=16B, key=32B) |
| Token expiry | Access: 15 min; Refresh: 24h with rotation |
| TOTP brute force | Rate-limited; 15-min lockout after failures |
| Login lockout | Configurable threshold per tenant (default: 5 attempts, 15-min lockout) |
| Transport | TLS only (Cloudflare terminates + re-encrypts to cluster) |
| Container | Distroless image, runAsNonRoot, readOnlyRootFilesystem, dropped ALL capabilities |

### 8.2 PDPA (Thailand Personal Data Protection Act)

- Explicit consent captured at registration
- Data minimization: only collect what's needed
- Right to deletion: `DELETE /api/v1/users/me` anonymizes PII and revokes all sessions
- Admin-initiated erasure: `DELETE /api/v1/admin/users/:id`
- Cross-tenant applicant data: applicant must explicitly consent to data portability between client companies
- Audit trail: 100% of identity events logged with actor, timestamp, IP

### 8.3 Performance

| Metric | Target |
|--------|--------|
| Login p95 | < 300ms |
| Token refresh p95 | < 100ms |
| JWKS endpoint p99 | < 50ms (cached at Recruitment) |
| Admin UI page load | < 2s (Next.js SSR) |

### 8.4 Availability

- Auth API: 2 replicas, rolling update, zero-downtime, liveness + readiness probes
- HPA: 2–10 replicas at 70% CPU
- Admin UI: 2 replicas (same pattern)
- Database: Central PostgreSQL in `database` namespace (shared cluster)

---

## 9. Open Questions

| # | Question | Owner | Deadline |
|---|----------|-------|---------|
| Q1 | Does the Recruitment frontend handle Auth redirect itself, or does it embed an Auth login widget (iframe/component)? | Priya + Jordan | Before Recruitment sprint 1 |
| Q2 | What is the applicant data consent UX — shown on Auth's register page or on Recruitment's applicant portal? | Poom + Jordan | Before Recruitment sprint 1 |
| Q3 | Should the Admin UI support Thai-language content for the login/error pages from Day 1, or TH/EN toggle only in the dashboard? | Design | Before UI sprint 1 |
| Q4 | Is the Auth Admin UI deployed publicly (`auth-admin.tgstack.dev`) or behind IP allowlist / VPN? | Jordan | Before UI infra setup |
| Q5 | Recruitment module URL — `recruit.tgstack.dev`? Confirm before registering Kong route | Priya | Before Recruitment sprint 1 |
| Q6 | LINE messaging integration for auth notifications (OTP, login alerts) — Phase 2 or future? | Poom | Quarter planning |

---

## 10. Work Breakdown — All Pending Items

### Phase 2 — Auth Admin UI

| # | Task | Type | Priority |
|---|------|------|---------|
| UI-01 | Scaffold Next.js 15 project + shadcn/ui + TGX Tailwind tokens | Frontend | P0 |
| UI-02 | Login page with TGX branding + TOTP step-up | Frontend | P0 |
| UI-03 | TGX sidebar (60px/298px) + header shell + language toggle | Frontend | P0 |
| UI-04 | API client lib (`lib/api.ts`) + auth store + token refresh | Frontend | P0 |
| UI-05 | Tenant list page + provision form (with module selector) | Frontend | P0 |
| UI-06 | Tenant detail page + M2M credentials panel | Frontend | P0 |
| UI-07 | Recruitment onboarding wizard (3-step) | Frontend | P0 |
| UI-08 | User list + invite modal | Frontend | P0 |
| UI-09 | User detail + role assignment + disable/erase | Frontend | P0 |
| UI-10 | Roles page + create custom role | Frontend | P1 |
| UI-11 | Audit log table + filters + CSV export | Frontend | P1 |
| UI-12 | Settings: MFA toggle + OAuth client registration | Frontend | P1 |
| UI-13 | Applicant user filter on platform tenant user list | Frontend | P1 |
| UI-14 | Dockerfile + K8s manifests (deployment/service/configmap) | DevOps | P0 |
| UI-15 | Kong route + Cloudflare Tunnel rule for auth-admin.tgstack.dev | DevOps | P0 |
| UI-16 | CI/CD pipeline (GitHub Actions → ECR → kubectl rollout) | DevOps | P1 |
| UI-17 | TH/EN bilingual strings for all labels | Frontend | P2 |

### Phase 3 — Recruitment Integration

| # | Task | Type | Owner | Priority |
|---|------|------|-------|---------|
| RI-01 | Produce Recruitment Integration Guide (JWKS, JWT, M2M, roles) | Docs | Auth team | P0 |
| RI-02 | Register Recruitment OAuth client in Auth Admin UI | Config | Priya | P0 |
| RI-03 | Implement JWKS-based JWT verification in Recruitment backend | Backend | Priya | P0 |
| RI-04 | Implement dual-tenant routing (client vs platform JWT) | Backend | Priya | P0 |
| RI-05 | Seed recruiter/hiring_manager/interviewer roles at onboarding | Backend | Priya | P0 |
| RI-06 | Applicant registration + login via platform tenant | Frontend | Priya | P0 |
| RI-07 | Validate PDPA consent flow design with Poom | Product | Poom | P0 |
| RI-08 | End-to-end SSO test: HR login → JWT → Recruitment backend | QA | QA | P1 |

---

## 11. Appendix — Auth API Quick Reference (for Recruitment team)

### Base URL
```
https://auth.tgstack.dev
```

### JWT structure
```json
{
  "header": { "alg": "RS256", "kid": "key-prod-1", "typ": "JWT" },
  "payload": {
    "iss": "https://auth.tgstack.dev",
    "sub": "<user-uuid>",
    "aud": ["tigersoft-auth"],
    "exp": 1234567890,
    "iat": 1234566990,
    "jti": "jwt_<uuid>",
    "tenant_id": "<tenant-uuid>",
    "roles": ["recruiter"]
  }
}
```

### Key endpoints for Recruitment

| Method | Path | Header | Purpose |
|--------|------|--------|---------|
| GET | `/.well-known/jwks.json` | — | Public keys for JWT verification |
| POST | `/api/v1/oauth/token` | — | Exchange code or client_credentials for token |
| POST | `/api/v1/auth/login` | `X-Tenant-ID: platform` | Applicant direct login |
| POST | `/api/v1/auth/register` | `X-Tenant-ID: platform` | Applicant registration |
| POST | `/api/v1/admin/roles` | `X-Tenant-ID: <slug>` + M2M Bearer | Create Recruitment roles |
| POST | `/api/v1/admin/users/invite` | `X-Tenant-ID: <slug>` + M2M Bearer | Invite HR user |
| GET | `/api/v1/admin/audit-log` | `X-Tenant-ID: <slug>` + M2M Bearer | PDPA audit queries |
