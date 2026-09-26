# Phase 18: Chrome contract - Pattern Map

**Mapped:** 2026-08-29
**Files analyzed:** 2 (1 new design doc, 1 reserved-index likely unchanged)
**Analogs found:** 2 / 2

> **Phase shape — DOCS-ONLY.** No code is written in Phase 18. `CHROME.md` is a *design document*, not a code component, so its analog is another **design document** (`LANGUAGE.md`), not a TS/TSX source file. The only "code" referenced below is the **implementation touchpoints the contract specifies for later phases** (`role-visibility.ts`, `sidebar.tsx`, `header.tsx`, `body.tsx`) — these are NOT modified in Phase 18. Do not fabricate code analogs.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `docs/design/CHROME.md` | design contract (doc) | documentation / transform (spec → prose) | `docs/design/LANGUAGE.md` (the Phase 17 design-language contract) | exact (same doc-pattern family) |
| `docs/design/INDEX.md` | design doc map (doc) | documentation (path registry) | `17-02-PLAN.md` Task 1 (`#A495` lines 56-86) INDEX authoring pattern | role-match (likely NO edit needed) |

**Why no code analogs:** CONTEXT SC4 + success criterion 4 forbid UI implementation, sketches, and route work. Both files are markdown under `docs/design/`. The CHROME author copies the *structural and stylistic pattern* of `LANGUAGE.md`, then fills the body with the four chrome sections (Frame / Navigation / Role-scoped chrome / Page anatomy) instead of the five vocabularies.

## Pattern Assignments

### `docs/design/CHROME.md` (design contract document, docs-only)

**Analog:** `docs/design/LANGUAGE.md` (142 lines, read in full this session — `#7FE3`).
**Index-update counterpart (if needed):** `17-02-PLAN.md` Task 1 (lines 56-86).

CHROME.md MUST mirror the established Phase 17 doc shape exactly. Extracted patterns to copy:

#### 1. Document skeleton / section order (mirror `LANGUAGE.md` + D-17-20 via RESEARCH `#E0B8` lines 236)

Title → Purpose → Changelog → **contract body (Foundations → for CHROME: Frame / Navigation / Role-scoped chrome / Page anatomy)** → Do/don't → Pointers → Not-in-this-file → input note → Gaps.

- `LANGUAGE.md` section headings in order: `# Hourglass Design Language` (1) → `## Purpose` (3) → `## Changelog` (12) → `## Foundations` (16) → `## Overlay` (78) → `## Light / dark` (82) → `## Do / don't` (86) → `## Pointers` (98) → `## Not in this file` (103) → `## 15-UI-SPEC note` (115) → `## Gaps` (132).
- For CHROME the "Foundations" block becomes **Frame / Navigation / Role-scoped chrome / Page anatomy** (RESEARCH `#E0B8` line 236). The `## Gaps` block becomes the **deferred** items: middle `Organize` feature mapping + page-action predicate extension + per-role visibility matrix (all Phase 19/20).

#### 2. Dual-audience + RFC 2119 Purpose (mirror `LANGUAGE.md` lines 3-10)

```markdown
## Purpose

This file is the source of truth for the Hourglass app shell — the frame, navigation,
role-scoped chrome, and page anatomy — across all later presentation work.

- **Authority split:** `LANGUAGE.md` wins on type, color, density, motion, and status
  vocabulary (D-17-09). This file adds surface, layout, copy, and composition only and
  `MUST NOT` override `LANGUAGE.md`.
- **RFC 2119 selectively:** `MUST` / `MUST NOT` for hard conformance, `SHOULD` for defaults,
  rationale in prose (D-17-06).
- **No invented values:** Do not invent CSS tokens or copy oklch/px/rem tables. Point at
  the live value stores by repo-root path.
```

Note the `LANGUAGE.md` Purpose carries the **authority stack (D-17-09)** inline (lines 7-9) and the **amendment rule (D-17-10)** — CHROME.md MUST restate the authority stack (it is downstream of LANGUAGE.md) but MUST NOT claim amendment rights over LANGUAGE.md.

#### 3. Authority-stack statement (copy `LANGUAGE.md` lines 7-9 verbatim in spirit)

```markdown
- **Authority stack (D-17-09):** `LANGUAGE.md` wins on type, color, density, motion, and
  status vocabulary. This file (and later `workflows/`, `COMPOSITION.md`, `UI-SPEC.md`
  files) may only add surface, layout, copy, or composition. They `MUST NOT` override
  `LANGUAGE.md`. If a chrome surface needs a language change, amend `LANGUAGE.md` first.
```

#### 4. Changelog (copy `LANGUAGE.md` lines 12-14 format — compact, dated, phase-keyed, no semver)

```markdown
## Changelog

- 2026-08-29 · Phase 18 · Added chrome contract
```

First entry exactly: `2026-08-29 · Phase 18 · Added chrome contract` (D-17-21 analog).

#### 5. "Not in this file" fence (copy `LANGUAGE.md` lines 103-113 shape)

```markdown
## Not in this file

This file is the chrome/shell contract only. Explicitly out of scope here:

- No type/color/density/motion/status vocabulary (that is `LANGUAGE.md`).
- No component API catalog or primitives inventory.
- No workflow/copy contracts (those are Phase 19).
- No composition map (that is Phase 20, `COMPOSITION.md`).
- No Admin/Settings chrome (SC2 — ADR-P-011 Admin group excluded).
- No token-value tables (oklch/hex/px) — values stay in CSS.
- No screenshots or worked examples.
```

#### 6. Do/don't (Don't-only digest mirroring the foundation `MUST NOT`s — `LANGUAGE.md` lines 86-96)

```markdown
## Do / don't

- Don't override `LANGUAGE.md` type/color/density/motion/status vocabulary.
- Don't claim PageHeader / FilterBar / DataTable / EmptyState / ConfirmDialog exist —
  they are frozen inputs, absent on disk (D-18-11).
- Don't invent a customer app surface (D-18-09 — deferred to Phase 19 CUST-01).
- Don't include Admin/Settings chrome (SC2).
- Don't resolve the per-role visibility matrix here (Phase 19/20, D-18-05/08).
- Don't treat nav-group hiding as authorization (`role-visibility.ts` is UX scoping only).
```
Every Don't MUST already appear as a body `MUST NOT` (D-17-22 analog).

#### 7. Pointers (repo-root citations only, no markdown links to CSS/config — `LANGUAGE.md` lines 98-101)

```markdown
## Pointers

- `web/src/lib/role-visibility.ts` — predicate-driven role-visibility mechanism (seed for nav + page-action gating).
- `web/src/components/layout/sidebar.tsx` — existing collapsible sidebar (regroup target).
- `web/src/components/layout/header.tsx` + `body.tsx` — current per-page header/body placeholders.
- `web/src/index.css` — live tokens for type, color, density, light/dark maps.
- `web/components.json` — shadcn kit identity (`base-mira` / olive).
```

#### 8. Input note + frozen-component table (copy `LANGUAGE.md` lines 115-130 pattern)

```markdown
## 15-UI-SPEC note

`.planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` is
untouched historical input and is **not** authority. If it conflicts with `LANGUAGE.md`
on vocabulary, `LANGUAGE.md` wins.

Phase 15 frozen components are listed here as *inputs* with their live status, not
silently reused:

| Input | Live tree (2026-08-29) |
|-------|------------------------|
| StatusBadge | Present at `web/src/components/shared/status-badge.tsx` — raw Tailwind, not role tokens (known debt) |
| PageHeader | Absent |
| FilterBar | Absent |
| DataTable | Absent |
| EmptyState | Absent |
| ConfirmDialog | Absent |

`entries-table.tsx` and `entries-filters.tsx` are present but are **not** the frozen set.
```

#### 9. Gaps (copy `LANGUAGE.md` lines 132-142 one-line-per-item form — here the deferred items)

```markdown
## Gaps

- `Organize` middle mapping — Activities, Working Groups, Org/Units, Availability, Contracts, Customers, Direction, Coverage → deferred to Phase 20 composition delta.
- `page-action` predicate — `role-visibility.ts` gates nav only today; the page-action enable/disable predicate is a target contract, built in a later job-cluster phase.
- `per-role visibility matrix` — carried from ADR-P-011 D-5 as input; authoritative reconciliation is Phase 19 (role contracts) + Phase 20 (composition map).
- `customer surface` — CUST-01 (Phase 19) may conclude "no app surface"; chrome defers to that decision (D-18-09).
```

#### Concrete body content the CHROME author should fill from CONTEXT/RESEARCH

- **Frame** (D-18-06/07): collapsible left `Sidebar` + per-page `PageHeader` (top bar) + content `Body`. Identity chrome (OrgSwitcher, ProfileMenu, ThemeToggle) lives in the sidebar — no separate global identity bar. PageHeader Row 1 = feature title + primary action; Row 2 = tabs + FilterBar.
- **Navigation** (D-18-01..04): revise ADR-P-011 D-1 (do NOT adopt). Four lifecycle groups — `Record` (Time/Expenses/Tickets) · `Organize` (structure) · `Approve` (Approvals) · `Report` (Exports) — plus `Today` landing at `/` outside the groups. Admin/Settings excluded (SC2). Adoption posture: keeps ADR-P-011 D-2/D-3/D-5/D-6, drops Admin group.
- **Role-scoped chrome** (D-18-05/08/09): single predicate-driven mechanism on `role-visibility.ts` governing **both** nav show/hide and page-action enable/disable. Customer deferred. Carry ADR-P-011 D-5 visibility matrix as *input*, not resolved here.
- **Page anatomy** (D-18-10/11): shell rules only — max content width, vertical scroll, `EmptyState` when no data, `ConfirmDialog` for destructive actions. List/detail/master-detail layouts are Phase 19.

### `docs/design/INDEX.md` (design doc map — likely NO change)

**Analog:** `17-02-PLAN.md` Task 1 (`#A495` lines 56-86) INDEX authoring pattern.

RESEARCH `#E0B8` lines 222-227 (verified `docs/design/INDEX.md#43AB` lines 5-7) confirm the reserved line is already correct and complete:

```markdown
- `docs/design/CHROME.md` (reserved — Phase 18) — app-shell / chrome contract (frame, navigation, role-scoped chrome, page anatomy).
```

Per CONTEXT write-set note #2 and RESEARCH, **Phase 18 most likely needs NO edit to INDEX.md.** If CHROME.md's scope wording must be adjusted, edit only that one reserved line, preserving style: existing paths as markdown links, reserved paths as backtick + `(reserved — Phase N)`, one-line purpose, no stack/MUST NOTs/Gaps (17-02-PLAN.md lines 64-73, D-17-13/14).

## Shared Patterns

### Authority stack (downstream doc must defer to LANGUAGE.md)
**Source:** `docs/design/LANGUAGE.md` lines 7-9 (D-17-09).
**Apply to:** Every section of `CHROME.md` — it may add surface/layout/copy/composition only; MUST NOT redefine type/color/density/motion/status vocabulary.
```markdown
LANGUAGE.md wins on type, color, density, motion, and status vocabulary over every later
presentation doc. Later docs (CHROME.md, workflows/, COMPOSITION.md) and later GSD
UI-SPEC.md files may only add surface, layout, copy, or composition. They MUST NOT override.
```

### RFC 2119 selective conformance
**Source:** `LANGUAGE.md` line 5 + RESEARCH `#E0B8` line 233 (D-17-06).
**Apply to:** All hard-conformance statements in CHROME.md (`MUST`/`MUST NOT` for rules, `SHOULD` for defaults, prose rationale).

### Frozen-component input listing (present/absent, never claimed to exist)
**Source:** `LANGUAGE.md` lines 119-130 (D-17-45/122).
**Apply to:** CHROME.md's input note — PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog absent; StatusBadge present (raw-Tailwind debt); `entries-*` present but not the frozen set.

### ADR-P-011 "revise, not adopt" posture
**Source:** RESEARCH `#E0B8` lines 246-295 (ADR-P-011 D-1..D-6).
**Apply to:** CHROME.md Navigation — replaces the 7+1 job-language groups with 4 lifecycle groups + `Today`; keeps D-2/D-3/D-5/D-6; drops Admin.

## Implementation Touchpoints (NOT modified in Phase 18)

These are the **later (Phase 20+) implementation targets** the CHROME contract specifies. They are referenced by repo-root path only; Phase 18 does NOT edit them. Listed so the planner knows exactly where the contract lands:

| Touchpoint | Verified state | What CHROME specifies for it |
|------------|----------------|------------------------------|
| `web/src/lib/role-visibility.ts` | Present (`#E0B8` lines 142-180). Pure predicates: `deriveApprovalStages`, `isReviewVisible`, `isEconomicsVisible`, `isAdminVisible`. Gates **nav groups only** today. | Extend with a page-action predicate (shared registry) so the mechanism governs nav **and** page-actions (D-18-08, GAP A). Keep the "UX scoping only — backend stays authoritative" warning (source lines 3-9). |
| `web/src/components/layout/sidebar.tsx` | Present (`#E0B8` lines 63-135). `navStructure` uses ADR-P-011 groups (Today/Track/Work/People/Economics/Review/Reports/Admin); role filter is a hardcoded inline `switch`. `sidebar-groups.test.tsx` asserts the current groups. | Regroup `navStructure` into the four lifecycle groups + `Today` landing (D-18-02/03). A later job-cluster phase updates `sidebar-groups.test.tsx` accordingly. |
| `web/src/components/layout/header.tsx` + `body.tsx` | Present (`#E0B8` lines 82-91). `Header` = 48px band (title only), `Body` = content wrapper. Used by 9 authenticated pages (tabs-in-body). | CHROME targets the two-row `PageHeader` (title+action / tabs+FilterBar) as the spec; the frozen `PageHeader` is absent, so this is a target, not current state (D-18-07/11). |
| `web/src/components/app/org-switcher.tsx`, `profile-menu.tsx` | Present (`#E0B8` lines 73-77). Identity chrome, sidebar-resident. | Documented as-is (D-18-06); no change. |
| `web/src/components/shared/` (PageHeader/FilterBar/DataTable/EmptyState/ConfirmDialog) | **Absent** (`#E0B8` lines 186-208). Only `status-badge.tsx`, `entries-table.tsx`, `entries-filters.tsx` present. | Referenced as intended implementers only; CHROME MUST NOT claim they exist; job-cluster build lands post-Phase 20 (D-18-11). |

## No Analog Found

None. Both files have a direct analog: `CHROME.md` → `LANGUAGE.md` (doc-pattern family), `INDEX.md` → `17-02-PLAN.md` Task 1 (path-map authoring). No code component is an appropriate analog because Phase 18 writes no code.

## Metadata

**Analog search scope:** `docs/design/` (LANGUAGE.md, INDEX.md), `.planning/phases/17-design-language-contract/` (17-CONTEXT/17-01-PLAN/17-02-PLAN), `web/src/lib/role-visibility.ts`, `web/src/components/layout/` (read-only confirmation in RESEARCH).
**Files scanned:** 9 (LANGUAGE.md, INDEX.md, 18-CONTEXT, 18-RESEARCH, 17-CONTEXT, 17-01-PLAN, 17-02-PLAN, role-visibility.ts, sidebar/header/body, org-switcher, profile-menu, shared/).
**Pattern extraction date:** 2026-08-29
**Phase type:** DOCS-ONLY design contract — no code analogs; CHROME.md mirrors LANGUAGE.md doc structure, INDEX.md most likely unchanged.

---

## PATTERN MAPPING COMPLETE

**Phase:** 18 - chrome-contract
**Files classified:** 2
**Analogs found:** 2 / 2

### Coverage
- Files with exact analog: 1 (`CHROME.md` → `LANGUAGE.md`)
- Files with role-match analog: 1 (`INDEX.md` → 17-02-PLAN.md Task 1; likely no edit)
- Files with no analog: 0

### Key Patterns Identified
- CHROME.md is a design *document*; its analog is the LANGUAGE.md doc pattern (section order, dual-audience, RFC 2119, authority stack, changelog, Not-in-this-file fence, Do/don't, Pointers, input note, Gaps) — NOT a code component.
- Authority stack (D-17-09): CHROME.md adds surface/layout/copy/composition only; MUST NOT override LANGUAGE.md vocabulary.
- ADR-P-011 is revised, not adopted: 4 lifecycle groups + `Today` landing, Admin/Settings excluded.
- Frozen components are contract inputs (absent on disk) — listed as present/absent, never claimed to exist.
- `role-visibility.ts` / `sidebar.tsx` / `header.tsx` are LATER (Phase 20+) implementation targets the contract specifies; NOT modified in Phase 18.

### File Created
`/Users/stefanoprivitera/Projects/hourglass/.planning/phases/18-chrome-contract/18-PATTERNS.md`

### Ready for Planning
Pattern mapping complete. Planner can reference the LANGUAGE.md doc pattern (sections 1-9 above) when authoring `docs/design/CHROME.md` in PLAN.md, and the 17-02 INDEX pattern if an INDEX.md edit is required (most likely none).
