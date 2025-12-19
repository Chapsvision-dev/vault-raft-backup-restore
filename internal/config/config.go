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
	vaultAddr := getEnvWithDefault("VAULT_ADDR", "")
	if strings.TrimSpace(vaultAddr) == "" {
		vaultAddr = "http://127.0.0.1:8200"
	}

	auth, err := loadAuthConfig()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Provider:  strings.ToLower(getEnvWithDefault("BACKUP_PROVIDER", "azure")),
		VaultAddr: vaultAddr,
		Auth:      auth,

		BackupSource:          getEnvWithDefault("BACKUP_SOURCE", ""),
		BackupTarget:          getEnvWithDefault("BACKUP_TARGET", ""),
		BackupTimestampFormat: getEnvWithDefault("BACKUP_TIMESTAMP_FORMAT", ""),
		RestoreSource:         getEnvWithDefault("RESTORE_SOURCE", ""),
		RestoreTarget:         getEnvWithDefault("RESTORE_TARGET", ""),

		Azure: loadAzureConfig(),

		RetryMaxAttempts:  parseEnvInt("RETRY_MAX_ATTEMPTS", retry.Default.MaxAttempts),
		RetryInitialDelay: parseEnvDuration("RETRY_INITIAL_DELAY", retry.Default.InitialDelay),
		RetryMaxDelay:     parseEnvDuration("RETRY_MAX_DELAY", retry.Default.MaxDelay),
		RetryMultiplier:   parseEnvFloat("RETRY_MULTIPLIER", retry.Default.Multiplier),
		RetryEnableJitter: parseEnvBool("RETRY_JITTER", retry.Default.Jitter),
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// loadAuthConfig parses authentication configuration from environment variables.
func loadAuthConfig() (AuthConfig, error) {
	const defaultJWTPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	method := strings.ToLower(strings.TrimSpace(getEnvWithDefault("VAULT_AUTH_METHOD", "")))
	tokenEnv := strings.TrimSpace(getEnvWithDefault("VAULT_TOKEN", ""))

	if method == "" {
		method = detectAuthMethod(tokenEnv, defaultJWTPath)
		if method == "" {
			return AuthConfig{}, errors.New("no auth method configured: set VAULT_AUTH_METHOD=token with VAULT_TOKEN, or provide a readable VAULT_K8S_JWT_PATH for kubernetes")
		}
	}

	auth := AuthConfig{
		Method:     method,
		Namespace:  strings.TrimSpace(getEnvWithDefault("VAULT_NAMESPACE", "")),
		CACert:     strings.TrimSpace(getEnvWithDefault("VAULT_CACERT", "")),
		CAPath:     strings.TrimSpace(getEnvWithDefault("VAULT_CAPATH", "")),
		SkipVerify: parseEnvBool("VAULT_SKIP_VERIFY", false),
	}

	if err := configureAuthMethod(&auth, method, tokenEnv, defaultJWTPath); err != nil {
		return AuthConfig{}, err
	}

	return auth, nil
}

// detectAuthMethod automatically detects the auth method based on available credentials.
func detectAuthMethod(tokenEnv, defaultJWTPath string) string {
	if tokenEnv != "" {
		return "token"
	}
	if isFileReadable(getEnvWithDefault("VAULT_K8S_JWT_PATH", defaultJWTPath)) {
		return "kubernetes"
	}
	return ""
}

// configureAuthMethod configures the auth method specific fields.
func configureAuthMethod(auth *AuthConfig, method, tokenEnv, defaultJWTPath string) error {
	switch method {
	case "token":
		auth.Token = tokenEnv
		if strings.TrimSpace(auth.Token) == "" {
			return errors.New("auth method token requires VAULT_TOKEN")
		}

	case "kubernetes":
		auth.Mount = strings.TrimSpace(getEnvWithDefault("VAULT_AUTH_MOUNT", "kubernetes"))
		if auth.Mount == "" {
			auth.Mount = "kubernetes"
		}
		auth.Role = strings.TrimSpace(getEnvWithDefault("VAULT_K8S_ROLE", ""))
		if auth.Role == "" {
			return errors.New("auth method kubernetes requires VAULT_K8S_ROLE")
		}
		auth.JWTPath = strings.TrimSpace(getEnvWithDefault("VAULT_K8S_JWT_PATH", defaultJWTPath))
		if !isFileReadable(auth.JWTPath) {
			return errors.New("auth method kubernetes requires a readable VAULT_K8S_JWT_PATH")
		}
		auth.Audience = strings.TrimSpace(getEnvWithDefault("VAULT_K8S_AUDIENCE", ""))

	default:
		return errors.New("unsupported auth method: " + method)
	}
	return nil
}

// loadAzureConfig loads Azure-specific configuration.
func loadAzureConfig() AzureConfig {
	return AzureConfig{
		Account:      getEnvWithDefault("AZURE_STORAGE_ACCOUNT", ""),
		Container:    getEnvWithDefault("AZURE_STORAGE_CONTAINER", ""),
		SASToken:     getEnvWithDefault("AZURE_STORAGE_SAS", ""),
		ClientID:     getEnvWithDefault("AZURE_CLIENT_ID", ""),
		ClientSecret: getEnvWithDefault("AZURE_CLIENT_SECRET", ""),
		TenantID:     getEnvWithDefault("AZURE_TENANT_ID", ""),
	}
}

// getEnvWithDefault returns environment variable value or default.
func getEnvWithDefault(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

// parseEnvInt parses an integer from environment variable.
func parseEnvInt(key string, def int) int {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return n
		}
	}
	return def
}

// parseEnvDuration parses a duration from environment variable.
func parseEnvDuration(key string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// parseEnvFloat parses a float from environment variable.
func parseEnvFloat(key string, def float64) float64 {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			return f
		}
	}
	return def
}

// parseEnvBool parses a boolean from environment variable.
func parseEnvBool(key string, def bool) bool {
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

// isFileReadable checks if a file exists and is readable.
func isFileReadable(path string) bool {
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
