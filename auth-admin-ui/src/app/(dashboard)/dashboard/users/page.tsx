"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  Plus,
  Search,
  MoreHorizontal,
  Loader2,
  Users as UsersIcon,
  UserCheck2,
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
import { useAuth } from "@/contexts/auth";
import {
  userApi,
  roleApi,
  type User,
  type Role,
  type InviteUserRequest,
  ApiError,
} from "@/lib/api";

export default function UsersPage() {
  const router = useRouter();
  const { getToken, tenantId } = useAuth();
  const [users, setUsers] = useState<User[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [showInvite, setShowInvite] = useState(false);
  const [inviting, setInviting] = useState(false);
  const [form, setForm] = useState<InviteUserRequest>({
    email: "",
    first_name: "",
    last_name: "",
    role_ids: [],
  });

  useEffect(() => {
    load();
  }, [tenantId]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token || !tenantId) return;
      const [usersResult, rolesResult] = await Promise.all([
        userApi.list(token, tenantId, { page_size: 100 }),
        roleApi.list(token, tenantId),
      ]);
      setUsers(usersResult.data);
      setRoles(rolesResult);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load users");
    } finally {
      setLoading(false);
    }
  }

  async function handleInvite(e: React.FormEvent) {
    e.preventDefault();
    setInviting(true);
    try {
      const token = await getToken();
      if (!token || !tenantId) return;
      await userApi.invite(form, token, tenantId);
      toast.success(`Invitation sent to ${form.email}`);
      setShowInvite(false);
      setForm({ email: "", first_name: "", last_name: "", role_ids: [] });
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
      if (!token || !tenantId) return;
      await userApi.suspend(id, token, tenantId);
      toast.success("User suspended");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  const filtered = users.filter((u) => {
    const matchSearch =
      u.email.toLowerCase().includes(search.toLowerCase()) ||
      `${u.first_name} ${u.last_name}`.toLowerCase().includes(search.toLowerCase());
    const matchRole =
      roleFilter === "all" ||
      (roleFilter === "applicant"
        ? u.roles.some((r) => r.name === "applicant")
        : u.roles.some((r) => r.name !== "applicant"));
    return matchSearch && matchRole;
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
        <div className="relative flex-1 max-w-sm">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-semi-grey" />
          <Input
            placeholder="ค้นหาผู้ใช้..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm"
          />
        </div>

        {/* Role filter chips */}
        <div className="flex gap-2">
          {["all", "users", "applicant"].map((f) => (
            <button
              key={f}
              onClick={() => setRoleFilter(f)}
              className={`px-3 py-1.5 rounded-[1000px] text-xs font-medium border transition-colors ${
                roleFilter === f
                  ? "bg-tiger-red text-white border-tiger-red"
                  : "border-border text-semi-grey hover:border-tiger-red hover:text-tiger-red"
              }`}
            >
              {f === "all" ? "All" : f === "applicant" ? "Applicants" : "HR Users"}
            </button>
          ))}
        </div>

        <Button
          onClick={() => setShowInvite(true)}
          className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm h-10 px-4 ml-auto"
        >
          <Plus size={16} className="mr-1.5" />
          Invite User
        </Button>
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
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">MFA</TableHead>
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
                        {user.first_name} {user.last_name}
                      </p>
                      <p className="text-xs text-semi-grey">{user.email}</p>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {user.roles.map((role) => (
                        <Badge
                          key={role.id}
                          variant="outline"
                          className={`text-[10px] ${
                            role.name === "super_admin"
                              ? "border-tiger-red text-tiger-red"
                              : role.name === "applicant"
                              ? "border-blue-400 text-blue-600"
                              : "border-border text-semi-grey"
                          }`}
                        >
                          {role.name}
                        </Badge>
                      ))}
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
                  <TableCell>
                    {user.mfa_enabled ? (
                      <UserCheck2 size={15} className="text-green-600" />
                    ) : (
                      <span className="text-xs text-semi-grey">—</span>
                    )}
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
                        <DropdownMenuItem
                          className="text-destructive focus:text-destructive"
                          onClick={() => handleSuspend(user.id)}
                        >
                          Suspend
                        </DropdownMenuItem>
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
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-semi-black">First Name</Label>
                <Input
                  required
                  placeholder="สมชาย"
                  value={form.first_name}
                  onChange={(e) => setForm((f) => ({ ...f, first_name: e.target.value }))}
                  className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-sm font-medium text-semi-black">Last Name</Label>
                <Input
                  required
                  placeholder="ใจดี"
                  value={form.last_name}
                  onChange={(e) => setForm((f) => ({ ...f, last_name: e.target.value }))}
                  className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11"
                />
              </div>
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
            <div className="space-y-2">
              <Label className="text-sm font-medium text-semi-black">Roles</Label>
              <div className="space-y-1.5">
                {roles
                  .filter((r) => !["super_admin"].includes(r.name))
                  .map((role) => (
                    <label
                      key={role.id}
                      className="flex items-center gap-3 cursor-pointer rounded-[10px] border border-border p-2.5 hover:bg-[#fafafa] transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={form.role_ids.includes(role.id)}
                        onChange={() =>
                          setForm((f) => ({
                            ...f,
                            role_ids: f.role_ids.includes(role.id)
                              ? f.role_ids.filter((id) => id !== role.id)
                              : [...f.role_ids, role.id],
                          }))
                        }
                        className="accent-tiger-red w-4 h-4"
                      />
                      <div>
                        <p className="text-sm text-semi-black font-medium">{role.name}</p>
                        <p className="text-xs text-semi-grey">{role.description}</p>
                      </div>
                    </label>
                  ))}
              </div>
            </div>
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
