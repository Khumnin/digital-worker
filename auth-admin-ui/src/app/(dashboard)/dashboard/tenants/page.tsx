"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Plus, Search, MoreHorizontal, Loader2, Building2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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
import { Label } from "@/components/ui/label";
import { useAuth } from "@/contexts/auth";
import { tenantApi, type Tenant, type CreateTenantRequest, ApiError } from "@/lib/api";

const MODULE_OPTIONS = [
  { id: "recruit", label: "Recruit" },
];

export default function TenantsPage() {
  const router = useRouter();
  const { getToken, isSuperAdmin } = useAuth();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<CreateTenantRequest>({
    name: "",
    slug: "",
    admin_email: "",
    config: { enabled_modules: [] },
  });

  useEffect(() => {
    if (!isSuperAdmin) {
      router.replace("/dashboard");
      return;
    }
    load();
  }, [isSuperAdmin]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const result = await tenantApi.list(token);
      setTenants(result.data);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load tenants");
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      const token = await getToken();
      if (!token) return;
      await tenantApi.create(form, token);
      toast.success(`Tenant "${form.name}" provisioned`);
      setShowCreate(false);
      setForm({ name: "", slug: "", admin_email: "", config: { enabled_modules: [] } });
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to provision tenant");
    } finally {
      setCreating(false);
    }
  }

  const toggleModule = (id: string) => {
    setForm((f) => {
      const modules = f.config?.enabled_modules ?? [];
      return {
        ...f,
        config: {
          ...f.config,
          enabled_modules: modules.includes(id)
            ? modules.filter((m) => m !== id)
            : [...modules, id],
        },
      };
    });
  };

  async function handleSuspend(id: string) {
    try {
      const token = await getToken();
      if (!token) return;
      await tenantApi.suspend(id, token);
      toast.success("Tenant suspended");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  async function handleActivate(id: string) {
    try {
      const token = await getToken();
      if (!token) return;
      await tenantApi.activate(id, token);
      toast.success("Tenant activated");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  const filtered = tenants.filter(
    (t) =>
      t.name.toLowerCase().includes(search.toLowerCase()) ||
      t.slug.toLowerCase().includes(search.toLowerCase())
  );

  const statusColor: Record<string, string> = {
    active: "bg-[#EDFBF5] dark:bg-green-900/30 text-[#34D186] border-[#34D186]/40",
    suspended: "bg-red-100 dark:bg-red-900/30 text-tiger-red dark:text-red-400 border-red-200 dark:border-red-800",
    pending: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-700 dark:text-yellow-400 border-yellow-200 dark:border-yellow-800",
  };

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search size={15} className="absolute left-3 top-1/2 -translate-y-1/2 text-semi-grey" />
          <Input
            placeholder="ค้นหา tenant..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 h-10 rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input text-sm"
          />
        </div>
        <Button
          onClick={() => setShowCreate(true)}
          className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm h-10 px-4"
        >
          <Plus size={16} className="mr-1.5" />
          Provision Tenant
        </Button>
      </div>

      {/* Table */}
      <div className="bg-card rounded-[10px] border border-border overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="animate-spin text-tiger-red" size={24} />
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <Building2 size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No tenants found</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-[#fafafa] dark:bg-[#1a2332] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]">
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Name</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Slug</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Status</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Modules</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Created</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((tenant) => (
                <TableRow
                  key={tenant.id}
                  className="cursor-pointer bg-white dark:bg-[#1E2533] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]"
                  onClick={() => router.push(`/dashboard/tenants/${tenant.id}`)}
                >
                  <TableCell className="font-medium text-semi-black text-sm">
                    {tenant.name}
                  </TableCell>
                  <TableCell className="text-sm text-semi-grey font-mono">
                    {tenant.slug}
                  </TableCell>
                  <TableCell>
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium border ${
                        statusColor[tenant.status] ?? ""
                      }`}
                    >
                      {tenant.status}
                    </span>
                  </TableCell>
                  <TableCell>
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
                  </TableCell>
                  <TableCell className="text-xs text-semi-grey">
                    {new Date(tenant.created_at).toLocaleDateString("th-TH")}
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
                          onClick={() => router.push(`/dashboard/tenants/${tenant.id}`)}
                        >
                          View details
                        </DropdownMenuItem>
                        {tenant.status === "active" ? (
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => handleSuspend(tenant.id)}
                          >
                            Suspend
                          </DropdownMenuItem>
                        ) : (
                          <DropdownMenuItem onClick={() => handleActivate(tenant.id)}>
                            Activate
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

      {/* Create Tenant Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-[500px] rounded-[10px]">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-semi-black">
              Provision New Tenant
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4 mt-2">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">
                Company Name
              </Label>
              <Input
                required
                placeholder="Acme Corporation"
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">
                Slug
                <span className="text-semi-grey font-normal ml-1">(unique identifier)</span>
              </Label>
              <Input
                required
                placeholder="acme"
                pattern="[a-z0-9\-]+"
                value={form.slug}
                onChange={(e) =>
                  setForm((f) => ({ ...f, slug: e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "") }))
                }
                className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11 font-mono"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">
                Admin Email
                <span className="text-semi-grey font-normal ml-1">(initial admin user)</span>
              </Label>
              <Input
                required
                type="email"
                placeholder="admin@acme.co.th"
                value={form.admin_email}
                onChange={(e) => setForm((f) => ({ ...f, admin_email: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
              />
            </div>

            {/* Module selector */}
            <div className="space-y-2">
              <Label className="text-sm font-medium text-semi-black">
                Enabled Modules
              </Label>
              <div className="space-y-2">
                {MODULE_OPTIONS.map((mod) => {
                  const checked = form.config?.enabled_modules?.includes(mod.id) ?? false;
                  return (
                    <label
                      key={mod.id}
                      className="flex items-center gap-3 cursor-pointer rounded-[10px] border border-border p-3 hover:bg-[#fafafa] dark:hover:bg-[#1a2332] transition-colors"
                    >
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() => toggleModule(mod.id)}
                        className="accent-tiger-red w-4 h-4 rounded"
                      />
                      <span className="text-sm text-semi-black">{mod.label}</span>
                      {checked && (
                        <Badge
                          variant="outline"
                          className="ml-auto text-[10px] border-tiger-red text-tiger-red uppercase"
                        >
                          Enabled
                        </Badge>
                      )}
                    </label>
                  );
                })}
              </div>
            </div>

            <DialogFooter className="pt-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setShowCreate(false)}
                className="rounded-[1000px]"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={creating}
                className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white"
              >
                {creating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Provision
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
