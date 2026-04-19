# Authentication & Organization Registration - Implementation Guide

## Overview

This document describes the authentication and organization registration system implemented in Hourglass, including all new endpoints and schema changes.

## Schema Changes

### Users Table (`schema/006_user_fields.surql`)

New fields added to the `users` table:

```sql
DEFINE FIELD username ON TABLE users TYPE option<string>;
DEFINE FIELD firstname ON TABLE users TYPE option<string>;
DEFINE FIELD lastname ON TABLE users TYPE option<string>;
DEFINE INDEX user_username ON TABLE users COLUMNS username UNIQUE;
```

### Invitations Table (`schema/007_invitations.surql`)

New table for invitation-based user registration:

```sql
DEFINE TABLE invitations SCHEMAFULL;
DEFINE FIELD organization_id ON TABLE invitations TYPE string;
DEFINE FIELD code ON TABLE invitations TYPE string;       -- 6-char alphanumeric code
DEFINE FIELD invite_token ON TABLE invitations TYPE string; -- UUID token for email links
DEFINE FIELD email ON TABLE invitations TYPE option<string>;
DEFINE FIELD status ON TABLE invitations TYPE string DEFAULT 'pending';
DEFINE FIELD expires_at ON TABLE invitations TYPE datetime;
DEFINE FIELD created_by ON TABLE invitations TYPE string;
DEFINE FIELD created_at ON TABLE invitations TYPE datetime DEFAULT time::now();
DEFINE INDEX invite_code ON TABLE invitations COLUMNS code UNIQUE;
DEFINE INDEX invite_token ON TABLE invitations COLUMNS invite_token UNIQUE;
```

### Password Resets Table (`schema/008_password_resets.surql`)

New table for password reset functionality:

```sql
DEFINE TABLE password_resets SCHEMAFULL;
DEFINE FIELD user_id ON TABLE password_resets TYPE record<users>;
DEFINE FIELD code_hash ON TABLE password_resets TYPE string;
DEFINE FIELD expires_at ON TABLE password_resets TYPE datetime;
DEFINE FIELD used_at ON TABLE password_resets TYPE option<datetime>;
DEFINE FIELD created_at ON TABLE password_resets TYPE datetime DEFAULT time::now();
DEFINE INDEX pr_user ON TABLE password_resets COLUMNS user_id;
```

## API Endpoints

### Authentication

#### POST /auth/login
Login with username OR email.

**Request:**
```json
{
  "identifier": "johndoe OR john@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "data": {
    "token": "eyJhbGci...",
    "refresh_token": "base64...",
    "user": {
      "id": "user-id",
      "email": "john@example.com",
      "name": "John Doe",
      "role": "employee",
      "org_id": "org-id"
    },
    "expires_at": "2026-04-19T23:00:00Z"
  }
}
```

#### POST /auth/bootstrap
Create organization and admin user atomically. Returns JWT for auto-login.

**Request:**
```json
{
  "org_name": "Acme Corp",
  "firstname": "John",
  "lastname": "Doe",
  "username": "johndoe",
  "email": "john@example.com",
  "password": "password123"
}
```

**Response:**
```json
{
  "data": {
    "token": "eyJhbGci...",
    "user": {
      "id": "user-id",
      "email": "john@example.com",
      "username": "johndoe",
      "name": "John Doe"
    },
    "organization": {
      "id": "org-id",
      "name": "Acme Corp"
    }
  }
}
```

#### POST /auth/register
Standard user registration (without org creation).

**Request:**
```json
{
  "email": "john@example.com",
  "username": "johndoe",
  "firstname": "John",
  "lastname": "Doe",
  "password": "password123",
  "organization_name": "Acme Corp"
}
```

#### POST /auth/password-reset/request
Request a password reset code.

**Request:**
```json
{
  "identifier": "john@example.com OR johndoe"
}
```

**Response:**
```json
{
  "data": {
    "message": "reset code sent",
    "code": "123456",
    "expires_at": "2026-04-19T23:00:00Z"
  }
}
```

#### POST /auth/password-reset/verify
Verify reset code and set new password.

**Request:**
```json
{
  "identifier": "john@example.com OR johndoe",
  "code": "123456",
  "password": "newpassword123"
}
```

### Invitations

#### POST /invitations
Create a new invitation (admin/manager only).

**Request:**
```json
{
  "organization_id": "org-id",
  "email": "invitee@example.com",
  "expires_in_days": 7
}
```

**Response:**
```json
{
  "data": {
    "id": "invitation-id",
    "code": "ABC123",
    "token": "uuid-token",
    "link": "https://app.example.com/invite/uuid-token",
    "email": "invitee@example.com",
    "status": "pending",
    "expires_at": "2026-04-26T22:00:00Z",
    "organization_id": "org-id"
  }
}
```

#### GET /invitations/validate/code/:code
Validate an invitation by its 6-character code.

**Response (valid):**
```json
{
  "data": {
    "id": "invitation-id",
    "code": "ABC123",
    "email": "invitee@example.com",
    "status": "pending",
    "expires_at": "2026-04-26T22:00:00Z",
    "organization_id": "org-id"
  }
}
```

**Response (expired):**
- `410 Gone` with `{"error": "invitation has expired"}`

#### GET /invitations/validate/token/:token
Validate an invitation by its UUID token (for email links).

Same response format as code validation.

#### POST /invitations/accept
Accept an invitation and create user account.

**Request:**
```json
{
  "token": "uuid-token OR code",
  "email": "newuser@example.com",
  "username": "newuser",
  "password": "password123"
}
```

## Frontend Routes

| Route | Component | Description |
|-------|-----------|-------------|
| `/register` | `BootstrapOrgForm` | Create organization + admin |
| `/login` | `LoginForm` | Login with username/email |
| `/password-reset` | `PasswordResetRequestForm` | Request reset code |
| `/password-reset/verify` | `PasswordResetVerifyForm` | Enter code + new password |
| `/invite` | `InvitationAcceptForm` | Accept invitation |

## Running Tests

### Start SurrealDB
```bash
docker-compose up -d surrealdb
```

### Apply Schema
```bash
# Apply schema changes to SurrealDB
curl -s -X POST "http://localhost:8000/sql" -u root:root \
  -H "Content-Type: text/plain" \
  -H "surreal-ns: hourglass" \
  -H "surreal-db: main" \
  -d "$(cat schema/006_user_fields.surql)"

curl -s -X POST "http://localhost:8000/sql" -u root:root \
  -H "Content-Type: text/plain" \
  -H "surreal-ns: hourglass" \
  -H "surreal-db: main" \
  -d "$(cat schema/007_invitations.surql)"

curl -s -X POST "http://localhost:8000/sql" -u root:root \
  -H "Content-Type: text/plain" \
  -H "surreal-ns: hourglass" \
  -H "surreal-db: main" \
  -d "$(cat schema/008_password_resets.surql)"
```

### Run Integration Tests
```bash
export SURREALDB_URL=ws://localhost:8000/rpc
go test ./internal/handlers/... -v -run "TestSurrealAuth"
```

## Test Coverage

### Authentication Tests
- ✅ User registration with organization
- ✅ Registration validation (email, password, username)
- ✅ Login with username OR email
- ✅ Login validation (wrong password, non-existent user)
- ✅ Bootstrap organization creation
- ✅ Token refresh flow
- ✅ Password reset request
- ✅ Password reset verification

### Invitation Tests
- ✅ Create invitation
- ✅ Validate invitation by code
- ✅ Validate invitation by token
- ✅ Accept invitation (partial - user creation to be completed)

## Migration Notes

When migrating existing data:

1. Add `username` field - set to `NULL` for existing users
2. Add `firstname`/`lastname` fields - set to `NULL` or derive from existing `name`
3. Create `invitations` table
4. Create `password_resets` table

Existing users can still login with email until they set a username.
