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

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const result = await auditApi.list(token, {
        page,
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
    load();
  };

  const totalPages = Math.ceil(total / PAGE_SIZE);

  const actionColors: Record<string, string> = {
    USER_LOGIN: "text-green-600 bg-green-50",
    USER_LOGOUT: "text-blue-600 bg-blue-50",
    USER_INVITED: "text-purple-600 bg-purple-50",
    USER_ENABLED: "text-teal-600 bg-teal-50",
    USER_DISABLED: "text-orange-600 bg-orange-50",
    ROLE_ASSIGNED: "text-teal-600 bg-teal-50",
    TENANT_SUSPENDED: "text-tiger-red bg-red-50",
    TENANT_ACTIVATED: "text-green-600 bg-green-50",
    // legacy lowercase keys for backward compat
    login: "text-green-600 bg-green-50",
    logout: "text-blue-600 bg-blue-50",
    register: "text-purple-600 bg-purple-50",
    "password.change": "text-orange-600 bg-orange-50",
    "user.suspend": "text-tiger-red bg-red-50",
    "tenant.create": "text-indigo-600 bg-indigo-50",
    "role.assign": "text-teal-600 bg-teal-50",
  };

  return (
    <div className="space-y-4">
      {/* Filter bar */}
      <form onSubmit={handleApply} className="flex items-end gap-3 flex-wrap">
        {/* Action dropdown */}
        <div className="space-y-1">
          <Label className="text-xs text-semi-grey font-medium">Action</Label>
          <Select
            value={actionFilter}
            onValueChange={(val) => setActionFilter(val)}
          >
            <SelectTrigger className="h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm min-w-[190px]">
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

        {/* Date range filters */}
        <div className="flex items-end gap-2">
          <div className="space-y-1">
            <Label className="text-xs text-semi-grey font-medium">From</Label>
            <Input
              type="date"
              value={fromDate}
              onChange={(e) => setFromDate(e.target.value)}
              className="h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm w-[148px]"
            />
          </div>
          <div className="space-y-1">
            <Label className="text-xs text-semi-grey font-medium">To</Label>
            <Input
              type="date"
              value={toDate}
              onChange={(e) => setToDate(e.target.value)}
              className="h-10 rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] text-sm w-[148px]"
            />
          </div>
        </div>

        <Button
          type="submit"
          variant="outline"
          className="rounded-[1000px] h-10 text-sm"
        >
          Apply
        </Button>
      </form>

      {/* Table */}
      <div className="bg-white rounded-[10px] border border-border overflow-hidden">
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
              <TableRow className="bg-[#fafafa] hover:bg-[#fafafa]">
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
                  actionColors[log.action] ?? "text-semi-black bg-[#f0f0f0]";
                return (
                  <TableRow key={log.id} className="hover:bg-[#fafafa]">
                    <TableCell className="text-xs text-semi-grey whitespace-nowrap">
                      {new Date(log.created_at).toLocaleString("th-TH")}
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
                      {log.target_id ? `${log.target_id.slice(0, 8)}…` : "—"}
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
