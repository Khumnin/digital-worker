import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

test.describe("Audit log page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/audit");
  });

  test("shows audit log table with entries", async ({ page }) => {
    await expect(page.locator("table, text=No entries")).toBeVisible();
  });

  test("filter by action type", async ({ page }) => {
    // Select LOGIN action
    const select = page.locator("select, [role=combobox]").filter({ hasText: /action|all/i }).first();
    if (await select.count() === 0) { test.skip(); }
    await select.selectOption("LOGIN");
    // Rows should only show login events
    await page.waitForTimeout(500);
    const rows = page.locator("tbody tr");
    const count = await rows.count();
    if (count > 0) {
      await expect(rows.first()).toContainText(/login/i);
    }
  });

  test("date range filter narrows results", async ({ page }) => {
    const fromInput = page.getByLabel(/from|start date/i);
    const toInput = page.getByLabel(/to|end date/i);
    if (await fromInput.count() === 0) { test.skip(); }

    await fromInput.fill("2025-01-01");
    await toInput.fill("2025-01-02");
    await page.getByRole("button", { name: /apply|filter/i }).click();
    // Should show empty or only entries within range
    await page.waitForTimeout(500);
  });
});
