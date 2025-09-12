package auth

import (
	"context"

	"github.com/rs/zerolog/log"
)

type tokenProvider struct {
	token string
}

func (p *tokenProvider) Acquire(ctx context.Context) (string, error) {
	// Never log the token content.
	if p.token == "" {
		log.Debug().
			Str("action", "auth_acquire").
			Str("method", "token").
			Msg("missing token")
		return "", ErrNoToken
	}
	log.Debug().
		Str("action", "auth_acquire").
		Str("method", "token").
		Msg("token acquired")
	return p.token, nil
}
