import { ArrowRightIcon, XIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import {
  KanbanItem,
  KanbanItemHandle,
} from "@/components/ui/kanban.tsx";
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
  /** Rendered inside the drag overlay: no handles, no actions. */
  overlay?: boolean;
}

/**
 * A card offers only the moves the server matrix allows (mirrored in
 * `-lib/ticket-rules.ts`), and only to someone the transition route admits:
 * manager|finance, the requester, or the assignee (D-15).
 *
 * Drag and drop is a second path to the same rule — the board resolves a drop
 * through the same matrix before it mutates anything.
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
  overlay = false,
}: TicketCardProps) {
  const requester = names[ticket.requester_id] ?? "unknown";
  const assignee = ticket.assignee_id
    ? (names[ticket.assignee_id] ?? "someone")
    : "unassigned";
  const actable = canActOnTicket(role, userId, ticket);
  const moves = TICKET_TRANSITIONS[ticket.status];
  const dismissable =
    canTriage(role) && DISMISSABLE_STATUSES.includes(ticket.status);

  const body = (
    <>
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
    </>
  );

  if (overlay) {
    return (
      <KanbanItem
        value={ticket.id}
        className="w-72 rounded-lg border bg-card p-2 shadow-lg"
      >
        {body}
      </KanbanItem>
    );
  }

  return (
    <KanbanItem
      value={ticket.id}
      className="rounded-lg border bg-background pb-2"
    >
      {/* The handle is the card text only: the action buttons stay outside it
          so a click can never be read as the start of a drag. */}
      <KanbanItemHandle className="block p-2">{body}</KanbanItemHandle>
      {actable && (moves.length > 0 || dismissable) && (
        <div className="flex flex-wrap items-center gap-1 px-2">
          {moves.map((next) => (
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
          {dismissable && (
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
    </KanbanItem>
  );
}
