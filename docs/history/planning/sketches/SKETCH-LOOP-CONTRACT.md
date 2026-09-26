---
status: active
applies-to: "phases 16-26 (surface and polish phases)"
source: "D-15-09 / D-15-11 (phase 15 decisions); gsd-sketch SKILL.md + sketch.md workflow"
---

# Sketch Loop Contract

Standalone process contract for the sketch-driven UI loop. Every surface/polish phase
(16–26) inherits these rules from **this file**. CONTEXT.md is not the source of
truth for the loop — this document is. If this file and CONTEXT.md ever disagree,
this file wins.

## The Rules

1. **Sketch first, plan second.** Every surface/polish phase (16–26) runs
   `gsd-sketch` FIRST, before its UI plan is written. No phase skips the sketch
   step, even when the design answer seems obvious — the user sees 2–3 variants
   before any plan is drafted.

2. **2–3 variants per round; one winner (the "2-3 variants" rule).** Each sketch
   round shows 2–3 (i.e. 2-3) variants for
   comparison. The user agrees on exactly one — variants never merge, and ties are
   resolved by the user picking one. Variants are labeled A/B/C in display order.
   Each sketch lives under `.planning/sketches/NNN-descriptive-name/` with a
   per-sketch `README.md` whose frontmatter marks the winner
   (`winner: "B"`). The winning tab in the sketch HTML carries a ★ indicator.

3. **MANIFEST is updated per sketch.** `.planning/sketches/MANIFEST.md` is created
   on the first sketch and updated on every sketch with the design direction,
   reference points, and the sketch table including winners. Sketches are listed in
   creation order; the winner column references the display label (A/B/C).

4. **Plans that follow an agreed sketch are UI-only.** No new API endpoints, no
   backend changes — the sketch round settles the design, and the plan implements
   exactly that design against the existing API surface.

5. **The 3-round cap (rounds ≤ 3) is the ONLY hard rule (D-15-11).** A phase runs at most 3
   sketch-option rounds; there is no minimum (a phase may run 1, 2, or 3 rounds —
   1–2 for polish, 2–3 for new surfaces like the scheduler). A 4th round is
   refused. The wrap-up step (`gsd-sketch --wrap-up`, which packages findings into
   `sketch-findings-*` skills under `.opencode/skills/`) is NOT a sketch round and
   does not consume the cap.

6. **Commit convention.** Sketch approvals are committed as
   `docs(sketch-NNN): [winner] ...` — e.g.
   `docs(sketch-001): variant B — two-panel layout with collapsible sidebar`.

7. **Inheritance.** Downstream phases inherit these rules from this file. Phase
   planning must reference this contract (not CONTEXT.md) when pinning sketch-loop
   mechanics.

---

## Mechanics Summary (for quick reference)

| Step | Where | Decision |
|------|-------|----------|
| Sketch round | `.planning/sketches/NNN-descriptive-name/` | 2–3 variants shown; user picks exactly one |
| Winner | sketch `README.md` frontmatter | `winner: "A"`/`"B"`/`"C"` + ★ on the winning tab |
| Design direction + table | `.planning/sketches/MANIFEST.md` | updated per sketch, creation order, winners column |
| Follow-up plan | UI-only | no new API endpoints, no backend changes |
| Round cap | — | rounds ≤ 3 per phase (1–2 polish, 2–3 new surfaces); wrap-up is not a round |
| Commit | `docs(sketch-NNN): [winner] ...` | one commit per approved sketch incl. MANIFEST |

---

*Source: D-15-09 (standalone contract doc), D-15-11 (3-round cap is the only hard rule),
research A6 (wrap-up is not a round); mechanics from `gsd-sketch` SKILL.md and the
sketch workflow (`~/.config/opencode/gsd-core/workflows/sketch.md`).*