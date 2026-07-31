---
phase: 10
slug: information-architecture-implementation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-31
---

# Phase 10 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution. Frontend-only phase: vitest (unit/component) + Playwright (e2e) + oxlint/tsc gates; Go suite untouched by non-interference.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.x + @testing-library/react 16 (jsdom) · Playwright 1.62 · oxlint --type-aware + tsc -b |
| **Config file** | `web/vitest.config.ts` · `web/playwright.config.ts` · `web/.oxlintrc.json` |
| **Quick run command** | `cd web && bun run test` (vitest run, unit/component only) |
| **Full suite command** | `cd web && bun run lint && bun run typecheck && bun run build && bunx playwright test` |
| **Estimated runtime** | unit ~30s · full (build + e2e) ~5–8 min |

---

## Sampling Rate

- **After every task commit:** Run `cd web && bun run test`
- **After every plan wave:** Run `cd web && bun run lint && bun run typecheck`
- **After waves touching routes/e2e specs:** Run affected Playwright specs (`bunx playwright test e2e/<spec>.spec.ts`)
- **Before `/gsd-verify-work`:** Full suite must be green (lint + typecheck + build + all e2e)
- **Max feedback latency:** 60 seconds (unit loop)

---

## Per-Task Verification Map

*Populated per-plan by the planner; rows below are the phase-level anchors every plan must map into.*

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 10-01-xx | 01 | 1 | P-007-D1 / P-011-D6 | — | `/projects` route gone; `/activities` serves list+detail; no `project`/`subproject` residue in `web/src` types | e2e + typecheck | `bunx playwright test e2e/activities.spec.ts && bun run typecheck` | ❌ W0 (rename) | ⬜ pending |
| 10-02-xx | 02 | 1 | P-011-D1 / D-5 | — | sidebar renders D-1 group order; Review/Economics/Admin hidden per role matrix; disabled placeholders show locked tooltips | unit | `bun run test -- sidebar` | ❌ W0 | ⬜ pending |
| 10-03-xx | 03 | 2 | UI-SPEC Page Shell | — | every authenticated page composes Header+Body; error/pending states render inside Body | unit + e2e smoke | `bun run test && bunx playwright test e2e/error-boundary.spec.ts` | ✅ | ⬜ pending |
| 10-04-xx | 04 | 2 | P-004 / P-011-D2 | — | `/` renders Today (never blank); Waiting-on-you only for stage holders; locked empty states reachable | unit + e2e | `bun run test -- today && bunx playwright test e2e/auth.spec.ts` | ❌ W0 | ⬜ pending |
| 10-05-xx | 05 | 3 | P-011-D3 / BE-014-R1 | — | Approvals hidden from HR/employee-without-stage; tabs match user stage(s); approve/reject round-trips; 403 ≠ empty conflation | unit + e2e | `bun run test -- approvals && bunx playwright test e2e/approvals.spec.ts` | ❌ W0 | ⬜ pending |
| 10-06-xx | 06 | 3 | P-011-D4 | — | WG list/create/edit/members against live API; locked empty state | e2e | `bunx playwright test e2e/working-groups.spec.ts` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `web/src/components/layout/__tests__/sidebar-groups.test.tsx` — role-matrix render stubs
- [ ] `web/e2e/activities.spec.ts` — rename of `projects.spec.ts` (paths, copy, API globs)
- [ ] `web/e2e/approvals.spec.ts` — new spec with seeded manager/finance stages (Phase 8 seed-helper convention)
- [ ] `web/e2e/working-groups.spec.ts` — new spec with WG seed helpers
- [ ] `web/e2e/auth.spec.ts` — landing assertion updated `/time-entries` → Today

*Framework install: not needed — vitest + Playwright already configured.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Visual fidelity of sidebar groups / Today composition vs UI-SPEC | UI-SPEC sign-off dims 1–6 | Pixel judgment, not assertable | Run `bun run dev`, log in as manager, walk Today/Approvals/WG against `10-UI-SPEC.md` focal-point table |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
