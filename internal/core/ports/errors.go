package ports

import "errors"

var (
	ErrNotFound    = errors.New("not found")
	ErrConflict    = errors.New("entity conflict")
	ErrForeignKey  = errors.New("foreign key violation")
)
