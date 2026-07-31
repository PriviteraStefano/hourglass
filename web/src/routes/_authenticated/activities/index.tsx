import { createFileRoute } from "@tanstack/react-router";
import { ActivityList } from "./-components/activity-list";

export const Route = createFileRoute("/_authenticated/activities/")({
  component: ActivityList,
  validateSearch: (search: Record<string, unknown>) => ({
    tab: (search.tab as "owned" | "adopted" | "all") || "owned",
  }),
});
