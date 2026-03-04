# Product Requirements Document
## TigerSoft Auth — User Management Portal
**Version:** 2.0
**Date:** 2026-03-04
**Status:** Approved for Development

---

## Vision

Evolve the current Admin-only Console into a **User Management Portal** — a single URL (`auth-admin.tgstack.dev`) that serves both system admins (full control) and regular users (self-service profile, password, session management). Every touchpoint must feel like TigerSoft.

---

## Brand Alignment Delta (CI Guide → Code)

| Token | Current (Wrong) | New (CI Guide) |
|---|---|---|
| Primary red | `#C10016` | `#F4001A` Vivid Red |
| Primary text | `#222222` semi-black | `#0B1F3A` Oxford Blue |
| Secondary text | `#666666` semi-grey | `#A3A3A3` Quick Silver |
| Border / divider | `#e5e7eb` | `#DBE1E1` Serene |
| Page background | `#f5f5f5` | `#FFFFFF` white + `#DBE1E1` Serene subtle |
| Success / active | none | `#34D186` UFO Green |
| English font | Poppins | **Plus Jakarta Sans** (Google Fonts) |
| Thai font | Noto Sans Thai | **FC Vision** (custom, `guide/CI Toolkit/Font/TH/FC Vision/`) |
| Logo mark | Custom hexagon SVG | **Tiger claw stripe mark** (`guide/CI Toolkit/File Assets/Logo/PNG/03 Logo mark/`) |
| Card radius | `rounded-[10px]` | Keep (8–16px — brand guideline: soft edges) |
| Button radius | `rounded-[1000px]` | Keep (fully rounded pill — soft edges) |

---

## Role & Permission Matrix

| Role | Admin Console | User Portal |
|---|---|---|
| `super_admin` | All pages | Own profile |
| `admin` | Dashboard, Users, Roles, Audit, Settings | Own profile |
| `user` | ❌ Redirect to `/me` | Own profile only |

### Sidebar Visibility Rules
| Menu Item | `super_admin` | `admin` | `user` |
|---|---|---|---|
| Dashboard | ✅ | ✅ | ❌ |
| Tenants | ✅ | ❌ | ❌ |
| Users | ✅ | ✅ | ❌ |
| Roles | ✅ | ✅ | ❌ |
| Audit Log | ✅ | ✅ | ❌ |
| Settings | ✅ | ✅ | ❌ |
| My Profile | ✅ | ✅ | ✅ |

### Action Button Visibility Rules
| Action | `super_admin` | `admin` | `user` |
|---|---|---|---|
| Invite User | ✅ | ✅ | ❌ hidden |
| Suspend User | ✅ | ✅ | ❌ hidden |
| Resend Invite | ✅ | ✅ | ❌ hidden |
| Send Password Reset | ✅ | ✅ | ❌ hidden |
| Assign Roles | ✅ | ✅ | ❌ hidden |
| Create Tenant | ✅ | ❌ hidden | ❌ hidden |
| Manage Modules | ✅ | ❌ hidden | ❌ hidden |
| Delete Role | ✅ | ✅ | ❌ hidden |

---

## Epic 1 — Brand Alignment (Must Have)

### US-01 — Update Color Tokens
**As a** user visiting the portal,
**I want** the UI to use official TigerSoft brand colors,
**So that** the product feels consistent with all other TigerSoft touchpoints.

**Acceptance Criteria:**
- All `#C10016` (tiger-red) replaced with `#F4001A` (vivid-red)
- All `#222222` / semi-black replaced with `#0B1F3A` (oxford-blue)
- All `#666666` / semi-grey replaced with `#A3A3A3` (quick-silver)
- Active nav background: `#FFF0F2` → lightest tint of Vivid Red
- Active nav text/icon: `#F4001A`
- Success badges / active status: `#34D186` (ufo-green)
- Border and divider: `#DBE1E1` (serene)
- Page background: white `#FFFFFF` (dominant), `#DBE1E1` for sidebar/subtle areas

### US-02 — Update Typography
**As a** user visiting the portal,
**I want** text to render in the official brand fonts,
**So that** the product is visually consistent with TigerSoft brand.

**Acceptance Criteria:**
- English: **Plus Jakarta Sans** loaded via Google Fonts (weights: 300, 500, 600)
- Thai: **FC Vision** loaded from local font files (`guide/CI Toolkit/Font/TH/FC Vision/`)
- Heading = Medium (500), Subheading = Medium (500), Paragraph = Light (300)
- Remove all references to Poppins, Inter, Noto Sans Thai

### US-03 — Replace Logo Mark
**As a** user,
**I want** to see the official TigerSoft tiger-claw logo mark in the sidebar and auth pages,
**So that** the brand is correctly represented.

**Acceptance Criteria:**
- Sidebar brand mark: official tiger-claw SVG (from `guide/...Logo mark/`)
- Accept-invite page header: official logo mark
- Login page header: official logo mark
- Inline SVG recreation of custom hexagon removed

---

## Epic 2 — Role-Based UI Visibility (Must Have)

### US-04 — Permission-Driven Sidebar
**As a** logged-in user,
**I want** the sidebar to only show pages I have permission to access,
**So that** I am never redirected to a 403 page from normal navigation.

**Acceptance Criteria:**
- `user` role: sidebar shows only "My Profile" link; no admin items
- `admin` role: sidebar shows Dashboard, Users, Roles, Audit Log, Settings, My Profile; NOT Tenants
- `super_admin`: all items including Tenants and My Profile
- Route guard: navigating to a restricted URL redirects to appropriate page (admin → `/dashboard`, user → `/me`)

### US-05 — Permission-Driven Action Buttons
**As an** admin,
**I want** action buttons to only appear when I have permission to use them,
**So that** users never see actions that will result in an error.

**Acceptance Criteria:**
- Invite User button: hidden if `!isAdmin`
- Suspend / Enable button: hidden if `!isAdmin`
- Resend Invite button: hidden if `!isAdmin`
- Send Password Reset button: hidden if `!isAdmin`
- Assign Roles section: hidden if `!isAdmin`
- Manage Modules on tenant: hidden if `!isSuperAdmin`
- Create Tenant button: hidden if `!isSuperAdmin`
- `user` role accessing `/dashboard`: redirected to `/me` with "Your account does not have admin access" message

---

## Epic 3 — Invite Flow with Role Assignment (Must Have / Bug Fix)

### US-06 — Invite with Initial Role
**As an** admin,
**I want** to select a role when inviting a new user,
**So that** the invited user can log in and use the system immediately without a follow-up role assignment step.

**Acceptance Criteria:**
- Invite modal: add required "Role" dropdown (options: `admin`, `user`)
- `super_admin` option NOT available in dropdown
- Backend: `POST /api/v1/admin/users/invite` accepts `initial_role: string`
- After invite accepted: user's JWT contains the assigned role
- If user logs in after accepting invite, they are NOT shown 403 on any permitted page
- Existing invited users (unverified, no role) can still be resent — they receive `user` role if not set

---

## Epic 4 — Password Management (Must Have)

### US-07 — Admin Sends Password Reset
**As an** admin,
**I want** to send a password reset link to any user from their profile page,
**So that** I can help users who are locked out without knowing their password.

**Acceptance Criteria:**
- User detail page: "Send Password Reset" button (visible to `admin` and `super_admin`)
- Clicking triggers `POST /api/v1/auth/forgot-password` (with user's email)
- Success toast: "Password reset email sent to [email]"
- Backend endpoint already exists ✅

### US-08 — User Self-Service Forgot Password
**As a** user who forgot my password,
**I want** to request a password reset email from the login page,
**So that** I can regain access without contacting an admin.

**Acceptance Criteria:**
- Login page: "Forgot password?" link below the sign-in button
- Clicking opens `/forgot-password` page with email input
- On submit: calls `POST /api/v1/auth/forgot-password`
- Always shows success message (anti-enumeration: never reveal if email exists)
- Password reset link in email → `/reset-password?token=...&tenant=...` page
- User enters new password + confirm → calls `POST /api/v1/auth/reset-password`
- On success: redirects to login with "Password updated. Please sign in."
- Backend endpoints already exist ✅

---

## Epic 5 — Super Admin: Tenant Module Management (Should Have)

### US-09 — Manage Enabled Modules per Tenant
**As a** super admin,
**I want** to enable or disable product modules for each tenant from the tenant detail page,
**So that** I can control which TigerSoft products each customer has access to.

**Acceptance Criteria:**
- Tenant detail page: "Enabled Modules" section with multi-select checkbox list
- Available modules (initial): `recruitment`
- Admin-only toggle: changes are saved via `PATCH /api/v1/admin/tenants/:id` with `enabled_modules`
- Confirmation dialog before saving
- Success toast on save; error toast on failure
- Visible only to `super_admin`

---

## Epic 6 — User Self-Service Portal (Should Have)

### US-10 — My Profile Page
**As any** logged-in user,
**I want** a "My Profile" page where I can view and update my own account information,
**So that** I can manage my identity without contacting an admin.

**Acceptance Criteria:**
- Route: `/me` (accessible to ALL roles including `user`)
- Sidebar: "My Profile" item always visible to authenticated users
- Profile page shows: display name, email (read-only), tenant, assigned roles (read-only)
- User can edit: Display Name
- Save button calls `PUT /api/v1/users/me` (new backend endpoint needed)

### US-11 — Self-Service Change Password
**As any** logged-in user,
**I want** to change my password from My Profile,
**So that** I can keep my account secure without admin involvement.

**Acceptance Criteria:**
- Profile page: "Security" section with "Change Password" form
- Fields: Current Password, New Password, Confirm New Password
- New password: min 8 chars
- Calls `PUT /api/v1/users/me/password` (new backend endpoint needed)
- Success: toast and form reset
- Wrong current password: "Current password is incorrect" error inline

### US-12 — View My Roles & Permissions
**As any** logged-in user,
**I want** to see what roles and permissions I have,
**So that** I understand what I can and cannot do in the system.

**Acceptance Criteria:**
- Profile page: "Roles & Access" section
- Shows: System Roles (e.g., admin), Module Roles (e.g., recruitment: recruiter)
- Read-only — user cannot change their own roles

---

## Epic 7 — UX Polish (Could Have)

### US-13 — Empty State for No-Permission
**As a** user with no admin role,
**I want** a clear, helpful page when I don't have access,
**So that** I understand what to do next.

**Acceptance Criteria:**
- When `user` role navigates to `/dashboard/*` routes: show `/me` redirect, NOT a blank page or 403 error
- If no profile page yet: show a friendly "You don't have admin access. Contact your administrator." card at `/dashboard`

### US-14 — Refresh Behavior After Accept Invite
**As a** new user who just accepted an invitation,
**I want** to be automatically redirected to login after account activation,
**So that** I don't need to manually navigate.

**Acceptance Criteria:**
- After `AcceptInvite` success: auto-redirect to `/login?email=[email]` after 3 seconds
- Login page pre-fills email from query param

---

## Backlog Priority

| Epic | Stories | Priority | Sprint |
|---|---|---|---|
| Epic 3 — Invite + Role (Bug Fix) | US-06 | Must Have | Sprint 1 |
| Epic 4 — Password Reset (Self-service) | US-07, US-08 | Must Have | Sprint 1 |
| Epic 1 — Brand Alignment | US-01, US-02, US-03 | Must Have | Sprint 1 |
| Epic 2 — Permission UI | US-04, US-05 | Must Have | Sprint 1 |
| Epic 5 — Tenant Modules | US-09 | Should Have | Sprint 2 |
| Epic 6 — User Portal | US-10, US-11, US-12 | Should Have | Sprint 2 |
| Epic 7 — UX Polish | US-13, US-14 | Could Have | Sprint 2 |

---

## New Backend Endpoints Required

| Method | Path | Description | Status |
|---|---|---|---|
| `POST` | `/api/v1/auth/forgot-password` | Request password reset email | ✅ Exists |
| `POST` | `/api/v1/auth/reset-password` | Submit new password with token | ✅ Exists |
| `GET` | `/api/v1/users/me` | Get own profile | ❌ New |
| `PUT` | `/api/v1/users/me` | Update own display name | ❌ New |
| `PUT` | `/api/v1/users/me/password` | Change own password | ❌ New |
| `POST` | `/api/v1/admin/users/invite` | + `initial_role` field | ⚠️ Modify |
| `PATCH` | `/api/v1/admin/tenants/:id` | + `enabled_modules` | ⚠️ Verify |

---

## Flow Validation Checklist

| # | Flow | Target Sprint | Backend | Frontend |
|---|---|---|---|---|
| 1 | Super admin logs in | Done | ✅ | ✅ |
| 2 | Admin logs in | Done | ✅ | ✅ |
| 3 | User with no roles → redirected to profile or blocked message | Sprint 1 | ✅ | ❌ |
| 4 | Super admin creates tenant | Done | ✅ | ✅ |
| 5 | Super admin suspends/activates tenant | Done | ✅ | ✅ |
| 6 | Super admin manages enabled_modules | Sprint 2 | ✅ | ❌ |
| 7 | Admin invites user **with role** | Sprint 1 | ❌ | ❌ |
| 8 | Invited user accepts → logs in with correct role → no 403 | Sprint 1 | ❌ | ❌ |
| 9 | Admin resends invite | Done | ✅ | ✅ |
| 10 | Admin sends password reset to user | Sprint 1 | ✅ | ❌ |
| 11 | User self-service forgot password | Sprint 1 | ✅ | ❌ |
| 12 | User resets password via email link | Sprint 1 | ✅ | ❌ |
| 13 | Admin assigns/changes roles | Done | ✅ | ✅ |
| 14 | Admin suspends user | Done | ✅ | ✅ |
| 15 | Sidebar hides menus by role | Sprint 1 | N/A | ❌ |
| 16 | Action buttons hidden when no permission | Sprint 1 | N/A | ❌ |
| 17 | Admin views audit log | Done | ✅ | ✅ |
| 18 | Admin updates tenant settings | Done | ✅ | ✅ |
| 19 | User views/edits own profile | Sprint 2 | ❌ | ❌ |
| 20 | User changes own password | Sprint 2 | ❌ | ❌ |
| 21 | Token refresh / session expiry handling | Done | ✅ | ✅ |
| 22 | Logout invalidates session | Done | ✅ | ✅ |
| 23 | Brand tokens match CI guide | Sprint 1 | N/A | ❌ |
| 24 | Plus Jakarta Sans font loaded | Sprint 1 | N/A | ❌ |
| 25 | Official TigerSoft logo mark shown | Sprint 1 | N/A | ❌ |
