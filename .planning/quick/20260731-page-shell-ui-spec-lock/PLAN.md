---
slug: page-shell-ui-spec-lock
created: 2026-07-31
---

# Lock page shell (Header + Body) in Phase 10 UI-SPEC

Amend `.planning/phases/10-information-architecture-implementation/10-UI-SPEC.md` so ALL pages get the "wrapped" composition already shipped on /org-hierarchy:

1. New "Page Shell" section: locked structure AppShell > Header + Body, component contracts
   (Header h-12 page-global affordances: title/tabs/filters/search/CTAs as long as they fit;
   Body class list locked, wraps all page content, owns padding + inner scrolling).
2. Per-surface header composition table (Today, Approvals, Working Groups, Activities, Org,
   carried-over surfaces wrap-only).
3. Amend Spacing (2xl usage), Typography (title placement in shell Header), Surface
   Composition Locks (new item 7).
4. Frontmatter amendment note.
