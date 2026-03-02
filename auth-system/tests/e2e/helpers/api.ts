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
