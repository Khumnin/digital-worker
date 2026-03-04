"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { useAuth } from "@/contexts/auth";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

const LANG_KEY = "tgx_lang";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isAuthenticated, isAdmin, accessToken, logout } = useAuth();
  const router = useRouter();
  const [lang, setLang] = useState<"th" | "en">("th");
  const [ready, setReady] = useState(false);

  // Restore lang preference
  useEffect(() => {
    const stored = localStorage.getItem(LANG_KEY);
    if (stored === "en" || stored === "th") setLang(stored);
    setReady(true);
  }, []);

  // Redirect to login if not authenticated after hydration
  useEffect(() => {
    if (ready && !isAuthenticated && accessToken === null) {
      // Give the auth context a moment to restore session from localStorage
      const timer = setTimeout(() => {
        if (!isAuthenticated) router.replace("/login");
      }, 800);
      return () => clearTimeout(timer);
    }
  }, [ready, isAuthenticated, accessToken, router]);

  const handleLangChange = (newLang: "th" | "en") => {
    setLang(newLang);
    localStorage.setItem(LANG_KEY, newLang);
  };

  if (!ready) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center">
        <Loader2 className="text-tiger-red animate-spin" size={28} />
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center">
        <Loader2 className="text-tiger-red animate-spin" size={28} />
      </div>
    );
  }

  if (isAuthenticated && !isAdmin) {
    return (
      <div className="min-h-screen bg-page-bg flex items-center justify-center">
        <div className="bg-white rounded-[10px] border border-border shadow-sm p-8 max-w-sm w-full text-center space-y-4">
          <div className="w-12 h-12 rounded-full bg-[#FFF0F2] flex items-center justify-center mx-auto">
            <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
              <path d="M12 9v4M12 17h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z" stroke="#F4001A" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </div>
          <div className="space-y-1.5">
            <h2 className="text-sm font-semibold text-semi-black">Limited Access</h2>
            <p className="text-xs text-semi-grey">
              You don&apos;t have admin access to this console.
            </p>
            <p className="text-xs text-semi-grey">
              Contact your administrator to request access.
            </p>
          </div>
          <Button
            onClick={async () => { await logout(); router.replace("/login"); }}
            variant="outline"
            className="rounded-[1000px] text-sm w-full"
          >
            Sign Out
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen overflow-hidden bg-page-bg">
      <Sidebar lang={lang} />
      <div className="flex flex-col flex-1 min-w-0 overflow-hidden">
        <Header lang={lang} onLangChange={handleLangChange} />
        <main className="flex-1 overflow-y-auto p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
