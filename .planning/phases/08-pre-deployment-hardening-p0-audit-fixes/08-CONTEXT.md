# Phase 8: Pre-Deployment Hardening (P0 audit fixes) - Context

**Gathered:** 2026-07-31 (revised 2026-07-31 — code verification corrections)
**Status:** Ready for planning
**Source:** Pre-Deployment Audit (2026-07-28) §6 + **Corrections (2026-07-31)** — verified against code

<domain>
## Phase Boundary

Close the **remaining** P0 findings from the 2026-07-28 Pre-Deployment Audit that gate the first deployment of Hourglass v0.1. Code verification (2026-07-31) closed two of the six audit items as already-fixed and rescoped a third; the phase now covers **4 items**: P0-2 (list views), P0-3 (`/customers` route), P0-4 (error boundaries), P0-5-lite (refresh-token reuse detection), plus folded-in S3 input caps. The phase ends when all four are merged, tested, and the audit's P0 table reads "Fixed" for every row.

This phase is deliberately sequenced BEFORE Phase 9 (Activity Ontology, ADR-P-007 big-bang): P-007's migration rewrites the same handlers/repos the P0 list touches — fixing first means testing the final shape once, not twice.

</domain>

<decisions>
## Implementation Decisions

All decisions are locked by the audit + its 2026-07-31 Corrections — they are the spec for this phase.

### ~~P0-1 — Time Entry DB status constraint~~ ✅ CLOSED (pre-audit)
- Already fixed by `migrations/004_time_entries_status_check.up.sql` (all six states + matching down). The audit quoted the `000` baseline, not the corrective migration. **No work.**

### P0-2 — List view placeholders (UX, Medium)
- `/time-entries` and `/expenses` `TabsContent value="list"` currently render only a placeholder comment. Implement real list views: filterable, sortable tables of the user's entries with status badges and click-through to detail.
- **D-02:** Reuse existing API hooks + query keys (ADR-FE-003). Flat list, paginated client-side per ADR-FE-018 conventions. List view complements the existing calendar view — it does not replace it.

### P0-3 — `/customers` index route missing (UX, Small)
- Only `/customers/$id` exists; no index route maps to the (already-built) customers list page. Add the route definition so the feature is reachable from navigation.
- **D-03:** Follow directory-per-route convention (ADR-FE-017) — `customers/index.tsx` alongside the existing `$id.tsx`.

### P0-4 — Error boundaries on routes (UX, Small)
- No `errorComponent` on any route; a failed loader crashes to a blank screen. Add error boundaries per ADR-FE-014.
- **D-04:** Shared error component in the layout route (`_authenticated.tsx`) + per-route overrides only where recovery differs. Toast-based mutation errors already work — this covers loader/query failures only.

### P0-5 — Refresh-token reuse detection (Security, Small — rescoped)
- Rotation **already exists** (`internal/core/services/auth/auth.go:349–404`: `RevokeByHash` old + issue new). The real gap is the detection layer: no `family_id`/`rotated_at`/`revoked_at` in token persistence, so replay of a rotated token is a silent generic 401 with no family revocation; rotate also runs as 3 untransacted statements (crash window + multi-tab race).
- **D-05:** Add reuse detection on top of the existing rotation — `family_id` + `rotated_at` tombstone model, replay → `ErrTokenReuse` + revoke the whole token family, rotation made atomic (single tx). Stays within the existing JWT+HttpOnly-cookie mechanism (ADR-FE-013); no mechanism change.

### ~~P0-6 — Password reset code exposure~~ ✅ CLOSED (pre-audit)
- Handler returns only `message` + `expires_at` (code discarded); `generateResetCode()` is 8 chars from a 62-char charset (crypto/rand). Regression tests already assert the absence (`password_reset_test.go:137-139`, `auth_integration_test.go:302-304` per D-16). **No work.**

### Claude's Discretion
- List-view table column selection within the existing API payloads
- Error boundary visual design (must show error + recovery action, per ADR-FE-014)
- Concurrent-refresh race semantics (legitimate tab race vs. attacker replay) — document the chosen behavior in tests

### Out of scope
- P1 items (skeletons, register OrgID validation, JSON decode handling, cookie name unification, audit log context) — separate phase
- P2 items (optimistic updates, component tests, dual-model cleanup, dashboard, i18n)
- Password-reset email delivery + enumeration-safe response (minor residuals of the verified P0-6 fix — track as P1)
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
- `migrations/` — sequential numbered migrations; 010 is the next free number (004/005 already widened the status CHECKs)
- `internal/core/services/auth/auth.go` (Refresh L349–404 — existing rotation) + `internal/core/ports/refresh_token_repo.go` + `internal/adapters/secondary/postgres/refresh_token_repo.go` — reuse-detection work sites
- `internal/adapters/primary/http/password_reset.go` + `password_reset_test.go` — verified-closed P0-6 evidence
- `web/src/routes/_authenticated/time-entries/index.tsx`, `expenses/index.tsx` — list-view placeholders
- `web/src/routes/_authenticated.tsx` — layout route (error boundary home)
- `web/src/routes/_authenticated/customers/` — `$id.tsx` exists; index route to add
- `.planning/REQUIREMENTS.md` + `.planning/ROADMAP.md` — requirements IDs and roadmap context

</canonical_refs>

<specifics>
## Specific Ideas

- P0-5 migration must backfill `family_id` for existing token rows (one family per existing row) — additive, no data loss; move `RevokeByHash` to a tombstone (`revoked_at`) model so replayed hashes stay detectable
- P0-2 list views should share a generic entries-table component between time entries and expenses where the shape overlaps (status badge, date, project/activity name, description, actions)
- All fixes land behind the existing test suites (Go service/handler tests + Vitest + Playwright E2E stay green); each fix also adds at least one regression test proving the gap is dead

</specifics>

<deferred>
## Deferred Ideas

- P1 batch (skeleton loading states, B2 register OrgID validation, B3 JSON decode handling, T4/T6/T7 tech debt) → candidate Phase 12
- P2 batch (optimistic updates, component-level tests, dual model layer consolidation T1, SurrealDB remnant T2, dashboard, i18n) → post-v0.1 backlog
- Password-reset email delivery + enumeration-safe response (minor residuals of the verified P0-6 fix) → fold into P1 batch
- Backend ADR creation (audit §7, ADR-BE-001…012) → partially covered by the 14 backend ADRs now in `decisions/backend/`; remainder post-v0.1

</deferred>

---

*Phase: 08-pre-deployment-hardening-p0-audit-fixes*
*Context gathered: 2026-07-31 via audit-ingest express path (manual, bypassing discuss-phase per GSD executory-phase rule)*
