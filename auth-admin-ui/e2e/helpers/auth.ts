import { Page } from "@playwright/test";

const ADMIN_EMAIL = process.env.TEST_ADMIN_EMAIL || "superadmin@tigersoft.co.th";
const ADMIN_PASSWORD = process.env.TEST_ADMIN_PASSWORD || "";

export async function loginAs(page: Page, email = ADMIN_EMAIL, password = ADMIN_PASSWORD) {
  await page.goto("/login");
  await page.getByPlaceholder(/email/i).fill(email);
  await page.getByPlaceholder(/password/i).fill(password);
  await page.getByRole("button", { name: /sign in/i }).click();
  await page.waitForURL("**/dashboard**");
}

export async function logout(page: Page) {
  // Click avatar/user menu then logout — adjust selector if header changes
  await page.getByRole("button", { name: /logout|sign out/i }).click();
  await page.waitForURL("**/login**");
}
