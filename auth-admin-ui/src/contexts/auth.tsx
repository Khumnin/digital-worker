"use client";

import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { authApi, decodeJwt, isTokenExpired, type JwtPayload } from "@/lib/api";

// ── Types ─────────────────────────────────────────────────────────────────────

interface AuthState {
  accessToken: string | null;
  refreshToken: string | null;
  user: JwtPayload | null;
  tenantId: string | null;
  tenantSlug: string;
  isAuthenticated: boolean;
  isSuperAdmin: boolean;
  isAdmin: boolean;
}

interface AuthContextValue extends AuthState {
  login: (accessToken: string, refreshToken: string, tenantSlug: string) => void;
  logout: () => Promise<void>;
  getToken: () => Promise<string | null>;
}

const REFRESH_TOKEN_KEY = "tgx_refresh_token";
const TENANT_SLUG_KEY = "tgx_tenant_slug";

// ── Context ───────────────────────────────────────────────────────────────────

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    accessToken: null,
    refreshToken: null,
    user: null,
    tenantId: null,
    tenantSlug: "platform",
    isAuthenticated: false,
    isSuperAdmin: false,
    isAdmin: false,
  });

  const refreshTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // ── Helpers ──────────────────────────────────────────────────────────────────

  const clearState = useCallback(() => {
    if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(TENANT_SLUG_KEY);
    setState({
      accessToken: null,
      refreshToken: null,
      user: null,
      tenantId: null,
      tenantSlug: "platform",
      isAuthenticated: false,
      isSuperAdmin: false,
      isAdmin: false,
    });
  }, []);

  const applyTokens = useCallback(
    (accessToken: string, refreshToken: string, tenantSlug: string) => {
      const payload = decodeJwt(accessToken);
      if (!payload) return;

      const roles = payload.roles ?? [];
      setState({
        accessToken,
        refreshToken,
        user: payload,
        tenantId: payload.tenant_id,
        tenantSlug,
        isAuthenticated: true,
        isSuperAdmin: roles.includes("super_admin"),
        isAdmin: roles.includes("admin") || roles.includes("super_admin"),
      });

      // Schedule refresh 60 seconds before expiry
      const expiresInMs = payload.exp * 1000 - Date.now() - 60_000;
      if (expiresInMs > 0) {
        if (refreshTimerRef.current) clearTimeout(refreshTimerRef.current);
        refreshTimerRef.current = setTimeout(async () => {
          try {
            const result = await authApi.refresh(refreshToken, tenantSlug);
            localStorage.setItem(REFRESH_TOKEN_KEY, result.refresh_token);
            applyTokens(result.access_token, result.refresh_token, tenantSlug);
          } catch {
            clearState();
          }
        }, Math.max(expiresInMs, 5_000));
      }
    },
    [clearState] // eslint-disable-line react-hooks/exhaustive-deps
  );

  // ── Restore session from localStorage on mount ─────────────────────────────

  useEffect(() => {
    const storedRefresh = localStorage.getItem(REFRESH_TOKEN_KEY);
    const storedSlug = localStorage.getItem(TENANT_SLUG_KEY) || "platform";
    if (!storedRefresh) return;

    authApi
      .refresh(storedRefresh, storedSlug)
      .then((result) => {
        localStorage.setItem(REFRESH_TOKEN_KEY, result.refresh_token);
        applyTokens(result.access_token, result.refresh_token, storedSlug);
      })
      .catch(() => {
        localStorage.removeItem(REFRESH_TOKEN_KEY);
        localStorage.removeItem(TENANT_SLUG_KEY);
      });
  }, [applyTokens]);

  // ── Public actions ────────────────────────────────────────────────────────────

  const login = useCallback(
    (accessToken: string, refreshToken: string, tenantSlug: string) => {
      localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
      localStorage.setItem(TENANT_SLUG_KEY, tenantSlug);
      applyTokens(accessToken, refreshToken, tenantSlug);
    },
    [applyTokens]
  );

  const logout = useCallback(async () => {
    if (state.accessToken) {
      try {
        await authApi.logout(state.accessToken);
      } catch {}
    }
    clearState();
  }, [state.accessToken, clearState]);

  const getToken = useCallback(async (): Promise<string | null> => {
    if (!state.accessToken) return null;
    if (!isTokenExpired(state.accessToken)) return state.accessToken;

    // Token expired — try refresh
    const storedRefresh = localStorage.getItem(REFRESH_TOKEN_KEY);
    const storedSlug = localStorage.getItem(TENANT_SLUG_KEY) || "platform";
    if (!storedRefresh) {
      clearState();
      return null;
    }
    try {
      const result = await authApi.refresh(storedRefresh, storedSlug);
      localStorage.setItem(REFRESH_TOKEN_KEY, result.refresh_token);
      applyTokens(result.access_token, result.refresh_token, storedSlug);
      return result.access_token;
    } catch {
      clearState();
      return null;
    }
  }, [state.accessToken, applyTokens, clearState]);

  return (
    <AuthContext.Provider value={{ ...state, login, logout, getToken }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
