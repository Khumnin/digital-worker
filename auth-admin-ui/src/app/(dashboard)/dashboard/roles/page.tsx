"use client";

import { useEffect, useState, useMemo } from "react";
import { toast } from "sonner";
import { Plus, Trash2, Loader2, ShieldCheck } from "lucide-react";
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
import { useAuth } from "@/contexts/auth";
import { roleApi, type Role, type CreateRoleRequest, ApiError } from "@/lib/api";
import { RoleCard } from "./_components/role-card";

export default function RolesPage() {
  const { getToken } = useAuth();
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [selectedModule, setSelectedModule] = useState<string>("all");
  const [form, setForm] = useState<CreateRoleRequest>({ name: "", description: "", module: "" });

  useEffect(() => {
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const result = await roleApi.list(token);
      setRoles(result);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load roles");
    } finally {
      setLoading(false);
    }
  }

  // Derive unique module names from roles (excluding null = system)
  const moduleNames = useMemo(() => {
    const mods = new Set<string>();
    roles.forEach((r) => {
      if (r.module) mods.add(r.module);
    });
    return Array.from(mods).sort();
  }, [roles]);

  // Tab list: All | System | <module1> | <module2> ...
  const tabs = useMemo(
    () => ["all", "system", ...moduleNames],
    [moduleNames]
  );

  // Filtered roles based on selected tab
  const filteredRoles = useMemo(() => {
    if (selectedModule === "all") return roles;
    if (selectedModule === "system") return roles.filter((r) => r.module === null);
    return roles.filter((r) => r.module === selectedModule);
  }, [roles, selectedModule]);

  const isModuleTab = selectedModule !== "all" && selectedModule !== "system";

  function openCreateDialog() {
    setForm({
      name: "",
      description: "",
      module: isModuleTab ? selectedModule : "",
    });
    setShowCreate(true);
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      const token = await getToken();
      if (!token) return;
      const payload: CreateRoleRequest = {
        name: form.name,
        description: form.description,
        ...(form.module ? { module: form.module } : {}),
      };
      await roleApi.create(payload, token);
      toast.success(`Role "${form.name}" created`);
      setShowCreate(false);
      setForm({ name: "", description: "", module: "" });
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to create role");
    } finally {
      setCreating(false);
    }
  }

  async function handleDelete(id: string, name: string) {
    try {
      const token = await getToken();
      if (!token) return;
      await roleApi.delete(id, token);
      toast.success(`Role "${name}" deleted`);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to delete role");
    }
  }

  function tabLabel(tab: string) {
    if (tab === "all") return "All";
    if (tab === "system") return "System";
    return tab.charAt(0).toUpperCase() + tab.slice(1);
  }

  return (
    <div className="space-y-4">
      {/* Module Tabs */}
      <div className="flex items-center gap-2 flex-wrap">
        {tabs.map((tab) => (
          <button
            key={tab}
            onClick={() => setSelectedModule(tab)}
            className={`px-3 py-1.5 rounded-[1000px] text-xs font-medium border transition-colors ${
              selectedModule === tab
                ? "bg-tiger-red text-white border-tiger-red"
                : "border-border text-semi-grey hover:border-tiger-red hover:text-tiger-red"
            }`}
          >
            {tabLabel(tab)}
          </button>
        ))}
      </div>

      {/* Toolbar — responsive: count on left, button on right; full-width button on mobile */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
        <p className="text-sm text-semi-grey">
          {filteredRoles.length} role{filteredRoles.length !== 1 ? "s" : ""}
          {selectedModule !== "all" ? ` in "${tabLabel(selectedModule)}"` : " in this tenant"}
        </p>
        {/* Show "Create Custom Role" only on module tabs; show generic "Create Role" on All tab; hide on System tab */}
        {selectedModule !== "system" && (
          <Button
            onClick={openCreateDialog}
            className="w-full sm:w-auto rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm h-11 px-4"
          >
            <Plus size={16} className="mr-1.5" />
            {isModuleTab ? "Create Custom Role" : "Create Role"}
          </Button>
        )}
      </div>

      {/* Desktop table */}
      <div className="hidden md:block bg-card rounded-[10px] border border-border overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="animate-spin text-tiger-red" size={24} />
          </div>
        ) : filteredRoles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <ShieldCheck size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No roles found</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-[#fafafa] dark:bg-[#1a2332] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]">
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Name</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Description</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Module</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Type</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Created</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredRoles.map((role) => (
                <TableRow key={role.id} className="bg-white dark:bg-[#1E2533] hover:bg-[#fafafa] dark:hover:bg-[#1a2332]">
                  <TableCell className="font-medium text-semi-black text-sm font-mono">
                    {role.name}
                  </TableCell>
                  <TableCell className="text-sm text-semi-grey max-w-[320px]">
                    {role.description || "—"}
                  </TableCell>
                  <TableCell className="text-sm text-semi-grey">
                    {role.module ? (
                      <Badge variant="outline" className="text-[10px] border-indigo-300 text-indigo-600">
                        {role.module}
                      </Badge>
                    ) : (
                      <span className="text-xs text-semi-grey">—</span>
                    )}
                  </TableCell>
                  <TableCell>
                    {role.is_system ? (
                      <Badge
                        variant="outline"
                        className="text-[10px] border-tiger-red text-tiger-red"
                      >
                        system
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="text-[10px] border-border text-semi-grey"
                      >
                        custom
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-xs text-semi-grey">
                    {new Date(role.created_at).toLocaleDateString("th-TH")}
                  </TableCell>
                  <TableCell>
                    {/* Hide delete button for system roles */}
                    {!role.is_system && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 rounded-full text-semi-grey hover:text-destructive"
                        onClick={() => handleDelete(role.id, role.name)}
                      >
                        <Trash2 size={15} />
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
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
        ) : filteredRoles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <ShieldCheck size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No roles found</p>
          </div>
        ) : (
          <div className="space-y-2">
            {filteredRoles.map((role) => (
              <RoleCard
                key={role.id}
                role={role}
                onDelete={!role.is_system ? handleDelete : undefined}
              />
            ))}
          </div>
        )}
      </div>

      {/* Create Role Dialog — responsive */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="w-[calc(100vw-32px)] sm:max-w-[400px] rounded-[10px]">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-semi-black">
              {isModuleTab ? `Create Custom Role — ${tabLabel(selectedModule)}` : "Create Custom Role"}
            </DialogTitle>
          </DialogHeader>
          <form onSubmit={handleCreate} className="space-y-4 mt-2">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Role Name</Label>
              <Input
                required
                placeholder="recruiter"
                pattern="[a-z0-9_]+"
                value={form.name}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    name: e.target.value.toLowerCase().replace(/[^a-z0-9_]/g, "_"),
                  }))
                }
                className="w-full rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11 font-mono"
              />
              <p className="text-xs text-semi-grey">lowercase letters, numbers, underscore</p>
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Description</Label>
              <Input
                placeholder="HR recruiter with access to job postings"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                className="w-full rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
              />
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Module</Label>
              <Input
                placeholder="recruit (leave blank for system scope)"
                value={form.module}
                onChange={(e) => setForm((f) => ({ ...f, module: e.target.value.toLowerCase() }))}
                className="w-full rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11"
              />
              <p className="text-xs text-semi-grey">Leave blank for tenant-wide custom role</p>
            </div>
            <DialogFooter className="flex-col-reverse sm:flex-row pt-2 gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={() => setShowCreate(false)}
                className="w-full sm:w-auto rounded-[1000px] h-11"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={creating}
                className="w-full sm:w-auto rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white h-11"
              >
                {creating && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Create
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
