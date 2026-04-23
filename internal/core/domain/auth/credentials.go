package auth

import "strings"

type Email string

func NewEmail(e string) (Email, error) {
	e = strings.TrimSpace(e)
	if e == "" {
		return "", ErrEmptyEmail
	}
	if !strings.Contains(e, "@") {
		return "", ErrInvalidEmail
	}
	return Email(e), nil
}

func (e Email) String() string {
	return string(e)
}

type Password string

func NewPassword(p string) (Password, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", ErrEmptyPassword
	}
	if len(p) < 8 {
		return "", ErrWeakPassword
	}
	return Password(p), nil
}

func (p Password) String() string {
	return string(p)
}

type Username string

func NewUsername(u string) (Username, error) {
	u = strings.TrimSpace(u)
	if u == "" {
		return "", ErrEmptyUsername
	}
	if len(u) < 3 {
		return "", ErrShortUsername
	}
	return Username(u), nil
}

func (u Username) String() string {
	return string(u)
}
