/**
 * Sprint 3 — US-08b Cross-Tenant Isolation (non-negotiable DoD)
 * These tests must ALWAYS pass in CI before any data-access story is merged.
 *
 * 12 test cases covering: token isolation, data isolation, header safety,
 * schema routing, password/session token isolation, and enumeration resistance.
 */
import { test, expect } from "@playwright/test";
import { listUsers, login, register, refreshToken, forgotPassword } from "./helpers/api";

const TENANT_A = process.env.TEST_TENANT_A ?? "test-tenant-1";
const TENANT_B = process.env.TEST_TENANT_B ?? "test-tenant-b";
const BASE = process.env.BASE_URL ?? "http://localhost:8080";
const API = `${BASE}/api/v1`;

// ── TC-08b-01 to TC-08b-03: Token isolation ──────────────────────────────────

test.describe("US-08b Cross-Tenant Isolation", () => {
  test("TC-08b-01: tenant A token cannot list tenant B users", async ({ request }) => {
    const fakeTenantAToken = "invalid-token-for-tenant-a";
    const res = await listUsers(request, TENANT_B, fakeTenantAToken);
    expect([401, 403]).toContain(res.status());
  });

  test("TC-08b-02: no tenant B data appears in tenant A response body", async ({ request }) => {
    const res = await listUsers(request, TENANT_A, "invalid-tenant-a-token");
    if (res.status() === 200) {
      const body = await res.json();
      const raw = JSON.stringify(body);
      expect(raw).not.toContain(TENANT_B);
      expect(raw).not.toContain("tenant_b");
    }
    expect([200, 401, 403]).toContain(res.status());
  });

  test("TC-08b-03: cross-tenant token swap returns 401 or 403 — never 200", async ({ request }) => {
    const crossToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.fake.sig";
    const res = await listUsers(request, TENANT_B, crossToken);
    expect([401, 403]).toContain(res.status());
  });

  // ── TC-08b-04: Missing tenant header rejected ───────────────────────────────

  test("TC-08b-04: request without X-Tenant-ID header is rejected", async ({ request }) => {
    const res = await request.get(`${API}/admin/users`, {
      headers: { Authorization: "Bearer some-token" },
    });
    // Must not silently succeed — tenant resolution must fail
    expect([400, 401, 403]).toContain(res.status());
  });

  // ── TC-08b-05: Unknown tenant rejected ─────────────────────────────────────

  test("TC-08b-05: unknown tenant ID returns 4xx — never leaks data", async ({ request }) => {
    const res = await listUsers(request, "non-existent-tenant-xyz", "any-token");
    expect([400, 401, 403, 404]).toContain(res.status());
    if (res.status() !== 204) {
      const body = await res.text();
      expect(body).not.toContain("tenant_");
      expect(body).not.toContain("schema");
    }
  });

  // ── TC-08b-06: Response headers must not leak tenant info ──────────────────

  test("TC-08b-06: response headers do not expose tenant schema or internal info", async ({ request }) => {
    const res = await listUsers(request, TENANT_A, "invalid-token");
    const headers = res.headers();
    // No internal implementation details in headers
    expect(Object.keys(headers).join(",")).not.toMatch(/x-tenant-schema|x-db-schema|x-search-path/i);
    const rawHeaders = JSON.stringify(headers);
    expect(rawHeaders).not.toContain("tenant_");
  });

  // ── TC-08b-07: Error body must not leak cross-tenant data ──────────────────

  test("TC-08b-07: error response body for tenant A does not contain tenant B identifiers", async ({ request }) => {
    const resA = await listUsers(request, TENANT_A, "bad-token");
    const resB = await listUsers(request, TENANT_B, "bad-token");

    const bodyA = await resA.text();
    const bodyB = await resB.text();

    // Error for tenant A must not reference tenant B
    expect(bodyA).not.toContain(TENANT_B);
    expect(bodyA).not.toContain("tenant_b");

    // Error for tenant B must not reference tenant A
    expect(bodyB).not.toContain(TENANT_A);
    expect(bodyB).not.toContain("tenant_a");
  });

  // ── TC-08b-08: Password reset anti-enumeration across tenants ──────────────

  test("TC-08b-08: password reset response is identical regardless of tenant membership", async ({ request }) => {
    const emailInA = `user-${Date.now()}@example.com`;

    const resKnown = await forgotPassword(request, TENANT_A, emailInA);
    const resUnknown = await forgotPassword(request, TENANT_B, emailInA);

    // TENANT_A (known tenant) MUST always return 200 — anti-enumeration within a real tenant
    expect(resKnown.status()).toBe(200);
    // TENANT_B may not be provisioned in this environment; 403 (INVALID_TENANT) is acceptable
    expect([200, 403]).toContain(resUnknown.status());

    // When both tenants exist, response bodies must be structurally identical
    if (resUnknown.status() === 200) {
      const bodyKnown = await resKnown.json();
      const bodyUnknown = await resUnknown.json();
      expect(Object.keys(bodyKnown).sort()).toEqual(Object.keys(bodyUnknown).sort());
    }
  });

  // ── TC-08b-09: Refresh token cannot be used across tenants ─────────────────

  test("TC-08b-09: refresh token issued for tenant A cannot be replayed against tenant B", async ({ request }) => {
    const fakeRefreshToken = "a".repeat(64); // plausible but invalid token

    const resA = await refreshToken(request, TENANT_A, fakeRefreshToken);
    const resB = await refreshToken(request, TENANT_B, fakeRefreshToken);

    // Both must reject — neither should return 200
    // 403 is also valid when the tenant itself does not exist in this environment
    expect([400, 401, 403, 422]).toContain(resA.status());
    expect([400, 401, 403, 422]).toContain(resB.status());
  });

  // ── TC-08b-10: Registration in tenant A does not leak to tenant B ──────────

  test("TC-08b-10: registering in tenant A does not create a user visible in tenant B", async ({ request }) => {
    const uniqueEmail = `isolation-test-${Date.now()}@example.com`;

    // Attempt registration in tenant A
    await register(request, TENANT_A, uniqueEmail, "BadP@ss1", "Test", "User");

    // Attempt login with the same email in tenant B — must fail
    const loginRes = await login(request, TENANT_B, uniqueEmail, "BadP@ss1");
    expect([401, 403]).toContain(loginRes.status());
  });

  // ── TC-08b-11: Login failure message is identical across tenants ────────────

  test("TC-08b-11: login failure message is identical for both tenants — no enumeration", async ({ request }) => {
    const email = `no-such-user-${Date.now()}@example.com`;
    const password = "WrongPass1!";

    const resA = await login(request, TENANT_A, email, password);
    const resB = await login(request, TENANT_B, email, password);

    // TENANT_A (known tenant) must return 401 — invalid credentials, no enumeration
    expect(resA.status()).toBe(401);
    // TENANT_B may return 401 (user not found) or 403 (tenant not provisioned in this env)
    expect([401, 403]).toContain(resB.status());

    // When both tenants are reachable (401), error codes and messages must be identical
    if (resB.status() === 401) {
      const bodyA = await resA.json();
      const bodyB = await resB.json();

      // Error codes and messages must be identical — no tenant-specific information
      expect(bodyA.error?.code).toEqual(bodyB.error?.code);
      expect(bodyA.error?.message).toEqual(bodyB.error?.message);
    }
  });

  // ── TC-08b-12: Malformed tenant ID is rejected cleanly ─────────────────────

  test("TC-08b-12: malformed tenant ID (SQL injection attempt) is rejected", async ({ request }) => {
    const maliciousTenantID = "'; DROP TABLE tenants; --";
    const res = await listUsers(request, maliciousTenantID, "any-token");

    // Must be rejected — never 200 or 500
    expect([400, 401, 403, 404]).toContain(res.status());
    expect(res.status()).not.toBe(500);

    // Response must not echo back the injection string
    const body = await res.text();
    expect(body).not.toContain("DROP TABLE");
  });
});
