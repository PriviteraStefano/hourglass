import { z } from "zod";
import { TICKET_KINDS, TICKET_STATUSES } from "@/types";

/**
 * Shared URL-search schemas for the list views (P0-2).
 * Both /time-entries and /expenses drive their list filters through
 * validateSearch per ADR-FE-017 so filter state is URL-shareable.
 */

export const entryStatusSchema = z.enum([
  "draft",
  "submitted",
  "pending_manager",
  "pending_finance",
  "approved",
  "rejected",
]);

export type EntryStatusSearch = z.infer<typeof entryStatusSchema>;

// Accept ?listStatuses=draft (single) or ?listStatuses=a&listStatuses=b
// (repeated), plus TanStack Router's JSON-array serialization.
export const listStatusesSchema = z
  .union([z.array(entryStatusSchema), entryStatusSchema])
  .optional()
  .transform((v) => (Array.isArray(v) ? v : v ? [v] : undefined));

export const ticketKindSchema = z.enum(TICKET_KINDS);

export const ticketStatusSchema = z.enum(TICKET_STATUSES);

// Board lanes, same single-or-repeated shape as listStatusesSchema. Absent
// means "the four live lanes" — the page applies that default, so the URL stays
// short until the reader changes the lane set.
export const ticketLanesSchema = z
  .union([z.array(ticketStatusSchema), ticketStatusSchema])
  .optional()
  .transform((v) => (Array.isArray(v) ? v : v ? [v] : undefined));
