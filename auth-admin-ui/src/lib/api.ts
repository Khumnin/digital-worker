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
  enabled_modules: string[];
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
  admin_email: string;
  config?: TenantConfig;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  status: "active" | "inactive" | "pending";
  system_roles: string[];
  module_roles: Record<string, string[]>;
  tenant_id: string;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  module: string | null;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateRoleRequest {
  name: string;
  description: string;
  module?: string;
}

export interface UpdateUserRolesRequest {
  system_roles: string[];
  module_roles: Record<string, string[]>;
}

export interface InviteUserRequest {
  email: string;
  display_name: string;
  initial_role?: string;
}

export interface AuditLog {
  id: string;
  action: string;
  actor_id: string;
  actor_email: string;
  ip_address: string;
  target_id: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface DashboardStats {
  total_tenants: number;
  active_tenants: number;
  total_users: number;
  active_users: number;
  recent_logins_24h: number;
}

export interface TenantSettings {
  id: string;
  name: string;
  slug: string;
  status: string;
  enabled_modules: string[];
  mfa_required: boolean;
  session_duration_hours: number;
  allowed_domains: string[];
  created_at: string;
  updated_at: string;
}

export interface TenantSettingsUpdate {
  mfa_required?: boolean;
  session_duration_hours?: number;
  allowed_domains?: string[];
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
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
    // Do NOT send X-Tenant-ID for authenticated requests — the middleware
    // extracts the tenant from the JWT itself. Sending it causes TENANT_MISMATCH.
  } else if (tenantId) {
    // Only send X-Tenant-ID for unauthenticated requests (e.g. login).
    headers["X-Tenant-ID"] = tenantId;
  }

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
      // Backend returns {"error": {"code": "...", "message": "..."}}
      message = err.error?.message || err.message || err.error || message;
      code = err.error?.code || err.code;
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

  acceptInvite: (token: string, password: string, tenantSlug: string) =>
    apiFetch<{ message: string }>("/api/v1/auth/accept-invite", {
      method: "POST",
      body: { token, password },
      tenantId: tenantSlug,
    }),

  refresh: (refreshToken: string, tenantSlug: string) =>
    apiFetch<LoginResponse>("/api/v1/auth/token/refresh", {
      method: "POST",
      body: { refresh_token: refreshToken },
      tenantId: tenantSlug,
    }),

  logout: (token: string) =>
    apiFetch<void>("/api/v1/auth/logout", {
      method: "POST",
      token,
    }),

  forgotPassword: (email: string, tenantSlug: string) =>
    apiFetch<{ message: string }>("/api/v1/auth/forgot-password", {
      method: "POST",
      body: { email },
      tenantId: tenantSlug,
    }),

  resetPassword: (token: string, password: string, tenantSlug: string) =>
    apiFetch<{ message: string }>("/api/v1/auth/reset-password", {
      method: "POST",
      body: { token, password },
      tenantId: tenantSlug,
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
  list: (token: string, params?: { page?: number; page_size?: number; role?: string; status?: string; module?: string }) => {
    const qs = params
      ? "?" + new URLSearchParams(Object.entries(params).filter(([, v]) => v !== undefined && v !== "").map(([k, v]) => [k, String(v)])).toString()
      : "";
    return apiFetch<PaginatedResponse<User>>(`/api/v1/admin/users${qs}`, { token });
  },

  get: (id: string, token: string) =>
    apiFetch<User>(`/api/v1/admin/users/${id}`, { token }),

  invite: (data: InviteUserRequest, token: string) =>
    apiFetch<User>("/api/v1/admin/users/invite", {
      method: "POST",
      body: data,
      token,
    }),

  updateRoles: (userId: string, data: UpdateUserRolesRequest, token: string) =>
    apiFetch<void>(`/api/v1/admin/users/${userId}/roles`, {
      method: "PUT",
      body: data,
      token,
    }),

  suspend: (id: string, token: string) =>
    apiFetch<void>(`/api/v1/admin/users/${id}/disable`, {
      method: "POST",
      token,
    }),

  resendInvite: (id: string, token: string) =>
    apiFetch<{ message: string }>(`/api/v1/admin/users/${id}/resend-invite`, {
      method: "POST",
      token,
    }),
};

// ── Role admin endpoints ──────────────────────────────────────────────────────

export const roleApi = {
  list: (token: string, params?: { module?: string }) => {
    const qs = params?.module
      ? "?" + new URLSearchParams({ module: params.module }).toString()
      : "";
    return apiFetch<Role[]>(`/api/v1/admin/roles${qs}`, { token });
  },

  create: (data: CreateRoleRequest, token: string) =>
    apiFetch<Role>("/api/v1/admin/roles", {
      method: "POST",
      body: data,
      token,
    }),

  delete: (id: string, token: string) =>
    apiFetch<void>(`/api/v1/admin/roles/${id}`, {
      method: "DELETE",
      token,
    }),
};

// ── Audit log endpoints ───────────────────────────────────────────────────────

export const auditApi = {
  list: (token: string, params?: { page?: number; page_size?: number; action?: string; actor_id?: string; from?: string; to?: string }) => {
    const qs = params
      ? "?" + new URLSearchParams(Object.entries(params).filter(([, v]) => v !== undefined).map(([k, v]) => [k, String(v)])).toString()
      : "";
    return apiFetch<PaginatedResponse<AuditLog>>(`/api/v1/admin/audit-log${qs}`, { token });
  },
};

// ── User self-service (me) endpoints ─────────────────────────────────────────

export interface MeProfile {
  user_id: string;
  email: string;
  first_name: string;
  last_name: string;
  mfa_enabled: boolean;
  created_at: string;
  tenant_id: string;
  roles: string[];
}

export const meApi = {
  get: (token: string) =>
    apiFetch<MeProfile>("/api/v1/users/me", { token }),

  updateProfile: (token: string, firstName: string, lastName: string) =>
    apiFetch<MeProfile>("/api/v1/users/me", {
      method: "PUT",
      token,
      body: { first_name: firstName, last_name: lastName },
    }),

  changePassword: (token: string, currentPassword: string, newPassword: string) =>
    apiFetch<{ message: string }>("/api/v1/users/me", {
      method: "PUT",
      token,
      body: { current_password: currentPassword, new_password: newPassword },
    }),
};

// ── Settings endpoints ────────────────────────────────────────────────────────

export const settingsApi = {
  get: (token: string) =>
    apiFetch<TenantSettings>("/api/v1/admin/tenant", { token }),

  update: (token: string, data: Partial<TenantSettingsUpdate>) =>
    apiFetch<TenantSettings>("/api/v1/admin/tenant", {
      method: "PUT",
      token,
      body: data,
    }),
};
