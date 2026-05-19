# Phase 6: API Audit - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-19
**Phase:** 6-api-audit
**Areas discussed:** Test tooling approach, Bug handling, Known-broken endpoints, Per-API depth, Frontend integration, Auth & authz

---

## Test Tooling Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Go test suite | testify assertions, build tag, CI-ready | ✓ |
| Shell script (curl + jq) | Simple but less structured | |
| Hybrid audit binary | Portable CLI | |
| Live server | Full bootstrap against real server | ✓ |
| httptest.Server | Isolated DB per test (Phase 0 style) | |
| New audit package | `internal/audit/` with build tag | ✓ |
| Co-located with handlers | Next to existing test files | |
| One file per domain | `audit_auth_test.go` etc. | ✓ |
| Single monolithic file | All checks in one file | |
| Seed user login | Login as demo users | ✓ |
| Direct context injection | Skip auth layer | |
| Existing seed data | Use 003_seed_demo.surql | ✓ |
| Fresh test data per test | Self-contained | |
| Integration-gap focused | Not re-testing Phase 0 | ✓ |
| Full re-test | Redundant with Phase 0 | |
| Phase 0 against live server | Different target, same tests | |
| Full bootstrap lifecycle | Docker → schema → server → test → teardown | ✓ |
| Minimal (server only) | Assume infra is running | |
| Dynamic port | Avoid conflicts | ✓ |
| Static port :8080 | Simple but conflicts with dev | |
| Docker Compose | Standard DB startup | ✓ |
| Native SurrealDB | Manual setup | |
| Dedicated namespace | `audit_<ts>` | ✓ |
| Reuse default | Simpler | |
| Read-only + revert | Don't pollute seed data | ✓ |
| No cleanup | Seed can be re-run | |
| Makefile target | `make audit` | ✓ |
| Manual steps | User manages lifecycle | |
| Ad-hoc runs | Not in CI | ✓ |
| CI integration | Part of pipeline | |
| testify/require | Standard assertions | ✓ |
| Standard library only | No external dep | |
| 2 waves (core + auxiliary) | Wave 1: main domains, Wave 2: rest | ✓ |
| Single wave | All at once | |
| 3 waves | Per test layer | |
| Stateful sequence tests | Chain workflow steps | ✓ |
| Isolated stateless tests | Each endpoint standalone | |
| Verify response shapes | Check fields and types | ✓ |
| Status codes only | Lighter check | |
| Test CORS | Verify preflight + headers | ✓ |
| No CORS testing | | |
| Both auth and unauth | Per endpoint | ✓ |
| Authenticated only | | |
| Verify error format | Check { error: ... } envelope | ✓ |
| Skip error format checks | | |
| List params for key endpoints | ?status=, ?org_id= | ✓ |
| No list params | | |
| No performance tests | Functional only | ✓ |
| Basic response time checks | | |

**User's choice:** Go test suite, live server, new audit package, one file per domain, seed user login, existing seed data, integration-gap focused, full bootstrap lifecycle, dynamic port, Docker Compose, dedicated namespace, read-only + revert, Makefile target, ad-hoc runs, testify/require, 2 waves, stateful sequence tests, verify response shapes, test CORS, both auth contexts, verify error format, list params for key endpoints, no performance tests.

---

## Bug Handling

| Option | Description | Selected |
|--------|-------------|----------|
| Log + batch-fix | BUGS.md then fix at end | ✓ |
| Fix immediately | Fix as found, slower | |
| Log + defer | Future phase | |
| BUGS.md | Markdown in phase dir | ✓ |
| GitHub Issues | Structured, assignable | |
| Critical + Major + Minor | Triage classification | ✓ |
| Flat list | No severity | |
| Fix within phase | Wave 3 | ✓ |
| Defer fixes | Separate phase | |
| Audit first, fix after | Discover all bugs then fix | ✓ |
| Fix before audit | Fix then test | |
| Fresh BUGS.md per phase | 06-BUGS.md in audit dir | ✓ |
| Reuse Phase 0 BUGS.md | All bugs in one file | |
| 3 waves | Core + Aux + Fix | ✓ |
| Single fix wave | After both audit waves | |
| Same agent does both | Audit + fix by same party | ✓ |
| Separate auditor/fixer | Different agents | |
| Structured entries | Endpoint/method/expected/actual/severity | ✓ |
| Free-form | No template | |
| Test-first fixes | Red/green per fix | ✓ |
| Fix-first | Fix then update tests | |
| Full re-audit after fixes | Confirm no regressions | ✓ |
| Targeted re-test only | Per-fix verification | |
| Phase-scoped bugs | Only audit-found bugs | ✓ |
| Consolidate with Phase 0 | Single BUGS.md | |

**User's choice:** Log + batch-fix, BUGS.md in phase dir, Critical/Major/Minor, fix within phase as Wave 3, audit first then fix, fresh BUGS.md per phase, 3-wave structure, same agent, structured entries, test-first fixes, full re-audit, phase-scoped bugs.

---

## Known-Broken Endpoints

| Option | Description | Selected |
|--------|-------------|----------|
| Test current state, fix later | Expect 500, update to 200 after fix | ✓ |
| Write correct-expectation tests | Test expects 200, fails until fix | |
| Skip them | Not worth testing | |

**User's choice:** Test current state (expect 500), update to 200 after fix.

---

## Per-API Depth

| Option | Description | Selected |
|--------|-------------|----------|
| Full CRUD | List, Get, Create, Update, Delete + verify | ✓ |
| List + Create only | Minimal | |
| Limited validation tests | 2-3 error cases per endpoint | ✓ |
| Skip (Phase 0 covers) | Already tested in isolation | |
| Verify against seed counts | Exact numbers per entity | ✓ |
| Non-empty check only | Just check array length > 0 | |

**User's choice:** Full CRUD, limited validation tests, verify against seed counts.

---

## Frontend Integration

| Option | Description | Selected |
|--------|-------------|----------|
| Backend only | No frontend checks | ✓ |
| Frontend smoke test | Login as each user, check pages | |

**User's choice:** Backend only.

---

## Auth & Authz

| Option | Description | Selected |
|--------|-------------|----------|
| Log role gaps, fix in batch | BUGS.md then fix | ✓ |
| Note only, no fix | Document concern | |
| Skip authz | Functional only | |
| Test cross-role access | Sampled per domain (write ops) | ✓ |
| Auth-only (401) | Skip role-level testing | |
| Login as each role | Real auth per seed user | ✓ |
| Inject role in context | Faster but not real auth | |
| Per-domain 401 check | One unauthenticated test per domain | ✓ |
| Single 401 test | One global test | |
| Sampled cross-role testing | Most sensitive actions per domain | ✓ |
| Exhaustive cross-role | Every role × every action | |
| Verify seed roles | Check DB has correct role assignments | ✓ |
| Assume seed correct | Skip role verification | |
| Test cookie flow | Login sets cookies, refresh on 401, logout clears | ✓ |
| Skip cookie flow | Just check headers | |
| Test switch-org | POST /auth/switch-organization | ✓ |
| Skip switch-org | | |
| Verify all 6 users login | Each seed user | ✓ |
| Manager only | Single user | |
| Test bootstrap | bootstrap-check + bootstrap | ✓ |
| Skip bootstrap | | |
| Test logout clears cookies | Verify cookie clearing | ✓ |
| Skip logout | | |
| Rate limit check on auth | Quick 429 test | ✓ |
| Skip rate limit | | |

**User's choice:** Log role gaps + fix in batch, cross-role testing sampled per domain (write ops), login as each role, per-domain 401 check, sampled per domain, verify seed roles, test full cookie flow, test switch-org, verify all 6 users, test bootstrap, verify logout clears cookies, quick rate limit check.

---

## Agent's Discretion

- Exact test case selection per domain
- Test file naming within `internal/audit/`
- Makefile recipe details
- Docker Compose service configuration for audit
- Seed data mismatch repair approach
- Tagged cleanup implementation details

## Deferred Ideas

- Frontend smoke test
- Phase 0 BUGS.md consolidation
- Go CLI seed command (cmd/seed)
- CI integration for audit suite
