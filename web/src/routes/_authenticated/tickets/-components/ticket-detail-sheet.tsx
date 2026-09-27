import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { format } from "date-fns";
import { ArrowRightIcon, SendIcon, XIcon } from "lucide-react";
import { TicketsApis } from "@/api/tickets.ts";
import { StatusPill } from "@/components/shared/status-pill.tsx";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet.tsx";
import { Skeleton } from "@/components/ui/skeleton.tsx";
import { TriagePanel } from "@/routes/_authenticated/tickets/-components/triage-panel.tsx";
import {
  DISMISSABLE_STATUSES,
  TICKET_STATUS_TONES,
  TICKET_TRANSITIONS,
  canActOnTicket,
  canTriage,
} from "@/routes/_authenticated/tickets/-lib/ticket-rules.ts";
import {
  TICKET_KIND_LABELS,
  TICKET_STATUS_HINTS,
  TICKET_STATUS_LABELS,
  type Role,
  type Ticket,
  type TicketHistoryEntry,
  type TicketKind,
  type TicketStatus,
  type TriageActivityPlan,
} from "@/types";

const HISTORY_ACTION_LABELS: Record<string, string> = {
  created: "Raised",
  updated: "Details updated",
  status_changed: "Status changed",
  triaged: "Triaged",
  activities_created: "Activity created",
  comment_added: "Comment added",
  dismissed: "Dismissed",
};

function describeHistoryEntry(entry: TicketHistoryEntry): string {
  const label =
    HISTORY_ACTION_LABELS[entry.action] ?? entry.action.replace(/_/g, " ");
  const status = entry.payload?.status;
  if (typeof status !== "string") return label;
  return `${label} → ${TICKET_STATUS_LABELS[status as TicketStatus] ?? status}`;
}

export interface TicketDetailSheetProps {
  /** Selected ticket id; undefined keeps the sheet closed. */
  ticketId: string | undefined;
  names: Record<string, string>;
  role: Role;
  userId: string;
  busy: boolean;
  onClose: () => void;
  onTransition: (id: string, status: TicketStatus, note?: string) => void;
  onComment: (id: string, body: string) => void;
  onDismiss: (ticket: Ticket) => void;
  onTriage: (id: string, kind: TicketKind, plans: TriageActivityPlan[]) => void;
}

/**
 * Everything a card cannot carry: the description, who raised it and who holds
 * it, the timestamps, the comment thread, the append-only history, the triage
 * decision and the guarded dismissal.
 */
export function TicketDetailSheet({
  ticketId,
  names,
  role,
  userId,
  busy,
  onClose,
  onTransition,
  onComment,
  onDismiss,
  onTriage,
}: TicketDetailSheetProps) {
  const detail = useQuery(TicketsApis.ticketDetailQueryOpts(ticketId ?? ""));
  const history = useQuery(TicketsApis.ticketHistoryQueryOpts(ticketId ?? ""));
  const [note, setNote] = useState("");
  const [comment, setComment] = useState("");

  // A different ticket means a different draft comment and transition note.
  useEffect(() => {
    setNote("");
    setComment("");
  }, [ticketId]);

  const ticket = detail.data?.ticket;
  const comments = detail.data?.comments ?? [];
  const actable = ticket ? canActOnTicket(role, userId, ticket) : false;

  const actorName = (id: string | undefined) =>
    id ? (names[id] ?? "someone") : "system";

  return (
    <Sheet
      open={Boolean(ticketId)}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <SheetContent side="right" className="w-full gap-0 sm:max-w-xl">
        {!ticket ? (
          <div className="space-y-3 p-6">
            <Skeleton className="h-6 w-2/3" />
            <Skeleton className="h-4 w-1/3" />
            <Skeleton className="h-24 w-full" />
          </div>
        ) : (
          <>
            <SheetHeader className="border-b pr-12">
              <SheetTitle>{ticket.title}</SheetTitle>
              <SheetDescription className="flex flex-wrap items-center gap-2">
                <StatusPill tone={TICKET_STATUS_TONES[ticket.status]}>
                  {TICKET_STATUS_LABELS[ticket.status]}
                </StatusPill>
                <Badge variant="outline" className="font-normal">
                  {TICKET_KIND_LABELS[ticket.kind]}
                </Badge>
                <span>{TICKET_STATUS_HINTS[ticket.status]}</span>
              </SheetDescription>
            </SheetHeader>

            <div className="min-h-0 flex-1 space-y-5 overflow-y-auto p-6">
              <dl className="grid grid-cols-[8rem_1fr] gap-x-3 gap-y-1 text-xs">
                <dt className="text-muted-foreground">Raised by</dt>
                <dd>{actorName(ticket.requester_id)}</dd>
                <dt className="text-muted-foreground">Held by</dt>
                <dd>
                  {ticket.assignee_id
                    ? actorName(ticket.assignee_id)
                    : "unassigned"}
                </dd>
                <dt className="text-muted-foreground">Raised</dt>
                <dd>{format(new Date(ticket.created_at), "MMM d, HH:mm")}</dd>
                <dt className="text-muted-foreground">Updated</dt>
                <dd>{format(new Date(ticket.updated_at), "MMM d, HH:mm")}</dd>
                {ticket.dismissed_note && (
                  <>
                    <dt className="text-muted-foreground">Dismissal</dt>
                    <dd>{ticket.dismissed_note}</dd>
                  </>
                )}
              </dl>

              {ticket.description && (
                <p className="text-xs whitespace-pre-wrap">
                  {ticket.description}
                </p>
              )}

              {actable && (
                <section className="space-y-2">
                  <h3 className="text-xs font-medium">Move it</h3>
                  {TICKET_TRANSITIONS[ticket.status].length > 0 ? (
                    <>
                      <Input
                        value={note}
                        onChange={(event) => setNote(event.target.value)}
                        placeholder="Note for the history (optional)"
                        aria-label="Transition note"
                        className="h-8 text-xs"
                      />
                      <div className="flex flex-wrap gap-2">
                        {TICKET_TRANSITIONS[ticket.status].map((next) => (
                          <Button
                            key={next}
                            size="sm"
                            variant="outline"
                            disabled={busy}
                            onClick={() => {
                              onTransition(
                                ticket.id,
                                next,
                                note.trim() || undefined
                              );
                              setNote("");
                            }}
                          >
                            <ArrowRightIcon /> {TICKET_STATUS_LABELS[next]}
                          </Button>
                        ))}
                      </div>
                    </>
                  ) : (
                    <p className="text-xs text-muted-foreground">
                      Terminal — no further transitions.
                    </p>
                  )}
                  {canTriage(role) &&
                    DISMISSABLE_STATUSES.includes(ticket.status) && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={busy}
                        onClick={() => onDismiss(ticket)}
                      >
                        <XIcon /> Dismiss
                      </Button>
                    )}
                </section>
              )}

              {ticket.status === "open" && actable && (
                <section className="space-y-1 rounded border border-dashed p-3">
                  <p className="text-xs font-medium">Start triage first</p>
                  <p className="text-[11px] text-muted-foreground">
                    An open ticket may only move to triage; planning the
                    activities unlocks from there.
                  </p>
                  <Button
                    size="sm"
                    className="mt-1"
                    disabled={busy}
                    onClick={() => onTransition(ticket.id, "triage")}
                  >
                    Move to triage
                  </Button>
                </section>
              )}

              {ticket.status === "triage" && canTriage(role) && (
                <TriagePanel
                  ticket={ticket}
                  busy={busy}
                  onTriage={(ticketKind, plans) =>
                    onTriage(ticket.id, ticketKind, plans)
                  }
                />
              )}

              <section className="space-y-2">
                <h3 className="text-xs font-medium">Conversation</h3>
                {comments.map((entry) => (
                  <div key={entry.id} className="rounded border p-2">
                    <p className="text-[11px] text-muted-foreground">
                      {actorName(entry.author_id)} ·{" "}
                      {format(new Date(entry.created_at), "MMM d, HH:mm")}
                    </p>
                    <p className="mt-0.5 text-xs whitespace-pre-wrap">
                      {entry.body}
                    </p>
                  </div>
                ))}
                {comments.length === 0 && (
                  <p className="text-xs text-muted-foreground">
                    No comments yet.
                  </p>
                )}
                {actable ? (
                  <div className="flex items-center gap-2">
                    <Input
                      value={comment}
                      onChange={(event) => setComment(event.target.value)}
                      placeholder="Add a comment"
                      aria-label="Add a comment"
                      className="h-8 text-xs"
                    />
                    <Button
                      size="icon-sm"
                      aria-label="Post comment"
                      disabled={busy || comment.trim().length === 0}
                      onClick={() => {
                        onComment(ticket.id, comment.trim());
                        setComment("");
                      }}
                    >
                      <SendIcon />
                    </Button>
                  </div>
                ) : (
                  <p className="text-[11px] text-muted-foreground">
                    Only the requester, the assignee, or manager/finance can
                    comment.
                  </p>
                )}
              </section>

              <section className="space-y-1">
                <h3 className="text-xs font-medium">History</h3>
                {(history.data ?? []).map((entry) => (
                  <p key={entry.id} className="text-[11px]">
                    <span className="text-muted-foreground">
                      {format(new Date(entry.created_at), "MMM d, HH:mm")}
                    </span>{" "}
                    {describeHistoryEntry(entry)}{" "}
                    <span className="text-muted-foreground">
                      {actorName(entry.actor_id)}
                      {entry.comment ? ` — ${entry.comment}` : ""}
                    </span>
                  </p>
                ))}
                {history.data?.length === 0 && (
                  <p className="text-xs text-muted-foreground">
                    Nothing recorded yet.
                  </p>
                )}
              </section>
            </div>
          </>
        )}
      </SheetContent>
    </Sheet>
  );
}
