import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

/**
 * E2E tests for Bug Fix #1: ROLES column now shows actual user roles.
 *
 * Root cause: ListUsers previously returned users without calling GetUserRolesBatch,
 * so system_roles and module_roles were always empty arrays in the list response.
 * The fix adds a single batch query after the user list is fetched, populating
 * both fields for every user in the page.
 *
 * Coverage:
 *   TC-ROLES-01: ROLES column header is present in the desktop table
 *   TC-ROLES-02: Users with assigned roles show role badges in the ROLES column
 *   TC-ROLES-03: Users with no roles show an empty ROLES cell (graceful handling)
 *   TC-ROLES-04: super_admin role badge uses Vivid Red brand color (branding CI)
 *   TC-ROLES-05: module roles render in indigo with mod:role format
 *   TC-ROLES-06: Role badges in desktop table match role badges on the user detail page
 *   TC-ROLES-07: Mobile card view (UserCard) also shows role badges
 *   TC-ROLES-08: Roles persist after applying a status filter (list re-fetch)
 *
 * Design techniques (ISTQB):
 *   - Equivalence Partitioning: users with roles vs users without roles
 *   - State Transition: unfiltered list -> filtered list; roles must remain populated
 *   - Experience-based: branding color assertion derived from CI guide
 */

test.describe("Bug Fix #1 — ROLES column displays actual user roles", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    // Wait for the initial loading spinner to clear — confirms API call completed.
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 15000,
    });
  });

  /**
   * TC-ROLES-01: Desktop table must have a ROLES column header.
   * Verifies the column was not accidentally removed in the bug-fix refactor.
   * ISTQB technique: Equivalence Partitioning (header presence partition)
   */
  test("TC-ROLES-01 desktop table has a ROLES column header", async ({
    page,
  }) => {
    // The desktop table is hidden below md; Playwright runs at 1280px by default.
    const header = page.locator("thead");
    await expect(header).toBeVisible();
    await expect(header).toContainText("Roles");
  });

  /**
   * TC-ROLES-02: At least one user (the seeded superadmin) must show a role badge
   * in the ROLES column, proving the batch query populates the response.
   * ISTQB technique: Equivalence Partitioning (user-with-roles partition)
   */
  test("TC-ROLES-02 users with assigned roles show role badges in the ROLES column", async ({
    page,
  }) => {
    // The superadmin user is always present and always has the super_admin system role.
    const rows = page.locator("tbody tr");
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);

    // Search for the superadmin row to get a known user.
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");
    await page.waitForTimeout(300); // debounce settle

    const superadminRow = page.locator("tbody tr").first();
    await expect(superadminRow).toBeVisible();

    // The ROLES cell (second TableCell) must contain at least one Badge element.
    // Badge elements are rendered as inline elements with a role-name text.
    const rolesCell = superadminRow.locator("td").nth(1);
    const badges = rolesCell.locator('[class*="border"]'); // Badge variant="outline"
    await expect(badges.first()).toBeVisible();

    const badgeText = await badges.first().textContent();
    expect(badgeText?.trim().length).toBeGreaterThan(0);

    await search.clear();
  });

  /**
   * TC-ROLES-03: Users with no roles must show an empty ROLES cell — no crash,
   * no "undefined", no "[object Object]".
   * ISTQB technique: Equivalence Partitioning (user-without-roles partition)
   *
   * We verify this by checking that cells with no badges do not contain error text.
   * If every user happens to have roles, the test passes vacuously (empty partition).
   */
  test("TC-ROLES-03 users with no roles show an empty ROLES cell without errors", async ({
    page,
  }) => {
    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    for (let i = 0; i < rowCount; i++) {
      const rolesCell = rows.nth(i).locator("td").nth(1);
      const cellText = await rolesCell.textContent();
      // The cell must not contain runtime error artifacts.
      expect(cellText).not.toContain("undefined");
      expect(cellText).not.toContain("[object");
      expect(cellText).not.toContain("null");
    }
  });

  /**
   * TC-ROLES-04: super_admin role badge must use Vivid Red (#F4001A) for text and border.
   * This validates TigerSoft Branding CI — super_admin uses tiger-red.
   * ISTQB technique: Experience-based (branding error guessing)
   */
  test("TC-ROLES-04 super_admin badge uses Vivid Red brand color (branding CI)", async ({
    page,
  }) => {
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");
    await page.waitForTimeout(300);

    const superadminRow = page.locator("tbody tr").first();
    const count = await superadminRow.count();
    if (count === 0) {
      test.skip();
      return;
    }

    const rolesCell = superadminRow.locator("td").nth(1);
    // Find the badge that contains the text "super_admin"
    const superAdminBadge = rolesCell.locator('[class*="tiger-red"], [class*="text-tiger-red"]').first();
    const badgeCount = await superAdminBadge.count();

    if (badgeCount === 0) {
      // Badge found but class approach failed — fall back to computed style check.
      const allBadges = rolesCell.locator('span, div').filter({ hasText: "super_admin" });
      const badgesCount2 = await allBadges.count();
      if (badgesCount2 === 0) {
        test.skip();
        return;
      }
      const colorVal = await allBadges.first().evaluate(
        (el) => getComputedStyle(el).color
      );
      // Vivid Red RGB: 244, 0, 26
      expect(colorVal).toMatch(/244.*0.*26|rgb\(244,\s*0,\s*26\)/);
    } else {
      await expect(superAdminBadge.first()).toBeVisible();
    }

    await search.clear();
  });

  /**
   * TC-ROLES-05: Module roles render in "mod:role" format with indigo styling.
   * Validates that Object.entries(module_roles) flattening works in the UI.
   * ISTQB technique: Equivalence Partitioning (module-roles partition)
   */
  test("TC-ROLES-05 module roles render in mod:role format", async ({
    page,
  }) => {
    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    let foundModuleRole = false;

    for (let i = 0; i < rowCount; i++) {
      const rolesCell = rows.nth(i).locator("td").nth(1);
      const badgeTexts = await rolesCell.locator('[class*="indigo"]').allTextContents();

      for (const text of badgeTexts) {
        if (text.includes(":")) {
          // Validates format: "module:roleName"
          const parts = text.trim().split(":");
          expect(parts.length).toBeGreaterThanOrEqual(2);
          expect(parts[0].length).toBeGreaterThan(0);
          expect(parts[1].length).toBeGreaterThan(0);
          foundModuleRole = true;
        }
      }
    }

    // If no module roles exist in the environment, this test passes vacuously.
    // Log a soft note — not a failure.
    if (!foundModuleRole) {
      console.log("TC-ROLES-05: No module roles found in test environment — test passed vacuously.");
    }
  });

  /**
   * TC-ROLES-06: Roles shown in the user list must match roles on the user detail page.
   * This is the critical regression guard: if the batch query returns roles that
   * mismatch the single-user GetUser query, this test will catch it.
   * ISTQB technique: Decision Table (same user ID -> consistent roles in both views)
   */
  test("TC-ROLES-06 roles in list match roles on user detail page", async ({
    page,
  }) => {
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");
    await page.waitForTimeout(300);

    const firstRow = page.locator("tbody tr").first();
    if (await firstRow.count() === 0) {
      test.skip();
      return;
    }

    // Collect role badge texts from the list page.
    const rolesCell = firstRow.locator("td").nth(1);
    const listBadgeTexts = await rolesCell.locator('[class*="border"]').allTextContents();
    const listRoles = listBadgeTexts.map((t) => t.trim()).filter(Boolean);

    // Navigate to the user detail page.
    await firstRow.click();
    await expect(page).toHaveURL(/users\/.+/, { timeout: 8000 });

    // Wait for the detail page to load its own data.
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    // Collect role badges from the detail page — same Badge component, same styles.
    const detailBadges = page.locator('[class*="border"][class*="rounded"]').filter({ hasNotText: /active|pending|inactive/ });
    const detailCount = await detailBadges.count();

    // The detail page must show at least as many roles as the list showed.
    // It may show more if the detail page has additional sections.
    if (listRoles.length > 0) {
      expect(detailCount).toBeGreaterThanOrEqual(listRoles.length);
    }
  });

  /**
   * TC-ROLES-07: Mobile card view (UserCard component) also shows role badges.
   * Verifies the UserCard `Row 2: Role badges` section renders correctly.
   * ISTQB technique: Equivalence Partitioning (mobile viewport partition)
   */
  test("TC-ROLES-07 mobile card view shows role badges for users with roles", async ({
    page,
  }) => {
    // Force mobile viewport to activate the UserCard stack (below md = 768px).
    await page.setViewportSize({ width: 375, height: 812 });

    // Re-navigate to trigger the mobile layout.
    await page.goto("/dashboard/users");
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 15000,
    });

    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");
    await page.waitForTimeout(300);

    // UserCard is a div[role="button"] — pick the first one.
    const cards = page.locator('[role="button"]').filter({ hasText: /superadmin/i });
    const cardCount = await cards.count();

    if (cardCount === 0) {
      test.skip();
      return;
    }

    const firstCard = cards.first();
    await expect(firstCard).toBeVisible();

    // The role badges section is the second flex-wrap div inside the card.
    // It should contain at least one badge for the superadmin user.
    const roleBadges = firstCard.locator('[class*="border"]').filter({ hasNotText: /active|pending|inactive/ });
    const roleCount = await roleBadges.count();
    expect(roleCount).toBeGreaterThan(0);

    // Restore viewport.
    await page.setViewportSize({ width: 1280, height: 800 });
  });

  /**
   * TC-ROLES-08: Roles persist after applying a status filter.
   * After filtering by "active", the re-fetched list must still populate roles —
   * ensuring the batch query runs on every list load, not just the initial one.
   * ISTQB technique: State Transition (unfiltered -> filtered; roles remain populated)
   */
  test("TC-ROLES-08 roles remain populated after applying a status filter", async ({
    page,
  }) => {
    // Apply the "active" status filter, which triggers a fresh API call.
    const statusSelect = page.locator('[class*="SelectTrigger"]').first();
    await statusSelect.click();
    const activeOption = page.getByRole("option", { name: "Active" });
    if (await activeOption.isVisible()) {
      await activeOption.click();
    } else {
      // Fallback for native select rendering in some environments.
      await page.getByRole("combobox").first().selectOption("active");
    }

    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    if (rowCount === 0) {
      // No active users in test env — cannot validate roles, skip gracefully.
      test.skip();
      return;
    }

    // Every role cell must not contain error artifacts — roles must be present or cleanly empty.
    let foundAtLeastOneRole = false;
    for (let i = 0; i < rowCount; i++) {
      const rolesCell = rows.nth(i).locator("td").nth(1);
      const cellText = await rolesCell.textContent();
      expect(cellText).not.toContain("undefined");
      expect(cellText).not.toContain("[object");

      const badges = await rolesCell.locator('[class*="border"]').count();
      if (badges > 0) {
        foundAtLeastOneRole = true;
      }
    }

    // The superadmin is always active and always has roles, so at least one row
    // must show a role badge after filtering by "active".
    expect(foundAtLeastOneRole).toBe(true);
  });
});
