// pkg/validator/password_policy.go
package validator

import (
	"fmt"
	"unicode"

	"tigersoft/auth-system/internal/domain"
)

// CheckPasswordPolicy validates a password against the tenant's password policy.
// Returns an error describing the first policy violation, or nil if valid.
func CheckPasswordPolicy(password string, policy domain.PasswordPolicy) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("password does not meet complexity requirements: minimum length is %d characters", policy.MinLength)
	}

	if policy.RequireUppercase {
		hasUpper := false
		for _, r := range password {
			if unicode.IsUpper(r) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			return fmt.Errorf("password does not meet complexity requirements: must contain at least one uppercase letter")
		}
	}

	if policy.RequireNumber {
		hasDigit := false
		for _, r := range password {
			if unicode.IsDigit(r) {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			return fmt.Errorf("password does not meet complexity requirements: must contain at least one number")
		}
	}

	if policy.RequireSpecial {
		hasSpecial := false
		for _, r := range password {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				hasSpecial = true
				break
			}
		}
		if !hasSpecial {
			return fmt.Errorf("password does not meet complexity requirements: must contain at least one special character")
		}
	}

	return nil
}
