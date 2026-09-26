# Research: Collapsed-sidebar hover/click dead zone over icons

**Quick task:** 260801-got
**Date:** 2026-08-01
**File under investigation:** `web/src/components/layout/sidebar.tsx` (with `web/src/components/ui/sidebar.tsx`)

## Symptom

In collapsed (icon) mode, hovering only the icon of **Today**, **Working Groups**,
**Customers**, and **Approvals** does not highlight the button and the click does not
navigate. Clicking just right of the icon (top-right sliver of the button) works.

## Root cause

The four broken buttons are exactly the buttons that are the **last item of their
group**, i.e. they are immediately followed in the DOM by the *next group's*
`SidebarGroupLabel`.

In collapsed mode, `SidebarGroupLabel` (`web/src/components/ui/sidebar.tsx:415`) keeps
its full box but is hidden visually only:

```
group-data-[collapsible=icon]:-mt-8 group-data-[collapsible=icon]:opacity-0
```

`opacity-0` does **not** disable pointer events, and `-mt-8` (a negative 2rem top
margin) pulls the 2rem-tall label box up so it sits **on top of the previous group's
last menu button**. Because it comes later in the DOM (static flow → painted above) and
is a plain interactive div, it swallows hover + click events over the overlapped region.

Geometry (collapsed, inset variant): the label box covers the bottom ~27px of the 32px
button, which fully contains the centered 16px icon (y≈8–24). Only the top ~5px strip
of the button stays clickable — matching "icon is dead, but just right of the icon
(top strip) works".

## Why only these four buttons

| Button | Group | Followed by |
|--------|-------|-------------|
| Today | (ungrouped) | Track label |
| Working Groups | Work (last) | People label |
| Customers | Economics (last) | Review label |
| Approvals | Review (last) | Reports label |

Every other button (Time, Expenses, Activities, Contracts, Org, Exports) is *not* the
last item of its group, so it is followed by another menu button (no negative-margin
label) and has no overlap.

The regression was introduced in `aaaf6b7 feat(10-02): regroup sidebar into D-1
pillars with role-scoped visibility` — the first commit to render `SidebarGroupLabel`s.
Note the repo's `SidebarGroup` uses `py-1` and 1px separators, which is a tighter
vertical rhythm than upstream shadcn (`p-2`, no separators) — the reason the overlap
lands squarely on the previous button here.

## Fix options (not yet applied)

1. **`pointer-events-none` on the group label** (root fix, minimal). The invisible label
   stops intercepting events while layout is untouched.
2. **`group-data-[collapsible=icon]:hidden` instead of `opacity-0`** on the label. The
   label's net flow contribution is already 0 (`-mt-8` + `h-8`), so hiding it keeps the
   collapsed layout identical and removes the dead region entirely.
3. App-level: add `relative z-10` to the `<Link>` render in `sidebar.tsx` so nav buttons
   paint above the static label. Works, but patches the symptom per-item rather than the
   component.

Recommended: **option 2** (`hidden` in collapsed mode) as the primary fix, or option 1
(`pointer-events-none`) for a smaller diff.
