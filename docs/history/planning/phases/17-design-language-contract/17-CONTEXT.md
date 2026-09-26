# Phase 17: Design-language contract - Context

**Gathered:** 2026-08-26
**Status:** Ready for planning

<domain>
## Phase Boundary

Docs-only design-language contract for v0.2.1. This phase delivers DL-01: `docs/design/LANGUAGE.md` is the source of truth for type, color, density, motion, and status vocabulary. Phase 15 tokens, `15-UI-SPEC.md`, and frozen-component intent are **inputs**, not a substitute.

**Write set (only these three files):**

1. `docs/design/INDEX.md`
2. `docs/design/LANGUAGE.md`
3. One fixed UI gate immediately after the title in `AGENTS.md`

**Success criteria that must be TRUE:**

1. A design-language contract exists and is referenced as source of truth for subsequent presentation work (DL-01)
2. The contract covers type, color, density, motion, and status vocabulary — not only the Phase 15 token dump
3. Phase 15 frozen components (PageHeader, FilterBar, DataTable, StatusBadge, EmptyState, ConfirmDialog) are listed as inputs, with gaps called out rather than silently reused
4. No UI implementation, no sketches, no route work

**Out of this phase:** CSS/token implementation, component rewrites, `docs/design/workflows/`, chrome, role/workflow/screen contracts, composition map, sketch session, job-cluster implementation, Admin/Settings, customer portal.

</domain>

<decisions>
## Implementation Decisions

### Contract artifact
- **D-17-01:** New product-level design docs live under `docs/design/`, separate from GSD phase artifacts. `LANGUAGE.md` is architected product language, not a Phase 15 restoration.
- **D-17-02:** One dual-audience file. No agent-only section.
- **D-17-03:** Foundations only (type, color, density, motion, status vocabulary) plus a concise shadcn/base-mira overlay. No component API catalog, no workflow/copy, no page layouts, no screenshots, no worked examples, no token tables, no oklch/hex, no component anatomy.
- **D-17-04:** Roles and MUST/SHOULD rules in the file; concrete values pointed at live CSS/config. CSS wins on values. `LANGUAGE.md` wins on meaning and usage. Do not invent CSS tokens in the doc. Do not duplicate oklch/hex tables.
- **D-17-05:** Amend in place with a compact changelog. No version fork. No separate Amend section.
- **D-17-06:** RFC 2119 selectively: `MUST` / `MUST NOT` for hard conformance; `SHOULD` for defaults; rationale in prose.
- **D-17-07:** A required role without a CSS token is a language gap: name its meaning, record it in Gaps, do not add CSS in Phase 17.
- **D-17-08:** Gaps are one line per role/rule: English name plus what is missing in `index.css`. No owner phase. No proposed token name. No speculative `--status-*` / `--duration-*` identifiers. Example form: `success — no status token in index.css.` Changelog records add/close of Gaps when later phases amend.
- **D-17-09:** Authority stack: `LANGUAGE.md` wins on type, color, density, motion, and status vocabulary. Later design docs (`CHROME.md`, workflow contracts, `COMPOSITION.md`) and later GSD `UI-SPEC.md` files may only add surface, layout, copy, or composition. They MUST NOT override `LANGUAGE.md`. If a later surface needs a language change, amend `LANGUAGE.md` first.
- **D-17-10:** Only later GSD docs/planning phases may amend `LANGUAGE.md`. Implementation / job-cluster work MUST NOT amend it as a side effect. If blocked, stop and perform a docs/planning amendment first. One `MUST NOT` is sufficient; no Amend procedure section.
- **D-17-11:** `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` stays untouched. Historical input / not-authority. If it conflicts with `LANGUAGE.md` on type, color, density, motion, or status vocabulary, `LANGUAGE.md` wins.
- **D-17-12:** Do not create `docs/design/workflows/` in this phase. Phase 19 creates it when the first workflow contract lands.
- **D-17-13:** `INDEX.md` is a simple path map: title, one sentence (“this is the design documentation map”), then the path list with one-line purposes. No stack, no MUST NOTs, no shadcn, no Gaps, no How to use, no AGENTS/CSS/`15-UI-SPEC` pointers.
- **D-17-14:** Existing files are markdown links. Reserved paths are backtick paths plus `(reserved — Phase N)`, not fake links.
- **D-17-15:** Phase 17 INDEX lists only: `LANGUAGE.md` (link) plus reserved `docs/design/CHROME.md` (Phase 18), `docs/design/workflows/` (Phase 19), `docs/design/COMPOSITION.md` (Phase 20).
- **D-17-16:** Explicit short “Not in this file” fence: no component APIs, chrome/layout, workflow/copy, token tables, or screenshots.
- **D-17-17:** `AGENTS.md` gate is one fixed sentence immediately after the title, then existing OpenWiki content. No other design text in `AGENTS.md`. Exact copy:

  > Before any `web/src` change to UI, tokens, components, copy, or layout, open `docs/design/INDEX.md` first. Backend-only work skips this. No other design text belongs in `AGENTS.md`.

- **D-17-18:** Pointers in `LANGUAGE.md` are repo-root citations only: `web/src/index.css` and `web/components.json`. No markdown links to CSS/config.
- **D-17-19:** Overlay is one short rule: Hourglass overlays shadcn style `base-mira` / olive. MUST use those primitives. MUST NOT invent a parallel visual system or restyle the kit in this file. Cite `web/components.json` + `web/src/index.css`. No component APIs.
- **D-17-20:** Section order: Title → Purpose → changelog → foundations → overlay → do/don’t → pointers → not-in-this-file → 15-UI-SPEC note → Gaps.
- **D-17-21:** Changelog: compact dated entries keyed by GSD phase, latest first, no semver, no Keep-a-Changelog category headings. Form: `2026-08-26 · Phase 17 · Added foundations`.
- **D-17-22:** Do/don’t is a Don’t-only digest of foundation `MUST NOT`s. No Do column. No examples. Every Don’t MUST already appear as a foundation `MUST NOT`. Section may still be titled Do/don’t.
- **D-17-23:** Light/dark: one short MUST — semantic roles are theme-invariant; UI MUST NOT branch on light/dark except for the theme toggle. Per-theme values remain in `index.css`. No duplicate per-mode role lists.
- **D-17-24:** Future contracts are workflow-group oriented, not one file per current route. A “page contract” means a complete workflow group, potentially spanning pages and API interactions. That work is Phase 19, not Phase 17.

### Type
- **D-17-25:** English type roles only, not font-family names. MUST NOT name Inter / Martian Mono / Geist Mono as the language. Point at `web/src/index.css` for families, weights, and sizes.
- **D-17-26:** Live CSS names: `display` (`--display-family` / `--display-weight` / `.font-display`) and `text` (`--text-family` / `--text-weight` / `.font-text`). Plus `ui` for the default Inter / `html { font-sans }` face. Do not invent `mono`.
- **D-17-27:** `ui` is named as meaning (default sans/UI face) and recorded as Gap: `ui — no type token in index.css.` `display` and `text` point at live CSS. Do not pretend a `--ui` token exists.
- **D-17-28:** Do not dump Phase 15’s four-size / two-weight table into `LANGUAGE.md`. Size and weight values stay in CSS / Tailwind. Phase 15 typography is historical input.

### Color
- **D-17-29:** Color names interaction/chrome roles only. Status stays in the Status foundation. MUST NOT treat Color roles as status.
- **D-17-30:** Product-wide shadcn colour jobs only, one line per role (English name + job). `*-foreground` named as the pair, not a second essay. Point at `index.css`. Do not name `--base-*` / `--primary-*` ramps, `sidebar-*`, or `chart-*`.
- **D-17-31:** Color inventory (live in `index.css`; include `*-foreground` only where the pair exists):

  * `background` — page/canvas
  * `foreground` — default text
  * `card` / `card-foreground` — raised surface and its text
  * `popover` / `popover-foreground` — overlay surface and its text
  * `primary` / `primary-foreground` — main action and its text
  * `secondary` / `secondary-foreground` — secondary action/surface and its text
  * `muted` / `muted-foreground` — quiet surface and quiet text
  * `accent` / `accent-foreground` — interaction/navigation (never status) and its text
  * `destructive` — destructive action (never status). No `destructive-foreground` pair in live CSS; do not invent one
  * `border` — strokes
  * `input` — form chrome
  * `ring` — focus ring

- **D-17-32:** `accent` is interaction/navigation, never status. `destructive` is an action role, not domain `danger`. MUST NOT map `danger` to `destructive`.

### Density
- **D-17-33:** Fast-locked: 4px spacing rhythm; compact default; no px tables; CSS owns values.
- **D-17-34:** Density names the rhythm and the compact default. MUST NOT copy the Phase 15 xs…3xl pixel table. Live `index.css` has no density/spacing tokens — record Gap: `density — no density token in index.css.`
- **D-17-35:** `--radius` in CSS is a value, not a density role. Do not catalog radius in `LANGUAGE.md`.

### Motion
- **D-17-36:** Fast-locked: duration and easing roles; reduced-motion MUST; CSS owns values; unbacked roles are Gaps; no animation catalog.
- **D-17-37:** English motion roles: `duration` (motion length) and `easing` (motion curve). MUST NOT catalog animations, transitions, or component motion recipes.
- **D-17-38:** Live `index.css` has no duration/easing tokens. Gaps: `duration — no motion token in index.css.` `easing — no motion token in index.css.`
- **D-17-39:** UI MUST honor `prefers-reduced-motion`. That is a rule, not a token, and not a Gap line. Do not invent a reduced-motion CSS identifier.

### Status vocabulary
- **D-17-40:** Hourglass status roles: `neutral`, `info`, `success`, `warning`, `danger`. Semantic meaning, separate from interaction/chrome tokens.
- **D-17-41:** Full separation. MUST NOT use interaction tokens (`primary`, `accent`, `destructive`, `muted`, …) as status. MUST NOT map `danger` to `destructive`.
- **D-17-42:** All five status roles are Gaps because live `index.css` has no status tokens:

  * `neutral — no status token in index.css.`
  * `info — no status token in index.css.`
  * `success — no status token in index.css.`
  * `warning — no status token in index.css.`
  * `danger — no status token in index.css.`

- **D-17-43:** Do not copy the Phase 15 domain-status → role mapping table into `LANGUAGE.md` (entries, tickets, direction, absences, governance, warnings). That mapping is historical input in `15-UI-SPEC.md` / `15-CONTEXT.md`. Workflow/copy contracts are out of this file. Unknown-status fallback and badge variants are component API, also out of this file.
- **D-17-44:** Live `web/src/components/shared/status-badge.tsx` still uses raw Tailwind palettes (`yellow`/`blue`/`green`/`purple`/`emerald`/`red` with `dark:` variants). Historical implementation debt. Phase 17 MUST NOT modify it.

### Phase 15 frozen components (inputs, not write-set)
- **D-17-45:** List as inputs, with live gaps, rather than silently reused. Do not restore or rewrite them in Phase 17.

  | Input | Live tree (2026-08-26) |
  |-------|------------------------|
  | StatusBadge | Present at `web/src/components/shared/status-badge.tsx` — raw Tailwind, not role tokens |
  | PageHeader | Absent |
  | FilterBar | Absent |
  | DataTable | Absent |
  | EmptyState | Absent |
  | ConfirmDialog | Absent |

  Also present and **not** the frozen set: `entries-table.tsx`, `entries-filters.tsx`. Phase 15 summaries (`15-01-SUMMARY.md`) claim tokens + frozen components landed; live `index.css` and `web/src/components/shared/` do not match those summaries. Planner MUST trust the live tree, not the summaries.

### Planner / executor constraints
- **D-17-46:** Phase 17 MUST NOT modify `web/src/index.css`, `web/components.json`, any `web/src` component, `15-UI-SPEC.md`, or create `docs/design/workflows/`.
- **D-17-47:** Do not implement before contracts and composition map are settled. Do not run a sketch session to close leftover UXFD-02. Do not recreate cancelled v0.2 Phases 17–26.
- **D-17-48:** Feature / screen / workflow contracts are Phase 19. Chrome is Phase 18. Composition map is Phase 20. Job-cluster implementation is inserted after Phase 20.

### Claude's Discretion
- Motion English role names `duration` and `easing` were derived from the user’s fast-lock (“duration/easing roles”) — do not expand into additional motion roles.
- Color job one-liners in D-17-31 are planner-facing glosses of the locked inventory; keep them that short in `LANGUAGE.md`.
- INDEX one-line purposes may be one clause each (language / chrome / workflows / composition). Do not add extra prose.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Milestone / phase
- `.planning/ROADMAP.md` — Phase 17 goal, DL-01, four success criteria, UI hint: no
- `.planning/REQUIREMENTS.md` — DL-01; CHR-01 / EMP-01 / MGR-01 / FIN-01 / HR-01 / CUST-01 / COMP-01 / SKETCH-01 / JOB-01 are later phases
- `.planning/PROJECT.md` — v0.2.1 is contract-first job clusters; do not recreate cancelled v0.2 Phases 17–26
- `.planning/STATE.md` — current focus Phase 17; do not sketch; do not implement

### Historical design input (not authority)
- `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` — Phase 15 freeze; untouched; LANGUAGE wins on foundations
- `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-CONTEXT.md` — D-15-01…D-15-13 status mapping and frozen-component intent
- `.planning/phases/15-ux-foundation-design-tokens-shared-components/15-01-SUMMARY.md` — claimed token/component landing; **do not trust over the live tree**

### Live value stores (CSS wins on values)
- `web/src/index.css` — live tokens: Inter via `font-sans`; `--display-family` / `--text-family`; shadcn semantic roles; light/dark maps; **no `--status-*`; no density tokens; no motion tokens**
- `web/components.json` — shadcn `style: base-mira`, `baseColor: olive`, `cssVariables: true`, `css: src/index.css`, `iconLibrary: lucide`

### Live UI debt (do not modify in Phase 17)
- `web/src/components/shared/status-badge.tsx` — raw Tailwind status map
- `AGENTS.md` — insert the one-sentence gate immediately after the title; OpenWiki remains the general starting point after that gate

### Sketch process (not this phase)
- `.planning/sketches/SKETCH-LOOP-CONTRACT.md` — amend only in Phase 20 if ambiguity remains (SKETCH-01). Do not run a sketch session in Phase 17.
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `web/src/index.css` — value store for type families/weights, shadcn color roles, radius, light/dark. Phase 17 cites it; does not edit it.
- `web/components.json` — kit identity (`base-mira` / olive). Phase 17 cites it; does not edit it.
- `web/src/components/ui/` — shadcn/base-ui primitives. Out of LANGUAGE.md (component APIs).
- Phase 15 frozen set is **not** fully present on disk. Do not generate those components here.

### Established Patterns
- Tailwind v4 CSS-first: `:root` / `.dark` + `@theme inline` in `index.css`.
- shadcn semantic roles already used for chrome/action (`primary`, `accent`, `destructive`, `muted`, …).
- `html { font-sans }` + decorative `.font-display` / `.font-text` — type jobs `ui` / `display` / `text`.
- No `prefers-reduced-motion` usage found in indexed sources.
- `AGENTS.md` currently starts with `# Hourglass Codebase Guide for AI Agents` then `## OpenWiki`. Gate goes between those.

### Integration Points
- New directory `docs/design/` (does not exist yet). Create INDEX + LANGUAGE only.
- `AGENTS.md` title → gate → existing OpenWiki section.
- No `docs/design/workflows/` until Phase 19.
- Backend-only work skips the design-document gate; UI work does not.

### Creative options
- None. Write-set and outline are locked. Brevity is a hard constraint: foundations are roles + MUST/SHOULD + CSS pointers, not catalogs.
</code_context>

<specifics>
## Specific Ideas

- User read Phase 15 and wants to leave its unstructured dump for the new architecture: INDEX + LANGUAGE + later chrome / workflow / composition docs.
- User asked to start feature/screen contracts now. That is Phase 19 (`docs/design/workflows/`). Phase 17 cannot write them. Path: plan 17 → execute 17 → Phase 18 chrome → Phase 19 role/workflow contracts.
- User asked to stop the long contract-artifact loop. Remaining Motion / Density / Status were fast-locked from the options they selected (duration/easing + reduced-motion; 4px compact density; five status Gaps with full chrome split).
- `status-badge.tsx` raw palettes are known debt, not Phase 17 work.
</specifics>

<deferred>
## Deferred Ideas

- **Feature / screen / workflow contracts** — Phase 19. `docs/design/workflows/` created then. Customer contract may conclude “no app surface”; customer portal remains out of scope.
- **Chrome contract** — Phase 18, `docs/design/CHROME.md`. Sidebar/chart tokens stay out of LANGUAGE.md; chrome may consume them later. Admin/Settings chrome out of scope. Org tree is manager/HR composition, not this phase.
- **Composition map** — Phase 20, `docs/design/COMPOSITION.md`.
- **Sketch-loop reconcile** — Phase 20, only if ambiguity remains. Do not sketch to close UXFD-02.
- **Job-cluster implementation** — insert after Phase 20. Not route phases. MUST NOT amend `LANGUAGE.md` as a side effect.
- **CSS status / density / motion / `ui` tokens** — later docs/planning amendment when a phase proves a language gap. Phase 17 records Gaps only.
- **StatusBadge rewrite onto status roles** — later job-cluster / presentation work after tokens exist. Not Phase 17.
- **Restore missing Phase 15 frozen components** (PageHeader, FilterBar, DataTable, EmptyState, ConfirmDialog) — not Phase 17.
- **Domain status → role mapping table** — stays in historical `15-UI-SPEC.md` until a later contract needs it; not copied into `LANGUAGE.md`.
</deferred>

---

*Phase: 17-Design-language contract*
*Context gathered: 2026-08-26*
