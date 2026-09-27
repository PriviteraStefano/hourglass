import { StatusPill } from "@/components/shared/status-pill.tsx";
import { TicketCard } from "@/routes/_authenticated/tickets/-components/ticket-card.tsx";
import { TICKET_STATUS_TONES } from "@/routes/_authenticated/tickets/-lib/ticket-rules.ts";
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
}

/**
 * One column per enabled status. Lanes wrap instead of scrolling
 * horizontally, so the terminal lanes can be switched on without hiding the
 * live ones off-screen (see the tickets spec, issue #56).
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
}: TicketBoardProps) {
  if (lanes.length === 0) {
    return (
      <p className="p-8 text-center text-sm text-muted-foreground">
        No lanes selected — pick a status to see tickets.
      </p>
    );
  }

  return (
    <div className="flex flex-wrap items-start gap-3 p-4">
      {lanes.map((status) => {
        const lane = tickets.filter((ticket) => ticket.status === status);
        return (
          <section
            key={status}
            className="flex w-72 shrink-0 flex-col rounded-lg border bg-card"
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
            <div className="space-y-2 p-2">
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
            </div>
          </section>
        );
      })}
    </div>
  );
}
