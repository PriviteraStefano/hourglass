# System Overview

## What is Hourglass?

Hourglass is a **time entry and expense tracking system** designed for organizations that need:

- **Granular time tracking** — employees log time against specific projects
- **Expense management** — mileage, meals, accommodations categorized and approved
- **Approval workflows** — manager and finance reviews before approval
- **Multi-tenancy** — multiple organizations with isolated data
- **Role-based access** — employee, manager, finance, customer roles with different permissions
- **Shared resources** — contracts and projects optionally shared across organizations

## Core Workflows

### Time Entry & Expense Approval Workflow

```
Employee (Draft) 
    ↓
Submit to Manager
    ↓
Manager Review (Approve/Reject)
    ↓
Finance Review (if required)
    ↓
Final Status (Approved/Rejected)
    ↓
History is immutable
```

Each entry has:
- **Status**: draft, submitted, pending_manager, pending_finance, approved, rejected
- **Current Approver Role**: tracks who needs to review next
- **Approval History**: immutable record of all actions and reviews

### Organization & Contract Model

```
Organization
├── Users (with roles: employee, manager, finance, customer)
├── Contracts (belong to org, optionally shared to others)
│   └── Projects (linked to contract)
└── Customers (company contact info for contracts)
```

**Shared Resources:** A contract created by Org A can be "adopted" by Org B, allowing both to use it and its projects.

## Key Features (Implemented)

| Issue | Feature | Description |
|-------|---------|-------------|
| #2 | Infrastructure & Auth Hardening | JWT tokens, bcrypt hashing, secure headers |
| #4 | Organization Settings | Timezone, currency, km rate, week start day |
| #5 | Customer Management | CRUD for company contacts linked to contracts |
| #6 | Project Manager Assignment | Assign managers to specific projects |
| #7 | Time Entries (Flattened Schema) | Redesigned entry structure for Phase 2 |
| #8 | Expenses (Flattened Schema) | Mileage, meal, accommodation, other categories |
| #10 | Batch Approvals | edit-approve endpoints for bulk operations |
| #11 | Contract Updates & Mileage | Update contract details, recalculate mileage costs |
| #12 | User Management & Org Switching | Multiple org membership, easy org switching |
| #13 | CSV Exports | Role-based report generation |

## System Architecture

See [[02-Architecture]] for detailed tech stack and component breakdown.

**High Level:**
- **Backend**: Go HTTP server + PostgreSQL
- **Frontend**: React SPA with TanStack Router + React Query
- **Database**: PostgreSQL with numbered migration system
- **Auth**: JWT-based with refresh tokens

## Data Flow: Time Entry Creation

```
1. Employee opens React app → authenticated via JWT
2. Frontend displays time entry form (TanStack React Query)
3. Employee submits → POST /time-entries
4. Backend validates, stores as "draft"
5. Employee later clicks "Submit" → PUT /time-entries/{id}/submit
6. Status changes to "submitted" → managers notified
7. Manager approves → POST /approvals (creates approval record)
8. Status advances to "pending_finance" 
9. Finance reviews, approves → final status "approved"
10. Both approval records stored immutably in time_entry_approvals table
```

## Multi-Tenancy & Data Isolation

- **Organization ID** is the primary isolation key
- Users can belong to multiple organizations (OrganizationMembership table)
- Queries filter by current org context
- Contracts/Projects created by org but can be shared to others via "adoption"
- All data includes `created_by_org_id` to track origin

## Role-Based Access

| Role | Capabilities |
|------|--------------|
| **Employee** | Create/edit own time entries, view own expenses |
| **Manager** | Review submitted entries, approve/reject, bulk operations |
| **Finance** | Final approval before payment, CSV exports |
| **Customer** | View invoicing details (limited access) |

Defined in [[13-Organization-Users]].

## Why This Approach?

1. **Workflow Accuracy** — immutable approval history prevents disputes
2. **Flexibility** — shared resources allow organizations to standardize
3. **Scalability** — PostgreSQL + simple HTTP API
4. **User Experience** — React SPA with instant feedback (React Query)
5. **Governance** — multiple approval models (creator-controlled, unanimous, majority)

---

**Next**: [[02-Architecture]] for technical details, or [[15-Development-Setup]] to start coding.
