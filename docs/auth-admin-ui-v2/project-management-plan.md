# Project Management Plan
## Auth Admin UI v2 — Backend Integration

**Version:** 1.0
**Date:** 2026-03-04
**Status:** Approved
**Author:** Project Manager Agent
**Methodology:** Agile/Scrum (2-week sprints)
**PRD Reference:** `/docs/auth-admin-ui-v2/prd.md`
**Architecture Reference:** `/docs/auth-admin-ui-v2/solution-architecture.md`

---

## 1. Project Charter

### Project Name
Auth Admin UI v2 — Backend Integration

### Objective
Close all contract gaps between the Auth Backend (Go/Gin) and Auth Admin UI (Next.js) so that tenant administrators and platform super-admins can manage users, roles, tenants, and security settings through a fully functional, production-ready admin interface.

### Scope
In scope:
- Backend: Fix JWT claims, pagination, field naming, HTTP methods, error format, audit log fields, missing endpoints, DB schema for module roles
- Frontend: Fix API client bugs, update TypeScript types, update role/user UI components, remove incorrect headers
- QA: E2E and integration test coverage for all changed endpoints

Out of scope (per PRD section 7): OAuth PKCE, password reset UI, email templates, i18n, dark mode, new K8s manifests, CI/CD changes

### Success Criteria
- Zero TypeScript type errors in frontend build
- All API integration tests pass (backend + frontend E2E)
- JWT `tenant_id` = slug confirmed by Recruitment team
- All 16 user stories accepted by Product Owner

### Constraints
- No version upgrades (Go, Next.js, TypeScript)
- Backward-compatible DB migrations only
- Existing JWKS endpoint unchanged
- EKS cluster: no new infrastructure provisioned

### Assumptions
- AI agent squad executes development (1 BE agent, 1 FE agent, 1 QA agent)
- Recruitment team available to verify JWT slug change
- No external approval gates beyond PO acceptance

### Stakeholders

| Name | Role | Involvement |
|------|------|-------------|
| Product Owner Agent | PO / Reviewer | Approves stories, Phase 1 & 6 |
| Solution Architect Agent | Architect | Reviews PRs with architectural concerns |
| Backend Developer Agent | BE Developer | Sprint 1–3 backend tasks |
| Frontend Developer Agent | FE Developer | Sprint 1–3 frontend tasks |
| QA/Tester Agent | QA | Sprint 1–3 test tasks |
| Tigersoft Operations | Deployment | ECR push + K8s rollout |
| Recruitment Team | Consumer | Verifies JWT slug fix |

---

## 2. Work Breakdown Structure (WBS)

```
1.0 Auth Admin UI v2 — Backend Integration
├── 1.1 Sprint 1 — Foundation Fixes (Must Have)
│   ├── 1.1.1 [BE] JWT tenant_id slug fix (BE-001)
│   │   ├── 1.1.1.1 Fix auth_service.go:313 — TenantID: tenant.Slug
│   │   ├── 1.1.1.2 Fix session_service.go:107 — refresh carries slug
│   │   └── 1.1.1.3 Unit tests for JWT claims
│   ├── 1.1.2 [BE+FE] Token refresh fix (BE-002)
│   │   ├── 1.1.2.1 [FE] Fix refresh URL in api.ts:189
│   │   ├── 1.1.2.2 [FE] Fix error parsing in api.ts:169
│   │   └── 1.1.2.3 [BE] Verify refresh endpoint path matches
│   ├── 1.1.3 [BE] Pagination standardization (BE-003)
│   │   ├── 1.1.3.1 Refactor ListUsers — page/page_size
│   │   ├── 1.1.3.2 Refactor ListTenants — page/page_size
│   │   ├── 1.1.3.3 Refactor ListAuditLog — page/page_size
│   │   └── 1.1.3.4 Update PaginatedResponse struct
│   ├── 1.1.4 [BE] Primary key rename (BE-004)
│   │   ├── 1.1.4.1 Rename user_id → id in all user responses
│   │   ├── 1.1.4.2 Rename tenant_id → id in all tenant responses
│   │   └── 1.1.4.3 Rename role_id → id in all role responses
│   ├── 1.1.5 [BE+FE] User status normalization (BE-005)
│   │   ├── 1.1.5.1 [BE] Map invited→pending in domain/user.go
│   │   ├── 1.1.5.2 [BE] Map disabled→inactive in domain/user.go
│   │   └── 1.1.5.3 [FE] Update status badge component
│   ├── 1.1.6 [BE] HTTP method fixes (BE-006)
│   │   ├── 1.1.6.1 Router: PUT /users/:id/disable → POST
│   │   └── 1.1.6.2 Router: PUT /tenants/:id/suspend → POST
│   ├── 1.1.7 [BE+FE] Error format standardization (BE-007)
│   │   ├── 1.1.7.1 [BE] Audit all handlers — respondWithServiceError
│   │   ├── 1.1.7.2 [BE] Ensure all errors use ErrorResponse{Error: ErrorDetail{Code, Message}}
│   │   └── 1.1.7.3 [FE] Update error parsing to err.error?.message
│   ├── 1.1.8 [FE] Remove X-Tenant-ID from authenticated requests (FE-002)
│   │   └── 1.1.8.1 Remove tenantId from apiFetch opts for authenticated calls
│   └── 1.1.9 [FE] Update TypeScript interfaces (FE-001)
│       ├── 1.1.9.1 Update User interface: system_roles + module_roles
│       ├── 1.1.9.2 Update Tenant interface: remove schema_name/config, add enabled_modules
│       ├── 1.1.9.3 Update PaginatedResponse: add total_pages
│       ├── 1.1.9.4 Update ApiError parsing
│       └── 1.1.9.5 Verify tsc --noEmit passes
│
├── 1.2 Sprint 2 — New Endpoints (Must Have)
│   ├── 1.2.1 [BE+FE] Audit log field renames (BE-008)
│   │   ├── 1.2.1.1 [BE] Rename event_type→action, actor_ip→ip_address, occurred_at→created_at, target_user_id→target_id
│   │   ├── 1.2.1.2 [BE] Add date range filter (from, to)
│   │   └── 1.2.1.3 [FE] Update AuditLog interface and audit page rendering
│   ├── 1.2.2 [BE] GET /admin/users/:id (BE-009)
│   │   ├── 1.2.2.1 Add GetUser handler method
│   │   ├── 1.2.2.2 Add GetUser service method with role resolution
│   │   └── 1.2.2.3 Register route in router
│   ├── 1.2.3 [BE] PUT /admin/users/:id/roles (BE-010)
│   │   ├── 1.2.3.1 Add ReplaceUserRoles handler
│   │   ├── 1.2.3.2 Add ReplaceUserRoles service (atomic delete + insert)
│   │   ├── 1.2.3.3 Add ReplaceUserRoles repository (transaction)
│   │   └── 1.2.3.4 Register route in router
│   ├── 1.2.4 [BE] Module column on roles (BE-011)
│   │   ├── 1.2.4.1 Migration 000017_add_module_to_roles
│   │   ├── 1.2.4.2 Migration 000018_seed_module_roles
│   │   ├── 1.2.4.3 Update Role domain struct (Module *string)
│   │   ├── 1.2.4.4 Update CreateRole handler — require module
│   │   ├── 1.2.4.5 Update ListRoles handler — include module in response
│   │   └── 1.2.4.6 Update resolveUserRoles to build module_roles map
│   ├── 1.2.5 [BE+FE] Remove schema_name from tenant responses (BE-013)
│   │   ├── 1.2.5.1 [BE] Remove schema_name from all TenantResponse builders
│   │   └── 1.2.5.2 [FE] Remove schema_name from Tenant interface
│   ├── 1.2.6 [BE+FE] Tenant settings GET/PUT endpoint (BE-014)
│   │   ├── 1.2.6.1 [BE] Add GetTenantSettings handler
│   │   ├── 1.2.6.2 [BE] Expand UpdateTenantSettings handler (mfa + session + domains)
│   │   ├── 1.2.6.3 [BE] Add GetTenantSettings service method
│   │   ├── 1.2.6.4 [BE] Register GET/PUT /admin/tenant routes
│   │   └── 1.2.6.5 [FE] Update Settings page to use GET/PUT /admin/tenant
│   ├── 1.2.7 [BE] POST /admin/tenants/:id/activate (BE-015)
│   │   ├── 1.2.7.1 Add ActivateTenant handler
│   │   ├── 1.2.7.2 Add ActivateTenant service method
│   │   └── 1.2.7.3 Register route in router
│   └── 1.2.8 [BE+FE] User invitation flow (BE-016)
│       ├── 1.2.8.1 [BE] Fix InviteUser response — return full UserObject with status=pending
│       ├── 1.2.8.2 [BE] POST /admin/users/:id/enable handler + service
│       ├── 1.2.8.3 [FE] Update invite form to not require role_ids (roles added separately)
│       └── 1.2.8.4 [FE] Verify invitation appears in user list as pending
│
├── 1.3 Sprint 3 — Should Have / Could Have
│   ├── 1.3.1 [BE] Protect system roles from deletion (BE-012)
│   │   ├── 1.3.1.1 Add is_system check in DeleteRole service
│   │   └── 1.3.1.2 Add role_in_use check before delete
│   ├── 1.3.2 [FE] Roles page — module tab grouping (FE-003)
│   │   ├── 1.3.2.1 Add module tabs component to Roles page
│   │   ├── 1.3.2.2 Fetch roles with ?module= filter per tab
│   │   └── 1.3.2.3 Add "Create Custom Role" button per module tab
│   ├── 1.3.3 [FE] User detail — system and module role sections (FE-004)
│   │   ├── 1.3.3.1 Add SystemRoles section to user edit form
│   │   ├── 1.3.3.2 Add ModuleRoles section (grouped by module)
│   │   └── 1.3.3.3 Wire Save button to PUT /admin/users/:id/roles
│   ├── 1.3.4 [FE] User list status/role filter (FE-005)
│   │   ├── 1.3.4.1 Add status dropdown filter
│   │   └── 1.3.4.2 Add module/role dropdown filter
│   └── 1.3.5 [FE] Audit log date range filter (FE-006)
│       ├── 1.3.5.1 Add action dropdown filter
│       └── 1.3.5.2 Add from/to date pickers
│
├── 1.4 QA — Sprint 1 Testing
│   ├── 1.4.1 Integration tests: JWT claims (slug verification)
│   ├── 1.4.2 Integration tests: pagination (page/page_size)
│   ├── 1.4.3 Integration tests: error format
│   ├── 1.4.4 E2E: login → refresh → logout flow
│   └── 1.4.5 Frontend: tsc --noEmit passes CI
│
├── 1.5 QA — Sprint 2 Testing
│   ├── 1.5.1 Integration tests: GET/PUT /admin/users/:id
│   ├── 1.5.2 Integration tests: PUT /admin/users/:id/roles
│   ├── 1.5.3 Integration tests: module roles (seed + CRUD)
│   ├── 1.5.4 Integration tests: tenant activate/suspend
│   ├── 1.5.5 E2E: invite user → pending status → enable → active
│   └── 1.5.6 E2E: tenant settings update
│
└── 1.6 QA — Sprint 3 Testing
    ├── 1.6.1 E2E: roles page module tabs
    ├── 1.6.2 E2E: user detail role assignment via UI
    ├── 1.6.3 E2E: user list filter by status
    └── 1.6.4 Regression: full user/tenant/role/audit E2E suite
```

---

## 3. Sprint Plan

### Sprint 1 — Foundation Fixes
**Duration:** 2026-03-04 → 2026-03-17
**Goal:** Fix all Must Have contract gaps that block basic functionality
**Capacity:** 80% of 40 pts = 32 pts max
**Planned velocity:** 28 pts

| Task ID | Description | Agent | Points | Tag | Priority |
|---------|-------------|-------|--------|-----|----------|
| S1-BE-001 | JWT tenant_id slug fix | Backend Dev | 3 | backend | urgent |
| S1-BE-002 | Token refresh endpoint/error fix | Backend Dev | 1 | backend | urgent |
| S1-BE-003 | Pagination page/page_size | Backend Dev | 5 | backend | urgent |
| S1-BE-004 | Rename primary keys to id | Backend Dev | 3 | backend | high |
| S1-BE-005 | User status normalization | Backend Dev | 2 | backend | high |
| S1-BE-006 | Fix HTTP methods (POST) | Backend Dev | 2 | backend | high |
| S1-BE-007 | Error format standardization | Backend Dev | 3 | backend | high |
| S1-FE-001 | Fix refresh URL + error parsing | Frontend Dev | 2 | frontend | urgent |
| S1-FE-002 | Remove X-Tenant-ID from auth requests | Frontend Dev | 2 | frontend | high |
| S1-FE-003 | Update TypeScript interfaces | Frontend Dev | 5 | frontend | high |
| S1-QA-001 | QA: JWT, pagination, error format | QA | — | test | high |

**Sprint 1 Total BE Points:** 19
**Sprint 1 Total FE Points:** 9

### Sprint 2 — New Endpoints
**Duration:** 2026-03-18 → 2026-03-31
**Goal:** Implement all new endpoints and complete Must Have feature set
**Capacity:** 32 pts max
**Planned velocity:** 30 pts

| Task ID | Description | Agent | Points | Tag | Priority |
|---------|-------------|-------|--------|-----|----------|
| S2-BE-008 | Audit log field renames + date filter | Backend Dev | 3 | backend | high |
| S2-BE-009 | GET /admin/users/:id | Backend Dev | 3 | backend | high |
| S2-BE-010 | PUT /admin/users/:id/roles | Backend Dev | 5 | backend | high |
| S2-BE-011 | Module column + role migrations + seeding | Backend Dev | 5 | backend | high |
| S2-BE-012 | Remove schema_name from tenant responses | Backend Dev | 2 | backend | high |
| S2-BE-013 | GET/PUT /admin/tenant settings | Backend Dev | 5 | backend | high |
| S2-BE-014 | POST /admin/tenants/:id/activate | Backend Dev | 2 | backend | normal |
| S2-BE-015 | POST /admin/users/:id/enable + fix InviteUser | Backend Dev | 3 | backend | high |
| S2-FE-004 | Audit log UI field updates | Frontend Dev | 2 | frontend | high |
| S2-FE-005 | Tenant settings page: GET/PUT /admin/tenant | Frontend Dev | 3 | frontend | high |
| S2-FE-006 | InviteUser form: remove role_ids field | Frontend Dev | 1 | frontend | normal |
| S2-QA-002 | QA: new endpoints + migrations | QA | — | test | high |

**Sprint 2 Total BE Points:** 28
**Sprint 2 Total FE Points:** 6

### Sprint 3 — Should Have / Could Have
**Duration:** 2026-04-01 → 2026-04-14
**Goal:** Module role UI, user role sections, filtering
**Capacity:** 32 pts max
**Planned velocity:** 18 pts

| Task ID | Description | Agent | Points | Tag | Priority |
|---------|-------------|-------|--------|-----|----------|
| S3-BE-016 | Protect system roles from deletion | Backend Dev | 2 | backend | normal |
| S3-FE-007 | Roles page: module tabs | Frontend Dev | 5 | frontend | normal |
| S3-FE-008 | User detail: system + module role sections | Frontend Dev | 5 | frontend | normal |
| S3-FE-009 | User list: status + module filter | Frontend Dev | 3 | frontend | low |
| S3-FE-010 | Audit log: action + date filter | Frontend Dev | 3 | frontend | low |
| S3-QA-003 | QA: roles UI, user detail, filters, regression | QA | — | test | normal |

**Sprint 3 Total BE Points:** 2
**Sprint 3 Total FE Points:** 16

---

## 4. ClickUp Task List (Structured for Orchestrator)

### Sprint 1 Tasks

```
TASK:
  name: "[BE] Fix JWT tenant_id to use slug (BE-001)"
  list: Sprint 1
  description: |
    Fix auth_service.go:313 to use tenant.Slug instead of tenant.ID.String() when signing JWTs.
    Also fix session_service.go:107 to ensure refresh tokens carry slug in TenantID, not empty string.
    AC: JWT access_token payload contains "tenant_id": "henderson" (slug), not a UUID format.
    AC: Token refresh also returns JWT with slug in tenant_id.
  assignee_role: Developer
  story_points: 3
  priority: urgent
  tags: [backend]
  due_date: 2026-03-10

TASK:
  name: "[FE] Fix token refresh URL and error parsing (BE-002)"
  list: Sprint 1
  description: |
    Fix src/lib/api.ts:189 — change URL from "/api/v1/auth/refresh" to "/api/v1/auth/token/refresh".
    Fix src/lib/api.ts:169 — change error parsing from err.message to err.error?.message ?? "An unexpected error occurred".
    AC: Frontend calls /api/v1/auth/token/refresh on token expiry. User sees correct error messages.
  assignee_role: Developer
  story_points: 2
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-10

TASK:
  name: "[BE] Standardize pagination to page/page_size (BE-003)"
  list: Sprint 1
  description: |
    Refactor ListUsers, ListTenants, ListAuditLog handlers to accept page and page_size query params
    instead of limit/offset. Update PaginatedResponse to include total_pages.
    Response shape: { data, total, page, page_size, total_pages }.
    AC: All list endpoints respond correctly to ?page=2&page_size=10 with correct meta.
  assignee_role: Developer
  story_points: 5
  priority: urgent
  tags: [backend]
  due_date: 2026-03-12

TASK:
  name: "[BE] Rename primary key fields to 'id' in all responses (BE-004)"
  list: Sprint 1
  description: |
    In all handler response builders: rename user_id → id, tenant_id → id, role_id → id.
    Affects: admin_handler.go, tenant_handler.go, role_handler.go, auth_handler.go.
    AC: All API responses use "id" for the primary identifier field. No *_id fields in responses.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [backend]
  due_date: 2026-03-12

TASK:
  name: "[BE] Normalize user status values: pending/inactive/active (BE-005)"
  list: Sprint 1
  description: |
    In domain/user.go, ensure UserStatus constants are: active, inactive, pending.
    Map old "invited" → "pending", "disabled" → "inactive" in all service/repository layers.
    AC: Invited user shows status=pending. Disabled user shows status=inactive. Re-enabled shows active.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [backend]
  due_date: 2026-03-12

TASK:
  name: "[BE] Fix HTTP methods for state transitions to POST (BE-006)"
  list: Sprint 1
  description: |
    Update router to use POST (not PUT) for:
    - POST /api/v1/admin/users/:id/disable
    - POST /api/v1/admin/tenants/:id/suspend
    Handler logic unchanged — only HTTP method changes.
    AC: POST /users/:id/disable returns 204. POST /tenants/:id/suspend returns 204. PUT returns 405.
  assignee_role: Developer
  story_points: 2
  priority: high
  tags: [backend]
  due_date: 2026-03-10

TASK:
  name: "[BE] Standardize all error responses to {error: {code, message}} (BE-007)"
  list: Sprint 1
  description: |
    Audit all handlers. Ensure every error path uses ErrorResponse{Error: ErrorDetail{Code, Message}}.
    Introduce or update respondWithServiceError helper to always produce standard format.
    AC: All 4xx/5xx responses match {"error": {"code": "string", "message": "string"}}.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [backend]
  due_date: 2026-03-14

TASK:
  name: "[FE] Remove X-Tenant-ID header from authenticated requests (FE-002)"
  list: Sprint 1
  description: |
    In src/lib/api.ts, remove the tenantId option from apiFetch for all authenticated calls.
    Only login (pre-auth) should send X-Tenant-ID. JWT carries tenant identity post-login.
    AC: Authenticated API calls (admin/users, admin/roles, etc.) do not include X-Tenant-ID header.
  assignee_role: Developer
  story_points: 2
  priority: high
  tags: [frontend]
  due_date: 2026-03-12

TASK:
  name: "[FE] Update all TypeScript interfaces to match API contract (FE-001)"
  list: Sprint 1
  description: |
    Update src/lib/api.ts interfaces:
    - User: add system_roles/module_roles, remove flat roles[], add email_verified
    - Tenant: remove schema_name/config, add enabled_modules[]
    - PaginatedResponse: add total_pages
    - ApiError: { error: { code, message } }
    - Role: add module field (string | null)
    - InviteUserRequest: remove role_ids
    AC: tsc --noEmit exits with code 0. No TypeScript errors in CI.
  assignee_role: Developer
  story_points: 5
  priority: high
  tags: [frontend]
  due_date: 2026-03-14

TASK:
  name: "[QA] Sprint 1 — Test JWT, pagination, errors, auth flow"
  list: Sprint 1
  description: |
    Write and execute tests for Sprint 1 stories:
    - Integration: JWT claims contain slug not UUID (BE-001)
    - Integration: all list endpoints accept page/page_size (BE-003)
    - Integration: all error responses match standard format (BE-007)
    - E2E: login → JWT decode → verify tenant_id = slug
    - E2E: access token expiry → refresh → new token returned
    - Frontend: tsc --noEmit passes
    AC: All Sprint 1 acceptance criteria validated. Pass rate >= 95%.
  assignee_role: QA
  story_points: 0
  priority: high
  tags: [test]
  due_date: 2026-03-17
```

### Sprint 2 Tasks

```
TASK:
  name: "[BE] Rename audit log fields to match contract (BE-008)"
  list: Sprint 2
  description: |
    In audit_handler.go, rename response fields:
    event_type → action, actor_ip → ip_address, occurred_at → created_at, target_user_id → target_id.
    Also add ?from and ?to date range filter support.
    AC: Audit log entries use correct field names. Date filtering works.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [backend]
  due_date: 2026-03-24

TASK:
  name: "[BE] Add GET /api/v1/admin/users/:id endpoint (BE-009)"
  list: Sprint 2
  description: |
    Add GetUser handler in admin_handler.go. Add GetUser service method in admin_service.go.
    Response must include system_roles[] and module_roles{} (not flat roles[]).
    Register GET /api/v1/admin/users/:id route in router.go.
    AC: Returns full UserObject with system_roles and module_roles. Returns 404 for unknown UUID.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [backend]
  due_date: 2026-03-24

TASK:
  name: "[BE] Add PUT /api/v1/admin/users/:id/roles endpoint (BE-010)"
  list: Sprint 2
  description: |
    Add ReplaceUserRoles handler, service, and repository method.
    Service: within a transaction, DELETE all user_roles for user, then INSERT new role_ids.
    Validate all role_ids exist before transaction. Log to audit log.
    AC: Replaces all roles atomically. Empty role_ids removes all. Returns 422 for invalid UUIDs.
  assignee_role: Developer
  story_points: 5
  priority: high
  tags: [backend]
  due_date: 2026-03-26

TASK:
  name: "[BE] Add module column to roles + seed module roles (BE-011)"
  list: Sprint 2
  description: |
    Create migration 000017_add_module_to_roles: ALTER TABLE roles ADD COLUMN module TEXT DEFAULT NULL.
    Create migration 000018_seed_module_roles: seed recruit/payroll/time module roles.
    Update Role domain struct (Module *string). Update CreateRole to require module.
    Update ListRoles to include module in response. Update resolveUserRoles to build module_roles map.
    AC: System roles have module=null. Module roles carry module name. Filter by ?module= works.
  assignee_role: Developer
  story_points: 5
  priority: high
  tags: [backend]
  due_date: 2026-03-26

TASK:
  name: "[BE] Remove schema_name from tenant responses (BE-013)"
  list: Sprint 2
  description: |
    In tenant_handler.go, remove schema_name from all response builders (ProvisionTenant, GetTenant, ListTenants).
    Add enabled_modules[] field (from tenant_config).
    AC: No tenant response contains schema_name. All tenant responses include enabled_modules[].
  assignee_role: Developer
  story_points: 2
  priority: high
  tags: [backend]
  due_date: 2026-03-21

TASK:
  name: "[BE] Add GET/PUT /api/v1/admin/tenant settings endpoint (BE-014)"
  list: Sprint 2
  description: |
    Add GetTenantSettings handler: returns id, name, slug, status, mfa_required, session_duration_minutes, allowed_domains, enabled_modules.
    Expand UpdateTenantSettings: accepts mfa_required?, session_duration_minutes?, allowed_domains? (partial update).
    Deprecate PUT /admin/tenant/mfa (return 410 Gone with migration message).
    AC: GET returns full settings. PUT updates any subset. Old /tenant/mfa returns 410.
  assignee_role: Developer
  story_points: 5
  priority: high
  tags: [backend]
  due_date: 2026-03-28

TASK:
  name: "[BE] Add POST /api/v1/admin/tenants/:id/activate endpoint (BE-015)"
  list: Sprint 2
  description: |
    Add ActivateTenant handler and service method. Handler calls tenantSvc.ActivateTenant.
    Idempotent: already-active tenant returns 204 without error.
    Register POST /api/v1/admin/tenants/:id/activate route.
    AC: POST activate on suspended tenant → status=active. Idempotent on active tenant.
  assignee_role: Developer
  story_points: 2
  priority: normal
  tags: [backend]
  due_date: 2026-03-24

TASK:
  name: "[BE] Fix InviteUser response + add POST /admin/users/:id/enable (BE-016)"
  list: Sprint 2
  description: |
    Fix InviteUser response to return full UserObject (not just user_id+email).
    Add EnableUser handler: POST /admin/users/:id/enable → sets status=active → 204.
    AC: InviteUser returns UserObject with status=pending. EnableUser returns 204, status=active.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [backend]
  due_date: 2026-03-24

TASK:
  name: "[FE] Update audit log page to use renamed fields (BE-008)"
  list: Sprint 2
  description: |
    Update AuditLog interface and audit log page to use: action, ip_address, created_at, target_id.
    Remove old field names from rendering logic.
    AC: Audit log page renders without errors. Field values display correctly.
  assignee_role: Developer
  story_points: 2
  priority: high
  tags: [frontend]
  due_date: 2026-03-24

TASK:
  name: "[FE] Update Settings page to use GET/PUT /api/v1/admin/tenant (BE-014)"
  list: Sprint 2
  description: |
    Update Settings page to call GET /api/v1/admin/tenant on load.
    Update save action to call PUT /api/v1/admin/tenant with partial update body.
    Show mfa_required, session_duration_minutes, allowed_domains fields.
    AC: Settings load from API. Changes save via PUT. Success toast on save.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [frontend]
  due_date: 2026-03-28

TASK:
  name: "[FE] Update InviteUser form — remove role_ids (BE-016)"
  list: Sprint 2
  description: |
    Remove role_ids field from InviteUserRequest and invite form.
    Roles are assigned separately after invitation via PUT /admin/users/:id/roles.
    AC: Invite form submits with only email, first_name, last_name. No role_ids sent.
  assignee_role: Developer
  story_points: 1
  priority: normal
  tags: [frontend]
  due_date: 2026-03-24

TASK:
  name: "[QA] Sprint 2 — Test new endpoints, migrations, settings"
  list: Sprint 2
  description: |
    Write and execute tests for Sprint 2 stories:
    - Integration: GET /admin/users/:id, PUT /admin/users/:id/roles
    - Integration: module roles migration and seeding
    - Integration: tenant settings GET/PUT
    - E2E: invite user → appears as pending → enable → active
    - E2E: tenant settings update flow
    - E2E: suspend/activate tenant
    AC: All Sprint 2 acceptance criteria validated. Pass rate >= 95%.
  assignee_role: QA
  story_points: 0
  priority: high
  tags: [test]
  due_date: 2026-03-31

TASK:
  name: "[BE] Protect system roles from deletion (BE-012)"
  list: Sprint 3
  description: |
    In DeleteRole service method: check is_system=true → return 409 with code=system_role_protected.
    Check if any user_roles rows reference this role → return 409 with code=role_in_use.
    AC: System roles return 409 on delete. Roles in use return 409. Unused custom roles delete successfully.
  assignee_role: Developer
  story_points: 2
  priority: normal
  tags: [backend]
  due_date: 2026-04-07

TASK:
  name: "[FE] Roles page — module tab grouping (FE-003)"
  list: Sprint 3
  description: |
    Add tab navigation to Roles page: "System" tab (default) + one tab per enabled module.
    Each tab fetches roles with ?module= filter. System tab fetches roles with no module.
    "Create Custom Role" button only visible on module tabs (not system tab).
    AC: Tabs show correct roles. Create button only on module tabs. Tabs filter correctly.
  assignee_role: Developer
  story_points: 5
  priority: normal
  tags: [frontend]
  due_date: 2026-04-07

TASK:
  name: "[FE] User detail — system and module role sections (FE-004)"
  list: Sprint 3
  description: |
    Add SystemRoles section and ModuleRoles section (grouped by module) to user detail/edit page.
    Pre-populate from GET /admin/users/:id. Save calls PUT /admin/users/:id/roles.
    AC: Both sections render. Checkboxes pre-selected. Save calls PUT roles. Success toast.
  assignee_role: Developer
  story_points: 5
  priority: normal
  tags: [frontend]
  due_date: 2026-04-09

TASK:
  name: "[FE] User list — status and module filter (FE-005)"
  list: Sprint 3
  description: |
    Add status filter dropdown (active/inactive/pending) to user list toolbar.
    Add module filter dropdown. Filters update ?status= and ?module= query params.
    "Clear filters" resets to default.
    AC: Filters call API with correct params. Filtered results display. Clear resets.
  assignee_role: Developer
  story_points: 3
  priority: low
  tags: [frontend]
  due_date: 2026-04-09

TASK:
  name: "[FE] Audit log — action type and date range filter (FE-006)"
  list: Sprint 3
  description: |
    Add action dropdown filter and from/to date pickers to audit log toolbar.
    Filters update query params ?action=, ?from=, ?to= on the API call.
    AC: Action filter narrows to that event type. Date range filters correctly.
  assignee_role: Developer
  story_points: 3
  priority: low
  tags: [frontend]
  due_date: 2026-04-09

TASK:
  name: "[QA] Sprint 3 — Test roles UI, user detail, filters, full regression"
  list: Sprint 3
  description: |
    Write and execute tests for Sprint 3 stories:
    - E2E: roles page tabs, create custom role, system role delete rejection
    - E2E: user detail role assignment via UI
    - E2E: user list status filter
    - E2E: audit log date + action filter
    - Regression: full user/tenant/role/audit E2E suite from Sprint 1 + 2
    AC: All Sprint 3 acceptance criteria validated. Full regression passes. Pass rate >= 95%.
  assignee_role: QA
  story_points: 0
  priority: normal
  tags: [test]
  due_date: 2026-04-14
```

---

## 5. Risk Register

| ID | Risk | Probability | Impact | Score | Mitigation | Owner | Status |
|----|------|-------------|--------|-------|------------|-------|--------|
| R-001 | JWT change breaks Recruitment backend | Low | Critical | High | Validate JWT with Recruitment team before deploying to prod | Architect | Open |
| R-002 | DB migration 000017/000018 fails on existing data | Low | High | Medium | Additive migration, tested on staging first | Backend Dev | Open |
| R-003 | TypeScript type changes break existing UI pages | Medium | Medium | Medium | Fix types in Sprint 1 before component changes in Sprint 2+ | Frontend Dev | Open |
| R-004 | Pagination change breaks existing API consumers (M2M) | Low | Medium | Low | Check for any external callers using limit/offset; deprecation notice | Backend Dev | Open |
| R-005 | Role replacement transaction causes deadlock | Low | Low | Low | Short transaction, bounded input; monitored via PostgreSQL slow query log | Backend Dev | Open |
| R-006 | Sprint 2 scope exceeds capacity | Medium | Medium | Medium | Tenant activate (S2-BE-014) and InviteUser fix (S2-BE-015) can slip to Sprint 3 if needed | PM | Open |
| R-007 | E2E tests cannot run against EKS staging | Low | Medium | Low | E2E tests can run against local docker-compose; EKS smoke test post-deploy | QA | Open |

---

## 6. RACI Chart

| Task Area | Backend Dev | Frontend Dev | QA | Architect | PO |
|-----------|-------------|--------------|----|-----------|----|
| JWT fix | R/A | I | C | C | I |
| Pagination fix | R/A | I | C | I | I |
| Primary key rename | R/A | I | C | I | I |
| Error format | R/A | I | C | I | I |
| Token refresh (FE) | I | R/A | C | I | I |
| X-Tenant-ID removal | I | R/A | C | I | I |
| TypeScript types | I | R/A | C | C | I |
| Audit log renames | R/A | C | C | I | I |
| GET/PUT user | R/A | C | C | C | I |
| Role replace | R/A | C | C | C | I |
| Module roles | R/A | I | C | C | I |
| Tenant settings | R/A | C | C | C | I |
| Roles UI tabs | C | R/A | C | I | I |
| User role sections | C | R/A | C | I | I |
| QA — all stories | C | C | R/A | I | C |
| PO acceptance | I | I | C | I | R/A |

---

## 7. Definition of Done

Applies to every user story before it can be marked Done:

```
Code Quality:
[ ] Code follows existing patterns (Handler→Service→Repository for BE; shadcn/ui for FE)
[ ] No TODO comments or dead code introduced
[ ] TypeScript strict mode passes (FE)
[ ] Go vet and golint pass with no new warnings (BE)

Testing:
[ ] Unit tests written for new service/repository methods (BE)
[ ] Integration test covers the endpoint (BE)
[ ] E2E test covers the user-facing flow where applicable (FE)
[ ] QA has executed all Gherkin scenarios from the story
[ ] Pass rate >= 95% on all acceptance criteria

Functionality:
[ ] Acceptance criteria from PRD verified
[ ] API contract shape matches solution-architecture.md exactly
[ ] Error cases handled and return correct error format

Security:
[ ] No new secrets in code or logs
[ ] JWT validation not bypassed
[ ] Role check enforced server-side

Deployment:
[ ] DB migration tested locally and on staging
[ ] Docker build succeeds for target platform (linux/amd64)
[ ] Deployment smoke test passes on EKS

Product Owner:
[ ] PO has reviewed and accepted the story (Phase 6 sign-off)
```

---

## 8. Milestones

| Milestone | Date | Deliverable |
|-----------|------|-------------|
| Sprint 1 Start | 2026-03-04 | All Sprint 1 tasks created in ClickUp |
| Sprint 1 Dev Complete | 2026-03-14 | All S1 BE+FE tasks in Review |
| Sprint 1 QA Complete | 2026-03-17 | Sprint 1 pass rate >= 95%, stories accepted |
| Sprint 2 Start | 2026-03-18 | Sprint 2 tasks in progress |
| Sprint 2 Dev Complete | 2026-03-28 | All S2 BE+FE tasks in Review |
| Sprint 2 QA Complete | 2026-03-31 | Sprint 2 pass rate >= 95%, stories accepted |
| Sprint 3 Start | 2026-04-01 | Sprint 3 tasks in progress |
| Sprint 3 Dev Complete | 2026-04-09 | All S3 FE tasks in Review |
| Sprint 3 QA + Regression | 2026-04-14 | Full regression suite passes |
| Production Deploy | 2026-04-15 | EKS rollout + smoke test + PO sign-off |

---

## 9. Budget Forecast

| Resource | Sprint 1 | Sprint 2 | Sprint 3 | Total |
|----------|----------|----------|----------|-------|
| Backend Developer (AI agent hours) | ~20h | ~25h | ~5h | ~50h |
| Frontend Developer (AI agent hours) | ~15h | ~12h | ~22h | ~49h |
| QA Agent (AI agent hours) | ~8h | ~10h | ~12h | ~30h |
| Contingency (20%) | — | — | — | ~26h |
| **Total** | | | | **~155h** |

Infrastructure cost: $0 incremental (existing EKS cluster, no new resources).

---

## Handoff: Phase 3 → Phase 4

```
## Handoff: Phase 3 → Phase 4
- Completed by: project-manager agent
- Output files: /docs/auth-admin-ui-v2/project-management-plan.md
- ClickUp structure: Created by Orchestrator (see main report)
- Key decisions:
  - 3 sprints, 6 weeks total (2026-03-04 → 2026-04-14)
  - Sprint 1 focuses on all Must Have fixes (foundation)
  - Sprint 2 adds all new endpoints (Must Have features)
  - Sprint 3 adds Should Have / Could Have UI improvements
  - BE and FE tasks within the same sprint run in parallel
  - QA runs at end of each sprint against that sprint's stories
- Gate status: Passed (capacity ≤ 80%, every story has dev + QA task, sprint plan defined)
- Approved by: automatic
```
