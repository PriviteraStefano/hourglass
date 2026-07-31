import { createFileRoute } from "@tanstack/react-router";
import { WorkingGroupsPage } from "./-components/working-groups-page";
import { WorkingGroupsApis } from "@/api/working-groups";
import { RouteError } from "@/components/layout/route-error";

export const Route = createFileRoute("/_authenticated/working-groups/")({
  component: WorkingGroupsPage,
  loader: ({ context: { client } }) =>
    client.ensureQueryData(WorkingGroupsApis.workingGroupsQueryOpts),
  // Leaf-level boundary (P0-4, Phase 8 convention): the error attaches to THIS
  // match, so the shell frame (sidebar + header) survives loader failures and
  // "Try again" re-runs the loader.
  errorComponent: RouteError,
  pendingMs: 50,
});
