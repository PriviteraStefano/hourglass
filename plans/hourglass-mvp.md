# Plan: Hourglass MVP

> Source PRD: [Design Specification](../docs/superpowers/specs/2026-03-27-hourglass-design.md)

## Architectural Decisions

Durable decisions that apply across all phases:

- **Routes**: 
  - `/auth/*` - authentication endpoints
  - `/organizations/*` - org management
  - `/contracts/*` - shared contracts catalog
  - `/projects/*` - shared projects catalog
  - `/time-entries/*` - time tracking
  - `/expenses/*` - expense tracking
  - `/exports/*` - CSV downloads
  - `/customer/*` - customer view-only endpoints

- **Schema**:
  - Multi-tenant: all core tables include `organization_id`
  - UUID primary keys via `gen_random_uuid()`
  - Entry statuses: `draft`, `submitted`, `pending_manager`, `pending_finance`, `approved`, `rejected`
  - Shared resources: contracts/projects can be owned or adopted via adoption tables

- **Key models**:
  - `User` - email, password_hash, name, is_active
  - `Organization` - name, slug
  - `OrganizationMembership` - user_id, organization_id, role (employee, manager, finance, customer)
  - `Contract` - name, km_rate, currency, governance_model, created_by_org_id, is_shared
  - `Project` - name, type, contract_id, governance_model, created_by_org_id, is_shared
  - `TimeEntry` + `TimeEntryItem` - date, status, multiple project hours per entry
  - `Expense` + `ExpenseItem` + `ExpenseReceipt` - date, status, file uploads
  - `TimeEntryApproval` / `ExpenseApproval` - action, changes (JSONB), comment

- **Auth**:
  - JWT with 24-hour expiration
  - Refresh tokens (7-day) in httpOnly cookie
  - bcrypt (cost 12)
  - RBAC middleware per endpoint

- **Tech Stack**:
  - Backend: Go 1.26.1 (stdlib net/http), PostgreSQL 15+, plain SQL migrations
  - Frontend: React 19 + TypeScript, Vite, Shadcn/UI with BaseUI components
  - Infrastructure: Docker multi-stage (Alpine), GitHub Actions

---

## Phase 1: Project Foundation & Auth

**User stories**: User registration, login, organization creation, multi-tenant isolation

### What to build

Set up the Go backend project with PostgreSQL connection, run migrations, and implement the authentication system. Users can register (creating an organization), log in, and receive JWT tokens. The RBAC middleware enforces role-based access per organization.

### Acceptance criteria

- [ ] Go project initialized with stdlib net/http server
- [ ] PostgreSQL connection pool configured
- [ ] Migration runner set up (plain SQL files)
- [ ] `users` table created
- [ ] `organizations` table created
- [ ] `organization_memberships` table created
- [ ] `POST /auth/register` - creates org + admin user, returns JWT
- [ ] `POST /auth/login` - validates credentials, returns JWT
- [ ] `POST /auth/logout` - invalidates token (optional: token blocklist)
- [ ] JWT middleware extracts user ID and org membership
- [ ] RBAC middleware checks role per endpoint
- [ ] All queries filter by `organization_id` where applicable

---

## Phase 2: Contract & Project Management (Shared Resources)

**User stories**: Create contracts, create projects, adopt shared resources, governance model metadata

### What to build

Implement the shared resources catalog. Users can create contracts and projects (private to their org or shared), and adopt shared resources from other organizations. Governance model is stored but the edit flow is not implemented yet.

### Acceptance criteria

- [ ] `contracts` table created
- [ ] `projects` table created
- [ ] `contract_adoptions` table created
- [ ] `project_adoptions` table created
- [ ] `POST /contracts` - create contract with governance_model, is_shared
- [ ] `GET /contracts?scope=owned|adopted|all` - list contracts
- [ ] `GET /contracts/:id` - get contract details
- [ ] `POST /contracts/:id/adopt` - adopt a shared contract
- [ ] `POST /projects` - create project linked to contract
- [ ] `GET /projects?scope=owned|adopted|all&contract_id` - list projects
- [ ] `GET /projects/:id` - get project details
- [ ] `POST /projects/:id/adopt` - adopt a shared project
- [ ] Adopted resources appear in org's workspace but entries remain org-scoped

---

## Phase 3: Time Entry CRUD + Draft Workflow

**User stories**: Create daily time entries, edit drafts, view monthly summary, delete drafts

### What to build

Implement time tracking with multiple project entries per day. Users can create, update, and delete time entries. Entries start in draft status and remain drafts until submitted. Monthly summary provides calendar matrix view.

### Acceptance criteria

- [ ] `time_entries` table created
- [ ] `time_entry_items` table created
- [ ] `POST /time-entries` - create entry with multiple items (draft status)
- [ ] `GET /time-entries?date=X` - list entries for a date
- [ ] `GET /time-entries/:id` - get entry with items
- [ ] `PUT /time-entries/:id` - update items (draft only)
- [ ] `DELETE /time-entries/:id` - delete draft entry
- [ ] `GET /time-entries/monthly-summary?month=X&year=Y` - calendar view with totals
- [ ] Validation: max 24 hours per day per user
- [ ] Employees can only see/edit own entries
- [ ] Managers/Finance can view team/all entries

---

## Phase 4: Expense Entry CRUD + Receipts

**User stories**: Create daily expenses with receipts, edit drafts, view monthly summary

### What to build

Implement expense tracking with file upload support. Users can create expense entries with multiple items and attach receipts. Mileage entries auto-calculate amount from km_distance using contract km_rate. Monthly summary shows breakdown by category.

### Acceptance criteria

- [ ] `expenses` table created
- [ ] `expense_items` table created
- [ ] `expense_receipts` table created
- [ ] `POST /expenses` (multipart) - create entry with items + receipt files
- [ ] `GET /expenses?date=X` - list entries for a date
- [ ] `GET /expenses/:id` - get entry with items and receipts
- [ ] `PUT /expenses/:id` (multipart) - update items + receipts (draft only)
- [ ] `DELETE /expenses/:id` - delete draft entry
- [ ] `GET /expenses/monthly-summary?month=X&year=Y` - summary by category
- [ ] File upload: save to `/uploads/receipts/{year}/{month}/{uuid}.{ext}`
- [ ] File validation: jpg, png, pdf only, max 10MB
- [ ] Mileage auto-calc: `amount = km_distance * contract.km_rate`
- [ ] Employees can only see/edit own entries

---

## Phase 5: Approval Workflow

**User stories**: Submit entries, manager approval, finance approval, rejection, edit & return, batch operations, delegate, partial approval

### What to build

Implement the full approval chain from draft to approved. Users submit entries individually or batch-submit an entire month. Approvers see pending entries grouped by user/month and can approve, reject, edit & approve, edit & return, partially approve, or delegate. Auto-skip logic handles authors with multiple roles.

### Acceptance criteria

- [ ] `time_entry_approvals` table created
- [ ] `expense_approvals` table created
- [ ] Add `pending_manager`, `pending_finance` statuses to entry tables
- [ ] Add `current_approver_role` column to entry tables
- [ ] `POST /time-entries/:id/submit` - single entry → pending_manager
- [ ] `POST /time-entries/submit-month` - batch submit all drafts for month
- [ ] `POST /expenses/:id/submit` - single expense → pending_manager
- [ ] `POST /expenses/submit-month` - batch submit all drafts for month
- [ ] `GET /time-entries/pending-approval` - list grouped by user/month
- [ ] `GET /expenses/pending-approval` - list grouped by user/month
- [ ] `POST /time-entries/:id/approve` - pass to next level or finalize
- [ ] `POST /time-entries/:id/reject` - return to author with comment
- [ ] `POST /time-entries/:id/edit-approve` - modify then approve
- [ ] `POST /time-entries/:id/edit-return` - modify, send back for confirmation
- [ ] `POST /time-entries/:id/partial-approve` - approve some items, flag others
- [ ] `POST /time-entries/:id/delegate` - forward to another approver at same level
- [ ] `POST /time-entries/batch-approve` - approve multiple entries
- [ ] `POST /time-entries/batch-reject` - reject multiple entries
- [ ] Same endpoints for expenses
- [ ] Auto-skip: if author is manager, skip manager level
- [ ] Auto-skip: if author is finance, skip finance level
- [ ] Audit trail: every action creates approval record with changes (JSONB)

---

## Phase 6: Governance Edit Flow

**User stories**: Request edits to shared contracts/projects, voting, approval thresholds

### What to build

Implement the edit request flow for shared resources. Users propose changes, eligible approvers vote based on governance model, and changes are applied when threshold is met. Platform admin can override.

### Acceptance criteria

- [ ] `contract_edit_requests` table created
- [ ] `contract_edit_request_votes` table created
- [ ] `project_edit_requests` table created
- [ ] `project_edit_request_votes` table created
- [ ] `POST /contracts/:id/request-edit` - propose changes (JSONB)
- [ ] `GET /contracts/:id/edit-requests` - list pending requests
- [ ] `POST /contracts/:id/edit-requests/:request_id/approve` - vote approve
- [ ] `POST /contracts/:id/edit-requests/:request_id/reject` - vote reject with reason
- [ ] Threshold logic:
  - `creator_controlled`: only creator org + platform admin
  - `unanimous`: all adopting orgs must approve
  - `majority`: >50% of adopting orgs must approve
- [ ] Apply changes when threshold met, set `resolved_at`
- [ ] Same endpoints for projects
- [ ] Platform admin role can force approve/reject

---

## Phase 7: Customer Access

**User stories**: Admin invites customer, customer activation, customer view-only access

### What to build

Implement the customer role with restricted access. Admins create customer accounts and assign contracts. Customers activate via email link and can only view summaries and exports for their assigned contracts.

### Acceptance criteria

- [ ] `customer_contract_access` table created
- [ ] `POST /organizations/:id/invite-customer` - create customer with assigned contracts
- [ ] Generate activation token, store in `organization_memberships.invited_at`
- [ ] `POST /auth/activate` - customer sets password, activates account
- [ ] Activation link expires after 7 days
- [ ] `GET /customer/contracts` - list assigned contracts only
- [ ] `GET /customer/contracts/:id/summary?month=X&year=Y` - time/expense totals
- [ ] Customer role cannot see other org members, settings, or unassigned contracts
- [ ] Customer cannot submit, approve, or edit anything

---

## Phase 8: CSV Exports

**User stories**: Export timesheets, expenses, combined report

### What to build

Implement CSV export endpoints matching the spec column format. Access is controlled by role - finance exports all, managers export team, employees export own entries, customers export assigned contracts.

### Acceptance criteria

- [ ] `GET /exports/timesheets?month=X&year=Y&user_id` - CSV download
- [ ] `GET /exports/expenses?month=X&year=Y&user_id` - CSV download
- [ ] `GET /exports/combined?month=X&year=Y&user_id` - CSV with Type column
- [ ] Timesheet columns: Date, Employee, Organization, Project, Contract, Hours, Description, Status
- [ ] Expense columns: Date, Employee, Organization, Project, Contract, Category, Amount, Km Distance, Description, Status
- [ ] Combined adds: Type (time/expense)
- [ ] Role-based filtering:
  - Finance: all org entries
  - Manager: team entries
  - Employee: own entries
  - Customer: assigned contracts only

---

## Phase 9: Frontend Foundation

**User stories**: Login UI, org creation, basic navigation

### What to build

Set up the React frontend with auth flow, organization context, and basic layout. Users can log in, register a new org, and navigate the main sections.

### Acceptance criteria

- [ ] React 19 + TypeScript + Vite project initialized
- [ ] Shadcn/UI + BaseUI components configured
- [ ] React Query for data fetching
- [ ] React Hook Form + Zod for form validation
- [ ] Login page with email/password
- [ ] Register page (creates org + admin user)
- [ ] Activation page for invited users/customers
- [ ] Auth state persisted (localStorage)
- [ ] Organization context provider
- [ ] Layout with sidebar navigation
- [ ] Protected routes requiring auth
- [ ] Logout functionality

---

## Phase 10: Frontend - Time Entry & Expense Entry

**User stories**: Time entry form, expense entry form with receipts, monthly calendar view, submit all for month

### What to build

Implement the core entry forms for time and expenses. Users see a monthly calendar view of their entries, can add/edit drafts, and submit all entries for a month at once.

### Acceptance criteria

- [ ] Monthly calendar view showing days with entries
- [ ] Time entry form with multiple project rows
- [ ] Expense entry form with category dropdown, km distance input, file dropzone
- [ ] Draft entries highlighted in calendar
- [ ] Click day to view/edit entries
- [ ] Receipt preview/download
- [ ] "Submit All for Month" button with confirmation
- [ ] Entry status badges (draft, pending, approved, rejected)
- [ ] Form validation (max 24 hours, required fields, file type/size)

---

## Phase 11: Frontend - Approval Workflow

**User stories**: Pending approval list, approve/reject actions, batch operations, edit & return, partial approval

### What to build

Implement the approver dashboard. Managers and Finance see pending entries grouped by user/month, can review individually or batch approve/reject, and use advanced actions like edit & return, partial approval, and delegate.

### Acceptance criteria

- [ ] Pending approvals list grouped by user/month
- [ ] Expand/collapse group to see individual entries
- [ ] Entry detail modal with items list
- [ ] Approve button (single entry)
- [ ] Reject button with comment input
- [ ] Batch select checkboxes
- [ ] Batch approve/reject buttons
- [ ] Edit & Approve modal (edit items then approve)
- [ ] Edit & Return modal (edit items, send back with comment)
- [ ] Partial Approval modal (select approved items, flag others)
- [ ] Delegate dropdown (select another approver at same level)
- [ ] Approval history visible on each entry

---

## Phase 12: Frontend - Contracts & Projects

**User stories**: Create/adopt contracts and projects, view shared catalog, governance model selection

### What to build

Implement the shared resources management UI. Users can view owned/adopted/all contracts and projects, create new ones with governance settings, and adopt shared resources.

### Acceptance criteria

- [ ] Contracts list with tabs: Owned, Adopted, All (shared catalog)
- [ ] Projects list with tabs and filter by contract
- [ ] Create Contract modal: name, km_rate, currency, governance_model, is_shared checkbox
- [ ] Create Project modal: name, type, contract selector, governance_model, is_shared
- [ ] "Adopt" button on shared resources
- [ ] Adopted badge on adopted resources
- [ ] Governance model explanation tooltip
- [ ] Contract/Project detail page
- [ ] Edit button (creates edit request for shared resources)

---

## Phase 13: Frontend - Customer View & Exports

**User stories**: Customer dashboard, admin customer management, CSV downloads

### What to build

Implement the customer experience and export functionality. Customers see their assigned contracts with summaries. Admins can invite customers and assign contracts. Everyone can download CSV exports.

### Acceptance criteria

- [ ] Customer dashboard: list of assigned contracts with monthly totals
- [ ] Customer contract detail: time/expense summary by project
- [ ] Admin: Invite Customer modal (email, contract multi-select)
- [ ] Admin: Customer list with assigned contracts
- [ ] Admin: Add/remove contracts from customer access
- [ ] Export button on monthly summary pages
- [ ] Download CSV triggers browser download
- [ ] Combined export option
- [ ] Export respects role permissions (filtered data)
