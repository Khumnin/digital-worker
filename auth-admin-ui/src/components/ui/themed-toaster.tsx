"use client";

import { Toaster } from "@/components/ui/sonner";
import { useTheme } from "@/contexts/theme";

export function ThemedToaster() {
  const { theme } = useTheme();
  return <Toaster richColors position="top-right" theme={theme} />;
}
