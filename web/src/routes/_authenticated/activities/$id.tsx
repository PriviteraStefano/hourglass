import { createFileRoute } from "@tanstack/react-router";
import { ActivityDetail } from "./-components/activity-detail";

export const Route = createFileRoute("/_authenticated/activities/$id")({
  component: ActivityDetailPage,
  validateSearch: (search: Record<string, unknown>) => ({
    from: (search.from as "owned" | "adopted" | "all") || "owned",
  }),
});

function ActivityDetailPage() {
  const { id } = Route.useParams();
  const search = Route.useSearch();

  return <ActivityDetail id={id} fromTab={search.from} />;
}
