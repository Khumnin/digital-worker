// internal/domain/domain_test.go
package domain_test

import (
	"testing"
	"time"

	"tigersoft/auth-system/internal/domain"
)

// ---------------------------------------------------------------------------
// User.IsLocked
// ---------------------------------------------------------------------------

func TestUser_IsLocked_NoLock(t *testing.T) {
	u := &domain.User{LockedUntil: nil}
	if u.IsLocked() {
		t.Error("expected IsLocked=false when LockedUntil is nil")
	}
}

func TestUser_IsLocked_FutureTime(t *testing.T) {
	future := time.Now().Add(30 * time.Minute)
	u := &domain.User{LockedUntil: &future}
	if !u.IsLocked() {
		t.Error("expected IsLocked=true when LockedUntil is in the future")
	}
}

func TestUser_IsLocked_PastTime(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute)
	u := &domain.User{LockedUntil: &past}
	if u.IsLocked() {
		t.Error("expected IsLocked=false when lock has expired")
	}
}

func TestUser_IsLocked_MarginPast(t *testing.T) {
	slightlyPast := time.Now().Add(-time.Nanosecond)
	u := &domain.User{LockedUntil: &slightlyPast}
	if u.IsLocked() {
		t.Error("expected IsLocked=false when LockedUntil is marginally in the past")
	}
}

// ---------------------------------------------------------------------------
// NormalizeEmail
// ---------------------------------------------------------------------------

func TestNormalizeEmail_LowercasesInput(t *testing.T) {
	norm, err := domain.NormalizeEmail("User@Example.COM")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != "user@example.com" {
		t.Errorf("got %q, want user@example.com", norm)
	}
}

func TestNormalizeEmail_TrimsWhitespace(t *testing.T) {
	norm, err := domain.NormalizeEmail("  alice@example.com  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != "alice@example.com" {
		t.Errorf("got %q, want alice@example.com", norm)
	}
}

func TestNormalizeEmail_TrimsAndLowercases(t *testing.T) {
	norm, err := domain.NormalizeEmail("  Bob.Smith+Test@GMAIL.COM  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if norm != "bob.smith+test@gmail.com" {
		t.Errorf("got %q, want bob.smith+test@gmail.com", norm)
	}
}

func TestNormalizeEmail_InvalidFormat_ReturnsError(t *testing.T) {
	cases := []string{"not-an-email", "@nodomain", "missingatdomain", ""}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			_, err := domain.NormalizeEmail(input)
			if err == nil {
				t.Errorf("expected error for %q", input)
			}
		})
	}
}

func TestNormalizeEmail_ValidEmail_NoError(t *testing.T) {
	cases := []string{"user@example.com", "a+b@sub.domain.org", "test.user@co.io"}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			_, err := domain.NormalizeEmail(input)
			if err != nil {
				t.Errorf("unexpected error for %q: %v", input, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SlugToSchemaName
// ---------------------------------------------------------------------------

func TestSlugToSchemaName_HasTenantPrefix(t *testing.T) {
	type tc struct{ slug, want string }
	cases := []tc{
		{"acme", "tenant_acme"},
		{"acme-corp", "tenant_acme_corp"},
		{"foo123", "tenant_foo123"},
	}
	for _, c := range cases {
		t.Run(c.slug, func(t *testing.T) {
			got := domain.SlugToSchemaName(c.slug)
			if got != c.want {
				t.Errorf("SlugToSchemaName(%q)=%q want %q", c.slug, got, c.want)
			}
		})
	}
}

func TestSlugToSchemaName_ReplacesHyphensWithUnderscores(t *testing.T) {
	got := domain.SlugToSchemaName("hello-world")
	if got != "tenant_hello_world" {
		t.Errorf("expected tenant_hello_world, got %q", got)
	}
}

// ---------------------------------------------------------------------------
// IsValidSchemaName
// ---------------------------------------------------------------------------

func TestIsValidSchemaName_ValidNames(t *testing.T) {
	valid := []string{"tenant_acme", "tenant_acme_corp", "tenant_123", "tenant_my_co"}
	for _, name := range valid {
		t.Run(name, func(t *testing.T) {
			if !domain.IsValidSchemaName(name) {
				t.Errorf("expected IsValidSchemaName(%q)=true", name)
			}
		})
	}
}

func TestIsValidSchemaName_InvalidNames(t *testing.T) {
	invalid := []string{"public", "acme", "tenant-acme", "tenant_", "TENANT_ACME", ""}
	for _, name := range invalid {
		t.Run(name, func(t *testing.T) {
			if domain.IsValidSchemaName(name) {
				t.Errorf("expected IsValidSchemaName(%q)=false", name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// PasswordResetToken.IsValid
// ---------------------------------------------------------------------------

func TestPasswordResetToken_IsValid_UnusedNotExpired(t *testing.T) {
	tok := &domain.PasswordResetToken{Used: false, ExpiresAt: time.Now().Add(10 * time.Minute)}
	if !tok.IsValid() {
		t.Error("expected true for unused non-expired token")
	}
}

func TestPasswordResetToken_IsValid_Used(t *testing.T) {
	tok := &domain.PasswordResetToken{Used: true, ExpiresAt: time.Now().Add(10 * time.Minute)}
	if tok.IsValid() {
		t.Error("expected false for used token")
	}
}

func TestPasswordResetToken_IsValid_Expired(t *testing.T) {
	tok := &domain.PasswordResetToken{Used: false, ExpiresAt: time.Now().Add(-time.Minute)}
	if tok.IsValid() {
		t.Error("expected false for expired token")
	}
}

func TestPasswordResetToken_IsValid_UsedAndExpired(t *testing.T) {
	tok := &domain.PasswordResetToken{Used: true, ExpiresAt: time.Now().Add(-time.Minute)}
	if tok.IsValid() {
		t.Error("expected false for used+expired token")
	}
}

// ---------------------------------------------------------------------------
// EmailVerificationToken.IsValid
// ---------------------------------------------------------------------------

func TestEmailVerificationToken_IsValid_UnusedNotExpired(t *testing.T) {
	tok := &domain.EmailVerificationToken{Used: false, ExpiresAt: time.Now().Add(24 * time.Hour)}
	if !tok.IsValid() {
		t.Error("expected true for unused non-expired verification token")
	}
}

func TestEmailVerificationToken_IsValid_Used(t *testing.T) {
	tok := &domain.EmailVerificationToken{Used: true, ExpiresAt: time.Now().Add(24 * time.Hour)}
	if tok.IsValid() {
		t.Error("expected false for used verification token")
	}
}

func TestEmailVerificationToken_IsValid_Expired(t *testing.T) {
	tok := &domain.EmailVerificationToken{Used: false, ExpiresAt: time.Now().Add(-time.Minute)}
	if tok.IsValid() {
		t.Error("expected false for expired verification token")
	}
}

func TestEmailVerificationToken_IsValid_UsedAndExpired(t *testing.T) {
	tok := &domain.EmailVerificationToken{Used: true, ExpiresAt: time.Now().Add(-time.Minute)}
	if tok.IsValid() {
		t.Error("expected false for used+expired verification token")
	}
}

// ---------------------------------------------------------------------------
// Session.IsValid
// ---------------------------------------------------------------------------

func TestSession_IsValid_ActiveNotRevoked(t *testing.T) {
	s := &domain.Session{IsRevoked: false, ExpiresAt: time.Now().Add(time.Hour)}
	if !s.IsValid() {
		t.Error("expected true for active non-revoked session")
	}
}

func TestSession_IsValid_Revoked(t *testing.T) {
	s := &domain.Session{IsRevoked: true, ExpiresAt: time.Now().Add(time.Hour)}
	if s.IsValid() {
		t.Error("expected false for revoked session")
	}
}

func TestSession_IsValid_Expired(t *testing.T) {
	s := &domain.Session{IsRevoked: false, ExpiresAt: time.Now().Add(-time.Minute)}
	if s.IsValid() {
		t.Error("expected false for expired session")
	}
}
