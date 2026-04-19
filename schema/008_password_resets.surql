-- ============================================================================
-- PASSWORD RESETS TABLE
-- ============================================================================

DEFINE TABLE password_resets SCHEMAFULL;

DEFINE FIELD id ON password_resets TYPE string;
DEFINE FIELD user_id ON password_resets TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD code_hash ON password_resets TYPE string ASSERT $value != NONE;
DEFINE FIELD expires_at ON password_resets TYPE datetime ASSERT $value != NONE;
DEFINE FIELD used_at ON password_resets TYPE option<datetime>;
DEFINE FIELD created_at ON password_resets TYPE datetime DEFAULT time::now();

DEFINE INDEX pr_user ON password_resets COLUMNS user_id;
