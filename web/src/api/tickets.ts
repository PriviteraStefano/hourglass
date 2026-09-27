import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { toast } from "sonner";
import { api } from "@/lib/api.ts";
import type {
  CreateTicketRequest,
  Ticket,
  TicketComment,
  TicketDetail,
  TicketHistoryEntry,
  TicketKind,
  TicketStatus,
  TriageTicketRequest,
  TransitionTicketRequest,
} from "@/types";

/**
 * Ticket API — E-05 (raise internal demand) and M-08 (triage).
 *
 * All eight routes exist server-side (`cmd/server/main.go`); there is no DELETE
 * and no comment/history mutation (TICK-05, append-only). Server-side scoping:
 * `GET /tickets` is org-wide for every internal role and rejects `customer`
 * with 403 (G2-class — no per-user scoping yet).
 *
 * Every query key starts with `["tickets"]`, so one invalidation prefix
 * refreshes the list, the detail and the history together.
 */
const ticketsQueryOpts = (status?: TicketStatus, kind?: TicketKind) =>
  queryOptions({
    queryKey: ["tickets", "list", status ?? "all", kind ?? "all"] as const,
    queryFn: () => {
      const query = new URLSearchParams();
      if (status) query.set("status", status);
      if (kind) query.set("kind", kind);
      const suffix = query.toString();
      return api<Ticket[]>(`/tickets${suffix ? `?${suffix}` : ""}`);
    },
  });

const ticketDetailQueryOpts = (id: string) =>
  queryOptions({
    queryKey: ["tickets", "detail", id] as const,
    queryFn: () => api<TicketDetail>(`/tickets/${id}`),
    enabled: Boolean(id),
  });

const ticketHistoryQueryOpts = (id: string) =>
  queryOptions({
    queryKey: ["tickets", "history", id] as const,
    queryFn: () => api<TicketHistoryEntry[]>(`/tickets/${id}/history`),
    enabled: Boolean(id),
  });

const createTicketMutationOpts = mutationOptions({
  mutationFn: (data: CreateTicketRequest) =>
    api<Ticket>("/tickets", { method: "POST", body: JSON.stringify(data) }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["tickets"] });
    toast.success("Ticket raised");
  },
  onError: () => {
    toast.error("Failed to raise the ticket");
  },
});

const triageTicketMutationOpts = mutationOptions({
  mutationFn: ({ id, ...data }: TriageTicketRequest & { id: string }) =>
    api(`/tickets/${id}/triage`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["tickets"] });
    toast.success("Ticket planned");
  },
  onError: () => {
    toast.error("Triage failed");
  },
});

const transitionTicketMutationOpts = mutationOptions({
  mutationFn: ({ id, ...data }: TransitionTicketRequest & { id: string }) =>
    api<Ticket>(`/tickets/${id}/transition`, {
      method: "POST",
      body: JSON.stringify(data),
    }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["tickets"] });
  },
  onError: (error: Error) => {
    toast.error(error.message || "Transition failed");
  },
});

const dismissTicketMutationOpts = mutationOptions({
  mutationFn: (id: string) =>
    api<Ticket>(`/tickets/${id}/dismiss`, { method: "POST" }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["tickets"] });
    toast.success("Ticket dismissed");
  },
  onError: (error: Error) => {
    // The guard-style 409 carries the blocking reason ("linked activities have
    // logged hours"); surface it verbatim rather than a generic failure.
    toast.error(error.message || "Dismissal failed");
  },
});

const commentTicketMutationOpts = mutationOptions({
  mutationFn: ({ id, body }: { id: string; body: string }) =>
    api<TicketComment>(`/tickets/${id}/comments`, {
      method: "POST",
      body: JSON.stringify({ body }),
    }),
  onSuccess: (_, __, ___, { client }) => {
    client.invalidateQueries({ queryKey: ["tickets"] });
  },
  onError: () => {
    toast.error("Failed to post the comment");
  },
});

export const TicketsApis = {
  ticketsQueryOpts,
  ticketDetailQueryOpts,
  ticketHistoryQueryOpts,
  createTicketMutationOpts,
  triageTicketMutationOpts,
  transitionTicketMutationOpts,
  dismissTicketMutationOpts,
  commentTicketMutationOpts,
};
