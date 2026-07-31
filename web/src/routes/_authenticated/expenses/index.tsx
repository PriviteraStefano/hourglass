import { createFileRoute } from "@tanstack/react-router";
import { ExpensesPage } from "@/routes/_authenticated/expenses/-components/expenses-page.tsx";
import { z } from "zod";
import { ExpensesApis } from "@/api/expenses.ts";
import { ProjectsApis } from "@/api/projects.ts";

export const Route = createFileRoute("/_authenticated/expenses/")({
  validateSearch: z.object({
    date: z.coerce.date().default(new Date()),
    month: z.coerce.date().default(new Date()),
  }),
  loaderDeps: ({ search }) => search,
  loader: ({ deps: { month, date }, context: { client } }) =>
    Promise.all([
      client.ensureQueryData(
        ExpensesApis.expensesForMonthQueryOpts(
          month.getMonth() + 1,
          month.getFullYear()
        )
      ),
      client.ensureQueryData(ExpensesApis.expenseQueryOpts(date)),
      client.ensureQueryData(ProjectsApis.projectsQueryOpts("all")),
    ]),
  component: ExpensesPage,
  pendingMs: 50,
});
