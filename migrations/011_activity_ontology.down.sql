-- 011_activity_ontology.down.sql — reverse the activity ontology rewrite
--
-- Restores the two-level projects/subprojects model. This is a genuine
-- reverse (ADR-BE-004: real schema restoration, not a stub), but it is
-- best-effort with documented lossiness:
--
--   * Metadata not representable in the old model is dropped:
--       - billable, budget_amount (kept only where it maps to projects.budget_amount),
--         kind (engagement/task/etc.), is_shared on children
--       - activities.description on subprojects is preserved (old model has one)
--   * Activities that do not map to the two-level model are NOT restored:
--       - depth > 2 (a grandchild's parent is a subproject, not a project)
--       - root activities whose kind is not 'engagement' (e.g. the 'internal'
--         'General & Admin' fallback created by the up migration)
--       - children of non-engagement roots
--     Expenses that referenced such activities restore project_id = NULL
--     (their pre-011 state); working_groups anchored to them are deleted
--     with their wg_members (CASCADE).
--   * subprojects.sequence_order restores as 0 (order was never captured on
--     activities).
--   * time_entries referencing a non-restored activity (a personal-activity
--     entry) cannot be represented and would fail the NOT NULL restore —
--     outside the supported pre-deploy seed scope.

-- ============================================================================
-- 1. Recreate projects (original shape from 000_full_schema.up.sql)
-- ============================================================================
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    project_type VARCHAR(50) NOT NULL CHECK (project_type IN ('billable', 'internal')),
    type VARCHAR(50) NOT NULL CHECK (type IN ('billable', 'internal')),
    contract_id UUID REFERENCES contracts(id) ON DELETE RESTRICT,
    customer_id UUID REFERENCES customers(id),
    governance_model VARCHAR(50) NOT NULL CHECK (governance_model IN ('creator_controlled', 'unanimous', 'majority')),
    created_by_org_id UUID NOT NULL REFERENCES organizations(id),
    is_shared BOOLEAN NOT NULL DEFAULT FALSE,
    budget_amount DECIMAL(12,2),
    financial_cutoff_config JSONB,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_org_id ON projects(org_id);
CREATE INDEX idx_projects_customer_id ON projects(customer_id);

-- ============================================================================
-- 2. Recreate subprojects (original shape)
-- ============================================================================
CREATE TABLE subprojects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    sequence_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_subprojects_project_id ON subprojects(project_id);

-- ============================================================================
-- 3. Recreate project_managers / project_adoptions (original shapes)
-- ============================================================================
CREATE TABLE project_managers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, user_id)
);

CREATE INDEX idx_project_managers_project_id ON project_managers(project_id);
CREATE INDEX idx_project_managers_user_id ON project_managers(user_id);

CREATE TABLE project_adoptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project_id, organization_id)
);

CREATE INDEX idx_project_adoptions_project_id ON project_adoptions(project_id);
CREATE INDEX idx_project_adoptions_organization_id ON project_adoptions(organization_id);

-- ============================================================================
-- 4. Reverse data mapping: engagement roots → projects
--    customer_id derived via the contract link (lossy where NULL)
-- ============================================================================
INSERT INTO projects (id, org_id, name, description, project_type, type, contract_id,
                      customer_id, governance_model, created_by_org_id, is_shared,
                      budget_amount, financial_cutoff_config, is_active, created_at, updated_at)
SELECT a.id, a.org_id, a.name, a.description,
       CASE WHEN a.billable THEN 'billable' ELSE 'internal' END,
       CASE WHEN a.billable THEN 'billable' ELSE 'internal' END,
       a.contract_id, c.customer_id, a.governance_model, a.created_by_org_id, a.is_shared,
       a.budget_amount, NULL, a.is_active, a.created_at, a.updated_at
FROM activities a
LEFT JOIN contracts c ON c.id = a.contract_id
WHERE a.parent_id IS NULL AND a.kind = 'engagement';

-- ============================================================================
-- 5. Reverse data mapping: children of restored projects → subprojects
--    (depth > 2 and children of non-engagement roots are excluded — lossy)
-- ============================================================================
INSERT INTO subprojects (id, project_id, name, description, sequence_order, is_active, created_at, updated_at)
SELECT a.id, a.parent_id, a.name, a.description, 0, a.is_active, a.created_at, a.updated_at
FROM activities a
WHERE a.parent_id IS NOT NULL
  AND a.parent_id IN (SELECT id FROM projects);

-- ============================================================================
-- 6. Reverse data mapping: activity_managers / activity_adoptions → old tables
--    (rows pointing at non-restored activities are dropped — lossy)
-- ============================================================================
INSERT INTO project_managers (id, project_id, user_id, created_at)
SELECT id, activity_id, user_id, created_at
FROM activity_managers
WHERE activity_id IN (SELECT id FROM projects);

INSERT INTO project_adoptions (id, project_id, organization_id, created_at)
SELECT id, activity_id, organization_id, created_at
FROM activity_adoptions
WHERE activity_id IN (SELECT id FROM projects);

-- ============================================================================
-- 7. working_groups: activity_id → subproject_id; restore enforce_unit_tuple
-- ============================================================================
ALTER TABLE working_groups DROP CONSTRAINT IF EXISTS working_groups_activity_id_fkey;
ALTER TABLE working_groups RENAME COLUMN activity_id TO subproject_id;

-- WGs anchored to a non-restored activity cannot exist in the old model;
-- delete them (cascades to wg_members) before re-adding the FK.
DELETE FROM working_groups WHERE subproject_id NOT IN (SELECT id FROM subprojects);

ALTER TABLE working_groups ADD CONSTRAINT working_groups_subproject_id_fkey
    FOREIGN KEY (subproject_id) REFERENCES subprojects(id) ON DELETE CASCADE;
ALTER TABLE working_groups ADD COLUMN enforce_unit_tuple BOOLEAN NOT NULL DEFAULT TRUE;

DROP INDEX IF EXISTS idx_working_groups_activity_id;
CREATE INDEX idx_working_groups_subproject_id ON working_groups(subproject_id);

-- ============================================================================
-- 8. time_entries: restore project_id / subproject_id / wg_id
--    (activity tree walk; entries on non-restored activities are unsupported)
-- ============================================================================
ALTER TABLE time_entries DROP CONSTRAINT IF EXISTS time_entries_activity_id_fkey;
ALTER TABLE time_entries ADD COLUMN project_id UUID;
ALTER TABLE time_entries ADD COLUMN subproject_id UUID;
ALTER TABLE time_entries ADD COLUMN wg_id UUID;

UPDATE time_entries te
SET project_id = a.parent_id,
    subproject_id = a.id,
    wg_id = wg.id
FROM activities a
LEFT JOIN working_groups wg ON wg.subproject_id = a.id
WHERE te.activity_id = a.id;

ALTER TABLE time_entries ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE time_entries ALTER COLUMN subproject_id SET NOT NULL;
ALTER TABLE time_entries ALTER COLUMN wg_id SET NOT NULL;

ALTER TABLE time_entries ADD CONSTRAINT time_entries_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE time_entries ADD CONSTRAINT time_entries_subproject_id_fkey
    FOREIGN KEY (subproject_id) REFERENCES subprojects(id);
ALTER TABLE time_entries ADD CONSTRAINT time_entries_wg_id_fkey
    FOREIGN KEY (wg_id) REFERENCES working_groups(id);

ALTER TABLE time_entries DROP COLUMN activity_id;    -- idx_time_entries_activity_id drops with it

CREATE INDEX idx_time_entries_project_id ON time_entries(project_id);
CREATE INDEX idx_time_entries_wg_id ON time_entries(wg_id);

-- ============================================================================
-- 9. expenses: restore project_id / customer_id (customer via contract link)
--    Expenses whose activity did not map to a restored project go back to
--    project_id = NULL — their pre-011 state.
-- ============================================================================
ALTER TABLE expenses DROP CONSTRAINT IF EXISTS expenses_activity_id_fkey;
ALTER TABLE expenses ADD COLUMN project_id UUID;
ALTER TABLE expenses ADD COLUMN customer_id UUID;

UPDATE expenses e
SET project_id = p.id,
    customer_id = c.customer_id
FROM activities a
LEFT JOIN projects p ON p.id = COALESCE(a.parent_id, a.id)
LEFT JOIN contracts c ON c.id = a.contract_id
WHERE e.activity_id = a.id;

ALTER TABLE expenses ADD CONSTRAINT expenses_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id);
ALTER TABLE expenses ADD CONSTRAINT expenses_customer_id_fkey
    FOREIGN KEY (customer_id) REFERENCES customers(id);

ALTER TABLE expenses DROP COLUMN activity_id;        -- idx_expenses_activity_id drops with it

CREATE INDEX idx_expenses_project_id ON expenses(project_id);

-- ============================================================================
-- 10. financial_cutoff_periods / budget_caps: activity_id → project_id
-- ============================================================================
ALTER TABLE financial_cutoff_periods DROP CONSTRAINT IF EXISTS financial_cutoff_periods_activity_id_fkey;
ALTER TABLE financial_cutoff_periods RENAME COLUMN activity_id TO project_id;
ALTER TABLE financial_cutoff_periods ADD CONSTRAINT financial_cutoff_periods_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id);
DROP INDEX IF EXISTS idx_financial_cutoff_periods_activity_id;
CREATE INDEX idx_financial_cutoff_periods_project_id ON financial_cutoff_periods(project_id);

ALTER TABLE budget_caps DROP CONSTRAINT IF EXISTS budget_caps_activity_id_fkey;
ALTER TABLE budget_caps RENAME COLUMN activity_id TO project_id;
ALTER TABLE budget_caps ADD CONSTRAINT budget_caps_project_id_fkey
    FOREIGN KEY (project_id) REFERENCES projects(id);
DROP INDEX IF EXISTS idx_budget_caps_activity_id;
CREATE INDEX idx_budget_caps_project_id ON budget_caps(project_id);

-- ============================================================================
-- 11. Drop the new ontology tables (dependency order)
-- ============================================================================
DROP TABLE activity_managers;
DROP TABLE activity_adoptions;
DROP TABLE activities;
DROP TABLE activity_kinds;
