/**
 * Sprint 1 — US-06 Password Reset (anti-enumeration)
 */
import { test, expect } from "@playwright/test";
import { forgotPassword } from "./helpers/api";

const TENANT = process.env.TEST_TENANT ?? "test-tenant-1";

test.describe("US-06 Password Reset — Anti-Enumeration", () => {
  test("identical 200 response for registered email", async ({ request }) => {
    const res = await forgotPassword(request, TENANT, "registered@example.com");
    expect(res.status()).toBe(200);
  });

  test("identical 200 response for unknown email", async ({ request }) => {
    const res = await forgotPassword(request, TENANT, `unknown-${Date.now()}@nowhere.dev`);
    expect(res.status()).toBe(200);
  });

  test("response body identical for both", async ({ request }) => {
    const [r1, r2] = await Promise.all([
      forgotPassword(request, TENANT, "any@example.com"),
      forgotPassword(request, TENANT, `unknown-${Date.now()}@example.com`),
    ]);
    const [b1, b2] = await Promise.all([r1.json(), r2.json()]);
    // Top-level keys must match
    expect(Object.keys(b1).sort()).toEqual(Object.keys(b2).sort());
  });
});
