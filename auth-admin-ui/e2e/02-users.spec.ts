import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

const TEST_EMAIL = `qa-${Date.now()}@tigersoft.co.th`;
const TEST_NAME = "QA Test User";

test.describe("Users page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await page.waitForSelector("table, text=No users found");
  });

  test("displays user list with columns including Roles", async ({ page }) => {
    const header = page.locator("thead");
    await expect(header).toContainText("User");
    // Bug Fix #1 regression guard: ROLES column must be present.
    // Before the fix, this column rendered empty for all users.
    await expect(header).toContainText("Roles");
    await expect(header).toContainText("Status");
    await expect(header).toContainText("Joined");
  });

  test("search filters users by name or email", async ({ page }) => {
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");
    await expect(page.locator("tbody tr")).toHaveCount(1);
    await search.clear();
  });

  test("status filter shows only active users", async ({ page }) => {
    await page.getByRole("combobox").first().selectOption("active");
    const badges = page.locator("span.rounded-full").filter({ hasText: "active" });
    const count = await badges.count();
    expect(count).toBeGreaterThan(0);
    // Ensure no 'pending' badge is visible
    await expect(page.locator("span.rounded-full").filter({ hasText: "pending" })).toHaveCount(0);
  });

  test("status filter shows only pending users", async ({ page }) => {
    await page.getByRole("combobox").first().selectOption("pending");
    const rows = page.locator("tbody tr");
    const count = await rows.count();
    if (count > 0) {
      await expect(page.locator("span.rounded-full").filter({ hasText: "active" })).toHaveCount(0);
    }
  });

  test("invite user opens dialog and sends invite", async ({ page }) => {
    await page.getByRole("button", { name: /invite user/i }).click();
    const dialog = page.getByRole("dialog");
    await expect(dialog).toBeVisible();

    await dialog.getByPlaceholder(/name|ชื่อ/i).fill(TEST_NAME);
    await dialog.getByPlaceholder(/email/i).fill(TEST_EMAIL);
    await dialog.getByRole("button", { name: /send invite/i }).click();

    // Toast success
    await expect(page.getByText(/invitation sent/i)).toBeVisible();

    // New user appears in list with pending status
    await expect(page.locator("td", { hasText: TEST_EMAIL })).toBeVisible();
  });

  test("re-invite option visible only for pending users in dropdown", async ({ page }) => {
    // Find a pending user row
    const pendingRow = page.locator("tr").filter({ has: page.locator("span.rounded-full", { hasText: "pending" }) }).first();
    const count = await pendingRow.count();

    if (count === 0) {
      test.skip(); // No pending users available
    }

    await pendingRow.locator("button[aria-haspopup=menu]").click();
    await expect(page.getByRole("menuitem", { name: /resend invite/i })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: /suspend/i })).toHaveCount(0);
  });

  test("suspend option visible only for active users in dropdown", async ({ page }) => {
    const activeRow = page.locator("tr").filter({ has: page.locator("span.rounded-full", { hasText: "active" }) }).first();
    await activeRow.locator("button[aria-haspopup=menu]").click();
    await expect(page.getByRole("menuitem", { name: /suspend/i })).toBeVisible();
    await expect(page.getByRole("menuitem", { name: /resend invite/i })).toHaveCount(0);
  });

  test("clicking row navigates to user detail", async ({ page }) => {
    const firstRow = page.locator("tbody tr").first();
    await firstRow.click();
    await expect(page).toHaveURL(/users\//);
  });
});
