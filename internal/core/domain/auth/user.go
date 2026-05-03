package auth

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	FirstName    string
	LastName     string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewUser(email, username, firstName, lastName, passwordHash string) *User {
	now := time.Now()
	return &User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
