"use client";

import { useEffect, useState } from "react";
import { toast } from "sonner";
import { Loader2, User, KeyRound, Shield, Eye, EyeOff } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Badge } from "@/components/ui/badge";
import { useAuth } from "@/contexts/auth";
import { meApi, type MeProfile, ApiError } from "@/lib/api";

export default function MyProfilePage() {
  const { getToken, tenantSlug } = useAuth();
  const [profile, setProfile] = useState<MeProfile | null>(null);
  const [loading, setLoading] = useState(true);

  // Edit name
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [savingName, setSavingName] = useState(false);

  // Change password
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showCurrent, setShowCurrent] = useState(false);
  const [showNew, setShowNew] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);

  useEffect(() => {
    load();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  async function load() {
    setLoading(true);
    try {
      const token = await getToken();
      if (!token) return;
      const data = await meApi.get(token);
      setProfile(data);
      setFirstName(data.first_name ?? "");
      setLastName(data.last_name ?? "");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to load profile");
    } finally {
      setLoading(false);
    }
  }

  async function handleSaveName(e: React.FormEvent) {
    e.preventDefault();
    setSavingName(true);
    try {
      const token = await getToken();
      if (!token) return;
      const data = await meApi.updateProfile(token, firstName.trim(), lastName.trim());
      setProfile((prev) => prev ? { ...prev, first_name: data.first_name, last_name: data.last_name } : prev);
      setFirstName(data.first_name ?? "");
      setLastName(data.last_name ?? "");
      toast.success("Profile updated.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to update profile");
    } finally {
      setSavingName(false);
    }
  }

  async function handleChangePassword(e: React.FormEvent) {
    e.preventDefault();
    if (newPassword !== confirmPassword) {
      toast.error("New passwords do not match.");
      return;
    }
    if (newPassword.length < 8) {
      toast.error("Password must be at least 8 characters.");
      return;
    }
    setSavingPassword(true);
    try {
      const token = await getToken();
      if (!token) return;
      await meApi.changePassword(token, currentPassword, newPassword);
      toast.success("Password changed. Other sessions have been revoked.");
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to change password");
    } finally {
      setSavingPassword(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Loader2 className="animate-spin text-tiger-red" size={28} />
      </div>
    );
  }

  return (
    <div className="space-y-5 max-w-2xl">
      <div>
        <h1 className="text-base font-semibold text-semi-black">My Profile</h1>
        <p className="text-xs text-semi-grey mt-0.5">View and manage your account information</p>
      </div>

      {/* Account Info */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
            <User size={15} className="text-tiger-red" />
            Account Info
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-sm">
          <div className="flex justify-between">
            <span className="text-semi-grey">Email</span>
            <span className="text-xs text-semi-black font-medium">{profile?.email}</span>
          </div>
          <Separator />
          <div className="flex justify-between">
            <span className="text-semi-grey">User ID</span>
            <span className="font-mono text-xs text-semi-black max-w-[260px] truncate">{profile?.user_id}</span>
          </div>
          <Separator />
          <div className="flex justify-between">
            <span className="text-semi-grey">Tenant</span>
            <span className="font-mono text-xs text-semi-black">{tenantSlug}</span>
          </div>
          <Separator />
          <div className="flex justify-between">
            <span className="text-semi-grey">Joined</span>
            <span className="text-xs text-semi-black">
              {profile?.created_at ? new Date(profile.created_at).toLocaleDateString("th-TH") : "—"}
            </span>
          </div>
        </CardContent>
      </Card>

      {/* Edit Display Name */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black">Display Name</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSaveName} className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-semi-grey">First Name</Label>
                <Input
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  placeholder="First name"
                  className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-10 text-sm"
                />
              </div>
              <div className="space-y-1.5">
                <Label className="text-xs font-medium text-semi-grey">Last Name</Label>
                <Input
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  placeholder="Last name"
                  className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-10 text-sm"
                />
              </div>
            </div>
            <div className="flex justify-end">
              <Button
                type="submit"
                disabled={savingName}
                className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-xs h-8 px-4"
              >
                {savingName && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
                Save
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* Roles & Access */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
            <Shield size={15} className="text-tiger-red" />
            Roles & Access
          </CardTitle>
        </CardHeader>
        <CardContent>
          {profile?.roles && profile.roles.length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {profile.roles.map((role) => (
                <Badge
                  key={role}
                  variant="outline"
                  className={`text-xs ${
                    role === "super_admin"
                      ? "border-tiger-red text-tiger-red"
                      : role === "admin"
                      ? "border-orange-400 text-orange-600"
                      : "border-border text-semi-grey"
                  }`}
                >
                  {role}
                </Badge>
              ))}
            </div>
          ) : (
            <p className="text-xs text-semi-grey">No roles assigned.</p>
          )}
        </CardContent>
      </Card>

      {/* Security — Change Password */}
      <Card className="rounded-[10px] border-border shadow-none">
        <CardHeader className="pb-3">
          <CardTitle className="text-sm font-semibold text-semi-black flex items-center gap-2">
            <KeyRound size={15} className="text-tiger-red" />
            Security
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleChangePassword} className="space-y-3">
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-semi-grey">Current Password</Label>
              <div className="relative">
                <Input
                  type={showCurrent ? "text" : "password"}
                  value={currentPassword}
                  onChange={(e) => setCurrentPassword(e.target.value)}
                  required
                  placeholder="••••••••"
                  className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-10 text-sm pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowCurrent((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-semi-grey hover:text-semi-black"
                  tabIndex={-1}
                >
                  {showCurrent ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </div>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-semi-grey">New Password</Label>
              <div className="relative">
                <Input
                  type={showNew ? "text" : "password"}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  required
                  placeholder="Min. 8 characters"
                  className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-10 text-sm pr-10"
                />
                <button
                  type="button"
                  onClick={() => setShowNew((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-semi-grey hover:text-semi-black"
                  tabIndex={-1}
                >
                  {showNew ? <EyeOff size={16} /> : <Eye size={16} />}
                </button>
              </div>
            </div>
            <div className="space-y-1.5">
              <Label className="text-xs font-medium text-semi-grey">Confirm New Password</Label>
              <Input
                type="password"
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                required
                placeholder="Re-enter new password"
                className="bg-[#f0f0f0] dark:bg-input border-[#f0f0f0] dark:border-input rounded-[10px] h-10 text-sm"
              />
            </div>
            <div className="flex justify-end">
              <Button
                type="submit"
                disabled={savingPassword}
                className="rounded-[1000px] bg-tiger-red hover:bg-tiger-red/90 text-white text-xs h-8 px-4"
              >
                {savingPassword && <Loader2 className="mr-1.5 h-3 w-3 animate-spin" />}
                Change Password
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
