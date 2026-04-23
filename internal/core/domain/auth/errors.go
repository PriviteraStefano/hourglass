package auth

import "errors"

var (
	ErrEmptyEmail    = errors.New("empty email supplied")
	ErrInvalidEmail  = errors.New("invalid email format")
	ErrEmptyPassword = errors.New("empty password supplied")
	ErrWeakPassword  = errors.New("password must be at least 8 characters")
	ErrEmptyUsername = errors.New("empty username supplied")
	ErrShortUsername = errors.New("username must be at least 3 characters")
)
