package ports

import (
	"context"

	"github.com/stefanoprivitera/hourglass/internal/core/domain/audit"
)

// GeneralAuditLogRepository appends rows to the general audit_logs table
// (D-05). Create is synchronous: the caller awaits the insert and the error
// propagates (T-11-08) — NOT the fire-and-forget goroutine pattern.
//
// The type is named General* because ports.AuditLogRepository already exists
// for the entry-scoped audit written to time_entry_approvals (BE-012 legacy
// behavior, untouched). The two ports share the Create shape but address
// different tables and entities.
type GeneralAuditLogRepository interface {
	Create(ctx context.Context, log *audit.AuditLog) error
}
