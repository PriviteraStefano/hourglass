import { createFileRoute } from "@tanstack/react-router";
import { TimeEntriesPage } from "@/routes/_authenticated/time-entries/-components/time-entries-page.tsx";
import { z } from "zod";
import { TimeEntriesApis } from "@/api/time-entries.ts";
import { ProjectsApis } from "@/api/projects.ts";
import { entryStatusSchema, listStatusesSchema } from "@/lib/list-filters";

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
      client.ensureQueryData(ProjectsApis.projectsQueryOpts("all")),
    ]),
  component: TimeEntriesPage,
  pendingMs: 50,
});
