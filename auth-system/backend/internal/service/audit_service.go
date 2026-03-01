// internal/service/audit_service.go
package service

import (
	"context"
	"fmt"

	"tigersoft/auth-system/internal/domain"
)

// AuditService exposes read access to the immutable audit log.
type AuditService interface {
	List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditEvent, int, error)
}

type auditServiceImpl struct {
	auditRepo domain.AuditRepository
}

// NewAuditService constructs an AuditService with the audit repository
// injected.
func NewAuditService(auditRepo domain.AuditRepository) AuditService {
	return &auditServiceImpl{
		auditRepo: auditRepo,
	}
}

// List returns a filtered, paginated slice of audit events and the total
// number of matching records.
func (s *auditServiceImpl) List(ctx context.Context, filter domain.AuditFilter) ([]*domain.AuditEvent, int, error) {
	events, total, err := s.auditRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}

	return events, total, nil
}
