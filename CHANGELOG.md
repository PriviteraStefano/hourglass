# Changelog

All notable changes to **Hourglass** are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- CSV export page with date-range selection (phase 07).
- Enforced `JWT_SECRET` requirement in production/staging environments.

### Changed

- Flattened `TimeEntry` type on the frontend; rewrote API module accordingly.
- Rewrote `MiniCalendar` for client-side status computation.

### Fixed

- Select `onValueChange` type and contracts data access in `EditProjectDialog`.

---

## [0.1.0] — 2026-07-07

First tagged development milestone. Covers phases 1–7 of the MVP roadmap.

### Added — Core platform

- **Project foundation & auth** (phase 1): Go HTTP server, PostgreSQL connection
  pool, JWT auth with HttpOnly cookies, bcrypt password hashing, registration,
  login, logout, refresh, `GET /auth/me`, `GET /auth/memberships`.
- **Organizations & memberships** (phase 2): multi-org model with role-based
  access (`employee`, `manager`, `finance`, `customer`), organization
  switching.
- **Contracts** (phase 4): CRUD with `customer_id` association, `HasProjects`
  delete protection, frontend customer combobox.
- **Projects** (phase 5): CRUD with update/delete, subproject listing,
  `billable` / `internal` project types, frontend edit/delete dialogs.
- **Time entries** (phase 6): two-stage approval workflow
  (employee → manager → finance), role-differentiated handlers, line items per
  project, approval history, calendar view, detail/row/form components.
- **Expenses** (phase 6): CRUD with the same two-stage approval workflow,
  categories (`mileage`, `meal`, `accommodation`, `other`), expense route,
  calendar, page layout, sidebar integration.
- **Exports** (phase 7): date-range CSV export of approved entries.

### Added — Architecture & tooling

- Hexagonal (ports & adapters) architecture: domain → ports → services →
  HTTP/Postgres adapters.
- Shared JSON response envelope (`{ data }` / `{ error }`) in `pkg/api`.
- PostgreSQL migration CLI (`cmd/migrate`) with up/down/all modes and seed.
- Multi-stage Dockerfile (golang:1.26.1-alpine → alpine) and docker-compose
  (postgres:15-alpine + app).
- Makefile targets: `build`, `run`, `test`, `setup`, `migrate-*`, `docker-*`.
- Frontend: React 19, TanStack Router v1 (file-based), TanStack React Query v5,
  Vite, Tailwind CSS v4, shadcn/ui, Zustand, Zod, Recharts, react-day-picker.
- Testing: `stretchr/testify` + `testcontainers-go` (backend), Vitest + MSW +
  Playwright (frontend).
- CI/CD scaffolding under `.github/`.

### Added — Domain model

- Roles: `employee`, `manager`, `finance`, `customer` (DB CHECK constraint).
- Entry status: `draft` → `submitted` → `pending_manager` → `pending_finance`
  → `approved` / `rejected`.
- Approval actions: `submit`, `approve`, `reject`, `edit_approve`,
  `edit_return`, `partial_approve`, `delegate`.
- Governance models: `creator_controlled`, `unanimous`, `majority`.
- Immutable approval history in `*_approvals` tables.

### Infrastructure

- Initial commit: 2026-03-26.
- 311 commits across phases 1–7.
- Repository: https://github.com/PriviteraStefano/hourglass
