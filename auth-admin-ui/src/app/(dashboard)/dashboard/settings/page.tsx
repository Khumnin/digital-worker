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
import { settingsApi, ApiError } from "@/lib/api";

export default function SettingsPage() {
  const { getToken } = useAuth();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [sessionHours, setSessionHours] = useState("1");
  const [mfaRequired, setMfaRequired] = useState(false);
  const [allowedDomains, setAllowedDomains] = useState("");

  useEffect(() => {
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const settings = await settingsApi.get(token);
      setMfaRequired(settings.mfa_required);
      setSessionHours(String(settings.session_duration_hours));
      setAllowedDomains((settings.allowed_domains ?? []).join(", "));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load settings");
    } finally {
      setLoading(false);
    }
  }

  async function handleSave() {
    setSaving(true);
    try {
      const token = await getToken();
      if (!token) return;
      const domains = allowedDomains
        .split(",")
        .map((d) => d.trim())
        .filter(Boolean);
      await settingsApi.update(token, {
        mfa_required: mfaRequired,
        session_duration_hours: parseInt(sessionHours, 10) || 1,
        allowed_domains: domains,
      });
      toast.success("Settings saved");
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
                mfaRequired ? "bg-tiger-red" : "bg-[#e5e5e5] dark:bg-[#3A3A45]"
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
              Session Duration (hours)
            </Label>
            <Input
              type="number"
              min="1"
              max="168"
              value={sessionHours}
              onChange={(e) => setSessionHours(e.target.value)}
              className="rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input h-11 max-w-[160px]"
            />
            <p className="text-xs text-semi-grey">
              How long refresh tokens remain valid. Default: 1 hour.
            </p>
          </div>

          <Separator />

          <div className="space-y-1.5">
            <Label className="text-sm font-medium text-semi-black">
              Allowed Domains
            </Label>
            <textarea
              value={allowedDomains}
              onChange={(e) => setAllowedDomains(e.target.value)}
              placeholder="company.co.th, partner.com"
              rows={3}
              className="w-full rounded-[10px] bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input border px-3 py-2 text-sm text-semi-black placeholder:text-semi-grey resize-none focus:outline-none focus:ring-2 focus:ring-tiger-red/30"
            />
            <p className="text-xs text-semi-grey">
              Comma-separated list of email domains allowed to sign up. Leave empty to allow all domains.
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
              <div className="bg-[#f0f0f0] dark:bg-input rounded-[10px] px-3 py-2 text-semi-black break-all">
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
