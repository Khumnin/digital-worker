// pkg/jwtutil/jwt.go
package jwtutil

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Subject  string
	TenantID string
	Roles    []string
	ClientID string
	Scope    string
	TTL      time.Duration
}

type ParsedClaims struct {
	ID        string
	Subject   string
	TenantID  string
	Roles     []string
	ClientID  string
	Scope     string
	IssuedAt  time.Time
	ExpiresAt time.Time
}

type jwtClaims struct {
	jwt.RegisteredClaims
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles,omitempty"`
	ClientID string   `json:"client_id,omitempty"`
	Scope    string   `json:"scope,omitempty"`
}

type Signer interface {
	Sign(claims Claims) (string, error)
}

type Verifier interface {
	Verify(tokenString string) (*ParsedClaims, error)
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type signingKey struct {
	Kid        string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

type KeyStore struct {
	mu        sync.RWMutex
	current   signingKey
	allPublic map[string]*rsa.PublicKey
	issuer    string
	audience  jwt.ClaimStrings
}

func NewKeyStore(issuer string, audience []string, kid string, privateKey *rsa.PrivateKey) *KeyStore {
	ks := &KeyStore{
		issuer:    issuer,
		audience:  jwt.ClaimStrings(audience),
		allPublic: make(map[string]*rsa.PublicKey),
	}
	ks.current = signingKey{
		Kid:        kid,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	ks.allPublic[kid] = &privateKey.PublicKey
	return ks
}

func (ks *KeyStore) Update(kid string, privateKey *rsa.PrivateKey) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.current = signingKey{
		Kid:        kid,
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
	}
	ks.allPublic[kid] = &privateKey.PublicKey
}

func (ks *KeyStore) AddPublicKey(kid string, publicKey *rsa.PublicKey) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	ks.allPublic[kid] = publicKey
}

func (ks *KeyStore) RemovePublicKey(kid string) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	delete(ks.allPublic, kid)
}

func (ks *KeyStore) Sign(claims Claims) (string, error) {
	ks.mu.RLock()
	key := ks.current
	ks.mu.RUnlock()

	now := time.Now()
	jti := "jwt_" + uuid.New().String()

	registered := jwt.RegisteredClaims{
		ID:        jti,
		Subject:   claims.Subject,
		Issuer:    ks.issuer,
		Audience:  ks.audience,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(claims.TTL)),
	}

	internal := jwtClaims{
		RegisteredClaims: registered,
		TenantID:         claims.TenantID,
		Roles:            claims.Roles,
		ClientID:         claims.ClientID,
		Scope:            claims.Scope,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, internal)
	token.Header["kid"] = key.Kid

	signed, err := token.SignedString(key.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}

	return signed, nil
}

func (ks *KeyStore) Verify(tokenString string) (*ParsedClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		kid, ok := t.Header["kid"].(string)
		if !ok || kid == "" {
			return nil, errors.New("JWT missing kid header")
		}

		ks.mu.RLock()
		pubKey, found := ks.allPublic[kid]
		ks.mu.RUnlock()

		if !found {
			return nil, fmt.Errorf("unknown kid: %q", kid)
		}

		return pubKey, nil
	},
		jwt.WithIssuer(ks.issuer),
		jwt.WithAudience(string(ks.audience[0])),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		return nil, fmt.Errorf("verify JWT: %w", err)
	}

	internal, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid JWT claims")
	}

	return &ParsedClaims{
		ID:        internal.ID,
		Subject:   internal.Subject,
		TenantID:  internal.TenantID,
		Roles:     internal.Roles,
		ClientID:  internal.ClientID,
		Scope:     internal.Scope,
		IssuedAt:  internal.IssuedAt.Time,
		ExpiresAt: internal.ExpiresAt.Time,
	}, nil
}

func (ks *KeyStore) JWKS() JWKS {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	keys := make([]JWK, 0, len(ks.allPublic))
	for kid, pubKey := range ks.allPublic {
		keys = append(keys, rsaPublicKeyToJWK(kid, pubKey))
	}
	return JWKS{Keys: keys}
}

func rsaPublicKeyToJWK(kid string, key *rsa.PublicKey) JWK {
	nBytes := key.N.Bytes()
	nEncoded := base64.RawURLEncoding.EncodeToString(nBytes)

	eBig := big.NewInt(int64(key.E))
	eBytes := eBig.Bytes()
	eEncoded := base64.RawURLEncoding.EncodeToString(eBytes)

	return JWK{
		Kty: "RSA",
		Use: "sig",
		Alg: "RS256",
		Kid: kid,
		N:   nEncoded,
		E:   eEncoded,
	}
}

func (ks *KeyStore) MarshalJWKS() ([]byte, error) {
	jwks := ks.JWKS()
	return json.Marshal(jwks)
}
