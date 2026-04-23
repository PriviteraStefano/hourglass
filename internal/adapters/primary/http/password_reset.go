package http

import (
	"encoding/json"
	"net/http"

	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	passwordresetsvc "github.com/stefanoprivitera/hourglass/internal/core/services/password_reset"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

type PasswordResetHandler struct {
	service *passwordresetsvc.Service
}

func NewPasswordResetHandler(service *passwordresetsvc.Service) *PasswordResetHandler {
	return &PasswordResetHandler{service: service}
}

type RequestResetRequest struct {
	Identifier string `json:"identifier"`
}

type VerifyResetRequest struct {
	Identifier string `json:"identifier"`
	Code       string `json:"code"`
	Password   string `json:"password"`
}

func (h *PasswordResetHandler) Request(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RequestResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" {
		api.RespondWithError(w, http.StatusBadRequest, "identifier is required")
		return
	}

	code, expiresAt, err := h.service.Request(ctx, req.Identifier)
	if err != nil {
		if err == password_reset.ErrUserNotFound {
			api.RespondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to request reset")
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

	var req VerifyResetRequest
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

	err := h.service.Verify(ctx, req.Identifier, req.Code, req.Password)
	if err != nil {
		if err == password_reset.ErrUserNotFound {
			api.RespondWithError(w, http.StatusNotFound, "user not found")
			return
		}
		if err == password_reset.ErrResetExpired {
			api.RespondWithError(w, http.StatusGone, "reset code expired or not found")
			return
		}
		if err == password_reset.ErrInvalidCode {
			api.RespondWithError(w, http.StatusUnauthorized, "invalid code")
			return
		}
		api.RespondWithError(w, http.StatusInternalServerError, "failed to reset password")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
		"message": "password reset successful",
	})
}
