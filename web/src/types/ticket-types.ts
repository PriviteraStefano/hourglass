/**
 * Ticket domain types — mirrors `internal/core/domain/ticket` (ADR-P-003 rev).
 *
 * A ticket is an internal-only demand record. `ticket → activity → entries`
 * keeps a single-FK capture path, and there is no customer-facing surface
 * (D-E). The lifecycle matrix is locked server-side (ADR-BE-016); the UI may
 * only offer moves that matrix allows.
 */

export const TICKET_KINDS = ["question", "bug", "change", "evolution"] as const;

export type TicketKind = (typeof TICKET_KINDS)[number];

/** Lifecycle order, terminal states last. */
export const TICKET_STATUSES = [
  "open",
  "triage",
  "planned",
  "in_progress",
  "resolved",
  "closed",
  "dismissed",
] as const;

export type TicketStatus = (typeof TICKET_STATUSES)[number];

export interface Ticket {
  id: string;
  org_id: string;
  title: string;
  description: string;
  kind: TicketKind;
  status: TicketStatus;
  requester_id: string;
  /** Absent (not null) when unassigned. */
  assignee_id?: string | null;
  /** Hours logged before dismissal (TICK-04); present only when dismissed. */
  dismissed_hours?: number;
  /** Derived on read: rendered as "dismissed with {N} h logged". */
  dismissed_note?: string;
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

/** GET /tickets/{id} — the ticket plus its comment thread. */
export interface TicketDetail {
  ticket: Ticket;
  comments: TicketComment[];
}

/** GET /tickets/{id}/history — the append-only audit stream (TICK-05). */
export interface TicketHistoryEntry {
  id: string;
  entity_type: string;
  entity_id: string;
  action: string;
  /** Absent for system-initiated events. */
  actor_id?: string;
  comment?: string;
  payload?: Record<string, unknown>;
  created_at: string;
}

export interface CreateTicketRequest {
  title: string;
  description: string;
  kind: TicketKind;
  assignee_id?: string;
}

/** One row of a triage plan; each becomes an activity linked to the ticket. */
export interface TriageActivityPlan {
  name: string;
  kind: string;
  parent_id?: string;
  contract_id?: string;
  governance_model: "creator_controlled" | "unanimous" | "majority";
  description?: string;
  billable?: boolean;
  is_shared?: boolean;
  budget_amount?: number;
}

export interface TriageTicketRequest {
  /** Optional correction of the ticket kind while triaging. */
  kind?: TicketKind;
  activities: TriageActivityPlan[];
}

export interface TransitionTicketRequest {
  status: TicketStatus;
  note?: string;
}

export const TICKET_KIND_LABELS: Record<TicketKind, string> = {
  question: "Question",
  bug: "Bug",
  change: "Change",
  evolution: "Evolution",
};

export const TICKET_STATUS_LABELS: Record<TicketStatus, string> = {
  open: "Open",
  triage: "In triage",
  planned: "Planned",
  in_progress: "In progress",
  resolved: "Resolved",
  closed: "Closed",
  dismissed: "Dismissed",
};

/** What each status means to the person reading the lane header. */
export const TICKET_STATUS_HINTS: Record<TicketStatus, string> = {
  open: "Raised, nobody has looked at it yet",
  triage: "Being decided: plan it into activities, or dismiss it",
  planned: "Triage created the work it belongs to",
  in_progress: "Someone is working on it",
  resolved: "Work is done; every linked activity is terminal",
  closed: "Done and archived — terminal",
  dismissed: "Rejected with the logged hours snapshotted — terminal",
};

/** The lanes the board shows by default. */
export const LIVE_TICKET_STATUSES: TicketStatus[] = [
  "open",
  "triage",
  "planned",
  "in_progress",
];

/** Terminal states — behind the status filter, never lanes by default. */
export const TERMINAL_TICKET_STATUSES: TicketStatus[] = [
  "resolved",
  "closed",
  "dismissed",
];
