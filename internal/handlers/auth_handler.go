package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/auth"
	"github.com/stefanoprivitera/hourglass/internal/db"
	"github.com/stefanoprivitera/hourglass/internal/middleware"
	"github.com/stefanoprivitera/hourglass/pkg/api"
)

var _ = uuid.UUID{} // prevent unused import error

type AuthHandler struct {
	sdb         *db.SurrealDB
	authService *auth.Service
}

func NewAuthHandler(sdb *db.SurrealDB, authService *auth.Service) *AuthHandler {
	return &AuthHandler{
		sdb:         sdb,
		authService: authService,
	}
}

type RegisterRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	Firstname   string `json:"firstname"`
	Lastname    string `json:"lastname"`
	Name        string `json:"name"`
	Password    string `json:"password"`
	OrgName     string `json:"organization_name"`
	InviteToken string `json:"invite_token,omitempty"`
}

type BootstrapRequest struct {
	OrgName   string `json:"org_name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Password  string `json:"password"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "email and password are required")
		return
	}

	if req.Username != "" && len(req.Username) < 3 {
		api.RespondWithError(w, http.StatusBadRequest, "username must be at least 3 characters")
		return
	}

	if len(req.Password) < 8 {
		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	checkQuery := `SELECT count() FROM users WHERE email = $email GROUP ALL`
	checkVars := map[string]interface{}{"email": req.Email}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err == nil && len(*checkResult) > 0 && (*checkResult)[0].Result != nil {
		var counts []map[string]interface{}
		checkBytes, _ := json.Marshal((*checkResult)[0].Result)
		if json.Unmarshal(checkBytes, &counts) == nil && len(counts) > 0 {
			if count, ok := counts[0]["count"].(float64); ok && count > 0 {
				api.RespondWithError(w, http.StatusConflict, "email already registered")
				return
			}
		}
	}

	if req.Username != "" {
		usernameCheckQuery := `SELECT count() FROM users WHERE username = $username GROUP ALL`
		usernameCheckVars := map[string]interface{}{"username": req.Username}
		usernameCheckResult, err := h.sdb.Query(ctx, usernameCheckQuery, usernameCheckVars)
		if err == nil && len(*usernameCheckResult) > 0 && (*usernameCheckResult)[0].Result != nil {
			var counts []map[string]interface{}
			checkBytes, _ := json.Marshal((*usernameCheckResult)[0].Result)
			if json.Unmarshal(checkBytes, &counts) == nil && len(counts) > 0 {
				if count, ok := counts[0]["count"].(float64); ok && count > 0 {
					api.RespondWithError(w, http.StatusConflict, "username already taken")
					return
				}
			}
		}
	}

	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	now := time.Now()

	// Compute display name from firstname/lastname if not provided
	if req.Name == "" && (req.Firstname != "" || req.Lastname != "") {
		req.Name = strings.TrimSpace(req.Firstname + " " + req.Lastname)
	}

	userData := map[string]interface{}{
		"email":         req.Email,
		"password_hash": passwordHash,
		"is_active":     true,
		"created_at":    now,
		"updated_at":    now,
	}
	if req.Name != "" {
		userData["name"] = req.Name
	}
	if req.Firstname != "" {
		userData["firstname"] = req.Firstname
	}
	if req.Lastname != "" {
		userData["lastname"] = req.Lastname
	}
	if req.Username != "" {
		userData["username"] = req.Username
	}

	// Build the SET clause dynamically
	setParts := []string{}
	vars := map[string]interface{}{}
	for key, value := range userData {
		setParts = append(setParts, key+" = $"+key)
		vars[key] = value
	}
	userQuery := `CREATE users SET ` + strings.Join(setParts, ", ")

	txQueries := []string{}
	txVars := []map[string]interface{}{}

	txQueries = append(txQueries, userQuery)
	txVars = append(txVars, vars)

	if req.OrgName != "" {
		orgSlug := generateSlug(req.OrgName)
		orgQuery := `
			CREATE organizations SET
				name = $name,
				slug = $slug,
				description = $description,
				financial_cutoff_days = 7,
				financial_cutoff_config = { cutoff_day_of_month: 28, grace_days: 7 },
				created_at = $now,
				updated_at = $now
		`
		txQueries = append(txQueries, orgQuery)
		txVars = append(txVars, map[string]interface{}{
			"name":        req.OrgName,
			"slug":        orgSlug,
			"description": "Organization created during registration",
			"now":         now,
		})
	}

	var createdUser map[string]interface{}
	for i, query := range txQueries {
		results, err := h.sdb.Query(ctx, query, txVars[i])
		if err != nil {
			slog.Error("failed to execute query", "query", query, "err", err)
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create account")
			return
		}

		if i == len(txQueries)-1 {
			var users []interface{}
			resultBytes, _ := json.Marshal((*results)[0].Result)
			if json.Unmarshal(resultBytes, &users) == nil && len(users) > 0 {
				if user, ok := users[0].(map[string]interface{}); ok {
					createdUser = user
				}
			}
		}
	}

	if createdUser == nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"id":      createdUser["id"],
		"email":   createdUser["email"],
		"name":    createdUser["name"],
		"message": "registration successful",
	})
}

func (h *AuthHandler) Bootstrap(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req BootstrapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrgName == "" || req.Email == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "org_name, email, and password are required")
		return
	}

	if req.Username != "" && len(req.Username) < 3 {
		api.RespondWithError(w, http.StatusBadRequest, "username must be at least 3 characters")
		return
	}

	if len(req.Password) < 8 {
		api.RespondWithError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	checkQuery := `SELECT count() FROM users WHERE email = $email GROUP ALL`
	checkVars := map[string]interface{}{"email": req.Email}
	checkResult, err := h.sdb.Query(ctx, checkQuery, checkVars)
	if err == nil && len(*checkResult) > 0 && (*checkResult)[0].Result != nil {
		var counts []map[string]interface{}
		checkBytes, _ := json.Marshal((*checkResult)[0].Result)
		if json.Unmarshal(checkBytes, &counts) == nil && len(counts) > 0 {
			if count, ok := counts[0]["count"].(float64); ok && count > 0 {
				api.RespondWithError(w, http.StatusConflict, "email already registered")
				return
			}
		}
	}

	if req.Username != "" {
		usernameCheckQuery := `SELECT count() FROM users WHERE username = $username GROUP ALL`
		usernameCheckVars := map[string]interface{}{"username": req.Username}
		usernameCheckResult, err := h.sdb.Query(ctx, usernameCheckQuery, usernameCheckVars)
		if err == nil && len(*usernameCheckResult) > 0 && (*usernameCheckResult)[0].Result != nil {
			var counts []map[string]interface{}
			checkBytes, _ := json.Marshal((*usernameCheckResult)[0].Result)
			if json.Unmarshal(checkBytes, &counts) == nil && len(counts) > 0 {
				if count, ok := counts[0]["count"].(float64); ok && count > 0 {
					api.RespondWithError(w, http.StatusConflict, "username already taken")
					return
				}
			}
		}
	}

	passwordHash, err := h.authService.HashPassword(req.Password)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	now := time.Now()
	orgSlug := generateSlug(req.OrgName)
	displayName := ""
	if req.Firstname != "" || req.Lastname != "" {
		displayName = strings.TrimSpace(req.Firstname + " " + req.Lastname)
	}

	userData := map[string]interface{}{
		"email":         req.Email,
		"firstname":     req.Firstname,
		"lastname":      req.Lastname,
		"name":          displayName,
		"password_hash": passwordHash,
		"is_active":     true,
		"created_at":    now,
		"updated_at":    now,
	}
	if req.Username != "" {
		userData["username"] = req.Username
	}

	setParts := []string{}
	vars := map[string]interface{}{}
	for key, value := range userData {
		setParts = append(setParts, key+" = $"+key)
		vars[key] = value
	}
	userQuery := `CREATE users SET ` + strings.Join(setParts, ", ")

	orgQuery := `
		CREATE organizations SET
			name = $name,
			slug = $slug,
			description = $description,
			financial_cutoff_days = 7,
			financial_cutoff_config = { cutoff_day_of_month: 28, grace_days: 7 },
			created_at = $now,
			updated_at = $now
	`

	txQueries := []string{orgQuery, userQuery}
	txVars := []map[string]interface{}{
		{"name": req.OrgName, "slug": orgSlug, "description": "Bootstrap organization", "now": now},
		vars,
	}

	var orgID string
	var createdUser map[string]interface{}
	for i, query := range txQueries {
		results, err := h.sdb.Query(ctx, query, txVars[i])
		if err != nil {
			slog.Error("failed to execute query", "query", query, "err", err)
			api.RespondWithError(w, http.StatusInternalServerError, "failed to create bootstrap organization")
			return
		}

		var items []interface{}
		resultBytes, _ := json.Marshal((*results)[0].Result)
		if json.Unmarshal(resultBytes, &items) == nil && len(items) > 0 {
			if item, ok := items[0].(map[string]interface{}); ok {
				if i == 0 {
					if id, ok := item["id"].(map[string]interface{}); ok {
						if idStr, ok := id["ID"].(string); ok {
							if idx := strings.LastIndex(idStr, ":"); idx >= 0 {
								orgID = idStr[idx+1:]
							} else {
								orgID = idStr
							}
						}
					}
				} else {
					createdUser = item
				}
			}
		}
	}

	if createdUser == nil || orgID == "" {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to create bootstrap organization")
		return
	}

	userIDraw, _ := createdUser["id"].(map[string]interface{})
	var userID string
	if userIDraw != nil {
		if id, ok := userIDraw["ID"].(string); ok {
			if idx := strings.LastIndex(id, ":"); idx >= 0 {
				userID = id[idx+1:]
			}
		}
	}

	tokenUserID := uuid.NewMD5(uuid.Nil, []byte(userID))
	token, err := h.authService.GenerateToken(tokenUserID, uuid.Nil, "admin", req.Email)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	api.RespondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":       userID,
			"email":    req.Email,
			"username": req.Username,
			"name":     displayName,
		},
		"organization": map[string]interface{}{
			"id":   orgID,
			"name": req.OrgName,
		},
	})
}

type SurrealLoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type SurrealLoginResponse struct {
	Token        string      `json:"token"`
	RefreshToken string      `json:"refresh_token"`
	User         interface{} `json:"user"`
	ExpiresAt    time.Time   `json:"expires_at"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req SurrealLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Identifier == "" || req.Password == "" {
		api.RespondWithError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}

	query := `SELECT * FROM users WHERE email = $identifier OR username = $identifier LIMIT 1`
	vars := map[string]interface{}{"identifier": req.Identifier}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	var users []interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	userMap := make(map[string]interface{})
	if err := json.Unmarshal(resultBytes, &users); err != nil || len(users) == 0 {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if user, ok := users[0].(map[string]interface{}); ok {
		userMap = user
	}

	passwordHash, ok := userMap["password_hash"].(string)
	if !ok {
		api.RespondWithError(w, http.StatusInternalServerError, "invalid user data")
		return
	}

	if !h.authService.CheckPassword(req.Password, passwordHash) {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	isActive, _ := userMap["is_active"].(bool)
	if !isActive {
		api.RespondWithError(w, http.StatusForbidden, "account is deactivated")
		return
	}

	var userID string
	userIDraw, _ := userMap["id"].(map[string]interface{})
	if userIDraw != nil {
		if id, ok := userIDraw["ID"].(string); ok {
			if idx := strings.LastIndex(id, ":"); idx >= 0 {
				userID = id[idx+1:]
			} else {
				userID = id
			}
		}
	}

	userEmail, _ := userMap["email"].(string)
	userName, _ := userMap["name"].(string)

	// SurrealDB uses string IDs, not UUIDs. Generate a deterministic UUID based on userID for JWT
	tokenUserID := uuid.NewMD5(uuid.Nil, []byte(userID))
	_ = tokenUserID // Used below

	var role string = "employee"
	var orgID uuid.UUID

	membershipQuery := `
		SELECT *, org_id as org_id FROM unit_memberships
		WHERE user_id = $user_id
		AND is_primary = true
		LIMIT 1
	`
	membershipVars := map[string]interface{}{"user_id": userID}
	membershipResults, err := h.sdb.Query(ctx, membershipQuery, membershipVars)

	if err == nil && len(*membershipResults) > 0 && (*membershipResults)[0].Result != nil {
		var memberships []map[string]interface{}
		membershipBytes, _ := json.Marshal((*membershipResults)[0].Result)
		if json.Unmarshal(membershipBytes, &memberships) == nil && len(memberships) > 0 {
			membership := memberships[0]
			if oid, ok := membership["org_id"].(string); ok {
				orgID, _ = uuid.Parse(oid)
			}
			if r, ok := membership["role"].(string); ok {
				role = r
			}
		}
	}

	token, err := h.authService.GenerateToken(tokenUserID, orgID, role, userEmail)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	refreshToken := generateRefreshToken()
	refreshHash := auth.HashRefreshToken(refreshToken)
	refreshExpiresAt := time.Now().Add(7 * 24 * time.Hour)

	refreshQuery := `
		CREATE refresh_tokens SET
			user_id = $user_id,
			token_hash = $token_hash,
			expires_at = $expires_at,
			created_at = $now
	`
	refreshVars := map[string]interface{}{
		"user_id":    userID,
		"token_hash": refreshHash,
		"expires_at": refreshExpiresAt,
		"now":        time.Now(),
	}
	h.sdb.Query(ctx, refreshQuery, refreshVars)

	expiresAt := time.Now().Add(15 * time.Minute)
	api.RespondWithJSON(w, http.StatusOK, SurrealLoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: map[string]interface{}{
			"id":     userID,
			"email":  userEmail,
			"name":   userName,
			"role":   role,
			"org_id": orgID,
		},
		ExpiresAt: expiresAt,
	})
}

func (h *AuthHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := middleware.GetUserID(ctx)

	query := `SELECT id, email, name, is_active, created_at, updated_at FROM $user_id`
	vars := map[string]interface{}{"user_id": userID.String()}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch profile")
		return
	}

	if len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusNotFound, "user not found")
		return
	}

	var users []map[string]interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &users); err != nil || len(users) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to parse profile")
		return
	}

	api.RespondWithJSON(w, http.StatusOK, users[0])
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.RespondWithError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.RefreshToken == "" {
		api.RespondWithError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	refreshHash := auth.HashRefreshToken(req.RefreshToken)

	query := `
		SELECT * FROM refresh_tokens
		WHERE token_hash = $token_hash
		AND expires_at > $now
		AND revoked_at = NONE
		LIMIT 1
	`
	vars := map[string]interface{}{
		"token_hash": refreshHash,
		"now":        time.Now(),
	}

	results, err := h.sdb.Query(ctx, query, vars)
	if err != nil || len(*results) == 0 || (*results)[0].Result == nil {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}

	var tokens []map[string]interface{}
	resultBytes, _ := json.Marshal((*results)[0].Result)
	if err := json.Unmarshal(resultBytes, &tokens); err != nil || len(tokens) == 0 {
		api.RespondWithError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	token := tokens[0]
	userID, _ := token["user_id"].(string)

	userQuery := `SELECT id, email, name FROM $user_id`
	userVars := map[string]interface{}{"user_id": userID}
	userResults, err := h.sdb.Query(ctx, userQuery, userVars)
	if err != nil || len(*userResults) == 0 {
		api.RespondWithError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}

	var users []map[string]interface{}
	userBytes, _ := json.Marshal((*userResults)[0].Result)
	if json.Unmarshal(userBytes, &users) == nil && len(users) > 0 {
		user := users[0]
		userEmail, _ := user["email"].(string)
		userName, _ := user["name"].(string)

		membershipQuery := `
			SELECT org_id, role FROM unit_memberships
			WHERE user_id = $user_id
			AND is_primary = true
			LIMIT 1
		`
		membershipVars := map[string]interface{}{"user_id": userID}
		membershipResults, _ := h.sdb.Query(ctx, membershipQuery, membershipVars)

		var orgID string
		var role string = "employee"

		if len(*membershipResults) > 0 && (*membershipResults)[0].Result != nil {
			var memberships []map[string]interface{}
			membershipBytes, _ := json.Marshal((*membershipResults)[0].Result)
			if json.Unmarshal(membershipBytes, &memberships) == nil && len(memberships) > 0 {
				membership := memberships[0]
				if oid, ok := membership["org_id"].(string); ok {
					orgID = oid
				}
				if r, ok := membership["role"].(string); ok {
					role = r
				}
			}
		}

		newToken, err := h.authService.GenerateToken(uuid.MustParse(userID), uuid.MustParse(orgID), role, userEmail)
		if err != nil {
			api.RespondWithError(w, http.StatusInternalServerError, "failed to generate token")
			return
		}

		expiresAt := time.Now().Add(15 * time.Minute)
		api.RespondWithJSON(w, http.StatusOK, map[string]interface{}{
			"token":      newToken,
			"expires_at": expiresAt,
			"user": map[string]interface{}{
				"id":     userID,
				"email":  userEmail,
				"name":   userName,
				"role":   role,
				"org_id": orgID,
			},
		})
		return
	}

	api.RespondWithError(w, http.StatusInternalServerError, "failed to refresh token")
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if req.RefreshToken != "" {
		refreshHash := auth.HashRefreshToken(req.RefreshToken)
		query := `UPDATE refresh_tokens SET revoked_at = $now WHERE token_hash = $token_hash`
		vars := map[string]interface{}{
			"token_hash": refreshHash,
			"now":        time.Now(),
		}
		h.sdb.Query(ctx, query, vars)
	}

	w.WriteHeader(http.StatusNoContent)
}

func generateSurrealSlug(name string) string {
	slug := make([]byte, 8)
	rand.Read(slug)
	return base64.URLEncoding.EncodeToString(slug)[:8]
}

func generateRefreshToken() string {
	token := make([]byte, 32)
	rand.Read(token)
	return base64.URLEncoding.EncodeToString(token)
}
