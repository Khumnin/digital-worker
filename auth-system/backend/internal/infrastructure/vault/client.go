// internal/infrastructure/vault/client.go
package vault

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"

	vault "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
	"tigersoft/auth-system/internal/config"
)

// Client wraps the Vault SDK client with auth-system-specific operations.
type Client struct {
	client *vault.Client
	mount  string
	cfg    config.VaultConfig
}

// NewClient creates a new Vault client and authenticates.
// In development, uses the static token. In production, uses AppRole.
func NewClient(cfg config.VaultConfig) (*Client, error) {
	c, err := vault.New(
		vault.WithAddress(cfg.Addr),
	)
	if err != nil {
		return nil, fmt.Errorf("create vault client: %w", err)
	}

	// Authenticate: prefer AppRole (production) over static token (dev).
	if cfg.RoleID != "" && cfg.SecretID != "" {
		// AppRole authentication.
		resp, err := c.Auth.AppRoleLogin(context.Background(),
			schema.AppRoleLoginRequest{
				RoleId:   cfg.RoleID,
				SecretId: cfg.SecretID,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("vault AppRole login: %w", err)
		}
		if err := c.SetToken(resp.Auth.ClientToken); err != nil {
			return nil, fmt.Errorf("set vault token: %w", err)
		}
		slog.Info("authenticated to vault via AppRole")
	} else if cfg.Token != "" {
		if err := c.SetToken(cfg.Token); err != nil {
			return nil, fmt.Errorf("set vault token: %w", err)
		}
		slog.Info("authenticated to vault via static token (dev mode)")
	} else {
		return nil, fmt.Errorf("vault: no authentication method configured (set VAULT_TOKEN or VAULT_ROLE_ID+VAULT_SECRET_ID)")
	}

	return &Client{client: c, mount: cfg.Mount, cfg: cfg}, nil
}

// LoadCurrentSigningKey retrieves the current RS256 signing key from Vault.
// In development (when VAULT_ADDR is empty or JWT_PRIVATE_KEY_PATH is set), loads from file.
func (c *Client) LoadCurrentSigningKey(ctx context.Context) (*rsa.PrivateKey, string, error) {
	// Dev mode: load from file if path is set.
	if keyPath := os.Getenv("JWT_PRIVATE_KEY_PATH"); keyPath != "" {
		privateKey, err := loadPrivateKeyFromFile(keyPath)
		if err != nil {
			return nil, "", fmt.Errorf("load private key from file %q: %w", keyPath, err)
		}
		kid := os.Getenv("JWT_KEY_ID")
		if kid == "" {
			kid = "key-dev-local"
		}
		slog.Info("loaded signing key from file", "path", keyPath, "kid", kid)
		return privateKey, kid, nil
	}

	// Production: load from Vault KV v2.
	secret, err := c.client.Secrets.KvV2Read(ctx, "auth-system/jwt-keys/current",
		vault.WithMountPath(c.mount),
	)
	if err != nil {
		return nil, "", fmt.Errorf("read jwt key from vault: %w", err)
	}

	data := secret.Data.Data
	privatePEM, ok := data["private_key_pem"].(string)
	if !ok {
		return nil, "", fmt.Errorf("vault: missing private_key_pem in jwt-keys/current")
	}
	kid, _ := data["kid"].(string)

	privateKey, err := parsePrivateKey([]byte(privatePEM))
	if err != nil {
		return nil, "", fmt.Errorf("parse private key from vault: %w", err)
	}

	slog.Info("loaded signing key from vault", "kid", kid)
	return privateKey, kid, nil
}

func loadPrivateKeyFromFile(path string) (*rsa.PrivateKey, error) {
	pemBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return parsePrivateKey(pemBytes)
}

func parsePrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS8 first, then PKCS1.
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
		return nil, fmt.Errorf("PKCS8 key is not RSA")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
