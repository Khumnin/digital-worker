"use client";

import { Trash2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { type Role } from "@/lib/api";

interface RoleCardProps {
  role: Role;
  onDelete?: (id: string, name: string) => void;
}

/**
 * Mobile card representation of a single role row.
 * Shown only on viewports below the `md` breakpoint via `block md:hidden` on
 * the parent container. Mirrors the desktop table columns: name, type, module,
 * description, and created date.
 */
export function RoleCard({ role, onDelete }: RoleCardProps) {
  return (
    <div className="bg-card rounded-[10px] border border-border p-4 space-y-3">
      {/* Row 1: Role name + type badge + optional delete */}
      <div className="flex items-start gap-3">
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-semi-black font-mono truncate">
            {role.name}
          </p>
        </div>

        {/* Type badge */}
        {role.is_system ? (
          <Badge
            variant="outline"
            className="text-[10px] border-tiger-red text-tiger-red shrink-0"
          >
            system
          </Badge>
        ) : (
          <Badge
            variant="outline"
            className="text-[10px] border-border text-semi-grey shrink-0"
          >
            custom
          </Badge>
        )}

        {/* Delete action — touch target 44px, system roles have no delete */}
        {!role.is_system && onDelete && (
          <Button
            variant="ghost"
            size="icon"
            className="h-11 w-11 rounded-full text-semi-grey hover:text-destructive -mr-1 -mt-1 shrink-0"
            onClick={() => onDelete(role.id, role.name)}
            aria-label={`Delete role ${role.name}`}
          >
            <Trash2 size={15} />
          </Button>
        )}
      </div>

      {/* Row 2: Module */}
      {role.module && (
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-semi-grey w-16 shrink-0">
            Module
          </span>
          <Badge
            variant="outline"
            className="text-[10px] border-indigo-300 text-indigo-600"
          >
            {role.module}
          </Badge>
        </div>
      )}

      {/* Row 3: Description */}
      {role.description && (
        <div className="flex items-start gap-2">
          <span className="text-xs font-medium text-semi-grey w-16 shrink-0 pt-px">
            Desc
          </span>
          <p className="text-xs text-semi-grey line-clamp-2">{role.description}</p>
        </div>
      )}

      {/* Row 4: Created date */}
      <p className="text-xs text-semi-grey">
        Created {new Date(role.created_at).toLocaleDateString("th-TH")}
      </p>
    </div>
  );
}
