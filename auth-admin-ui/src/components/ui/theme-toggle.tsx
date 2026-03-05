"use client";

import { Sun, Moon } from "lucide-react";
import { useTheme } from "@/contexts/theme";

export function ThemeToggle() {
  const { theme, toggleTheme } = useTheme();

  return (
    <button
      onClick={toggleTheme}
      className="flex items-center justify-center w-9 h-9 rounded-[8px] text-semi-grey hover:text-semi-black dark:text-[#9CA3AF] dark:hover:text-[#E8E8E8] hover:bg-[#f5f5f5] dark:hover:bg-[#2A2A35] transition-colors"
      aria-label={theme === "dark" ? "Switch to light mode" : "Switch to dark mode"}
    >
      {theme === "dark" ? <Sun size={16} /> : <Moon size={16} />}
    </button>
  );
}
