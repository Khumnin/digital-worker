// internal/infrastructure/vault/key_rotation.go
package vault

import (
	"context"
	"log/slog"
	"time"

	"tigersoft/auth-system/pkg/jwtutil"
)

// KeyRotationWatcher polls Vault for signing key updates and rotates the KeyStore.
type KeyRotationWatcher struct {
	vaultClient  *Client
	keyStore     *jwtutil.KeyStore
	pollInterval time.Duration
}

// NewKeyRotationWatcher creates a watcher that polls for key rotation.
func NewKeyRotationWatcher(client *Client, keyStore *jwtutil.KeyStore, interval time.Duration) *KeyRotationWatcher {
	return &KeyRotationWatcher{
		vaultClient:  client,
		keyStore:     keyStore,
		pollInterval: interval,
	}
}

// Watch starts the background key rotation polling loop.
// It runs until the context is cancelled (e.g., on server shutdown).
func (w *KeyRotationWatcher) Watch(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	slog.Info("key rotation watcher started", "interval", w.pollInterval)

	for {
		select {
		case <-ctx.Done():
			slog.Info("key rotation watcher stopped")
			return
		case <-ticker.C:
			if err := w.rotate(ctx); err != nil {
				slog.Error("key rotation check failed", "error", err)
				// Continue — do not stop the watcher on transient errors.
			}
		}
	}
}

func (w *KeyRotationWatcher) rotate(ctx context.Context) error {
	newKey, kid, err := w.vaultClient.LoadCurrentSigningKey(ctx)
	if err != nil {
		return err
	}

	w.keyStore.Update(kid, newKey)
	slog.Info("signing key checked", "kid", kid)
	return nil
}
