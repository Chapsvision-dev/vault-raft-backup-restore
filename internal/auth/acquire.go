package auth

import (
	"context"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
)

// AcquireToken is a convenience for call sites that only need the string token.
func AcquireToken(ctx context.Context, cfg config.Config) (string, error) {
	p, err := New(cfg)
	if err != nil {
		return "", err
	}
	return p.Acquire(ctx)
}
