/**
 * Sprint 1 — US-01 (Registration) & US-03 (Login)
 * Runs in API mode against a live backend (no browser).
 */
import { test, expect } from "@playwright/test";
import { register, login } from "./helpers/api";

const TENANT = process.env.TEST_TENANT ?? "test-tenant-1";
const uniqueEmail = () => `e2e-${Date.now()}@example.com`;
const PASSWORD = "P@ssw0rd123!";

// ── US-01: User Registration ───────────────────────────────────
test.describe("US-01 Registration", () => {
  test("201 on valid registration", async ({ request }) => {
    const res = await register(request, TENANT, uniqueEmail(), PASSWORD);
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body).toHaveProperty("user_id");
    expect(body).not.toHaveProperty("password");
    expect(body).not.toHaveProperty("password_hash");
  });

  test("409 on duplicate email", async ({ request }) => {
    const email = uniqueEmail();
    await register(request, TENANT, email, PASSWORD);
    const res = await register(request, TENANT, email, PASSWORD);
    expect(res.status()).toBe(409);
  });

  test("422 on weak password", async ({ request }) => {
    const res = await register(request, TENANT, uniqueEmail(), "short");
    expect(res.status()).toBe(422);
  });

  test("422 on invalid email format", async ({ request }) => {
    const res = await register(request, TENANT, "not-an-email", PASSWORD);
    expect(res.status()).toBe(422);
  });
});

// ── US-03: Login ───────────────────────────────────────────────
test.describe("US-03 Login", () => {
  test("401 on wrong credentials — ambiguous error message", async ({ request }) => {
    const res = await login(request, TENANT, uniqueEmail(), "wrong");
    expect(res.status()).toBe(401);
    const body = await res.json();
    // Must NOT distinguish "wrong email" from "wrong password"
    const msg: string = body?.error?.message ?? "";
    expect(msg.toLowerCase()).not.toMatch(/email not found|unknown email/);
    expect(msg.toLowerCase()).not.toMatch(/wrong password|incorrect password/);
  });

  test("response contains no sensitive data on failure", async ({ request }) => {
    const res = await login(request, TENANT, uniqueEmail(), "wrong");
    const body = await res.json();
    expect(JSON.stringify(body)).not.toMatch(/password_hash|argon2/i);
  });
});
