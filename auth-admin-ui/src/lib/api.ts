const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public code?: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface LoginResponse {
  access_token: string;
  refresh_token: string;
  token_type: string;
  expires_in: number;
}

export interface JwtPayload {
  sub: string;
  email: string;
  tenant_id: string;
  roles: string[];
  exp: number;
  iat: number;
  iss: string;
}

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  status: "active" | "suspended" | "pending";
  schema_name: string;
  config: TenantConfig;
  created_at: string;
  updated_at: string;
}

export interface TenantConfig {
  mfa_required?: boolean;
  allowed_domains?: string[];
  session_duration_minutes?: number;
  enabled_modules?: string[];
}

export interface CreateTenantRequest {
  name: string;
  slug: string;
  config?: TenantConfig;
}

export interface User {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  status: "active" | "inactive" | "pending";
  email_verified_at?: string;
  mfa_enabled: boolean;
  roles: Role[];
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  created_at: string;
}

export interface CreateRoleRequest {
  name: string;
  description: string;
}

export interface InviteUserRequest {
  email: string;
  first_name: string;
  last_name: string;
  role_ids: string[];
}

export interface AuditLog {
  id: string;
  action: string;
  actor_id?: string;
  actor_email?: string;
  target_id?: string;
  target_type?: string;
  metadata?: Record<string, unknown>;
  ip_address?: string;
  created_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface DashboardStats {
  total_tenants: number;
  active_tenants: number;
  total_users: number;
  active_users: number;
  recent_logins_24h: number;
}

// ── JWT helpers ──────────────────────────────────────────────────────────────

export function decodeJwt(token: string): JwtPayload | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = JSON.parse(atob(parts[1].replace(/-/g, "+").replace(/_/g, "/")));
    return payload as JwtPayload;
  } catch {
    return null;
  }
}

export function isTokenExpired(token: string): boolean {
  const payload = decodeJwt(token);
  if (!payload) return true;
  return payload.exp * 1000 < Date.now();
}

// ── Fetch wrapper ─────────────────────────────────────────────────────────────

interface FetchOptions {
  method?: string;
  body?: unknown;
  token?: string;
  tenantId?: string;
}

async function apiFetch<T>(path: string, opts: FetchOptions = {}): Promise<T> {
  const { method = "GET", body, token, tenantId } = opts;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;
  if (tenantId) headers["X-Tenant-ID"] = tenantId;

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
    cache: "no-store",
  });

  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    let code: string | undefined;
    try {
      const err = await res.json();
      message = err.message || err.error || message;
      code = err.code;
    } catch {}
    throw new ApiError(res.status, message, code);
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

// ── Auth endpoints ────────────────────────────────────────────────────────────

export const authApi = {
  login: (data: LoginRequest, tenantSlug: string) =>
    apiFetch<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: data,
      tenantId: tenantSlug,
    }),

  refresh: (refreshToken: string, tenantId: string) =>
    apiFetch<LoginResponse>("/api/v1/auth/refresh", {
      method: "POST",
      body: { refresh_token: refreshToken },
      tenantId,
    }),

  logout: (token: string, tenantId: string) =>
    apiFetch<void>("/api/v1/auth/logout", {
      method: "POST",
      token,
      tenantId,
    }),
};

// ── Tenant admin endpoints ────────────────────────────────────────────────────

export const tenantApi = {
  list: (token: string) =>
    apiFetch<PaginatedResponse<Tenant>>("/api/v1/admin/tenants", { token }),

  get: (id: string, token: string) =>
    apiFetch<Tenant>(`/api/v1/admin/tenants/${id}`, { token }),

  create: (data: CreateTenantRequest, token: string) =>
    apiFetch<Tenant>("/api/v1/admin/tenants", {
      method: "POST",
      body: data,
      token,
    }),

  update: (id: string, data: Partial<CreateTenantRequest>, token: string) =>
    apiFetch<Tenant>(`/api/v1/admin/tenants/${id}`, {
      method: "PATCH",
      body: data,
      token,
    }),

  suspend: (id: string, token: string) =>
    apiFetch<void>(`/api/v1/admin/tenants/${id}/suspend`, {
      method: "POST",
      token,
    }),

  activate: (id: string, token: string) =>
    apiFetch<void>(`/api/v1/admin/tenants/${id}/activate`, {
      method: "POST",
      token,
    }),
};

// ── User admin endpoints ──────────────────────────────────────────────────────

export const userApi = {
  list: (token: string, tenantId: string, params?: { page?: number; page_size?: number; role?: string }) => {
    const qs = params
      ? "?" + new URLSearchParams(Object.entries(params).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])).toString()
      : "";
    return apiFetch<PaginatedResponse<User>>(`/api/v1/admin/users${qs}`, { token, tenantId });
  },

  get: (id: string, token: string, tenantId: string) =>
    apiFetch<User>(`/api/v1/admin/users/${id}`, { token, tenantId }),

  invite: (data: InviteUserRequest, token: string, tenantId: string) =>
    apiFetch<User>("/api/v1/admin/users/invite", {
      method: "POST",
      body: data,
      token,
      tenantId,
    }),

  updateRoles: (userId: string, roleIds: string[], token: string, tenantId: string) =>
    apiFetch<void>(`/api/v1/admin/users/${userId}/roles`, {
      method: "PUT",
      body: { role_ids: roleIds },
      token,
      tenantId,
    }),

  suspend: (id: string, token: string, tenantId: string) =>
    apiFetch<void>(`/api/v1/admin/users/${id}/suspend`, {
      method: "POST",
      token,
      tenantId,
    }),
};

// ── Role admin endpoints ──────────────────────────────────────────────────────

export const roleApi = {
  list: (token: string, tenantId: string) =>
    apiFetch<Role[]>("/api/v1/admin/roles", { token, tenantId }),

  create: (data: CreateRoleRequest, token: string, tenantId: string) =>
    apiFetch<Role>("/api/v1/admin/roles", {
      method: "POST",
      body: data,
      token,
      tenantId,
    }),

  delete: (id: string, token: string, tenantId: string) =>
    apiFetch<void>(`/api/v1/admin/roles/${id}`, {
      method: "DELETE",
      token,
      tenantId,
    }),
};

// ── Audit log endpoints ───────────────────────────────────────────────────────

export const auditApi = {
  list: (token: string, tenantId: string, params?: { page?: number; page_size?: number; action?: string; actor_id?: string }) => {
    const qs = params
      ? "?" + new URLSearchParams(Object.entries(params).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])).toString()
      : "";
    return apiFetch<PaginatedResponse<AuditLog>>(`/api/v1/admin/audit-log${qs}`, { token, tenantId });
  },
};
