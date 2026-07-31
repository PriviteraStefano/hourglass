import { createFileRoute, Outlet, redirect } from "@tanstack/react-router";
import { AppShell } from "@/components/layout/app-shell.tsx";
import { AuthApis } from "@/api/auth.ts";
import { LoaderIcon } from "lucide-react";
import { RouteError } from "@/components/layout/route-error";

export const Route = createFileRoute("/_authenticated")({
  beforeLoad: async ({ context: { client } }) => {
    try {
      const profile = await client.fetchQuery(AuthApis.profileQueryOpts);
      return { profile };
    } catch {
      // Clear stale/errored profile cache so /login's best-effort check
      // doesn't pick up a cached error and loop
      client.removeQueries({ queryKey: ["auth", "me"] });
      throw redirect({ to: "/login", replace: true });
    }
  },
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
  errorComponent: RouteError,
  pendingComponent: () => (
    <AppShell>
      <LoaderIcon className="animate-spin m-auto" />
    </AppShell>
  ),
});
