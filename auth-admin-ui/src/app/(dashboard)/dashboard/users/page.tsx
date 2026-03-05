"use client";

import { useEffect, useState, useCallback, useRef } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  Plus,
  Search,
  MoreHorizontal,
  Loader2,
  Users as UsersIcon,
  X,
  Mail,
  Filter,
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
import { InviteUserDialog } from "@/components/invite-user-dialog";
import { UserCard } from "./_components/user-card";
import {
  userApi,
  type User,
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
  const [pendingCount, setPendingCount] = useState<number>(0);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [moduleFilter, setModuleFilter] = useState<string>("all");
  const [showInvite, setShowInvite] = useState(false);
  // Tracks whether the user just clicked the pending banner so we can show a
  // transitional "Filtering..." state instead of abruptly hiding the banner.
  const [pendingBannerApplying, setPendingBannerApplying] = useState(false);
  const bannerApplyingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) {
        setLoading(false);
        return;
      }

      // When a specific status filter is active, prefer server-side filtering.
      // Otherwise fetch all users so client-side module filtering still works.
      const statusParam =
        statusFilter !== "all" ? statusFilter : undefined;

      const [usersResult, pendingResult] = await Promise.all([
        userApi.list(token, { page_size: 100, status: statusParam }),
        // Always fetch the pending count independently so the banner stays
        // accurate even while another status filter is active.
        statusFilter !== "pending"
          ? userApi.list(token, { page: 1, page_size: 1, status: "pending" })
          : null,
      ]);

      setUsers(usersResult.data);
      // When the status filter is "pending" the main result already contains
      // the count; otherwise use the dedicated pending query.
      setPendingCount(
        statusFilter === "pending"
          ? usersResult.total
          : (pendingResult?.total ?? 0)
      );
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load users");
    } finally {
      setLoading(false);
      // Banner transition complete — clear the "applying" state regardless of
      // whether the fetch succeeded or failed.
      setPendingBannerApplying(false);
      if (bannerApplyingTimerRef.current) {
        clearTimeout(bannerApplyingTimerRef.current);
        bannerApplyingTimerRef.current = null;
      }
    }
  }, [getToken, statusFilter]);

  useEffect(() => {
    load();
  }, [load]);

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

  function handlePendingBannerClick() {
    // Mark the transition as "applying" immediately so the banner transforms
    // into a loading state rather than vanishing before results arrive.
    setPendingBannerApplying(true);
    // Safety valve: if the network is very slow, clear the applying state after
    // 8 s so the banner doesn't persist indefinitely.
    bannerApplyingTimerRef.current = setTimeout(() => {
      setPendingBannerApplying(false);
    }, 8000);
    setStatusFilter("pending");
  }

  function clearFilters() {
    setStatusFilter("all");
    setModuleFilter("all");
  }

  const hasActiveFilters = statusFilter !== "all" || moduleFilter !== "all";

  // Client-side filters.
  // When a specific status is selected the API already returns only those
  // users, so we only apply the status filter locally when showing "all".
  const filtered = users.filter((u) => {
    const matchSearch =
      u.email.toLowerCase().includes(search.toLowerCase()) ||
      (u.display_name ?? "").toLowerCase().includes(search.toLowerCase());
    const matchModule =
      moduleFilter === "all" ||
      Object.keys(u.module_roles ?? {}).includes(moduleFilter);
    return matchSearch && matchModule;
  });

  const statusColor: Record<string, string> = {
    active: "bg-[#EDFBF5] dark:bg-green-900/30 text-[#34D186] border-[#34D186]/40",
    inactive: "bg-gray-100 dark:bg-gray-800 text-semi-grey border-gray-200 dark:border-gray-700",
    pending: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800",
  };

  return (
    <div className="space-y-4">
      {/* Pending invitations banner
          — visible when there are pending users AND the admin has not yet
            applied the pending filter (statusFilter !== "pending").
          — while the filter is being applied (pendingBannerApplying) we keep
            the banner mounted but switch it to a "Filtering…" state so the
            user gets immediate feedback instead of seeing it vanish with no
            visible result. */}
      {!loading && pendingCount > 0 && statusFilter !== "pending" && isAdmin && (
        <button
          type="button"
          onClick={handlePendingBannerClick}
          disabled={pendingBannerApplying}
          className="w-full flex items-center gap-3 px-4 py-3 rounded-[10px] border border-yellow-200 dark:border-yellow-800/60 bg-yellow-50 dark:bg-yellow-900/20 text-left transition-colors group disabled:cursor-default"
          aria-label={
            pendingBannerApplying
              ? "Applying pending filter…"
              : `View ${pendingCount} pending invitation${pendingCount !== 1 ? "s" : ""}`
          }
          aria-live="polite"
          aria-busy={pendingBannerApplying}
        >
          {/* Icon — swap to spinner while applying */}
          <span className="flex items-center justify-center w-8 h-8 rounded-full bg-yellow-100 dark:bg-yellow-800/40 shrink-0">
            {pendingBannerApplying ? (
              <Loader2 size={15} className="animate-spin text-yellow-600 dark:text-yellow-400" />
            ) : (
              <Mail size={15} className="text-yellow-600 dark:text-yellow-400" />
            )}
          </span>

          <div className="flex-1 min-w-0">
            {pendingBannerApplying ? (
              <>
                <p className="text-sm font-medium text-[#0B1F3A] dark:text-foreground leading-tight flex items-center gap-1.5">
                  <Filter size={13} className="text-yellow-600 dark:text-yellow-400 shrink-0" />
                  Filtering by pending status…
                </p>
                <p className="text-xs text-semi-grey mt-0.5">
                  Loading pending invitations
                </p>
              </>
            ) : (
              <>
                <p className="text-sm font-medium text-[#0B1F3A] dark:text-foreground leading-tight">
                  {pendingCount} pending invitation{pendingCount !== 1 ? "s" : ""} awaiting acceptance
                </p>
                <p className="text-xs text-semi-grey mt-0.5">
                  Click to filter and view pending users
                </p>
              </>
            )}
          </div>

          {/* Badge count — fade out while applying */}
          <span
            className={`inline-flex items-center justify-center min-w-[1.5rem] h-6 px-1.5 rounded-full bg-yellow-200 dark:bg-yellow-700 text-yellow-800 dark:text-yellow-200 text-xs font-semibold shrink-0 transition-opacity ${
              pendingBannerApplying
                ? "opacity-40"
                : "group-hover:bg-yellow-300 dark:group-hover:bg-yellow-600"
            }`}
          >
            {pendingCount}
          </span>
        </button>
      )}

      {/* "Applying" banner shown while loading === true (after banner click).
          This keeps something visible in the slot while the spinner in the
          table body below is loading, giving the user a continuous trail. */}
      {pendingBannerApplying && loading && (
        <div
          role="status"
          aria-label="Applying pending filter…"
          className="w-full flex items-center gap-3 px-4 py-3 rounded-[10px] border border-yellow-200 dark:border-yellow-800/60 bg-yellow-50 dark:bg-yellow-900/20"
        >
          <span className="flex items-center justify-center w-8 h-8 rounded-full bg-yellow-100 dark:bg-yellow-800/40 shrink-0">
            <Loader2 size={15} className="animate-spin text-yellow-600 dark:text-yellow-400" />
          </span>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-medium text-[#0B1F3A] dark:text-foreground leading-tight flex items-center gap-1.5">
              <Filter size={13} className="text-yellow-600 dark:text-yellow-400 shrink-0" />
              Filtering by pending status…
            </p>
            <p className="text-xs text-semi-grey mt-0.5">
              Loading pending invitations
            </p>
          </div>
          <span className="inline-flex items-center justify-center min-w-[1.5rem] h-6 px-1.5 rounded-full bg-yellow-200 dark:bg-yellow-700 text-yellow-800 dark:text-yellow-200 text-xs font-semibold shrink-0 opacity-40">
            {pendingCount}
          </span>
        </div>
      )}

      {/* Responsive Toolbar */}
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center sm:gap-3">
        {/* Search — full width on mobile */}
        <div className="relative w-full sm:flex-1 sm:min-w-[200px] sm:max-w-sm">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-semi-grey" />
          <Input
            placeholder="ค้นหาผู้ใช้..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm"
          />
        </div>

        {/* Filters — 2-col grid on mobile, inline on sm+ */}
        <div className="grid grid-cols-2 gap-2 sm:flex sm:items-center sm:gap-3">
          {/* Status filter */}
          <div className="flex items-center gap-1.5">
            <Label className="text-xs text-semi-grey font-medium whitespace-nowrap hidden sm:block">Status</Label>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm w-full sm:min-w-[140px]">
                <SelectValue placeholder="Status" />
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
            <Label className="text-xs text-semi-grey font-medium whitespace-nowrap hidden sm:block">Module</Label>
            <Select value={moduleFilter} onValueChange={setModuleFilter}>
              <SelectTrigger className="h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm w-full sm:min-w-[140px]">
                <SelectValue placeholder="Module" />
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
        </div>

        {/* Clear filters */}
        {hasActiveFilters && (
          <Button
            variant="ghost"
            size="sm"
            onClick={clearFilters}
            className="h-10 rounded-[1000px] text-xs text-semi-grey hover:text-semi-black gap-1 w-full sm:w-auto"
          >
            <X size={13} />
            Clear filters
          </Button>
        )}

        {/* Invite User button — full width on mobile */}
        {isAdmin && (
          <Button
            onClick={() => setShowInvite(true)}
            className="w-full sm:w-auto sm:ml-auto rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm h-10 px-4"
          >
            <Plus size={16} className="mr-1.5" />
            Invite User
          </Button>
        )}
      </div>

      {/* Desktop table — hidden on mobile */}
      <div className="hidden md:block bg-card rounded-[10px] border border-border overflow-hidden">
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
              <TableRow className="bg-[#fafafa] dark:bg-[#1a2332] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]">
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
                  className="cursor-pointer bg-white dark:bg-[#1E2533] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]"
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

      {/* Mobile card stack — visible only below md breakpoint */}
      <div className="block md:hidden">
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
          <div className="space-y-3">
            {filtered.map((user) => (
              <UserCard
                key={user.id}
                user={user}
                isAdmin={isAdmin}
                canSuspend={canSuspend(user)}
                statusColor={statusColor}
                onView={(id) => router.push(`/dashboard/users/${id}`)}
                onSuspend={handleSuspend}
                onResendInvite={handleResendInvite}
              />
            ))}
          </div>
        )}
      </div>

      {/* Invite User Dialog */}
      <InviteUserDialog
        open={showInvite}
        onOpenChange={setShowInvite}
        onSuccess={load}
      />
    </div>
  );
}
