package export

import (
	"context"
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
	rows := mergeExportRowsDesc(timesheets, expenses)
	return rows, nil
}

// mergeExportRowsDesc merges two slices already ordered by Date ascending
// (as returned by the repository) into a single descending-ordered slice
// without an intermediate O(n log n) sort.
func mergeExportRowsDesc(a, b []ports.ExportRow) []ports.ExportRow {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make([]ports.ExportRow, 0, len(a)+len(b))
	i, j := len(a)-1, len(b)-1
	for i >= 0 && j >= 0 {
		if a[i].Date.After(b[j].Date) {
			out = append(out, a[i])
			i--
		} else {
			out = append(out, b[j])
			j--
		}
	}
	for ; i >= 0; i-- {
		out = append(out, a[i])
	}
	for ; j >= 0; j-- {
		out = append(out, b[j])
	}
	return out
}

func (s *Service) CountTimesheets(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	return s.repo.CountTimesheets(ctx, orgID, from, to, role, userID)
}

func (s *Service) CountExpenses(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	return s.repo.CountExpenses(ctx, orgID, from, to, role, userID)
}

func (s *Service) CountCombined(ctx context.Context, orgID uuid.UUID, from, to time.Time, role string, userID uuid.UUID) (int, error) {
	teCount, err := s.repo.CountTimesheets(ctx, orgID, from, to, role, userID)
	if err != nil {
		return 0, err
	}
	expCount, err := s.repo.CountExpenses(ctx, orgID, from, to, role, userID)
	if err != nil {
		return 0, err
	}
	return teCount + expCount, nil
}
