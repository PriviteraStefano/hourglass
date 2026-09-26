# Phase 19: Role contracts - Context

**Gathered:** 2026-09-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Docs-only design contract for v0.2.1. This phase delivers EMP-01, MGR-01, FIN-01, HR-01, CUST-01: five **role contracts** under `docs/design/workflows/` (reserved by `docs/design/INDEX.md` line 7). Each contract names the **jobs** that role performs and the surfaces those jobs need. Jobs are workflow-shaped, not current routes. Customer may conclude "no app surface".

**Write set (only these files):**
1. `docs/design/workflows/employee.md` (new)
2. `docs/design/workflows/manager.md` (new)
3. `docs/design/workflows/finance.md` (new)
4. `docs/design/workflows/hr.md` (new)
5. `docs/design/workflows/customer.md` (new)
6. `docs/design/workflows/README.md` (new — index + shared gap register + job-id scheme)
7. `docs/design/INDEX.md` only if the reserved `workflows/` line needs adjustment once the files exist

**Success criteria that must be TRUE (ROADMAP Phase 19):**
1. Employee role contract exists and names jobs, not routes (EMP-01)
2. Manager role contract exists; org-tree work is a manager job, not Admin (MGR-01)
3. Finance role contract exists covering cutoffs, coverage money-labeling, reporting (FIN-01)
4. HR role contract exists; org-tree / people composition is shared with manager, not Admin (HR-01)
5. Customer role contract exists and may conclude "no app surface" (CUST-01, D-E)
6. Archived v0.2 leftovers (TICK-06, AVAIL-03..05, SURF-*, POLS-*) are used as job-shaped hints, not copied as page requirements
7. No UI implementation, no sketches, no route work

**Out of this phase:** UI implementation, component work, route/page work, sketches (SKETCH-01 is Phase 20), the cross-role composition map (COMP-01, Phase 20), Admin/Settings surfaces, the assembled role×surface matrix, POLS-* quality-bar enforcement, and any backend change.

</domain>

<decisions>
## Implementation Decisions

### Deliverable shape
- **D-19-01:** Five role contracts, **one file per role**, lowercase filenames under the reserved folder: `docs/design/workflows/employee.md`, `manager.md`, `finance.md`, `hr.md`, `customer.md`, plus `README.md` as index, job-id scheme, and shared gap register. — **Reversibility:** costly — every later composition reference and job-cluster plan cites these paths; renaming means re-citing across Phase 20 and the inserted implementation phases.
- **D-19-02:** **Full job record.** Each job carries: stable id · title · actor (membership role + any derived hat) · trigger · steps · surfaces · API interactions (method + path) · current state (exists / partial / absent, with route + code evidence) · backend-authority note (what the server enforces vs what is UX scoping). Rationale: LANGUAGE.md pins Phase 19 contracts as "a complete workflow spanning pages and API interactions".
- **D-19-03:** **Numbered job ids**, one namespace per contract: `E-01…` (employee), `M-01…` (manager), `F-01…` (finance), `H-01…` (hr), `C-01…` (customer). Ids are stable so Phase 20 `COMPOSITION.md` and post-Phase-20 job-cluster phases can cite them. — **Reversibility:** costly — downstream phases cite ids; renumbering invalidates those citations.

### Roles and identity
- **D-19-04:** **One membership role per session.** The JWT/`/auth/me` role comes from `organization_memberships.role`; the org switcher changes org + role. Working-group manager/delegate approval authority is a **second, independent dimension** (`deriveApprovalStages` over WG `manager_id`/`delegate_ids`), which every contract that touches approval must name explicitly rather than treating WG authority as an org role. — **Reversibility:** costly — the identity statement is consumed by all five contracts and by the chrome/page-action predicate work.
- **D-19-05:** **Stage-1 approval is owned by `manager.md`** (one job, one owning contract). `employee.md` records that a WG member may hold that hat without holding the manager membership role, and cross-references the job id. The synthesized `wg_manager` label used by the pending-queues endpoint is named as a derived hat, never as a role.
- **D-19-06:** **HR is first-class** (`hr.md` exists as a full contract), with the backend gap named (see D-19-08). HR is a curator/consumer, **never an approver** (ADR-P-008 D-4; ADR-P-011 D-3).
- **D-19-07:** **CUST-01 concludes "no app surface."** `customer.md` carries the decision, its rationale, the fence (portal out of scope, tickets internal-only per D-E), the **internal jobs that serve customers** cross-referenced to their owning role (customer-record upkeep → finance; customer-ticket-origin activities → manager/finance; warranty/goodwill coverage labeling → manager/finance; per-customer non-billed cost report → finance), and the conditions that would reopen the decision. Note recorded as a **gap**: the note that decided D-E is **not on disk** — D-E survives only as secondhand citations (ADR-P-003 lines 11, 27, 67, 93). — **Reversibility:** costly — later composition and implementation phases inherit the "customer has no surface" assumption; formally adding a customer surface means a new ADR plus a new milestone surface.

### Visibility, gaps, and authority
- **D-19-08:** **Per-role visibility rows now, the assembled matrix in Phase 20.** Each contract declares its own role-scoped rows (what this role sees/holds) in a scope statement; the single role×surface matrix from ADR-P-011 D-5 is assembled in `COMPOSITION.md` (COMP-01). ADR-P-011 D-5 is therefore treated as an input to confirm or revise per role, never silently adopted as the matrix. — **Reversibility:** reversible — rows are doc-local; Phase 20 can re-cut them.
- **D-19-09:** **Backend-truth gaps are recorded twice**: a `Gaps` section in the affected contract and the shared register in `workflows/README.md`. Contracts still state the **target authorization** each job requires, so an implementation phase can see the delta. Verified gaps to record: `middleware.RequireRole` is defined and tested but **never wired** (`internal/middleware/middleware.go:46`); `GET /time-entries` and `GET /expenses` are **org-wide for every authenticated role** because `ports.ListFilters.Role/RequestUserID` are never consumed (`internal/adapters/secondary/postgres/time_entry_repository.go:97-152`); all 12 unit routes and all 8 working-group routes have **no role gate**; `POST /invitations` has **no Auth middleware** and takes `organization_id` from the body; **`hr` is rejected by `models.Role.IsValid()`** (present only in the migration 012 CHECK) so it cannot be invited or assigned through the API and every equality gate fails closed; exports' row scoping ends in a `default` branch that makes **finance/hr/customer org-wide** (`export_repository.go:29-37`); `POST /organizations/invite-customer` is a **stub**; `POST /coverage/close` is **manager-only**.
- **D-19-10:** **Period-close split, no gap.** FIN-01's "cutoffs" is honored as: finance owns the **snapshot read + reporting cadence**, the manager owns the **close action**. Nothing in the contracts claims finance can close a period.
- **D-19-11:** **POLS-01..11 are not carried as a quality bar.** Contracts record existing surfaces as `current state` evidence only; the quality bar for touched surfaces is JOB-01 / job-cluster territory.
- **D-19-12:** **Authority stack honored.** Contracts may only add surface, layout, copy, and composition. `LANGUAGE.md` wins on type/color/density/motion/status vocabulary; `CHROME.md` wins on frame, navigation, and page anatomy. Contracts `MUST NOT` restate the four lifecycle nav groups as their own model, `MUST NOT` include Admin/Settings, `MUST NOT` treat nav/visibility hiding as authorization, and `MUST NOT` claim the frozen components (PageHeader, FilterBar, DataTable, EmptyState, ConfirmDialog) exist.

### the agent's Discretion
- Section order inside each contract, provided it follows the house doc pattern (Purpose → Changelog → contract body → Do/don't → Pointers → Not in this file → Gaps) established by `LANGUAGE.md`/`CHROME.md`.
- Job granularity inside the confirmed inventory (merging or splitting a job is allowed if the workflow stays intact and the id namespace stays stable).
- How much of the API-interaction column is a table vs inline; RFC 2119 usage follows the house rule (MUST/MUST NOT for conformance, SHOULD for defaults, rationale in prose).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone / phase
- `.planning/ROADMAP.md` §Phase 19 — goal, the 7 success criteria, dependency on Phase 18, `UI hint: no`
- `.planning/REQUIREMENTS.md` — EMP-01/MGR-01/FIN-01/HR-01/CUST-01, the "Historical inputs (not requirements)" hint table, Out of Scope table, traceability
- `.planning/PROJECT.md` §Current Milestone — "contract-first presentation — job clusters", hard rules
- `.planning/STATE.md` — current focus, deferred items, "do not sketch; do not implement"
- `.planning/milestones/v0.2-REQUIREMENTS.md` — exact wording of TICK-06 (line 33), AVAIL-03..05 (56-58), UXFD-02 (63), SURF-01..08 (67-74), POLS-01..11 (78-88) + dropped-traceability table (160-183)

### Design authority (contracts must not override)
- `docs/design/INDEX.md` — design doc map; line 7 reserves `docs/design/workflows/` for Phase 19
- `docs/design/LANGUAGE.md` — language authority (type/color/density/motion/status); "workflow contracts are workflow-group oriented … spanning pages and API interactions"; authority stack lines 3-10
- `docs/design/CHROME.md` — shell authority (frame, four lifecycle nav groups, role-scoped chrome, page anatomy); its `Gaps` line 96 (GAP A) and the per-role-matrix deferral
- `.planning/phases/17-design-language-contract/17-CONTEXT.md` + `17-VERIFICATION.md` — doc-pattern rules (D-17-02/04/05/06/09/10/20/21/22) and the verified deliverable
- `.planning/phases/18-chrome-contract/18-CONTEXT.md` + `18-DISCUSSION-LOG.md` — D-18-01..11, especially the Phase 19 ownership handoffs

### Domain / IA inputs to confirm or revise
- `hourglass-vault/decisions/project/ADR-P-011 — Information Architecture & Role-Scoped Surfaces.md` — D-1..D-6; D-5 matrix lines 61-70; UX-scoping caveat line 72
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` — D-4 (HR curator/consumer, never approver), D-1a certificate_ref
- `hourglass-vault/decisions/project/ADR-P-003 — Tickets as the Second Capture Layer.md` — D-E citations (11, 27, 67, 93), ticket permission gate table 69-76
- `hourglass-vault/decisions/project/ADR-P-001 — Units vs Working Groups.md`, `ADR-P-004 — The Today View.md`, `ADR-P-012 — Facts vs Decisions: The Coverage-Allocation Ledger.md`, `ADR-P-015 — Direction, The Plan Plane.md`
- `.planning/sketches/SKETCH-LOOP-CONTRACT.md` — read-only context; its reconcile is Phase 20, not now

### Live surfaces and mechanisms
- `web/src/types/models.ts:1` — `Role = "employee" | "manager" | "finance" | "hr" | "customer"`
- `web/src/lib/role-visibility.ts` — the only role predicates (`deriveApprovalStages:25-48`, `isReviewVisible:51-53`, `isEconomicsVisible:56-58`, `isAdminVisible:64-67`)
- `web/src/lib/__tests__/role-visibility.test.ts`, `web/src/components/layout/__tests__/sidebar-groups.test.tsx` — behavior pinned today
- `web/src/components/layout/sidebar.tsx:65-128` — live ADR-P-011 nav groups (not yet the four lifecycle groups)
- `web/src/components/approval/approval-buttons.tsx:32-42` — role × status action gate
- `web/src/routes/_authenticated/**` — the 13 live route files (current-state evidence)
- `cmd/server/main.go:48-192` — all 118 route registrations and middleware composition
- `internal/models/models.go:11-27` — role constants + `IsValid` (no HR)
- `internal/middleware/middleware.go:46` — `RequireRole`, defined and never wired
- `internal/core/services/routing/routing.go` — stage-1 resolution (WG approver set, R-2 unit-manager walk, `RoleGated` terminal)
- `internal/adapters/secondary/postgres/export_repository.go:29-37` — export row scoping
- `migrations/012_staffing_schema.up.sql:39-50` — membership validity columns + the `hr` CHECK extension

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/lib/role-visibility.ts` — the single predicate module every contract's scope statement must agree with; contracts name the target mechanism, not a second one.
- `web/src/components/approval/approval-buttons.tsx` — the existing role × status action gate; manager/finance stage split already encoded.
- `web/src/routes/_authenticated/{today,time-entries,expenses,approvals,activities,working-groups,customers,contracts,org-hierarchy,exports,...}` — 13 functional pages, no stubs; they are `current state` evidence for jobs, not the contract's structure.

### Established Patterns
- Docs-only phases in this milestone author `docs/design/*.md` with a locked pattern: dual-audience Purpose with authority stack, Changelog, RFC 2119 selectively, Do/don't, Pointers, "Not in this file" fence, `15-UI-SPEC` note, Gaps.
- Contract files cite live code by path rather than copying values; they never claim frozen components exist.
- Phase work commits on the phase branch with `docs(<phase>):` messages.

### Integration Points
- New docs: `docs/design/workflows/*.md` (files absent today — verified). No code changes in this phase.
- `docs/design/INDEX.md` line 7 is the only map entry to revisit (reserved → live link once the files land, mirroring the CHROME.md flip in `d94f35a`).
- Downstream: Phase 20 `COMPOSITION.md` consumes the job ids; post-Phase-20 job-cluster phases consume jobs + gaps.

</code_context>

<specifics>
## Specific Ideas

- The four lifecycle groups (Record · Organize · Approve · Report) are the **chrome** model; role contracts describe jobs and may tag a job's lifecycle group for Phase 20's benefit, but must not re-model navigation.
- Employee-side "what waits on me" is a composition rule, not a second page (SURF-06: both Today shapes are composition).
- Org tree / people composition is named as a **manager + HR job**, explicitly not Admin/Settings (SC2, MGR-01, HR-01, COMP-01 SC2).
- Every job that a role cannot do must say **who does it instead** (e.g. employee cannot close a period, cannot read org-wide reports; finance cannot close a period; HR never approves) — the negative space is part of the contract.
- API interactions are cited with method + path so an implementation phase can find the surface without re-deriving it.

</specifics>

<deferred>
## Deferred Ideas

- **Assembled role×surface matrix** — Phase 20 `COMPOSITION.md` (COMP-01); Phase 19 declares only per-role rows.
- **Admin/Settings** — out of v0.2.1 entirely (SC2); no contract may open it.
- **Customer portal / external ticket intake** — out of scope (D-E); `customer.md` records the fence only.
- **ABAC** — the lifecycle groups seed it; ABAC itself is not in v0.2.1 (RBAC today).
- **POLS-01..11 quality bar** — JOB-01 / job-cluster territory once a cluster touches a surface.
- **Page-action predicate (CHROME GAP A)** — later job-cluster work; contracts only say which actions a role may take.
- **Availability window writes** — no route exists (reads only, `direction_repository.go:763-817`); employee declaration and HR curation are future implementation, recorded as gaps.
- **`work_permit_expires_at`** — column exists (012:41-42), no Go reader; HR validity job records it as a gap.
- **Sketch sessions** — SKETCH-01 is Phase 20.
- **D-E provenance** — the deciding note is off-disk; a future ADR-hygiene task may re-record it.

</deferred>

---

*Phase: 19-Role contracts*
*Context gathered: 2026-09-26*
