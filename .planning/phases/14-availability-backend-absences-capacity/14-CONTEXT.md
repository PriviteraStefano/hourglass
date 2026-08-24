# Phase 14: Availability Backend — Absences + Capacity - Context

**Gathered:** 2026-08-08
**Status:** Ready for planning

<domain>
## Phase Boundary

Backend-only phase (no UI). The absence lifecycle works server-side over the shipped `availability_windows` schema (migration 012), plus derived capacity queries and the work-schedule model that backs them:

- **Absence lifecycle** — declare → confirm/reject/withdraw over the shipped `availability_windows` schema; only `declared` windows confirmable; rejects carry a reason; HR curates medical absences with `certificate_ref` + attached certificate document (AVAIL-01, AVAIL-02)
- **Work schedules** — reusable `contract_types` templates (cadence week/month, hours per period, day-hours matrix) + per-employee override on membership; the capacity basis per employment contract
- **Capacity queries** — weekly hours − confirmed absences + workload from submitted+approved entries on the activity subtree, per employee/activity/WG/unit/org (supports AVAIL-04 in Phase 16 and Phase 13's DIR-05)
- **Confirmed-only read path** — Phase 13's direction warning path switches from declared+confirmed to confirmed-only (D-13-29 closure)
- **ADRs** — ADR-P-008 revision (D-1a routing simplification + D-5 boundary change: medical document storage) + BE encoding ADR drafted and recorded

Deliverables are API endpoints + migrations + domain/ports/services/adapters + integration tests. All migrations append-only per ADR-BE-004 with up/down pairs + cycle tests.

</domain>

<decisions>
## Implementation Decisions

### Confirmation routing & approvers
- **D-14-01:** **Unit manager only confirms** — holiday/permit/unavailable are confirmable; the confirm/reject authority is the employee's unit manager resolved via the unit-tree upward walk (`routing.ResolveUnitManager`, BE-014). **ADR-P-008 D-1a's second line (one WG manager) is dropped** — no WG-manager confirmation. This is an ADR revision. — **Reversibility:** costly — D-1a routing is referenced in ADR-P-008 and the payroll/export story; reintroducing a WG confirm line later needs a schema/status-model change.
- **D-14-02:** **Medical is record-only** — no approval step; medical windows are confirmed immediately at declare (notification, not a request, per ADR D-1a).
- **D-14-03:** **HR curates, never confirms** — HR may create windows for anyone, correct medical windows, and set/attach certificate data; confirmation/rejection is exclusively the unit manager's call (ADR-P-008 D-4 "never an approver"). AVAIL-02's "Manager/HR can confirm" reads as manager-confirms + HR-curates.
- **D-14-04:** **Self-confirmation allowed** — if the employee IS the resolved unit manager, they confirm their own window; no upward walk for self-absences (deliberate deviation from entry-approval's self-approval prevention; the manager declaring their own absence knows their own calendar).
- **D-14-05:** **certificate_ref is required at declare for medical** — the employee provides the INPS protocol number; HR may correct it later as curator.

### Medical certificate documents (ADR D-5 boundary change)
- **D-14-06:** **The certificate document (image/pdf) IS stored in Hourglass** — required at declare alongside the ref. The employee uploads the document with the medical window. — **Reversibility:** one-way — ADR-P-008 D-5 explicitly rejected medical document storage citing GDPR special-category data; this phase revises that boundary. Undoing means deleting the attachment table + storage path.
- **D-14-07:** **DB-backed attachment table** — documents stored in PostgreSQL (new table: org_id, entity_type='availability_window', entity_id, content_type, size, storage/bytes), not file storage or object store (nothing in the stack; no S3/minio in compose). Served back only to `hr` + the employee's unit manager (ADR D-1a visibility scope).

### Absence lifecycle & schema gap
- **D-14-08:** **Status vocabulary extended** — migration adds `rejected` + `withdrawn` to the `status` CHECK (schema today: `declared`, `confirmed`) and a `rejection_reason` column. Full vocabulary: `declared → confirmed | rejected | withdrawn`; `rejected` and `withdrawn` are terminal; `medical` auto-confirms at declare.
- **D-14-09:** **Rejected = terminal** — no editing, no re-submission on the same row; the employee creates a NEW window if circumstances changed. Reject requires a reason (mirrors entry-reject-with-reason pattern).
- **D-14-10:** **Withdraw = declared-only, status not delete** — an employee changes an absence window by withdrawing the first one and creating a new one that goes through the normal declare → confirm flow. Withdrawn windows are terminal rows (`status='withdrawn'`, audit-logged), NOT hard-deleted — history stays intact per BE-012.
- **D-14-11:** **Nobody edits windows** — no in-place editing of dates/kind/hours/note by employees or managers (change = withdraw + redeclare). The single carve-out: **HR may edit medical windows directly** (dates + certificate_ref) since medical is record-only and withdrawal never applies to it — HR edits are the medical correction path.
- **D-14-12:** **Full in-tx audit trail** — every window event (declare, withdraw, confirm, reject, HR medical edit, certificate attach) writes an `audit_logs` row (`entity_type='availability_window'`, payload before/after), synchronous inside the mutator transaction (BE-016 house style, mirroring tickets/coverage/direction).

### Overlap rejection semantics (AVAIL-01)
- **D-14-13:** **Active-only, kind-blind overlap** — the overlap guard counts only `declared` + `confirmed` windows of the same user; `withdrawn`/`rejected` are excluded. Kind-blind: a declared holiday overlapping a confirmed medical still rejects.
- **D-14-14:** **Date-range only** — overlap = date-range intersection; the schema stores `hours` but no time-of-day, so same-day partial windows count as overlap (no hours-aware comparison possible).
- **D-14-15:** **Service in-tx check** — the overlap guard runs inside the declare transaction (CR-01 pattern: SELECT the user's active overlapping windows, reject if any, under row lock); no DB EXCLUDE constraint, no btree_gist extension (first-extension-free house style preserved).

### Work schedules & capacity basis
- **D-14-16:** **Work-schedule model: contract_types + membership override** — new `contract_types` table (org_id, name, cadence `week`|`month`, hours_per_period, default day-hours matrix) as the reusable template; `organization_memberships` gains `contract_type_id` + per-employee day-hours override rows (weekday → hours, e.g. Mon 6h instead of the type's 8h). The type answers "reuse a contract type across employees"; the override answers "different days or different hours on the same day for this user" — the user's explicitly named most-complex case.
- **D-14-17:** **Monthly cadence with dynamic days = derived per-day** — monthly hours ÷ working days in the month (working days = the fixed weekday list when fixed, else calendar workdays). No per-day hours stored for dynamic patterns.
- **D-14-18:** **Fallback chain: override → contract_type → org default → 8h × Mon–Fri** — capacity never breaks when nothing is configured; the resolution level is documented in the response. The org default schedule is a `contract_type` flagged as default (planner discretion: flag column vs org_settings reference).
- **D-14-19:** **Workload = Σ submitted+approved entries on the activity subtree** — recursive CTE (Phase 11 terminal-activity semantics), grouped per employee. Only `time` entries with those statuses count.

### Capacity read-model
- **D-14-20:** **One endpoint, scope params** — `GET /availability/capacity?scope=activity|wg|unit|org&scope_id=&period=` mirroring Phase 13 D-13-25's shape; aggregation differs only in employee-universe resolution (WG members / unit subtree members / employees with entries on the activity subtree / org).
- **D-14-21:** **Confirmed-only subtraction, declared advisory** — capacity subtracts only `confirmed` windows; `declared` windows surface as an advisory field in the response, never subtracted. **This closes Phase 13's D-13-29 deferred item: the direction warning read path (DIR-05) switches to confirmed-only in this phase.**
- **D-14-22:** **Employment validity filters capacity** — employees outside their `valid_from`/`valid_until` window are EXCLUDED from capacity responses entirely (parity with Phase 13 D-13-31: "can't plan what can't work").
- **D-14-23:** **Partial-day windows reduce capacity by their `hours`**; full-day windows zero the day (Phase 13 D-13-24 semantics carried over).

### Read visibility & permission gates
- **D-14-24:** **Absence windows are org-wide visible** — "absence concerns everyone in the org" — any org member sees any employee's windows: kind + dates + status, including declared. The privacy carve-out is the medical record: **`certificate_ref` + attached documents visible only to `hr` + the employee's unit manager** (ADR D-1a); the medical *kind label* stays public. Server-side field filtering, not client-side.
- **D-14-25:** **REST endpoints under `/availability`** — POST /availability/windows (declare), POST /availability/windows/{id}/withdraw, POST /availability/windows/{id}/confirm, POST /availability/windows/{id}/reject (reason required), PUT /availability/windows/{id} (HR medical edit), POST /availability/windows/{id}/certificate (attach doc, medical only), GET /availability/windows (org-wide read), GET /availability/capacity, contract-types CRUD. Exact URL details are planner discretion.
- **D-14-26:** **Role gates** — declare → employee (self) or hr (anyone); withdraw → window owner only (declared-only by lifecycle); confirm/reject → resolved unit manager; HR medical edit + certificate doc write → `hr` role only (first consumer of the `hr` role in backend code); capacity read + org-wide windows read → any authenticated org member (with the D-14-24 field filtering).

### Contract-type management
- **D-14-27:** **HR-owned full CRUD this phase** — contract_types create/edit/delete is `hr`-gated; managers read-only. Full CRUD lands now because capacity's fallback chain depends on it.
- **D-14-28:** **Hard delete if unused** — contract_types with no referencing memberships can be deleted (FK blocks otherwise); no soft-deactivate lifecycle.
- **D-14-29:** **Override attached via membership endpoint extension** — `contract_type_id` + day-hours override set through the existing membership/org endpoints (Phase 10 territory), extended this phase.

### the agent's Discretion
- Exact endpoint URL shapes and route registration within the `/availability` REST surface (D-14-25)
- `rejection_reason` column shape (TEXT vs VARCHAR + CHECK) and whether `confirmed_by`/`rejected_by`/timestamps land on the window row or live in audit only
- Certificate document storage details (BYTEA vs chunking; size limits; MIME allowlist) within the DB-backed decision (D-14-07)
- Org default schedule representation (flagged contract_type vs org_settings key) within D-14-18
- Day-hours override storage shape (rows table vs JSONB on membership) within D-14-16
- Windows list read-model filters/pagination; capacity period format (ISO week vs date range)
- Contract-types CRUD endpoints' exact shapes; whether schedule CRUD for the org default uses the same routes
- Test layout for the new availability domain package (follow per-package suite pattern)

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### ADRs (vault decisions)
- `hourglass-vault/decisions/project/ADR-P-008 — Availability & Employment Validity.md` — THE governing ADR: D-1a confirmation routing (REVISED this phase: WG-manager line dropped, medical record-only kept), D-1b one-window-one-wage-code, D-2 validity dates, D-4 HR curator-never-approver, D-5 rejections (REVISED this phase: medical document storage now allowed via DB-backed attachments), D-3 no enforcement coupling (entries during absence are legal data)
- `hourglass-vault/decisions/backend/ADR-BE-012 — Audit Log Writes.md` — `audit_logs` table (migration 017) — the audit channel for every window event (D-14-12)
- `hourglass-vault/decisions/backend/ADR-BE-014 — Approval-Routing Precedence & Activity-Chain Resolution.md` — `routing.ResolveUnitManager` unit-tree walk reused for the absence confirmation authority (D-14-01)
- `hourglass-vault/decisions/backend/ADR-BE-016 — Origins Tickets & Audit Schema Encoding.md` — Schema-encoding house style: CHECK vocabularies, additive migrations, in-tx synchronous audit writes
- `hourglass-vault/decisions/backend/ADR-BE-004 — Database Migrations.md` — Append-only migrations rule with up/down pairs
- New ADRs drafted this phase: **ADR-P-008 revision** (D-1a simplification + D-5 document-storage boundary change) + **BE encoding ADR** (schema: window statuses, rejection_reason, contract_types, membership override, attachment table, overlap guard encoding — per milestone convention "Each backend phase drafts its ADR + BE encoding ADR")

### Prior phase context (locked decisions)
- `.planning/phases/13-direction-backend-the-plan-plane/13-CONTEXT.md` — D-13-24 (daily hours = org setting, default 8; partial-day permits reduce by hours; full absences zero the day), D-13-25 (scope-param read-model shape), D-13-29 (direction warnings read declared+confirmed for now — **Phase 14 restricts to confirmed-only**), D-13-31 (validity-aware surfacing — excluded, not flagged)
- `.planning/phases/12-coverage-backend-the-allocation-loop/12-CONTEXT.md` — CR-01 in-tx re-validation house pattern precedent (FOR UPDATE + re-check inside mutator tx)
- `.planning/phases/11-foundations-schema-origins-tickets-backend/11-CONTEXT.md` — ticket state-machine pattern (matrix + in-tx re-validation), terminal-activity recursive CTE, audit-first BE-012 usage

### Milestone docs
- `.planning/ROADMAP.md` — Phase 14 entry: goal, requirements (AVAIL-01/02), 4 success criteria
- `.planning/REQUIREMENTS.md` — AVAIL-01..05 requirement text (AVAIL-03/04/05 are Phase 16 frontend; backend gates/read-models land here)

### Codebase (read-only context)
- `migrations/012_staffing_schema.up.sql` — `availability_windows` (kinds, status declared/confirmed, partial-day `hours`, `certificate_ref`, note, created_by) + `organization_memberships` validity columns (`valid_from`/`valid_until`/`work_permit_expires_at`) + `hr` role in the role CHECK
- `migrations/017_audit_logs.up.sql` — general audit table for window events
- `internal/core/services/routing/routing.go` — `ResolveUnitManager` (upward unit-tree walk) — the confirm/reject authority resolution
- `internal/core/services/ticket/ticket.go` + `internal/core/domain/ticket/ticket.go` — state-machine + in-tx FOR UPDATE re-validation pattern (CR-01 closure) to mirror for lifecycle/overlap
- `internal/adapters/secondary/postgres/exported_test_helpers.go` — `availability_windows` already in the teardown list (no test changes needed for it)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `internal/core/services/routing/routing.go` — `ResolveUnitManager` (BE-014 upward walk) for the unit-manager confirmation authority
- `audit_logs` table (017) + Phase 11/12/13 audit repo patterns — every window event appends here in-tx
- Ticket service state machine (matrix + in-tx re-validation under FOR UPDATE, CR-01 closure) — the pattern for window lifecycle and overlap races
- Terminal-activity recursive CTE (Phase 11) — reused for the capacity workload aggregation (submitted+approved entries on the subtree)
- `organization_memberships` validity columns (migration 012) — capacity validity filtering input
- `internal/adapters/primary/http/` handler tests + testcontainers per-package suites — test scaffolding for the new availability package

### Established Patterns
- Hexagonal: domain → ports → services → HTTP handlers → postgres repos; services own invariants, DB owns shapes (CHECK constraints)
- Hand-written SQL with pgx, no ORM; migrations append-only with up/down pairs + cycle tests (ADR-BE-004)
- CHECK-enforced vocabularies (house style); status CHECK extension pattern from migration 012's role CHECK (drop + recreate constraint)
- API response envelope `{ data | error }` via `pkg/api/response.go`; sentinel errors in domain, `wrapPGError` in postgres adapters
- Integration tests via testcontainers-go; per-package suites
- Audit-first via BE-012 for governed changes; in-tx synchronous audit writes (BE-016)

### Integration Points
- New: `/availability` routes registered in `cmd/server/main.go` (Go 1.22+ pattern)
- New: migrations for status CHECK extension + `rejection_reason`, `contract_types`, membership override, attachment table (append-only, numbered after 017 — see Phase 11 A8 ordering lesson)
- `organization_memberships` — contract_type_id + override columns/rows (D-14-16); validity columns consumed by capacity
- Phase 13 direction warning path (DIR-05) — switches to confirmed-only windows via this phase's read path (D-14-21)
- `audit_logs` (017) — every window event; first consumer of the `hr` role in backend permission gates
- Phase 16 UI consumes: windows read (org-wide, filtered), capacity read-model, contract-types CRUD

</code_context>

<specifics>
## Specific Ideas

- User on schedule model (verbatim intent): "we should have different capacities, so that we can cover different type of employment contract and then specify which day are covered and for how many hours. A smart move could be having a work week definition where we define: whether it is hours by month or by week, how many hours a day, how many hours a week/month, fixed days or dynamic"
- User on two-tier contract model (verbatim intent): "we need a contract definition and the employee attached to a specific contract type, so that we don't have n contract types to n employees, but we can reuse a contract type and just change days for a specific user. A part-time contract with 24 hours a week may give two employees different days or different hours on the same day (this is the most complex case)"
- User on visibility (verbatim intent): "absence concerns everyone in the org, so we need to show it without limits (only privacy wise)" — resolved as org-wide kind+dates, private certificate_ref + documents
- User on medical certificate (verbatim intent): "employee usually sends the documents images/pdf of his medical leave. That is required for medical leave." — this overrode ADR D-5's no-document-storage boundary
- User on changing windows (verbatim intent): "an employee can change its absence window by withdrawing the first one and creating a new one (that goes under the same flow as a normal one)"

</specifics>

<deferred>
## Deferred Ideas

- **Payroll export** (ADR-P-008 D-1c): confirmed absence windows feed the payroll view in Exports — that's Phase 25 (Exports + People polish); this phase only guarantees confirmed windows are queryable
- **work_permit_expires_at** (migration 012): not consumed by Phase 14 capacity; it stays for Phase 13's warning path and future validity surfacing — do NOT build permit-expiry logic here
- **Block-vs-nag soft policy** (D-X, carried from Phase 13): still UI-decided in Phase 19 — backend never blocks
- **Absence balances/accruals/carry-over** (ADR-P-008 D-5): still rejected — work schedules are a capacity basis, not entitlement counters; do not build balances

</deferred>

---

*Phase: 14-Availability Backend — Absences + Capacity*
*Context gathered: 2026-08-08*
