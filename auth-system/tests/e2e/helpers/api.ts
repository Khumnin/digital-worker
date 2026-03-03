import { APIRequestContext } from "@playwright/test";

const BASE = process.env.BASE_URL ?? "http://localhost:8080";
const API   = `${BASE}/api/v1`;

// ── Tenant header ──────────────────────────────────────────────
export function tenantHeaders(tenantId: string, token?: string) {
  return {
    "X-Tenant-ID": tenantId,
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  };
}

// ── Auth helpers ───────────────────────────────────────────────
export async function register(
  req: APIRequestContext,
  tenantId: string,
  email: string,
  password: string,
  firstName = "Test",
  lastName = "User"
) {
  return req.post(`${API}/auth/register`, {
    headers: tenantHeaders(tenantId),
    data: { email, password, first_name: firstName, last_name: lastName },
  });
}

export async function login(
  req: APIRequestContext,
  tenantId: string,
  email: string,
  password: string
) {
  return req.post(`${API}/auth/login`, {
    headers: tenantHeaders(tenantId),
    data: { email, password },
  });
}

export async function refreshToken(
  req: APIRequestContext,
  tenantId: string,
  refreshToken: string
) {
  return req.post(`${API}/auth/token/refresh`, {
    headers: tenantHeaders(tenantId),
    data: { refresh_token: refreshToken },
  });
}

export async function logout(
  req: APIRequestContext,
  tenantId: string,
  accessToken: string,
  refreshToken: string,
  all = false
) {
  const path = all ? "/auth/logout/all" : "/auth/logout";
  return req.post(`${API}${path}`, {
    headers: tenantHeaders(tenantId, accessToken),
    data: { refresh_token: refreshToken },
  });
}

export async function forgotPassword(
  req: APIRequestContext,
  tenantId: string,
  email: string
) {
  return req.post(`${API}/auth/forgot-password`, {
    headers: tenantHeaders(tenantId),
    data: { email },
  });
}

// ── Admin helpers ──────────────────────────────────────────────
export async function inviteUser(
  req: APIRequestContext,
  tenantId: string,
  adminToken: string,
  email: string,
  roleId?: string
) {
  return req.post(`${API}/admin/users/invite`, {
    headers: tenantHeaders(tenantId, adminToken),
    data: { email, ...(roleId ? { role_id: roleId } : {}) },
  });
}

export async function listUsers(
  req: APIRequestContext,
  tenantId: string,
  adminToken: string
) {
  return req.get(`${API}/admin/users`, {
    headers: tenantHeaders(tenantId, adminToken),
  });
}

// ── OAuth helpers ──────────────────────────────────────────────

/**
 * POST /api/v1/admin/oauth/clients
 * Registers a new OAuth client. Requires an admin bearer token.
 * Returns the created client including client_id and client_secret.
 */
export async function registerOAuthClient(
  req: APIRequestContext,
  tenantId: string,
  adminToken: string,
  name: string,
  redirectUris: string[],
  scopes?: string[]
) {
  return req.post(`${API}/admin/oauth/clients`, {
    headers: tenantHeaders(tenantId, adminToken),
    data: {
      name,
      redirect_uris: redirectUris,
      ...(scopes ? { scopes } : {}),
    },
  });
}

/**
 * POST /api/v1/oauth/introspect
 * Introspects a token per RFC 7662. No tenant header required.
 * Always returns 200; active:false means the token is invalid/expired.
 */
export async function introspectToken(req: APIRequestContext, token: string) {
  return req.post(`${API}/oauth/introspect`, {
    data: { token },
  });
}

/**
 * POST /api/v1/oauth/token  grant_type=client_credentials
 * Obtains an M2M access token using client credentials.
 */
export async function m2mToken(
  req: APIRequestContext,
  tenantId: string,
  clientId: string,
  clientSecret: string,
  scope?: string
) {
  return req.post(`${API}/oauth/token`, {
    headers: { "X-Tenant-ID": tenantId },
    data: {
      grant_type: "client_credentials",
      client_id: clientId,
      client_secret: clientSecret,
      ...(scope ? { scope } : {}),
    },
  });
}

// ── MFA helpers ────────────────────────────────────────────────

/**
 * POST /api/v1/users/me/mfa/generate
 * Generates a TOTP secret for the authenticated user.
 * Returns otp_auth_url and secret. Does NOT persist until /mfa/confirm is called.
 */
export async function mfaGenerate(
  req: APIRequestContext,
  tenantId: string,
  accessToken: string
) {
  return req.post(`${API}/users/me/mfa/generate`, {
    headers: tenantHeaders(tenantId, accessToken),
  });
}

/**
 * DELETE /api/v1/users/me/mfa
 * Disables TOTP MFA for the authenticated user.
 * Requires the user's current password for confirmation.
 */
export async function mfaDisable(
  req: APIRequestContext,
  tenantId: string,
  accessToken: string,
  password: string
) {
  return req.delete(`${API}/users/me/mfa`, {
    headers: tenantHeaders(tenantId, accessToken),
    data: { password },
  });
}

// ── Profile helpers ────────────────────────────────────────────

/**
 * GET /api/v1/users/me
 * Returns the authenticated user's full profile.
 */
export async function getMe(
  req: APIRequestContext,
  tenantId: string,
  accessToken: string
) {
  return req.get(`${API}/users/me`, {
    headers: tenantHeaders(tenantId, accessToken),
  });
}

/**
 * PUT /api/v1/users/me
 * Updates the authenticated user's profile.
 * Dispatches based on supplied fields:
 *   first_name / last_name          → display name update
 *   current_password + new_password → password change
 *   new_email                       → email change request
 */
export async function updateMe(
  req: APIRequestContext,
  tenantId: string,
  accessToken: string,
  body: Record<string, unknown>
) {
  return req.put(`${API}/users/me`, {
    headers: tenantHeaders(tenantId, accessToken),
    data: body,
  });
}

/**
 * DELETE /api/v1/users/me
 * GDPR self-erasure. Requires current password for confirmation.
 * Returns 204 on success.
 */
export async function deleteMe(
  req: APIRequestContext,
  tenantId: string,
  accessToken: string,
  password: string
) {
  return req.delete(`${API}/users/me`, {
    headers: tenantHeaders(tenantId, accessToken),
    data: { password },
  });
}

// ── Admin user management helpers ─────────────────────────────

/**
 * POST /api/v1/admin/users/:id/roles
 * Assigns a role to a user. Requires an admin bearer token.
 */
export async function assignRole(
  req: APIRequestContext,
  tenantId: string,
  adminToken: string,
  userId: string,
  roleId: string
) {
  return req.post(`${API}/admin/users/${userId}/roles`, {
    headers: tenantHeaders(tenantId, adminToken),
    data: { role_id: roleId },
  });
}

/**
 * DELETE /api/v1/admin/users/:id
 * GDPR admin erasure of a user. Returns 204 on success.
 */
export async function eraseUser(
  req: APIRequestContext,
  tenantId: string,
  adminToken: string,
  userId: string
) {
  return req.delete(`${API}/admin/users/${userId}`, {
    headers: tenantHeaders(tenantId, adminToken),
  });
}

// ── Health ─────────────────────────────────────────────────────

/**
 * GET /health
 * Dependency-aware health check. No tenant header required.
 * Returns 200 {"status":"ok","checks":{...}} or 503 {"status":"degraded",...}.
 */
export async function healthCheck(req: APIRequestContext) {
  return req.get(`${BASE}/health`);
}
