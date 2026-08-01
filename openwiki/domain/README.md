# Domain Concepts

## Core Entities

### Users & Organizations

- **User** — A person who can log in. Has `email`, `name`, and a `password_hash`.
- **Organization** — A logical grouping of users (e.g., a company). Has `name`, `slug`.
- **OrganizationMembership** — Joins a User to an Organization with a **Role**. A user can belong to multiple orgs.
- **UserWithMembership** — Aggregate combining User + Membership + Organization (returned by `/auth/me`).

### Roles

Roles enforced by database `CHECK` constraints (the `hr` role was added by
migration `012`):

| Role       | Description                                             |
|------------|---------------------------------------------------------|
| `employee` | Can create/edit/submit time entries and expenses        |
| `manager`  | Can approve/reject entries at the first approval stage  |
| `finance`  | Can approve/reject entries at the second approval stage |
| `hr`       | People/employment data; never holds an approval stage   |
| `customer` | Limited read-only access                                |

### Organization Structure

- **Unit** — A self-referencing hierarchy tree (`parent_unit_id`). Represents departments/teams. Members are added via
  `unit_memberships`.
- **WorkingGroup** — A flat group of users within an org. Members are added via `wg_members`.
- **Unit Member** — A user assigned to a unit with a role and optional `is_primary` flag.

---

## Approval Workflow

The approval workflow is the core business process in Hourglass. Both time
entries and expenses follow the same two-stage approval flow:

```
                ┌──────────┐
                │  DRAFT   │
                └────┬─────┘
                     │ submit
                     ▼
              ┌──────────────┐
              │  SUBMITTED   │
              └──────┬───────┘
                     │ manager approves
                     ▼
          ┌────────────────────┐
          │ PENDING_MANAGER    │  ← manager can also reject here
          └────────┬───────────┘
                   │ finance approves
                   ▼
            ┌──────────────┐
            │PENDING_FINANCE│  ← finance can also reject here
            └───────┬───────┘
                    │ approve
                    ▼
             ┌───────────┐
             │ APPROVED  │
             └───────────┘

Rejection can happen from any pending state (submitted, pending_manager,
pending_finance). Once rejected, the entry goes back to draft.
```

### Approval Actions

| Action            | Description                       | Who Can Perform    |
|-------------------|-----------------------------------|--------------------|
| `submit`          | Submit a draft entry for approval | Employee           |
| `approve`         | Approve at the current stage      | Manager or Finance |
| `reject`          | Reject with a reason              | Manager or Finance |
| `edit_approve`    | Edit and approve simultaneously   | Manager or Finance |
| `edit_return`     | Return for edits                  | Manager or Finance |
| `partial_approve` | Approve part of an entry          | Manager or Finance |
| `delegate`        | Delegate approval to another user | Manager or Finance |

### Approver Resolution

Who approves the **manager stage** is resolved from the entry's activity chain
(ADR-BE-014 R-1/R-2) rather than the org role alone:

- **R-1 (anchored WG):** if the entry's activity anchors a working group, the
  approver set is that WG's `manager_id` + `delegate_ids`. If the entry owner is
  inside that set, submission skips straight to `pending_finance` (D-11 skip).
- **R-2 (commercial without WG):** a commercial activity (one carrying a
  contract via its ancestor chain) with no anchored working group rejects
  submissions with `ErrActivityNotLoggable` — commercial activities must anchor
  a WG before accepting entries.
- **R-2 fallback (personal activity):** an activity with no contract and no WG
  routes to the submitter's unit manager, walking the unit tree upward.
- **Terminal case:** an org root with no unit manager falls back to a
  role-gated manager stage.

Activity parent assignments additionally reject cycles (`ErrActivityCycle`):
a proposed `parent_id` whose ancestry already contains the activity itself is
refused.

### Approval History

Every action on a time entry or expense is recorded in the approval history
table (`time_entry_approvals` / `expense_approvals`) with:

- `action` — The action taken
- `actor_user_id` — Who performed it
- `actor_role` — Their role at the time
- `comment` — Optional note (required for rejection)
- `created_at` — Timestamp

### Status Checks

Database-level `CHECK` constraints enforce valid status transitions
(migration `004` for time entries, `005` for expenses).

---

## Time Entries

Time entries track hours worked by an employee on a specific date against one
or more projects (via `TimeEntryItem`).

**Key fields:**

- `date` — The work date
- `user_id` — Who logged the entry
- `organization_id` — Which org
- `project_id` — Which project (optional)
- `hours` — Total hours
- `description` — Work description
- `status` — Current workflow status
- `current_approver_role` — Who needs to act next
- `submitted_at` — When it was submitted
- `deleted_at` — Soft delete support

**TimeEntryItem** — Individual line items with `project_id`, `hours`, `description`.

**TimeEntryMonthlySummary** — Aggregate for the calendar view, containing daily
summaries, totals per project, and a matrix view.

### Frontend UI

- The time entries page (`/time-entries`) shows a three-tab layout: **List**, **Calendar**, **Export**
- The **Calendar** tab shows a `MiniCalendar` with status-colored dates + an `EntryDetail` side panel
- The **Export** tab uses a shared `ExportForm` component

---

## Expenses

Expenses track costs incurred by employees with support for receipts and
mileage calculation.

**Categories:** `mileage`, `meal`, `accommodation`, `parking`, `travel_tickets`,
`tolls`, `taxi`, `equipment`, `other`

**Key fields:**

- `date` — The expense date
- `category` — Expense type
- `amount` — Monetary amount
- `km_distance` — For mileage expenses (distance in km)
- `activity_id` — Required activity association (migration `011` backfills
  non-project expenses with a per-org "General & Admin" internal activity)
- `unit_id` — Optional unit association
- `status` — Same workflow as time entries
- `description` — Expense description

**ExpenseItem** — Line items with `category`, `amount`, `km_distance`, and optional receipts.

**ExpenseReceipt** — Binary receipt data stored as `receipt_data` (bytea) with `mime_type`, `file_path`, and
`original_filename`.

### Frontend UI

- The expenses page (`/expenses`) shows `ExpenseCalendar` + `ExpenseDetail` side by side
- Supports receipt upload via `FormData` and file input
- Same approval buttons and history components as time entries

---

## Contracts

Contracts define the commercial terms for projects.

**Key fields:**

- `name` — Contract name
- `km_rate` — Rate per km for mileage reimbursement
- `currency` — Currency code (e.g., EUR)
- `customer_id` — Optional linked customer
- `governance_model` — How contract changes are governed
- `created_by_org_id` — The org that created it
- `is_shared` — Whether other orgs can adopt it
- `is_active` — Soft delete flag

**ContractAdoption** — Cross-org sharing. An org can adopt a contract created by another org.

**Governance Models:**

| Model                | Description                          |
|----------------------|--------------------------------------|
| `creator_controlled` | Only the creating org can modify     |
| `unanimous`          | All adopting orgs must agree         |
| `majority`           | Majority of adopting orgs must agree |

**Recalculate Mileage** — A special endpoint (`POST /contracts/{id}/recalculate-mileage`) that recalculates
mileage-based expenses at the contract's `km_rate`.

---

## Projects

Projects are the work items that time entries and expenses are tracked against.

**Key fields:**

- `name` — Project name
- `project_type` / `type` — `billable` or `internal`
- `contract_id` — Optional linked contract
- `customer_id` — Optional linked customer
- `governance_model` — How project changes are governed
- `created_by_org_id` — The org that created it
- `is_shared` — Whether other orgs can adopt it
- `budget_amount` — Optional budget cap
- `financial_cutoff_config` — JSONB configuration for financial cutoffs
- `is_active` — Soft delete flag

**ProjectAdoption** — Cross-org sharing (same pattern as contracts).

**ProjectManagers** — Users assigned as managers for a project. Separate from org-level role.

---

## Customers

Customers are entities that contracts and projects are associated with.

**Key fields:**

- `org_id` — Which org owns this customer record
- `name` — Customer name
- `contact_name`, `email`, `phone` — Contact info
- `vat_number` — VAT / tax ID
- `address` — Physical address
- `is_active` — Soft delete flag

---

## Exports

Date-range export of approved time entries and expenses.

**Formats:**

- CSV
- XLSX (via `tealeg/xlsx` Go library)

**Key features:**

- Date range picker
- Format selector (CSV / XLSX)
- Count endpoints (`CountTimesheets`, `CountExpenses`) for pre-export validation
- Download hook (`web/src/lib/use-download.ts`) for frontend file download

### Frontend

- The `/exports` page uses `ExportForm` (shared component in `web/src/components/exports/`)
- The **Export** tabs in time-entries and expenses pages also embed `ExportForm` with the type pre-set

---

## Invitations

Users can be invited to join an organization.

**Flow:**

1. Admin creates an invitation (`POST /invitations`) — generates a code and a token
2. Invitee validates the code (`GET /invitations/validate/code/{code}`) or token (
   `GET /invitations/validate/token/{token}`)
3. Invitee accepts (`POST /invitations/accept`) — sets their email, username, password, and activates the membership

---

## Password Reset

**Flow:**

1. User requests reset (`POST /auth/password-reset/request` with email)
2. Reset token is stored in the database
3. User verifies with token + new password (`POST /auth/password-reset/verify`)

Rate-limited to 3 requests per 60 seconds per client.

---

## Key Source Files

| Domain         | Backend                                             | Frontend                       |
|----------------|-----------------------------------------------------|--------------------------------|
| Auth           | `/internal/core/services/auth/` + `/internal/auth/` | `/web/src/api/auth.ts`         |
| Time Entries   | `/internal/core/services/time_entry/`               | `/web/src/api/time-entries.ts` |
| Expenses       | `/internal/core/services/expense/`                  | `/web/src/api/expenses.ts`     |
| Contracts      | `/internal/core/services/contract/`                 | `/web/src/api/contracts.ts`    |
| Projects       | `/internal/core/services/project/`                  | `/web/src/api/projects.ts`     |
| Customers      | `/internal/core/services/customer/`                 | `/web/src/api/customers.ts`    |
| Units          | `/internal/core/services/unit/`                     | `/web/src/api/units.ts`        |
| Working Groups | `/internal/core/services/working_group/`            | (frontend embedded)            |
| Exports        | `/internal/core/services/export/`                   | `/web/src/api/exports.ts`      |
| Invitations    | `/internal/core/services/invitation/`               | `/web/src/api/auth.ts`         |
| Password Reset | `/internal/core/services/password_reset/`           | `/web/src/api/auth.ts`         |
| Organizations  | `/internal/core/services/organization/`             | (frontend embedded)            |

### Source: `internal/models/models.go`

All shared data structures, enums, and request/response DTOs are defined in
the `models` package. This is the authoritative source for entity shapes,
status transitions, and validation logic.
entity shapes,
status transitions, and validation logic.
