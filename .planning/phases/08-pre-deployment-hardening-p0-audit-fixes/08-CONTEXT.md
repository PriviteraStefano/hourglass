# Phase 8: Pre-Deployment Hardening (P0 audit fixes) - Context

**Gathered:** 2026-07-31
**Status:** Ready for planning
**Source:** Pre-Deployment Audit (2026-07-28) §6 — all findings locked; no open decisions

<domain>
## Phase Boundary

Close the six P0 findings from the 2026-07-28 Pre-Deployment Audit that gate the first deployment of Hourglass v0.1. Each P0 item maps to a standalone fix with its own verification; the phase ends when all six are merged, tested, and the audit's P0 table reads "Fixed" for every row.

This phase is deliberately sequenced BEFORE Phase 9 (Activity Ontology, ADR-P-007 big-bang): P-007's migration rewrites the same handlers/repos the P0 list touches — fixing first means testing the final shape once, not twice.

</domain>

<decisions>
## Implementation Decisions

All decisions are locked by the audit — it is the spec for this phase.

### P0-1 — Time Entry DB status constraint (Bug, Small)
- Add `pending_manager`, `pending_finance`, `rejected` to the time entry status CHECK constraint via a new migration. The approval workflow (two-stage, Phase 6) cannot persist these states until the constraint allows them.
- **D-01:** Migration-only fix — no code changes required; the service layer already uses these states. New sequential migration file per BE migration strategy.

### P0-2 — List view placeholders (UX, Medium)
- `/time-entries` and `/expenses` `TabsContent value="list"` currently render only a placeholder comment. Implement real list views: filterable, sortable tables of the user's entries with status badges and click-through to detail.
- **D-02:** Reuse existing API hooks + query keys (ADR-FE-003). Flat list, paginated client-side per ADR-FE-018 conventions. List view complements the existing calendar view — it does not replace it.

### P0-3 — `/customers` index route missing (UX, Small)
- Only `/customers/$id` exists; no index route maps to the (already-built) customers list page. Add the route definition so the feature is reachable from navigation.
- **D-03:** Follow directory-per-route convention (ADR-FE-017) — `customers/index.tsx` alongside the existing `$id.tsx`.

### P0-4 — Error boundaries on routes (UX, Small)
- No `errorComponent` on any route; a failed loader crashes to a blank screen. Add error boundaries per ADR-FE-014.
- **D-04:** Shared error component in the layout route (`_authenticated.tsx`) + per-route overrides only where recovery differs. Toast-based mutation errors already work — this covers loader/query failures only.

### P0-5 — Refresh token rotation (Security, Medium)
- Old refresh token is not revoked on use; a stolen token grants access up to the 7-day TTL. Implement rotation: revoke on use, issue new token on every refresh.
- **D-05:** Standard rotation with reuse detection — if an already-rotated token is presented, revoke the whole token family. Stays within the existing JWT+HttpOnly-cookie mechanism (ADR-FE-013); no mechanism change.

### P0-6 — Password reset code exposure (Security, Small)
- Reset code is returned in the API response body; 3-digit code is trivially brute-forceable. Remove the code from the response; delivery via email only.
- **D-06:** Response returns a generic success message regardless of account existence (no user enumeration). Code entropy increase (beyond 3 digits) folded in since the endpoint is being touched anyway.

### Claude's Discretion
- List-view table column selection within the existing API payloads
- Error boundary visual design (must show error + recovery action, per ADR-FE-014)
- Exact email-delivery mechanism for reset codes (log-to-stdout in dev is acceptable; production mailer is out of scope — see below)

### Out of scope
- P1 items (skeletons, register OrgID validation, JSON decode handling, cookie name unification, audit log context) — separate phase
- P2 items (optimistic updates, component tests, dual-model cleanup, dashboard, i18n)
- Production SMTP/email provider setup (reset codes logged in dev; a mailer is a deployment-config concern, not this phase)
- Activity ontology rename (Phase 9, ADR-P-007) — do not pre-rename anything in this phase

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Audit (the spec for this phase)
- `hourglass-vault/research/2026-07-28 — Pre-Deployment Audit — Hourglass v0.1.md` — §4 Bug Catalogue, §6 Priority Matrix (P0-1…P0-6), §3 ADR compliance (FE-014, FE-016)

### Decisions
- `hourglass-vault/decisions/backend/_index.md` — backend ADR index (error handling, repo/service/handler patterns, migrations)
- `hourglass-vault/00-Index.md` — vault hub; open resolutions

### Code anchors (verify before planning)
- `migrations/` — sequential numbered migrations; latest number determines P0-1's file
- `internal/adapters/primary/http/auth.go` + `internal/core/services/auth/` — refresh + password reset endpoints
- `web/src/routes/_authenticated/time-entries/index.tsx`, `expenses/index.tsx` — list-view placeholders
- `web/src/routes/_authenticated.tsx` — layout route (error boundary home)
- `web/src/routes/_authenticated/customers/` — `$id.tsx` exists; index route to add
- `.planning/REQUIREMENTS.md` + `.planning/ROADMAP.md` — requirements IDs and roadmap context

</canonical_refs>

<specifics>
## Specific Ideas

- P0-1 migration must be additive-only (widen CHECK constraint) — no data rewrite, zero risk to existing rows
- P0-2 list views should share a generic entries-table component between time entries and expenses where the shape overlaps (status badge, date, project/activity name, description, actions)
- P0-5 rotation should include a token-family table or family-id claim — pick whichever fits the existing refresh-token persistence with the least schema churn
- All six fixes land behind the existing test suites (Go service/handler tests + Vitest + Playwright E2E stay green); each fix also adds at least one regression test proving the bug is dead

</specifics>

<deferred>
## Deferred Ideas

- P1 batch (skeleton loading states, B2 register OrgID validation, B3 JSON decode handling, T4/T6/T7 tech debt) → candidate Phase 12
- P2 batch (optimistic updates, component-level tests, dual model layer consolidation T1, SurrealDB remnant T2, dashboard, i18n) → post-v0.1 backlog
- Backend ADR creation (audit §7, ADR-BE-001…012) → partially covered by the 14 backend ADRs now in `decisions/backend/`; remainder post-v0.1

</deferred>

---

*Phase: 08-pre-deployment-hardening-p0-audit-fixes*
*Context gathered: 2026-07-31 via audit-ingest express path (manual, bypassing discuss-phase per GSD executory-phase rule)*
