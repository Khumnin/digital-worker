import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

const TEST_EMAIL = `qa-${Date.now()}@tigersoft.co.th`;
const TEST_NAME = "QA Test User";

// ---------------------------------------------------------------------------
// Helper — wait until the users page loading spinner is gone.
// ---------------------------------------------------------------------------
async function waitForUsersPageLoad(page: import("@playwright/test").Page) {
  await page.waitForSelector('[class*="animate-spin"]', {
    state: "hidden",
    timeout: 12000,
  });
}

test.describe("Users page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await waitForUsersPageLoad(page);
    await page.waitForSelector("table, text=No users found");
  });

  // -------------------------------------------------------------------------
  // TC-USR-01: Table column headings are all present.
  // ISTQB: Functional Suitability — Completeness
  // -------------------------------------------------------------------------
  test("TC-USR-01 displays user list with columns including Roles", async ({ page }) => {
    const header = page.locator("thead");
    await expect(header).toContainText("User");
    // Bug Fix #1 regression guard: ROLES column must be present.
    await expect(header).toContainText("Roles");
    await expect(header).toContainText("Status");
    await expect(header).toContainText("Joined");
  });

  // -------------------------------------------------------------------------
  // TC-USR-02: Search filter by name or email.
  // ISTQB: Equivalence Partitioning (valid search text)
  // -------------------------------------------------------------------------
  test("TC-USR-02 search filters users by name or email", async ({ page }) => {
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");
    await expect(page.locator("tbody tr")).toHaveCount(1);
    await search.clear();
  });

  // -------------------------------------------------------------------------
  // TC-USR-03: Status filter — active only.
  // ISTQB: Equivalence Partitioning (active partition)
  // Bug Fix #2 regression: status filter must actually filter the table.
  // -------------------------------------------------------------------------
  test("TC-USR-03 status filter shows only active users", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Active" }).click();
    await waitForUsersPageLoad(page);

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();
    if (rowCount === 0) {
      await expect(page.getByText(/no users found/i)).toBeVisible();
      return;
    }

    const activeBadges  = page.locator("span.rounded-full").filter({ hasText: /^active$/ });
    const pendingBadges = page.locator("span.rounded-full").filter({ hasText: /^pending$/ });
    const inactiveBadges = page.locator("span.rounded-full").filter({ hasText: /^inactive$/ });

    await expect(activeBadges.first()).toBeVisible();
    await expect(pendingBadges).toHaveCount(0);
    await expect(inactiveBadges).toHaveCount(0);
  });

  // -------------------------------------------------------------------------
  // TC-USR-04: Status filter — pending only.
  // ISTQB: Equivalence Partitioning (pending partition)
  // Bug Fix #2 regression: no active badges must appear in pending-filtered view.
  // -------------------------------------------------------------------------
  test("TC-USR-04 status filter shows only pending users", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Pending" }).click();
    await waitForUsersPageLoad(page);

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();
    if (rowCount > 0) {
      await expect(page.locator("span.rounded-full").filter({ hasText: /^active$/ })).toHaveCount(0);
      await expect(page.locator("span.rounded-full").filter({ hasText: /^inactive$/ })).toHaveCount(0);
    }
  });

  // -------------------------------------------------------------------------
  // TC-USR-05: Status filter — inactive only.
  // ISTQB: Equivalence Partitioning (inactive partition)
  // -------------------------------------------------------------------------
  test("TC-USR-05 status filter shows only inactive users", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Inactive" }).click();
    await waitForUsersPageLoad(page);

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();
    if (rowCount === 0) {
      await expect(page.getByText(/no users found/i)).toBeVisible();
      return;
    }

    await expect(page.locator("span.rounded-full").filter({ hasText: /^active$/ })).toHaveCount(0);
    await expect(page.locator("span.rounded-full").filter({ hasText: /^pending$/ })).toHaveCount(0);
    await expect(page.locator("span.rounded-full").filter({ hasText: /^inactive$/ }).first()).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // TC-USR-06: Invite user flow.
  // ISTQB: State Transition (no-user state -> pending-invite state)
  // -------------------------------------------------------------------------
  test("TC-USR-06 invite user opens dialog and sends invite", async ({ page }) => {
    await page.getByRole("button", { name: /invite user/i }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    await dialog.getByPlaceholder(/name|ชื่อ/i).fill(TEST_NAME);
    await dialog.getByPlaceholder(/email/i).fill(TEST_EMAIL);
    await dialog.getByRole("button", { name: /send invite/i }).click();

    await expect(page.getByText(/invitation sent/i)).toBeVisible();
    await expect(page.locator("td", { hasText: TEST_EMAIL })).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // TC-USR-07: Resend invite option visible only for pending users.
  // ISTQB: Decision Table (status=pending -> resend visible, suspend hidden)
  // Regression: user action menu must still work after filter bug fix.
  // -------------------------------------------------------------------------
  test("TC-USR-07 re-invite option visible only for pending users in dropdown", async ({ page }) => {
    const pendingRow = page.locator("tr").filter({
      has: page.locator("span.rounded-full", { hasText: /^pending$/ }),
    }).first();

    if (await pendingRow.count() === 0) {
      test.skip();
      return;
    }

    await pendingRow.locator("button[aria-haspopup=menu]").click();
    await expect(page.getByRole("menuitem", { name: /resend invite/i })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: /suspend/i })).toHaveCount(0);
  });

  // -------------------------------------------------------------------------
  // TC-USR-08: Suspend option visible only for active users.
  // ISTQB: Decision Table (status=active -> suspend visible, resend hidden)
  // Regression: user action menu must still work after filter bug fix.
  // -------------------------------------------------------------------------
  test("TC-USR-08 suspend option visible only for active users in dropdown", async ({ page }) => {
    const activeRow = page.locator("tr").filter({
      has: page.locator("span.rounded-full", { hasText: /^active$/ }),
    }).first();
    await activeRow.locator("button[aria-haspopup=menu]").click();
    await expect(page.getByRole("menuitem", { name: /suspend/i })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: /resend invite/i })).toHaveCount(0);
  });

  // -------------------------------------------------------------------------
  // TC-USR-09: Row click navigates to user detail.
  // ISTQB: State Transition (list view -> detail view)
  // Regression: navigation must still work after filter bug fix.
  // -------------------------------------------------------------------------
  test("TC-USR-09 clicking row navigates to user detail", async ({ page }) => {
    const firstRow = page.locator("tbody tr").first();
    await firstRow.click();
    await expect(page).toHaveURL(/users\//);
  });

  // -------------------------------------------------------------------------
  // TC-USR-10: Search still works when status filter is active (combination).
  // ISTQB: Decision Table (statusFilter=active AND searchText=X)
  // Bug Fix #2 regression: combined client-side filters must both apply.
  // -------------------------------------------------------------------------
  test("TC-USR-10 search and status filter work together", async ({ page }) => {
    // Apply status filter first.
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Active" }).click();
    await waitForUsersPageLoad(page);

    // Then apply a search that should narrow results further.
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");

    // At most 1 row matching both active AND "superadmin" in name/email.
    const rows = page.locator("tbody tr");
    const count = await rows.count();
    expect(count).toBeLessThanOrEqual(1);

    // If rows exist, all must be active status.
    if (count > 0) {
      await expect(page.locator("span.rounded-full").filter({ hasText: /^pending$/ })).toHaveCount(0);
    }
  });

  // -------------------------------------------------------------------------
  // TC-USR-11: Clear filters button resets both status and module filters.
  // ISTQB: State Transition (filtered state -> cleared/all state)
  // -------------------------------------------------------------------------
  test("TC-USR-11 clear filters resets status and module to all", async ({ page }) => {
    // Apply status filter to make the clear button appear.
    const statusSelect = page.getByRole("combobox").first();
    await statusSelect.click();
    await page.getByRole("option", { name: "Active" }).click();
    await waitForUsersPageLoad(page);

    const clearBtn = page.getByRole("button", { name: /clear filters/i });
    await expect(clearBtn).toBeVisible();
    await clearBtn.click();
    await waitForUsersPageLoad(page);

    // Clear button should disappear when no active filters remain.
    await expect(clearBtn).toHaveCount(0);

    // All-users view must be restored — at least the superadmin user is present.
    const rows = page.locator("tbody tr");
    await expect(rows.first()).toBeVisible();
  });

  // -------------------------------------------------------------------------
  // TC-USR-12: Banner count matches number of pending rows after filter applied.
  // ISTQB: Equivalence Partitioning (pending count consistency)
  // Bug Fix #1 regression: banner count must equal actual pending row count.
  // -------------------------------------------------------------------------
  test("TC-USR-12 banner pending count matches actual pending row count in table", async ({ page }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Extract the count from the badge inside the banner.
    const badgeSpan = banner.locator("span.rounded-full").last();
    const bannerCountText = await badgeSpan.textContent();
    const bannerCount = parseInt(bannerCountText?.trim() ?? "0", 10);
    expect(bannerCount).toBeGreaterThan(0);

    // Click banner to apply pending filter.
    await banner.click();
    await waitForUsersPageLoad(page);

    // Count the actual rows returned with pending status badges.
    const pendingBadges = page.locator("span.rounded-full").filter({ hasText: /^pending$/ });
    const tableCount = await pendingBadges.count();

    // The table row count for pending must equal the banner count.
    // (The banner uses total from the API, and the table shows up to page_size=100.)
    // They must match when total <= 100.
    if (bannerCount <= 100) {
      expect(tableCount).toBe(bannerCount);
    } else {
      // If more than 100 pending users, the table can show at most 100.
      expect(tableCount).toBe(100);
    }
  });

  // -------------------------------------------------------------------------
  // TC-USR-13: Mobile view still shows users on narrow viewport.
  // ISTQB: ISO 25010 Compatibility / Portability
  // Regression: responsive layout must survive filter bug fix.
  // -------------------------------------------------------------------------
  test("TC-USR-13 mobile view shows user cards on narrow viewport", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto("/dashboard/users");
    await waitForUsersPageLoad(page);

    // On mobile the table is hidden; UserCard stack is shown instead.
    const table = page.locator("table");
    await expect(table).toBeHidden();

    // At least one user card must exist (superadmin is always present).
    const cards = page.locator('[data-testid="user-card"], .space-y-3 > *');
    const count = await cards.count();
    expect(count).toBeGreaterThan(0);
  });

  // -------------------------------------------------------------------------
  // TC-USR-14: Branding CI — primary CTA (Invite User) uses Vivid Red.
  // TigerSoft Branding CI compliance check.
  // -------------------------------------------------------------------------
  test("TC-USR-14 branding — Invite User button uses Vivid Red (#F4001A)", async ({ page }) => {
    const inviteBtn = page.getByRole("button", { name: /invite user/i });
    await expect(inviteBtn).toBeVisible();

    const bgColor = await inviteBtn.evaluate(
      (el) => getComputedStyle(el).backgroundColor
    );
    // Vivid Red RGB: 244, 0, 26
    expect(bgColor).toMatch(/rgb\(244,\s*0,\s*26\)/);
  });

  // -------------------------------------------------------------------------
  // TC-USR-15: Branding CI — no pure black (#000000) in text elements.
  // TigerSoft Branding CI compliance check.
  // -------------------------------------------------------------------------
  test("TC-USR-15 branding — user name text uses Oxford Blue not pure black", async ({ page }) => {
    const firstNameCell = page.locator("tbody tr").first().locator("td").first().locator("p").first();
    const textColor = await firstNameCell.evaluate(
      (el) => getComputedStyle(el).color
    );
    // Must NOT be pure black rgb(0, 0, 0).
    expect(textColor).not.toBe("rgb(0, 0, 0)");
  });
});
