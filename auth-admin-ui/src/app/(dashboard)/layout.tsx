"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";
import { useAuth } from "@/contexts/auth";
import { Loader2 } from "lucide-react";

const LANG_KEY = "tgx_lang";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { isAuthenticated, accessToken } = useAuth();
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
