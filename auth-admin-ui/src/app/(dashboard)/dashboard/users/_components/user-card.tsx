"use client";

import { MoreHorizontal } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { type User } from "@/lib/api";

interface UserCardProps {
  user: User;
  isAdmin: boolean;
  canSuspend: boolean;
  /** When true, a "Tenant" row is rendered on the card (Super Admin view only). */
  showTenant?: boolean;
  statusColor: Record<string, string>;
  onView: (id: string) => void;
  onSuspend?: (id: string) => void;
  onResendInvite?: (id: string) => void;
}

/**
 * Mobile card representation of a single user row.
 * Shown only on viewports below the `md` breakpoint via `block md:hidden` on the parent.
 */
export function UserCard({
  user,
  isAdmin,
  canSuspend,
  showTenant = false,
  statusColor,
  onView,
  onSuspend,
  onResendInvite,
}: UserCardProps) {
  const initial = (user.display_name || user.email).charAt(0).toUpperCase();

  return (
    <div
      onClick={() => onView(user.id)}
      className="bg-card rounded-[10px] border border-border p-4 space-y-3 active:bg-[#fafafa] dark:active:bg-[#1a2332] cursor-pointer transition-colors"
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onView(user.id);
        }
      }}
      aria-label={`View details for ${user.display_name || user.email}`}
    >
      {/* Row 1: Avatar + Name/Email + Status badge + Actions dropdown */}
      <div className="flex items-start gap-3">
        {/* Avatar */}
        <div className="w-10 h-10 rounded-full bg-tiger-red/10 flex items-center justify-center shrink-0">
          <span className="text-sm font-semibold text-tiger-red">
            {initial}
          </span>
        </div>

        {/* Name + Email */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-semi-black truncate">
            {user.display_name || "—"}
          </p>
          <p className="text-xs text-semi-grey truncate">{user.email}</p>
        </div>

        {/* Status badge */}
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border shrink-0 ${
            statusColor[user.status] ?? ""
          }`}
        >
          {user.status}
        </span>

        {/* Actions dropdown — stop propagation so card click doesn't fire */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              onClick={(e) => e.stopPropagation()}
              onKeyDown={(e) => e.stopPropagation()}
              className="flex items-center justify-center w-11 h-11 rounded-full hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] shrink-0 -mr-2 -mt-1 transition-colors"
              aria-label="User actions"
            >
              <MoreHorizontal size={16} className="text-semi-grey" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
            <DropdownMenuItem onClick={() => onView(user.id)}>
              View details
            </DropdownMenuItem>
            {isAdmin && user.status === "pending" && onResendInvite && (
              <DropdownMenuItem onClick={() => onResendInvite(user.id)}>
                Resend Invite
              </DropdownMenuItem>
            )}
            {canSuspend && user.status === "active" && onSuspend && (
              <DropdownMenuItem
                className="text-destructive focus:text-destructive"
                onClick={() => onSuspend(user.id)}
              >
                Suspend
              </DropdownMenuItem>
            )}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>

      {/* Row 2: Role badges */}
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

      {/* Row 3: Tenant (Super Admin only) */}
      {showTenant && (
        <div className="flex items-center gap-1.5">
          <span className="text-[10px] font-medium text-semi-grey uppercase tracking-wide">
            Tenant
          </span>
          <span className="text-xs text-semi-black font-medium truncate">
            {user.tenant_name || "—"}
          </span>
        </div>
      )}

      {/* Row 4: Joined date */}
      <p className="text-xs text-semi-grey">
        Joined {new Date(user.created_at).toLocaleDateString("th-TH")}
      </p>
    </div>
  );
}
