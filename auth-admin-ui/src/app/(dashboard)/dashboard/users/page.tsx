"use client";

import { useEffect, useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  Plus,
  Search,
  MoreHorizontal,
  Loader2,
  Users as UsersIcon,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useAuth } from "@/contexts/auth";
import {
  userApi,
  type User,
  type InviteUserRequest,
  ApiError,
} from "@/lib/api";

// Known modules — can be extended or derived from roles data
const KNOWN_MODULES = ["recruit"];

const STATUS_OPTIONS = [
  { value: "all", label: "All Statuses" },
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "pending", label: "Pending" },
];

const MODULE_OPTIONS = [
  { value: "all", label: "All Modules" },
  ...KNOWN_MODULES.map((m) => ({ value: m, label: m.charAt(0).toUpperCase() + m.slice(1) })),
];

export default function UsersPage() {
  const router = useRouter();
  const { getToken, isAdmin, isSuperAdmin, user: authUser } = useAuth();

  // Suspend is allowed only when: actor is admin, target is not self,
  // and target is not a super_admin (unless the actor is also super_admin).
  const canSuspend = (u: User) =>
    isAdmin &&
    u.id !== authUser?.sub &&
    (!u.system_roles?.includes("super_admin") || isSuperAdmin);
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [moduleFilter, setModuleFilter] = useState<string>("all");
  const [showInvite, setShowInvite] = useState(false);
  const [inviting, setInviting] = useState(false);
  const [inviteRole, setInviteRole] = useState<"admin" | "user">("user");
  const [form, setForm] = useState<InviteUserRequest>({
    email: "",
    display_name: "",
  });

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const usersResult = await userApi.list(token, { page_size: 100 });
      setUsers(usersResult.data);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load users");
    } finally {
      setLoading(false);
    }
  }, [getToken]);

  useEffect(() => {
    load();
  }, [load]);

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    setInviting(true);
    try {
      const token = await getToken();
      if (!token) return;
      await userApi.invite({ ...form, initial_role: inviteRole }, token);
      toast.success(`Invitation sent to ${form.email}`);
      setShowInvite(false);
      setForm({ email: "", display_name: "" });
      setInviteRole("user");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to invite user");
    } finally {
      setInviting(false);
    }
  }

  async function handleSuspend(id: string) {
    try {
      const token = await getToken();
      if (!token) return;
      await userApi.suspend(id, token);
      toast.success("User suspended");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  async function handleResendInvite(id: string) {
    try {
      const token = await getToken();
      if (!token) return;
      await userApi.resendInvite(id, token);
      toast.success("Invitation re-sent successfully");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to resend invitation");
    }
  }

  function clearFilters() {
    setStatusFilter("all");
    setModuleFilter("all");
  }

  const hasActiveFilters = statusFilter !== "all" || moduleFilter !== "all";

  // Client-side filters (backend returns all users; we filter here)
  const filtered = users.filter((u) => {
    const matchSearch =
      u.email.toLowerCase().includes(search.toLowerCase()) ||
      (u.display_name ?? "").toLowerCase().includes(search.toLowerCase());
    const matchStatus = statusFilter === "all" || u.status === statusFilter;
    const matchModule =
      moduleFilter === "all" ||
      Object.keys(u.module_roles ?? {}).includes(moduleFilter);
    return matchSearch && matchStatus && matchModule;
  });

  const statusColor: Record<string, string> = {
    active: "bg-green-100 text-green-700 border-green-200",
    inactive: "bg-gray-100 text-semi-grey border-gray-200",
    pending: "bg-yellow-100 text-yellow-700 border-yellow-200",
  };

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center gap-3 flex-wrap">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-semi-grey" />
          <Input
            placeholder="ค้นหาผู้ใช้..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm"
          />
        </div>

        {/* Status filter */}
        <div className="flex items-center gap-1.5">
          <Label className="text-xs text-semi-grey font-medium whitespace-nowrap">Status</Label>
          <Select value={statusFilter} onValueChange={setStatusFilter}>
            <SelectTrigger className="h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm min-w-[140px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {STATUS_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Module filter */}
        <div className="flex items-center gap-1.5">
          <Label className="text-xs text-semi-grey font-medium whitespace-nowrap">Module</Label>
          <Select value={moduleFilter} onValueChange={setModuleFilter}>
            <SelectTrigger className="h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm min-w-[140px]">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {MODULE_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Clear filters */}
        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="h-10 rounded-[1000px] text-xs text-semi-grey hover:text-semi-black gap-1"
          >
            <X size={13} />
            Clear filters
          </Button>
        )}

        {isAdmin && (
          <Button
            onClick={() => setShowInvite(true)}
            className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm h-10 px-4 ml-auto"
          >
            <Plus size={16} className="mr-1.5" />
            Invite User
          </Button>
        )}
      </div>

      {/* Table */}
      <div className="bg-white rounded-[10px] border border-border overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="animate-spin text-tiger-red" size={24} />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <UsersIcon size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No users found</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-[#fafafa] hover:bg-[#fafafa]">
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">User</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Roles</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Status</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Joined</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((user) => (
                <TableRow
                  key={user.id}
                  className="cursor-pointer hover:bg-[#fafafa]"
                  onClick={() => router.push(`/dashboard/users/${user.id}`)}
                >
                  <TableCell>
                    <div>
                      <p className="text-sm font-medium text-semi-black">
                        {user.display_name}
                      </p>
                      <p className="text-xs text-semi-grey">{user.email}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {(user.system_roles ?? []).map((roleName) => (
                        <Badge
                          key={roleName}
                          variant="outline"
                          className={`text-[10px] ${
                            roleName === "super_admin"
                              ? "border-tiger-red text-tiger-red"
                              : roleName === "applicant"
                              ? "border-blue-400 text-blue-600"
                              : "border-border text-semi-grey"
                          }`}
                        >
                          {roleName}
                        </Badge>
                      ))}
                      {Object.entries(user.module_roles ?? {}).flatMap(([mod, modRoles]) =>
                        modRoles.map((roleName) => (
                          <Badge
                            key={`${mod}:${roleName}`}
                            variant="outline"
                            className="text-[10px] border-indigo-300 text-indigo-600"
                          >
                            {mod}:{roleName}
                          </Badge>
                        ))
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${
                        statusColor[user.status] ?? ""
                      }`}
                    >
                      {user.status}
                    </span>
                  </TableCell>
                  <TableCell className="text-xs text-semi-grey">
                    {new Date(user.created_at).toLocaleDateString("th-TH")}
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8 rounded-full">
                          <MoreHorizontal size={16} />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        <DropdownMenuItem
                          onClick={() => router.push(`/dashboard/users/${user.id}`)}
                        >
                          View details
                        </DropdownMenuItem>
                        {isAdmin && user.status === "pending" && (
                          <DropdownMenuItem
                            onClick={() => handleResendInvite(user.id)}
                          >
                            Resend Invite
                          </DropdownMenuItem>
                        )}
                        {canSuspend(user) && user.status === "active" && (
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => handleSuspend(user.id)}
                          >
                            Suspend
                          </DropdownMenuItem>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Invite User Dialog */}
      <Dialog open={showInvite} onOpenChange={setShowInvite}>
        <DialogContent className="sm:max-w-[440px] rounded-[10px]">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-semi-black">
              Invite User
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleInvite} className="space-y-4 mt-2">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Display Name</Label>
              <Input
                required
                placeholder="สมชาย ใจดี"
                value={form.display_name}
                onChange={(e) => setForm((f) => ({ ...f, display_name: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Email</Label>
              <Input
                type="email"
                required
                placeholder="user@company.co.th"
                value={form.email}
                onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Role</Label>
              <select
                value={inviteRole}
                onChange={e => setInviteRole(e.target.value as "admin" | "user")}
                className="w-full h-12 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] px-4 text-sm text-semi-black"
              >
                <option value="user">User</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <p className="text-xs text-semi-grey">
              Additional roles can be assigned after the user accepts their invitation.
            </p>
            <DialogFooter className="pt-2">
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
                Send Invite
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
