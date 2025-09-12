package logx

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitFromEnv configures zerolog using env vars.
// - LOG_LEVEL  : trace|debug|info|warn|error (default: info)
// - LOG_FORMAT : json|console                (default: json)
func InitFromEnv() {
	level := strings.ToLower(getenv("LOG_LEVEL", "info"))
	format := strings.ToLower(getenv("LOG_FORMAT", "json"))

	// Always use UTC timestamps in RFC3339.
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }

	// Set global log level.
	switch level {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn", "warning":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Select output format.
	var logger zerolog.Logger
	if format == "console" {
		cw := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stdout
			w.TimeFormat = time.RFC3339
		})
		logger = zerolog.New(cw).With().Timestamp().Logger()
	} else {
		// Default: structured JSON logs.
		logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
	log.Logger = logger
}

// getenv returns the env var value if set and non-empty, otherwise def.
func getenv(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return def
}
