---
phase: quick
plan: 260801-got
subsystem: frontend/sidebar
tags: [sidebar, collapsed-mode, pointer-events, dead-zone, ui]
requires: []
provides: []
affects: [web/src/components/ui/sidebar.tsx]
tech-stack:
  added: []
  patterns: []
key-files:
  created: []
  modified: [web/src/components/ui/sidebar.tsx]
decisions:
  - D-01 (LOCKED pre-plan decision): fix collapsed-sidebar dead zone via plain `pointer-events-none` on SidebarGroupLabel — NOT option 2 (`hidden`) or option 3 (z-index); layout untouched
metrics:
  duration: "~10 min"
  completed: "2026-08-01"
---

# Quick Plan 260801-got: Collapsed-sidebar pointer-events fix

One-line Tailwind fix: add `pointer-events-none` to the `SidebarGroupLabel`
className so the invisible collapsed-mode group label stops intercepting hover
and click events over the previous group's last nav button (Today, Working
Groups, Customers, Approvals).

## What was built

`web/src/components/ui/sidebar.tsx` line 415 — `SidebarGroupLabel`'s `cn(...)`
className now starts with `pointer-events-none`, exactly per the locked decision
D-01 (option 1). The existing utilities `-mt-8`, `opacity-0`, and all other
layout classes are untouched, so collapsed-mode geometry is pixel-identical —
only pointer-event capture is disabled. `SidebarGroupAction` and the app-level
layout file `web/src/components/layout/sidebar.tsx` were deliberately not
touched.

## Verification

- `grep -c "pointer-events-none flex h-8" web/src/components/ui/sidebar.tsx` → **1** (passes plan criterion ≥1)
- Diff is a single-line change (verified via `git diff`; commit `54f465a` = 1 insertion, 1 deletion)
- `cd web && bun run build` → **fails on 6 pre-existing TypeScript errors + 1 pre-existing rolldown `[MISSING_EXPORT]` error, all in files unrelated to this change** (A/B verified: reverting this change reproduces the identical error set; zero new errors introduced)
- `bunx vite build` (bypassing tsc) → fails on the same pre-existing `useSearchParams` missing-export error in `invitation-accept-form.tsx`
- Human visual check (collapsed-mode hover/navigate on the 4 affected buttons; expanded mode unchanged) → **deferred to user verification** — requires running server + dev, out of agent scope for this quick task

## Deviations from Plan

### Out-of-scope discovery: pre-existing build breakage (not a regression)

The plan's verification criterion "`cd web && bun run build` exits 0" cannot be
satisfied on this branch for any commit — the build was already broken before
this change. A/B tested: pristine HEAD produces the identical 6 tsc errors and
1 rolldown missing-export error. Root causes appear to be dependency drift
(`@tanstack/react-router` `useSearchParams` export removed, `next-themes`
`ThemeProviderProps.attribute` changed) and stale types
(`BootstrapRequest.org_name` vs `organization_name`). Logged in full to
`.planning/quick/260801-got-investigate-sidebar-collapsed-mode-hover/deferred-items.md`
— **not fixed here** (out of scope; a future plan must repair them).

## Decisions Made

- **D-01 (carried from research, locked by user):** plain `pointer-events-none`
  on `SidebarGroupLabel` — group labels are non-interactive `div` headers, so
  disabling pointer events is safe in every mode; no collapsed-mode variant
  needed. Alternatives (`hidden`, z-index) explicitly rejected.

## Known Stubs

None — this plan adds a single CSS utility to an existing component; no new
surfaces, placeholders, or unwired data.

## Threat Flags

None — no new network endpoints, auth paths, file access patterns, or schema
changes. A CSS class on a non-interactive header introduces no security
surface.

## Files Changed

- `web/src/components/ui/sidebar.tsx` — `SidebarGroupLabel` className: prepended `pointer-events-none` (the only change)

Commit: `54f465a` — `fix(web): prevent invisible collapsed sidebar group labels from blocking icon hover/clicks`
