import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

test.describe("Settings page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/settings");
  });

  test("displays current tenant settings", async ({ page }) => {
    await expect(page.getByText(/tenant|settings/i)).toBeVisible();
    await expect(page.getByText(/session/i)).toBeVisible();
  });

  test("can update session duration", async ({ page }) => {
    const input = page.getByLabel(/session duration/i);
    await input.clear();
    await input.fill("8");
    await page.getByRole("button", { name: /save/i }).click();
    await expect(page.getByText(/saved|updated/i)).toBeVisible({ timeout: 5000 });
  });
});
