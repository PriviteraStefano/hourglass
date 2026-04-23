package handlers

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/pkg/api"

	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type PasswordResetHandler struct {
	db          *sdb.DB
	authService *auth.Service
}

func NewPasswordResetHandler(db *sdb.DB, authService *auth.Service) *PasswordResetHandler {
	return &PasswordResetHandler{db: db, authService: authService}
}

type PasswordResetRequest struct {
	Identifier string `json:"identifier"`
}

type PasswordResetVerify struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Password   string `json:"password"`
}

func generateResetCode() string {
	code := make([]byte, 3)
	rand.Read(code)
	result := ""
	for _, b := range code {
		result += strconv.Itoa(int(b) % 10)
	}
	return result
}

func (h *PasswordResetHandler) Request(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" {
		api.RespondWithError(w, http.StatusBadRequest, "email or username is required")
		return
	}

	user, userID, err := h.findUserByIdentifier(ctx, req.Identifier)
	if err != nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}
	if user == nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	code := generateResetCode()
	codeHash := auth.HashRefreshToken(code)
	expiresAt := time.Now().Add(2 * time.Hour)

	reset := &surrealdb.SurrealPasswordReset{
		UserID:    models.NewRecordID("users", userID),
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	_, err = sdb.Create[surrealdb.SurrealPasswordReset](ctx, h.db, models.Table("password_resets"), reset)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create reset request")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":    "reset code sent",
		"code":       code,
		"expires_at": expiresAt,
	})
}

func (h *PasswordResetHandler) Verify(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req PasswordResetVerify
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" || req.Code == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "identifier, code, and password are required")
		return
	}

	if len(req.Password) < 8 {
		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	user, userID, err := h.findUserByIdentifier(ctx, req.Identifier)
	if err != nil || user == nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	reset, err := h.findActiveReset(ctx, userID)
	if err != nil || reset == nil {
		api.RespondWithError(w, http.StatusGone, "reset code expired or not found")
		return
	}

	if auth.HashRefreshToken(req.Code) != reset.CodeHash {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid code")
		return
	}

	newPasswordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	userRecordID := models.NewRecordID("users", userID)
	_, err = sdb.Merge[map[string]interface{}](ctx, h.db, userRecordID, map[string]interface{}{
		"password_hash": newPasswordHash,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	usedAt := time.Now()
	_, _ = sdb.Merge[surrealdb.SurrealPasswordReset](ctx, h.db, reset.ID, map[string]interface{}{
		"used_at": usedAt,
	})

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "password reset successful",
	})
}

func (h *PasswordResetHandler) findUserByIdentifier(ctx context.Context, identifier string) (map[string]interface{}, string, error) {
	results, err := sdb.Query[[]map[string]interface{}](ctx, h.db,
		"SELECT * FROM users WHERE email = $identifier OR username = $identifier LIMIT 1",
		map[string]interface{}{"identifier": identifier})
	if err != nil || results == nil || len(*results) == 0 {
		return nil, "", err
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return nil, "", nil
	}
	user := resultItems[0]

	userID := extractRecordID(user)
	if userID == "" {
		return nil, "", nil
	}

	return user, userID, nil
}

func (h *PasswordResetHandler) findActiveReset(ctx context.Context, userID string) (*surrealdb.SurrealPasswordReset, error) {
	results, err := sdb.Query[[]surrealdb.SurrealPasswordReset](ctx, h.db,
		`SELECT * FROM password_resets WHERE user_id = $user_id AND expires_at > $now AND used_at = NONE LIMIT 1`,
		map[string]interface{}{
			"user_id": models.NewRecordID("users", userID),
			"now":     time.Now(),
		})
	if err != nil {
		return nil, err
	}
	if results == nil || len(*results) == 0 {
		return nil, nil
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		return nil, nil
	}
	return &resultItems[0], nil
}

func extractRecordID(data map[string]interface{}) string {
	if id, ok := data["id"].(map[string]interface{}); ok {
		if idStr, ok := id["ID"].(string); ok {
			return idStr
		}
	}
	return ""
}
