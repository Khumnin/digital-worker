import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

test.describe("Roles page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/roles");
  });

  test("displays existing system roles", async ({ page }) => {
    await expect(page.getByText("admin")).toBeVisible();
    await expect(page.getByText("user")).toBeVisible();
  });

  test("create a custom role", async ({ page }) => {
    const roleName = `qa-role-${Date.now()}`;

    await page.getByRole("button", { name: /create role|add role/i }).click();
    const dialog = page.getByRole("dialog");
    await dialog.getByLabel(/name/i).fill(roleName);
    await dialog.getByLabel(/description/i).fill("QA automated test role");
    await dialog.getByRole("button", { name: /create|save/i }).click();

    await expect(page.getByText(roleName)).toBeVisible();
  });

  test("system roles cannot be deleted", async ({ page }) => {
    // System role rows should not show a delete button
    const systemBadge = page.locator("text=system").first();
    await expect(systemBadge).toBeVisible();
    // The delete button for system roles should be absent or disabled
    const row = systemBadge.locator("../..");
    const deleteBtn = row.getByRole("button", { name: /delete/i });
    const deleteBtnCount = await deleteBtn.count();
    if (deleteBtnCount > 0) {
      await expect(deleteBtn).toBeDisabled();
    }
  });
});
