// internal/service/jwt_service.go
package service

import (
	"fmt"

	"tigersoft/auth-system/pkg/jwtutil"
)

// JWTService exposes the public JWKS endpoint data so that resource servers
// can fetch signing keys for JWT verification.
type JWTService interface {
	JWKS() jwtutil.JWKS
	MarshalJWKS() ([]byte, error)
}

type jwtServiceImpl struct {
	keyStore *jwtutil.KeyStore
}

// NewJWTService constructs a JWTService backed by the given KeyStore.
func NewJWTService(keyStore *jwtutil.KeyStore) JWTService {
	return &jwtServiceImpl{
		keyStore: keyStore,
	}
}

// JWKS returns the current set of public JSON Web Keys.
func (s *jwtServiceImpl) JWKS() jwtutil.JWKS {
	return s.keyStore.JWKS()
}

// MarshalJWKS serialises the JWKS to JSON, ready for serving at /.well-known/jwks.json.
func (s *jwtServiceImpl) MarshalJWKS() ([]byte, error) {
	data, err := s.keyStore.MarshalJWKS()
	if err != nil {
		return nil, fmt.Errorf("marshal JWKS: %w", err)
	}

	return data, nil
}
