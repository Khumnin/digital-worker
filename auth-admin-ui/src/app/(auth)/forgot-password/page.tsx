"use client";

import { useState, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { Loader2, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError } from "@/lib/api";
import Link from "next/link";

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL || "https://auth.tgstack.dev";

function ForgotPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const tenantSlug = searchParams.get("tenant") ?? "platform";

  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    try {
      const res = await fetch(`${API_BASE}/api/v1/auth/forgot-password`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-Tenant-ID": tenantSlug,
        },
        body: JSON.stringify({ email }),
        cache: "no-store",
      });

      if (!res.ok && res.status !== 404) {
        let message = `HTTP ${res.status}`;
        let code: string | undefined;
        try {
          const err = await res.json();
          message = err.error?.message || err.message || err.error || message;
          code = err.error?.code || err.code;
        } catch {}
        throw new ApiError(res.status, message, code);
      }
    } catch (err) {
      // Anti-enumeration: even on unexpected errors we show success to the user,
      // but log to console for debugging. We only throw on truly unexpected errors
      // (not 404, which means "no account found" — expected).
      if (err instanceof ApiError && err.status >= 500) {
        toast.error("A server error occurred. Please try again later.");
        setLoading(false);
        return;
      }
    } finally {
      setLoading(false);
    }
    // Always show success — anti-enumeration
    setDone(true);
  };

  if (done) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center px-4 py-8">
        <div className="w-[calc(100vw-32px)] sm:w-full sm:max-w-[440px] bg-card rounded-[10px] p-6 sm:p-10 shadow-sm text-center">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-green-50 dark:bg-green-900/30 mb-5">
            <CheckCircle2 size={28} className="text-green-500" />
          </div>
          <h1 className="text-xl font-semibold text-semi-black mb-2">
            Check your email
          </h1>
          <p className="text-sm text-semi-grey mb-8">
            If an account with that email exists, a password reset link has been
            sent. The link expires in 1 hour.
          </p>
          <Button
            onClick={() => router.push("/login")}
            className="w-full sm:w-auto rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white h-12 px-8 text-sm font-semibold"
          >
            Back to Sign In
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-page-bg flex items-center justify-center px-4 py-8">
      <div className="w-[calc(100vw-32px)] sm:w-full sm:max-w-[440px] bg-card rounded-[10px] p-6 sm:p-10 shadow-sm">
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
            Forgot password?
          </h1>
          <p className="text-sm text-semi-grey mt-1">
            Enter your email and we&apos;ll send you a reset link
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="email" className="text-sm font-medium text-semi-black">
              Email
            </Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              placeholder="you@example.com"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="w-full bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-12 px-4 text-sm focus-visible:ring-tiger-red"
            />
          </div>

          <Button
            type="submit"
            disabled={loading}
            className="w-full h-12 rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white font-semibold text-sm mt-2"
          >
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            Send Reset Link
          </Button>
        </form>

        {/* "Sign in" link — touch target >= 44px */}
        <p className="text-center text-sm text-semi-grey mt-6">
          Remember your password?{" "}
          <Link
            href="/login"
            className="inline-flex items-center min-h-[44px] text-tiger-red hover:underline text-sm"
          >
            Sign in
          </Link>
        </p>

        <p className="text-center text-xs text-semi-grey mt-4">
          TGX Auth Console v2.0 · Tigersoft Co., Ltd.
        </p>
      </div>
    </div>
  );
}

export default function ForgotPasswordPage() {
  return (
    <Suspense>
      <ForgotPasswordForm />
    </Suspense>
  );
}
