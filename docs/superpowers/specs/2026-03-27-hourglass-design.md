# Hourglass Design Specification

**Multi-tenant Time & Expense Tracking Application**

---

## 1. Project Goal

Replace Excel-based time-logging and expense reporting with a web application supporting:

- Daily time entries across multiple projects
- Daily expense entries with receipt uploads and auto-calculated km reimbursement
- Multi-level approval workflows (employee → manager → finance)
- Monthly matrix views for time and expenses
- CSV export endpoints matching existing Excel formats
- Multi-tenant architecture with shared contracts/projects
- Customer view-only access to assigned contracts

---

## 2. Architecture Overview

### Multi-tenant Architecture

Organizations (tenants) are the primary isolation boundary. Each organization has its own users with roles (employee, manager, finance). Data is logically separated by `organization_id` on all core entities.

### Shared Resources

Contracts and Projects exist in a global catalog that organizations can "adopt". When adopted:

- The resource appears in the organization's workspace
- Time/expense entries remain org-scoped (each org sees only their own data)
- Governance model defined at creation: creator-controlled, unanimous approval, or majority approval

### Role-Based Access

Users have exactly one role per organization:

| Role | Permissions |
|------|-------------|
| **Employee** | Submit time/expense entries, view own entries |
| **Manager** | Approve/reject entries with full approval action set (approve, reject, edit & approve, edit & return, partially approve, delegate) |
| **Finance/Admin** | Final approval, manage org settings, invite users, manage customer access |
| **Customer** | View-only access to assigned contracts (hybrid: admin creates account → assigns contracts → customer activates via email) |

### Customer Access

- Admin creates customer account with assigned contracts
- Email activation link (7-day expiration)
- Read-only: can view summaries, download CSV exports for assigned contracts
- No participation in approval workflows (future enhancement)

---

## 3. Data Model

### Core Tables

```sql
organizations
├── id (UUID, PK)
├── name
├── slug (unique identifier)
├── created_at

users
├── id (UUID, PK)
├── email (unique)
├── password_hash
├── name
├── is_active
├── created_at

organization_memberships
├── id (UUID, PK)
├── user_id (FK → users)
├── organization_id (FK → organizations)
├── role (enum: employee, manager, finance, customer)
├── is_active
├── invited_by (FK → users)
├── invited_at
├── activated_at
├── UNIQUE (user_id, organization_id)
```

### Shared Resource Tables

```sql
contracts
├── id (UUID, PK)
├── name
├── km_rate (numeric)
├── currency (varchar)
├── governance_model (enum: creator_controlled, unanimous, majority)
├── created_by_org_id (FK → organizations)
├── is_shared (boolean)
├── is_active (boolean)
├── created_at

projects
├── id (UUID, PK)
├── name
├── type (enum: billable, internal)
├── contract_id (FK → contracts)
├── governance_model (enum: creator_controlled, unanimous, majority)
├── created_by_org_id (FK → organizations)
├── is_shared (boolean)
├── is_active (boolean)
├── created_at

contract_adoptions
├── id (UUID, PK)
├── contract_id (FK → contracts)
├── organization_id (FK → organizations)
├── adopted_at

project_adoptions
├── id (UUID, PK)
├── project_id (FK → projects)
├── organization_id (FK → organizations)
├── adopted_at
```

### Entry Tables

```sql
time_entries
├── id (UUID, PK)
├── user_id (FK → users)
├── organization_id (FK → organizations)
├── date
├── status (enum: draft, submitted, pending_manager, pending_finance, approved, rejected)
├── current_approver_role (enum: manager, finance, nullable)
├── submitted_at
├── created_at
├── updated_at

time_entry_items
├── id (UUID, PK)
├── time_entry_id (FK → time_entries)
├── project_id (FK → projects)
├── hours (numeric)
├── description

expenses
├── id (UUID, PK)
├── user_id (FK → users)
├── organization_id (FK → organizations)
├── date
├── status (enum: draft, submitted, pending_manager, pending_finance, approved, rejected)
├── current_approver_role (enum: manager, finance, nullable)
├── submitted_at
├── created_at
├── updated_at

expense_items
├── id (UUID, PK)
├── expense_id (FK → expenses)
├── project_id (FK → projects)
├── category (enum: mileage, meal, accommodation, other)
├── amount (numeric)
├── km_distance (numeric, nullable)
├── description

expense_receipts
├── id (UUID, PK)
├── expense_item_id (FK → expense_items)
├── file_path
├── original_filename
├── uploaded_at
```

### Approval Tables

```sql
time_entry_approvals
├── id (UUID, PK)
├── time_entry_id (FK → time_entries)
├── approver_id (FK → users)
├── organization_id (FK → organizations)
├── action (enum: approve, reject, edit_approve, edit_return, partial_approve, delegate)
├── changes (JSONB, nullable)
├── comment (text, nullable)
├── created_at

expense_approvals
├── id (UUID, PK)
├── expense_id (FK → expenses)
├── approver_id (FK → users)
├── organization_id (FK → organizations)
├── action (enum: approve, reject, edit_approve, edit_return, partial_approve, delegate)
├── changes (JSONB, nullable)
├── comment (text, nullable)
├── created_at
```

### Governance Tables

```sql
contract_edit_requests
├── id (UUID, PK)
├── contract_id (FK → contracts)
├── requested_by (FK → users)
├── requested_by_org_id (FK → organizations)
├── changes (JSONB)
├── status (enum: pending, approved, rejected)
├── created_at
├── resolved_at

contract_edit_request_votes
├── id (UUID, PK)
├── edit_request_id (FK → contract_edit_requests)
├── org_id (FK → organizations)
├── voter_id (FK → users)
├── vote (enum: approve, reject)
├── comment (text, nullable)
├── created_at

project_edit_requests
├── id (UUID, PK)
├── project_id (FK → projects)
├── requested_by (FK → users)
├── requested_by_org_id (FK → organizations)
├── changes (JSONB)
├── status (enum: pending, approved, rejected)
├── created_at
├── resolved_at

project_edit_request_votes
├── id (UUID, PK)
├── edit_request_id (FK → project_edit_requests)
├── org_id (FK → organizations)
├── voter_id (FK → users)
├── vote (enum: approve, reject)
├── comment (text, nullable)
├── created_at
```

### Customer Access Tables

```sql
customer_contract_access
├── id (UUID, PK)
├── user_id (FK → users)
├── contract_id (FK → contracts)
├── granted_by (FK → users)
├── granted_at
├── UNIQUE (user_id, contract_id)
```

---

## 4. API Endpoints

### Authentication

```
POST /auth/register
  Body: { email, password, name, organization_name?, invite_token? }
  Response: { user, token }

POST /auth/login
  Body: { email, password }
  Response: { user, token }

POST /auth/logout
  Headers: Authorization: Bearer <token>

POST /auth/activate
  Body: { token }
  Response: { user, token }
```

### Organizations

```
POST /organizations
  Body: { name, slug }
  Response: { organization }

GET /organizations/:id
  Response: { organization }

POST /organizations/:id/invite
  Body: { email, role }
  Response: { invitation }

POST /organizations/:id/invite-customer
  Body: { email, contract_ids }
  Response: { invitation }
```

### Contracts

```
GET /contracts
  Query: ?scope=owned|adopted|all
  Response: { contracts[] }

POST /contracts
  Body: { name, km_rate, currency, governance_model, is_shared }
  Response: { contract }

GET /contracts/:id
  Response: { contract }

POST /contracts/:id/adopt
  Response: { adoption }

POST /contracts/:id/request-edit
  Body: { changes }
  Response: { edit_request }

GET /contracts/:id/edit-requests
  Response: { edit_requests[] }

POST /contracts/:id/edit-requests/:request_id/approve
  Response: { contract }

POST /contracts/:id/edit-requests/:request_id/reject
  Body: { reason }
  Response: { edit_request }
```

### Projects

```
GET /projects
  Query: ?scope=owned|adopted|all, ?contract_id
  Response: { projects[] }

POST /projects
  Body: { name, type, contract_id, governance_model, is_shared }
  Response: { project }

GET /projects/:id
  Response: { project }

POST /projects/:id/adopt
  Response: { adoption }

POST /projects/:id/request-edit
  Body: { changes }
  Response: { edit_request }

GET /projects/:id/edit-requests
  Response: { edit_requests[] }

POST /projects/:id/edit-requests/:request_id/approve
  Response: { project }

POST /projects/:id/edit-requests/:request_id/reject
  Body: { reason }
  Response: { edit_request }
```

### Time Entries

```
GET /time-entries
  Query: ?date, ?month, ?year, ?user_id, ?status
  Response: { time_entries[], total }

POST /time-entries
  Body: { date, items: [{ project_id, hours, description }] }
  Response: { time_entry }

GET /time-entries/:id
  Response: { time_entry, items[] }

PUT /time-entries/:id
  Body: { items: [{ project_id, hours, description }] }
  Response: { time_entry }

DELETE /time-entries/:id
  Response: { success }

POST /time-entries/:id/submit
  Response: { time_entry }

POST /time-entries/submit-month
  Body: { month, year }
  Response: { submitted_count, entries[] }

GET /time-entries/pending-approval
  Query: ?page, ?limit, ?user_id, ?month, ?year
  Response: { entries[], total, grouped_by_user_month }

POST /time-entries/batch-approve
  Body: { entry_ids, comment? }
  Response: { entries[] }

POST /time-entries/batch-reject
  Body: { entry_ids, comment }
  Response: { entries[] }

GET /time-entries/pending-approval
  Query: ?page, ?limit
  Response: { entries[], total }

POST /time-entries/:id/approve
  Body: { comment? }
  Response: { time_entry }

POST /time-entries/:id/reject
  Body: { comment }
  Response: { time_entry }

POST /time-entries/:id/edit-approve
  Body: { items, comment? }
  Response: { time_entry }

POST /time-entries/:id/edit-return
  Body: { items, comment }
  Response: { time_entry }

POST /time-entries/:id/partial-approve
  Body: { approved_items, items_needing_changes, comment }
  Response: { time_entry }

POST /time-entries/:id/delegate
  Body: { delegate_to_user_id }
  Response: { time_entry }

GET /time-entries/monthly-summary
  Query: ?user_id, ?month, ?year
  Response: { days[], totals, matrix }
```

### Expenses

```
GET /expenses
  Query: ?date, ?month, ?year, ?user_id, ?status
  Response: { expenses[], total }

POST /expenses
  Body: multipart/form-data { date, items: JSON string, receipts: files }
  Response: { expense }

GET /expenses/:id
  Response: { expense, items[], receipts[] }

PUT /expenses/:id
  Body: multipart/form-data { items: JSON string, receipts: files }
  Response: { expense }

DELETE /expenses/:id
  Response: { success }

POST /expenses/:id/submit
  Response: { expense }

POST /expenses/submit-month
  Body: { month, year }
  Response: { submitted_count, expenses[] }

GET /expenses/pending-approval
  Query: ?page, ?limit, ?user_id, ?month, ?year
  Response: { expenses[], total, grouped_by_user_month }

POST /expenses/batch-approve
  Body: { expense_ids, comment? }
  Response: { expenses[] }

POST /expenses/batch-reject
  Body: { expense_ids, comment }
  Response: { expenses[] }

GET /expenses/pending-approval

POST /expenses/:id/approve
  Body: { comment? }
  Response: { expense }

POST /expenses/:id/reject
  Body: { comment }
  Response: { expense }

POST /expenses/:id/edit-approve
  Body: { items, comment? }
  Response: { expense }

POST /expenses/:id/edit-return
  Body: { items, comment }
  Response: { expense }

POST /expenses/:id/partial-approve
  Body: { approved_items, items_needing_changes, comment }
  Response: { expense }

POST /expenses/:id/delegate
  Body: { delegate_to_user_id }
  Response: { expense }

GET /expenses/monthly-summary
  Query: ?user_id, ?month, ?year
  Response: { days[], totals, categories }
```

### Exports

```
GET /exports/timesheets
  Query: ?month, ?year, ?format (csv), ?user_id?
  Response: CSV file

GET /exports/expenses
  Query: ?month, ?year, ?format (csv), ?user_id?
  Response: CSV file

GET /exports/combined
  Query: ?month, ?year, ?format (csv), ?user_id?
  Response: CSV file
```

### Customer Endpoints

```
GET /customer/contracts
  Response: { contracts[] }

GET /customer/contracts/:id/summary
  Query: ?month, ?year
  Response: { time_summary, expense_summary, entries[] }
```

---

## 5. Approval Workflow Logic

### Approval Chain

**Employee → Manager → Finance**

### Monthly Draft Workflow

Users typically create entries throughout the month without submitting them until the last working day:

1. Employee creates entries in `draft` status throughout the month
2. Multiple drafts accumulate per day/project
3. On last working day (or when ready), employee reviews and submits

### Submission Flow

Two submission modes:

**Single Entry Submission:**
1. Employee opens draft entry
2. Employee clicks "Submit" → status → `submitted`, `submitted_at` set
3. System queries org members with eligible roles
4. Entry enters first pending approval level

**Batch Submission (Submit All for Month):**
1. Employee views monthly summary showing all drafts
2. Employee clicks "Submit All Entries for [Month]"
3. All draft entries for that month transition to `submitted`
4. Each entry enters the approval chain independently

### Batch Review for Approvers

When multiple entries are submitted together:

- Approver sees all pending entries for a user/month grouped together
- Approver can review batch as a unit or expand to individual entries
- Approver can approve all, reject all, or handle individually (hybrid)
- Individual reject within batch: only that entry returns to author
- Batch approval: all entries move to next approval level together

### Approval Actions

| Action | Description | Audit Trail |
|--------|-------------|--------------|
| Approve | Pass to next level or finalize | Actor, timestamp, action |
| Reject | Return to author, status → rejected | Actor, timestamp, comment |
| Edit & Approve | Modify entry, then approve | Actor, timestamp, changes (diff), action |
| Edit & Return | Modify and send back to author for confirmation | Actor, timestamp, changes, action |
| Partially Approve | Approve some items, request changes on others | Actor, timestamp, partial approval details |
| Delegate | Forward to another approver at same level | Actor, timestamp, delegated_to |

### Auto-Skip Logic

If author holds multiple roles, skip levels they outrank:

- Manager author: skip manager level
- Finance author: skip finance level

### Rejection & Return Handling

- **Rejected**: Entry returns to drafts, author can edit and resubmit
- **Edit & Return**: Author confirms or rejects changes; if confirmed, entry continues in chain

### Partial Approval

- Approver marks which items approved, which need changes
- Approved items proceed through remaining levels
- Items needing changes return to author with comments

### State Transitions

```
draft → submitted → pending_manager → pending_finance → approved
                  ↓                    ↓
               rejected            rejected

Author can resubmit rejected entries after editing.
Edit & Return keeps entry in pending state until author confirms.
Partial approval keeps entry pending until all items resolved.
```

---

## 6. Shared Resource Governance

### Governance Models

| Model | Approval Requirement |
|-------|---------------------|
| Creator-controlled | Only creator org + platform admin can approve edits |
| Unanimous | All adopting orgs must approve edits |
| Majority | >50% of adopting orgs must approve edits |

### Edit Request Flow

1. User creates edit request with proposed changes
2. System applies governance model
3. Edit request enters pending state
4. Eligible approvers notified
5. Votes collected
6. Threshold reached:
   - Approved → changes applied
   - Rejected → edit request closed

### Platform Admin

- Global role overriding governance decisions
- Can deprecate shared resources
- Can resolve disputes

---

## 7. File Upload & Receipt Handling

### Storage

- Local path: `/uploads/receipts/{year}/{month}/{uuid}.{ext}`
- File validation: jpg, png, pdf only
- Max size: 10MB

### Upload Flow

- Multipart form-data upload
- UUID-based filename
- Path stored in `expense_receipts.file_path`

### Access

- Only authenticated users with expense access
- Customers can download receipts for their assigned contracts

---

## 8. CSV Export Format

### Timesheet

| Column | Description |
|--------|-------------|
| Date | YYYY-MM-DD |
| Employee | User name |
| Organization | Org name |
| Project | Project name |
| Contract | Contract name |
| Hours | Decimal hours |
| Description | Item description |
| Status | approved/pending/rejected |

### Expense

| Column | Description |
|--------|-------------|
| Date | YYYY-MM-DD |
| Employee | User name |
| Organization | Org name |
| Project | Project name |
| Contract | Contract name |
| Category | mileage/meal/accommodation/other |
| Amount | Decimal currency |
| Km Distance | For mileage entries |
| Description | Item description |
| Status | approved/pending/rejected |

### Combined

Both timesheets and expenses with additional "Type" column (time/expense).

### Export Access Control

| Role | Export Scope |
|------|--------------|
| Finance/Admin | All org entries |
| Manager | Team entries |
| Employee | Own entries |
| Customer | Assigned contracts only |

---

## 9. Security & Validation

### Authentication

- JWT with 24-hour expiration
- Refresh tokens (7-day) in httpOnly cookie
- bcrypt (cost 12) for passwords
- Rate limiting: 5 failed attempts → 15-minute lockout

### Authorization

- Middleware extracts user ID and org membership from JWT
- RBAC per endpoint
- Row-level security: filter by `organization_id`
- Customers filtered to assigned contracts

### Input Validation

| Field | Rules |
|-------|-------|
| Email | Valid format, unique across users |
| Password | Min 8 chars, 1 uppercase, 1 number |
| Time entry hours | Max 24 hours per day per user |
| Expense amounts | Positive decimal, max 2 decimal places |
| File uploads | jpg/png/pdf, max 10MB |

### Data Isolation

- Every org-scoped table includes `organization_id`
- Prepared statements prevent SQL injection
- No cross-tenant data in API responses

---

## 10. Tech Stack

### Backend

- Go 1.26.1 (stdlib net/http)
- database/sql + lib/pq
- bcrypt, golang-jwt/jwt/v5
- Plain SQL migrations
- PostgreSQL 15+ with pgcrypto

### Frontend

- React 19 + TypeScript
- Vite
- Shadcn/UI with BaseUI components
- React Query, React Hook Form + Zod
- TanStack Table
- react-dropzone

### Infrastructure

- Docker (multi-stage, Alpine)
- GitHub Actions → AWS ECS / DO App Platform
- Local file storage (later S3/Supabase)
