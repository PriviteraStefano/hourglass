---
phase: 17
slug: design-language-contract
status: validated
nyquist_compliant: false
wave_0_complete: true
created: 2026-08-29T15:40:00Z
---

# Phase 17 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | none — documentation-only phase (no code modules) |
| **Config file** | none |
| **Quick run command** | `n/a` |
| **Full suite command** | `n/a` |
| **Estimated runtime** | ~0s |

---

## Sampling Rate

- **After every task commit:** N/A (docs authored, not compiled)
- **After every plan wave:** N/A
- **Before `/gsd-verify-work`:** UAT performed manually (3/3 passed)
- **Max feedback latency:** n/a

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 17-01-01 | 01 | 1 | DL-01 | — | N/A | manual | `grep -c 'oklch(' docs/design/LANGUAGE.md => 0` | ✅ | ✅ pass (UAT #1) |
| 17-01-02 | 01 | 1 | DL-01 | — | N/A | manual | section-order check vs D-17-20 | ✅ | ✅ pass (UAT #1) |
| 17-02-01 | 02 | 1 | DL-01 | — | N/A | manual | `test -f docs/design/INDEX.md && grep -q 'design documentation map'` | ✅ | ✅ pass (UAT #2) |
| 17-02-02 | 02 | 1 | DL-01 | — | N/A | manual | `sed -n '2p' AGENTS.md | grep -q "open \`docs/design/INDEX.md\` first"` | ✅ | ✅ pass (UAT #3) |

*Status: ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

*Existing infrastructure covers all phase requirements — no test framework required for a documentation deliverable.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| LANGUAGE.md reads as authoritative design-language contract (vocabulary/meaning, not values) | DL-01 | Documentation intent requires human judgment on scope/authority framing | Open `docs/design/LANGUAGE.md`; confirm five vocabularies + authority-split wording (D-17-09) |
| INDEX.md is a clean path map with reserved downstream paths | DL-01 | Advisory map; human confirms reserved-path phrasing before Phases 18/19/20 build on it | Open `docs/design/INDEX.md`; confirm LANGUAGE.md link + CHROME/workflows/COMPOSITION reservations |
| AGENTS.md gate is exactly one sentence, no leaked design prose | DL-01 | Pinned copy (D-17-17); human confirms no extra design text leaked in | Open repo-root `AGENTS.md`; confirm line 2 is the gate and appears once |

*All three manual checks were executed via `17-UAT.md` (3/3 passed, no issues) on 2026-08-28.*

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify (N/A — docs phase)
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < n/a
- [x] `status: validated` set in frontmatter

**Approval:** approved 2026-08-29 (manual-only; nyquist_compliant: false is expected for a documentation-only phase)
