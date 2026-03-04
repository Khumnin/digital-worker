import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

/**
 * Invite flow tests — cover the full invite lifecycle:
 * invite → pending status → resend → accept invite page
 *
 * NOTE: Acceptance of invite requires a real email inbox.
 * The accept-invite page can be tested directly via URL with a known token
 * (generate one via the API in a CI fixture if needed).
 */

const INVITE_EMAIL = `qa-invite-${Date.now()}@tigersoft.co.th`;

test.describe("Invite flow", () => {
  test("newly invited user appears with pending status", async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");

    await page.getByRole("button", { name: /invite user/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByPlaceholder(/name|ชื่อ/i).fill("QA Invite Test");
    await dialog.getByPlaceholder(/email/i).fill(INVITE_EMAIL);
    await dialog.getByRole("button", { name: /send invite/i }).click();

    await expect(page.getByText(/invitation sent/i)).toBeVisible();

    // Locate row and verify badge is "pending"
    const row = page.locator("tr").filter({ hasText: INVITE_EMAIL });
    await expect(row).toBeVisible();
    await expect(row.locator("span.rounded-full", { hasText: "pending" })).toBeVisible();
  });

  test("re-inviting same email resends (no 409 error)", async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");

    // Invite same email twice
    for (let i = 0; i < 2; i++) {
      await page.getByRole("button", { name: /invite user/i }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByPlaceholder(/name|ชื่อ/i).fill("QA Invite Test");
      await dialog.getByPlaceholder(/email/i).fill(INVITE_EMAIL);
      await dialog.getByRole("button", { name: /send invite/i }).click();
      await expect(page.getByText(/invitation sent/i)).toBeVisible();
      await page.waitForTimeout(500);
    }
  });

  test("resend invite button triggers success on pending user", async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");

    const pendingRow = page.locator("tr").filter({
      has: page.locator("span.rounded-full", { hasText: "pending" }),
    }).first();
    if (await pendingRow.count() === 0) { test.skip(); }

    await pendingRow.locator("button[aria-haspopup=menu]").click();
    await page.getByRole("menuitem", { name: /resend invite/i }).click();
    await expect(page.getByText(/re-sent/i)).toBeVisible({ timeout: 5000 });
  });
});

test.describe("Accept invite page", () => {
  test("shows error when token is missing", async ({ page }) => {
    await page.goto("/accept-invite");
    await expect(page.getByText(/invalid or expired/i)).toBeVisible();
  });

  test("shows error when token is invalid", async ({ page }) => {
    await page.goto("/accept-invite?token=invalidtoken123&tenant=platform");
    await expect(page.getByPlaceholder(/min. 8/i)).toBeVisible();

    await page.getByPlaceholder(/min. 8/i).fill("Password1!");
    await page.getByPlaceholder(/re-enter/i).fill("Password1!");
    await page.getByRole("button", { name: /activate/i }).click();

    // Should show an error toast (invalid token)
    await expect(page.getByText(/invalid|expired|failed/i)).toBeVisible({ timeout: 5000 });
  });

  test("password mismatch shows validation error", async ({ page }) => {
    await page.goto("/accept-invite?token=anytoken&tenant=platform");
    await page.getByPlaceholder(/min. 8/i).fill("Password1!");
    await page.getByPlaceholder(/re-enter/i).fill("Different1!");
    await page.getByRole("button", { name: /activate/i }).click();
    await expect(page.getByText(/do not match/i)).toBeVisible();
  });

  test("password too short shows validation error", async ({ page }) => {
    await page.goto("/accept-invite?token=anytoken&tenant=platform");
    await page.getByPlaceholder(/min. 8/i).fill("short");
    await page.getByPlaceholder(/re-enter/i).fill("short");
    await page.getByRole("button", { name: /activate/i }).click();
    await expect(page.getByText(/8 characters/i)).toBeVisible();
  });
});
