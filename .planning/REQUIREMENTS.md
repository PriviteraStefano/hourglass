# Requirements: Hourglass v0.2.1

**Defined:** 2026-08-25
**Core Value:** Role-based approval workflows (employee → manager → finance) with hierarchical organization structures, contract/activity management, and export capabilities.

## v0.2.1 Requirements

Contract-first presentation. These requirements define **done** for the contract/map sequence. Job-cluster implementation is in this milestone but **gated**: phases are inserted after the composition map, not pre-drawn as routes.

Phase 15 tokens/components and archived v0.2 presentation leftovers are **inputs**, not this file's requirements. Full leftover list: `.planning/milestones/v0.2-REQUIREMENTS.md`.

### Design language

- [ ] **DL-01**: A design-language contract exists and is the source of truth for type, color, density, motion, and status vocabulary across all subsequent presentation work. Phase 15 tokens and frozen components are inputs to this contract, not a substitute for it.

### Chrome

- [ ] **CHR-01**: A chrome contract exists and is the source of truth for the app shell: frame, navigation, role-scoped chrome, and page anatomy. Admin/Settings chrome is out of scope.

### Role contracts

Each contract names the **jobs** that role performs and the surfaces those jobs need. Jobs are not current routes.

- [ ] **EMP-01**: Employee role contract exists (jobs an employee performs in Hourglass).
- [ ] **MGR-01**: Manager role contract exists (jobs a manager performs, including org-tree work that is composition not Admin).
- [ ] **FIN-01**: Finance role contract exists (jobs finance performs: cutoffs, coverage money-labeling, reporting).
- [ ] **HR-01**: HR role contract exists (jobs HR performs, including org-tree / people composition shared with manager).
- [ ] **CUST-01**: Customer role contract exists. It may conclude **"no app surface"**. A customer portal is out of scope (D-E).

### Composition

- [ ] **COMP-01**: One cross-role composition map exists. It shows how the five role contracts share chrome and surfaces. The org tree belongs here as manager/HR composition, not as Admin/Settings.
- [ ] **SKETCH-01**: After DL, CHR, role contracts, and COMP-01, the sketch-loop contract is reconciled. Amend `.planning/sketches/SKETCH-LOOP-CONTRACT.md` **only if ambiguity remains**. Do not run a sketch session to close leftover UXFD-02. Sketching follows this reconciliation; it does not precede it.

### Job-cluster implementation (gated)

- [ ] **JOB-01**: After contracts + composition + sketch, presentation is implemented by **job cluster**, not by the current route/page tree. Phases for this requirement are **inserted after Phase 20**. Do not recreate cancelled v0.2 route phases (old 17–26: Coverage Surfaces, Today+Tickets, Direction Surfaces, per-page polish).

## Historical inputs (not requirements)

Do not copy these into phases as pages. Use them as job-shaped hints while writing role contracts:

| Archived ID | Hint for |
|-------------|----------|
| TICK-06 | Employee/manager ticket jobs |
| AVAIL-03..05 | Employee absence job; manager/HR capacity job |
| SURF-01..02 | Manager coverage/allocation jobs |
| SURF-03 | Employee own-coverage job |
| SURF-04..05 | Finance/economics jobs |
| SURF-06 | Employee Today job (both shapes are composition, not two pages) |
| SURF-07..08 | Manager/self direction jobs |
| POLS-01..11 | Quality bar for existing surfaces once a job cluster touches them |
| UXFD-02 | Process: SKETCH-01, not a sketch-now debt |

## Out of Scope

| Feature | Reason |
|---------|--------|
| Recreating cancelled v0.2 Phases 17–26 | Those were route/page phases. v0.2.1 is job clusters after contracts. |
| Admin / Settings work | Explicitly out of this milestone |
| Customer-facing ticket portal | Tickets are internal-only (D-E); CUST-01 may conclude no app surface |
| Implementing UI before contracts + composition map | Sequence is locked: DL → chrome → roles → map → (sketch-loop amend iff needed) → sketch → implement |
| Copying SURF-*/POLS-* as live page requirements | Archived v0.2 leftovers are inputs, not scope |
| Expense coverage allocations | Schema-ready only (D-K); `time` only |
| External ticket intake | Future hexagonal port |
| Full budget machinery / V5 analytics / smart allocation | Unchanged from v0.2 out of scope |

## Traceability

Filled during roadmap creation. JOB-01 is intentionally unmapped until Phase 20 completes.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DL-01 | Phase 17 | Pending |
| CHR-01 | Phase 18 | Pending |
| EMP-01 | Phase 19 | Pending |
| MGR-01 | Phase 19 | Pending |
| FIN-01 | Phase 19 | Pending |
| HR-01 | Phase 19 | Pending |
| CUST-01 | Phase 19 | Pending |
| COMP-01 | Phase 20 | Pending |
| SKETCH-01 | Phase 20 | Pending |
| JOB-01 | (insert after Phase 20) | Blocked |

**Coverage:**

- v0.2.1 requirements: 10 total
- Mapped to initial phases: 9
- Gated (insert later): 1 (JOB-01) — by design

---
*Requirements defined: 2026-08-25 at v0.2.1 start*
*Last updated: 2026-08-25 after roadmap creation*
