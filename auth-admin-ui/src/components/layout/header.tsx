"use client";

import { useAuth } from "@/contexts/auth";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Badge } from "@/components/ui/badge";
import { ThemeToggle } from "@/components/ui/theme-toggle";
import { ChevronDown, Globe, Menu } from "lucide-react";
import { useRouter } from "next/navigation";

interface HeaderProps {
  lang: "th" | "en";
  onLangChange: (lang: "th" | "en") => void;
  title?: string;
  /** Called when the hamburger button is tapped on mobile to open the nav drawer. */
  onMenuOpen?: () => void;
}

export function Header({ lang, onLangChange, title, onMenuOpen }: HeaderProps) {
  const { user, isSuperAdmin, logout } = useAuth();
  const router = useRouter();

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  const initials =
    user?.email
      ? user.email.charAt(0).toUpperCase()
      : "?";

  return (
    <header className="h-[64px] bg-card border-b border-border flex items-center justify-between px-4 md:px-6 shrink-0">
      <div className="flex items-center gap-2">
        {/* Hamburger button — mobile only */}
        {onMenuOpen && (
          <button
            onClick={onMenuOpen}
            className="md:hidden flex items-center justify-center w-11 h-11 rounded-[10px] text-semi-black hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] transition-colors"
            aria-label="Open navigation menu"
          >
            <Menu size={20} />
          </button>
        )}
        {/* Page title */}
        <h2 className="text-[15px] font-semibold text-semi-black">
          {title ?? "Dashboard"}
        </h2>
      </div>

      <div className="flex items-center gap-3">
        {/* Theme toggle */}
        <ThemeToggle />

        {/* Language toggle */}
        <button
          onClick={() => onLangChange(lang === "th" ? "en" : "th")}
          className="flex items-center gap-1.5 text-sm text-semi-grey hover:text-semi-black transition-colors px-2 py-1.5 rounded-[8px] hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35]"
        >
          <Globe size={16} />
          <span className="font-medium">{lang.toUpperCase()}</span>
        </button>

        {/* User menu */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button className="flex items-center gap-2 rounded-[1000px] px-3 py-1.5 hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] transition-colors">
              {/* Avatar */}
              <div className="w-8 h-8 rounded-full bg-tiger-red text-white flex items-center justify-center text-sm font-semibold">
                {initials}
              </div>
              {/* Info */}
              <div className="text-left hidden sm:block">
                <p className="text-xs font-medium text-semi-black leading-tight max-w-[160px] truncate">
                  {user?.email ?? "—"}
                </p>
                <p className="text-[11px] text-semi-grey leading-tight">
                  {isSuperAdmin
                    ? lang === "th"
                      ? "ผู้ดูแลระบบสูงสุด"
                      : "Super Admin"
                    : lang === "th"
                    ? "ผู้ดูแลระบบ"
                    : "Admin"}
                </p>
              </div>
              <ChevronDown size={14} className="text-semi-grey" />
            </button>
          </DropdownMenuTrigger>

          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel className="text-xs text-semi-grey font-normal">
              {user?.email}
            </DropdownMenuLabel>
            {isSuperAdmin && (
              <div className="px-2 pb-1">
                <Badge
                  variant="outline"
                  className="text-[10px] border-tiger-red text-tiger-red"
                >
                  Super Admin
                </Badge>
              </div>
            )}
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={handleLogout}
              className="text-sm cursor-pointer text-destructive focus:text-destructive"
            >
              {lang === "th" ? "ออกจากระบบ" : "Sign out"}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  );
}
