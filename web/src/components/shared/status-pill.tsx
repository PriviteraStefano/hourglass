import * as React from "react";
import { cn } from "@/lib/utils.ts";

/**
 * The five semantic status roles from `docs/design/LANGUAGE.md`, backed by the
 * `--status-*` token pairs in `web/src/index.css`.
 *
 * These are *meaning* (neutral / info / success / warning / danger), never
 * interaction chrome: do not reach for `primary`, `accent`, `muted`, or
 * `destructive` to express a domain state, and never map `danger` onto
 * `destructive` (an action role).
 */
export type StatusTone = "neutral" | "info" | "success" | "warning" | "danger";

const TONE_CLASS: Record<StatusTone, string> = {
  neutral: "bg-status-neutral text-status-neutral-foreground",
  info: "bg-status-info text-status-info-foreground",
  success: "bg-status-success text-status-success-foreground",
  warning: "bg-status-warning text-status-warning-foreground",
  danger: "bg-status-danger text-status-danger-foreground",
};

export interface StatusPillProps {
  tone: StatusTone;
  children: React.ReactNode;
  className?: string;
}

export function StatusPill({ tone, children, className }: StatusPillProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded px-2 py-0.5 text-xs font-medium whitespace-nowrap",
        TONE_CLASS[tone],
        className
      )}
    >
      {children}
    </span>
  );
}
