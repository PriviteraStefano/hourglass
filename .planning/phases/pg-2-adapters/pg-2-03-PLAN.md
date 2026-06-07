---
phase: pg-2-adapters
plan: 03
type: execute
wave: 2
depends_on: [pg-2-01]
files_modified:
  - internal/adapters/secondary/postgres/unit_repository.go
  - internal/adapters/secondary/postgres/unit_repository_test.go
  - internal/adapters/secondary/postgres/unit_member_repository.go
  - internal/adapters/secondary/postgres/unit_member_repository_test.go
  - internal/adapters/secondary/postgres/working_group_repository.go
  - internal/adapters/secondary/postgres/working_group_repository_test.go
  - internal/adapters/secondary/postgres/wg_member_repository.go
  - internal/adapters/secondary/postgres/wg_member_repository_test.go
  - internal/adapters/secondary/postgres/invitation_repository.go
  - internal/adapters/secondary/postgres/invitation_repository_test.go
autonomous: true
requirements: []
must_haves:
  truths:
    - UnitRepository supports ListByOrg, GetByID, Create, Update, Delete, GetDescendants, HasMembers, ListMembers, AddMember, RemoveMember, GetMemberCountsByOrg
    - Unit/UnitMember ID conversion works: domain uses string, PG uses UUID
    - WorkingGroupRepository supports ListByOrg, GetByID, Create, Update, Delete, HasMembers, ListMembers, AddMember, RemoveMember
    - WorkingGroup UUID[] arrays (unit_ids, delegate_ids) correctly scanned/converted between []string and pgtype.Array[pgtype.UUID]
    - InvitationRepository supports Create, FindByCode, FindByToken, Update
    - All repos use wrapPGError for error translation
  artifacts:
    - path: internal/adapters/secondary/postgres/unit_repository.go
      provides: implements ports.UnitRepository (11 methods with string↔UUID conversion)
    - path: internal/adapters/secondary/postgres/unit_member_repository.go
      provides: split-out unit member queries (ListMembers, AddMember, RemoveMember delegated from UnitRepository)
    - path: internal/adapters/secondary/postgres/working_group_repository.go
      provides: implements ports.WorkingGroupRepository (9 methods with UUID[] scanning)
    - path: internal/adapters/secondary/postgres/wg_member_repository.go
      provides: split-out WG member queries (ListMembers, AddMember, RemoveMember)
    - path: internal/adapters/secondary/postgres/invitation_repository.go
      provides: implements ports.InvitationRepository (4 methods)
  key_links:
    - from: unit_repository.go
      to: domain/unit/unit.go (ID is string in domain, UUID in PG)
      via: uuid.Parse → uuid.UUID.String()
    - from: working_group_repository.go
      to: working_group/working_group.go (UnitIDs/DelegateIDs are []string)
      via: pgtype.Array[pgtype.UUID] ↔ []string conversion
---

<objective>
Implement UnitRepository (with split UnitMemberRepository), WorkingGroupRepository (with split WGMemberRepository), and InvitationRepository for PostgreSQL.

Purpose: Port org hierarchy (units), execution teams (working groups), and invitation management from SurrealDB. These repos have special type mapping challenges: string↔UUID for unit IDs, UUID[] array scanning for working groups.

Output:
- unit_repository.go + unit_member_repository.go + tests
- working_group_repository.go + wg_member_repository.go + tests
- invitation_repository.go + tests
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-2-adapters/pg-2-PATTERNS.md
@.planning/phases/pg-2-adapters/pg-2-RESEARCH.md

# Port interfaces
@internal/core/ports/unit_repository.go
@internal/core/ports/working_group_repository.go
@internal/core/ports/invitation_repository.go

# SurrealDB analogs
@internal/adapters/secondary/surrealdb/unit_repository.go
@internal/adapters/secondary/surrealdb/working_group_repository.go
@internal/adapters/secondary/surrealdb/invitation_repository.go

# Domain models
@internal/core/domain/unit/unit.go
@internal/core/domain/working_group/working_group.go
@internal/core/domain/invitation/invitation.go

# Foundation
@internal/adapters/secondary/postgres/postgres.go
@internal/adapters/secondary/postgres/exported_test_helpers.go

# Schema
@migrations/002_full_schema.up.sql (units, unit_memberships, working_groups, wg_members, invitations tables)

<interfaces>
From internal/core/ports/unit_repository.go:
```go
type UnitRepository interface {
    ListByOrg(ctx context.Context, orgID uuid.UUID) ([]unit.Unit, error)
    GetByID(ctx context.Context, id string) (*unit.Unit, error)
    Create(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
    Update(ctx context.Context, u *unit.Unit) (*unit.Unit, error)
    Delete(ctx context.Context, id string) error
    GetDescendants(ctx context.Context, id string) ([]unit.Unit, error)
    HasMembers(ctx context.Context, id string) (bool, error)
    ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error)
    AddMember(ctx context.Context, m *unit.UnitMember) (*unit.UnitMember, error)
    RemoveMember(ctx context.Context, id string) error
    GetMemberCountsByOrg(ctx context.Context, orgID uuid.UUID) (map[string]int, error)
}
```

From internal/core/ports/working_group_repository.go:
```go
type WorkingGroupRepository interface {
    ListByOrg(ctx context.Context, orgID uuid.UUID, subprojectID *uuid.UUID) ([]working_group.WorkingGroup, error)
    GetByID(ctx context.Context, id uuid.UUID) (*working_group.WorkingGroup, error)
    Create(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error)
    Update(ctx context.Context, wg *working_group.WorkingGroup) (*working_group.WorkingGroup, error)
    Delete(ctx context.Context, id uuid.UUID) error
    HasMembers(ctx context.Context, id uuid.UUID) (bool, error)
    ListMembers(ctx context.Context, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error)
    AddMember(ctx context.Context, m *working_group.WorkingGroupMember) (*working_group.WorkingGroupMember, error)
    RemoveMember(ctx context.Context, id uuid.UUID) error
}
```

From internal/core/ports/invitation_repository.go:
```go
type InvitationRepository interface {
    Create(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error)
    FindByCode(ctx context.Context, code string) (*invitation.Invitation, error)
    FindByToken(ctx context.Context, token string) (*invitation.Invitation, error)
    Update(ctx context.Context, inv *invitation.Invitation) (*invitation.Invitation, error)
}
```
</interfaces>
</context>

<tasks>

<task type="auto">
  <name>Task 1: UnitRepository + UnitMemberRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/unit_repository.go
    internal/adapters/secondary/postgres/unit_member_repository.go
    internal/adapters/secondary/postgres/unit_repository_test.go
    internal/adapters/secondary/postgres/unit_member_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/unit_repository.go (interface — 11 methods)
    @internal/core/domain/unit/unit.go (Unit struct: ID string, ParentUnitID string, OrgID uuid.UUID, etc.)
    @internal/adapters/secondary/surrealdb/unit_repository.go (analog — pattern to port)
    @migrations/002_full_schema.up.sql (units table: id UUID, org_id UUID, parent_unit_id UUID self-ref)
  </read_first>

  <action>
    **A) unit_repository.go** — implements ports.UnitRepository (11 methods)
    Struct `UnitRepository` with `pool *pgxpool.Pool`, constructor `NewUnitRepository`.

    **Key pattern: string↔UUID conversion at adapter boundary**
    - Domain `unit.Unit.ID` is `string`, PG `units.id` is `UUID`
    - Every method that takes/returns string ID: `uuid.Parse()` → query → scan `uuid.UUID` → `.String()`
    - `parent_unit_id` is nullable UUID → scan into `*uuid.UUID`, convert to string if non-nil

    **Methods:**

    1. **ListByOrg**: `SELECT id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at FROM units WHERE org_id = $1 ORDER BY hierarchy_level, name`
       - Scan each row: local vars `var sqlID uuid.UUID`, `var parentID *uuid.UUID`, `u.OrgID`, etc.
       - After scan: `u.ID = sqlID.String()`, if parentID != nil `u.ParentUnitID = parentID.String()`
       - Return `[]unit.Unit{}` not nil

    2. **GetByID**: Parse id → QueryRow → scan → convert UUID→string → return
       - ErrNoRows → `unit.ErrUnitNotFound`

    3. **Create**: Generate `uuid.New()` for the new unit's UUID. `INSERT INTO units (id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at`
       - If `u.ParentUnitID != ""`, parse to uuid.UUID for INSERT
       - QueryRow scan back UUID→string for return value
       - Wrap with wrapPGError

    4. **Update**: Dynamic SET building (like RESEARCH Pattern 11). Set name always, conditionally description, code, parent_unit_id. Always set updated_at = NOW(). `UPDATE units SET ... WHERE id = $N` then GetByID the result.

    5. **Delete**: `DELETE FROM units WHERE id = $1` (parse string to uuid.UUID first)

    6. **GetDescendants**: `WITH RECURSIVE unit_tree AS (SELECT id, org_id, name, description, parent_unit_id, hierarchy_level, code, created_at, updated_at FROM units WHERE id = $1 UNION ALL SELECT u.id, u.org_id, u.name, u.description, u.parent_unit_id, u.hierarchy_level, u.code, u.created_at, u.updated_at FROM units u INNER JOIN unit_tree ut ON u.parent_unit_id = ut.id) SELECT * FROM unit_tree WHERE id != $1 ORDER BY hierarchy_level`
       - Parse input id, CTE query, scan each row (UUID→string conversion)

    7. **HasMembers**: `SELECT EXISTS(SELECT 1 FROM unit_memberships WHERE unit_id = $1)`

    8. **ListMembers**: Delegates to unit_member_repository.go (same struct or separate). For UnitRepository, implement delegation:
       ```go
       func (r *UnitRepository) ListMembers(ctx context.Context, unitID string) ([]unit.UnitMember, error) {
           memberRepo := &UnitMemberRepository{pool: r.pool}
           return memberRepo.ListByUnit(ctx, unitID)
       }
       ```

    9. **AddMember**: Delegates to UnitMemberRepository

    10. **RemoveMember**: Delegates to UnitMemberRepository

    11. **GetMemberCountsByOrg**: `SELECT unit_id, COUNT(*) FROM unit_memberships WHERE org_id = $1 GROUP BY unit_id`
        - Scan uuid.UUID → String() for map keys

    **B) unit_member_repository.go** — split-out file
    Struct `UnitMemberRepository` with `pool *pgxpool.Pool`, constructor `NewUnitMemberRepository`.
    Methods:
    - `ListByUnit(ctx, unitID string) ([]unit.UnitMember, error)` — JOINs unit_memberships with users for user_name/user_email
      ```sql
      SELECT um.id, um.org_id, um.user_id, u.firstname || ' ' || u.lastname AS user_name, u.email AS user_email, um.unit_id, um.is_primary, um.role, um.start_date, um.end_date, um.created_at
      FROM unit_memberships um
      LEFT JOIN users u ON u.id = um.user_id
      WHERE um.unit_id = $1 ORDER BY um.created_at
      ```
    - `Add(ctx, m *unit.UnitMember) (*unit.UnitMember, error)` — INSERT ... RETURNING *
    - `Remove(ctx, id string) error` — DELETE FROM unit_memberships WHERE id = $1

    **C) Test files:**
    - `unit_repository_test.go`: Test ListByOrg, Test Create→GetByID, Test Update, Test Delete, Test GetDescendants, Test HasMembers, Test GetMemberCountsByOrg
    - `unit_member_repository_test.go`: Test ListByUnit, Test Add→ListByUnit, Test Remove
    - Create seed org + units via raw SQL. Use unique names for each test.
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. UnitRepository with 11 methods compiles
    2. UnitMemberRepository with 3 methods compiles
    3. All ID conversions (string↔UUID) handled correctly
    4. Test files compile
    5. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 2: WorkingGroupRepository + WGMemberRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/working_group_repository.go
    internal/adapters/secondary/postgres/wg_member_repository.go
    internal/adapters/secondary/postgres/working_group_repository_test.go
    internal/adapters/secondary/postgres/wg_member_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/working_group_repository.go (interface — 9 methods)
    @internal/core/domain/working_group/working_group.go (WorkingGroup: UnitIDs []string, DelegateIDs []string)
    @internal/adapters/secondary/surrealdb/working_group_repository.go (analog)
    @migrations/002_full_schema.up.sql (working_groups: unit_ids UUID[], delegate_ids UUID[], subproject_id UUID FK)
    @internal/adapters/secondary/postgres/exported_test_helpers.go
  </read_first>

  <action>
    **A) working_group_repository.go** — implements ports.WorkingGroupRepository (9 methods)
    Struct `WorkingGroupRepository` with `pool *pgxpool.Pool`, constructor `NewWorkingGroupRepository`.

    **UUID[] array scanning pattern for UnitIDs and DelegateIDs:**
    ```go
    var unitIDs pgtype.Array[pgtype.UUID]
    var delegateIDs pgtype.Array[pgtype.UUID]
    // ... in Scan: &unitIDs, &delegateIDs
    wg.UnitIDs = make([]string, len(unitIDs.Elements))
    for i, u := range unitIDs.Elements {
        if u.Valid {
            wg.UnitIDs[i] = uuid.UUID(u.Bytes).String()
        }
    }
    ```

    **UUID[] array parameter pattern for INSERT/UPDATE:**
    ```go
    uuidParams := make([]pgtype.UUID, len(wg.UnitIDs))
    for i, idStr := range wg.UnitIDs {
        uid, _ := uuid.Parse(idStr)
        uuidParams[i] = pgtype.UUID{Bytes: uid, Valid: true}
    }
    ```

    **Methods:**

    1. **ListByOrg**: `SELECT id, org_id, subproject_id, name, description, unit_ids, enforce_unit_tuple, manager_id, delegate_ids, is_active, created_at, updated_at FROM working_groups WHERE org_id = $1 AND is_active = true ORDER BY name`
       - Optional: if subprojectID != nil, add `AND subproject_id = $2`

    2. **GetByID**: Same SELECT with `WHERE id = $1`, ErrNoRows → `working_group.ErrWorkingGroupNotFound`

    3. **Create**: INSERT with RETURNING, scan back UUID[] arrays

    4. **Update**: Dynamic SET, include unit_ids/delegate_ids as UUID[] params

    5. **Delete**: `DELETE FROM working_groups WHERE id = $1`

    6. **HasMembers**: `SELECT EXISTS(SELECT 1 FROM wg_members WHERE wg_id = $1)`

    7. **ListMembers**: Delegates to WGMemberRepository

    8. **AddMember**: Delegates to WGMemberRepository

    9. **RemoveMember**: Delegates to WGMemberRepository

    **B) wg_member_repository.go** — split-out file
    Struct `WGMemberRepository` with `pool *pgxpool.Pool`, constructor `NewWGMemberRepository`.
    Methods:
    - `ListByWG(ctx, wgID uuid.UUID) ([]working_group.WorkingGroupMember, error)` — SELECT from wg_members WHERE wg_id = $1
    - `Add(ctx, m *working_group.WorkingGroupMember) (*working_group.WorkingGroupMember, error)` — INSERT ... RETURNING *
    - `Remove(ctx, id uuid.UUID) error` — DELETE FROM wg_members WHERE id = $1
    - `HasMembers(ctx, wgID uuid.UUID) (bool, error)` — SELECT EXISTS for wg_members

    **C) Test files:**
    - `working_group_repository_test.go`: Test ListByOrg, Test Create→GetByID (verify UUID[] arrays round-trip correctly), Test Update (change arrays), Test Delete, Test HasMembers
    - `wg_member_repository_test.go`: Test ListByWG, Test Add→ListByWG, Test Remove
    - Create seed subproject + working groups with UUID[] arrays. Verify array scan/parameter round-trip: insert a WG with unit_ids=["uuid1","uuid2"], retrieve and assert []string contains both values.
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. WorkingGroupRepository with 9 methods compiles
    2. WGMemberRepository with 4 methods compiles
    3. UUID[] array scanning/conversion works correctly
    4. Test files compile
    5. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

<task type="auto">
  <name>Task 3: InvitationRepository + tests</name>

  <files>
    internal/adapters/secondary/postgres/invitation_repository.go
    internal/adapters/secondary/postgres/invitation_repository_test.go
  </files>

  <read_first>
    @internal/core/ports/invitation_repository.go (interface — 4 methods)
    @internal/core/domain/invitation/invitation.go (Invitation struct)
    @internal/adapters/secondary/surrealdb/invitation_repository.go (analog)
    @migrations/002_full_schema.up.sql (invitations table: id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at)
  </read_first>

  <action>
    **A) invitation_repository.go** — implements ports.InvitationRepository (4 methods)
    Struct `InvitationRepository` with `pool *pgxpool.Pool`, constructor `NewInvitationRepository`.

    **Methods:**

    1. **Create**: `INSERT INTO invitations (id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at`
       - inv.CreatedBy is `string` (uuid as string) — parse to uuid.UUID for INSERT
       - QueryRow → scan into *invitation.Invitation
       - Wrap with wrapPGError

    2. **FindByCode**: `SELECT id, organization_id, code, invite_token, email, status, created_by, expires_at, created_at FROM invitations WHERE code = $1 LIMIT 1`
       - QueryRow → Scan. ErrNoRows → `invitation.ErrInvitationNotFound`
       - created_by UUID → .String() for Invitation.CreatedBy

    3. **FindByToken**: Same query with `WHERE invite_token = $1`

    4. **Update**: `UPDATE invitations SET status = $1 WHERE id = $2` then GetByID (or RETURNING * with QueryRow)

    **B) invitation_repository_test.go:**
    - Test Create → FindByCode (create, retrieve by code, assert match)
    - Test Create → FindByToken (create, retrieve by token, assert match)
    - Test FindByCode not found returns ErrInvitationNotFound
    - Test Update (create, update status, retrieve, assert updated)
    - Create seed org + user for FK references via raw SQL
  </action>

  <verify>
    <automated>go vet ./internal/adapters/secondary/postgres/ && go build ./internal/adapters/secondary/postgres/</automated>
  </verify>

  <done>
    1. InvitationRepository with 4 methods compiles
    2. Test file compiles
    3. `go build ./internal/adapters/secondary/postgres/` passes
  </done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| Application → PostgreSQL unit/WG queries | Unit hierarchy and WG member data queries |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg2-08 | Tampering | unit_repository GetDescendants (CTE) | mitigate | CTE is fully static SQL — no dynamic WHERE. uuid.Parse rejects invalid input before query. |
| T-pg2-09 | Tampering | working_group UUID[] params | mitigate | UUID[] parameter building uses uuid.Parse for each string — invalid strings rejected before SQL |
| T-pg2-10 | Information Disclosure | invitation FindByCode/FindByToken | mitigate | Only returns a single invitation by unique code/token — no bulk enumeration possible |
</threat_model>

<verification>
```bash
go vet ./internal/adapters/secondary/postgres/
go build ./internal/adapters/secondary/postgres/
# Unit tests:
# DATABASE_URL=... go test ./internal/adapters/secondary/postgres/ -run "TestUnit|TestWorkingGroup|TestInvitation" -count=1 -v
```
</verification>

<success_criteria>
1. UnitRepository implements all 11 methods, with string↔UUID conversion at adapter boundary
2. WorkingGroupRepository implements all 9 methods, with UUID[] array scanning
3. InvitationRepository implements all 4 methods
4. All tests compile
5. Files committed to git
</success_criteria>

<output>
After completion, create `.planning/phases/pg-2-adapters/pg-2-03-SUMMARY.md`
</output>
