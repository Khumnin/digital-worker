"use client";

import { useState, useEffect } from "react";

/** Tailwind default breakpoints */
export type Breakpoint = "base" | "sm" | "md" | "lg" | "xl";

const BREAKPOINTS: { name: Breakpoint; minWidth: number }[] = [
  { name: "xl", minWidth: 1280 },
  { name: "lg", minWidth: 1024 },
  { name: "md", minWidth: 768 },
  { name: "sm", minWidth: 640 },
  { name: "base", minWidth: 0 },
];

/**
 * Returns the current Tailwind breakpoint name.
 * SSR-safe: defaults to "lg" (desktop) on the server to avoid hydration mismatches
 * and flash-of-wrong-layout.
 */
export function useBreakpoint(): Breakpoint {
  // Default to "lg" so SSR renders desktop layout; client hydrates immediately.
  const [breakpoint, setBreakpoint] = useState<Breakpoint>("lg");

  useEffect(() => {
    function update() {
      const width = window.innerWidth;
      for (const bp of BREAKPOINTS) {
        if (width >= bp.minWidth) {
          setBreakpoint(bp.name);
          return;
        }
      }
    }

    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);

  return breakpoint;
}

/**
 * Convenience hook — returns true when viewport is below the `md` breakpoint
 * (i.e. phones and small tablets, < 768px).
 */
export function useIsMobile(): boolean {
  const bp = useBreakpoint();
  return bp === "base" || bp === "sm";
}
