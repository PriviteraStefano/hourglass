package testdata

import (
	"context"
	"sync"

	auditdomain "github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
	"github.com/stefanoprivitera/hourglass/internal/core/ports"
)

// MockAuditLogRepo implements ports.GeneralAuditLogRepository for service
// unit tests: Create records the general audit row (D-05) for later
// assertions.
type MockAuditLogRepo struct {
	mu   sync.Mutex
	Logs []*auditdomain.AuditLog
}

var _ ports.GeneralAuditLogRepository = (*MockAuditLogRepo)(nil)

func (m *MockAuditLogRepo) Create(ctx context.Context, log *auditdomain.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Logs = append(m.Logs, log)
	return nil
}
