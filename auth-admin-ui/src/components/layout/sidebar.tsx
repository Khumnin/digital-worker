"use client";

import Link from "next/link";
import Image from "next/image";
import { usePathname } from "next/navigation";
import { useState, useEffect } from "react";
import {
  LayoutDashboard,
  Building2,
  Users,
  ShieldCheck,
  ScrollText,
  Settings,
  ChevronLeft,
  ChevronRight,
  LogOut,
  UserCircle,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAuth } from "@/contexts/auth";
import { useRouter } from "next/navigation";

const SIDEBAR_KEY = "tgx_sidebar_expanded";

interface NavItem {
  label: string;
  labelTh: string;
  href: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  superAdminOnly?: boolean;
  adminOnly?: boolean;
}

const NAV_ITEMS: NavItem[] = [
  {
    label: "Dashboard",
    labelTh: "แดชบอร์ด",
    href: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    label: "Tenants",
    labelTh: "ผู้เช่าระบบ",
    href: "/dashboard/tenants",
    icon: Building2,
    superAdminOnly: true,
  },
  {
    label: "Users",
    labelTh: "ผู้ใช้งาน",
    href: "/dashboard/users",
    icon: Users,
    adminOnly: true,
  },
  {
    label: "Roles",
    labelTh: "บทบาท",
    href: "/dashboard/roles",
    icon: ShieldCheck,
    adminOnly: true,
  },
  {
    label: "Audit Log",
    labelTh: "ประวัติการใช้งาน",
    href: "/dashboard/audit",
    icon: ScrollText,
    adminOnly: true,
  },
  {
    label: "Settings",
    labelTh: "ตั้งค่า",
    href: "/dashboard/settings",
    icon: Settings,
    adminOnly: true,
  },
  {
    label: "My Profile",
    labelTh: "โปรไฟล์",
    href: "/me",
    icon: UserCircle,
  },
];

interface SidebarProps {
  lang: "th" | "en";
}

export function Sidebar({ lang }: SidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const { isSuperAdmin, isAdmin, logout } = useAuth();
  const [expanded, setExpanded] = useState(true);

  useEffect(() => {
    const stored = localStorage.getItem(SIDEBAR_KEY);
    if (stored !== null) setExpanded(stored === "true");
  }, []);

  const toggleExpanded = () => {
    setExpanded((v) => {
      localStorage.setItem(SIDEBAR_KEY, String(!v));
      return !v;
    });
  };

  const handleLogout = async () => {
    await logout();
    router.push("/login");
  };

  const visibleItems = NAV_ITEMS.filter((item) => {
    if (item.superAdminOnly && !isSuperAdmin) return false;
    if (item.adminOnly && !isAdmin) return false;
    return true;
  });

  return (
    <aside
      className={cn(
        "flex flex-col h-screen bg-white border-r border-border transition-[width] duration-200 ease-in-out shrink-0",
        expanded ? "w-[298px]" : "w-[60px]"
      )}
    >
      {/* Brand */}
      <div
        className={cn(
          "flex items-center h-[64px] px-4 border-b border-border shrink-0",
          !expanded && "justify-center"
        )}
      >
        {/* TigerSoft Logo Mark */}
        <Image
          src="/logo-mark.svg"
          alt="TigerSoft"
          width={32}
          height={32}
          className="shrink-0"
        />
        {expanded && (
          <span className="ml-3 text-sm font-semibold text-semi-black truncate">
            TGX Auth Console
          </span>
        )}
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-3 space-y-1 px-2">
        {visibleItems.map((item) => {
          const isActive =
            item.href === "/dashboard"
              ? pathname === "/dashboard"
              : pathname.startsWith(item.href);
          const label = lang === "th" ? item.labelTh : item.label;

          const linkContent = (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-[10px] px-3 py-2.5 text-sm font-medium transition-colors",
                isActive
                  ? "bg-[#FFF0F2] text-tiger-red"
                  : "text-semi-black hover:bg-[#f5f5f5]",
                !expanded && "justify-center px-0"
              )}
            >
              <item.icon
                size={20}
                className={cn(
                  "shrink-0",
                  isActive ? "text-tiger-red" : "text-semi-grey"
                )}
              />
              {expanded && <span className="truncate">{label}</span>}
            </Link>
          );

          if (!expanded) {
            return (
              <Tooltip key={item.href} delayDuration={0}>
                <TooltipTrigger asChild>{linkContent}</TooltipTrigger>
                <TooltipContent side="right" sideOffset={8}>
                  {label}
                </TooltipContent>
              </Tooltip>
            );
          }
          return linkContent;
        })}
      </nav>

      {/* Logout */}
      <div className="px-2 pb-3 space-y-1 border-t border-border pt-3">
        {expanded ? (
          <button
            onClick={handleLogout}
            className="flex items-center gap-3 w-full rounded-[10px] px-3 py-2.5 text-sm font-medium text-semi-black hover:bg-[#f5f5f5] transition-colors"
          >
            <LogOut size={20} className="text-semi-grey shrink-0" />
            <span>{lang === "th" ? "ออกจากระบบ" : "Sign out"}</span>
          </button>
        ) : (
          <Tooltip delayDuration={0}>
            <TooltipTrigger asChild>
              <button
                onClick={handleLogout}
                className="flex items-center justify-center w-full rounded-[10px] py-2.5 text-sm text-semi-black hover:bg-[#f5f5f5] transition-colors"
              >
                <LogOut size={20} className="text-semi-grey" />
              </button>
            </TooltipTrigger>
            <TooltipContent side="right" sideOffset={8}>
              {lang === "th" ? "ออกจากระบบ" : "Sign out"}
            </TooltipContent>
          </Tooltip>
        )}

        {/* Toggle button */}
        <button
          onClick={toggleExpanded}
          className={cn(
            "flex items-center w-full rounded-[10px] px-3 py-2 text-xs text-semi-grey hover:bg-[#f5f5f5] transition-colors",
            !expanded && "justify-center px-0"
          )}
        >
          {expanded ? (
            <>
              <ChevronLeft size={16} className="mr-2" />
              <span>{lang === "th" ? "ย่อเมนู" : "Collapse"}</span>
            </>
          ) : (
            <ChevronRight size={16} />
          )}
        </button>
      </div>
    </aside>
  );
}
