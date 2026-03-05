// internal/handler/build_user_response_test.go
package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/internal/service"
)

// ---------------------------------------------------------------------------
// buildUserResponse tests
// Tests verify the response shape produced by the shared helper that is used
// by AdminHandler.ListUsers, AdminHandler.GetUser, AdminHandler.ReplaceUserRoles,
// and TenantHandler.ListTenantUsers.
// ---------------------------------------------------------------------------

func makeTestUser(email string) *domain.User {
	return &domain.User{
		ID:        uuid.New(),
		Email:     email,
		FirstName: "Test",
		LastName:  "User",
		Status:    domain.UserStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// TC-BR-1: Single-tenant path — TenantID/TenantName resolved from caller values
// when UserWithRoles has no cross-tenant data set.
func TestBuildUserResponse_SingleTenantPath_UsesFallbackTenantValues(t *testing.T) {
	uwr := &service.UserWithRoles{
		User:        makeTestUser("user@acme.com"),
		SystemRoles: []string{"admin"},
		ModuleRoles: map[string][]string{},
		// TenantID and TenantName are intentionally empty (single-tenant path)
	}

	resp := buildUserResponse(uwr, "acme", "ACME Corp")

	if resp["tenant_id"] != "acme" {
		t.Errorf("tenant_id: got %v, want acme", resp["tenant_id"])
	}
	if resp["tenant_name"] != "ACME Corp" {
		t.Errorf("tenant_name: got %v, want ACME Corp", resp["tenant_name"])
	}
}

// TC-BR-2: Cross-tenant path — TenantID/TenantName from UserWithRoles take precedence
// over the caller-supplied fallback values.
func TestBuildUserResponse_CrossTenantPath_PrefersTenantValuesFromUWR(t *testing.T) {
	uwr := &service.UserWithRoles{
		User:        makeTestUser("admin@corp.com"),
		SystemRoles: []string{"admin"},
		ModuleRoles: map[string][]string{},
		TenantID:    "corp",
		TenantName:  "Corp Ltd",
	}

	// Pass different fallback values — they must NOT win.
	resp := buildUserResponse(uwr, "other-tenant", "Other Tenant Name")

	if resp["tenant_id"] != "corp" {
		t.Errorf("tenant_id: got %v, want corp (from UWR)", resp["tenant_id"])
	}
	if resp["tenant_name"] != "Corp Ltd" {
		t.Errorf("tenant_name: got %v, want Corp Ltd (from UWR)", resp["tenant_name"])
	}
}

// TC-BR-3: Empty tenant_name falls back to the caller-supplied tenantName string,
// not an empty string. This mirrors the AC-4 edge case on the handler side.
func TestBuildUserResponse_EmptyUWRTenantName_FallsBackToCallerTenantName(t *testing.T) {
	uwr := &service.UserWithRoles{
		User:        makeTestUser("user@x.com"),
		SystemRoles: []string{"user"},
		ModuleRoles: map[string][]string{},
		TenantID:    "",
		TenantName:  "", // empty — caller value should be used
	}

	resp := buildUserResponse(uwr, "x-slug", "X Corp")

	if resp["tenant_name"] != "X Corp" {
		t.Errorf("tenant_name: got %v, want X Corp (fallback)", resp["tenant_name"])
	}
}

// TC-BR-4: Nil system_roles must be serialized as an empty slice, never nil,
// to avoid JSON null on the frontend.
func TestBuildUserResponse_NilSystemRoles_SerializedAsEmptySlice(t *testing.T) {
	uwr := &service.UserWithRoles{
		User:        makeTestUser("norole@example.com"),
		SystemRoles: nil,
		ModuleRoles: nil,
	}

	resp := buildUserResponse(uwr, "t", "Tenant")

	roles, ok := resp["system_roles"].([]string)
	if !ok {
		t.Fatalf("system_roles is not []string, got %T", resp["system_roles"])
	}
	if len(roles) != 0 {
		t.Errorf("expected empty system_roles slice, got %v", roles)
	}
}

// TC-BR-5: Nil module_roles must be serialized as an empty map, never nil.
func TestBuildUserResponse_NilModuleRoles_SerializedAsEmptyMap(t *testing.T) {
	uwr := &service.UserWithRoles{
		User:        makeTestUser("norole@example.com"),
		SystemRoles: []string{},
		ModuleRoles: nil,
	}

	resp := buildUserResponse(uwr, "t", "Tenant")

	mods, ok := resp["module_roles"].(map[string][]string)
	if !ok {
		t.Fatalf("module_roles is not map[string][]string, got %T", resp["module_roles"])
	}
	if len(mods) != 0 {
		t.Errorf("expected empty module_roles map, got %v", mods)
	}
}

// TC-BR-6: Status is normalized from internal DB value to API contract value.
func TestBuildUserResponse_StatusNormalized(t *testing.T) {
	cases := []struct {
		dbStatus  domain.UserStatus
		wantAPI   string
	}{
		{domain.UserStatusUnverified, "pending"},
		{domain.UserStatusDisabled, "inactive"},
		{domain.UserStatusActive, "active"},
	}

	for _, tc := range cases {
		t.Run(string(tc.dbStatus), func(t *testing.T) {
			u := makeTestUser("x@y.com")
			u.Status = tc.dbStatus
			uwr := &service.UserWithRoles{
				User:        u,
				SystemRoles: []string{},
				ModuleRoles: map[string][]string{},
			}
			resp := buildUserResponse(uwr, "t", "Tenant")
			if resp["status"] != tc.wantAPI {
				t.Errorf("status: got %v, want %v", resp["status"], tc.wantAPI)
			}
		})
	}
}

// TC-BR-7: Nil UserWithRoles returns an error response, not a panic.
func TestBuildUserResponse_NilInput_ReturnsErrorShape(t *testing.T) {
	resp := buildUserResponse(nil, "t", "Tenant")
	if _, ok := resp["error"]; !ok {
		t.Error("expected error key in response for nil input, got none")
	}
}

// TC-BR-8: Nil User inside UserWithRoles returns an error response, not a panic.
func TestBuildUserResponse_NilUser_ReturnsErrorShape(t *testing.T) {
	uwr := &service.UserWithRoles{
		User: nil,
	}
	resp := buildUserResponse(uwr, "t", "Tenant")
	if _, ok := resp["error"]; !ok {
		t.Error("expected error key in response for nil User, got none")
	}
}
