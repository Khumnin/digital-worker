"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, ScrollText, ChevronLeft, ChevronRight } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAuth } from "@/contexts/auth";
import { auditApi, type AuditLog, ApiError } from "@/lib/api";
import { AuditLogCard } from "./_components/audit-log-card";

const PAGE_SIZE = 25;

const ACTION_OPTIONS = [
  { value: "all", label: "All Actions" },
  { value: "USER_LOGIN", label: "USER_LOGIN" },
  { value: "USER_LOGOUT", label: "USER_LOGOUT" },
  { value: "USER_INVITED", label: "USER_INVITED" },
  { value: "USER_ENABLED", label: "USER_ENABLED" },
  { value: "USER_DISABLED", label: "USER_DISABLED" },
  { value: "ROLE_ASSIGNED", label: "ROLE_ASSIGNED" },
  { value: "TENANT_SUSPENDED", label: "TENANT_SUSPENDED" },
  { value: "TENANT_ACTIVATED", label: "TENANT_ACTIVATED" },
];

export default function AuditPage() {
  const { getToken } = useAuth();
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [actionFilter, setActionFilter] = useState("all");
  const [fromDate, setFromDate] = useState("");
  const [toDate, setToDate] = useState("");

  useEffect(() => {
    load();
  }, [page]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load(targetPage?: number) {
    const currentPage = targetPage ?? page;
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const result = await auditApi.list(token, {
        page: currentPage,
        page_size: PAGE_SIZE,
        ...(actionFilter && actionFilter !== "all" ? { action: actionFilter } : {}),
        ...(fromDate ? { from: fromDate } : {}),
        ...(toDate ? { to: toDate } : {}),
      });
      setLogs(result.data);
      setTotal(result.total);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load audit log");
    } finally {
      setLoading(false);
    }
  }

  const handleApply = (e: React.FormEvent) => {
    e.preventDefault();
    setPage(1);
    load(1);
  };

  const totalPages = Math.ceil(total / PAGE_SIZE);

  const actionColors: Record<string, string> = {
    USER_LOGIN: "text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30",
    USER_LOGOUT: "text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30",
    USER_INVITED: "text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-purple-900/30",
    USER_ENABLED: "text-teal-600 dark:text-teal-400 bg-teal-50 dark:bg-teal-900/30",
    USER_DISABLED: "text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-900/30",
    ROLE_ASSIGNED: "text-teal-600 dark:text-teal-400 bg-teal-50 dark:bg-teal-900/30",
    TENANT_SUSPENDED: "text-tiger-red dark:text-red-400 bg-red-50 dark:bg-red-900/30",
    TENANT_ACTIVATED: "text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30",
    // legacy lowercase keys for backward compat
    login: "text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30",
    logout: "text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30",
    register: "text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-purple-900/30",
    "password.change": "text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-900/30",
    "user.suspend": "text-tiger-red dark:text-red-400 bg-red-50 dark:bg-red-900/30",
    "tenant.create": "text-indigo-600 dark:text-indigo-400 bg-indigo-50 dark:bg-indigo-900/30",
    "role.assign": "text-teal-600 dark:text-teal-400 bg-teal-50 dark:bg-teal-900/30",
  };

  return (
    <div className="space-y-4">
      {/* Filter bar — stacks vertically on mobile, inline on desktop */}
      <form onSubmit={handleApply} className="flex flex-col sm:flex-row sm:items-end gap-3">
        {/* Action dropdown — full width on mobile */}
        <div className="space-y-1">
          <Label className="text-xs text-semi-grey font-medium">Action</Label>
          <Select
            value={actionFilter}
            onValueChange={(val) => setActionFilter(val)}
          >
            <SelectTrigger className="w-full sm:w-[190px] h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm">
              <SelectValue placeholder="All Actions" />
            </SelectTrigger>
            <SelectContent>
              {ACTION_OPTIONS.map((opt) => (
                <SelectItem key={opt.value} value={opt.value}>
                  {opt.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {/* Date range filters — stack vertically on mobile, side-by-side on sm+ */}
        <div className="flex flex-col sm:flex-row sm:items-end gap-2">
          <div className="space-y-1">
            <Label className="text-xs text-semi-grey font-medium">From</Label>
            <Input
              type="date"
              value={fromDate}
              onChange={(e) => setFromDate(e.target.value)}
              className="w-full sm:w-[148px] h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs text-semi-grey font-medium">To</Label>
            <Input
              type="date"
              value={toDate}
              onChange={(e) => setToDate(e.target.value)}
              className="w-full sm:w-[148px] h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm"
            />
          </div>
        </div>

        <Button
          type="submit"
          variant="outline"
          className="w-full sm:w-auto rounded-[1000px] h-10 text-sm"
        >
          Apply
        </Button>
      </form>

      {/* Desktop table */}
      <div className="hidden md:block bg-card rounded-[10px] border border-border overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="animate-spin text-tiger-red" size={24} />
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <ScrollText size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No audit events found</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-[#fafafa] dark:bg-[#1a2332] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]">
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">
                  Time
                </TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">
                  Action
                </TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">
                  Actor
                </TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">
                  Target
                </TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">
                  IP
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log) => {
                const colorClass =
                  actionColors[log.action] ?? "text-semi-black dark:text-gray-300 bg-[#f0f0f0] dark:bg-gray-700/40";
                return (
                  <TableRow key={log.id} className="bg-white dark:bg-[#1E2533] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]">
                    <TableCell className="text-xs text-semi-grey whitespace-nowrap">
                      {(() => {
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
                      })()}
                    </TableCell>
                    <TableCell>
                      <span
                        className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${colorClass}`}
                      >
                        {log.action}
                      </span>
                    </TableCell>
                    <TableCell className="text-xs text-semi-black max-w-[180px] truncate">
                      {log.actor_email || log.actor_id || "—"}
                    </TableCell>
                    <TableCell className="text-xs text-semi-grey max-w-[160px] truncate">
                      {log.target_email || (log.target_id ? `${log.target_id.slice(0, 8)}…` : "—")}
                    </TableCell>
                    <TableCell className="text-xs text-semi-grey font-mono">
                      {log.ip_address || "—"}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      {/* Mobile card list */}
      <div className="block md:hidden">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="animate-spin text-tiger-red" size={24} />
          </div>
        ) : logs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <ScrollText size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No audit events found</p>
          </div>
        ) : (
          <div className="space-y-2">
            {logs.map((log) => (
              <AuditLogCard
                key={log.id}
                log={log}
                actionColors={actionColors}
              />
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between text-sm text-semi-grey">
          <span>
            {(page - 1) * PAGE_SIZE + 1}–{Math.min(page * PAGE_SIZE, total)} of {total}
          </span>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8 rounded-full"
              disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
            >
              <ChevronLeft size={14} />
            </Button>
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8 rounded-full"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
            >
              <ChevronRight size={14} />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
