package handlers

import (
	"crypto/rand"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type PasswordResetHandler struct {
	sdb         *db.SurrealDB
	authService *auth.Service
}

func NewPasswordResetHandler(sdb *db.SurrealDB, authService *auth.Service) *PasswordResetHandler {
	return &PasswordResetHandler{sdb: sdb, authService: authService}
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

	userQuery := `SELECT * FROM users WHERE email = $identifier OR username = $identifier LIMIT 1`
	userVars := map[string]interface{}{"identifier": req.Identifier}

	userResults, err := h.sdb.Query(ctx, userQuery, userVars)
	if err != nil || len(*userResults) == 0 || (*userResults)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	var users []interface{}
	userBytes, _ := json.Marshal((*userResults)[0].Result)
	if json.Unmarshal(userBytes, &users) != nil || len(users) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to find user")
		return
	}

	user, ok := users[0].(map[string]interface{})
	if !ok {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse user")
		return
	}

	var userID string
	if id, ok := user["id"].(map[string]interface{}); ok {
		if idStr, ok := id["ID"].(string); ok {
			userID = idStr
		}
	}

	if userID == "" {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to identify user")
		return
	}

	code := generateResetCode()
	codeHash := auth.HashRefreshToken(code)
	expiresAt := time.Now().Add(2 * time.Hour)

	resetQuery := `
		CREATE password_resets SET
			user_id = $user_id,
			code_hash = $code_hash,
			expires_at = $expires_at,
			created_at = $created_at
	`
	resetVars := map[string]interface{}{
		"user_id":    userID,
		"code_hash":  codeHash,
		"expires_at": expiresAt,
		"created_at": time.Now(),
	}

	_, err = h.sdb.Query(ctx, resetQuery, resetVars)
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

	userQuery := `SELECT * FROM users WHERE email = $identifier OR username = $identifier LIMIT 1`
	userVars := map[string]interface{}{"identifier": req.Identifier}

	userResults, err := h.sdb.Query(ctx, userQuery, userVars)
	if err != nil || len(*userResults) == 0 || (*userResults)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	var users []interface{}
	userBytes, _ := json.Marshal((*userResults)[0].Result)
	if json.Unmarshal(userBytes, &users) != nil || len(users) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to find user")
		return
	}

	user, ok := users[0].(map[string]interface{})
	if !ok {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse user")
		return
	}

	var userID string
	if id, ok := user["id"].(map[string]interface{}); ok {
		if idStr, ok := id["ID"].(string); ok {
			userID = idStr
		}
	}

	if userID == "" {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to identify user")
		return
	}

	resetQuery := `
		SELECT * FROM password_resets
		WHERE user_id = $user_id
		AND expires_at > $now
		AND used_at = NONE
		LIMIT 1
	`
	resetVars := map[string]interface{}{
		"user_id": userID,
		"now":     time.Now(),
	}

	resetResults, err := h.sdb.Query(ctx, resetQuery, resetVars)
	if err != nil || len(*resetResults) == 0 || (*resetResults)[0].Result == nil {
		api.RespondWithError(w, http.StatusGone, "reset code expired or not found")
		return
	}

	var resets []interface{}
	resetBytes, _ := json.Marshal((*resetResults)[0].Result)
	if json.Unmarshal(resetBytes, &resets) != nil || len(resets) == 0 {
		api.RespondWithError(w, http.StatusGone, "reset code expired or not found")
		return
	}

	reset, ok := resets[0].(map[string]interface{})
	if !ok {
		api.RespondWithError(w, http.StatusGone, "reset code expired or not found")
		return
	}

	codeHash, ok := reset["code_hash"].(string)
	if !ok {
		api.RespondWithError(w, http.StatusGone, "reset code expired or not found")
		return
	}

	if auth.HashRefreshToken(req.Code) != codeHash {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid code")
		return
	}

	newPasswordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	updateQuery := `UPDATE $user_id SET password_hash = $password_hash`
	_, err = h.sdb.Query(ctx, updateQuery, map[string]interface{}{
		"user_id":       userID,
		"password_hash": newPasswordHash,
	})
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to update password")
		return
	}

	markUsedQuery := `UPDATE $reset_id SET used_at = $used_at`
	_, _ = h.sdb.Query(ctx, markUsedQuery, map[string]interface{}{
		"reset_id": reset["id"],
		"used_at":  time.Now(),
	})

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "password reset successful",
	})
}
