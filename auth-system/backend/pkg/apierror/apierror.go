// pkg/apierror/apierror.go
package apierror

import "time"

// APIError is the standard error response format for all API errors.
type APIError struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains the structured error details.
type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   []map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Timestamp string            `json:"timestamp"`
}

// New creates a new APIError with the given code, message, details, and request ID.
func New(code, message string, details []map[string]string, requestID string) APIError {
	return APIError{
		Error: ErrorBody{
			Code:      code,
			Message:   message,
			Details:   details,
			RequestID: requestID,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		},
	}
}
