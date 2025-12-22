package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/logx"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/provider"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/restore"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/snapshot"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/version"

	_ "github.com/Chapsvision-dev/vault-raft-backup-restore/internal/provider/azure"
)

// Test seams — overridden in unit tests. Keep signatures in sync with packages.
var (
	loadConfig  func() (config.Config, error)                                                   = config.Load
	newProvider func(name string, cfg any) (provider.Provider, error)                           = provider.New
	snapCreate  func(context.Context, config.Config, snapshot.Options) (snapshot.Result, error) = snapshot.Create
	restoreRun  func(context.Context, config.Config, provider.Provider, restore.Options) error  = restore.Run
	exit        func(int)                                                                       = os.Exit
)

const usage = `
Usage:
  operator backup  [source] [targetPrefix]
  operator restore [remoteKey] [localFile]
  operator version | --version | -v
  operator help    | --help    | -h

Notes:
  - You can also set env vars:
      BACKUP_SOURCE, BACKUP_TARGET, RESTORE_SOURCE, RESTORE_TARGET
  - Provider is selected with BACKUP_PROVIDER (default: azure).
  - Vault address/token: VAULT_ADDR (default http://vault-hashicorp.localhost), VAULT_TOKEN
`

// main wires CLI -> config -> provider -> backup/restore.
// Exit codes: 0 success, 1 runtime error, 2 usage error.
func main() {
	_ = godotenv.Load() // best-effort
	logx.InitFromEnv()

	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Print(usage)
		exit(2)
	}
	action := strings.ToLower(args[0])

	// Handle version command
	if action == "version" || action == "--version" || action == "-v" {
		fmt.Printf("vault-raft-backup-operator %s\n", version.Info())
		exit(0)
	}

	// Handle help command
	if action == "help" || action == "--help" || action == "-h" {
		fmt.Print(usage)
		exit(0)
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Error().Err(err).Msg("config error")
		exit(1)
	}

	// Build provider from config.
	p, err := newProvider(cfg.Provider, cfg)
	if err != nil {
		log.Error().Err(err).Str("provider", cfg.Provider).Msg("provider init error")
		exit(1)
	}

	ctx := withSignals(context.Background())

	switch action {
	case "backup":
		source := pickArgOrEnv(2, "BACKUP_SOURCE", cfg.BackupSource)
		targetPrefix := pickArgOrEnv(3, "BACKUP_TARGET", cfg.BackupTarget)

		start := time.Now()
		res, err := snapCreate(ctx, cfg, snapshot.Options{
			LocalPath:       source,
			RemotePrefix:    targetPrefix,
			TimestampFormat: cfg.BackupTimestampFormat,
		})
		if err != nil {
			log.Error().Err(err).Str("action", "snapshot").Msg("snapshot failed")
			exit(1)
		}
		log.Info().
			Str("action", "snapshot").
			Str("local", res.LocalPath).
			Str("remote", res.RemoteKey).
			Dur("elapsed_ms", time.Since(start)).
			Msg("vault raft snapshot OK")

		upStart := time.Now()
		if err := p.Backup(ctx, res.LocalPath, res.RemoteKey); err != nil {
			log.Error().Err(err).Str("action", "upload").Str("remote", res.RemoteKey).Msg("upload failed")
			exit(1)
		}
		log.Info().
			Str("action", "upload").
			Str("provider", cfg.Provider).
			Str("remote", res.RemoteKey).
			Dur("elapsed_ms", time.Since(upStart)).
			Msg("backup OK")

	case "restore":
		source := pickArgOrEnv(2, "RESTORE_SOURCE", cfg.RestoreSource) // remote key
		target := pickArgOrEnv(3, "RESTORE_TARGET", cfg.RestoreTarget) // local file (optional)

		start := time.Now()
		// Force restore can be toggled via env if you veux (OPTIONAL): VAULT_SNAPSHOT_FORCE=true
		force := strings.EqualFold(os.Getenv("VAULT_SNAPSHOT_FORCE"), "true")
		if err := restoreRun(ctx, cfg, p, restore.Options{
			RemoteKey: source,
			LocalPath: target,
			Force:     force,
		}); err != nil {
			log.Error().Err(err).Str("action", "restore").Str("remote", source).Msg("restore failed")
			exit(1)
		}
		log.Info().
			Str("action", "restore").
			Str("provider", cfg.Provider).
			Str("remote", source).
			Dur("elapsed_ms", time.Since(start)).
			Msg("restore OK")

	default:
		fmt.Print(usage)
		exit(2)
	}
}

func pickArgOrEnv(idx int, env string, def string) string {
	if len(os.Args) > idx && os.Args[idx] != "" {
		return os.Args[idx]
	}
	if v, ok := os.LookupEnv(env); ok && v != "" {
		return v
	}
	return def
}

func withSignals(parent context.Context) context.Context {
	ctx, cancel := context.WithCancel(parent)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		<-ch
		cancel()
	}()
	return ctx
}
