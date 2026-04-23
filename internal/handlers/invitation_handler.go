package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/stefanoprivitera/hourglass/internal/adapters/secondary/surrealdb"
	"github.com/stefanoprivitera/hourglass/pkg/api"

	sdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type InvitationHandler struct {
	db *sdb.DB
}

func NewInvitationHandler(db *sdb.DB) *InvitationHandler {
	return &InvitationHandler{db: db}
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

	invitation := &surrealdb.SurrealInvitation{
		OrganizationID: models.NewRecordID("organizations", req.OrganizationID),
		Code:           code,
		InviteToken:    token,
		Email:          req.Email,
		Status:         "pending",
		ExpiresAt:      expiresAt,
		CreatedBy:      "system",
		CreatedAt:      time.Now(),
	}

	created, err := sdb.Create[surrealdb.SurrealInvitation](ctx, h.db, models.Table("invitations"), invitation)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create invitation: "+err.Error())
		return
	}

	link := "https://app.example.com/invite/" + token
	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":              recordIDToString(created.ID),
		"code":            code,
		"token":           token,
		"link":            link,
		"email":           req.Email,
		"status":          "pending",
		"expires_at":      expiresAt,
		"organization_id": req.OrganizationID,
	})
}

func (h *InvitationHandler) ValidateCode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	code := strings.TrimPrefix(r.PathValue("code"), "/")

	results, err := sdb.Query[[]surrealdb.SurrealInvitation](ctx, h.db,
		"SELECT * FROM invitations WHERE code = $code LIMIT 1",
		map[string]interface{}{"code": code})
	if err != nil || results == nil || len(*results) == 0 {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}

	invitation := resultItems[0]
	if invitation.Status == "expired" || invitation.ExpiresAt.Before(time.Now()) {
		api.RespondWithError(w, http.StatusGone, "invitation has expired")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":              recordIDToString(invitation.ID),
		"code":            invitation.Code,
		"email":           invitation.Email,
		"status":          invitation.Status,
		"expires_at":      invitation.ExpiresAt,
		"organization_id": recordIDToString(invitation.OrganizationID),
	})
}

func (h *InvitationHandler) ValidateToken(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimPrefix(r.PathValue("token"), "/")

	results, err := sdb.Query[[]surrealdb.SurrealInvitation](ctx, h.db,
		"SELECT * FROM invitations WHERE invite_token = $invite_token LIMIT 1",
		map[string]interface{}{"invite_token": token})
	if err != nil || results == nil || len(*results) == 0 {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}

	invitation := resultItems[0]
	if invitation.Status == "expired" || invitation.ExpiresAt.Before(time.Now()) {
		api.RespondWithError(w, http.StatusGone, "invitation has expired")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"id":              recordIDToString(invitation.ID),
		"code":            invitation.Code,
		"email":           invitation.Email,
		"status":          invitation.Status,
		"expires_at":      invitation.ExpiresAt,
		"organization_id": recordIDToString(invitation.OrganizationID),
	})
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

	results, err := sdb.Query[[]surrealdb.SurrealInvitation](ctx, h.db,
		"SELECT * FROM invitations WHERE invite_token = $invite_token LIMIT 1",
		map[string]interface{}{"invite_token": req.Token})
	if err != nil || results == nil || len(*results) == 0 {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}
	resultItems := (*results)[0].Result
	if len(resultItems) == 0 {
		api.RespondWithError(w, http.StatusNotFound, "invitation not found")
		return
	}

	invitation := resultItems[0]
	if invitation.Status != "pending" {
		api.RespondWithError(w, http.StatusGone, "invitation already used")
		return
	}

	if invitation.ExpiresAt.Before(time.Now()) {
		api.RespondWithError(w, http.StatusGone, "invitation has expired")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "invitation valid, user creation would proceed here",
		"organization_id": recordIDToString(invitation.OrganizationID),
	})
}

func recordIDToString(id models.RecordID) string {
	switch v := id.ID.(type) {
	case string:
		return v
	default:
		return ""
	}
}
