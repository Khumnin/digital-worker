"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, CheckCircle2, Send, RefreshCw, Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { useAuth } from "@/contexts/auth";
import {
  userApi,
  tenantApi,
  type Tenant,
  type InviteUserRequest,
  ApiError,
} from "@/lib/api";

interface InviteUserDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

const EMPTY_FORM: InviteUserRequest = { email: "", display_name: "" };

// ── IU-4: Custom tenant combobox with client-side search ───────────────────
interface TenantComboboxProps {
  tenants: Tenant[];
  loading: boolean;
  value: string;
  onChange: (id: string) => void;
}

function TenantCombobox({ tenants, loading, value, onChange }: TenantComboboxProps) {
  const [search, setSearch] = useState("");
  const [open, setOpen] = useState(false);

  const filtered = tenants.filter(
    (t) =>
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.slug.toLowerCase().includes(search.toLowerCase())
  );

  const selected = tenants.find((t) => t.id === value);

  return (
    <div className="relative">
      {/* Trigger */}
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        disabled={loading}
        className="w-full h-11 rounded-[10px] bg-[#f0f0f0] dark:bg-input border border-[#f0f0f0] dark:border-input px-4 text-sm text-left flex items-center justify-between text-semi-black transition-colors focus:outline-none focus:ring-2 focus:ring-tiger-red/50"
      >
        <span className={selected ? "text-semi-black" : "text-semi-grey"}>
          {loading
            ? "Loading tenants…"
            : selected
            ? `${selected.name} (${selected.slug})`
            : "Select a tenant (optional)"}
        </span>
        <span className="text-semi-grey text-xs">▾</span>
      </button>

      {/* Dropdown */}
      {open && !loading && (
        <div className="absolute z-50 mt-1 w-full bg-card border border-border rounded-[10px] shadow-lg overflow-hidden">
          {/* Search input */}
          <div className="p-2 border-b border-border">
            <div className="relative">
              <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-semi-grey" />
              <input
                autoFocus
                placeholder="Search tenant…"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="w-full pl-8 pr-2 py-1.5 text-sm bg-[#f0f0f0] dark:bg-input rounded-[8px] outline-none text-semi-black placeholder:text-semi-grey"
              />
            </div>
          </div>

          {/* Options */}
          <div className="max-h-[200px] overflow-y-auto">
            {/* Clear option */}
            <button
              type="button"
              onClick={() => { onChange(""); setOpen(false); setSearch(""); }}
              className="w-full px-4 py-2 text-sm text-left text-semi-grey hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] italic"
            >
              None (invite to own tenant)
            </button>

            {filtered.length === 0 ? (
              <p className="px-4 py-3 text-sm text-semi-grey">No tenants found</p>
            ) : (
              filtered.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => { onChange(t.id); setOpen(false); setSearch(""); }}
                  className={`w-full px-4 py-2 text-sm text-left flex items-center justify-between hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] ${
                    t.id === value ? "text-tiger-red font-medium" : "text-semi-black"
                  }`}
                >
                  <span>{t.name}</span>
                  <span className="text-xs text-semi-grey">{t.slug}</span>
                </button>
              ))
            )}
          </div>
        </div>
      )}

      {/* Overlay to close on outside click */}
      {open && (
        <div
          className="fixed inset-0 z-40"
          onClick={() => { setOpen(false); setSearch(""); }}
        />
      )}
    </div>
  );
}

// ── Main dialog ────────────────────────────────────────────────────────────
export function InviteUserDialog({
  open,
  onOpenChange,
  onSuccess,
}: InviteUserDialogProps) {
  const { getToken, isSuperAdmin } = useAuth();
  const [form, setForm] = useState<InviteUserRequest>(EMPTY_FORM);
  const [inviteRole, setInviteRole] = useState<"admin" | "user">("user");
  const [selectedTenantId, setSelectedTenantId] = useState<string>("");
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [tenantsLoading, setTenantsLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  // IU-5: confirmation state
  const [sentEmail, setSentEmail] = useState<string | null>(null);
  const [resending, setResending] = useState(false);

  // Load active tenants when dialog opens (super_admin only)
  useEffect(() => {
    if (!open || !isSuperAdmin) return;
    let cancelled = false;
    setTenantsLoading(true);
    getToken().then(async (token) => {
      if (!token || cancelled) return;
      try {
        const result = await tenantApi.list(token, { page_size: 200 });
        if (!cancelled) {
          setTenants(result.data.filter((t) => t.status === "active"));
        }
      } catch {
        // silently ignore
      } finally {
        if (!cancelled) setTenantsLoading(false);
      }
    });
    return () => { cancelled = true; };
  }, [open, isSuperAdmin, getToken]);

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      setForm(EMPTY_FORM);
      setInviteRole("user");
      setSelectedTenantId("");
      setSentEmail(null);
    }
    onOpenChange(isOpen);
  };

  const isCrossTenant = isSuperAdmin && selectedTenantId !== "";

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    try {
      const token = await getToken();
      if (!token) return;
      if (isCrossTenant) {
        await tenantApi.inviteAdmin(
          selectedTenantId,
          { email: form.email, display_name: form.display_name },
          token
        );
      } else {
        await userApi.invite({ ...form, initial_role: inviteRole }, token);
      }
      // IU-5: Show confirmation instead of closing immediately
      setSentEmail(form.email);
      onSuccess?.();
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to send invitation"
      );
    } finally {
      setSubmitting(false);
    }
  }

  // IU-5: Resend from confirmation state
  async function handleResend() {
    if (!sentEmail) return;
    setResending(true);
    try {
      const token = await getToken();
      if (!token) return;
      // Re-trigger the same invite (backend is idempotent on pending users)
      if (isCrossTenant) {
        await tenantApi.inviteAdmin(
          selectedTenantId,
          { email: form.email, display_name: form.display_name },
          token
        );
      } else {
        await userApi.invite({ ...form, initial_role: inviteRole }, token);
      }
      toast.success(`Invitation re-sent to ${sentEmail}`);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to resend invitation"
      );
    } finally {
      setResending(false);
    }
  }

  // ── IU-5: Confirmation view ─────────────────────────────────────────────
  if (sentEmail) {
    return (
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-[440px] rounded-[10px]">
          <div className="flex flex-col items-center text-center py-4 space-y-4">
            <div className="flex items-center justify-center w-14 h-14 rounded-full bg-green-50 dark:bg-green-900/20">
              <CheckCircle2 size={28} className="text-green-500" />
            </div>
            <div>
              <h3 className="text-base font-semibold text-semi-black">
                Invitation sent!
              </h3>
              <p className="text-sm text-semi-grey mt-1">
                An invitation email has been sent to
              </p>
              <p className="text-sm font-medium text-semi-black mt-0.5">
                {sentEmail}
              </p>
            </div>

            <div className="flex gap-2 pt-2 w-full">
              <Button
                type="button"
                variant="outline"
                disabled={resending}
                onClick={handleResend}
                className="flex-1 rounded-[1000px] text-sm gap-1.5"
              >
                {resending ? (
                  <Loader2 size={14} className="animate-spin" />
                ) : (
                  <RefreshCw size={14} />
                )}
                Resend
              </Button>
              <Button
                type="button"
                onClick={() => handleOpenChange(false)}
                className="flex-1 rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm gap-1.5"
              >
                <Send size={14} />
                Done
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    );
  }

  // ── Main form view ──────────────────────────────────────────────────────
  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-[440px] rounded-[10px]">
        <DialogHeader>
          <DialogTitle className="text-base font-semibold text-semi-black">
            Invite User
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4 mt-2">
          {isSuperAdmin && (
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">
                Tenant{" "}
                <span className="text-semi-grey font-normal">
                  (leave blank to invite to your own tenant)
                </span>
              </Label>
              {/* IU-4: Combobox with search */}
              <TenantCombobox
                tenants={tenants}
                loading={tenantsLoading}
                value={selectedTenantId}
                onChange={setSelectedTenantId}
              />
              {isCrossTenant && (
                <p className="text-xs text-amber-600 dark:text-amber-400">
                  This will invite the user as an <strong>admin</strong> of the
                  selected tenant.
                </p>
              )}
            </div>
          )}

          <div className="space-y-1.5">
            <Label className="text-sm font-medium text-semi-black">
              Display Name
            </Label>
            <Input
              required
              placeholder="สมชาย ใจดี"
              value={form.display_name}
              onChange={(e) =>
                setForm((f) => ({ ...f, display_name: e.target.value }))
              }
              className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
            />
          </div>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium text-semi-black">
              Email
            </Label>
            <Input
              type="email"
              required
              placeholder="user@company.co.th"
              value={form.email}
              onChange={(e) =>
                setForm((f) => ({ ...f, email: e.target.value }))
              }
              className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
            />
          </div>

          {!isCrossTenant && (
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">
                Role
              </Label>
              <select
                value={inviteRole}
                onChange={(e) =>
                  setInviteRole(e.target.value as "admin" | "user")
                }
                className="w-full h-11 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-0 px-4 text-sm text-semi-black"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </div>
          )}

          <p className="text-xs text-semi-grey">
            {isCrossTenant
              ? "The user will receive an invitation email to set up their account."
              : "Additional roles can be assigned after the user accepts their invitation."}
          </p>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="ghost"
              onClick={() => handleOpenChange(false)}
              className="rounded-[1000px]"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={submitting}
              className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white"
            >
              {submitting && (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              )}
              Send Invite
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
