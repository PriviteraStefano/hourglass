# Feature: User Authentication

## Overview
Unified authentication system allowing users to log in with either username or email, request password resets via 6-digit codes, and securely manage their sessions with JWT tokens. This replaces the previous email-only login and adds self-service password recovery.

## User Stories
| ID | Story | Status | PR |
|----|-------|--------|-----|
| US-001 | As a user, I can log in with my username OR email so that I have flexibility in how I access my account | ✅ Implemented | #4ab2fb9 |
| US-002 | As a user who forgot my password, I can request a reset code so that I can regain access to my account | ✅ Implemented | #41d8f09 |
| US-003 | As a security-conscious user, I want reset codes to expire after 2 hours so that old codes cannot be misused | ✅ Implemented | #41d8f09 |
| US-004 | As a user, I want clear error messages when login fails so that I know whether to retry or reset my password | ✅ Implemented | #4ab2fb9 |

## User Workflows

### Workflow 1: Unified Login (Username or Email)
```mermaid
flowchart TD
    A[User navigates to /login] --> B[Enter username/email + password]
    B --> C[Click Login button]
    C --> D{Backend validates}
    D -->|WHERE email=X OR username=X| E[Password check bcrypt]
    E --> F{Credentials valid?}
    F -->|Yes| G[Generate JWT + refresh token]
    F -->|No| H[Show generic error message]
    G --> I[Store tokens in http only cookie]
    I --> J[Redirect to dashboard]
    H --> B
```

**Steps:**
1. User navigates to `/login` route
2. User enters either username (e.g., `johndoe`) OR email (e.g., `john@example.com`) in the "Username or Email" field
3. User enters password and clicks "Login"
4. Frontend sends `POST /auth/login` with `{ identifier: "johndoe", password: "***" }`
5. Backend queries database: `WHERE email = $identifier OR username = $identifier`
6. Backend validates password using bcrypt comparison
7. On success: Backend generates JWT access token and refresh token
8. Frontend stores tokens and redirects to authenticated dashboard
9. On failure: Generic error message shown (doesn't reveal if username exists)

### Workflow 2: Password Reset Request
```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant B as Backend
    participant DB as Database
    participant E as Email Service
    
    U->>F: Click "Forgot Password?"
    F->>U: Show reset request form
    U->>F: Enter email/username
    F->>B: POST /auth/password-reset/request
    B->>DB: Find user by identifier
    DB-->>B: User record
    B->>B: Generate 6-digit code
    B->>B: Hash code with bcrypt
    B->>DB: Store code_hash + expires_at
    B->>E: Send email with code
    B-->>F: { message: "reset code sent" }
    F->>U: Show verify form
```

### Workflow 3: Password Reset Verification
```mermaid
stateDiagram-v2
    [*] --> request_form: User clicks "Forgot Password?"
    request_form --> code_sent: POST /password-reset/request
    code_sent --> verify_form: User receives email
    verify_form --> validating: User submits code + new password
    validating --> password_updated: Code valid + password meets requirements
    validating --> invalid_code: Code incorrect
    invalid_code --> verify_form: Allow retry (max 3 attempts)
    password_updated --> [*]: Success - redirect to login
    code_sent --> expired: 2 hours pass
    expired --> [*]: Code unusable - must request new
```

## Acceptance Criteria
- [x] Login accepts username OR email in single `identifier` field
- [x] Existing users without usernames can still login with email
- [x] Password reset codes are 6 digits and expire after 2 hours
- [x] Rate limiting prevents more than 3 reset requests per hour
- [x] Generic error messages don't reveal whether username exists
- [x] Reset codes are bcrypt-hashed before storage
- [x] Used reset codes are marked as used and cannot be reused

## Related Features
- [[F05-Org-Bootstrap]] - Organization creation during registration
- [[F06-Invitation-System]] - Alternative registration via invitation
- [[T02-Auth-Implementation]] - Technical implementation details

## Last Updated
- **PR**: #4ab2fb9, #41d8f09
- **Merged**: 2026-04-19
- **Author**: @hourglass-team
