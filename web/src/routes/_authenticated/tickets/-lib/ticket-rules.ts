import type { StatusTone } from "@/components/shared/status-pill.tsx";
import type { Role, Ticket, TicketStatus } from "@/types";

/**
 * Ticket presentation rules — the client-side mirror of the locked server
 * matrix (`internal/core/domain/ticket/ticket.go`, ADR-BE-016) and the role
 * gates (`ticket.go:250`).
 *
 * The server stays authoritative: this module exists so the UI cannot *offer*
 * a move the API would reject. `dismissed` is absent from the matrix on
 * purpose — dismissal is its own route with its own guard.
 */
export const TICKET_TRANSITIONS: Record<TicketStatus, TicketStatus[]> = {
  open: ["triage"],
  triage: ["planned"],
  planned: ["in_progress"],
  in_progress: ["resolved"],
  resolved: ["closed", "in_progress"],
  closed: [],
  dismissed: [],
};

/** Dismissal is allowed from these states (ADR-BE-016 superset of A7). */
export const DISMISSABLE_STATUSES: TicketStatus[] = ["open", "triage"];

/** Status → semantic status role. Never interaction chrome (LANGUAGE.md). */
export const TICKET_STATUS_TONES: Record<TicketStatus, StatusTone> = {
  open: "info",
  triage: "warning",
  planned: "neutral",
  in_progress: "info",
  resolved: "success",
  closed: "neutral",
  dismissed: "danger",
};

/** Triage and dismissal are manager|finance (D-11). */
export function canTriage(role: Role): boolean {
  return role === "manager" || role === "finance";
}

/**
 * Transitions and comments admit manager|finance, the requester, or the
 * assignee (D-15). Dismissal is narrower — see `canTriage`.
 */
export function canActOnTicket(
  role: Role,
  userId: string,
  ticket: Ticket
): boolean {
  return (
    canTriage(role) ||
    ticket.requester_id === userId ||
    ticket.assignee_id === userId
  );
}
