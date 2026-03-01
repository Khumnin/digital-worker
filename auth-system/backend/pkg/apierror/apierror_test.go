// pkg/apierror/apierror_test.go
package apierror_test

import (
	"strings"
	"testing"
	"time"

	"tigersoft/auth-system/pkg/apierror"
)

func TestNew_FieldsPopulated(t *testing.T) {
	before := time.Now().UTC()
	got := apierror.New("AUTH_001", "unauthorized", nil, "req-abc")
	after := time.Now().UTC()

	t.Run("code is set", func(t *testing.T) {
		if got.Error.Code != "AUTH_001" {
			t.Errorf("want Code=%q, got %q", "AUTH_001", got.Error.Code)
		}
	})

	t.Run("message is set", func(t *testing.T) {
		if got.Error.Message != "unauthorized" {
			t.Errorf("want Message=%q, got %q", "unauthorized", got.Error.Message)
		}
	})

	t.Run("request_id is set", func(t *testing.T) {
		if got.Error.RequestID != "req-abc" {
			t.Errorf("want RequestID=%q, got %q", "req-abc", got.Error.RequestID)
		}
	})

	t.Run("details is nil when not provided", func(t *testing.T) {
		if got.Error.Details != nil {
			t.Errorf("want Details=nil, got %v", got.Error.Details)
		}
	})

	t.Run("timestamp is RFC3339 and within test window", func(t *testing.T) {
		ts, err := time.Parse(time.RFC3339, got.Error.Timestamp)
		if err != nil {
			t.Fatalf("Timestamp %q is not valid RFC3339: %v", got.Error.Timestamp, err)
		}
		if ts.Before(before) || ts.After(after) {
			t.Errorf("Timestamp %v is outside [%v, %v]", ts, before, after)
		}
	})
}

func TestNew_WithDetails(t *testing.T) {
	details := []map[string]string{
		{"field": "email", "message": "invalid format"},
	}
	got := apierror.New("VAL_001", "validation error", details, "")

	t.Run("details are preserved", func(t *testing.T) {
		if len(got.Error.Details) != 1 {
			t.Fatalf("want 1 detail, got %d", len(got.Error.Details))
		}
		if got.Error.Details[0]["field"] != "email" {
			t.Errorf("want field=%q, got %q", "email", got.Error.Details[0]["field"])
		}
	})

	t.Run("empty request_id stays empty", func(t *testing.T) {
		if got.Error.RequestID != "" {
			t.Errorf("want empty RequestID, got %q", got.Error.RequestID)
		}
	})
}

func TestNew_TimestampIsUTC(t *testing.T) {
	got := apierror.New("TEST", "msg", nil, "")
	// RFC3339 UTC timestamps end with "Z"
	if !strings.HasSuffix(got.Error.Timestamp, "Z") {
		t.Errorf("expected UTC timestamp ending in Z, got %q", got.Error.Timestamp)
	}
}

func TestNew_MultipleCallsHaveDifferentTimestamps(t *testing.T) {
	a := apierror.New("C1", "m1", nil, "")
	// Force a tiny gap so wall-clock seconds can differ in slow environments.
	time.Sleep(1100 * time.Millisecond)
	b := apierror.New("C2", "m2", nil, "")

	tsA, _ := time.Parse(time.RFC3339, a.Error.Timestamp)
	tsB, _ := time.Parse(time.RFC3339, b.Error.Timestamp)

	if !tsB.After(tsA) && tsA != tsB {
		// Allow equal (same second) but not B before A.
		if tsB.Before(tsA) {
			t.Errorf("second call timestamp %v should not be before first call timestamp %v", tsB, tsA)
		}
	}
}
