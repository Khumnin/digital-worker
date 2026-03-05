# Product Requirements Document
# TGX Auth Console — Responsive Design

**Document version:** 1.0
**Date:** 2026-03-05
**Product Owner:** TigerSoft Product Team
**Status:** Ready for Sprint Planning
**Project:** Auth Admin UI — Responsive Design Initiative

---

## Table of Contents

1. [Problem Statement](#1-problem-statement)
2. [Goals and Success Metrics](#2-goals-and-success-metrics)
3. [User Personas](#3-user-personas)
4. [Current State Assessment](#4-current-state-assessment)
5. [Feature List — MoSCoW Prioritization](#5-feature-list--moscow-prioritization)
6. [User Stories with Acceptance Criteria](#6-user-stories-with-acceptance-criteria)
7. [Non-Functional Requirements](#7-non-functional-requirements)
8. [Out of Scope](#8-out-of-scope)
9. [Sprint Capacity Plan](#9-sprint-capacity-plan)
10. [Definition of Done](#10-definition-of-done)

---

## 1. Problem Statement

### The Problem
The TGX Auth Console (`auth-admin-ui`) is a Next.js web application built exclusively for desktop viewport widths. The current layout architecture uses a fixed-width sidebar (298px expanded, 60px collapsed) occupying a permanent horizontal slice of the viewport inside a `flex h-screen overflow-hidden` shell. Every page renders inside `<main className="flex-1 overflow-y-auto p-6">` with no mobile breakpoint accommodations. Data-heavy pages (Tenants, Users, Roles, Audit Log) use `<Table>` components that overflow on screens narrower than approximately 900px, causing horizontal scroll at best and completely broken layouts at worst.

### For Whom
Admin users and Super Admin users of TigerSoft client organizations who need to take action — invite a user, check audit logs, suspend a tenant, or update settings — while away from their desks. Additionally, regular end-users arriving at auth pages (Login, Accept Invite, Forgot Password, Reset Password) via email links typically open those links on mobile devices.

### Why It Matters
- Email-based flows (Accept Invite, Reset Password) are triggered from email. The majority of email opens in Thailand occur on mobile devices. A broken mobile layout at the activation step creates failed onboarding and increased support requests.
- Admin tasks such as checking a user status, resending an invitation, or reviewing an audit event are frequently time-sensitive. Forcing users back to a desktop creates unnecessary delays.
- TigerSoft's Brand DNA is defined as "Agile, Empowering, Seamless." A non-responsive product is inconsistent with these brand values.

---

## 2. Goals and Success Metrics

| Goal | Success Metric | Target |
|------|---------------|--------|
| All auth pages usable on mobile | Lighthouse Mobile score | >= 90 |
| Zero horizontal scrollbars on any page at 375px width | Manual QA pass rate | 100% |
| No layout shift on page load | Core Web Vitals CLS | < 0.1 |
| Touch targets meet accessibility standards | Minimum tap target size | >= 44 x 44px |
| Dashboard shell renders correctly on tablet | Visual regression pass | 100% on 768px viewport |
| Table data remains accessible on mobile | Card/stack pattern replaces table on mobile | All data pages pass |
| Sidebar navigation usable on mobile | Drawer/overlay pattern in place | Functional on 375px |

---

## 3. User Personas

### Persona A — Napat, Super Admin (Desktop primary, occasional tablet)
**Role:** Platform Super Admin at TigerSoft
**Devices:** MacBook Pro (primary), iPad Pro (occasional), iPhone 14 (rare, emergency use)
**Pain points:** When travelling, cannot provision a new tenant or check why a user is suspended. The current sidebar occupies too much space even on a 1024px tablet, leaving tables cramped.
**Goals:** Manage tenants and users from any device without losing context.

### Persona B — Somchai, Tenant Admin (Mixed desktop/mobile)
**Role:** HR Admin at a client company, manages their tenant's users and roles
**Devices:** Windows PC (office), Samsung Galaxy S23 (commuting, field work)
**Pain points:** Receives "new user registered" notifications on phone, wants to quickly approve or resend invite without waiting to get back to PC. Current mobile view is broken — table overflows, sidebar covers content.
**Goals:** Invite users, check user status, and update settings from his phone during the workday.

### Persona C — Ploy, Invited User (Mobile only for onboarding)
**Role:** New employee at a client company
**Devices:** iPhone 13 (primary), no access to company PC until IT setup is complete
**Pain points:** Receives invite email on phone, clicks the activation link, and lands on a form with padding and sizing designed for desktop. Password fields are cramped and touch targets are too small.
**Goals:** Successfully activate her account via mobile with minimal friction.

### Persona D — Krit, Regular User (Mobile, self-service profile)
**Role:** Staff member with no admin role
**Devices:** iPhone 12, Android tablet
**Pain points:** Needs to update his display name and change his password via `/me` but the two-column name grid is too small on mobile.
**Goals:** Self-serve his profile and password without needing desktop access.

---

## 4. Current State Assessment

This section documents the specific responsive gaps found in each file during the codebase review.

### 4.1 Layout Shell — `src/app/(dashboard)/layout.tsx`

**Current code:**
```tsx
<div className="flex h-screen overflow-hidden bg-page-bg">
  <Sidebar lang={lang} />
  <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
    <Header lang={lang} onLangChange={handleLangChange} />
    <main className="flex-1 overflow-y-auto p-6">
```

**Problem:** `flex h-screen` with a persistent sidebar renders identically at all widths. On mobile the sidebar (60px collapsed minimum) immediately consumes critical viewport width. No mechanism exists to hide the sidebar or render a drawer/sheet overlay on small screens. No `md:` or `lg:` breakpoint logic is applied.

### 4.2 Sidebar — `src/components/layout/sidebar.tsx`

**Current code:**
```tsx
<aside className={cn(
  "flex flex-col h-screen bg-sidebar border-r border-sidebar-border ...",
  expanded ? "w-[298px]" : "w-[60px]"
)}>
```

**Problem:** Fixed pixel widths with no responsive consideration. The collapse toggle is a desktop UX pattern. On mobile, even the 60px collapsed state occupies meaningful viewport real estate. No `hidden` class at any breakpoint. No hamburger/drawer pattern exists.

### 4.3 Header — `src/components/layout/header.tsx`

**Current code:**
```tsx
<header className="h-[64px] ... flex items-center justify-between px-6 shrink-0">
  <h2 className="text-[15px] font-semibold text-semi-black">{title ?? "Dashboard"}</h2>
  <div className="flex items-center gap-3">
    <ThemeToggle />
    <button ...><Globe size={16} /><span>...</span></button>
    <DropdownMenu>...</DropdownMenu>
  </div>
```

**Problem:** No hamburger menu button to open the mobile sidebar drawer. The `px-6` padding is appropriate for desktop but leaves very little room for the three right-side controls on narrow screens. The user email text is already `hidden sm:block` — a sign that mobile awareness was partially attempted but not completed.

### 4.4 Dashboard Page — `src/app/(dashboard)/dashboard/page.tsx`

**Current code:**
```tsx
<div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
```
and
```tsx
<div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
```

**Assessment:** Partially responsive. The stat cards grid already uses responsive breakpoints. The Quick Actions grid collapses to 2 columns on mobile. This page needs the least work — its content is already stack-friendly.

**Remaining gap:** Quick Actions buttons text may truncate awkwardly on very small widths with 2-column grid.

### 4.5 Tenants Page — `src/app/(dashboard)/dashboard/tenants/page.tsx`

**Current code:**
```tsx
<Table>
  <TableHeader>
    <TableRow>
      <TableHead>Name</TableHead>
      <TableHead>Slug</TableHead>
      <TableHead>Status</TableHead>
      <TableHead>Modules</TableHead>
      <TableHead>Created</TableHead>
      <TableHead />  {/* actions */}
    </TableRow>
  </TableHeader>
```

**Problem:** Six-column table with no responsive handling. On mobile this will overflow horizontally. The toolbar `flex items-center justify-between gap-3` with a search input and a "Provision Tenant" button will wrap awkwardly on 375px — the button label is long and will push below the search bar.

### 4.6 Users Page — `src/app/(dashboard)/dashboard/users/page.tsx`

**Current code:**
```tsx
<div className="flex items-center gap-3 flex-wrap">
  <div className="relative flex-1 min-w-[200px] max-w-sm">
  <div className="flex items-center gap-1.5">  {/* Status filter */}
  <div className="flex items-center gap-1.5">  {/* Module filter */}
  {/* Clear filters button */}
  <Button className="ml-auto">Invite User</Button>
```

**Problem:** The toolbar has four interactive elements plus a CTA button in one flex row. On mobile these wrap via `flex-wrap` but the result is a multi-row filter area that takes up excessive vertical space before the actual table appears. The five-column table (User, Roles, Status, Joined, Actions) overflows on mobile. Role badges in the Roles column can produce very wide cells.

### 4.7 Tenant Detail Page — `src/app/(dashboard)/dashboard/tenants/[id]/page.tsx`

**Current code:**
```tsx
<div className="space-y-5 max-w-4xl">
  <div className="flex items-center gap-3">
    {/* Back button + name + status badge + Suspend/Activate button */}
  </div>
  <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
```

**Problem:** The header row `flex items-center gap-3` contains: back button, tenant name/slug block (flex-1), status badge, and two action buttons. On mobile, these will compress and the action buttons may overflow or require very small tap targets. The `lg:grid-cols-2` card grid correctly collapses to 1 column below lg breakpoint. The administrators list and module config card below are essentially already single-column — low effort to fix.

**Specific issue:** The `CopyField` component renders `flex items-center gap-2` with a monospace value that can be very long (UUIDs). The `break-all` class on the code element helps, but the copy button's 14px icon may be too small for reliable touch targeting (< 44px hit area).

### 4.8 User Detail Page — `src/app/(dashboard)/dashboard/users/[id]/page.tsx`

**Current code:**
```tsx
<div className="space-y-5 max-w-3xl">
  <div className="flex items-center gap-3">
    {/* Back + name/email + status + up to 3 action buttons */}
  </div>
```

**Problem:** The header row can contain up to four items: back button, user info (flex-1), status badge, and up to three action buttons (Resend Invite, Send Password Reset, Suspend) simultaneously. On a 375px screen this row becomes completely unusable. The account info card uses `flex justify-between` rows — these work fine on mobile. The roles card with checkboxes is single-column and works well.

### 4.9 Roles Page — `src/app/(dashboard)/dashboard/roles/page.tsx`

**Current code:**
```tsx
<div className="flex items-center gap-2 flex-wrap">
  {tabs.map(...)}  {/* pill tabs */}
</div>
<div className="flex items-center justify-between">
  <p>N roles in ...</p>
  <Button>Create Role</Button>
</div>
```

**Problem:** Tab pills use `flex-wrap` and will reflow gracefully. The toolbar with count + CTA is a simple 2-item flex that handles 375px fine. The main issue is the six-column table (Name, Description, Module, Type, Created, Delete). Description column with `max-w-[320px]` will cause overflow on mobile.

### 4.10 Audit Log Page — `src/app/(dashboard)/dashboard/audit/page.tsx`

**Current code:**
```tsx
<form className="flex items-end gap-3 flex-wrap">
  <Select .../>   {/* Action filter — min-w-[190px] */}
  <div className="flex items-end gap-2">
    <Input type="date" className="w-[148px]" />
    <Input type="date" className="w-[148px]" />
  </div>
  <Button>Apply</Button>
</form>
```

**Problem:** The filter bar uses `flex-wrap` so it reflows, but the fixed-width date inputs (`w-[148px]`) on a 375px viewport will be very tight side by side inside the `flex items-end gap-2` wrapper. The five-column audit table (Time, Action, Actor, Target, IP) is one of the densest tables — Time column alone (`whitespace-nowrap`) will be around 160px wide, making this the most challenging table to adapt for mobile.

### 4.11 Settings Page — `src/app/(dashboard)/dashboard/settings/page.tsx`

**Current code:**
```tsx
<div className="space-y-5 max-w-2xl">
  <div className="flex items-center justify-between">
    {/* MFA toggle row */}
  </div>
  <Input className="max-w-[160px]" />  {/* Session duration */}
```

**Assessment:** This page is nearly responsive already. The `max-w-2xl` container, vertical card stacking, and simple field layout work well on mobile. The MFA toggle row `flex items-center justify-between` works on all widths. The only edge case is the Integration Endpoints card with long monospace URL strings that use `break-all` — this already handles wrapping correctly.

**Remaining gap:** The `flex justify-end` Save button position is fine. The `max-w-[160px]` session duration input may need `w-full` on mobile to avoid an orphaned small input.

### 4.12 My Profile Page — `src/app/(dashboard)/me/page.tsx`

**Current code:**
```tsx
<div className="space-y-5 max-w-2xl">
  <div className="grid grid-cols-2 gap-3">
    {/* First Name + Last Name side by side */}
  </div>
```

**Problem:** The `grid grid-cols-2` for first/last name has no responsive breakpoint. On 375px, 2-column inputs become uncomfortably narrow (approximately 155px each after gaps and padding). Needs `grid-cols-1 sm:grid-cols-2`.

### 4.13 Auth Pages — Login, Forgot Password, Reset Password, Accept Invite

**Current code pattern across all four:**
```tsx
<div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
  <div className="w-full max-w-[440px] bg-card rounded-[10px] p-10 shadow-sm">
```

**Assessment:** These are the best-positioned pages for mobile. The centered card pattern with `px-4` on the outer container and `max-w-[440px]` on the card already produces a good mobile layout. The `p-10` (40px) padding on the card is slightly generous for small screens (on 375px the content area becomes 375 - 8px - 8px outer - 80px padding = 279px). This is workable but would benefit from `p-6 sm:p-10` to give a bit more breathing room.

**Touch target concern:** The `h-12` inputs (48px) are appropriate. The show/hide password buttons are positioned absolutely and their hit areas depend on the icon size (18px) with no explicit padding — they may not reliably hit 44px.

---

## 5. Feature List — MoSCoW Prioritization

### Must Have (Sprint 1)

| ID | Feature | Rationale |
|----|---------|-----------|
| M-01 | Mobile sidebar drawer with hamburger toggle | Without this, the dashboard layout is completely broken on mobile |
| M-02 | Responsive dashboard layout shell | Core structural change — all other pages depend on this |
| M-03 | Auth pages mobile polish (Login, Forgot Password, Reset Password, Accept Invite) | Most likely reached via mobile email links; blocking for onboarding |
| M-04 | Users page — mobile card stack pattern replacing table | Most used admin page by Tenant Admins |
| M-05 | User Detail page — responsive action header | Up to 3 action buttons in one flex row collapse on mobile |

### Should Have (Sprint 2)

| ID | Feature | Rationale |
|----|---------|-----------|
| S-01 | Tenants page — mobile card stack pattern replacing table | Super Admin use case; table has 6 columns |
| S-02 | Tenant Detail page — responsive header and touch-safe copy buttons | Detail pages with multi-button headers |
| S-03 | Audit Log page — mobile stacked log entries replacing table | Dense 5-column table; date filter layout fix |
| S-04 | Roles page — mobile card stack replacing table | 6-column table with long description column |
| S-05 | My Profile page — responsive name grid | Two-column name grid breaks on mobile |

### Could Have (Sprint 3 / Backlog)

| ID | Feature | Rationale |
|----|---------|-----------|
| C-01 | Bottom navigation bar for mobile (alternative to drawer) | Enhanced mobile UX; faster switching between pages |
| C-02 | Mobile-optimized filter sheets (slide-up panel for filter controls) | Avoids multi-row filter wrapping on narrow screens |
| C-03 | Swipe-to-dismiss sidebar drawer | Native mobile feel |
| C-04 | Settings page — full-width input at mobile | Minor; current layout is acceptable but not ideal |
| C-05 | Dashboard quick actions — single-column at mobile | Minor UX refinement |
| C-06 | Sticky action bar on detail pages at mobile | Prevents scrolling past header to reach CTA buttons |

### Won't Have (This Release)

| ID | Feature | Rationale |
|----|---------|-----------|
| W-01 | Native iOS or Android application | Out of scope; web-only product |
| W-02 | Push notifications for mobile | Requires native wrapper |
| W-03 | Offline mode / PWA caching | Not required for admin tool |
| W-04 | Fingerprint/biometric login for mobile web | Future roadmap |
| W-05 | Custom mobile-specific data visualizations/charts | No charts exist in current product |

---

## 6. User Stories with Acceptance Criteria

### Sprint 1 Stories — Must Have

---

#### RESP-01: Mobile Sidebar Drawer

**As a** tenant admin using a mobile device,
**I want** a hamburger menu button that opens the navigation as a slide-in drawer,
**so that** I can navigate between sections without the sidebar permanently consuming screen width.

**Story Points:** 8
**MoSCoW:** Must Have
**INVEST Check:**
- Independent: Can be developed alongside other stories since it only modifies `sidebar.tsx` and `header.tsx` and `layout.tsx`
- Negotiable: Drawer animation style and overlay opacity are negotiable
- Valuable: Without this, no other dashboard page is usable on mobile
- Estimable: Clear scope — three files, one new state variable for drawer open/close
- Small: Fits within a single sprint
- Testable: Yes — verified via Playwright viewport tests

**Acceptance Criteria:**

```gherkin
Feature: Mobile sidebar navigation drawer

  Background:
    Given the user is authenticated and on any dashboard page
    And the viewport width is less than 768px

  Scenario: Sidebar is hidden by default on mobile
    When the page loads on a 375px wide viewport
    Then the sidebar is not visible
    And no horizontal scrollbar appears
    And the main content area occupies the full viewport width

  Scenario: Opening the drawer via hamburger button
    Given the sidebar is hidden
    When the user taps the hamburger icon in the header
    Then the sidebar slides in from the left as an overlay
    And a semi-transparent backdrop appears over the main content
    And the sidebar is fully visible within the viewport
    And the hamburger icon changes to a close (X) icon

  Scenario: Closing the drawer by tapping outside
    Given the sidebar drawer is open
    When the user taps the backdrop overlay
    Then the sidebar slides out and is hidden
    And the backdrop is removed
    And no horizontal scrollbar appears

  Scenario: Closing the drawer after navigation
    Given the sidebar drawer is open
    When the user taps a navigation link (e.g., "Users")
    Then the drawer closes automatically
    And the user is navigated to the selected page

  Scenario: Desktop sidebar is unaffected
    Given the viewport width is 1024px or wider
    Then the sidebar renders inline (not as a drawer)
    And no hamburger button is visible in the header
    And the existing collapse/expand toggle remains functional

  Scenario: Touch target compliance
    Given the sidebar drawer is open
    Then every navigation link in the drawer has a minimum touch target of 44 x 44px
    And the close (X) button has a minimum touch target of 44 x 44px
```

**Technical Notes:**
- Add `open` state to `DashboardLayout` or a new `MobileMenuContext`
- On `< md` (768px): render `Sidebar` inside a `Sheet`/overlay component with `position: fixed inset-0 z-50`
- Hamburger button is added to `Header` component with `md:hidden` visibility
- On `>= md`: sidebar renders in the existing inline flex position with `hidden md:flex`
- Preserve localStorage-backed expand/collapse state for desktop; drawer state is session-only for mobile

---

#### RESP-02: Responsive Dashboard Layout Shell

**As a** user of TGX Auth Console on any device,
**I want** the overall page shell (header + main content) to fill and use available screen width correctly,
**so that** content is readable without horizontal scrolling or clipped areas on any viewport size.

**Story Points:** 3
**MoSCoW:** Must Have
**INVEST Check:** Independent of individual page content changes; delivers structural fix.

**Acceptance Criteria:**

```gherkin
Feature: Responsive layout shell

  Scenario: Mobile layout uses full width
    Given the user is authenticated
    And the viewport is 375px wide
    When any dashboard page loads
    Then the main content area width equals 100vw
    And no element extends beyond the viewport width
    And no horizontal scrollbar is present on the body

  Scenario: Header adapts to mobile
    Given the viewport is 375px wide
    When any dashboard page loads
    Then the header renders at full width
    And the page title is visible and not truncated beyond legibility
    And the hamburger menu button is visible on the left of the header
    And the user avatar/dropdown is visible on the right

  Scenario: Header controls are accessible on mobile
    Given the viewport is 375px wide
    And the user menu dropdown is open
    Then the dropdown panel does not extend beyond the right viewport edge
    And all dropdown items have a minimum touch target height of 44px

  Scenario: Content padding adapts by breakpoint
    Given the viewport is 375px wide
    Then the main content area has horizontal padding of at least 16px
    And the main content area has horizontal padding of at most 24px

    Given the viewport is 768px wide
    Then the main content area has horizontal padding of at least 20px

    Given the viewport is 1280px wide
    Then the main content area padding matches the existing p-6 (24px)

  Scenario: Page does not shift on load (CLS)
    When any dashboard page loads on any device
    Then the Cumulative Layout Shift score is less than 0.1
    As measured by Chrome DevTools Lighthouse
```

**Technical Notes:**
- Change `<main className="flex-1 overflow-y-auto p-6">` to `<main className="flex-1 overflow-y-auto p-4 sm:p-6">`
- Header: add `md:hidden` hamburger button; `onMenuOpen` prop passed from layout
- Ensure `min-w-0` is applied to the flex child containing header + main

---

#### RESP-03: Auth Pages Mobile Polish

**As an** invited user or forgotten-password user opening a link on my mobile phone,
**I want** the auth forms to display correctly and be easy to interact with on a small screen,
**so that** I can complete account setup or password reset without needing a desktop.

**Story Points:** 3
**MoSCoW:** Must Have
**INVEST Check:** Auth pages are a separate route group `(auth)` and fully independent of dashboard changes.

**Acceptance Criteria:**

```gherkin
Feature: Auth pages — mobile responsive

  Background:
    Given the user opens an auth page on a 375px wide viewport
    And the page is one of: /login, /forgot-password, /reset-password, /accept-invite

  Scenario: Card padding adapts on mobile
    When the page loads at 375px
    Then the auth card has horizontal padding of at least 24px (not 40px)
    And all form inputs are visible without horizontal scroll
    And the card does not overflow the viewport width

  Scenario: Input fields are full width and touch-friendly
    Given any auth page is open at 375px
    Then all text/email/password inputs have height of at least 48px
    And all inputs span the full card width
    And the user can tap any input with a standard fingertip without accidental adjacent taps

  Scenario: Password show/hide toggle has adequate touch target
    Given a password field with a show/hide toggle button is visible
    Then the toggle button's touchable area is at least 44 x 44px
    Even if the icon itself is smaller

  Scenario: Submit button spans full width on mobile
    Then the primary submit button is full-width on viewports < 640px
    And the button height is at least 48px
    And the button text is fully visible without truncation

  Scenario: Forgot password link is tappable
    Given the login page is open at 375px
    Then the "Forgot password?" link has a touch target of at least 44px height
    And the link is positioned with adequate spacing from the password input

  Scenario: Accept invite page with no token shows error gracefully
    Given the viewport is 375px
    When /accept-invite is opened without a token parameter
    Then the error message displays correctly within the card
    And the card does not overflow the viewport

  Scenario: Success states display correctly on mobile
    Given any auth page is in a success/confirmation state
    When viewed at 375px
    Then the success icon, heading, body text, and CTA button all fit within the card
    And the CTA button is full-width and at least 48px tall
```

**Technical Notes:**
- Change `p-10` to `p-6 sm:p-10` on all four auth page card containers
- Add explicit `min-h-[44px]` or padding to the show/hide password absolute button
- All auth pages already use `px-4` on the outer container — no change needed there
- Applies to: `(auth)/login/page.tsx`, `(auth)/forgot-password/page.tsx`, `(auth)/reset-password/page.tsx`, `(auth)/accept-invite/page.tsx`

---

#### RESP-04: Users Page — Mobile Card Stack

**As a** tenant admin on a mobile device,
**I want** the Users list to display as a vertical card stack instead of a table,
**so that** I can read user information and take actions (view, suspend, resend invite) without horizontal scrolling.

**Story Points:** 8
**MoSCoW:** Must Have
**INVEST Check:** Scoped to `users/page.tsx` only; pattern (hide table on mobile, show card stack) is reusable across other list pages.

**Acceptance Criteria:**

```gherkin
Feature: Users page — responsive list

  Scenario: Table is replaced by cards on mobile
    Given the user is on /dashboard/users
    And the viewport is 375px wide
    Then the data table (TableHeader, TableBody) is not visible
    And each user is displayed as a card containing:
      | field         | content                            |
      | Avatar/Initial| first letter of display name       |
      | Display name  | full name, wrapping if needed      |
      | Email         | full email, wrapping if needed     |
      | Status badge  | colored pill (active/inactive/pending) |
      | Role badges   | up to 3 role badges, with overflow indicator |
      | Action button | tap target >= 44x44px, reveals dropdown |

  Scenario: Table is shown on tablet and desktop
    Given the viewport is 768px or wider
    Then the standard data table is visible
    And the mobile card stack is hidden

  Scenario: Filter toolbar is accessible on mobile
    Given the viewport is 375px
    When the user scrolls to the top of the page
    Then the search input is full-width on mobile
    And the Status and Module filter selects stack vertically or in a 2-column grid below the search
    And the "Invite User" button is full-width at the bottom of the filter area
    And the total height of the filter area does not exceed 180px

  Scenario: Invite User button is always visible on mobile
    Given the viewport is 375px
    Then the "Invite User" button is visible without horizontal scrolling
    And the button has a tap target of at least 44px height
    And the button label is fully visible (not truncated)

  Scenario: Empty state displays correctly on mobile
    Given no users match the current filter
    And the viewport is 375px
    Then the empty state icon and message are centered
    And the empty state fills the card width without overflow

  Scenario: User card actions open a dropdown on tap
    Given a user card is visible at 375px
    When the user taps the more-actions button on a user card
    Then a dropdown appears with options: "View details", "Suspend" (if applicable), "Resend Invite" (if applicable)
    And each dropdown item has a minimum touch target of 44px
    And the dropdown does not extend beyond the viewport edge
```

**Technical Notes:**
- Implement a `UserCard` sub-component that renders the card layout
- Use `hidden md:block` on the `<Table>` wrapper div
- Use `block md:hidden` on a new `<div className="space-y-2">` containing `UserCard` instances
- Filter toolbar: `flex-wrap` is already applied; change to `grid grid-cols-1 gap-2 sm:flex sm:flex-wrap` for cleaner mobile stacking
- The `ml-auto` on the Invite User button should become `w-full sm:w-auto sm:ml-auto` on mobile

---

#### RESP-05: User Detail Page — Responsive Action Header

**As an** admin on a mobile device viewing a user detail page,
**I want** the page header (user name, status, and action buttons) to display in a readable layout,
**so that** I can identify the user and take actions without UI elements overflowing or being too small to tap.

**Story Points:** 5
**MoSCoW:** Must Have
**INVEST Check:** Scoped to `users/[id]/page.tsx` header section; independent of sidebar and list page changes.

**Acceptance Criteria:**

```gherkin
Feature: User detail page — responsive header

  Scenario: Header wraps gracefully on mobile
    Given the user navigates to /dashboard/users/:id on a 375px viewport
    Then the back button, user name/email, and status badge appear on one row
    And action buttons (Resend Invite, Send Password Reset, Suspend) appear on a second row below
    And no element extends beyond the viewport width
    And all buttons have a minimum tap target of 44px height

  Scenario: Full header on tablet and desktop is unchanged
    Given the viewport is 768px or wider
    Then the header renders as a single horizontal row
    And the layout matches the existing desktop design

  Scenario: Account Info card is readable on mobile
    Given the viewport is 375px
    When the Account Info card is visible
    Then each row (User ID, Tenant, Status, Joined) displays label and value
    And long values (e.g., UUID) truncate with ellipsis or wrap within the card boundaries
    And no card content overflows horizontally

  Scenario: Roles checkboxes are touch-friendly on mobile
    Given the Roles card is visible at 375px
    Then each role checkbox row has a minimum height of 44px
    And the checkbox input is at least 20x20px
    And the role name and description are fully readable without truncation beyond the card width

  Scenario: Save Roles button is accessible on mobile
    Given the Roles card header is visible at 375px
    Then the "Save Roles" button has a tap target of at least 44px height
    And is positioned without overlapping the card title
```

**Technical Notes:**
- Wrap the action buttons `div` with `flex flex-wrap gap-2 justify-end` and change the header to `flex flex-wrap items-center gap-3`
- On mobile (`< sm`), action buttons move below: use `flex-col sm:flex-row` on the outer header div and `flex-wrap` on the button group
- Consider a `flex-col gap-3 sm:flex-row sm:items-center` pattern for the entire header

---

### Sprint 2 Stories — Should Have

---

#### RESP-06: Tenants Page — Mobile Card Stack

**As a** super admin on a mobile device,
**I want** the Tenants list to display as a vertical card stack instead of a table,
**so that** I can review tenant status and take actions from my phone.

**Story Points:** 8
**MoSCoW:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: Tenants page — responsive list

  Scenario: Table replaced by cards on mobile
    Given the user is on /dashboard/tenants
    And the viewport is 375px
    Then the data table is not visible
    And each tenant is displayed as a card containing:
      | field       | content                          |
      | Name        | tenant display name              |
      | Slug        | monospace slug text              |
      | Status      | colored status badge             |
      | Modules     | module badges (flex-wrap)        |
      | Created     | localized date string            |
      | Actions     | dropdown button (>= 44x44px)     |

  Scenario: Table visible on tablet and desktop
    Given the viewport is 768px or wider
    Then the standard table is visible and the card stack is hidden

  Scenario: Toolbar adapts on mobile
    Given the viewport is 375px
    Then the search input is full-width
    And the "Provision Tenant" button is below the search input and full-width
    And the button tap target is at least 44px height

  Scenario: Provision Tenant dialog is mobile-friendly
    Given the viewport is 375px
    When the user taps "Provision Tenant"
    Then the dialog opens full-width (or near full-width) on mobile
    And all form inputs are visible and scrollable within the dialog
    And the Cancel and Provision buttons are both visible without scrolling to find them
    And the module checkbox rows have a minimum height of 44px

  Scenario: Tenant card tap navigates to detail
    Given a tenant card is visible at 375px
    When the user taps anywhere on the card (not the actions button)
    Then the user is navigated to /dashboard/tenants/:id
```

**Technical Notes:**
- Implement `TenantCard` sub-component; apply `hidden md:block` / `block md:hidden` pattern
- Dialog: change `sm:max-w-[500px]` to `w-[calc(100vw-32px)] sm:max-w-[500px]` and add `max-h-[90vh] overflow-y-auto`
- Toolbar: `flex flex-col gap-2 sm:flex-row sm:items-center`

---

#### RESP-07: Tenant Detail Page — Responsive Header and Touch-Safe Copy

**As a** super admin on a mobile device viewing a tenant detail page,
**I want** the header actions and copy-to-clipboard fields to be comfortably usable,
**so that** I can manage a tenant without the UI elements colliding or being too small to tap.

**Story Points:** 5
**MoSCoW:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: Tenant detail page — responsive

  Scenario: Header wraps on mobile
    Given the user is on /dashboard/tenants/:id at 375px
    Then the back button, tenant name/slug, and status badge appear on row one
    And the Suspend/Activate button appears on row two, full-width
    And no element extends beyond the viewport

  Scenario: Copy field touch targets are adequate
    Given any CopyField component is visible at 375px
    Then the copy icon button has a touchable area of at least 44 x 44px
    And the copy button does not overlap the UUID/slug text value

  Scenario: Integration env vars block is scrollable
    Given the Recruitment Integration card is visible at 375px
    Then the env var code block scrolls horizontally within its container
    And the parent card does not cause horizontal overflow of the page

  Scenario: Administrators list is readable on mobile
    Given the Administrators section is visible at 375px
    Then each admin row shows: avatar, name/email (truncated with ellipsis), role badges, status, and action menu
    And the action menu button (3 dots) has a tap target of at least 44 x 44px
    And role badges do not cause the row to overflow horizontally

  Scenario: Info/integration cards stack on mobile
    Given the viewport is 375px
    Then the Tenant Info card and Recruitment Integration card stack vertically
    (The lg:grid-cols-2 already collapses to 1 column below lg — this verifies no regression)
```

**Technical Notes:**
- CopyField: wrap the button in a `<div className="flex items-center justify-center w-11 h-11">` to ensure 44px tap target
- Header: use `flex flex-col gap-3 sm:flex-row sm:items-center` pattern matching RESP-05
- Code block: ensure `overflow-x-auto` is on the pre/code container, not the card

---

#### RESP-08: Audit Log Page — Mobile Stacked Entries

**As an** admin on a mobile device reviewing the audit log,
**I want** each log entry displayed as a stacked card instead of a multi-column table row,
**so that** I can read the time, action, actor, and target without horizontal scrolling.

**Story Points:** 8
**MoSCoW:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: Audit log page — responsive

  Scenario: Table replaced by log entry cards on mobile
    Given the user is on /dashboard/audit at 375px
    Then the data table is not visible
    And each audit log entry is displayed as a card containing:
      | field   | format                          |
      | Time    | "DD/MM/YYYY HH:MM:SS" (localized) |
      | Action  | colored badge (same color coding as desktop) |
      | Actor   | email or ID string (wrapping allowed) |
      | Target  | email or truncated ID            |
      | IP      | monospace, wraps if needed       |
    And the cards are separated by a subtle divider or spacing

  Scenario: Table visible on tablet and desktop
    Given the viewport is 768px or wider
    Then the data table is visible and the card stack is hidden

  Scenario: Filter bar adapts on mobile
    Given the viewport is 375px
    When the filter form is visible
    Then the Action dropdown is full-width
    And the From/To date inputs each occupy 50% width in a 2-column grid (or stack vertically)
    And the Apply button is full-width below the filters
    And the total filter area height does not exceed 200px

  Scenario: Pagination is touch-friendly
    Given there are multiple pages of audit events
    And the viewport is 375px
    Then the pagination row shows: entry range text on the left, prev/next buttons on the right
    And the prev/next buttons have a minimum tap target of 44 x 44px
    And the pagination row does not overflow the viewport

  Scenario: Empty state is centered and readable
    Given no audit events match the filter at 375px
    Then the empty state icon and "No audit events found" text are centered
    And the container fills the full card width
```

**Technical Notes:**
- Implement `AuditLogCard` sub-component
- Use `hidden md:block` on `<Table>` wrapper, `block md:hidden` on cards container
- Filter form: `flex flex-col gap-3 sm:flex-row sm:items-end`
- Date inputs wrapper: `grid grid-cols-2 gap-2 sm:flex sm:items-end`

---

#### RESP-09: Roles Page — Mobile Card Stack

**As an** admin on a mobile device reviewing roles,
**I want** the Roles list to display as a stacked card layout,
**so that** I can view role names, descriptions, and types without a table overflowing the screen.

**Story Points:** 5
**MoSCoW:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: Roles page — responsive

  Scenario: Table replaced by role cards on mobile
    Given the user is on /dashboard/roles at 375px
    Then the data table is not visible
    And each role is shown as a card containing:
      | field       | content                          |
      | Name        | monospace role name              |
      | Description | full description (wrapping)      |
      | Module      | module badge or "—"              |
      | Type        | system / custom badge            |
      | Created     | localized date                   |
      | Delete      | icon button (>= 44x44px) for custom roles only |

  Scenario: Table is shown on tablet and desktop
    Given the viewport is 768px or wider
    Then the standard table is visible

  Scenario: Tab filter pills wrap gracefully on mobile
    Given multiple module tabs are visible at 375px
    Then the tabs use flex-wrap and remain fully visible without horizontal scroll
    And each tab pill has a minimum height of 36px and is tappable

  Scenario: Create Role dialog is mobile-friendly
    Given the viewport is 375px
    When the user taps "Create Role"
    Then the dialog opens at near-full-width
    And all input fields are visible and usable
    And the Cancel and Create buttons are both visible
    And each input field has height of at least 44px

  Scenario: Toolbar count and CTA adapt on mobile
    Given the viewport is 375px
    Then the count text and "Create Role" button are on separate rows
    Or the Create Role button is full-width below the count
```

**Technical Notes:**
- Implement `RoleCard` sub-component
- Delete button: wrap in `<div className="flex items-center justify-center w-11 h-11">` for touch target
- Dialog: change `sm:max-w-[400px]` to `w-[calc(100vw-32px)] sm:max-w-[400px]`
- Toolbar: `flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between`

---

#### RESP-10: My Profile Page — Responsive Name Grid

**As a** regular user accessing My Profile on a mobile device,
**I want** the first name and last name fields to stack vertically on small screens,
**so that** I can comfortably type in each field without the inputs being too narrow.

**Story Points:** 2
**MoSCoW:** Should Have

**Acceptance Criteria:**

```gherkin
Feature: My Profile page — responsive name grid

  Scenario: Name fields stack on mobile
    Given the user is on /me at 375px
    When the Display Name card is visible
    Then the First Name and Last Name fields are stacked vertically (1 column)
    And each input field spans the full card width
    And each input has a minimum height of 40px

  Scenario: Name fields are side-by-side on tablet and desktop
    Given the viewport is 640px or wider
    Then the First Name and Last Name fields appear side by side (2 columns)
    And the layout matches the existing desktop design

  Scenario: Account Info key-value rows are readable on mobile
    Given the viewport is 375px
    Then each row (Email, User ID, Tenant, Joined) displays label and value
    And the User ID value truncates gracefully (max-w-[260px] may need to become max-w-full on mobile)
    And the row does not cause horizontal overflow

  Scenario: Password form is fully usable on mobile
    Given the Security card is visible at 375px
    Then all three password inputs (Current, New, Confirm) are full-width
    And show/hide password toggles have a tap target of at least 44px
    And the "Change Password" button is accessible (full-width or right-aligned with adequate height)
```

**Technical Notes:**
- Change `grid grid-cols-2` to `grid grid-cols-1 sm:grid-cols-2` in the name form
- User ID `max-w-[260px]`: change to `max-w-full truncate` so it respects card width on mobile
- Show/hide password buttons: same fix as RESP-03 — add `min-w-[44px] min-h-[44px] flex items-center justify-center`

---

### Sprint 3 / Backlog Stories — Could Have

---

#### RESP-11: Mobile-Optimized Filter Sheet

**As a** mobile user on the Users or Audit Log page,
**I want** filter controls to open in a slide-up bottom sheet instead of stacking above the list,
**so that** the list content is immediately visible and filters are available on demand.

**Story Points:** 8
**MoSCoW:** Could Have

**Acceptance Criteria:**

```gherkin
Feature: Mobile filter sheet

  Scenario: Filter button triggers a bottom sheet on mobile
    Given the user is on /dashboard/users or /dashboard/audit at 375px
    When the user taps a "Filter" button (or funnel icon)
    Then a bottom sheet slides up containing all filter controls
    And the list content remains partially visible behind the sheet
    And a drag handle or close button is visible

  Scenario: Filters apply and sheet closes
    Given the filter sheet is open
    When the user selects filter values and taps "Apply"
    Then the sheet closes
    And the list refreshes with the applied filters
    And a filter indicator badge shows the number of active filters

  Scenario: Clearing all filters
    Given one or more filters are active
    When the user taps "Clear all" in the filter sheet
    Then all filter values reset to default
    And the filter badge is removed
```

**Technical Notes:** Requires a new `FilterSheet` component using a shadcn-compatible drawer or Sheet primitive. Significant UX redesign — this is why it is Could Have.

---

#### RESP-12: Sticky Mobile Action Bar on Detail Pages

**As a** mobile admin on a user or tenant detail page,
**I want** the primary action buttons to remain accessible at the bottom of the screen as I scroll,
**so that** I can act on a record without scrolling back to the top of the page.

**Story Points:** 5
**MoSCoW:** Could Have

**Acceptance Criteria:**

```gherkin
Feature: Sticky action bar on detail pages at mobile

  Scenario: Action bar is fixed at bottom on mobile
    Given the user is on /dashboard/users/:id or /dashboard/tenants/:id at 375px
    And the page content is longer than the viewport
    When the user scrolls down
    Then a sticky bar at the bottom of the viewport shows the primary action buttons
    And the bar has a background matching the card color with a top border
    And the main content is not obscured by the bar (bottom padding applied to main)

  Scenario: Bar is absent on desktop
    Given the viewport is 768px or wider
    Then no sticky bottom bar is rendered
    And the actions remain in the page header
```

---

## 7. Non-Functional Requirements

### 7.1 Breakpoints

The following Tailwind CSS breakpoint system is in use (default Tailwind v4 breakpoints, confirmed by the absence of a `tailwind.config.ts` override file):

| Name | Minimum Width | Usage |
|------|--------------|-------|
| (default) | 0px | Mobile-first base styles |
| `sm` | 640px | Large phones, small tablets |
| `md` | 768px | Tablets — sidebar switches from drawer to inline |
| `lg` | 1024px | Laptops — multi-column grids fully expanded |
| `xl` | 1280px | Wide desktop |

**Target device profiles for QA:**
- Mobile: 375px (iPhone SE), 390px (iPhone 14), 412px (Samsung Galaxy S23)
- Tablet: 768px (iPad Mini), 1024px (iPad Pro landscape)
- Desktop: 1280px, 1440px

### 7.2 Performance

| Metric | Constraint | Measurement Tool |
|--------|-----------|-----------------|
| Cumulative Layout Shift (CLS) | < 0.1 | Chrome DevTools Lighthouse / WebPageTest |
| Largest Contentful Paint (LCP) | < 2.5s on 4G | Chrome DevTools Lighthouse |
| Total Blocking Time (TBT) | < 200ms | Chrome DevTools Lighthouse |
| No new JavaScript bundles added | Responsive changes are CSS-only or minimal JS state | Bundle analyzer |
| No layout shift when sidebar state loads from localStorage | Sidebar width transition must not cause reflow of sibling elements | Manual visual inspection |

### 7.3 Accessibility

| Requirement | Standard | Specification |
|-------------|---------|--------------|
| Minimum touch target size | WCAG 2.5.5 (Level AAA, recommended) / Apple HIG / Google Material | 44 x 44px for all interactive elements |
| Minimum font size (body) | WCAG 1.4.4 | No text smaller than 12px at default browser zoom |
| Focus indicators | WCAG 2.4.7 | All interactive elements must show visible focus ring on keyboard navigation |
| Color contrast | WCAG 2.1 AA | Text contrast ratio >= 4.5:1; large text >= 3:1 |
| Keyboard navigation unbroken | WCAG 2.1.1 | Tab order must remain logical after responsive changes; drawer must trap focus when open |
| ARIA for drawer | WCAG 4.1.2 | Sidebar drawer must use `role="dialog"` with `aria-label="Navigation"` and `aria-modal="true"` when open |

### 7.4 TigerSoft Branding CI Compliance

All responsive changes must preserve the following CI-mandated design elements:

| Element | Specification | Responsive Constraint |
|---------|--------------|----------------------|
| Tiger Red (`#F4001A`) | CTA buttons, active nav items, brand accents | Must be applied to mobile hamburger button and drawer active states |
| Oxford Blue (`#0B1F3A`) | All body text and headings | Minimum contrast maintained at all breakpoints |
| Serene (`#DBE1E1`) | Borders, dividers | Mobile card dividers must use `border-border` (Serene) |
| UFO Green (`#34D186`) | Success states | Status badge colors unchanged by responsive work |
| Rounded corners | `rounded-[10px]` cards, `rounded-[1000px]` buttons | All mobile cards and buttons must maintain existing border radius |
| Plus Jakarta Sans / FC Vision | Typography | Font stack unchanged; no mobile-specific font substitution |
| White-dominant layout | Minimum 45% white space | Mobile card stacks must not feel dense — adequate vertical spacing (`space-y-3` minimum between cards) |
| No pure black | Use Oxford Blue for text | Verified — current codebase already uses `text-semi-black` throughout |
| Soft edges on interactive elements | Rounded corners for inputs and cards | Mobile inputs retain `rounded-[10px]` |

### 7.5 Browser and OS Support

| Platform | Browser | Minimum version |
|----------|---------|----------------|
| iOS | Safari | 16+ |
| iOS | Chrome | Latest |
| Android | Chrome | Latest |
| Android | Samsung Internet | Latest |
| macOS | Safari, Chrome, Firefox | Latest - 1 |
| Windows | Chrome, Edge | Latest - 1 |

---

## 8. Out of Scope

The following items are explicitly excluded from this release to prevent scope creep:

| Item | Reason |
|------|--------|
| Native iOS / Android application | Web-only product; native app is a future roadmap item |
| Server-side rendering (SSR) changes | All pages are currently client-side; responsive changes are CSS and client-side state only |
| Internationalization (i18n) improvements | Language switching (TH/EN) already exists; mobile layout changes do not alter language logic |
| New dashboard metrics or charts | No chart components exist; adding them is a separate feature initiative |
| Progressive Web App (PWA) capabilities | Offline caching, install prompts, and push notifications are out of scope |
| Redesign of existing desktop UI | This PRD is additive — existing desktop layouts must remain pixel-identical after responsive changes |
| Performance optimization of API calls | Data loading, pagination limits, and API response times are backend concerns outside this UI sprint |
| Dark mode-specific responsive bugs | Dark mode and responsive design are orthogonal; dark mode is already implemented and must not regress |
| Email template responsive design | Invite and password reset email templates are managed by the backend (Resend) team |
| Any changes to the Go/Gin backend or API contracts | This PRD is frontend-only |

---

## 9. Sprint Capacity Plan

**Assumed team velocity:** 34 story points per sprint
**Sprint duration:** 2 weeks

### Sprint 1 — Foundation and Critical Paths (Must Have)

| Story ID | Title | Points | Dependency |
|----------|-------|--------|------------|
| RESP-02 | Responsive Dashboard Layout Shell | 3 | None — do first |
| RESP-01 | Mobile Sidebar Drawer | 8 | RESP-02 |
| RESP-03 | Auth Pages Mobile Polish | 3 | None (separate route group) |
| RESP-05 | User Detail Page — Responsive Action Header | 5 | RESP-02 |
| RESP-04 | Users Page — Mobile Card Stack | 8 | RESP-02 |
| **Buffer** | Regression testing, PR review, a11y validation | 5 | — |
| **Sprint 1 Total** | | **32** | |

**Sprint 1 Goal:** Any user can open and use the auth flows on a mobile phone, and any admin can navigate and perform basic user management on a mobile device.

### Sprint 2 — Complete Dashboard Coverage (Should Have)

| Story ID | Title | Points | Dependency |
|----------|-------|--------|------------|
| RESP-06 | Tenants Page — Mobile Card Stack | 8 | RESP-01, RESP-02 |
| RESP-07 | Tenant Detail Page — Responsive Header and Copy | 5 | RESP-01, RESP-02 |
| RESP-09 | Roles Page — Mobile Card Stack | 5 | RESP-01, RESP-02 |
| RESP-10 | My Profile Page — Responsive Name Grid | 2 | RESP-02 |
| RESP-08 | Audit Log Page — Mobile Stacked Entries | 8 | RESP-01, RESP-02 |
| **Buffer** | Cross-browser QA, a11y audit, CLS measurement | 5 | — |
| **Sprint 2 Total** | | **33** | |

**Sprint 2 Goal:** Every page in TGX Auth Console is fully usable on a 375px mobile viewport with no horizontal overflow and all touch targets >= 44px.

### Sprint 3 — Enhanced Mobile UX (Could Have)

| Story ID | Title | Points | Dependency |
|----------|-------|--------|------------|
| RESP-11 | Mobile-Optimized Filter Sheet | 8 | Sprint 2 complete |
| RESP-12 | Sticky Mobile Action Bar on Detail Pages | 5 | Sprint 2 complete |
| — | End-to-end Playwright mobile viewport test suite | 8 | Sprint 2 complete |
| — | Final lighthouse audit and CLS report | 3 | All stories done |
| **Sprint 3 Total** | | **24** | |

**Sprint 3 Goal:** Enhanced mobile UX patterns in place; automated test coverage ensures no regressions.

---

## 10. Definition of Done

A user story is considered Done when all of the following are true:

### Per Story
- [ ] All Gherkin acceptance criteria have been manually verified on Chrome DevTools device simulation (375px, 768px, 1280px)
- [ ] No horizontal scrollbar appears at 375px viewport width
- [ ] All interactive elements have a minimum touch target of 44 x 44px (verified by DevTools element inspector)
- [ ] CLS measured in Lighthouse is < 0.1 for the affected page
- [ ] Dark mode has been verified — no regression in dark theme at any breakpoint
- [ ] Code has been peer reviewed by one other developer
- [ ] TigerSoft branding colors and typography remain unchanged (verified visually)

### Per Sprint
- [ ] All stories in the sprint have passed per-story DoD
- [ ] Cross-browser testing completed on: Chrome, Safari (iOS), Samsung Internet
- [ ] No new TypeScript errors introduced (`tsc --noEmit` passes)
- [ ] No new console errors at any breakpoint
- [ ] Accessibility audit run with axe-core or Lighthouse — zero critical violations
- [ ] Product Owner sign-off on visual review recording or live demo

### Release Readiness (after Sprint 2)
- [ ] All Must Have and Should Have stories are Done
- [ ] Lighthouse Mobile score >= 90 on /login, /dashboard, /dashboard/users
- [ ] CLS < 0.1 on all tested pages
- [ ] Zero horizontal overflow on any page at 375px (automated with Playwright `page.evaluate(() => document.documentElement.scrollWidth > document.documentElement.clientWidth)`)
- [ ] Visual regression screenshots archived for future comparison

---

*Document prepared by: TigerSoft Product Owner Agent*
*Reviewed against codebase at commit: `9483143` (branch: `fix/module-config-tenant-provisioning`)*
*Branding compliance verified against: `guide/BRANDING.md`*
