package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/retry"
)

type Config struct {
	Provider  string
	VaultAddr string
	Auth      AuthConfig

	// Back/restore I/O
	BackupSource          string
	BackupTarget          string
	BackupTimestampFormat string
	RestoreSource         string
	RestoreTarget         string

	Azure AzureConfig

	RetryMaxAttempts  int
	RetryInitialDelay time.Duration
	RetryMaxDelay     time.Duration
	RetryMultiplier   float64
	RetryEnableJitter bool
}

type AzureConfig struct {
	Account   string
	Container string
	SASToken  string

	ClientID     string
	ClientSecret string
	TenantID     string
}

type AuthConfig struct {
	Method     string // "token" or "kubernetes"
	Token      string // only if Method == token
	Mount      string // default "kubernetes"
	Role       string // required if Method == kubernetes
	JWTPath    string // default /var/run/secrets/kubernetes.io/serviceaccount/token
	Audience   string // optional, for projected SA tokens
	Namespace  string // optional, Vault Enterprise namespace
	CACert     string // optional
	CAPath     string // optional
	SkipVerify bool   // optional
}

// Load reads config from environment variables, applies defaults and validates.
func Load() (Config, error) {
	get := func(key, def string) string {
		if v, ok := os.LookupEnv(key); ok {
			return v
		}
		return def
	}

	parseInt := func(key string, def int) int {
		if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
			if n, err := strconv.Atoi(v); err == nil && n >= 0 {
				return n
			}
		}
		return def
	}

	parseDur := func(key string, def time.Duration) time.Duration {
		if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
			if d, err := time.ParseDuration(v); err == nil {
				return d
			}
		}
		return def
	}

	parseFloat := func(key string, def float64) float64 {
		if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
			if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
				return f
			}
		}
		return def
	}

	parseBool := func(key string, def bool) bool {
		if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
			switch strings.ToLower(v) {
			case "1", "true", "yes", "y", "on":
				return true
			case "0", "false", "no", "n", "off":
				return false
			}
		}
		return def
	}

	fileReadable := func(path string) bool {
		if strings.TrimSpace(path) == "" {
			return false
		}
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		_ = f.Close()
		return true
	}

	// Vault address with default
	vaultAddr := get("VAULT_ADDR", "")
	if strings.TrimSpace(vaultAddr) == "" {
		vaultAddr = "http://127.0.0.1:8200"
	}

	// -------------------------
	// Auth parsing (fallbacks)
	// -------------------------
	const defaultJWTPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	method := strings.ToLower(strings.TrimSpace(get("VAULT_AUTH_METHOD", "")))
	tokenEnv := strings.TrimSpace(get("VAULT_TOKEN", ""))

	if method == "" {
		switch {
		case tokenEnv != "":
			method = "token"
		case fileReadable(get("VAULT_K8S_JWT_PATH", defaultJWTPath)):
			method = "kubernetes"
		default:
			return Config{}, errors.New("no auth method configured: set VAULT_AUTH_METHOD=token with VAULT_TOKEN, or provide a readable VAULT_K8S_JWT_PATH for kubernetes")
		}
	}

	auth := AuthConfig{
		Method:     method,
		Namespace:  strings.TrimSpace(get("VAULT_NAMESPACE", "")),
		CACert:     strings.TrimSpace(get("VAULT_CACERT", "")),
		CAPath:     strings.TrimSpace(get("VAULT_CAPATH", "")),
		SkipVerify: parseBool("VAULT_SKIP_VERIFY", false),
	}

	switch method {
	case "token":
		auth.Token = tokenEnv
		if strings.TrimSpace(auth.Token) == "" {
			return Config{}, errors.New("auth method token requires VAULT_TOKEN")
		}

	case "kubernetes":
		auth.Mount = strings.TrimSpace(get("VAULT_AUTH_MOUNT", "kubernetes"))
		if auth.Mount == "" {
			auth.Mount = "kubernetes"
		}
		auth.Role = strings.TrimSpace(get("VAULT_K8S_ROLE", ""))
		if auth.Role == "" {
			return Config{}, errors.New("auth method kubernetes requires VAULT_K8S_ROLE")
		}
		auth.JWTPath = strings.TrimSpace(get("VAULT_K8S_JWT_PATH", defaultJWTPath))
		if !fileReadable(auth.JWTPath) {
			return Config{}, errors.New("auth method kubernetes requires a readable VAULT_K8S_JWT_PATH")
		}
		auth.Audience = strings.TrimSpace(get("VAULT_K8S_AUDIENCE", ""))

	default:
		return Config{}, errors.New("unsupported auth method: " + method)
	}

	cfg := Config{
		Provider:  strings.ToLower(get("BACKUP_PROVIDER", "azure")),
		VaultAddr: vaultAddr,
		Auth:      auth,
		// compat: if your Config still has VaultToken, mirror it for existing call sites
		// VaultToken:            auth.Token,

		BackupSource:          get("BACKUP_SOURCE", ""),
		BackupTarget:          get("BACKUP_TARGET", ""),
		BackupTimestampFormat: get("BACKUP_TIMESTAMP_FORMAT", ""),
		RestoreSource:         get("RESTORE_SOURCE", ""),
		RestoreTarget:         get("RESTORE_TARGET", ""),

		Azure: AzureConfig{
			Account:      get("AZURE_STORAGE_ACCOUNT", ""),
			Container:    get("AZURE_STORAGE_CONTAINER", ""),
			SASToken:     get("AZURE_STORAGE_SAS", ""),
			ClientID:     get("AZURE_CLIENT_ID", ""),
			ClientSecret: get("AZURE_CLIENT_SECRET", ""),
			TenantID:     get("AZURE_TENANT_ID", ""),
		},

		RetryMaxAttempts:  parseInt("RETRY_MAX_ATTEMPTS", retry.Default.MaxAttempts),
		RetryInitialDelay: parseDur("RETRY_INITIAL_DELAY", retry.Default.InitialDelay),
		RetryMaxDelay:     parseDur("RETRY_MAX_DELAY", retry.Default.MaxDelay),
		RetryMultiplier:   parseFloat("RETRY_MULTIPLIER", retry.Default.Multiplier),
		RetryEnableJitter: parseBool("RETRY_JITTER", retry.Default.Jitter),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validate checks provider-specific requirements.
// For Azure: must have Account+Container and either SAS or Service Principal (or MSI if present in your providers).
func (c *Config) validate() error {
	switch c.Provider {
	case "azure":
		if c.Azure.Account == "" || c.Azure.Container == "" {
			return errors.New("azure: AZURE_STORAGE_ACCOUNT and AZURE_STORAGE_CONTAINER are required")
		}
		// Accept SAS or SP (ClientID/Secret/Tenant). If neither, we still allow for MSI in provider impl.
	default:
		return errors.New("unsupported provider: " + c.Provider)
	}
	return nil
}

// RetryOptions converts retry-related config values to retry.Options.
func (c Config) RetryOptions() retry.Options {
	return retry.Options{
		MaxAttempts:  c.RetryMaxAttempts,
		InitialDelay: c.RetryInitialDelay,
		MaxDelay:     c.RetryMaxDelay,
		Multiplier:   c.RetryMultiplier,
		Jitter:       c.RetryEnableJitter,
	}
}
