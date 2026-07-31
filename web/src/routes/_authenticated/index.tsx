import { createFileRoute } from "@tanstack/react-router";
import { LoaderIcon } from "lucide-react";
import { Body } from "@/components/layout/body.tsx";
import { Header } from "@/components/layout/header.tsx";
import { RouteError } from "@/components/layout/route-error";
import { TodayPage } from "@/routes/_authenticated/-components/today-page.tsx";

export const Route = createFileRoute("/_authenticated/")({
  component: TodayPage,
  // Leaf-level boundary (Phase 8 convention): loader/query errors attach to
  // THIS match, which is rebuilt on navigation; the layout-level fallback stays
  // as backup. Today's data queries are component-level (non-suspense), so
  // query failures render the locked in-Body error state inside TodayPage.
  errorComponent: RouteError,
  pendingComponent: () => (
    <>
      <Header>
        <h1 className="text-3xl font-semibold">Today</h1>
      </Header>
      <Body>
        <div className="flex h-full items-center justify-center">
          <LoaderIcon className="animate-spin text-muted-foreground" />
        </div>
      </Body>
    </>
  ),
});
