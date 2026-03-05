"use client";

import { MoreHorizontal } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { type Tenant } from "@/lib/api";

interface TenantCardProps {
  tenant: Tenant;
  statusColor: Record<string, string>;
  onView: (id: string) => void;
  onSuspend: (id: string) => void;
  onActivate: (id: string) => void;
}

/**
 * Mobile card representation of a single tenant row.
 * Shown only on viewports below the `md` breakpoint via `block md:hidden` on the parent.
 */
export function TenantCard({
  tenant,
  statusColor,
  onView,
  onSuspend,
  onActivate,
}: TenantCardProps) {
  return (
    <div
      onClick={() => onView(tenant.id)}
      className="bg-card rounded-[10px] border border-border p-4 space-y-3 active:bg-[#fafafa] dark:active:bg-[#1a2332] cursor-pointer transition-colors"
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onView(tenant.id);
        }
      }}
      aria-label={`View details for ${tenant.name}`}
    >
      {/* Row 1: Name/Slug + Status + Actions dropdown */}
      <div className="flex items-start gap-2">
        {/* Name + Slug */}
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-semi-black truncate">
            {tenant.name}
          </p>
          <p className="text-xs text-semi-grey font-mono truncate">
            {tenant.slug}
          </p>
        </div>

        {/* Status + Actions */}
        <div className="flex items-center gap-2 shrink-0">
          <span
            className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${
              statusColor[tenant.status] ?? ""
            }`}
          >
            {tenant.status}
          </span>

          {/* Actions dropdown — stop propagation so card click doesn't fire */}
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                onClick={(e) => e.stopPropagation()}
                onKeyDown={(e) => e.stopPropagation()}
                className="flex items-center justify-center w-11 h-11 rounded-full hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] -mr-2 transition-colors"
                aria-label="Tenant actions"
              >
                <MoreHorizontal size={16} className="text-semi-grey" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
              <DropdownMenuItem onClick={() => onView(tenant.id)}>
                View details
              </DropdownMenuItem>
              {tenant.status === "active" ? (
                <DropdownMenuItem
                  className="text-destructive focus:text-destructive"
                  onClick={() => onSuspend(tenant.id)}
                >
                  Suspend
                </DropdownMenuItem>
              ) : (
                <DropdownMenuItem onClick={() => onActivate(tenant.id)}>
                  Activate
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Row 2: Module badges */}
      {(tenant.enabled_modules ?? []).length > 0 && (
        <div className="flex flex-wrap gap-1">
          {(tenant.enabled_modules ?? []).map((mod) => (
            <Badge
              key={mod}
              variant="outline"
              className="text-[10px] border-tiger-red text-tiger-red uppercase"
            >
              {mod}
            </Badge>
          ))}
        </div>
      )}

      {/* Row 3: Created date */}
      <p className="text-xs text-semi-grey">
        Created {new Date(tenant.created_at).toLocaleDateString("th-TH")}
      </p>
    </div>
  );
}
