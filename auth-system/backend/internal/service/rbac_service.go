// internal/service/rbac_service.go
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
)

// RBACService manages roles and their assignment to users within a tenant.
type RBACService interface {
	CreateRole(ctx context.Context, name, description string, module *string) (*domain.Role, error)
	ListRoles(ctx context.Context) ([]*domain.Role, error)
	// DeleteRole removes a non-system role that is not currently assigned to any user.
	// Returns domain.ErrSystemRole if the role is a system role, or
	// domain.ErrRoleInUse if any user currently holds the role.
	DeleteRole(ctx context.Context, roleID string) error
	AssignRole(ctx context.Context, userID, roleID, assignedBy string) error
	UnassignRole(ctx context.Context, userID, roleID, assignedBy string) error
	GetUserRoles(ctx context.Context, userID string) ([]*domain.Role, error)
}

type rbacServiceImpl struct {
	roleRepo  domain.RoleRepository
	userRepo  domain.UserRepository
	auditRepo domain.AuditRepository
}

// NewRBACService constructs an RBACService with all dependencies injected.
func NewRBACService(
	roleRepo domain.RoleRepository,
	userRepo domain.UserRepository,
	auditRepo domain.AuditRepository,
) RBACService {
	return &rbacServiceImpl{
		roleRepo:  roleRepo,
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

// CreateRole persists a new role in the tenant schema.
func (s *rbacServiceImpl) CreateRole(ctx context.Context, name, description string, module *string) (*domain.Role, error) {
	role, err := s.roleRepo.Create(ctx, name, description, module)
	if err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	return role, nil
}

// ListRoles returns every role defined in the current tenant schema.
func (s *rbacServiceImpl) ListRoles(ctx context.Context) ([]*domain.Role, error) {
	roles, err := s.roleRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}

	return roles, nil
}

// DeleteRole validates that the role exists, is not a system role, and is not
// currently assigned to any user — then deletes it.
func (s *rbacServiceImpl) DeleteRole(ctx context.Context, roleID string) error {
	parsedID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	role, err := s.roleRepo.FindByID(ctx, parsedID)
	if err != nil {
		return fmt.Errorf("find role: %w", err)
	}

	if role.IsSystem {
		return domain.ErrSystemRole
	}

	inUse, err := s.roleRepo.IsAssignedToAnyUser(ctx, parsedID)
	if err != nil {
		return fmt.Errorf("check role assignment: %w", err)
	}
	if inUse {
		return domain.ErrRoleInUse
	}

	if err := s.roleRepo.Delete(ctx, parsedID); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}

	return nil
}

// AssignRole links a role to a user and writes an audit record.
func (s *rbacServiceImpl) AssignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	parsedRoleID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	parsedAssignedBy, err := uuid.Parse(assignedBy)
	if err != nil {
		return fmt.Errorf("invalid assignedBy ID: %w", err)
	}

	if err := s.roleRepo.AssignToUser(ctx, parsedUserID, parsedRoleID, parsedAssignedBy); err != nil {
		return fmt.Errorf("assign role to user: %w", err)
	}

	roleMeta := map[string]interface{}{"role_id": parsedRoleID.String()}
	if role, lookupErr := s.roleRepo.FindByID(ctx, parsedRoleID); lookupErr == nil {
		roleMeta["role_name"] = role.Name
	} else {
		slog.Warn("assign role: could not look up role name for audit", "role_id", parsedRoleID, "error", lookupErr)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventRoleAssigned,
		ActorID:      &parsedAssignedBy,
		TargetUserID: &parsedUserID,
		Metadata:     roleMeta,
	})

	return nil
}

// UnassignRole removes a role from a user and writes an audit record.
func (s *rbacServiceImpl) UnassignRole(ctx context.Context, userID, roleID, assignedBy string) error {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	parsedRoleID, err := uuid.Parse(roleID)
	if err != nil {
		return fmt.Errorf("invalid role ID: %w", err)
	}

	parsedAssignedBy, err := uuid.Parse(assignedBy)
	if err != nil {
		return fmt.Errorf("invalid assignedBy ID: %w", err)
	}

	if err := s.roleRepo.UnassignFromUser(ctx, parsedUserID, parsedRoleID); err != nil {
		return fmt.Errorf("unassign role from user: %w", err)
	}

	roleMeta := map[string]interface{}{"role_id": parsedRoleID.String()}
	if role, lookupErr := s.roleRepo.FindByID(ctx, parsedRoleID); lookupErr == nil {
		roleMeta["role_name"] = role.Name
	} else {
		slog.Warn("unassign role: could not look up role name for audit", "role_id", parsedRoleID, "error", lookupErr)
	}

	s.writeAuditEvent(ctx, domain.AuditEvent{
		EventType:    domain.EventRoleUnassigned,
		ActorID:      &parsedAssignedBy,
		TargetUserID: &parsedUserID,
		Metadata:     roleMeta,
	})

	return nil
}

// GetUserRoles returns all roles currently assigned to a user.
func (s *rbacServiceImpl) GetUserRoles(ctx context.Context, userID string) ([]*domain.Role, error) {
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	roles, err := s.roleRepo.GetUserRoles(ctx, parsedUserID)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}

	return roles, nil
}

func (s *rbacServiceImpl) writeAuditEvent(ctx context.Context, event domain.AuditEvent) {
	event.ID = uuid.New()
	event.OccurredAt = time.Now()
	if err := s.auditRepo.Append(ctx, &event); err != nil {
		slog.Error("failed to write audit event", "event_type", event.EventType, "error", err)
	}
}
