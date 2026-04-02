# Database Migrations

How Hourglass manages database schema evolution with numbered SQL files.

---

## Overview

**Hourglass uses a simple migration system:**
- Numbered SQL files: `001_init.up.sql`, `001_init.down.sql`, etc.
- Mounted into PostgreSQL init directory in Docker
- Applied automatically on first container start
- Manual execution for local PostgreSQL

---

## Migration Files

Location: `/migrations/`

### Naming Convention

```
NNN_description.up.sql     # Apply migration (NNN = 001, 002, 003, ...)
NNN_description.down.sql   # Rollback migration
```

### Example Structure

```
001_init.up.sql           # Create users, organizations, basic schema
001_init.down.sql         # Drop all tables

002_contracts_projects.up.sql
002_contracts_projects.down.sql

003_time_entries.up.sql    # Add time entry tables
003_time_entries.down.sql

...

008_verification_tokens.up.sql
008_verification_tokens.down.sql
```

---

## Current Migrations

| # | Description | Tables Added |
|---|-------------|--------------|
| 001 | Initial schema | users, organizations, memberships |
| 002 | Contracts & projects | contracts, projects, adoptions |
| 003 | Time entries | time_entries, time_entry_items |
| 004 | Expenses | expenses, expense_mileage_details |
| 005 | Approvals | time_entry_approvals, expense_approvals |
| 006 | Refresh tokens | refresh_tokens |
| 007 | Phase 2 schema | Flattened entry structure, new indexes |
| 008 | Verification tokens | verification_tokens |

---

## Creating a New Migration

### Step 1: Create Files

```bash
# New feature: Add project managers table
touch migrations/009_project_managers.up.sql
touch migrations/009_project_managers.down.sql
```

### Step 2: Write Up Migration

**File**: `009_project_managers.up.sql`

```sql
-- Create table
CREATE TABLE project_managers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, user_id)
);

-- Create index for queries
CREATE INDEX idx_project_managers_project_id 
ON project_managers(project_id);

CREATE INDEX idx_project_managers_user_id 
ON project_managers(user_id);

-- If modifying existing table:
-- ALTER TABLE time_entries ADD COLUMN new_column VARCHAR;
-- ALTER TABLE time_entries DROP COLUMN old_column;
```

### Step 3: Write Down Migration

**File**: `009_project_managers.down.sql`

```sql
-- Drop in reverse order
DROP INDEX IF EXISTS idx_project_managers_user_id;
DROP INDEX IF EXISTS idx_project_managers_project_id;
DROP TABLE IF EXISTS project_managers;

-- If altered a table, reverse the change:
-- ALTER TABLE time_entries DROP COLUMN new_column;
-- ALTER TABLE time_entries ADD COLUMN old_column VARCHAR;
```

### Step 4: Test Locally

**Option A: Docker**
```bash
# Stop container to reset
docker-compose down -v

# Restart (will run all migrations)
docker-compose up -d

# Verify
docker-compose exec postgres psql -U hourglass -d hourglass -c "\dt"
```

**Option B: Local PostgreSQL**
```bash
# Apply
psql -U hourglass -d hourglass -f migrations/009_project_managers.up.sql

# Verify
psql -U hourglass -d hourglass -c "\dt project_managers"

# Rollback test
psql -U hourglass -d hourglass -f migrations/009_project_managers.down.sql

# Re-apply
psql -U hourglass -d hourglass -f migrations/009_project_managers.up.sql
```

---

## Migration Best Practices

### 1. Idempotency

Use `IF NOT EXISTS` / `IF EXISTS` to make migrations safe to re-run:

```sql
-- Good: Safe to run multiple times
CREATE TABLE IF NOT EXISTS users (...);
DROP TABLE IF EXISTS old_table;

-- Bad: Will fail if already exists
CREATE TABLE users (...);
```

### 2. Backwards Compatibility

Don't break existing queries when adding columns:

```sql
-- Good: Add nullable column (doesn't break code)
ALTER TABLE time_entries ADD COLUMN notes TEXT;

-- Better: Add with default (for backfill)
ALTER TABLE time_entries ADD COLUMN status VARCHAR DEFAULT 'draft';

-- Bad: Makes column NOT NULL without backfill
ALTER TABLE time_entries ADD COLUMN status VARCHAR NOT NULL;
```

### 3. Data Migration

When transforming data, do it in the up migration:

```sql
-- Add new column
ALTER TABLE time_entries ADD COLUMN new_status VARCHAR;

-- Backfill based on old column
UPDATE time_entries SET new_status = 'approved' WHERE old_status = 'approved';
UPDATE time_entries SET new_status = 'draft' WHERE old_status IS NULL;

-- Remove old column
ALTER TABLE time_entries DROP COLUMN old_status;

-- Make NOT NULL after data is present
ALTER TABLE time_entries ALTER COLUMN new_status SET NOT NULL;
```

**Rollback handles both**:
```sql
-- Undo in reverse
ALTER TABLE time_entries DROP COLUMN new_status;
ALTER TABLE time_entries ADD COLUMN old_status VARCHAR;
-- ... restore data ...
```

### 4. Performance

For large tables, consider:

```sql
-- Add index after adding column to avoid rebuilds
ALTER TABLE time_entries ADD COLUMN organization_id UUID;

-- Build index separately (PostgreSQL allows concurrent index creation)
CREATE INDEX CONCURRENTLY idx_te_org_id ON time_entries(organization_id);
```

### 5. Foreign Keys

Always specify CASCADE behavior:

```sql
-- Good: Cascade deletes when contract is removed
ALTER TABLE projects ADD CONSTRAINT fk_projects_contracts
FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE CASCADE;

-- Also good: Prevent deletion if children exist (safer)
FOREIGN KEY (contract_id) REFERENCES contracts(id) ON DELETE RESTRICT;
```

---

## Testing Migrations

### Full Test Cycle

```bash
# 1. Start fresh
docker-compose down -v
docker-compose up -d

# 2. Verify schema
docker-compose exec postgres psql -U hourglass -d hourglass -c "\dt"

# 3. Verify data integrity
docker-compose exec postgres psql -U hourglass -d hourglass -c "SELECT COUNT(*) FROM users;"

# 4. Run application tests
go test ./...
```

### Check Specific Migration

```bash
# List all tables
psql -U hourglass -d hourglass -c "\dt"

# Describe a table
psql -U hourglass -d hourglass -c "\d time_entries"

# View indexes
psql -U hourglass -d hourglass -c "\di"

# View constraints
psql -U hourglass -d hourglass -c "\d time_entries" | grep "Constraints"
```

---

## Common Migration Patterns

### Add Role Constraint

```sql
-- Add new role value to CHECK constraint
ALTER TABLE organization_memberships
DROP CONSTRAINT organization_memberships_role_check;

ALTER TABLE organization_memberships
ADD CONSTRAINT organization_memberships_role_check
CHECK (role IN ('employee', 'manager', 'finance', 'customer', 'admin'));
```

### Add Soft Delete

```sql
-- Track deletion without removing data
ALTER TABLE users ADD COLUMN is_active BOOLEAN DEFAULT true;

-- Update existing records
UPDATE users SET is_active = true WHERE is_active IS NULL;

-- Make not null
ALTER TABLE users ALTER COLUMN is_active SET NOT NULL;

-- Index for queries that filter by is_active
CREATE INDEX idx_users_active ON users(is_active) WHERE is_active = true;
```

### Add Audit Trail

```sql
-- Track updates
ALTER TABLE time_entries ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Trigger to update timestamp (PostgreSQL)
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_time_entries_timestamp
BEFORE UPDATE ON time_entries
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();
```

---

## Deployment Migrations

### In Production

Assuming you add a migration tool (not yet in codebase):

```bash
# Before deploying new code
./hourglass migrate up

# If something goes wrong
./hourglass migrate down --count=1

# Deploy new code
# ...
```

### Zero-Downtime Migrations

For large tables, follow this pattern:

```sql
-- Phase 1 (before code deploy): Add column, index separately
ALTER TABLE time_entries ADD COLUMN new_column VARCHAR;
CREATE INDEX CONCURRENTLY idx_te_new_col ON time_entries(new_column);

-- Phase 2: Deploy code that uses new column
-- ... code changes ...

-- Phase 3: (optional) Drop old column
ALTER TABLE time_entries DROP COLUMN old_column;
```

---

## Viewing Migration History

Currently, Hourglass doesn't track which migrations ran. Future enhancement:

```sql
-- Table to track applied migrations
CREATE TABLE schema_migrations (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- After each migration
INSERT INTO schema_migrations (name) VALUES ('001_init');
```

---

## Troubleshooting

### Migration Failed

Check logs:
```bash
docker-compose logs postgres

# Or manually test
psql -U hourglass -d hourglass -f migrations/009_project_managers.up.sql
# Check error message
```

### Rollback Complications

If a down migration is missing or broken:

1. **Manual rollback:**
   ```bash
   # Manually undo the SQL changes
   psql -U hourglass -d hourglass
   # DROP TABLE broken_table;
   # ALTER TABLE old_table DROP COLUMN new_col;
   ```

2. **Skip future application:**
   - Delete the .up.sql file to prevent re-application
   - Document the manual fix

### Schema Out of Sync

```bash
# Rebuild from migrations
docker-compose down -v
docker-compose up -d

# Verify
docker-compose exec postgres psql -U hourglass -d hourglass -c "\dt"
```

---

**Next**: [[03-Database-Schema]] for table reference, or [[17-Testing]] for test patterns.
