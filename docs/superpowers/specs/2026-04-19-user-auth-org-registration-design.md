# User Authentication & Organization Registration Flow

## Problem Statement

The current authentication system lacks proper user registration flows. Users cannot:
- Log in with a username (only email)
- Create or join organizations in a structured way
- Invite users to organizations via multiple methods

## Solution

Implement three distinct user flows:
1. **Bootstrap Organization Creation** - Admin creates organization and first admin user atomically
2. **Invitation-based Registration** - Users join existing organizations via email link or 6-character code
3. **Unified Login** - Single login field accepting username OR email with password
4. **Password Reset** - Users can reset password via 6-digit code sent to email

## User Stories

### Authentication & Login
1. As a user, I want to log in with my username and password, so that I can access my account
2. As a user, I want to log in with my email and password, so that I can access my account if I prefer email over username
3. As a user, I want to see clear error messages when login fails, so that I understand what went wrong

### Bootstrap Organization Creation
4. As an admin, I want to create a new organization, so that my team can track time and expenses
5. As an admin, I want to create the first admin user during organization creation, so that I can manage the organization immediately
6. As an admin, I want to be automatically logged in after creating the organization, so that I can start using the system right away
7. As an admin, I want to set my first name, last name, and username during registration, so that other team members can identify me

### Invitation System
8. As an admin, I want to invite a user via email, so that they receive a direct link to join my organization
9. As an admin, I want to generate a 6-character invite code, so that I can share it manually with users
10. As an admin, I want to copy an invitation link, so that I can share it via any communication channel
11. As an invited user, I want to enter a 6-character code, so that I can join an organization quickly
12. As an invited user, I want to click an email link, so that I can join an organization seamlessly
13. As an invited user, I want to create my account during invitation acceptance, so that I can access the organization immediately
14. As an invited user, I want to be assigned the "employee" role by default, so that the admin can manage permissions

### Password Reset
15. As a user, I want to request a password reset by providing my email or username, so that I can recover access to my account
16. As a user, I want to enter a 6-digit code sent to my email, so that I can verify my identity
17. As a user, I want to set a new password after code verification, so that I can regain access to my account
18. As a user, I want the reset code to expire after 2 hours, so that stale requests don't remain valid

### Data Integrity
19. As a system, I want all related database operations to be wrapped in transactions, so that partial states are never persisted

## Implementation Decisions

### Database Schema Changes

**Users Table**
- Add `username` (TEXT, UNIQUE, NOT NULL) - user's unique login identifier
- Add `firstname` (TEXT, NOT NULL)
- Add `lastname` (TEXT, NOT NULL)
- Keep `email` (TEXT, UNIQUE, NOT NULL)
- Deprecate `name` field - migrate to firstname/lastname

**Organizations Table**
- No schema changes required

**Invitations Table (NEW)**
- `id` (uuid, primary key)
- `organization_id` (uuid, foreign key to organizations)
- `code` (TEXT, UNIQUE, NOT NULL) - 6-char alphanumeric
- `token` (TEXT, UNIQUE, NOT NULL) - random UUID for email links
- `email` (TEXT, nullable) - for email invitations
- `status` (TEXT) - "pending", "accepted", "expired"
- `expires_at` (timestamp)
- `created_by` (uuid, foreign key to users)
- `created_at` (timestamp)

**Password Resets Table (NEW)**
- `id` (uuid, primary key)
- `user_id` (uuid, foreign key to users)
- `code_hash` (TEXT, NOT NULL) - bcrypt hashed 6-digit code
- `expires_at` (timestamp, NOT NULL) - 2 hours from creation
- `used_at` (timestamp, nullable)
- `created_at` (timestamp)

### API Endpoints

**Authentication**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `POST /auth/login` | POST | Accept `{"identifier": "username_or_email", "password": "..."}` - identifier accepts either username or email |
| `POST /auth/register` | POST | Accept `{"email", "username", "firstname", "lastname", "password", "organization_id"}` for invited users |
| `POST /auth/bootstrap` | POST | Accept `{"org_name", "email", "username", "firstname", "lastname", "password"}` - creates org + admin atomically |

**Organizations**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `POST /organizations` | POST | Create new organization (used during bootstrap) |
| `GET /organizations/:id` | GET | Get organization details |

**Invitations**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `POST /invitations` | POST | Create invitation (admin only) - returns `{code, link, sent_via}` where sent_via is "email", "link", or "code" |
| `GET /invitations/validate/:code` | GET | Validate a code |
| `GET /invitations/validate/:token` | GET | Validate an email link token |
| `POST /invitations/accept` | POST | Accept invitation, create user account |

**Password Reset**
| Endpoint | Method | Purpose |
|----------|--------|---------|
| `POST /auth/password-reset/request` | POST | Send reset code to user's email |
| `POST /auth/password-reset/verify` | POST | Verify code + set new password |

### Module Design

**Backend**
1. **AuthHandler** - Modify login to check username OR email
2. **AuthHandler** - Add bootstrap endpoint with transaction wrapping org + user creation
3. **InvitationHandler** (NEW) - Manage invitation lifecycle
4. **InvitationService** (NEW) - Business logic for invitation codes, tokens, expiration
5. **PasswordResetHandler** (NEW) - Password reset request and verify

**Frontend**
1. **LoginPage** - Single input for username/email + password field
2. **RegisterPage** - Path selector (Bootstrap org creation vs Join via invitation)
3. **BootstrapOrgForm** - Create org + admin user in single flow
4. **JoinOrgForm** - Accept invitation (code input or email link)
5. **InviteCodeInput** (component) - 6-character code entry with auto-advance
6. **PasswordResetRequestForm** - Enter username/email to request reset
7. **PasswordResetVerifyForm** - Enter 6-digit code and new password

### Transaction Requirements

All multi-step operations MUST use database transactions:
- `POST /auth/bootstrap` - Atomic: create organization + create admin user
- `POST /invitations/accept` - Atomic: mark invitation accepted + create user

### Validation Rules

- Username: 3-30 characters, alphanumeric + underscores, unique
- Password: minimum 8 characters
- Invite code: 6 alphanumeric characters, case-insensitive
- Organization name: 2-100 characters
- Firstname/Lastname: 1-50 characters each
- Password reset code: 6 digits

### Error Handling

- 400 Bad Request - Invalid input, validation failures
- 401 Unauthorized - Invalid credentials
- 403 Forbidden - Not authorized for operation
- 404 Not Found - Invalid invitation code/token
- 409 Conflict - Username or email already exists
- 410 Gone - Invitation or reset code expired
- 429 Too Many Requests - Rate limit on password reset requests

## Testing Decisions

### Backend Tests
- Test login with username
- Test login with email
- Test bootstrap creates org + user atomically
- Test bootstrap rolls back org if user creation fails
- Test invitation code validation
- Test invitation token validation
- Test invitation acceptance
- Test duplicate username/email rejection
- Test password reset request
- Test password reset code verification
- Test expired reset code rejection

### Frontend Tests
- Test login form validation
- Test bootstrap flow auto-login
- Test code input component
- Test invitation link deep link handling
- Test password reset flow

## Out of Scope

- Email sending infrastructure (invitation emails, reset emails)
- Organization switching (users belong to single org)
- OAuth/SSO integration
- Two-factor authentication
- Session management (beyond JWT)
