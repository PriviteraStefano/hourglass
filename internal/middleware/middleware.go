package middleware

import (
	"context"
	"net/http"
	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type contextKey string

const (
	UserIDKey         contextKey = "userID"
	OrganizationIDKey contextKey = "organizationID"
	RoleKey           contextKey = "role"
	EmailKey          contextKey = "email"
)

func Auth(authService *auth.Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil {
			api.RespondWithError(w, http.StatusUnauthorized, "missing access token")
			return
		}

		claims, err := authService.ValidateToken(cookie.Value)
		if err != nil {
			api.RespondWithError(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, OrganizationIDKey, claims.OrganizationID)
		ctx = context.WithValue(ctx, RoleKey, claims.Role)
		ctx = context.WithValue(ctx, EmailKey, claims.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func RequireRole(roles []string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userRole := r.Context().Value(RoleKey).(string)

		allowed := false
		for _, role := range roles {
			if userRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			api.RespondWithError(w, http.StatusForbidden, "insufficient permissions")
			return
		}

		next.ServeHTTP(w, r)
	}
}

func GetUserID(ctx context.Context) uuid.UUID {
    if userID, ok := ctx.Value(UserIDKey).(uuid.UUID); ok {
        return userID
    }
    return uuid.UUID{}
}
func GetOrganizationID(ctx context.Context) uuid.UUID {
    if orgID, ok := ctx.Value(OrganizationIDKey).(uuid.UUID); ok {
        return orgID
    }
    return uuid.UUID{}
}

func GetRole(ctx context.Context) string {
	if role, ok := ctx.Value(RoleKey).(string); ok {
		return role
	}
	return ""
}

func GetEmail(ctx context.Context) string {
	if email, ok := ctx.Value(EmailKey).(string); ok {
		return email
	}
	return ""
}
