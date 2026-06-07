---
phase: pg-1-foundation
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - migrations/002_full_schema.up.sql
  - migrations/002_full_schema.down.sql
  - migrations/003_seed.up.sql
  - migrations/003_seed.down.sql
autonomous: true
requirements:
  - D-01 (CREATE TABLE IF NOT EXISTS)
  - D-02 (UUID PK gen_random_uuid)
  - D-03 (FK ON DELETE CASCADE/RESTRICT)
  - D-04 (TIMESTAMPTZ)
  - D-05 (JSONB)
  - D-06 (CHECK constraints)
  - D-07 (Indexes)
  - D-12 (002_full_schema.up.sql)
  - D-13 (002_full_schema.down.sql)
  - D-14 (003_seed.up.sql ON CONFLICT DO NOTHING)
  - D-15 (003_seed.down.sql)
  - D-16 (bcrypt pre-hashed passwords)
must_haves:
  truths:
    - "002_full_schema.up.sql contains all 24 tables with the correct columns, types, constraints, and indexes"
    - "002_full_schema.down.sql drops all 24 tables in reverse FK dependency order"
    - "003_seed.up.sql inserts the full MVP demo dataset idempotently with ON CONFLICT DO NOTHING"
    - "003_seed.down.sql deletes only seed records by known UUIDs (not TRUNCATE)"
    - "Every seed entity has a deterministic UUID (string-IDs from SurrealDB are replaced with UUIDs)"
  artifacts:
    - path: "migrations/002_full_schema.up.sql"
      provides: "Full PostgreSQL schema for all 24 entities"
    - path: "migrations/002_full_schema.down.sql"
      provides: "Complete schema rollback"
    - path: "migrations/003_seed.up.sql"
      provides: "Idempotent MVP demo data"
    - path: "migrations/003_seed.down.sql"
      provides: "Seed data rollback"
  key_links:
    - from: "003_seed.up.sql"
      to: "002_full_schema.up.sql"
      via: "FK references — seed INSERT order must respect FK dependency order"
---

<objective>
Create the complete PostgreSQL schema migration (002_full_schema) and the MVP demo seed data migration (003_seed).

Purpose: Translate the SurrealDB schema (schema/001_schema.surql) into strict PostgreSQL DDL with proper types, constraints, indexes, and FKs. Seed the database with the same 6-user / 8-unit / 3-contract / 6-project MVP demo data currently in schema/003_seed_demo.surql, using deterministic UUIDs for every entity.

Output:
- migrations/002_full_schema.up.sql — all 24 tables
- migrations/002_full_schema.down.sql — DROP TABLE IF EXISTS CASCADE for all
- migrations/003_seed.up.sql — idempotent INSERT ... ON CONFLICT DO NOTHING
- migrations/003_seed.down.sql — DELETE by known seed UUIDs
</objective>

<execution_context>
@/Users/stefanoprivitera/.config/opencode/get-shit-done/workflows/execute-plan.md
@/Users/stefanoprivitera/.config/opencode/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/phases/pg-1-foundation/pg-1-RESEARCH.md
@.planning/phases/pg-1-foundation/pg-1-PATTERNS.md
@migrations/001_init.up.sql
@migrations/001_init.down.sql
@schema/001_schema.surql
@schema/003_seed_demo.surql
@internal/models/models.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Create 002_full_schema.up.sql and 002_full_schema.down.sql</name>
  <files>
    migrations/002_full_schema.up.sql
    migrations/002_full_schema.down.sql
  </files>

  <read_first>
    - migrations/001_init.up.sql — existing DDL pattern (CREATE TABLE IF NOT EXISTS, gen_random_uuid, FKs, CHECK, indexes, timestamptz)
    - migrations/001_init.down.sql — existing rollback pattern (DROP INDEX IF EXISTS, DROP TABLE IF EXISTS CASCADE)
    .planning/phases/pg-1-foundation/pg-1-RESEARCH.md — full table-by-table mapping (lines 326-669)
    .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — pattern assignments section (lines 27-108)
  </read_first>

  <action>
    Create migrations/002_full_schema.up.sql with all 24 tables. Every table must follow the exact column types, constraints, and defaults from the RESEARCH.md table-by-table mapping (sections 1-24, lines 326-669). Use the following patterns consistently:

    Leave a blank line after `CREATE EXTENSION IF NOT EXISTS pgcrypto;` (may already exist from 001_init — idempotent).
    After each table, define its indexes using `CREATE INDEX IF NOT EXISTS` naming.

    Apply these rules to every table:
    - D-01: `CREATE TABLE IF NOT EXISTS` for every table
    - D-02: `id UUID PRIMARY KEY DEFAULT gen_random_uuid()` on every table
    - D-03: `REFERENCES parent(id) ON DELETE CASCADE` where SurrealDB cascaded (organization_memberships, unit_memberships, project_adoptions, contract_adoptions, project_managers, subprojects, wg_members, time_entries, time_entry_approvals, expenses, expense_approvals, invitations, password_resets, refresh_tokens, backup_approvers). Use `ON DELETE RESTRICT` for: units(parent_unit_id), contracts(customer_id), projects(contract_id), wg_members(unit_id). Use plain `REFERENCES parent(id)` without explicit ON DELETE for: organizations, customers, units(org_id), working_groups(org_id), budget_caps, financial_cutoff_periods (these reference parents that shouldn't casually delete children).
    - D-04: `TIMESTAMPTZ` (not TIMESTAMP) for every datetime column. Use `NOT NULL DEFAULT NOW()` for created_at/updated_at.
    - D-05: JSONB for financial_cutoff_config (organizations, projects), receipt_ocr_data (expenses)
    - D-06: CHECK constraints for: role (employee/manager/finance/customer), status fields, category (expenses: mileage/meal/accommodation/parking/travel_tickets/tolls/taxi/equipment/other), governance_model (creator_controlled/unanimous/majority), project_type (billable/internal), period (daily/weekly/monthly/yearly), action (submit/approve/reject/edit_approve/edit_return/partial_approve/delegate)
    - D-07: Indexes matching SurrealDB INDEX definitions — add `CREATE INDEX IF NOT EXISTS idx_{table}_{column} ON {table}({column})` for every field that had an INDEX in schema/001_schema.surql. Also add composite indexes where SurrealDB had them (e.g., user_id + entry_date on time_entries, user_id + is_primary on unit_memberships).

    24 tables in dependency order (parents before children):
    1. organizations
    2. users
    3. customers
    4. units (self-referencing FK: parent_unit_id → units(id) ON DELETE RESTRICT)
    5. organization_memberships (FKs to organizations, users)
    6. unit_memberships (FKs to organizations, users, units)
    7. contracts (FK to customers ON DELETE RESTRICT, organizations)
    8. contract_adoptions (FKs to contracts, organizations; UNIQUE(contract_id, organization_id))
    9. projects (FK to organizations, contracts ON DELETE RESTRICT, customers; CHECK for project_type, type, governance_model)
    10. project_adoptions (FKs to projects, organizations; UNIQUE(project_id, organization_id))
    11. project_managers (FKs to projects ON DELETE CASCADE, users ON DELETE CASCADE; UNIQUE(project_id, user_id))
    12. subprojects (FK to projects ON DELETE CASCADE)
    13. working_groups (FK to organizations, subprojects ON DELETE CASCADE, users(manager_id); unit_ids UUID[] DEFAULT '{}', delegate_ids UUID[] DEFAULT '{}')
    14. wg_members (FKs to working_groups ON DELETE CASCADE, users ON DELETE CASCADE, units ON DELETE RESTRICT; UNIQUE(wg_id, user_id))
    15. time_entries (FKs to organizations, users, projects, subprojects, working_groups, units; CHECK(hours > 0), CHECK status in draft/submitted/approved; self-ref FK created_from_entry_id → time_entries(id))
    16. time_entry_approvals (FK to time_entries ON DELETE CASCADE, users; CHECK action in submit/approve/reject/edit_approve/edit_return/partial_approve/delegate)
    17. expenses (FKs to organizations, users, projects, units; CHECK(category in mileage/meal/accommodation/parking/travel_tickets/tolls/taxi/equipment/other), CHECK(amount > 0), CHECK status in draft/submitted/approved/rejected)
    18. expense_approvals (FK to expenses ON DELETE CASCADE, users; same action CHECK as time_entry_approvals)
    19. invitations (FK to organizations ON DELETE CASCADE, users; CHECK status in pending/accepted/expired; UNIQUE code, UNIQUE invite_token)
    20. password_resets (FK to users ON DELETE CASCADE)
    21. refresh_tokens (FKs to users ON DELETE CASCADE, organizations ON DELETE CASCADE; UNIQUE token_hash)
    22. financial_cutoff_periods (FKs to organizations, projects)
    23. budget_caps (FKs to organizations, users, projects; CHECK category in mileage/meal/accommodation/other, CHECK(limit_amount > 0), CHECK period in daily/weekly/monthly/yearly)
    24. backup_approvers (FKs to organizations ON DELETE CASCADE, users ON DELETE CASCADE; CHECK role in employee/manager/finance/customer)

    After all CREATE TABLE statements, add all CREATE INDEX IF NOT EXISTS statements. Use idx_{table}_{column} naming.

    Create migrations/002_full_schema.down.sql:
    - DROP INDEX IF EXISTS for all indexes first
    - DROP TABLE IF EXISTS for all 24 tables in **reverse** dependency order (backup_approvers → budget_caps → ... → organizations)
    - DROP EXTENSION IF EXISTS pgcrypto at the end
  </action>

  <verify>
    <automated>grep -c "CREATE TABLE IF NOT EXISTS" migrations/002_full_schema.up.sql</automated>
  </verify>

  <acceptance_criteria>
    - migrations/002_full_schema.up.sql contains exactly 24 CREATE TABLE IF NOT EXISTS statements
    - Every table has `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
    - Every datetime column uses TIMESTAMPTZ (verify with `grep -i "timestamp" migrations/002_full_schema.up.sql` — no TIMESTAMP without TIME ZONE)
    - JSONB columns exist for financial_cutoff_config (organizations and projects) and receipt_ocr_data (expenses)
    - CHECK constraints exist for: role, status, category, governance_model, project_type, period, action
    - Foreign keys reference correct parent tables
    - Indexes exist for all key query columns
    - migrations/002_full_schema.down.sql contains DROP TABLE IF EXISTS for all 24 tables in reverse order
    - Down migration drops DROP INDEX IF EXISTS before tables
  </acceptance_criteria>
</task>

<task type="auto">
  <name>Task 2: Create 003_seed.up.sql and 003_seed.down.sql</name>
  <files>
    migrations/003_seed.up.sql
    migrations/003_seed.down.sql
  </files>

  <read_first>
    - schema/003_seed_demo.surql — source of truth for seed data values (users, units, memberships, contracts, projects, subprojects, WGs, time entries, expenses)
    - .planning/phases/pg-1-foundation/pg-1-RESEARCH.md — seed UUID strategy section (lines 670-760), insert order rules, password hash value
    .planning/phases/pg-1-foundation/pg-1-PATTERNS.md — seed pattern assignments (lines 111-176)
  </read_first>

  <action>
    Create migrations/003_seed.up.sql that inserts the same MVP demo data as schema/003_seed_demo.surql, translated to PostgreSQL INSERT statements. All INSERTs use `ON CONFLICT (id) DO NOTHING` for idempotency (D-14). Include a header comment block documenting the full UUID mapping table.

    **Deterministic UUID assignment for entities that currently use string IDs:**

    For seed entities that had SurrealDB string IDs (units, customer, projects, subprojects, working_groups, wg_members, unit_memberships, time_entries, expenses), generate deterministic UUIDs using Python's `uuid` module:
    ```python
    python3 -c "import uuid; print(uuid.uuid5(uuid.UUID('6ba7b810-9dad-11d1-80b4-00c04fd430c8'), 'units:engineering'))"
    ```
    Use the DNS namespace UUID `6ba7b810-9dad-11d1-80b4-00c04fd430c8` with name strings like `"units:engineering"`, `"customers:novatech"`, `"projects:proj_platform_eng"`, `"subprojects:subproj_platform_eng"`, `"working_groups:wg_platform_eng"`, `"wgm_001"`, `"um_001"`, `"te_001"`, `"exp_001"`. This produces deterministic, reproducible UUIDs that anyone can verify.

    **Entities that already have UUIDs** (keep as-is from SurrealDB seed):
    - Organization: `019df8b0-0001-7000-8000-000000000001`
    - Users (6): `019df6f5-ea95-735d-888b-158583ae4516` (Alex Rivera), `019df6f6-8cdb-70b9-9d0b-ed032caf9f4b` (Sarah Chen), `019df6f6-c6ca-7581-83f1-f50ab6c436cf` (Mike O'Brien), `019df6f7-3d87-734d-8a97-e93c89641c79` (Emma Wilson), `019df6f7-8ebf-779a-866d-46e196d4928d` (James Park), `019df6f7-c8c8-75c5-a6a7-224bbcd9cff0` (Lisa Torres)
    - Contracts (3): `019df8b1-0001-7000-8000-000000000001` (Digital Transformation), `019df8b1-0002-7000-8000-000000000002` (Cloud Infrastructure), `019df8b1-0003-7000-8000-000000000003` (Internal Operations)
    - Organization memberships (6): `019df8b2-0001-7000-8000-000000000001` through `000006`
    - Unit memberships used UUID-style: `019df8b2-0011-7000-8000-000000000001` through `000006` (generate these via the uuid5 method with names "um_001".."um_006")

    **Insert order** (must respect FK dependencies):
    1. organization (one)
    2. users (6)
    3. customer (NovaTech)
    4. units (8 — engineering, consulting, operations as level-1; platform, cloud under engineering; data under consulting; finance, hr under operations)
    5. organization_memberships (6)
    6. unit_memberships (6)
    7. contracts (3)
    8. contract_adoptions (none in current seed — skip)
    9. projects (6 — 2 per contract)
    10. project_adoptions (none in current seed — skip)
    11. project_managers (none in current seed — skip)
    12. subprojects (6 — one per project)
    13. working_groups (6 — one per subproject)
    14. wg_members (6)
    15. time_entries (12 — 4 per employee)
    16. time_entry_approvals (none in current seed — skip)
    17. expenses (6 — 2 per employee)
    18. expense_approvals (none in current seed — skip)

    **Password hash for all 6 users** (same as current seed, bcrypt cost 12 for "demo123"):
    `$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6`

    **Field mapping from SurrealDB to PostgreSQL:**
    - `time::now()` → `NOW()`
    - `<datetime>"2026-05-18T09:00:00Z"` → `'2026-05-18 09:00:00+00'::TIMESTAMPTZ` (for explicit date values like entry_date, expense_date)
    - SurrealDB record links `organizations:u'...'` → UUID literal
    - SurrealDB string record links `units:engineering` → the generated UUID for that unit
    - Arrays `[units:engineering, units:platform]` → PostgreSQL ARRAY['uuid1', 'uuid2']::UUID[]
    - `NONE` → omit column or use NULL

    **Contract 3 (Internal Operations) has no customer_id** — omit that column, let it be NULL.

    Create migrations/003_seed.down.sql:
    - DELETE FROM each table WHERE id IN (list of all seed UUIDs for that table)
    - Order: reverse of insert order
    - Only delete the seed UUIDs (no TRUNCATE — would wipe production data per D-15)
  </action>

  <verify>
    <automated>grep -c "ON CONFLICT" migrations/003_seed.up.sql</automated>
  </verify>

  <acceptance_criteria>
    - migrations/003_seed.up.sql exists with header comment documenting the UUID mapping table
    - Every INSERT uses ON CONFLICT (id) DO NOTHING
    - Organization uses UUID `019df8b0-0001-7000-8000-000000000001`
    - All 6 users inserted with exact UUIDs from SurrealDB seed
    - All 6 users have the bcrypt hash `$2a$12$6a3VkBiIBAV3sQtOqxYEVulOVqYy4USw/CZ8wonOE6odHjyPo4ep6` for password "demo123"
    - 8 units inserted with correct parent hierarchy (engineering→platform+cloud, consulting→data, operations→finance+hr)
    - 6 organization_memberships with correct user-org-role mapping
    - 6 unit_memberships with correct user-unit mapping
    - 3 contracts with correct UUIDs and properties
    - 6 projects with correct contract and customer references
    - 6 subprojects with correct project references
    - 6 working_groups with correct manager_id and unit_ids arrays
    - 6 wg_members with correct user-wg-unit mapping
    - 12 time_entries with correct user-project-WG references, May 2026 dates, "submitted" status
    - 6 expenses with correct user and category values
    - Insert order respects FK dependency ordering
    - migrations/003_seed.down.sql deletes by seed UUIDs in reverse FK order
    - Down migration uses DELETE WHERE id IN (...) — no TRUNCATE
  </acceptance_criteria>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| migration file → database | Static SQL files are applied to the database; no user input enters this pipeline |

## STRIDE Threat Register

| Threat ID | Category | Component | Disposition | Mitigation Plan |
|-----------|----------|-----------|-------------|-----------------|
| T-pg1-01 | Tampering | Migration SQL files | mitigate | Files are static SQL committed to repo, not user-controllable input — code review before merge |
| T-pg1-02 | Spoofing | UUID generation | mitigate | `gen_random_uuid()` uses cryptographically secure RNG built into PostgreSQL pg_catalog (available in PG 15 without pgcrypto) |
</threat_model>

<verification>
- `migrations/002_full_schema.up.sql` has 24 CREATE TABLE IF NOT EXISTS statements
- `migrations/002_full_schema.down.sql` has 24 DROP TABLE IF EXISTS statements in reverse order
- `migrations/003_seed.up.sql` has the correct entity counts (1 org, 6 users, 8 units, 6 org memberships, 6 unit memberships, 3 contracts, 6 projects, 6 subprojects, 6 WGs, 6 WG members, 12 time entries, 6 expenses)
- All INSERTs in seed use ON CONFLICT DO NOTHING
</verification>

<success_criteria>
- All 4 migration files created with correct SQL
- Schema covers all 24 entities with proper types, FKs, CHECK constraints, indexes
- Seed data is idempotent (re-runnable with ON CONFLICT DO NOTHING)
- Seed data handles all FK relationships correctly
- All entities use UUID PKs with gen_random_uuid() default
</success_criteria>

<output>
After completion, create `.planning/phases/pg-1-foundation/pg-1-01-SUMMARY.md`
</output>
