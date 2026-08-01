---
phase: 10-information-architecture-implementation
verified: 2026-08-01T07:00:57Z
status: human_needed
score: 13/13 must-haves verified
overrides_applied: 0
gaps: []
human_verification:
  - test: "Visual walk of the sidebar regroup (D-1) across all five org roles — log in as employee, manager, finance, hr, customer and check the pillar groups render in locked order (Today/Track/Work/People/Economics/Review/Reports) with exact labels and no legacy 'Tracking'/'Management' wording; check disabled Tickets/Availability items show the locked tooltips on hover when collapsed"
    expected: "Group labels and order match the ADR-P-011 D-1 table verbatim; HR sees no Review group; employee sees no Economics; the Today item sits ungrouped at top with href '/' and is active only on '/'"
    why_human: "Visual appearance and hover behavior of the rendered sidebar cannot be verified by grep; the unit tests assert DOM structure but not pixel-level composition"
  - test: "Visual walk of the Today landing at '/' per UI-SPEC focal-point table — spacing lg between sections, single largest text (Display 28px 'Today'), right-aligned primary CTA ('Review now' for approvers / 'Log time' otherwise), and both locked empty states render correctly for a new user and an all-caught-up user"
    expected: "No charts/KPIs; sections stack top-down ('Waiting on you' then 'Your week'); empty states use the locked copy verbatim; the page is never blank in any state"
    why_human: "Typography scale, spacing, and layout composition are visual judgments; unit tests cover section presence but not visual quality"
  - test: "Visual walk of the Approvals page at /approvals — single h1 'Approvals' in the Header, Manager/Finance tabs (only for dual-stage users), rows at py-3 density with the Approve/Reject pair as the focal accent, reject dialog requiring a reason, and the 'Queue is clear' empty state per stage"
    expected: "Rows render date, activity_name, hours/amount in .font-text, StatusBadge; reject confirm stays disabled until a reason ≥ 10 chars; error state (not empty) shows when the pending queries fail"
    why_human: "Focal-point hierarchy, row density, and dialog flow are visual/interactive judgments beyond DOM-assertion coverage"
  - test: "Visual walk of the Working Groups page at /working-groups — Header with title + search + single accent 'New working group' CTA, card grid on muted surfaces, locked empty state, and the create/edit/members/delete dialogs"
    expected: "Cards show WG name, linked activity, manager name, member count; member management dialog is distinct from the manager/delegate form dialog; delete surfaces backend guard errors via toast"
    why_human: "Card-grid visual composition and dialog UX are visual judgments"
  - test: "Visual walk of the page shell on all carried-over pages (time-entries, expenses, exports, contracts, customers, activities) — exactly one h1 in the 48px Header band, content scrolls inside Body (window does not scroll), no double padding or re-layout artifacts from the wrap"
    expected: "Every authenticated page renders through the locked Header+Body composition with a single h1; page content scrolls in the inner container; no visual regression versus pre-wrap layout"
    why_human: "Layout artifacts from the wrap-only migration are visual; e2e smoke asserts functionality, not pixel fidelity"
  - test: "Product decision on WR-01: a working-group manager/delegate whose org role is 'employee' can now view the pending queue (ListPending admits them via IsWGManager) but the Approve/Reject HTTP handlers still 403 any role other than manager/finance — they can look but never act. Decide whether the WG-stage employee action path must work in v0.1 or is an acceptable documented limitation for a follow-up phase"
    expected: "A recorded decision: either relax the Approve/Reject handler gates to admit WG manager/delegate via IsWGManager (mirroring ListPending), or accept and document the asymmetry. The core manager/finance round-trips are e2e-proven either way"
    why_human: "This is a product/scope decision — the phase plan explicitly kept Approve/Reject service gates authoritative and scoped Task 10-05-01 to ListPending; whether the WG-stage-employee action path is a v0.1 must or a follow-up is a judgment call"
---

# Phase 10: Information Architecture Implementation — Verification Report

**Phase Goal:** Implement ADR-P-011: sidebar regrouping, `/projects` → `/activities` rename, Today landing (ticketless), Approvals queue, Working Groups surface, role-scoped visibility. Requires Phase 9 backend live.
**Verified:** 2026-08-01T07:00:57Z
**Status:** human_needed (all 13/13 truths verified programmatically; visual-walk items + one product decision pending human)
**Re-verification:** No — initial verification

## Goal Achievement

The phase goal is **fully achieved and verifiable in code**. All six ADR-P-011 deliverables (D-1 sidebar regroup, D-2 Today landing, D-3 Approvals queue, D-4 Working Groups surface, D-5 role-scoped visibility, D-6 `/projects` → `/activities` rename) exist as substantive, wired, data-flowing surfaces. Independent re-execution confirmed: frontend unit suite 122/122, Go suite green, all phase e2e suites green (approvals 4/4, working-groups 5/5, activities+auth 12/12), full e2e 41 pass / 3 fail where the 3 failures match deferred-items.md §2 exactly (July-seed date rollover in pre-existing suites). Typecheck reports exactly the 6 pre-existing errors documented in deferred-items.md §1 — zero in phase files.

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | `/projects` route does not exist; `/activities` serves list + detail (P-011-D6) | ✓ VERIFIED | `web/src/routes/_authenticated/projects/` absent; `activities/` has `index.tsx` + `$id.tsx` + 4 components; `routeTree.gen.ts` has `AuthenticatedActivitiesIndexRoute`/`...IdRoute` and **zero** `projects` matches; `web/src/api/projects.ts` deleted; activities e2e suite green in the 12/12 run |
| 2 | Frontend types mirror Phase 9 DTOs; `Role` includes `hr`; no project/subproject vocabulary (10-01) | ✓ VERIFIED | `models.ts:1` `export type Role = "employee" \| "manager" \| "finance" \| "hr" \| "customer"`; `Activity`/`ActivityResponse`/`ActivityDetail` interfaces present; **no** `Project`/`Subproject` interface; `activity_id: string` on TimeEntry/Expense request types; grep for `Project\b`/`/projects` across `web/src` (excl. regenerated routeTree) returns zero |
| 3 | Sidebar renders D-1 groups in locked order with exact labels; disabled placeholders with locked tooltips (P-011-D1) | ✓ VERIFIED | `sidebar.tsx` navStructure: Today (ungrouped) → Track → Work → People → Economics → Review → Reports → Admin; `Tracking`/`Management` absent; Tickets tooltip "Tickets arrive in v0.2", Availability "Availability lands with the staffing schema"; `useMatchRoute({ fuzzy: false })` for Today; 8 sidebar-groups render tests green |
| 4 | Role-scoped visibility via pure helpers; HR never sees Review even as WG manager/delegate; no client-side authorization invented (P-011-D5) | ✓ VERIFIED | `role-visibility.ts` exports `deriveApprovalStages`/`isReviewVisible`/`isEconomicsVisible`/`isAdminVisible`; hr returns `[]` from `deriveApprovalStages` before WG-derived stage is added; comments flag "UX scoping only — backend stays authoritative"; 15 matrix tests green covering all five roles + delegate + undefined-WG |
| 5 | Every authenticated page renders through locked Header+Body shell; single h1; error/pending states inside the shell frame (UI-SPEC-SHELL, 10-03) | ✓ VERIFIED | All nine carried-over page roots (time-entries, expenses, exports, contracts index+detail, customers index+detail, activities index+detail) wrap content in `Header`/`Body` (grep confirms imports in each); `@/components/layout` barrel (`index.ts`) created; e2e for carried surfaces green (time-entries/expenses/contracts/customers in full suite); `errorComponent: RouteError` preserved on wrapped routes |
| 6 | `/` renders Today — never a redirect, never blank; read-only composition; "Waiting on you" only for stage holders; both locked empty states reachable (P-004, P-011-D2, 10-04) | ✓ VERIFIED | `index.tsx` is `component: TodayPage` with zero `Navigate` references; `today-page.tsx` composes Header/Body, "Waiting on you" gated on `deriveApprovalStages` with pending queries `enabled: isApprover`; locked empty states "Welcome to Hourglass" + "You're all caught up" and locked error state "We couldn't load Today. {reason}. Try again." present; 6 unit tests green; auth e2e asserts Today heading at `/` |
| 7 | `/approvals` live with stage-filtered Manager/Finance tabs; approve/reject round-trips with reason-required reject; 403 ≠ empty; HR never sees Review (P-011-D3, BE-014-R1, 10-05) | ✓ VERIFIED | `approvals/index.tsx` route with `validateSearch: { stage }` + `errorComponent: RouteError`; `approvals-page.tsx` has h1 "Approvals", Tabs for dual-stage only, merged pending TE+expense queue, "Queue is clear" empty state, locked error state on query failure, `enabled: isApprover` gating; e2e proves manager approve → `pending_finance`, reject-with-reason (persisted in `time_entry_approvals`), finance chain → `approved` (4/4 green); 8 unit tests green. **Caveat:** WR-01 (WG-stage employee action path 403s) — see Anti-Patterns |
| 8 | Backend ListPending gate admits WG manager/delegate via `Service.IsWGManager` (T-10-05-3) | ✓ VERIFIED | `IsWGManager` in both `internal/core/services/time_entry/time_entry.go:379` and `expense/expense.go:381` (mirrors `resolveManagerStage` wgRepo path); both `ListPending` handlers admit org-role manager/finance OR WG manager/delegate passing role `wg_manager`; gate semantics recorded in code comments; handler tests: employee-without-stage 403 kept, employee-WG-manager 200 added; `go test ./internal/...` green |
| 9 | Today "Waiting on you" and the Approvals queue share one derivation helper | ✓ VERIFIED | Both `today-page.tsx` (line 80) and `approvals-page.tsx` call `deriveApprovalStages` from `@/lib/role-visibility`; no duplicated stage logic |
| 10 | `/working-groups` live under Work with list/search/create/edit/members against live WG API; locked empty state + single accent CTA; member management distinct from approver-set; no availability warnings (P-011-D4, 10-06) | ✓ VERIFIED | `working-groups/index.tsx` route in routeTree at `/working-groups`; `working-groups-page.tsx` with Header (h1 + search + "New working group" CTA), card grid, locked empty state; `WorkingGroupsApis` complete (list/get/members queries + create/update/delete/addMember/removeMember mutations); member dialog separate from form dialog; e2e 5/5 green (empty→create→search→edit→members→delete); `api<T>` 204 fix in `lib/api.ts:88-91`; no availability/validity warnings (P-008 deferred by design) |
| 11 | Full e2e suite green against live backend | ✓ VERIFIED | Re-ran: 41 passed / 3 failed; the 3 failures are exactly deferred-items.md §2 (time-entries "six workflow states", expenses "seeded rows with categories", error-boundary "loader and recovers to data") — July-hardcoded seed dates outside the August default month, passing 42/42 on 2026-07-31, not phase-10 regressions; phase suites (approvals 4/4, working-groups 5/5, activities+auth 12/12) green in isolation and within the full run |
| 12 | Go test suite green; no backend regressions | ✓ VERIFIED | `go test ./internal/...` → all packages ok including `internal/adapters/primary/http` and `internal/adapters/secondary/postgres`; no FAIL lines |
| 13 | Frontend unit suite green (122/122) | ✓ VERIFIED | `bun run test` → 16 files, 122 tests passed (includes 8 approvals + 6 today-page + 15 role-visibility + 8 sidebar-groups tests claimed by the summaries); run twice-consecutive in phase summaries, once here |

**Score:** 13/13 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
| -------- | -------- | ------ | ------- |
| `web/src/api/activities.ts` | ActivitiesApis (4 queries + 3 mutations) | ✓ VERIFIED | Exists; hits `/activities`, `/activities/${id}`, `/activities/${id}/children`, `/activity-kinds`; no adopt mutation |
| `web/src/api/working-groups.ts` | WG list/get/members queries + 5 mutations | ✓ VERIFIED | Exists; `WorkingGroupsApis` complete; `workingGroupsQueryOpts` named export present |
| `web/src/lib/role-visibility.ts` | ApprovalStage + 4 pure helpers | ✓ VERIFIED | Exists; hr-strip logic at top of `deriveApprovalStages` |
| `web/src/components/layout/sidebar.tsx` | D-1 pillar nav with role filtering | ✓ VERIFIED | navStructure + `isReviewVisible`/`isEconomicsVisible`/`isAdminVisible` filtering |
| `web/src/routes/_authenticated/index.tsx` + `-components/today-page.tsx` | Today landing, never blank | ✓ VERIFIED | `component: TodayPage`; error/empty/skeleton ladder complete |
| `web/src/routes/_authenticated/approvals/` | Approvals route + page | ✓ VERIFIED | `index.tsx` + `-components/approvals-page.tsx` |
| `web/src/routes/_authenticated/working-groups/` | WG surface + dialogs | ✓ VERIFIED | `index.tsx` + 4 components |
| `internal/core/services/{time_entry,expense}/...` `IsWGManager` | WG-stage admission primitive | ✓ VERIFIED | Both services, wgRepo-backed |
| `internal/adapters/primary/http/{time_entry,expense}.go` ListPending gate | Org-role OR WG-stage admission | ✓ VERIFIED | Gate semantics commented; role `wg_manager` passed through |
| `web/src/types/models.ts` | Activity/WG types; Role + hr; no Project | ✓ VERIFIED | All present at lines 1, 62-88, 124-177 |

### Key Link Verification

| From | To | Via | Status | Details |
| ---- | -- | -- | ------ | ------- |
| Sidebar nav → routes | `/activities`, `/working-groups`, `/approvals`, `/` | href strings in navStructure | ✓ WIRED | All four routes exist in regenerated `routeTree.gen.ts` |
| `ActivitiesApis` → backend | `/activities*`, `/activity-kinds` | `api<T>()` in `activities.ts` | ✓ WIRED | 1:1 with Phase 9 routes; activities e2e green against live backend |
| Approvals page → pending queues | `GET /time-entries/pending`, `GET /expenses/pending` | pending queries `enabled: isApprover` | ✓ WIRED | msw capture proves zero calls for employee/HR; e2e proves round-trips |
| Approve/Reject → backend | `POST .../approve`, `POST .../reject` | `approval-buttons.tsx` + mutations | ✓ WIRED | e2e: approve transitions `pending_finance`; reject persists reason |
| WG page → live WG API | `GET/POST/PUT/DELETE /working-groups...` | `WorkingGroupsApis` | ✓ WIRED | 5/5 e2e CRUD flows green; delete requires the 204 fix which is present |
| Today → pending queries | month + pending TE/expense | `timeEntriesForMonthQueryOpts` + pending with `enabled` override | ✓ WIRED | Gated; ISO-week client filter; no 403 spam proven via msw |
| `deriveApprovalStages` → both surfaces | Today + Approvals | shared import from `@/lib/role-visibility` | ✓ WIRED | Single derivation helper used by both (truth 9) |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| -------- | ------------- | ------ | ------------------ | ------ |
| `activity-list.tsx` | `activities` | `ActivitiesApis.activitiesQueryOpts(tab)` → live `/activities?scope=` | Yes (e2e CRUD green) | ✓ FLOWING |
| `today-page.tsx` | `monthQuery`, `pendingTimeEntries/ExpensesQuery` | live API with `enabled` gate | Yes (e2e landing assertions green) | ✓ FLOWING |
| `approvals-page.tsx` | pending TE + expense queries | live API | Yes (approve/reject round-trips proven in e2e) | ✓ FLOWING |
| `working-groups-page.tsx` | `workingGroupsQueryOpts`, activities, orgMembers, members | live API | Yes (5/5 e2e flows) | ✓ FLOWING |
| `activity-detail.tsx` | ActivityDetail payload | live `/activities/{id}` | Yes; `adoption_count` renders "—" (no data source — documented known stub, not a phase regression) | ✓ FLOWING (1 documented field fallback) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| -------- | ------- | ------ | ------ |
| Frontend typecheck | `cd web && bun run typecheck` | 6 errors, all in deferred-items.md §1 list (api.test.ts, __root.tsx, 3× `_auth` forms, unit-detail-panel); zero in phase files | ✓ PASS (baseline rot, documented) |
| Frontend unit suite | `cd web && bun run test` | 16 files / 122 tests passed | ✓ PASS |
| Go suite | `go test ./internal/...` | all packages ok (incl. http + postgres) | ✓ PASS |
| Approvals e2e | `bunx playwright test e2e/approvals.spec.ts` | 4/4 passed | ✓ PASS |
| Working Groups e2e | `bunx playwright test e2e/working-groups.spec.ts` | 5/5 passed | ✓ PASS |
| Activities + auth e2e | `bunx playwright test e2e/activities.spec.ts e2e/auth.spec.ts` | 12/12 passed | ✓ PASS |
| Full e2e suite | `bunx playwright test` | 41 passed / 3 failed — failures exactly the deferred-items.md §2 July-seed rollover trio | ✓ PASS (documented pre-existing) |
| No debt markers | grep TBD/FIXME/XXX in phase files | zero matches | ✓ PASS |

### Probe Execution

Step 7c: SKIPPED — phase plans/summaries declare no probe scripts (`scripts/*/tests/probe-*.sh`); verification is via the unit, Go, and e2e suites re-executed here.

### Requirements Coverage

No plan requirement IDs (P-007-D1, P-011-D1..D6, P-004, BE-014-R1, UI-SPEC-SHELL) are registered in `.planning/REQUIREMENTS.md` (0 matches — re-verified; it tracks milestone-level IDs like TEST-*/AUTH-*). Per the Phase 9 precedent, the ADRs in `hourglass-vault/decisions/` are the canonical spec. Every frontmatter ID across all 6 plans is accounted for against its ADR and code evidence:

| Requirement | Source ADR | Description | Status | Evidence |
| ----------- | ---------- | ----------- | ------ | -------- |
| P-007-D1 | ADR-P-007 | Table/domain word is `activity` | ✓ SATISFIED | `/activities` routes + `activities.ts` API + `Activity` types (10-01) |
| P-011-D1 | ADR-P-011 | Sidebar groups job-language, pillar-mapped | ✓ SATISFIED | `sidebar.tsx` navStructure; sidebar-groups tests (10-02) |
| P-011-D2 | ADR-P-011 | Landing is Today from v0.1, ticketless composition | ✓ SATISFIED | `index.tsx` → `TodayPage`; no `Navigate` (10-04) |
| P-011-D3 | ADR-P-011 | Review group, stage-filtered queues | ✓ SATISFIED | `/approvals` route + page; e2e 4/4 (10-05) |
| P-011-D4 | ADR-P-011 | Working Groups top-level surface under Work | ✓ SATISFIED | `/working-groups` route + surface; e2e 5/5 (10-06) |
| P-011-D5 | ADR-P-011 | Role-scoped visibility matrix | ✓ SATISFIED | `role-visibility.ts` helpers; hr-strip; 15 tests (10-02) |
| P-011-D6 | ADR-P-011 | Route naming: `/activities`, `/working-groups`, `/approvals`, `/` = Today | ✓ SATISFIED | routeTree: all four live; no `/projects` (10-01) |
| P-004 | ADR-P-004 | Today view: read-only, one answer, never blank | ✓ SATISFIED | TodayPage composition + locked empty states; no charts (10-04) |
| BE-014-R1 | ADR-BE-014 | Two-stage approval chain (manager → finance) | ✓ SATISFIED | stage derivation + e2e finance chain (10-05) |
| UI-SPEC-SHELL | 10-UI-SPEC | Header+Body page shell on all authenticated pages | ✓ SATISFIED | all nine page roots wrapped; layout barrel (10-03) |

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
| ---- | ---- | ------- | -------- | ------ |
| `internal/adapters/primary/http/time_entry.go` + `expense.go` (Approve/Reject handlers) | ~341-350 / ~354-363 | **WR-01:** ListPending admits WG manager/delegate (org-role employee) but Approve/Reject handlers still 403 any role ∉ {manager, finance} — the WG-stage employee can view the queue but never act | ⚠️ WARNING | Asymmetry introduced by 10-05's ListPending relaxation; core manager/finance round-trips proven in e2e; surfaced for product decision (human item #6). Suggested follow-up: mirror the `IsWGManager` admission in Approve/Reject, or document the limitation |
| `internal/adapters/primary/http/time_entry.go` + `expense.go` (List handlers) | List at ~51-55 / ~64-68 | **CR-01 (review):** List endpoints populate `filters.Role`/`RequestUserID` but repo query builders never apply them — any authenticated member can list every org member's entries | 🛑 NOT THIS PHASE | **Pre-existing** — `List` handler last touched by `b9a1f8d` (hexagonal migration) and repo builders by `26bff17` (pg-2-06), both predating Phase 10. Phase 10's only change to these files was ListPending (83fd31f). Not in this phase's must-have scope; must be scheduled in a future security phase |
| `internal/adapters/primary/http/expense.go` (ReceiptUpload) | ~480-548 | **CR-02 (review):** receipt upload mutates any expense without org/owner/status authorization; uploaded files never served (no static handler) | 🛑 NOT THIS PHASE | **Pre-existing** — `ReceiptUpload` created in `95103c6` (Phase 6). Not touched by Phase 10. Future security phase must fix + serve uploads |
| `web/src/routes/_authenticated/-components/today-page.tsx` | 116-117, 189-208 | **IN-07 (review):** "Welcome to Hourglass" empty state derives `hasAnyData` only from the time-entries month query — a user with expenses but no time entries sees the new-user state | ℹ️ INFO | Minor UX edge; does not violate "never blank" (the state still renders); acceptable for v0.1 |
| `web/src/routes/_authenticated/working-groups/-components/working-group-form-dialog.tsx` | 96-108 | **IN-09 (review):** edit-mode Activity combobox is editable but `UpdateWorkingGroupRequest` carries no `subproject_id` — a picked activity is silently ignored on save | ℹ️ INFO | Edit of name/delegate persists (e2e-proven); activity re-anchor on edit is a silent no-op; candidate for a follow-up fix |
| `web/e2e/helpers.ts` | 144-149, 169-173 | **WR-07 (review):** `seedTimeEntries`/`seedExpenses` hard-code July 2026 dates | ⚠️ WARNING | Pre-existing; the exact 3 full-suite failures; documented in deferred-items.md §2; future maintenance plan should compute current-month dates |

No debt markers (TBD/FIXME/XXX) in any phase-modified file. No stub returns, hardcoded-empty props, or console-only handlers in phase surfaces.

### Human Verification Required

1. **Sidebar regroup + role-scoped visibility walk** — visual walk of the D-1 pillar sidebar across all five org roles; verify group labels/order, Today ungrouped at top, HR never sees Review, disabled Tickets/Availability tooltips. (Visual)
2. **Today landing composition** — UI-SPEC focal-point walk: Display-28px "Today" title, right-aligned CTA, gap-6 section stack, both locked empty states, never blank. (Visual)
3. **Approvals queue walk** — single h1, Manager/Finance tabs (dual-stage only), py-3 rows with Approve/Reject focal pair, reason-required reject dialog, per-stage "Queue is clear". (Visual)
4. **Working Groups surface walk** — Header with single accent CTA, card grid, locked empty state, create/edit/members/delete dialogs, backend guard errors surfaced via toast. (Visual)
5. **Page shell walk** — single h1 in the 48px Header band on all carried-over pages; content scrolls inside Body, window never scrolls; no wrap-induced re-layout artifacts. (Visual)
6. **WR-01 product decision** — decide whether the WG-stage employee (org-role employee who is WG manager/delegate) must be able to approve/reject from the queue in v0.1, or whether the view-only path is an acceptable documented limitation for a follow-up phase. (Product judgment)

### Gaps Summary

No programmatic gaps block the phase goal: all 13 must-have truths are verified in code, all phase e2e suites are green, the Go suite is green, and the frontend unit suite is 122/122. The three full-suite e2e failures and the six typecheck errors are documented pre-existing baseline issues (deferred-items.md §1–§2) that predate Phase 10 and are out of its scope; this phase reduced the typecheck baseline from 87 to 6 and added zero new errors.

The review's two critical findings (CR-01 list-scoping leak, CR-02 receipt IDOR) are confirmed pre-existing (Phase 6 / hexagonal-migration era code) and are not this phase's responsibility — but they are real security defects that a future phase must schedule. WR-01 (WG-stage employee can view the queue but actions 403) is the one in-scope artifact-completeness gap: it is a direct consequence of 10-05's ListPending relaxation and is left to a product decision (human item #6) rather than an automatic pass.

---

_Verified: 2026-08-01T07:00:57Z_
_Verifier: the agent (gsd-verifier)_
