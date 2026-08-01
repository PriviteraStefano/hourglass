# OpenWiki Update Plan (maintenance run)

Last successful run: init at 44c588f (2026-07-08). Current HEAD: 8384f9c.

## Source changes since last run (high signal)

1. **Activity ontology (phase 09)** — migration 011 rewrites projects/subprojects into one
   recursive `activities` table; `internal/core/domain/activity` + `ports/activity_repository.go`
   + `services/activity` + `http/activity_handler.go` replace project/subproject equivalents;
   `web/src/api/activities.ts` + `/activities` routes replace `/projects`. Entries FK collapse:
   `time_entries`/`expenses` drop project_id/subproject_id/wg_id → NOT NULL `activity_id`;
   working_groups.subproject_id → activity_id, enforce_unit_tuple dropped.
2. **Staffing schema (migration 012)** — availability_windows, membership validity columns,
   `hr` role added to DB CHECK; web Role type includes "hr".
3. **Migration 013** — relabels subproject activities kind=task→phase.
4. **Approval routing rewrite (ADR-BE-014)** — manager stage resolved from anchored WG
   manager+delegates (R-1), commercial activity without WG → ErrActivityNotLoggable (R-2),
   personal activity → unit-manager fallback, D-11 owner-in-approver-set skip to finance.
   Cycle prevention: ErrActivityCycle sentinel on parent path check.
5. **Info architecture (phase 10)** — Today landing page at `/`, sidebar regrouped into
   pillar groups with role-scoped visibility (`web/src/lib/role-visibility.ts`), Header+Body
   page shell, Approvals queue page (stage tabs), Working Groups page + `web/src/api/working-groups.ts`.
6. **Security hardening (phase 08)** — refresh-token reuse detection (migration 010),
   request-string length caps (`http/validate.go`), route error boundaries.
7. **Tooling** — ESLint/Prettier → Oxlint/Oxfmt (`web/package.json` scripts lint/fmt);
   `cmd/schema` removed; component tests now exist (@testing-library/react).

## Docs impact plan

| Page | Edit | Why |
|------|------|-----|
| quickstart.md | repo tree (drop cmd/schema); doc-map + next-steps "projects" → "activities" | product/nav changed |
| architecture/README.md | domain/ports/services lists (activity); route layout (activities, approvals, working-groups, Today, pillars) | architecture changed |
| domain/README.md | roles (hr); approval routing (BE-014); time-entries/expenses activity_id; replace Projects section with Activities | domain changed |
| operations/README.md | schema/tables/migrations (010–013, activities tables) | DB changed |
| testing/README.md | test file lists (activities, approvals, working-groups, error-boundary, expenses); gaps section (component tests now exist) | tests changed |

## Remaining questions
- None blocking. Legacy `internal/models` package still contains Project structs; docs will
  point to `internal/core/domain` as canonical (matching init-era structure).
