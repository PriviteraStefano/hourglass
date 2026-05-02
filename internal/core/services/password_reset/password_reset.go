package password_reset

import (
	"context"
	"crypto/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/password_reset"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo             ports.PasswordResetRepository
	userRepo         ports.UserRepository
	userFinder       ports.UserFinder
	hasher           ports.PasswordHasher
	tokenService     ports.TokenService
	refreshTokenRepo ports.RefreshTokenRepository
}

func NewService(
	repo ports.PasswordResetRepository,
	userRepo ports.UserRepository,
	userFinder ports.UserFinder,
	hasher ports.PasswordHasher,
	tokenService ports.TokenService,
	refreshTokenRepo ports.RefreshTokenRepository,
) *Service {
	return &Service{
		repo:             repo,
		userRepo:         userRepo,
		userFinder:       userFinder,
		hasher:           hasher,
		tokenService:     tokenService,
		refreshTokenRepo: refreshTokenRepo,
	}
}

func (s *Service) Request(ctx context.Context, identifier string) (code string, expiresAt time.Time, err error) {
	userID, err := s.userFinder.FindByIdentifier(ctx, identifier)
	if err != nil {
		return "", time.Time{}, password_reset.ErrUserNotFound
	}

	code = generateResetCode()
	codeHash, err := s.hasher.Hash(code)
	if err != nil {
		return "", time.Time{}, err
	}

	expiresAt = time.Now().Add(2 * time.Hour)

	pr := &password_reset.PasswordReset{
		ID:        uuid.New(),
		UserID:    uuid.MustParse(userID),
		CodeHash:  codeHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	_, err = s.repo.Create(ctx, pr)
	if err != nil {
		return "", time.Time{}, err
	}

	return code, expiresAt, nil
}

func (s *Service) Verify(ctx context.Context, identifier, code, password string) error {
	userID, err := s.userFinder.FindByIdentifier(ctx, identifier)
	if err != nil {
		return password_reset.ErrUserNotFound
	}

	pr, err := s.repo.FindActiveByUserID(ctx, userID)
	if err != nil {
		return password_reset.ErrResetExpired
	}

	if !s.hasher.Check(code, pr.CodeHash) {
		return password_reset.ErrInvalidCode
	}

	newHash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}

	userUUID := uuid.MustParse(userID)
	if err := s.userRepo.UpdatePassword(ctx, userUUID, newHash); err != nil {
		return err
	}

	_ = s.repo.MarkUsed(ctx, pr.ID.String())
	_ = s.refreshTokenRepo.RevokeAllByUser(ctx, userUUID)
	return nil
}

func generateResetCode() string {
	b := make([]byte, 3)
	rand.Read(b)
	result := ""
	for _, v := range b {
		result += strconv.Itoa(int(v) % 10)
	}
	return result
}
