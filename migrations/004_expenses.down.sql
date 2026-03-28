-- Drop trigger
DROP TRIGGER IF EXISTS update_expenses_updated_at ON expenses;

-- Drop indexes
DROP INDEX IF EXISTS idx_expense_receipts_expense_item_id;
DROP INDEX IF EXISTS idx_expense_items_project_id;
DROP INDEX IF EXISTS idx_expense_items_expense_id;
DROP INDEX IF EXISTS idx_expenses_status;
DROP INDEX IF EXISTS idx_expenses_date;
DROP INDEX IF EXISTS idx_expenses_organization_id;
DROP INDEX IF EXISTS idx_expenses_user_id;

-- Drop tables
DROP TABLE IF EXISTS expense_receipts;
DROP TABLE IF EXISTS expense_items;
DROP TABLE IF EXISTS expenses;