---
status: complete
phase: 17-design-language-contract
source: [17-01-SUMMARY.md, 17-02-SUMMARY.md]
started: 2026-08-28T11:20:00Z
updated: 2026-08-28T11:25:00Z
---

## Current Test
<!-- OVERWRITE each test - shows where we are -->

[testing complete]

## Tests

### 1. Design-language contract (LANGUAGE.md)
expected: |
  Open docs/design/LANGUAGE.md. It should read as the source-of-truth design-language
  contract covering Type, Color, Density, Motion, and Status vocabularies in the D-17-20
  section order (Purpose → Changelog → Foundations → Overlay → Light/dark → Do/don't →
  Pointers → Not-in-this-file → 15-UI-SPEC note → Gaps). It should cite web/src/index.css
  and web/components.json by repo-root path only — no oklch/hex/px value tables — and state
  that LANGUAGE.md wins on vocabulary/meaning while CSS wins on values.
result: pass

### 2. Design-documentation map (INDEX.md)
expected: |
  Open docs/design/INDEX.md. It should be a simple path map: a markdown link to LANGUAGE.md
  plus three reserved paths (docs/design/CHROME.md for Phase 18, docs/design/workflows/ for
  Phase 19, docs/design/COMPOSITION.md for Phase 20), each with a one-line purpose. No other
  design content beyond the map.
result: pass

### 3. AGENTS.md design gate
expected: |
  Open repo-root AGENTS.md. Line 2 (immediately after the title) should be exactly the design
  gate: "Before any `web/src` change to UI, tokens, components, copy, or layout, open
  `docs/design/INDEX.md` first. Backend-only work skips this. No other design text belongs in
  `AGENTS.md`." It should appear exactly once, with the existing OpenWiki/architecture content
  below unchanged and no extra design prose leaked in.
result: pass

## Summary

total: 3
passed: 3
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
