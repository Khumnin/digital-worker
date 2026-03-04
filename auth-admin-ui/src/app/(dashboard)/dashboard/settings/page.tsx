"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, Save } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/contexts/auth";
import { tenantApi, type TenantConfig, ApiError } from "@/lib/api";

export default function SettingsPage() {
  const { getToken, tenantId } = useAuth();
  const [config, setConfig] = useState<TenantConfig>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [tenantDbId, setTenantDbId] = useState<string | null>(null);
  const [sessionMinutes, setSessionMinutes] = useState("60");
  const [mfaRequired, setMfaRequired] = useState(false);

  useEffect(() => {
    load();
  }, [tenantId]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token || !tenantId) return;
      const tenants = await tenantApi.list(token);
      const current = tenants.data.find((t) => t.id === tenantId || t.slug === tenantId);
      if (current) {
        setTenantDbId(current.id);
        setConfig(current.config ?? {});
        setMfaRequired(current.config?.mfa_required ?? false);
        setSessionMinutes(String(current.config?.session_duration_minutes ?? 60));
      }
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load settings");
    } finally {
      setLoading(false);
    }
  }

  async function handleSave() {
    if (!tenantDbId) return;
    setSaving(true);
    try {
      const token = await getToken();
      if (!token) return;
      const updatedConfig: TenantConfig = {
        ...config,
        mfa_required: mfaRequired,
        session_duration_minutes: parseInt(sessionMinutes, 10) || 60,
      };
      await tenantApi.update(tenantDbId, { config: updatedConfig }, token);
      toast.success("Settings saved");
      setConfig(updatedConfig);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to save settings");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-tiger-red" size={28} />
      </div>
    );
  }

  return (
    <div className="space-y-5 max-w-2xl">
      {/* Security Settings */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black">
            Security
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-semi-black">
                Require MFA
              </p>
              <p className="text-xs text-semi-grey mt-0.5">
                Force all users to enable TOTP before accessing the system
              </p>
            </div>
            <button
              onClick={() => setMfaRequired((v) => !v)}
              className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                mfaRequired ? "bg-tiger-red" : "bg-[#e5e5e5]"
              }`}
            >
              <span
                className={`inline-block h-4 w-4 transform rounded-full bg-white shadow transition-transform ${
                  mfaRequired ? "translate-x-6" : "translate-x-1"
                }`}
              />
            </button>
          </div>

          <Separator />

          <div className="space-y-1.5">
            <Label className="text-sm font-medium text-semi-black">
              Session Duration (minutes)
            </Label>
            <Input
              type="number"
              min="15"
              max="1440"
              value={sessionMinutes}
              onChange={(e) => setSessionMinutes(e.target.value)}
              className="rounded-[10px] bg-[#f0f0f0] border-[#f0f0f0] h-11 max-w-[160px]"
            />
            <p className="text-xs text-semi-grey">
              How long refresh tokens remain valid. Default: 60 minutes.
            </p>
          </div>
        </CardContent>
      </Card>

      {/* JWKS / Integration Info */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black">
            Integration Endpoints
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-xs font-mono">
          {[
            { label: "JWKS", value: `${process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev"}/.well-known/jwks.json` },
            { label: "Login", value: `${process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev"}/api/v1/auth/login` },
            { label: "Token", value: `${process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev"}/api/v1/oauth/token` },
          ].map((item) => (
            <div key={item.label} className="space-y-1">
              <p className="text-semi-grey text-[10px] uppercase font-semibold not-italic">
                {item.label}
              </p>
              <div className="bg-[#f0f0f0] rounded-[10px] px-3 py-2 text-semi-black break-all">
                {item.value}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Save button */}
      <div className="flex justify-end">
        <Button
          onClick={handleSave}
          disabled={saving}
          className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white h-10 px-6"
        >
          {saving ? (
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
          ) : (
            <Save size={15} className="mr-2" />
          )}
          Save Settings
        </Button>
      </div>
    </div>
  );
}
