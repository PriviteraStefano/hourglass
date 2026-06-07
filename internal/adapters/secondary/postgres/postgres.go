package postgres

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// wrapPGError translates known pgx errors into domain sentinel errors.
//   - pgx.ErrNoRows → ports.ErrNotFound
//   - unique_violation (23505) → ports.ErrConflict
//   - foreign_key_violation (23503) → ports.ErrForeignKey
func wrapPGError(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, ports.ErrNotFound)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%s: %w", op, ports.ErrConflict)
		case "23503":
			return fmt.Errorf("%s: %w", op, ports.ErrForeignKey)
		}
	}
	return fmt.Errorf("%s: %w", op, err)
}
