# Hourglass Customer Role Contract

## Purpose

This contract names the jobs a user acting as a **customer** performs in Hourglass. The answer is that there is **no app surface**: the role has no jobs of its own, no surfaces, and no entry point (CUST-01, D-E).

- **Authority split:** `docs/design/LANGUAGE.md` wins on type, color, density, motion, and status vocabulary; `docs/design/CHROME.md` wins on the frame, sidebar, and page anatomy. This file adds jobs, surfaces, copy, and composition only and `MUST NOT` override either.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose.
- **A decision, not an omission:** `no app surface` is the recorded conclusion of D-E (tickets are internal-only), re-affirmed by the rejected-list of `ADR-P-011`. This file exists so later phases cite the decision instead of re-litigating it.
- **Not authorization:** the few statements here that describe reachability are UX scoping notes. The server stays authoritative (D-19-12); where it does not enforce something today, that is a gap, never a permission.

## Changelog

- 2026-09-26 · Phase 19 · Added customer role contract

## Actor

- **Role string:** `customer`, present in the frontend role union (`web/src/types/models.ts:1`), in the Go role vocabulary (`internal/models/models.go:15`), and in the database role vocabulary (`migrations/012_staffing_schema.up.sql:48-50`). Like every membership role it is carried by the active membership (`GET /auth/me` → memberships) and minted into the JWT claim, and switching organizations switches the active membership and therefore the role.
- **How the membership appears:** `Role.IsValid()` accepts `customer` (`internal/models/models.go:19-27`), so an existing member can be moved to it through the finance-owned member-roles route. The customer-facing path — `POST /organizations/invite-customer` — is a stub today (`G7`), so an external party cannot be onboarded as a customer through the app.
- **Derived hats:** none. The customer role never carries the working-group manager/delegate hat described in `employee.md`, because it is never a working-group member.
- **What a customer is not:** not an org member in any working sense, not a ticket actor (the ticket service rejects the role outright), not a reader of records, not an approver, not an approver of their own requests, and not an actor in any lifecycle group. The jobs that touch customer data are performed by other roles and are cross-referenced in `C-02`.

## Scope

Per-role rows (the assembled role×surface matrix is Phase 20, D-19-08):

| Surface | Customer sees | Evidence today |
|---|---|---|
| Today (landing) | nothing — no composition is built for this role | `web/src/routes/_authenticated/-components/today-page.tsx` (employee shape only) |
| Lifecycle groups (`Record` · `Organize` · `Approve` · `Report`) | nothing | `CHROME.md` role-scoped chrome |
| Tickets | nothing — tickets are internal-only | `internal/core/services/ticket/ticket.go:79` |
| Customer records | nothing — the surface belongs to the role that keeps the records | `F-03` |
| Contracts, activities, coverage, direction | nothing | `F-02`, `F-04`, `M-04` |
| Exports | nothing intended — today the export `default` branch would hand a customer org-wide rows | `G6` |
| Nav entry, route, page | **never** — absence is the mechanism | `C-01` |

- Customer sees **nothing in the app**. Visibility is not the mechanism: no role predicate hides a customer surface, because no customer surface exists to hide. Absence is the mechanism.
- Never sees, by decision rather than by predicate: tickets and their status; contract and bucket economics; approval queues; org-wide reporting; people, org-tree, and staffing; and any configuration surface.
- Not authorization: a `customer` membership reaches auth-only routes that carry no role gate, so some reads are technically possible today (org-wide entry/expense lists `G2`, org-wide export rows `G6`). Those are enforcement gaps, not an intended surface, and `MUST NOT` be treated as capability.

## Jobs

### C-01 — No app surface

| field | value |
|---|---|
| Actor | `customer` (membership role; no derived hats) |
| Trigger | the role is assigned to a membership, or an external party is expected to be invited as a customer |
| Steps | 1. None — there is `no app surface` for this role: no entry point to open, nothing to read, nothing to act on. 2. Every customer-facing need is routed to the internal job that owns it (`C-02`). |
| Surfaces | none — no nav entry, no route, no page. The target state is absence. |
| API interactions | none |
| Current state | exists — the role string exists (`web/src/types/models.ts:1`, `internal/models/models.go:15`, `migrations/012_staffing_schema.up.sql:48-50`); the surface does not exist and is not planned |
| Backend authority | The ticket service rejects the `customer` role on create, list, get, and history (`internal/core/services/ticket/ticket.go:79`, `:120`, `:135`, `:443`), which is the one server-side statement of this decision. Tickets are internal-only (D-E, cited at `ADR-P-003` lines 11, 27, 67, 93). Nothing else enforces the role: a customer membership reaching auth-only routes without a gate is an enforcement gap (`G2`, `G6`), never an intended surface. |

Fences:

- `MUST NOT` present a customer nav entry, route, or surface — in contracts, composition maps, sketches, or implementation plans.
- `MUST NOT` treat customer records as a user-facing surface: records are internal objects maintained by finance.
- `MUST NOT` infer that the presence of `customer` in the role union is a backlog item. The union lists the vocabulary the identity layer supports, not the surfaces that must be built.
- A customer portal and external ticket intake are out of scope for v0.2.1 (`ADR-P-011` rejected list; D-E).

### C-02 — The internal jobs that serve customers

| field | value |
|---|---|
| Actor | other roles only — finance and manager, per the owning job. Never the customer. |
| Trigger | a commercial or customer need arises inside another role's workflow |
| Steps | none owned here — the steps live in the owning contract under the cited job id |
| Surfaces | the owning roles' surfaces (customer records, activities, coverage, exports); never a customer surface |
| API interactions | `GET /customers`, `POST /customers`, `PUT /customers/{id}`, `DELETE /customers/{id}` (`F-03`) · `POST /activities` with the `customer_ticket` origin (`F-04`) · `PUT /time-entries/{id}/allocations` (`M-04`) · `GET /exports/timesheets`, `GET /exports/expenses`, `GET /exports/combined` (+`/count` variants) (`F-07`) · `POST /organizations/invite-customer`, a stub that calls no service (`G7`). No per-customer report endpoint exists. |
| Current state | partial — customer records, activities, coverage allocation, and exports exist; the customer invitation is a stub (`G7`) and no per-customer report surface exists |
| Backend authority | Customer record writes are finance-gated (`internal/adapters/primary/http/customer.go:101`, `:190`, `:222`); activity creation is gated by origin (manager or finance for `customer_ticket`); coverage allocation writes go through the routing approver with the coverage owner forbidden; export row scoping's `default` branch makes `customer` org-wide (`G6`). |

Cross-references — the only places customers are handled app-visibly (do not restate their steps):

| Customer-facing need | Owning contract | Job id |
|---|---|---|
| Customer record upkeep | `finance.md` | `F-03` |
| Customer-ticket-origin activities | `finance.md` | `F-04` |
| Warranty / goodwill coverage labeling | `manager.md` | `M-04` |
| Per-customer non-billed cost report | `finance.md` | `F-07` |
| Customer invitations | no owning job — the route is a stub (`G7`) | — |

This is the whole of the app-visible handling of customers: it happens inside other roles' surfaces, under other roles' gates. A future customer surface would not "move" these jobs; it would have to justify itself against them (`C-03`).

### C-03 — What would reopen the decision

| field | value |
|---|---|
| Actor | no membership role — this is a decision record, owned by the project's design/ADR process |
| Trigger | evidence arrives that a customer must interact with the app directly |
| Steps | 1. Record the need with evidence, not preference. 2. Write a new ADR that supersedes D-E and states the surface, its scope, and its gates. 3. Land a milestone change that carries the surface into a phase. 4. Only then may this contract gain jobs, and only then may composition reference them. |
| Surfaces | none today |
| API interactions | absent — the only customer-facing endpoint is the `POST /organizations/invite-customer` stub (`G7`) |
| Current state | absent — no portal, no external intake, no customer reads; the decision survives only as secondhand citations in `ADR-P-003` (lines 11, 27, 67, 93) — `G11` |
| Backend authority | Nothing enforces or forbids a customer surface, because none exists. The process is the authority: a new ADR plus a milestone change, never a silent addition of a route, nav entry, or page. |

Evidence that would justify reopening:

- A business need to expose ticket status or cost reports to the customer, stated as a need and not as symmetry with the other roles.
- An external intake requirement — a demand for a channel outside the organization, which tickets deliberately do not provide in v0.2.
- An explicit decision to open a portal, with its identity, invitation, and access model defined rather than assumed.

Absent that evidence the conclusion stands: `no app surface`.

## Do / don't

- Don't build a nav entry, route, or page to "complete" the five-value role union — the union describes the identity vocabulary, this contract describes jobs, and the jobs are absent.
- Don't read `GET /customers` or org-wide export reachability by a customer membership as capability: both are enforcement gaps (`G2`, `G6`).
- Don't restate the steps or gates of `F-03`, `F-04`, `M-04`, or `F-07`; cite the ids.
- Don't promise external ticket intake or a portal; both were rejected for v0.2 (`ADR-P-011` rejected list, D-E).
- Don't treat the `POST /organizations/invite-customer` stub as a surface: it returns a canned message and calls no service (`G7`).
- Don't restate nav groups, tokens, or component APIs.

## Pointers

- `docs/design/workflows/README.md` — job-id scheme and the shared gap register.
- `docs/design/workflows/finance.md` — `F-03` (customer records), `F-04` (activities), `F-07` (reporting).
- `docs/design/workflows/manager.md` — `M-04` (coverage allocation, money labels).
- `docs/design/LANGUAGE.md`, `docs/design/CHROME.md` — the two authorities this file must not override.
- `internal/core/services/ticket/ticket.go` — the customer rejection that grounds `C-01`.
- `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md` — D-E policy and the permission gate table.
- `.planning/REQUIREMENTS.md` — CUST-01.

## Not in this file

- No chrome, sidebar, or navigation model (that is `CHROME.md`).
- No design-language vocabulary (that is `LANGUAGE.md`).
- No cross-role composition map (Phase 20, `COMPOSITION.md`).
- No page layouts, route definitions, component APIs, or implementation sequencing.
- No value tables (colors, spacing, sizes) — those live in the CSS and `LANGUAGE.md`.
- No Admin/Settings surface.

## Gaps

- `G11` — the note that decided D-E ("no customer app surface") is not on disk; D-E survives only as secondhand citations in `ADR-P-003` (lines 11, 27, 67, 93). This contract is a re-recording of the decision, not the original record.
- `G7` — `POST /organizations/invite-customer` is a stub that returns a canned message and calls no service, so a customer cannot be onboarded through the app and the only customer-facing endpoint is inert.
- `G6` — export row scoping ends in a `default` branch that makes `customer` org-wide; a customer membership with a token can export other members' rows.
- `G2` — `GET /time-entries` and `GET /expenses` are org-wide for every authenticated role, so the same membership can list rows it was never intended to see.
- `G1` — no route-level role enforcement exists to rely on; the only customer-specific check in the system is the ticket service's inline rejection.
- The gaps above `MUST NOT` be closed by editing this contract; they are the delta list for the job-cluster phases inserted after Phase 20.
