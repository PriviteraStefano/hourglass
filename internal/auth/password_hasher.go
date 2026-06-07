package auth

import (
	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher implements ports.PasswordHasher using bcrypt.
type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

func (h *PasswordHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func (h *PasswordHasher) Check(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
