# Plan: User Authentication & Organization Registration

> Source PRD: `docs/superpowers/specs/2026-04-19-user-auth-org-registration-design.md`

## Architectural Decisions

Durable decisions that apply across all phases:

- **Routes**:
  - `POST /auth/login` - Accept `{"identifier": "username_or_email", "password": "..."}` (modified)
  - `POST /auth/register` - Modified to include username, firstname, lastname
  - `POST /auth/bootstrap` - NEW: Atomic org + admin creation
  - `POST /auth/password-reset/request` - NEW: Request reset
  - `POST /auth/password-reset/verify` - NEW: Verify code + set password
  - `POST /invitations` - NEW: Create invitation (admin)
  - `GET /invitations/validate/:code` - NEW: Validate invite code
  - `GET /invitations/validate/token/:token` - NEW: Validate email token
  - `POST /invitations/accept` - NEW: Accept invitation + create user

- **Schema Changes**:
  - `users`: Add `username` (UNIQUE), `firstname`, `lastname`; deprecate `name`
  - `invitations`: NEW table (id, organization_id, code, token, email, status, expires_at, created_by, created_at)
  - `password_resets`: NEW table (id, user_id, code_hash, expires_at, used_at, created_at)

- **Key Models**:
  - `Invitation` struct with code/token generation
  - `PasswordReset` struct with bcrypt-hashed codes
  - Modified `User` struct with username, firstname, lastname

- **Validation Rules**:
  - Username: 3-30 chars, alphanumeric + underscore, unique
  - Password: min 8 chars
  - Invite code: 6 alphanumeric chars, case-insensitive
  - Password reset code: 6 digits

---

## Phase 1: Unified Login (Username or Email)

**User stories**: 1, 2, 3

### What to build

Modify the existing login system to accept either username OR email in a single `identifier` field. Users can log in with their username instead of email. The backend query changes from `WHERE email = $email` to `WHERE email = $identifier OR username = $identifier`. Frontend login form changes from labeled "Email" to labeled "Username or Email".

### Acceptance criteria

- [ ] Backend query accepts username OR email for login
- [ ] Existing users (without username) can still log in with email
- [ ] Frontend login form shows "Username or Email" label
- [ ] Clear error message when credentials are invalid
- [ ] Test: Login with username succeeds
- [ ] Test: Login with email succeeds
- [ ] Test: Invalid credentials returns 401 with generic error

---

## Phase 2: Bootstrap Organization Creation

**User stories**: 4, 5, 6, 7

### What to build

New endpoint `POST /auth/bootstrap` that atomically creates an organization AND the first admin user in a single transaction. The user provides org name, username, firstname, lastname, email, and password. On success, returns JWT token (auto-login). Frontend gets a new registration flow: a form that creates org + admin in one step, then redirects to dashboard.

### Acceptance criteria

- [ ] `POST /auth/bootstrap` creates org + user in atomic transaction
- [ ] User is auto-logged in (JWT returned in response)
- [ ] Username uniqueness enforced (409 Conflict if duplicate)
- [ ] Email uniqueness enforced (409 Conflict if duplicate)
- [ ] Password validation (min 8 chars)
- [ ] Frontend bootstrap form validates all fields
- [ ] Test: Bootstrap creates org and user
- [ ] Test: Duplicate username/email returns 409
- [ ] Test: Transaction rollback on partial failure

---

## Phase 3: Invitation Code Generation

**User stories**: 9, 10

### What to build

New `invitations` table and `POST /invitations` endpoint (admin only). Generates a 6-character alphanumeric code and a UUID token. Returns both the shareable code and a full invitation link. Admins can copy either to share. Codes expire after configurable period (default 7 days). Frontend component shows generated code and link with copy buttons.

### Acceptance criteria

- [ ] `invitations` table created with migration
- [ ] `POST /invitations` generates unique 6-char code + UUID token
- [ ] Admin-only authorization (403 for non-admins)
- [ ] Response includes `code`, `link` (full URL with token)
- [ ] Frontend displays code and link with copy buttons
- [ ] Test: Invitation created with valid code/token
- [ ] Test: Non-admin receives 403
- [ ] Test: Code uniqueness enforced

---

## Phase 4: Invitation Acceptance & Registration

**User stories**: 11, 12, 13, 14

### What to build

Endpoints for invitation validation and acceptance. `GET /invitations/validate/:code` and `GET /invitations/validate/token/:token` check if invitation is valid/pending. `POST /invitations/accept` creates user account + membership atomically (assigning `employee` role by default). Frontend: code input component (6-char with auto-advance), email link deep linking to registration form, registration form pre-populated from invitation.

### Acceptance criteria

- [ ] `GET /invitations/validate/:code` returns invitation details or 404/410
- [ ] `GET /invitations/validate/token/:token` handles email links
- [ ] `POST /invitations/accept` creates user + membership atomically
- [ ] New users assigned `employee` role automatically
- [ ] Invitation marked as `accepted` after use
- [ ] Expired invitations return 410 Gone
- [ ] Frontend code input with auto-advance between characters
- [ ] Test: Valid invitation accepts and creates user
- [ ] Test: Invalid code returns 404
- [ ] Test: Expired invitation returns 410
- [ ] Test: Duplicate email registration rejected (409)

---

## Phase 5: Password Reset

**User stories**: 15, 16, 17, 18

### What to build

New `password_resets` table. `POST /auth/password-reset/request` accepts email or username, generates 6-digit code (bcrypt hashed), stores with 2-hour expiry. `POST /auth/password-reset/verify` accepts email/username, code, and new password; verifies and updates password. Frontend: request form (enter email/username), verify form (enter code + new password).

### Acceptance criteria

- [ ] `password_resets` table created with migration
- [ ] `POST /auth/password-reset/request` creates hashed reset code
- [ ] Code expires after 2 hours (410 on use)
- [ ] `POST /auth/password-reset/verify` validates code and updates password
- [ ] Rate limiting on reset requests (429 Too Many Requests)
- [ ] Frontend request form with email/username input
- [ ] Frontend verify form with code input and new password
- [ ] Test: Reset request creates valid code
- [ ] Test: Valid code allows password reset
- [ ] Test: Expired code rejected
- [ ] Test: Rate limiting enforced