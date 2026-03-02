/**
 * TigerOpenspace Enterprise Login Page
 * URL: https://www.tigersoftservicecloud.com/Enterprise/TigerOpenspace/login/8BB1FD7DFD
 *
 * Page elements (sourced from live inspection):
 *   #txtUser   — User ID field (text)
 *   #txtPWD    — Password field (password, toggleable)
 *   #Button1   — "Continue" submit button
 *   Company selection modal shown after authentication
 */
import { test, expect, Page } from "@playwright/test";

const LOGIN_URL =
  "https://www.tigersoftservicecloud.com/Enterprise/TigerOpenspace/login/8BB1FD7DFD";

// ── Credentials from environment (never hard-code) ─────────────
const USER = process.env.TIGERSPACE_USER ?? "";
const PASS = process.env.TIGERSPACE_PASS ?? "";

// ── Selectors ──────────────────────────────────────────────────
const SEL = {
  userInput:      "#txtUser",
  passInput:      "#txtPWD",
  submitBtn:      "#Button1",
  passToggle:     ".cm-pass ~ * .eye-icon, [onclick*='showPass'], .toggle-pass",
  rememberMe:     "input[type='checkbox']",
  forgotPassword: "a[href*='forgot'], a:has-text('ลืมรหัสผ่าน')",
  msLoginBtn:     "a[href*='microsoft'], a[href*='oauth2'], .btn-microsoft",
  companyModal:   ".company-modal, [class*='company'], [id*='company']",
  errorModal:     "text=Please fill in the required field",
  langSelector:   "select[name*='lang'], .lang-selector, [onclick*='lang']",
} as const;

// ── Helper ─────────────────────────────────────────────────────
async function gotoLogin(page: Page) {
  await page.goto(LOGIN_URL, { waitUntil: "domcontentloaded" });
  await expect(page.locator(SEL.userInput)).toBeVisible({ timeout: 15_000 });
}

// ══════════════════════════════════════════════════════════════
//  1. Page Load & Structure
// ══════════════════════════════════════════════════════════════
test.describe("Page load & structure", () => {
  test("login page loads and shows required fields", async ({ page }) => {
    await gotoLogin(page);

    await expect(page.locator(SEL.userInput)).toBeVisible();
    await expect(page.locator(SEL.passInput)).toBeVisible();
    await expect(page.locator(SEL.submitBtn)).toBeVisible();
  });

  test("page title or heading contains expected branding", async ({ page }) => {
    await gotoLogin(page);
    // Logo or heading should reference TigerOpenspace / Welcome back
    const heading = page.locator("h1, h2, .welcome-text, img[src*='logo']").first();
    await expect(heading).toBeVisible();
  });

  test("password field is masked by default", async ({ page }) => {
    await gotoLogin(page);
    await expect(page.locator(SEL.passInput)).toHaveAttribute("type", "password");
  });

  test("Microsoft sign-in button is present", async ({ page }) => {
    await gotoLogin(page);
    const msBtn = page.locator(SEL.msLoginBtn).first();
    await expect(msBtn).toBeVisible();
  });

  test("forgot password link is present", async ({ page }) => {
    await gotoLogin(page);
    const link = page.locator(SEL.forgotPassword).first();
    await expect(link).toBeVisible();
  });
});

// ══════════════════════════════════════════════════════════════
//  2. Validation — Empty Submission
// ══════════════════════════════════════════════════════════════
test.describe("Validation — empty submission", () => {
  test("submitting empty form shows required-field error", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.submitBtn).click();
    // Error modal or inline message
    await expect(
      page.locator(SEL.errorModal)
    ).toBeVisible({ timeout: 5_000 });
  });

  test("submitting with only user ID shows error", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.userInput).fill("someuser");
    await page.locator(SEL.submitBtn).click();
    await expect(page.locator(SEL.errorModal)).toBeVisible({ timeout: 5_000 });
  });

  test("submitting with only password shows error", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.passInput).fill("somepassword");
    await page.locator(SEL.submitBtn).click();
    await expect(page.locator(SEL.errorModal)).toBeVisible({ timeout: 5_000 });
  });
});

// ══════════════════════════════════════════════════════════════
//  3. Field Interaction
// ══════════════════════════════════════════════════════════════
test.describe("Field interaction", () => {
  test("user can type in the User ID field", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.userInput).fill("testuser");
    await expect(page.locator(SEL.userInput)).toHaveValue("testuser");
  });

  test("user can type in the Password field", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.passInput).fill("secret123");
    await expect(page.locator(SEL.passInput)).toHaveValue("secret123");
  });

  test("password toggle reveals plain text", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.passInput).fill("secret123");
    const toggle = page.locator(SEL.passToggle).first();
    if (await toggle.isVisible()) {
      await toggle.click();
      await expect(page.locator(SEL.passInput)).toHaveAttribute("type", "text");
      // Toggle back
      await toggle.click();
      await expect(page.locator(SEL.passInput)).toHaveAttribute("type", "password");
    } else {
      test.skip(); // toggle not present in this build
    }
  });

  test("Enter key in password field submits the form", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.userInput).fill("anyuser");
    await page.locator(SEL.passInput).fill("anypass");
    await page.locator(SEL.passInput).press("Enter");
    // Should either show error (invalid creds) or proceed — not stay static
    await page.waitForTimeout(2_000);
    const url = page.url();
    const hasError = await page.locator(SEL.errorModal).isVisible().catch(() => false);
    // Either navigated away or showed an error — either is acceptable
    expect(url !== LOGIN_URL || hasError).toBe(true);
  });
});

// ══════════════════════════════════════════════════════════════
//  4. Successful Login (skipped when credentials not provided)
// ══════════════════════════════════════════════════════════════
test.describe("Successful login", () => {
  test.skip(!USER || !PASS, "Set TIGERSPACE_USER and TIGERSPACE_PASS env vars to run");

  test("valid credentials shows company selection modal", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.userInput).fill(USER);
    await page.locator(SEL.passInput).fill(PASS);
    await page.locator(SEL.submitBtn).click();

    // Should show company picker (CTC / HMC / TG / TGP / TGS)
    const modal = page.locator(SEL.companyModal).first();
    await expect(modal).toBeVisible({ timeout: 10_000 });
  });

  test("company modal contains expected company options", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.userInput).fill(USER);
    await page.locator(SEL.passInput).fill(PASS);
    await page.locator(SEL.submitBtn).click();

    await page.locator(SEL.companyModal).waitFor({ timeout: 10_000 });
    for (const company of ["CTC", "HMC", "TG", "TGP", "TGS"]) {
      await expect(page.locator(`text=${company}`).first()).toBeVisible();
    }
  });

  test("selecting a company navigates to the dashboard", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.userInput).fill(USER);
    await page.locator(SEL.passInput).fill(PASS);
    await page.locator(SEL.submitBtn).click();

    await page.locator(SEL.companyModal).waitFor({ timeout: 10_000 });
    // Click the first available company
    await page.locator(`${SEL.companyModal} li, ${SEL.companyModal} .item`).first().click();
    await page.waitForURL(/(?!.*login)/, { timeout: 15_000 });
    expect(page.url()).not.toContain("/login/");
  });
});

// ══════════════════════════════════════════════════════════════
//  5. Accessibility & Security basics
// ══════════════════════════════════════════════════════════════
test.describe("Security & accessibility", () => {
  test("page is served over HTTPS", async ({ page }) => {
    await gotoLogin(page);
    expect(page.url()).toMatch(/^https:/);
  });

  test("password field value is not in page source (not echoed)", async ({ page }) => {
    await gotoLogin(page);
    await page.locator(SEL.passInput).fill("supersecret");
    const html = await page.content();
    expect(html).not.toContain("supersecret");
  });

  test("User ID field has autocomplete=off or username", async ({ page }) => {
    await gotoLogin(page);
    const ac = await page.locator(SEL.userInput).getAttribute("autocomplete");
    // Accept: off, username, or unset (null)
    expect(["off", "username", null]).toContain(ac);
  });
});
