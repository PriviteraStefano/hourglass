import { useState } from "react";
import { CheckIcon, PlusIcon, TrashIcon, XIcon } from "lucide-react";
import { Badge } from "@/components/ui/badge.tsx";
import { Button } from "@/components/ui/button.tsx";
import { Input } from "@/components/ui/input.tsx";
import {
  NativeSelect,
  NativeSelectOption,
} from "@/components/ui/native-select.tsx";
import {
  KIND_LABEL,
  STATUS_LABEL,
  type Ticket,
  type TicketKind,
  type TicketStatus,
  type TriagePlan,
} from "@/routes/_authenticated/prototype/-fixtures/tickets.ts";
import { cn } from "@/lib/utils.ts";

/**
 * PROTOTYPE VARIANT A — "triage inbox".
 *
 * Information hierarchy: the decision is the unit. Open tickets on the left
 * (nobody has looked), in-triage tickets on the right (a decision is owed);
 * picking one opens the decision panel where triage plans the activities it
 * creates, or dismissal runs against the logged-hours guard.
 *
 * Primary affordance: work the decision queue.
 */

export interface TicketsVariantAProps {
  tickets: Ticket[];
  names: Record<string, string>;
  activities: { id: string; name: string }[];
  onTriage: (id: string, kind: TicketKind, plans: TriagePlan[]) => void;
  onTransition: (id: string, status: TicketStatus) => void;
  onDismiss: (id: string) => void;
  onComment: (id: string, body: string) => void;
  busy: boolean;
  error: string | null;
}

export function TicketsVariantA({
  tickets,
  names,
  activities,
  onTriage,
  onTransition,
  onDismiss,
  onComment,
  busy,
  error,
}: TicketsVariantAProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [kind, setKind] = useState<TicketKind>("bug");
  const [plans, setPlans] = useState<TriagePlan[]>([
    { name: "", kind: "phase", parent_id: "", contract_id: "" },
  ]);
  const [note, setNote] = useState("");

  const open = tickets.filter((t) => t.status === "open");
  const triage = tickets.filter((t) => t.status === "triage");
  const selected = tickets.find((t) => t.id === selectedId) ?? null;

  const column = (title: string, rows: Ticket[], hint: string) => (
    <section className="min-w-0 flex-1 rounded-lg border">
      <header className="flex items-center gap-2 border-b px-3 py-2">
        <h2 className="text-sm font-medium">{title}</h2>
        <span className="text-[11px] text-muted-foreground">{rows.length}</span>
        <span className="ml-auto text-[10px] text-muted-foreground">{hint}</span>
      </header>
      <ul className="divide-y">
        {rows.map((ticket) => (
          <li key={ticket.id}>
            <button
              type="button"
              onClick={() => {
                setSelectedId(ticket.id);
                setKind(ticket.kind);
                setPlans([{ name: ticket.title, kind: "phase", parent_id: "", contract_id: "" }]);
                setNote("");
              }}
              className={cn(
                "flex w-full flex-col gap-0.5 px-3 py-2 text-left",
                selectedId === ticket.id ? "bg-muted" : "hover:bg-muted/50"
              )}
            >
              <span className="flex items-center gap-2">
                <Badge variant="outline" className="shrink-0 font-normal">
                  {KIND_LABEL[ticket.kind]}
                </Badge>
                <span className="truncate text-xs">{ticket.title}</span>
              </span>
              <span className="text-[10px] text-muted-foreground">
                {names[ticket.requester_id] ?? "unknown"} ·{" "}
                {ticket.assignee_id
                  ? `assigned to ${names[ticket.assignee_id] ?? "someone"}`
                  : "unassigned"}
              </span>
            </button>
          </li>
        ))}
        {rows.length === 0 && (
          <li className="px-3 py-6 text-center text-xs text-muted-foreground">
            Nothing here.
          </li>
        )}
      </ul>
    </section>
  );

  return (
    <div className="flex h-full flex-col pb-16">
      <div className="flex items-start justify-between gap-4 border-b px-4 py-3">
        <div>
          <h1 className="text-xl font-semibold">Tickets</h1>
          <p className="text-xs text-muted-foreground">
            {open.length} waiting for a first look · {triage.length} awaiting a
            decision
          </p>
        </div>
        <Badge variant="outline" className="font-normal">
          triage and dismissal are manager / finance
        </Badge>
      </div>

      {error && (
        <p className="border-b bg-muted/40 px-4 py-2 text-xs">{error}</p>
      )}

      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4 lg:flex-row">
        {column("Open", open, "nobody has triaged these yet")}
        {column("In triage", triage, "a decision is owed")}
      </div>

      {selected && (
        <div className="border-t bg-muted/20 p-3">
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-sm font-medium">{selected.title}</span>
            <Badge variant="secondary" className="font-normal">
              {STATUS_LABEL[selected.status]}
            </Badge>
            <span className="text-[11px] text-muted-foreground">
              {selected.description}
            </span>
          </div>

          <div className="mt-2 grid gap-3 lg:grid-cols-[1fr_20rem]">
            <div className="space-y-2">
              <p className="text-[11px] font-medium">
                Triage into planned work — each row becomes an activity linked to
                this ticket
              </p>
              {plans.map((plan, index) => (
                <div key={index} className="flex flex-wrap items-center gap-1">
                  <Input
                    value={plan.name}
                    onChange={(e) =>
                      setPlans((prev) =>
                        prev.map((p, i) =>
                          i === index ? { ...p, name: e.target.value } : p
                        )
                      )
                    }
                    placeholder="Activity name"
                    className="h-8 w-64 text-xs"
                  />
                  <NativeSelect
                    value={plan.parent_id}
                    onChange={(e) =>
                      setPlans((prev) =>
                        prev.map((p, i) =>
                          i === index ? { ...p, parent_id: e.target.value } : p
                        )
                      )
                    }
                    className="h-8 w-56 text-xs"
                  >
                    <NativeSelectOption value="">No parent</NativeSelectOption>
                    {activities.map((a) => (
                      <NativeSelectOption key={a.id} value={a.id}>
                        {a.name}
                      </NativeSelectOption>
                    ))}
                  </NativeSelect>
                  {plans.length > 1 && (
                    <Button
                      size="icon-sm"
                      variant="ghost"
                      aria-label="Remove plan row"
                      onClick={() =>
                        setPlans((prev) => prev.filter((_, i) => i !== index))
                      }
                    >
                      <TrashIcon />
                    </Button>
                  )}
                </div>
              ))}
              <Button
                size="sm"
                variant="ghost"
                onClick={() =>
                  setPlans((prev) => [
                    ...prev,
                    { name: "", kind: "phase", parent_id: "", contract_id: "" },
                  ])
                }
              >
                <PlusIcon /> Another activity
              </Button>
            </div>

            <div className="space-y-2">
              {selected.status === "open" ? (
                <div className="space-y-1.5 rounded border border-dashed p-2">
                  <p className="text-[11px] font-medium">Start triage first</p>
                  <p className="text-[10px] text-muted-foreground">
                    An open ticket may only move to `triage` — planning and the
                    activity rows unlock from there. Dismissal is allowed from
                    either state.
                  </p>
                  <Button
                    size="sm"
                    disabled={busy}
                    onClick={() => onTransition(selected.id, "triage")}
                  >
                    <CheckIcon /> Move to triage
                  </Button>
                </div>
              ) : (
                <>
                  <label className="flex flex-col gap-1 text-[11px]">
                    Ticket kind after triage
                    <NativeSelect
                      value={kind}
                      onChange={(e) => setKind(e.target.value as TicketKind)}
                      className="h-8 text-xs"
                    >
                      {(Object.keys(KIND_LABEL) as TicketKind[]).map((k) => (
                        <NativeSelectOption key={k} value={k}>
                          {KIND_LABEL[k]}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </label>
                  <Input
                    value={note}
                    onChange={(e) => setNote(e.target.value)}
                    placeholder="Note for the history (optional)"
                    className="h-8 text-xs"
                  />
                  <div className="flex items-center gap-2">
                    <Button
                      size="sm"
                      disabled={busy || plans.every((p) => p.name.trim() === "")}
                      onClick={() => onTriage(selected.id, kind, plans)}
                    >
                      <CheckIcon /> Plan it
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      disabled={busy}
                      onClick={() => onDismiss(selected.id)}
                    >
                      <XIcon /> Dismiss
                    </Button>
                  </div>
                </>
              )}
              {selected.status === "open" && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={busy}
                  onClick={() => onDismiss(selected.id)}
                >
                  <XIcon /> Dismiss
                </Button>
              )}
              <Button
                size="sm"
                variant="ghost"
                disabled={busy || note.trim().length === 0}
                onClick={() => {
                  onComment(selected.id, note.trim());
                  setNote("");
                }}
              >
                Comment without deciding
              </Button>
              <p className="text-[10px] text-muted-foreground">
                Dismissal is refused while an activity linked to the ticket
                carries submitted or approved hours — that is why the button
                reports an error instead of succeeding.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
