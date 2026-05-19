# Phase 5: MVP Consolidation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-19
**Phase:** 5-mvp-consolidation
**Areas discussed:** Seed entity scope, Seed mechanism, Demo scenario design, Consolidation scope

---

## Seed Entity Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Org model only | Just org + users + units + memberships | |
| Full flow with data | Projects, contracts, customers, WGs, TEs, expenses | |
| Org + projects | Org + users + units + memberships + projects + contracts + one customer | ✓ |

**User's choice:** Org + projects — core entities

| Option | Description | Selected |
|--------|-------------|----------|
| Small — 1 contract, 2 projects | Minimal setup | |
| Medium — 3 contracts, 6 projects | Realistic but not overwhelming | ✓ |
| Large — full consulting firm | 6+ contracts, many projects | |

**User's choice:** Medium — 3 contracts, 6 projects

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — seed one customer | Makes customer relationships visible | ✓ |
| No customers | Simpler, less realistic | |

**User's choice:** Yes — seed one customer

| Option | Description | Selected |
|--------|-------------|----------|
| Manager + employees | 1 admin + 3 employees | |
| Admin only | Single admin | |
| Full role spectrum | Manager, finance, employees per department | ✓ |

**User's choice:** Full role spectrum

| Option | Description | Selected |
|--------|-------------|----------|
| Fixed demo credentials | Pre-defined login | ✓ |
| Bootstrap + auto-seed | User registers | |

**User's choice:** Fixed demo credentials

---

## Seed Mechanism

| Option | Description | Selected |
|--------|-------------|----------|
| Single monolithic seed file | 003_seed_demo.surql | |
| Multiple domain files | Split by domain | |
| Go CLI seed command | cmd/seed tool (future) | ✓ |

**User's choice:** Go CLI seed command deferred to future end-of-milestone phase. SurQL file for now.
**Notes:** "we need to be compatible with the current auth implementation, ideally the seeding is done via API, I know many things are broken, so we might need to start from the .surql then move to API if they are working correctly"

| Option | Description | Selected |
|--------|-------------|----------|
| All UUIDs | Consistent UUID format throughout | ✓ |
| Keep mixed pattern | Short strings for units, UUIDs for users | |

**User's choice:** All UUIDs throughout

| Option | Description | Selected |
|--------|-------------|----------|
| After Phase 0 testing | Build CLI when APIs verified | |
| Build CLI in this phase | Fix API issues along the way | |
| Keep SurQL forever | No CLI migration | |

**User's choice:** At end of this milestone, a simpler ending phase for the CLI seed command

| Option | Description | Selected |
|--------|-------------|----------|
| Idempotent — safe to re-run | IF NOT EXISTS / OR REPLACE | ✓ |
| Replace — not idempotent | Clean seed, errors on re-run | |

**User's choice:** Idempotent

| Option | Description | Selected |
|--------|-------------|----------|
| Deprecate — keep as reference | Rename to .deprecated | ✓ |
| Delete | Remove entirely | |
| Keep both | Both seeds coexist | |

**User's choice:** Deprecate old seed (rename to 002_seed_tcg.deprecated.surql)

| Option | Description | Selected |
|--------|-------------|----------|
| 003_seed_demo.surql | Clear naming, alphabetical ordering | ✓ |
| 002_seed_demo.surql | Reuse 002 slot | |

**User's choice:** 003_seed_demo.surql

| Option | Description | Selected |
|--------|-------------|----------|
| Keep separate | Bootstrap ≠ seed | ✓ |
| Bootstrap includes seeding | Single call creates everything | |

**User's choice:** Keep separate

---

## Demo Scenario Design

| Option | Description | Selected |
|--------|-------------|----------|
| Consulting company (TCG) | Keep Tech Consulting Group theme | ✓ |
| Generic — Demo Corp | No theme | |
| New theme | Create different company | |

**User's choice:** Keep TCG theme

| Option | Description | Selected |
|--------|-------------|----------|
| 6 users | 2 managers + 1 finance + 3 employees | ✓ |
| Extend from 8 | Convert TCG users to proper roles | |
| Minimal — 3 users | 1 admin + 1 manager + 1 employee | |

**User's choice:** 6 users

| Option | Description | Selected |
|--------|-------------|----------|
| Manager as primary | Sees all pages, approvals | ✓ |
| Finance persona | Expenses, reporting | |
| Employee persona | Day-to-day time tracking | |

**User's choice:** Manager as primary demo persona

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — seed sample entries | 3-5 per employee, past week | ✓ |
| No — empty | User adds fresh during demo | |

**User's choice:** Seed sample time entries

| Option | Description | Selected |
|--------|-------------|----------|
| Seed sample expenses | 1-2 per employee | ✓ |
| No expenses | User adds during demo | |
| Defer to future | Skip for now | |

**User's choice:** Seed sample expenses

---

## Consolidation Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Seed-focused | Structure cleanup as side effect | ✓ |
| Deep consolidation | Fix broken APIs, clean up routes | |
| Major restructuring | Reorg layers, align patterns | |

**User's choice:** Seed-focused

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal — rename file | Keep old seed but skip loading | ✓ |
| Full cleanup | Remove from tests and codebase too | |

**User's choice:** Minimal — rename only

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — manual verification | Run seed, log in, check pages | ✓ |
| No — trust the seed | Fix issues as they come | |

**User's choice:** Manual verification pass

---

## the agent's Discretion

- Exact bcrypt hashes for demo passwords
- Seed data structure and file formatting
- Time entry and expense sample values
- Verification checklist format

## Deferred Ideas

- Go CLI seed command (`cmd/seed`) — end of milestone phase
- Deep consolidation / API fixes for broken endpoints — separate phase or Phase 0 bugs
- Major codebase restructuring — out of scope
