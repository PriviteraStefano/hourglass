import { createFileRoute } from "@tanstack/react-router";
import { TimeEntriesPage } from "@/routes/_authenticated/time-entries/-components/time-entries-page.tsx";
import { z } from "zod";
import { TimeEntriesApis } from "@/api/time-entries.ts";
import { ActivitiesApis } from "@/api/activities.ts";
import { entryStatusSchema, listStatusesSchema } from "@/lib/list-filters";
import { RouteError } from "@/components/layout/route-error";

export const Route = createFileRoute("/_authenticated/time-entries/")({
  validateSearch: z.object({
    date: z.coerce.date().default(new Date()),
    month: z.coerce.date().default(new Date()),
    // List-view filters (P0-2): URL-shareable per ADR-FE-017.
    listStatuses: listStatusesSchema,
    listFrom: z.string().optional(),
    listTo: z.string().optional(),
  }),
  loaderDeps: ({ search }) => search,
  loader: ({ deps: { month, date }, context: { client } }) =>
    Promise.all([
      client.ensureQueryData(
        TimeEntriesApis.timeEntriesForMonthQueryOpts(
          month.getMonth() + 1,
          month.getFullYear()
        )
      ),
      client.ensureQueryData(TimeEntriesApis.timeEntryQueryOpts(date)),
      client.ensureQueryData(ActivitiesApis.activitiesQueryOpts("all")),
    ]),
  // Leaf-level boundary (P0-4): the error attaches to THIS match, which is
  // rebuilt on navigation — unlike the layout-level fallback, whose match
  // persists and can keep the panel tripped after recovery.
  errorComponent: RouteError,
  component: TimeEntriesPage,
  pendingMs: 50,
});
