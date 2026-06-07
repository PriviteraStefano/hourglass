---
phase: pg-2-adapters
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - migrations/002_full_schema.up.sql
  - internal/core/ports/errors.go
  - internal/core/ports/expense_repository.go
  - internal/core/ports/subproject_repository.go
  - internal/adapters/secondary/postgres/postgres.go
  - internal/adapters/secondary/postgres/exported_test_helpers.go
  - internal/adapters/secondary/postgres/user_finder.go
  - internal/adapters/secondary/postgres/user_finder_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - users.name column removed from schema; no PG query references it
    - Shared sentinel errors (ErrNotFound, ErrConflict, ErrForeignKey) exist in ports/
    - ExpenseRepository and SubprojectRepository port interfaces exist in ports/
    - wrapPGError helper translates pgx.ErrNoRows to ErrNotFound, unique_violation to ErrConflict, foreign_key_violation to ErrForeignKey
    - Repository tests can obtain a *pgxpool.Pool and apply schema via SetupTestSchema
    - UserFinder.FindByIdentifier returns userID string for email or username match
  artifacts:
    - path: migrations/002_full_schema.up.sql
      provides: users table without name column
      differs: "name VARCHAR(255) NOT NULL" line removed
    - path: internal/core/ports/errors.go
      provides: var ErrNotFound, ErrConflict, ErrForeignKey sentinel errors
    - path: internal/core/ports/expense_repository.go
      provides: ExpenseRepository port interface with Create/GetByID/ListByOrg/Update/Delete
    - path: internal/core/ports/subproject_repository.go
      provides: SubprojectRepository port interface with ListByProject/GetByID/Create/Update/Delete
    - path: internal/adapters/secondary/postgres/postgres.go
      provides: wrapPGError helper
      exports: func wrapPGError(err error, op string) error
    - path: internal/adapters/secondary/postgres/exported_test_helpers.go
      provides: TestPool, SetupTestSchema, uniqueEmail, uniqueUsername
    - path: internal/adapters/secondary/postgres/user_finder.go
      provides: implements ports.UserFinder
    - path: internal/adapters/secondary/postgres/user_finder_test.go
      provides: validates FindByIdentifier with email and username
  key_links:
    - from: postgres.go
      to: ports/errors.go
      via: errors.Is(err, pgx.ErrNoRows) → ports.ErrNotFound
    - from: exported_test_helpers.go
      to: internal/db/pgpool.go
      via: db.NewPool()
    - from: user_finder.go
      to: users table (via 002_full_schema)
      via: SELECT id FROM users WHERE email = $1 OR username = $1
---

<objective>
Create the PostgreSQL adapter foundation — schema correction, shared sentinel errors, new port interfaces, package-level helpers, test infrastructure, and the UserFinder implementation.

Purpose: Establish shared patterns and infrastructure that all 17+ repository implementations depend on. Without this plan, downstream repos would duplicate error handling, test setup, and connection management.

Output:
- migrations/002_full_schema.up.sql modified (remove users.name)
- internal/core/ports/errors.go (sentinel errors)
- internal/core/ports/expense_repository.go (new port interface)
- internal/core/ports/subproject_repository.go (new port interface)
- internal/adapters/secondary/postgres/postgres.go (wrapPGError)
- internal/adapters/secondary/postgres/exported_test_helpers.go (test infrastructure)
- internal/adapters/secondary/postgres/user_finder.go (UserFinder)
- internal/adapters/secondary/postgres/user_finder_test.go
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/ROADMAP.md
@.planning/STATE.md

# Port interfaces (source of truth for method signatures)
@internal/core/ports/user_finder.go

# Existing surrealdb UserFinder (pattern to port)
@internal/adapters/secondary/surrealdb/user_finder.go

# Schema to modify
@migrations/002_full_schema.up.sql

# DB pool singleton
@internal/db/pgpool.go

# SurrealDB helpers (test setup pattern to replicate)
@internal/adapters/secondary/surrealdb/helpers.go

# SurrealDB user repository test (test pattern reference)
@internal/adapters/secondary/surrealdb/user_repository_test.go

# App models for Expense and Subproject types
@internal/models/models.go

<interfaces>
Source of truth for types used by new port interfaces:

From internal/models/models.go:
```go
type Expense struct {
    ID                  uuid.UUID
    UserID              uuid.UUID
    OrganizationID      uuid.UUID
    ProjectID           *uuid.UUID
    CustomerID          *uuid.UUID
    Date                time.Time
    Type                *ExpenseCategory
    Amount              *float64
    KmDistance          *float64
    Description         string
    Status              EntryStatus
    CurrentApproverRole *string
    SubmittedAt         *time.Time
    DeletedAt           *time.Time
    CreatedAt           time.Time
    UpdatedAt           time.Time
    Items               []ExpenseItem
}
```

From internal/models/surreal_models.go:
```go
type Subproject struct {
    ID            string
    ProjectID     string
    Name          string
    Description   string
    SequenceOrder int
    IsActive      bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}
```

From internal/core/ports/user_finder.go (existing):
```go
type UserFinder interface {
    FindByIdentifier(ctx context.Context, identifier string) (userID string, err error)
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: Schema correction, sentinel errors, new port interfaces</name>

  <files>
    migrations/002_full_schema.up.sql
    internal/core/ports/errors.go
    internal/core/ports/expense_repository.go
    internal/core/ports/subproject_repository.go
  </files>

  <read_first>
    @migrations/002_full_schema.up.sql (current state — exactly as read)
    @internal/models/models.go (Expense struct lines 277-295)
    @internal/models/surreal_models.go (Subproject struct lines 47-56)
    @internal/core/ports/user_repository.go (existing port pattern)
    @internal/core/ports/time_entry_repository.go (existing port pattern with ListFilters)
  </read_first>

  <action>
    **A) Schema change — remove users.name column**
    In `migrations/002_full_schema.up.sql`, remove the line `name VARCHAR(255) NOT NULL,` (line 30) from the `users` table definition. The users table should have columns: id, email, firstname, lastname, username, password_hash, is_active, created_at, updated_at. Do NOT change anything else in this file.

    **B) Create shared sentinel errors**
    Create `internal/core/ports/errors.go`:
    - Package `ports`
    - Three sentinel errors: `var ErrNotFound = errors.New("not found")`, `var ErrConflict = errors.New("entity conflict")`, `var ErrForeignKey = errors.New("foreign key violation")`
    - Import `"errors"` only

    **C) Create ExpenseRepository port interface**
    Create `internal/core/ports/expense_repository.go`:
    ```go
    package ports

    import (
        "context"
        "github.com/google/uuid"
        "github.com/stefanoprivitera/hourglass/internal/models"
    )

    type ExpenseRepository interface {
        Create(ctx context.Context, e *models.Expense) (*models.Expense, error)
        GetByID(ctx context.Context, id uuid.UUID) (*models.Expense, error)
        ListByOrg(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]models.Expense, error)
        Update(ctx context.Context, e *models.Expense) (*models.Expense, error)
        Delete(ctx context.Context, id uuid.UUID) error
    }
    ```

    **D) Create SubprojectRepository port interface**
    Create `internal/core/ports/subproject_repository.go`:
    ```go
    package ports

    import (
        "context"
        "github.com/google/uuid"
        "github.com/stefanoprivitera/hourglass/internal/models"
    )

    type SubprojectRepository interface {
        ListByProject(ctx context.Context, projectID uuid.UUID) ([]models.Subproject, error)
        GetByID(ctx context.Context, id string) (*models.Subproject, error)
        Create(ctx context.Context, sp *models.Subproject) (*models.Subproject, error)
        Update(ctx context.Context, sp *models.Subproject) (*models.Subproject, error)
        Delete(ctx context.Context, id string) error
    }
    ```
  </action>

  <verify>
    <automated>go build ./internal/core/ports/ && go vet ./internal/core/ports/</automated>
  </verify>

  <done>
    1. `users.name` column line removed from 002_full_schema.up.sql
    2. `internal/core/ports/errors.go` exists with ErrNotFound, ErrConflict, ErrForeignKey
    3. `internal/core/ports/expense_repository.go` exists with ExpenseRepository interface
    4. `internal/core/ports/subproject_repository.go` exists with SubprojectRepository interface
    5. `go build ./internal/core/ports/` passes without errors
  </done>
</task>

<task type="auto">
  <name>Task 2: Package helpers (postgres.go, exported_test_helpers.go) + UserFinder + tests</name>

  <files>
    internal/adapters/secondary/postgres/postgres.go
    internal/adapters/secondary/postgres/exported_test_helpers.go
    internal/adapters/secondary/postgres/user_finder.go
    internal/adapters/secondary/postgres/user_finder_test.go
  </files>

  <read_first>
    @internal/core/ports/errors.go (just created by Task 1)
    @internal/core/ports/user_finder.go (FindByIdentifier signature)
    @internal/db/pgpool.go (NewPool signature)
    @internal/adapters/secondary/surrealdb/user_finder.go (pattern to port)
    @internal/models/surreal_models.go (SurrealUser struct for type reference)
    @internal/adapters/secondary/surrealdb/user_repository_test.go (test patterns)
  </read_first>

  <action>
    Create postgres package directory at `internal/adapters/secondary/postgres/`. Create four files:

    **A) postgres.go** — Package-level error wrapper
    ```go
    package postgres

    import (
        "errors"
        "fmt"

        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgconn"
        "github.com/stefanoprivitera/hourglass/internal/core/ports"
    )

    // wrapPGError translates known pgx errors into domain sentinel errors.
    // - pgx.ErrNoRows → ports.ErrNotFound
    // - unique_violation (23505) → ports.ErrConflict
    // - foreign_key_violation (23503) → ports.ErrForeignKey
    func wrapPGError(err error, op string) error {
        if err == nil {
            return nil
        }
        if errors.Is(err, pgx.ErrNoRows) {
            return fmt.Errorf("%s: %w", op, ports.ErrNotFound)
        }
        var pgErr *pgconn.PgError
        if errors.As(err, &pgErr) {
            switch pgErr.Code {
            case "23505":
                return fmt.Errorf("%s: %w", op, ports.ErrConflict)
            case "23503":
                return fmt.Errorf("%s: %w", op, ports.ErrForeignKey)
            }
        }
        return fmt.Errorf("%s: %w", op, err)
    }
    ```

    **B) exported_test_helpers.go** — Test infrastructure
    - `TestPool(t *testing.T) *pgxpool.Pool` — skips if DATABASE_URL not set, calls `db.NewPool()`
    - `SetupTestSchema(t *testing.T, pool *pgxpool.Pool)` — reads all `migrations/*.up.sql` (except seed), sorts, executes each
    - `TeardownTestSchema(t *testing.T, pool *pgxpool.Pool)` — drops all tables in correct order (CASCADE) for cleanup between test packages
    - `uniqueEmail() string` — returns `uuid.New().String() + "@test.com"`
    - `uniqueUsername() string` — returns `"user_" + uuid.New().String()[:8]`
    - `uniqueCode() string` — returns uuid.New().String()[:12]
    - Import: `"context"`, `"os"`, `"path/filepath"`, `"sort"`, `"strings"`, `"testing"`, `"github.com/google/uuid"`, `"github.com/jackc/pgx/v5/pgxpool"`, `"github.com/stefanoprivitera/hourglass/internal/db"`, `"github.com/stretchr/testify/require"`

    **C) user_finder.go** — UserFinder implementation
    ```go
    package postgres

    import (
        "context"
        "fmt"

        "github.com/google/uuid"
        "github.com/jackc/pgx/v5"
        "github.com/jackc/pgx/v5/pgxpool"
        "github.com/stefanoprivitera/hourglass/internal/core/ports"
    )

    type UserFinder struct {
        pool *pgxpool.Pool
    }

    func NewUserFinder(pool *pgxpool.Pool) *UserFinder {
        return &UserFinder{pool: pool}
    }

    func (f *UserFinder) FindByIdentifier(ctx context.Context, identifier string) (string, error) {
        const query = `SELECT id FROM users WHERE email = $1 OR username = $1 LIMIT 1`
        var id uuid.UUID
        err := f.pool.QueryRow(ctx, query, identifier).Scan(&id)
        if err != nil {
            if errors.Is(err, pgx.ErrNoRows) {
                return "", ports.ErrUserNotFound
            }
            return "", fmt.Errorf("find user by identifier: %w", err)
        }
        return id.String(), nil
    }
    ```
    NOTE: Import `"errors"` and `"github.com/jackc/pgx/v5"` in user_finder.go.

    **D) user_finder_test.go** — Integration test
    - `TestUserFinder_FindByIdentifier_ByEmail(t *testing.T)` — create a test user, call FindByIdentifier with email, expect userID string returned
    - `TestUserFinder_FindByIdentifier_ByUsername(t *testing.T)` — call FindByIdentifier with username, expect userID string returned
    - `TestUserFinder_FindByIdentifier_NotFound(t *testing.T)` — call with nonexistent identifier, expect ports.ErrUserNotFound
    - Each test: get pool from TestPool, call SetupTestSchema, create a test user via raw SQL INSERT (no repo), then call FindByIdentifier
    - Test user INSERT must NOT include `name` column (since it was removed)
    - Use `uuid.New()` for ID, `uniqueEmail()`, `uniqueUsername()`, static firstname/lastname
    - Use `context.Background()` for all queries
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. `internal/adapters/secondary/postgres/postgres.go` compiled — wrapPGError exports
    2. `internal/adapters/secondary/postgres/exported_test_helpers.go` compiled — TestPool, SetupTestSchema, unique helpers
    3. `internal/adapters/secondary/postgres/user_finder.go` implements ports.UserFinder
    4. All three test cases created and compile
    5. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Application → PostgreSQL | All SQL queries cross from Go app into PG via pgx pool. No user input reaches this boundary unparameterized. |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg2-01 | Tampering | postgres.go wrapPGError | mitigate | Use pgconn.PgError.Code for structured error detection (not string matching on error text) |
| T-pg2-02 | DoS | exported_test_helpers.go | accept | Test helpers skip when no DATABASE_URL; tests are dev-only, not production |
| T-pg2-03 | Spoofing | user_finder.go | mitigate | Query uses parameterized $1 for identifier — no string interpolation. uuid.Parse not needed since output is .String() |
</threat_model>

<verification>
```bash
go build ./internal/core/ports/
go vet ./internal/core/ports/
go build ./internal/adapters/secondary/postgres/
go vet ./internal/adapters/secondary/postgres/
# Integration tests (requires running PostgreSQL):
# DATABASE_URL="postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable" go test ./internal/adapters/secondary/postgres/ -run TestUserFinder -count=1 -v
```
</verification>

<success_criteria>
1. Schema change: `grep "name VARCHAR" migrations/002_full_schema.up.sql` returns no match for users table
2. Sentinel errors compile: `go build ./internal/core/ports/` exits 0
3. New port interfaces compile: expense_repository.go + subproject_repository.go pass vet
4. Package helpers compile: postgres.go + exported_test_helpers.go pass vet
5. UserFinder compiles and implements ports.UserFinder
6. All files committed to git
</success_criteria>

<output>
After completion, create `.planning/phases/pg-2-adapters/pg-2-01-SUMMARY.md`
</output>
