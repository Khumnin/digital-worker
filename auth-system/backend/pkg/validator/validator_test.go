// pkg/validator/validator_test.go
package validator_test

import (
	"strings"
	"testing"

	"tigersoft/auth-system/internal/domain"
	"tigersoft/auth-system/pkg/validator"
)

// defaultPolicy returns the production-default strict password policy.
func defaultPolicy() domain.PasswordPolicy {
	return domain.PasswordPolicy{
		MinLength:        12,
		RequireUppercase: true,
		RequireNumber:    true,
		RequireSpecial:   true,
	}
}

// ---------------------------------------------------------------------------
// CheckPasswordPolicy -- passing cases
// ---------------------------------------------------------------------------

func TestCheckPasswordPolicy_ValidPassword(t *testing.T) {
	policy := defaultPolicy()
	validPasswords := []string{
		"CorrectHorse99!",
		"MyP@ssw0rd123",
		"Abcdefghij1!",
		"SuperSecure#8Password",
	}
	for _, pw := range validPasswords {
		t.Run(pw, func(t *testing.T) {
			if err := validator.CheckPasswordPolicy(pw, policy); err != nil {
				t.Errorf("expected no error for valid password %q, got: %v", pw, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// CheckPasswordPolicy -- failing cases (default strict policy)
// ---------------------------------------------------------------------------

func TestCheckPasswordPolicy_TooShort(t *testing.T) {
	policy := defaultPolicy()
	err := validator.CheckPasswordPolicy("Short1!", policy)
	if err == nil {
		t.Error("expected error for password below minimum length, got nil")
	}
	if !strings.Contains(err.Error(), "minimum length") {
		t.Errorf("expected minimum length message, got: %v", err)
	}
}

func TestCheckPasswordPolicy_MissingUppercase(t *testing.T) {
	policy := defaultPolicy()
	err := validator.CheckPasswordPolicy("nouppercase1!", policy)
	if err == nil {
		t.Error("expected error for missing uppercase, got nil")
	}
	if !strings.Contains(err.Error(), "uppercase") {
		t.Errorf("expected uppercase message, got: %v", err)
	}
}

func TestCheckPasswordPolicy_MissingNumber(t *testing.T) {
	policy := defaultPolicy()
	err := validator.CheckPasswordPolicy("NoNumbersHere!", policy)
	if err == nil {
		t.Error("expected error for missing number, got nil")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("expected number message, got: %v", err)
	}
}

func TestCheckPasswordPolicy_MissingSpecialChar(t *testing.T) {
	policy := defaultPolicy()
	err := validator.CheckPasswordPolicy("NoSpecialChar1", policy)
	if err == nil {
		t.Error("expected error for missing special character, got nil")
	}
	if !strings.Contains(err.Error(), "special") {
		t.Errorf("expected special character message, got: %v", err)
	}
}

func TestCheckPasswordPolicy_EmptyPassword(t *testing.T) {
	policy := defaultPolicy()
	err := validator.CheckPasswordPolicy("", policy)
	if err == nil {
		t.Error("expected error for empty password, got nil")
	}
}

// ---------------------------------------------------------------------------
// CheckPasswordPolicy -- custom (relaxed) policies
// ---------------------------------------------------------------------------

func TestCheckPasswordPolicy_RelaxedPolicy_OnlyMinLength_Passes(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        6,
		RequireUppercase: false,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("simple", policy); err != nil {
		t.Errorf("expected no error for relaxed policy, got: %v", err)
	}
}

func TestCheckPasswordPolicy_RelaxedPolicy_TooShort_Fails(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        6,
		RequireUppercase: false,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("tiny", policy); err == nil {
		t.Error("expected error for password shorter than min length under relaxed policy")
	}
}

func TestCheckPasswordPolicy_UppercaseRequired_AllLower_Fails(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("alllower1", policy); err == nil {
		t.Error("expected error when uppercase required but not present")
	}
}

func TestCheckPasswordPolicy_UppercaseRequired_HasUpper_Passes(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: true,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("HasUpper1", policy); err != nil {
		t.Errorf("expected no error when uppercase present, got: %v", err)
	}
}

func TestCheckPasswordPolicy_NumberRequired_NoDigit_Fails(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: false,
		RequireNumber:    true,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("NoDigitHere", policy); err == nil {
		t.Error("expected error when number required but not present")
	}
}

func TestCheckPasswordPolicy_SpecialRequired_NoSpecial_Fails(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: false,
		RequireNumber:    false,
		RequireSpecial:   true,
	}
	if err := validator.CheckPasswordPolicy("NoSpecial1", policy); err == nil {
		t.Error("expected error when special char required but not present")
	}
}

func TestCheckPasswordPolicy_ExactMinLength_Passes(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: false,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("exactly8", policy); err != nil {
		t.Errorf("expected no error at exact min length, got: %v", err)
	}
}

func TestCheckPasswordPolicy_OneLessThanMinLength_Fails(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        8,
		RequireUppercase: false,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("only7ch", policy); err == nil {
		t.Error("expected error for password one character below min length")
	}
}

func TestCheckPasswordPolicy_ZeroMinLength_EmptyPasses(t *testing.T) {
	policy := domain.PasswordPolicy{
		MinLength:        0,
		RequireUppercase: false,
		RequireNumber:    false,
		RequireSpecial:   false,
	}
	if err := validator.CheckPasswordPolicy("", policy); err != nil {
		t.Errorf("expected no error when MinLength=0 and password is empty, got: %v", err)
	}
}
