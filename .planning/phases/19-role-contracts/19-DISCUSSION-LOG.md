# Phase 19: Role contracts - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-09-26
**Phase:** 19-role-contracts
**Areas discussed:** Contract structure & job record, Role set & HR status, Per-role visibility matrix, Customer conclusion, Existing-surface reconciliation, Backend-truth gaps

---

## Areas selected

| Option | Selected |
|--------|----------|
| Contract structure & job record | ✓ |
| Role set & HR status | ✓ |
| Per-role visibility matrix | ✓ |
| Customer conclusion | ✓ |
| Existing-surface reconciliation | ✓ |
| Backend-truth gaps | ✓ |

**User's choice:** all six — no area deferred out of the discussion.

---

## Contract structure & job record

| Option | Description | Selected |
|--------|-------------|----------|
| One file per role | `employee.md`, `manager.md`, `finance.md`, `hr.md`, `customer.md` under `docs/design/workflows/` | ✓ |
| Uppercase per role | `EMPLOYEE.md`, `MANAGER.md`, … matching LANGUAGE/CHROME casing | |

**User's choice:** One file per role.

| Option | Description | Selected |
|--------|-------------|----------|
| Full record | id · actor · trigger · steps · surfaces · API interactions · current state · backend-authority note | ✓ |
| Surfaces + current state | no API-interaction detail | |
| Job + surfaces only | leanest | |

**User's choice:** Full record. **Notes:** LANGUAGE.md already pins Phase 19 contracts as "a complete workflow spanning pages and API interactions", so the API column is contract-mandated, not garnish.

| Option | Description | Selected |
|--------|-------------|----------|
| Numbered ids (`E-01`, `M-01`, `F-01`, `H-01`, `C-01`) | stable citations for Phase 20 + job-cluster phases | ✓ |
| Titles only | cite by role + title text | |

**User's choice:** Numbered ids.

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm inventory as drafted | five contracts authored against the drafted job lists | ✓ |
| Request changes first | add/remove/rename before authoring | |
| Drafts reviewed before commit | author, then user reviews | |

**User's choice:** Confirm as drafted.

---

## Role set & HR status

**Verified facts presented:** `hr` exists only in the migration 012 `organization_memberships` CHECK; there is no `RoleHR` constant; `models.Role.IsValid()` rejects it, so HR cannot be invited or assigned through the API; the frontend `Role` type and `role-visibility.ts` already model `hr` (never an approver).

| Option | Description | Selected |
|--------|-------------|----------|
| First-class + gap | full `hr.md`; record the Go/IsValid gap as a Gap line | ✓ |
| Shared people jobs | org-tree/people/capacity live under manager.md; HR is a visibility variant | |
| Defer HR | named but no contract content this milestone | |

**User's choice:** First-class + gap.

| Option | Description | Selected |
|--------|-------------|----------|
| One role + derived stages | one membership role per session; WG manager/delegate is a second, independent dimension | ✓ |
| Persona switching | contracts describe explicit hat-switching chrome | |
| Stay silent | contracts name jobs only | |

**User's choice:** One role + derived stages.

| Option | Description | Selected |
|--------|-------------|----------|
| Manager owns the stage-1 job | one job, one owning contract; employee.md notes the WG-member hat | ✓ |
| Employee owns it | derived-stage dimension documented in employee.md | |
| Shared with pointer | documented in both | |

**User's choice:** Manager owns the job.

---

## Per-role visibility matrix

| Option | Description | Selected |
|--------|-------------|----------|
| Per-role rows now, matrix in 20 | each contract declares its own scope rows; COMPOSITION.md assembles the matrix | ✓ |
| Full matrix in 19 | ADR-P-011-style table in `workflows/README.md` | |
| Both places | duplicate tables, drift risk | |

**User's choice:** Per-role rows now, matrix in 20. **Notes:** keeps COMP-01's "how the five contracts share chrome and surfaces" ownership intact.

---

## Customer conclusion

| Option | Description | Selected |
|--------|-------------|----------|
| No app surface | customer is a data subject; the login role claims no jobs | ✓ |
| Read-only report recipient | future: customer receives exports only | |
| Minimal self-service | own ticket status — conflicts with ADR-P-003 D-E (tickets internal-only) | |

**User's choice:** No app surface.

| Option | Description | Selected |
|--------|-------------|----------|
| Statement + serving jobs | decision, rationale, the internal jobs that serve customers, reopen conditions | ✓ |
| Statement only | short fence doc | |
| Statement + future fence | plus a fenced portal scope | |

**User's choice:** Statement + serving jobs.

**Notes:** a provenance gap was surfaced during the discussion — the note that decided **D-E is not on disk**; D-E survives only as secondhand citations inside ADR-P-003 (lines 11, 27, 67, 93). Recorded as a gap, not resolved.

---

## Existing-surface reconciliation & quality bar

| Option | Description | Selected |
|--------|-------------|----------|
| Evidence only | live routes appear as `current state`; POLS-* stays JOB-01 territory | ✓ |
| Conformance note per job | every job touching a live route carries a "meets LANGUAGE/CHROME on touch" line | |

**User's choice:** Evidence only.

---

## Backend-truth gaps

**Verified facts presented:** `RequireRole` unwired; `GET /time-entries` + `GET /expenses` org-wide for every role; units (12) and working groups (8) routes ungated; `POST /invitations` unauthenticated, org id from body; exports default branch org-wide for finance/hr/customer; `POST /organizations/invite-customer` stub; `POST /coverage/close` manager-only; `hr` invalid in Go.

| Option | Description | Selected |
|--------|-------------|----------|
| Per-contract + shared register | Gaps section in the affected contract + `workflows/README.md` register; contracts still state the target authorization | ✓ |
| Shared register only | per-role docs stay clean | |
| Out of scope | implementation phases re-derive | |

**User's choice:** Per-contract + shared register.

| Option | Description | Selected |
|--------|-------------|----------|
| Finance reads, manager closes | "cutoffs" = snapshot read + reporting cadence for finance; close action stays manager; no gap recorded | ✓ |
| Finance owns close (gap) | contract claims finance closes; manager-only gate recorded as a Gap | |
| Shared job | one job, two surfaces | |

**User's choice:** Finance reads, manager closes.

---

## the agent's Discretion

- Section order inside each contract, following the LANGUAGE.md/CHROME.md doc pattern.
- Job granularity inside the confirmed inventory (merge/split allowed while the id namespace stays stable).
- Presentation of the API-interaction column (table vs inline) and RFC 2119 usage per the house rule.

## Deferred Ideas

- Assembled role×surface matrix → Phase 20 `COMPOSITION.md`.
- Admin/Settings → out of v0.2.1 (SC2).
- Customer portal / external ticket intake → out of scope (D-E).
- ABAC → future; lifecycle groups are the taxonomy seed.
- POLS-01..11 quality bar → JOB-01 / job-cluster phases.
- Page-action predicate (CHROME GAP A) → later job-cluster work.
- Availability window **writes** (no route exists) → implementation phase.
- `work_permit_expires_at` (column without a reader) → HR validity implementation.
- Sketch sessions → SKETCH-01, Phase 20.
- D-E provenance (deciding note off-disk) → future ADR hygiene.
