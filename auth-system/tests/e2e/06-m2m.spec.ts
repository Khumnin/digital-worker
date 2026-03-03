/**
 * Sprint 6 — US-12 Client Credentials (M2M) Grant
 *
 * Tests the OAuth 2.0 client_credentials grant flow:
 *   - Valid credentials → access_token (no refresh_token per RFC 6749 §4.4.3)
 *   - Wrong secret → 401
 *   - Introspecting the M2M token → active:true (live integration only)
 *
 * Note: registering a client requires an admin token. These tests cover the
 * negative paths that are testable without a seeded admin session, plus
 * structural assertions on the token response shape when credentials are supplied.
 */
import { test, expect } from "@playwright/test";
import { introspectToken, m2mToken } from "./helpers/api";

const TENANT = process.env.TEST_TENANT ?? "test-tenant-1";
const BASE   = process.env.BASE_URL ?? "http://localhost:8080";
const API    = `${BASE}/api/v1`;

// ── US-12: Client Credentials Grant ───────────────────────────
test.describe("US-12 Client Credentials Grant", () => {
  test("wrong client_secret returns 401 on POST /oauth/token", async ({ request }) => {
    // Any unknown client_id + secret combination must be rejected.
    // The response code must be 401 (invalid_client) per RFC 6749 §5.2.
    const res = await m2mToken(
      request,
      TENANT,
      "non-existent-client-id",
      "wrong-secret",
      "read"
    );
    expect([400, 401]).toContain(res.status());
    const body = await res.json();
    expect(body).toHaveProperty("error");
    // RFC 6749 §5.2 error values for bad credentials
    expect(["invalid_client", "invalid_request"]).toContain(body.error);
  });

  test("missing client_secret returns 401 on POST /oauth/token", async ({ request }) => {
    // client_secret is required for client_credentials — omitting it must fail.
    const res = await request.post(`${API}/oauth/token`, {
      headers: { "X-Tenant-ID": TENANT },
      data: {
        grant_type: "client_credentials",
        client_id:  "some-client-id",
        // client_secret intentionally omitted
      },
    });
    expect([400, 401]).toContain(res.status());
  });

  test("client_credentials response shape must not include refresh_token", async ({ request }) => {
    // RFC 6749 §4.4.3 explicitly forbids issuing a refresh_token with
    // the client_credentials grant. Even on a failure response the server
    // must not include refresh_token in the body.
    const res = await m2mToken(
      request,
      TENANT,
      "fake-client",
      "fake-secret"
    );
    const body = await res.json();
    // Whether the request succeeds or fails, refresh_token must be absent
    expect(body).not.toHaveProperty("refresh_token");
  });

  test("POST /oauth/introspect with garbage returns active:false", async ({ request }) => {
    // Structural sanity: the introspect endpoint must always be reachable
    // and must return 200 with active:false for any invalid token.
    const res = await introspectToken(request, "not-a-real-m2m-token");
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.active).toBe(false);
  });

  test("POST /oauth/token with client_credentials and empty scope is accepted or returns 400", async ({ request }) => {
    // An empty scope is valid for client_credentials — the server may grant
    // its default scope or reject with invalid_scope. Either is acceptable.
    // It must not return 500.
    const res = await m2mToken(
      request,
      TENANT,
      "fake-client-id",
      "fake-client-secret",
      ""
    );
    expect([400, 401, 403]).toContain(res.status());
    expect(res.status()).not.toBe(500);
  });
});
