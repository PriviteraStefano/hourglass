# Schema: State Machines

## Overview
State machines and status transitions for authentication flows, invitations, and password resets.

---

## Invitation State Machine

### States
```mermaid
stateDiagram-v2
    [*] --> pending: Admin creates invitation
    pending --> accepted: Invitee accepts
    pending --> expired: 7 days pass
    accepted --> [*]: Consumed
    expired --> [*]: Cannot be used
    
    note right of pending
        - Code is valid
        - Token is valid
        - Can be accepted
    end note
    
    note right of accepted
        - User created
        - Membership created
        - Cannot be reused
    end note
    
    note right of expired
        - Past expires_at
        - Returns 410 Gone
        - Must create new invite
    end note
```

### State Transitions

| From | To | Trigger | Guard Condition | Action |
|------|----|---------|-----------------|--------|
| `[*]` | `pending` | Admin creates | User is admin | Generate code + token |
| `pending` | `accepted` | Invitee accepts | Not expired + Status=pending | Create user + membership |
| `pending` | `expired` | Time passes | `now() > expires_at` | Mark as expired |
| `accepted` | `[*]` | Complete | Always | None (terminal) |
| `expired` | `[*]` | Complete | Always | None (terminal) |

### Business Rules
- **Invariant 1:** Only one active invitation per email at a time
- **Invariant 2:** Codes are case-insensitive but stored uppercase
- **Invariant 3:** Once accepted, invitation cannot be modified
- **Guard:** Accept fails if `status != pending` or `now() > expires_at`

---

## Password Reset State Machine

### States
```mermaid
stateDiagram-v2
    [*] --> idle: No active reset request
    idle --> requested: User requests reset
    requested --> verified: User enters correct code
    requested --> expired: 2 hours pass
    requested --> exhausted: 3 failed attempts
    verified --> completed: Password updated
    verified --> failed: Invalid attempt
    failed --> verified: Retry (max 3)
    failed --> exhausted: Max attempts reached
    completed --> [*]: Success
    expired --> [*]: Must request new
    exhausted --> [*]: Must request new
    
    state requested {
        [*] --> code_sent
        code_sent --> waiting_verification
    }
    
    state verified {
        [*] --> validating
        validating --> password_change_allowed
    }
```

### State Transitions

| From | To | Trigger | Guard Condition | Action |
|------|----|---------|-----------------|--------|
| `idle` | `requested` | POST /password-reset/request | Rate limit OK | Generate 6-digit code, send email |
| `requested` | `verified` | POST /password-reset/verify | Code matches + not expired | Allow password change |
| `requested` | `expired` | Time passes | `now() > expires_at` | Invalidate code |
| `requested` | `exhausted` | Failed attempts | `attempts >= 3` | Lock reset request |
| `verified` | `completed` | Valid password submitted | Password meets requirements | Update password, mark code used |
| `verified` | `failed` | Invalid code entered | `attempts < 3` | Increment attempt counter |
| `failed` | `exhausted` | Max attempts | `attempts >= 3` | Invalidate code |
| `completed` | `[*]` | Success | Always | Redirect to login |

### Business Rules
- **Invariant 1:** Only one active reset per user at a time
- **Invariant 2:** Codes expire after exactly 2 hours
- **Invariant 3:** Maximum 3 verification attempts per code
- **Invariant 4:** Used codes are marked and cannot be reused
- **Rate Limit:** Max 3 reset requests per hour per user
- **Guard:** Verification fails if `used_at != nil` or `now() > expires_at`

### Rate Limiting Logic
```mermaid
flowchart TD
    A[User requests reset] --> B{Check last 3 requests}
    B -->|All within 1 hour| C[Return 429 Too Many Requests]
    B -->|Less than 3 in hour| D[Allow request]
    D --> E[Generate new code]
    E --> F[Send email]
    F --> G[Update rate limit tracking]
```

---

## User Authentication State Machine

### Login Flow States
```mermaid
stateDiagram-v2
    [*] --> unauthenticated
    unauthenticated --> validating: Credentials submitted
    validating --> authenticated: Valid credentials
    validating --> failed: Invalid credentials
    failed --> unauthenticated: Show error
    authenticated --> token_expired: Access token expires
    token_expired --> refreshing: Use refresh token
    refreshing --> authenticated: Refresh successful
    refreshing --> unauthenticated: Refresh failed/expired
    authenticated --> logged_out: User logs out
    logged_out --> [*]
    
    note right of authenticated
        - JWT access token valid
        - Can make API requests
        - Token expires in 15 min
    end note
    
    note right of token_expired
        - Access token expired
        - Refresh token still valid
        - Must refresh before continuing
    end note
```

### Registration State Machine
```mermaid
stateDiagram-v2
    [*] --> form_displayed: User navigates to /register
    form_displayed --> validating: Form submitted
    validating --> validation_failed: Invalid data
    validating --> checking_uniqueness: Data valid
    checking_uniqueness --> uniqueness_failed: Username/email exists
    checking_uniqueness --> creating: Unique
    creating --> transaction_started: Begin transaction
    transaction_started --> org_created: Create organization
    org_created --> user_created: Create user
    user_created --> membership_created: Create admin membership
    membership_created --> committed: Commit transaction
    committed --> auto_logged_in: Generate JWT
    auto_logged_in --> [*]: Redirect to dashboard
    
    validation_failed --> form_displayed: Show errors
    uniqueness_failed --> form_displayed: Show conflict error
```

---

## Organization Bootstrap Transaction

### Atomic Operation Flow
```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant S as Service
    participant TX as Transaction
    participant DB as Database
    
    C->>H: POST /auth/bootstrap
    H->>S: Bootstrap(request)
    S->>TX: BEGIN
    TX-->>S: Transaction started
    
    S->>DB: CREATE organization
    DB-->>S: org_id
    
    S->>DB: CREATE user
    DB-->>S: user_id
    
    S->>DB: CREATE membership (user_id, org_id, admin)
    DB-->>S: membership_id
    
    S->>S: All operations successful?
    S->>TX: COMMIT
    TX-->>S: Committed
    
    S->>S: Generate JWT
    S-->>H: {token, user, org}
    H-->>C: 200 OK
```

### Rollback Scenarios
```mermaid
flowchart TD
    A[Begin transaction] --> B[Create organization]
    B --> C{Success?}
    C -->|No| D[ROLLBACK]
    C -->|Yes| E[Create user]
    E --> F{Success?}
    F -->|No| D
    F -->|Yes| G[Create membership]
    G --> H{Success?}
    H -->|No| D
    H -->|Yes| I[COMMIT]
    
    D --> J[Return 500 Error]
    I --> K[Return 200 Success]
```

**Rollback Triggers:**
1. Organization creation fails (duplicate slug)
2. User creation fails (duplicate username/email)
3. Membership creation fails (foreign key violation)
4. Any database constraint violation

---

## Invitation Acceptance Flow

### Combined State Machine
```mermaid
stateDiagram-v2
    state "Invitation" as Inv {
        [*] --> pending
        pending --> accepted
        pending --> expired
    }
    
    state "User" as User {
        [*] --> registering
        registering --> created
    }
    
    state "Membership" as Mem {
        [*] --> none
        none --> employee
    }
    
    pending --> registering: Accept invitation
    registering --> created: Validate + create user
    created --> employee: Create membership
    employee --> accepted: Mark invitation accepted
    
    expired --> [*]: Cannot accept
    accepted --> [*]: Complete
```

---

## Related Schema Docs
- [[S01-Database-ERD]] - Database tables that store states
- [[S02-Domain-Models]] - Entities with state fields
- [[S04-API-Contracts]] - Endpoints that trigger transitions

## Last Updated
- **PR**: #0ed701a, #41d8f09, #d400192
- **Merged**: 2026-04-19
- **Author**: @hourglass-team
