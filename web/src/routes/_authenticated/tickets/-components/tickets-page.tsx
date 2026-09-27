import { useEffect, useMemo, useState } from "react";
import { useMutation, useSuspenseQuery } from "@tanstack/react-query";
import {
  useNavigate,
  useRouteContext,
  useSearch,
} from "@tanstack/react-router";
import { PlusIcon } from "lucide-react";
import { TicketsApis } from "@/api/tickets.ts";
import { orgMembersQueryOpts } from "@/api/units.ts";
import { Body, Header } from "@/components/layout";
import { StatusFilterSelect } from "@/components/shared/entries-filters.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select.tsx";
import { CreateTicketDialog } from "@/routes/_authenticated/tickets/-components/create-ticket-dialog.tsx";
import { DismissTicketDialog } from "@/routes/_authenticated/tickets/-components/dismiss-ticket-dialog.tsx";
import { TicketBoard } from "@/routes/_authenticated/tickets/-components/ticket-board.tsx";
import { TicketDetailSheet } from "@/routes/_authenticated/tickets/-components/ticket-detail-sheet.tsx";
import { canTriage } from "@/routes/_authenticated/tickets/-lib/ticket-rules.ts";
import {
  LIVE_TICKET_STATUSES,
  TICKET_KINDS,
  TICKET_KIND_LABELS,
  TICKET_STATUSES,
  TICKET_STATUS_LABELS,
  type Ticket,
  type TicketKind,
  type TicketStatus,
  type TriageActivityPlan,
} from "@/types";

/** Title/description search — client-side; the API takes status and kind only. */
function matchQuery(tickets: Ticket[], query: string | undefined): Ticket[] {
  const needle = query?.trim().toLowerCase();
  if (!needle) return tickets;
  return tickets.filter(
    (t) =>
      t.title.toLowerCase().includes(needle) ||
      t.description.toLowerCase().includes(needle)
  );
}

export function TicketsPage() {
  const { lanes, kind, q, ticket: selectedId } = useSearch({
    from: "/_authenticated/tickets/",
  });
  const navigate = useNavigate();
  const { profile } = useRouteContext({ from: "/_authenticated" });

  const { data: tickets } = useSuspenseQuery(
    TicketsApis.ticketsQueryOpts(undefined, kind)
  );
  const { data: members } = useSuspenseQuery(orgMembersQueryOpts);

  const createTicket = useMutation(TicketsApis.createTicketMutationOpts);
  const transition = useMutation(TicketsApis.transitionTicketMutationOpts);
  const dismiss = useMutation(TicketsApis.dismissTicketMutationOpts);
  const comment = useMutation(TicketsApis.commentTicketMutationOpts);
  const triage = useMutation(TicketsApis.triageTicketMutationOpts);

  const [createOpen, setCreateOpen] = useState(false);
  const [dismissTarget, setDismissTarget] = useState<Ticket | null>(null);
  /** A refused drag says why, then fades — see `onRejectMove` on the board. */
  const [moveNotice, setMoveNotice] = useState<string | null>(null);

  useEffect(() => {
    if (!moveNotice) return;
    const timer = setTimeout(() => setMoveNotice(null), 8000);
    return () => clearTimeout(timer);
  }, [moveNotice]);

  const role = profile.membership.role;
  const userId = profile.user.id;

  const names = useMemo(
    () =>
      Object.fromEntries(
        (members ?? [])
          .filter((member) => member.user_id)
          .map((member) => [
            member.user_id as string,
            member.user_name ?? "Unnamed member",
          ])
      ),
    [members]
  );

  const visibleLanes = lanes ?? LIVE_TICKET_STATUSES;
  const rows = useMemo(() => matchQuery(tickets ?? [], q), [tickets, q]);

  const busy =
    transition.isPending ||
    dismiss.isPending ||
    comment.isPending ||
    triage.isPending;

  const setSearch = (
    patch: {
      lanes?: TicketStatus[];
      kind?: TicketKind;
      q?: string;
      ticket?: string;
    },
    replace = false
  ) => {
    void navigate({
      to: "/tickets",
      search: (prev) => ({ ...prev, ...patch }),
      replace,
    });
  };

  const runTriage = (
    id: string,
    ticketKind: TicketKind,
    plans: TriageActivityPlan[]
  ) =>
    triage.mutate({
      id,
      kind: ticketKind,
      // Empty ids must be omitted, not sent as "": the handler parses every
      // present id as a UUID and rejects the whole request otherwise.
      activities: plans.map((plan) => ({
        ...plan,
        name: plan.name.trim(),
        parent_id: plan.parent_id || undefined,
        contract_id: plan.contract_id || undefined,
      })),
    });

  return (
    <>
      <Header>
        <h1 className="text-xl font-semibold">Tickets</h1>
        <span className="text-xs text-muted-foreground">
          {tickets.length} tickets ·{" "}
          {canTriage(role)
            ? "you can triage and dismiss"
            : "you can move the ones you raised or hold"}
        </span>
        <Button
          size="sm"
          className="ml-auto"
          onClick={() => setCreateOpen(true)}
        >
          <PlusIcon /> New ticket
        </Button>
      </Header>
      <Body className="flex min-h-0 flex-col">
        <div className="flex flex-wrap items-center gap-2 border-b px-4 py-2">
          <StatusFilterSelect
            options={TICKET_STATUSES.map((status) => ({
              value: status,
              label: TICKET_STATUS_LABELS[status],
            }))}
            selected={visibleLanes}
            onChange={(next) => setSearch({ lanes: next as TicketStatus[] })}
          />
          <NativeSelect
            aria-label="Filter by kind"
            value={kind ?? ""}
            onChange={(event) =>
              setSearch({
                kind: (event.target.value || undefined) as
                  | TicketKind
                  | undefined,
              })
            }
            className="h-8 w-32 text-xs"
          >
            <NativeSelectOption value="">Every kind</NativeSelectOption>
            {TICKET_KINDS.map((value) => (
              <NativeSelectOption key={value} value={value}>
                {TICKET_KIND_LABELS[value]}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <Input
            value={q ?? ""}
            onChange={(event) => setSearch({ q: event.target.value })}
            placeholder="Search tickets"
            aria-label="Search tickets"
            className="h-8 w-56 text-xs"
          />
          <output className="ml-auto text-xs text-muted-foreground">
            {moveNotice ?? `${rows.length} shown`}
          </output>
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto">
          <TicketBoard
            tickets={rows}
            lanes={visibleLanes}
            names={names}
            role={role}
            userId={userId}
            busy={busy}
            onOpen={(id) => setSearch({ ticket: id }, true)}
            onTransition={(id, status) => {
              setMoveNotice(null);
              transition.mutate({ id, status });
            }}
            onDismiss={setDismissTarget}
            onRejectMove={setMoveNotice}
          />
        </div>

        <TicketDetailSheet
          ticketId={selectedId}
          names={names}
          role={role}
          userId={userId}
          busy={busy}
          onClose={() => setSearch({ ticket: undefined }, true)}
          onTransition={(id, status, note) =>
            transition.mutate({ id, status, note })
          }
          onComment={(id, body) => comment.mutate({ id, body })}
          onDismiss={setDismissTarget}
          onTriage={runTriage}
        />

        <CreateTicketDialog
          open={createOpen}
          onOpenChange={setCreateOpen}
          members={members ?? []}
          pending={createTicket.isPending}
          onCreate={(request) =>
            createTicket.mutate(request, {
              onSuccess: (created) => setSearch({ ticket: created.id }, true),
            })
          }
        />

        <DismissTicketDialog
          ticket={dismissTarget}
          pending={dismiss.isPending}
          onOpenChange={(open) => {
            if (!open) setDismissTarget(null);
          }}
          onConfirm={(id) => {
            dismiss.mutate(id);
            setDismissTarget(null);
          }}
        />
      </Body>
    </>
  );
}
