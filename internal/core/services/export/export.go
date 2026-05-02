package export

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

type Service struct {
	repo ports.ExportRepository
}

func NewService(repo ports.ExportRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Timesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return s.repo.Timesheets(ctx, orgID, from, to, role, userID)
}

func (s *Service) Expenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	return s.repo.Expenses(ctx, orgID, from, to, role, userID)
}

func (s *Service) Combined(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) ([]ports.ExportRow, error) {
	timesheets, err := s.repo.Timesheets(ctx, orgID, from, to, role, userID)
	if err != nil {
		return nil, err
	}
	expenses, err := s.repo.Expenses(ctx, orgID, from, to, role, userID)
	if err != nil {
		return nil, err
	}
	rows := append(timesheets, expenses...)
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Date.After(rows[j].Date)
	})
	return rows, nil
}
