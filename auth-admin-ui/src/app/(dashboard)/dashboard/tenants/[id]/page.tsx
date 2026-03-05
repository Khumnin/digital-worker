"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  ArrowLeft,
  Copy,
  Check,
  Loader2,
  RefreshCw,
  Globe,
  Code2,
  Users,
  Plus,
  Mail,
  MoreHorizontal,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/contexts/auth";
import {
  tenantApi,
  type Tenant,
  type User,
  type InviteAdminRequest,
  ApiError,
} from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev";

function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="space-y-1">
      <p className="text-xs text-semi-grey font-medium uppercase">{label}</p>
      <div className="flex items-center gap-2 bg-[#f0f0f0] dark:bg-input rounded-[10px] px-3 py-2">
        <code className="text-xs text-semi-black flex-1 break-all font-mono">{value}</code>
        <button
          onClick={copy}
          className="shrink-0 text-semi-grey hover:text-semi-black transition-colors"
        >
          {copied ? <Check size={14} className="text-green-600" /> : <Copy size={14} />}
        </button>
      </div>
    </div>
  );
}

const statusColor: Record<string, string> = {
  active: "bg-[#EDFBF5] dark:bg-green-900/30 text-[#34D186] border-[#34D186]/40",
  pending: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800",
  inactive: "bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border-gray-200 dark:border-gray-700",
};

export default function TenantDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { getToken } = useAuth();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [usersLoading, setUsersLoading] = useState(false);
  const [showInvite, setShowInvite] = useState(false);
  const [inviting, setInviting] = useState(false);
  const [form, setForm] = useState<InviteAdminRequest>({ email: "", display_name: "" });

  const id = params?.id as string;

  useEffect(() => {
    load();
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const t = await tenantApi.get(id, token);
      setTenant(t);
      loadUsers(token);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load tenant");
    } finally {
      setLoading(false);
    }
  }

  async function loadUsers(token?: string) {
    setUsersLoading(true);
    try {
      const tok = token ?? (await getToken());
      if (!tok) return;
      const result = await tenantApi.listUsers(id, tok);
      setUsers(result.data);
    } catch {
      // Non-fatal: users section may be empty
    } finally {
      setUsersLoading(false);
    }
  }

  async function handleToggleStatus() {
    if (!tenant) return;
    try {
      const token = await getToken();
      if (!token) return;
      if (tenant.status === "active") {
        await tenantApi.suspend(tenant.id, token);
        toast.success("Tenant suspended");
      } else {
        await tenantApi.activate(tenant.id, token);
        toast.success("Tenant activated");
      }
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  async function handleInviteAdmin(e: React.FormEvent) {
    e.preventDefault();
    setInviting(true);
    try {
      const token = await getToken();
      if (!token) return;
      await tenantApi.inviteAdmin(id, form, token);
      toast.success(`Invitation sent to ${form.email}`);
      setShowInvite(false);
      setForm({ email: "", display_name: "" });
      await loadUsers();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to send invitation");
    } finally {
      setInviting(false);
    }
  }

  async function handleResendInvite(userId: string, email: string) {
    try {
      const token = await getToken();
      if (!token) return;
      await tenantApi.resendAdminInvite(id, userId, token);
      toast.success(`Invitation re-sent to ${email}`);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to resend");
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-tiger-red" size={28} />
      </div>
    );
  }

  if (!tenant) {
    return (
      <div className="text-center py-24 text-semi-grey text-sm">
        Tenant not found.
      </div>
    );
  }

  const integrationEnv = `AUTH_API_URL=${API_BASE}
AUTH_JWKS_URL=${API_BASE}/.well-known/jwks.json
AUTH_ISSUER=${API_BASE}
AUTH_AUDIENCE=tigersoft-auth
AUTH_TENANT_SLUG=${tenant.slug}
AUTH_PLATFORM_TENANT_SLUG=platform`;

  return (
    <div className="space-y-5 max-w-4xl">
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
          <h1 className="text-base font-semibold text-semi-black">{tenant.name}</h1>
          <p className="text-xs text-semi-grey font-mono">{tenant.slug}</p>
        </div>
        <div className="flex items-center gap-2">
          <span className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border ${
            statusColor[tenant.status] ?? "bg-gray-100 dark:bg-gray-800 text-gray-500 dark:text-gray-400 border-gray-200 dark:border-gray-700"
          }`}>
            {tenant.status}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={handleToggleStatus}
            className="rounded-[1000px] text-xs h-8"
          >
            <RefreshCw size={12} className="mr-1.5" />
            {tenant.status === "active" ? "Suspend" : "Activate"}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Tenant info */}
        <Card className="rounded-[10px] border-border shadow-none">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
              <Globe size={15} className="text-tiger-red" />
              Tenant Info
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <CopyField label="Tenant ID" value={tenant.id} />
            <CopyField label="Slug" value={tenant.slug} />
            <div className="space-y-1">
              <p className="text-xs text-semi-grey font-medium uppercase">Enabled Modules</p>
              <div className="flex flex-wrap gap-1 pt-1">
                {(tenant.enabled_modules ?? []).length === 0 ? (
                  <span className="text-xs text-semi-grey">None</span>
                ) : (
                  (tenant.enabled_modules ?? []).map((mod) => (
                    <Badge
                      key={mod}
                      variant="outline"
                      className="text-[10px] border-tiger-red text-tiger-red uppercase"
                    >
                      {mod}
                    </Badge>
                  ))
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Recruitment Integration Panel */}
        {(tenant.enabled_modules ?? []).includes("recruitment") && (
          <Card className="rounded-[10px] border-tiger-red/20 bg-[#FFF8F8] dark:bg-tiger-red/5 shadow-none">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
                <Code2 size={15} className="text-tiger-red" />
                Recruitment Integration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <p className="text-xs text-semi-grey">
                Copy these env vars into your Recruitment backend deployment:
              </p>
              <div className="bg-card rounded-[10px] border border-border p-3">
                <code className="text-xs font-mono text-semi-black whitespace-pre-wrap break-all leading-relaxed">
                  {integrationEnv}
                </code>
              </div>
              <button
                onClick={async () => {
                  await navigator.clipboard.writeText(integrationEnv);
                  toast.success("Copied to clipboard");
                }}
                className="text-xs text-tiger-red hover:underline font-medium"
              >
                Copy all env vars
              </button>
            </CardContent>
          </Card>
        )}
      </div>

      <Separator />

      {/* Administrators Section */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
              <Users size={15} className="text-tiger-red" />
              Administrators
            </CardTitle>
            <Button
              size="sm"
              onClick={() => setShowInvite(true)}
              className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-xs h-8 px-3"
            >
              <Plus size={13} className="mr-1" />
              Invite Admin
            </Button>
          </div>
          <p className="text-xs text-semi-grey mt-1">
            Admins manage users, roles, and settings within this tenant.
          </p>
        </CardHeader>
        <CardContent>
          {usersLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="animate-spin text-tiger-red" size={20} />
            </div>
          ) : users.length === 0 ? (
            <div className="text-center py-8 text-semi-grey">
              <Users size={28} className="mx-auto mb-2 opacity-30" />
              <p className="text-xs">No users found in this tenant.</p>
            </div>
          ) : (
            <div className="divide-y divide-border">
              {users.map((user) => (
                <div key={user.id} className="flex items-center gap-3 py-3">
                  {/* Avatar */}
                  <div className="w-8 h-8 rounded-full bg-tiger-red/10 flex items-center justify-center shrink-0">
                    <span className="text-xs font-semibold text-tiger-red uppercase">
                      {(user.display_name || user.email).charAt(0)}
                    </span>
                  </div>

                  {/* Info */}
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-semi-black truncate">
                      {user.display_name || user.email}
                    </p>
                    <p className="text-xs text-semi-grey truncate">{user.email}</p>
                  </div>

                  {/* Roles */}
                  <div className="flex items-center gap-2 shrink-0">
                    {(user.system_roles ?? []).map((r) => (
                      <Badge
                        key={r}
                        variant="outline"
                        className="text-[10px] border-tiger-red/40 text-tiger-red uppercase"
                      >
                        {r}
                      </Badge>
                    ))}
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded-full text-[10px] font-medium border ${
                        statusColor[user.status] ?? "bg-gray-100 text-gray-500 border-gray-200"
                      }`}
                    >
                      {user.status}
                    </span>
                  </div>

                  {/* Actions */}
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-7 w-7 rounded-full shrink-0">
                        <MoreHorizontal size={14} />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {user.status === "pending" && (
                        <DropdownMenuItem
                          onClick={() => handleResendInvite(user.id, user.email)}
                          className="gap-2"
                        >
                          <Mail size={13} />
                          Resend Invite
                        </DropdownMenuItem>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <Separator />

      {/* Module Configuration */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black">
            Module Configuration
          </CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="text-xs font-mono text-semi-black bg-[#f0f0f0] dark:bg-input rounded-[10px] p-4 overflow-auto">
            {JSON.stringify({ enabled_modules: tenant.enabled_modules }, null, 2)}
          </pre>
        </CardContent>
      </Card>

      {/* Invite Admin Dialog */}
      <Dialog open={showInvite} onOpenChange={(open) => { setShowInvite(open); if (!open) setForm({ email: "", display_name: "" }); }}>
        <DialogContent className="sm:max-w-[440px] rounded-[10px]">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-semi-black">
              Invite Admin to {tenant.name}
            </DialogTitle>
          </DialogHeader>
          <p className="text-sm text-semi-grey -mt-1">
            The user will receive an email to set their password and activate their account.
          </p>
          <form onSubmit={handleInviteAdmin} className="space-y-4 mt-1">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Email</Label>
              <Input
                required
                type="email"
                placeholder="admin@company.co.th"
                value={form.email}
                onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Display Name</Label>
              <Input
                required
                placeholder="John Smith"
                value={form.display_name}
                onChange={(e) => setForm((f) => ({ ...f, display_name: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
              />
            </div>

            {/* Flow explanation */}
            <div className="bg-[#FFF8F8] dark:bg-tiger-red/5 border border-tiger-red/20 rounded-[10px] p-3 space-y-1.5">
              <p className="text-xs font-medium text-semi-black">What happens next</p>
              <ol className="text-xs text-semi-grey space-y-1 list-decimal list-inside">
                <li>User receives invitation email from TGX</li>
                <li>User sets password via the activation link</li>
                <li>User can sign in to TGX Auth Console as <span className="font-mono text-tiger-red">admin</span></li>
              </ol>
            </div>

            <DialogFooter className="pt-1">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setShowInvite(false)}
                className="rounded-[1000px]"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={inviting}
                className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white"
              >
                {inviting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Send Invitation
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
