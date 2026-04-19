-- ============================================================================
-- USER FIELDS: username, firstname, lastname
-- ============================================================================

DEFINE FIELD username ON TABLE users TYPE option<string>;
DEFINE FIELD firstname ON TABLE users TYPE option<string>;
DEFINE FIELD lastname ON TABLE users TYPE option<string>;

DEFINE INDEX user_username ON TABLE users COLUMNS username UNIQUE;
