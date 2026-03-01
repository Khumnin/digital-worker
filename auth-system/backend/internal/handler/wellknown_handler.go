// internal/handler/wellknown_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"tigersoft/auth-system/internal/service"
)

// WellKnownHandler serves the JWKS endpoint used by resource servers to
// verify RS256 JWTs without contacting the auth server on every request.
type WellKnownHandler struct {
	jwtSvc service.JWTService
}

// NewWellKnownHandler constructs a WellKnownHandler with its required service dependency.
func NewWellKnownHandler(svc service.JWTService) *WellKnownHandler {
	return &WellKnownHandler{jwtSvc: svc}
}

// JWKS handles GET /.well-known/jwks.json.
// Returns the public key set in JWK format so clients can verify JWT signatures.
func (h *WellKnownHandler) JWKS(c *gin.Context) {
	jwks := h.jwtSvc.JWKS()
	c.JSON(http.StatusOK, jwks)
}
