# Hourglass Design Language

## Purpose

This file is the source of truth for the Hourglass design *language* — the vocabulary of type, color, density, motion, and status — across all later presentation work. It defines roles and the rules that govern them (using `MUST`/`MUST NOT` for hard conformance and `SHOULD` for defaults); concrete values live in the live CSS and config and are cited, not duplicated.

- **Authority split:** CSS wins on *values* (the literal oklch/px/rem numbers). `LANGUAGE.md` wins on *meaning and usage* (what a role is for, how it may be combined). LANGUAGE.md wins on type, color, density, motion, and status vocabulary over every later presentation doc.
- **No invented tokens:** Do not invent CSS tokens inside this document. Point at the live value stores by repo-root path instead of copying their tables.
- **Authority stack (D-17-09):** This file wins on type, color, density, motion, and status vocabulary. Later design docs (`CHROME.md`, workflow contracts under `docs/design/workflows/`, `COMPOSITION.md`) and later GSD `UI-SPEC.md` files may only add surface, layout, copy, or composition. They `MUST NOT` override this file. If a later surface needs a language change, amend this file first.
- **Amendment rule (D-17-10):** Only later GSD docs/planning phases may amend this file (in place, via the changelog — no version fork). Implementation or job-cluster work `MUST NOT` amend it as a side effect.

## Changelog

- 2026-08-26 · Phase 17 · Added foundations

## Foundations

### Type

Three English type roles, not font-family names. `MUST NOT` name Inter / Martian Mono / Geist Mono as the language. Size and weight values stay in CSS; do not dump a pixel/weight table here.

- `display` — the display/decorative face. Lives in `web/src/index.css` as `--display-family` / `--display-weight` and the `.font-display` utility.
- `text` — the reading/body mono face. Lives in `web/src/index.css` as `--text-family` / `--text-weight` and the `.font-text` utility.
- `ui` — the default sans/UI face applied via `html { font-sans }` (Inter in the live tree). Recorded as a Gap below: no dedicated `--ui` token exists, so do not pretend one does.

`MUST NOT` invent a `mono` role; the two mono faces above are already named `display` and `text`.

### Color

Interaction and chrome roles only — `MUST NOT` treat these as status. One line per product-wide shadcn role; name `*-foreground` only where the pair exists. Point at `web/src/index.css`; do not name the `--base-*` / `--primary-*` ramps, `sidebar-*`, or `chart-*`.

- `background` — page/canvas
- `foreground` — default text
- `card` / `card-foreground` — raised surface and its text
- `popover` / `popover-foreground` — overlay surface and its text
- `primary` / `primary-foreground` — main action and its text
- `secondary` / `secondary-foreground` — secondary action/surface and its text
- `muted` / `muted-foreground` — quiet surface and quiet text
- `accent` / `accent-foreground` — interaction/navigation (never status) and its text
- `destructive` — destructive action (never status). No `destructive-foreground` pair exists in the live CSS; do not invent one.
- `border` — strokes
- `input` — form chrome
- `ring` — focus ring

`accent` is interaction/navigation, never status. `destructive` is an action role, not domain `danger`. `MUST NOT` map `danger` to `destructive`.

### Density

4px spacing rhythm, compact default. CSS owns the values; do not copy a pixel-spacing table here. Recorded as a Gap below.

`--radius` is a value, not a density role — do not catalog it in the language.

### Motion

Two English motion roles, not recipes:

- `duration` — motion length
- `easing` — motion curve

`MUST NOT` catalog animations, transitions, or component motion recipes. The live `web/src/index.css` has no duration or easing tokens — both are recorded as Gaps below.

UI `MUST` honor `prefers-reduced-motion`. That is a rule, not a token, and not a Gap line; do not invent a reduced-motion CSS identifier.

### Status vocabulary

Five semantic status roles, kept separate from interaction/chrome tokens:

- `neutral`
- `info`
- `success`
- `warning`
- `danger`

These are *meaning*, not the chrome/action tokens above. `MUST NOT` use interaction tokens (`primary`, `accent`, `destructive`, `muted`, …) as status, and `MUST NOT` map `danger` to `destructive`. All five are Gaps because the live `web/src/index.css` has no status tokens — recorded below.

Do not copy the Phase 15 domain-status → role mapping table. Note `web/src/components/shared/status-badge.tsx` still uses raw Tailwind palettes with `dark:` variants as known historical debt; this file `MUST NOT` modify it.

## Overlay

Hourglass overlays the shadcn kit with style `base-mira` and baseColor `olive` (see `web/components.json`) and `MUST` use those primitives. `MUST NOT` invent a parallel visual system or restyle the kit in this file. Cite `web/components.json` and `web/src/index.css`; no component APIs belong here.

## Light / dark

Semantic roles are theme-invariant. UI `MUST NOT` branch on light/dark except for the theme toggle itself; per-theme values stay in `web/src/index.css`. Do not duplicate per-mode role lists.

## Do / don't

- Don't name Inter / Martian Mono / Geist Mono as the language; name the English roles.
- Don't invent a `mono`, `--ui`, or `destructive-foreground` token.
- Don't treat color roles as status, and don't map `danger` to `destructive`.
- Don't copy pixel/weight, oklch/hex, or spacing tables into this file.
- Don't catalog `--radius` as a density role or animations/transitions as motion roles.
- Don't use interaction tokens (`primary`, `accent`, `destructive`, `muted`, …) as status.
- Don't invent a reduced-motion CSS identifier.
- Don't modify `web/src/components/shared/status-badge.tsx` or restyle the shadcn kit here.
- Don't amend this file from implementation or job-cluster work.

## Pointers

- `web/src/index.css` — live tokens for type, color, density (`--radius` only), and light/dark maps.
- `web/components.json` — shadcn kit identity (`base-mira` / olive).

## Not in this file

This file is foundations only. Explicitly out of scope here:

- No component APIs or primitives catalog.
- No chrome/layout contracts (those are Phase 18).
- No workflow/copy contracts (those are Phase 19).
- No token-value tables (oklch/hex/px) — values stay in CSS.
- No screenshots or worked examples.

Future contracts are workflow-group oriented (a complete workflow spanning pages and API interactions), not one file per current route. That work is Phase 19; this file does not pre-empt it.

## 15-UI-SPEC note

`.planning/phases/15-ux-foundation-design-tokens-shared-components/15-UI-SPEC.md` is untouched historical input and is **not** authority. If it conflicts with this file on type, color, density, motion, or status vocabulary, this file wins.

Phase 15 frozen components are listed here as *inputs* with their live status, not silently reused:

| Input | Live tree (2026-08-26) |
|-------|------------------------|
| StatusBadge | Present at `web/src/components/shared/status-badge.tsx` — raw Tailwind, not role tokens |
| PageHeader | Absent |
| FilterBar | Absent |
| DataTable | Absent |
| EmptyState | Absent |
| ConfirmDialog | Absent |

`entries-table.tsx` and `entries-filters.tsx` are present in the live tree but are **not** the frozen set. Do not restore or rewrite any of the above in Phase 17.

## Gaps

- `ui — no type token in index.css.`
- `density — no density token in index.css.`
- `duration — no motion token in index.css.`
- `easing — no motion token in index.css.`
- `neutral — no status token in index.css.`
- `info — no status token in index.css.`
- `success — no status token in index.css.`
- `warning — no status token in index.css.`
- `danger — no status token in index.css.`
