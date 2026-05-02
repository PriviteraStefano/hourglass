package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ExportRow struct {
	EntryType   string
	Date        time.Time
	Employee    string
	Project     string
	Contract    string
	Customer    string
	Hours       *float64
	Amount      *float64
	KmDistance  *float64
	Type        string
	Description string
	Status      string
}

type ExportRepository interface {
	Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ExportRow, error)
	Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ExportRow, error)
}
