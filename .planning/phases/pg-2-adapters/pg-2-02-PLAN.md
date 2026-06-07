---
phase: pg-2-adapters
plan: 02
type: execute
wave: 2
depends_on: [pg-2-01]
files_modified:
  - internal/adapters/secondary/postgres/user_repository.go
  - internal/adapters/secondary/postgres/user_repository_test.go
  - internal/adapters/secondary/postgres/organization_repo.go
  - internal/adapters/secondary/postgres/organization_repo_test.go
  - internal/adapters/secondary/postgres/organization_membership_repo.go
  - internal/adapters/secondary/postgres/organization_membership_repo_test.go
  - internal/adapters/secondary/postgres/organization_management_repo.go
  - internal/adapters/secondary/postgres/organization_management_repo_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - UserRepository supports Add, GetByEmail, GetByUsername, GetByID, EmailExists, UsernameExists,
      AnyExists, UpdatePassword, GetMemberships, AddWithMembership, AddWithOrgAndMembership
    - OrganizationRepository supports Add, GetByID, GetMembership, AddMembership
    - OrganizationManagementRepository supports CreateOrganization, GetOrganization, InviteMember,
      GetSettings, UpdateSettings, ListMembers, UpdateMemberRole, DeactivateMember, CountActiveFinance, GetMemberRole
    - All queries exclude users.name column (removed from schema)
    - All repos use wrapPGError for error translation
    - Repository tests pass against a live PostgreSQL with all migrations applied
  artifacts:
    - path: internal/adapters/secondary/postgres/user_repository.go
      provides: implements ports.UserRepository (11 methods)
    - path: internal/adapters/secondary/postgres/organization_repo.go
      provides: implements ports.OrganizationRepository (4 methods)
    - path: internal/adapters/secondary/postgres/organization_membership_repo.go
      provides: OrganizationMembershipRepository (standalone membership queries extracted from org_repo)
    - path: internal/adapters/secondary/postgres/organization_management_repo.go
      provides: implements ports.OrganizationManagementRepository (10 methods)
    - path: internal/adapters/secondary/postgres/organization_management_repo.go
      provides: settings queries against organization_settings table (007_phase2_schema)
  key_links:
    - from: user_repository.go
      to: ports.UserRepository
      via: func (*UserRepository) matches each interface method
    - from: organization_management_repo.go
      to: organization_settings table (007_phase2_schema)
      via: SELECT/UPDATE on organization_settings
---

<objective>
Implement UserRepository, OrganizationRepository, OrganizationMembershipRepository, and OrganizationManagementRepository for PostgreSQL.

Purpose: Port the auth/org domain's SurrealDB repositories to handwritten pgx SQL. These are the most foundational repos — used by auth flows, org management, and nearly every other domain through FK references.

Output:
- user_repository.go (11 methods: Add, GetByEmail, GetByUsername, GetByID, EmailExists, UsernameExists, AnyExists, UpdatePassword, GetMemberships, AddWithMembership, AddWithOrgAndMembership)
- organization_repo.go (4 methods: Add, GetByID, GetMembership, AddMembership)
- organization_membership_repo.go (standalone queries split from org_repo)
- organization_management_repo.go (10 methods: CreateOrganization, GetOrganization, InviteMember, GetSettings, UpdateSettings, ListMembers, UpdateMemberRole, DeactivateMember, CountActiveFinance, GetMemberRole)
- Corresponding test files
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-2-adapters/pg-2-01-PLAN.md
@.planning/phases/pg-2-adapters/pg-2-PATTERNS.md
@.planning/phases/pg-2-adapters/pg-2-RESEARCH.md

# Port interfaces (method contracts)
@internal/core/ports/user_repository.go
@internal/core/ports/organization_repository.go
@internal/core/ports/organization_management_repository.go

# SurrealDB analogs (patterns to port)
@internal/adapters/secondary/surrealdb/user_repository.go
@internal/adapters/secondary/surrealdb/organization_repo.go
@internal/adapters/secondary/surrealdb/organization_management_repository.go

# Domain models
@internal/core/domain/auth/user.go
@internal/core/domain/auth/organization.go
@internal/core/domain/auth/membership.go
@internal/core/domain/organization/organization.go

# Foundation
@internal/adapters/secondary/postgres/postgres.go
@internal/adapters/secondary/postgres/exported_test_helpers.go

# Schema for settings table
@migrations/007_phase2_schema.up.sql

<interfaces>
From internal/core/ports/user_repository.go:
```go
type UserRepository interface {
    Add(ctx context.Context, user *auth.User) error
    AddWithMembership(ctx context.Context, user *auth.User, membership *auth.OrganizationMembership) error
    AddWithOrgAndMembership(ctx context.Context, user *auth.User, org *auth.Organization, membership *auth.OrganizationMembership) error
    GetByEmail(ctx context.Context, email string) (*auth.User, error)
    GetByUsername(ctx context.Context, username string) (*auth.User, error)
    GetByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
    EmailExists(ctx context.Context, email string) (bool, error)
    UsernameExists(ctx context.Context, username string) (bool, error)
    AnyExists(ctx context.Context) (bool, error)
    UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
    GetMemberships(ctx context.Context, userID uuid.UUID) ([]auth.OrganizationMembership, error)
}
```

From internal/core/ports/organization_repository.go:
```go
type OrganizationRepository interface {
    Add(ctx context.Context, org *auth.Organization) error
    GetByID(ctx context.Context, id uuid.UUID) (*auth.Organization, error)
    GetMembership(ctx context.Context, userID, orgID uuid.UUID) (*auth.OrganizationMembership, error)
    AddMembership(ctx context.Context, membership *auth.OrganizationMembership) error
}
```

From internal/core/ports/organization_management_repository.go:
```go
type OrganizationManagementRepository interface {
    CreateOrganization(ctx context.Context, org *orgdomain.Organization, ownerUserID uuid.UUID, ownerRole models.Role) error
    GetOrganization(ctx context.Context, id uuid.UUID) (*orgdomain.Organization, error)
    InviteMember(ctx context.Context, orgID uuid.UUID, req *orgdomain.InviteRequest, invitedBy uuid.UUID) (uuid.UUID, time.Time, error)
    GetSettings(ctx context.Context, orgID uuid.UUID) (*orgdomain.Settings, error)
    UpdateSettings(ctx context.Context, orgID uuid.UUID, req *orgdomain.UpdateSettingsRequest) (*orgdomain.Settings, error)
    ListMembers(ctx context.Context, orgID uuid.UUID) ([]orgdomain.Member, error)
    UpdateMemberRole(ctx context.Context, orgID, memberID uuid.UUID, role models.Role) error
    DeactivateMember(ctx context.Context, orgID, memberID uuid.UUID) error
    CountActiveFinance(ctx context.Context, orgID uuid.UUID) (int, error)
    GetMemberRole(ctx context.Context, memberID uuid.UUID) (models.Role, error)
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: UserRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/user_repository.go
    internal/adapters/secondary/postgres/user_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/user_repository.go (interface — every method required)
    @internal/core/domain/auth/user.go (User struct — NO Name field)
    @internal/core/domain/auth/membership.go (OrganizationMembership struct)
    @internal/adapters/secondary/surrealdb/user_repository.go (analog to port)
    @internal/adapters/secondary/postgres/postgres.go (wrapPGError)
    @internal/adapters/secondary/postgres/exported_test_helpers.go (TestPool, SetupTestSchema)
    @internal/adapters/secondary/surrealdb/user_repository_test.go (test pattern)
  </read_first>

  <action>
    **A) user_repository.go**
    Create `internal/adapters/secondary/postgres/user_repository.go`:
    - Package `postgres`
    - Struct `UserRepository` with `pool *pgxpool.Pool`
    - Constructor `NewUserRepository(pool *pgxpool.Pool) *UserRepository`
    - Implement ALL 11 methods from `ports.UserRepository`:

    **SQL patterns for each method (users.name column removed — never reference it):**

    1. **Add**: `INSERT INTO users (id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)` — use pool.Exec, wrap error with wrapPGError

    2. **GetByEmail**: `SELECT id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at FROM users WHERE email = $1` — QueryRow + Scan, ErrNoRows → ports.ErrUserNotFound

    3. **GetByUsername**: Same SELECT with `WHERE username = $1`

    4. **GetByID**: Same SELECT with `WHERE id = $1`, takes uuid.UUID param

    5. **EmailExists**: `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

    6. **UsernameExists**: `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`

    7. **AnyExists**: `SELECT EXISTS(SELECT 1 FROM users)`

    8. **UpdatePassword**: `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`

    9. **GetMemberships**: `SELECT id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at FROM organization_memberships WHERE user_id = $1 ORDER BY created_at DESC` — pool.Query + rows iteration, return `[]auth.OrganizationMembership{}` not nil

    10. **AddWithMembership**: pool.Begin → tx.Exec users INSERT → tx.Exec organization_memberships INSERT → tx.Commit. defer tx.Rollback. Use wrapPGError on each Exec.

    11. **AddWithOrgAndMembership**: pool.Begin → tx.Exec organizations INSERT → tx.Exec users INSERT → tx.Exec organization_memberships INSERT → tx.Commit. defer tx.Rollback.

    **Column mapping for users table (name column removed):**
    - SELECT: `id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at` (8 columns)
    - INSERT: `(id, email, username, firstname, lastname, password_hash, is_active, created_at, updated_at)` (9 params)
    - Scan order: &u.ID, &u.Email, &u.Username, &u.FirstName, &u.LastName, &u.PasswordHash, &u.IsActive, &u.CreatedAt, &u.UpdatedAt

    **Column mapping for organization_memberships table:**
    - SELECT: `id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at` (10 columns)
    - Scan into: &m.ID, &m.UserID, &m.OrganizationID, &m.Role, &m.IsActive, &m.InvitedBy, &m.InvitedAt, &m.ActivatedAt, &m.CreatedAt, &m.UpdatedAt
    - invited_by is *uuid.UUID, invited_at and activated_at are *time.Time

    **B) user_repository_test.go**
    Test file with following test functions (each using TestPool + SetupTestSchema):
    - `TestUserRepository_AddAndGetByID` — create user via Add, retrieve via GetByID, assert all fields match
    - `TestUserRepository_GetByEmail` — create user, GetByEmail, assert match
    - `TestUserRepository_GetByUsername` — create user, GetByUsername, assert match
    - `TestUserRepository_EmailExists` — create user, assert EmailExists true for created email, false for unknown
    - `TestUserRepository_UsernameExists` — same pattern
    - `TestUserRepository_AnyExists` — assert true when users exist, false in fresh DB
    - `TestUserRepository_UpdatePassword` — create user, update password, verify updated via GetByID
    - `TestUserRepository_GetMemberships` — create user + org + membership, assert membership returned
    - `TestUserRepository_AddWithMembership` — create user+membership in transaction, assert both exist
    - `TestUserRepository_AddWithOrgAndMembership` — create user+org+membership in transaction, assert all exist

    Use `context.Background()` for all queries. Use `require.NoError` for success assertions. Use `uniqueEmail()`, `uniqueUsername()` helpers. Create test organizations and memberships as needed (raw SQL INSERT since org_repo doesn't exist yet in this task).
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. UserRepository with all 11 methods compiles
    2. No query references `users.name` column
    3. All 9 test cases compile
    4. `go build ./internal/adapters/secondary/postgres/` passes
    5. `DATABASE_URL=... go test ./internal/adapters/secondary/postgres/ -run TestUserRepository -count=1` passes (if PG running)
  </done>
</task>

<task type="auto">
  <name>Task 2: Organization repos (org_repo, org_membership_repo, org_management_repo) + tests</name>

  <files>
    internal/adapters/secondary/postgres/organization_repo.go
    internal/adapters/secondary/postgres/organization_repo_test.go
    internal/adapters/secondary/postgres/organization_membership_repo.go
    internal/adapters/secondary/postgres/organization_membership_repo_test.go
    internal/adapters/secondary/postgres/organization_management_repo.go
    internal/adapters/secondary/postgres/organization_management_repo_test.go
  </files>

  <read_first>
    @internal/core/ports/organization_repository.go (interface — 4 methods)
    @internal/core/ports/organization_management_repository.go (interface — 10 methods)
    @internal/core/domain/auth/organization.go (Organization struct with FinancialCutoffConfig)
    @internal/core/domain/auth/membership.go (OrganizationMembership)
    @internal/core/domain/organization/organization.go (orgdomain.Organization, Settings, Member, InviteRequest, UpdateSettingsRequest)
    @internal/adapters/secondary/surrealdb/organization_repo.go (analog to port)
    @internal/adapters/secondary/surrealdb/organization_management_repository.go (analog to port)
    @migrations/002_full_schema.up.sql (organizations, organization_memberships tables)
    @migrations/007_phase2_schema.up.sql (organization_settings table)
  </read_first>

  <action>
    **A) organization_repo.go** — implements ports.OrganizationRepository
    Struct `OrganizationRepository` with `pool *pgxpool.Pool`, constructor `NewOrganizationRepository`.
    4 methods:

    1. **Add**: `INSERT INTO organizations (id, name, slug, description, financial_cutoff_days, financial_cutoff_config, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
       - `org.FinancialCutoffConfig` is `map[string]interface{}` — marshal to JSON for JSONB column:
         ```go
         configJSON, err := json.Marshal(org.FinancialCutoffConfig)
         ```
       - Wrap with wrapPGError

    2. **GetByID**: `SELECT id, name, slug, description, financial_cutoff_days, financial_cutoff_config, created_at, updated_at FROM organizations WHERE id = $1`
       - Scan `financial_cutoff_config` into `json.RawMessage`, then `json.Unmarshal` to `map[string]interface{}`
       - ErrNoRows → return nil, nil (matching current SurrealDB behavior which returns nil not ErrOrganizationNotFound)

    3. **GetMembership**: `SELECT id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at FROM organization_memberships WHERE user_id = $1 AND organization_id = $2 LIMIT 1`
       - Returns nil, nil if not found (matching existing behavior)

    4. **AddMembership**: `INSERT INTO organization_memberships (id, user_id, organization_id, role, is_active, invited_by, invited_at, activated_at, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
       - Wrap with wrapPGError

    **B) organization_membership_repo.go** — split-out membership queries
    This file holds membership-specific queries that surrealdb keeps in organization_repo.go but we split for clarity.
    Struct `OrganizationMembershipRepository` with `pool *pgxpool.Pool`:
    - Not a port interface — it's a concrete helper that services can construct
    - Constructor `NewOrganizationMembershipRepository(pool *pgxpool.Pool)`
    - Methods extracted from the existing surrealdb org_repo patterns:
      - `ListByUser(ctx, userID uuid.UUID) ([]auth.OrganizationMembership, error)` — same query as UserRepository.GetMemberships (avoids coupling)
      - `ListByOrg(ctx, orgID uuid.UUID) ([]auth.OrganizationMembership, error)` — SELECT * FROM organization_memberships WHERE organization_id = $1 ORDER BY created_at DESC

    **C) organization_management_repo.go** — implements ports.OrganizationManagementRepository
    Struct `OrganizationManagementRepository` with `pool *pgxpool.Pool`, constructor `NewOrganizationManagementRepository`.
    10 methods:

    1. **CreateOrganization**: pool.Begin → INSERT organizations → INSERT organization_memberships (with owner role) → Commit. The `organization_settings` row is auto-created by a DB trigger (from 007 migration). Return wrapPGError.

    2. **GetOrganization**: `SELECT id, name, slug, created_at FROM organizations WHERE id = $1`
       - Scan into `orgdomain.Organization`. ErrNoRows → orgdomain.ErrOrganizationNotFound.

    3. **InviteMember**: INSERT into organization_memberships with user_id=NULL, invited_by set, invited_at=NOW(). Return (membershipID, invitedAt, error).

    4. **GetSettings**: `SELECT organization_id, default_km_rate, currency, week_start_day, timezone, show_approval_history, created_at, updated_at FROM organization_settings WHERE organization_id = $1`
       - ErrNoRows → orgdomain.ErrOrganizationNotFound
       - default_km_rate is *float64 (nullable)

    5. **UpdateSettings**: Build dynamic UPDATE SET from non-zero fields (like Pattern 11 in RESEARCH), then SELECT the updated row. Use `UPDATE organization_settings SET ... WHERE organization_id = $N` then re-GetSettings.

    6. **ListMembers**: `SELECT om.id, om.user_id, om.role, om.is_active, om.invited_by, om.invited_at, om.activated_at, u.firstname || ' ' || u.lastname AS user_name, u.email AS user_email FROM organization_memberships om LEFT JOIN users u ON u.id = om.user_id WHERE om.organization_id = $1 ORDER BY om.created_at DESC`
       - Scan into orgdomain.Member struct fields. user_id and invited_by are *uuid.UUID.

    7. **UpdateMemberRole**: `UPDATE organization_memberships SET role = $1, updated_at = NOW() WHERE id = $2 AND organization_id = $3`

    8. **DeactivateMember**: `UPDATE organization_memberships SET is_active = false, updated_at = NOW() WHERE id = $1 AND organization_id = $2`

    9. **CountActiveFinance**: `SELECT COUNT(*) FROM organization_memberships WHERE organization_id = $1 AND role = 'finance' AND is_active = true`

    10. **GetMemberRole**: `SELECT role FROM organization_memberships WHERE id = $1` — ErrNoRows → orgdomain.ErrMemberNotFound

    **D) Test files** for all three repos:
    - `organization_repo_test.go`: Test Add → GetByID; Test AddMembership → GetMembership; Test GetMembership not found returns nil,nil
    - `organization_membership_repo_test.go`: Test ListByUser (create org+membership, list); Test ListByOrg
    - `organization_management_repo_test.go`: Test CreateOrganization (verifies org + membership created); Test GetSettings; Test UpdateSettings (change specific field, verify); Test ListMembers (verify user_name is concatenated); Test UpdateMemberRole; Test DeactivateMember; Test CountActiveFinance; Test InviteMember
    - Each test creates seed data (users, orgs) via raw SQL INSERT
    - Use `context.Background()`, `require.NoError`, unique helpers
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. OrganizationRepository with 4 methods compiles
    2. OrganizationMembershipRepository with 2 methods compiles
    3. OrganizationManagementRepository with 10 methods compiles
    4. All test cases compile
    5. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Application → PostgreSQL org queries | Organization data flows through parameterized queries |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg2-04 | Tampering | org_management CreateOrganization | mitigate | Uses pgx.Begin/Commit for atomic org+membership creation — partial creation impossible |
| T-pg2-05 | Spoofing | org_repo GetMembership | mitigate | Parameterized $1/$2 for userID/orgID — no string interpolation |
| T-pg2-06 | Tampering | org_management UpdateSettings dynamic WHERE | mitigate | Dynamic SET building only injects placeholder positions ($N), never raw values |
| T-pg2-07 | Information Disclosure | org_management ListMembers | mitigate | Uses LEFT JOIN on users — only exposes firstname/lastname (concatenated) and email, not password_hash |
</threat_model>

<verification>
```bash
go vet ./internal/adapters/secondary/postgres/
go build ./internal/adapters/secondary/postgres/
# Integration tests:
# DATABASE_URL="postgres://hourglass:hourglass@localhost:5432/hourglass?sslmode=disable" go test ./internal/adapters/secondary/postgres/ -run "TestUserRepository|TestOrganization" -count=1 -v
```
</verification>

<success_criteria>
1. UserRepository implements all 11 methods from ports.UserRepository
2. OrganizationRepository implements all 4 methods from ports.OrganizationRepository
3. OrganizationManagementRepository implements all 10 methods from ports.OrganizationManagementRepository
4. No query references users.name column
5. All tests compile and can pass against PostgreSQL
6. Files committed to git
</success_criteria>

<output>
After completion, create `.planning/phases/pg-2-adapters/pg-2-02-SUMMARY.md`
</output>
