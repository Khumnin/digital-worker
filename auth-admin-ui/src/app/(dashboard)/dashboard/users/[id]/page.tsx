"use client";

import { useEffect, useState, useMemo } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft, Loader2, Shield, Send, KeyRound } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useAuth } from "@/contexts/auth";
import { userApi, roleApi, authApi, type User, type Role, ApiError } from "@/lib/api";

export default function UserDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { getToken, isAdmin, isSuperAdmin, tenantSlug, user: authUser } = useAuth();
  const [user, setUser] = useState<User | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingRoles, setSavingRoles] = useState(false);
  const [resending, setResending] = useState(false);
  const [sendingReset, setSendingReset] = useState(false);

  // Selected system role names (string set)
  const [selectedSystemRoles, setSelectedSystemRoles] = useState<string[]>([]);
  // Selected module roles: Record<module, Set<roleName>>
  const [selectedModuleRoles, setSelectedModuleRoles] = useState<Record<string, string[]>>({});

  const id = params?.id as string;

  useEffect(() => {
    load();
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const [u, allRoles] = await Promise.all([
        userApi.get(id, token),
        roleApi.list(token),
      ]);
      setUser(u);
      setRoles(allRoles);
      // Pre-populate selections from user data
      setSelectedSystemRoles(u.system_roles ? [...u.system_roles] : []);
      setSelectedModuleRoles(
        Object.fromEntries(
          Object.entries(u.module_roles ?? {}).map(([mod, names]) => [mod, [...names]])
        )
      );
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load user");
    } finally {
      setLoading(false);
    }
  }

  // Group roles by module
  const systemRoles = useMemo(() => roles.filter((r) => r.module === null), [roles]);
  const moduleGroups = useMemo(() => {
    const groups: Record<string, Role[]> = {};
    roles.forEach((r) => {
      if (r.module) {
        if (!groups[r.module]) groups[r.module] = [];
        groups[r.module].push(r);
      }
    });
    return groups;
  }, [roles]);

  function toggleSystemRole(name: string) {
    setSelectedSystemRoles((prev) =>
      prev.includes(name) ? prev.filter((n) => n !== name) : [...prev, name]
    );
  }

  function toggleModuleRole(mod: string, name: string) {
    setSelectedModuleRoles((prev) => {
      const current = prev[mod] ?? [];
      const updated = current.includes(name)
        ? current.filter((n) => n !== name)
        : [...current, name];
      return { ...prev, [mod]: updated };
    });
  }

  async function handleSaveRoles() {
    setSavingRoles(true);
    try {
      const token = await getToken();
      if (!token || !user) return;
      // Only include modules that have at least one role selected
      const filteredModuleRoles = Object.fromEntries(
        Object.entries(selectedModuleRoles).filter(([, names]) => names.length > 0)
      );
      await userApi.updateRoles(
        user.id,
        {
          system_roles: selectedSystemRoles,
          module_roles: filteredModuleRoles,
        },
        token
      );
      toast.success("Roles updated");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to update roles");
    } finally {
      setSavingRoles(false);
    }
  }

  async function handleSuspend() {
    if (!user) return;
    try {
      const token = await getToken();
      if (!token) return;
      await userApi.suspend(user.id, token);
      toast.success("User suspended");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  async function handleResendInvite() {
    if (!user) return;
    setResending(true);
    try {
      const token = await getToken();
      if (!token) return;
      await userApi.resendInvite(user.id, token);
      toast.success("Invitation re-sent successfully");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to resend invitation");
    } finally {
      setResending(false);
    }
  }

  async function handleSendPasswordReset() {
    if (!user) return;
    setSendingReset(true);
    try {
      await authApi.forgotPassword(user.email, tenantSlug);
      toast.success("Password reset email sent.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to send password reset.");
    } finally {
      setSendingReset(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-tiger-red" size={28} />
      </div>
    );
  }

  if (!user) {
    return <div className="text-center py-24 text-semi-grey text-sm">User not found.</div>;
  }

  const statusColor: Record<string, string> = {
    active: "bg-[#EDFBF5] dark:bg-green-900/30 text-[#34D186] border-[#34D186]/40",
    inactive: "bg-gray-100 dark:bg-gray-800 text-semi-grey border-gray-200 dark:border-gray-700",
    pending: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800",
  };

  return (
    <div className="space-y-5 max-w-3xl">
      {/* Back + Header — stacks on mobile */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        {/* Back button + name row */}
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => router.back()}
            className="h-8 w-8 rounded-full shrink-0"
          >
            <ArrowLeft size={16} />
          </Button>
          <div className="flex-1 min-w-0">
            <h1 className="text-base font-semibold text-semi-black truncate">
              {user.display_name}
            </h1>
            <p className="text-xs text-semi-grey truncate">{user.email}</p>
          </div>
          {/* Status badge — always visible next to name */}
          <span
            className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border shrink-0 ${
              statusColor[user.status] ?? ""
            }`}
          >
            {user.status}
          </span>
        </div>

        {/* Action buttons — full-width stacked on mobile, inline on sm+ */}
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-2">
          {isAdmin && user.status === "pending" && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleResendInvite}
              disabled={resending}
              className="w-full sm:w-auto rounded-[1000px] text-xs h-10 sm:h-8 text-tiger-red border-tiger-red/30 hover:bg-tiger-red/5"
            >
              {resending ? (
                <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
              ) : (
                <Send size={12} className="mr-1.5" />
              )}
              Resend Invite
            </Button>
          )}
          {isAdmin && user.status === "active" && (
            <Button
              onClick={handleSendPasswordReset}
              disabled={sendingReset}
              variant="outline"
              className="w-full sm:w-auto rounded-[1000px] text-xs h-10 sm:h-8 text-semi-black border-border"
            >
              {sendingReset ? (
                <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />
              ) : (
                <KeyRound size={12} className="mr-1.5" />
              )}
              Send Password Reset
            </Button>
          )}
          {isAdmin &&
            user.status === "active" &&
            user.id !== authUser?.sub &&
            (!user.system_roles?.includes("super_admin") || isSuperAdmin) && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleSuspend}
              className="w-full sm:w-auto rounded-[1000px] text-xs h-10 sm:h-8 text-destructive border-destructive/30 hover:bg-destructive/5"
            >
              Suspend
            </Button>
          )}
        </div>
      </div>

      {/* Account Info Card */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black">
            Account Info
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div className="flex flex-col gap-0.5 sm:flex-row sm:justify-between">
            <span className="text-semi-grey shrink-0">User ID</span>
            <span className="font-mono text-xs text-semi-black max-w-full sm:max-w-[260px] truncate">
              {user.id}
            </span>
          </div>
          <Separator />
          <div className="flex flex-col gap-0.5 sm:flex-row sm:justify-between">
            <span className="text-semi-grey shrink-0">Tenant</span>
            <span className="font-mono text-xs text-semi-black truncate">
              {user.tenant_id}
            </span>
          </div>
          <Separator />
          <div className="flex justify-between">
            <span className="text-semi-grey">Status</span>
            <span
              className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${
                statusColor[user.status] ?? ""
              }`}
            >
              {user.status}
            </span>
          </div>
          <Separator />
          <div className="flex justify-between">
            <span className="text-semi-grey">Joined</span>
            <span className="text-xs text-semi-black">
              {new Date(user.created_at).toLocaleDateString("th-TH")}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Roles Section */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3 flex flex-row items-center justify-between gap-2">
          <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
            <Shield size={15} className="text-tiger-red" />
            Roles
          </CardTitle>
          {isAdmin && (
            <Button
              size="sm"
              onClick={handleSaveRoles}
              disabled={savingRoles}
              className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-xs h-9 sm:h-8 px-3 shrink-0"
            >
              {savingRoles && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              Save Roles
            </Button>
          )}
        </CardHeader>
        <CardContent className="space-y-5">
          {/* System Roles sub-section */}
          <div className="space-y-2">
            <p className="text-xs font-semibold text-semi-grey uppercase tracking-wide">
              System Roles
            </p>
            {systemRoles.length === 0 ? (
              <p className="text-xs text-semi-grey">No system roles available</p>
            ) : (
              <div className="space-y-1.5">
                {systemRoles.map((role) => (
                  <label
                    key={role.id}
                    className={`flex items-center gap-3 rounded-[10px] border p-2.5 transition-colors ${
                      isAdmin ? "cursor-pointer" : "cursor-default"
                    } ${
                      selectedSystemRoles.includes(role.name)
                        ? "border-tiger-red/30 bg-[#FFF8F8] dark:bg-tiger-red/5"
                        : "border-border hover:bg-[#fafafa] dark:hover:bg-[#1a2332]"
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={selectedSystemRoles.includes(role.name)}
                      onChange={() => isAdmin && toggleSystemRole(role.name)}
                      disabled={!isAdmin}
                      className="accent-tiger-red w-4 h-4"
                    />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm text-semi-black font-medium">{role.name}</p>
                      {role.description && (
                        <p className="text-xs text-semi-grey truncate">{role.description}</p>
                      )}
                    </div>
                    {role.is_system && (
                      <Badge variant="outline" className="text-[10px] border-tiger-red text-tiger-red shrink-0">
                        system
                      </Badge>
                    )}
                  </label>
                ))}
              </div>
            )}
          </div>

          {/* Module Roles sub-sections */}
          {Object.keys(moduleGroups).length > 0 && (
            <>
              <Separator />
              {Object.entries(moduleGroups).map(([mod, modRoles]) => (
                <div key={mod} className="space-y-2">
                  <div className="flex items-center gap-2">
                    <p className="text-xs font-semibold text-semi-grey uppercase tracking-wide">
                      Module Roles
                    </p>
                    <Badge variant="outline" className="text-[10px] border-indigo-300 text-indigo-600">
                      {mod}
                    </Badge>
                  </div>
                  <div className="space-y-1.5">
                    {modRoles.map((role) => {
                      const checked = (selectedModuleRoles[mod] ?? []).includes(role.name);
                      return (
                        <label
                          key={role.id}
                          className={`flex items-center gap-3 rounded-[10px] border p-2.5 transition-colors ${
                            isAdmin ? "cursor-pointer" : "cursor-default"
                          } ${
                            checked
                              ? "border-indigo-300/60 bg-indigo-50/50 dark:bg-indigo-900/20 dark:border-indigo-500/30"
                              : "border-border hover:bg-[#fafafa] dark:hover:bg-[#1a2332]"
                          }`}
                        >
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => isAdmin && toggleModuleRole(mod, role.name)}
                            disabled={!isAdmin}
                            className="accent-indigo-600 w-4 h-4"
                          />
                          <div className="flex-1 min-w-0">
                            <p className="text-sm text-semi-black font-medium">{role.name}</p>
                            {role.description && (
                              <p className="text-xs text-semi-grey truncate">{role.description}</p>
                            )}
                          </div>
                        </label>
                      );
                    })}
                  </div>
                </div>
              ))}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
