import { createFileRoute } from "@tanstack/react-router";
import { z } from "zod";
import { TicketsApis } from "@/api/tickets.ts";
import { orgMembersQueryOpts } from "@/api/units.ts";
import { RouteError } from "@/components/layout/route-error";
import { ticketKindSchema, ticketLanesSchema } from "@/lib/list-filters";
import { TicketsPage } from "@/routes/_authenticated/tickets/-components/tickets-page.tsx";

export const Route = createFileRoute("/_authenticated/tickets/")({
  validateSearch: z.object({
    /** Board lanes; absent = the four live statuses (see ticketLanesSchema). */
    lanes: ticketLanesSchema,
    kind: ticketKindSchema.optional(),
    q: z.string().optional(),
    /** Selected ticket id — the detail sheet is URL-shareable (ADR-FE-017). */
    ticket: z.string().optional(),
  }),
  loaderDeps: ({ search: { kind } }) => ({ kind }),
  loader: ({ deps: { kind }, context: { client } }) =>
    Promise.all([
      client.ensureQueryData(TicketsApis.ticketsQueryOpts(undefined, kind)),
      client.ensureQueryData(orgMembersQueryOpts),
    ]),
  // Leaf-level boundary: the error attaches to THIS match (P0-4).
  errorComponent: RouteError,
  component: TicketsPage,
  pendingMs: 50,
});
