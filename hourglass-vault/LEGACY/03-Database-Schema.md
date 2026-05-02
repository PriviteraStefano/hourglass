# Database Schema

Complete reference for all PostgreSQL tables in Hourglass.

## Core Tables

### `users`
Stores user accounts across all organizations.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| email | VARCHAR UNIQUE | Login identifier |
| password_hash | VARCHAR | bcrypt hashed |
| name | VARCHAR | Display name |
| is_active | BOOLEAN | Soft delete flag |
| created_at | TIMESTAMP | Account creation |

**Indexes**: `idx_users_email`

---

### `organizations`
Top-level tenant/company containers.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| name | VARCHAR | Company name |
| slug | VARCHAR UNIQUE | URL-friendly identifier |
| created_at | TIMESTAMP | Creation date |

**Indexes**: `idx_organizations_slug`

---

### `organization_memberships`
Many-to-many: users → organizations with role assignment.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| user_id | UUID FK | References users(id) |
| organization_id | UUID FK | References organizations(id) |
| role | VARCHAR CHECK | employee \| manager \| finance \| customer |
| is_active | BOOLEAN | Soft delete |
| invited_by | UUID FK | Who sent invite (nullable) |
| invited_at | TIMESTAMP | Invite timestamp (nullable) |
| activated_at | TIMESTAMP | When user accepted invite (nullable) |

**Indexes**: `idx_organization_memberships_user_id`, `idx_organization_memberships_org_id`

**Constraint**: `UNIQUE (user_id, organization_id)` — one role per user per org

---

### `organization_settings`
Per-organization configuration.

| Column | Type | Notes |
|--------|------|-------|
| organization_id | UUID PK FK | References organizations(id) |
| default_km_rate | DECIMAL | Mileage rate (nullable) |
| currency | VARCHAR | e.g., "USD", "EUR" |
| week_start_day | INTEGER | 0=Sunday, 1=Monday, etc. |
| timezone | VARCHAR | e.g., "UTC", "America/New_York" |
| show_approval_history | BOOLEAN | Display approval chain |
| created_at | TIMESTAMP | Creation date |
| updated_at | TIMESTAMP | Last update |

---

### `customers`
Company contacts linked to contracts.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| organization_id | UUID FK | References organizations(id) |
| company_name | VARCHAR | Legal company name |
| contact_name | VARCHAR | Primary contact (nullable) |
| email | VARCHAR | Contact email (nullable) |
| phone | VARCHAR | Contact phone (nullable) |
| vat_number | VARCHAR | Tax ID (nullable) |
| address | TEXT | Full address (nullable) |
| is_active | BOOLEAN | Soft delete |
| created_at | TIMESTAMP | Creation date |

**Indexes**: `idx_customers_organization_id`

---

## Contracts & Projects

### `contracts`
Billable agreements with projects attached.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| name | VARCHAR | Contract name |
| km_rate | DECIMAL | Mileage rate override |
| currency | VARCHAR | e.g., "USD", "EUR" |
| customer_id | UUID FK | References customers(id) (nullable) |
| governance_model | VARCHAR | creator_controlled \| unanimous \| majority |
| created_by_org_id | UUID FK | References organizations(id) |
| is_shared | BOOLEAN | Available to other orgs |
| is_active | BOOLEAN | Soft delete |
| created_at | TIMESTAMP | Creation date |

**Indexes**: `idx_contracts_created_by_org_id`, `idx_contracts_customer_id`

---

### `contract_adoptions`
Tracks which orgs have adopted (can use) a contract.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| contract_id | UUID FK | References contracts(id) |
| organization_id | UUID FK | References organizations(id) |
| adopted_at | TIMESTAMP | When adoption occurred |

**Indexes**: `idx_contract_adoptions_org_id`

**Constraint**: `UNIQUE (contract_id, organization_id)` — org can't adopt twice

---

### `projects`
Billable/internal work categories under contracts.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| name | VARCHAR | Project name |
| type | VARCHAR CHECK | billable \| internal |
| contract_id | UUID FK | References contracts(id) |
| governance_model | VARCHAR | creator_controlled \| unanimous \| majority |
| created_by_org_id | UUID FK | References organizations(id) |
| is_shared | BOOLEAN | Available to other orgs |
| is_active | BOOLEAN | Soft delete |
| created_at | TIMESTAMP | Creation date |

**Indexes**: `idx_projects_contract_id`, `idx_projects_created_by_org_id`

---

### `project_adoptions`
Tracks which orgs have adopted a project.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| project_id | UUID FK | References projects(id) |
| organization_id | UUID FK | References organizations(id) |
| adopted_at | TIMESTAMP | When adoption occurred |

**Indexes**: `idx_project_adoptions_org_id`

---

### `project_managers`
Assigns managers to specific projects for approval routing.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| project_id | UUID FK | References projects(id) |
| user_id | UUID FK | References users(id) |
| assigned_at | TIMESTAMP | When assigned |

**Constraint**: `UNIQUE (project_id, user_id)`

---

## Time Entries

### `time_entries` (Phase 2 Flattened)
Header record for daily/weekly time submissions.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| organization_id | UUID FK | References organizations(id) |
| user_id | UUID FK | References users(id) |
| status | VARCHAR CHECK | draft \| submitted \| pending_manager \| pending_finance \| approved \| rejected |
| current_approver_role | VARCHAR | manager \| finance (nullable after approval) |
| submitted_at | TIMESTAMP | When submitted (nullable) |
| work_date | DATE | Date of work (flattened) |
| notes | TEXT | Additional notes (nullable) |
| created_at | TIMESTAMP | Creation date |
| updated_at | TIMESTAMP | Last update |

**Indexes**: `idx_time_entries_user_id`, `idx_time_entries_org_id`, `idx_time_entries_status`

---

### `time_entry_items` (Phase 2 Flattened)
Line items within a time entry (hours per project).

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| time_entry_id | UUID FK | References time_entries(id) |
| project_id | UUID FK | References projects(id) |
| hours | DECIMAL | Hours worked |
| notes | TEXT | Item-level notes (nullable) |

**Indexes**: `idx_time_entry_items_entry_id`

---

## Expenses

### `expenses`
Expense submissions (mileage, meals, accommodations).

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| organization_id | UUID FK | References organizations(id) |
| user_id | UUID FK | References users(id) |
| status | VARCHAR CHECK | draft \| submitted \| pending_manager \| pending_finance \| approved \| rejected |
| current_approver_role | VARCHAR | manager \| finance (nullable) |
| category | VARCHAR CHECK | mileage \| meal \| accommodation \| other |
| amount | DECIMAL | Expense amount |
| currency | VARCHAR | e.g., "USD" |
| description | TEXT | Expense details |
| expense_date | DATE | When expense occurred |
| submitted_at | TIMESTAMP | When submitted (nullable) |
| created_at | TIMESTAMP | Creation date |
| updated_at | TIMESTAMP | Last update |

**Indexes**: `idx_expenses_user_id`, `idx_expenses_org_id`, `idx_expenses_status`

---

### `expense_mileage_details`
Additional data for mileage expenses.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| expense_id | UUID FK | References expenses(id) |
| contract_id | UUID FK | References contracts(id) |
| distance_km | DECIMAL | Kilometers traveled |
| rate_per_km | DECIMAL | Applied rate |

---

## Approvals

### `time_entry_approvals`
Immutable approval history for time entries.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| time_entry_id | UUID FK | References time_entries(id) |
| approver_id | UUID FK | References users(id) |
| approver_role | VARCHAR | Role of approver |
| action | VARCHAR | submit \| approve \| reject \| edit_approve \| edit_return \| partial_approve \| delegate |
| reason | TEXT | Approval reason (nullable) |
| created_at | TIMESTAMP | Action timestamp |

**Indexes**: `idx_time_entry_approvals_entry_id`, `idx_time_entry_approvals_approver_id`

---

### `expense_approvals`
Immutable approval history for expenses.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| expense_id | UUID FK | References expenses(id) |
| approver_id | UUID FK | References users(id) |
| approver_role | VARCHAR | Role of approver |
| action | VARCHAR | submit \| approve \| reject \| edit_approve \| edit_return \| partial_approve \| delegate |
| reason | TEXT | Approval reason (nullable) |
| created_at | TIMESTAMP | Action timestamp |

**Indexes**: `idx_expense_approvals_expense_id`, `idx_expense_approvals_approver_id`

---

## Auth & Sessions

### `refresh_tokens`
Long-lived tokens for refreshing JWT access tokens.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| user_id | UUID FK | References users(id) |
| token_hash | VARCHAR | Hashed token for validation |
| expires_at | TIMESTAMP | Token expiration |
| created_at | TIMESTAMP | Creation date |

**Indexes**: `idx_refresh_tokens_user_id`, `idx_refresh_tokens_expires_at`

---

### `verification_tokens`
Tokens for email verification and password resets.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID PK | Primary key |
| user_id | UUID FK | References users(id) |
| token_hash | VARCHAR | Hashed token |
| token_type | VARCHAR | verify_email \| reset_password |
| expires_at | TIMESTAMP | Token expiration |
| created_at | TIMESTAMP | Creation date |

**Indexes**: `idx_verification_tokens_user_id`, `idx_verification_tokens_expires_at`

---

## Key Relationships

```
organizations
  ├─ organization_memberships (users → orgs with roles)
  ├─ organization_settings
  ├─ customers
  ├─ contracts (created_by_org_id)
  │   ├─ contract_adoptions (shared to other orgs)
  │   └─ projects
  │       ├─ project_adoptions (shared to other orgs)
  │       └─ project_managers (assigns reviewers)
  ├─ time_entries (user submitted)
  │   ├─ time_entry_items (linked projects)
  │   └─ time_entry_approvals (immutable history)
  └─ expenses
      ├─ expense_mileage_details (contract reference)
      └─ expense_approvals (immutable history)
```

---

**Next**: [[04-Backend-Patterns]] for handler development patterns.
