"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import { ArrowLeft, Loader2, UserCheck2, Shield } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useAuth } from "@/contexts/auth";
import { userApi, roleApi, type User, type Role, ApiError } from "@/lib/api";

export default function UserDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { getToken, tenantId } = useAuth();
  const [user, setUser] = useState<User | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [savingRoles, setSavingRoles] = useState(false);
  const [selectedRoleIds, setSelectedRoleIds] = useState<string[]>([]);

  const id = params?.id as string;

  useEffect(() => {
    load();
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token || !tenantId) return;
      const [u, allRoles] = await Promise.all([
        userApi.get(id, token, tenantId),
        roleApi.list(token, tenantId),
      ]);
      setUser(u);
      setRoles(allRoles);
      setSelectedRoleIds(u.roles.map((r) => r.id));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load user");
    } finally {
      setLoading(false);
    }
  }

  async function handleSaveRoles() {
    setSavingRoles(true);
    try {
      const token = await getToken();
      if (!token || !tenantId || !user) return;
      await userApi.updateRoles(user.id, selectedRoleIds, token, tenantId);
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
      if (!token || !tenantId) return;
      await userApi.suspend(user.id, token, tenantId);
      toast.success("User suspended");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
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
    active: "bg-green-100 text-green-700 border-green-200",
    inactive: "bg-gray-100 text-semi-grey border-gray-200",
    pending: "bg-yellow-100 text-yellow-700 border-yellow-200",
  };

  return (
    <div className="space-y-5 max-w-3xl">
      {/* Back + Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.back()}
          className="h-8 w-8 rounded-full"
        >
          <ArrowLeft size={16} />
        </Button>
        <div className="flex-1">
          <h1 className="text-base font-semibold text-semi-black">
            {user.first_name} {user.last_name}
          </h1>
          <p className="text-xs text-semi-grey">{user.email}</p>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border ${
              statusColor[user.status] ?? ""
            }`}
          >
            {user.status}
          </span>
          {user.status === "active" && (
            <Button
              variant="outline"
              size="sm"
              onClick={handleSuspend}
              className="rounded-[1000px] text-xs h-8 text-destructive border-destructive/30 hover:bg-destructive/5"
            >
              Suspend
            </Button>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* User info */}
        <Card className="rounded-[10px] border-border shadow-none">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold text-semi-black">
              Account Info
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3 text-sm">
            <div className="flex justify-between">
              <span className="text-semi-grey">User ID</span>
              <span className="font-mono text-xs text-semi-black max-w-[180px] truncate">
                {user.id}
              </span>
            </div>
            <Separator />
            <div className="flex justify-between items-center">
              <span className="text-semi-grey">Email verified</span>
              {user.email_verified_at ? (
                <span className="text-green-600 text-xs">
                  {new Date(user.email_verified_at).toLocaleDateString("th-TH")}
                </span>
              ) : (
                <span className="text-semi-grey text-xs">Not verified</span>
              )}
            </div>
            <Separator />
            <div className="flex justify-between items-center">
              <span className="text-semi-grey">MFA</span>
              {user.mfa_enabled ? (
                <span className="flex items-center gap-1 text-green-600 text-xs">
                  <UserCheck2 size={13} /> Enabled
                </span>
              ) : (
                <span className="text-semi-grey text-xs">Disabled</span>
              )}
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

        {/* Role assignment */}
        <Card className="rounded-[10px] border-border shadow-none">
          <CardHeader className="pb-3 flex flex-row items-center justify-between">
            <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
              <Shield size={15} className="text-tiger-red" />
              Roles
            </CardTitle>
            <Button
              size="sm"
              onClick={handleSaveRoles}
              disabled={savingRoles}
              className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-xs h-8 px-3"
            >
              {savingRoles && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
              Save
            </Button>
          </CardHeader>
          <CardContent className="space-y-1.5">
            {roles.map((role) => (
              <label
                key={role.id}
                className={`flex items-center gap-3 cursor-pointer rounded-[10px] border p-2.5 transition-colors ${
                  selectedRoleIds.includes(role.id)
                    ? "border-tiger-red/30 bg-[#FFF8F8]"
                    : "border-border hover:bg-[#fafafa]"
                }`}
              >
                <input
                  type="checkbox"
                  checked={selectedRoleIds.includes(role.id)}
                  onChange={() =>
                    setSelectedRoleIds((ids) =>
                      ids.includes(role.id)
                        ? ids.filter((id) => id !== role.id)
                        : [...ids, role.id]
                    )
                  }
                  className="accent-tiger-red w-4 h-4"
                />
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-semi-black font-medium">{role.name}</p>
                  {role.description && (
                    <p className="text-xs text-semi-grey truncate">{role.description}</p>
                  )}
                </div>
                {role.is_system && (
                  <Badge variant="outline" className="text-[10px] border-semi-grey text-semi-grey shrink-0">
                    system
                  </Badge>
                )}
              </label>
            ))}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
