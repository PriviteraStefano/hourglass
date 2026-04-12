# SurrealDB Migration Design

**Date:** 2026-04-12  
**Status:** Approved  
**Author:** AI Assistant

---

## Executive Summary

Migrate Hourglass from PostgreSQL to SurrealDB. This is a full switch-over migration with no existing data to preserve. The new schema implements the vault architecture with units hierarchy, working groups, and simplified approval workflow.

---

## Architecture

### Before (PostgreSQL)

```
Go Backend → database/sql → PostgreSQL
              ↓
         internal/db/postgres.go
              ↓
         migrations/*.sql
```

### After (SurrealDB)

```
Go Backend → surrealdb.go SDK → SurrealDB
              ↓
         internal/db/surreal.go
              ↓
         schema/ (DEFINE TABLE statements)
```

---

## Authentication Decision

**Keep JWT authentication in Go** (do not migrate to SurrealDB's built-in auth).

**Reasons:**
1. Organization switching, password reset, activation tokens not modeled well in SurrealDB scope auth
2. Custom permission logic required (WG manager approves entries in their WG, org manager reallocates)
3. Less migration risk - auth stays stable
4. SurrealDB `PERMISSIONS` clauses used as backup safety layer

```sql
-- SurrealDB permissions as defense-in-depth, not primary enforcement
DEFINE TABLE time_entries SCHEMAFULL
    PERMISSIONS
        FOR select ALLOW $auth.org_id == org_id
        FOR create ALLOW $auth.id == user_id;
```

---

## Schema Migration

### Tables by Layer

| Layer | Tables | Migration Order |
|-------|--------|-----------------|
| **Organization** | `organizations`, `units`, `users`, `unit_memberships` | 1st |
| **Project** | `projects`, `subprojects`, `working_groups`, `wg_members`, `customers` | 2nd |
| **Time Entry** | `time_entries`, `audit_logs` | 3rd |
| **Expense** | `expenses`, `budget_caps`, `financial_cutoff_periods` | 4th |

### Key Schema Changes

| Old (PostgreSQL) | New (SurrealDB) |
|------------------|-----------------|
| Flat `organizations` | `organizations` + nested `units` (parent_unit_id) |
| `organization_memberships` | `unit_memberships` (primary/secondary units) |
| Separate `time_entry_approvals`, `expense_approvals` | Single `audit_logs` table |
| Multi-stage status: `draft→submitted→pending_manager→pending_finance→approved/rejected` | Simplified: `draft→submitted→approved` |
| No working groups | `working_groups` + `wg_members` |
| No subprojects | `subprojects` linked to projects |

---

## New Data Models

### Organization Layer

```go
type Unit struct {
    ID             string  `json:"id"`
    OrgID          string  `json:"org_id"`
    Name           string  `json:"name"`
    Description    string  `json:"description,omitempty"`
    ParentUnitID   *string `json:"parent_unit_id,omitempty"`
    HierarchyLevel int     `json:"hierarchy_level"`
    Code           string  `json:"code"`
    CreatedAt      string  `json:"created_at"`
    UpdatedAt      string  `json:"updated_at"`
}

type UnitMembership struct {
    ID         string  `json:"id"`
    OrgID      string  `json:"org_id"`
    UserID     string  `json:"user_id"`
    UnitID     string  `json:"unit_id"`
    IsPrimary  bool    `json:"is_primary"`
    Role       string  `json:"role"` // employee, manager, finance
    StartDate  string  `json:"start_date"`
    EndDate    *string `json:"end_date,omitempty"`
}
```

### Project Layer

```go
type WorkingGroup struct {
    ID               string   `json:"id"`
    OrgID            string   `json:"org_id"`
    SubprojectID     string   `json:"subproject_id"`
    Name             string   `json:"name"`
    Description      string   `json:"description,omitempty"`
    UnitIDs          []string `json:"unit_ids"`
    EnforceUnitTuple bool     `json:"enforce_unit_tuple"`
    ManagerID        string   `json:"manager_id"`
    DelegateIDs      []string `json:"delegate_ids,omitempty"`
}

type WorkingGroupMember struct {
    ID                 string  `json:"id"`
    WGID               string  `json:"wg_id"`
    UserID             string  `json:"user_id"`
    UnitID             string  `json:"unit_id"`
    Role               string  `json:"role"`
    IsDefaultSubproject bool   `json:"is_default_subproject"`
}
```

### Time Entry (Simplified)

```go
type TimeEntry struct {
    ID                  string   `json:"id"`
    OrgID               string   `json:"org_id"`
    UserID              string   `json:"user_id"`
    ProjectID           string   `json:"project_id"`
    SubprojectID        string   `json:"subproject_id"`
    WGID                string   `json:"wg_id"`
    UnitID              string   `json:"unit_id"`
    Hours               float64  `json:"hours"`
    Description         string   `json:"description"`
    EntryDate           string   `json:"entry_date"`
    Status              string   `json:"status"` // draft, submitted, approved
    IsDeleted           bool     `json:"is_deleted"`
    CreatedFromEntryID  *string  `json:"created_from_entry_id,omitempty"` // for splits
    CreatedAt           string   `json:"created_at"`
    UpdatedAt           string   `json:"updated_at"`
}
```

### Audit Log (Unified)

```go
type AuditLog struct {
    ID         string  `json:"id"`
    OrgID      string  `json:"org_id"`
    EntryID    string  `json:"entry_id"`
    EntryType  string  `json:"entry_type"` // time_entry, expense
    Action     string  `json:"action"`    // created, submitted, approved, rejected, split, moved, reallocated
    ActorRole  string  `json:"actor_role"` // user, wg_manager, org_manager, finance
    ActorID    string  `json:"actor_id"`
    Reason     *string `json:"reason,omitempty"`
    Changes    *string `json:"changes,omitempty"` // JSON of before/after
    Timestamp  string  `json:"timestamp"`
}
```

---

## Approval Workflow (Simplified)

### Old Flow (4 states)
```
draft → submitted → pending_manager → pending_finance → approved/rejected
```

### New Flow (3 states)
```
draft → submitted → approved
                      ↓
               [post-approval actions in audit_log only]
                      ↓
         reallocate / split / finance_override
```

**Approval Actions:**

| Actor | Before Approval | After Approval |
|-------|-----------------|----------------|
| User | Create draft, submit | N/A |
| WG Manager | Approve, reject, split, move | N/A |
| Org Manager | N/A | Reallocate (change unit/project) |
| Finance | N/A | Override after cutoff |

All actions logged to `audit_logs` with immutable records.

---

## Implementation Plan

### Phase 1: Infrastructure Setup
1. Add SurrealDB to `docker-compose.yml`
2. Create `internal/db/surreal.go` connection module
3. Create `schema/` directory with SurrealDB DEFINE TABLE files
4. Write schema loading script

### Phase 2: Organization Layer
1. Migrate `organizations`, `units`, `users`, `unit_memberships` to SurrealDB schema
2. Update `internal/models/` with new structs
3. Rewrite `OrganizationHandler` queries
4. Rewrite `UserHandler` queries

### Phase 3: Project Layer
1. Schema for `projects`, `subprojects`, `working_groups`, `wg_members`, `customers`
2. Add new handlers for units and working groups
3. Update `ProjectHandler` for subprojects
4. Add `WorkingGroupHandler`

### Phase 4: Time Entry Layer
1. Schema for `time_entries`, `audit_logs`
2. Simplify status enum
3. Rewrite `TimeEntryHandler` for working groups
4. Rewrite `ApprovalHandler` for simplified flow
5. Add split/move endpoints

### Phase 5: Expense Layer
1. Schema for `expenses`, `budget_caps`, `financial_cutoff_periods`
2. Rewrite `ExpenseHandler`
3. Add Finance override capability

### Phase 6: Cleanup
1. Remove PostgreSQL dependencies (`internal/db/postgres.go`)
2. Remove old migrations
3. Update `docker-compose.yml` to remove PostgreSQL
4. Update tests

---

## File Changes

### New Files
```
internal/db/surreal.go           # SurrealDB connection
schema/                          # SurrealDB schema files
  001_organizations.surql
  002_units.surql
  003_users.surql
  004_unit_memberships.surql
  005_projects.surql
  006_subprojects.surql
  007_working_groups.surql
  008_wg_members.surql
  009_time_entries.surql
  010_audit_logs.surql
  011_expenses.surql
  012_financial_cutoffs.surql
internal/handlers/unit_handler.go
internal/handlers/working_group_handler.go
```

### Modified Files
```
internal/models/models.go        # New structs, remove old statuses
internal/handlers/*.go           # SurrealDB queries
cmd/server/main.go               # SurrealDB connection
docker-compose.yml               # Add SurrealDB service
go.mod                           # Add surrealdb.go dependency
```

### Deleted Files
```
internal/db/postgres.go          # PostgreSQL connection
migrations/*.sql                 # PostgreSQL migrations
```

---

## Testing Strategy

1. **Unit tests:** Query generation tests for SurrealDB queries
2. **Integration tests:** Docker Compose with SurrealDB test instance
3. **Schema tests:** Verify all DEFINE TABLE statements parse correctly
4. **Handler tests:** Test permission boundaries for WG managers, org managers, finance

---

## Open Questions

None - design approved.

---

## References

- Vault design: `hourglass-vault/design/plan.md`
- Schema: `hourglass-vault/design/files/surrealdb-schema.sql`
- API spec: `hourglass-vault/design/files/api-specification.md`