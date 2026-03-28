# Plan: Frontend Phase 9 - Time Entry End-to-End Slice

> Source Spec: `docs/superpowers/specs/2026-03-27-frontend-phase9-design.md`

## Architectural Decisions

Durable decisions that apply across all phases:

- **Routes**: 
  - `/login` - public
  - `/register` - public
  - `/time-entries` - protected, main calendar view
  - `/` - redirects to `/time-entries`
- **Tech Stack**: React 19, TypeScript, Vite, TanStack Router/Query/Form, Tailwind, Shadcn/UI
- **API Base**: `http://localhost:8080` (dev), env var `VITE_API_URL`
- **Auth**: JWT in localStorage (`auth_token`), auto-redirect on 401
- **State**: TanStack Query for server state, local state for UI

---

## Phase 1: Project Foundation

**User stories**: Setup development environment

### What to build

Initialize the React project with Vite, configure TypeScript paths, set up Tailwind CSS, and initialize Shadcn/UI. This phase delivers a runnable dev server with basic styling infrastructure.

### Acceptance criteria

- [ ] Vite project created in `web/` directory with React 19 + TypeScript
- [ ] Tailwind CSS configured with CSS variables for theming
- [ ] Shadcn/UI initialized with `components.json`
- [ ] Path alias `@/` configured in tsconfig and vite.config
- [ ] Dev server runs on port 3000
- [ ] Basic index.html with app root

---

## Phase 2: Routing + Auth Foundation

**User stories**: User can register, login, logout

### What to build

Set up TanStack Router with file-based routing, create public auth pages (login, register), implement protected route pattern, and build the auth API client with token management.

### Acceptance criteria

- [ ] TanStack Router configured with file-based routes
- [ ] `routes/__root.tsx` - root layout with QueryClientProvider
- [ ] `routes/index.tsx` - login page with form
- [ ] `routes/index.tsx` - register page with form
- [ ] `routes/_authenticated.tsx` - protected route group with auth check
- [ ] `lib/api.ts` - base fetch wrapper with auth header injection
- [ ] `hooks/useAuth.ts` - auth state and login/logout mutations
- [ ] Login persists token to localStorage
- [ ] 401 responses redirect to login
- [ ] Types defined for User, Organization, AuthResponse

---

## Phase 3: App Shell

**User stories**: User sees navigation after login

### What to build

Build the application shell with sidebar navigation and header. The sidebar shows all navigation items but marks non-implemented sections as "coming soon". Header shows org name and user dropdown with logout.

### Acceptance criteria

- [ ] `components/layout/app-shell.tsx` - main layout wrapper
- [ ] `components/layout/sidebar.tsx` - navigation with active state
- [ ] `components/layout/header.tsx` - logo, org name, user dropdown
- [ ] Time nav item links to `/time-entries`
- [ ] Other nav items (Expenses, Approvals, Contracts, Projects, Settings) show "coming soon" tooltip
- [ ] User dropdown has logout action
- [ ] Sidebar is 224px (w-56) fixed width

---

## Phase 4: Time Entries Calendar View

**User stories**: User can view monthly time entries

### What to build

Build the time entries page with hybrid calendar view. Mini-calendar on the left for navigation, detail panel on the right. Fetch monthly summary from API and color-code days by status.

### Acceptance criteria

- [ ] `routes/_authenticated/index.tsx` - main page
- [ ] `components/time-entries/mini-calendar.tsx` - month grid with color coding
- [ ] `components/time-entries/month-summary-bar.tsx` - month nav + total hours
- [ ] `hooks/useTimeEntries.ts` - monthly summary query
- [ ] `hooks/useProjects.ts` - projects for dropdown
- [ ] Days colored: yellow=draft, green=submitted/pending, blue=approved, red=rejected
- [ ] Month navigation arrows work
- [ ] Clicking a day sets selected date state

---

## Phase 5: Entry Detail + CRUD

**User stories**: User can create, edit, delete time entries

### What to build

Build the entry detail panel that shows the selected day's entry. Include inline form with project rows (dropdown + hours + description). Support create, update, delete operations for draft entries.

### Acceptance criteria

- [ ] `components/time-entries/entry-detail.tsx` - entry editor
- [ ] `components/time-entries/entry-row.tsx` - single project row
- [ ] `components/time-entries/status-badge.tsx` - status indicator
- [ ] `hooks/useTimeEntry.ts` - single entry query
- [ ] `hooks/useCreateTimeEntry.ts` - create mutation
- [ ] `hooks/useUpdateTimeEntry.ts` - update mutation
- [ ] `hooks/useDeleteTimeEntry.ts` - delete mutation
- [ ] No entry state: "Create Entry" button
- [ ] Draft/rejected: editable form
- [ ] Submitted/pending/approved: read-only view
- [ ] Add/remove project rows
- [ ] Zod validation: max 24 hours total
- [ ] TanStack Form for form state

---

## Phase 6: Submit Workflow

**User stories**: User can submit entries for approval

### What to build

Add submit actions to the entry detail form. Single entry submit and batch submit all drafts for the month. Show success toasts and update UI state after submission.

### Acceptance criteria

- [ ] `hooks/useSubmitEntry.ts` - submit single entry mutation
- [ ] `hooks/useSubmitMonth.ts` - batch submit mutation
- [ ] "Submit Entry" button on draft entries
- [ ] "Submit All Drafts" button in month summary bar (shows count)
- [ ] Success toast on submission
- [ ] Entry status updates to pending_manager after submit
- [ ] Submitted entries become read-only
- [ ] Button disabled if total hours is 0
