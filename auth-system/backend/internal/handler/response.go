// internal/handler/response.go
package handler

import (
	"tigersoft/auth-system/pkg/validator"
)

// globalValidator is the package-level singleton used by all handler bind functions.
var globalValidator = validator.New()
