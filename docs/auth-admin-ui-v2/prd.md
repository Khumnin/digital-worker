# Product Requirements Document
## Auth Admin UI v2 — Backend Integration

**Version:** 1.0
**Date:** 2026-03-04
**Status:** Approved — Locked
**Author:** Product Owner Agent
**Project:** Auth Admin UI v2 — Backend Integration

---

## 1. Problem Statement

The Auth Admin UI (https://auth-admin.tgstack.dev) and the Auth Backend (https://auth.tgstack.dev) were developed independently. As a result, the two systems have diverged on API contracts, field naming conventions, data shapes, and HTTP semantics. The frontend currently cannot reliably communicate with the backend, causing authentication failures, broken user management flows, and non-functional tenant administration.

**Core problem:** The systems exist but do not work together. This integration project closes every gap between them so that tenant administrators and platform super-admins can manage users, roles, tenants, and security settings through a functional, production-ready admin interface.

**Who is affected:**
- Tenant administrators who cannot manage users or configure tenant security settings
- Platform super-admins (Tigersoft operations team) who cannot onboard new tenants
- End users who cannot be invited, activated, or have their roles updated

**Why it matters:**
- The Recruitment module (first V4 consumer) is blocked until Auth Admin works end-to-end
- Client onboarding cannot happen until tenant management is functional
- Security posture is compromised: user role assignments are untested and potentially broken

---

## 2. Goals & Success Metrics

| Goal | Success Metric | Target |
|------|---------------|--------|
| All API calls return the correct shape | Zero 4xx errors from contract mismatches in E2E test suite | 100% pass rate |
| JWT carries tenant slug (not UUID) | Authenticated requests to downstream services pass tenant check | 100% of JWTs |
| User invitation flow functional | Admin invites user → user receives magic-link → sets password → logs in | Full flow passes E2E |
| Role assignment functional | Admin assigns system + module roles → JWT reflects changes on next login | Correct roles in JWT |
| Tenant management functional | Super-admin creates/suspends/activates tenant → state persists | All state transitions pass |
| Frontend type safety | Zero TypeScript type errors in CI | 0 errors |
| Pagination consistent | All list endpoints use `page`/`page_size` | All endpoints |
| Error handling uniform | All errors use `{ error: { code, message } }` shape | All error paths |

---

## 3. User Personas

### Persona 1: Tenant Administrator (Admin)
- **Name:** Siriwan K.
- **Role:** HR Director / IT Admin at a client company
- **Technical level:** Non-technical; uses admin UI only
- **Goals:** Invite employees to the system, assign roles, deactivate users who leave, configure MFA and session policies
- **Pain points:** Currently receives no feedback when user invitation fails; role dropdowns are empty; cannot see which users are active vs. pending; audit log does not load
- **Permissions:** `admin` role on their tenant schema; sees Users, Roles, Settings, Audit Log sections

### Persona 2: Platform Super-Admin (Tigersoft Operations)
- **Name:** Thanakorn W.
- **Role:** Tigersoft platform engineer / operations
- **Technical level:** Technical; manages the multi-tenant platform
- **Goals:** Onboard new client tenants, set enabled modules per tenant, suspend/activate tenants, rotate API credentials for M2M integrations
- **Pain points:** Tenant list shows broken data (UUID in tenant_id field, schema_name exposed); suspend action returns 405 Method Not Allowed; tenant creation fails silently
- **Permissions:** `super_admin` role on the platform tenant; sees full Tenants management section plus all admin sections

### Persona 3: New Employee (End User / Applicant)
- **Name:** Nattapong B.
- **Role:** Newly hired employee at a client company
- **Technical level:** Non-technical
- **Goals:** Accept invitation, set a password, log in, access the modules they are assigned to
- **Pain points:** Magic-link invitation email never arrives (backend bug); login shows JWT errors
- **Permissions:** Initially `pending` status; transitions to `active` after accepting invitation

### Persona 4: Recruitment Module Backend (Machine-to-Machine Client)
- **Name:** Recruitment Service (M2M)
- **Role:** Automated service consumer of Auth API
- **Technical level:** N/A (software system)
- **Goals:** Verify JWT tokens via JWKS, use client_credentials grant for API calls, receive correct `tenant_id` (slug) in JWT claims
- **Pain points:** JWT `tenant_id` is currently a UUID, causing tenant resolution failures in the Recruitment backend
- **Permissions:** API credentials (client_id + client_secret) per tenant

---

## 4. MoSCoW Prioritized Feature List

### Must Have (P1 — Sprint 1 & 2)
1. **[BE] JWT tenant_id fix** — JWT must carry tenant slug, not UUID
2. **[BE] JWT refresh tenant_id fix** — Token refresh must also carry slug
3. **[BE] Pagination standardization** — All list endpoints: `page`/`page_size` (not `limit`/`offset`)
4. **[BE] Primary key rename** — All `*_id` fields in responses renamed to `id`
5. **[BE] User status normalization** — `invited→pending`, `disabled→inactive`
6. **[BE] HTTP method fixes** — `PUT /users/:id/disable → POST`, `PUT /tenants/:id/suspend → POST`
7. **[BE] Error format standardization** — All errors: `{ error: { code, message } }`
8. **[BE] Audit log field renames** — `event_type→action`, `actor_ip→ip_address`, `occurred_at→created_at`, `target_user_id→target_id`
9. **[BE] Remove schema_name from tenant responses**
10. **[BE] Roles module column** — Add `module` column to roles table
11. **[BE] New endpoints** — GET /admin/users/:id, PUT /admin/users/:id/roles, POST /admin/users/:id/enable, GET/PUT /admin/tenant, POST /admin/tenants/:id/activate
12. **[FE] Fix refresh token URL** — `/api/v1/auth/refresh → /api/v1/auth/token/refresh`
13. **[FE] Fix error parsing** — `err.message → err.error?.message`
14. **[FE] Remove X-Tenant-ID header** from authenticated requests
15. **[FE] Update all TypeScript interfaces** to match new API contract

### Should Have (P2 — Sprint 2 & 3)
16. **[FE] Roles page: module tabs** — Group roles by module in the UI
17. **[FE] User detail: two role sections** — Separate system roles from module roles
18. **[FE] Module role assignment in user form** — Support assigning module roles to users
19. **[BE] Module role seeding** — Seed `recruit`, `payroll`, `time` module roles at tenant creation
20. **[BE] Module role delete restriction** — Reject deletion of `is_system=true` roles or roles in use
21. **[FE] Tenant enabled_modules display** — Show enabled modules on tenant list and detail

### Could Have (P3 — Sprint 3)
22. **[FE] Audit log filters** — Filter by `action`, `from`, `to` date range in UI
23. **[FE] Tenant settings expanded form** — `allowed_domains`, `session_duration_minutes` fields
24. **[BE] Custom module role support** — Tenant admin can create custom roles scoped to a module
25. **[FE] User status filter** — Filter user list by `status` in UI

### Won't Have (Out of Scope for This Release)
- Applicant self-registration UI (handled by Recruitment module)
- OAuth PKCE flow UI (handled by Recruitment frontend)
- Vault integration changes
- Email template customization
- Multi-language (i18n) support
- Dark mode
- Mobile responsive layout changes

---

## 5. User Stories

### Epic 1: JWT & Auth Token Correctness

---

#### Story BE-001: JWT carries tenant slug
**Tag:** [BE]
**As a** Recruitment service consuming Auth JWTs,
**I want** the `tenant_id` claim in JWTs to contain the tenant's slug (e.g., `"platform"`) rather than the tenant's internal UUID,
**so that** downstream services can resolve tenants by slug without additional database lookups.

**Story Points:** 3
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: JWT tenant_id claim contains slug

  Scenario: Login returns JWT with slug in tenant_id
    Given a user "admin@henderson.co.th" belongs to tenant with slug "henderson"
    When the user POSTs to /api/v1/auth/login with valid credentials
    And the X-Tenant-ID header is "henderson"
    Then the response access_token JWT payload contains "tenant_id": "henderson"
    And "tenant_id" is NOT a UUID format

  Scenario: Refresh also returns JWT with slug in tenant_id
    Given a valid refresh_token was issued with slug in tenant_id
    When the user POSTs to /api/v1/auth/token/refresh with that refresh_token
    Then the new access_token JWT payload contains "tenant_id": "henderson"
    And the new refresh_token JWT payload contains "tenant_id": "henderson"
```

**Implementation notes:** Fix `auth_service.go:313` — change `TenantID: tenant.ID.String()` to `TenantID: tenant.Slug`. Fix `session_service.go:107` — ensure refresh tokens also use slug, not empty string.

---

#### Story BE-002: Token refresh URL and error parsing
**Tag:** [BE] + [FE]
**As an** authenticated user whose access token has expired,
**I want** the frontend to successfully refresh my token using the correct endpoint,
**so that** I am not logged out unexpectedly during normal use.

**Story Points:** 2
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Token refresh works end-to-end

  Scenario: Frontend calls the correct refresh endpoint
    Given a user is authenticated with a valid refresh_token
    When the access_token expires
    Then the frontend makes a POST request to "/api/v1/auth/token/refresh"
    And NOT to "/api/v1/auth/refresh"
    And the request body contains {"refresh_token": "<token>"}
    And the response contains a new access_token

  Scenario: Error messages are correctly parsed from API responses
    Given the API returns {"error": {"code": "invalid_token", "message": "Token has expired"}}
    When the frontend processes this error
    Then the user sees the message "Token has expired"
    And NOT "undefined" or an empty message
```

**Implementation notes (FE):** Fix `src/lib/api.ts:189` — update URL from `/api/v1/auth/refresh` to `/api/v1/auth/token/refresh`. Fix `src/lib/api.ts:169` — change `err.message` to `err.error?.message ?? "An unexpected error occurred"`.

---

### Epic 2: API Contract Alignment — Response Shape

---

#### Story BE-003: Standardize pagination to page/page_size
**Tag:** [BE]
**As a** frontend developer consuming list endpoints,
**I want** all paginated responses to use `page` and `page_size` parameters (not `limit`/`offset`),
**so that** pagination UI components work consistently across all list views.

**Story Points:** 5
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Consistent pagination contract

  Scenario: User list accepts page and page_size
    Given I am an authenticated admin
    When I GET /api/v1/admin/users?page=2&page_size=10
    Then the response contains {"data": [...], "total": N, "page": 2, "page_size": 10, "total_pages": M}
    And the data array contains at most 10 items

  Scenario: Tenant list accepts page and page_size
    When I GET /api/v1/admin/tenants?page=1&page_size=20
    Then the response meta contains "page": 1, "page_size": 20

  Scenario: Audit log accepts page and page_size
    When I GET /api/v1/admin/audit-log?page=1&page_size=50
    Then the response meta contains "page": 1, "page_size": 50

  Scenario: Legacy limit/offset parameters are rejected or ignored
    When I GET /api/v1/admin/users?limit=10&offset=0
    Then the response uses default pagination (page=1, page_size=20)
```

---

#### Story BE-004: Rename primary key fields to `id` in all responses
**Tag:** [BE]
**As a** frontend developer rendering data from the API,
**I want** every object in every API response to use `id` as the primary key field name,
**so that** I can write consistent TypeScript interfaces and rendering logic without per-entity field mapping.

**Story Points:** 3
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Consistent primary key field name

  Scenario: User object uses "id" not "user_id"
    Given I GET /api/v1/admin/users/:id
    Then the response object contains "id": "<uuid>"
    And does NOT contain "user_id"

  Scenario: Tenant object uses "id" not "tenant_id"
    Given I GET /api/v1/admin/tenants/:id
    Then the response object contains "id": "<uuid>"
    And does NOT contain "tenant_id"

  Scenario: Role object uses "id" not "role_id"
    Given I GET /api/v1/admin/roles
    Then each role in the array contains "id": "<uuid>"
    And does NOT contain "role_id"

  Scenario: Audit log entry uses "id" not "log_id" or "audit_id"
    Given I GET /api/v1/admin/audit-log
    Then each entry in the array contains "id": "<uuid>"
```

---

#### Story BE-005: Normalize user status values
**Tag:** [BE] + [FE]
**As a** tenant administrator viewing the user list,
**I want** user status to be one of `active`, `inactive`, or `pending`,
**so that** I understand each user's lifecycle state in clear, unambiguous terms.

**Story Points:** 3
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: User status normalization

  Scenario: Invited user shows as pending
    Given admin invites a new user via POST /api/v1/admin/users/invite
    When I GET /api/v1/admin/users
    Then the newly created user has "status": "pending"
    And NOT "status": "invited"

  Scenario: Disabled user shows as inactive
    Given a user has been disabled via POST /api/v1/admin/users/:id/disable
    When I GET /api/v1/admin/users/:id
    Then the user has "status": "inactive"
    And NOT "status": "disabled"

  Scenario: Re-enabled user shows as active
    Given a user has "status": "inactive"
    When admin POSTs to /api/v1/admin/users/:id/enable
    Then the user has "status": "active"

  Scenario: Frontend displays correct status badge
    Given the API returns a user with "status": "pending"
    Then the UI displays a "Pending" badge in yellow/orange color
    Given the API returns a user with "status": "inactive"
    Then the UI displays an "Inactive" badge in gray/red color
    Given the API returns a user with "status": "active"
    Then the UI displays an "Active" badge in green color
```

---

#### Story BE-006: Fix HTTP methods — POST for state transitions
**Tag:** [BE]
**As a** frontend developer calling state-change actions,
**I want** all state transition endpoints to use POST (not PUT),
**so that** REST semantics are correct and browser CORS preflight works as expected.

**Story Points:** 2
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Correct HTTP methods for state transitions

  Scenario: Disable user uses POST
    Given I am an authenticated admin
    When I POST to /api/v1/admin/users/:id/disable
    Then I receive 204 No Content
    And NOT 405 Method Not Allowed

  Scenario: Enable user uses POST
    When I POST to /api/v1/admin/users/:id/enable
    Then I receive 204 No Content

  Scenario: Suspend tenant uses POST
    Given I am an authenticated super_admin
    When I POST to /api/v1/admin/tenants/:id/suspend
    Then I receive 204 No Content
    And NOT 405 Method Not Allowed

  Scenario: Activate tenant uses POST
    When I POST to /api/v1/admin/tenants/:id/activate
    Then I receive 204 No Content
```

---

#### Story BE-007: Standardize error response format
**Tag:** [BE] + [FE]
**As a** frontend developer handling API errors,
**I want** all error responses to use the shape `{ "error": { "code": "string", "message": "string" } }`,
**so that** I can write a single error handler that works for all endpoints.

**Story Points:** 3
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Uniform error response format

  Scenario: 400 validation error uses standard format
    Given I POST /api/v1/admin/users/invite with missing email
    Then the response status is 400
    And the body is {"error": {"code": "validation_error", "message": "email is required"}}

  Scenario: 401 unauthorized uses standard format
    Given I call any protected endpoint without a token
    Then the response status is 401
    And the body is {"error": {"code": "unauthorized", "message": "..."}}

  Scenario: 404 not found uses standard format
    Given I GET /api/v1/admin/users/nonexistent-uuid
    Then the response status is 404
    And the body is {"error": {"code": "not_found", "message": "user not found"}}

  Scenario: 409 conflict uses standard format
    Given a role is in use by users
    When I DELETE /api/v1/admin/roles/:id
    Then the response status is 409
    And the body is {"error": {"code": "role_in_use", "message": "..."}}

  Scenario: Frontend parses error correctly
    Given the API returns {"error": {"code": "validation_error", "message": "email is required"}}
    Then the frontend displays "email is required" to the user
    And NOT "undefined"
```

---

### Epic 3: Audit Log Field Alignment

---

#### Story BE-008: Rename audit log fields to match contract
**Tag:** [BE] + [FE]
**As a** tenant administrator reviewing the audit log,
**I want** audit log entries to use the field names defined in the API contract,
**so that** the frontend can render audit log entries without custom field mapping.

**Story Points:** 3
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Audit log field name alignment

  Scenario: Audit log entry uses correct field names
    Given I GET /api/v1/admin/audit-log?page=1&page_size=20
    Then each entry contains:
      | Field         | Old Name          |
      | action        | event_type        |
      | ip_address    | actor_ip          |
      | created_at    | occurred_at       |
      | target_id     | target_user_id    |
    And does NOT contain "event_type", "actor_ip", "occurred_at", or "target_user_id"

  Scenario: Audit log supports filtering by action
    When I GET /api/v1/admin/audit-log?action=user.login
    Then all entries in the response have "action": "user.login"

  Scenario: Audit log supports date range filtering
    When I GET /api/v1/admin/audit-log?from=2026-01-01&to=2026-03-01
    Then all entries have created_at within the specified range
```

---

### Epic 4: User Management — New Endpoints

---

#### Story BE-009: GET single user endpoint
**Tag:** [BE]
**As a** tenant administrator,
**I want** to fetch a single user's full profile including their system and module roles,
**so that** I can view and edit a user's details without loading the entire user list.

**Story Points:** 3
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Single user retrieval

  Scenario: Fetch user by ID returns full profile
    Given I am an authenticated admin
    When I GET /api/v1/admin/users/:id with a valid user UUID
    Then the response status is 200
    And the response body matches the User object shape:
      {
        "id": "<uuid>",
        "email": "...",
        "first_name": "...",
        "last_name": "...",
        "status": "active|inactive|pending",
        "email_verified": true,
        "mfa_enabled": false,
        "system_roles": [{"id": "<uuid>", "name": "admin"}],
        "module_roles": {"recruit": [{"id": "<uuid>", "name": "recruiter"}]},
        "created_at": "..."
      }

  Scenario: Fetch non-existent user returns 404
    When I GET /api/v1/admin/users/00000000-0000-0000-0000-000000000000
    Then the response status is 404
    And the body contains {"error": {"code": "not_found", "message": "user not found"}}
```

---

#### Story BE-010: User role assignment endpoint
**Tag:** [BE]
**As a** tenant administrator,
**I want** to replace a user's complete set of roles via a single API call,
**so that** role management is atomic and not subject to partial-update race conditions.

**Story Points:** 5
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: User role assignment

  Scenario: Replace user roles with a new set
    Given a user currently has roles ["user"]
    And I have role UUIDs for "admin" and "recruiter"
    When I PUT /api/v1/admin/users/:id/roles with {"role_ids": ["<admin-uuid>", "<recruiter-uuid>"]}
    Then the response status is 204
    And GET /api/v1/admin/users/:id returns system_roles containing "admin"
    And module_roles.recruit containing "recruiter"

  Scenario: Assigning empty role_ids removes all roles
    When I PUT /api/v1/admin/users/:id/roles with {"role_ids": []}
    Then the response status is 204
    And GET /api/v1/admin/users/:id returns empty system_roles and module_roles

  Scenario: Invalid role UUID returns 422
    When I PUT /api/v1/admin/users/:id/roles with {"role_ids": ["invalid-uuid"]}
    Then the response status is 422
    And the body contains {"error": {"code": "invalid_role_id", "message": "..."}}

  Scenario: JWT reflects new roles on next login
    Given a user's roles were updated to include "admin"
    When the user logs in fresh (not refreshes)
    Then the JWT access_token payload contains "roles": ["user", "admin"]
```

---

### Epic 5: RBAC — Module-scoped Roles

---

#### Story BE-011: Add module column to roles table
**Tag:** [BE]
**As a** tenant administrator managing roles,
**I want** each role to be associated with either no module (system role) or a specific module,
**so that** module roles are organized separately from system roles and I can filter them.

**Story Points:** 5
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Role module association

  Scenario: System roles have null module
    Given I GET /api/v1/admin/roles
    Then "user", "admin", "super_admin" roles have "module": null
    And "is_system": true for those roles

  Scenario: Module roles carry their module name
    Given the tenant has module "recruit" enabled
    Then GET /api/v1/admin/roles returns roles with "module": "recruit"
    And roles for recruit include "recruiter", "hiring_manager", "interviewer"

  Scenario: Filter roles by module
    When I GET /api/v1/admin/roles?module=recruit
    Then all returned roles have "module": "recruit"

  Scenario: Create new module role requires module field
    When I POST /api/v1/admin/roles with {"name": "my-role", "module": "recruit"}
    Then the response status is 201
    And the new role has "module": "recruit", "is_system": false

  Scenario: Create role without module field returns 400
    When I POST /api/v1/admin/roles with {"name": "my-role"}
    Then the response status is 400
    And the body contains {"error": {"code": "validation_error", "message": "module is required"}}
```

---

#### Story BE-012: Protect system roles from deletion
**Tag:** [BE]
**As a** platform super-admin,
**I want** seeded system roles to be undeletable,
**so that** tenant admins cannot accidentally break role-based access control.

**Story Points:** 2
**Priority:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: System role deletion protection

  Scenario: Delete system role is rejected
    Given the role "admin" has "is_system": true
    When I DELETE /api/v1/admin/roles/:id for the admin role
    Then the response status is 409
    And the body contains {"error": {"code": "system_role_protected", "message": "..."}}

  Scenario: Delete role in use by users is rejected
    Given the role "recruiter" is assigned to at least one user
    When I DELETE /api/v1/admin/roles/:id for recruiter
    Then the response status is 409
    And the body contains {"error": {"code": "role_in_use", "message": "..."}}

  Scenario: Delete unused custom role succeeds
    Given a custom role exists with is_system=false and no users assigned
    When I DELETE /api/v1/admin/roles/:id for that role
    Then the response status is 204
```

---

### Epic 6: Tenant Management Fixes

---

#### Story BE-013: Remove schema_name from tenant responses
**Tag:** [BE] + [FE]
**As a** tenant administrator,
**I want** the tenant API response to not expose internal database implementation details,
**so that** the UI does not accidentally display or leak internal schema names.

**Story Points:** 2
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Tenant response excludes schema_name

  Scenario: Tenant list does not include schema_name
    Given I am a super_admin
    When I GET /api/v1/admin/tenants
    Then no tenant object in the response contains "schema_name"

  Scenario: Single tenant does not include schema_name
    When I GET /api/v1/admin/tenants/:id
    Then the response body does NOT contain "schema_name"

  Scenario: Tenant object shape matches contract
    Then the tenant response matches:
      {
        "id": "<uuid>",
        "name": "string",
        "slug": "string",
        "status": "active|suspended",
        "enabled_modules": ["string"],
        "created_at": "string"
      }
```

---

#### Story BE-014: Tenant settings GET/PUT endpoint
**Tag:** [BE] + [FE]
**As a** tenant administrator,
**I want** to view and update my tenant's security settings including MFA requirements, session duration, and allowed domains,
**so that** I can configure security policies appropriate for my organization without calling Tigersoft support.

**Story Points:** 5
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Tenant settings management

  Scenario: Admin retrieves current tenant settings
    Given I am an authenticated admin on tenant "henderson"
    When I GET /api/v1/admin/tenant
    Then the response status is 200
    And the body contains:
      {
        "id": "<uuid>",
        "name": "Henderson Corp",
        "slug": "henderson",
        "status": "active",
        "mfa_required": false,
        "session_duration_minutes": 60,
        "allowed_domains": ["henderson.co.th"],
        "enabled_modules": ["recruit"]
      }

  Scenario: Admin updates MFA requirement
    When I PUT /api/v1/admin/tenant with {"mfa_required": true}
    Then the response status is 200
    And GET /api/v1/admin/tenant returns "mfa_required": true

  Scenario: Admin updates session duration
    When I PUT /api/v1/admin/tenant with {"session_duration_minutes": 120}
    Then the response status is 200
    And GET /api/v1/admin/tenant returns "session_duration_minutes": 120

  Scenario: Admin updates allowed domains
    When I PUT /api/v1/admin/tenant with {"allowed_domains": ["henderson.co.th", "henderson.com"]}
    Then the response status is 200
    And GET /api/v1/admin/tenant returns "allowed_domains": ["henderson.co.th", "henderson.com"]
```

---

#### Story BE-015: Tenant activation endpoint (super_admin)
**Tag:** [BE]
**As a** platform super-admin,
**I want** to activate a previously suspended tenant,
**so that** I can restore service to a tenant after an issue is resolved.

**Story Points:** 2
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: Tenant activation

  Scenario: Super-admin activates a suspended tenant
    Given a tenant exists with "status": "suspended"
    When I POST /api/v1/admin/tenants/:id/activate
    Then the response status is 204
    And GET /api/v1/admin/tenants/:id returns "status": "active"

  Scenario: Activating an already-active tenant is idempotent
    Given a tenant exists with "status": "active"
    When I POST /api/v1/admin/tenants/:id/activate
    Then the response status is 204
    And the tenant remains "status": "active"

  Scenario: Suspending a tenant
    Given a tenant exists with "status": "active"
    When I POST /api/v1/admin/tenants/:id/suspend
    Then the response status is 204
    And GET /api/v1/admin/tenants/:id returns "status": "suspended"
```

---

### Epic 7: Frontend TypeScript & API Client Alignment

---

#### Story FE-001: Update TypeScript interfaces to match contract
**Tag:** [FE]
**As a** frontend developer,
**I want** all TypeScript type definitions to accurately reflect the current API contract,
**so that** the TypeScript compiler catches API integration errors at build time, not at runtime.

**Story Points:** 5
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: TypeScript interface accuracy

  Scenario: User interface uses system_roles and module_roles
    Given the User TypeScript interface exists
    Then it contains "system_roles: Role[]"
    And "module_roles: Record<string, Role[]>"
    And does NOT contain a flat "roles: Role[]" field

  Scenario: Tenant interface excludes schema_name and config
    Given the Tenant TypeScript interface exists
    Then it does NOT contain "schema_name" or "config"
    And it contains "enabled_modules: string[]"

  Scenario: Pagination interface uses page and page_size
    Given the PaginatedResponse TypeScript interface exists
    Then it contains "page: number" and "page_size: number" and "total_pages: number"
    And does NOT contain "limit" or "offset"

  Scenario: Error interface matches contract
    Given the ApiError TypeScript interface exists
    Then it has shape: { error: { code: string; message: string } }

  Scenario: TypeScript build passes with zero errors
    When "npm run build" or "tsc --noEmit" is executed
    Then the exit code is 0
    And no type errors are reported
```

---

#### Story FE-002: Remove X-Tenant-ID header from authenticated requests
**Tag:** [FE]
**As a** backend service receiving authenticated API calls,
**I want** the frontend to NOT send X-Tenant-ID headers on authenticated requests,
**so that** tenant identity is derived from the JWT (which is the authoritative source), preventing header-spoofing attacks.

**Story Points:** 2
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: X-Tenant-ID header removal from authenticated calls

  Scenario: Authenticated requests do not include X-Tenant-ID
    Given a user is logged in with a valid JWT
    When the frontend makes a GET request to /api/v1/admin/users
    Then the request headers do NOT include "X-Tenant-ID"
    And the backend correctly identifies the tenant from the JWT

  Scenario: Login request still includes X-Tenant-ID (pre-auth)
    When the frontend POSTs to /api/v1/auth/login
    Then the request headers DO include "X-Tenant-ID" (required for tenant resolution before JWT exists)
```

---

#### Story FE-003: Roles page — module tab grouping
**Tag:** [FE]
**As a** tenant administrator managing roles,
**I want** the Roles page to group roles by module using tabs,
**so that** I can quickly find and manage roles for a specific module without scrolling through an unsorted list.

**Story Points:** 5
**Priority:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: Roles page module tab grouping

  Scenario: Roles page shows System Roles tab by default
    Given I navigate to the Roles page
    Then I see a tab labeled "System" selected by default
    And it displays "user", "admin", "super_admin" roles
    And "is_system" badge shown for each

  Scenario: Module tabs appear for enabled modules
    Given the tenant has modules "recruit" and "payroll" enabled
    Then I see tabs "System", "Recruit", "Payroll"

  Scenario: Clicking a module tab shows that module's roles
    When I click the "Recruit" tab
    Then I see "recruiter", "hiring_manager", "interviewer" roles
    And a "Create Custom Role" button is visible

  Scenario: System roles tab does not show Create button
    When I view the System tab
    Then the "Create Role" button is NOT present for system roles
```

---

#### Story FE-004: User detail — system and module role sections
**Tag:** [FE]
**As a** tenant administrator editing a user's permissions,
**I want** the user detail/edit page to display system roles and module roles in separate sections,
**so that** I can clearly understand and manage a user's full permission set.

**Story Points:** 5
**Priority:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: User detail role sections

  Scenario: User detail page shows two role sections
    Given I navigate to /users/:id
    Then I see a "System Roles" section
    And a "Module Roles" section

  Scenario: System Roles section shows multi-select of system roles
    Given the system roles are "user", "admin", "super_admin"
    When I view the System Roles section
    Then I see checkboxes or a multi-select for each system role
    And the user's current system roles are pre-selected

  Scenario: Module Roles section is grouped by module
    Given the tenant has modules "recruit" and "payroll"
    When I view the Module Roles section
    Then I see a subsection for "Recruit" with checkboxes for recruit roles
    And a subsection for "Payroll" with checkboxes for payroll roles

  Scenario: Saving role changes calls PUT /users/:id/roles
    Given I check "admin" and "recruiter" for a user
    When I click Save
    Then the frontend calls PUT /api/v1/admin/users/:id/roles
    With body {"role_ids": ["<admin-uuid>", "<recruiter-uuid>"]}
    And a success toast appears
```

---

### Epic 8: User Invitation Flow

---

#### Story BE-016: User invitation creates pending user
**Tag:** [BE] + [FE]
**As a** tenant administrator,
**I want** to invite new users by email, which creates a pending account and sends them a magic-link,
**so that** employees can self-onboard without the administrator needing to set initial passwords.

**Story Points:** 5
**Priority:** Must Have

**Acceptance Criteria:**

```gherkin
Feature: User invitation flow

  Scenario: Admin invites a new user
    Given I am an authenticated admin
    When I POST /api/v1/admin/users/invite with {"email": "new@henderson.co.th", "first_name": "Nattapong", "last_name": "Buakhao"}
    Then the response status is 201
    And the response body contains a user object with "status": "pending"
    And an invitation email is sent to "new@henderson.co.th"
    And the email contains a magic-link URL

  Scenario: Duplicate email invitation returns 409
    Given a user with email "existing@henderson.co.th" already exists
    When I POST /api/v1/admin/users/invite with {"email": "existing@henderson.co.th", ...}
    Then the response status is 409
    And the body contains {"error": {"code": "email_already_exists", "message": "..."}}

  Scenario: Magic-link allows user to set password
    Given a user received an invitation magic-link
    When the user clicks the link and sets a password
    Then the user's status changes to "active"
    And the user can log in with their email and new password

  Scenario: Invitation appears in user list
    Given an invitation was just sent to "new@henderson.co.th"
    When I GET /api/v1/admin/users
    Then the list includes the new user with "status": "pending"
```

---

### Epic 9: Frontend Global Improvements

---

#### Story FE-005: User list filtering by status and role
**Tag:** [FE]
**As a** tenant administrator,
**I want** to filter the user list by status and by module/role,
**so that** I can quickly find users in a specific state (e.g., all pending users awaiting onboarding).

**Story Points:** 3
**Priority:** Could Have

**Acceptance Criteria:**

```gherkin
Feature: User list filtering

  Scenario: Filter by status=pending
    Given I am on the Users page
    When I select "Pending" from the status filter
    Then the API is called with ?status=pending
    And only users with "status": "pending" are displayed

  Scenario: Filter by module role
    When I select module "recruit" from the module filter
    Then the API is called with ?module=recruit
    And only users with at least one recruit module role are displayed

  Scenario: Clear filter shows all users
    When I click "Clear filters"
    Then the API is called without status or module parameters
    And all users are displayed
```

---

#### Story FE-006: Audit log filters in UI
**Tag:** [FE]
**As a** tenant administrator reviewing audit logs,
**I want** to filter audit log entries by action type and date range,
**so that** I can efficiently investigate specific security events.

**Story Points:** 3
**Priority:** Could Have

**Acceptance Criteria:**

```gherkin
Feature: Audit log filtering UI

  Scenario: Filter by action type
    Given I am on the Audit Log page
    When I select "user.login" from the action dropdown
    Then the API is called with ?action=user.login
    And only login events are shown

  Scenario: Filter by date range
    When I set from=2026-03-01 and to=2026-03-04
    Then the API is called with ?from=2026-03-01&to=2026-03-04
    And entries outside the range are not shown

  Scenario: Combined filters work together
    When I select action="user.login" AND from=2026-03-01
    Then the API is called with both parameters combined
```

---

## 6. Non-Functional Requirements

### NFR-001: Security
- All authenticated API endpoints must validate JWT signature and expiry before processing
- JWT must not expose `schema_name` or internal database identifiers in claims
- `X-Tenant-ID` header must not be trusted for tenant resolution on authenticated endpoints (use JWT claim instead)
- Role check must happen server-side; frontend role display is for UX only
- Invitation magic-links must expire after 24 hours
- Magic-link tokens must be single-use (invalidated after first use)
- All state-change operations (disable, enable, suspend, activate) must be logged to audit log

### NFR-002: Performance
- All list endpoints must respond within 500ms at p95 for page sizes up to 100
- Login endpoint must respond within 300ms at p95
- Frontend initial page load (LCP) must be under 2.5 seconds on 4G connection
- Pagination queries must use indexed columns; no full-table scans on paginated endpoints

### NFR-003: Reliability
- API must maintain 99.9% uptime (excluding planned maintenance)
- Graceful shutdown: all in-flight requests must complete before pod termination
- Redis failures must not cause login failures — JWT signing must function without Redis (Redis used for session revocation only)
- Database connection pool must handle 100 concurrent connections per pod

### NFR-004: Compatibility
- JWT RS256 signature algorithm must remain unchanged (Recruitment backend depends on current JWKS)
- JWKS endpoint (GET /.well-known/jwks.json) must remain available without breaking changes
- All existing API endpoints must remain backward-compatible in URL structure (only fields and methods change)
- Next.js version and TypeScript version must not be upgraded as part of this project

### NFR-005: Observability
- All API errors must be logged with request ID, tenant slug, user ID (if available), and error code
- Audit log must capture all admin actions (user invite, role change, tenant suspend, settings change)
- Frontend must not log tokens or sensitive user data to browser console

### NFR-006: Accessibility
- All form inputs must have accessible labels
- Status badges must not rely on color alone (include text label)
- Keyboard navigation must work for all critical flows (invite user, assign role, suspend tenant)

### NFR-007: Maintainability
- Backend: new endpoints must follow existing Handler → Service → Repository pattern
- Frontend: new components must follow existing shadcn/ui component conventions
- All database migrations must be backward-compatible (additive changes only in this release)
- TypeScript strict mode must be maintained; no `any` types introduced

---

## 7. Out of Scope

The following items are explicitly excluded from this release to prevent scope creep:

- Applicant self-registration UI or flow
- OAuth PKCE flow implementation or UI
- Password reset flow UI changes
- Two-factor authentication setup UI
- Email template customization
- Role permission matrix UI (granular permission management beyond role assignment)
- Tenant billing or subscription management
- User bulk import (CSV)
- Multi-language / i18n support
- Dark mode implementation
- Mobile responsive layout redesign
- Vault integration changes
- CI/CD pipeline changes
- Kubernetes manifest changes (unless required for a new environment variable)

---

## 8. Sprint Capacity Plan

**Team composition (AI agent squad):**
- 1 Backend Developer agent
- 1 Frontend Developer agent
- 1 QA/Tester agent
- Sprint length: 2 weeks
- Capacity per sprint: 40 story points per discipline at 80% = 32 story points max per sprint

**Velocity estimate:** 26–30 points per sprint (first sprint, conservative)

### Sprint 1 (Priority: Must Have — Foundation)

| Story | Points | Tag |
|-------|--------|-----|
| BE-001: JWT tenant_id fix | 3 | BE |
| BE-002: Token refresh fix | 2 | BE+FE |
| BE-003: Pagination standardization | 5 | BE |
| BE-004: Rename primary keys | 3 | BE |
| BE-005: Normalize user status | 3 | BE+FE |
| BE-006: Fix HTTP methods | 2 | BE |
| BE-007: Error format | 3 | BE+FE |
| FE-002: Remove X-Tenant-ID header | 2 | FE |
| FE-001: Update TypeScript interfaces | 5 | FE |
| **Total** | **28** | |

### Sprint 2 (Priority: Must Have — New Endpoints)

| Story | Points | Tag |
|-------|--------|-----|
| BE-008: Audit log field renames | 3 | BE+FE |
| BE-009: GET single user endpoint | 3 | BE |
| BE-010: User role assignment | 5 | BE |
| BE-011: Module column on roles | 5 | BE |
| BE-013: Remove schema_name | 2 | BE+FE |
| BE-014: Tenant settings endpoint | 5 | BE+FE |
| BE-015: Tenant activate endpoint | 2 | BE |
| BE-016: User invitation flow | 5 | BE+FE |
| **Total** | **30** | |

### Sprint 3 (Priority: Should Have + Could Have)

| Story | Points | Tag |
|-------|--------|-----|
| BE-012: Protect system roles | 2 | BE |
| FE-003: Roles page module tabs | 5 | FE |
| FE-004: User detail role sections | 5 | FE |
| FE-005: User list filtering | 3 | FE |
| FE-006: Audit log filters | 3 | FE |
| **Total** | **18** | |

**Total project:** 76 story points across 3 sprints (~6 weeks)

---

## 9. Glossary

| Term | Definition |
|------|-----------|
| Tenant | A client organization with its own isolated database schema |
| Slug | URL-safe short identifier for a tenant (e.g., `henderson`, `platform`) |
| System Role | A role with `module=null` that applies across the entire system (`user`, `admin`, `super_admin`) |
| Module Role | A role scoped to a specific module (`recruit`, `payroll`, `time`) |
| Magic-link | A one-time-use URL sent via email that authenticates the user and allows password setup |
| M2M | Machine-to-Machine — automated service-to-service authentication using client credentials |
| JWKS | JSON Web Key Set — the public key endpoint used by downstream services to verify JWTs |
| Pending | User status: invitation sent, password not yet set |
| Active | User status: account fully set up and enabled |
| Inactive | User status: account disabled by administrator |
