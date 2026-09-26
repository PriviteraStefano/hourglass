# Hourglass HR Role Contract

## Purpose

This contract names the jobs a user acting as **HR** performs in Hourglass, and the surfaces those jobs need. Jobs are workflows, not routes (D-19-02/D-19-11).

- **Authority split:** `docs/design/LANGUAGE.md` wins on type, color, density, motion, and status vocabulary; `docs/design/CHROME.md` wins on the frame, the sidebar, and page anatomy. This file adds jobs, surfaces, copy, and composition only and `MUST NOT` override either.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults, rationale in prose.
- **Not authorization:** every visibility statement here is UX scoping. The server's routing and role checks are authoritative; where enforcement is missing it is listed as a gap, never as a permission.

## Changelog

- 2026-09-26 · Phase 19 · Added hr role contract

## Actor

- **Role string:** `hr`, carried by the current membership (`GET /auth/me` → memberships) and minted into the JWT role claim from `organization_memberships.role`. Switching organizations switches the active membership and therefore the role.
- **Curator and consumer, never an approver.** HR curates the people data other jobs depend on — availability windows, employment validity, unit composition — and consumes the capacity and payroll read models. HR does not own a stage in any approval chain (ADR-P-008 D-4; ADR-P-011 D-3), and the frontend already strips it: `deriveApprovalStages` returns no stage for `hr` (`web/src/lib/role-visibility.ts:33`).
- **The role union already knows `hr`:** `Role = "employee" | "manager" | "finance" | "hr" | "customer"` (`web/src/types/models.ts:1`); the sidebar, the approval stages, and the review predicates are written against that union.
- **What HR is not:** not a stage-1 or stage-2 approver, not a capture surface for time or expenses, not the owner of commercial records (contracts, customers, activities), not the owner of membership role changes (that is finance, `F-06`), and not the owner of an Admin/Settings surface — none exists (SC2).
- **Backend reality (the headline gap):** `hr` is not a valid Go role. There is no HR constant in `models.Role` and `Role.IsValid()` rejects it (`internal/models/models.go:19`); only the database CHECK accepts it (`migrations/012_staffing_schema.up.sql:49-50`). An out-of-band `hr` membership therefore reaches auth-only routes and every equality gate fails closed, treating it as unprivileged. No job in this contract can be performed end-to-end today (`G5`).

## Scope

Per-role rows (the assembled role×surface matrix is Phase 20, D-19-08):

| Surface | HR sees | Evidence today |
|---|---|---|
| People composition (units, tree, memberships) | the unit tree and membership, shared with the manager | `web/src/routes/_authenticated/org-hierarchy/`; `H-03` |
| Availability | every member's declared/confirmed windows, read and curate | `H-01` (no route — `G8`) |
| Employment validity | `valid_from` / `valid_until` / permit expiry per membership | `H-02` (no reader — `G9`) |
| Capacity watch | capacity, planned, and gap per employee, unit, and working group, with warnings | `H-04` (manager-gated today) |
| Exports (payroll view) | timesheet export rows under the payroll scope | `H-04`; today org-wide through `G6` |
| `Approve` | **never** — no stage-1 and no stage-2 | `H-05`; `web/src/lib/role-visibility.ts:33` |
| Time-entry and expense capture | never — capture belongs to the employee (`E-01`, `E-02`) | — |
| Commercial records (contracts, customers, activities) | never — finance owns them (`F-02`, `F-03`, `F-04`) | — |
| Admin/Settings | does not exist; see *Not in this file* | — |

These rows are UX scoping, not authorization: reachability is not entitlement, and the server stays authoritative (D-19-12). Until `G5` closes, the honest statement is that HR *would* hold these rows and holds none of them in practice.

## Jobs

### H-01 — Curate availability

| field | value |
|---|---|
| Actor | hr (curator); the employee is the declarer (`E-08`) |
| Trigger | an employee declares an absence, or a window needs confirming before planning relies on it |
| Steps | 1. Open the availability surface for the member. 2. Read the declared window (kind, dates, hours, certificate reference). 3. Confirm it, or correct it and return it to declared. 4. Mark it confirmed once the certificate reference checks out. |
| Surfaces | Availability (target surface — no route exists today) · People composition (per-member view) |
| API interactions | `absent` — no curation write route; the only availability read is server-internal (`internal/adapters/secondary/postgres/direction_repository.go:763`) — `G8` |
| Current state | absent — `availability_windows` exists with kinds `holiday\|permit\|medical\|unavailable` and status `declared\|confirmed`, reads are wired, writes are not |
| Backend authority | Nothing is exposed, so nothing is enforced. The target: the employee declares, HR curates and confirms (ADR-P-008 D-4). `hours` NULL means full absence; `certificate_ref` is medical-only (ADR-P-008 D-1a). A window never blocks planning — it feeds the capacity read-model, where capacity = daily hours − absence hours, and the scheduler warns `away` or `partial` only. |

### H-02 — Curate employment validity

| field | value |
|---|---|
| Actor | hr |
| Trigger | a membership's employment window or work permit needs to be set or corrected |
| Steps | 1. Open the member record. 2. Set `valid_from` / `valid_until`. 3. Record `work_permit_expires_at` when a permit applies. 4. Save. |
| Surfaces | People composition → member record (target surface) |
| API interactions | reads `GET /units/{id}/members`, `GET /organizations/members`; writes `absent` — no membership-validity write route — `G9` |
| Current state | absent — the columns exist (`migrations/012_staffing_schema.up.sql:39-42`); `valid_from` / `valid_until` are consumed by direction planning, `work_permit_expires_at` has no Go reader (`G9`) |
| Backend authority | Target: validity dates bound the days direction planning may schedule a member for, and permit expiry warns before it lapses. Today the permit column is read by nothing, and neither validity field is writable through the API — `PUT /organizations/members/{member_id}/roles` is finance-owned and does not carry validity. The write path is the gap; validity itself is read only inside the membership and direction paths. |

### H-03 — People composition, shared with the manager

| field | value |
|---|---|
| Actor | hr, shared with the manager (`M-06` owns the manager half) |
| Trigger | the unit tree, a unit's structure, or its membership needs shaping |
| Steps | 1. Open the composition surface. 2. Read the unit tree. 3. Add or move a member within a unit. 4. Adjust unit structure when responsibilities change. |
| Surfaces | People composition (unit tree, unit detail, member list) |
| API interactions | `GET /units`, `GET /units/tree`, `GET /units/{id}/descendants`, `GET /units/{id}/members`, `GET /organizations/members` |
| Current state | exists on the client — `web/src/routes/_authenticated/org-hierarchy/` — with no role predicate in the client and no role gate on the server (`G3`); `M-06` is the owning job for the manager half |
| Backend authority | **This is shared manager/HR composition and is explicitly not Admin/Settings** (SC2). No unit route and no working-group route carries a role gate today (`G3`), so reachability is not entitlement; the target state is server-enforced for manager and HR. Membership role changes and deactivation stay finance (`F-06`), and working-group membership is a separate dimension (`M-07`). |

### H-04 — Capacity and payroll view

| field | value |
|---|---|
| Actor | hr (consumer) |
| Trigger | I need to see absence load and capacity, or a payroll-scoped export of reporting data |
| Steps | 1. Open the capacity view scoped to an employee, unit, or working group. 2. Read capacity, planned, and gap, plus warnings. 3. Open exports. 4. Download the payroll-scoped CSV or XLSX. |
| Surfaces | Capacity watch (target surface) · Exports (payroll view — target surface) |
| API interactions | `GET /direction/coverage`, `GET /exports/timesheets` |
| Current state | partial — coverage is manager-gated today (`internal/adapters/primary/http/direction_handler.go:238`), so HR cannot reach it; `/exports/timesheets` is reachable but row scoping falls through the `default` branch to org-wide (`internal/adapters/secondary/postgres/export_repository.go:29`) — `G6`. Cross-references `M-03` (capacity read) and `F-07` (org-wide reporting). |
| Backend authority | The capacity read is a read-model: daily hours − absence hours, coverage rows carrying capacity/planned/gap, warnings `away\|partial\|over-capacity\|invalid`, advisory only and never blocking. The target adds HR to its scope per employee/unit/working group; today the gate admits manager only, which is the HR gap. The payroll view is a target scoping of the existing timesheet export — not a new export format — and the server-enforced range cap stays in force. |

### H-05 — Never an approver

| field | value |
|---|---|
| Actor | hr |
| Trigger | an entry, expense, proposal, or coverage allocation reaches an approval stage HR might otherwise hold |
| Steps | 1. HR holds no stage, so no approval action is offered. 2. Stage-1 stays with the manager (`M-01`) and stage-2 with finance (`F-01`). 3. If an HR membership also appears as a working-group manager or delegate, the working-group-derived stage is still stripped. |
| Surfaces | `Approve` (never) — HR `MUST NOT` be offered an approve or reject action |
| API interactions | `absent` — there is no HR-specific eligibility check: the generic `manager`/`finance` equality gate rejects the action (403), which is how HR is excluded today — an accident of that gate (`G5`), not a rule. The deliberate enforcement is the frontend predicate (`web/src/lib/role-visibility.ts:33`), which is UX scoping — `G1`, `G5` |
| Current state | exists for the FRONTEND predicate — `deriveApprovalStages` returns `[]` for `hr`, stripping even a working-group-derived stage, pinned by `web/src/lib/__tests__/role-visibility.test.ts`; absent for backend enforcement *as a rule* — an `hr` membership is rejected only because the generic `manager`/`finance` equality gate does not match it (`G5`) |
| Backend authority | Target: HR is never an approver at either stage (ADR-P-008 D-4; ADR-P-011 D-3). Today an out-of-band `hr` membership is failed closed by accident rather than by rule: there is no Go role constant, `Role.IsValid()` rejects it, and every equality gate therefore treats it as unprivileged (`G5`) — which also means HR cannot be assigned through the API at all. The only deliberate enforcement is the frontend predicate, so HR's exclusion is currently scoping plus an equality-gate accident, not an authorization rule. |

## Do / don't

- Don't offer HR an approve or reject action, even when the membership is also a working-group manager or delegate — the working-group-derived stage is stripped (`H-05`).
- Don't treat people composition as Admin/Settings: it is a shared manager + HR job (`M-06`, `H-03`), and working-group membership is a further dimension (`M-07`).
- Don't promise availability curation, validity writes, or the payroll export view as available — they are target surfaces with no write route (`G8`, `G9`, `G6`).
- Don't give HR commercial records, money labeling, or org-wide report ownership; name the owning job (`F-02`, `F-05`, `F-07`).
- Don't restate nav groups, tokens, or component APIs.

## Pointers

- `docs/design/CHROME.md` — frame, lifecycle nav groups, role-scoped chrome.
- `docs/design/LANGUAGE.md` — status and warning vocabulary.
- `web/src/lib/role-visibility.ts` — the predicate mechanism this contract's scope rows agree with.
- `docs/history/planning/REQUIREMENTS.md` — HR-01 (this contract) and the archived leftovers `AVAIL-03`…`AVAIL-05`, `SURF-05` used as job hints.
- `internal/adapters/secondary/postgres/direction_repository.go` — the availability read behind `H-01` and the capacity read-model behind `H-04`.
- `cmd/server/main.go` — the route surface the `API interactions` columns cite.
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` — D-4 curator/consumer, never approver; D-1a certificate reference.
- `hourglass-vault/decisions/project/ADR-P-011 — Information Architecture & Role-Scoped Surfaces.md` — D-3 review gating.

## Not in this file

- No chrome/layout contract and no nav-model restatement (that is `CHROME.md`).
- No design-language vocabulary (that is `LANGUAGE.md`).
- No cross-role composition map (Phase 20, `COMPOSITION.md`).
- No Admin/Settings surface (out of v0.2.1, SC2).
- No page layouts, route definitions, component APIs, or implementation sequencing.
- No value tables (colors, spacing, sizes).
- No employee, manager, finance, or customer jobs (see the sibling contracts).

## Gaps

- **`G5` — headline: `hr` is not a valid Go role.** There is no HR constant in `models.Role` and `Role.IsValid()` rejects it (`internal/models/models.go:19`); `hr` appears only in the migration CHECK (`migrations/012_staffing_schema.up.sql:49-50`). **Consequence: no HR job in this contract can be performed end-to-end today** — HR cannot be invited or assigned through the API, reaches only auth-only routes, and is treated as unprivileged by every equality gate. The authorization stated per job above is the **target**; the gap register is the delta list an implementation phase works from.
- `G8` — availability windows are read-only: no write route exists for employee declarations or HR curation (`H-01`).
- `G9` — `organization_memberships.work_permit_expires_at` has no Go reader, and no route writes validity (`H-02`).
- `G3` — neither the 12 unit routes nor the 8 working-group routes carry a role gate, so `H-03` composition has no server-side entitlement (`H-03`).
- `G6` — export row scoping's `default` branch makes `hr` org-wide, which is not the payroll scope `H-04` describes (`H-04`).
- `G1` — `middleware.RequireRole` is defined and unit-tested but never wired to a route, so `H-05` has no backend enforcement to rely on (`H-05`).
