"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Building2, Users, Activity, TrendingUp } from "lucide-react";
import { useAuth } from "@/contexts/auth";
import { tenantApi, userApi, type DashboardStats } from "@/lib/api";

const EMPTY_STATS: DashboardStats = {
  total_tenants: 0,
  active_tenants: 0,
  total_users: 0,
  active_users: 0,
  recent_logins_24h: 0,
};

export default function DashboardPage() {
  const { getToken, isSuperAdmin } = useAuth();
  const [stats, setStats] = useState<DashboardStats>(EMPTY_STATS);

  useEffect(() => {
    async function load() {
      const token = await getToken();
      if (!token) return;
      try {
        const [tenants, users] = await Promise.allSettled([
          isSuperAdmin ? tenantApi.list(token) : Promise.resolve(null),
          userApi.list(token, { page_size: 1 }),
        ]);

        const tenantData =
          tenants.status === "fulfilled" && tenants.value
            ? tenants.value
            : null;
        const userData =
          users.status === "fulfilled" && users.value ? users.value : null;

        setStats({
          total_tenants: tenantData?.total ?? 0,
          active_tenants: tenantData?.total ?? 0,
          total_users: userData?.total ?? 0,
          active_users: userData?.total ?? 0,
          recent_logins_24h: 0,
        });
      } catch {}
    }
    load();
  }, [getToken, isSuperAdmin]);

  const statCards = [
    ...(isSuperAdmin
      ? [
          {
            title: "Tenants",
            titleTh: "ผู้เช่าระบบ",
            value: stats.total_tenants,
            sub: `${stats.active_tenants} active`,
            icon: Building2,
          },
        ]
      : []),
    {
      title: "Total Users",
      titleTh: "ผู้ใช้งานทั้งหมด",
      value: stats.total_users,
      sub: `${stats.active_users} active`,
      icon: Users,
    },
    {
      title: "Logins (24h)",
      titleTh: "เข้าสู่ระบบ (24 ชม.)",
      value: stats.recent_logins_24h,
      sub: "recent activity",
      icon: Activity,
    },
  ];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {statCards.map((card) => (
          <Card key={card.title} className="rounded-[10px] border-border shadow-none">
            <CardHeader className="flex flex-row items-center justify-between pb-2">
              <CardTitle className="text-sm font-medium text-semi-grey">
                {card.titleTh}
              </CardTitle>
              <card.icon size={18} className="text-tiger-red" />
            </CardHeader>
            <CardContent>
              <p className="text-3xl font-bold text-semi-black">{card.value}</p>
              <p className="text-xs text-semi-grey mt-1">{card.sub}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      {/* Quick links */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
            <TrendingUp size={16} className="text-tiger-red" />
            Quick Actions
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {[
              { label: "Provision Tenant", href: "/dashboard/tenants", show: isSuperAdmin },
              { label: "Invite User", href: "/dashboard/users", show: true },
              { label: "Manage Roles", href: "/dashboard/roles", show: true },
              { label: "View Audit Log", href: "/dashboard/audit", show: true },
            ]
              .filter((a) => a.show)
              .map((action) => (
                <a
                  key={action.label}
                  href={action.href}
                  className="flex items-center justify-center px-4 py-3 rounded-[10px] bg-[#f0f0f0] text-sm font-medium text-semi-black hover:bg-[#e8e8e8] transition-colors text-center"
                >
                  {action.label}
                </a>
              ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
