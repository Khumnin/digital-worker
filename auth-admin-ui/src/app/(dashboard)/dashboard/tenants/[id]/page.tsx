"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { toast } from "sonner";
import {
  ArrowLeft,
  Copy,
  Check,
  Loader2,
  RefreshCw,
  Globe,
  Code2,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { useAuth } from "@/contexts/auth";
import { tenantApi, type Tenant, ApiError } from "@/lib/api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev";

function CopyField({ label, value }: { label: string; value: string }) {
  const [copied, setCopied] = useState(false);
  const copy = async () => {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };
  return (
    <div className="space-y-1">
      <p className="text-xs text-semi-grey font-medium uppercase">{label}</p>
      <div className="flex items-center gap-2 bg-[#f0f0f0] rounded-[10px] px-3 py-2">
        <code className="text-xs text-semi-black flex-1 break-all font-mono">{value}</code>
        <button
          onClick={copy}
          className="shrink-0 text-semi-grey hover:text-semi-black transition-colors"
        >
          {copied ? <Check size={14} className="text-green-600" /> : <Copy size={14} />}
        </button>
      </div>
    </div>
  );
}

export default function TenantDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { getToken } = useAuth();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(true);

  const id = params?.id as string;

  useEffect(() => {
    load();
  }, [id]); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const t = await tenantApi.get(id, token);
      setTenant(t);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load tenant");
    } finally {
      setLoading(false);
    }
  }

  async function handleToggleStatus() {
    if (!tenant) return;
    try {
      const token = await getToken();
      if (!token) return;
      if (tenant.status === "active") {
        await tenantApi.suspend(tenant.id, token);
        toast.success("Tenant suspended");
      } else {
        await tenantApi.activate(tenant.id, token);
        toast.success("Tenant activated");
      }
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed");
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-tiger-red" size={28} />
      </div>
    );
  }

  if (!tenant) {
    return (
      <div className="text-center py-24 text-semi-grey text-sm">
        Tenant not found.
      </div>
    );
  }

  const integrationEnv = `AUTH_API_URL=${API_BASE}
AUTH_JWKS_URL=${API_BASE}/.well-known/jwks.json
AUTH_ISSUER=${API_BASE}
AUTH_AUDIENCE=tigersoft-auth
AUTH_TENANT_SLUG=${tenant.slug}
AUTH_PLATFORM_TENANT_SLUG=platform`;

  return (
    <div className="space-y-5 max-w-4xl">
      {/* Back + Header */}
      <div className="flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          onClick={() => router.back()}
          className="h-8 w-8 rounded-full"
        >
          <ArrowLeft size={16} />
        </Button>
        <div className="flex-1">
          <h1 className="text-base font-semibold text-semi-black">{tenant.name}</h1>
          <p className="text-xs text-semi-grey font-mono">{tenant.slug}</p>
        </div>
        <div className="flex items-center gap-2">
          <span
            className={`inline-flex items-center px-2.5 py-1 rounded-full text-xs font-medium border ${
              tenant.status === "active"
                ? "bg-[#EDFBF5] text-[#34D186] border-[#34D186]/40"
                : tenant.status === "pending"
                ? "bg-yellow-100 text-yellow-700 border-yellow-200"
                : "bg-red-100 text-tiger-red border-red-200"
            }`}
          >
            {tenant.status}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={handleToggleStatus}
            className="rounded-[1000px] text-xs h-8"
          >
            <RefreshCw size={12} className="mr-1.5" />
            {tenant.status === "active" ? "Suspend" : "Activate"}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Tenant info */}
        <Card className="rounded-[10px] border-border shadow-none">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
              <Globe size={15} className="text-tiger-red" />
              Tenant Info
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <CopyField label="Tenant ID" value={tenant.id} />
            <CopyField label="Slug" value={tenant.slug} />
            <div className="space-y-1">
              <p className="text-xs text-semi-grey font-medium uppercase">Enabled Modules</p>
              <div className="flex flex-wrap gap-1 pt-1">
                {(tenant.enabled_modules ?? []).length === 0 ? (
                  <span className="text-xs text-semi-grey">None</span>
                ) : (
                  (tenant.enabled_modules ?? []).map((mod) => (
                    <Badge
                      key={mod}
                      variant="outline"
                      className="text-[10px] border-tiger-red text-tiger-red uppercase"
                    >
                      {mod}
                    </Badge>
                  ))
                )}
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Recruitment Integration Panel */}
        {(tenant.enabled_modules ?? []).includes("recruitment") && (
          <Card className="rounded-[10px] border-tiger-red/20 bg-[#FFF8F8] shadow-none">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
                <Code2 size={15} className="text-tiger-red" />
                Recruitment Integration
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <p className="text-xs text-semi-grey">
                Copy these env vars into your Recruitment backend deployment:
              </p>
              <div className="bg-white rounded-[10px] border border-border p-3">
                <code className="text-xs font-mono text-semi-black whitespace-pre-wrap break-all leading-relaxed">
                  {integrationEnv}
                </code>
              </div>
              <button
                onClick={async () => {
                  await navigator.clipboard.writeText(integrationEnv);
                  toast.success("Copied to clipboard");
                }}
                className="text-xs text-tiger-red hover:underline font-medium"
              >
                Copy all env vars
              </button>
            </CardContent>
          </Card>
        )}
      </div>

      <Separator />

      {/* Enabled modules detail */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black">
            Module Configuration
          </CardTitle>
        </CardHeader>
        <CardContent>
          <pre className="text-xs font-mono text-semi-black bg-[#f0f0f0] rounded-[10px] p-4 overflow-auto">
            {JSON.stringify({ enabled_modules: tenant.enabled_modules }, null, 2)}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
