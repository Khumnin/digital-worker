import { test, expect } from "@playwright/test";

const ADMIN_EMAIL = process.env.TEST_ADMIN_EMAIL || "superadmin@tigersoft.co.th";
const ADMIN_PASSWORD = process.env.TEST_ADMIN_PASSWORD || "";

test.describe("Login", () => {
  test("shows login form", async ({ page }) => {
    await page.goto("/login");
    await expect(page.getByPlaceholder(/email/i)).toBeVisible();
    await expect(page.getByPlaceholder(/password/i)).toBeVisible();
    await expect(page.getByRole("button", { name: /sign in/i })).toBeVisible();
  });

  test("rejects wrong credentials", async ({ page }) => {
    await page.goto("/login");
    await page.getByPlaceholder(/email/i).fill("wrong@example.com");
    await page.getByPlaceholder(/password/i).fill("badpassword");
    await page.getByRole("button", { name: /sign in/i }).click();
    // Toast or inline error should appear; URL stays on /login
    await expect(page).toHaveURL(/login/);
  });

  test("successful login redirects to dashboard", async ({ page }) => {
    await page.goto("/login");
    await page.getByPlaceholder(/email/i).fill(ADMIN_EMAIL);
    await page.getByPlaceholder(/password/i).fill(ADMIN_PASSWORD);
    await page.getByRole("button", { name: /sign in/i }).click();
    await expect(page).toHaveURL(/dashboard/);
  });

  test("unauthenticated access to dashboard redirects to login", async ({ page }) => {
    await page.goto("/dashboard");
    await expect(page).toHaveURL(/login/);
  });
});
