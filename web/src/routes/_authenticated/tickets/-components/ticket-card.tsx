import { ArrowRightIcon, XIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  DISMISSABLE_STATUSES,
  TICKET_TRANSITIONS,
  canActOnTicket,
  canTriage,
} from "@/routes/_authenticated/tickets/-lib/ticket-rules.ts";
import {
  TICKET_KIND_LABELS,
  TICKET_STATUS_LABELS,
  type Role,
  type Ticket,
  type TicketStatus,
} from "@/types";

export interface TicketCardProps {
  ticket: Ticket;
  names: Record<string, string>;
  role: Role;
  userId: string;
  busy: boolean;
  onOpen: (id: string) => void;
  onTransition: (id: string, status: TicketStatus) => void;
  onDismiss: (ticket: Ticket) => void;
}

/**
 * A card offers only the moves the server matrix allows (the matrix lives in
 * `-lib/ticket-rules.ts`, mirrored from `internal/core/domain/ticket`), and
 * only to someone the transition route admits: manager|finance, the requester,
 * or the assignee (D-15).
 */
export function TicketCard({
  ticket,
  names,
  role,
  userId,
  busy,
  onOpen,
  onTransition,
  onDismiss,
}: TicketCardProps) {
  const requester = names[ticket.requester_id] ?? "unknown";
  const assignee = ticket.assignee_id
    ? (names[ticket.assignee_id] ?? "someone")
    : "unassigned";

  return (
    <div className="rounded-lg border bg-background p-2">
      <button
        type="button"
        className="w-full text-left text-xs font-medium hover:underline"
        onClick={() => onOpen(ticket.id)}
      >
        {ticket.title}
      </button>
      <div className="mt-1 flex items-center gap-2">
        <Badge variant="outline" className="font-normal">
          {TICKET_KIND_LABELS[ticket.kind]}
        </Badge>
        <span className="text-[10px] text-muted-foreground">
          {requester} → {assignee}
        </span>
      </div>

      {ticket.dismissed_note && (
        <p className="mt-1 text-[10px] text-muted-foreground">
          {ticket.dismissed_note}
        </p>
      )}

      {canActOnTicket(role, userId, ticket) &&
        TICKET_TRANSITIONS[ticket.status].length > 0 && (
          <div className="mt-1.5 flex flex-wrap items-center gap-1">
            {TICKET_TRANSITIONS[ticket.status].map((next) => (
              <Button
                key={next}
                size="sm"
                variant="outline"
                disabled={busy}
                onClick={() => onTransition(ticket.id, next)}
              >
                <ArrowRightIcon /> {TICKET_STATUS_LABELS[next]}
              </Button>
            ))}
            {canTriage(role) && DISMISSABLE_STATUSES.includes(ticket.status) && (
              <Button
                size="sm"
                variant="ghost"
                disabled={busy}
                onClick={() => onDismiss(ticket)}
              >
                <XIcon /> Dismiss
              </Button>
            )}
          </div>
        )}
    </div>
  );
}
