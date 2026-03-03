/**
 * Sprint 9 — Smoke Tests: Full Auth Happy Path
 *
 * Exercises the complete happy-path flow in one pass:
 *   register → (pre-verification state) → login (negative) → forgot-password
 *   → refresh (negative) → logout (negative) → JWKS → introspect (negative)
 *   → tenant-less request (negative)
 *
 * These tests are intentionally lightweight. They verify the server is up and
 * that every critical endpoint responds correctly with no configuration. They
 * are the first suite to run in CI and the last gate before a production deploy.
 */
import { test, expect } from "@playwright/test";
import {
  register,
  login,
  refreshToken,
  logout,
  forgotPassword,
  introspectToken,
  healthCheck,
} from "./helpers/api";

const TENANT = process.env.TEST_TENANT ?? "test-tenant-1";
const BASE   = process.env.BASE_URL ?? "http://localhost:8080";
const API    = `${BASE}/api/v1`;

const uniqueEmail = () =>
  `e2e-${Date.now()}-${Math.random().toString(36).slice(2)}@example.com`;

const PASSWORD = "P@ssw0rd123!";

test.describe("Smoke Test — Full Auth Happy Path", () => {
  test("health check passes", async ({ request }) => {
    const res = await healthCheck(request);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.status).toBe("ok");
  });

  test("register new user returns 201 with user_id", async ({ request }) => {
    const res = await register(request, TENANT, uniqueEmail(), PASSWORD);
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body).toHaveProperty("user_id");
    // Must never leak the password hash in a registration response
    expect(JSON.stringify(body)).not.toMatch(/password_hash|argon2/i);
  });

  test("login before email verification does not return 200", async ({ request }) => {
    // Register a fresh user who has not verified their email
    const email = uniqueEmail();
    await register(request, TENANT, email, PASSWORD);

    // Unverified accounts must be blocked — the exact status code
    // (401, 403, or similar) may vary by implementation; 200 is never acceptable
    const res = await login(request, TENANT, email, PASSWORD);
    expect(res.status()).not.toBe(200);
  });

  test("login with wrong password returns 401", async ({ request }) => {
    const res = await login(request, TENANT, uniqueEmail(), "completely-wrong");
    expect(res.status()).toBe(401);
  });

  test("POST /auth/forgot-password returns 200 for registered email (anti-enumeration)", async ({ request }) => {
    const email = uniqueEmail();
    // Register first so the email exists in the tenant
    await register(request, TENANT, email, PASSWORD);

    const res = await forgotPassword(request, TENANT, email);
    expect(res.status()).toBe(200);
  });

  test("POST /auth/forgot-password returns 200 for unknown email (anti-enumeration)", async ({ request }) => {
    // This email was never registered — the response must be identical to avoid
    // user enumeration attacks
    const res = await forgotPassword(
      request,
      TENANT,
      `unknown-smoke-${Date.now()}@nowhere.dev`
    );
    expect(res.status()).toBe(200);
  });

  test("token refresh with invalid token returns 401", async ({ request }) => {
    const res = await refreshToken(request, TENANT, "garbage-refresh-token");
    expect(res.status()).toBe(401);
  });

  test("logout without auth returns 401", async ({ request }) => {
    const res = await logout(request, TENANT, "bad-access-token", "bad-refresh-token");
    expect(res.status()).toBe(401);
  });

  test("GET /.well-known/jwks.json returns keys array", async ({ request }) => {
    const res = await request.get(`${BASE}/.well-known/jwks.json`);
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body).toHaveProperty("keys");
    expect(Array.isArray(body.keys)).toBe(true);
    expect(body.keys.length).toBeGreaterThan(0);
  });

  test("POST /oauth/introspect with garbage token returns active:false", async ({ request }) => {
    const res = await introspectToken(request, "this.is.garbage");
    // RFC 7662 — introspect always returns 200; active:false means invalid
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.active).toBe(false);
  });

  test("request without X-Tenant-ID header returns 4xx", async ({ request }) => {
    // Any tenant-scoped endpoint must reject requests missing the tenant header
    const res = await request.post(`${API}/auth/login`, {
      data: { email: "any@example.com", password: PASSWORD },
    });
    expect([400, 401, 403]).toContain(res.status());
  });
});
