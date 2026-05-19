# Phase 5: MVP Consolidation - Research

**Researched:** 2026-05-19
**Domain:** SurrealDB seed data, entity dependency graph, bcrypt hashing
**Confidence:** HIGH

## Summary

This phase creates a new `003_seed_demo.surql` seed file and deprecates the old `002_seed_tcg.surql`. The new seed populates all core entities (org, users, units, memberships, projects, contracts, customer, subprojects, working groups, time entries, expenses) with consistent UUIDs and bcrypt-hashed passwords so the app is immediately demonstrable.

**Critical findings:**
1. **Schema-to-Go field mismatch exists** for `projects` and `customers` tables — the Go code writes fields NOT defined in the SCHEMAFULL schema. Seed must include BOTH schema-defined and Go-expected fields.
2. **Multiple tables are schemaless** — `contracts`, `contract_adoptions`, `project_adoptions`, `project_managers` have no `DEFINE TABLE` in `001_schema.surql` but are fully functional.
3. **Bootstrap is not blocked by seed** — D-12 explicitly states bootstrap and seed are separate. With seed data present, `GET /auth/bootstrap-check` returns `needs_bootstrap: false`, which is correct.
4. **The old seed has no `password_hash`** — users can't log in. The new seed MUST include bcrypt hashes.
5. **bcrypt cost factor is 12** (`internal/auth/auth.go:43`), consistent across the codebase.

**Primary recommendation:** Single monolithic `003_seed_demo.surql` with idempotent `IF NOT EXISTS` guards, consistent UUIDs throughout, and pre-computed bcrypt hashes for the fixed password `demo123`.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Seed org, users, units, memberships, projects, contracts, and one customer. Time entries and expenses included as sample data.
- **D-02:** Medium setup — 3 contracts with 6 projects total.
- **D-03:** One demo customer.
- **D-04:** Full role spectrum — 6 users: 2 managers, 1 finance, 3 employees.
- **D-05:** Fresh seed file — `003_seed_demo.surql` with clean, consistent UUIDs.
- **D-06:** Keep TCG theme (Tech Consulting Group).
- **D-07:** SurQL file first, Go CLI seed command deferred to future phase.
- **D-08:** Single monolithic seed file, not split by domain.
- **D-09:** All UUIDs throughout — no mixed short-string IDs.
- **D-10:** Idempotent — use `IF NOT EXISTS` / `OR REPLACE` patterns.
- **D-11:** Old seed renamed to `002_seed_tcg.deprecated.surql` so `cmd/schema` skips it.
- **D-12:** Bootstrap ≠ seed — they do not conflict.
- **D-13:** Pre-hashed bcrypt passwords in seed data (not plaintext).
- **D-14:** Manager as primary demo persona.
- **D-15:** 6 users total — 2 managers, 1 finance, 3 employees.
- **D-16:** Sample time entries — 3-5 per employee from past week.
- **D-17:** Sample expenses — 1-2 per employee.
- **D-18:** Seed-focused — no deep structural changes.
- **D-19:** Minimal deprecation — rename old seed file only.
- **D-20:** Manual verification pass — run seed, log in, check pages.

### the agent's Discretion
- Exact bcrypt hashes for demo passwords
- Seed data structure and file formatting
- Time entry and expense sample value specifics
- Verification checklist format

### Deferred Ideas (OUT OF SCOPE)
- Go CLI seed command (`cmd/seed`) — end of milestone
- Deep consolidation / API fixes for broken endpoints
- Major codebase restructuring
</user_constraints>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Seed data population | Database | — | Seed runs directly against SurrealDB via SurQL, no API layer |
| Idempotency enforcement | Database | — | Handled by SurrealDB `IF NOT EXISTS` / `OR REPLACE` patterns |
| Password verification | Backend (Go auth) | — | bcrypt comparison in `internal/auth/auth.go`, seed provides pre-hashed values |
| Bootstrap detection | Backend (Go auth) | Frontend | `AnyExists()` check in auth service, frontend shows/hides bootstrap UI |
| Entity dependency ordering | Database (seed file) | — | SurQL file execution order within the seed file itself |
| Field name normalization | Backend (Go adapters) | — | Go `surrealProjectCompat` / `surrealContractCompat` structs normalize DB field names |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| SurrealDB SurQL | 2.x | Database schema + seed language | DB-native, no ORM needed for seed scripts |
| golang.org/x/crypto/bcrypt | v0.48.0 | Password hashing | Standard for Go bcrypt, used by `internal/auth/auth.go` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go `cmd/schema` | — | Applies `*.surql` files | Local dev DB bootstrap (runs seeds) |
| `github.com/google/uuid` | — | UUID generation (v7) | All seed entity IDs must be valid UUIDs |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| SurQL seed file | Go CLI seed command | Deferred to future phase; SurQL is immediately usable without fixing broken APIs |
| OR REPLACE | DELETE + INSERT | OR REPLACE is atomic and idempotent; DELETE+INSERT risks partial state on error |
| Monolithic seed | Split-by-domain seeds | Monolithic is simpler for a single seed file; splitting adds complexity with no benefit at this stage |

**Installation:**
```bash
# Nothing to install — bcrypt is already in go.mod (golang.org/x/crypto v0.48.0)
# Seed file is applied via existing cmd/schema/main.go
```

**Version verification:**
```bash
# SurrealDB version (from docker-compose or CLI)
surrealdb version  # Already installed

# bcrypt version in go.mod
grep 'x/crypto' go.mod
# Output: golang.org/x/crypto v0.48.0
```

## Architecture Patterns

### Entity Dependency Graph

```
Organizations (no deps)
  ├── Users (no deps)
  ├── Customers (→ Organizations)
  ├── Units (→ Organizations)
  ├── Organization Memberships (→ Users, Organizations)
  ├── Unit Memberships (→ Organizations, Users, Units)
  ├── Contracts (→ Organizations, Customers) [SCHEMALESS]
  │   ├── Contract Adoptions (→ Contracts, Organizations) [SCHEMALESS]
  │   └── Projects (→ Organizations, Contracts, Customers)
  │       ├── Project Adoptions (→ Projects, Organizations) [SCHEMALESS]
  │       ├── Project Managers (→ Projects, Users) [SCHEMALESS]
  │       └── Subprojects (→ Projects)
  │           └── Working Groups (→ Organizations, Subprojects, Users)
  │               └── WG Members (→ Working Groups, Users, Units)
  ├── Time Entries (→ Organizations, Users, Projects, Subprojects, Working Groups, Units)
  └── Expenses (→ Organizations, Users, Projects, Units)
```

**Creation order in the seed file must follow this graph bottom-up.**

### Recommended Seed Structure

```
-- ============================================================================
-- IDEMPOTENT SEED: MVP Demo Data
-- ============================================================================
-- Organization (root entity)
-- Users (independent)
-- Units (→ org)
-- Customers (→ org)
-- Organization Memberships (→ users, org)
-- Unit Memberships (→ org, users, units)
-- Contracts (→ org, customer) [schemaless]
-- Contract Adoptions (→ contract, org) [schemaless]
-- Projects (→ org, contract, customer)
-- Project Adoptions (→ project, org) [schemaless]
-- Project Managers (→ project, user) [schemaless]
-- Subprojects (→ project)
-- Working Groups (→ org, subproject, user)
-- WG Members (→ wg, user, unit)
-- Time Entries (→ org, user, project, subproject, wg, unit)
-- Expenses (→ org, user, project, unit)
```

### SurrealDB UUID Record Link Syntax

```surql
-- UUID-type record IDs use u'...' syntax
CREATE users:u'019df6f5-ea95-735d-888b-158583ae4516' SET ...;

-- Record links to UUID-type records
    user_id = users:u'019df6f5-ea95-735d-888b-158583ae4516',

-- String-type record IDs (units, customers, projects, etc. use string IDs)
-- These use string literals (NO u'...' prefix needed)
CREATE units:engineering SET ...;

-- Record links to string-type records
    unit_id = units:engineering,
```

**Key distinction:**
- Tables with `TYPE uuid` ID in schema → `u'uuid-string'` for IDs and record links
- Tables with `TYPE string` ID in schema → plain string IDs
- The `organizations` and `users` tables use `TYPE uuid` for `id`
- Tables like `units`, `customers`, `projects`, `subprojects`, `time_entries`, `expenses` etc. use `TYPE string`

### Idempotency Pattern

```surql
-- IF NOT EXISTS checks if record exists before creating
-- But SurrealDB doesn't have IF NOT EXISTS for CREATE.
-- Use this approach instead:

-- Option A: ON DUPLICATE KEY (SurrealDB native)
-- SurrealDB CREATE returns existing record if ID already exists
-- CREATE table:id SET ... is naturally idempotent for the exact same ID
-- Re-running the same seed with same IDs = no duplicate records

-- Option B: DELETE + CREATE (not truly idempotent)
DELETE users:u'...';
CREATE users:u'...' SET ...;

-- Option C: Conditional with IF
-- Best approach: use a consistent set of UUIDs.
-- If a record with the same ID already exists, SurrealDB's CREATE
-- will return it without creating a duplicate (the ID is the primary key).
-- For truly idempotent re-runs with updates, use UPSERT:
UPDATE table:id SET field = value;

-- Recommended for this seed: Use fixed UUIDs throughout.
-- CREATE with an existing ID is a no-op (SurrealDB returns existing record).
-- To update data on re-run: use UPDATE instead of CREATE.
```

**D-10 calls for idempotent patterns.** The safest approach: use `CREATE table:id SET ...` with fixed UUIDs. Re-running the same SurQL file with the same IDs will succeed because SurrealDB prevents duplicate primary keys (the `CREATE` returns the existing record instead of erroring). For full idempotency with updates, use `UPDATE table:id SET ...` which acts as an upsert.

### Anti-Patterns to Avoid
- **Mixing ID formats:** D-09 requires all UUIDs. Do NOT mix short-string IDs (like `users:u108`) — the old seed had one such orphan record (`u108`) that referenced a non-existent user.
- **Omitting `password_hash`:** The old seed omitted it, making all users un-login-able. Every seeded user MUST have a bcrypt-hashed `password_hash` field.
- **Using the bootstrap org UUID:** The new seed creates its OWN organization. Do NOT reference the bootstrap-created org UUID (`8d152bac-...` from the old seed).
- **Hardcoding UUIDs in tests/code:** D-19 says minimal deprecation — don't clean up hardcoded UUIDs in tests; just don't introduce new hardcoded ones.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Password hashing | Custom hash function | `bcrypt.GenerateFromPassword()` with cost 12 | bcrypt is already a dependency (`golang.org/x/crypto`), cost factor is established at 12, and the auth service expects bcrypt format |
| UUID generation | Manual UUIDs | `github.com/google/uuid` | Already imported, produces standard UUID v4/v7 strings that SurrealDB accepts |

**Key insight:** The entire seed infrastructure (cmd/schema loading, SurQL parsing, bcrypt verification) already exists — this phase does not need to add any new Go code dependencies.

## Common Pitfalls

### Pitfall 1: Schema vs. Go Field Name Mismatch
**What goes wrong:** The seed data uses schema field names, but the Go code reads different field names, causing "null" or empty data in the frontend.

**Why it happens:** The `projects` table in `001_schema.surql` defines `project_type` but the Go code reads `type` from its `surrealProjectCompat` struct. The schema also lacks `contract_id`, `governance_model`, `created_by_org_id`, `is_shared` — all fields the Go code writes and reads.

**Tables affected:**
| Table | Schema field | Go field | Severity |
|-------|-------------|----------|----------|
| `projects` | `project_type` | `type` | HIGH — Go reads `type`, schema requires `project_type` |
| `projects` | *undefined* | `contract_id` | HIGH — Go reads for JOINs |
| `projects` | *undefined* | `governance_model` | HIGH — Go reads for display |
| `projects` | *undefined* | `created_by_org_id` | HIGH — Go reads for permission checks |
| `projects` | *undefined* | `is_shared` | MEDIUM — used for org scoping |
| `customers` | *undefined* | `contact_name` | MEDIUM — Go reads for display |
| `customers` | *undefined* | `phone` | LOW — optional field |
| `organizations` | `financial_cutoff_days` (option) | `financial_cutoff_days` | LOW — optional |

**Solution:** Write BOTH sets of fields in the seed:
```surql
CREATE projects:proj_aerospace SET
    name = "Aerospace Platform",
    type = "billable",         -- Go reads this
    project_type = "billable", -- Schema expects this
    contract_id = contracts:u'...',
    governance_model = "creator_controlled",
    created_by_org_id = organizations:u'...',
    org_id = organizations:u'...',     -- Go also reads this
    ...
;
```

### Pitfall 2: Time Entry Has Many Required Record Links
**What goes wrong:** Time entries require `org_id`, `user_id`, `project_id`, `subproject_id`, `wg_id`, AND `unit_id` — all non-nullable `record<>` fields. Missing any one causes the CREATE to fail.

**Why it happens:** The `time_entries` table has 6 record-type fields with `ASSERT $value != NONE`. Seed data must ensure all these entities exist before creating time entries.

**Solution:** Ensure the dependency graph is respected. Create all supporting entities (projects, subprojects, WGs, units) BEFORE time entries. Use a consistent UUID to reference.

### Pitfall 3: Schema Loading with Mixed UUID/String IDs
**What goes wrong:** The `organization_memberships` table uses `TYPE uuid` for IDs, but the `unit_memberships` table uses `TYPE string`. Using the wrong ID format fails.

**How to avoid:** Check the schema field type for each table's `id` field before writing the CREATE statement. Reference table below.

### Pitfall 4: cm" />**How to avoid:** The old seed has a ghost reference to non-existent user `users:u108` in `organization_memberships`. Ensure every record link points to an entity that's actually created in the seed (or skip non-existent references).

## Domain Models and Field Maps

### Organization (`organizations`)
- **Schema-defined fields:** `id` (uuid), `name`, `slug`, `description` (option), `financial_cutoff_days` (option), `financial_cutoff_config` (option), `created_at`, `updated_at`
- **Notes:** Bootstrap creates org with `slug` = lowercased name with hyphens. Seed should use same convention.

### Users (`users`)
- **Schema-defined fields:** `id` (uuid), `email`, `name`, `password_hash` (option), `is_active` (bool), `created_at`, `updated_at`
- **Additional schema fields:** `username` (option), `firstname` (option), `lastname` (option) — defined in final section of schema
- **Go model fields (SurrealUser):** `id`, `email`, `username`, `firstname`, `lastname`, `name`, `password_hash`, `is_active`, `created_at`, `updated_at`
- **Password:** Use `bcrypt.GenerateFromPassword([]byte("demo123"), 12)`. Store result in `password_hash`.

### Organization Memberships (`organization_memberships`)
- **Schema-defined fields:** `id` (uuid), `user_id` (record\<users\> option), `organization_id` (record\<organizations\>), `role` (string), `is_active` (bool), `invited_by` (option<record\<users\>>), `invited_at` (option), `activated_at` (option), `created_at`, `updated_at`
- **Allowed roles:** `'employee'`, `'manager'`, `'finance'`, `'customer'`
- **Unique index on:** `(user_id, organization_id)`
- **Note:** `user_id` is OPTIONAL (`TYPE option<record<users>>`), but for seed data it should be set.

### Units (`units`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `name`, `description` (option), `parent_unit_id` (option<record\<units\>>), `hierarchy_level` (number), `code` (option), `created_at`, `updated_at`
- **Unique index on:** `(org_id, code)`
- **Notes:** Unit IDs are STRING type (not UUID). Old seed uses short codes like `jde`, `netsuite`.

### Unit Memberships (`unit_memberships`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `user_id` (record\<users\>), `unit_id` (record\<units\>), `is_primary` (bool), `role` (string), `start_date` (datetime), `end_date` (option), `created_at`
- **Notes:** ID is STRING type. Old seed uses `um001`, `um002`, etc.

### Customers (`customers`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `name`, `email` (option), `address` (option), `vat_number` (option), `is_active` (bool), `created_at`, `updated_at`
- **Go code ALSO writes:** `contact_name`, `phone` (these are NOT in schema, may cause SCHEMAFULL rejection)
- **Risk:** The Go model `surrealCustomer` has `ContactName` and `Phone`. If the schema rejects these, the seed must omit them; the Go code will then get empty strings for those fields.

### Contracts (`contracts`) — SCHEMALESS
- **No DEFINE TABLE in schema** — fully schemaless
- **Go code fields:** `name`, `km_rate` (float), `currency`, `customer_id` (record\<customers\>, optional), `governance_model` (string: 'creator_controlled'|'unanimous'|'majority'), `created_by_org_id` (record\<organizations\>), `is_shared` (bool), `is_active` (bool), `created_at`, `updated_at`
- **Contract ID format:** UUID (the Go adapter uses `uuidToRecordID("contracts", id)`, so the SurQL CREATE should use UUID string, e.g., `contracts:u'uuid-here'`)

### Projects (`projects`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `name`, `description` (option), `project_type` ('billable'|'internal'), `customer_id` (option<record\<customers\>>), `budget_amount` (option), `financial_cutoff_config` (option), `is_active` (bool)
- **Go code ALSO writes:** `type` (Go reads `json:"type"`), `contract_id` (record\<contracts\>), `governance_model`, `created_by_org_id` (record\<organizations\>), `is_shared` (bool)
- **CRITICAL:** The Go code writes AND reads `type` (not `project_type`), `contract_id`, `governance_model`, `created_by_org_id`, and `is_shared`. The SCHEMAFULL definition does NOT define these fields. Seed must write BOTH the schema fields AND the Go fields. If SCHEMAFULL rejects Go-only fields, this is a pre-existing bug that will be caught during manual verification.

### Subprojects (`subprojects`)
- **Schema-defined fields:** `id` (string), `project_id` (record\<projects\>), `name`, `description` (option), `sequence_order` (number), `is_active` (bool), `created_at`, `updated_at`
- **Index on:** `project_id`

### Working Groups (`working_groups`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `subproject_id` (record\<subprojects\>), `name`, `description` (option), `unit_ids` (array<record\<units\>>), `enforce_unit_tuple` (bool), `manager_id` (record\<users\>), `delegate_ids` (array<record\<users\>>), `is_active` (bool), `created_at`, `updated_at`

### WG Members (`wg_members`)
- **Schema-defined fields:** `id` (string), `wg_id` (record\<working_groups\>), `user_id` (record\<users\>), `unit_id` (record\<units\>), `role` (string), `is_default_subproject` (bool), `start_date` (datetime), `end_date` (option), `created_at`

### Time Entries (`time_entries`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `user_id` (record\<users\>), `project_id` (record\<projects\>), `subproject_id` (record\<subprojects\>), `wg_id` (record\<working_groups\>), `unit_id` (record\<units\>), `hours` (number > 0), `description`, `entry_date` (datetime), `status` ('draft'|'submitted'|'approved'), `is_deleted` (bool), `created_from_entry_id` (option), `created_at`, `updated_at`

### Expenses (`expenses`)
- **Schema-defined fields:** `id` (string), `org_id` (record\<organizations\>), `user_id` (record\<users\>), `project_id` (option<record\<projects\>>), `unit_id` (record\<units\>), `category` ('mileage'|'meal'|'accommodation'|'other'), `amount` (number > 0), `currency` (string), `description` (option), `expense_date` (datetime), `receipt_url` (option), `receipt_ocr_data` (option), `status` ('draft'|'submitted'|'approved'|'rejected'), `is_deleted` (bool), `created_at`, `updated_at`

### Audit Logs (`audit_logs`) — Not seeded
- Immutable, append-only. No seed data needed. Time entry/expense submission through the app creates them.

### Tables NOT in Schema (Schemaless)
- **`contract_adoptions`**: `id` (uuid? — Go uses UUID), `contract_id` (record\<contracts\>), `organization_id` (record\<organizations\>), `adopted_at` (datetime)
- **`project_adoptions`**: `id` (uuid), `project_id` (record\<projects\>), `organization_id` (record\<organizations\>), `adopted_at` (datetime)
- **`project_managers`**: `id` (uuid), `project_id` (record\<projects\>), `user_id` (record\<users\>), `created_at` (datetime)

## bcrypt Hash Generation

**Cost factor:** 12 (hardcoded in `internal/auth/auth.go:43`)
**Password:** `demo123` for all demo users
**Generate with:**

```bash
# Using Go's bcrypt (already in go.mod)
cat > /tmp/gen_hash.go << 'EOF'
package main

import (
    "fmt"
    "os"
    "golang.org/x/crypto/bcrypt"
)

func main() {
    password := "demo123"
    if len(os.Args) > 1 {
        password = os.Args[1]
    }
    hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    fmt.Println(string(hash))
}
EOF
cd /path/to/hourglass && go run /tmp/gen_hash.go demo123
```

**Example output:** `$2a$12$y4bXx63xuz1NGgGC45CmouusDUupIDD/Gt8bPK1pJgbMe7XUEPKm6`

**Important:** bcrypt generates a different hash each time (random salt). Pick ONE hash and hardcode it in the seed file. Every user gets the same password `demo123` for simplicity. D-13 notes "Demo login documented for easy access" — document `demo123` as the password for all seeded users.

## Schema Loading Mechanism

`cmd/schema/main.go`:
1. Connects to SurrealDB at `SURREALDB_URL` (default: `ws://localhost:8000/rpc`)
2. Signs in as `SURREALDB_USER`/`SURREALDB_PASS` (default: root/root)
3. Uses namespace `SURREALDB_NS` (default: `hourglass`) and database `SURREALDB_DB` (default: `main`)
4. Globs `schema/*.surql` and sorts alphabetically
5. Splits each file by `;` and executes each query
6. **Critical:** File ordering matters — `001_schema.surql` runs first, then `002_*.surql`, then `003_*.surql`
7. When `002_seed_tcg.surql` is renamed to `002_seed_tcg.deprecated.surql`, the glob `*.surql` will no longer match it — it's skipped automatically

**Execution:**
```bash
go run ./cmd/schema
```

## Bootstrap and Seed Non-Conflict

| Aspect | Bootstrap (`POST /auth/bootstrap`) | Seed (`003_seed_demo.surql`) |
|--------|------------------------------------|------------------------------|
| When | First-run, no users exist | After schema, independently |
| Creates | Admin org + admin user with role 'employee' | Demo org + 6 users with varied roles |
| User check | `AnyExists()` — blocked if ANY user exists | Creates users directly via SurQL |
| Frontend | Shows bootstrap form if `needs_bootstrap: true` | If users exist, bootstrap form hides |
| Conflict | None — bootstrap won't run if seed has created users | Must NOT rely on bootstrap org UUID |

**What this means for seed design:**
- The seed creates its own organization with a unique UUID
- The bootstrap org (`8d152bac-...` from old seed) is NOT referenced by the new seed
- After seeding, `GET /auth/bootstrap-check` returns `{"needs_bootstrap": false}`
- Users log in directly with demo credentials (email + `demo123`)

## Recommended Seed Entity Plan

### Organization
- `name`: "Tech Consulting Group"
- `slug`: "tech-consulting-group"
- Uses fresh UUID (e.g., `019df6f5-0001-...` — v7 UUID for consistency)

### Units (3 departments, matching 3 contracts)
1. `engineering` — Engineering Unit (code: "ENG")
2. `consulting` — Consulting Unit (code: "CONS")
3. `operations` — Operations Unit (code: "OPS")

### Users (6 total, per D-15)
| # | Name | Role | Unit | Email |
|---|------|------|------|-------|
| 1 | Alex Rivera (Manager) | manager | engineering | alex.rivera@tcg.com |
| 2 | Sarah Chen (Manager) | manager | consulting | sarah.chen@tcg.com |
| 3 | Mike O'Brien | finance | operations | mike.obrien@tcg.com |
| 4 | Emma Wilson | employee | engineering | emma.wilson@tcg.com |
| 5 | James Park | employee | consulting | james.park@tcg.com |
| 6 | Lisa Torres | employee | engineering | lisa.torres@tcg.com |

### Customer (1, per D-03)
- `name`: "NovaTech Industries"
- Contact info + address

### Contracts (3, per D-02)
1. "Digital Transformation Program" — km_rate: 0.42, billable, linked to NovaTech
2. "Cloud Infrastructure Migration" — km_rate: 0.35, billable, linked to NovaTech
3. "Internal Operations" — km_rate: 0.00, internal, no customer

### Projects (6 total, 2 per contract)
- Contract 1 → 2 projects (Platform Engineering, Data Analytics)
- Contract 2 → 2 projects (Cloud Migration, DevOps Setup)
- Contract 3 → 2 projects (HR System, Finance Tools)

### Subprojects (1 per project)

### Working Groups (1 per subproject)

### Time Entries (3-5 per employee from past week)

### Expenses (1-2 per employee)

## Code Examples

### Basic Seed Pattern (User)
```surql
-- Source: Existing 002_seed_tcg.surql pattern, verified against schema
CREATE IF NOT EXISTS users:u'019df6f5-ea95-735d-888b-158583ae4516' SET
    email = "alex.rivera@tcg.com",
    name = "Alex Rivera",
    firstname = "Alex",
    lastname = "Rivera",
    username = "arivera",
    password_hash = "$2a$12$y4bXx63xuz1NGgGC45CmouusDUupIDD/Gt8bPK1pJgbMe7XUEPKm6",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;
```

### Record Link Pattern (Organization Membership)
```surql
-- Source: Existing pattern, verified against organization_memberships schema
CREATE organization_memberships:u'org-membership-uuid' SET
    user_id = users:u'user-uuid',
    organization_id = organizations:u'org-uuid',
    role = "manager",
    is_active = true,
    created_at = time::now(),
    updated_at = time::now()
;
```

### Unit with String ID
```surql
-- Source: Existing pattern, units id is TYPE string
CREATE units:engineering SET
    id = "engineering",
    org_id = organizations:u'org-uuid',
    name = "Engineering Unit",
    description = "Software engineering and platform development",
    hierarchy_level = 1,
    code = "ENG",
    created_at = time::now(),
    updated_at = time::now()
;
```

### Time Entry (Complex Record Links)
```surql
-- Source: Verified against time_entries schema
CREATE time_entries:te_sample_001 SET
    org_id = organizations:u'org-uuid',
    user_id = users:u'user-uuid',
    project_id = projects:proj_platform,
    subproject_id = subprojects:subproj_platform,
    wg_id = working_groups:wg_platform,
    unit_id = units:engineering,
    hours = 7.5,
    description = "Frontend dashboard development",
    entry_date = <datetime>"2026-05-18T00:00:00Z",
    status = "submitted",
    is_deleted = false,
    created_at = time::now(),
    updated_at = time::now()
;
```

### Idempotent UPSERT Pattern
```surql
-- For entities that need to be idempotent with data updates:
-- Create only if not exists, then update to ensure latest data
-- Actually, in SurrealDB, CREATE with existing ID returns the existing record (no-op).
-- For true idempotency with overwrite, use:
UPDATE contracts:u'contract-uuid' SET
    name = "Digital Transformation Program",
    km_rate = 0.42,
    ...
;
-- UPDATE acts as an upsert — creates if not exists, updates if exists
```

## Verification Checklist

1. Run `go run ./cmd/schema` — should succeed silently
2. Verify file loading order: `001_schema.surql`, `003_seed_demo.surql` (002 is renamed to .deprecated)
3. Start server: `go run ./cmd/server`
4. Start frontend: `cd web && bun run dev`
5. Verify bootstrap check: `GET /auth/bootstrap-check` returns `{"needs_bootstrap": false}`
6. Login as demo manager: `POST /auth/login` with email `alex.rivera@tcg.com`, password `demo123`
7. Verify each page renders with seed data:
   - Dashboard `/` — welcome page
   - Org hierarchy `/org-hierarchy` — units tree
   - Contracts `/contracts` — 3 contracts listed
   - Projects `/projects` — 6 projects listed
   - Customers `/customers` — 1 customer listed
   - Time entries `/time-entries` — sample entries visible
   - Expenses — sample expenses visible
8. Login as other roles to verify role-based access:
   - Finance user: `mike.obrien@tcg.com` — should see expense approvals
   - Employee: `emma.wilson@tcg.com` — should see own time entries

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `schema/*.deprecated.surql` won't be matched by `*.surql` glob | Schema Loading | LOW — confirmed by `filepath.Glob("*.surql")` in `cmd/schema/main.go` |
| A2 | UUIDs from `github.com/google/uuid` are accepted by SurrealDB in `u'...'` format | UUID Format | LOW — verified in existing seed and Go code patterns |
| A3 | SCHEMAFULL `projects` table accepts extra fields (`type`, `contract_id`, etc.) written by Go code | Pitfall 1 | MEDIUM — if SCHEMAFULL rejects them, the Go code won't work either, indicating a pre-existing bug |
| A4 | `organization_memberships.user_id` being `option<record<users>>` can still accept a record link | Field Map | LOW — Go code writes it as record link and reads it successfully |
| A5 | bcrypt cost 12 produces hashes that start with `$2a$12$` | Password Hashing | LOW — verified by generating a hash with the Go code |
| A6 | The `contracts` table being schemaless means any fields are accepted | Schemaless Tables | LOW — confirmed by Go code writing to it without schema definition |
| A7 | `UPDATE` acts as upsert (creates if not exists, updates if exists) | Idempotency | MEDIUM — SurrealDB docs recommend `UPSERT` for this; `UPDATE` without `CREATE` may fail on non-existent records. Test during implementation. |

## Open Questions

1. **Does the `projects` table SCHEMAFULL definition reject Go-code-specific fields?**
   - What we know: The schema defines `project_type` but Go code writes `type`, `contract_id`, `governance_model`, `created_by_org_id`, `is_shared`
   - What's unclear: Whether current Go code actually works against the strict schema
   - Recommendation: Include BOTH sets of fields in the seed. Test during manual verification. If SCHEMAFULL rejects `contract_id` etc., this is a pre-existing bug.

2. **Does `UPDATE table:id SET ...` work as an upsert on a non-existent record?**
   - What we know: SurrealDB's `UPDATE` is documented as upsert-like when used with `UPDATE table:id`
   - What's unclear: Whether it truly creates if absent or just updates existing
   - Recommendation: Use `CREATE` for first-time seed. Use `CREATE` (not `UPDATE`) since the seed creates records that don't exist yet. If full idempotency is needed, wrap in a transaction or check with `SELECT` first.

3. **Does the `unit_memberships` table accept UUID-type IDs?**
   - What we know: Schema says `id TYPE string`
   - Recommendation: Use short string IDs (`um_001`, `um_002`) for unit memberships, or use UUID strings (`u'uuid'` for UUID format, or just the string `"uuid-string"`)

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go 1.26+ | schema loading | ✓ | go1.26.1 | — |
| SurrealDB | seed data target | Via docker-compose | — | Must be running |
| bcrypt | hash generation | ✓ (via `go run`) | golang.org/x/crypto v0.48.0 | — |

**Missing dependencies with no fallback:**
- None — all tools are already available in the project environment

**Missing dependencies with fallback:**
- None

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + testify |
| Config file | none — standard Go test conventions |
| Quick run command | `go test ./internal/adapters/secondary/surrealdb/... -run TestSeed -v -count=1` |
| Full suite command | `make test` |

### Phase Requirements → Test Map

This phase has no explicit requirements IDs — all requirements are embedded in CONTEXT.md as decisions (D-01 through D-20). Testing is manual verification (D-20).

| Decision | Behavior | Test Type | Automated Command | File Exists? |
|----------|----------|-----------|-------------------|-------------|
| D-10 | Idempotent seed re-run | Manual | Run `go run ./cmd/schema` twice | ❌ D-20 manual |
| D-13 | Demo login works | Manual | Login with demo credentials | ❌ D-20 manual |
| D-11 | Old seed skipped | Manual | Check server logs for applied files | ❌ D-20 manual |

### Sampling Rate
- **Per task commit:** N/A (manual verification only)
- **Per wave merge:** Run `go run ./cmd/schema`
- **Phase gate:** Full manual verification (D-20 checklist)

### Wave 0 Gaps
- None — existing test infrastructure is sufficient for schema loading
- Manual verification checklist is in the Verify steps of the plan, not in automated tests

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | bcrypt hashed passwords in seed, cost factor 12 |
| V3 Session Management | no | Session management is server-side, seed creates no sessions |
| V4 Access Control | no | Access control is role-based, seed assigns roles |
| V6 Cryptography | yes | bcrypt for password hashing (never hand-roll) |

### Known Threat Patterns

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Hardcoded credentials | Information Disclosure | Demo password `demo123` is intentionally weak for dev/demo; production would use stronger credentials |
| Weak bcrypt cost | Tampering | Cost factor 12 matches auth service default; adequate for dev |

## Sources

### Primary (HIGH confidence)
- `schema/001_schema.surql` — All table definitions, field types, indexes, permissions [VERIFIED: codebase read]
- `schema/002_seed_tcg.surql` — Existing seed as reference for SurQL syntax and patterns [VERIFIED: codebase read]
- `cmd/schema/main.go` — Schema/seed loader mechanism [VERIFIED: codebase read]
- `internal/auth/auth.go` — bcrypt hash cost (line 43), password verification [VERIFIED: codebase read]
- `internal/adapters/secondary/surrealdb/models.go` — Surreal types mapping (SurrealUser, SurrealTimeEntry, etc.) [VERIFIED: codebase read]
- `internal/adapters/secondary/surrealdb/helpers.go` — UUID to RecordID conversion (`uuidToRecordID`) [VERIFIED: codebase read]
- `internal/adapters/secondary/surrealdb/contract_repository.go` — Contract field names and schemaless table usage [VERIFIED: codebase read]
- `internal/adapters/secondary/surrealdb/customer_repository.go` — Customer field names (name, contact_name, etc.) [VERIFIED: codebase read]
- `internal/adapters/secondary/surrealdb/project_repository.go` — Project field names (type, contract_id, etc.) [VERIFIED: codebase read]
- `internal/models/models.go` — Role, Status, Governance constants [VERIFIED: codebase read]

### Secondary (MEDIUM confidence)
- `internal/core/services/auth/auth.go:Bootstrap` — Bootstrap flow (creates admin org+user) [VERIFIED: codebase read]
- `internal/adapters/primary/http/auth.go:Bootstrap` — HTTP bootstrap handler [VERIFIED: codebase read]
- `internal/core/domain/auth/organization.go` — Organization domain model with FinancialCutoffConfig [VERIFIED: codebase read]
- `internal/core/domain/auth/user.go` — User domain model (NewUser uses uuid.New()) [VERIFIED: codebase read]
- `web/src/routes/_authenticated/{org-hierarchy,contracts,projects,customers,time-entries}/` — Frontend routes [VERIFIED: codebase read]

### Tertiary (LOW confidence)
- SurrealDB `UPDATE` as upsert behavior — [ASSUMED] based on documentation pattern. Verify during implementation.
- SCHEMAFULL behavior with extra fields — [ASSUMED] because the Go code currently works in development

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All dependencies verified in `go.mod` and codebase
- Architecture: HIGH — Entity dependency graph verified against schema and Go adapters
- Pitfalls: HIGH — Field name mismatches confirmed by reading schema + Go structs
- Validation: HIGH — Manual verification approach documented with specific steps

**Research date:** 2026-05-19
**Valid until:** 2026-06-19 (stable project — no fast-moving dependencies)
