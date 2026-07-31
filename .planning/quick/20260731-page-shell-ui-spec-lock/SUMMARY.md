---
slug: page-shell-ui-spec-lock
status: complete
completed: 2026-07-31
---

# Summary — page shell UI-SPEC lock

Amended Phase 10 UI-SPEC to lock the Header + Body "wrapped" page shell (org-hierarchy pattern) as the universal contract for all authenticated pages.

Changes to `10-UI-SPEC.md`:
- Frontmatter: `amended_at` note recorded.
- New "Page Shell" section: locked structure (AppShell > Header + Body, no bare SidebarInset
  content), shell component reuse from `@/components/layout`, Header role (page-global
  affordances: title/tabs/filters/search/CTAs in a single 48px band), overflow rule,
  locked Body class list, padding/scrolling ownership, route boundaries inside Body,
  per-surface header composition table, a11y landmark/h1 note.
- Spacing: 2xl row re-pointed to the shell (title->content separation now shell-owned).
- Typography: Heading/Display rows state page titles render in the shell Header.
- Surface Composition Locks: new item 7 binding every page (new + carried-over,
  wrap-only) to the shell composition; org-hierarchy named canonical reference.

Last action: committed as docs(10): amend UI-SPEC with page shell composition lock
