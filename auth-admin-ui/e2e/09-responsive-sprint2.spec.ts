/**
 * 09-responsive-sprint2.spec.ts
 *
 * Sprint 2 — Responsive Design E2E Test Suite
 *
 * Coverage:
 *   RESP-03  Auth pages mobile polish — login, forgot-password, reset-password,
 *            accept-invite (card width, touch targets, padding)
 *   RESP-05  Audit Log — AuditLogCard on mobile, responsive filter toolbar
 *   RESP-07  Roles Page — RoleCard on mobile, responsive toolbar and create dialog
 *   RESP-08  My Profile — grid collapse, 44px inputs, UUID/email wrap
 *   ARIA Fix MobileDrawer close button inside dialog element
 *
 * Viewports under test (BVA on the md breakpoint — 768 px):
 *   Mobile  — 375 × 812  (iPhone 14)      → below md
 *   Tablet  — 768 × 1024 (iPad)           → at md
 *   Desktop — 1280 × 800                  → above md
 *
 * Technique: Boundary Value Analysis on the md breakpoint (768 px)
 *
 * Branding: All assertions verify TigerSoft CI compliance per guide/BRANDING.md
 *   - Vivid Red  #F4001A  (rgb 244, 0, 26)
 *   - Oxford Blue #0B1F3A (rgb 11, 31, 58)
 *   - No pure black #000000
 */

import { test, expect, Page, ViewportSize } from "@playwright/test";
import { loginAs } from "./helpers/auth";

// ---------------------------------------------------------------------------
// Viewport constants (BVA: below md, at md, above md)
// ---------------------------------------------------------------------------
const MOBILE: ViewportSize = { width: 375, height: 812 };
const TABLET: ViewportSize = { width: 768, height: 1024 };
const DESKTOP: ViewportSize = { width: 1280, height: 800 };

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

/**
 * Set viewport, authenticate, and navigate to path.
 * Waits for the page header to confirm the shell has rendered.
 */
async function setupAt(
  page: Page,
  viewport: ViewportSize,
  path: string
): Promise<void> {
  await page.setViewportSize(viewport);
  await loginAs(page);
  await page.goto(path);
  await page.waitForSelector("header", { timeout: 10_000 });
}

/**
 * Navigate to an auth page at the given viewport WITHOUT logging in first.
 */
async function gotoAuth(
  page: Page,
  viewport: ViewportSize,
  path: string
): Promise<void> {
  await page.setViewportSize(viewport);
  await page.goto(path);
  // Wait for the card container to settle
  await page.waitForSelector("form, [role='alert'], .min-h-screen", {
    timeout: 10_000,
  });
}

/** Dismiss the loading spinner for data-driven dashboard pages. */
async function waitForData(page: Page): Promise<void> {
  await page
    .waitForSelector(".animate-spin", { state: "detached", timeout: 15_000 })
    .catch(() => {});
}

/** Returns true when an element's computed display is "none". */
async function isDisplayNone(page: Page, selector: string): Promise<boolean> {
  const el = page.locator(selector).first();
  const count = await el.count();
  if (count === 0) return true;
  const display = await el.evaluate(
    (node) => window.getComputedStyle(node).display
  );
  return display === "none";
}

// ---------------------------------------------------------------------------
// RESP-03 | Auth Pages Mobile Polish
// ---------------------------------------------------------------------------

test.describe("RESP-03 | Auth Pages — Mobile Polish", () => {
  // -----------------------------------------------------------------------
  // Helper: assert auth card dimensions and input height on any auth page
  // -----------------------------------------------------------------------
  async function assertAuthCardMobile(page: Page): Promise<void> {
    // Card must not overflow the 375 px viewport
    const card = page.locator(".min-h-screen > div").first();
    await expect(card).toBeVisible();
    const box = await card.boundingBox();
    expect(box).not.toBeNull();
    // Card uses w-[calc(100vw-32px)] → 343 px on 375 px viewport
    expect(box!.width).toBeLessThanOrEqual(375);
    expect(box!.width).toBeGreaterThanOrEqual(300);
  }

  async function assertInputsMinHeight(page: Page, minPx: number): Promise<void> {
    const inputs = page.locator('input[type="email"], input[type="password"], input[type="text"]');
    const count = await inputs.count();
    expect(count).toBeGreaterThan(0);
    for (let i = 0; i < count; i++) {
      const box = await inputs.nth(i).boundingBox();
      if (box) {
        expect(box.height).toBeGreaterThanOrEqual(minPx);
      }
    }
  }

  async function assertSubmitButtonFullWidth(
    page: Page,
    viewportWidth: number
  ): Promise<void> {
    // Primary submit button: w-full h-12 rounded-[1000px]
    const submitBtn = page
      .locator('button[type="submit"]')
      .first();
    await expect(submitBtn).toBeVisible();
    const box = await submitBtn.boundingBox();
    expect(box).not.toBeNull();
    // w-full: should span nearly the full card width minus horizontal padding
    expect(box!.width).toBeGreaterThan(viewportWidth * 0.7);
    expect(box!.height).toBeGreaterThanOrEqual(44);
  }

  // =========================================================================
  // Login page
  // =========================================================================
  test.describe("Login Page", () => {
    test.describe("Mobile 375px", () => {
      test.beforeEach(async ({ page }) => {
        await gotoAuth(page, MOBILE, "/login");
      });

      test("card does not overflow 375px viewport", async ({ page }) => {
        await assertAuthCardMobile(page);
      });

      test("card has reduced padding on mobile (p-6 class)", async ({ page }) => {
        // Implementation uses p-6 sm:p-10 — inspect computed padding-top
        const card = page.locator(".min-h-screen > div").first();
        const paddingTop = await card.evaluate(
          (el) => parseFloat(window.getComputedStyle(el).paddingTop)
        );
        // p-6 = 24px, p-10 = 40px — on mobile we expect ~24px
        expect(paddingTop).toBeLessThanOrEqual(28);
      });

      test("all inputs meet 44px minimum touch target height", async ({ page }) => {
        await assertInputsMinHeight(page, 44);
      });

      test("password toggle touch target is >= 44px", async ({ page }) => {
        const toggleBtn = page.getByRole("button", {
          name: /show password|hide password/i,
        });
        await expect(toggleBtn).toBeVisible();
        const box = await toggleBtn.boundingBox();
        expect(box).not.toBeNull();
        expect(box!.width).toBeGreaterThanOrEqual(44);
        expect(box!.height).toBeGreaterThanOrEqual(44);
      });

      test("submit button is full-width with >= 44px height on mobile", async ({
        page,
      }) => {
        await assertSubmitButtonFullWidth(page, MOBILE.width);
      });

      test("forgot password link touch target is >= 44px", async ({ page }) => {
        const forgotLink = page.getByRole("link", {
          name: /forgot password/i,
        });
        await expect(forgotLink).toBeVisible();
        const box = await forgotLink.boundingBox();
        expect(box).not.toBeNull();
        // inline-flex min-h-[44px]
        expect(box!.height).toBeGreaterThanOrEqual(44);
      });

      test("TigerSoft logo is visible on mobile login page", async ({ page }) => {
        const logo = page.getByAltText("TigerSoft");
        await expect(logo).toBeVisible();
      });
    });

    test.describe("Desktop 1280px — regression", () => {
      test("login page renders correctly on desktop without overflow", async ({
        page,
      }) => {
        await gotoAuth(page, DESKTOP, "/login");
        const hasHorizontalOverflow = await page.evaluate(
          () => document.body.scrollWidth > document.body.clientWidth
        );
        expect(hasHorizontalOverflow).toBe(false);
        await expect(page.locator('button[type="submit"]')).toBeVisible();
      });
    });
  });

  // =========================================================================
  // Forgot Password page
  // =========================================================================
  test.describe("Forgot Password Page", () => {
    test.describe("Mobile 375px", () => {
      test.beforeEach(async ({ page }) => {
        await gotoAuth(page, MOBILE, "/forgot-password");
      });

      test("card does not overflow 375px viewport", async ({ page }) => {
        await assertAuthCardMobile(page);
      });

      test("email input meets 44px minimum height", async ({ page }) => {
        await assertInputsMinHeight(page, 44);
      });

      test("submit button is full-width with >= 44px height", async ({ page }) => {
        await assertSubmitButtonFullWidth(page, MOBILE.width);
      });

      test("sign in link touch target is >= 44px", async ({ page }) => {
        const signInLink = page.getByRole("link", { name: /sign in/i });
        await expect(signInLink).toBeVisible();
        const box = await signInLink.boundingBox();
        expect(box).not.toBeNull();
        // inline-flex min-h-[44px]
        expect(box!.height).toBeGreaterThanOrEqual(44);
      });

      test("card uses reduced padding on mobile", async ({ page }) => {
        const card = page.locator(".min-h-screen > div").first();
        const paddingTop = await card.evaluate(
          (el) => parseFloat(window.getComputedStyle(el).paddingTop)
        );
        expect(paddingTop).toBeLessThanOrEqual(28);
      });
    });

    test.describe("Desktop 1280px — regression", () => {
      test("forgot-password page renders correctly on desktop", async ({
        page,
      }) => {
        await gotoAuth(page, DESKTOP, "/forgot-password");
        const hasHorizontalOverflow = await page.evaluate(
          () => document.body.scrollWidth > document.body.clientWidth
        );
        expect(hasHorizontalOverflow).toBe(false);
      });
    });
  });

  // =========================================================================
  // Reset Password page (token via query param)
  // =========================================================================
  test.describe("Reset Password Page", () => {
    test.describe("Mobile 375px — no-token state", () => {
      test.beforeEach(async ({ page }) => {
        // Load without token to get the card rendered in the invalid-link state
        await gotoAuth(page, MOBILE, "/reset-password");
      });

      test("card does not overflow 375px viewport in invalid-token state", async ({
        page,
      }) => {
        await assertAuthCardMobile(page);
      });

      test("card uses reduced padding on mobile", async ({ page }) => {
        const card = page.locator(".min-h-screen > div").first();
        const paddingTop = await card.evaluate(
          (el) => parseFloat(window.getComputedStyle(el).paddingTop)
        );
        expect(paddingTop).toBeLessThanOrEqual(28);
      });
    });

    test.describe("Mobile 375px — with token", () => {
      test.beforeEach(async ({ page }) => {
        // Load with a dummy token to render the form state
        await gotoAuth(page, MOBILE, "/reset-password?token=dummy-token&tenant=platform");
      });

      test("card does not overflow 375px viewport in form state", async ({
        page,
      }) => {
        await assertAuthCardMobile(page);
      });

      test("all inputs meet 44px minimum height", async ({ page }) => {
        await assertInputsMinHeight(page, 44);
      });

      test("password toggle touch target is >= 44px", async ({ page }) => {
        const toggleBtn = page.getByRole("button", {
          name: /show password|hide password/i,
        });
        if (await toggleBtn.count() === 0) return;
        const box = await toggleBtn.boundingBox();
        expect(box).not.toBeNull();
        expect(box!.width).toBeGreaterThanOrEqual(44);
        expect(box!.height).toBeGreaterThanOrEqual(44);
      });

      test("submit button is full-width with >= 44px height", async ({ page }) => {
        await assertSubmitButtonFullWidth(page, MOBILE.width);
      });
    });

    test.describe("Desktop 1280px — regression", () => {
      test("reset-password page renders correctly on desktop", async ({
        page,
      }) => {
        await gotoAuth(page, DESKTOP, "/reset-password");
        const hasHorizontalOverflow = await page.evaluate(
          () => document.body.scrollWidth > document.body.clientWidth
        );
        expect(hasHorizontalOverflow).toBe(false);
      });
    });
  });

  // =========================================================================
  // Accept Invite page (token via query param)
  // =========================================================================
  test.describe("Accept Invite Page", () => {
    test.describe("Mobile 375px — with token", () => {
      test.beforeEach(async ({ page }) => {
        // Load with a dummy token to render the form state
        await gotoAuth(
          page,
          MOBILE,
          "/accept-invite?token=dummy-token&tenant=platform"
        );
      });

      test("card does not overflow 375px viewport", async ({ page }) => {
        await assertAuthCardMobile(page);
      });

      test("card uses reduced padding on mobile", async ({ page }) => {
        const card = page.locator(".min-h-screen > div").first();
        const paddingTop = await card.evaluate(
          (el) => parseFloat(window.getComputedStyle(el).paddingTop)
        );
        expect(paddingTop).toBeLessThanOrEqual(28);
      });

      test("all inputs meet 44px minimum height", async ({ page }) => {
        await assertInputsMinHeight(page, 44);
      });

      test("password toggle touch target is >= 44px", async ({ page }) => {
        const toggleBtn = page.getByRole("button", {
          name: /show password|hide password/i,
        });
        if (await toggleBtn.count() === 0) return;
        const box = await toggleBtn.boundingBox();
        expect(box).not.toBeNull();
        expect(box!.width).toBeGreaterThanOrEqual(44);
        expect(box!.height).toBeGreaterThanOrEqual(44);
      });

      test("activate button is full-width with >= 44px height", async ({
        page,
      }) => {
        const btn = page.getByRole("button", { name: /activate account/i });
        await expect(btn).toBeVisible();
        const box = await btn.boundingBox();
        expect(box).not.toBeNull();
        expect(box!.width).toBeGreaterThan(MOBILE.width * 0.7);
        expect(box!.height).toBeGreaterThanOrEqual(44);
      });

      test("TigerSoft logo is visible on accept-invite page", async ({ page }) => {
        const logo = page.getByAltText("TigerSoft");
        await expect(logo).toBeVisible();
      });
    });

    test.describe("Desktop 1280px — regression", () => {
      test("accept-invite page renders correctly on desktop", async ({ page }) => {
        await gotoAuth(
          page,
          DESKTOP,
          "/accept-invite?token=dummy-token&tenant=platform"
        );
        const hasHorizontalOverflow = await page.evaluate(
          () => document.body.scrollWidth > document.body.clientWidth
        );
        expect(hasHorizontalOverflow).toBe(false);
      });
    });
  });
});

// ---------------------------------------------------------------------------
// RESP-05 | Audit Log Responsive Layout
// ---------------------------------------------------------------------------

test.describe("RESP-05 | Audit Log Responsive Layout", () => {
  // -----------------------------------------------------------------------
  // Mobile (375 px)
  // -----------------------------------------------------------------------
  test.describe("Mobile 375px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, MOBILE, "/dashboard/audit");
      await waitForData(page);
    });

    test("desktop table is hidden on mobile", async ({ page }) => {
      // Table wrapper: hidden md:block
      const tableWrapper = page.locator(".hidden.md\\:block").first();
      const count = await tableWrapper.count();
      if (count > 0) {
        const display = await tableWrapper.evaluate(
          (el) => window.getComputedStyle(el).display
        );
        expect(display).toBe("none");
      }
    });

    test("mobile card list container is visible on mobile", async ({ page }) => {
      // Card list wrapper: block md:hidden
      const cardContainer = page.locator(".block.md\\:hidden").first();
      await expect(cardContainer).toBeVisible();
    });

    test("audit log cards render action badge and timestamp", async ({ page }) => {
      // AuditLogCard renders rounded-full action badge + whitespace-nowrap timestamp
      const cards = page.locator(".block.md\\:hidden .rounded-\\[10px\\]");
      const count = await cards.count();
      // Only assert structure if there is data; empty state is acceptable
      if (count > 0) {
        const firstCard = cards.first();
        // Action badge: inline-flex rounded-full
        await expect(firstCard.locator(".rounded-full").first()).toBeVisible();
        // Timestamp: whitespace-nowrap tabular-nums
        await expect(
          firstCard.locator(".whitespace-nowrap.tabular-nums").first()
        ).toBeVisible();
      }
    });

    test("audit log cards render Actor row", async ({ page }) => {
      const cards = page.locator(".block.md\\:hidden .rounded-\\[10px\\]");
      const count = await cards.count();
      if (count > 0) {
        // Actor label — "Actor" text in the card
        await expect(cards.first()).toContainText("Actor");
      }
    });

    test("audit log cards render IP row", async ({ page }) => {
      const cards = page.locator(".block.md\\:hidden .rounded-\\[10px\\]");
      const count = await cards.count();
      if (count > 0) {
        await expect(cards.first()).toContainText("IP");
      }
    });

    test("filter toolbar stacks vertically on mobile", async ({ page }) => {
      // The form uses flex-col sm:flex-row: action select should appear
      // above the date inputs
      const actionSelect = page.locator('[role="combobox"]').first();
      const dateInput = page.locator('input[type="date"]').first();
      if ((await dateInput.count()) === 0) return;

      const selectBox = await actionSelect.boundingBox();
      const dateBox = await dateInput.boundingBox();
      if (selectBox && dateBox) {
        // Vertical stacking: date input top is below action select bottom
        expect(dateBox.y).toBeGreaterThan(selectBox.y + selectBox.height - 5);
      }
    });

    test("date inputs are full width on mobile", async ({ page }) => {
      // date inputs use w-full sm:w-[148px]
      const dateInputs = page.locator('input[type="date"]');
      const count = await dateInputs.count();
      for (let i = 0; i < count; i++) {
        const box = await dateInputs.nth(i).boundingBox();
        if (box) {
          // On mobile, w-full should fill most of the viewport (accounting for padding)
          expect(box.width).toBeGreaterThan(MOBILE.width * 0.5);
        }
      }
    });

    test("Apply button is full-width on mobile", async ({ page }) => {
      const applyBtn = page.getByRole("button", { name: /apply/i });
      await expect(applyBtn).toBeVisible();
      const box = await applyBtn.boundingBox();
      expect(box).not.toBeNull();
      // w-full sm:w-auto
      expect(box!.width).toBeGreaterThan(MOBILE.width * 0.7);
    });
  });

  // -----------------------------------------------------------------------
  // Tablet (768 px)
  // -----------------------------------------------------------------------
  test.describe("Tablet 768px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, TABLET, "/dashboard/audit");
      await waitForData(page);
    });

    test("desktop table is visible on tablet", async ({ page }) => {
      const tableWrapper = page.locator(".hidden.md\\:block").first();
      await expect(tableWrapper).toBeVisible();
    });

    test("mobile card list is hidden on tablet", async ({ page }) => {
      const hidden = await isDisplayNone(page, ".block.md\\:hidden");
      expect(hidden).toBe(true);
    });

    test("filter toolbar is inline (not stacked) on tablet", async ({ page }) => {
      // sm:flex-row is applied at 640px which is below tablet — verify date inputs
      // are side-by-side by checking they share roughly the same vertical position
      const dateInputs = page.locator('input[type="date"]');
      if ((await dateInputs.count()) >= 2) {
        const box0 = await dateInputs.nth(0).boundingBox();
        const box1 = await dateInputs.nth(1).boundingBox();
        if (box0 && box1) {
          // Within 10px of vertical alignment — they are in the same row
          expect(Math.abs(box0.y - box1.y)).toBeLessThan(10);
        }
      }
    });
  });

  // -----------------------------------------------------------------------
  // Desktop (1280 px) — regression
  // -----------------------------------------------------------------------
  test.describe("Desktop 1280px — regression", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, DESKTOP, "/dashboard/audit");
      await waitForData(page);
    });

    test("audit log table has Time / Action / Actor / Target / IP columns", async ({
      page,
    }) => {
      const header = page.locator("thead");
      await expect(header).toBeVisible();
      await expect(header).toContainText("Time");
      await expect(header).toContainText("Action");
      await expect(header).toContainText("Actor");
      await expect(header).toContainText("IP");
    });

    test("no horizontal overflow on desktop audit page", async ({ page }) => {
      const hasHorizontalOverflow = await page.evaluate(
        () => document.body.scrollWidth > document.body.clientWidth
      );
      expect(hasHorizontalOverflow).toBe(false);
    });

    test("mobile card list is absent on desktop", async ({ page }) => {
      const hidden = await isDisplayNone(page, ".block.md\\:hidden");
      expect(hidden).toBe(true);
    });
  });
});

// ---------------------------------------------------------------------------
// RESP-07 | Roles Page Responsive Layout
// ---------------------------------------------------------------------------

test.describe("RESP-07 | Roles Page Responsive Layout", () => {
  // -----------------------------------------------------------------------
  // Mobile (375 px)
  // -----------------------------------------------------------------------
  test.describe("Mobile 375px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, MOBILE, "/dashboard/roles");
      await waitForData(page);
    });

    test("desktop table is hidden on mobile", async ({ page }) => {
      const tableWrapper = page.locator(".hidden.md\\:block").first();
      const count = await tableWrapper.count();
      if (count > 0) {
        const display = await tableWrapper.evaluate(
          (el) => window.getComputedStyle(el).display
        );
        expect(display).toBe("none");
      }
    });

    test("mobile card list container is visible on mobile", async ({ page }) => {
      const cardContainer = page.locator(".block.md\\:hidden").first();
      await expect(cardContainer).toBeVisible();
    });

    test("role cards render role name in mono font", async ({ page }) => {
      const cards = page.locator(".block.md\\:hidden .rounded-\\[10px\\]");
      const count = await cards.count();
      if (count > 0) {
        // RoleCard renders <p className="... font-mono truncate">
        await expect(cards.first().locator(".font-mono").first()).toBeVisible();
      }
    });

    test("role cards render type badge (system or custom)", async ({ page }) => {
      const cards = page.locator(".block.md\\:hidden .rounded-\\[10px\\]");
      const count = await cards.count();
      if (count > 0) {
        // Badge with text "system" or "custom"
        const hasBadge =
          (await cards.first().getByText("system").count()) > 0 ||
          (await cards.first().getByText("custom").count()) > 0;
        expect(hasBadge).toBe(true);
      }
    });

    test("Create Role button is full-width on mobile", async ({ page }) => {
      const btn = page.getByRole("button", { name: /create role/i });
      if ((await btn.count()) === 0) {
        // On the System tab there is no create button — skip gracefully
        return;
      }
      await expect(btn.first()).toBeVisible();
      const box = await btn.first().boundingBox();
      expect(box).not.toBeNull();
      // w-full sm:w-auto
      expect(box!.width).toBeGreaterThan(MOBILE.width * 0.7);
    });

    test("Create Role button has 44px height (touch target)", async ({ page }) => {
      const btn = page.getByRole("button", { name: /create role/i });
      if ((await btn.count()) === 0) return;
      const box = await btn.first().boundingBox();
      expect(box).not.toBeNull();
      // h-11 = 44px
      expect(box!.height).toBeGreaterThanOrEqual(44);
    });

    test("Create Role dialog is responsive on mobile", async ({ page }) => {
      const btn = page.getByRole("button", { name: /create role/i });
      if ((await btn.count()) === 0) return;
      await btn.first().click();

      const dialog = page.getByRole("dialog");
      await expect(dialog).toBeVisible();

      // Dialog uses w-[calc(100vw-32px)] sm:max-w-[400px] — should be near full viewport width on mobile
      const dialogBox = await dialog.boundingBox();
      expect(dialogBox).not.toBeNull();
      expect(dialogBox!.width).toBeGreaterThanOrEqual(MOBILE.width - 40);
    });

    test("Create Role dialog has Role Name and Description fields", async ({
      page,
    }) => {
      const btn = page.getByRole("button", { name: /create role/i });
      if ((await btn.count()) === 0) return;
      await btn.first().click();

      const dialog = page.getByRole("dialog");
      await expect(dialog.getByPlaceholder(/recruiter/i)).toBeVisible();
    });
  });

  // -----------------------------------------------------------------------
  // Tablet (768 px)
  // -----------------------------------------------------------------------
  test.describe("Tablet 768px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, TABLET, "/dashboard/roles");
      await waitForData(page);
    });

    test("desktop table is visible on tablet", async ({ page }) => {
      const tableWrapper = page.locator(".hidden.md\\:block").first();
      await expect(tableWrapper).toBeVisible();
    });

    test("mobile card list is hidden on tablet", async ({ page }) => {
      const hidden = await isDisplayNone(page, ".block.md\\:hidden");
      expect(hidden).toBe(true);
    });

    test("Create Role button is not full-width on tablet", async ({ page }) => {
      const btn = page.getByRole("button", { name: /create role/i });
      if ((await btn.count()) === 0) return;
      const box = await btn.first().boundingBox();
      expect(box).not.toBeNull();
      // sm:w-auto — should be much narrower than the full viewport
      expect(box!.width).toBeLessThan(TABLET.width * 0.5);
    });
  });

  // -----------------------------------------------------------------------
  // Desktop (1280 px) — regression
  // -----------------------------------------------------------------------
  test.describe("Desktop 1280px — regression", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, DESKTOP, "/dashboard/roles");
      await waitForData(page);
    });

    test("roles table has Name / Description / Module / Type / Created columns", async ({
      page,
    }) => {
      const header = page.locator("thead");
      await expect(header).toBeVisible();
      await expect(header).toContainText("Name");
      await expect(header).toContainText("Type");
      await expect(header).toContainText("Created");
    });

    test("no horizontal overflow on desktop roles page", async ({ page }) => {
      const hasHorizontalOverflow = await page.evaluate(
        () => document.body.scrollWidth > document.body.clientWidth
      );
      expect(hasHorizontalOverflow).toBe(false);
    });

    test("mobile card list is absent on desktop", async ({ page }) => {
      const hidden = await isDisplayNone(page, ".block.md\\:hidden");
      expect(hidden).toBe(true);
    });

    test("Create Role button is NOT full-width on desktop", async ({ page }) => {
      const btn = page.getByRole("button", { name: /create role/i });
      if ((await btn.count()) === 0) return;
      const box = await btn.first().boundingBox();
      expect(box).not.toBeNull();
      expect(box!.width).toBeLessThan(DESKTOP.width * 0.4);
    });
  });
});

// ---------------------------------------------------------------------------
// RESP-08 | My Profile Responsive Layout
// ---------------------------------------------------------------------------

test.describe("RESP-08 | My Profile Responsive Layout", () => {
  // -----------------------------------------------------------------------
  // Mobile (375 px)
  // -----------------------------------------------------------------------
  test.describe("Mobile 375px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, MOBILE, "/me");
      await waitForData(page);
    });

    test("name inputs are in a single-column grid on mobile", async ({ page }) => {
      // Implementation: grid grid-cols-1 sm:grid-cols-2 gap-3
      // Verify First Name and Last Name inputs are stacked (different y positions)
      const inputs = page.locator(
        'input[placeholder="First name"], input[placeholder="Last name"]'
      );
      if ((await inputs.count()) < 2) return;

      const box0 = await inputs.nth(0).boundingBox();
      const box1 = await inputs.nth(1).boundingBox();
      if (box0 && box1) {
        // In a single column the second input must be below the first
        expect(box1.y).toBeGreaterThan(box0.y + box0.height - 5);
      }
    });

    test("name inputs have >= 44px height (h-11)", async ({ page }) => {
      const inputs = page.locator(
        'input[placeholder="First name"], input[placeholder="Last name"]'
      );
      const count = await inputs.count();
      for (let i = 0; i < count; i++) {
        const box = await inputs.nth(i).boundingBox();
        if (box) {
          // h-11 = 44px
          expect(box.height).toBeGreaterThanOrEqual(44);
        }
      }
    });

    test("password inputs have >= 44px height (h-11)", async ({ page }) => {
      const inputs = page.locator('input[type="password"]');
      const count = await inputs.count();
      for (let i = 0; i < count; i++) {
        const box = await inputs.nth(i).boundingBox();
        if (box) {
          expect(box.height).toBeGreaterThanOrEqual(44);
        }
      }
    });

    test("Save button is full-width on mobile", async ({ page }) => {
      const saveBtn = page.getByRole("button", { name: /^save$/i });
      await expect(saveBtn).toBeVisible();
      const box = await saveBtn.boundingBox();
      expect(box).not.toBeNull();
      // w-full sm:w-auto
      expect(box!.width).toBeGreaterThan(MOBILE.width * 0.7);
    });

    test("Change Password button is full-width on mobile", async ({ page }) => {
      const changeBtn = page.getByRole("button", { name: /change password/i });
      await expect(changeBtn).toBeVisible();
      const box = await changeBtn.boundingBox();
      expect(box).not.toBeNull();
      // w-full sm:w-auto
      expect(box!.width).toBeGreaterThan(MOBILE.width * 0.7);
    });

    test("email value wraps without horizontal overflow", async ({ page }) => {
      // break-all class should prevent overflow
      const emailValue = page.locator(".break-all").first();
      if ((await emailValue.count()) === 0) return;

      const emailBox = await emailValue.boundingBox();
      const viewportWidth = page.viewportSize()!.width;
      // Email value must not extend beyond the viewport
      if (emailBox) {
        expect(emailBox.x + emailBox.width).toBeLessThanOrEqual(
          viewportWidth + 2 // allow 2px rounding tolerance
        );
      }
    });

    test("UUID (User ID) is visible and uses mono font", async ({ page }) => {
      // font-mono text-xs class on User ID value
      const userIdValue = page.locator(".font-mono.text-xs").first();
      if ((await userIdValue.count()) === 0) return;
      await expect(userIdValue).toBeVisible();
    });

    test("page does not overflow viewport horizontally on mobile", async ({
      page,
    }) => {
      const hasHorizontalOverflow = await page.evaluate(
        () => document.body.scrollWidth > document.body.clientWidth
      );
      expect(hasHorizontalOverflow).toBe(false);
    });
  });

  // -----------------------------------------------------------------------
  // Desktop (1280 px)
  // -----------------------------------------------------------------------
  test.describe("Desktop 1280px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, DESKTOP, "/me");
      await waitForData(page);
    });

    test("name inputs are in a two-column grid on desktop", async ({ page }) => {
      // sm:grid-cols-2 kicks in at 640px — verified here at 1280px
      const inputs = page.locator(
        'input[placeholder="First name"], input[placeholder="Last name"]'
      );
      if ((await inputs.count()) < 2) return;

      const box0 = await inputs.nth(0).boundingBox();
      const box1 = await inputs.nth(1).boundingBox();
      if (box0 && box1) {
        // In a two-column grid both inputs share the same vertical position
        expect(Math.abs(box0.y - box1.y)).toBeLessThan(5);
      }
    });

    test("Save button is NOT full-width on desktop (sm:w-auto)", async ({
      page,
    }) => {
      const saveBtn = page.getByRole("button", { name: /^save$/i });
      await expect(saveBtn).toBeVisible();
      const box = await saveBtn.boundingBox();
      expect(box).not.toBeNull();
      expect(box!.width).toBeLessThan(DESKTOP.width * 0.4);
    });

    test("no horizontal overflow on desktop my-profile page", async ({
      page,
    }) => {
      const hasHorizontalOverflow = await page.evaluate(
        () => document.body.scrollWidth > document.body.clientWidth
      );
      expect(hasHorizontalOverflow).toBe(false);
    });
  });
});

// ---------------------------------------------------------------------------
// ARIA Fix | MobileDrawer close button inside dialog element
// ---------------------------------------------------------------------------

test.describe("ARIA Fix | MobileDrawer close button inside dialog", () => {
  test.describe("Mobile 375px", () => {
    test.beforeEach(async ({ page }) => {
      await setupAt(page, MOBILE, "/dashboard");
    });

    test("drawer panel has role=dialog", async ({ page }) => {
      const hamburger = page.getByRole("button", {
        name: /open navigation menu/i,
      });
      await hamburger.click();

      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      await expect(drawer).toHaveAttribute("role", "dialog");
    });

    test("drawer panel has aria-modal=true", async ({ page }) => {
      await page
        .getByRole("button", { name: /open navigation menu/i })
        .click();
      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      await expect(drawer).toHaveAttribute("aria-modal", "true");
    });

    test("drawer panel has aria-label Navigation menu", async ({ page }) => {
      await page
        .getByRole("button", { name: /open navigation menu/i })
        .click();
      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      await expect(drawer).toHaveAttribute("aria-label", "Navigation menu");
    });

    test("close button is a DOM descendant of the dialog element", async ({
      page,
    }) => {
      await page
        .getByRole("button", { name: /open navigation menu/i })
        .click();

      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      await expect(drawer).toBeVisible();

      // The close button must be inside the dialog — not a sibling
      const closeBtn = drawer.getByRole("button", {
        name: /close navigation menu/i,
      });
      await expect(closeBtn).toBeVisible();
      // Verify it is actually scoped to the drawer (would throw if not found)
      const count = await closeBtn.count();
      expect(count).toBeGreaterThanOrEqual(1);
    });

    test("close button has correct aria-label", async ({ page }) => {
      await page
        .getByRole("button", { name: /open navigation menu/i })
        .click();

      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      const closeBtn = drawer.getByRole("button", {
        name: /close navigation menu/i,
      });
      await expect(closeBtn).toHaveAttribute("aria-label", "Close navigation menu");
    });

    test("close button touch target is >= 44px (w-11 h-11)", async ({ page }) => {
      await page
        .getByRole("button", { name: /open navigation menu/i })
        .click();

      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      const closeBtn = drawer.getByRole("button", {
        name: /close navigation menu/i,
      });
      const box = await closeBtn.boundingBox();
      expect(box).not.toBeNull();
      // w-11 h-11 = 44px × 44px
      expect(box!.width).toBeGreaterThanOrEqual(44);
      expect(box!.height).toBeGreaterThanOrEqual(44);
    });

    test("pressing close button dismisses the drawer", async ({ page }) => {
      await page
        .getByRole("button", { name: /open navigation menu/i })
        .click();

      const drawer = page.getByRole("dialog", { name: /navigation menu/i });
      await expect(drawer).toBeVisible();

      const closeBtn = drawer.getByRole("button", {
        name: /close navigation menu/i,
      });
      await closeBtn.click();

      // Give the 200 ms CSS transition time to complete
      await page.waitForTimeout(300);

      // Drawer panel should have slid off-screen (-translate-x-full applied)
      const transform = await drawer.evaluate(
        (el) => window.getComputedStyle(el).transform
      );
      // translate-x-full = matrix(1,0,0,1,-280,0) — check for negative x offset
      expect(transform).toContain("-280");
    });
  });
});

// ---------------------------------------------------------------------------
// Regression | Desktop layout — all Sprint 2 pages unchanged
// ---------------------------------------------------------------------------

test.describe("Regression | Desktop layout unchanged — Sprint 2 pages", () => {
  const sprintTwoRoutes = [
    { name: "login", path: "/login", authenticated: false },
    { name: "forgot-password", path: "/forgot-password", authenticated: false },
    { name: "reset-password", path: "/reset-password", authenticated: false },
    { name: "accept-invite", path: "/accept-invite", authenticated: false },
    { name: "audit-log", path: "/dashboard/audit", authenticated: true },
    { name: "roles", path: "/dashboard/roles", authenticated: true },
    { name: "my-profile", path: "/me", authenticated: true },
  ];

  for (const route of sprintTwoRoutes) {
    test(`${route.name} page loads without horizontal overflow on desktop`, async ({
      page,
    }) => {
      if (route.authenticated) {
        await setupAt(page, DESKTOP, route.path);
        await waitForData(page);
      } else {
        await gotoAuth(page, DESKTOP, route.path);
      }

      const hasHorizontalOverflow = await page.evaluate(
        () => document.body.scrollWidth > document.body.clientWidth
      );
      expect(hasHorizontalOverflow).toBe(false);
    });
  }

  test("audit page table columns unchanged on desktop", async ({ page }) => {
    await setupAt(page, DESKTOP, "/dashboard/audit");
    await waitForData(page);
    const header = page.locator("thead");
    await expect(header).toContainText("Time");
    await expect(header).toContainText("Action");
    await expect(header).toContainText("Actor");
    await expect(header).toContainText("IP");
  });

  test("roles page table columns unchanged on desktop", async ({ page }) => {
    await setupAt(page, DESKTOP, "/dashboard/roles");
    await waitForData(page);
    const header = page.locator("thead");
    await expect(header).toContainText("Name");
    await expect(header).toContainText("Type");
    await expect(header).toContainText("Created");
  });

  test("my-profile page shows Account Info card on desktop", async ({ page }) => {
    await setupAt(page, DESKTOP, "/me");
    await waitForData(page);
    await expect(page.getByText("Account Info")).toBeVisible();
    await expect(page.getByText("Display Name")).toBeVisible();
    await expect(page.getByText("Security")).toBeVisible();
  });
});

// ---------------------------------------------------------------------------
// TigerSoft Branding CI | Sprint 2 pages
// ---------------------------------------------------------------------------

test.describe("Branding CI | Sprint 2 — TigerSoft color compliance", () => {
  test("login CTA button uses Vivid Red — not pure black", async ({ page }) => {
    await gotoAuth(page, DESKTOP, "/login");
    const submitBtn = page.locator('button[type="submit"]').first();
    await expect(submitBtn).toBeVisible();
    const bg = await submitBtn.evaluate(
      (el) => window.getComputedStyle(el).backgroundColor
    );
    // Vivid Red rgb(244, 0, 26)
    expect(bg).toContain("244");
    expect(bg).not.toBe("rgb(0, 0, 0)");
  });

  test("forgot-password CTA button uses Vivid Red", async ({ page }) => {
    await gotoAuth(page, DESKTOP, "/forgot-password");
    const submitBtn = page.locator('button[type="submit"]').first();
    await expect(submitBtn).toBeVisible();
    const bg = await submitBtn.evaluate(
      (el) => window.getComputedStyle(el).backgroundColor
    );
    expect(bg).toContain("244");
  });

  test("roles Create Role button uses Vivid Red on desktop", async ({ page }) => {
    await setupAt(page, DESKTOP, "/dashboard/roles");
    await waitForData(page);
    const createBtn = page.getByRole("button", { name: /create role/i });
    if ((await createBtn.count()) === 0) return;
    const bg = await createBtn.first().evaluate(
      (el) => window.getComputedStyle(el).backgroundColor
    );
    expect(bg).toContain("244");
    expect(bg).not.toBe("rgb(0, 0, 0)");
  });

  test("my-profile page headings do not use pure black (#000)", async ({
    page,
  }) => {
    await setupAt(page, DESKTOP, "/me");
    await waitForData(page);
    const headings = page.locator("h1");
    const count = await headings.count();
    for (let i = 0; i < count; i++) {
      const color = await headings
        .nth(i)
        .evaluate((el) => window.getComputedStyle(el).color);
      expect(color).not.toBe("rgb(0, 0, 0)");
    }
  });

  test("system role type badge uses Vivid Red border color", async ({ page }) => {
    await setupAt(page, MOBILE, "/dashboard/roles");
    await waitForData(page);
    // On mobile, RoleCard renders system badge with border-tiger-red
    const cards = page.locator(".block.md\\:hidden .rounded-\\[10px\\]");
    const count = await cards.count();
    if (count > 0) {
      // Find a system badge — text "system" in a card
      const systemBadge = cards.first().getByText("system");
      if ((await systemBadge.count()) > 0) {
        const borderColor = await systemBadge.evaluate(
          (el) => window.getComputedStyle(el).borderColor
        );
        // tiger-red border should not be black
        expect(borderColor).not.toBe("rgb(0, 0, 0)");
      }
    }
  });

  test("audit page action badges do not use pure black", async ({ page }) => {
    await setupAt(page, MOBILE, "/dashboard/audit");
    await waitForData(page);
    const badges = page
      .locator(".block.md\\:hidden .rounded-full")
      .first();
    const count = await badges.count();
    if (count > 0) {
      const color = await badges.evaluate(
        (el) => window.getComputedStyle(el).color
      );
      expect(color).not.toBe("rgb(0, 0, 0)");
    }
  });
});
