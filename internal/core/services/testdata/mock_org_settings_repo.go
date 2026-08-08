package testdata

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// MockOrgSettingsRepo implements ports.OrgSettingsRepository for service
// unit tests (13-04/13-07).
//
// Default behaviors: Get serves the Values map keyed by key — (nil, nil)
// when absent (absence is not an error; the service applies code-level
// defaults, e.g. orgsettings.DefaultDailyHours). Upsert stores the value
// and captures its audit row (D-13-22 in-tx shape); UpsertErr forces an
// error when set. GetFn overrides Get for non-derived answers.
type MockOrgSettingsRepo struct {
	mu     sync.Mutex
	Values map[string]json.RawMessage

	GetFn     func(ctx context.Context, orgID uuid.UUID, key string) (json.RawMessage, error)
	UpsertErr error

	// Audit capture: every audit row passed to Upsert lands here.
	Audits []*audit.AuditLog
}

var _ ports.OrgSettingsRepository = (*MockOrgSettingsRepo)(nil)

func (m *MockOrgSettingsRepo) Get(ctx context.Context, orgID uuid.UUID, key string) (json.RawMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.GetFn != nil {
		return m.GetFn(ctx, orgID, key)
	}
	v, ok := m.Values[key]
	if !ok {
		return nil, nil
	}
	return v, nil
}

func (m *MockOrgSettingsRepo) List(ctx context.Context, orgID uuid.UUID) (map[string]json.RawMessage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]json.RawMessage, len(m.Values))
	for k, v := range m.Values {
		out[k] = v
	}
	return out, nil
}

func (m *MockOrgSettingsRepo) Upsert(ctx context.Context, orgID uuid.UUID, key string, value json.RawMessage, a *audit.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.UpsertErr != nil {
		return m.UpsertErr
	}
	if m.Values == nil {
		m.Values = make(map[string]json.RawMessage)
	}
	m.Values[key] = value
	if a != nil {
		m.Audits = append(m.Audits, a)
	}
	return nil
}
