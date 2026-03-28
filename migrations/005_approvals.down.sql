-- Drop indexes
DROP INDEX IF EXISTS idx_backup_approvers_user_id;
DROP INDEX IF EXISTS idx_backup_approvers_org_role;
DROP INDEX IF EXISTS idx_expense_approvals_created_at;
DROP INDEX IF EXISTS idx_expense_approvals_actor_id;
DROP INDEX IF EXISTS idx_expense_approvals_expense_id;
DROP INDEX IF EXISTS idx_time_entry_approvals_created_at;
DROP INDEX IF EXISTS idx_time_entry_approvals_actor_id;
DROP INDEX IF EXISTS idx_time_entry_approvals_entry_id;

-- Drop tables
DROP TABLE IF EXISTS backup_approvers;
DROP TABLE IF EXISTS expense_approvals;
DROP TABLE IF EXISTS time_entry_approvals;
