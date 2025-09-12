package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
)

// kubernetesProvider implements Vault auth using the Kubernetes method.
type kubernetesProvider struct {
	cfg  config.AuthConfig
	addr string
}

// newKubernetesProvider validates configuration and returns a provider.
// Role and JWT path are mandatory.
func newKubernetesProvider(cfg config.Config) (*kubernetesProvider, error) {
	if strings.TrimSpace(cfg.Auth.Role) == "" {
		return nil, errors.New("kubernetes auth requires role")
	}
	if strings.TrimSpace(cfg.Auth.JWTPath) == "" {
		return nil, errors.New("kubernetes auth requires jwt path")
	}
	return &kubernetesProvider{cfg: cfg.Auth, addr: cfg.VaultAddr}, nil
}

// Acquire exchanges a Kubernetes ServiceAccount JWT for a Vault client token.
func (p *kubernetesProvider) Acquire(ctx context.Context) (string, error) {
	// Read the projected ServiceAccount JWT.
	jwt, err := os.ReadFile(p.cfg.JWTPath)
	if err != nil {
		return "", fmt.Errorf("read jwt: %w", err)
	}

	// Build login request payload.
	url := fmt.Sprintf("%s/v1/auth/%s/login", strings.TrimRight(p.addr, "/"), p.cfg.Mount)
	body := map[string]string{
		"role": p.cfg.Role,
		"jwt":  strings.TrimSpace(string(jwt)),
	}
	if p.cfg.Audience != "" {
		body["audience"] = p.cfg.Audience
	}
	reqBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	// Prepare HTTP request.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	// Send the login request.
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vault login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle non-200 responses with a trimmed body snippet.
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("vault login failed: %s (%s)", resp.Status, strings.TrimSpace(string(data)))
	}

	// Decode response and extract client token.
	var out struct {
		Auth struct {
			ClientToken string `json:"client_token"`
		} `json:"auth"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("decode vault response: %w", err)
	}
	if out.Auth.ClientToken == "" {
		return "", errors.New("vault login: empty client_token")
	}

	// Log success.
	log.Info().
		Str("action", "auth_acquire").
		Str("method", "kubernetes").
		Str("mount", p.cfg.Mount).
		Str("role", p.cfg.Role).
		Msg("kubernetes login OK")

	return out.Auth.ClientToken, nil
}
