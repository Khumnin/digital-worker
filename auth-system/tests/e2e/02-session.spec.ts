/**
 * Sprint 2 — US-04 (Session Refresh) & US-05 (Logout)
 */
import { test, expect } from "@playwright/test";
import { refreshToken, logout } from "./helpers/api";

const TENANT = process.env.TEST_TENANT ?? "test-tenant-1";

test.describe("US-04 Session Refresh", () => {
  test("401 on invalid refresh token", async ({ request }) => {
    const res = await refreshToken(request, TENANT, "invalid-token");
    expect(res.status()).toBe(401);
  });

  test("401 on empty refresh token", async ({ request }) => {
    const res = await refreshToken(request, TENANT, "");
    expect([400, 401, 422]).toContain(res.status());
  });
});

test.describe("US-05 Logout", () => {
  test("401 logout without auth token", async ({ request }) => {
    const res = await logout(request, TENANT, "bad-token", "bad-refresh");
    expect(res.status()).toBe(401);
  });
});
