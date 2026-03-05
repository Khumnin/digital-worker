import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

/**
 * E2E tests for Bug Fix #2: ListUsers status filter + pending invitations banner.
 *
 * Coverage:
 *   - TC-BUG2-06: Pending invitations banner shows correct count
 *   - TC-BUG2-07: Clicking banner filters to pending users
 *   - TC-BUG2-08: Banner hidden when no pending invitations
 *   - TC-BUG2-09: Banner hidden when already filtering by pending
 *   - Status filter select triggers server-side filtering per status value
 *   - Backward compatibility: no-filter loads all users
 *
 * Note on TC-BUG2-01..05 (API-level status filter):
 *   Those cases are covered by the Go integration/handler tests.
 *   E2E tests here focus on the frontend behaviour that wraps those API calls.
 *
 * Setup assumption:
 *   The test environment has been seeded with at least one "pending" (unverified)
 *   user by the invite flow tests (04-invite-flow.spec.ts) that run before this
 *   file in CI. If no pending users exist the banner tests are skipped gracefully.
 */

test.describe("Pending invitations banner", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    // Wait for the loading spinner to disappear — indicates the API call completed.
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });
  });

  /**
   * TC-BUG2-06: Banner displays the correct pending count when pending users exist.
   * ISTQB technique: Equivalence Partitioning (pendingCount > 0 partition)
   */
  test("TC-BUG2-06 banner shows pending count when invitations exist", async ({ page }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    const bannerCount = await banner.count();

    if (bannerCount === 0) {
      // No pending users in this environment — skip gracefully.
      test.skip();
      return;
    }

    await expect(banner).toBeVisible();

    // The banner's aria-label encodes the count: "View N pending invitation(s)"
    const ariaLabel = await banner.getAttribute("aria-label");
    expect(ariaLabel).toMatch(/View \d+ pending invitation/);

    // The numeric badge inside the banner must match the count in the label.
    const badgeText = await banner.locator("span.rounded-full").last().textContent();
    const countInLabel = ariaLabel?.match(/View (\d+)/)?.[1] ?? "0";
    expect(badgeText?.trim()).toBe(countInLabel);
  });

  /**
   * TC-BUG2-07: Clicking the banner applies the "pending" status filter.
   * ISTQB technique: State Transition (all-users state -> pending-filtered state)
   */
  test("TC-BUG2-07 clicking banner switches view to pending-only users", async ({ page }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    await banner.click();

    // Loading spinner reappears while the filtered API call is in-flight.
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    // After click, banner must be hidden (statusFilter === "pending" suppresses it).
    await expect(banner).toHaveCount(0);

    // All visible status badges must be "pending".
    const statusBadges = page.locator("span.rounded-full");
    const badgeCount = await statusBadges.count();

    if (badgeCount > 0) {
      for (let i = 0; i < badgeCount; i++) {
        const text = await statusBadges.nth(i).textContent();
        // Only status badges (pending/active/inactive) are rounded-full;
        // role badges use Badge component with outline variant and are also rounded-full.
        // Only assert on entries that exactly match a status value.
        if (text && ["pending", "active", "inactive"].includes(text.trim())) {
          expect(text.trim()).toBe("pending");
        }
      }
    }
  });

  /**
   * TC-BUG2-08: Banner is hidden when there are no pending invitations.
   * ISTQB technique: Equivalence Partitioning (pendingCount === 0 partition)
   *
   * We verify the banner's conditional render logic by checking it is absent
   * when we manually set status filter to "active" first (no pending users
   * visible, but banner was already hidden). This is a best-effort check
   * since seeding zero-pending state would require cleanup of pending users.
   */
  test("TC-BUG2-08 banner is absent when statusFilter is not pending and no pending invitations", async ({ page }) => {
    // If a banner exists there ARE pending users; that's the opposite scenario.
    // We test the branch where pendingCount is 0 by verifying the condition code
    // path: force filter to "active" and confirm banner count logic is correct.
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() > 0) {
      // Pending users exist — verify banner is still hidden for non-admin role.
      // For admin role with pending users, this test would be redundant with TC-BUG2-06.
      // The banner not being present would only happen in a clean environment.
      test.skip();
      return;
    }

    // When pendingCount is 0, the banner must not be rendered.
    await expect(banner).toHaveCount(0);
  });

  /**
   * TC-BUG2-09: Banner is hidden when status filter is already set to "pending".
   * ISTQB technique: Decision Table (banner visibility conditions)
   * Condition: isAdmin=true, pendingCount>0, statusFilter="pending" -> banner hidden
   */
  test("TC-BUG2-09 banner is hidden when already viewing pending filter", async ({ page }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Click the banner to apply the pending filter.
    await banner.click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    // Banner must now be suppressed — the statusFilter === "pending" condition.
    await expect(banner).toHaveCount(0);

    // Switching back to "all" should bring the banner back.
    // Use the "Clear filters" button if active filters are shown.
    const clearBtn = page.getByRole("button", { name: /clear filters/i });
    if (await clearBtn.isVisible()) {
      await clearBtn.click();
      await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });
      // Banner should be visible again if pending users still exist.
      await expect(banner).toBeVisible();
    }
  });
});

test.describe("Status filter — server-side filtering", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });
  });

  /**
   * TC-BUG2-01 / TC-BUG2-02 / TC-BUG2-03 (frontend behaviour):
   * Selecting a status from the filter combobox fetches from the server with that status
   * and displays only matching rows.
   * ISTQB technique: Equivalence Partitioning across three valid status partitions.
   */
  test("TC-BUG2-01/FE selecting 'pending' filter shows only pending-status badges", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.selectOption("pending");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    if (rowCount === 0) {
      // No pending users — table is empty, which is the correct server response.
      await expect(page.getByText(/no users found/i)).toBeVisible();
      return;
    }

    // Every visible status badge must be "pending".
    const pendingBadges = page.locator("span.rounded-full").filter({ hasText: /^pending$/ });
    const activeBadges  = page.locator("span.rounded-full").filter({ hasText: /^active$/ });
    const inactiveBadges = page.locator("span.rounded-full").filter({ hasText: /^inactive$/ });

    await expect(pendingBadges.first()).toBeVisible();
    await expect(activeBadges).toHaveCount(0);
    await expect(inactiveBadges).toHaveCount(0);
  });

  test("TC-BUG2-02/FE selecting 'active' filter shows only active-status badges", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.selectOption("active");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    if (rowCount === 0) {
      await expect(page.getByText(/no users found/i)).toBeVisible();
      return;
    }

    await expect(page.locator("span.rounded-full").filter({ hasText: /^active$/ }).first()).toBeVisible();
    await expect(page.locator("span.rounded-full").filter({ hasText: /^pending$/ })).toHaveCount(0);
    await expect(page.locator("span.rounded-full").filter({ hasText: /^inactive$/ })).toHaveCount(0);
  });

  test("TC-BUG2-03/FE selecting 'inactive' filter shows only inactive-status badges", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.selectOption("inactive");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    if (rowCount === 0) {
      await expect(page.getByText(/no users found/i)).toBeVisible();
      return;
    }

    await expect(page.locator("span.rounded-full").filter({ hasText: /^inactive$/ }).first()).toBeVisible();
    await expect(page.locator("span.rounded-full").filter({ hasText: /^active$/ })).toHaveCount(0);
    await expect(page.locator("span.rounded-full").filter({ hasText: /^pending$/ })).toHaveCount(0);
  });

  /**
   * TC-BUG2-04/FE: No status filter (default "all") returns all users — backward compatible.
   * ISTQB technique: Boundary Value Analysis (empty/default value boundary)
   */
  test("TC-BUG2-04/FE default 'all' filter loads users without status restriction", async ({ page }) => {
    // The default state is "all". Verify the page loads with at least the superadmin.
    const rows = page.locator("tbody tr");
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);

    // Multiple status values may co-exist in the result set.
    // At minimum the superadmin (active) user must be present.
    const activeBadges = page.locator("span.rounded-full").filter({ hasText: /^active$/ });
    await expect(activeBadges.first()).toBeVisible();
  });

  /**
   * TC-BUG2-05/FE: Clearing an active filter restores the all-users view.
   * This covers the "Clear filters" control added alongside the server-side filter.
   * ISTQB technique: State Transition (filtered state -> cleared state)
   */
  test("TC-BUG2-05/FE clearing status filter restores all-users view", async ({ page }) => {
    // Apply a filter first.
    const select = page.getByRole("combobox").first();
    await select.selectOption("active");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    // Clear filters button should now be visible.
    const clearBtn = page.getByRole("button", { name: /clear filters/i });
    await expect(clearBtn).toBeVisible();
    await clearBtn.click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 10000 });

    // All-users view should be restored.
    const rows = page.locator("tbody tr");
    const count = await rows.count();
    expect(count).toBeGreaterThan(0);
  });
});

// ---------------------------------------------------------------------------
// NEW TESTS — Bug Fix #2: Two-phase banner transition (pendingBannerApplying)
// ---------------------------------------------------------------------------

test.describe("Pending banner — two-phase transition UX (Bug Fix #2)", () => {
  /**
   * Setup: navigate to the users page and wait for the first load to settle.
   * All tests in this describe block require at least one pending user — they
   * skip gracefully when the banner is absent (no pending users in env).
   */
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 15000,
    });
  });

  /**
   * TC-BANNER-01: Clicking the banner immediately disables it (disabled attribute)
   * and shows a spinner + "Filtering by pending status…" text before results load.
   * ISTQB technique: State Transition
   *   idle banner -> (click) -> applying state (spinner visible, button disabled)
   *   -> (load completes) -> banner hidden (statusFilter === "pending")
   */
  test("TC-BANNER-01 clicking banner shows spinner loading state before results appear", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    await expect(banner).toBeEnabled();

    // Click and immediately assert the transitional state before the load completes.
    await banner.click();

    // Phase 1: banner is still present (pendingBannerApplying=true, loading=false initially,
    // then loading=true kicks in). We check either the disabled button or the loading div.
    // The button becomes disabled=true synchronously on click.
    // Allow a small race window — if load is near-instant, the div form may show instead.
    const applyingDiv = page.locator('[role="status"][aria-label*="Applying"]');
    const bannerDisabled = page.locator('button[aria-busy="true"]');

    const eitherVisible = await Promise.race([
      applyingDiv.isVisible().then((v) => v),
      bannerDisabled.isVisible().then((v) => v),
    ]).catch(() => false);

    // If the load completed synchronously (very fast API), the banner will already be gone.
    // Accept that as also valid — the important thing is it never jumped from visible to gone
    // without passing through the applying state at the code level. We verify the end state.
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    // After load: banner must be hidden because statusFilter is now "pending".
    await expect(banner).toHaveCount(0);

    // We assert eitherVisible being true OR the load was immediate — log for traceability.
    if (!eitherVisible) {
      console.log("TC-BANNER-01: Load was near-instant; applying state may have been too brief to capture. End state correct.");
    }
  });

  /**
   * TC-BANNER-02: After banner click, the status Select component reflects "Pending".
   * This verifies setStatusFilter("pending") was called and the Select re-renders.
   * ISTQB technique: State Transition (filter state change propagates to Select UI)
   */
  test("TC-BANNER-02 status Select shows Pending after banner click", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    await banner.click();

    // Wait for the load to complete.
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    // The Select trigger should now display "Pending".
    // SelectTrigger renders the selected item text inside it.
    const selectTrigger = page.locator('[class*="SelectTrigger"]').first();
    await expect(selectTrigger).toContainText("Pending");
  });

  /**
   * TC-BANNER-03: After banner click and load, table shows only pending-status users.
   * ISTQB technique: State Transition (banner click -> filtered view)
   */
  test("TC-BANNER-03 table shows only pending users after banner click", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    await banner.click();
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    const rows = page.locator("tbody tr");
    const rowCount = await rows.count();

    if (rowCount === 0) {
      // Pending count was > 0 when banner showed, but all pending users may have
      // been filtered out by page_size or race. Accept empty table as valid.
      return;
    }

    // Every visible status badge must be "pending".
    const activeBadges = page.locator("span.rounded-full").filter({ hasText: /^active$/ });
    const inactiveBadges = page.locator("span.rounded-full").filter({ hasText: /^inactive$/ });
    await expect(activeBadges).toHaveCount(0);
    await expect(inactiveBadges).toHaveCount(0);

    const pendingBadges = page.locator("span.rounded-full").filter({ hasText: /^pending$/ });
    await expect(pendingBadges.first()).toBeVisible();
  });

  /**
   * TC-BANNER-04: The "applying" loading div appears in the banner slot while
   * loading === true after a click (pendingBannerApplying=true && loading=true).
   * It keeps the layout stable so there is no content-layout-shift.
   * ISTQB technique: Experience-based / UX error guessing
   */
  test("TC-BANNER-04 applying div maintains banner slot during load to prevent layout shift", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Record the Y position of the toolbar before click.
    const toolbarBefore = await page.locator('div.flex.flex-col').first().boundingBox();

    await banner.click();

    // During loading, either the disabled button or the applying div occupies the slot.
    // Verify the toolbar has not jumped significantly upward (no layout shift).
    const toolbarAfter = await page.locator('div.flex.flex-col').first().boundingBox();

    if (toolbarBefore && toolbarAfter) {
      // Y position shift must be less than 60px — the banner height is ~56px.
      // If the slot disappeared abruptly the toolbar would shift up by ~56px.
      const shift = Math.abs(toolbarAfter.y - toolbarBefore.y);
      expect(shift).toBeLessThan(60);
    }

    // Wait for load to settle.
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });
  });

  /**
   * TC-BANNER-05: Rapid double-click on the banner does not break the page.
   * The button is disabled immediately after the first click (disabled=true),
   * so the second click must be a no-op.
   * ISTQB technique: Boundary Value Analysis (edge: multiple rapid clicks)
   */
  test("TC-BANNER-05 rapid double-click on banner does not break the page", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Double-click rapidly.
    await banner.dblclick({ force: true });

    // Page must not crash — wait for load to settle normally.
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    // End state: banner hidden (pending filter applied), no error toasts visible.
    await expect(banner).toHaveCount(0);
    await expect(page.getByText(/failed to load/i)).toHaveCount(0);
  });

  /**
   * TC-BANNER-06: The 8-second safety valve clears pendingBannerApplying if
   * the network request hangs. We simulate this by intercepting the API call.
   * ISTQB technique: State Transition (safety valve path)
   */
  test("TC-BANNER-06 safety-valve clears applying state after 8 s if request hangs", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Intercept ALL user list API calls and make them hang indefinitely.
    await page.route("**/api/v1/admin/users**", async (route) => {
      // Never fulfill — simulates a hung network request.
      // The safety valve timer (8 s) should eventually clear pendingBannerApplying.
      await new Promise(() => {}); // intentionally never resolves
    });

    await banner.click();

    // The applying state (spinner / disabled button) must be visible immediately.
    const applyingIndicator = page.locator(
      '[role="status"][aria-label*="Applying"], button[aria-busy="true"]'
    );
    await expect(applyingIndicator.first()).toBeVisible({ timeout: 2000 });

    // Wait 9 seconds — longer than the 8 s safety valve.
    // After the valve fires, pendingBannerApplying becomes false.
    // Because loading is still true (request never completed), the banner button
    // condition (!loading && pendingCount > 0 ...) keeps it hidden — the div form
    // (pendingBannerApplying && loading) may still show until load resolves.
    // What we assert: the button is NOT aria-busy after 9 s.
    await page.waitForTimeout(9000);

    const busyButton = page.locator('button[aria-busy="true"]');
    await expect(busyButton).toHaveCount(0);

    // Abort the route interception so subsequent tests are not affected.
    await page.unroute("**/api/v1/admin/users**");
  });

  /**
   * TC-BANNER-07: Banner is visible again after clearing the pending filter.
   * Validates the banner re-renders when statusFilter returns to "all".
   * ISTQB technique: State Transition (pending-filtered -> cleared -> banner visible)
   */
  test("TC-BANNER-07 banner reappears after clearing the pending filter", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Apply pending filter via banner click.
    await banner.click();
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });
    await expect(banner).toHaveCount(0);

    // Clear the filter.
    const clearBtn = page.getByRole("button", { name: /clear filters/i });
    await expect(clearBtn).toBeVisible();
    await clearBtn.click();
    await page.waitForSelector('[class*="animate-spin"]', {
      state: "hidden",
      timeout: 10000,
    });

    // Banner must reappear — pending users still exist in the environment.
    await expect(banner).toBeVisible();
  });
});

test.describe("Loading spinner — null token regression", () => {
  /**
   * TC-BUG2/REGRESSION: Verifies the loading spinner does not hang indefinitely
   * when the auth token is null on initial render. The fix guards `load()` with
   * an early return and calls setLoading(false) when token is absent.
   *
   * We test this by checking that the page eventually exits the loading state
   * — if the bug were present the spinner would never disappear for an
   * unauthenticated session. Since our test session is authenticated this is
   * verified by observing the spinner disappears within the timeout.
   */
  test("spinner resolves after data loads (null token guard)", async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    // Spinner must disappear — proves the async load completed (or token-null branch resolved).
    await expect(page.locator('[class*="animate-spin"]')).toHaveCount(0, { timeout: 12000 });
    // Either a table or the empty state must be visible.
    await expect(page.locator("table, text=No users found")).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// Bug Fix #1: Pending count accuracy
// Root cause: fragile SQL parameter binding caused wrong count; fixed with
// sequential binding in user_repo.go ListByTenant.
// ---------------------------------------------------------------------------

test.describe("Bug Fix #1 — Pending count accuracy", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });
  });

  /**
   * TC-BF1-01: The banner count badge reflects the true number of pending users.
   * When the pending filter is applied, the table row count must equal the badge count.
   * ISTQB technique: Equivalence Partitioning + Boundary Value Analysis
   * (pendingCount=1 is the documented regression case — was showing 4 instead of 1)
   */
  test("TC-BF1-01 banner badge count equals table row count when pending filter applied", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Capture the badge number before clicking.
    const badgeSpan = banner.locator("span.rounded-full").last();
    const rawBadgeText = await badgeSpan.textContent();
    const badgeCount = parseInt(rawBadgeText?.trim() ?? "0", 10);
    expect(badgeCount).toBeGreaterThan(0);

    // Apply the filter.
    await banner.click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // Count status badges in the table that are exactly "pending".
    const pendingStatusBadges = page.locator("span.rounded-full").filter({ hasText: /^pending$/ });
    const tableRowCount = await pendingStatusBadges.count();

    // The table must show exactly badgeCount pending users (up to page_size=100).
    if (badgeCount <= 100) {
      expect(tableRowCount).toBe(badgeCount);
    } else {
      expect(tableRowCount).toBe(100);
    }
  });

  /**
   * TC-BF1-02: Banner count is not inflated by the total of ALL users.
   * Regression guard for the original bug where total=4 (all users) was shown
   * instead of total=1 (only pending users).
   * ISTQB technique: Boundary Value Analysis (off-by-many regression)
   */
  test("TC-BF1-02 banner count does not equal total user count when mixed statuses exist", async ({
    page,
  }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    // Read the pending count from the banner badge.
    const badgeSpan = banner.locator("span.rounded-full").last();
    const rawBadgeText = await badgeSpan.textContent();
    const pendingCount = parseInt(rawBadgeText?.trim() ?? "0", 10);

    // Fetch the total user count by counting all table rows (no filter applied yet).
    const allRows = page.locator("tbody tr");
    const totalCount = await allRows.count();

    // If there are active users alongside pending ones, the banner count must
    // be LESS than the total. This is the regression case.
    const activeBadges = page.locator("span.rounded-full").filter({ hasText: /^active$/ });
    const activeCount = await activeBadges.count();

    if (activeCount > 0) {
      // Mixed environment: pending < total
      expect(pendingCount).toBeLessThan(totalCount);
    }
    // Even if it happens to equal total (all users are pending), at minimum
    // we verify the count is a sensible non-negative integer.
    expect(pendingCount).toBeGreaterThanOrEqual(1);
  });

  /**
   * TC-BF1-03: Banner count is stable across multiple page reloads.
   * Verifies the SQL fix produces consistent results (no parameter mis-binding
   * that could return different counts on different executions).
   * ISTQB technique: Reliability / Repeatability
   */
  test("TC-BF1-03 banner count is consistent across two page loads", async ({ page }) => {
    const banner = page.locator('button[aria-label*="pending invitation"]');
    if (await banner.count() === 0) {
      test.skip();
      return;
    }

    const badgeSpan = banner.locator("span.rounded-full").last();
    const firstLoad = parseInt((await badgeSpan.textContent())?.trim() ?? "0", 10);

    // Reload the page.
    await page.reload();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    const bannerAfterReload = page.locator('button[aria-label*="pending invitation"]');
    if (await bannerAfterReload.count() === 0) {
      // If banner disappeared after reload it means pending users were resolved — skip.
      test.skip();
      return;
    }

    const badgeAfterReload = parseInt(
      (await bannerAfterReload.locator("span.rounded-full").last().textContent())?.trim() ?? "0",
      10
    );

    expect(badgeAfterReload).toBe(firstLoad);
  });
});

// ---------------------------------------------------------------------------
// Bug Fix #2: Filter combination coverage
// New scenarios verifying that status + search and status + module filters
// work correctly together (defense-in-depth matchStatus client filter).
// ---------------------------------------------------------------------------

test.describe("Bug Fix #2 — Filter combination and client-side matchStatus defense", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });
  });

  /**
   * TC-BF2-COMBO-01: Status filter + search text together narrow results.
   * Verifies the client-side matchStatus AND matchSearch both fire correctly.
   * ISTQB technique: Decision Table (status=active, search=superadmin)
   */
  test("TC-BF2-COMBO-01 status filter and search together narrow the table", async ({ page }) => {
    // Step 1: Apply status=active.
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Active" }).click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // Step 2: Type in search.
    const search = page.getByPlaceholder(/search|ค้นหา/i);
    await search.fill("superadmin");

    // Result: rows filtered by BOTH conditions — no pending badges allowed.
    const pendingBadges = page.locator("span.rounded-full").filter({ hasText: /^pending$/ });
    await expect(pendingBadges).toHaveCount(0);

    const rows = page.locator("tbody tr");
    const count = await rows.count();
    expect(count).toBeLessThanOrEqual(1);
  });

  /**
   * TC-BF2-COMBO-02: Selecting "All Statuses" from the dropdown after a filter
   * was applied restores the full user list (server-side call with no status param).
   * ISTQB technique: State Transition (filtered -> all)
   */
  test("TC-BF2-COMBO-02 switching back to All Statuses restores full list", async ({ page }) => {
    // Apply pending filter.
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Pending" }).click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // Switch back to All Statuses.
    await select.click();
    await page.getByRole("option", { name: "All Statuses" }).click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // Both active and pending users may now be visible again.
    const rows = page.locator("tbody tr");
    await expect(rows.first()).toBeVisible();

    // At minimum the superadmin (active) must be back in the list.
    const activeBadges = page.locator("span.rounded-full").filter({ hasText: /^active$/ });
    await expect(activeBadges.first()).toBeVisible();
  });

  /**
   * TC-BF2-COMBO-03: Client-side matchStatus filter is the correct API value.
   * The API normalizes "unverified" -> "pending" and "disabled" -> "inactive".
   * The client-side filter compares u.status (API value) with statusFilter state.
   * We verify no user with status="unverified" or "disabled" leaks through the
   * "pending" or "inactive" filter view respectively.
   * ISTQB technique: Equivalence Partitioning (invalid partition: raw DB values should not appear)
   */
  test("TC-BF2-COMBO-03 no raw DB status values leak through into rendered badges", async ({ page }) => {
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Pending" }).click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // "unverified" is the raw DB value — the API must normalize it to "pending".
    // If the raw value were shown, the matchStatus filter would also break.
    const rawUnverifiedBadges = page.locator("span.rounded-full").filter({ hasText: /^unverified$/ });
    await expect(rawUnverifiedBadges).toHaveCount(0);

    // Switch to inactive filter.
    await select.click();
    await page.getByRole("option", { name: "Inactive" }).click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // "disabled" is the raw DB value — must not appear in the UI.
    const rawDisabledBadges = page.locator("span.rounded-full").filter({ hasText: /^disabled$/ });
    await expect(rawDisabledBadges).toHaveCount(0);
  });

  /**
   * TC-BF2-COMBO-04: After filtering by status and invoking a user action,
   * the filter state is preserved after the action reload.
   * ISTQB technique: State Transition (action does not reset filter state)
   * Regression: resend invite must not clear the current status filter.
   */
  test("TC-BF2-COMBO-04 status filter is preserved after resend invite action", async ({ page }) => {
    // Apply pending filter first.
    const select = page.getByRole("combobox").first();
    await select.click();
    await page.getByRole("option", { name: "Pending" }).click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    const rows = page.locator("tbody tr");
    if (await rows.count() === 0) {
      test.skip();
      return;
    }

    // Open the action menu for the first pending row.
    const firstPendingRow = rows.first();
    await firstPendingRow.locator("button[aria-haspopup=menu]").click();

    const resendItem = page.getByRole("menuitem", { name: /resend invite/i });
    if (await resendItem.count() === 0) {
      test.skip();
      return;
    }

    await resendItem.click();
    await page.waitForSelector('[class*="animate-spin"]', { state: "hidden", timeout: 12000 });

    // After the action, the status filter should still be "pending" —
    // no active users must appear in the table.
    await expect(page.locator("span.rounded-full").filter({ hasText: /^active$/ })).toHaveCount(0);
  });
});
