# Project Management Plan
# TGX Auth Console — Responsive Design Initiative

**Document version:** 1.0
**Date:** 2026-03-05
**Project Manager:** TigerSoft Project Manager Agent
**Methodology:** Agile / Scrum (3-sprint delivery)
**PRD Reference:** `docs/auth-system/responsive-design-prd.md`
**Architecture Reference:** `docs/auth-system/responsive-design-architecture.md`
**Project:** Auth Admin UI — Responsive Design

---

## Table of Contents

1. [Project Charter](#1-project-charter)
2. [Work Breakdown Structure (WBS)](#2-work-breakdown-structure-wbs)
3. [Sprint Plan and Timeline](#3-sprint-plan-and-timeline)
4. [ClickUp Task Structure](#4-clickup-task-structure)
5. [Risk Register](#5-risk-register)
6. [Definition of Done](#6-definition-of-done)
7. [RACI Chart](#7-raci-chart)
8. [Budget and Effort Forecast](#8-budget-and-effort-forecast)

---

## 1. Project Charter

### 1.1 Project Overview

| Field | Value |
|-------|-------|
| Project Name | TGX Auth Console — Responsive Design |
| Project Code | RESP-2026-01 |
| Client | TigerSoft (internal product) |
| Delivery Methodology | Agile / Scrum — 3 x 2-week sprints |
| Start Date | 2026-03-09 |
| Target Completion | 2026-04-17 (end of Sprint 3) |
| Release Readiness Date | 2026-04-03 (end of Sprint 2 — Must Have + Should Have complete) |

### 1.2 Problem Statement

The TGX Auth Console is a Next.js admin application built exclusively for desktop viewport widths. The fixed-width sidebar (298px expanded, 60px collapsed), persistent table layouts, and desktop-only flex patterns cause horizontal overflow, broken layouts, and unusably small touch targets on mobile devices (< 768px). End-users receiving email-based auth links (Accept Invite, Reset Password) open those links on mobile phones — creating broken onboarding experiences and increased support requests. Tenant Admins and Super Admins who need to perform time-sensitive administrative actions while away from their desks are blocked by the current implementation.

### 1.3 Project Objectives

1. Make all four auth pages (Login, Forgot Password, Reset Password, Accept Invite) fully usable on 375px mobile viewports by end of Sprint 1.
2. Implement a hamburger/drawer navigation pattern that replaces the persistent sidebar on mobile by end of Sprint 1.
3. Replace all five-column and six-column data tables with mobile card-stack patterns for Users, Tenants, Roles, and Audit Log by end of Sprint 2.
4. Achieve zero horizontal scrollbar on any page at 375px viewport width.
5. Achieve Lighthouse Mobile score >= 90 on `/login`, `/dashboard`, and `/dashboard/users`.
6. Ensure all interactive elements have a minimum 44 x 44px touch target (WCAG 2.5.5).
7. Preserve existing desktop layout pixel-identically — no visual regression on desktop.

### 1.4 Scope

**In Scope:**
- Frontend only: Next.js + Tailwind CSS v4 responsive class additions
- New shared components: `MobileDrawer`, `useBreakpoint` hook, 4 card sub-components
- 15 modified files + 6 new files (per architecture document)
- Playwright automated responsive test suite
- Manual QA across 7 viewport sizes

**Out of Scope (hard boundary):**
- Backend / Go/Gin API changes — zero backend work
- Native iOS/Android applications
- PWA capabilities (offline, push notifications)
- Mobile-first rewrite of existing class lists (desktop-first adaptation only per ADR-R01)
- Dark mode re-implementation (must not regress; no new dark-mode work)
- Email template responsive design
- New npm dependencies

### 1.5 Stakeholders

| Stakeholder | Role | Engagement |
|-------------|------|-----------|
| TigerSoft Product Team | Product Owner — story acceptance, sprint sign-off | Sprint reviews, PR approval |
| Solution Architect Agent | Architecture decisions, ADR ownership | Consulted on pattern decisions |
| Developer Agent | Implementation — all [FE] tasks | Responsible for all code |
| Tester Agent | QA — all [QA] tasks, Playwright suite | Responsible for all test tasks |
| Project Manager Agent | Planning, tracking, risk management | Accountable for delivery |

### 1.6 Key Constraints

| Constraint | Detail |
|-----------|--------|
| No new npm dependencies | All responsive work is pure Tailwind v4 + React state (ADR-R03) |
| Desktop layout must remain pixel-identical | PRD Section 8; ADR-R01 rationale |
| Card-stack pattern for all tables on mobile | ADR-R02 mandates this over horizontal scroll or column hiding |
| Touch targets minimum 44 x 44px | WCAG 2.5.5; applies to all interactive elements |
| 80% sprint capacity rule | Sprints planned with buffer allocation per PM standards |

### 1.7 Assumptions

- Developer and Tester agents are available full capacity for all three sprints.
- The existing codebase at commit `9483143` (branch: `fix/module-config-tenant-provisioning`) is the implementation baseline.
- Tailwind CSS v4 is already configured; no `tailwind.config.ts` customization is required.
- shadcn/ui components (Dialog, DropdownMenu, Badge, Table) are already installed.
- The Playwright test suite (`test:e2e`) is operational and the developer can add new spec files.
- Chrome DevTools device simulation is the primary QA verification method.

---

## 2. Work Breakdown Structure (WBS)

The WBS is organized by user story. Each story decomposes into atomic [FE] (frontend implementation) and [QA] (testing) subtasks. Story point estimates use modified Fibonacci: 1, 2, 3, 5, 8, 13.

**Prefix conventions:**
- `[FE]` — Frontend implementation task (tag: `frontend`)
- `[QA]` — Quality assurance / test task (tag: `test`)

---

### WBS-00: Shared Foundation (Sprint 1 prerequisite, no parent story)

These tasks are not attached to a single user story but are dependencies for all other Sprint 1 work.

```
Infrastructure / Foundation
  |
  +-- [FE] Create useBreakpoint hook — src/hooks/use-breakpoint.ts
  |     Implement hook per architecture Section 8.1. SSR default "lg". Expose
  |     useIsMobile() convenience wrapper.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Create MobileDrawer component — src/components/layout/mobile-drawer.tsx
  |     Implement full component per architecture Section 4.5: backdrop, drawer panel,
  |     Escape key, body scroll lock, focus management, aria-modal.
  |     Points: 2 | Tag: frontend | Deps: useBreakpoint hook
  |
  +-- [FE] Add touch target CSS variable to globals.css
        Add --size-touch-target: 2.75rem to @theme inline block per architecture
        Section 3.4 and 9.3 (File 18).
        Points: 1 | Tag: frontend | Deps: none
```

---

### WBS-01: RESP-01 — Mobile Sidebar Drawer (8 pts)

**User Story:** As a tenant admin on mobile, I want a hamburger menu that opens the navigation as a slide-in drawer so I can navigate without the sidebar consuming screen width.

**Dependencies:** WBS-00 must be complete.

```
Parent: RESP-01 Mobile Sidebar Drawer (8 pts)
  |
  +-- [FE] Update layout.tsx — add mobileMenuOpen state and MobileDrawer
  |     File: src/app/(dashboard)/layout.tsx
  |     Add useState(false) for mobileMenuOpen. Wrap existing <Sidebar> in
  |     hidden md:flex div. Add <MobileDrawer> containing <Sidebar onNavigate={}>.
  |     Pass onMenuOpen to Header. Change main padding p-6 -> p-4 md:p-5 lg:p-6.
  |     Points: 3 | Tag: frontend | Deps: MobileDrawer component
  |
  +-- [FE] Update sidebar.tsx — add onNavigate prop and mobile drawer mode
  |     File: src/components/layout/sidebar.tsx
  |     Add onNavigate?: () => void to SidebarProps. Derive isMobileDrawer = !!onNavigate.
  |     Force effectiveExpanded = true when isMobileDrawer. Call onNavigate?.() on all
  |     nav link clicks and logout. Change aside width: isMobileDrawer ? "w-[280px]" :
  |     (effectiveExpanded ? "w-[298px]" : "w-[60px]"). Add useBreakpoint for tablet
  |     forced-collapse default (architecture Section 4.6).
  |     Points: 3 | Tag: frontend | Deps: layout.tsx update, useBreakpoint hook
  |
  +-- [FE] Update header.tsx — add hamburger button and onMenuOpen prop
  |     File: src/components/layout/header.tsx
  |     Add onMenuOpen?: () => void to HeaderProps. Import Menu from lucide-react.
  |     Add hamburger button with md:hidden, w-11 h-11, aria-label="Open navigation menu".
  |     Change px-6 -> px-4 md:px-6.
  |     Points: 2 | Tag: frontend | Deps: layout.tsx update
  |
  +-- [QA] Test mobile sidebar drawer — Playwright spec
  |     File: tests/responsive/sidebar-drawer.spec.ts
  |     Verify: hamburger visible at 375px, hidden at 1280px. Drawer opens on tap.
  |     Backdrop click closes drawer. Nav link click closes drawer and navigates.
  |     Escape key closes drawer. Desktop sidebar unaffected. Touch targets 44px.
  |     Points: 3 | Tag: test | Deps: All [FE] tasks for RESP-01
  |
  +-- [QA] Manual QA — sidebar drawer at 375px, 768px, 1024px, 1280px
        Verify sidebar hidden by default at 375px. Verify collapsed 60px at 768px.
        Verify expand/collapse toggle still works at desktop. Verify no horizontal
        scrollbar on any page after nav drawer changes. Verify dark mode renders correctly.
        Points: 2 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-02: RESP-02 — Responsive Dashboard Layout Shell (3 pts)

**User Story:** As any user, I want the page shell to fill available screen width correctly so content is readable without horizontal scrolling on any device.

**Dependencies:** None (delivered as part of RESP-01 layout.tsx changes).

**Note:** RESP-02 is primarily implemented within the layout.tsx changes of WBS-01. The subtasks below cover the isolated verification and any header-specific items.

```
Parent: RESP-02 Responsive Dashboard Layout Shell (3 pts)
  |
  +-- [FE] Verify main content padding breakpoints — layout.tsx
  |     Confirm p-4 md:p-5 lg:p-6 is applied to <main> element. Verify min-w-0 on
  |     the flex child div wrapping header + main. No horizontal overflow on body.
  |     Points: 1 | Tag: frontend | Deps: WBS-01 layout.tsx update
  |
  +-- [FE] Verify header controls — dropdown bounds on mobile
  |     File: src/components/layout/header.tsx
  |     Confirm user avatar dropdown panel does not extend beyond right viewport edge
  |     at 375px. Ensure all dropdown items have min touch target 44px height.
  |     Points: 1 | Tag: frontend | Deps: header.tsx update
  |
  +-- [QA] Manual QA — layout shell at 375px, 768px, 1280px
        Verify: main content = 100vw at 375px, no horizontal scrollbar, header full width,
        page title visible, hamburger visible (mobile only), avatar dropdown visible,
        content padding 16px at mobile, 20px at 768px, 24px at 1280px. Run Lighthouse
        CLS check (target < 0.1) on /dashboard at 375px.
        Points: 1 | Tag: test | Deps: All [FE] tasks for RESP-02
```

---

### WBS-03: RESP-03 — Auth Pages Mobile Polish (3 pts)

**User Story:** As an invited user opening an email link on mobile, I want auth forms to display correctly and be easy to interact with on a small screen.

**Dependencies:** None (auth pages are in a separate route group, independent of dashboard changes).

```
Parent: RESP-03 Auth Pages Mobile Polish (3 pts)
  |
  +-- [FE] Update login/page.tsx — card padding, password toggle, forgot link
  |     File: src/app/(auth)/login/page.tsx
  |     Change p-10 -> p-6 sm:p-10 on card div. Password toggle button:
  |     add w-11 h-11 flex items-center justify-center rounded-full. Change pr-12 -> pr-14
  |     on password input. Forgot password link: add inline-flex items-center min-h-[44px] px-1.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Update forgot-password/page.tsx — card padding (both states)
  |     File: src/app/(auth)/forgot-password/page.tsx
  |     Change p-10 -> p-6 sm:p-10 on both the form state card and the done/success state card.
  |     Two instances to update.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Update reset-password/page.tsx — card padding (3 states), password toggles
  |     File: src/app/(auth)/reset-password/page.tsx
  |     Change p-10 -> p-6 sm:p-10 on no-token state, done state, and form state (3 instances).
  |     Apply w-11 h-11 password toggle pattern to all toggle buttons. Change pr-12 -> pr-14.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Update accept-invite/page.tsx — card padding (2 states), password toggle
  |     File: src/app/(auth)/accept-invite/page.tsx
  |     Change p-10 -> p-6 sm:p-10 on both states. Apply w-11 h-11 password toggle pattern.
  |     Change pr-12 -> pr-14 on password inputs.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [QA] Playwright spec — auth pages horizontal overflow at 375px
  |     File: tests/responsive/auth-pages.spec.ts
  |     Verify no horizontal overflow on /login, /forgot-password, /reset-password,
  |     /accept-invite at 375px. Verify password toggle bounding box >= 44x44px.
  |     Verify forgot password link min-height >= 44px. Verify card does not overflow.
  |     Verify success state card fits viewport.
  |     Points: 2 | Tag: test | Deps: All [FE] tasks for RESP-03
  |
  +-- [QA] Manual QA — auth pages on 375px, 390px, 768px
  |     Verify card padding 24px on mobile (p-6 = 24px). Verify inputs full-width.
  |     Verify submit button full-width. Verify show/hide password toggle tappable.
  |     Verify Lighthouse Mobile score >= 90 on /login at 375px. Verify dark mode.
        Points: 2 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-04: RESP-04 — Users Page Mobile Card Stack (8 pts)

**User Story:** As a tenant admin on mobile, I want the Users list to display as a vertical card stack so I can read user information and take actions without horizontal scrolling.

**Dependencies:** WBS-01 (layout shell must be done first so mobile viewport renders correctly).

```
Parent: RESP-04 Users Page Mobile Card Stack (8 pts)
  |
  +-- [FE] Create UserCard component
  |     File: src/app/(dashboard)/dashboard/users/_components/user-card.tsx
  |     Implement per architecture Section 5.2 (UserCard spec). Avatar initial, display
  |     name, email (truncated), status badge, role badges (flex-wrap with overflow indicator),
  |     MoreHorizontal dropdown (w-11 h-11 touch target, e.stopPropagation). Whole card
  |     navigates to user detail on tap. bg-card rounded-[10px] border border-border p-4.
  |     Points: 3 | Tag: frontend | Deps: layout shell (WBS-01)
  |
  +-- [FE] Update users/page.tsx — responsive toolbar
  |     File: src/app/(dashboard)/dashboard/users/page.tsx
  |     Toolbar: flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:gap-3.
  |     Search: w-full sm:flex-1 sm:min-w-[200px] sm:max-w-sm.
  |     Filters: grid grid-cols-2 gap-2 sm:flex sm:items-center sm:gap-3.
  |     Invite button: w-full sm:w-auto sm:ml-auto.
  |     Points: 2 | Tag: frontend | Deps: none
  |
  +-- [FE] Update users/page.tsx — table/card swap
  |     File: src/app/(dashboard)/dashboard/users/page.tsx
  |     Wrap existing table div with hidden md:block class. Add block md:hidden space-y-3
  |     container rendering UserCard components. Handle loading, empty state, and populated
  |     states in card view. Pass same action handlers (onView, onSuspend, onResendInvite)
  |     to UserCard as table row uses.
  |     Points: 2 | Tag: frontend | Deps: UserCard component
  |
  +-- [QA] Playwright spec — users page mobile
  |     File: tests/responsive/users-page.spec.ts
  |     Verify: table hidden at 375px, card stack visible. Card contains name, email, status,
  |     roles, actions. Table visible at 768px, cards hidden. Filter toolbar height <= 180px
  |     at 375px. Invite User button visible without scroll. Dropdown does not overflow viewport.
  |     No horizontal overflow at 375px.
  |     Points: 3 | Tag: test | Deps: All [FE] tasks for RESP-04
  |
  +-- [QA] Manual QA — users page at 375px, 768px, 1280px
        Verify card actions (view, suspend, resend invite) all function correctly on mobile.
        Verify status filter and module filter work from mobile layout. Verify empty state
        centered correctly. Verify pending banner at top (if applicable). Verify dark mode.
        Verify no regression on desktop table view.
        Points: 2 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-05: RESP-05 — User Detail Page Responsive Header (5 pts)

**User Story:** As an admin on mobile viewing a user detail page, I want the header (user name, status, action buttons) to display readably so I can identify the user and take actions.

**Dependencies:** WBS-01 (layout shell).

```
Parent: RESP-05 User Detail Responsive Header (5 pts)
  |
  +-- [FE] Update users/[id]/page.tsx — responsive header layout
  |     File: src/app/(dashboard)/dashboard/users/[id]/page.tsx
  |     Outer header div: flex items-center gap-3 -> flex flex-col gap-3 sm:flex-row sm:items-center.
  |     Action buttons group: flex items-center gap-2 -> flex flex-wrap items-center gap-2 w-full sm:w-auto.
  |     Each action button: add flex-1 sm:flex-none so buttons distribute on mobile.
  |     Points: 2 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [FE] Update users/[id]/page.tsx — touch targets on roles checkboxes
  |     File: src/app/(dashboard)/dashboard/users/[id]/page.tsx
  |     Roles checkbox rows: add min-h-[44px] to the label/div element
  |     (flex items-center gap-3 rounded-[10px] border p-2.5 min-h-[44px]).
  |     Confirm Save Roles button has >= 44px height (already h-9 = 36px; change to h-11).
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Verify account info card — UUID truncation and overflow
  |     File: src/app/(dashboard)/dashboard/users/[id]/page.tsx
  |     Confirm max-w-[260px] truncate on User ID value works within card boundary at 375px.
  |     Verify flex justify-between rows do not overflow. No changes expected; confirm only.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [QA] Playwright spec — user detail mobile header
  |     File: tests/responsive/user-detail.spec.ts
  |     Verify: header wraps to two rows at 375px (back+name+status on row 1, buttons on row 2).
  |     All buttons >= 44px height. No horizontal overflow. Desktop single-row header unchanged.
  |     Roles checkboxes min-height 44px. Account info card rows no overflow.
  |     Points: 2 | Tag: test | Deps: All [FE] tasks for RESP-05
  |
  +-- [QA] Manual QA — user detail at 375px, 768px, 1280px
        Verify all action buttons (Resend Invite, Send Password Reset, Suspend/Activate) are
        tappable. Verify roles checkbox saves work on mobile. Verify dark mode. Verify no
        regression on desktop.
        Points: 1 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-06: RESP-06 — Tenants Page Mobile Card Stack (8 pts)

**User Story:** As a super admin on mobile, I want the Tenants list as a vertical card stack so I can review tenant status and take actions from my phone.

**Dependencies:** WBS-01 complete.

```
Parent: RESP-06 Tenants Page Mobile Card Stack (8 pts)
  |
  +-- [FE] Create TenantCard component
  |     File: src/app/(dashboard)/dashboard/tenants/_components/tenant-card.tsx
  |     Implement per architecture Section 5.2 (TenantCard spec). Name, slug (monospace),
  |     status badge, modules (flex-wrap badges), created date. MoreHorizontal dropdown
  |     w-11 h-11. Full card tap navigates to tenant detail.
  |     Points: 3 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [FE] Update tenants/page.tsx — responsive toolbar
  |     File: src/app/(dashboard)/dashboard/tenants/page.tsx
  |     Toolbar: flex items-center justify-between gap-3 -> flex flex-col gap-2 sm:flex-row
  |     sm:items-center sm:justify-between sm:gap-3. Search: w-full sm:flex-1 sm:max-w-sm.
  |     Provision Tenant button: w-full sm:w-auto.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Update tenants/page.tsx — table/card swap
  |     File: src/app/(dashboard)/dashboard/tenants/page.tsx
  |     Add hidden md:block on table wrapper div. Add block md:hidden space-y-3 container
  |     rendering TenantCard components. Handle loading, empty, and populated states.
  |     Points: 2 | Tag: frontend | Deps: TenantCard component
  |
  +-- [FE] Make Provision Tenant dialog mobile-friendly
  |     File: src/app/(dashboard)/dashboard/tenants/page.tsx (dialog inline or extracted)
  |     DialogContent: sm:max-w-[500px] -> w-[calc(100vw-32px)] sm:max-w-[500px] max-h-[90vh]
  |     overflow-y-auto. Module checkbox rows: add min-h-[44px].
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [QA] Playwright spec — tenants page mobile
  |     File: tests/responsive/tenants-page.spec.ts
  |     Verify: table hidden at 375px, card stack visible with all fields. Table visible 768px+.
  |     Provision Tenant button full-width on mobile. Dialog opens full-width. Module checkboxes
  |     44px height. Card tap navigates to detail. No horizontal overflow.
  |     Points: 3 | Tag: test | Deps: All [FE] tasks for RESP-06
  |
  +-- [QA] Manual QA — tenants page at 375px, 768px, 1280px
        Verify status filter/search works on mobile. Verify tenant actions (suspend/activate)
        in dropdown functional. Verify provision tenant flow end-to-end on mobile.
        Verify dark mode. Verify no regression on desktop table.
        Points: 2 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-07: RESP-07 — Tenant Detail Responsive Header and Touch-Safe Copy (5 pts)

**User Story:** As a super admin on mobile viewing a tenant detail page, I want the header actions and copy-to-clipboard fields to be comfortably usable.

**Dependencies:** WBS-01 complete.

```
Parent: RESP-07 Tenant Detail Responsive Header (5 pts)
  |
  +-- [FE] Update tenants/[id]/page.tsx — responsive header
  |     File: src/app/(dashboard)/dashboard/tenants/[id]/page.tsx
  |     Header div: flex items-center gap-3 -> flex flex-col gap-3 sm:flex-row sm:items-center.
  |     Status + action group: flex items-center gap-2 -> flex flex-wrap items-center gap-2
  |     w-full sm:w-auto. Suspend/Activate button: add flex-1 sm:flex-none w-full sm:w-auto.
  |     Points: 2 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [FE] Update CopyField component — 44px touch target on copy button
  |     File: whichever file contains CopyField component (likely src/components/ui/ or inline)
  |     Wrap copy icon button in div className="flex items-center justify-center w-11 h-11"
  |     to guarantee 44px touch target. The 14px icon remains visually unchanged.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [FE] Update tenants/[id]/page.tsx — admin list action buttons and integration block
  |     File: src/app/(dashboard)/dashboard/tenants/[id]/page.tsx
  |     Admin list action dots: change h-7 w-7 -> h-11 w-11.
  |     Integration env code block: confirm overflow-x-auto on pre/code container (add if absent).
  |     Invite Admin dialog: sm:max-w-[440px] -> w-[calc(100vw-32px)] sm:max-w-[440px]
  |     max-h-[90vh] overflow-y-auto.
  |     Points: 1 | Tag: frontend | Deps: none
  |
  +-- [QA] Playwright spec — tenant detail mobile
  |     File: tests/responsive/tenant-detail.spec.ts
  |     Verify: header wraps to two rows at 375px. Suspend/Activate button full-width on mobile.
  |     Copy button bounding box >= 44x44px. Integration env block scrolls horizontally within
  |     container without page overflow. Admin list action button >= 44px. No horizontal overflow.
  |     Points: 2 | Tag: test | Deps: All [FE] tasks for RESP-07
  |
  +-- [QA] Manual QA — tenant detail at 375px, 768px, 1280px
        Verify copy-to-clipboard works on mobile Safari/Chrome. Verify info + integration cards
        stack vertically at 375px (lg:grid-cols-2 regression check). Verify dark mode.
        Verify no regression on desktop.
        Points: 1 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-08: RESP-08 — Audit Log Page Mobile Stacked Entries (8 pts)

**User Story:** As an admin on mobile reviewing the audit log, I want each log entry as a stacked card so I can read time, action, actor, and target without horizontal scrolling.

**Dependencies:** WBS-01 complete.

```
Parent: RESP-08 Audit Log Mobile Stacked Entries (8 pts)
  |
  +-- [FE] Create AuditLogCard component
  |     File: src/app/(dashboard)/dashboard/audit/_components/audit-log-card.tsx
  |     Implement per architecture Section 5.2 (AuditLogCard spec). Action badge (color-coded),
  |     timestamp (whitespace-nowrap, xs text), Actor row (label + email/id truncated), Target row,
  |     IP row (monospace). bg-card rounded-[10px] border border-border p-4 space-y-2.
  |     Points: 3 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [FE] Update audit/page.tsx — responsive filter form
  |     File: src/app/(dashboard)/dashboard/audit/page.tsx
  |     Filter form: flex items-end gap-3 flex-wrap -> flex flex-col gap-3 sm:flex-row sm:items-end.
  |     Action dropdown: min-w-[190px] -> w-full sm:min-w-[190px] sm:w-auto.
  |     Date inputs wrapper: flex items-end gap-2 -> grid grid-cols-2 gap-2 sm:flex sm:items-end.
  |     Date inputs: w-[148px] -> w-full sm:w-[148px].
  |     Apply button: add w-full sm:w-auto.
  |     Points: 2 | Tag: frontend | Deps: none
  |
  +-- [FE] Update audit/page.tsx — table/card swap and pagination touch targets
  |     File: src/app/(dashboard)/dashboard/audit/page.tsx
  |     Add hidden md:block on table wrapper. Add block md:hidden space-y-3 container
  |     rendering AuditLogCard components. Handle loading, empty, and populated states.
  |     Pagination prev/next buttons: h-8 w-8 -> h-10 w-10 (40px, acceptable for secondary).
  |     Points: 2 | Tag: frontend | Deps: AuditLogCard component
  |
  +-- [QA] Playwright spec — audit log mobile
  |     File: tests/responsive/audit-page.spec.ts
  |     Verify: table hidden 375px, card stack visible with all fields (time, action, actor,
  |     target, IP). Table visible 768px+. Filter form height <= 200px at 375px. Date inputs
  |     in 2-col grid. Apply button full-width. Pagination buttons >= 40px. No horizontal overflow.
  |     Points: 3 | Tag: test | Deps: All [FE] tasks for RESP-08
  |
  +-- [QA] Manual QA — audit log at 375px, 768px, 1280px
        Verify filter apply/clear work on mobile. Verify pagination navigation on mobile.
        Verify action color badges match desktop. Verify empty state centered. Verify dark mode.
        Points: 2 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-09: RESP-09 — Roles Page Mobile Card Stack (5 pts)

**User Story:** As an admin on mobile reviewing roles, I want the Roles list as stacked cards so I can view role names, descriptions, and types without table overflow.

**Dependencies:** WBS-01 complete.

```
Parent: RESP-09 Roles Page Mobile Card Stack (5 pts)
  |
  +-- [FE] Create RoleCard component
  |     File: src/app/(dashboard)/dashboard/roles/_components/role-card.tsx
  |     Implement per architecture Section 5.2 (RoleCard spec). Name (monospace), description
  |     (wrapping), module badge (or system scope text), type badge (system/custom), created date,
  |     delete button for custom roles (w-11 h-11 touch target). bg-card rounded-[10px] border.
  |     Points: 2 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [FE] Update roles/page.tsx — responsive toolbar and table/card swap
  |     File: src/app/(dashboard)/dashboard/roles/page.tsx
  |     Toolbar: flex items-center justify-between -> flex flex-col gap-2 sm:flex-row sm:items-center
  |     sm:justify-between. Create Role button: add w-full sm:w-auto.
  |     Tab pills: add min-h-[36px] to each tab button.
  |     Table wrapper: add hidden md:block. Card stack: block md:hidden space-y-3 with RoleCard.
  |     Create Role dialog: sm:max-w-[400px] -> w-[calc(100vw-32px)] sm:max-w-[400px]
  |     max-h-[90vh] overflow-y-auto.
  |     Points: 2 | Tag: frontend | Deps: RoleCard component
  |
  +-- [QA] Playwright spec — roles page mobile
  |     File: tests/responsive/roles-page.spec.ts
  |     Verify: table hidden 375px, card stack visible. Table visible 768px+. Tab pills wrap
  |     without horizontal scroll. Create Role button full-width mobile. Dialog near-full-width.
  |     Custom role delete button >= 44px. No horizontal overflow.
  |     Points: 2 | Tag: test | Deps: All [FE] tasks for RESP-09
  |
  +-- [QA] Manual QA — roles page at 375px, 768px, 1280px
        Verify tab filter by module works. Verify create role flow on mobile. Verify delete
        confirmation works. Verify dark mode. Verify no regression on desktop.
        Points: 1 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-10: RESP-10 — My Profile Responsive Name Grid (2 pts)

**User Story:** As a regular user accessing My Profile on mobile, I want first and last name fields to stack vertically so I can type comfortably.

**Dependencies:** WBS-01 complete (main padding applies).

```
Parent: RESP-10 My Profile Responsive Name Grid (2 pts)
  |
  +-- [FE] Update me/page.tsx — name grid and password toggles
  |     File: src/app/(dashboard)/me/page.tsx
  |     Name grid: grid grid-cols-2 gap-3 -> grid grid-cols-1 sm:grid-cols-2 gap-3.
  |     Password toggle buttons: change className to include w-11 h-11 flex items-center
  |     justify-center rounded-full (apply to both showCurrent and showNew toggles).
  |     Password inputs: pr-10 -> pr-14 for adequate clearance.
  |     Points: 1 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [QA] Playwright spec — my profile mobile
  |     File: tests/responsive/profile-page.spec.ts
  |     Verify: name grid is 1-column at 375px, 2-column at 640px+. Inputs full card width.
  |     Password toggle >= 44px. Account info rows no overflow. No horizontal overflow.
  |     Points: 1 | Tag: test | Deps: [FE] task for RESP-10
  |
  +-- [QA] Manual QA — my profile at 375px, 640px, 1280px
        Verify name save works on mobile. Verify password change form usable on mobile.
        Verify account info (User ID truncation). Verify dark mode. Verify desktop unchanged.
        Points: 1 | Tag: test | Deps: Playwright spec passes
```

---

### WBS-11: RESP-11 — Settings Page Touch Polish (Could Have)

**Note:** The settings page is nearly responsive already. This is classified Could Have per PRD.

```
Parent: RESP-11 Settings Page Touch Polish (2 pts)
  |
  +-- [FE] Update settings/page.tsx — session duration input width
  |     File: src/app/(dashboard)/dashboard/settings/page.tsx
  |     Session duration input: max-w-[160px] -> w-full sm:max-w-[160px] to fill card width
  |     on mobile. Verify textarea already w-full. Verify Save button accessible.
  |     Points: 1 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [QA] Manual QA — settings at 375px, 1280px
        Verify MFA toggle row usable. Verify session duration input full-width on mobile.
        Verify integration endpoints with long URLs do not overflow (break-all already applied).
        Verify Save flow works on mobile. Verify no regression.
        Points: 1 | Tag: test | Deps: [FE] task
```

---

### WBS-12: RESP-12 — Dashboard Quick Actions Polish (Could Have)

**Note:** Dashboard page is already partially responsive. This is a minor refinement.

```
Parent: RESP-12 Dashboard Quick Actions Polish (2 pts)
  |
  +-- [FE] Update dashboard/page.tsx — Quick Actions single-column at mobile
  |     File: src/app/(dashboard)/dashboard/page.tsx
  |     Change Quick Actions grid: grid grid-cols-2 sm:grid-cols-4 -> verify button labels
  |     do not truncate at 2-column on 375px. If needed, change to grid-cols-1 sm:grid-cols-2
  |     lg:grid-cols-4. Stat cards grid already has responsive breakpoints — confirm no changes.
  |     Points: 1 | Tag: frontend | Deps: WBS-01 layout shell
  |
  +-- [QA] Manual QA — dashboard at 375px, 768px, 1280px
        Verify stat cards stack correctly. Verify quick action button labels visible. Verify
        no horizontal overflow. Run Lighthouse Mobile: target >= 90. Check CLS < 0.1.
        Points: 1 | Tag: test | Deps: [FE] task
```

---

### WBS-13: Sprint 3 — Cross-Browser and End-to-End QA

```
Parent: Sprint 3 QA and Validation (Sprint 3 only)
  |
  +-- [QA] Cross-browser testing — Safari iOS, Chrome Android, Samsung Internet
  |     Manually test all Must Have and Should Have stories on: Safari iOS 16 (375px),
  |     Chrome Android (412px), Samsung Internet (412px). Document any browser-specific
  |     rendering differences. File issues as separate tasks if regressions found.
  |     Points: 5 | Tag: test | Deps: All Sprint 2 [FE] tasks complete
  |
  +-- [QA] Final Playwright horizontal overflow suite
  |     File: tests/responsive/no-overflow.spec.ts
  |     Automated check: for every page route at 375px, 768px, 1280px — verify
  |     document.documentElement.scrollWidth <= document.documentElement.clientWidth.
  |     This is the release-readiness gate check per PRD Section 10.
  |     Points: 3 | Tag: test | Deps: All Sprint 2 [FE] tasks complete
  |
  +-- [QA] Lighthouse audit report — mobile scores and CLS
  |     Run Lighthouse Mobile on: /login, /dashboard, /dashboard/users, /dashboard/tenants,
  |     /dashboard/audit. Record Mobile score and CLS for each. Target: Mobile >= 90 on
  |     /login and /dashboard; >= 85 on table pages. CLS < 0.1 on all.
  |     Produce a one-page audit summary for PO sign-off.
  |     Points: 3 | Tag: test | Deps: All Sprint 2 [FE] tasks complete
  |
  +-- [QA] Accessibility audit — axe-core scan on key pages
  |     Run axe-core (via Playwright axe integration or browser extension) on /login,
  |     /dashboard, /dashboard/users at 375px. Resolve any critical or serious violations
  |     introduced by responsive changes. WCAG 2.1 AA target.
  |     Points: 3 | Tag: test | Deps: Sprint 3 FE complete
  |
  +-- [QA] Visual regression screenshot archive
        Capture screenshots at 375px, 768px, 1280px for all 14 pages in both light and
        dark mode using Playwright. Archive for future regression comparison baseline.
        Points: 2 | Tag: test | Deps: All Sprint 3 complete
```

---

## 3. Sprint Plan and Timeline

### 3.1 Sprint Capacity

**Team velocity:** 34 story points per sprint
**80% capacity rule:** Maximum 27 implementation points per sprint (buffer reserved for QA integration, PR review, and regression testing)
**Sprint duration:** 2 weeks
**Sprints:** 3 total

### 3.2 Sprint 1 — Foundation and Critical Paths (Must Have)

**Sprint dates:** 2026-03-09 to 2026-03-20
**Sprint goal:** Any user can open auth flows on mobile. Any admin can navigate and perform basic user management on mobile.

| ID | Story | [FE] pts | [QA] pts | Total | Dependency |
|----|-------|----------|----------|-------|------------|
| WBS-00 | Shared Foundation (hook, drawer, CSS var) | 4 | 0 | 4 | None — do first |
| WBS-02 | Layout Shell (partly delivered in WBS-01) | 2 | 1 | 3 | WBS-00 |
| WBS-01 | Mobile Sidebar Drawer | 8 | 5 | 13 | WBS-00 |
| WBS-03 | Auth Pages Mobile Polish | 4 | 4 | 8 | None (parallel) |
| WBS-05 | User Detail Responsive Header | 4 | 3 | 7 | WBS-01 |
| WBS-04 | Users Page Mobile Card Stack | 7 | 5 | 12 | WBS-01 |
| **Buffer** | PR review, regression, a11y spot-check | — | — | 5 | — |
| **Sprint 1 Total** | | **29** | **18** | **47** | |

**Critical path in Sprint 1:**
WBS-00 (foundation) -> WBS-01 (layout/sidebar/header) -> WBS-04 (users card stack)

**Note on capacity:** Total story implementation points = 47. Sprint capacity is 34 x 2 teams (dev + QA) with buffer reserved. Developer and QA work in parallel — developer completes [FE] tasks while QA builds test specs against completed items. The buffer absorbs PR review cycles and any discovered regressions.

**Milestones:**
- Day 3: WBS-00 complete — useBreakpoint, MobileDrawer, CSS var merged
- Day 5: Auth pages (WBS-03) merged — independently deployable
- Day 8: Sidebar drawer fully functional (WBS-01 complete)
- Day 10: Users page card stack functional (WBS-04 [FE] complete)
- Day 14: Sprint 1 QA complete, PO demo, Sprint 1 retro

---

### 3.3 Sprint 2 — Complete Dashboard Coverage (Should Have)

**Sprint dates:** 2026-03-23 to 2026-04-03
**Sprint goal:** Every page in TGX Auth Console is fully usable on 375px mobile viewport with no horizontal overflow and all touch targets >= 44px.

| ID | Story | [FE] pts | [QA] pts | Total | Dependency |
|----|-------|----------|----------|-------|------------|
| WBS-06 | Tenants Page Mobile Card Stack | 7 | 5 | 12 | Sprint 1 complete |
| WBS-07 | Tenant Detail Responsive Header | 4 | 3 | 7 | Sprint 1 complete |
| WBS-09 | Roles Page Mobile Card Stack | 4 | 3 | 7 | Sprint 1 complete |
| WBS-10 | My Profile Responsive Name Grid | 1 | 2 | 3 | Sprint 1 complete |
| WBS-08 | Audit Log Mobile Stacked Entries | 7 | 5 | 12 | Sprint 1 complete |
| **Buffer** | Cross-browser QA, CLS measurement, a11y audit | — | — | 5 | — |
| **Sprint 2 Total** | | **23** | **18** | **41** | |

**Critical path in Sprint 2:**
WBS-06 (tenants) -> WBS-07 (tenant detail) in parallel with WBS-08 (audit, highest complexity)

**Milestones:**
- Day 4: Tenants page card stack and tenant detail complete
- Day 6: Roles page and My Profile complete
- Day 10: Audit log card stack complete
- Day 12: All Should Have [QA] tasks complete
- Day 14: Release readiness checklist verified, PO sign-off, Sprint 2 retro

**Release gate (end of Sprint 2):**
- All Must Have and Should Have stories Done
- Lighthouse Mobile >= 90 on /login and /dashboard
- CLS < 0.1 on all pages tested
- Zero horizontal overflow at 375px on all pages

---

### 3.4 Sprint 3 — Polish and Enhanced UX (Could Have)

**Sprint dates:** 2026-04-06 to 2026-04-17
**Sprint goal:** Enhanced mobile UX patterns in place; automated test coverage ensures no regressions.

| ID | Story | [FE] pts | [QA] pts | Total | Dependency |
|----|-------|----------|----------|-------|------------|
| WBS-11 | Settings Page Touch Polish | 1 | 1 | 2 | Sprint 1 complete |
| WBS-12 | Dashboard Quick Actions Polish | 1 | 1 | 2 | Sprint 1 complete |
| WBS-13 | Cross-browser + E2E QA suite | 0 | 16 | 16 | Sprint 2 complete |
| **Buffer** | Regression fixes, stakeholder demo prep | — | — | 4 | — |
| **Sprint 3 Total** | | **2** | **18** | **24** | |

**Milestones:**
- Day 2: Settings and Dashboard polish complete
- Day 8: Cross-browser testing complete (Safari iOS, Chrome Android, Samsung Internet)
- Day 10: Playwright full suite green, Lighthouse report produced
- Day 12: Accessibility audit complete, zero critical violations
- Day 14: Visual regression screenshots archived, lessons learned session, project close

---

### 3.5 Milestone Summary

| Milestone | Date | Criteria |
|-----------|------|----------|
| Sprint 1 Start | 2026-03-09 | Foundation tasks kicked off |
| Auth Pages Done | 2026-03-13 | RESP-03 merged and verified |
| Sidebar Drawer Done | 2026-03-16 | RESP-01 + RESP-02 merged |
| Sprint 1 Complete | 2026-03-20 | All Must Have stories Done, PO sign-off |
| Sprint 2 Start | 2026-03-23 | Should Have stories kicked off |
| Release Readiness | 2026-04-03 | All Must + Should Have Done, release gate passed |
| Sprint 3 Complete | 2026-04-17 | Could Have done, E2E suite green, project closed |

---

## 4. ClickUp Task Structure

**ClickUp Workspace:** `9018768826`
**Target Space:** `AI Project` (ID: `901810085735`)

### 4.1 Folder and List Structure

```
Folder: TGX Auth Console — Responsive Design
  |
  +-- Backlog          (Could Have stories; overflow tasks)
  +-- Sprint 1         (Must Have: WBS-00 through WBS-05)
  +-- Sprint 2         (Should Have: WBS-06 through WBS-10)
  +-- Sprint 3         (Polish: WBS-11, WBS-12, WBS-13)
  +-- Done             (Accepted and closed)
```

---

### 4.2 Sprint 1 Tasks

---

**TASK: [FE] Create useBreakpoint hook**

```
TASK:
  name: "[FE] Create useBreakpoint hook — src/hooks/use-breakpoint.ts"
  list: Sprint 1
  description: |
    Create the useBreakpoint hook per architecture document Section 8.1.
    Implement the BREAKPOINTS array (xl/lg/md/sm/base), the useBreakpoint() function
    with SSR default of "lg", and the useIsMobile() convenience wrapper.
    AC: Hook returns correct breakpoint string for each viewport width.
    useIsMobile() returns true for base and sm, false for md+.
  assignee_role: Developer
  story_points: 1
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-10
```

---

**TASK: [FE] Create MobileDrawer component**

```
TASK:
  name: "[FE] Create MobileDrawer component — src/components/layout/mobile-drawer.tsx"
  list: Sprint 1
  description: |
    Create the MobileDrawer component per architecture Section 4.5. Includes: backdrop
    (z-40, bg-black/50, md:hidden), drawer panel (z-50, w-[280px], translate-x transition),
    close button (w-11 h-11, right-[-48px] positioned), body scroll lock useEffect,
    Escape key handler, focus management (tabIndex={-1}), aria-modal + aria-label.
    AC: Drawer opens/closes with CSS transition. Backdrop click closes. Escape closes.
    Body scroll locked when open. ARIA attributes present. md:hidden on both elements.
  assignee_role: Developer
  story_points: 2
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-10
```

---

**TASK: [FE] Add touch target CSS variable to globals.css**

```
TASK:
  name: "[FE] Add --size-touch-target CSS variable to globals.css"
  list: Sprint 1
  description: |
    Add --size-touch-target: 2.75rem to the @theme inline block in
    src/app/globals.css per architecture Section 3.4 and 9.3 (File 18).
    This documents the 44px touch target standard in the design token system.
    AC: Variable exists in globals.css. tsc --noEmit passes. No other CSS changes made.
  assignee_role: Developer
  story_points: 1
  priority: normal
  tags: [frontend]
  due_date: 2026-03-10
```

---

**TASK: [FE] Update layout.tsx — add MobileDrawer and mobileMenuOpen state**

```
TASK:
  name: "[FE] Update dashboard layout.tsx — mobile drawer integration"
  list: Sprint 1
  description: |
    Modify src/app/(dashboard)/layout.tsx: Add useState(false) for mobileMenuOpen.
    Wrap existing <Sidebar> in <div className="hidden md:flex">. Add <MobileDrawer> with
    <Sidebar onNavigate={() => setMobileMenuOpen(false)}> inside. Pass onMenuOpen prop
    to <Header>. Change <main> padding from p-6 to p-4 md:p-5 lg:p-6.
    AC: Sidebar hidden on mobile (< 768px). Drawer renders correctly. Main content has
    correct padding at all breakpoints. Desktop layout pixel-identical.
  assignee_role: Developer
  story_points: 3
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-11
```

---

**TASK: [FE] Update sidebar.tsx — onNavigate prop and mobile drawer mode**

```
TASK:
  name: "[FE] Update sidebar.tsx — add onNavigate prop and mobile drawer mode"
  list: Sprint 1
  description: |
    Modify src/components/layout/sidebar.tsx: Add onNavigate?: () => void to SidebarProps.
    Derive isMobileDrawer = !!onNavigate and effectiveExpanded = isMobileDrawer ? true : expanded.
    Replace all expanded refs with effectiveExpanded. Call onNavigate?.() on nav link clicks
    and logout. Change aside width: isMobileDrawer -> w-[280px]; else expanded ? w-[298px] : w-[60px].
    Add useBreakpoint logic to default-collapse sidebar on first tablet visit.
    AC: Sidebar always expanded in drawer. onNavigate called on nav. Desktop expand/collapse unchanged.
  assignee_role: Developer
  story_points: 3
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-12
```

---

**TASK: [FE] Update header.tsx — hamburger button and responsive padding**

```
TASK:
  name: "[FE] Update header.tsx — add hamburger button and onMenuOpen prop"
  list: Sprint 1
  description: |
    Modify src/components/layout/header.tsx: Add onMenuOpen?: () => void to HeaderProps.
    Import Menu from lucide-react. Add hamburger button with md:hidden, w-11 h-11,
    aria-label="Open navigation menu", hover states. Change header padding from px-6 to px-4 md:px-6.
    AC: Hamburger visible at < 768px. Hidden at >= 768px. Calls onMenuOpen on click.
    Desktop header layout unchanged. Touch target 44x44px.
  assignee_role: Developer
  story_points: 2
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-12
```

---

**TASK: [QA] Playwright spec — mobile sidebar drawer**

```
TASK:
  name: "[QA] Playwright spec — mobile sidebar drawer (RESP-01)"
  list: Sprint 1
  description: |
    Create tests/responsive/sidebar-drawer.spec.ts. Cover: hamburger visible at 375px,
    hidden at 1280px. Sidebar hidden by default on mobile. Drawer opens on hamburger click.
    Backdrop click closes drawer. Nav link click closes drawer + navigates. Escape key closes.
    Desktop sidebar renders inline (not drawer). Touch target bounding boxes >= 44px.
    AC: All test scenarios green. Spec can be run with npx playwright test sidebar-drawer.
  assignee_role: QA
  story_points: 3
  priority: urgent
  tags: [test]
  due_date: 2026-03-16
```

---

**TASK: [QA] Manual QA — sidebar drawer and layout shell**

```
TASK:
  name: "[QA] Manual QA — sidebar drawer and layout shell (RESP-01, RESP-02)"
  list: Sprint 1
  description: |
    Manually verify at 375px, 768px, 1024px, 1280px: sidebar hidden on mobile, drawer opens
    and closes correctly. Collapsed sidebar at 768px. Desktop expand/collapse unchanged.
    No horizontal scrollbar on any page. Content padding 16px/20px/24px per breakpoint.
    Header full-width. Dark mode no regression. CLS < 0.1 on Lighthouse for /dashboard.
    AC: All checklist items pass at all viewport sizes. Screenshot evidence recorded.
  assignee_role: QA
  story_points: 2
  priority: urgent
  tags: [test]
  due_date: 2026-03-17
```

---

**TASK: [FE] Update auth pages — card padding, password toggles, forgot link (RESP-03)**

```
TASK:
  name: "[FE] Auth pages — card padding p-6 sm:p-10, password toggle w-11 h-11"
  list: Sprint 1
  description: |
    Modify 4 files: (auth)/login/page.tsx, forgot-password/page.tsx, reset-password/page.tsx,
    accept-invite/page.tsx. Changes per architecture Sections 7.2 and 9.1 (Files 6-9):
    p-10 -> p-6 sm:p-10 on all card divs. Password toggle buttons: add w-11 h-11 flex
    items-center justify-center rounded-full. Change pr-12 -> pr-14 on password inputs.
    Login: forgot password link add inline-flex items-center min-h-[44px] px-1.
    AC: Cards have 24px padding on 375px. Password toggle bounding box >= 44x44px. No overflow.
  assignee_role: Developer
  story_points: 4
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-13
```

---

**TASK: [QA] Playwright spec — auth pages mobile (RESP-03)**

```
TASK:
  name: "[QA] Playwright spec — auth pages mobile (RESP-03)"
  list: Sprint 1
  description: |
    Create tests/responsive/auth-pages.spec.ts. Cover all 4 auth pages at 375px:
    no horizontal overflow, card does not overflow viewport, password toggle >= 44x44px,
    forgot password link min-height >= 44px, inputs full-width, success states fit viewport.
    Also verify layout unchanged at 1280px (regression check).
    AC: All scenarios green. Lighthouse Mobile score >= 90 on /login.
  assignee_role: QA
  story_points: 2
  priority: urgent
  tags: [test]
  due_date: 2026-03-14
```

---

**TASK: [FE] Create UserCard component (RESP-04)**

```
TASK:
  name: "[FE] Create UserCard component — users/_components/user-card.tsx"
  list: Sprint 1
  description: |
    Create src/app/(dashboard)/dashboard/users/_components/user-card.tsx per architecture
    Section 5.2. Fields: avatar initial (tiger-red/10 bg), display name (truncated), email
    (truncated), status badge, role badges (flex-wrap), MoreHorizontal dropdown (w-11 h-11,
    e.stopPropagation), joined date. Full card tap navigates. bg-card rounded-[10px] border p-4.
    AC: Card renders all fields. Dropdown shows View/Suspend/Resend Invite as applicable.
    Tap on card (not dropdown) navigates to user detail. Touch target 44px on action button.
  assignee_role: Developer
  story_points: 3
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-14
```

---

**TASK: [FE] Update users/page.tsx — toolbar and table/card swap (RESP-04)**

```
TASK:
  name: "[FE] Update users/page.tsx — responsive toolbar and card stack"
  list: Sprint 1
  description: |
    Modify src/app/(dashboard)/dashboard/users/page.tsx per architecture Section 9.1 (File 10).
    Toolbar: flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center. Search: w-full sm:flex-1.
    Filters: grid grid-cols-2 gap-2 sm:flex. Invite button: w-full sm:w-auto sm:ml-auto.
    Table wrapper: add hidden md:block. Card stack: block md:hidden space-y-3 with UserCard
    components. Render loading spinner, empty state, and populated state in card view.
    AC: Table hidden 375px. Card stack visible. Table visible 768px+. Cards hidden 768px+.
    Invite button full-width on mobile. Filter area height <= 180px.
  assignee_role: Developer
  story_points: 4
  priority: urgent
  tags: [frontend]
  due_date: 2026-03-16
```

---

**TASK: [QA] Playwright spec — users page mobile (RESP-04)**

```
TASK:
  name: "[QA] Playwright spec — users page mobile (RESP-04)"
  list: Sprint 1
  description: |
    Create tests/responsive/users-page.spec.ts. Verify: table hidden at 375px, card stack
    visible. All card fields present. Table visible 768px+. Filter toolbar <= 180px height.
    Invite User button visible without scroll, full-width. Dropdown does not extend beyond
    viewport. Empty state centered. No horizontal overflow at 375px and 768px and 1280px.
    AC: All scenarios green. No TypeScript errors introduced.
  assignee_role: QA
  story_points: 3
  priority: urgent
  tags: [test]
  due_date: 2026-03-18
```

---

**TASK: [FE] Update users/[id]/page.tsx — responsive header and touch targets (RESP-05)**

```
TASK:
  name: "[FE] Update users/[id]/page.tsx — responsive header and role checkboxes"
  list: Sprint 1
  description: |
    Modify src/app/(dashboard)/dashboard/users/[id]/page.tsx per architecture Section 9.1 (File 11).
    Header: flex items-center gap-3 -> flex flex-col gap-3 sm:flex-row sm:items-center.
    Action buttons group: flex items-center gap-2 -> flex flex-wrap items-center gap-2 w-full sm:w-auto.
    Each button: add flex-1 sm:flex-none. Roles rows: add min-h-[44px].
    AC: Header wraps to 2 rows at 375px. Buttons evenly distributed on mobile. Desktop single-row.
    Roles checkboxes min-height 44px. No overflow.
  assignee_role: Developer
  story_points: 4
  priority: high
  tags: [frontend]
  due_date: 2026-03-17
```

---

**TASK: [QA] Playwright spec — user detail mobile (RESP-05)**

```
TASK:
  name: "[QA] Playwright spec — user detail page mobile (RESP-05)"
  list: Sprint 1
  description: |
    Create tests/responsive/user-detail.spec.ts. Verify: header wraps to 2 rows at 375px.
    All action buttons >= 44px height. No element beyond viewport. Desktop single-row unchanged.
    Roles checkboxes min-height 44px. Account info rows no overflow. UUID truncates gracefully.
    AC: All scenarios green at 375px, 768px, 1280px.
  assignee_role: QA
  story_points: 2
  priority: high
  tags: [test]
  due_date: 2026-03-19
```

---

### 4.3 Sprint 2 Tasks

---

**TASK: [FE] Create TenantCard component (RESP-06)**

```
TASK:
  name: "[FE] Create TenantCard component — tenants/_components/tenant-card.tsx"
  list: Sprint 2
  description: |
    Create src/app/(dashboard)/dashboard/tenants/_components/tenant-card.tsx per architecture
    Section 5.2. Fields: name, slug (monospace, semi-grey), status badge, enabled_modules
    (flex-wrap tiger-red badges), created date. MoreHorizontal dropdown (w-11 h-11). Full card
    navigates to tenant detail. bg-card rounded-[10px] border border-border p-4.
    AC: All fields render. Dropdown shows Suspend/Activate. Tap navigates to detail. 44px target.
  assignee_role: Developer
  story_points: 3
  priority: high
  tags: [frontend]
  due_date: 2026-03-25
```

---

**TASK: [FE] Update tenants/page.tsx — toolbar, card stack, dialog (RESP-06)**

```
TASK:
  name: "[FE] Update tenants/page.tsx — responsive toolbar, card swap, dialog"
  list: Sprint 2
  description: |
    Modify src/app/(dashboard)/dashboard/tenants/page.tsx per architecture Section 9.2 (File 12).
    Toolbar: flex flex-col gap-2 sm:flex-row. Search: w-full sm:flex-1. Provision button: w-full sm:w-auto.
    Table wrapper: add hidden md:block. Card stack: block md:hidden with TenantCard.
    Dialog: w-[calc(100vw-32px)] sm:max-w-[500px] max-h-[90vh] overflow-y-auto.
    Module checkboxes: add min-h-[44px].
    AC: Table hidden 375px. Cards visible. Table visible 768px+. Provision button full-width. Dialog mobile-friendly.
  assignee_role: Developer
  story_points: 4
  priority: high
  tags: [frontend]
  due_date: 2026-03-26
```

---

**TASK: [QA] Playwright spec — tenants page mobile (RESP-06)**

```
TASK:
  name: "[QA] Playwright spec — tenants page mobile (RESP-06)"
  list: Sprint 2
  description: |
    Create tests/responsive/tenants-page.spec.ts. Verify: table hidden 375px, cards visible.
    Table visible 768px+. Provision button full-width. Dialog near-full-width with scrollable content.
    Module checkboxes 44px. Card tap navigates. No horizontal overflow at any breakpoint.
    AC: All scenarios green. Dialog usable within 90vh.
  assignee_role: QA
  story_points: 3
  priority: high
  tags: [test]
  due_date: 2026-03-28
```

---

**TASK: [FE] Update tenants/[id]/page.tsx — header, CopyField, admin actions (RESP-07)**

```
TASK:
  name: "[FE] Update tenants/[id]/page.tsx — responsive header, CopyField 44px, admin list"
  list: Sprint 2
  description: |
    Modify src/app/(dashboard)/dashboard/tenants/[id]/page.tsx per architecture Section 9.2 (File 13).
    Header: flex flex-col gap-3 sm:flex-row sm:items-center. Action group: flex flex-wrap w-full sm:w-auto.
    CopyField copy button: wrap in div w-11 h-11 flex items-center justify-center.
    Admin list action dots: h-7 w-7 -> h-11 w-11. Integration code block: add overflow-x-auto to pre.
    Invite Admin dialog: w-[calc(100vw-32px)] sm:max-w-[440px] max-h-[90vh] overflow-y-auto.
    AC: Header 2-row on mobile. Copy button 44px. Admin actions 44px. Code block scrolls in container.
  assignee_role: Developer
  story_points: 4
  priority: high
  tags: [frontend]
  due_date: 2026-03-26
```

---

**TASK: [QA] Playwright spec — tenant detail mobile (RESP-07)**

```
TASK:
  name: "[QA] Playwright spec — tenant detail page mobile (RESP-07)"
  list: Sprint 2
  description: |
    Create tests/responsive/tenant-detail.spec.ts. Verify: header wraps 2 rows at 375px.
    Copy button >= 44x44px. Integration env block horizontal scrolls within container, no page overflow.
    Admin list action button >= 44px. No horizontal overflow. Desktop unchanged.
    AC: All scenarios green at 375px, 768px, 1280px.
  assignee_role: QA
  story_points: 2
  priority: high
  tags: [test]
  due_date: 2026-03-30
```

---

**TASK: [FE] Create AuditLogCard and update audit/page.tsx (RESP-08)**

```
TASK:
  name: "[FE] Create AuditLogCard + update audit/page.tsx — filter, card swap, pagination"
  list: Sprint 2
  description: |
    Create src/app/(dashboard)/dashboard/audit/_components/audit-log-card.tsx per architecture
    Section 5.2. Then modify audit/page.tsx per Section 9.2 (File 14): Filter form flex-col
    on mobile. Action dropdown w-full sm:min-w-[190px]. Date inputs grid grid-cols-2 sm:flex.
    Date inputs w-full sm:w-[148px]. Apply button w-full sm:w-auto. Table hidden md:block.
    Card stack block md:hidden. Pagination: h-8 w-8 -> h-10 w-10.
    AC: Card renders action badge, timestamp, actor, target, IP. Filter bar <= 200px at 375px.
    Table/card swap correct. Pagination accessible.
  assignee_role: Developer
  story_points: 7
  priority: high
  tags: [frontend]
  due_date: 2026-03-28
```

---

**TASK: [QA] Playwright spec — audit log mobile (RESP-08)**

```
TASK:
  name: "[QA] Playwright spec — audit log page mobile (RESP-08)"
  list: Sprint 2
  description: |
    Create tests/responsive/audit-page.spec.ts. Verify: table hidden 375px, cards visible.
    Cards show time, action, actor, target, IP. Table visible 768px+. Filter form <= 200px height.
    Date inputs in 2-col grid. Apply button full-width. Pagination >= 40px buttons. No overflow.
    AC: All scenarios green. Empty state centered.
  assignee_role: QA
  story_points: 3
  priority: high
  tags: [test]
  due_date: 2026-04-01
```

---

**TASK: [FE] Create RoleCard and update roles/page.tsx (RESP-09)**

```
TASK:
  name: "[FE] Create RoleCard + update roles/page.tsx — toolbar, card swap, dialog"
  list: Sprint 2
  description: |
    Create src/app/(dashboard)/dashboard/roles/_components/role-card.tsx per architecture
    Section 5.2. Then modify roles/page.tsx per Section 9.2 (File 15): Toolbar flex-col sm:flex-row.
    Create button w-full sm:w-auto. Tab pills add min-h-[36px]. Table hidden md:block. Card stack
    block md:hidden. Delete button: h-8 w-8 -> h-11 w-11. Create dialog: w-[calc(100vw-32px)]
    sm:max-w-[400px] max-h-[90vh] overflow-y-auto.
    AC: Cards show name, description, module, type, created, delete. Table/card swap correct.
    Delete button 44px. Dialog mobile-friendly. No overflow.
  assignee_role: Developer
  story_points: 4
  priority: high
  tags: [frontend]
  due_date: 2026-03-27
```

---

**TASK: [QA] Playwright spec — roles page mobile (RESP-09)**

```
TASK:
  name: "[QA] Playwright spec — roles page mobile (RESP-09)"
  list: Sprint 2
  description: |
    Create tests/responsive/roles-page.spec.ts. Verify: table hidden 375px, cards visible.
    Table visible 768px+. Tab pills no horizontal overflow, min-height 36px. Create button
    full-width. Dialog near-full-width. Custom role delete button >= 44px. No overflow.
    AC: All scenarios green at 375px and 768px.
  assignee_role: QA
  story_points: 2
  priority: high
  tags: [test]
  due_date: 2026-03-31
```

---

**TASK: [FE] Update me/page.tsx — name grid and password toggles (RESP-10)**

```
TASK:
  name: "[FE] Update me/page.tsx — responsive name grid and password toggle touch targets"
  list: Sprint 2
  description: |
    Modify src/app/(dashboard)/me/page.tsx per architecture Section 9.2 (File 16).
    Name grid: grid grid-cols-2 gap-3 -> grid grid-cols-1 sm:grid-cols-2 gap-3.
    Password toggle buttons: add w-11 h-11 flex items-center justify-center rounded-full (both toggles).
    Password inputs: pr-10 -> pr-14. Confirm User ID truncation works within card (max-w-[260px] OK).
    AC: Name grid 1-col at 375px, 2-col at 640px+. Password toggle >= 44px. No overflow.
  assignee_role: Developer
  story_points: 1
  priority: normal
  tags: [frontend]
  due_date: 2026-03-26
```

---

**TASK: [QA] Playwright spec and manual QA — my profile mobile (RESP-10)**

```
TASK:
  name: "[QA] Playwright spec + manual QA — my profile mobile (RESP-10)"
  list: Sprint 2
  description: |
    Create tests/responsive/profile-page.spec.ts. Verify: name grid 1-col at 375px, 2-col 640px+.
    Password toggle >= 44px. Account info rows no overflow. No horizontal overflow at any breakpoint.
    Manual: verify name save and password change work on mobile. Verify dark mode.
    AC: Spec green. Manual checklist complete.
  assignee_role: QA
  story_points: 2
  priority: normal
  tags: [test]
  due_date: 2026-03-31
```

---

**TASK: [QA] Sprint 2 cross-browser testing**

```
TASK:
  name: "[QA] Sprint 2 cross-browser testing — Safari iOS, Chrome Android, Samsung Internet"
  list: Sprint 2
  description: |
    Test all Must Have and Should Have stories on real/simulated browsers: Safari iOS 16 (375px),
    Chrome Android (412px), Samsung Internet (412px). Focus on: drawer animation, touch targets,
    card stacks, form inputs, dialog scrolling. Document findings. File bug tasks for regressions.
    AC: Testing complete on all 3 browser/OS targets. Zero P1/P2 bugs unresolved at sprint close.
  assignee_role: QA
  story_points: 5
  priority: high
  tags: [test]
  due_date: 2026-04-02
```

---

### 4.4 Sprint 3 Tasks (Backlog)

---

**TASK: [FE] Settings and Dashboard polish (RESP-11, RESP-12)**

```
TASK:
  name: "[FE] Settings session input w-full + Dashboard quick actions grid polish"
  list: Sprint 3
  description: |
    settings/page.tsx: session duration input max-w-[160px] -> w-full sm:max-w-[160px].
    dashboard/page.tsx: verify Quick Actions grid at 375px — if button labels truncate,
    change grid to grid-cols-1 sm:grid-cols-2 lg:grid-cols-4.
    AC: Settings input full-width on mobile. Dashboard quick action labels fully visible at 375px.
  assignee_role: Developer
  story_points: 2
  priority: low
  tags: [frontend]
  due_date: 2026-04-08
```

---

**TASK: [QA] Full horizontal overflow Playwright suite**

```
TASK:
  name: "[QA] Playwright — full no-overflow suite across all routes"
  list: Sprint 3
  description: |
    Create tests/responsive/no-overflow.spec.ts. For every page route at 375px, 768px, 1280px:
    assert document.documentElement.scrollWidth <= document.documentElement.clientWidth.
    Routes: /login, /forgot-password, /reset-password, /accept-invite, /dashboard,
    /dashboard/users, /dashboard/users/:id, /dashboard/tenants, /dashboard/tenants/:id,
    /dashboard/roles, /dashboard/audit, /me, /dashboard/settings.
    AC: All assertions pass. This is the release gate test from PRD Section 10.
  assignee_role: QA
  story_points: 3
  priority: high
  tags: [test]
  due_date: 2026-04-10
```

---

**TASK: [QA] Lighthouse mobile audit report**

```
TASK:
  name: "[QA] Lighthouse mobile audit — scores and CLS report for all key pages"
  list: Sprint 3
  description: |
    Run Chrome Lighthouse Mobile on: /login, /dashboard, /dashboard/users, /dashboard/tenants,
    /dashboard/audit. Record: Mobile performance score, CLS, LCP, TBT. Target: >= 90 on /login
    and /dashboard; >= 85 on table pages. CLS < 0.1 on all. Produce audit summary document
    for PO sign-off. Include before/after comparison if baseline was captured at project start.
    AC: Report produced. All pages meet or exceed targets. PO sign-off obtained.
  assignee_role: QA
  story_points: 3
  priority: high
  tags: [test]
  due_date: 2026-04-11
```

---

**TASK: [QA] Accessibility audit — axe-core on key pages**

```
TASK:
  name: "[QA] Accessibility audit — axe-core scan, resolve critical violations"
  list: Sprint 3
  description: |
    Run axe-core via Playwright axe integration on /login, /dashboard, /dashboard/users at 375px.
    Target: zero critical or serious violations introduced by responsive changes. WCAG 2.1 AA.
    Pay particular attention to: drawer aria-modal, hamburger aria-label, card stack tap
    targets, focus order after drawer closes.
    AC: Zero critical/serious violations on scanned pages. Report archived.
  assignee_role: QA
  story_points: 3
  priority: high
  tags: [test]
  due_date: 2026-04-12
```

---

**TASK: [QA] Visual regression screenshot archive**

```
TASK:
  name: "[QA] Visual regression baseline — screenshots at all breakpoints, light + dark"
  list: Sprint 3
  description: |
    Using Playwright screenshot capability, capture all 14 pages at 375px, 768px, 1280px
    in both light and dark mode (28 x 2 = 56 screenshots total). Archive to tests/screenshots/.
    These become the baseline for future regression comparison.
    AC: 56 screenshots captured and committed. File naming convention: [page]-[width]-[theme].png.
  assignee_role: QA
  story_points: 2
  priority: normal
  tags: [test]
  due_date: 2026-04-14
```

---

## 5. Risk Register

| ID | Risk | Probability | Impact | Score | Mitigation | Contingency | Owner | Status |
|----|------|-------------|--------|-------|------------|-------------|-------|--------|
| R-01 | Desktop visual regression introduced by responsive class changes | Medium | High | 12 | ADR-R01 (desktop-first adaptation preserves existing class lists). Every PR must include visual check at 1280px. | Revert individual file PR if regression detected; do not block sprint. | Developer | Open |
| R-02 | Safari iOS renders drawer animation or touch targets differently from Chrome | Medium | Medium | 9 | Architecture specifies standard CSS transform transitions and `position: fixed` — these are well-supported in Safari 16+. QA to test on Safari in Sprint 2. | Document workaround; add -webkit- prefix if needed. | Tester | Open |
| R-03 | CLS > 0.1 caused by sidebar localStorage state loading and triggering layout shift | Low | High | 10 | Architecture Section 10.1: sidebar is `hidden md:flex` on mobile (no render, no shift). useBreakpoint defaults to "lg" on SSR. Transition uses CSS `transition-[width]` not layout properties. | Add `suppressHydrationWarning` on sidebar width container if shift detected. | Developer | Open |
| R-04 | Scope creep — stakeholders request mobile-first rewrite or new features | Low | High | 10 | ADR-R01 is formally documented and signed off. Project charter explicitly excludes mobile-first rewrite. Change requests go through formal change control. | Log as future sprint; do not include in this release. | Project Manager | Open |
| R-05 | Horizontal overflow on a page not caught during manual QA | Medium | Medium | 9 | Playwright automated overflow suite (WBS-13) runs on every page at 375px. Acts as the release gate. | Fix the overflow before release gate can close. | Tester | Open |
| R-06 | MoreHorizontal dropdown in UserCard extends beyond viewport on right side of screen | Medium | Medium | 9 | Architecture specifies `align="end"` on DropdownMenuContent (inheriting from shadcn default). Test at 375px. | Override with custom positioning if needed. | Developer | Open |
| R-07 | Playwright test failures in CI/CD due to auth state required for protected pages | Medium | Low | 6 | Auth tests for protected pages should use stored auth state or mock. Architecture already references existing `test:e2e` suite — follow its auth setup pattern. | Skip auth-dependent assertions if auth mock not available; file follow-up task. | Tester | Open |
| R-08 | Card component code duplication across UserCard, TenantCard, AuditLogCard, RoleCard increases maintenance burden | Low | Low | 4 | Each card is small (30-50 lines per architecture). Shared patterns are documented in architecture Section 8.4. Extract a shared `BaseCard` wrapper only if a fifth card is needed. | Accept duplication for this release per YAGNI. | Developer | Open |
| R-09 | Dark mode regression — mobile-specific classes interact with dark: variants unexpectedly | Medium | Medium | 9 | All new mobile classes use existing design token variables (`bg-card`, `border-border`, `text-semi-black`) which already have dark mode variants defined in globals.css. QA explicitly verifies dark mode at each breakpoint per DoD. | Patch the specific dark: override on the affected element. | Developer / Tester | Open |
| R-10 | Sprint 1 delayed due to sidebar/drawer complexity — cascading delay to Sprint 2 | Low | High | 10 | Foundation tasks (WBS-00) are estimated at 4 pts, low complexity. Auth pages (WBS-03) are independent and can be merged even if sidebar is delayed. Sprint 1 has 5-pt buffer for exactly this risk. | De-scope WBS-05 (user detail) from Sprint 1 to Sprint 2 if sidebar takes longer than expected. | Project Manager | Open |

**Risk scoring:** Probability (Low=1, Medium=2, High=3) x Impact (Low=2, Medium=3, High=4).

---

## 6. Definition of Done

### 6.1 Per Story Definition of Done

A user story is Done when ALL of the following are verified:

**Functional verification:**
- [ ] All Gherkin acceptance criteria in the PRD have been manually verified
- [ ] Verified on Chrome DevTools device simulation at: 375px, 768px, 1280px
- [ ] No horizontal scrollbar appears at 375px viewport width
- [ ] Table/card swap correct: table visible >= 768px; card stack visible < 768px (list pages only)
- [ ] All action buttons and interactive elements reachable without excessive scrolling

**Touch and accessibility:**
- [ ] All interactive elements have a minimum touch target of 44 x 44px (verified by DevTools element inspector bounding box)
- [ ] Password show/hide toggles: bounding box >= 44 x 44px
- [ ] Dropdown menus do not extend beyond viewport edge at 375px
- [ ] Keyboard navigation (Tab order) remains logical after changes

**Quality:**
- [ ] CLS measured in Lighthouse < 0.1 on the affected page at 375px
- [ ] No new TypeScript errors (`tsc --noEmit` passes)
- [ ] No new console errors at any breakpoint
- [ ] Playwright spec for the story is green
- [ ] Code has been peer reviewed by one other agent/developer

**Brand and theme:**
- [ ] Dark mode verified — no regression in dark theme at any breakpoint
- [ ] TigerSoft branding colors and typography unchanged (Tiger Red, Oxford Blue, Serene borders, rounded-[10px] cards)
- [ ] UFO Green status badges unchanged

---

### 6.2 Per Sprint Definition of Done

In addition to all per-story DoD items:

- [ ] All stories in the sprint have passed per-story DoD
- [ ] Cross-browser testing completed on: Chrome desktop, Safari iOS, Chrome Android
- [ ] No new TypeScript errors project-wide
- [ ] No new console errors at any breakpoint on any page
- [ ] Accessibility scan with axe-core — zero critical violations introduced
- [ ] Product Owner sign-off on visual review (live demo or recording)
- [ ] Sprint retrospective completed and action items logged

---

### 6.3 Release Readiness Gate (End of Sprint 2)

All of the following must be true before release tagging:

- [ ] All Must Have stories (RESP-01 through RESP-05) are Done
- [ ] All Should Have stories (RESP-06 through RESP-10) are Done
- [ ] Lighthouse Mobile score >= 90 on `/login` and `/dashboard`
- [ ] Lighthouse Mobile score >= 85 on `/dashboard/users` and `/dashboard/tenants`
- [ ] CLS < 0.1 on all tested pages
- [ ] Zero horizontal overflow on any page at 375px (Playwright no-overflow suite green)
- [ ] Cross-browser verified: Safari iOS 16, Chrome Android, Samsung Internet
- [ ] Zero P1 or P2 bugs open
- [ ] Product Owner formal sign-off recorded

---

### 6.4 QA Checklist Per Page Per Viewport

For each page, at each of 375px, 768px, 1280px:

| Check | Pass criteria |
|-------|--------------|
| No horizontal scrollbar | `scrollWidth <= clientWidth` |
| All text readable | No meaningful truncation; text >= 12px |
| All touch targets | Bounding box >= 44 x 44px |
| No element beyond viewport | Verified by DevTools |
| Table visible (>= 768px) | Table DOM visible |
| Card stack visible (< 768px) | Card DOM visible, table hidden |
| Dark mode no regression | Visual check in dark theme |
| Language toggle no layout break | Switch TH/EN and verify |
| Empty state displays | Trigger empty state and verify centered |
| Loading state displays | Verify spinner or skeleton visible |

---

## 7. RACI Chart

### 7.1 Story-Level RACI

| Task/Activity | Product Owner | Project Manager | Solution Architect | Developer | Tester |
|---------------|:------------:|:---------------:|:-----------------:|:---------:|:------:|
| PRD acceptance criteria definition | A/R | I | C | I | I |
| Architecture decisions (ADRs) | C | I | A/R | C | I |
| Sprint planning | A | R | C | C | C |
| [FE] implementation tasks | I | I | C | A/R | I |
| [QA] test spec authoring | I | I | I | C | A/R |
| PR code review | I | I | C | A/R | C |
| Manual QA execution | I | I | I | C | A/R |
| Story acceptance (demo sign-off) | A/R | C | I | I | I |
| Risk identification and logging | C | A/R | C | C | C |
| Risk mitigation execution | I | A | C | R | R |
| Sprint retrospective facilitation | I | A/R | I | C | C |
| Release readiness gate decision | A | R | C | I | C |
| ClickUp task updates | I | A/R | I | R | R |
| Lessons learned documentation | C | A/R | C | C | C |

### 7.2 File-Level RACI (Key Files)

| File | Responsible | Accountable | Consulted | Informed |
|------|-------------|-------------|-----------|---------|
| `src/hooks/use-breakpoint.ts` | Developer | Developer | Solution Architect | PM |
| `src/components/layout/mobile-drawer.tsx` | Developer | Developer | Solution Architect | PM, Tester |
| `src/app/(dashboard)/layout.tsx` | Developer | Developer | Solution Architect | PM, Tester |
| `src/components/layout/sidebar.tsx` | Developer | Developer | Solution Architect | PM, Tester |
| `src/components/layout/header.tsx` | Developer | Developer | Solution Architect | PM, Tester |
| `src/app/(auth)/*/page.tsx` (4 files) | Developer | Developer | SA | PM, Tester |
| `users/_components/user-card.tsx` | Developer | Developer | SA, PO | PM, Tester |
| `tenants/_components/tenant-card.tsx` | Developer | Developer | SA, PO | PM, Tester |
| `audit/_components/audit-log-card.tsx` | Developer | Developer | SA, PO | PM, Tester |
| `roles/_components/role-card.tsx` | Developer | Developer | SA, PO | PM, Tester |
| `tests/responsive/*.spec.ts` | Tester | Tester | Developer | PM |
| `src/app/globals.css` | Developer | Developer | SA | PM |

### 7.3 Team Structure

```
Project: TGX Auth Console — Responsive Design
Sprint Duration: 2 weeks x 3 sprints

Role                    Agent                           Allocation
---------------------------------------------------------------------------
Product Owner           TigerSoft Product Owner Agent   20% (sprint reviews,
                                                         story acceptance)
Project Manager         TigerSoft PM Agent              30% (planning, tracking,
                                                         risk, ClickUp sync)
Solution Architect      TigerSoft SA Agent              15% (ADR ownership,
                                                         PR architecture review)
Developer               TigerSoft Developer Agent       100% (all [FE] tasks)
Tester                  TigerSoft Tester Agent          100% (all [QA] tasks,
                                                         Playwright suite)
```

---

## 8. Budget and Effort Forecast

### 8.1 Story Point Summary

| Sprint | [FE] Points | [QA] Points | Buffer | Total |
|--------|-------------|-------------|--------|-------|
| Sprint 1 | 29 | 18 | 5 | 52 |
| Sprint 2 | 23 | 18 | 5 | 46 |
| Sprint 3 | 2 | 16 | 4 | 22 |
| **Total** | **54** | **52** | **14** | **120** |

**Note:** Story points across Dev and QA are counted separately since Developer and Tester work in parallel on different task streams.

### 8.2 Effort by Category

| Category | Files Affected | Story Points |
|----------|---------------|--------------|
| New shared components (hook, drawer, CSS var) | 3 new files | 4 |
| Navigation (layout, sidebar, header) | 3 modified | 8 |
| Auth pages (login, forgot, reset, accept) | 4 modified | 4 |
| Card components (4 cards) | 4 new files | 11 |
| Page updates (toolbar + card swap + dialogs) | 8 modified | 18 |
| QA specs and manual testing | 7 new spec files | 36 |
| Sprint 3 E2E, Lighthouse, a11y, screenshots | — | 11 |
| **Total implementation** | **6 new + 15 modified** | **92** |
| **Buffer (across 3 sprints)** | | **14** |
| **Grand total** | | **106** |

### 8.3 Contingency

Per PM planning standards, a 20% contingency is built in:
- Timeline: Sprint dates include 14 total buffer points distributed across all three sprints (13% of total sprint capacity).
- The Sprint 3 Could Have stories (WBS-11, WBS-12) act as additional contingency absorbers — they can be deferred to the backlog without impacting the release readiness gate.
- The Sprint 3 QA activities can be partially overlapped with any Sprint 2 overflow if needed.

### 8.4 EVM Reference Points

| Metric | Sprint 1 Target | Sprint 2 Target | Sprint 3 Target |
|--------|----------------|----------------|----------------|
| Planned Value (PV) | 52 pts completed | 98 pts cumulative | 120 pts cumulative |
| SPI target | >= 1.0 | >= 1.0 | >= 1.0 |
| CPI target | N/A (agent-based, no monetary budget) | N/A | N/A |
| Sprint review action | SPI < 0.8: escalate scope to PM; consider deferring WBS-05 | SPI < 0.8: defer Sprint 3 Could Have items; protect release gate | SPI < 0.8: release without Could Have items |

---

## Appendix A: Implementation Order Reference

Execute in this order within each sprint to minimize integration risk:

**Sprint 1 (strict order for foundation items):**
1. `use-breakpoint.ts` (WBS-00) — no deps
2. `mobile-drawer.tsx` (WBS-00) — needs hook
3. `layout.tsx` (WBS-01) — needs drawer
4. `sidebar.tsx` (WBS-01) — needs layout
5. `header.tsx` (WBS-01) — needs layout
6. Auth pages x4 (WBS-03) — independent; run in parallel with steps 3-5
7. `user-card.tsx` (WBS-04) — needs layout shell
8. `users/page.tsx` (WBS-04) — needs UserCard
9. `users/[id]/page.tsx` (WBS-05) — needs layout shell

**Sprint 2 (can be partially parallelized):**
- Group A (parallel): `tenant-card.tsx` + `tenants/page.tsx` + `tenants/[id]/page.tsx`
- Group B (parallel): `role-card.tsx` + `roles/page.tsx`
- Group C (independent): `me/page.tsx`
- Group D (last, most complex): `audit-log-card.tsx` + `audit/page.tsx`

**Sprint 3:**
- `settings/page.tsx` + `dashboard/page.tsx` (minor, fast)
- QA suite activities (can start as soon as Sprint 2 merges are complete)

---

## Appendix B: New Files to Create

| File | When | Purpose |
|------|------|---------|
| `src/hooks/use-breakpoint.ts` | Sprint 1, Day 1 | Breakpoint detection hook |
| `src/components/layout/mobile-drawer.tsx` | Sprint 1, Day 1-2 | Mobile navigation drawer |
| `src/app/(dashboard)/dashboard/users/_components/user-card.tsx` | Sprint 1 | User list mobile card |
| `src/app/(dashboard)/dashboard/tenants/_components/tenant-card.tsx` | Sprint 2 | Tenant list mobile card |
| `src/app/(dashboard)/dashboard/audit/_components/audit-log-card.tsx` | Sprint 2 | Audit log mobile card |
| `src/app/(dashboard)/dashboard/roles/_components/role-card.tsx` | Sprint 2 | Role list mobile card |
| `tests/responsive/sidebar-drawer.spec.ts` | Sprint 1 | Playwright sidebar spec |
| `tests/responsive/auth-pages.spec.ts` | Sprint 1 | Playwright auth spec |
| `tests/responsive/users-page.spec.ts` | Sprint 1 | Playwright users spec |
| `tests/responsive/user-detail.spec.ts` | Sprint 1 | Playwright user detail spec |
| `tests/responsive/tenants-page.spec.ts` | Sprint 2 | Playwright tenants spec |
| `tests/responsive/tenant-detail.spec.ts` | Sprint 2 | Playwright tenant detail spec |
| `tests/responsive/audit-page.spec.ts` | Sprint 2 | Playwright audit spec |
| `tests/responsive/roles-page.spec.ts` | Sprint 2 | Playwright roles spec |
| `tests/responsive/profile-page.spec.ts` | Sprint 2 | Playwright profile spec |
| `tests/responsive/no-overflow.spec.ts` | Sprint 3 | Full no-overflow release gate |

---

## Appendix C: Modified Files Reference

| File | Sprint | Key Changes |
|------|--------|-------------|
| `src/app/globals.css` | Sprint 1 | Add `--size-touch-target` CSS variable |
| `src/app/(dashboard)/layout.tsx` | Sprint 1 | mobileMenuOpen state, MobileDrawer, main padding |
| `src/components/layout/sidebar.tsx` | Sprint 1 | onNavigate prop, mobile drawer mode, useBreakpoint |
| `src/components/layout/header.tsx` | Sprint 1 | onMenuOpen prop, hamburger button, px-4 md:px-6 |
| `src/app/(auth)/login/page.tsx` | Sprint 1 | p-6 sm:p-10, password toggle w-11 h-11, forgot link min-h |
| `src/app/(auth)/forgot-password/page.tsx` | Sprint 1 | p-6 sm:p-10 (2 instances) |
| `src/app/(auth)/reset-password/page.tsx` | Sprint 1 | p-6 sm:p-10 (3 instances), password toggles |
| `src/app/(auth)/accept-invite/page.tsx` | Sprint 1 | p-6 sm:p-10 (2 instances), password toggle |
| `src/app/(dashboard)/dashboard/users/page.tsx` | Sprint 1 | Toolbar, table/card swap |
| `src/app/(dashboard)/dashboard/users/[id]/page.tsx` | Sprint 1 | Header flex-col sm:flex-row, role min-h |
| `src/app/(dashboard)/dashboard/tenants/page.tsx` | Sprint 2 | Toolbar, table/card swap, dialog |
| `src/app/(dashboard)/dashboard/tenants/[id]/page.tsx` | Sprint 2 | Header, CopyField, admin actions, dialog |
| `src/app/(dashboard)/dashboard/audit/page.tsx` | Sprint 2 | Filter form, table/card swap, pagination |
| `src/app/(dashboard)/dashboard/roles/page.tsx` | Sprint 2 | Toolbar, tab pills, table/card swap, dialog |
| `src/app/(dashboard)/me/page.tsx` | Sprint 2 | Name grid, password toggles |
| `src/app/(dashboard)/dashboard/settings/page.tsx` | Sprint 3 | Session input width |
| `src/app/(dashboard)/dashboard/page.tsx` | Sprint 3 | Quick actions grid (minor) |

---

*Document prepared by: TigerSoft Project Manager Agent*
*Based on PRD: `docs/auth-system/responsive-design-prd.md`*
*Based on Architecture: `docs/auth-system/responsive-design-architecture.md`*
*Codebase baseline: commit `9483143` (branch: `fix/module-config-tenant-provisioning`)*
*Date: 2026-03-05*
