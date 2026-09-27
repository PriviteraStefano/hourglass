import { useState } from "react";
import { ArrowRightIcon, MessageSquareIcon, SendIcon, XIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  KIND_LABEL,
  STATUS_HINT,
  STATUS_LABEL,
  canTriage,
  isPrototypeTicket,
  nextStatuses,
  type Ticket,
  type TicketStatus,
} from "@/routes/_authenticated/prototype/-fixtures/tickets.ts";
import { cn } from "@/lib/utils.ts";

/**
 * PROTOTYPE VARIANT B — "lifecycle board".
 *
 * Information hierarchy: the status is the column, the ticket is the card. Every
 * action offered on a card is one the locked matrix allows (plus Dismiss on
 * open/triage for manager|finance) — the board is a picture of the state machine
 * rather than a work queue.
 *
 * Primary affordance: move a ticket one legal step, or leave a comment.
 */

const COLUMNS: TicketStatus[] = [
  "open",
  "triage",
  "planned",
  "in_progress",
  "resolved",
  "closed",
  "dismissed",
];

export interface TicketsVariantBProps {
  tickets: Ticket[];
  names: Record<string, string>;
  role: string;
  userId: string;
  onTransition: (id: string, status: TicketStatus, note?: string) => void;
  onDismiss: (id: string) => void;
  onComment: (id: string, body: string) => void;
  busy: boolean;
  error: string | null;
}

export function TicketsVariantB({
  tickets,
  names,
  role,
  userId,
  onTransition,
  onDismiss,
  onComment,
  busy,
  error,
}: TicketsVariantBProps) {
  const [commenting, setCommenting] = useState<string | null>(null);
  const [body, setBody] = useState("");

  return (
    <div className="flex h-full flex-col pb-16">
      <div className="flex items-start justify-between gap-4 border-b px-4 py-3">
        <div>
          <h1 className="text-xl font-semibold">Tickets</h1>
          <p className="text-xs text-muted-foreground">
            {tickets.length} tickets · the board only offers transitions the
            matrix allows
          </p>
        </div>
        <Badge variant="outline" className="font-normal">
          {canTriage(role) ? "you can triage and dismiss" : "you can move your own"}
        </Badge>
      </div>

      {error && <p className="border-b bg-muted/40 px-4 py-2 text-xs">{error}</p>}

      <div className="flex min-h-0 flex-1 gap-3 overflow-auto p-4">
        {COLUMNS.map((status) => {
          const lane = tickets.filter((t) => t.status === status);
          return (
            <div key={status} className="flex w-72 shrink-0 flex-col rounded-lg border">
              <div className="border-b px-3 py-2">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-medium">{STATUS_LABEL[status]}</span>
                  <span className="text-[11px] text-muted-foreground">
                    {lane.length}
                  </span>
                </div>
                <p className="text-[10px] text-muted-foreground">
                  {STATUS_HINT[status]}
                </p>
              </div>
              <div className="flex-1 space-y-2 overflow-auto p-2">
                {lane.map((ticket) => (
                  <div
                    key={ticket.id}
                    className={cn(
                      "rounded-lg border bg-card p-2",
                      isPrototypeTicket(ticket.id) && "ring-1 ring-ring/20"
                    )}
                  >
                    <div className="flex items-start gap-2">
                      <span className="text-xs font-medium">{ticket.title}</span>
                      <Badge
                        variant="outline"
                        className="ml-auto shrink-0 font-normal"
                      >
                        {KIND_LABEL[ticket.kind]}
                      </Badge>
                    </div>
                    <p className="mt-0.5 text-[10px] text-muted-foreground">
                      {names[ticket.requester_id] ?? "unknown"}
                      {ticket.assignee_id
                        ? ` → ${names[ticket.assignee_id] ?? "someone"}`
                        : " → unassigned"}
                    </p>

                    <div className="mt-1.5 flex flex-wrap items-center gap-1">
                      {nextStatuses(ticket.status).map((next) => (
                        <Button
                          key={next}
                          size="sm"
                          variant="outline"
                          disabled={busy}
                          onClick={() => onTransition(ticket.id, next)}
                        >
                          <ArrowRightIcon /> {STATUS_LABEL[next]}
                        </Button>
                      ))}
                      {canTriage(role) &&
                        (ticket.status === "open" || ticket.status === "triage") && (
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={busy}
                            onClick={() => onDismiss(ticket.id)}
                          >
                            <XIcon /> Dismiss
                          </Button>
                        )}
                      <Button
                        size="sm"
                        variant="ghost"
                        className="ml-auto"
                        onClick={() =>
                          setCommenting(commenting === ticket.id ? null : ticket.id)
                        }
                      >
                        <MessageSquareIcon />
                      </Button>
                    </div>

                    {commenting === ticket.id && (
                      <div className="mt-2 flex items-center gap-1">
                        <Input
                          value={body}
                          onChange={(e) => setBody(e.target.value)}
                          placeholder="Comment"
                          className="h-7 text-xs"
                        />
                        <Button
                          size="icon-sm"
                          aria-label="Send comment"
                          disabled={busy || body.trim().length === 0}
                          onClick={() => {
                            onComment(ticket.id, body.trim());
                            setBody("");
                            setCommenting(null);
                          }}
                        >
                          <SendIcon />
                        </Button>
                      </div>
                    )}

                    {ticket.assignee_id === userId && (
                      <p className="mt-1 text-[10px] text-muted-foreground">
                        assigned to you
                      </p>
                    )}
                  </div>
                ))}
                {lane.length === 0 && (
                  <p className="py-6 text-center text-xs text-muted-foreground">
                    Empty
                  </p>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
