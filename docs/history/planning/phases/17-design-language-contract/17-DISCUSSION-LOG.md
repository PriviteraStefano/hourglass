# Phase 17: Design-language contract - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-26
**Phase:** 17-Design-language contract
**Areas discussed:** Contract artifact, Motion language, Density model, Status vocabulary

---

## Contract artifact

| Option | Description | Selected |
|--------|-------------|----------|
| Title + one sentence + path list | INDEX is a map only. No stack, MUST NOTs, shadcn, Gaps. | ✓ |
| Title + How to use | Short usage intro still without language rules. | |
| You decide (title + one sentence) | How to use belongs in AGENTS.md. | |

**User's choice:** Title, one sentence (this is the design documentation map), then the path list. No stack, no MUST NOTs, no shadcn, no Gaps.
**Notes:** Locked INDEX prose.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Color = interaction/chrome only | Status stays its own foundation (all Gaps). MUST NOT treat Color as status. | ✓ |
| Color lists both interaction and five status roles | One colour chapter; status marked Gaps. | |
| You decide | Color = interaction/chrome only. | |

**User's choice:** Color names interaction/chrome roles only. Point at index.css. Status stays in the Status foundation (all Gaps). MUST NOT treat Color roles as status.

---

| Option | Description | Selected |
|--------|-------------|----------|
| English type roles only | display, ui, mono — or live Hourglass names. MUST NOT name font files. | ✓ |
| Name the three live faces | Inter, Martian Mono, Geist Mono plus jobs. | |
| You decide | English type roles; font files stay in CSS. | |

**User's choice:** English type roles only. MUST NOT name font files as the language.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Live CSS names display, text, plus ui | Do not invent mono. | ✓ |
| Force display / ui / mono | Record mismatches as Gaps. | |
| You decide | Live CSS names; no invented mono. | |

**User's choice:** Live CSS names: display, text, plus ui for the default Inter/font-sans face. Do not invent mono.

---

| Option | Description | Selected |
|--------|-------------|----------|
| More questions | Stay on Contract artifact. | ✓ (first check) |
| Next area | Motion language, Density, Status. | |

**User's choice:** More questions (first area-complete check).

---

| Option | Description | Selected |
|--------|-------------|----------|
| Repo-root citations only | `web/src/index.css` and `web/components.json`. No markdown links to CSS/config. | ✓ |
| Relative markdown links | From docs/design/. | |
| You decide | Repo-root citations; INDEX is the only linked map. | |

**User's choice:** Repo-root citations only. No markdown links to CSS/config.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Product-wide shadcn colour jobs | background…ring and *-foreground pairs. No ramps, sidebar-*, chart-*. | ✓ |
| Every live shadcn colour token | Including sidebar-* and chart-*. Still no primitive ramps. | |
| You decide | Product-wide colour jobs; sidebar/chart out. | |

**User's choice:** Product-wide shadcn colour jobs only. Do not name --base-*/--primary-* ramps, sidebar-*, or chart-*.

---

| Option | Description | Selected |
|--------|-------------|----------|
| ui named + Gap | ui — no type token in index.css. display/text point at live CSS. | ✓ |
| Treat ui as backed by html font-sans | No Gap. Still MUST NOT name the font file. | |
| You decide | ui named + Gap. | |

**User's choice:** Name ui as meaning (default sans/UI face). Record Gap: ui — no type token in index.css.

---

| Option | Description | Selected |
|--------|-------------|----------|
| LANGUAGE.md + reserved CHROME / workflows / COMPOSITION | Nothing else in INDEX. | ✓ |
| Also list AGENTS.md, 15-UI-SPEC.md, index.css | Related pointers. | |
| You decide | Design docs only. | |

**User's choice:** LANGUAGE.md (link) plus reserved CHROME.md (18), workflows/ (19), COMPOSITION.md (20). Nothing else.

---

| Option | Description | Selected |
|--------|-------------|----------|
| More questions | Stay on Contract artifact. | ✓ (second check) |
| Next area | Motion, Density, Status. | |

**User's choice:** More questions (second area-complete check).

---

| Option | Description | Selected |
|--------|-------------|----------|
| One short overlay rule | Overlay base-mira / olive. MUST use primitives. MUST NOT invent a parallel system or restyle the kit in this file. | ✓ |
| Name kit and list restyleable pieces | Still no component APIs. | |
| You decide | One-short-rule overlay. | |

**User's choice:** One short rule. Cite components.json + index.css. No component APIs.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Title → Purpose → changelog → rest | Changelog near top, not above Purpose. | ✓ |
| Title → changelog → Purpose | Changelog first body section. | |
| You decide | Purpose then changelog. | |

**User's choice:** Title → Purpose → changelog → foundations → overlay → do/don’t → pointers → not-in-this-file → 15-UI-SPEC note → Gaps.

---

| Option | Description | Selected |
|--------|-------------|----------|
| One line per Color role + job | *-foreground as the pair, not a second essay. | ✓ |
| Name-only list | Usage only in MUST/SHOULD rules. | |
| You decide | One line per role + job. | |

**User's choice:** One line per role: English name + job. *-foreground named as the pair.

---

| Option | Description | Selected |
|--------|-------------|----------|
| Don’t-only digest | Short list of foundation MUST NOTs. No Do column. No examples. | ✓ |
| Do + Don’t | Do may only restate existing MUSTs. | |
| You decide | Don’t-only. | |

**User's choice:** Don’t-only digest. Section may still be titled Do/don’t.

---

| Option | Description | Selected |
|--------|-------------|----------|
| More questions | Stay on Contract artifact. | |
| Next area | Motion, Density, Status. | |

**User's choice:** Interrupted by free-text: user asked to start feature/screen contracts and said the loop was taking too long. Interpreted as: close Contract artifact; do not start Phase 19 work in this phase.

---

## Close-out (user interrupt)

| Option | Description | Selected |
|--------|-------------|----------|
| Fast-lock remaining areas then CONTEXT | Motion: duration/easing, reduced-motion MUST, CSS owns values, unbacked Gaps, no catalog. Density: 4px, compact default, no px tables. Status: already locked. | ✓ |
| One short round each for Motion and Density | Status already locked. Then CONTEXT. | |
| Skip remaining and write CONTEXT | Planner fills Motion/Density from Phase 15. | |

**User's choice:** Fast-lock the rest, then write CONTEXT. Then plan 17 and reach Phase 19 after 18+20.
**Notes:** Feature/screen contracts redirected to Phase 19. Phase 17 write set unchanged.

---

## Motion language

Fast-locked from the close-out option (no separate question round).

**User's choice:** Duration/easing roles; reduced-motion MUST; CSS owns values; unbacked roles are Gaps; no animation catalog.
**Notes:** Live `index.css` has no motion tokens → Gaps for duration and easing. `prefers-reduced-motion` is a rule, not a Gap line.

---

## Density model

Fast-locked from the close-out option (no separate question round).

**User's choice:** 4px rhythm; compact default; no px tables; CSS owns values.
**Notes:** Live `index.css` has no density tokens → Gap `density — no density token in index.css.` Do not copy Phase 15 xs…3xl table.

---

## Status vocabulary

Locked earlier in the conversation (full separation; five Gaps; no proposed token names). Not re-asked.

**User's choice (prior lock):** Full separation from chrome/action tokens. Five Hourglass roles (neutral, info, success, warning, danger) are all Gaps. MUST NOT map danger to destructive. Gap form: `success — no status token in index.css.`
**Notes:** Domain-status mapping tables stay in historical 15-UI-SPEC. status-badge.tsx raw Tailwind is debt, not Phase 17 work.

---

## Claude's Discretion

- Motion English names `duration` and `easing` taken from the user’s fast-lock wording.
- Color one-line jobs (background — page/canvas, etc.) written as planner glosses of the locked inventory.
- INDEX one-line purposes left as one clause each.

## Deferred Ideas

- Feature/screen/workflow contracts → Phase 19
- Chrome contract → Phase 18
- Composition map → Phase 20
- Sketch-loop reconcile → Phase 20 iff ambiguity remains
- Job-cluster implementation → insert after Phase 20
- CSS tokens for status / density / motion / ui → later docs/planning amendment
- StatusBadge rewrite and missing Phase 15 frozen components → not Phase 17
