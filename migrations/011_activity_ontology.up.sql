-- 011_activity_ontology.up.sql — Activity Ontology: one recursive work entity
--
-- Big-bang schema rewrite per ADR-P-007 (D-1…D-8) and ADR-BE-014 (R-5).
-- Replaces the two-level projects/subprojects model with the recursive
-- `activities` table, rewrites every FK that pinned the old chain, drops the
-- dead `enforce_unit_tuple` toggle (ADR-P-001 Q3), and migrates the MVP seed
-- data in the same transaction.
--
-- Data migration notes (pre-deploy, MVP seed only — per ADR-P-007 D-6):
--   * projects   → activities, kind = 'engagement', parent_id = NULL,
--                  contract_id preserved, billable set from project_type.
--   * subprojects→ activities, kind = 'task', parent_id = the project's new
--                  activity id, contract_id = NULL (commercial context
--                  inherits upward per D-3).
--   * Activity ids keep the old row ids (project id / subproject id) so the
--     mapping is exact in both directions and the down migration restores
--     the original rows 1:1.
--   * Expenses with project_id NULL (non-project internal spend) get a new
--     per-org fallback activity of kind 'internal' named 'General & Admin'
--     so `expenses.activity_id` can be NOT NULL (D-4: required-activity).
--   * budget_caps.project_id is rewritten to activity_id (same pattern as
--     financial_cutoff_periods) — its FK to projects would otherwise block
--     the drop of the old table.

-- ============================================================================
-- 1. activity_kinds — org-level kind catalog (D-2), seeded for the MVP org
-- ============================================================================
CREATE TABLE activity_kinds (
    org_id   UUID NOT NULL REFERENCES organizations(id),
    name     VARCHAR(50) NOT NULL,                  -- 'engagement','phase','task','internal', + org's own
    is_seed  BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (org_id, name)
);

-- Seed the four canonical kinds for the MVP seed organization only.
INSERT INTO activity_kinds (org_id, name, is_seed)
SELECT o.id, k.name, TRUE
FROM organizations o
CROSS JOIN (VALUES ('engagement'), ('phase'), ('task'), ('internal')) AS k(name)
WHERE o.id = '019df8b0-0001-7000-8000-000000000001'
ON CONFLICT (org_id, name) DO NOTHING;

-- ============================================================================
-- 2. activities — the single recursive work entity (D-1, D-2, D-3, D-7)
-- ============================================================================
CREATE TABLE activities (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES organizations(id),
    parent_id         UUID REFERENCES activities(id) ON DELETE RESTRICT,   -- D-2: nullable, no level meaning
    name              VARCHAR(255) NOT NULL,
    description       TEXT,
    kind              VARCHAR(50) NOT NULL,          -- catalog FK (D-2), NOT a CHECK enum
    contract_id       UUID REFERENCES contracts(id) ON DELETE RESTRICT,    -- D-3: nullable = internal work
    governance_model  VARCHAR(50) NOT NULL DEFAULT 'creator_controlled'
                      CHECK (governance_model IN ('creator_controlled', 'unanimous', 'majority')),
    created_by_org_id UUID NOT NULL REFERENCES organizations(id),
    is_shared         BOOLEAN NOT NULL DEFAULT FALSE,
    billable          BOOLEAN,                       -- D-7: NULL = inherit from contract link / nearest ancestor
    budget_amount     DECIMAL(12,2),
    is_active         BOOLEAN NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (org_id, kind) REFERENCES activity_kinds(org_id, name)
);

CREATE INDEX idx_activities_org_id ON activities(org_id);
CREATE INDEX idx_activities_parent_id ON activities(parent_id);  -- BE-014 R-5: recursive walk support

-- ============================================================================
-- 3. activity_adoptions — mirrors project_adoptions (sharing preserved)
-- ============================================================================
CREATE TABLE activity_adoptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id     UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(activity_id, organization_id)
);

CREATE INDEX idx_activity_adoptions_activity_id ON activity_adoptions(activity_id);
CREATE INDEX idx_activity_adoptions_organization_id ON activity_adoptions(organization_id);

-- ============================================================================
-- 4. activity_managers — renamed from project_managers (ADR-P-007 D-1 note)
-- ============================================================================
CREATE TABLE activity_managers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    activity_id UUID NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(activity_id, user_id)
);

CREATE INDEX idx_activity_managers_activity_id ON activity_managers(activity_id);
CREATE INDEX idx_activity_managers_user_id ON activity_managers(user_id);

-- ============================================================================
-- 5. Data migration: projects → activities (kind 'engagement')
-- ============================================================================
INSERT INTO activities (id, org_id, parent_id, name, description, kind, contract_id,
                        governance_model, created_by_org_id, is_shared, billable,
                        budget_amount, is_active, created_at, updated_at)
SELECT id, org_id, NULL, name, description, 'engagement', contract_id,
       governance_model, created_by_org_id, is_shared,
       CASE WHEN project_type = 'billable' THEN TRUE ELSE FALSE END,
       budget_amount, is_active, created_at, updated_at
FROM projects;

-- ============================================================================
-- 6. Data migration: subprojects → activities (kind 'task', child of project)
-- ============================================================================
INSERT INTO activities (id, org_id, parent_id, name, description, kind, contract_id,
                        governance_model, created_by_org_id, is_shared, billable,
                        budget_amount, is_active, created_at, updated_at)
SELECT sp.id, p.org_id, sp.project_id, sp.name, sp.description, 'task', NULL,
       'creator_controlled', p.created_by_org_id, FALSE, NULL,
       NULL, sp.is_active, sp.created_at, sp.updated_at
FROM subprojects sp
JOIN projects p ON p.id = sp.project_id;

-- ============================================================================
-- 7. Data migration: project_managers → activity_managers
-- ============================================================================
INSERT INTO activity_managers (id, activity_id, user_id, created_at)
SELECT id, project_id, user_id, created_at FROM project_managers;

-- ============================================================================
-- 8. Data migration: project_adoptions → activity_adoptions
-- ============================================================================
INSERT INTO activity_adoptions (id, activity_id, organization_id, created_at)
SELECT id, project_id, organization_id, created_at FROM project_adoptions;

-- ============================================================================
-- 9. working_groups: subproject_id → activity_id; drop enforce_unit_tuple
-- ============================================================================
ALTER TABLE working_groups DROP CONSTRAINT IF EXISTS working_groups_subproject_id_fkey;
ALTER TABLE working_groups RENAME COLUMN subproject_id TO activity_id;
ALTER TABLE working_groups DROP COLUMN enforce_unit_tuple;
ALTER TABLE working_groups ADD CONSTRAINT working_groups_activity_id_fkey
    FOREIGN KEY (activity_id) REFERENCES activities(id) ON DELETE CASCADE;
DROP INDEX IF EXISTS idx_working_groups_subproject_id;
CREATE INDEX idx_working_groups_activity_id ON working_groups(activity_id);  -- BE-014 R-5

-- ============================================================================
-- 10. time_entries: four-FK chain collapses to activity_id (D-4)
-- ============================================================================
ALTER TABLE time_entries DROP CONSTRAINT IF EXISTS time_entries_project_id_fkey;
ALTER TABLE time_entries DROP CONSTRAINT IF EXISTS time_entries_subproject_id_fkey;
ALTER TABLE time_entries DROP CONSTRAINT IF EXISTS time_entries_wg_id_fkey;

ALTER TABLE time_entries ADD COLUMN activity_id UUID;
UPDATE time_entries SET activity_id = subproject_id;   -- child activity keeps the subproject id
ALTER TABLE time_entries ALTER COLUMN activity_id SET NOT NULL;
ALTER TABLE time_entries ADD CONSTRAINT time_entries_activity_id_fkey
    FOREIGN KEY (activity_id) REFERENCES activities(id);

ALTER TABLE time_entries DROP COLUMN project_id;        -- indexes on dropped columns drop automatically
ALTER TABLE time_entries DROP COLUMN subproject_id;
ALTER TABLE time_entries DROP COLUMN wg_id;

CREATE INDEX idx_time_entries_activity_id ON time_entries(activity_id);

-- ============================================================================
-- 11. expenses: project_id/customer_id → activity_id (D-4, symmetric with time)
--     Expenses with no project get a per-org 'internal' fallback activity.
-- ============================================================================
CREATE TEMP TABLE _expense_fallback (org_id UUID PRIMARY KEY, activity_id UUID NOT NULL);
INSERT INTO _expense_fallback (org_id, activity_id)
SELECT DISTINCT ON (org_id) org_id, gen_random_uuid() FROM expenses WHERE project_id IS NULL;

-- The fallback activity needs a kind row in its org's catalog (created
-- on-demand; not a seed kind for non-seed orgs).
INSERT INTO activity_kinds (org_id, name, is_seed)
SELECT org_id, 'internal', FALSE FROM _expense_fallback
ON CONFLICT (org_id, name) DO NOTHING;

INSERT INTO activities (id, org_id, parent_id, name, description, kind, contract_id,
                        governance_model, created_by_org_id, is_shared, billable,
                        budget_amount, is_active, created_at, updated_at)
SELECT f.activity_id, f.org_id, NULL, 'General & Admin',
       'Fallback internal activity for non-project expenses', 'internal', NULL,
       'creator_controlled', f.org_id, FALSE, FALSE,
       NULL, TRUE, NOW(), NOW()
FROM _expense_fallback f;

ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_project_id_fkey;

ALTER TABLE expenses ADD COLUMN activity_id UUID;
UPDATE expenses SET activity_id = project_id WHERE project_id IS NOT NULL;
UPDATE expenses e SET activity_id = f.activity_id
FROM _expense_fallback f
WHERE e.project_id IS NULL AND e.org_id = f.org_id;
ALTER TABLE expenses ALTER COLUMN activity_id SET NOT NULL;
ALTER TABLE expenses ADD CONSTRAINT expenses_activity_id_fkey
    FOREIGN KEY (activity_id) REFERENCES activities(id);

ALTER TABLE expenses DROP COLUMN project_id;            -- customer_id FK drops with the column
ALTER TABLE expenses DROP COLUMN customer_id;

CREATE INDEX idx_expenses_activity_id ON expenses(activity_id);

-- ============================================================================
-- 12. financial_cutoff_periods: project_id → activity_id
-- ============================================================================
ALTER TABLE financial_cutoff_periods DROP CONSTRAINT IF EXISTS financial_cutoff_periods_project_id_fkey;
ALTER TABLE financial_cutoff_periods RENAME COLUMN project_id TO activity_id;
ALTER TABLE financial_cutoff_periods ADD CONSTRAINT financial_cutoff_periods_activity_id_fkey
    FOREIGN KEY (activity_id) REFERENCES activities(id);
DROP INDEX IF EXISTS idx_financial_cutoff_periods_project_id;
CREATE INDEX idx_financial_cutoff_periods_activity_id ON financial_cutoff_periods(activity_id);

-- ============================================================================
-- 13. budget_caps: project_id → activity_id (FK would block the projects drop)
-- ============================================================================
ALTER TABLE budget_caps DROP CONSTRAINT IF EXISTS budget_caps_project_id_fkey;
ALTER TABLE budget_caps RENAME COLUMN project_id TO activity_id;
ALTER TABLE budget_caps ADD CONSTRAINT budget_caps_activity_id_fkey
    FOREIGN KEY (activity_id) REFERENCES activities(id);
DROP INDEX IF EXISTS idx_budget_caps_project_id;
CREATE INDEX idx_budget_caps_activity_id ON budget_caps(activity_id);

-- ============================================================================
-- 14. Drop the old two-level tables
-- ============================================================================
DROP TABLE project_managers;
DROP TABLE project_adoptions;
DROP TABLE subprojects;
DROP TABLE projects;

DROP TABLE _expense_fallback;
