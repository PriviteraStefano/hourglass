package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type InvitationHandler struct {
	sdb *db.SurrealDB
}

func NewInvitationHandler(sdb *db.SurrealDB) *InvitationHandler {
	return &InvitationHandler{sdb: sdb}
}

type CreateInvitationRequest struct {
	OrganizationID string `json:"organization_id"`
	Email          string `json:"email,omitempty"`
	ExpiresInDays  int    `json:"expires_in_days,omitempty"`
}

type InvitationResponse struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	Token          string    `json:"token"`
	Link           string    `json:"link"`
	Email          string    `json:"email,omitempty"`
	Status         string    `json:"status"`
	ExpiresAt      time.Time `json:"expires_at"`
	OrganizationID string    `json:"organization_id"`
}

func generateInviteCode() string {
	chars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	rand.Read(code)
	for i := range code {
		code[i] = chars[int(code[i])%len(chars)]
	}
	return string(code)
}

func generateToken() string {
	token := make([]byte, 16)
	rand.Read(token)
	return base64.URLEncoding.EncodeToString(token)
}

func (h *InvitationHandler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateInvitationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrganizationID == "" {
		api.RespondWithError(w, http.StatusBadRequest, "organization_id is required")
		return
	}

	code := generateInviteCode()
	token := generateToken()
	expiresInDays := req.ExpiresInDays
	if expiresInDays <= 0 {
		expiresInDays = 7
	}
	expiresAt := time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour)

	invitationQuery := `
		CREATE invitations SET
			organization_id = $organization_id,
			code = $code,
			invite_token = $invite_token,
			email = $email,
			status = 'pending',
			expires_at = $expires_at,
			created_at = $created_at
	`
	invitationVars := map[string]interface{}{
		"organization_id": req.OrganizationID,
		"code":            code,
		"invite_token":    token,
		"email":           req.Email,
		"status":          "pending",
		"expires_at":      expiresAt,
		"created_by":      "system",
		"created_at":      time.Now(),
	}

	results, err := h.sdb.Query(ctx, invitationQuery, invitationVars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create invitation: "+err.Error())
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create invitation: no result")
		return
	}

	var invitations []interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if json.Unmarshal(resultBytes, &invitations) != nil || len(invitations) == 0 {
		return
	}

	if invitation, ok := invitations[0].(map[string]interface{}); ok {
		idStr := ""
		if id, ok := invitation["id"].(map[string]interface{}); ok {
			if s, ok := id["ID"].(string); ok {
				idStr = s
			}
		}
		link := "https://app.example.com/invite/" + token
		api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
			"id":              idStr,
			"code":            code,
			"token":           token,
			"link":            link,
			"email":           req.Email,
			"status":          "pending",
			"expires_at":      expiresAt,
			"organization_id": req.OrganizationID,
		})
		return
	}

	api.RespondWithError(w, http.StatusInternalServerError, "failed to create invitation")
}

func (h *InvitationHandler) ValidateCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := strings.TrimPrefix(r.PathValue("code"), "/")

	query := `SELECT * FROM invitations WHERE code = $code LIMIT 1`
	vars := map[string]interface{}{"code": code}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil || len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}

	var invitations []interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if json.Unmarshal(resultBytes, &invitations) == nil && len(invitations) > 0 {
		if invitation, ok := invitations[0].(map[string]interface{}); ok {
			if status, ok := invitation["status"].(string); ok && status == "expired" {
				api.RespondWithError(w, http.StatusGone, "invitation has expired")
				return
			}
			if expiresAt, ok := invitation["expires_at"].(time.Time); ok && expiresAt.Before(time.Now()) {
				api.RespondWithError(w, http.StatusGone, "invitation has expired")
				return
			}
			api.RespondWithJSON(w, http.StatusOK, invitation)
			return
		}
	}

	api.RespondWithError(w, http.StatusInternalServerError, "failed to validate invitation")
}

func (h *InvitationHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimPrefix(r.PathValue("token"), "/")

	query := `SELECT * FROM invitations WHERE invite_token = $invite_token LIMIT 1`
	vars := map[string]interface{}{"invite_token": token}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil || len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}

	var invitations []interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if json.Unmarshal(resultBytes, &invitations) == nil && len(invitations) > 0 {
		if invitation, ok := invitations[0].(map[string]interface{}); ok {
			if status, ok := invitation["status"].(string); ok && status == "expired" {
				api.RespondWithError(w, http.StatusGone, "invitation has expired")
				return
			}
			if expiresAt, ok := invitation["expires_at"].(time.Time); ok && expiresAt.Before(time.Now()) {
				api.RespondWithError(w, http.StatusGone, "invitation has expired")
				return
			}
			api.RespondWithJSON(w, http.StatusOK, invitation)
			return
		}
	}

	api.RespondWithError(w, http.StatusInternalServerError, "failed to validate invitation")
}

func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Token    string `json:"token"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.Email == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "token, email, and password are required")
		return
	}

	if len(req.Password) < 8 {
		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	query := `SELECT * FROM invitations WHERE invite_token = $invite_token LIMIT 1`
	vars := map[string]interface{}{"invite_token": req.Token}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil || len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}

	var invitations []interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if json.Unmarshal(resultBytes, &invitations) != nil || len(invitations) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to process invitation")
		return
	}

	invitation, ok := invitations[0].(map[string]interface{})
	if !ok {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to process invitation")
		return
	}

	if status, ok := invitation["status"].(string); ok && status != "pending" {
		api.RespondWithError(w, http.StatusGone, "invitation already used")
		return
	}

	if expiresAt, ok := invitation["expires_at"].(time.Time); ok && expiresAt.Before(time.Now()) {
		api.RespondWithError(w, http.StatusGone, "invitation has expired")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "invitation valid, user creation would proceed here",
		"organization_id": invitation["organization_id"],
	})
}
