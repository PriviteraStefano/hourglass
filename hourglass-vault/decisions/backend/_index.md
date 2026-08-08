# Backend Decisions — Index

---
tags: ["adr", "backend", "index"]

---

# Backend ADRs (Project-Specific)

These ADRs record **how we build in this repo** — Go/backend decisions **specific to Hourglass**.

**Relationship to the knowledge vault:** The global vault (`knowledge/adr/…`) is the main source for technical decisions *globally*. A backend ADR here exists **only** where Hourglass deviates, extends, or decides something the global ADRs don't cover. If a topic is already decided globally, **link it — do not restate it.**

> The global vault currently holds frontend ADRs (ADR-FE-*) and Effect-learning ADRs (ADR-EFF-*), but **no Go/backend set**. Until one exists, this folder is the authoritative backend record. Entries marked `(promote)` are candidates for a future global Go ADR set — at promotion they're *moved* there and this folder links them.

## Format

`ADR-BE-NNN — Title.md` · tags: `adr`, `backend`, plus topic.

## ADRs

| ADR | Decision | Resolves | Status |
|-----|----------|----------|--------|
| [[ADR-BE-001 — Error Handling Sentinel Errors]] `(promote)` | Sentinel errors → wrapPGError → handler status mapping | B2, B3 | Accepted |
| [[ADR-BE-002 — Hexagonal Wiring]] `(promote)` | Ports/services/adapters dependency rule + struct conventions | — | Accepted |
| [[ADR-BE-003 — Data Access pgxpool No ORM]] `(promote)` | Hand-written SQL on pgx/v5, single shared pool | — | Accepted |
| [[ADR-BE-004 — Database Migrations]] | Sequential up/down SQL, constraint fixes via migration | T3, B1 | Accepted |
| [[ADR-BE-005 — Auth JWT + Refresh Rotation]] | Two-token cookies, rotation + reuse detection, cookie-name unity | T5, T6, T9 (P0-5) | Accepted |
| [[ADR-BE-006 — Middleware Composition]] | `func(Handler) Handler` chain, global + per-route order | — | Accepted |
| [[ADR-BE-007 — Rate Limiting]] | Tiered in-memory limits, stricter on auth | S4, T4 | Accepted |
| [[ADR-BE-008 — API Versioning Accept Header]] | Media-type versioning, v1 now | — | Accepted |
| [[ADR-BE-009 — Testing testcontainers testify]] `(promote)` | 3 tiers: mock unit / testcontainers integration / smoke | B4, test gaps | Accepted |
| [[ADR-BE-010 — Logging stdlib slog successor]] | stdlib `log` now, `slog` chosen successor | — | Accepted |
| [[ADR-BE-011 — CORS Policy]] | Explicit whitelist, **no wildcard**, credentials | S2 | Accepted |
| [[ADR-BE-012 — Audit Log Writes]] | Detached context, logged failures, no silent drops | T7 | Accepted |
| [[ADR-BE-013 — Password Reset Out-of-Band Codes]] | Codes never in responses, 6+ char entropy, attempt limits | T8 (P0-6) | Accepted |
| [[ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution]] | Routing via activity → anchored WG manager/delegate; no fallback; D-11 skip incl. delegates; unit-subtree visibility; `enforce_unit_tuple` drop | P-001 Q1–Q4, P-007 | Accepted |
| [[ADR-BE-015 — Demo Environment Topology]] | Compose + Caddy + cloudflared (named tunnel + Access); zero published ports; seed via `scripts/seed_demo.sql`; VPS as portable escalation path; k8s deferred | Phase 11 demo | Accepted |
| [[ADR-BE-016 — Origins Tickets & Audit Schema Encoding]] | Origins/tickets/audit schema: migrations 014–017 shapes, 3VL CHECK guard rule, three-layer ticket model (state/comments/audit), synchronous in-tx ticket audit writes (BE-012 scope extended), terminal-activity definition, `LoggedHours` guard signature, transition matrix, sold_hours semantics | Phase 11 semantic resolutions (OQ1–OQ6, D-05/06/10/13/14) | Proposed |
| [[ADR-BE-017 — Coverage Encoding]] | Allocation ledger (migrations 018-020): tagged-union source_type with 3VL CHECK guard, chain-driven proposals on read, replace-set with in-tx Σ validation, manager gate via BE-014, frozen period-close snapshots, D-K polymorphic validation cost | COV-01..05, D-01..D-12 | Accepted |

## Coverage vs. the audit

All 12 undocumented decision areas from [[research/2026-07-28 — Pre-Deployment Audit — Hourglass v0.1]] §7 now have ADRs. P0-adjacent items (auth rotation, audit writes, password reset) are ADR-BE-005/012/013; the remaining P0 fixes (DB constraint, list views, customers route, error boundaries) are tracked in the audit's priority matrix, not as ADRs, because they're defects/gaps rather than new decisions.

## Rules

* Append-only; supersede via status + links.
* Each ADR cites the code it describes and the audit item it resolves, where applicable.
* `(promote)` entries move to the global Go set when one is created; this folder then links instead of restating.
