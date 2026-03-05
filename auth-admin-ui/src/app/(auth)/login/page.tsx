"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { Eye, EyeOff, Loader2 } from "lucide-react";
import Link from "next/link";
import Image from "next/image";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authApi, ApiError } from "@/lib/api";
import { useAuth } from "@/contexts/auth";

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuth();

  const [form, setForm] = useState({
    email: "",
    password: "",
    tenantSlug: "platform",
  });
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const result = await authApi.login(
        { email: form.email, password: form.password },
        form.tenantSlug
      );
      login(result.access_token, result.refresh_token, form.tenantSlug);
      router.push("/dashboard");
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? err.message
          : "เกิดข้อผิดพลาด กรุณาลองใหม่";
      toast.error(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-page-bg flex items-center justify-center px-4 py-8">
      {/* Card — constrained width with safe horizontal margin on 375px screens */}
      <div className="w-[calc(100vw-32px)] sm:w-full sm:max-w-[440px] bg-card rounded-[10px] p-6 sm:p-10 shadow-sm">
        {/* Logo / Brand */}
        <div className="mb-8 text-center">
          <Image
            src="/logo-mark.svg"
            alt="TigerSoft"
            width={56}
            height={56}
            className="mx-auto mb-4"
            priority
          />
          <h1 className="text-xl font-semibold text-semi-black">
            TGX Auth Console
          </h1>
          <p className="text-sm text-semi-grey mt-1">
            Tigersoft Authentication Platform
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Tenant Slug */}
          <div className="space-y-1.5">
            <Label htmlFor="tenantSlug" className="text-sm font-medium text-semi-black">
              Tenant
            </Label>
            <Input
              id="tenantSlug"
              type="text"
              autoComplete="off"
              placeholder="platform"
              value={form.tenantSlug}
              onChange={(e) =>
                setForm((f) => ({ ...f, tenantSlug: e.target.value }))
              }
              className="w-full bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-12 px-4 text-sm focus-visible:ring-tiger-red"
            />
          </div>

          {/* Email */}
          <div className="space-y-1.5">
            <Label htmlFor="email" className="text-sm font-medium text-semi-black">
              อีเมล / Email
            </Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="admin@tigersoft.co.th"
              required
              value={form.email}
              onChange={(e) =>
                setForm((f) => ({ ...f, email: e.target.value }))
              }
              className="w-full bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-12 px-4 text-sm focus-visible:ring-tiger-red"
            />
          </div>

          {/* Password */}
          <div className="space-y-1.5">
            <Label htmlFor="password" className="text-sm font-medium text-semi-black">
              รหัสผ่าน / Password
            </Label>
            <div className="relative">
              <Input
                id="password"
                type={showPassword ? "text" : "password"}
                autoComplete="current-password"
                placeholder="••••••••"
                required
                value={form.password}
                onChange={(e) =>
                  setForm((f) => ({ ...f, password: e.target.value }))
                }
                className="w-full bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-12 px-4 pr-14 text-sm focus-visible:ring-tiger-red"
              />
              {/* Touch target >= 44px */}
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-label={showPassword ? "Hide password" : "Show password"}
                className="absolute right-1 top-1/2 -translate-y-1/2 flex items-center justify-center w-11 h-11 text-semi-grey hover:text-semi-black transition-colors rounded-[10px]"
                tabIndex={-1}
              >
                {showPassword ? (
                  <EyeOff size={18} />
                ) : (
                  <Eye size={18} />
                )}
              </button>
            </div>
          </div>

          {/* Forgot password — touch target >= 44px via inline-flex + min-h */}
          <div className="text-right">
            <Link
              href="/forgot-password"
              className="inline-flex items-center min-h-[44px] text-xs text-tiger-red hover:underline"
            >
              Forgot password?
            </Link>
          </div>

          {/* Submit — full width, 48px height */}
          <Button
            type="submit"
            disabled={loading}
            className="w-full h-12 rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white font-semibold text-sm mt-2"
          >
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            เข้าสู่ระบบ / Sign In
          </Button>
        </form>

        <p className="text-center text-xs text-semi-grey mt-6">
          TGX Auth Console v2.0 · Tigersoft Co., Ltd.
        </p>
      </div>
    </div>
  );
}
