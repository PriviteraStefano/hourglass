import { createFileRoute } from "@tanstack/react-router";
import { ExpensesPage } from "@/routes/_authenticated/expenses/-components/expenses-page.tsx";
import { z } from "zod";
import { ExpensesApis } from "@/api/expenses.ts";
import { ActivitiesApis } from "@/api/activities.ts";
import { listStatusesSchema } from "@/lib/list-filters";
import { RouteError } from "@/components/layout/route-error";

export const Route = createFileRoute("/_authenticated/expenses/")({
  validateSearch: z.object({
    date: z.coerce.date().default(new Date()),
    month: z.coerce.date().default(new Date()),
    // List-view filters (P0-2): URL-shareable per ADR-FE-017.
    listStatuses: listStatusesSchema,
    listCategory: z.string().optional(),
    listFrom: z.string().optional(),
    listTo: z.string().optional(),
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
      client.ensureQueryData(ActivitiesApis.activitiesQueryOpts("all")),
    ]),
  // Leaf-level boundary (P0-4): the error attaches to THIS match, which is
  // rebuilt on navigation — unlike the layout-level fallback, whose match
  // persists and can keep the panel tripped after recovery.
  errorComponent: RouteError,
  component: ExpensesPage,
  pendingMs: 50,
});
