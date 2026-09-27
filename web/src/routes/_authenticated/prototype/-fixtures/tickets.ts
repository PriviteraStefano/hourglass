import { queryOptions } from "@tanstack/react-query";
import { api } from "@/lib/api.ts";

/**
 * PROTOTYPE module for the **tickets** surface — E-05 (raise internal demand)
 * and M-08 (triage).
 *
 * Reads and writes are both real: `GET /tickets`, `GET /tickets/{id}`,
 * `POST /tickets/{id}/triage|transition|dismiss|comments`. The rules below
 * mirror the server so the UI cannot offer an action the API would reject:
 *   · the transition matrix is locked (`ticket.go:89`) — nothing outside it;
 *   · `→ dismissed` is Dismiss-only, so an owner cannot bypass the guard;
 *   · `→ resolved` requires every linked activity to be terminal;
 *   · dismissal is blocked while linked activities carry submitted/approved
 *     hours (`ticket_repository.go:744`);
 *   · triage/dismiss/update are manager|finance; transition/comment also admit
 *     the requester or the assignee.
 */

export type TicketKind = "question" | "bug" | "change" | "evolution";
export type TicketStatus =
  | "open"
  | "triage"
  | "planned"
  | "in_progress"
  | "resolved"
  | "closed"
  | "dismissed";

export interface Ticket {
  id: string;
  org_id: string;
  title: string;
  description?: string;
  kind: TicketKind;
  status: TicketStatus;
  requester_id: string;
  /** Absent (not null) when unassigned. */
  assignee_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface TicketComment {
  id: string;
  ticket_id: string;
  author_id: string;
  body: string;
  created_at: string;
}

export interface TicketDetail {
  ticket: Ticket;
  comments: TicketComment[];
}

export const KIND_LABEL: Record<TicketKind, string> = {
  question: "Question",
  bug: "Bug",
  change: "Change",
  evolution: "Evolution",
};

export const STATUS_LABEL: Record<TicketStatus, string> = {
  open: "Open",
  triage: "In triage",
  planned: "Planned",
  in_progress: "In progress",
  resolved: "Resolved",
  closed: "Closed",
  dismissed: "Dismissed",
};

export const STATUS_HINT: Record<TicketStatus, string> = {
  open: "Raised, nobody has looked at it yet",
  triage: "Being decided: plan it into activities, or dismiss it",
  planned: "Triage created the work it belongs to",
  in_progress: "Someone is working on it",
  resolved: "Work is done; every linked activity is terminal",
  closed: "Done and archived — terminal",
  dismissed: "Rejected with the logged hours snapshotted — terminal",
};

export const LIFECYCLE: TicketStatus[] = [
  "open",
  "triage",
  "planned",
  "in_progress",
  "resolved",
  "closed",
];

/** The locked matrix, minus `→ dismissed` (Dismiss-only). */
const MATRIX: Record<TicketStatus, TicketStatus[]> = {
  open: ["triage"],
  triage: ["planned"],
  planned: ["in_progress"],
  in_progress: ["resolved"],
  resolved: ["closed", "in_progress"],
  closed: [],
  dismissed: [],
};

export function nextStatuses(status: TicketStatus): TicketStatus[] {
  return MATRIX[status];
}

export function isTerminal(status: TicketStatus): boolean {
  return status === "closed" || status === "dismissed";
}

/** manager|finance, or the requester, or the assignee (`ticket.go:250`). */
export function canActOnTicket(
  role: string,
  userId: string,
  ticket: Ticket
): boolean {
  return (
    role === "manager" ||
    role === "finance" ||
    ticket.requester_id === userId ||
    ticket.assignee_id === userId
  );
}

export function canTriage(role: string): boolean {
  return role === "manager" || role === "finance";
}

/** Prototype-owned rows (seeded by `scripts/seed_prototype.sql`), so a stray
 *  click cannot mutate the tickets that predate this session. */
export function isPrototypeTicket(id: string): boolean {
  return id.startsWith("019df940");
}

export const ticketsQueryOpts = (status?: TicketStatus) =>
  queryOptions({
    queryKey: ["prototype", "tickets", status ?? "all"] as const,
    queryFn: () =>
      api<Ticket[]>(`/tickets${status ? `?status=${status}` : ""}`),
  });

export const ticketDetailQueryOpts = (id: string) =>
  queryOptions({
    queryKey: ["prototype", "ticket", id] as const,
    queryFn: () => api<TicketDetail>(`/tickets/${id}`),
    enabled: Boolean(id),
  });

export interface TriagePlan {
  name: string;
  kind: string;
  parent_id: string;
  contract_id: string;
  /** Required by the API; the handler rejects the whole request without it. */
  governance_model?: string;
}

export function triageTicket(
  id: string,
  kind: TicketKind,
  plans: TriagePlan[]
): Promise<unknown> {
  return api(`/tickets/${id}/triage`, {
    method: "POST",
    body: JSON.stringify({
      kind,
      // Empty ids must be omitted, not sent as "": the handler parses every
      // present id as a UUID and rejects the whole request otherwise.
      activities: plans.map((plan) => ({
        name: plan.name,
        kind: plan.kind,
        governance_model: plan.governance_model ?? "creator_controlled",
        ...(plan.parent_id ? { parent_id: plan.parent_id } : {}),
        ...(plan.contract_id ? { contract_id: plan.contract_id } : {}),
      })),
    }),
  });
}

export function transitionTicket(
  id: string,
  status: TicketStatus,
  note?: string
): Promise<Ticket> {
  return api<Ticket>(`/tickets/${id}/transition`, {
    method: "POST",
    body: JSON.stringify({ status, note: note ? note : undefined }),
  });
}

export function dismissTicket(id: string): Promise<Ticket> {
  return api<Ticket>(`/tickets/${id}/dismiss`, { method: "POST" });
}

export function commentTicket(id: string, body: string): Promise<TicketComment> {
  return api<TicketComment>(`/tickets/${id}/comments`, {
    method: "POST",
    body: JSON.stringify({ body }),
  });
}
