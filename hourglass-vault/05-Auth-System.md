# Auth System

Complete guide to Hourglass authentication: JWT tokens, password hashing, and refresh flows.

## Overview

Hourglass uses **JWT-based stateless authentication**:

1. User registers with email/password
2. Backend hashes password with bcrypt
3. User logs in with email/password
4. Backend validates, generates JWT access token
5. Frontend stores JWT in localStorage
6. All requests include `Authorization: Bearer <token>`
7. Tokens expire; frontend uses refresh tokens to get new ones

---

## JWT Token Structure

### Access Token

**Payload:**
```json
{
  "sub": "user-id-uuid",
  "org_id": "org-id-uuid", 
  "role": "manager",
  "exp": 1704067200
}
```

**Header:**
```
Authorization: Bearer eyJhbGc...
```

**Expiration**: Short-lived (typically 15 minutes)

**Secret**: Set via `JWT_SECRET` environment variable (defaults to "dev-secret-change-in-production")

### Refresh Token

**Storage**: `refresh_tokens` table

| Column | Details |
|--------|---------|
| id | UUID primary key |
| user_id | References users(id) |
| token_hash | Hashed token (never store plaintext) |
| expires_at | Expiration time (typically 7 days) |

**Rotation**: Each refresh generates new token, old one invalidated

---

## Endpoints

### POST /auth/register

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure123",
  "name": "John Doe"
}
```

**Flow:**
1. Validate email not already registered
2. Validate password strength (recommend 8+ chars)
3. Hash password with bcrypt
4. Create user record
5. Create verification token
6. Send verification email (if email enabled)
7. Return user object (no tokens yet)

**Response (201):**
```json
{
  "data": {
    "id": "user-uuid",
    "email": "user@example.com",
    "name": "John Doe",
    "is_active": false
  }
}
```

---

### POST /auth/verify

**Request:**
```json
{
  "token": "verification-token-from-email"
}
```

**Flow:**
1. Validate verification token exists and not expired
2. Mark user as `is_active = true`
3. Delete verification token
4. Return success

**Response (200):**
```json
{
  "data": {
    "message": "Email verified. Please log in."
  }
}
```

---

### POST /auth/login

**Request:**
```json
{
  "email": "user@example.com",
  "password": "secure123",
  "organization_id": "org-uuid"  // Which org to log into
}
```

**Flow:**
1. Find user by email
2. Validate password against bcrypt hash
3. Check user is active
4. Find org membership (user must have role in org)
5. Generate JWT access token
6. Generate refresh token, store in DB
7. Return tokens to frontend

**Response (200):**
```json
{
  "data": {
    "access_token": "eyJhbGc...",
    "refresh_token": "long-token-string",
    "user": {
      "id": "user-uuid",
      "email": "user@example.com",
      "name": "John Doe"
    },
    "organization": {
      "id": "org-uuid",
      "name": "Company Inc"
    }
  }
}
```

**Frontend Storage:**
```javascript
localStorage.setItem('access_token', data.access_token)
localStorage.setItem('refresh_token', data.refresh_token)
localStorage.setItem('current_org', data.organization.id)
```

---

### POST /auth/refresh

**Request:**
```json
{
  "refresh_token": "long-token-string"
}
```

**Flow:**
1. Find refresh token in DB
2. Validate not expired
3. Validate hash matches (to prevent tampering)
4. Generate new access token
5. Optionally: generate new refresh token (rotation)
6. Return new tokens

**Response (200):**
```json
{
  "data": {
    "access_token": "new-jwt-token",
    "refresh_token": "new-refresh-token"
  }
}
```

---

### POST /auth/logout

**Request:**
```json
{
  "refresh_token": "long-token-string"
}
```

**Flow:**
1. Delete refresh token from DB (optional but recommended)
2. Frontend deletes tokens from localStorage

**Response (200):**
```json
{
  "data": {
    "message": "Logged out successfully"
  }
}
```

---

### GET /auth/me

**Headers:**
```
Authorization: Bearer <access-token>
```

**Flow:**
1. Middleware validates JWT token
2. Extract user_id from token
3. Query user from DB
4. Return full profile with org info

**Response (200):**
```json
{
  "data": {
    "user": {
      "id": "user-uuid",
      "email": "user@example.com",
      "name": "John Doe"
    },
    "organization": {
      "id": "org-uuid",
      "name": "Company Inc"
    },
    "role": "manager"
  }
}
```

---

### POST /auth/forgot-password

**Request:**
```json
{
  "email": "user@example.com"
}
```

**Flow:**
1. Find user by email (don't reveal if exists for security)
2. Create verification token with type "reset_password"
3. Send reset link in email: `{frontend_url}/reset-password?token={token}`
4. Return generic success message

**Response (200):**
```json
{
  "data": {
    "message": "Check your email for reset instructions"
  }
}
```

---

### POST /auth/reset-password

**Request:**
```json
{
  "token": "verification-token-from-email",
  "new_password": "new-secure-password"
}
```

**Flow:**
1. Validate verification token exists, not expired, type is "reset_password"
2. Hash new password with bcrypt
3. Update user.password_hash
4. Delete verification token
5. Return success

**Response (200):**
```json
{
  "data": {
    "message": "Password reset successfully"
  }
}
```

---

### POST /auth/activate

**Request:**
```json
{
  "email": "user@example.com",
  "password": "account-password"
}
```

**Flow:**
1. Find pending user invite (not yet activated)
2. Validate password
3. Mark membership as `is_active = true`, set `activated_at`
4. Return login credentials

**Response (200):**
```json
{
  "data": {
    "message": "Account activated"
  }
}
```

---

### POST /auth/switch-org

**Headers:**
```
Authorization: Bearer <current-token>
```

**Request:**
```json
{
  "organization_id": "new-org-uuid"
}
```

**Flow:**
1. Validate user has membership in new org
2. Generate new JWT with new org_id
3. Optionally invalidate old token (security best practice)
4. Return new token

**Response (200):**
```json
{
  "data": {
    "access_token": "new-jwt-token"
  }
}
```

---

## Middleware: Auth Validation

All protected routes use `middleware.Auth(authService, handler)`.

**Process:**
```
1. Extract Authorization header: "Bearer <token>"
2. Validate JWT signature using JWT_SECRET
3. Check token not expired
4. Extract claims: user_id, org_id, role
5. Add to request context:
   - r.Context().Value("user_id")  // uuid.UUID
   - r.Context().Value("org_id")   // uuid.UUID
   - r.Context().Value("role")     // models.Role
6. Call wrapped handler
```

**Failures:**
- Missing header → 401 Unauthorized
- Invalid signature → 401 Unauthorized
- Expired token → 401 Unauthorized (frontend should refresh)

---

## Best Practices

### Frontend: Token Management

```typescript
// lib/api.ts
const api = axios.create({
  baseURL: process.env.VITE_API_URL || 'http://localhost:8080'
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  async (error) => {
    if (error.response?.status === 401) {
      // Token expired, try refresh
      const refreshToken = localStorage.getItem('refresh_token')
      const { data } = await api.post('/auth/refresh', { refresh_token: refreshToken })
      localStorage.setItem('access_token', data.data.access_token)
      
      // Retry original request
      return api(error.config)
    }
    return Promise.reject(error)
  }
)
```

### Backend: Password Hashing

**Always** use bcrypt:

```go
import "golang.org/x/crypto/bcrypt"

// Hash
hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

// Verify
err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(inputPassword))
```

**Never** store plaintext passwords.

### JWT Secret Management

- **Development**: Defaults to "dev-secret-change-in-production" (fine for local)
- **Production**: Set `JWT_SECRET` to strong random string (32+ chars)
- **Rotation**: No built-in key rotation; manage separately

---

## Security Considerations

1. **HTTPS Only** — JWT tokens in URLs/headers must be encrypted in transit
2. **localStorage vs Cookies** — localStorage used here (accessible to XSS), consider httpOnly cookies for higher security
3. **Token Expiration** — Short-lived access tokens + refresh tokens for better security
4. **Refresh Token Rotation** — Each refresh invalidates old token
5. **Rate Limiting** — Protect login/refresh endpoints from brute force
6. **Password Requirements** — Enforce minimum 8 chars, recommend 12+
7. **Email Verification** — Verify emails before activating accounts

---

**Next**: [[06-Middleware]] for request processing details, or [[15-Development-Setup]] for local testing.
