"use client";

import { useState, useEffect, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { toast } from "sonner";
import { Eye, EyeOff, Loader2, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { authApi, ApiError } from "@/lib/api";

function AcceptInviteForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token") ?? "";
  const tenantSlug = searchParams.get("tenant") ?? "platform";

  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  useEffect(() => {
    if (!token) {
      toast.error("Invalid or missing invitation token.");
    }
  }, [token]);

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
      await authApi.acceptInvite(token, password, tenantSlug);
      setDone(true);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to accept invitation.");
    } finally {
      setLoading(false);
    }
  };

  if (done) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
        <div className="w-full max-w-[440px] bg-white rounded-[10px] p-10 shadow-sm text-center">
          <div className="inline-flex items-center justify-center w-14 h-14 rounded-full bg-green-50 mb-5">
            <CheckCircle2 size={28} className="text-green-500" />
          </div>
          <h1 className="text-xl font-semibold text-semi-black mb-2">
            Account activated!
          </h1>
          <p className="text-sm text-semi-grey mb-8">
            Your account has been set up successfully. You can now sign in.
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

  return (
    <div className="min-h-screen bg-page-bg flex items-center justify-center px-4">
      <div className="w-full max-w-[440px] bg-white rounded-[10px] p-10 shadow-sm">
        {/* Brand */}
        <div className="mb-8 text-center">
          <div className="inline-flex items-center justify-center w-12 h-12 rounded-full bg-tiger-red mb-4">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 2L3 7V17L12 22L21 17V7L12 2Z" stroke="white" strokeWidth="2" strokeLinejoin="round" />
              <path d="M12 2V22M3 7L21 17M21 7L3 17" stroke="white" strokeWidth="1.5" strokeLinecap="round" />
            </svg>
          </div>
          <h1 className="text-xl font-semibold text-semi-black">
            Set up your account
          </h1>
          <p className="text-sm text-semi-grey mt-1">
            Choose a password to activate your invitation
          </p>
        </div>

        {!token ? (
          <p className="text-center text-sm text-destructive">
            This invitation link is invalid or has expired.
          </p>
        ) : (
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
                  className="bg-[#f0f0f0] border-[#f0f0f0] rounded-[10px] h-12 px-4 pr-12 text-sm focus-visible:ring-tiger-red"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-4 top-1/2 -translate-y-1/2 text-semi-grey hover:text-semi-black"
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
                className="bg-[#f0f0f0] border-[#f0f0f0] rounded-[10px] h-12 px-4 text-sm focus-visible:ring-tiger-red"
              />
            </div>

            <Button
              type="submit"
              disabled={loading || !token}
              className="w-full h-12 rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white font-semibold text-sm mt-2"
            >
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Activate Account
            </Button>
          </form>
        )}

        <p className="text-center text-xs text-semi-grey mt-6">
          TGX Auth Console v2.0 · Tigersoft Co., Ltd.
        </p>
      </div>
    </div>
  );
}

export default function AcceptInvitePage() {
  return (
    <Suspense>
      <AcceptInviteForm />
    </Suspense>
  );
}
