import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { RouteError } from "@/components/layout/route-error";
import { ApprovalsPage } from "@/routes/_authenticated/approvals/-components/approvals-page.tsx";

export const Route = createFileRoute("/_authenticated/approvals/")({
  // URL-shareable stage filter (ADR-FE-017): /approvals?stage=finance.
  // Optional — defaults to the user's first approval stage at render time.
  validateSearch: z.object({
    stage: z.enum(["manager", "finance"]).optional(),
  }),
  // Leaf-level boundary (Phase 8 convention). The pending queries are
  // component-level (non-suspense), so a 403/failure renders the locked
  // in-Body error state inside ApprovalsPage — never "Queue is clear"
  // (T-10-05-2). RouteError stays as the loader-level fallback.
  errorComponent: RouteError,
  component: ApprovalsPage,
});
