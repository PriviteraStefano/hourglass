---
phase: 15
slug: ux-foundation-design-tokens-shared-components
# status lifecycle: draft (seeded by plan-phase) → validated (set by validate-phase §6)
# audit-milestone §5.5 distinguishes NOT-VALIDATED (draft) from PARTIAL (validated + nyquist_compliant: false) (#2117)
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-08-12
---

# Phase 15 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | vitest 4.1.10 + @testing-library/react 16.3.2 + jsdom (globals: true) |
| **Config file** | `web/vitest.config.ts` (setup: `./src/lib/__tests__/setup.ts` — jest-dom + matchMedia polyfill) |
| **Quick run command** | `bun run test -- src/components/shared/__tests__/<file>.test.tsx` |
| **Full suite command** | `bun run test` (plus `bun run typecheck`, `bun run lint`, `bun run build` for the phase gate) |
| **Estimated runtime** | ~60 seconds |

---

## Sampling Rate

- **After every task commit:** Run `bun run test -- src/components/shared/__tests__/` (affected component file) + `bun run typecheck`
- **After every plan wave:** Run `bun run test` + `bun run lint` + `bun run build`
- **Before `/gsd-verify-work`:** Full suite must be green + `bun run fmt:check` + human visual check (dark-mode badge smoke + sidebar collapsed mode)
- **Max feedback latency:** 60 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 15-01-01 | 01 | 1 | UXFD-01 | T-15-01 / — | status strings rendered as text, never `dangerouslySetInnerHTML` | unit | `bun run test -- src/components/shared/__tests__/status-badge.test.tsx` | ❌ W0 | ⬜ pending |
| 15-01-02 | 01 | 1 | UXFD-01 | T-15-03 / — | a11y labels on sort/page controls (`aria-label`, `aria-sort`) | unit | `bun run test -- src/components/shared/__tests__/data-table.test.tsx` | ❌ W0 | ⬜ pending |
| 15-01-03 | 01 | 1 | UXFD-01 | T-15-02 / — | required-reason gate disables confirm until non-empty; server remains authoritative | unit | `bun run test -- src/components/shared/__tests__/confirm-dialog.test.tsx` | ❌ W0 | ⬜ pending |
| 15-02-01 | 02 | 2 | UXFD-01 | — | N/A (type-level + doc-only) | typecheck | `bun run typecheck` | ❌ | ⬜ pending |
| 15-02-02 | 02 | 2 | UXFD-02 | — | N/A (process contract doc) | manual/static | file-presence check `.planning/sketches/SKETCH-LOOP-CONTRACT.md` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `web/src/components/shared/__tests__/status-badge.test.tsx` — role mapping, variants, unknown fallback (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/data-table.test.tsx` — sorting/pagination/a11y/empty/loading (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/filter-bar.test.tsx` — active count/reset (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/page-header.test.tsx` — summary strip (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/empty-state.test.tsx` — default icon/slots (REQ UXFD-01)
- [ ] `web/src/components/shared/__tests__/confirm-dialog.test.tsx` — required-reason gate (REQ UXFD-01)
- [ ] `bun add @tanstack/react-table@^8.21.3` — before any DataTable work (framework dep)

*(No new framework config needed — vitest/jsdom/aliases already configured; framework install not required.)*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Dark-mode badge smoke test | UXFD-01 | Color/token rendering is visual; jsdom cannot verify oklch resolution or `.dark` swap | Run `bun run dev`, toggle `.dark`, confirm status badges render the correct role colors; no identical light/dark rendering |
| Sidebar collapsed-mode human visual check | UXFD-01 (SC4) | Human visual verification of collapsed-mode hover/navigation (code fix `54f465a` landed) | Collapse the sidebar, hover/navigate each item, verify no visual regressions |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 60s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
