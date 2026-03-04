import { test, expect } from "@playwright/test";
import { loginAs } from "./helpers/auth";

test.describe("User detail page", () => {
  test.beforeEach(async ({ page }) => {
    await loginAs(page);
    await page.goto("/dashboard/users");
    await page.waitForSelector("tbody tr");
    // Navigate to first user's detail page
    await page.locator("tbody tr").first().click();
    await page.waitForURL(/users\//);
  });

  test("shows account info card", async ({ page }) => {
    await expect(page.getByText("Account Info")).toBeVisible();
    await expect(page.getByText("User ID")).toBeVisible();
    await expect(page.getByText("Status")).toBeVisible();
    await expect(page.getByText("Joined")).toBeVisible();
  });

  test("shows roles section with save button", async ({ page }) => {
    await expect(page.getByText("Roles")).toBeVisible();
    await expect(page.getByRole("button", { name: /save roles/i })).toBeVisible();
  });

  test("back button navigates to users list", async ({ page }) => {
    await page.getByRole("button").filter({ has: page.locator("svg") }).first().click();
    await expect(page).toHaveURL(/\/users$/);
  });

  test("active user shows Suspend button, no Resend Invite button", async ({ page }) => {
    // Navigate to a known active user
    await page.goto("/dashboard/users");
    const activeRow = page.locator("tr").filter({
      has: page.locator("span.rounded-full", { hasText: "active" }),
    }).first();
    if (await activeRow.count() === 0) { test.skip(); }
    await activeRow.click();
    await page.waitForURL(/users\//);

    await expect(page.getByRole("button", { name: /suspend/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /resend invite/i })).toHaveCount(0);
  });

  test("pending user shows Resend Invite button, no Suspend button", async ({ page }) => {
    await page.goto("/dashboard/users");
    const pendingRow = page.locator("tr").filter({
      has: page.locator("span.rounded-full", { hasText: "pending" }),
    }).first();
    if (await pendingRow.count() === 0) { test.skip(); }
    await pendingRow.click();
    await page.waitForURL(/users\//);

    await expect(page.getByRole("button", { name: /resend invite/i })).toBeVisible();
    await expect(page.getByRole("button", { name: /suspend/i })).toHaveCount(0);
  });

  test("resend invite shows success toast", async ({ page }) => {
    await page.goto("/dashboard/users");
    const pendingRow = page.locator("tr").filter({
      has: page.locator("span.rounded-full", { hasText: "pending" }),
    }).first();
    if (await pendingRow.count() === 0) { test.skip(); }
    await pendingRow.click();
    await page.waitForURL(/users\//);

    await page.getByRole("button", { name: /resend invite/i }).click();
    await expect(page.getByText(/re-sent/i)).toBeVisible({ timeout: 5000 });
  });
});
