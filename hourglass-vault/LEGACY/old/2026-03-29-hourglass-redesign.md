# Hourglass Redesign Specification

**Multi-tenant Time & Expense Tracking Application - Architecture Redesign**

Date: 2026-03-29 | Updated: 2026-03-30

---

## Executive Summary

This document captures a comprehensive redesign of the Hourglass application based on refined user journey analysis and detailed design decisions. Key architectural changes include:

1. **Time Entry Model**: Simplified from multi-project entries to single-project entries
2. **Expense Model**: Flattened from nested structure to standalone entries with multiple receipts (BLOB storage)
3. **Customer Entity**: New entity for contract-to-customer linking
4. **Project-Manager Assignment**: Approval routing based on project managers (no primary designation)
5. **Multi-Role Users**: Support for users holding multiple roles with auto-skip logic
6. **Unified Entry View**: Combined time/expense page with toggles
7. **Organization Settings**: New settings table for configurable defaults
8. **Email Infrastructure**: SMTP for production, console logging for development

---

## 1. Data Model

### 1.1 Time Entries (Redesigned)

**Previous Model:**
```
time_entries (header)
└── time_entry_items (multiple project lines)
```

**New Model:**
```sql
CREATE TABLE time_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    date DATE NOT NULL,
    hours DECIMAL(5,2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    description TEXT,
    status entry_status NOT NULL DEFAULT 'draft',
    current_approver_role approver_role,
    submitted_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- One entry per user per project per date (optional, can be relaxed)
    CONSTRAINT unique_user_project_date UNIQUE (user_id, project_id, date)
);

CREATE INDEX idx_time_entries_user_date ON time_entries(user_id, date);
CREATE INDEX idx_time_entries_status ON time_entries(status);
CREATE INDEX idx_time_entries_project ON time_entries(project_id);
CREATE INDEX idx_time_entries_org ON time_entries(organization_id);
```

**Key Change:** Each time entry is for exactly ONE project. If an employee works on 3 projects in a day, they create 3 separate entries.

### 1.2 Expenses (Redesigned)

**Previous Model:**
```
expenses (header)
└── expense_items (multiple items)
    └── expense_receipts (files)
```

**New Model:**
```sql
CREATE TYPE expense_type AS ENUM (
    'mileage', 'meal', 'accommodation', 
    'parking', 'travel_tickets', 'tolls', 
    'taxi', 'equipment', 'other'
);

CREATE TABLE expenses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    customer_id UUID REFERENCES customers(id),  -- Optional, for reporting
    date DATE NOT NULL,
    type expense_type NOT NULL,
    amount DECIMAL(10,2) NOT NULL CHECK (amount >= 0),
    km_distance DECIMAL(10,2),  -- Required for mileage type
    description TEXT,
    status entry_status NOT NULL DEFAULT 'draft',
    current_approver_role approver_role,
    submitted_at TIMESTAMP,
    deleted_at TIMESTAMP,  -- Soft delete
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT mileage_requires_distance 
        CHECK (type != 'mileage' OR km_distance IS NOT NULL)
);

CREATE INDEX idx_expenses_user_date ON expenses(user_id, date);
CREATE INDEX idx_expenses_status ON expenses(status);
CREATE INDEX idx_expenses_project ON expenses(project_id);
CREATE INDEX idx_expenses_type ON expenses(type);
CREATE INDEX idx_expenses_org ON expenses(organization_id);

-- Multiple receipts per expense (BLOB storage)
CREATE TABLE expense_receipts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id UUID NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
    receipt_data BYTEA NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_expense_receipts_expense ON expense_receipts(expense_id);
```

**Key Changes:**
- Each expense is a standalone entry with one project, one type
- Multiple receipts per expense stored as BLOB
- Soft delete with `deleted_at` column (approval history preserved for audit)

**Type-Specific Fields:**

| Type | Required Fields | Auto-Calculated |
|------|-----------------|-----------------|
| mileage | km_distance | amount = km_distance * contract.km_rate (or org default) |
| meal | amount, receipt | - |
| accommodation | amount, receipt | - |
| parking | amount, receipt | - |
| travel_tickets | amount, receipt | - |
| tolls | amount, receipt | - |
| taxi | amount, receipt | - |
| equipment | amount, receipt | - |
| other | amount | receipt optional |

**Mileage Rate Source:** Contract km_rate → Organization default km_rate → Error if neither configured

### 1.3 Customers (New Entity)

```sql
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    company_name VARCHAR(255) NOT NULL,
    contact_name VARCHAR(255),
    email VARCHAR(255),
    phone VARCHAR(50),
    vat_number VARCHAR(50),
    address TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_customers_organization ON customers(organization_id);
CREATE INDEX idx_customers_active ON customers(is_active);
```

**Contract Update:**
```sql
ALTER TABLE contracts ADD COLUMN customer_id UUID REFERENCES customers(id);
CREATE INDEX idx_contracts_customer ON contracts(customer_id);
```

**Customer Visibility:** Each contract links to ONE customer. When a contract is shared across organizations, the customer is visible to all adopting organizations (read-only for non-owners).

**Customer Edit Permission:** Only the organization that created the customer can edit it.

**Customer Deactivation:** Deactivated customers cannot be linked to new contracts, but existing links are preserved.

### 1.4 Project Managers (New Entity)

```sql
CREATE TABLE project_managers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    UNIQUE (project_id, user_id)
);

CREATE INDEX idx_project_managers_project ON project_managers(project_id);
CREATE INDEX idx_project_managers_user ON project_managers(user_id);
```

**Approval Routing:** When an entry is submitted for a project, all assigned managers for that project can approve it. All managers have equal authority (no primary/secondary distinction in MVP).

**Delegation:** When a manager delegates an approval to a non-manager user, that user becomes a manager for the project. A disclaimer is shown before delegation to non-managers.

### 1.5 Multi-Role Users

**Modified Constraint:**
```sql
-- Previous: UNIQUE (user_id, organization_id)
-- New: Allow multiple roles per user per org
ALTER TABLE organization_memberships 
    DROP CONSTRAINT IF EXISTS organization_memberships_user_id_organization_id_key;

ALTER TABLE organization_memberships 
    ADD CONSTRAINT organization_memberships_user_org_role_key 
    UNIQUE (user_id, organization_id, role);
```

A user can now have multiple rows:
- (user_id: alice, org_id: acme, role: employee)
- (user_id: alice, org_id: acme, role: manager)

**At Least One Role Required:** A membership must have at least one role. Removing the last role row removes the user from the organization.

**Role Hierarchy (Finance = Admin):**
- **Employee**: Create/edit own entries, view own entries, submit for approval, export own entries
- **Manager**: All Employee permissions + approve/reject/edit&approve/edit&return/delegate entries for managed projects, view team entries, export managed project entries
- **Finance** (= Admin): All Manager permissions + final approval, manage users, manage customers, manage contracts/projects, manage org settings, export all entries

**Note:** There is no separate "admin" role in MVP. Finance has full administrative access.

### 1.6 Organization Settings (New Entity)

```sql
CREATE TABLE organization_settings (
    organization_id UUID PRIMARY KEY REFERENCES organizations(id),
    default_km_rate DECIMAL(10,2),  -- Nullable, requires explicit setup
    currency VARCHAR(3) NOT NULL DEFAULT 'EUR',
    week_start_day SMALLINT NOT NULL DEFAULT 1,  -- 0=Sunday, 1=Monday
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    show_approval_history BOOLEAN NOT NULL DEFAULT true,  -- Show approval edits to employees
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Settings:**
- `default_km_rate`: Fallback if contract has no km_rate. Nullable — organization must configure before mileage expenses.
- `currency`: ISO 4217 code, validated. Organization default, contracts can override.
- `week_start_day`: Calendar week start (0=Sunday, 1=Monday).
- `timezone`: Business logic timezone (billing cycles, week boundaries).
- `show_approval_history`: If true, employees can see approval edit history. If false, only managers/finance see it.

### 1.7 Approval Tables (Updated)

```sql
CREATE TABLE time_entry_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    time_entry_id UUID NOT NULL REFERENCES time_entries(id),
    approver_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    action approval_action NOT NULL,
    changes JSONB,  -- Field-level diff: {"hours": {"from": 8, "to": 7.5}}
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE expense_approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expense_id UUID NOT NULL REFERENCES expenses(id),
    approver_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    action approval_action NOT NULL,
    changes JSONB,  -- Field-level diff: {"amount": {"from": 50, "to": 45}}
    comment TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Removed: partial_approve action (not needed for flat entries)
CREATE TYPE approval_action AS ENUM (
    'submit', 'approve', 'reject', 
    'edit_approve', 'edit_return', 'delegate',
    'rate_recalculate'  -- New: logged when km_rate changes affect expense amount
);

-- Removed: 'submitted' status (goes directly to pending_manager)
CREATE TYPE entry_status AS ENUM (
    'draft', 'pending_manager', 'pending_finance', 
    'approved', 'rejected'
);

CREATE TYPE approver_role AS ENUM ('manager', 'finance');
```

### 1.8 Time Entries (Updated Schema)

```sql
CREATE TABLE time_entries_new (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    project_id UUID NOT NULL REFERENCES projects(id),
    date DATE NOT NULL,
    hours DECIMAL(5,2) NOT NULL CHECK (hours > 0 AND hours <= 24),
    description TEXT,
    status entry_status NOT NULL DEFAULT 'draft',
    current_approver_role approver_role,
    submitted_at TIMESTAMP,
    deleted_at TIMESTAMP,  -- Soft delete
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Constraint: Total hours per user per day <= 24
-- Application-level validation, not database constraint

CREATE INDEX idx_time_entries_user_date ON time_entries(user_id, date);
CREATE INDEX idx_time_entries_status ON time_entries(status);
CREATE INDEX idx_time_entries_project ON time_entries(project_id);
CREATE INDEX idx_time_entries_org ON time_entries(organization_id);
```

### 1.9 Entity Relationship Diagram

```
organizations
├── users (via memberships)
├── customers (1:N)
│   └── contracts (1:1)
│       └── projects (1:N)
│           ├── project_managers (M:N with users)
│           ├── time_entries (1:N)
│           └── expenses (1:N)
├── time_entries (direct scope)
└── expenses (direct scope)
```

---

## 2. User Roles & Permissions

### 2.1 Roles

| Role | Permissions |
|------|-------------|
| **Employee** | Create/edit/delete own draft entries, view own entries, submit for approval, export own entries (limited to own data) |
| **Manager** | All Employee permissions + approve/reject/edit&approve/edit&return/delegate entries for managed projects, view team entries on managed projects, bulk edit/approve/reject, export managed project entries |
| **Finance** (Admin) | All Manager permissions + final approval, manage users, manage customers, manage contracts, manage projects, manage org settings, view all entries, export all data, recalculate mileage rates |

**Note:** Finance role is equivalent to Admin. There is no separate Admin role.

**Customer Role:** Exists in code but unused in MVP. Customers are external entities without login access.

### 2.2 Multi-Role Behavior

Users can hold multiple roles simultaneously. The system automatically handles approval level skipping:

| User Roles | Own Entry Approval Path |
|------------|------------------------|
| Employee only | Manager → Finance |
| Manager only | Finance (skip manager) |
| Finance only | Auto-approve (skip both) |
| Employee + Manager | Finance (skip manager) |
| Employee + Finance | Auto-approve (skip both) |
| Manager + Finance | Auto-approve (skip both) |
| Employee + Manager + Finance | Auto-approve (skip both) |

**Implementation:**
```go
func (s *Service) getRequiredApprovalLevels(userRoles []Role) []Role {
    allLevels := []Role{RoleManager, RoleFinance}
    var required []Role
    for _, level := range allLevels {
        if !containsRole(userRoles, level) {
            required = append(required, level)
        }
    }
    return required
}
```

### 2.3 Project Access

- **Open Access:** Any organization member can create time entries and expenses on any project accessible to their org (created by org, shared, or adopted).
- **Project Adoption (Finance Only):** For shared projects, only Finance users can adopt the project for their organization. Members then log time/expenses on adopted projects.

### 2.4 Shared Resources Governance

- **Shared Contracts/Projects:** Only the creating organization can edit shared resources (creator-controlled).
- **Projects Adopted Separately:** Adopting a contract does NOT automatically adopt its projects. Each project must be adopted separately.
- **Customer Visibility:** Customers on shared contracts are visible to all adopting organizations (read-only for non-owners).

---

## 3. User Journeys

### 3.1 Employee Journey

#### 3.1.1 View Entries (Unified Page)

**Page:** `/entries?view=time&type=calendar` (URL-based toggles)

**Layout:**
```
┌─────────────────────────────────────────────────────────────┐
│  My Entries                              [Org Selector ▼]    │
│                                          [User Menu]         │
├─────────────────────────────────────────────────────────────┤
│  [Time Entries] [Expenses]              [Calendar] [List]   │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   March 2026                           [+ Add Entry]        │
│   Su Mo Tu We Th Fr Sa                                      │
│      1  2  3  4  5  6                                      │
│    7  8  9 10 11 12 13      Legend:                         │
│   14 15 16 17 18 19 20       ● 3 entries (click to view)  │
│   21 22 23 24 25 26 27      ● Draft (gray)                 │
│   28 29 30 31               ● Pending (yellow)              │
│                             ● Approved (green)              │
│                             ● Rejected (red)                │
└─────────────────────────────────────────────────────────────┘
```

**Calendar View:**
- Number indicator shows entry count per day
- Click opens side panel (drawer) with all entries for that day
- Color indicates "worst" status in cell (rejected > pending > draft > approved)

**List View:**
- Table with columns: Date, Project, Hours/Amount, Status, Actions
- Filters: date range (`from`, `to`), project, status, search (description)
- Pagination: `?page=1&per_page=50`

**Toggle Behavior:**
- **Time/Expense Toggle:** URL param `?view=time` or `?view=expense`
- **Calendar/List Toggle:** URL param `?type=calendar` or `?type=list`

#### 3.1.2 Create Time Entry

**Trigger:** Click day on calendar OR "Add Entry" button

**Form:**
```
┌─────────────────────────────────────────────┐
│  New Time Entry                        [X]  │
├─────────────────────────────────────────────┤
│                                             │
│  Date *         [2026-03-15        ▼]       │
│                                             │
│  Project *      [Select project... ▼]       │
│                                             │
│  Hours *        [____] (max 24)             │
│                                             │
│  Description    [________________]          │
│                                             │
│              [Cancel]  [Save as Draft]      │
└─────────────────────────────────────────────┘
```

**Validations:**
- Date: Required, cannot be future date (configurable)
- Project: Required, must be active project user has access to
- Hours: Required, decimal > 0 and <= 24
- Combined total per date <= 24 hours
- Description: Optional, max 500 characters

**After Save:**
- Entry status = `draft`
- User sees entry in calendar/list with draft indicator
- Can edit anytime until submitted

#### 3.1.3 Create Expense

**Form (varies by type):**
```
┌─────────────────────────────────────────────┐
│  New Expense                           [X]  │
├─────────────────────────────────────────────┤
│                                             │
│  Date *         [2026-03-15        ▼]       │
│                                             │
│  Project *      [Select project... ▼]       │
│                                             │
│  Type *         [Mileage        ▼]          │
│                                             │
│  ─── Type-specific fields ───               │
│  Distance (km) * [____]                     │
│  Amount: €0.45/km × ___ km = €__.->__       │
│                                             │
│  Description    [________________]          │
│                                             │
│              [Cancel]  [Save as Draft]      │
└─────────────────────────────────────────────┘
```

**Type-Specific UI:**
- **Mileage:** Shows km input, auto-calculates amount from contract rate
- **All others:** Shows amount input, receipt upload (required except 'other')

**Receipt Upload:**
- Drag-drop zone
- Accepts: JPG, PNG, PDF
- Max size: 10MB
- Preview for images

#### 3.1.4 Submit for Approval

**Single Entry:**
- From entry detail view: "Submit" button
- Confirmation: "Submit this entry for approval?"
- Status → `pending_manager`

**Batch Submit (Month):**
- From monthly view
- "Submit All for March 2026" button
- Shows draft count: "Submit 5 draft entries for March 2026?"
- All drafts for that month → `pending_manager`

**Post-Submission:**
- Entry cannot be edited
- User can view status
- Entry visible in pending approvals for assigned managers

#### 3.1.5 Edit Entry

**Draft Entries:** Can edit project, hours/distance, description. Project is immutable after creation — if wrong project, delete and recreate, or create new entry and delete old.

**Rejected Entries:**
1. Modal/banner shows rejection reason and approver comment
2. User clicks "Edit Entry" (dismissable)
3. Disclaimer shown underneath edit form while editing
4. Status changes to `draft` when editing starts
5. User can edit and resubmit

**Submitted/Pending/Approved Entries:** Cannot edit.

#### 3.1.6 Delete Entry

**Draft Entries:** Soft delete with `deleted_at`. Approval history preserved for audit trail.

**Rejected Entries:** Can delete (option for user to discard rather than fix).

**Submitted/Pending/Approved Entries:** Cannot delete.

### 3.2 Manager Journey

#### 3.2.1 View Own Entries

Same as Employee journey. Manager's own entries auto-skip manager approval level.

#### 3.2.2 Approval Dashboard

**Page:** `/approvals`

**Layout:**
```
┌─────────────────────────────────────────────────────────────┐
│  Approvals                                                  │
├─────────────────────────────────────────────────────────────┤
│  [Time Entries] [Expenses]                                  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ▼ March 2026 - John Smith (3 entries)                     │
│    ┌─────────────────────────────────────────────────────┐ │
│    │ [✓] 2026-03-01 | Project Alpha   |  8.0h | Pending │ │
│    │ [✓] 2026-03-02 | Project Beta    |  4.0h | Pending │ │
│    │ [✓] 2026-03-03 | Project Alpha   |  6.5h | Pending │ │
│    │                     [Approve Selected] [Reject Selected] │
│    └─────────────────────────────────────────────────────┘ │
│                                                             │
│  ▼ March 2026 - Jane Doe (2 entries)                       │
│    ┌─────────────────────────────────────────────────────┐ │
│    │ [ ] 2026-03-05 | Project Gamma  |  8.0h | Pending │ │
│    │ [ ] 2026-03-06 | Project Gamma  |  7.5h | Pending │ │
│    └─────────────────────────────────────────────────────┘ │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Grouping:** By Month → By Employee → List of entries

**Entry Actions:**
- Click entry row → Detail modal with full info
- Checkbox for batch selection

#### 3.2.3 Approval Actions

**Approve:**
- Single: Click "Approve" on entry detail
- Batch: Select multiple entries across employees → "Approve Selected"
- Entry moves to next level (finance) or final approval
- Comment optional

**Reject:**
- Modal: "Reason for rejection" (required)
- Single or batch — same comment applies to all
- Entry → `rejected` status
- Employee sees status change and rejection reason
- Employee can edit and resubmit

**Edit & Approve:**
- Manager opens entry
- Modifies hours/description/project
- Clicks "Edit & Approve"
- Entry updated and moves to next level
- Changes logged in approval history as field-level diff: `{"hours": {"from": 8, "to": 7.5}}`
- Employee sees edit in approval history (if org setting `show_approval_history` is true)

**Bulk Edit & Approve:**
- Select multiple entries
- Choose field to edit (project, hours, description)
- Set new value (replace for all selected)
- Click "Edit & Approve Selected"
- All entries updated and advanced in one action
- Audit trail shows bulk edit

**Edit & Return:**
- Manager opens entry
- Modifies hours/description
- Clicks "Edit & Return"
- Modal: "Message to employee" (required)
- Entry → `draft` status with changes applied
- Employee sees modified entry, must resubmit

**Delegate:**
- Select user from dropdown (any org member)
- If user is not a manager on the project:
  - Show disclaimer: "This user will become a manager for this project. Continue?"
  - User becomes project manager on confirmation
- Entry visible in delegate's approval queue
- Delegation logged in approval history

### 3.3 Finance Journey

**Note:** Finance role = Admin role. There is no separate Admin role.

#### 3.3.1 View Own Entries

Finance user's own entries auto-approve immediately (skip both manager and finance levels). Audit trail still records submission and auto-approval.

#### 3.3.2 Final Approval

Same interface as Manager approval dashboard, but sees entries in `pending_finance` status.

After finance approval: Entry → `approved`

#### 3.3.3 Rate Recalculation

When contract km_rate is updated:
1. Finance sees option: "Recalculate affected mileage expenses?"
2. If yes: Select date to recalculate from
3. All mileage expenses (draft, pending, approved) from that date are recalculated
4. Recalculation logged in approval history: `"action": "rate_recalculate", "changes": {"amount": {"from": 45, "to": 50}}`
5. Entries keep their current status (no reset to draft)

#### 3.3.4 Admin Settings

**Page:** `/settings`

**Sections:**

**1. Users**
```
┌─────────────────────────────────────────────────────────────┐
│  Settings > Users                              [+ Invite]   │
├─────────────────────────────────────────────────────────────┤
│  Name           Email              Roles        Status      │
│  John Smith     john@acme.com      Manager      Active      │
│  Jane Doe       jane@acme.com      Employee     Active      │
│  Bob Wilson     bob@acme.com       Finance      Inactive    │
└─────────────────────────────────────────────────────────────┘
```

**Invite User Modal:**
- Email (required)
- Role checkboxes: [ ] Employee [ ] Manager [ ] Finance — select one or more
- Sends activation link (7-day expiration)
- Email verification required before account activation

**User Management:**
- Cannot remove last Finance user from org (enforced by application)
- Deactivation sets membership `is_active = false`
- User can have multiple roles per org

**2. Customers**
```
┌─────────────────────────────────────────────────────────────┐
│  Settings > Customers                          [+ Add]      │
├─────────────────────────────────────────────────────────────┤
│  Company          Contact        Contracts    Status        │
│  Contoso Ltd      John Contoso   2            Active        │
│  Fabrikam Inc     Jane Fabrikam  1            Active        │
└─────────────────────────────────────────────────────────────┘
```

**Customer Detail:**
- Company Name
- Contact Name
- Email
- Phone
- VAT Number
- Address
- Linked Contracts (multi-select)

**3. Contracts**
Existing implementation, add:
- Customer assignment dropdown
- Manager count indicator

**4. Projects**
Existing implementation, add:
- Manager assignment (multi-select)
- Shows linked customer from contract

**5. Organization Settings**
```
┌─────────────────────────────────────────────────────────────┐
│  Settings > Organization                                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Default Km Rate:    [0.45] €/km                            │
│  Currency:           [EUR ▼]                                │
│  Week Starts On:     [Monday ▼]                             │
│  Timezone:           [Europe/Rome ▼]                        │
│  Show Approval History to Employees: [✓]                    │
│                                                             │
│  [Save Settings]                                             │
└─────────────────────────────────────────────────────────────┘
```

**6. Exports**
```
┌─────────────────────────────────────────────────────────────┐
│  Settings > Export Data                                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Month: [March 2026 ▼]                                      │
│                                                             │
│  [Export Timesheets CSV]    [Export Expenses CSV]           │
│                                                             │
│  [Export Combined Report]                                   │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**CSV Exports:**
- Date range filter: `?from=YYYY-MM-DD&to=YYYY-MM-DD` (defaults to current month)
- Access scope: Employees export own entries, Managers export managed projects, Finance exports all
- Flat list format: `Date, Employee, Project, Contract, Customer, Hours/Amount, Description, Status`

**Timesheet CSV:**
`Date, Employee, Project, Contract, Customer, Hours, Description, Status`

Expense columns:
`Date, Employee, Project, Contract, Customer, Type, Amount, Km Distance, Description, Status`

### 3.4 Customer Journey (Future Phase)

Customers are created by Admin but do not have login access in MVP.

**Future capabilities:**
- Customer receives login link
- Customer can view summary reports for their assigned contracts
- Customer can export data for their contracts

---

## 4. Approval Routing Logic

### 4.1 Determining Approvers

```
Entry Submitted
    │
    ├── Get project_id from entry
    │
    ├── Query project_managers WHERE project_id = ?
    │   └── Returns: [manager1, manager2, ...]
    │
    ├── If no managers found:
    │   └── Skip manager level → go to finance
    │
    └── Entry visible to ALL project managers
        └── First manager to act wins
```

### 4.2 Approval Chain Flow

**Note:** There is no `submitted` status. Entries go directly from `draft` to `pending_manager`.

```
┌──────────┐     ┌──────────────────┐     ┌─────────────────┐     ┌──────────┐
│  Draft   │────▶│ pending_manager  │────▶│ pending_finance │────▶│ Approved │
└──────────┘     └──────────────────┘     └─────────────────┘     └──────────┘
                         │                        │
                         │                        │
                         ▼                        ▼
                    ┌──────────┐            ┌──────────┐
                    │ Rejected │◀───────────│ Rejected │
                    └──────────┘            └──────────┘
```

**Status Transitions:**
- `draft` → `pending_manager`: User submits
- `pending_manager` → `pending_finance`: Manager approves
- `pending_manager` → `rejected`: Manager rejects
- `pending_manager` → `draft`: Manager edit-returns
- `pending_finance` → `approved`: Finance approves
- `pending_finance` → `rejected`: Finance rejects
- `pending_finance` → `draft`: Finance edit-returns
- Any status → soft delete: Admin deletes (audit trail preserved)

### 4.3 Auto-Skip Logic

```go
func (s *Service) ProcessSubmission(entryID uuid.UUID, authorRoles []Role) error {
    entry := s.getEntry(entryID)
    
    approvalLevels := []Role{RoleManager, RoleFinance}
    
    // Remove levels where author has the role
    var requiredLevels []Role
    for _, level := range approvalLevels {
        hasRole := false
        for _, r := range authorRoles {
            if r == level {
                hasRole = true
                break
            }
        }
        if !hasRole {
            requiredLevels = append(requiredLevels, level)
        }
    }
    
    if len(requiredLevels) == 0 {
        // Auto-approve
        entry.Status = StatusApproved
    } else {
        // Set first required level
        entry.Status = StatusPendingManager
        entry.CurrentApproverRole = string(requiredLevels[0])
    }
    
    return s.saveEntry(entry)
}
```

### 4.4 Edge Cases

| Scenario | Resolution |
|----------|------------|
| Project has no managers | Skip manager level, go to finance |
| Org has no finance users | Prevent removing last Finance user; cannot occur |
| User is employee + manager | Skip manager level, requires finance |
| User is manager + finance | Auto-approve (skip both levels) |
| User is employee + finance | Auto-approve (skip both levels) |
| User is employee + manager + finance | Auto-approve (skip both levels) |
| Project is deactivated | No new entries allowed; existing entries unaffected |
| Contract is deactivated | No new projects allowed; existing projects active |
| Customer is deactivated | Cannot link to new contracts; existing links preserved |
| Contract has no km_rate | Use org default_km_rate; error if neither configured |
| Shared project not adopted | Not visible to org; finance must adopt first |

### 4.5 Mileage Rate Changes

When contract km_rate is updated by finance:

1. Finance sees prompt: "Recalculate mileage expenses from a specific date?"
2. If confirmed, select start date
3. All mileage expenses (draft + submitted) from that date recalculated
4. Recalculation logged in approval history
5. Entries keep current status (no reset)

---

## 5. API Endpoints

### 5.1 Time Entries

```
GET    /v1/time-entries
        Header: Accept: application/vnd.hourglass.v1+json
        Query: ?page=1&per_page=50
               &date=YYYY-MM-DD
               &month=M&year=YYYY
               &status=draft|pending_manager|pending_finance|approved|rejected
               &project_id=UUID
               &from=YYYY-MM-DD&to=YYYY-MM-DD
               &search=text
        Response: { data: { entries: [...], total: N, page: 1, per_page: 50 } }

POST   /v1/time-entries
        Header: Accept: application/vnd.hourglass.v1+json
        Body: { project_id, date, hours, description }
        Response: { data: { entry } }
        Validation:
          - date: required, cannot be future
          - project_id: required, must be active and accessible
          - hours: required, decimal > 0 and <= 24
          - combined total per user per day <= 24

GET    /v1/time-entries/:id
        Response: { data: { entry } }

PUT    /v1/time-entries/:id
        Body: { project_id, hours, description }
        Response: { data: { entry } }
        Note: Only drafts can be updated

DELETE /v1/time-entries/:id
        Note: Drafts and rejected can be deleted (soft delete)

POST   /v1/time-entries/:id/submit
        Response: { data: { entry } }
        Note: Status changes from 'draft' to 'pending_manager' (or auto-approves)

### 5.2 Expenses

```
GET    /v1/expenses
        Header: Accept: application/vnd.hourglass.v1+json
        Query: ?page=1&per_page=50
               &date=YYYY-MM-DD
               &month=M&year=YYYY
               &status=draft|pending_manager|pending_finance|approved|rejected
               &project_id=UUID
               &type=mileage|meal|accommodation|parking|travel_tickets|tolls|taxi|equipment|other
               &from=YYYY-MM-DD&to=YYYY-MM-DD
               &search=text
        Response: { data: { expenses: [...], total: N, page: 1, per_page: 50 } }

POST   /v1/expenses
        Header: Accept: application/vnd.hourglass.v1+json
        Body: multipart/form-data
              { project_id, date, type, amount, km_distance?, description }
              Files: receipt[] (multiple files supported)
        Response: { data: { expense } }
        Note: Mileage expenses: amount calculated from km_distance * contract.km_rate (or org default)

GET    /v1/expenses/:id
        Response: { data: { expense } }

PUT    /v1/expenses/:id
        Body: multipart/form-data { ... }
        Note: Only drafts can be updated

DELETE /v1/expenses/:id
        Note: Drafts and rejected can be deleted (soft delete)

POST   /v1/expenses/:id/submit
        Response: { data: { expense } }

GET    /v1/expenses/:id/receipts/:receipt_id
        Response: File download (receipt data as BLOB)
```

### 5.3 Approvals

```
GET    /v1/approvals/time-entries
        Query: ?status=pending_manager|pending_finance&page=1&per_page=50
        Response: { data: { groups: [{ user, month, year, entries: [...] }] } }

GET    /v1/approvals/expenses
        Query: ?status=pending_manager|pending_finance&page=1&per_page=50
        Response: { data: { groups: [...] } }

POST   /v1/approvals/time-entries/:id/approve
        Body: { comment? }
        Response: { data: { entry } }

POST   /v1/approvals/time-entries/:id/reject
        Body: { comment }
        Response: { data: { entry } }

POST   /v1/approvals/time-entries/:id/edit-approve
        Body: { project_id?, hours?, description?, comment? }
        Response: { data: { entry } }

POST   /v1/approvals/time-entries/:id/edit-return
        Body: { project_id?, hours?, description?, comment }
        Response: { data: { entry } }

POST   /v1/approvals/time-entries/:id/delegate
        Body: { delegate_to_user_id }
        Response: { data: { entry } }

POST   /v1/approvals/time-entries/bulk-edit-approve
        Body: { entry_ids: [...], field: "project_id"|"hours"|"description", value: "...", comment? }
        Response: { data: { entries: [...] } }

POST   /v1/approvals/time-entries/batch-reject
        Body: { entry_ids: [...], comment }
        Response: { data: { entries: [...] } }

-- Same endpoints for expenses with /v1/approvals/expenses/... --
```

### 5.4 Customers

```
GET    /v1/customers
        Query: ?page=1&per_page=50
        Response: { data: { customers: [...] } }

POST   /v1/customers
        Body: { company_name, contact_name?, email?, phone?, vat_number?, address? }
        Response: { data: { customer } }

GET    /v1/customers/:id
        Response: { data: { customer, contracts: [...] } }

PUT    /v1/customers/:id
        Body: { company_name?, contact_name?, email?, phone?, vat_number?, address?, is_active? }
        Response: { data: { customer } }
        Note: Only creating org can edit

DELETE /v1/customers/:id
        Note: Cannot delete if linked to contracts (409 Conflict)
```

### 5.5 Organization Management

```
GET    /v1/organizations/:id
        Response: { data: { organization, settings } }

GET    /v1/organizations/:id/members
        Query: ?page=1&per_page=50
        Response: { data: { members: [{ user, roles: [...] }] } }

POST   /v1/organizations/:id/invite
        Body: { email, roles: [...] }
        Response: { data: { invitation } }
        Note: Email sent with 7-day activation link

PUT    /v1/organizations/:id/members/:user_id/roles
        Body: { roles: [...] }
        Response: { data: { membership } }
        Note: Cannot remove last role; cannot remove last Finance user

DELETE /v1/organizations/:id/members/:user_id
        Response: { data: { success: true } }
        Note: Soft deactivation (is_active = false)

POST   /v1/auth/switch-org
        Body: { organization_id }
        Response: { data: { user, organization, role }, access_token, refresh_token }
        Note: Issues new JWT for switched org

POST   /v1/auth/forgot-password
        Body: { email }
        Note: Sends reset link (1-hour expiration)

POST   /v1/auth/reset-password
        Body: { token, password }
        Note: Password requirements: min 8 chars (entropy check planned)
```

### 5.6 Project Managers

```
GET    /v1/projects/:id/managers
        Response: { data: { managers: [{ user, created_at }] } }
        Note: No "is_primary" field in MVP

POST   /v1/projects/:id/managers
        Body: { user_id }
        Response: { data: { project_manager } }

DELETE /v1/projects/:id/managers/:user_id
        Response: { data: { success: true } }
```

### 5.7 Exports

```
GET    /v1/exports/timesheets
        Query: ?from=YYYY-MM-DD&to=YYYY-MM-DD
        Header: Accept: text/csv
        Response: text/csv (download)
        Note: Scope limited to user's access (own entries / managed projects / all)

GET    /v1/exports/expenses
        Query: ?from=YYYY-MM-DD&to=YYYY-MM-DD
        Header: Accept: text/csv
        Response: text/csv (download)

GET    /v1/exports/combined
        Query: ?from=YYYY-MM-DD&to=YYYY-MM-DD
        Header: Accept: text/csv
        Response: text/csv (download)
```

### 5.8 Health Check

```
GET    /v1/health
        Response: { "status": "ok" }
```

---

## 6. Frontend Architecture

### 6.1 Tech Stack

| Layer | Technology | Notes |
|-------|------------|-------|
| UI Framework | React 19 | Current |
| Routing | TanStack Router v1 (file-based) | Current |
| State Management | TanStack React Query v5 | Current |
| Forms | react-hook-form + zod | Current |
| Styling | Tailwind CSS | Current |
| UI Components | shadcn + lucide-react | Current |
| Date Handling | date-fns | Locale-aware formatting |

### 6.2 State Management

- **Server State:** TanStack Query with cache invalidation
- **Form State:** react-hook-form with zod validation
- **URL State:** TanStack Router params (toggles, filters, pagination)
- **Local State:** React useState/useReducer for UI-only state

**Org Switching:** Call `/auth/switch-org`, receive new JWT, clear query cache, refetch profile. No page reload.

### 6.3 Error Handling

- **General Errors:** Toast notification (sonner)
- **Form Validation Errors:** Inline field errors
- **Render Errors:** React error boundary (fallback UI)

### 6.4 Loading States

- **Initial Page Load:** Skeleton loaders (matching content structure)
- **Mutations:** Optimistic updates with rollback on error
- **Background Refetch:** Silent refresh, no loading indicator

### 6.5 New Routes

```
/entries           - Unified time/expense page
  ?view=time|expense
  ?type=calendar|list
  ?from=YYYY-MM-DD&to=YYYY-MM-DD
  ?project_id=UUID
  ?search=text

/approvals         - Manager/Finance approval dashboard
  ?view=time|expense
  ?status=pending_manager|pending_finance

/settings          - Admin settings
  /settings/users
  /settings/customers
  /settings/contracts
  /settings/projects
  /settings/organization
  /settings/exports
```

### 6.6 Deprecate Routes

```
/time-entries      - Merged into /entries
```

### 6.3 Key Components

**EntryCalendar**
- Renders calendar grid with status colors
- Click handlers for day selection

**EntryList**
- DataTable with sorting/filtering
- Status badges
- Quick actions (edit, delete, submit)

**TimeEntryForm**
- Project dropdown
- Hours input (max 24)
- Description textarea

**ExpenseForm**
- Type radio/select
- Conditional fields based on type
- Receipt upload for applicable types

**ApprovalGroups**
- Expandable/collapsible groups by user/month
- Checkbox selection for batch actions

**CustomerForm**
- Business info fields
- Contract multi-select

**ProjectManagerSelect**
- User multi-select filtered by manager+ roles
- Shows warning when delegating to non-manager

---

## 7. Backend Architecture

### 7.1 API Versioning

- **Header-based:** `Accept: application/vnd.hourglass.v1+json`
- **Default:** v1 if header not specified
- **Response Content-Type:** Same as request

### 7.2 Rate Limiting

| Request Type | Limit | Scope |
|--------------|-------|-------|
| Anonymous (login, register, forgot-password) | 10 requests/minute | Per IP |
| Authenticated | 100 requests/minute | Per user |

**Implementation:** Middleware-based. Redis for distributed rate limiting (future).

### 7.3 Request Logging

- **Fields:** Method, path, status, duration
- **Format:** Structured text logs
- **Future:** Request ID tracing, JSON structured logs

### 7.4 CORS

- **Configuration:** `ALLOWED_ORIGINS` environment variable (comma-separated)
- **Default:** `http://localhost:3000` (dev only)
- **Production:** Must be explicitly set
- **Credentials:** Allowed for all origins

### 7.5 Database Connection Pool

| Setting | Value |
|---------|-------|
| MaxOpenConns | 25 |
| MaxIdleConns | 5 |
| ConnMaxLifetime | 5 minutes |

### 7.6 Email Infrastructure

| Environment | Behavior |
|-------------|----------|
| Production | SMTP (configurable via env vars: SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASS) |
| Development | Console/file logging |

**Email Format:** Multipart (HTML + plain text)

**Email Types:**
- Invitation (7-day expiration)
- Password reset (1-hour expiration)
- Email verification

### 7.7 Pagination

- **Default:** 50 items per page
- **Max:** 100 items per page
- **Params:** `?page=1&per_page=50`
- **Response:** `{ data: [...], total: N, page: 1, per_page: 50 }`

### 7.8 Filtering & Search

| Filter | Endpoints | Implementation |
|--------|-----------|----------------|
| `?project_id=` | time-entries, expenses | Exact match |
| `?type=` | expenses | Exact match (enum) |
| `?from=&to=` | time-entries, expenses, exports | Date range |
| `?status=` | time-entries, expenses, approvals | Exact match (enum) |
| `?search=` | time-entries, expenses | `ILIKE '%term%'` on description |

### 7.9 Sorting

- **Default sort per endpoint** (no custom sorting in MVP)
- Time entries: date DESC
- Expenses: date DESC
- Approvals: date DESC

---

## 8. Authentication & Authorization

### 8.1 User Registration

**Self-Registration:**
- Enabled by default
- Creates user + organization with Finance role
- Rate limited: 5 registrations per IP per hour
- Email verification required before account activation

**Flow:**
1. User submits email + password + name + org name
2. User created with `is_active = false`
3. Verification email sent
4. User clicks link, sets password, `is_active = true`

### 8.2 User Invitation

**Org Invitation (Finance only):**
1. Finance enters email + role(s)
2. Pending membership created with `invited_at`
3. Invitation email sent (7-day expiration)
4. User clicks link, sets password, `activated_at` set

### 8.3 Password Requirements

- **Minimum:** 8 characters
- **Planned:** Entropy check for stronger passwords

### 8.4 JWT Structure

```json
{
  "user_id": "uuid",
  "organization_id": "uuid",
  "role": "string",
  "email": "string",
  "exp": timestamp
}
```

**Org Switching:** New JWT issued with new `organization_id` and corresponding `role`.

### 8.5 Session Management

- **Access Token:** HTTP-only cookie, 15-minute expiry
- **Refresh Token:** HTTP-only cookie, 7-day expiry, stored hashed in DB
- **Revocation:** Logout revokes refresh token

### 8.6 Multi-Org Support

- One user account per email globally
- User can belong to multiple orgs with different roles
- Org switcher dropdown in header
- Each org has separate entries, contracts, projects

---

## 9. Internationalization

### 9.1 Timezone Handling

| Context | Timezone |
|---------|----------|
| Business Logic (billing cycles, week boundaries) | Organization timezone |
| Display (calendar, entry dates) | User browser timezone |

### 9.2 Date Formatting

- **Format:** Browser locale (date-fns)
- **Example:** User in US sees "03/15/2026", user in EU sees "15/03/2026"

### 9.3 Number Formatting

- **Format:** Browser locale
- **Example:** US: "1,234.56", EU: "1.234,56"
- **Currency Symbol:** From contract/organization currency (ISO 4217)

### 9.4 Week Start Day

- **Default:** Monday (1)
- **Configurable:** Organization setting `week_start_day`
- **Values:** 0=Sunday, 1=Monday

---

## 10. Migration Strategy

### 10.1 Preparation

1. **Backup database** before migration
2. **Run in staging** environment first
3. **Communicate breaking changes** to users
4. **Schedule downtime** for migration
5. **Add migration CLI:** `go run ./cmd/migrate up/down`

### 10.2 Migration Files

**Migration Order:**
1. `007_add_customers.up.sql`
2. `008_add_organization_settings.up.sql`
3. `009_add_project_managers.up.sql`
4. `010_modify_memberships_multi_role.up.sql`
5. `011_flatten_time_entries.up.sql`
6. `012_flatten_expenses.up.sql`
7. `013_add_receipts_table.up.sql`
8. `014_remove_submitted_status.up.sql`
9. `015_add_deleted_at.up.sql`

### 10.3 Data Migration Scripts

**Time Entries:**
```sql
-- Step 1: Create new table
CREATE TABLE time_entries_new (...);

-- Step 2: Flatten items into separate entries
INSERT INTO time_entries_new (id, user_id, organization_id, project_id, date, hours, description, status, ...)
SELECT 
    gen_random_uuid(),
    te.user_id,
    te.organization_id,
    tei.project_id,
    te.date,
    tei.hours,
    tei.description,
    te.status,
    ...
FROM time_entries te
JOIN time_entry_items tei ON tei.time_entry_id = te.id;

-- Step 3: Rename tables
DROP TABLE time_entry_items;
ALTER TABLE time_entries RENAME TO time_entries_old;
ALTER TABLE time_entries_new RENAME TO time_entries;
```

**Expenses:**
```sql
-- Step 1: Create new flat expenses table
CREATE TABLE expenses_new (...);

-- Step 2: Flatten items into separate entries
INSERT INTO expenses_new (id, user_id, organization_id, project_id, date, type, amount, km_distance, description, status, ...)
SELECT 
    gen_random_uuid(),
    e.user_id,
    e.organization_id,
    ei.project_id,
    e.date,
    ei.category,
    ei.amount,
    ei.km_distance,
    ei.description,
    e.status,
    ...
FROM expenses e
JOIN expense_items ei ON ei.expense_id = e.id;

-- Step 3: Migrate receipts to new BLOB table
INSERT INTO expense_receipts (id, expense_id, receipt_data, original_filename, mime_type, uploaded_at)
SELECT 
    gen_random_uuid(),
    en.id,  -- New expense ID mapped from old item
    ...file data...,
    er.original_filename,
    'application/octet-type',  -- or detect from filename
    er.uploaded_at
FROM expense_receipts er
JOIN expense_items ei ON er.expense_item_id = ei.id
JOIN expenses_new en ON en.project_id = ei.project_id AND en.date = (SELECT date FROM expenses WHERE id = ei.expense_id);

-- Step 4: Rename tables
ALTER TABLE expense_receipts RENAME TO expense_receipts_old;
ALTER TABLE expense_items RENAME TO expense_items_old;
ALTER TABLE expenses RENAME TO expenses_old;
ALTER TABLE expenses_new RENAME TO expenses;
```

### 10.4 Rollback Plan

Keep old tables renamed (not dropped) until migration verified. Can restore by:
```sql
ALTER TABLE time_entries RENAME TO time_entries_new;
ALTER TABLE time_entries_old RENAME TO time_entries;
```

---

## 11. Implementation Phases

### Phase 0: Infrastructure

**Backend:**
- [ ] Add migration CLI tool (`cmd/migrate`)
- [ ] Add health check endpoint (`GET /v1/health`)
- [ ] Add request logging middleware
- [ ] Add rate limiting middleware
- [ ] Add SMTP email service (console for dev)
- [ ] Add password reset flow
- [ ] Add email verification for self-registration
- [ ] Update API versioning (header-based)

**Frontend:**
- [ ] Add org switcher component
- [ ] Add toast notification system (sonner)
- [ ] Add skeleton loader components

### Phase 1: Data Model Migration

**Backend:**
- [ ] Create new migration files
- [ ] Add `customers` table
- [ ] Add `organization_settings` table
- [ ] Add `project_managers` table (no is_primary)
- [ ] Modify `organization_memberships` constraint (multi-role)
- [ ] Add `customer_id` to `contracts`
- [ ] Add `deleted_at` to entries (soft delete)
- [ ] Create flattened `time_entries` migration
- [ ] Create flattened `expenses` migration
- [ ] Create `expense_receipts` table (BLOB)
- [ ] Update models in `internal/models/`
- [ ] Update all handlers for new structures
- [ ] Add approval routing logic
- [ ] Remove `submitted` status
- [ ] Add `rate_recalculate` approval action

**Risks:** Data loss, constraint violations

### Phase 2: Organization Settings

**Backend:**
- [ ] Organization settings CRUD endpoints
- [ ] Km rate recalculation endpoint
- [ ] Currency validation (ISO 4217)

**Frontend:**
- [ ] Settings page: organization tab
- [ ] Org timezone, week start, currency config
- [ ] Km rate management with recalculation option

### Phase 3: Customer Management

**Backend:**
- [ ] Customer CRUD handlers
- [ ] Update contract handlers with customer_id
- [ ] Customer edit permission check (creator org only)

**Frontend:**
- [ ] Customer list/view components
- [ ] Customer form dialog
- [ ] Contract form: customer dropdown

### Phase 4: Project-Manager Assignment

### Phase 3: Project-Manager Assignment

**Backend:**
- [ ] Project manager CRUD endpoints
- [ ] Update approval routing

**Frontend:**
- [ ] Project form: manager multi-select
- [ ] Manager badges on project views

### Phase 4: Unified Entry View

**Backend:**
- [ ] Update time entry endpoints
- [ ] Update expense endpoints
- [ ] Add unified query support

**Frontend:**
- [ ] Create `/entries` page
- [ ] Time/Expense toggle
- [ ] Calendar/List toggle
- [ ] Simplified time entry form
- [ ] Type-aware expense form
- [ ] Deprecate `/time-entries` route

### Phase 5: Approval Dashboard

**Backend:**
- [ ] Grouped pending approvals endpoint
- [ ] Approval routing by project managers
- [ ] All approval action endpoints

**Frontend:**
- [ ] Approval groups component
- [ ] Batch action implementation
- [ ] Entry detail modal

### Phase 6: Settings & Admin

**Backend:**
- [ ] User management endpoints
- [ ] Multi-role assignment endpoints

**Frontend:**
- [ ] Settings page layout
- [ ] Users tab
- [ ] Customers tab
- [ ] Role assignment UI

### Phase 7: CSV Exports

**Backend:**
- [ ] Timesheet export
- [ ] Expense export
- [ ] Combined export

**Frontend:**
- [ ] Export buttons in settings

---

## 12. Out of Scope (MVP)

| Feature | Reason | Future |
|---------|--------|--------|
| Email/in-app notifications | Complexity | Phase 2 |
| Mobile application | Complexity | Phase 3+ |
| Advanced reporting/dashboard | Complexity | Phase 2 |
| Budget tracking | Complexity | Phase 2 |
| Invoice generation | Complexity | Phase 2 |
| Customer login portal | Complexity | Phase 2 |
| Shared resource governance edit flow | Complexity | Phase 2 |
| Partial approval workflow | Not needed for flat entries | Never |
| Password entropy check | Security enhancement | Phase 2 |
| Request ID tracing | Observability | Phase 2 |
| Redis rate limiting | Distributed deployment | Phase 2 |
| Excel export | Format variety | Phase 2 |
| Asset tracking for equipment | Complexity | Phase 3+ |
| CAPTCHA for registration | Anti-abuse | Phase 2 |
| Customer analytics portal | Complexity | Phase 3+ |

---

## 13. Success Criteria

### Data Model
- [ ] Time entries flattened (one project per entry)
- [ ] Expenses flattened (one item per entry)
- [ ] Multiple receipts per expense (BLOB storage)
- [ ] Customers linked to contracts
- [ ] Project managers assigned to projects (no primary distinction)
- [ ] Multi-role users supported
- [ ] Organization settings table created
- [ ] Soft delete with audit trail

### User Accounts
- [ ] Self-registration with email verification
- [ ] Rate limiting (5/hour per IP for registration)
- [ ] Multi-org membership per user
- [ ] Org switching without page reload
- [ ] Password reset via email
- [ ] 7-day invite expiration

### Entries
- [ ] Unified /entries page with toggles
- [ ] Calendar view with number indicator
- [ ] Side panel for entry details
- [ ] 24-hour max per user per day
- [ ] Custom date range exports
- [ ] All roles export within access scope

### Approvals
- [ ] Project-based approval routing
- [ ] Delegation to any user (makes them manager)
- [ ] Bulk edit + approve combined
- [ ] Rejection reason shown before editing
- [ ] Approval history with field-level diffs
- [ ] Mileage rate recalculation logged

### Organization
- [ ] Finance = Admin role
- [ ] Cannot remove last Finance user
- [ ] At least one role required per membership
- [ ] Open project access for org members
- [ ] Finance-only project adoption
- [ ] Creator-only edit for shared customers

### Technical
- [ ] API versioning via header
- [ ] Rate limiting (10/min anon, 100/min auth)
- [ ] Basic request logging
- [ ] SMTP email (console for dev)
- [ ] Pagination on all list endpoints
- [ ] Filters: project_id, type, from/to, search
- [ ] Browser locale for dates/numbers
- [ ] Org timezone for business logic

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| Entry | A time entry or expense record |
| Draft | Entry not yet submitted for approval |
| Pending | Entry awaiting approval at a specific level |
| Manager | User with approval authority for specific projects |
| Finance | User with final approval authority |
| Admin | User with full organization management access |
| Customer | External entity linked to contracts (no login in MVP) |
| Project Manager | User assigned to approve entries for a project |

---

## Appendix B: Decision Log

| # | Decision | Reason | Date |
|---|----------|--------|------|
| 1 | Single project per time entry | Simplifies approval routing to project managers | 2026-03-29 |
| 2 | One expense per item | Different expense types need different fields | 2026-03-29 |
| 3 | Multiple receipts per expense | Real-world scenario: multi-day trips | 2026-03-30 |
| 4 | Receipt BLOB storage | Fastest MVP, migrate to SharePoint/filesystem later | 2026-03-30 |
| 5 | No is_primary on project managers | MVP simplicity, add later if needed | 2026-03-30 |
| 6 | Customer visible to adopting orgs | Customers tied to contracts for reporting | 2026-03-30 |
| 7 | Only creator org edits customer | Clear ownership, no conflicts | 2026-03-30 |
| 8 | Delegation makes user manager | With disclaimer to prevent accidents | 2026-03-30 |
| 9 | Finance = Admin | Small orgs need combined role | 2026-03-30 |
| 10 | Remove submitted status | Redundant, pending_manager indicates submission | 2026-03-30 |
| 11 | At least one role required | No role-less memberships | 2026-03-30 |
| 12 | Cannot remove last Finance user | Every org needs at least one admin | 2026-03-30 |
| 13 | Rejected entries deletable | User may want to discard, not fix | 2026-03-30 |
| 14 | Soft delete with audit trail | Compliance, dispute resolution | 2026-03-30 |
| 15 | Deactivated projects: no new entries | Existing entries unaffected | 2026-03-30 |
| 16 | Projects adopted separately | Explicit opt-in for each project | 2026-03-30 |
| 17 | Finance-only project adoption | Prevents sprawl | 2026-03-30 |
| 18 | Add taxi & equipment expenses | User request | 2026-03-30 |
| 19 | Org default km_rate fallback | Contracts may not have rate set | 2026-03-30 |
| 20 | Organization settings table | Needed for default_km_rate and other configs | 2026-03-30 |
| 21 | Rate recalculation for all statuses | Draft + submitted affected by rate change | 2026-03-30 |
| 22 | Bulk edit + approve combined | Efficiency, one action updates and advances | 2026-03-30 |
| 23 | Approval history toggleable | Some orgs want transparency, others don't | 2026-03-30 |
| 24 | Self-registration with rate limit | Prevent abuse, allow onboarding | 2026-03-30 |
| 25 | Email verification required | Prevent fake accounts | 2026-03-30 |
| 26 | 8-char password min (entropy later) | KIS for MVP | 2026-03-30 |
| 27 | Pagination on all lists | Prevent large responses | 2026-03-30 |
| 28 | Custom date range exports | User request, default to month | 2026-03-30 |
| 29 | All filters for MVP | project_id, type, from/to, search | 2026-03-30 |
| 30 | ILIKE for text search | Simple, add full-text later | 2026-03-30 |
| 31 | Header API versioning | Cleaner than URL versioning | 2026-03-30 |
| 32 | Rate limiting per user/IP | 10/min anon, 100/min auth | 2026-03-30 |
| 33 | SMTP for prod, console for dev | No external deps for dev | 2026-03-30 |
| 34 | Multipart emails | Better deliverability | 2026-03-30 |
| 35 | Browser locale for display | User expectations | 2026-03-30 |
| 36 | Org timezone for business logic | Billing cycles, week boundaries | 2026-03-30 |
| 37 | Week start day configurable | Org preference | 2026-03-30 |
| 38 | ISO 4217 currency codes | Data consistency | 2026-03-30 |
| 39 | Remove partial approval | Not needed for flat entries | 2026-03-30 |
| 40 | Creator-controlled shared resources | MVP simplicity | 2026-03-30 |
| 41 | Open project access | Any member can log time on accessible projects | 2026-03-30 |
| 42 | Side panel for calendar click | Keeps calendar visible | 2026-03-30 |
| 43 | Number indicator in calendar | Cleaner than list preview | 2026-03-30 |
| 44 | URL-based toggles | Shareable links | 2026-03-30 |
| 45 | Skeleton loaders + optimistic updates | Better UX | 2026-03-30 |
| 46 | Toast for general errors, inline for forms | Context-appropriate | 2026-03-30 |
| 47 | No offline support | Time tracking needs connectivity | 2026-03-30 |

---

## Appendix C: Environment Variables

### Backend

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `JWT_SECRET` | Token signing key | `dev-secret-change-in-production` |
| `PORT` | Server port | `8080` |
| `ALLOWED_ORIGINS` | CORS origins (comma-separated) | `http://localhost:3000` |
| `SMTP_HOST` | SMTP server host | - |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USER` | SMTP username | - |
| `SMTP_PASS` | SMTP password | - |

### Frontend

| Variable | Description | Default |
|----------|-------------|---------|
| `VITE_API_URL` | Backend API URL | `/api` (proxied) |

---

## Appendix D: File Structure

```
backend/
├── cmd/
│   ├── server/main.go        # Server entry, routes
│   └── migrate/main.go       # Migration CLI (NEW)
├── internal/
│   ├── auth/                 # JWT, password hashing
│   ├── db/                   # PostgreSQL connection
│   ├── email/                # SMTP service (NEW)
│   ├── handlers/             # HTTP handlers
│   ├── middleware/           # Auth, rate limiting (NEW), logging (NEW)
│   └── models/               # Data structures
├── migrations/               # SQL migrations
├── pkg/api/                  # Shared response format
└── Makefile

frontend/web/src/
├── api/                      # React Query options
├── components/
│   ├── ui/                   # shadcn components
│   ├── entries/              # Entry-related components (NEW)
│   ├── approvals/            # Approval components (NEW)
│   ├── settings/             # Settings components (NEW)
│   └── shared/               # Skeleton loaders, etc. (NEW)
├── hooks/                    # Domain hooks
├── lib/                      # API client, utilities
├── routes/                   # TanStack Router routes
└── types/                    # TypeScript types
```
