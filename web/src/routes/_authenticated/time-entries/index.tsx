import {createFileRoute} from '@tanstack/react-router'
import {TimeEntriesPage} from "@/src/routes/_authenticated/time-entries/-components/time-entries-page.tsx";
import {z} from "zod";
import {TimeEntriesApis} from "@/src/api/time-entries.ts";
import {ProjectsApis} from "@/src/api/projects.ts";

export const Route = createFileRoute('/_authenticated/time-entries/')({
  validateSearch: z.object({
    date: z.coerce.date().default(new Date()),
    month: z.coerce.date().default(new Date()),
  }),
  loaderDeps: ({search}) => (search),
  loader: ({deps: {month, date}, context: {client}}) =>
    Promise.all([
      client.ensureQueryData(TimeEntriesApis.timeEntriesMonthlySummaryQueryOpts(
        month.getMonth() + 1,
        month.getFullYear()
      )),
      client.ensureQueryData(TimeEntriesApis.timeEntryQueryOpts(date)),
      client.ensureQueryData(ProjectsApis.projectsQueryOpts("all"))
    ]),
  component: TimeEntriesPage,
  pendingMs: 50
})


