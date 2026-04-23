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
	ErrEmailExists        = errors.New("email already registered")
	ErrUsernameExists     = errors.New("username already taken")
	ErrInvalidCreds       = errors.New("invalid credentials")
	ErrAccountDeactivated = errors.New("account is deactivated")
	ErrUserNotFound       = errors.New("user not found")
)

type RegisterRequest struct {
	Email     string
	Username  string
	FirstName string
	LastName  string
	Name      string
	Password  string
	OrgName   string
}

type RegisterResponse struct {
	UserID string
	Email  string
	Name   string
}

type LoginRequest struct {
	Identifier string
	Password   string
}

type LoginResponse struct {
	UserID       string
	Email        string
	Name         string
	Role         string
	OrgID        uuid.UUID
	Token        string
	RefreshToken string
	ExpiresAt    time.Time
}

type ProfileResponse struct {
	ID        string
	Email     string
	Name      string
	IsActive  bool
	CreatedAt time.Time
}

type RefreshResponse struct {
	UserID    string
	Email     string
	Name      string
	Role      string
	OrgID     uuid.UUID
	Token     string
	ExpiresAt time.Time
}

type BootstrapRequest struct {
	OrgName   string
	Email     string
	Username  string
	FirstName string
	LastName  string
	Password  string
}

type BootstrapResponse struct {
	Token    string
	UserID   string
	Email    string
	Username string
	Name     string
	OrgID    string
	OrgName  string
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

	return &RegisterResponse{
		UserID: user.ID.String(),
		Email:  user.Email,
		Name:   user.Name,
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	email, err := authdomain.NewEmail(req.Identifier)
	if err != nil {
		_, err := authdomain.NewUsername(req.Identifier)
		if err != nil {
			return nil, ErrInvalidCreds
		}
	}

	var user *authdomain.User
	if email != "" {
		user, err = s.userRepo.GetByEmail(ctx, email.String())
	} else {
		user, err = s.userRepo.GetByEmail(ctx, req.Identifier)
	}
	if err != nil {
		return nil, ErrInvalidCreds
	}

	if !s.hasher.Check(req.Password, user.PasswordHash) {
		return nil, ErrInvalidCreds
	}

	if !user.IsActive {
		return nil, ErrAccountDeactivated
	}

	token, err := s.tokenService.GenerateToken(user.ID, uuid.Nil, "employee", user.Email)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshHash := s.tokenService.HashRefreshToken(refreshToken)
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	if err := s.refreshTokenRepo.Add(ctx, user.ID, refreshHash, expiresAt); err != nil {
		return nil, err
	}

	return &LoginResponse{
		UserID:       user.ID.String(),
		Email:        user.Email,
		Name:         user.Name,
		Role:         "employee",
		OrgID:        uuid.Nil,
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(15 * time.Minute),
	}, nil
}

func (s *Service) GetProfile(ctx context.Context, userID string) (*ProfileResponse, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	user, err := s.userRepo.GetByID(ctx, uid)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return &ProfileResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
	}, nil
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

	newToken, err := s.tokenService.GenerateToken(user.ID, uuid.Nil, "employee", user.Email)
	if err != nil {
		return nil, err
	}

	return &RefreshResponse{
		UserID:    user.ID.String(),
		Email:     user.Email,
		Name:      user.Name,
		Role:      "employee",
		OrgID:     uuid.Nil,
		Token:     newToken,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}, nil
}

func (s *Service) Bootstrap(ctx context.Context, req BootstrapRequest) (*BootstrapResponse, error) {
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

	tokenUserID := uuid.NewMD5(uuid.Nil, []byte(user.ID.String()))
	token, err := s.tokenService.GenerateToken(tokenUserID, uuid.Nil, "admin", user.Email)
	if err != nil {
		return nil, err
	}

	return &BootstrapResponse{
		Token:    token,
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: req.Username,
		Name:     displayName,
		OrgID:    org.ID.String(),
		OrgName:  org.Name,
	}, nil
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
