# Phase 0: testing-foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-18
**Phase:** 0-testing-foundation
**Areas discussed:** Bug discovery strategy, Backend coverage scope, Frontend test approach, Service-layer testing

---

## Bug Discovery Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Tests-that-fail-first | Write tests for expected behavior first — failing tests reveal bugs naturally. Fix both bug + test. | ✓ |
| Manual exploration first | Manually hit all endpoints, review frontend flows, log bugs, then write tests. | |
| Scripted probe suite | Write a lightweight probe script/curl runner that exercises all API endpoints. | |

**User's choice:** Tests-that-fail-first
**Notes:** Also chose a quick-scan pass (curl/endpoint walkthrough) before structured tests to catch obvious issues fast.

| Option | Description | Selected |
|--------|-------------|----------|
| Fix inline | When a test reveals a bug, fix it immediately and add the test. | |
| Log and batch | Log each bug to a tracking list, fix them all at the end. | ✓ |

**User's choice:** Log and batch

| Option | Description | Selected |
|--------|-------------|----------|
| Same phase | Write all tests first, then fix all logged bugs in a wave at the end. | ✓ |
| Separate 'bug fixes' phase | Dedicated follow-up phase for fixes. | |

**User's choice:** Same phase, two waves. "Middle approach" — one phase divided into two major steps.

| Option | Description | Selected |
|--------|-------------|----------|
| BUGS.md per phase dir | Accumulate findings with severity, location, description. Ticked off as fixed. | ✓ |
| GitHub Issues | One issue per bug. More overhead. | |
| Inline TODO/FIXME | Lightest weight but easy to miss. | |

**User's choice:** BUGS.md per phase dir

| Option | Description | Selected |
|--------|-------------|----------|
| Happy-path + validation | Test what should work and basic validation. | |
| Adversarial/break-it approach | Actively probe edge cases: empty payloads, missing auth, boundary values. | ✓ |

**User's choice:** Adversarial/break-it approach

| Option | Description | Selected |
|--------|-------------|----------|
| Approval workflows | Complex state transitions. | |
| Auth & membership | Registration, login, org membership. | |
| Data edge cases | Deleting with relationships, orphans. | |
| Everything equally | No specific suspicions. | ✓ |

**User's choice:** Everything equally — cover all domains systematically.

---

## Backend Coverage Scope

| Option | Description | Selected |
|--------|-------------|----------|
| All domains | Write tests for every handler+service. | ✓ |
| High-risk first | Auth + time entries + contracts first. | |
| Domain by domain | 1-2 domains per wave, full test+fix cycle per domain. | |

**User's choice:** All domains

| Option | Description | Selected |
|--------|-------------|----------|
| Handler + service | Handler integration tests + service unit tests. Skip repositories. | |
| All three layers | Handler + service + repository tests. | ✓ |
| Handler + repository | Handler tests for API + repository tests for data. | |

**User's choice:** All three layers

| Option | Description | Selected |
|--------|-------------|----------|
| Mock with interfaces | Fast, isolated service tests with mocked repos. | |
| Real DB | Use existing GetTestDBWithNamespace pattern. | |
| Test containers | Isolated DB per test run with Testcontainers. | ✓ |

**User's choice:** Test containers if possible, real DB if not.

| Option | Description | Selected |
|--------|-------------|----------|
| No target | Ship what we write. | ✓ |
| Soft 50% | Aim for 50% statement coverage. | |
| Hard 70% | CI-enforced minimum. | |

**User's choice:** No target, ship what we write.

---

## Frontend Test Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Playwright E2E + Vitest/RTL | Deepen E2E + add component/hook unit tests. | ✓ |
| Playwright E2E only | Extend existing Playwright tests. | |
| Vitest/RTL only | Drop E2E, focus on component tests. | |

**User's choice:** Playwright E2E coverage + Vitest/RTL

| Option | Description | Selected |
|--------|-------------|----------|
| API client + hooks + forms | Test api.ts, React Query hooks, Zod schemas. | ✓ |
| All components | Test everything including UI primitives. | |
| Forms + Zod schemas only | Lightest lift — form validation only. | |

**User's choice:** API client + hooks + forms

| Option | Description | Selected |
|--------|-------------|----------|
| All CRUD flows | Create, read, update, delete for each entity. | ✓ |
| Read-only + critical paths | Viewing pages and key write flows. | |
| Select flows per domain | One happy-path per domain. | |

**User's choice:** All CRUD flows

---

## Service-Layer Testing

| Option | Description | Selected |
|--------|-------------|----------|
| State transitions | Approval workflow state machines. Deepest coverage. | ✓ |
| Authorization rules | Role-based access per entity. | |
| Validation logic | Input validation, business rules. | |
| All business rules equally | Every if/else in service methods. | |

**User's choice:** State transitions (approval workflows) as priority.

| Option | Description | Selected |
|--------|-------------|----------|
| Table-driven tests | Go idiom: one function with slice of test cases. | ✓ |
| Function-per-scenario | One function per scenario. More boilerplate. | |

**User's choice:** Table-driven tests

| Option | Description | Selected |
|--------|-------------|----------|
| Shared testdata package | Factory functions for reusable entities. | ✓ |
| Per-file factories | Keep current inline pattern. | |

**User's choice:** Shared testdata package

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, full matrix | Test every role×action×state combination. | ✓ |
| Happy-path state machine only | Valid transitions only. | |

**User's choice:** Yes, full matrix including unauthorized transitions.

| Option | Description | Selected |
|--------|-------------|----------|
| Yes | Domain validation rules tested at service layer. | ✓ |
| No — handler tests cover it | Validation tested indirectly through handlers. | |

**User's choice:** Yes — business rule validation belongs in service tests.

| Option | Description | Selected |
|--------|-------------|----------|
| Co-located | _test.go next to source. Go convention. | ✓ |
| Parallel test tree | Separate test directory. | |

**User's choice:** Co-located

---

## the agent's Discretion

- Table-driven test case selection per domain
- Shared testdata package shape and structure
- Vitest configuration details
- Test containers feasibility assessment during planning

## Deferred Ideas

None — discussion stayed within phase scope
