# Schema: Ports & Interfaces

## Overview
Port interfaces (contracts) that define the boundaries between domain services and external implementations. Following hexagonal architecture principles.

---

## Authentication Ports

### UserRepository Port
```go
// internal/core/ports/user_repository.go
package ports

import (
    "context"
    "github.com/google/uuid"
    "hourglass/internal/core/domain/auth"
)

type UserRepository interface {
    // Find user by ID
    FindByID(ctx context.Context, id uuid.UUID) (*auth.User, error)
    
    // Find user by email OR username (unified login)
    FindByIdentifier(ctx context.Context, identifier string) (*auth.User, error)
    
    // Find user by email only
    FindByEmail(ctx context.Context, email auth.Email) (*auth.User, error)
    
    // Find user by username only
    FindByUsername(ctx context.Context, username auth.Username) (*auth.User, error)
    
    // Create new user
    Create(ctx context.Context, user *auth.User) error
    
    // Update existing user
    Update(ctx context.Context, user *auth.User) error
    
    // Check if username exists
    UsernameExists(ctx context.Context, username auth.Username) (bool, error)
    
    // Check if email exists
    EmailExists(ctx context.Context, email auth.Email) (bool, error)
}
```

**Implementation Notes:**
- `FindByIdentifier` queries: `WHERE email = $1 OR username = $1`
- All methods respect soft delete (`is_active = true`)
- Implementations: `SurrealUserRepository`, `PostgreSQLUserRepository`

---

### TokenService Port
```go
// internal/core/ports/token_service.go
package ports

import "github.com/google/uuid"

type TokenService interface {
    // Generate JWT access token and refresh token
    Generate(userID uuid.UUID, email string) (*TokenPair, error)
    
    // Validate JWT access token
    Validate(token string) (*Claims, error)
    
    // Refresh access token using refresh token
    Refresh(refreshToken string) (*TokenPair, error)
    
    // Revoke refresh token (logout)
    Revoke(refreshToken string) error
}

type TokenPair struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

type Claims struct {
    UserID   uuid.UUID
    Email    string
    Exp      int64
    IssuedAt int64
}
```

**Implementation Notes:**
- Implementations: `JWTTokenService`, `MockTokenService` (for testing)
- Access token expiry: 15 minutes
- Refresh token expiry: 7 days
- Refresh tokens are tracked for revocation

---

### PasswordHasher Port
```go
// internal/core/ports/password_hasher.go
package ports

type PasswordHasher interface {
    // Hash plain text password
    Hash(plain string) (string, error)
    
    // Compare plain text with hash
    Compare(hash, plain string) bool
}
```

**Implementation Notes:**
- Uses bcrypt with cost 12
- Implementations: `BcryptPasswordHasher`, `MockPasswordHasher`

---

### InvitationRepository Port
```go
// internal/core/ports/invitation_repository.go
package ports

import (
    "context"
    "github.com/google/uuid"
    "hourglass/internal/core/domain/auth"
)

type InvitationRepository interface {
    // Create new invitation
    Create(ctx context.Context, invitation *auth.Invitation) error
    
    // Find by 6-character code
    FindByCode(ctx context.Context, code auth.InviteCode) (*auth.Invitation, error)
    
    // Find by UUID token
    FindByToken(ctx context.Context, token uuid.UUID) (*auth.Invitation, error)
    
    // Mark invitation as accepted
    Accept(ctx context.Context, id uuid.UUID) error
    
    // Delete expired invitations (cleanup)
    DeleteExpired(ctx context.Context, olderThan time.Time) error
}
```

**Implementation Notes:**
- Code lookups are case-insensitive
- Token lookups are exact match
- Invitations expire after 7 days by default

---

### PasswordResetRepository Port
```go
// internal/core/ports/password_reset_repository.go
package ports

import (
    "context"
    "github.com/google/uuid"
    "hourglass/internal/core/domain/auth"
)

type PasswordResetRepository interface {
    // Create new password reset request
    Create(ctx context.Context, reset *auth.PasswordReset) error
    
    // Find by user ID (get latest active reset)
    FindByUserID(ctx context.Context, userID uuid.UUID) (*auth.PasswordReset, error)
    
    // Find by code hash
    FindByCodeHash(ctx context.Context, codeHash string) (*auth.PasswordReset, error)
    
    // Mark as used
    MarkAsUsed(ctx context.Context, id uuid.UUID) error
    
    // Delete old resets (cleanup)
    DeleteOlderThan(ctx context.Context, olderThan time.Time) error
}
```

**Implementation Notes:**
- Only one active reset per user at a time
- Old resets are cleaned up automatically
- Codes are hashed before storage

---

## Organization Management Ports

### OrganizationRepository Port
```go
// internal/core/ports/organization_repository.go
package ports

import (
    "context"
    "github.com/google/uuid"
    "hourglass/internal/core/domain/org"
)

type OrganizationRepository interface {
    // Create new organization
    Create(ctx context.Context, org *org.Organization) error
    
    // Find by ID
    FindByID(ctx context.Context, id uuid.UUID) (*org.Organization, error)
    
    // Find by slug
    FindBySlug(ctx context.Context, slug string) (*org.Organization, error)
    
    // Update organization
    Update(ctx context.Context, org *org.Organization) error
    
    // Generate unique slug from name
    GenerateUniqueSlug(ctx context.Context, name string) (string, error)
}
```

---

### MembershipRepository Port
```go
// internal/core/ports/membership_repository.go
package ports

import (
    "context"
    "github.com/google/uuid"
    "hourglass/internal/core/domain/org"
)

type MembershipRepository interface {
    // Create membership
    Create(ctx context.Context, userID, orgID uuid.UUID, role org.Role) error
    
    // Get user's role in organization
    GetRole(ctx context.Context, userID, orgID uuid.UUID) (org.Role, error)
    
    // Update user's role
    UpdateRole(ctx context.Context, userID, orgID uuid.UUID, role org.Role) error
    
    // List all members of organization
    ListMembers(ctx context.Context, orgID uuid.UUID) ([]*org.Membership, error)
    
    // List all organizations for user
    ListForUser(ctx context.Context, userID uuid.UUID) ([]*org.Membership, error)
    
    // Deactivate membership
    Deactivate(ctx context.Context, userID, orgID uuid.UUID) error
}
```

---

## Common Ports

### IDGenerator Port
```go
// internal/core/ports/id_generator.go
package ports

import "github.com/google/uuid"

type IDGenerator interface {
    Generate() uuid.UUID
}

// Default implementation uses uuid.New()
type UUIDGenerator struct{}

func (g *UUIDGenerator) Generate() uuid.UUID {
    return uuid.New()
}
```

---

### Clock Port (for testing time-dependent logic)
```go
// internal/core/ports/clock.go
package ports

import "time"

type Clock interface {
    Now() time.Time
    After(d time.Duration) <-chan time.Time
}

// Real clock implementation
type RealClock struct{}

func (c *RealClock) Now() time.Time {
    return time.Now()
}

func (c *RealClock) After(d time.Duration) <-chan time.Time {
    return time.After(d)
}

// Mock clock for testing
type MockClock struct {
    currentTime time.Time
}

func NewMockClock(t time.Time) *MockClock {
    return &MockClock{currentTime: t}
}

func (c *MockClock) Now() time.Time {
    return c.currentTime
}

func (c *MockClock) After(d time.Duration) <-chan time.Time {
    ch := make(chan time.Time, 1)
    go func() {
        c.currentTime = c.currentTime.Add(d)
        ch <- c.currentTime
    }()
    return ch
}
```

**Usage Example:**
```go
type AuthService struct {
    repo  ports.UserRepository
    clock ports.Clock  // Injected for testability
}

func (s *AuthService) IsExpired(t time.Time) bool {
    return s.clock.Now().After(t)
}
```

---

## Secondary Adapter Implementations

### SurrealDB Adapters
| Port | Implementation | File |
|------|---------------|------|
| `UserRepository` | `SurrealUserRepository` | `adapters/secondary/surrealdb/user_repository.go` |
| `TokenService` | `SurrealTokenService` | `adapters/secondary/surrealdb/token_service.go` |
| `PasswordHasher` | `BcryptPasswordHasher` | `adapters/secondary/surrealdb/password_hasher.go` |
| `InvitationRepository` | `SurrealInvitationRepository` | `adapters/secondary/surrealdb/invitation_repository.go` |
| `OrganizationRepository` | `SurrealOrganizationRepository` | `adapters/secondary/surrealdb/organization_repository.go` |

### PostgreSQL Adapters (Future)
| Port | Implementation | File |
|------|---------------|------|
| `UserRepository` | `PostgreSQLUserRepository` | `adapters/secondary/postgresql/user_repository.go` |
| `TokenService` | `JWTTokenService` | `adapters/secondary/postgresql/token_service.go` |
| ... | ... | ... |

---

## Testing with Mocks

### Mock UserRepository
```go
// internal/core/ports/mocks/user_repository_mock.go
package mocks

import (
    "context"
    "github.com/google/uuid"
    "hourglass/internal/core/domain/auth"
    "hourglass/internal/core/ports"
)

type MockUserRepository struct {
    users map[uuid.UUID]*auth.User
    emails map[string]*auth.User
    usernames map[string]*auth.User
}

func NewMockUserRepository() *MockUserRepository {
    return &MockUserRepository{
        users: make(map[uuid.UUID]*auth.User),
        emails: make(map[string]*auth.User),
        usernames: make(map[string]*auth.User),
    }
}

func (m *MockUserRepository) FindByIdentifier(ctx context.Context, identifier string) (*auth.User, error) {
    // Try email first
    if user, ok := m.emails[identifier]; ok {
        return user, nil
    }
    // Try username
    if user, ok := m.usernames[identifier]; ok {
        return user, nil
    }
    return nil, auth.ErrUserNotFound
}

func (m *MockUserRepository) Create(ctx context.Context, user *auth.User) error {
    m.users[user.ID] = user
    m.emails[user.Email.String()] = user
    if user.Username.String() != "" {
        m.usernames[user.Username.String()] = user
    }
    return nil
}

// ... implement other methods
```

### Test Example
```go
func TestAuthService_Login_Success(t *testing.T) {
    // Arrange
    mockRepo := ports.NewMockUserRepository()
    mockToken := &MockTokenService{}
    mockHasher := &MockPasswordHasher{}
    
    service := auth.NewAuthService(mockRepo, mockToken, mockHasher)
    
    // Act
    user, token, err := service.Login(context.Background(), "johndoe", "password123")
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.NotNil(t, token)
}
```

---

## Related Schema Docs
- [[S02-Domain-Models]] - Domain entities
- [[S01-Database-ERD]] - Database schema
- [[T01-Hexagonal-Architecture]] - Architecture pattern

## Last Updated
- **PR**: #b9a1f8d, #4ab2fb9
- **Merged**: 2026-04-19
- **Author**: @hourglass-team
