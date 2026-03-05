package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
)

// resolveRoles fetches the user's assigned roles and splits them into
// system roles (module IS NULL) and module roles (module IS NOT NULL).
// Falls back to (["user"], {}) on error so token issuance is never blocked.
func resolveRoles(ctx context.Context, roleRepo domain.RoleRepository, userID uuid.UUID) ([]string, map[string][]string) {
	roles, err := roleRepo.GetUserRoles(ctx, userID)
	if err != nil {
		slog.Warn("failed to fetch user roles for JWT; falling back to [user]",
			"user_id", userID, "error", err)
		return []string{"user"}, map[string][]string{}
	}

	systemRoles := make([]string, 0)
	hasUser := false
	for _, r := range roles {
		if r.Name == "user" {
			hasUser = true
		}
	}
	if !hasUser {
		systemRoles = append(systemRoles, "user")
	}

	moduleRoles := make(map[string][]string)
	for _, r := range roles {
		if r.Module == nil {
			systemRoles = append(systemRoles, r.Name)
		} else {
			mod := *r.Module
			moduleRoles[mod] = append(moduleRoles[mod], r.Name)
		}
	}

	total := len(systemRoles) + len(moduleRoles)
	if total > 20 {
		slog.Warn("user has more than 20 roles — JWT payload may be large",
			"user_id", userID, "role_count", total)
	}
	return systemRoles, moduleRoles
}
