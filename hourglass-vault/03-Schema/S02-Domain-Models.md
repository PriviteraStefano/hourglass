# Schema: Domain Models

## Overview
Domain entities and value objects for Hourglass authentication system. These are pure Go types with no external dependencies (hexagonal domain layer).

---

## Authentication Domain

### User Entity
```go
// internal/core/domain/auth/user.go
package auth

import "github.com/google/uuid"

type User struct {
    ID        uuid.UUID
    Email     Email       // Value object
    Username  Username    // Value object (nullable)
    Password  Password    // Value object
    FirstName string
    LastName  string
    IsActive  bool
    CreatedAt time.Time
}

// Business rules
func NewUser(email, username, firstname, lastname string, password string) (*User, error) {
    // Validate email format
    emailVO, err := NewEmail(email)
    if err != nil {
        return nil, err
    }
    
    // Validate username (if provided)
    var usernameVO Username
    if username != "" {
        usernameVO, err = NewUsername(username)
        if err != nil {
            return nil, err
        }
    }
    
    // Validate and hash password
    passwordVO, err := NewPassword(password)
    if err != nil {
        return nil, err
    }
    
    return &User{
        ID:        uuid.New(),
        Email:     emailVO,
        Username:  usernameVO,
        Password:  passwordVO,
        FirstName: firstname,
        LastName:  lastname,
        IsActive:  true,
        CreatedAt: time.Now(),
    }, nil
}
```

---

### Value Object: Email
```go
// internal/core/domain/auth/credentials.go
package auth

import (
    "regexp"
    "strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type Email struct {
    address string
}

func NewEmail(address string) (Email, error) {
    address = strings.TrimSpace(strings.ToLower(address))
    
    if address == "" {
        return Email{}, ErrEmailEmpty
    }
    
    if !emailRegex.MatchString(address) {
        return Email{}, ErrEmailInvalid
    }
    
    if len(address) > 255 {
        return Email{}, ErrEmailTooLong
    }
    
    return Email{address: address}, nil
}

func (e Email) String() string {
    return e.address
}

func (e Email) Equals(other Email) bool {
    return e.address == other.address
}
```

**Validation Rules:**
- Must be valid email format
- Max length: 255 characters
- Normalized to lowercase
- Trimmed whitespace

---

### Value Object: Username
```go
// internal/core/domain/auth/credentials.go
package auth

import (
    "regexp"
    "strings"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type Username struct {
    value string
}

func NewUsername(value string) (Username, error) {
    value = strings.TrimSpace(strings.ToLower(value))
    
    if value == "" {
        return Username{}, ErrUsernameEmpty
    }
    
    if len(value) < 3 {
        return Username{}, ErrUsernameTooShort
    }
    
    if len(value) > 30 {
        return Username{}, ErrUsernameTooLong
    }
    
    if !usernameRegex.MatchString(value) {
        return Username{}, ErrUsernameInvalid
    }
    
    return Username{value: value}, nil
}

func (u Username) String() string {
    return u.value
}

func (u Username) Equals(other Username) bool {
    return u.value == other.value
}
```

**Validation Rules:**
- Length: 3-30 characters
- Characters: alphanumeric + underscore only
- Normalized to lowercase
- Trimmed whitespace

---

### Value Object: Password
```go
// internal/core/domain/auth/credentials.go
package auth

import (
    "golang.org/x/crypto/bcrypt"
)

type Password struct {
    hash string
}

func NewPassword(plain string) (Password, error) {
    if len(plain) < 8 {
        return Password{}, ErrPasswordTooShort
    }
    
    hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
    if err != nil {
        return Password{}, err
    }
    
    return Password{hash: string(hash)}, nil
}

func (p Password) Compare(plain string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plain))
    return err == nil
}

// Unsafe method - use only for testing
func (p Password) Hash() string {
    return p.hash
}
```

**Validation Rules:**
- Minimum length: 8 characters
- Hashed using bcrypt (cost 12)
- Plain text never stored or exposed

---

### Domain Errors
```go
// internal/core/domain/auth/errors.go
package auth

import "errors"

// Email errors
var (
    ErrEmailEmpty   = errors.New("email cannot be empty")
    ErrEmailInvalid = errors.New("invalid email format")
    ErrEmailTooLong = errors.New("email too long (max 255 characters)")
)

// Username errors
var (
    ErrUsernameEmpty      = errors.New("username cannot be empty")
    ErrUsernameTooShort   = errors.New("username must be at least 3 characters")
    ErrUsernameTooLong    = errors.New("username cannot exceed 30 characters")
    ErrUsernameInvalid    = errors.New("username can only contain letters, numbers, and underscores")
)

// Password errors
var (
    ErrPasswordTooShort = errors.New("password must be at least 8 characters")
    ErrPasswordMismatch = errors.New("password does not match")
)

// User errors
var (
    ErrUserNotFound     = errors.New("user not found")
    ErrUserInactive     = errors.New("user account is inactive")
    ErrInvalidCredentials = errors.New("invalid credentials")
)

// Invitation errors
var (
    ErrInvitationNotFound = errors.New("invitation not found")
    ErrInvitationExpired  = errors.New("invitation has expired")
    ErrInvitationUsed     = errors.New("invitation already used")
    ErrInvalidInviteCode  = errors.New("invalid invitation code")
)

// Password reset errors
var (
    ErrResetCodeExpired = errors.New("reset code has expired")
    ErrResetCodeUsed    = errors.New("reset code already used")
    ErrResetCodeInvalid = errors.New("invalid reset code")
    ErrRateLimitExceeded = errors.New("too many requests, please try again later")
)
```

---

## Invitation Domain

### Invitation Entity
```go
// internal/core/domain/auth/invitation.go
package auth

import (
    "github.com/google/uuid"
    "time"
)

type InvitationStatus string

const (
    StatusPending   InvitationStatus = "pending"
    StatusAccepted  InvitationStatus = "accepted"
    StatusExpired   InvitationStatus = "expired"
)

type Invitation struct {
    ID             uuid.UUID
    OrganizationID uuid.UUID
    Code           InviteCode
    Token          uuid.UUID
    Email          Email
    Status         InvitationStatus
    ExpiresAt      time.Time
    CreatedBy      uuid.UUID
    CreatedAt      time.Time
}

func NewInvitation(orgID uuid.UUID, email string, expiresInDays int, createdBy uuid.UUID) (*Invitation, error) {
    emailVO, err := NewEmail(email)
    if err != nil {
        return nil, err
    }
    
    code := GenerateInviteCode()
    token := uuid.New()
    expiresAt := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)
    
    return &Invitation{
        ID:             uuid.New(),
        OrganizationID: orgID,
        Code:           code,
        Token:          token,
        Email:          emailVO,
        Status:         StatusPending,
        ExpiresAt:      expiresAt,
        CreatedBy:      createdBy,
        CreatedAt:      time.Now(),
    }, nil
}

func (i *Invitation) Accept() error {
    if i.Status != StatusPending {
        return ErrInvitationUsed
    }
    
    if time.Now().After(i.ExpiresAt) {
        i.Status = StatusExpired
        return ErrInvitationExpired
    }
    
    i.Status = StatusAccepted
    return nil
}

func (i *Invitation) IsExpired() bool {
    return time.Now().After(i.ExpiresAt)
}
```

---

### Value Object: InviteCode
```go
// internal/core/domain/auth/invite_code.go
package auth

import (
    "crypto/rand"
    "math/big"
    "strings"
)

const inviteCodeLength = 6
const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // No I, O, 0, 1

type InviteCode struct {
    value string
}

func GenerateInviteCode() InviteCode {
    code := make([]byte, inviteCodeLength)
    for i := range code {
        n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
        code[i] = charset[n.Int64()]
    }
    return InviteCode{value: string(code)}
}

func NewInviteCode(value string) (InviteCode, error) {
    value = strings.ToUpper(strings.TrimSpace(value))
    
    if len(value) != inviteCodeLength {
        return InviteCode{}, ErrInvalidInviteCode
    }
    
    for _, c := range value {
        if !strings.ContainsRune(charset, c) {
            return InviteCode{}, ErrInvalidInviteCode
        }
    }
    
    return InviteCode{value: value}, nil
}

func (c InviteCode) String() string {
    return c.value
}

func (c InviteCode) Equals(other InviteCode) bool {
    return strings.EqualFold(c.value, other.value)
}
```

**Generation Rules:**
- Length: 6 characters
- Charset: A-Z (excluding I, O) + 2-9 (excluding 0, 1)
- Case-insensitive comparison
- Cryptographically random

---

## Password Reset Domain

### PasswordReset Entity
```go
// internal/core/domain/auth/password_reset.go
package auth

import (
    "github.com/google/uuid"
    "time"
)

type PasswordReset struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Code      ResetCode
    ExpiresAt time.Time
    UsedAt    *time.Time
    CreatedAt time.Time
}

func NewPasswordReset(userID uuid.UUID) *PasswordReset {
    code := GenerateResetCode()
    expiresAt := time.Now().Add(2 * time.Hour)
    
    return &PasswordReset{
        ID:        uuid.New(),
        UserID:    userID,
        Code:      code,
        ExpiresAt: expiresAt,
        UsedAt:    nil,
        CreatedAt: time.Now(),
    }
}

func (pr *PasswordReset) Verify(code string) error {
    if pr.UsedAt != nil {
        return ErrResetCodeUsed
    }
    
    if time.Now().After(pr.ExpiresAt) {
        return ErrResetCodeExpired
    }
    
    if !pr.Code.Equals(code) {
        return ErrResetCodeInvalid
    }
    
    return nil
}

func (pr *PasswordReset) MarkAsUsed() {
    now := time.Now()
    pr.UsedAt = &now
}
```

---

### Value Object: ResetCode
```go
// internal/core/domain/auth/reset_code.go
package auth

import (
    "crypto/rand"
    "fmt"
    "math/big"
)

const resetCodeLength = 6

type ResetCode struct {
    digits string
}

func GenerateResetCode() ResetCode {
    digits := make([]byte, resetCodeLength)
    for i := range digits {
        n, _ := rand.Int(rand.Reader, big.NewInt(10))
        digits[i] = '0' + byte(n.Int64())
    }
    return ResetCode{digits: string(digits)}
}

func NewResetCode(digits string) (ResetCode, error) {
    if len(digits) != resetCodeLength {
        return ResetCode{}, ErrResetCodeInvalid
    }
    
    for _, c := range digits {
        if c < '0' || c > '9' {
            return ResetCode{}, ErrResetCodeInvalid
        }
    }
    
    return ResetCode{digits: digits}, nil
}

func (c ResetCode) String() string {
    return c.digits
}

func (c ResetCode) Equals(other string) bool {
    return c.digits == other
}

func (c ResetCode) Hash() (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(c.digits), bcrypt.DefaultCost)
    return string(hash), err
}

func (c ResetCode) Compare(hashed string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(c.digits))
    return err == nil
}
```

**Generation Rules:**
- Length: 6 digits (e.g., "123456")
- Numeric only (0-9)
- Cryptographically random
- Hashed before storage

---

## Related Schema Docs
- [[S01-Database-ERD]] - Database tables
- [[S03-Ports-Interfaces]] - Repository ports
- [[S06-Value-Objects]] - Additional value objects

## Last Updated
- **PR**: #4ab2fb9, #d400192, #0ed701a, #41d8f09
- **Merged**: 2026-04-19
- **Author**: @hourglass-team
