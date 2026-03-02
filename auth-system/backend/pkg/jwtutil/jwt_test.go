// pkg/jwtutil/jwt_test.go
package jwtutil_test

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"tigersoft/auth-system/pkg/jwtutil"
)

func generateTestKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	return key
}

func TestGenerateRSAKey_ProducesValidKey(t *testing.T) {
	key := generateTestKey(t)
	if key == nil {
		t.Fatal("expected non-nil RSA private key")
	}
	if key.N == nil {
		t.Error("expected non-nil modulus N")
	}
	if key.E == 0 {
		t.Error("expected non-zero public exponent E")
	}
	if err := key.Validate(); err != nil {
		t.Errorf("RSA key failed self-validation: %v", err)
	}
}

func TestNewKeyStore_ReturnsNonNil(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	if ks == nil {
		t.Fatal("NewKeyStore returned nil")
	}
}

func TestSign_ReturnsNonEmptyToken(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	claims := jwtutil.Claims{
		Subject:  "user-123",
		TenantID: "tenant-abc",
		Roles:    []string{"member"},
		TTL:      15 * time.Minute,
	}
	token, err := ks.Sign(claims)
	if err != nil {
		t.Fatalf("Sign returned error: %v", err)
	}
	if token == "" {
		t.Error("Sign returned empty token string")
	}
}

func TestSign_TokenHasThreeJWTParts(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	token, err := ks.Sign(jwtutil.Claims{Subject: "u1", TenantID: "t1", TTL: 5 * time.Minute})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("expected JWT with 3 parts, got %d", len(parts))
	}
}

func TestVerify_ValidToken_ReturnsClaims(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	input := jwtutil.Claims{
		Subject:  "user-456",
		TenantID: "tenant-xyz",
		Roles:    []string{"admin", "member"},
		ClientID: "client-1",
		Scope:    "openid profile",
		TTL:      15 * time.Minute,
	}
	tokenStr, err := ks.Sign(input)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parsed, err := ks.Verify(tokenStr)
	if err != nil {
		t.Fatalf("Verify returned unexpected error: %v", err)
	}
	t.Run("subject matches", func(t *testing.T) {
		if parsed.Subject != input.Subject {
			t.Errorf("want Subject=%q, got %q", input.Subject, parsed.Subject)
		}
	})
	t.Run("tenant_id matches", func(t *testing.T) {
		if parsed.TenantID != input.TenantID {
			t.Errorf("want TenantID=%q, got %q", input.TenantID, parsed.TenantID)
		}
	})
	t.Run("roles match", func(t *testing.T) {
		if len(parsed.Roles) != len(input.Roles) {
			t.Fatalf("want %d roles, got %d", len(input.Roles), len(parsed.Roles))
		}
	})
	t.Run("client_id matches", func(t *testing.T) {
		if parsed.ClientID != input.ClientID {
			t.Errorf("want ClientID=%q, got %q", input.ClientID, parsed.ClientID)
		}
	})
	t.Run("scope matches", func(t *testing.T) {
		if parsed.Scope != input.Scope {
			t.Errorf("want Scope=%q, got %q", input.Scope, parsed.Scope)
		}
	})
	t.Run("ID is non-empty", func(t *testing.T) {
		if parsed.ID == "" {
			t.Error("expected non-empty JWT ID (jti)")
		}
	})
	t.Run("ExpiresAt is in the future", func(t *testing.T) {
		if !parsed.ExpiresAt.After(time.Now()) {
			t.Errorf("expected ExpiresAt in the future, got %v", parsed.ExpiresAt)
		}
	})
}

func TestVerify_ExpiredToken_ReturnsError(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	tokenStr, err := ks.Sign(jwtutil.Claims{
		Subject:  "user-expired",
		TenantID: "t1",
		TTL:      -1 * time.Second,
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_, err = ks.Verify(tokenStr)
	if err == nil {
		t.Error("expected Verify to return error for expired token, got nil")
	}
}

func TestVerify_WrongKey_ReturnsError(t *testing.T) {
	key1 := generateTestKey(t)
	key2 := generateTestKey(t)
	signer := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key1)
	verifier := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key2)
	tokenStr, err := signer.Sign(jwtutil.Claims{
		Subject:  "user-wrongkey",
		TenantID: "t1",
		TTL:      15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	_, err = verifier.Verify(tokenStr)
	if err == nil {
		t.Error("expected Verify to return error when signed with a different key")
	}
}

func TestVerify_TamperedToken_ReturnsError(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	tokenStr, err := ks.Sign(jwtutil.Claims{Subject: "u1", TenantID: "t1", TTL: 15 * time.Minute})
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	parts := strings.Split(tokenStr, ".")
	parts[2] = parts[2] + "AAAA"
	tampered := strings.Join(parts, ".")
	_, err = ks.Verify(tampered)
	if err == nil {
		t.Error("expected Verify to return error for tampered token")
	}
}

func TestMarshalJWKS_ReturnsValidJSON(t *testing.T) {
	key := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key)
	data, err := ks.MarshalJWKS()
	if err != nil {
		t.Fatalf("MarshalJWKS: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("MarshalJWKS returned empty bytes")
	}
	var jwks jwtutil.JWKS
	if err := json.Unmarshal(data, &jwks); err != nil {
		t.Fatalf("MarshalJWKS output is not valid JSON: %v", err)
	}
	t.Run("keys array is present", func(t *testing.T) {
		if jwks.Keys == nil {
			t.Error("expected non-nil keys array in JWKS")
		}
	})
	t.Run("contains exactly one key", func(t *testing.T) {
		if len(jwks.Keys) != 1 {
			t.Errorf("expected 1 key in JWKS, got %d", len(jwks.Keys))
		}
	})
	t.Run("key has correct fields", func(t *testing.T) {
		k := jwks.Keys[0]
		if k.Kty != "RSA" {
			t.Errorf("expected kty=RSA, got %q", k.Kty)
		}
		if k.Use != "sig" {
			t.Errorf("expected use=sig, got %q", k.Use)
		}
		if k.Alg != "RS256" {
			t.Errorf("expected alg=RS256, got %q", k.Alg)
		}
		if k.Kid != "kid-1" {
			t.Errorf("expected kid=kid-1, got %q", k.Kid)
		}
		if k.N == "" {
			t.Error("expected non-empty N (modulus)")
		}
		if k.E == "" {
			t.Error("expected non-empty E (exponent)")
		}
	})
}

func TestMarshalJWKS_TwoKeys_BothPresent(t *testing.T) {
	key1 := generateTestKey(t)
	key2 := generateTestKey(t)
	ks := jwtutil.NewKeyStore("https://auth.example.com", []string{"api"}, "kid-1", key1)
	ks.AddPublicKey("kid-2", &key2.PublicKey)
	data, err := ks.MarshalJWKS()
	if err != nil {
		t.Fatalf("MarshalJWKS: %v", err)
	}
	var jwks jwtutil.JWKS
	if err := json.Unmarshal(data, &jwks); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	if len(jwks.Keys) != 2 {
		t.Errorf("expected 2 keys in JWKS after AddPublicKey, got %d", len(jwks.Keys))
	}
}
