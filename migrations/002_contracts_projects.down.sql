-- Drop indexes
DROP INDEX IF EXISTS idx_project_adoptions_organization_id;
DROP INDEX IF EXISTS idx_project_adoptions_project_id;
DROP INDEX IF EXISTS idx_contract_adoptions_organization_id;
DROP INDEX IF EXISTS idx_contract_adoptions_contract_id;
DROP INDEX IF EXISTS idx_projects_is_shared;
DROP INDEX IF EXISTS idx_projects_created_by_org;
DROP INDEX IF EXISTS idx_projects_contract_id;
DROP INDEX IF EXISTS idx_contracts_is_shared;
DROP INDEX IF EXISTS idx_contracts_created_by_org;

-- Drop tables
DROP TABLE IF EXISTS project_adoptions;
DROP TABLE IF EXISTS contract_adoptions;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS contracts;