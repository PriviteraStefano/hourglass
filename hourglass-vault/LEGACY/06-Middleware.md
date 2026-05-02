# Middleware

Request processing pipeline for Hourglass backend.

## Overview

Middleware wraps HTTP handlers to perform cross-cutting concerns before handler execution:

```
Request
  ↓
Middleware 1 (Auth validation)
  ↓
Middleware 2 (Logging)
  ↓
Middleware 3 (CORS)
  ↓
Handler (user code)
  ↓
Response
```

---

## Auth Middleware

**File**: `internal/middleware/auth.go`

**Purpose**: Validate JWT token and inject user context into requests.

### Usage

```go
// Unprotected route
mux.HandleFunc("POST /auth/login", userHandler.Login)

// Protected route
mux.HandleFunc("GET /time-entries", middleware.Auth(authService, timeEntryHandler.List))
```

### Implementation

```go
func Auth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. Extract Bearer token
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
            return
        }
        
        const prefix = "Bearer "
        if !strings.HasPrefix(authHeader, prefix) {
            http.Error(w, "Invalid Authorization header", http.StatusUnauthorized)
            return
        }
        
        token := strings.TrimPrefix(authHeader, prefix)
        
        // 2. Validate token (signature, expiration)
        claims, err := authService.ValidateToken(token)
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
        
        // 3. Add claims to context
        ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
        ctx = context.WithValue(ctx, "org_id", claims.OrgID)
        ctx = context.WithValue(ctx, "role", claims.Role)
        
        // 4. Call wrapped handler with augmented request
        next(w, r.WithContext(ctx))
    }
}
```

### Accessing Context in Handlers

```go
func (h *TimeEntryHandler) List(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(uuid.UUID)
    orgID := r.Context().Value("org_id").(uuid.UUID)
    role := r.Context().Value("role").(models.Role)
    
    // Use values for queries
    query := "SELECT * FROM time_entries WHERE organization_id = $1"
    // ...
}
```

---

## Type Assertions & Safety

**Safe type assertion:**
```go
userID, ok := r.Context().Value("user_id").(uuid.UUID)
if !ok {
    // Fallback or error
    log.Printf("user_id not in context or wrong type")
    http.Error(w, "Internal error", http.StatusInternalServerError)
    return
}
```

**Unsafe (panics if wrong type):**
```go
userID := r.Context().Value("user_id").(uuid.UUID)  // Will panic!
```

---

## API Versioning Middleware

**File**: `internal/middleware/versioning.go` (if exists)

**Purpose**: Handle multiple API versions via headers.

**Usage:**
```
GET /time-entries
X-API-Version: 1
```

**Implementation Pattern:**
```go
func APIVersion(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        version := r.Header.Get("X-API-Version")
        if version == "" {
            version = "1"  // Default version
        }
        
        ctx := context.WithValue(r.Context(), "api_version", version)
        next(w, r.WithContext(ctx))
    }
}
```

---

## CORS Middleware

If needed, add CORS middleware:

```go
func CORS(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == http.MethodOptions {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        next(w, r)
    }
}
```

---

## Error Handling Middleware

Wrap handlers to catch panics:

```go
func Recover(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if err := recover(); err != nil {
                log.Printf("Panic recovered: %v", err)
                http.Error(w, "Internal server error", http.StatusInternalServerError)
            }
        }()
        next(w, r)
    }
}
```

---

## Middleware Chain Pattern

For multiple middlewares on a single route:

```go
// Manual chaining
handler := timeEntryHandler.List
handler = middleware.Recover(handler)
handler = middleware.Auth(authService, handler)

mux.HandleFunc("GET /time-entries", handler)
```

Or create a helper:

```go
func Protected(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
    return middleware.Auth(authService, next)
}

func Public(next http.HandlerFunc) http.HandlerFunc {
    return middleware.Recover(next)
}
```

---

## Request/Response Logging Middleware

```go
func LogRequest(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Wrap ResponseWriter to capture status code
        wrapped := &statusWriter{ResponseWriter: w}
        
        log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
        
        next(wrapped, r)
        
        log.Printf("%s %s %d (%dms)", 
            r.Method, r.URL.Path, wrapped.status, time.Since(start).Milliseconds())
    }
}

type statusWriter struct {
    http.ResponseWriter
    status int
}

func (w *statusWriter) WriteHeader(status int) {
    w.status = status
    w.ResponseWriter.WriteHeader(status)
}
```

---

## Organization Context Middleware

Ensure user belongs to requested org:

```go
func OrgContext(db *sql.DB) func(next http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            userID := r.Context().Value("user_id").(uuid.UUID)
            orgID := r.Context().Value("org_id").(uuid.UUID)
            
            // Verify membership
            var exists bool
            err := db.QueryRowContext(r.Context(),
                `SELECT EXISTS(
                    SELECT 1 FROM organization_memberships 
                    WHERE user_id = $1 AND organization_id = $2 AND is_active = true
                )`, userID, orgID).Scan(&exists)
            
            if err != nil || !exists {
                http.Error(w, "Not a member of this organization", http.StatusForbidden)
                return
            }
            
            next(w, r)
        }
    }
}
```

---

## Best Practices

### 1. Order Matters

```go
// Good: Auth first, then logging
handler = middleware.LogRequest(handler)
handler = middleware.Auth(authService, handler)

// Bad: Logging won't see auth context
handler = middleware.Auth(authService, handler)
handler = middleware.LogRequest(handler)
```

### 2. Context is Immutable

```go
// Correct
ctx := context.WithValue(r.Context(), "key", value)
r = r.WithContext(ctx)

// Wrong (doesn't affect r)
context.WithValue(r.Context(), "key", value)
```

### 3. Short Handler Signatures

Middleware should be simple, focused, testable:

```go
// Good: Single responsibility
func Auth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc

// Avoid: Too many parameters
func Auth(authService, cache, logger, metrics, etc...)
```

### 4. Document Context Keys

```go
// constants.go
const (
    ContextKeyUserID = "user_id"    // uuid.UUID
    ContextKeyOrgID = "org_id"      // uuid.UUID
    ContextKeyRole = "role"         // models.Role
)

// Usage
userID := r.Context().Value(ContextKeyUserID).(uuid.UUID)
```

---

**Next**: [[07-Frontend-Architecture]] for React development patterns.
