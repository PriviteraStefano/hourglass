import { useState } from "react";
import { createFileRoute, useNavigate, useRouteContext } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { LoaderIcon } from "lucide-react";
import { z } from "zod";
import { ActivitiesApis } from "@/api/activities.ts";
import { orgMembersQueryOpts } from "@/api/units.ts";
import { Body } from "@/components/layout/body.tsx";
import { RouteError } from "@/components/layout/route-error";
import { PrototypeSwitcher } from "@/components/prototype/prototype-switcher.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  commentTicket,
  dismissTicket,
  isPrototypeTicket,
  ticketsQueryOpts,
  transitionTicket,
  triageTicket,
  type TicketKind,
  type TicketStatus,
  type TriagePlan,
} from "@/routes/_authenticated/prototype/-fixtures/tickets.ts";
import { TicketsVariantA } from "@/routes/_authenticated/prototype/-components/tickets-variant-a.tsx";
import { TicketsVariantB } from "@/routes/_authenticated/prototype/-components/tickets-variant-b.tsx";
import { TicketsVariantC } from "@/routes/_authenticated/prototype/-components/tickets-variant-c.tsx";

/**
 * THROWAWAY PROTOTYPE ROUTE — `skill://prototype` (UI branch, sub-shape B).
 *
 * Question: what should the **tickets** surface look like? (contracts:
 * `employee.md` E-05 raise internal demand · `manager.md` M-08 triage). No route
 * exists today; the nav item is a disabled placeholder.
 *
 * Everything here is live against the demo DB — including triage, which creates
 * the activities it plans. To keep that honest but contained, the list defaults
 * to the tickets this prototype seeded (`019df940…`, see
 * `scripts/seed_prototype.sql` §4) with a toggle to include the rest.
 *
 * Three variants, switchable with `?variant=A|B|C`:
 *   A — triage inbox       (open vs in-triage, decision panel with the plan rows)
 *   B — lifecycle board    (one column per status, only legal moves offered)
 *   C — table + conversation (thread first, actions second)
 */
export const Route = createFileRoute("/_authenticated/prototype/tickets")({
  validateSearch: z.object({
    variant: z.enum(["A", "B", "C"]).optional(),
    scope: z.enum(["fixtures", "all"]).optional(),
  }),
  component: TicketsPrototype,
  errorComponent: RouteError,
});

const VARIANTS = [
  { key: "A", name: "Triage inbox" },
  { key: "B", name: "Lifecycle board" },
  { key: "C", name: "Table + conversation" },
];

function TicketsPrototype() {
  const { variant = "A", scope = "fixtures" } = Route.useSearch();
  const navigate = useNavigate({ from: Route.fullPath });
  const { profile } = useRouteContext({ from: "/_authenticated" });
  const queryClient = useQueryClient();
  const [error, setError] = useState<string | null>(null);

  const tickets = useQuery(ticketsQueryOpts());
  const members = useQuery(orgMembersQueryOpts);
  const owned = useQuery(ActivitiesApis.activitiesQueryOpts("owned"));
  const adopted = useQuery(ActivitiesApis.activitiesQueryOpts("adopted"));

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["prototype", "tickets"] });
    void queryClient.invalidateQueries({ queryKey: ["prototype", "ticket"] });
  };

  const run = {
    onSuccess: () => {
      setError(null);
      invalidate();
    },
    onError: (e: Error) => setError(e.message),
  };

  const triage = useMutation({
    mutationFn: ({
      id,
      kind,
      plans,
    }: {
      id: string;
      kind: TicketKind;
      plans: TriagePlan[];
    }) =>
      triageTicket(
        id,
        kind,
        plans.filter((plan) => plan.name.trim() !== "")
      ),
    ...run,
  });

  const transition = useMutation({
    mutationFn: ({
      id,
      status,
      note,
    }: {
      id: string;
      status: TicketStatus;
      note?: string;
    }) => transitionTicket(id, status, note),
    ...run,
  });

  const dismiss = useMutation({ mutationFn: (id: string) => dismissTicket(id), ...run });
  const comment = useMutation({
    mutationFn: ({ id, body }: { id: string; body: string }) => commentTicket(id, body),
    ...run,
  });

  const all = tickets.data ?? [];
  const rows = scope === "fixtures" ? all.filter((t) => isPrototypeTicket(t.id)) : all;

  const names: Record<string, string> = Object.fromEntries(
    (members.data ?? [])
      .filter((m) => m.user_id)
      .map((m) => [m.user_id as string, m.user_name ?? "Unnamed member"])
  );
  const activityOptions = [...(owned.data ?? []), ...(adopted.data ?? [])].map(
    (a) => ({ id: a.id, name: a.name })
  );

  const busy =
    triage.isPending ||
    transition.isPending ||
    dismiss.isPending ||
    comment.isPending;

  const shared = {
    tickets: rows,
    names,
    role: profile.membership.role,
    userId: profile.user.id,
    onTransition: (id: string, status: TicketStatus, note?: string) =>
      transition.mutate({ id, status, note }),
    onDismiss: (id: string) => dismiss.mutate(id),
    onComment: (id: string, body: string) => comment.mutate({ id, body }),
    busy,
    error,
  };

  return (
    <Body className="flex min-h-0 flex-col">
      <div className="flex flex-wrap items-center gap-2 border-b bg-muted/30 px-4 py-1.5 text-[11px] text-muted-foreground">
        <span className="font-medium text-foreground">Prototype</span>
        <span>
          triage, transitions and comments write to the local demo DB; triage also
          creates the activities it plans.
        </span>
        <div className="ml-auto flex items-center gap-2">
          {(
            [
              ["fixtures", "Prototype fixtures"],
              ["all", "All tickets"],
            ] as const
          ).map(([key, label]) => (
            <Button
              key={key}
              size="sm"
              variant={scope === key ? "secondary" : "ghost"}
              onClick={() =>
                void navigate({
                  search: (prev) => ({ ...prev, scope: key }),
                  replace: true,
                })
              }
            >
              {label}
            </Button>
          ))}
        </div>
      </div>

      {tickets.isPending && (
        <div className="flex flex-1 items-center justify-center">
          <LoaderIcon className="animate-spin text-muted-foreground" />
        </div>
      )}

      {tickets.data && variant === "A" && (
        <TicketsVariantA
          {...shared}
          activities={activityOptions}
          onTriage={(id, kind, plans) => triage.mutate({ id, kind, plans })}
        />
      )}
      {tickets.data && variant === "B" && <TicketsVariantB {...shared} />}
      {tickets.data && variant === "C" && <TicketsVariantC {...shared} />}

      <PrototypeSwitcher
        variants={VARIANTS}
        current={variant}
        onSelect={(key) =>
          void navigate({
            search: (prev) => ({ ...prev, variant: key as "A" | "B" | "C" }),
            replace: true,
          })
        }
      />
    </Body>
  );
}
