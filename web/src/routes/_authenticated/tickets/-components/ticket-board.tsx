import { useMemo, useState } from "react";
import { StatusPill } from "@/components/shared/status-pill.tsx";
import {
  Kanban,
  KanbanBoard,
  KanbanColumn,
  KanbanColumnContent,
  KanbanOverlay,
} from "@/components/ui/kanban.tsx";
import { TicketCard } from "@/routes/_authenticated/tickets/-components/ticket-card.tsx";
import {
  TICKET_STATUS_TONES,
  TICKET_TRANSITIONS,
  canActOnTicket,
} from "@/routes/_authenticated/tickets/-lib/ticket-rules.ts";
import { cn } from "@/lib/utils.ts";
import {
  TICKET_STATUS_HINTS,
  TICKET_STATUS_LABELS,
  type Role,
  type Ticket,
  type TicketStatus,
} from "@/types";

export interface TicketBoardProps {
  tickets: Ticket[];
  lanes: TicketStatus[];
  names: Record<string, string>;
  role: Role;
  userId: string;
  busy: boolean;
  onOpen: (id: string) => void;
  onTransition: (id: string, status: TicketStatus) => void;
  onDismiss: (ticket: Ticket) => void;
  /** A drop the matrix or the role gate refuses — surfaced by the page. */
  onRejectMove: (message: string) => void;
}

/**
 * One lane per enabled status, wrapping instead of scrolling horizontally so
 * the terminal lanes can be switched on without hiding the live ones.
 *
 * Drag and drop runs in the primitive's `onMove` mode: the board is the only
 * place a move is applied, and only when the locked matrix (ADR-BE-016) and the
 * D-15 role gate both allow it. A refused drop changes nothing and says why —
 * an illegal state is never previewed, so there is nothing to roll back. Lane
 * order is server-owned (`updated_at desc`), so same-lane drops are inert.
 */
export function TicketBoard({
  tickets,
  lanes,
  names,
  role,
  userId,
  busy,
  onOpen,
  onTransition,
  onDismiss,
  onRejectMove,
}: TicketBoardProps) {
  const [dragging, setDragging] = useState<Ticket | null>(null);

  const byId = useMemo(
    () =>
      Object.fromEntries(tickets.map((ticket) => [ticket.id, ticket])) as Record<
        string,
        Ticket
      >,
    [tickets]
  );

  const columns = useMemo(
    () =>
      Object.fromEntries(
        lanes.map((status) => [
          status,
          tickets.filter((ticket) => ticket.status === status),
        ])
      ) as Record<string, Ticket[]>,
    [tickets, lanes]
  );

  const handleMove: NonNullable<
    React.ComponentProps<typeof Kanban<Ticket>>["onMove"]
  > = ({ event, activeContainer, overContainer }) => {
    setDragging(null);

    const ticket = byId[String(event.active.id)];
    if (!ticket || activeContainer === overContainer) return;

    const target = overContainer as TicketStatus;
    if (!canActOnTicket(role, userId, ticket)) {
      onRejectMove(
        `“${ticket.title}” is not yours to move — only the requester, the assignee, or manager/finance can.`
      );
      return;
    }
    if (!TICKET_TRANSITIONS[ticket.status].includes(target)) {
      onRejectMove(
        `${TICKET_STATUS_LABELS[ticket.status]} may not move straight to ${
          TICKET_STATUS_LABELS[target] ?? target
        } — the lifecycle only allows ${TICKET_TRANSITIONS[ticket.status]
          .map((next) => TICKET_STATUS_LABELS[next])
          .join(", ")}.`
      );
      return;
    }
    onTransition(ticket.id, target);
  };

  if (lanes.length === 0) {
    return (
      <p className="p-8 text-center text-sm text-muted-foreground">
        No lanes selected — pick a status to see tickets.
      </p>
    );
  }

  return (
    <Kanban
      value={columns}
      // `onMove` mode owns item moves; lanes are never reordered by drag, so
      // this stays a no-op rather than a second source of truth.
      onValueChange={() => {}}
      getItemValue={(ticket) => ticket.id}
      onMove={handleMove}
      onDragStart={(event) =>
        setDragging(byId[String(event.active.id)] ?? null)
      }
      onDragEnd={() => setDragging(null)}
      onDragCancel={() => setDragging(null)}
    >
      <KanbanBoard className="flex flex-wrap items-start gap-3 p-4">
        {lanes.map((status) => {
          const lane = columns[status];
          const isTarget =
            dragging != null &&
            dragging.status !== status &&
            canActOnTicket(role, userId, dragging) &&
            TICKET_TRANSITIONS[dragging.status].includes(status);
          return (
            <KanbanColumn
              key={status}
              value={status}
              className={cn(
                "w-72 shrink-0 rounded-lg border bg-card",
                isTarget && "border-primary ring-2 ring-primary/30"
              )}
            >
              <header className="border-b px-3 py-2">
                <div className="flex items-center gap-2">
                  <StatusPill tone={TICKET_STATUS_TONES[status]}>
                    {TICKET_STATUS_LABELS[status]}
                  </StatusPill>
                  <span className="text-[11px] text-muted-foreground">
                    {lane.length}
                  </span>
                </div>
                <p className="mt-1 text-[10px] text-muted-foreground">
                  {TICKET_STATUS_HINTS[status]}
                </p>
              </header>
              <KanbanColumnContent value={status} className="space-y-2 p-2">
                {lane.map((ticket) => (
                  <TicketCard
                    key={ticket.id}
                    ticket={ticket}
                    names={names}
                    role={role}
                    userId={userId}
                    busy={busy}
                    onOpen={onOpen}
                    onTransition={onTransition}
                    onDismiss={onDismiss}
                  />
                ))}
                {lane.length === 0 && (
                  <p className="py-6 text-center text-xs text-muted-foreground">
                    Empty
                  </p>
                )}
              </KanbanColumnContent>
            </KanbanColumn>
          );
        })}
      </KanbanBoard>
      <KanbanOverlay>
        {({ value }) => {
          const ticket = byId[String(value)];
          if (!ticket) return null;
          return (
            <TicketCard
              overlay
              ticket={ticket}
              names={names}
              role={role}
              userId={userId}
              busy={busy}
              onOpen={onOpen}
              onTransition={onTransition}
              onDismiss={onDismiss}
            />
          );
        }}
      </KanbanOverlay>
    </Kanban>
  );
}
