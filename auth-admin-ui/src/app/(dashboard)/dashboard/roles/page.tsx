"use client";

import { useEffect, useState } from "react";
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

export default function RolesPage() {
  const { getToken, tenantId } = useAuth();
  const [roles, setRoles] = useState<Role[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState<CreateRoleRequest>({ name: "", description: "" });

  useEffect(() => {
    load();
  }, [tenantId]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token || !tenantId) return;
      const result = await roleApi.list(token, tenantId);
      setRoles(result);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load roles");
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    try {
      const token = await getToken();
      if (!token || !tenantId) return;
      await roleApi.create(form, token, tenantId);
      toast.success(`Role "${form.name}" created`);
      setShowCreate(false);
      setForm({ name: "", description: "" });
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
      if (!token || !tenantId) return;
      await roleApi.delete(id, token, tenantId);
      toast.success(`Role "${name}" deleted`);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to delete role");
    }
  }

  return (
    <div className="space-y-4">
      {/* Toolbar */}
      <div className="flex items-center justify-between">
        <p className="text-sm text-semi-grey">
          {roles.length} role{roles.length !== 1 ? "s" : ""} in this tenant
        </p>
        <Button
          onClick={() => setShowCreate(true)}
          className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-sm h-10 px-4"
        >
          <Plus size={16} className="mr-1.5" />
          Create Role
        </Button>
      </div>

      {/* Table */}
      <div className="bg-white rounded-[10px] border border-border overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-16">
            <Loader2 className="animate-spin text-tiger-red" size={24} />
          </div>
        ) : roles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-semi-grey">
            <ShieldCheck size={36} className="mb-3 opacity-40" />
            <p className="text-sm">No roles found</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-[#fafafa] hover:bg-[#fafafa]">
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Name</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Description</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Type</TableHead>
                <TableHead className="text-xs font-semibold text-semi-grey uppercase">Created</TableHead>
                <TableHead className="w-10" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {roles.map((role) => (
                <TableRow key={role.id} className="hover:bg-[#fafafa]">
                  <TableCell className="font-medium text-semi-black text-sm font-mono">
                    {role.name}
                  </TableCell>
                  <TableCell className="text-sm text-semi-grey max-w-[320px]">
                    {role.description || "—"}
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

      {/* Create Role Dialog */}
      <Dialog open={showCreate} onOpenChange={setShowCreate}>
        <DialogContent className="sm:max-w-[400px] rounded-[10px]">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold text-semi-black">
              Create Custom Role
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
                className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11 font-mono"
              />
              <p className="text-xs text-semi-grey">lowercase letters, numbers, underscore</p>
            </div>
            <div className="space-y-1.5">
              <Label className="text-sm font-medium text-semi-black">Description</Label>
              <Input
                placeholder="HR recruiter with access to job postings"
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11"
              />
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
                Create
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
