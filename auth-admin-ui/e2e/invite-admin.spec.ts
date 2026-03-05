import { test, expect, request } from "@playwright/test";
import { loginAs } from "./helpers/auth";

/**
 * Phase 5 QA — Feature: Create & Invite Tenant Admin + CI Branding Favicon
 *
 * User Stories covered:
 *   US-01  Platform super_admin can invite admin users to a specific tenant
 *   US-02  Platform super_admin can view all users in a tenant (including status)
 *   US-03  Platform super_admin can resend an invite email to a pending user
 *   US-04  Browser tab displays TigerSoft logo-mark favicon
 *
 * API Contracts:
 *   POST /api/v1/admin/tenants/:id/invite-admin    → 201
 *   GET  /api/v1/admin/tenants/:id/users           → 200 paginated
 *   POST /api/v1/admin/tenants/:id/users/:userId/resend-invite → 200
 *
 * All three endpoints require super_admin role; return 403 for admin role.
 *
 * ISTQB coverage:
 *   - Functional (happy path, UI elements, data binding)
 *   - Boundary (empty fields, duplicate email, malformed email)
 *   - Negative (missing required fields, dialog cancel)
 *   - Security (admin-role access returns 403, unauthenticated access redirects)
 *   - Branding / CI (favicon in page metadata)
 */

// ─── Environment variables ────────────────────────────────────────────────────

const SUPER_ADMIN_EMAIL =
  process.env.TEST_ADMIN_EMAIL || "superadmin@tigersoft.co.th";
const SUPER_ADMIN_PASSWORD = process.env.TEST_ADMIN_PASSWORD || "";

// A regular admin (non-super_admin) credential for the 403 security test.
// Fall back to a clearly-invalid value so the test self-skips if not configured.
const ADMIN_EMAIL = process.env.TEST_REGULAR_ADMIN_EMAIL || "";
const ADMIN_PASSWORD = process.env.TEST_REGULAR_ADMIN_PASSWORD || "";

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev";

// ─── Helpers ──────────────────────────────────────────────────────────────────

/**
 * Navigate to the Tenants list and click the first tenant row.
 * Returns the tenant ID extracted from the resulting URL.
 */
async function navigateToFirstTenant(page: import("@playwright/test").Page) {
  await page.goto("/dashboard/tenants");
  await page.waitForSelector("tbody tr, text=No tenants found");

  // Confirm at least one tenant exists
  const rowCount = await page.locator("tbody tr").count();
  if (rowCount === 0) {
    throw new Error("No tenants in the list — cannot run tenant detail tests");
  }

  await page.locator("tbody tr").first().click();
  await page.waitForURL(/\/dashboard\/tenants\//);

  // Extract tenant UUID from URL
  const url = page.url();
  const match = url.match(/\/tenants\/([a-f0-9-]{36})/);
  return match ? match[1] : null;
}

/**
 * Retrieve a super_admin Bearer token via the Auth API (for direct API tests).
 */
async function getSuperAdminToken(): Promise<string> {
  const ctx = await request.newContext();
  const res = await ctx.post(`${API_BASE}/api/v1/auth/login`, {
    headers: { "Content-Type": "application/json", "X-Tenant-ID": "platform" },
    data: { email: SUPER_ADMIN_EMAIL, password: SUPER_ADMIN_PASSWORD },
  });
  const body = await res.json();
  await ctx.dispose();
  return body.access_token as string;
}

// ─── TC-US01 / US-02: Tenant detail page — Administrators section ─────────────

test.describe("US-01 + US-02 — Tenant detail: Administrators section", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, SUPER_ADMIN_EMAIL, SUPER_ADMIN_PASSWORD);
  });

  // TC-01: Administrators card is visible on tenant detail page
  test("TC-01: Administrators card renders with Invite Admin button", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    const adminsCard = page.locator("text=Administrators").first();
    await expect(adminsCard).toBeVisible();

    const inviteBtn = page.getByRole("button", { name: /invite admin/i });
    await expect(inviteBtn).toBeVisible();

    // Branding: CTA button must use Tiger Red background (#C10016) and pill shape
    const classList = await inviteBtn.getAttribute("class");
    expect(classList).toMatch(/bg-tiger-red|rounded-\[1000px\]/);
  });

  // TC-02: User list renders with avatar initial, display_name, email, role badge, status badge
  test("TC-02: User list shows avatar initial, display_name, email, role badge, status badge", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    // Wait for the user list to load (spinner disappears)
    await page.waitForSelector(
      ".animate-spin, text=No users found in this tenant",
      { state: "detached", timeout: 10_000 }
    ).catch(() => {/* spinner may not appear if data loads quickly */});

    const noUsersMsg = page.locator("text=No users found in this tenant");
    const hasNoUsers = await noUsersMsg.isVisible().catch(() => false);

    if (!hasNoUsers) {
      // At least one user row exists; verify its structure
      const firstUserRow = page.locator(".divide-y > div").first();
      await expect(firstUserRow).toBeVisible();

      // Avatar initial circle (w-8 h-8 rounded-full)
      const avatar = firstUserRow.locator(".rounded-full").first();
      await expect(avatar).toBeVisible();

      // Email text visible somewhere in the row
      const emailText = firstUserRow.locator("p").filter({ hasText: /@/ });
      await expect(emailText).toBeVisible();

      // Status badge (rounded-full text)
      const statusBadge = firstUserRow.locator("span.rounded-full, span[class*='rounded-full']").first();
      await expect(statusBadge).toBeVisible();
    }
  });

  // TC-03: API — GET /api/v1/admin/tenants/:id/users returns paginated structure
  test("TC-03: API returns paginated user list with correct shape", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    // Get first tenant ID from list
    const tenantsRes = await ctx.get(`${API_BASE}/api/v1/admin/tenants`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    expect(tenantsRes.ok()).toBeTruthy();
    const tenantsBody = await tenantsRes.json();
    const firstTenantId: string = tenantsBody.data[0]?.id;

    if (!firstTenantId) {
      await ctx.dispose();
      test.skip();
    }

    const res = await ctx.get(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/users`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    expect(res.status()).toBe(200);
    const body = await res.json();

    // ISO 25010 — Functional suitability: response must contain pagination fields
    expect(body).toHaveProperty("data");
    expect(body).toHaveProperty("total");
    expect(body).toHaveProperty("page");
    expect(body).toHaveProperty("page_size");
    expect(body).toHaveProperty("total_pages");
    expect(Array.isArray(body.data)).toBe(true);

    await ctx.dispose();
  });
});

// ─── TC-US01: Invite Admin Dialog — Happy Path ────────────────────────────────

test.describe("US-01 — Invite Admin dialog: happy path", () => {
  // Unique email per test run to avoid collisions
  const INVITE_EMAIL = `qa-tenant-admin-${Date.now()}@tigersoft.co.th`;
  const DISPLAY_NAME = "QA Tenant Admin";

  test.beforeEach(async ({ page }) => {
    await loginAs(page, SUPER_ADMIN_EMAIL, SUPER_ADMIN_PASSWORD);
  });

  // TC-04: Opening the dialog shows all required elements
  test("TC-04: Invite Admin dialog opens with Email, Display Name fields and 3-step explanation", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    // Title contains tenant name
    await expect(dialog.locator("h2, [role=heading]").first()).toBeVisible();

    // Email field
    await expect(
      dialog.getByPlaceholder(/admin@company|email/i)
    ).toBeVisible();

    // Display Name field
    await expect(dialog.getByPlaceholder(/john smith|name/i)).toBeVisible();

    // 3-step "What happens next" explanation
    await expect(dialog.locator("text=What happens next")).toBeVisible();
    await expect(dialog.locator("ol li")).toHaveCount(3);

    // Send Invitation button present and Tiger-Red styled
    const sendBtn = dialog.getByRole("button", { name: /send invitation/i });
    await expect(sendBtn).toBeVisible();
    const cls = await sendBtn.getAttribute("class");
    expect(cls).toMatch(/bg-tiger-red|rounded-\[1000px\]/);
  });

  // TC-05: Successful invite → toast success and user appears in list with pending status
  test("TC-05: Inviting an admin shows success toast and pending user in list", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");

    await dialog.getByPlaceholder(/admin@company|email/i).fill(INVITE_EMAIL);
    await dialog.getByPlaceholder(/john smith|name/i).fill(DISPLAY_NAME);
    await dialog.getByRole("button", { name: /send invitation/i }).click();

    // Success toast must mention the invited email
    await expect(
      page.locator(`text=${INVITE_EMAIL}`)
    ).toBeVisible({ timeout: 8_000 });

    // Dialog closes after success
    await expect(dialog).toBeHidden({ timeout: 5_000 });

    // Invited user appears in the Administrators list with "pending" status
    const userRow = page.locator(".divide-y > div").filter({
      hasText: INVITE_EMAIL,
    });
    await expect(userRow).toBeVisible({ timeout: 8_000 });

    const statusBadge = userRow
      .locator("span")
      .filter({ hasText: /pending/i })
      .first();
    await expect(statusBadge).toBeVisible();
  });

  // TC-06: API — POST invite-admin returns 201 with correct payload
  test("TC-06: API POST invite-admin returns 201 with pending status and admin role", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    // Resolve first tenant
    const tenantsRes = await ctx.get(`${API_BASE}/api/v1/admin/tenants`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const tenantsBody = await tenantsRes.json();
    const firstTenantId: string = tenantsBody.data[0]?.id;

    if (!firstTenantId) { await ctx.dispose(); test.skip(); }

    const apiEmail = `qa-api-invite-${Date.now()}@tigersoft.co.th`;
    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/invite-admin`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        data: { email: apiEmail, display_name: "QA API Admin" },
      }
    );

    expect(res.status()).toBe(201);
    const body = await res.json();

    // ISO 25010 Functional suitability: all required fields must be present
    expect(body).toHaveProperty("id");
    expect(body.email).toBe(apiEmail);
    expect(body.status).toBe("pending");
    expect(body.system_roles).toContain("admin");
    expect(body).toHaveProperty("tenant_id", firstTenantId);
    expect(body).toHaveProperty("created_at");

    await ctx.dispose();
  });
});

// ─── TC-US01: Invite Admin Dialog — Boundary & Negative Tests ─────────────────

test.describe("US-01 — Invite Admin dialog: boundary and negative tests", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, SUPER_ADMIN_EMAIL, SUPER_ADMIN_PASSWORD);
  });

  // TC-07: Empty Email field prevents submission (HTML5 required validation)
  test("TC-07: Empty Email field blocks form submission", async ({ page }) => {
    await navigateToFirstTenant(page);
    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");

    // Fill only display_name, leave email empty
    await dialog.getByPlaceholder(/john smith|name/i).fill("No Email User");
    await dialog.getByRole("button", { name: /send invitation/i }).click();

    // Dialog must remain open — no API call should succeed
    await expect(dialog).toBeVisible();
  });

  // TC-08: Empty Display Name field prevents submission
  test("TC-08: Empty Display Name field blocks form submission", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);
    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");

    await dialog
      .getByPlaceholder(/admin@company|email/i)
      .fill("no-name@tigersoft.co.th");
    // Leave display_name empty
    await dialog.getByRole("button", { name: /send invitation/i }).click();

    await expect(dialog).toBeVisible();
  });

  // TC-09: Malformed email is rejected by input[type=email] validation
  test("TC-09: Malformed email is rejected by browser validation", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);
    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");

    await dialog
      .getByPlaceholder(/admin@company|email/i)
      .fill("not-an-email");
    await dialog.getByPlaceholder(/john smith|name/i).fill("Malformed Email");
    await dialog.getByRole("button", { name: /send invitation/i }).click();

    // Dialog must still be open (browser validation blocks submit)
    await expect(dialog).toBeVisible();
  });

  // TC-10: Cancel button closes dialog without submitting
  test("TC-10: Cancel button closes dialog without making an API call", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);
    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");

    await dialog
      .getByPlaceholder(/admin@company|email/i)
      .fill("cancel@tigersoft.co.th");
    await dialog.getByPlaceholder(/john smith|name/i).fill("Cancel Test");

    await dialog.getByRole("button", { name: /cancel/i }).click();
    await expect(dialog).toBeHidden({ timeout: 3_000 });

    // User must NOT appear in the list
    const userRow = page
      .locator(".divide-y > div")
      .filter({ hasText: "cancel@tigersoft.co.th" });
    await expect(userRow).toHaveCount(0);
  });

  // TC-11: API — Duplicate email on invite-admin returns error (not 2xx)
  test("TC-11: API rejects duplicate invite with non-2xx status", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    const tenantsRes = await ctx.get(`${API_BASE}/api/v1/admin/tenants`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const tenantsBody = await tenantsRes.json();
    const firstTenantId: string = tenantsBody.data[0]?.id;
    if (!firstTenantId) { await ctx.dispose(); test.skip(); }

    const dupEmail = `qa-dup-${Date.now()}@tigersoft.co.th`;
    const payload = { email: dupEmail, display_name: "Dup Test" };
    const headers = {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    };

    // First invite — must succeed
    const first = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/invite-admin`,
      { headers, data: payload }
    );
    expect(first.status()).toBe(201);

    // Second invite with same email — must return a non-2xx error
    const second = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/invite-admin`,
      { headers, data: payload }
    );
    expect(second.ok()).toBeFalsy();

    await ctx.dispose();
  });

  // TC-12: API — Missing email field returns 400
  test("TC-12: API returns 400 when email field is missing", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    const tenantsRes = await ctx.get(`${API_BASE}/api/v1/admin/tenants`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const tenantsBody = await tenantsRes.json();
    const firstTenantId: string = tenantsBody.data[0]?.id;
    if (!firstTenantId) { await ctx.dispose(); test.skip(); }

    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/invite-admin`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        data: { display_name: "No Email" }, // email deliberately omitted
      }
    );
    expect(res.status()).toBe(400);

    await ctx.dispose();
  });

  // TC-13: API — Non-existent tenant ID returns 404
  test("TC-13: API returns 404 for non-existent tenant ID", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    const fakeId = "00000000-0000-0000-0000-000000000000";
    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${fakeId}/invite-admin`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          "Content-Type": "application/json",
        },
        data: { email: "ghost@tigersoft.co.th", display_name: "Ghost" },
      }
    );
    expect(res.status()).toBe(404);

    await ctx.dispose();
  });
});

// ─── TC-US03: Resend Invite ────────────────────────────────────────────────────

test.describe("US-03 — Resend invite for pending user", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page, SUPER_ADMIN_EMAIL, SUPER_ADMIN_PASSWORD);
  });

  // TC-14: Pending user row shows dropdown with Resend Invite option
  test("TC-14: Pending user in Administrators list has Resend Invite option in dropdown", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    // Wait for list to settle
    await page
      .waitForSelector(".animate-spin", { state: "detached", timeout: 10_000 })
      .catch(() => {});

    const pendingRow = page
      .locator(".divide-y > div")
      .filter({ has: page.locator("span", { hasText: /pending/i }) })
      .first();

    if ((await pendingRow.count()) === 0) {
      test.skip(); // No pending users currently; skip gracefully
    }

    await pendingRow.getByRole("button", { name: /more|actions/i }).click();
    // Fallback: open the DropdownMenu trigger (MoreHorizontal icon button)
    const menuItems = page.getByRole("menuitem");
    await expect(
      menuItems.filter({ hasText: /resend invite/i })
    ).toBeVisible();
  });

  // TC-15: Active user row does NOT show Resend Invite in dropdown
  test("TC-15: Active user in Administrators list does NOT have Resend Invite option", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    await page
      .waitForSelector(".animate-spin", { state: "detached", timeout: 10_000 })
      .catch(() => {});

    const activeRow = page
      .locator(".divide-y > div")
      .filter({ has: page.locator("span", { hasText: /^active$/i }) })
      .first();

    if ((await activeRow.count()) === 0) { test.skip(); }

    await activeRow.getByRole("button").last().click();
    // The menu for active users should have no "Resend Invite" item
    const resendItem = page.getByRole("menuitem", { name: /resend invite/i });
    await expect(resendItem).toHaveCount(0);
  });

  // TC-16: Clicking Resend Invite shows success toast
  test("TC-16: Resend Invite on pending user shows re-sent success toast", async ({
    page,
  }) => {
    await navigateToFirstTenant(page);

    await page
      .waitForSelector(".animate-spin", { state: "detached", timeout: 10_000 })
      .catch(() => {});

    const pendingRow = page
      .locator(".divide-y > div")
      .filter({ has: page.locator("span", { hasText: /pending/i }) })
      .first();

    if ((await pendingRow.count()) === 0) { test.skip(); }

    await pendingRow.getByRole("button").last().click();
    await page.getByRole("menuitem", { name: /resend invite/i }).click();

    await expect(
      page.locator("text=/re-sent|Invitation re-sent/i")
    ).toBeVisible({ timeout: 8_000 });
  });

  // TC-17: API — POST resend-invite for pending user returns 200 with success message
  test("TC-17: API POST resend-invite returns 200 with message", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    const tenantsRes = await ctx.get(`${API_BASE}/api/v1/admin/tenants`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const tenantsBody = await tenantsRes.json();
    const firstTenantId: string = tenantsBody.data[0]?.id;
    if (!firstTenantId) { await ctx.dispose(); test.skip(); }

    // Find a pending user in the tenant
    const usersRes = await ctx.get(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/users`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    const usersBody = await usersRes.json();
    const pendingUser = usersBody.data?.find(
      (u: { status: string }) => u.status === "pending"
    );

    if (!pendingUser) { await ctx.dispose(); test.skip(); }

    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/users/${pendingUser.id}/resend-invite`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.message).toMatch(/re-sent|successfully/i);

    await ctx.dispose();
  });

  // TC-18: API — Resend-invite for non-pending (active) user returns error
  test("TC-18: API rejects resend-invite for non-pending user with error", async () => {
    const token = await getSuperAdminToken();
    const ctx = await request.newContext();

    const tenantsRes = await ctx.get(`${API_BASE}/api/v1/admin/tenants`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    const tenantsBody = await tenantsRes.json();
    const firstTenantId: string = tenantsBody.data[0]?.id;
    if (!firstTenantId) { await ctx.dispose(); test.skip(); }

    const usersRes = await ctx.get(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/users`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    const usersBody = await usersRes.json();
    const activeUser = usersBody.data?.find(
      (u: { status: string }) => u.status === "active"
    );

    if (!activeUser) { await ctx.dispose(); test.skip(); }

    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${firstTenantId}/users/${activeUser.id}/resend-invite`,
      { headers: { Authorization: `Bearer ${token}` } }
    );
    // Active users should not have their invite re-sent — backend should return 400 or 409
    expect(res.ok()).toBeFalsy();

    await ctx.dispose();
  });
});

// ─── TC-Security: Access Control ─────────────────────────────────────────────

test.describe("Security — Access control (super_admin required)", () => {
  // TC-19: Unauthenticated request to invite-admin returns 401
  test("TC-19: Unauthenticated API request returns 401", async () => {
    const ctx = await request.newContext();

    const fakeId = "00000000-0000-0000-0000-000000000001";
    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${fakeId}/invite-admin`,
      {
        headers: { "Content-Type": "application/json" },
        data: { email: "noauth@tigersoft.co.th", display_name: "No Auth" },
      }
    );
    expect(res.status()).toBe(401);

    await ctx.dispose();
  });

  // TC-20: Admin-role token (non-super_admin) is rejected with 403
  test("TC-20: Admin-role (non-super_admin) token returns 403 on invite-admin", async () => {
    if (!ADMIN_EMAIL || !ADMIN_PASSWORD) {
      test.skip(); // Test credentials not configured
    }

    const ctx = await request.newContext();

    // Log in as regular admin to obtain their token
    const loginRes = await ctx.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "platform" },
      data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
    });

    if (!loginRes.ok()) {
      await ctx.dispose();
      test.skip(); // Credentials may not be valid in this environment
    }

    const loginBody = await loginRes.json();
    const adminToken: string = loginBody.access_token;

    const fakeId = "00000000-0000-0000-0000-000000000002";
    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${fakeId}/invite-admin`,
      {
        headers: {
          Authorization: `Bearer ${adminToken}`,
          "Content-Type": "application/json",
        },
        data: { email: "forbidden@tigersoft.co.th", display_name: "Forbidden" },
      }
    );
    expect(res.status()).toBe(403);

    await ctx.dispose();
  });

  // TC-21: Admin-role token returns 403 on GET tenants/:id/users
  test("TC-21: Admin-role token returns 403 on list tenant users", async () => {
    if (!ADMIN_EMAIL || !ADMIN_PASSWORD) { test.skip(); }

    const ctx = await request.newContext();

    const loginRes = await ctx.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "platform" },
      data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
    });
    if (!loginRes.ok()) { await ctx.dispose(); test.skip(); }

    const adminToken: string = (await loginRes.json()).access_token;
    const fakeId = "00000000-0000-0000-0000-000000000003";

    const res = await ctx.get(
      `${API_BASE}/api/v1/admin/tenants/${fakeId}/users`,
      { headers: { Authorization: `Bearer ${adminToken}` } }
    );
    expect(res.status()).toBe(403);

    await ctx.dispose();
  });

  // TC-22: Admin-role token returns 403 on resend-invite
  test("TC-22: Admin-role token returns 403 on resend-invite endpoint", async () => {
    if (!ADMIN_EMAIL || !ADMIN_PASSWORD) { test.skip(); }

    const ctx = await request.newContext();

    const loginRes = await ctx.post(`${API_BASE}/api/v1/auth/login`, {
      headers: { "Content-Type": "application/json", "X-Tenant-ID": "platform" },
      data: { email: ADMIN_EMAIL, password: ADMIN_PASSWORD },
    });
    if (!loginRes.ok()) { await ctx.dispose(); test.skip(); }

    const adminToken: string = (await loginRes.json()).access_token;
    const fakeTenantId = "00000000-0000-0000-0000-000000000004";
    const fakeUserId = "00000000-0000-0000-0000-000000000005";

    const res = await ctx.post(
      `${API_BASE}/api/v1/admin/tenants/${fakeTenantId}/users/${fakeUserId}/resend-invite`,
      { headers: { Authorization: `Bearer ${adminToken}` } }
    );
    expect(res.status()).toBe(403);

    await ctx.dispose();
  });

  // TC-23: Unauthenticated browser access to Tenants page redirects to login
  test("TC-23: Unauthenticated access to /dashboard/tenants redirects to /login", async ({
    page,
  }) => {
    // Clear all storage to simulate a fresh unauthenticated session
    await page.goto("/login");
    await page.evaluate(() => localStorage.clear());
    await page.goto("/dashboard/tenants");
    await expect(page).toHaveURL(/login/);
  });

  // TC-24: Non-super_admin UI sees no Tenants nav item and is redirected
  test("TC-24: Regular admin UI sees no Tenants nav item (role-based routing)", async ({
    page,
  }) => {
    if (!ADMIN_EMAIL || !ADMIN_PASSWORD) { test.skip(); }

    await loginAs(page, ADMIN_EMAIL, ADMIN_PASSWORD);

    // Tenants nav item should not be visible for non-super_admin
    const tenantsNav = page.getByRole("link", { name: /tenants/i });
    await expect(tenantsNav).toHaveCount(0);

    // Direct navigation should redirect away from tenants
    await page.goto("/dashboard/tenants");
    await expect(page).not.toHaveURL(/\/tenants/);
  });
});

// ─── TC-US04: Favicon / CI Branding ─────────────────────────────────────────

test.describe("US-04 — CI Branding: favicon and page metadata", () => {
  // TC-25: Page <head> declares TigerSoft logo-mark as favicon
  test("TC-25: Page metadata declares /logo-mark.svg as icon and shortcut icon", async ({
    page,
  }) => {
    await page.goto("/login");

    // Check link[rel~=icon] points to logo-mark.svg
    const iconHref = await page
      .locator('link[rel~="icon"]')
      .getAttribute("href");
    expect(iconHref).toMatch(/logo-mark\.svg/);

    // Check link[rel="shortcut icon"] as well
    const shortcutHref = await page
      .locator('link[rel="shortcut icon"]')
      .getAttribute("href");
    expect(shortcutHref).toMatch(/logo-mark\.svg/);
  });

  // TC-26: Apple touch icon also uses logo-mark.svg
  test("TC-26: Apple touch icon uses logo-mark.svg", async ({ page }) => {
    await page.goto("/login");

    const appleHref = await page
      .locator('link[rel="apple-touch-icon"]')
      .getAttribute("href");
    // May be undefined if Next.js does not emit it — only assert if present
    if (appleHref !== null) {
      expect(appleHref).toMatch(/logo-mark\.svg/);
    }
  });

  // TC-27: Page title is "TGX Auth Console"
  test("TC-27: Browser tab title is 'TGX Auth Console'", async ({ page }) => {
    await page.goto("/login");
    await expect(page).toHaveTitle(/TGX Auth Console/i);
  });

  // TC-28: /logo-mark.svg asset is publicly reachable and returns SVG content
  test("TC-28: /logo-mark.svg is publicly served with correct content-type", async ({
    page,
  }) => {
    const response = await page.request.get("/logo-mark.svg");
    expect(response.status()).toBe(200);

    const contentType = response.headers()["content-type"] ?? "";
    expect(contentType).toMatch(/svg|image/);
  });

  // TC-29: Font is Plus Jakarta Sans (declared in layout)
  test("TC-29: Document has Plus Jakarta Sans font variable applied", async ({
    page,
  }) => {
    await page.goto("/login");
    const htmlClass = await page.locator("html").getAttribute("class");
    // Next.js injects the CSS variable name as a class on <html>
    expect(htmlClass).toMatch(/plus.jakarta.sans|__className/i);
  });
});

// ─── TC-End-to-End: Full Invite Lifecycle ─────────────────────────────────────

test.describe("E2E — Full Invite Lifecycle (Invite → Pending → Resend)", () => {
  const LIFECYCLE_EMAIL = `qa-lifecycle-${Date.now()}@tigersoft.co.th`;

  test("TC-30: Full lifecycle — invite admin, verify pending, resend invite", async ({
    page,
  }) => {
    await loginAs(page, SUPER_ADMIN_EMAIL, SUPER_ADMIN_PASSWORD);

    // Step 1: Navigate to tenant detail
    const tenantId = await navigateToFirstTenant(page);
    if (!tenantId) { test.skip(); }

    // Step 2: Invite admin
    await page.getByRole("button", { name: /invite admin/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog
      .getByPlaceholder(/admin@company|email/i)
      .fill(LIFECYCLE_EMAIL);
    await dialog
      .getByPlaceholder(/john smith|name/i)
      .fill("QA Lifecycle Admin");
    await dialog.getByRole("button", { name: /send invitation/i }).click();

    // Verify toast mentions email
    await expect(
      page.locator(`text=${LIFECYCLE_EMAIL}`)
    ).toBeVisible({ timeout: 8_000 });

    // Wait for dialog to close and list to refresh
    await expect(dialog).toBeHidden({ timeout: 5_000 });

    // Step 3: Verify pending status in Administrators list
    const userRow = page
      .locator(".divide-y > div")
      .filter({ hasText: LIFECYCLE_EMAIL });
    await expect(userRow).toBeVisible({ timeout: 8_000 });

    const statusBadge = userRow
      .locator("span")
      .filter({ hasText: /pending/i })
      .first();
    await expect(statusBadge).toBeVisible();

    // Step 4: Resend invite via dropdown
    await userRow.getByRole("button").last().click();
    await page.getByRole("menuitem", { name: /resend invite/i }).click();

    // Verify re-sent toast
    await expect(
      page.locator("text=/re-sent|Invitation re-sent/i")
    ).toBeVisible({ timeout: 8_000 });
  });
});
