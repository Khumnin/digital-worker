"use client";

import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { Eye, EyeOff, Loader2, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api";

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev";

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const tenantSlug = searchParams.get("tenant") ?? "platform";

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (password.length < 8) {
      toast.error("Password must be at least 8 characters.");
      return;
    }
    if (password !== confirm) {
      toast.error("Passwords do not match.");
      return;
    }

    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/reset-password`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-ID": tenantSlug,
        },
        body: JSON.stringify({ token, password }),
        cache: "no-store",
      });

      if (!res.ok) {
        let message = `HTTP ${res.status}`;
        let code: string | undefined;
        try {
          const err = await res.json();
          message = err.error?.message || err.message || err.error || message;
          code = err.error?.code || err.code;
        } catch {}
        throw new ApiError(res.status, message, code);
      }

      setDone(true);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to reset password. Please try again."
      );
    } finally {
      setLoading(false);
    }
  };

  // No token in URL — invalid/expired link
  if (!token) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
        <div className="w-full max-w-[440px] bg-card rounded-[10px] p-10 shadow-sm text-center">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-red-50 dark:bg-red-900/30 mb-5">
            <svg
              width="28"
              height="28"
              viewBox="0 0 24 24"
              fill="none"
              className="text-red-500"
            >
              <circle cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="2" />
              <line x1="12" y1="8" x2="12" y2="12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" />
              <circle cx="12" cy="16" r="1" fill="currentColor" />
            </svg>
          </div>
          <h1 className="text-xl font-semibold text-semi-black mb-2">
            Link invalid or expired
          </h1>
          <p className="text-sm text-semi-grey mb-8">
            This reset link is invalid or has expired. Please request a new
            password reset link.
          </p>
          <Button
            onClick={() => router.push("/forgot-password")}
            className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white h-11 px-8 text-sm font-semibold"
          >
            Request New Link
          </Button>
        </div>
      </div>
    );
  }

  // Done state — password updated successfully
  if (done) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
        <div className="w-full max-w-[440px] bg-card rounded-[10px] p-10 shadow-sm text-center">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-green-50 dark:bg-green-900/30 mb-5">
            <CheckCircle2 size={28} className="text-green-500" />
          </div>
          <h1 className="text-xl font-semibold text-semi-black mb-2">
            Password updated!
          </h1>
          <p className="text-sm text-semi-grey mb-8">
            Your password has been changed. You can now sign in with your new
            password.
          </p>
          <Button
            onClick={() => router.push("/login")}
            className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white h-11 px-8 text-sm font-semibold"
          >
            Go to Sign In
          </Button>
        </div>
      </div>
    );
  }

  // Form state
  return (
    <div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
      <div className="w-full max-w-[440px] bg-card rounded-[10px] p-10 shadow-sm">
        {/* Brand */}
        <div className="mb-8 text-center">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-tiger-red mb-4">
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <line x1="3" y1="14" x2="10" y2="3" stroke="white" strokeWidth="2.5" strokeLinecap="round"/>
              <line x1="7" y1="15" x2="14" y2="4" stroke="white" strokeWidth="2.5" strokeLinecap="round"/>
              <line x1="11" y1="15" x2="18" y2="4" stroke="white" strokeWidth="2" strokeLinecap="round" opacity="0.7"/>
            </svg>
          </div>
          <h1 className="text-xl font-semibold text-semi-black">
            Set new password
          </h1>
          <p className="text-sm text-semi-grey mt-1">
            Choose a strong password for your account
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label className="text-sm font-medium text-semi-black">
              New Password
            </Label>
            <div className="relative">
              <Input
                type={showPassword ? "text" : "password"}
                autoComplete="new-password"
                placeholder="Min. 8 characters"
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-12 px-4 pr-12 text-sm focus-visible:ring-tiger-red"
              />
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                className="absolute right-4 top-1/2 -translate-y-1/2 text-semi-grey hover:text-semi-black transition-colors"
                tabIndex={-1}
              >
                {showPassword ? <EyeOff size={18} /> : <Eye size={18} />}
              </button>
            </div>
          </div>

          <div className="space-y-1.5">
            <Label className="text-sm font-medium text-semi-black">
              Confirm Password
            </Label>
            <Input
              type="password"
              autoComplete="new-password"
              placeholder="Re-enter password"
              required
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
              className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-12 px-4 text-sm focus-visible:ring-tiger-red"
            />
          </div>

          <Button
            type="submit"
            disabled={loading}
            className="w-full h-12 rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white font-semibold text-sm mt-2"
          >
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Reset Password
          </Button>
        </form>

        <p className="text-center text-xs text-semi-grey mt-6">
          TGX Auth Console v2.0 · Tigersoft Co., Ltd.
        </p>
      </div>
    </div>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordForm />
    </Suspense>
  );
}
