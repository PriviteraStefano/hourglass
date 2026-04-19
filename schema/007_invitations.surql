-- ============================================================================
-- INVITATIONS TABLE
-- ============================================================================

DEFINE TABLE invitations SCHEMAFULL
    PERMISSIONS
        FOR select WHERE $auth.role == 'admin' OR $auth.role == 'manager'
        FOR create WHERE $auth.role == 'admin' OR $auth.role == 'manager'
        FOR delete WHERE $auth.role == 'admin'
;

DEFINE FIELD id ON invitations TYPE string;
DEFINE FIELD organization_id ON invitations TYPE record<organizations> ASSERT $value != NONE;
DEFINE FIELD code ON invitations TYPE string ASSERT $value != NONE;
DEFINE FIELD invite_token ON invitations TYPE string ASSERT $value != NONE;
DEFINE FIELD email ON invitations TYPE option<string>;
DEFINE FIELD status ON invitations TYPE string DEFAULT 'pending' ASSERT $value IN ['pending', 'accepted', 'expired'];
DEFINE FIELD expires_at ON invitations TYPE datetime ASSERT $value != NONE;
DEFINE FIELD created_by ON invitations TYPE record<users> ASSERT $value != NONE;
DEFINE FIELD created_at ON invitations TYPE datetime DEFAULT time::now();

DEFINE INDEX invite_code ON invitations COLUMNS code UNIQUE;
DEFINE INDEX invite_token ON invitations COLUMNS invite_token UNIQUE;
DEFINE INDEX invite_org ON invitations COLUMNS organization_id;
