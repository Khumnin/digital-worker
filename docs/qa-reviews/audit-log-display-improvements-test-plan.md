# Test Plan — Audit Log Display Improvements
**Feature:** Audit Log Display Improvements (actor/target email enrichment + IP normalization)
**Date:** 2026-03-05
**QA Engineer:** Senior QA / ISTQB
**Sprint:** Post-Sprint-1 improvement
**Files Changed:** 5 (3 backend, 2 frontend)
**Risk Level:** Medium — additive change with LEFT JOIN on hot read path

---

## 1. Scope

### 1.1 In Scope

| Layer | Component | Change |
|-------|-----------|--------|
| Domain | `internal/domain/repository.go` — `AuditEvent` struct | Added `ActorEmail *string`, `TargetEmail *string` |
| Repository | `internal/repository/postgres/audit_repo.go` — `List()` | LEFT JOIN users twice (actor + target) |
| Handler | `internal/handler/audit_handler.go` | `normalizeIP()` function; conditional inclusion of `actor_email`/`target_email` in JSON response |
| API Contract | `auth-admin-ui/src/lib/api.ts` — `AuditLog` interface | Optional `actor_email?` and `target_email?` fields |
| UI | `auth-admin-ui/src/app/(dashboard)/dashboard/audit/page.tsx` | Actor column: email or UUID fallback; Target column: email or truncated UUID fallback; Time locale changed to `en-US` |

### 1.2 Out of Scope
- Audit log filtering logic (covered in previous QA review `audit-log-test-cases.md`)
- Authentication and JWT handling
- GDPR erasure / `AnonymizeActor` path
- `ListForArchive` path (not changed; no email JOIN)

---

## 2. Risk Assessment

| Risk | Probability | Impact | Priority |
|------|------------|--------|----------|
| LEFT JOIN degrades query performance on large audit_log tables | Medium | High | P1 |
| Deleted user (row removed) causes NULL email — UI shows raw UUID or dash | High | Medium | P1 |
| GDPR-erased user (actor_id replaced with tombstone UUID) JOIN returns NULL — no personal data leak | Low | Critical | P1 |
| IPv6-mapped IPv4 addresses not starting with `::ffff:` are incorrectly stripped | Low | Medium | P2 |
| Pure IPv6 addresses (e.g., `2001:db8::1`) incorrectly stripped | Low | Medium | P2 |
| System-generated events with `actor_id = NULL` cause JOIN to produce NULL email — frontend shows dash | High | Low | P2 |
| `actor_email` / `target_email` absent from JSON for NULL cases — TypeScript optional fields handle this correctly | Low | Low | P3 |
| Time locale change (`en-US`) breaks existing snapshot or integration tests | Medium | Low | P3 |
| `en-US` date format differs from `th-TH/gregory` format used in earlier fix — inconsistency between audit log and other date displays | Medium | Low | P3 |

---

## 3. Test Strategy (ISTQB)

**Approach:** Risk-based; prioritise P1 scenarios first.
**Test levels:** Manual exploration + automated (Playwright E2E, Go httptest unit/integration).
**Design techniques used:**
- Equivalence Partitioning (IP address classes, user existence states)
- Boundary Value Analysis (NULL vs. non-NULL email, empty string vs. absent JSON key)
- State Transition Testing (user active → deleted → tombstoned)
- Decision Table Testing (actor present/absent × target present/absent × user deleted/active)
- Experience-based / exploratory (IP edge cases, pagination + email enrichment)

---

## 4. Test Cases

### 4.1 Backend — API Contract (GET /api/v1/admin/audit-log)

| ID | Title | ISTQB Technique | Priority | Type |
|----|-------|----------------|----------|------|
| TC-ALD-001 | Happy path: response includes `actor_email` and `target_email` for active users | EP | P1 | Integration |
| TC-ALD-002 | Deleted actor: `actor_email` key absent from JSON item (LEFT JOIN returns NULL, handler skips nil) | State Transition | P1 | Integration |
| TC-ALD-003 | Deleted target: `target_email` key absent from JSON item | State Transition | P1 | Integration |
| TC-ALD-004 | System event with no actor: `actor_id` absent, `actor_email` absent | EP | P1 | Integration |
| TC-ALD-005 | Event with actor but no target: `target_id` absent, `target_email` absent | EP | P2 | Integration |
| TC-ALD-006 | GDPR tombstoned actor: actor_id is tombstone UUID, no users row → `actor_email` absent | State Transition | P1 | Integration |
| TC-ALD-007 | IPv4-mapped IPv6 address normalised: `::ffff:192.168.1.1` → `192.168.1.1` | EP | P1 | Integration |
| TC-ALD-008 | Pure IPv4 address unchanged: `10.0.0.1` → `10.0.0.1` | EP | P1 | Integration |
| TC-ALD-009 | Pure IPv6 address unchanged: `2001:db8::1` → `2001:db8::1` (no strip) | BVA | P1 | Integration |
| TC-ALD-010 | IPv6 loopback `::1` unchanged: not stripped | BVA | P2 | Integration |
| TC-ALD-011 | Empty `actor_ip` field: `ip_address` returns empty string, not error | BVA | P2 | Integration |
| TC-ALD-012 | `::ffff:` prefix with uppercase letters not stripped (case-sensitive check) | BVA | P2 | Unit |
| TC-ALD-013 | Response pagination unaffected: email enrichment does not change `total`, `page`, `total_pages` | EP | P2 | Integration |
| TC-ALD-014 | Filtering by `actor_id` still works with email enrichment present | Regression | P1 | Integration |
| TC-ALD-015 | Filtering by `event_type` / `action` still works with email enrichment present | Regression | P1 | Integration |
| TC-ALD-016 | Empty audit log returns empty `data` array with no error | EP | P2 | Integration |
| TC-ALD-017 | Unauthenticated request returns 401 | Regression | P1 | Integration |
| TC-ALD-018 | Non-admin token returns 403 | Regression | P1 | Integration |

---

#### TC-ALD-001 — Happy Path: email fields present for active users

**Precondition:** Two active users exist. One audit event with both `actor_id` and `target_user_id` pointing to those users.

**Steps:**
1. Authenticate as super_admin, obtain JWT.
2. `GET /api/v1/admin/audit-log?page=1&page_size=10`

**Expected JSON item:**
```json
{
  "id": "<uuid>",
  "action": "ROLE_ASSIGNED",
  "actor_id": "<actor-uuid>",
  "actor_email": "actor@example.com",
  "target_id": "<target-uuid>",
  "target_email": "target@example.com",
  "ip_address": "192.168.1.1",
  "created_at": "<timestamp>"
}
```

**Assertions:**
- HTTP 200
- `actor_email` present and equals actor user's email
- `target_email` present and equals target user's email
- `actor_id` present
- `target_id` present

---

#### TC-ALD-002 — Deleted Actor: `actor_email` absent from JSON

**Precondition:** Create audit event with actor_id = X. Then soft-delete or hard-delete user X so the users table row is gone or email column is NULL.

**Steps:**
1. `GET /api/v1/admin/audit-log`

**Expected JSON item:**
```json
{
  "id": "<uuid>",
  "action": "LOGIN_SUCCESS",
  "actor_id": "<deleted-user-uuid>",
  "ip_address": "...",
  "created_at": "..."
}
```

**Assertions:**
- `actor_email` key must NOT be present in the JSON object
- `actor_id` remains present (UUID is preserved in audit_log row)
- No 500 error

**Rationale:** Handler conditionally includes `actor_email` only when `e.ActorEmail != nil`. If the LEFT JOIN returns NULL, the pointer stays nil and the key is omitted. This is the correct, safe behaviour. If the key were present with a null value, the frontend would render null in the Actor column.

---

#### TC-ALD-004 — System Event: Both actor fields absent

**Precondition:** Create audit event where `actor_id` is NULL (e.g., a scheduled system task or background job that logs without a user actor).

**Steps:**
1. `GET /api/v1/admin/audit-log`

**Expected JSON item:**
```json
{
  "id": "<uuid>",
  "action": "EMAIL_VERIFICATION_SENT",
  "ip_address": "",
  "metadata": {},
  "created_at": "..."
}
```

**Assertions:**
- `actor_id` key absent
- `actor_email` key absent
- No 500 error

---

#### TC-ALD-007 — normalizeIP: IPv4-mapped IPv6

**Steps:**
1. Seed audit event with `actor_ip = "::ffff:192.168.1.100"`
2. `GET /api/v1/admin/audit-log`

**Expected:**
```json
{ "ip_address": "192.168.1.100" }
```

**Assertions:** `ip_address` equals `"192.168.1.100"` — prefix stripped.

---

#### TC-ALD-009 — normalizeIP: Pure IPv6 must NOT be stripped

**Steps:**
1. Seed audit event with `actor_ip = "2001:db8::cafe"`
2. `GET /api/v1/admin/audit-log`

**Expected:**
```json
{ "ip_address": "2001:db8::cafe" }
```

**Assertions:** `ip_address` equals `"2001:db8::cafe"` — unchanged.

**Risk:** If `normalizeIP` matched on a partial string or the wrong prefix, real IPv6 addresses would be corrupted.

---

#### TC-ALD-012 — normalizeIP: Case Sensitivity

**Unit test logic (Go):**
```
normalizeIP("::FFFF:10.0.0.1")  → must return "::FFFF:10.0.0.1" (NOT stripped)
normalizeIP("::ffff:10.0.0.1")  → must return "10.0.0.1" (stripped)
```

**Rationale:** `strings.HasPrefix` is case-sensitive in Go. The actual prefix in the wild is always lowercase `::ffff:` as stored by the OS network stack. Uppercase variants should not be stripped (defensive correctness).

---

### 4.2 Backend — normalizeIP Unit Logic

| ID | Input | Expected Output | Assertion |
|----|-------|----------------|-----------|
| TC-IP-001 | `"::ffff:192.168.1.1"` | `"192.168.1.1"` | Strip prefix |
| TC-IP-002 | `"::ffff:10.0.0.1"` | `"10.0.0.1"` | Strip prefix |
| TC-IP-003 | `"::ffff:0.0.0.0"` | `"0.0.0.0"` | Strip prefix |
| TC-IP-004 | `"192.168.1.1"` | `"192.168.1.1"` | Unchanged |
| TC-IP-005 | `"2001:db8::1"` | `"2001:db8::1"` | Unchanged |
| TC-IP-006 | `"::1"` | `"::1"` | Unchanged (loopback) |
| TC-IP-007 | `""` | `""` | Unchanged (empty) |
| TC-IP-008 | `"::FFFF:1.2.3.4"` | `"::FFFF:1.2.3.4"` | Unchanged (uppercase) |
| TC-IP-009 | `"::ffff:"` | `""` | Strip prefix, empty remainder |

---

### 4.3 Repository — LEFT JOIN Correctness

| ID | Title | ISTQB Technique | Priority | Type |
|----|-------|----------------|----------|------|
| TC-REPO-001 | COUNT query unaffected by JOIN (no row multiplication) | EP | P1 | Integration |
| TC-REPO-002 | One actor maps to many events: each event has correct email | EP | P1 | Integration |
| TC-REPO-003 | Actor and target are the same user: both email fields equal | Decision Table | P2 | Integration |
| TC-REPO-004 | Ordering by `occurred_at DESC` preserved after JOIN | Regression | P1 | Integration |
| TC-REPO-005 | `archived = false` filter still works: archived events excluded | Regression | P1 | Integration |
| TC-REPO-006 | `ListForArchive` path still works without email fields in scan | Regression | P1 | Integration |
| TC-REPO-007 | Pagination LIMIT/OFFSET correct when JOINed rows do not multiply | BVA | P1 | Integration |

---

#### TC-REPO-001 — COUNT query excludes JOIN

**Risk identified from code review:** The COUNT query (`countSQL`) queries `FROM audit_log a %s` without the JOIN. The data query adds the JOIN. These are separate SQL statements so there is no fan-out risk — but this must be verified explicitly.

**Steps:**
1. Insert 5 audit events; 2 actors exist, 3 actors do not.
2. `GET /api/v1/admin/audit-log`

**Assertions:**
- `total` = 5 (not 2 or 10)
- `data` array length = 5 (up to page_size)

---

#### TC-REPO-006 — `ListForArchive` regression

**Rationale:** `ListForArchive` in `audit_repo.go` (lines 169–206) does NOT have the email JOIN — it uses a 9-column scan without `actor_email`/`target_email`. This path must not be affected by the structural change to the `AuditEvent` struct.

**Steps:**
1. Insert an audit event older than the archive cutoff.
2. Call `ListForArchive` (or trigger the archive job).

**Assertions:**
- No scan error (column count mismatch would panic/error)
- Event returned with correct fields
- `ActorEmail` and `TargetEmail` remain nil on returned struct

---

### 4.4 Frontend — UI Display

| ID | Title | ISTQB Technique | Priority | Type |
|----|-------|----------------|----------|------|
| TC-UI-001 | Actor column displays `actor_email` when present | EP | P1 | E2E |
| TC-UI-002 | Actor column falls back to `actor_id` UUID when `actor_email` absent | EP | P1 | E2E |
| TC-UI-003 | Actor column displays `—` when both `actor_email` and `actor_id` absent | BVA | P2 | E2E |
| TC-UI-004 | Target column displays `target_email` when present | EP | P1 | E2E |
| TC-UI-005 | Target column displays truncated UUID (`xxxxxxxx…`) when `target_email` absent but `target_id` present | BVA | P1 | E2E |
| TC-UI-006 | Target column displays `—` when both `target_email` and `target_id` absent | BVA | P2 | E2E |
| TC-UI-007 | Time column displays in mm/dd/yyyy format (en-US locale) | EP | P1 | E2E |
| TC-UI-008 | Time column displays in Asia/Bangkok timezone (UTC+7 offset) | EP | P1 | E2E |
| TC-UI-009 | Long email addresses truncated with CSS `truncate` class — not overflowing table cell | EP | P2 | Visual |
| TC-UI-010 | Empty audit log shows "No audit events found" empty state | EP | P2 | E2E |
| TC-UI-011 | Pagination controls present and functional with email enrichment | Regression | P1 | E2E |
| TC-UI-012 | Action filter still works with email enrichment present | Regression | P1 | E2E |

---

#### TC-UI-001 — Actor column shows email

**Precondition:** API mocked to return:
```json
{
  "actor_id": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
  "actor_email": "alice@tigersoft.co.th"
}
```

**Steps:**
1. Navigate to `/dashboard/audit`
2. Locate the Actor cell for this row

**Expected:** Cell text = `alice@tigersoft.co.th`

**Assertions:**
- Actor cell does NOT show the UUID
- Actor cell shows the email address

---

#### TC-UI-002 — Actor column falls back to UUID

**Precondition:** API mocked to return item WITHOUT `actor_email` key; `actor_id` is present.

**Steps:**
1. Navigate to `/dashboard/audit`

**Expected:** Cell text = UUID string (e.g., `aaaaaaaa-aaaa-...`)

**Frontend logic under test:**
```typescript
{log.actor_email || log.actor_id || "—"}
```

**Assertions:** Cell shows raw UUID, not `—` and not a blank cell.

---

#### TC-UI-005 — Target column truncated UUID

**Precondition:** API returns item with `target_id` but no `target_email`.

**Frontend logic under test:**
```typescript
{log.target_email || (log.target_id ? `${log.target_id.slice(0, 8)}…` : "—")}
```

**Expected:** Cell shows first 8 characters of UUID followed by `…`

**Assertions:**
- Cell text matches pattern `/^[a-f0-9]{8}…$/`
- Cell does NOT show the full UUID

---

#### TC-UI-007 — Time column en-US format

**Precondition:** API returns `created_at = "2026-03-05T10:00:00Z"` (UTC).

**Expected display (Asia/Bangkok = UTC+7):** `03/05/2026, 05:00:00 PM`

The formatter in use:
```typescript
new Intl.DateTimeFormat("en-US", {
  year: "numeric",
  month: "2-digit",
  day: "2-digit",
  hour: "2-digit",
  minute: "2-digit",
  second: "2-digit",
  timeZone: "Asia/Bangkok",
}).format(new Date(log.created_at))
```

**Assertions:**
- Month comes before day (mm/dd/yyyy order)
- Year is 4-digit Gregorian (2026, not 2569)
- Time is in Asia/Bangkok timezone

**Note:** Previous locale was `th-TH` with `calendar: "gregory"` (per prior review). This change switches to `en-US`. Both produce Gregorian years but differ in separator and month/day order. Verify the actual browser output format on Node 20 / Chromium.

---

#### TC-UI-009 — Long email truncation (Visual/Branding)

**Precondition:** Actor email = `very-long-email-address-that-is-definitely-too-long@somedomain.co.th` (60+ characters).

**Steps:**
1. Navigate to `/dashboard/audit`
2. Inspect Actor cell

**Assertions:**
- Cell text is truncated with ellipsis
- Cell does not overflow the table column boundary
- Surrounding cells are not pushed out of alignment
- Branding: Text color is Oxford Blue `#0B1F3A`, NOT pure black `#000000`

---

### 4.5 Frontend — Branding Compliance (TigerSoft CI)

Mandatory checks per `guide/BRANDING.md` for all frontend-tagged changes.

| ID | Check | Expected | Block/Warn |
|----|-------|----------|-----------|
| TC-BRAND-001 | Actor/Target cell text color | Oxford Blue `#0B1F3A` — not pure black `#000` | BLOCK |
| TC-BRAND-002 | Time cell text color uses `text-semi-grey` token (maps to Quick Silver `#A3A3A3`) | Confirmed via computed style | WARN |
| TC-BRAND-003 | No new hardcoded color values introduced in changed lines | Audit `page.tsx` diff contains no `#000`, no off-brand hex | BLOCK |
| TC-BRAND-004 | Action badge colors are brand-consistent | `text-tiger-red` (Vivid Red `#F4001A`) used for TENANT_SUSPENDED only | WARN |
| TC-BRAND-005 | Table card container uses `rounded-[10px]` soft edge | Confirmed in existing markup — regression check | BLOCK |

**Note:** The `actionColors` map in `audit/page.tsx` (lines 84–101) contains `text-green-600`, `text-blue-600`, `text-purple-600`, `text-teal-600`, `text-orange-600`, `text-indigo-600` for status badge variants. These are Tailwind utility classes that do not map to TigerSoft brand palette colors. This is a pre-existing condition, not introduced by this change. Flag as a separate design debt item — do not block this test plan.

---

## 5. Decision Table — Actor/Target Email Display Logic

| actor_email present | actor_id present | Displayed in Actor column |
|--------------------|-----------------|--------------------------|
| Yes | Yes | `actor_email` |
| No | Yes | `actor_id` (full UUID) |
| Yes | No | `actor_email` |
| No | No | `—` |

| target_email present | target_id present | Displayed in Target column |
|---------------------|-----------------|--------------------------|
| Yes | Yes | `target_email` |
| No | Yes | First 8 chars of UUID + `…` |
| Yes | No | `target_email` |
| No | No | `—` |

These 8 combinations map directly to TC-UI-001 through TC-UI-006.

---

## 6. Edge Cases — Explicit Coverage

### 6.1 Deleted Users (LEFT JOIN returns NULL)
- **Scenario A — Actor deleted after event recorded:** `audit_log.actor_id` references a UUID that no longer exists in `users`. LEFT JOIN yields NULL for `actor_u.email`. Handler omits `actor_email` from JSON. Frontend falls back to raw UUID in Actor column.
- **Scenario B — Target deleted after event recorded:** Same as A for target.
- **Scenario C — Both actor and target deleted:** Both email keys absent; actor shows UUID, target shows truncated UUID.
- **Risk:** If hard-delete removes the users row entirely vs. soft-delete setting `deleted_at`, the JOIN behaviour is identical (NULL in both cases). Verify the `users` table schema permits the LEFT JOIN to return NULL rather than cascading the deletion to audit_log.

### 6.2 System Events with No Actor
Events such as `EMAIL_VERIFICATION_SENT` may be logged with `actor_id = NULL`. The LEFT JOIN ON `actor_u.id = a.actor_id` will correctly produce NULL for `actor_email` when `actor_id` is NULL. `actor_id` itself will be NULL — the handler skips both `actor_id` and `actor_email` JSON keys. The frontend renders `—` in the Actor column.

### 6.3 Pure IPv6 Addresses Must NOT Be Stripped
`normalizeIP` uses `strings.HasPrefix(ip, "::ffff:")`. A pure IPv6 address such as `2001:db8::cafe` does not start with that prefix. It must pass through unchanged. Verify with TC-ALD-009 and TC-IP-005.

### 6.4 Empty Audit Log
When the tenant has no audit events, `List()` returns an empty slice. The handler produces `"data": []`. The frontend checks `logs.length === 0` and renders the empty state (ScrollText icon + "No audit events found"). Email fields are not involved. Verify with TC-UI-010 and TC-ALD-016.

### 6.5 `::ffff:` with Empty Remainder
Input `"::ffff:"` (prefix only, no IP after it). `normalizeIP` returns `""`. The JSON field `ip_address` would be an empty string. The frontend renders `—` for the IP column (`{log.ip_address || "—"}`). This is unusual but should not error.

### 6.6 GDPR Tombstone Actor
After `AnonymizeActor(userID, tombstoneID)`, the `audit_log.actor_id` is rewritten to the tombstone UUID. There is no `users` row for a tombstone UUID, so the LEFT JOIN returns NULL for `actor_email`. No personal data is surfaced. Covered by TC-ALD-006.

---

## 7. Regression Risks

| Risk | Affected Path | Severity | Test Coverage |
|------|--------------|----------|---------------|
| `ListForArchive` scan column count mismatch | `audit_repo.go` lines 169–206 (9-column scan, no email JOIN) | Critical | TC-REPO-006 |
| COUNT query returns inflated total due to JOIN (if ever changed to include JOIN) | `audit_repo.go` countSQL | High | TC-REPO-001 |
| `Append` writes to audit_log without email fields — struct nil fields must not cause INSERT errors | `audit_repo.go` lines 27–49 | High | Existing behavior |
| `AnonymizeActor` still rewrites actor_id correctly — struct change does not break UPDATE | `audit_repo.go` lines 153–167 | High | TC-ALD-006 |
| `action` filter parameter (`event_type` legacy alias) still functions | `audit_handler.go` lines 59–64 | Medium | TC-ALD-015 |
| Pagination `total_pages` calculation correct when `total = 0` | `audit_handler.go` lines 140–143 | Low | TC-ALD-016 |
| `actor_id` / `target_id` still conditionally omitted when nil (existing logic) | `audit_handler.go` lines 123–135 | Medium | TC-ALD-004, TC-ALD-005 |
| Frontend `AuditLog` interface change is additive (optional fields) — no type error in existing consumers | `api.ts` line 105–115 | Low | Build-time TypeScript check |
| Time locale change (`en-US` replacing `th-TH/gregory`) — verify no Buddhist Era regression | `audit/page.tsx` line 197 | Medium | TC-UI-007, TC-UI-008 |

---

## 8. Quality Gates

### Entry Criteria (before this test plan executes)
- All 5 changed files merged to branch and deployed to test environment
- Existing unit tests (`admin_service_test.go`) still pass in CI
- Test database seeded with representative data: active users, deleted users, events with and without actors/targets
- Authentication (super_admin JWT) available for API calls

### Exit Criteria (before the feature ships)
- All P1 test cases executed with PASS result
- All P2 test cases executed with PASS or documented waiver
- Zero Critical or High severity defects open
- TC-REPO-006 (`ListForArchive` regression) explicitly verified
- TC-ALD-009 (pure IPv6 not stripped) explicitly verified
- Branding checks TC-BRAND-001 and TC-BRAND-003 pass
- TypeScript build clean (no type errors from `AuditLog` interface change)

---

## 9. Test Data Requirements

| Dataset | Purpose |
|---------|---------|
| User A (active, has email) | Actor email resolution (happy path) |
| User B (active, has email) | Target email resolution (happy path) |
| User C (soft-deleted or removed from users table) | Deleted actor / deleted target (NULL JOIN) |
| Tombstone UUID (no users row) | GDPR-erased actor (TC-ALD-006) |
| Audit event: actor=A, target=B | Happy path |
| Audit event: actor=A, target=C | Target deleted edge case |
| Audit event: actor=C, target=B | Actor deleted edge case |
| Audit event: actor=NULL, target=NULL | System event (no actor) |
| Audit event: actor_ip=`::ffff:192.168.1.1` | IPv4-mapped IPv6 normalisation |
| Audit event: actor_ip=`2001:db8::cafe` | Pure IPv6 no-strip |
| Audit event: actor_ip=`10.0.0.1` | Plain IPv4 unchanged |
| 26+ events (for pagination test) | TC-REPO-001, TC-UI-011 |
| 0 events | Empty state test TC-UI-010, TC-ALD-016 |

---

## 10. ISO 25010 Quality Characteristics Assessment

| Characteristic | Scope for this change | Verdict |
|---------------|----------------------|---------|
| Functional Suitability | Email fields returned correctly; IP normalised; UI displays email/UUID/dash correctly | Verify via TC-ALD-001 to TC-UI-012 |
| Performance Efficiency | Two LEFT JOINs added to the read query on the per-tenant `audit_log` table. Risk of full-table scan if `users.id` index is missing. | Verify query plan; ensure users(id) PK index is used for JOIN |
| Reliability | NULL handling for missing users — no panic, no 500 error | TC-ALD-002, TC-ALD-003, TC-ALD-004 |
| Security | `actor_email` / `target_email` returned only to authenticated admins. Deleted user emails not leaked — key absent from JSON. | TC-ALD-017, TC-ALD-018, TC-ALD-006 |
| Maintainability | `ActorEmail`/`TargetEmail` on struct clearly documented as "populated by List query only". `normalizeIP` is a pure function — easy to unit test. | Code comment at repository.go line 161 |
| Usability | Actor/Target columns now human-readable (email over UUID). Fallback hierarchy is clear. | TC-UI-001 to TC-UI-006 |
| Compatibility | `en-US` locale supported by all modern browsers. Optional JSON keys backward-compatible with older API consumers. | TC-UI-007 |

---

## 11. Definition of Done Checklist

- [ ] TC-ALD-001: Happy path email fields present — PASS
- [ ] TC-ALD-002: Deleted actor — `actor_email` absent — PASS
- [ ] TC-ALD-003: Deleted target — `target_email` absent — PASS
- [ ] TC-ALD-004: System event (null actor) — no actor keys — PASS
- [ ] TC-ALD-006: GDPR tombstone — no personal data in response — PASS
- [ ] TC-ALD-007: `::ffff:` prefix stripped correctly — PASS
- [ ] TC-ALD-009: Pure IPv6 unchanged — PASS
- [ ] TC-REPO-001: COUNT not inflated by JOIN — PASS
- [ ] TC-REPO-006: `ListForArchive` scan unaffected — PASS
- [ ] TC-UI-001: Actor email displayed — PASS
- [ ] TC-UI-002: Actor UUID fallback — PASS
- [ ] TC-UI-004: Target email displayed — PASS
- [ ] TC-UI-005: Target truncated UUID fallback — PASS
- [ ] TC-UI-007: Time format is mm/dd/yyyy en-US — PASS
- [ ] TC-UI-008: Timezone is Asia/Bangkok — PASS
- [ ] TC-ALD-017: Unauthenticated → 401 — PASS
- [ ] TC-ALD-018: Non-admin → 403 — PASS
- [ ] TC-BRAND-001: Actor/Target cell text is Oxford Blue — PASS
- [ ] TC-BRAND-003: No off-brand hex colors introduced — PASS
- [ ] TypeScript build clean (no errors from optional field additions) — PASS
- [ ] No open Critical or High defects

---

## 12. Open Questions / Observations

1. **`en-US` vs `th-TH/gregory` locale change** — The previous QA review (2026-03-05) approved `th-TH` with `calendar: "gregory"`. This PR changes the locale to `en-US` without `calendar` option. Both produce Gregorian years. The month/day order and separators differ. Confirm with the product owner whether `en-US` (03/05/2026) is the intended display format for this Thai-market product or whether `th-TH` with explicit Gregorian calendar was preferred for number formatting.

2. **`action` filter dropdown includes values not in `EventType` constants** — `audit/page.tsx` ACTION_OPTIONS includes `USER_LOGIN`, `USER_LOGOUT`, `USER_ENABLED`, `USER_DISABLED`, `TENANT_SUSPENDED`, `TENANT_ACTIVATED`. These do not match exactly with the EventType constants in `audit_event.go` (which are `LOGIN_SUCCESS`, `LOGOUT`, `USER_ENABLED`, `USER_DISABLED`). The filter will return zero results for mismatched values. This is a pre-existing mismatch, not introduced by this PR, but should be logged as a separate defect.

3. **Long email addresses in Actor column (max-w-[180px] truncate)** — The cell has `max-w-[180px] truncate`. Emails longer than approximately 22 characters at the cell font size will be truncated. This is acceptable but should be confirmed as intentional UX.

4. **Performance of double LEFT JOIN** — For tenants with large audit logs (100k+ rows), the two LEFT JOINs add per-row lookups. If `users(id)` has a primary key index (which it should), the lookup is O(log n) per row. Recommend verifying with `EXPLAIN ANALYZE` against a representative dataset before production deployment.
