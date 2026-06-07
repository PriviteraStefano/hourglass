---
phase: pg-2-adapters
plan: 06
type: execute
wave: 3
depends_on: [pg-2-01]
files_modified:
  - internal/adapters/secondary/postgres/time_entry_repository.go
  - internal/adapters/secondary/postgres/time_entry_repository_test.go
  - internal/adapters/secondary/postgres/audit_log_repository.go
  - internal/adapters/secondary/postgres/audit_log_repository_test.go
  - internal/adapters/secondary/postgres/expense_repository.go
  - internal/adapters/secondary/postgres/expense_repository_test.go
  - internal/adapters/secondary/postgres/export_repository.go
  - internal/adapters/secondary/postgres/export_repository_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - TimeEntryRepository supports List (dynamic WHERE), GetByID, Create, Update, Delete (soft), IsPeriodLocked, ListPending
    - Dynamic WHERE building uses numbered placeholders ($1, $2...) for all filter conditions
    - AuditLogRepository supports Create (immutable append-only, changes as JSONB)
    - ExpenseRepository supports Create, GetByID, ListByOrg, Update, Delete (soft) using models.Expense
    - ExportRepository supports Timesheets and Expenses queries with 4-level LEFT JOINs (user→project→contract→customer)
    - Export queries replace users.name with firstname || ' ' || lastname for employee display
    - Role-based filtering in export queries uses parameterized conditions
  artifacts:
    - path: internal/adapters/secondary/postgres/time_entry_repository.go
      provides: implements ports.TimeEntryRepository (7 methods + buildQuery with dynamic WHERE)
    - path: internal/adapters/secondary/postgres/audit_log_repository.go
      provides: implements ports.AuditLogRepository (1 method — Create)
    - path: internal/adapters/secondary/postgres/expense_repository.go
      provides: implements ports.ExpenseRepository (5 methods)
    - path: internal/adapters/secondary/postgres/export_repository.go
      provides: implements ports.ExportRepository (2 methods with 4-level JOIN chains)
  key_links:
    - from: time_entry_repository.go buildQuery
      to: dynamic WHERE building
      via: fmt.Sprintf(" AND column = $%d", n) for each active filter
    - from: time_entry_repository.go ListPending
      to: working_groups table
      via: subquery for manager/delegate WG IDs
    - from: export_repository.go
      to: users table (employee name)
      via: u.firstname || ' ' || u.lastname (users.name column removed)
    - from: expense_repository.go
      to: expenses table + models.Expense
      via: positional Scan with column→field mapping (org_id→OrganizationID, etc.)
---

<objective>
Implement the three most complex repositories — TimeEntryRepository (dynamic WHERE building), ExpenseRepository (new), AuditLogRepository (append-only), and ExportRepository (4-level JOIN chains) — for PostgreSQL.

Purpose: These are the most SQL-intensive repos in the system. TimeEntry has dynamic filtering, list-pending with WG-resolution, and period-locking. Export has 4-level JOIN chains replacing deeply nested SurrealDB subqueries. Expense is the new repository ported from models. These all benefit from patterns established in earlier Wave 2 repos.

Output:
- time_entry_repository.go + audit_log_repository.go + tests
- expense_repository.go + tests
- export_repository.go + tests
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-2-adapters/pg-2-PATTERNS.md
@.planning/phases/pg-2-adapters/pg-2-RESEARCH.md

# Port interfaces
@internal/core/ports/time_entry_repository.go (TimeEntryRepository + AuditLogRepository + ListFilters)
@internal/core/ports/expense_repository.go (just created in Plan 1)
@internal/core/ports/export_repository.go (ExportRepository + ExportRow)

# SurrealDB analogs
@internal/adapters/secondary/surrealdb/time_entry_repository.go (dynamic WHERE, AuditLog)
@internal/adapters/secondary/surrealdb/export_repository.go (4-level nested subqueries)

# Domain models
@internal/core/domain/time_entry/time_entry.go (TimeEntry, AuditLog)
@internal/models/models.go (Expense struct)

# Foundation helpers
@internal/adapters/secondary/postgres/postgres.go
@internal/adapters/secondary/postgres/exported_test_helpers.go

# Schema
@migrations/002_full_schema.up.sql (time_entries, expenses, financial_cutoff_periods tables)

<interfaces>
From internal/core/ports/time_entry_repository.go:
```go
type TimeEntryRepository interface {
    List(ctx context.Context, orgID uuid.UUID, filters ListFilters) ([]time_entry.TimeEntry, error)
    GetByID(ctx context.Context, id uuid.UUID) (*time_entry.TimeEntry, error)
    Create(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error)
    Update(ctx context.Context, e *time_entry.TimeEntry) (*time_entry.TimeEntry, error)
    Delete(ctx context.Context, id uuid.UUID) error
    IsPeriodLocked(ctx context.Context, orgID, projectID uuid.UUID, entryDate string) (bool, error)
    ListPending(ctx context.Context, orgID uuid.UUID, role, userID string) ([]time_entry.TimeEntry, error)
}

type AuditLogRepository interface {
    Create(ctx context.Context, log *time_entry.AuditLog) error
}

type ListFilters struct {
    OrgID         interface{}
    Date          string
    Month         string
    Year          string
    UserID        string
    Status        string
    WGID          string
    ProjectID     string
    Role          string
    IsDeleted     bool
    RequestUserID string
}
```

From internal/core/ports/expense_repository.go (new):
```go
type ExpenseRepository interface {
    Create(ctx context.Context, e *models.Expense) (*models.Expense, error)
    GetByID(ctx context.Context, id uuid.UUID) (*models.Expense, error)
    ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Expense, error)
    Update(ctx context.Context, e *models.Expense) (*models.Expense, error)
    Delete(ctx context.Context, id uuid.UUID) error
}
```

From internal/core/ports/export_repository.go:
```go
type ExportRow struct {
    EntryType   string
    Date        time.Time
    Employee    string
    Project     string
    Contract    string
    Customer    string
    Hours       *float64
    Amount      *float64
    KmDistance  *float64
    Type        string
    Description string
    Status      string
}

type ExportRepository interface {
    Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ExportRow, error)
    Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ExportRow, error)
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: TimeEntryRepository + AuditLogRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/time_entry_repository.go
    internal/adapters/secondary/postgres/audit_log_repository.go
    internal/adapters/secondary/postgres/time_entry_repository_test.go
    internal/adapters/secondary/postgres/audit_log_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/time_entry_repository.go (interface + ListFilters struct)
    @internal/core/domain/time_entry/time_entry.go (TimeEntry: CreatedFromEntryID *uuid.UUID, AuditLog: Changes map[string]any)
    @internal/adapters/secondary/surrealdb/time_entry_repository.go (analog — dynamic WHERE, ListPending, IsPeriodLocked, AuditLog)
    @migrations/002_full_schema.up.sql (time_entries, time_entry_approvals, financial_cutoff_periods tables)
  </read_first>

  <action>
    **A) time_entry_repository.go** — implements ports.TimeEntryRepository (7 methods)
    Struct `TimeEntryRepository` with `pool *pgxpool.Pool`, constructor `NewTimeEntryRepository`.

    **Private helper — buildQuery (dynamic WHERE):**
    ```go
    func (r *TimeEntryRepository) buildQuery(orgID uuid.UUID, filters ports.ListFilters) (string, []any) {
        args := []any{orgID, filters.IsDeleted}
        query := `SELECT te.id, te.org_id, te.user_id, te.project_id, te.subproject_id, te.wg_id,
                         te.unit_id, te.hours, te.description, te.entry_date, te.status,
                         te.is_deleted, te.created_from_entry_id, te.created_at, te.updated_at
                  FROM time_entries te WHERE te.org_id = $1 AND te.is_deleted = $2`
        n := 3
        if filters.Date != "" {
            query += fmt.Sprintf(` AND te.entry_date::date = $%d::date`, n)
            args = append(args, filters.Date)
            n++
        }
        if filters.Month != "" && filters.Year != "" {
            query += fmt.Sprintf(` AND EXTRACT(MONTH FROM te.entry_date) = $%d AND EXTRACT(YEAR FROM te.entry_date) = $%d`, n, n+1)
            args = append(args, filters.Month, filters.Year)
            n += 2
        }
        if filters.UserID != "" {
            uid, _ := uuid.Parse(filters.UserID)
            if filters.Role == "employee" && filters.UserID != filters.RequestUserID {
                reqUID, _ := uuid.Parse(filters.RequestUserID)
                query += fmt.Sprintf(` AND te.user_id = $%d`, n)
                args = append(args, reqUID)
            } else {
                query += fmt.Sprintf(` AND te.user_id = $%d`, n)
                args = append(args, uid)
            }
            n++
        }
        if filters.Status != "" {
            query += fmt.Sprintf(` AND te.status = $%d`, n)
            args = append(args, filters.Status)
            n++
        }
        if filters.WGID != "" {
            wgID, _ := uuid.Parse(filters.WGID)
            query += fmt.Sprintf(` AND te.wg_id = $%d`, n)
            args = append(args, wgID)
            n++
        }
        if filters.ProjectID != "" {
            projID, _ := uuid.Parse(filters.ProjectID)
            query += fmt.Sprintf(` AND te.project_id = $%d`, n)
            args = append(args, projID)
            n++
        }
        query += ` ORDER BY te.entry_date DESC, te.created_at DESC`
        return query, args
    }
    ```

    **Methods:**

    1. **List**: Call buildQuery, pool.Query, iterate rows. created_from_entry_id is *uuid.UUID (nullable). Return []time_entry.TimeEntry{} not nil.

    2. **GetByID**: `SELECT id, org_id, user_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, status, is_deleted, created_from_entry_id, created_at, updated_at FROM time_entries WHERE id = $1`
       - ErrNoRows → time_entry.ErrTimeEntryNotFound

    3. **Create**: `INSERT INTO time_entries (id, org_id, user_id, project_id, subproject_id, wg_id, unit_id, hours, description, entry_date, status, is_deleted, created_from_entry_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING *`
       - QueryRow + Scan into time_entry.TimeEntry

    4. **Update**: Dynamic SET similar to contract Update. Set updated_at=NOW(). RETURNING *.

    5. **Delete** (soft): `UPDATE time_entries SET is_deleted = true, updated_at = NOW() WHERE id = $1`

    6. **IsPeriodLocked**: Check financial_cutoff_periods for org-level AND project-level locks:
       ```sql
       SELECT EXISTS(
           SELECT 1 FROM financial_cutoff_periods
           WHERE org_id = $1 AND $2::timestamptz BETWEEN period_start AND period_end AND is_locked = true
       )
       ```
       If projectID is not nil, also check `OR (project_id = $3 AND $2::timestamptz BETWEEN period_start AND period_end AND is_locked = true)`

    7. **ListPending**: SELECT time_entries WHERE org_id=$1 AND status='submitted' AND is_deleted=false.
       If role is "wg_manager", subquery to find WGs where user is manager or delegate:
       ```sql
       AND te.wg_id IN (SELECT id FROM working_groups WHERE manager_id = $3 OR $3 = ANY(delegate_ids))
       ```
       ORDER BY created_at ASC.

    **B) audit_log_repository.go** — implements ports.AuditLogRepository (1 method)
    Struct `AuditLogRepository` with `pool *pgxpool.Pool`, constructor `NewAuditLogRepository`.

    **Create**: `INSERT INTO time_entry_approvals (id, time_entry_id, user_id, action, comment, created_at) VALUES ($1,$2,$3,$4,$5,$6)`
    NOTE: The PG schema has `time_entry_approvals` table (not `audit_logs`). The domain model `time_entry.AuditLog` has `Changes map[string]any` but the PG schema doesn't have a `changes` column in `time_entry_approvals`. For now, skip the Changes field in INSERT — it maps to the `comment` field.

    Wait — let me re-check the schema. Looking at 002_full_schema, `time_entry_approvals` has: id, time_entry_id, user_id, action, comment, created_at. There's no `changes` JSONB column. But the surrealdb version stores to `audit_logs` table which may have a different schema.

    Looking at the surrealdb AuditLogRepository.Create, it creates into `audit_logs` table with all fields. But in PG the schema may not have an `audit_logs` table. Let me check...

    Actually, looking at the domain model, `time_entry.AuditLog` has `Changes map[string]any`. And the schema at time_entry_approvals doesn't have a changes column. So either:
    a) There's an audit_logs table I missed
    b) The changes field is stored in the comment field

    Looking at 007_phase2_schema or other migration files... Actually, the implementation should match what the service expects. Since we're creating a NEW adapter, we need to store audit logs somewhere. The simplest approach: use the `time_entry_approvals` table for time entry audit logs (where `action` is the approval action) and map `Changes` to a JSON string in `comment` or create a helper function.

    Actually, looking at this more carefully, the surrealdb AuditLogRepository creates into `audit_logs` table. The PG schema might not have this table. Let me just use the `time_entry_approvals` table with `comment` storing a JSON representation of changes. The port interface accepts `*time_entry.AuditLog` — we map fields positionally.

    For this plan, I'll define it as:
    ```go
    func (r *AuditLogRepository) Create(ctx context.Context, log *time_entry.AuditLog) error {
        // Marshal Changes to JSON string for comment field
        changesJSON, _ := json.Marshal(log.Changes)
        const query = `INSERT INTO time_entry_approvals (id, time_entry_id, user_id, action, comment, created_at) VALUES ($1,$2,$3,$4,$5,$6)`
        ...
    }
    ```
    Where time_entry_id maps from log.EntryID (string → uuid.UUID), user_id from log.ActorID.

    **C) Test files:**
    - `time_entry_repository_test.go`: Test Create→GetByID (verify all fields round-trip, including created_from_entry_id nullable), Test List with filters (by date, by user, by status, by project), Test List with no filters (returns all for org), Test Update (change hours, verify), Test Delete (soft — verify is_deleted=true), Test ListPending (create submitted entries, list them), Test IsPeriodLocked (create cutoff period, verify lock detection)
    - `audit_log_repository_test.go`: Test Create (create audit log, verify via raw SQL SELECT)
    - Seed: org, user, project, subproject, wg, unit for FK references
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. TimeEntryRepository with 7 methods compiles, dynamic WHERE handles all filter combinations
    2. AuditLogRepository with Create method compiles
    3. Test files compile
    4. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 2: ExpenseRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/expense_repository.go
    internal/adapters/secondary/postgres/expense_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/expense_repository.go (interface — 5 methods)
    @internal/models/models.go (Expense struct lines 277-295)
    @internal/adapters/secondary/surrealdb/time_entry_repository.go (similar CRUD pattern — no surrealdb expense repo exists)
    @migrations/002_full_schema.up.sql (expenses table: id, org_id, user_id, project_id nullable, unit_id, category, amount, currency, description, expense_date, receipt_url, receipt_ocr_data JSONB, status, is_deleted, created_at, updated_at)
  </read_first>

  <action>
    **A) expense_repository.go** — implements ports.ExpenseRepository (5 methods)
    Struct `ExpenseRepository` with `pool *pgxpool.Pool`, constructor `NewExpenseRepository`.

    **Column mapping (SQL name ≠ domain field):**
    - SQL `org_id` → domain `OrganizationID`
    - SQL `user_id` → domain `UserID`
    - SQL `project_id` (nullable) → domain `ProjectID *uuid.UUID`
    - SQL `category` → domain `Type *ExpenseCategory`
    - SQL `amount` → domain `Amount *float64`
    - SQL `km_distance` → domain `KmDistance *float64` (not in schema? Let me check)
    
    Actually, looking at the expenses table schema: id, org_id, user_id, project_id, unit_id, category, amount, currency, description, expense_date, receipt_url, receipt_ocr_data, status, is_deleted, created_at, updated_at. There's no km_distance column in the PG schema!
    
    But models.Expense has KmDistance *float64. This means either:
    1. The schema needs a km_distance column (it exists in the old SurrealDB setup)
    2. The domain model has fields not in the schema

    For this plan, since we're building the adapter to match the current schema, we should SELECT/INSERT the columns that exist. The `km_distance` field in models.Expense will need to be ignored for now (no corresponding column in PG expenses table). This is a known discrepancy — the expense domain model has more fields than the PG schema.

    **Methods:**

    1. **Create**: `INSERT INTO expenses (id, org_id, user_id, project_id, unit_id, category, amount, currency, description, expense_date, status, is_deleted, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) RETURNING id, org_id, user_id, project_id, unit_id, category, amount, currency, description, expense_date, receipt_url, receipt_ocr_data, status, is_deleted, created_at, updated_at`
       - Note: receipt_url and receipt_ocr_data are nullable/optional fields in schema but not in Expense struct. Initialize as empty.
       - Wrap with wrapPGError

    2. **GetByID**: `SELECT id, org_id, user_id, project_id, unit_id, category, amount, currency, description, expense_date, receipt_url, receipt_ocr_data, status, is_deleted, created_at, updated_at FROM expenses WHERE id = $1`
       - ErrNoRows → `customer.ErrCustomerNotFound` or a generic not-found? Use wrapPGError which maps to ports.ErrNotFound.
       - receipt_ocr_data is JSONB → scan as json.RawMessage

    3. **ListByOrg**: Same SELECT with `WHERE org_id = $1 AND is_deleted = false ORDER BY expense_date DESC LIMIT $2 OFFSET $3`
       - Return []models.Expense{} not nil

    4. **Update**: Full-field UPDATE with RETURNING *

    5. **Delete** (soft): `UPDATE expenses SET is_deleted = true, updated_at = NOW() WHERE id = $1`

    **B) expense_repository_test.go:**
    - Test Create→GetByID (create expense, retrieve, assert fields match)
    - Test ListByOrg (create 2 expenses for org, list, verify count)
    - Test Update (change amount, verify)
    - Test Delete (soft delete, verify is_deleted=true)
    - Test ListByOrg excludes deleted (create + delete, list returns empty)
    - Seed: org, user, unit for FK references. project_id is nullable.
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. ExpenseRepository with 5 methods compiles
    2. Column mapping (org_id→OrganizationID, etc.) handled via positional Scan
    3. Test file compiles
    4. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 3: ExportRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/export_repository.go
    internal/adapters/secondary/postgres/export_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/export_repository.go (interface + ExportRow struct)
    @internal/adapters/secondary/surrealdb/export_repository.go (analog — 4-level nested subqueries)
    @migrations/002_full_schema.up.sql (time_entries, expenses, projects, contracts, customers tables)
  </read_first>

  <action>
    **A) export_repository.go** — implements ports.ExportRepository (2 methods)
    Struct `ExportRepository` with `pool *pgxpool.Pool`, constructor `NewExportRepository`.

    **SQL JOIN translation (SurrealDB 4-level nested subqueries → SQL LEFT JOINs):**
    Instead of deeply nested subqueries, use a chain of LEFT JOINs.

    **Private helper — roleFilter:**
    ```go
    func (r *ExportRepository) roleFilter(field string, role string, n int) (string, int) {
        switch role {
        case string(models.RoleEmployee):
            return fmt.Sprintf(" AND te.%s = $%d", field, n), n + 1
        case string(models.RoleManager):
            return fmt.Sprintf(" AND (te.%s = $%d OR te.project_id IN (SELECT project_id FROM project_managers WHERE user_id = $%d))", field, n, n), n + 1
        default:
            return "", n
        }
    }
    ```

    **Methods:**

    1. **Timesheets**: 
       ```sql
       SELECT 'time_entry' AS entry_type,
              te.entry_date AS date,
              u.firstname || ' ' || u.lastname AS employee,
              p.name AS project,
              c.name AS contract,
              COALESCE(cu.name, '') AS customer,
              te.hours,
              NULL::decimal AS amount,
              NULL::decimal AS km_distance,
              NULL::varchar AS type,
              te.description,
              te.status
       FROM time_entries te
       LEFT JOIN users u ON u.id = te.user_id
       LEFT JOIN projects p ON p.id = te.project_id
       LEFT JOIN contracts c ON c.id = p.contract_id
       LEFT JOIN customers cu ON cu.id = c.customer_id
       WHERE te.org_id = $1 AND te.entry_date >= $2 AND te.entry_date <= $3 AND te.is_deleted = false`
       ```
       Append roleFilter for role-based access. Add user_id param if needed. ORDER BY te.entry_date, te.created_at.

       Query with pool.Query, iterate rows, scan into locals, map to ExportRow.

    2. **Expenses**:
       ```sql
       SELECT 'expense' AS entry_type,
              e.expense_date AS date,
              u.firstname || ' ' || u.lastname AS employee,
              p.name AS project,
              c.name AS contract,
              COALESCE(cu.name, '') AS customer,
              NULL::decimal AS hours,
              e.amount,
              e.km_distance,
              e.category AS type,
              e.description,
              e.status
       FROM expenses e
       LEFT JOIN users u ON u.id = e.user_id
       LEFT JOIN projects p ON p.id = e.project_id
       LEFT JOIN contracts c ON c.id = p.contract_id
       LEFT JOIN customers cu ON cu.id = c.customer_id
       WHERE e.org_id = $1 AND e.expense_date >= $2 AND e.expense_date <= $3 AND e.is_deleted = false`
       ```
       Note: `e.km_distance` is in the expenses table SELECT but may not exist in the PG column set. If not, use NULL::decimal.

       Append roleFilter. ORDER BY e.expense_date.

    **Scan pattern:**
    Both methods scan into the same ExportRow fields. Use individual variables or a scan struct. Key nullable fields: Hours (*float64), Amount (*float64), KmDistance (*float64).

    **B) export_repository_test.go:**
    - Test Timesheets (create time entries with full FK chain: org→user→project→contract→customer, export by date range, verify all fields)
    - Test Timesheets employee role filter (create entries for different users, filter as employee role)
    - Test Timesheets manager role filter (create entries, user is project manager)
    - Test Expenses (create expenses with same FK chain, export by date range, verify)
    - Test empty export (no data in date range returns empty slice)
    - Seed full FK chain for each test
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. ExportRepository with 2 methods compiles
    2. JOIN chains replace SurrealDB's 4-level nested subqueries
    3. Employee name uses firstname||' '||lastname (not users.name column)
    4. Role-based filtering works for employee/manager/finance roles
    5. Test file compiles
    6. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Application → PostgreSQL time_entry/expense/export queries | Transactional and reporting data flows through parameterized queries |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg2-17 | Tampering | time_entry buildQuery dynamic WHERE | mitigate | Only placeholder positions ($N) are dynamically injected; filter values are always parameterized. String concatenation only adds SQL keywords (AND, OR, ORDER BY) and column identifiers from constants. |
| T-pg2-18 | Information Disclosure | export roleFilter | mitigate | User ID is always parameterized ($N). Role string comes from domain type (not user input) and switches on fixed enum values. |
| T-pg2-19 | Tampering | expense JSONB (receipt_ocr_data) | mitigate | JSONB values are parameterized — never built from string concatenation. json.Marshal ensures safe encoding. |
| T-pg2-20 | Information Disclosure | export Timesheets/Expenses | mitigate | Role-based filtering prevents employees from seeing other users' entries. Manager role provides project-based access via project_managers subquery. |
</threat_model>

<verification>
```bash
go vet ./internal/adapters/secondary/postgres/
go build ./internal/adapters/secondary/postgres/
# Integration tests:
# DATABASE_URL=... go test ./internal/adapters/secondary/postgres/ -run "TestTimeEntry|TestAuditLog|TestExpense|TestExport" -count=1 -v
```
</verification>

<success_criteria>
1. TimeEntryRepository implements all 7 methods with dynamic WHERE building
2. AuditLogRepository implements Create (append-only)
3. ExpenseRepository implements all 5 methods
4. ExportRepository implements both Timesheets and Expenses with 4-level JOIN chains
5. No export query references users.name column (uses firstname||' '||lastname)
6. All tests compile
7. Files committed to git
</success_criteria>

<output>
After completion, create `.planning/phases/pg-2-adapters/pg-2-06-SUMMARY.md`
</output>
