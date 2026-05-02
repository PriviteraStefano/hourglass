package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	authdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/auth"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

var (
	ErrEmailExists           = errors.New("email already registered")
	ErrUsernameExists        = errors.New("username already taken")
	ErrInvalidCreds          = errors.New("invalid credentials")
	ErrAccountDeactivated    = errors.New("account is deactivated")
	ErrUserNotFound          = errors.New("user not found")
	ErrMembershipNotFound    = errors.New("membership not found")
	ErrNoActiveMembership    = errors.New("no active organization membership")
)

type RegisterRequest struct {
	Email     string
	Username  string
	FirstName string
	LastName  string
	Name      string
	Password  string
	OrgName   string
	OrgID     string
	Role      string
}

type UserWithMembership struct {
	User         User         `json:"user"`
	Membership   Membership   `json:"membership"`
	Organization Organization `json:"organization"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Membership struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	OrganizationID string     `json:"organization_id"`
	Role           string     `json:"role"`
	IsActive       bool       `json:"is_active"`
	ActivatedAt    *time.Time `json:"activated_at,omitempty"`
}

type Organization struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	CreatedAt string `json:"created_at"`
}

type RegisterResponse struct {
	UserWithMembership
}

type LoginRequest struct {
	Identifier string
	Password   string
}

type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserWithMembership
}

type RefreshResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserWithMembership
}

type BootstrapRequest struct {
	OrgName   string `json:"org_name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Password  string `json:"password"`
}

type BootstrapResponse struct {
	Token            string `json:"token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresAt        time.Time `json:"expires_at"`
	UserWithMembership
}

type Service struct {
	userRepo         ports.UserRepository
	orgRepo          ports.OrganizationRepository
	tokenService     ports.TokenService
	hasher           ports.PasswordHasher
	refreshTokenRepo ports.RefreshTokenRepository
}

func NewService(
	userRepo ports.UserRepository,
	orgRepo ports.OrganizationRepository,
	tokenService ports.TokenService,
	hasher ports.PasswordHasher,
	refreshTokenRepo ports.RefreshTokenRepository,
) *Service {
	return &Service{
		userRepo:         userRepo,
		orgRepo:          orgRepo,
		tokenService:     tokenService,
		hasher:           hasher,
		refreshTokenRepo: refreshTokenRepo,
	}
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	email, err := authdomain.NewEmail(req.Email)
	if err != nil {
		return nil, err
	}

	password, err := authdomain.NewPassword(req.Password)
	if err != nil {
		return nil, err
	}

	if req.Username != "" {
		if _, err := authdomain.NewUsername(req.Username); err != nil {
			return nil, err
		}
	}

	exists, err := s.userRepo.EmailExists(ctx, email.String())
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailExists
	}

	if req.Username != "" {
		exists, err := s.userRepo.UsernameExists(ctx, req.Username)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrUsernameExists
		}
	}

	hashedPassword, err := s.hasher.Hash(password.String())
	if err != nil {
		return nil, err
	}

	displayName := req.Name
	if displayName == "" && (req.FirstName != "" || req.LastName != "") {
		displayName = req.FirstName + " " + req.LastName
	}

	user := authdomain.NewUser(
		email.String(),
		req.Username,
		req.FirstName,
		req.LastName,
		displayName,
		hashedPassword,
	)

	if err := s.userRepo.Add(ctx, user); err != nil {
		return nil, err
	}

	var orgID uuid.UUID
	if req.OrgID != "" {
		parsed, _ := uuid.Parse(req.OrgID)
		orgID = parsed
	} else if req.OrgName != "" {
		orgSlug := generateSlug(req.OrgName)
		org := authdomain.NewOrganization(req.OrgName, orgSlug, "Organization")
		if err := s.orgRepo.Add(ctx, org); err != nil {
			return nil, err
		}
		orgID = org.ID
	}

	if orgID != uuid.Nil {
		membership := authdomain.NewOrganizationMembership(user.ID, orgID, "employee")
		if err := s.orgRepo.AddMembership(ctx, membership); err != nil {
			return nil, err
		}
	}

	var org *authdomain.Organization
	if orgID != uuid.Nil {
		org, _ = s.orgRepo.GetByID(ctx, orgID)
	}

	var membership *authdomain.OrganizationMembership
	if orgID != uuid.Nil {
		membership, _ = s.orgRepo.GetMembership(ctx, user.ID, orgID)
	}

	return &RegisterResponse{
		UserWithMembership: *buildUserWithMembershipPtr(user, orgID, org, membership),
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	var user *authdomain.User
	var err error

	email, emailErr := authdomain.NewEmail(req.Identifier)
	if emailErr == nil {
		user, err = s.userRepo.GetByEmail(ctx, email.String())
	} else {
		_, usernameErr := authdomain.NewUsername(req.Identifier)
		if usernameErr != nil {
			return nil, ErrInvalidCreds
		}
		user, err = s.userRepo.GetByUsername(ctx, req.Identifier)
	}
	if err != nil {
		return nil, ErrInvalidCreds
	}

	return s.authenticateUser(ctx, user, req.Password)
}

func (s *Service) authenticateUser(ctx context.Context, user *authdomain.User, password string) (*LoginResponse, error) {
	if !s.hasher.Check(password, user.PasswordHash) {
		return nil, ErrInvalidCreds
	}

	if !user.IsActive {
		return nil, ErrAccountDeactivated
	}

	memberships, err := s.userRepo.GetMemberships(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	var active *authdomain.OrganizationMembership
	for i := range memberships {
		if memberships[i].IsActive {
			active = &memberships[i]
			break
		}
	}
	if active == nil && len(memberships) > 0 {
		active = &memberships[0]
	}

	var orgID uuid.UUID
	var role string
	if active != nil {
		orgID = active.OrganizationID
		role = active.Role
	}

	token, err := s.tokenService.GenerateToken(user.ID, orgID, role, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshHash := s.tokenService.HashRefreshToken(refreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.refreshTokenRepo.Add(ctx, user.ID, orgID, refreshHash, expiresAt); err != nil {
		return nil, err
	}

	var org *authdomain.Organization
	if orgID != uuid.Nil {
		org, _ = s.orgRepo.GetByID(ctx, orgID)
	}

	return &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
		UserWithMembership: *buildUserWithMembershipPtr(user, orgID, org, active),
	}, nil
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) (*UserWithMembership, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	var membership *authdomain.OrganizationMembership
	if orgID != uuid.Nil {
		membership, _ = s.orgRepo.GetMembership(ctx, userID, orgID)
	}

	var org *authdomain.Organization
	if orgID != uuid.Nil {
		org, _ = s.orgRepo.GetByID(ctx, orgID)
	}

	return buildUserWithMembershipPtr(user, orgID, org, membership), nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	hash := s.tokenService.HashRefreshToken(refreshToken)

	token, err := s.refreshTokenRepo.FindByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, ErrInvalidCreds
	}

	user, err := s.userRepo.GetByID(ctx, token.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	membership, _ := s.orgRepo.GetMembership(ctx, token.UserID, token.OrganizationID)

	var org *authdomain.Organization
	if token.OrganizationID != uuid.Nil {
		org, _ = s.orgRepo.GetByID(ctx, token.OrganizationID)
	}

	var role string
	if membership != nil {
		role = membership.Role
	}

	newToken, err := s.tokenService.GenerateToken(user.ID, token.OrganizationID, role, user.Email)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		Token:            newToken,
		ExpiresAt:        time.Now().Add(15 * time.Minute),
		UserWithMembership: *buildUserWithMembershipPtr(user, token.OrganizationID, org, membership),
	}, nil
}

func (s *Service) Bootstrap(ctx context.Context, req BootstrapRequest) (*BootstrapResponse, error) {
	email, err := authdomain.NewEmail(req.Email)
	if err != nil {
		return nil, err
	}

	// Check if ANY user already exists - bootstrap is only for first user
	anyExists, err := s.userRepo.AnyExists(ctx)
	if err != nil {
		return nil, err
	}
	if anyExists {
		return nil, ErrEmailExists // Reuse the error for simplicity
	}

	password, err := authdomain.NewPassword(req.Password)
	if err != nil {
		return nil, err
	}

	if req.Username != "" {
		if _, err := authdomain.NewUsername(req.Username); err != nil {
			return nil, err
		}
	}

	orgSlug := generateSlug(req.OrgName)
	org := authdomain.NewOrganization(req.OrgName, orgSlug, "Bootstrap organization")
	if err := s.orgRepo.Add(ctx, org); err != nil {
		return nil, err
	}

	hashedPassword, err := s.hasher.Hash(password.String())
	if err != nil {
		return nil, err
	}

	displayName := ""
	if req.FirstName != "" || req.LastName != "" {
		displayName = req.FirstName + " " + req.LastName
	}

	user := authdomain.NewUser(
		email.String(),
		req.Username,
		req.FirstName,
		req.LastName,
		displayName,
		hashedPassword,
	)

	if err := s.userRepo.Add(ctx, user); err != nil {
		return nil, err
	}

	membership := authdomain.NewOrganizationMembership(user.ID, org.ID, "admin")
	if err := s.orgRepo.AddMembership(ctx, membership); err != nil {
		return nil, err
	}

	tokenUserID := user.ID
	token, err := s.tokenService.GenerateToken(tokenUserID, org.ID, "admin", user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	refreshHash := s.tokenService.HashRefreshToken(refreshToken)
	if err := s.refreshTokenRepo.Add(ctx, user.ID, org.ID, refreshHash, time.Now().Add(7*24*time.Hour)); err != nil {
		return nil, err
	}

	return &BootstrapResponse{
		Token:            token,
		RefreshToken:     refreshToken,
		ExpiresAt:        time.Now().Add(15 * time.Minute),
		UserWithMembership: *buildUserWithMembershipPtr(user, org.ID, org, membership),
	}, nil
}

func (s *Service) SwitchOrganization(ctx context.Context, userID, orgID uuid.UUID) (*LoginResponse, error) {
	membership, err := s.orgRepo.GetMembership(ctx, userID, orgID)
	if err != nil {
		return nil, err
	}
	if membership == nil {
		return nil, ErrMembershipNotFound
	}
	if !membership.IsActive {
		return nil, ErrMembershipNotFound
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	token, err := s.tokenService.GenerateToken(userID, orgID, membership.Role, user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshHash := s.tokenService.HashRefreshToken(refreshToken)
	if err := s.refreshTokenRepo.Add(ctx, userID, orgID, refreshHash, time.Now().Add(7*24*time.Hour)); err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:            token,
		RefreshToken:     refreshToken,
		ExpiresAt:        time.Now().Add(15 * time.Minute),
		UserWithMembership: *buildUserWithMembershipPtr(user, orgID, org, membership),
	}, nil
}

func (s *Service) AnyExists(ctx context.Context) (bool, error) {
	return s.userRepo.AnyExists(ctx)
}

func (s *Service) GetMemberships(ctx context.Context, userID uuid.UUID) ([]authdomain.OrganizationMembership, error) {
	return s.userRepo.GetMemberships(ctx, userID)
}

func (s *Service) GetOrgByID(ctx context.Context, orgID uuid.UUID) (*authdomain.Organization, error) {
	return s.orgRepo.GetByID(ctx, orgID)
}

func generateSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	hash := s.tokenService.HashRefreshToken(refreshToken)
	return s.refreshTokenRepo.RevokeByHash(ctx, hash)
}

func buildUserWithMembershipPtr(user *authdomain.User, orgID uuid.UUID, org *authdomain.Organization, membership *authdomain.OrganizationMembership) *UserWithMembership {
	var m Membership
	if membership != nil {
		m = Membership{
			ID:             membership.ID.String(),
			UserID:         membership.UserID.String(),
			OrganizationID: membership.OrganizationID.String(),
			Role:           membership.Role,
			IsActive:       membership.IsActive,
			ActivatedAt:    membership.ActivatedAt,
		}
	}

	var org_out Organization
	if org != nil {
		org_out = Organization{
			ID:   org.ID.String(),
			Name: org.Name,
			Slug: org.Slug,
		}
	}

	return &UserWithMembership{
		User: User{
			ID:        user.ID.String(),
			Email:     user.Email,
			Username:  user.Username,
			Name:      user.Name,
			IsActive:  user.IsActive,
			CreatedAt: user.CreatedAt,
		},
		Membership:   m,
		Organization: org_out,
	}
}
