package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
)

var (
	ErrNoToken = errors.New("no token available for vault auth")
)

// Provider abstracts how we acquire a Vault token (no renew here).
type Provider interface {
	Acquire(ctx context.Context) (string, error)
}

// New selects the provider based on cfg.Auth.Method.
// NOTE: This package never initializes logging; main() does via logx.InitFromEnv().
func New(cfg config.Config) (Provider, error) {
	method := strings.ToLower(strings.TrimSpace(cfg.Auth.Method))
	switch method {
	case "token":
		log.Debug().
			Str("action", "auth_new").
			Str("method", "token").
			Msg("auth provider selected")
		return &tokenProvider{token: strings.TrimSpace(cfg.Auth.Token)}, nil

	case "kubernetes":
		log.Debug().
			Str("action", "auth_new").
			Str("method", "kubernetes").
			Str("mount", cfg.Auth.Mount).
			Str("role", cfg.Auth.Role).
			Msg("auth provider selected (not implemented yet)")
		return newKubernetesProvider(cfg)

	default:
		return nil, errors.New("unsupported auth method: " + method)
	}
}
