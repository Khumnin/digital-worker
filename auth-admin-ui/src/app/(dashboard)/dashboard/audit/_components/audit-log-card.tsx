"use client";

import { type AuditLog } from "@/lib/api";

interface AuditLogCardProps {
  log: AuditLog;
  actionColors: Record<string, string>;
}

/**
 * Mobile card representation of a single audit log entry.
 * Shown only on viewports below the `md` breakpoint via `block md:hidden` on
 * the parent container. Mirrors the desktop table columns: time, action, actor,
 * target, IP address.
 */
export function AuditLogCard({ log, actionColors }: AuditLogCardProps) {
  const colorClass =
    actionColors[log.action] ??
    "text-semi-black dark:text-gray-300 bg-[#f0f0f0] dark:bg-gray-700/40";

  const formattedTime = (() => {
    const d = new Date(log.created_at);
    return isNaN(d.getTime())
      ? log.created_at || "—"
      : new Intl.DateTimeFormat("en-US", {
          year: "numeric",
          month: "2-digit",
          day: "2-digit",
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
          hour12: false,
          timeZone: "Asia/Bangkok",
        }).format(d);
  })();

  return (
    <div className="bg-card rounded-[10px] border border-border p-4 space-y-3">
      {/* Row 1: Action badge + timestamp */}
      <div className="flex items-start justify-between gap-3">
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium shrink-0 ${colorClass}`}
        >
          {log.action}
        </span>
        <span className="text-xs text-semi-grey whitespace-nowrap tabular-nums">
          {formattedTime}
        </span>
      </div>

      {/* Row 2: Actor */}
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-semi-grey w-12 shrink-0">
          Actor
        </span>
        <span className="text-xs text-semi-black truncate">
          {log.actor_email || log.actor_id || "—"}
        </span>
      </div>

      {/* Row 3: Target */}
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-semi-grey w-12 shrink-0">
          Target
        </span>
        <span className="text-xs text-semi-grey truncate">
          {log.target_email ||
            (log.target_id ? `${log.target_id.slice(0, 8)}…` : "—")}
        </span>
      </div>

      {/* Row 4: IP address */}
      <div className="flex items-center gap-2">
        <span className="text-xs font-medium text-semi-grey w-12 shrink-0">
          IP
        </span>
        <span className="text-xs text-semi-grey font-mono">
          {log.ip_address || "—"}
        </span>
      </div>
    </div>
  );
}
