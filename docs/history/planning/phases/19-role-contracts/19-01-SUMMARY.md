---
phase: 19-role-contracts
plan: 01
subsystem: docs
tags: [docs, role-contracts, workflows, jobs, gap-register]

# Dependency graph
requires:
  - phase: 18-chrome-contract
    provides: docs/design/CHROME.md (shell/nav authority the contracts must not override)
  - phase: 17-design-language-contract
    provides: docs/design/LANGUAGE.md (language authority) + the doc pattern
provides:
  - docs/design/workflows/README.md — index, stable job-id scheme (E-/M-/F-/H-/C-), shared gap register G1..G12
  - docs/design/workflows/employee.md — 11 jobs (E-01..E-11)
  - docs/design/workflows/manager.md — 10 jobs (M-01..M-10)
  - docs/design/workflows/finance.md — 8 jobs (F-01..F-08)
  - docs/design/workflows/hr.md — 5 jobs (H-01..H-05)
  - docs/design/workflows/customer.md — 3 jobs (C-01..C-03), concludes "no app surface"
affects: [phase 20 composition map (cites job ids), job-cluster phases inserted after Phase 20 (consume jobs + gaps)]

# Actuals (chars/4 over the realized diff)
actuals:
  tokens: 21894
  tasks: 6
  commits: 1

tech-stack:
  added: []
  patterns: [role contract = job record (id · actor · trigger · steps · surfaces · API interactions · current state · backend authority); jobs are workflows, never routes; gaps recorded per-contract + in a shared register]

key-files:
  created:
    - docs/design/workflows/README.md
    - docs/design/workflows/employee.md
    - docs/design/workflows/manager.md
    - docs/design/workflows/finance.md
    - docs/design/workflows/hr.md
    - docs/design/workflows/customer.md
  modified: []

key-decisions:
  - "D-19-01/02/03: one file per role, full job record, stable namespaced job ids"
  - "D-19-04/05: one membership role per session; WG manager/delegate is a derived dimension; manager.md owns the stage-1 job"
  - "D-19-06/07: HR first-class with the Go gap named; customer concludes no app surface (serving jobs cross-referenced to finance/manager)"
  - "D-19-08: per-role scope rows here, the assembled role×surface matrix in Phase 20"
  - "D-19-09: gaps twice (per-contract + register) while contracts still state target authorization"
  - "D-19-10: FIN-01 cutoffs = snapshot read; the close action stays M-05 (no gap)"
  - "D-19-11/12: POLS-* stays JOB-01 territory; LANGUAGE.md/CHROME.md authority honored"
  - "G12 added after audit: the WG manager/delegate hat reads its group's stage-1 queue but cannot approve/reject — the routes admit only the manager|finance JWT role and the stage-1 transition accepts only manager"

patterns-established:
  - "Role contracts cite real methods/paths (verified against cmd/server/main.go) and mark absent write routes explicitly with a gap id"
  - "Every contract states the role's negative space and who owns the job instead"

requirements-completed: [EMP-01, MGR-01, FIN-01, HR-01, CUST-01]

coverage:
  - id: D1
    description: "Five role contracts + index authored under docs/design/workflows/, each job carrying the seven-field record; customer.md concludes 'no app surface'"
    requirement: EMP-01, MGR-01, FIN-01, HR-01, CUST-01
    verification:
      - kind: other
        ref: "per-file job-id counts 11/10/8/5/3; no '### .*/' headings; no frozen-component existence claims; no oklch/hex/px; changelog + authority lines present"
        status: pass
      - kind: other
        ref: "every `METHOD /path` cited in the six files exists in cmd/server/main.go wiring (94/94 after the /customers shorthand fix)"
        status: pass
    human_judgment: false

metrics:
  duration: ~35min
  completed: 2026-09-26
status: complete
---

# Phase 19 Plan 01: Role contracts — Summary

**Authored `docs/design/workflows/{README,employee,manager,finance,hr,customer}.md`: 38 job records across five roles, each citing real API interactions and current state, with a 12-entry gap register.**

## Accomplishments

- **README.md** — index, the stable job-id scheme (`E-`/`M-`/`F-`/`H-`/`C-`, never renumbered; Phase 20 cites ids), the authority statement (LANGUAGE.md > CHROME.md > workflows), the scope-row note (matrix is Phase 20), and gap register **G1..G12**.
- **employee.md** (E-01..E-11) — capture time/expenses, Today, follow submissions, tickets, activity proposal, own coverage, absence declaration (`absent` write route → G8), claim queued WG work, self-planning, own exports.
- **manager.md** (M-01..M-10) — stage-1 approval, direction plan, capacity read, coverage allocation, period close + snapshot, org units, working groups, ticket triage, planning policy, team exports. Org-tree work is stated as a manager job, explicitly not Admin/Settings (MGR-01/SC2).
- **finance.md** (F-01..F-08) — stage-2 approval, commercial record, customer records, activities, money-label reads, membership governance, org-wide reporting, cutoffs read side.
- **hr.md** (H-01..H-05) — availability curation, employment validity, people composition shared with the manager, capacity/payroll view, and never-an-approver. Headline gap: `hr` is not a valid Go role (G5).
- **customer.md** (C-01..C-03) — **no app surface**; the internal jobs that serve customers cross-referenced (`F-03`, `F-04`, `M-04`, `F-07`); reopen conditions; D-E provenance gap (G11).

## Deviations from Plan

1. **Audit-driven erratum (major).** The first draft asserted that the WG manager/delegate hat executes the stage-1 approval. Verified false: `POST /time-entries/{id}/approve` and the expense equivalent return 403 for every JWT role but `manager`/`finance` (`internal/adapters/primary/http/time_entry.go:351`, `expense.go:374`), and the service's stage-1 branch accepts only `manager` (`internal/core/services/time_entry/time_entry.go:193`). The hat grants **queue visibility** (synthesized `wg_manager` actor) but not the action; the client already hides the button. Contract text corrected in `manager.md`, `employee.md`, `finance.md`; recorded as **G12**.
2. **Wording corrections from the same audit** — entry/expense editability is `draft|submitted|rejected` (delete: draft only); `/customers` shorthand expanded to the four real paths; `LANGUAGE.md` no longer credited with the status set (it owns the vocabulary); employee negative space now points at `F-02`/`F-03`; migration line refs corrected to `012:49-50` (CHECK) and `012:42` (permit column).
3. **Job-count deviation:** none — 11/10/8/5/3 exactly as planned.

## Evidence

- Invariant sweep (per file): job-id counts, zero route-path headings, zero frozen-component existence claims, zero value tables, changelog + authority lines present, all 12 gap ids in the register.
- Route sweep: every cited `METHOD /path` resolves in `cmd/server/main.go` — clean after the `/customers` fix.
- Independent conformance audit (reviewer agent) over the same six files: after fixes, the only remaining structural note was the fixed `/customers` shorthand.

## Next Phase Readiness

- Phase 20 (`COMPOSITION.md`) can cite stable job ids and the gap register; the assembled role×surface matrix is its deliverable (D-19-08).
- Job-cluster phases inserted after Phase 20 consume the contracts' `Current state` + `Gaps` as their delta list.
