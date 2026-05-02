# Plan: Hourglass Redesign

> Source PRD: docs/superpowers/specs/2026-03-29-hourglass-redesign.md

## Architectural Decisions

Durable decisions that apply across all phases:

- **Routes**: All new backend routes under `/v1/` prefix (header-based versioning: `Accept: application/vnd.hourglass.v1+json`). Frontend routes: `/entries` (unified time/expense), `/approvals` (manager/finance dashboard), `/settings/*` (admin pages).
- **Schema**: Flattened entries — one project per time entry, one item per expense. New tables: `customers`, `organization_settings`, `project_managers`. Multi-role memberships via unique constraint on `(user_id, organization_id, role)`. Receipts stored as BLOB in `expense_receipts`. Soft delete via `deleted_at` column.
- **Key models**: `TimeEntry` (single project, hours), `Expense` (single project, type, amount), `Customer`, `OrganizationSettings`, `ProjectManager`. Status enum: `draft`, `pending_manager`, `pending_finance`, `approved`, `rejected` (removed `submitted`). Roles: `employee`, `manager`, `finance` (finance = admin).
- **Auth**: JWT in HttpOnly cookies. Multi-org support via org switch endpoint that issues new JWT. Rate limiting: 10/min anonymous, 100/min authenticated. Password reset + email verification flows. SMTP for production, console logging for dev.
- **Approval routing**: Based on `project_managers` table. Auto-skip logic for multi-role users (employee+manager skips manager level, finance auto-approves own entries).

---

## Phase 1: Infrastructure & Auth Hardening

**User stories**: Self-registration with email verification, password reset, rate limiting, health check, migration CLI, API versioning, request logging.

### What to build

A new user can register with email verification (console logging in dev). Existing users can request password reset via email. Rate limiting prevents abuse on anonymous endpoints. Health check endpoint is available for monitoring. Migration CLI tool allows running up/down migrations. All routes move to `/v1/` prefix with header-based versioning. Request logging captures method, path, status, duration.

### Acceptance criteria

- [ ] `GET /v1/health` returns `{ "status": "ok" }`
- [ ] `POST /v1/auth/register` creates inactive user, logs verification link (dev) or sends email (prod)
- [ ] `POST /v1/auth/verify` activates user with valid token
- [ ] `POST /v1/auth/forgot-password` logs reset link (dev) or sends email (prod)
- [ ] `POST /v1/auth/reset-password` sets new password with valid token
- [ ] Rate limiting: 10 req/min on anonymous endpoints per IP, 100 req/min on authenticated per user
- [ ] Migration CLI: `go run ./cmd/migrate [up|down]` works
- [ ] All routes under `/v1/` prefix, version header optional (defaults to v1)
- [ ] Request logs include method, path, status, duration

---

## Phase 2: Schema Migration — Customers, Org Settings, Multi-Role

**User stories**: Customer entity, organization settings, multi-role users, project managers table.

### What to build

New database tables created without breaking existing functionality. Customers can be stored with business info. Organization settings have defaults for km rate, currency, timezone, week start. Project managers link users to projects. Memberships allow multiple roles per user per org. Existing data migrates cleanly.

### Acceptance criteria

- [ ] `customers` table created with columns: id, organization_id, company_name, contact_name, email, phone, vat_number, address, is_active, created_at
- [ ] `organization_settings` table created with columns: organization_id (PK), default_km_rate, currency, week_start_day, timezone, show_approval_history, created_at, updated_at
- [ ] `project_managers` table created with columns: id, project_id, user_id, created_at; unique constraint on (project_id, user_id)
- [ ] `organization_memberships` unique constraint changed from (user_id, organization_id) to (user_id, organization_id, role)
- [ ] `contracts` table gains `customer_id` column (nullable FK)
- [ ] `time_entries` table gains `project_id` column, `deleted_at` column
- [ ] `expenses` table gains `project_id`, `customer_id`, `type`, `deleted_at` columns
- [ ] `expense_receipts` table gains `receipt_data BYTEA`, `mime_type` columns
- [ ] Data migration flattens time_entry_items and expense_items into parent tables
- [ ] Existing app still runs after migration
- [ ] Backend models updated to match new schema

---

## Phase 3: Organization Settings CRUD

**User stories**: Finance manages org settings (km rate, currency, week start, timezone, approval history toggle).

### What to build

Finance users can view and edit organization settings through a dedicated settings page. The settings form includes fields for default km rate (used as fallback when contract has none), currency code, week start day, timezone, and approval history visibility toggle. Settings are auto-created with defaults when a new organization is created.

### Acceptance criteria

- [ ] `GET /v1/organizations/:id/settings` returns current settings
- [ ] `PUT /v1/organizations/:id/settings` updates settings (finance only)
- [ ] Currency validated against ISO 4217 codes
- [ ] Settings row auto-created on organization creation
- [ ] Frontend: `/settings/organization` page with form for all settings
- [ ] Form validation shows inline errors
- [ ] Success saves and shows toast notification

---

## Phase 4: Customer Management

**User stories**: Finance creates/edits/deactivates customers, links customers to contracts.

### What to build

Finance users manage customers through a dedicated settings section. Customers have business contact info and can be linked to contracts. Only the creating organization can edit a customer. Deactivated customers cannot be linked to new contracts but existing links are preserved. Customer dropdown appears in contract forms.

### Acceptance criteria

- [ ] `GET /v1/customers` lists customers for organization
- [ ] `POST /v1/customers` creates customer (finance only)
- [ ] `GET /v1/customers/:id` returns customer details with linked contracts
- [ ] `PUT /v1/customers/:id` updates customer (creating org only)
- [ ] `DELETE /v1/customers/:id` returns 409 if linked to contracts, otherwise deactivates
- [ ] `contracts` endpoint returns customer info; contract create/update accepts `customer_id`
- [ ] Frontend: `/settings/customers` page with list, create/edit modal
- [ ] Frontend: Contract form shows customer dropdown

---

## Phase 5: Project Manager Assignment

**User stories**: Finance assigns managers to projects, approval routing by project managers.

### What to build

Finance users assign one or more managers to each project. Managers can view which projects they are assigned to. The approval routing logic changes from role-based lookup to project-based lookup using the project_managers table. If a project has no managers, entries skip to finance level.

### Acceptance criteria

- [ ] `GET /v1/projects/:id/managers` lists managers for project
- [ ] `POST /v1/projects/:id/managers` adds manager to project (finance only)
- [ ] `DELETE /v1/projects/:id/managers/:user_id` removes manager (finance only)
- [ ] Approval routing queries project_managers to find approvers
- [ ] Projects with no managers skip manager level → go to finance
- [ ] Frontend: Project form includes manager multi-select
- [ ] Frontend: Project detail shows assigned managers
- [ ] Managers can see list of their managed projects

---

## Phase 6: Unified Entry View — Time Entries

**User stories**: Employee views time entries on unified `/entries` page (calendar + list), creates single-project time entry, edits draft, deletes draft, submits for approval.

### What to build

Employees access a unified entries page with time/expense toggle and calendar/list toggle. Calendar view shows colored day indicators based on entry status. Clicking a day opens a side panel showing entries. Time entry creation form has single project selection, hours, date, and description. Combined hours per day cannot exceed 24. Draft entries can be edited or deleted. Submit sends entry to pending_manager (or auto-approves based on user roles).

### Acceptance criteria

- [ ] `GET /v1/time-entries` supports pagination, date range, project, status filters
- [ ] `POST /v1/time-entries` creates entry with single project, validates hours > 0 and <= 24
- [ ] `PUT /v1/time-entries/:id` updates draft entry
- [ ] `DELETE /v1/time-entries/:id` soft-deletes draft or rejected entry
- [ ] `POST /v1/time-entries/:id/submit` routes to pending_manager or auto-approves based on user roles
- [ ] Validation: total hours per user per day <= 24
- [ ] Frontend: `/entries?view=time&type=calendar` shows calendar with status-colored days
- [ ] Frontend: `/entries?view=time&type=list` shows filterable table
- [ ] Frontend: Create/edit modal with project dropdown, hours input, date picker, description
- [ ] Frontend: Side panel shows entries for clicked day
- [ ] Old `/time-entries` route redirects to `/entries?view=time`

---

## Phase 7: Unified Entry View — Expenses

**User stories**: Employee creates expense with type-specific fields, uploads receipts, mileage auto-calculates from km rate.

### What to build

Employees create expenses through a type-aware form. Mileage expenses show distance input with automatic amount calculation using contract km_rate or org default. Other expense types show amount input and receipt upload (required except 'other'). Multiple receipts can be uploaded per expense. Receipts stored as BLOB and can be downloaded.

### Acceptance criteria

- [ ] `GET /v1/expenses` supports pagination, date range, project, type, status filters
- [ ] `POST /v1/expenses` accepts multipart/form-data with multiple receipt files
- [ ] Mileage expenses: amount = km_distance * km_rate (contract rate or org default)
- [ ] Non-mileage expenses (except 'other'): receipt required
- [ ] `GET /v1/expenses/:id/receipts/:receipt_id` returns file download
- [ ] `DELETE /v1/expenses/:id` soft-deletes draft or rejected entry
- [ ] Frontend: `/entries?view=expense` shows expense list
- [ ] Frontend: Expense form shows type-specific fields (mileage: km input + auto-calc; others: amount + receipts)
- [ ] Frontend: Receipt upload with drag-drop, preview for images, max 10MB
- [ ] Error shown if mileage created but no km_rate configured on contract or org

---

## Phase 8: Approval Dashboard — View & Single Actions

**User stories**: Manager/Finance views pending approvals grouped by employee/month, approves, rejects, edit-approves, edit-returns, delegates single entries.

### What to build

Managers and Finance see a dedicated approvals dashboard with entries grouped by month → employee. Time and expense tabs separate the views. Clicking an entry opens a detail modal. Actions include approve, reject (with required reason), edit-approve (modify fields and approve with field-level diff logged), edit-return (modify and send back to draft with required message), and delegate (assign to another user who becomes a project manager if not already).

### Acceptance criteria

- [ ] `GET /v1/approvals/time-entries?status=pending_manager|pending_finance` returns grouped entries
- [ ] `GET /v1/approvals/expenses?status=pending_manager|pending_finance` returns grouped entries
- [ ] `POST /v1/approvals/time-entries/:id/approve` advances entry (manager→finance, finance→approved)
- [ ] `POST /v1/approvals/time-entries/:id/reject` sets status to rejected with comment
- [ ] `POST /v1/approvals/time-entries/:id/edit-approve` modifies fields, logs diff, advances
- [ ] `POST /v1/approvals/time-entries/:id/edit-return` modifies fields, sets to draft, requires comment
- [ ] `POST /v1/approvals/time-entries/:id/delegate` assigns to another user, makes them manager if needed
- [ ] Same endpoints exist for expenses under `/v1/approvals/expenses/`
- [ ] Approval history logged with field-level JSONB changes
- [ ] Frontend: `/approvals?view=time` shows grouped pending entries
- [ ] Frontend: Entry detail modal with approve/reject/edit-approve/edit-return/delegate actions
- [ ] Frontend: Reject modal requires reason text
- [ ] Frontend: Delegate shows warning if target is not already a manager

---

## Phase 9: Approval Dashboard — Batch Actions

**User stories**: Manager/Finance bulk-approves, bulk-rejects, bulk-edit-approves across employees.

### What to build

The approval dashboard supports selecting multiple entries via checkboxes and performing batch operations. Bulk approve and bulk reject apply a single action to all selected. Bulk edit-approve allows changing one field (project, hours/description) across all selected entries and advancing them. Audit trail records the batch action.

### Acceptance criteria

- [ ] `POST /v1/approvals/time-entries/batch-approve` approves multiple entries with optional comment
- [ ] `POST /v1/approvals/time-entries/batch-reject` rejects multiple entries with required comment
- [ ] `POST /v1/approvals/time-entries/bulk-edit-approve` modifies one field across entries, advances all
- [ ] Same batch endpoints exist for expenses
- [ ] Frontend: Checkboxes on approval dashboard entries
- [ ] Frontend: Batch action buttons appear when entries selected
- [ ] Frontend: Bulk edit-approve modal shows field selector and value input

---

## Phase 10: Mileage Rate Recalculation

**User stories**: Finance updates contract km_rate, recalculates affected mileage expenses.

### What to build

When finance updates a contract's km_rate, they see an option to recalculate affected mileage expenses from a specific date. If confirmed, all mileage expenses from that date onward are recalculated with the new rate. The recalculation is logged in approval history. Entry status remains unchanged.

### Acceptance criteria

- [ ] `PUT /v1/contracts/:id` can update km_rate
- [ ] On km_rate change, response includes `affected_mileage_count` and prompt for recalculation
- [ ] `POST /v1/contracts/:id/recalculate-mileage` accepts `from_date`, recalculates all mileage expenses
- [ ] Recalculation logged with `rate_recalculate` action and before/after amounts
- [ ] Frontend: Contract edit shows affected count when km_rate changed
- [ ] Frontend: Modal prompts for recalculation date range

---

## Phase 11: User Management & Org Switching

**User stories**: Finance invites users with multi-role, manages roles, deactivates users, users switch orgs.

### What to build

Finance users can invite new members with one or more roles. Invited users receive an email with activation link. Finance can edit user roles and deactivate users (preventing removal of the last finance user). A header dropdown allows users belonging to multiple organizations to switch contexts without page reload, receiving a new JWT for the selected org.

### Acceptance criteria

- [ ] `GET /v1/organizations/:id/members` lists members with their roles
- [ ] `POST /v1/organizations/:id/invite` accepts email + roles array, sends invitation (console log in dev)
- [ ] `PUT /v1/organizations/:id/members/:user_id/roles` updates roles array
- [ ] `DELETE /v1/organizations/:id/members/:user_id` deactivates membership
- [ ] Cannot remove last role from a membership
- [ ] Cannot deactivate last finance user in org
- [ ] `POST /v1/auth/switch-org` accepts organization_id, returns new JWT
- [ ] Frontend: `/settings/users` page with user list, invite modal, role edit
- [ ] Frontend: Role selection uses checkboxes (multi-select)
- [ ] Frontend: Header org switcher dropdown for multi-org users
- [ ] Switching org clears query cache and refetches profile

---

## Phase 12: CSV Exports

**User stories**: All roles export data within access scope (timesheets, expenses, combined).

### What to build

Users access an export page in settings to download CSV reports. Date range filter defaults to current month. Access scope limits data: employees export their own entries, managers export entries from managed projects, finance exports all. Three export options: timesheets, expenses, combined report.

### Acceptance criteria

- [ ] `GET /v1/exports/timesheets?from=&to=` returns CSV download, scoped to user access
- [ ] `GET /v1/exports/expenses?from=&to=` returns CSV download, scoped to user access
- [ ] `GET /v1/exports/combined?from=&to=` returns CSV with both time and expense data
- [ ] CSV includes: Date, Employee, Project, Contract, Customer, Hours/Amount, Description, Status
- [ ] Expense CSV adds: Type, Km Distance
- [ ] Frontend: `/settings/exports` page with date range picker
- [ ] Frontend: Three export buttons trigger downloads
