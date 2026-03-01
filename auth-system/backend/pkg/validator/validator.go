// pkg/validator/validator.go
package validator

import (
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	once     sync.Once
	validate *validator.Validate
)

// Validator wraps go-playground/validator for use in handlers.
type Validator struct {
	v *validator.Validate
}

// New creates and returns a singleton Validator instance.
func New() *Validator {
	once.Do(func() {
		validate = validator.New()
	})
	return &Validator{v: validate}
}

// ValidateStruct validates a struct and returns a slice of field-level error maps.
// Each map has "field" and "message" keys. Returns nil if validation passes.
func (val *Validator) ValidateStruct(s interface{}) []map[string]string {
	if err := val.v.Struct(s); err != nil {
		var errs []map[string]string
		for _, fe := range err.(validator.ValidationErrors) {
			errs = append(errs, map[string]string{
				"field":   strings.ToLower(fe.Field()),
				"message": validationMessage(fe),
			})
		}
		return errs
	}
	return nil
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required."
	case "email":
		return "Must be a valid email address."
	case "min":
		return "Must be at least " + fe.Param() + " characters long."
	case "max":
		return "Must be at most " + fe.Param() + " characters long."
	default:
		return "Failed validation: " + fe.Tag()
	}
}

// GlobalValidator is the package-level singleton used by handlers.
var GlobalValidator = New()
