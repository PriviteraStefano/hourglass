-- Drop indexes
DROP INDEX IF EXISTS idx_organization_memberships_organization_id;
DROP INDEX IF EXISTS idx_organization_memberships_user_id;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables
DROP TABLE IF EXISTS organization_memberships;
DROP TABLE IF EXISTS organizations;
DROP TABLE IF EXISTS users;

-- Drop extension
DROP EXTENSION IF EXISTS pgcrypto;