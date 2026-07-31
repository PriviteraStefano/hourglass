---
phase: 09
slug: activity-ontology
status: verified
threats_open: 0
asvs_level: 1
created: 2026-07-31
---

# Phase 09 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| migrations/ → database | Migration SQL executes with full schema privileges; a bad UPDATE corrupts seed data | migration SQL / schema rows |
| test code → test database | seedWGData writes rows into the testcontainers schema | fixture rows / activity_kinds FK |
| client → /api/activities (PUT/Update) | Untrusted ParentID crosses here; previously accepted any parent including descendants (cycle) and cross-org parents | parent_id / org_id |
| client → /api/activities (POST/Create) | Untrusted ParentID; previously validated exists + same-org but no path check | parent_id / org_id |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-09-06-01 | Tampering / Data integrity | migrations/013 UPDATE predicate | mitigate | Predicate scoped to `kind='task' AND parent_id IS NOT NULL` (013 up:19); down013 reverses same row set (down:11); cycle test asserts distribution engagement 6 / phase 6 / task 0 / internal 1 in all UP states and down013 reversal | closed |
| T-09-06-02 | Tampering | migrations/011, 012 files | mitigate | Applied history immutable per ADR-BE-004; verification greps git diff for 011/012/000 (empty); immutability rule documented in 013 header (up:7-9) | closed |
| T-09-06-03 | Spoofing | applyMigrations helper | accept | Test-only helper; signature change additive (variadic skip); both call sites updated same task — see Accepted Risks Log | closed |
| T-09-07-01 | Tampering | seedWGData re-seed | accept | Test-only fixture with zero runtime surface; FK to activity_kinds satisfied by seeded catalog row (ON CONFLICT DO NOTHING); NOT NULL activity columns populated — see Accepted Risks Log | closed |
| T-09-07-02 | Spoofing | tests referencing production paths | accept | No production symbols modified (commit d94c8ed touches only test file); repo mapping exercised exactly as in production — see Accepted Risks Log | closed |
| T-09-08-01 | Tampering | activities.parent_id via API | mitigate | Server-side enforcement in service `validateParent` (services/activity/activity.go:124-145) shared by Create + Update, walking repo.GetAncestry (activity_repository.go:187-221); cycle → ErrActivityCycle → HTTP 400 (activity_handler.go:291-292) | closed |
| T-09-08-02 | Tampering | Update parent same-org bypass | mitigate | Update mirrors Create's Get + OrgID check (activity.go:98,102,132-134) before delegating to repo; cross-org parent → ErrInvalidRequest → 400 (activity_handler.go:289-290); test-covered (activity_test.go:282-289) | closed |
| T-09-08-03 | DoS | GetAncestry walk per request | accept | Recursive CTE over depth < 6 (SPEC constraint); O(depth) per validated write — negligible — see Accepted Risks Log | closed |
| T-09-08-04 | Tampering | cycle via direct repo use | accept | Service is the only call path the API exposes (cmd/server/main.go:191-196); repo permissive by design, single enforcement point documented in validateParent comment — see Accepted Risks Log | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-09-01 | T-09-06-03 | applyMigrations is a test-only helper (never compiled into the server binary); signature change is additive (variadic skip) and both call sites updated in the same task | plan-time (09-06-PLAN.md) | 2026-07-31 |
| AR-09-02 | T-09-07-01 | seedWGData is a test-only fixture with zero runtime surface — no production code, no secrets, no network exposure; FK to activity_kinds satisfied by seeded catalog row; NOT NULL activity columns fully populated | plan-time (09-07-PLAN.md) | 2026-07-31 |
| AR-09-03 | T-09-07-02 | No production symbols modified; the repo mapping (`SubprojectID` → `activity_id` column) is exercised exactly as in production | plan-time (09-07-PLAN.md) | 2026-07-31 |
| AR-09-04 | T-09-08-03 | GetAncestry recursive CTE over depth < 6 (SPEC constraint) — O(depth) per validated write is negligible; repo is a dumb adapter and the service is the sole enforcement point | plan-time (09-08-PLAN.md) | 2026-07-31 |
| AR-09-05 | T-09-08-04 | Repo stays permissive by design; the service is the only call path the API exposes (verified in main.go route wiring), so direct-repo cycles are unreachable from the HTTP surface | plan-time (09-08-PLAN.md) | 2026-07-31 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-31 | 9 | 9 | 0 | gsd-security-auditor (retroactive verification, register authored at plan time) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-31
