"use client";

import { useEffect, useRef } from "react";
import { usePathname } from "next/navigation";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

interface MobileDrawerProps {
  open: boolean;
  onClose: () => void;
  children: React.ReactNode;
}

/**
 * Slide-in navigation drawer for mobile viewports.
 *
 * Accessibility:
 * - role="dialog" + aria-modal="true" + aria-label="Navigation menu"
 * - Focus is moved to the drawer panel when it opens
 * - Escape key closes the drawer
 * - Backdrop click closes the drawer
 * - Body scroll is locked while the drawer is open
 *
 * Rendered with md:hidden so it is completely absent from the DOM on tablet/desktop.
 */
export function MobileDrawer({ open, onClose, children }: MobileDrawerProps) {
  const drawerRef = useRef<HTMLDivElement>(null);
  const pathname = usePathname();

  // Auto-close on route change (e.g. user taps a nav link)
  const prevPathRef = useRef(pathname);
  useEffect(() => {
    if (prevPathRef.current !== pathname) {
      prevPathRef.current = pathname;
      onClose();
    }
  }, [pathname, onClose]);

  // Prevent body scroll when the drawer is open
  useEffect(() => {
    if (open) {
      document.body.style.overflow = "hidden";
    } else {
      document.body.style.overflow = "";
    }
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  // Close on Escape key
  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open, onClose]);

  // Move focus into the drawer when it opens
  useEffect(() => {
    if (open && drawerRef.current) {
      drawerRef.current.focus();
    }
  }, [open]);

  return (
    <>
      {/* Backdrop — click to close */}
      <div
        className={cn(
          "fixed inset-0 z-40 bg-black/50 transition-opacity duration-200 md:hidden",
          open ? "opacity-100" : "opacity-0 pointer-events-none"
        )}
        onClick={onClose}
        aria-hidden="true"
      />

      {/* Drawer panel */}
      <div
        ref={drawerRef}
        role="dialog"
        aria-label="Navigation menu"
        aria-modal="true"
        tabIndex={-1}
        className={cn(
          "fixed inset-y-0 left-0 z-50 w-[280px] transform transition-transform duration-200 ease-in-out md:hidden focus:outline-none",
          open ? "translate-x-0" : "-translate-x-full"
        )}
      >
        {/* Close button — positioned outside the drawer edge for clear affordance */}
        <button
          onClick={onClose}
          className="absolute top-4 right-[-48px] flex items-center justify-center w-11 h-11 rounded-full bg-black/30 text-white hover:bg-black/50 transition-colors"
          aria-label="Close navigation menu"
        >
          <X size={18} />
        </button>

        {children}
      </div>
    </>
  );
}
