import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { format, parseISO } from "date-fns";
import { ArrowRightIcon, LoaderIcon, SendIcon, XIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table.tsx";
import {
  KIND_LABEL,
  STATUS_HINT,
  STATUS_LABEL,
  canTriage,
  isPrototypeTicket,
  nextStatuses,
  ticketDetailQueryOpts,
  type Ticket,
  type TicketStatus,
} from "@/routes/_authenticated/prototype/-fixtures/tickets.ts";
import { cn } from "@/lib/utils.ts";

/**
 * PROTOTYPE VARIANT C — "ticket table + conversation".
 *
 * Information hierarchy: the conversation is the object. The table answers
 * "which tickets exist"; the detail pane shows the description, the comment
 * history, the composer and the legal next steps, in that order.
 *
 * Primary affordance: read the thread, then move or answer it.
 */

export interface TicketsVariantCProps {
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

export function TicketsVariantC({
  tickets,
  names,
  role,
  userId,
  onTransition,
  onDismiss,
  onComment,
  busy,
  error,
}: TicketsVariantCProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState<TicketStatus | "all">("all");
  const [body, setBody] = useState("");
  const [note, setNote] = useState("");

  const rows = tickets
    .filter((t) => statusFilter === "all" || t.status === statusFilter)
    .sort((a, b) => b.updated_at.localeCompare(a.updated_at));

  const selected = tickets.find((t) => t.id === selectedId) ?? rows[0] ?? null;
  const detail = useQuery(ticketDetailQueryOpts(selected?.id ?? ""));

  return (
    <div className="flex h-full flex-col pb-16">
      <div className="flex items-start justify-between gap-4 border-b px-4 py-3">
        <div>
          <h1 className="text-xl font-semibold">Tickets</h1>
          <p className="text-xs text-muted-foreground">
            {tickets.length} tickets · most recently updated first
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-1">
          {(["all", ...Object.keys(STATUS_LABEL)] as (TicketStatus | "all")[]).map(
            (status) => (
              <button
                key={status}
                type="button"
                onClick={() => setStatusFilter(status)}
                className={cn(
                  "rounded-full border px-2 py-0.5 text-[11px]",
                  statusFilter === status
                    ? "bg-muted font-medium text-foreground"
                    : "text-muted-foreground hover:text-foreground"
                )}
              >
                {status === "all" ? "Every status" : STATUS_LABEL[status]}
              </button>
            )
          )}
        </div>
      </div>

      {error && <p className="border-b bg-muted/40 px-4 py-2 text-xs">{error}</p>}

      <div className="flex min-h-0 flex-1 flex-col overflow-auto 2xl:flex-row">
        <div className="w-full min-w-0 p-4 2xl:w-0 2xl:flex-1">
          <div className="rounded-lg border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Ticket</TableHead>
                  <TableHead>Kind</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Requester</TableHead>
                  <TableHead>Assignee</TableHead>
                  <TableHead>Updated</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((ticket) => (
                  <TableRow
                    key={ticket.id}
                    onClick={() => {
                      setSelectedId(ticket.id);
                      setBody("");
                    }}
                    className={cn(
                      "cursor-pointer",
                      selected?.id === ticket.id && "bg-muted"
                    )}
                  >
                    <TableCell className="text-xs">
                      <span className="flex items-center gap-2">
                        <span className="truncate">{ticket.title}</span>
                        {isPrototypeTicket(ticket.id) && (
                          <span className="shrink-0 text-[10px] text-muted-foreground">
                            fixture
                          </span>
                        )}
                      </span>
                    </TableCell>
                    <TableCell className="text-xs">
                      {KIND_LABEL[ticket.kind]}
                    </TableCell>
                    <TableCell className="text-xs">
                      <Badge variant="secondary" className="font-normal">
                        {STATUS_LABEL[ticket.status]}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs">
                      {names[ticket.requester_id] ?? "unknown"}
                    </TableCell>
                    <TableCell className="text-xs">
                      {ticket.assignee_id
                        ? (names[ticket.assignee_id] ?? "someone")
                        : "—"}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {format(parseISO(ticket.updated_at), "d MMM HH:mm")}
                    </TableCell>
                  </TableRow>
                ))}
                {rows.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="py-10 text-center">
                      <p className="text-sm font-medium">No tickets match</p>
                      <p className="text-xs text-muted-foreground">
                        Try “Every status”.
                      </p>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>

        <aside className="w-full shrink-0 border-t p-4 2xl:w-96 2xl:overflow-auto 2xl:border-t-0 2xl:border-l">
          {selected ? (
            <div className="space-y-3">
              <div>
                <h2 className="text-sm font-medium">{selected.title}</h2>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                  <Badge variant="outline" className="font-normal">
                    {KIND_LABEL[selected.kind]}
                  </Badge>
                  <Badge variant="secondary" className="font-normal">
                    {STATUS_LABEL[selected.status]}
                  </Badge>
                  <span>
                    {names[selected.requester_id] ?? "unknown"}
                    {selected.assignee_id
                      ? ` → ${names[selected.assignee_id] ?? "someone"}`
                      : " → unassigned"}
                  </span>
                </div>
                <p className="mt-1 text-[11px] text-muted-foreground">
                  {STATUS_HINT[selected.status]}
                </p>
                {selected.description && (
                  <p className="mt-2 text-xs">{selected.description}</p>
                )}
              </div>

              <div className="flex flex-wrap items-center gap-1">
                {nextStatuses(selected.status).map((next) => (
                  <Button
                    key={next}
                    size="sm"
                    variant="outline"
                    disabled={busy}
                    onClick={() => onTransition(selected.id, next)}
                  >
                    <ArrowRightIcon /> {STATUS_LABEL[next]}
                  </Button>
                ))}
                {canTriage(role) &&
                  (selected.status === "open" || selected.status === "triage") && (
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={busy}
                      onClick={() => onDismiss(selected.id)}
                    >
                      <XIcon /> Dismiss
                    </Button>
                  )}
              </div>

              <div className="rounded border">
                <p className="border-b px-2 py-1 text-[11px] font-medium">
                  Conversation
                </p>
                {detail.isPending && (
                  <p className="px-2 py-3">
                    <LoaderIcon className="size-4 animate-spin text-muted-foreground" />
                  </p>
                )}
                <ul className="divide-y">
                  {(detail.data?.comments ?? []).map((comment) => (
                    <li key={comment.id} className="px-2 py-1.5">
                      <p className="text-[11px]">
                        <span className="font-medium">
                          {names[comment.author_id] ?? "someone"}
                        </span>{" "}
                        <span className="text-muted-foreground">
                          {format(parseISO(comment.created_at), "d MMM HH:mm")}
                        </span>
                      </p>
                      <p className="text-xs">{comment.body}</p>
                    </li>
                  ))}
                  {detail.data && detail.data.comments.length === 0 && (
                    <li className="px-2 py-3 text-[11px] text-muted-foreground">
                      No comments yet.
                    </li>
                  )}
                </ul>
                <div className="flex items-center gap-1 border-t p-2">
                  <Input
                    value={body}
                    onChange={(e) => setBody(e.target.value)}
                    placeholder="Comment"
                    className="h-8 text-xs"
                  />
                  <Button
                    size="icon-sm"
                    aria-label="Send comment"
                    disabled={busy || body.trim().length === 0}
                    onClick={() => {
                      onComment(selected.id, body.trim());
                      setBody("");
                    }}
                  >
                    <SendIcon />
                  </Button>
                </div>
              </div>

              <div className="space-y-1.5">
                <Input
                  value={note}
                  onChange={(e) => setNote(e.target.value)}
                  placeholder="Note for the next transition (optional)"
                  className="h-8 text-xs"
                />
                <p className="text-[10px] text-muted-foreground">
                  {selected.assignee_id === userId
                    ? "You are the assignee — the matrix buttons act as you."
                    : "Transitions run as you: manager, finance, requester or assignee."}
                </p>
              </div>
            </div>
          ) : (
            <p className="mt-6 text-center text-xs text-muted-foreground">
              Select a ticket to read its thread.
            </p>
          )}
        </aside>
      </div>
    </div>
  );
}
