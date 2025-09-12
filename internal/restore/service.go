package restore

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/auth"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/provider"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/vault"
)

// Options controls the restore workflow.
type Options struct {
	// RemoteKey is the provider object key (e.g., "vault/snapshots/2025-09-08T15-42-01Z.snap").
	RemoteKey string
	// LocalPath is where the blob will be downloaded before pushing to Vault.
	// If empty, defaults to "./restored.snap".
	LocalPath string
	// Force uses /snapshot-force (optional).
	Force bool
}

// Run downloads the snapshot blob to a local file, then restores it into Vault (Raft).
func Run(ctx context.Context, cfg config.Config, p provider.Provider, opt Options) error {
	remote := strings.TrimSpace(opt.RemoteKey)
	if remote == "" {
		return fmt.Errorf("restore: remote key is empty (provide RESTORE_SOURCE or CLI arg)")
	}

	local := strings.TrimSpace(opt.LocalPath)
	if local == "" {
		local = "./restored.snap"
	}
	local = filepath.Clean(local)

	// 1) Download from provider to local file
	dlStart := time.Now()
	log.Info().
		Str("action", "download").
		Str("provider", cfg.Provider).
		Str("remote", remote).
		Str("local", local).
		Msg("starting download")
	if err := p.Restore(ctx, remote, local); err != nil {
		log.Error().
			Err(err).
			Str("action", "download").
			Str("provider", cfg.Provider).
			Str("remote", remote).
			Str("local", local).
			Dur("elapsed_ms", time.Since(dlStart)).
			Msg("download failed")
		return fmt.Errorf("download from provider: %w", err)
	}
	log.Info().
		Str("action", "download").
		Str("provider", cfg.Provider).
		Str("remote", remote).
		Str("local", local).
		Dur("elapsed_ms", time.Since(dlStart)).
		Msg("download OK")

	// 2) Acquire Vault token via auth provider
	token, err := auth.AcquireToken(ctx, cfg)
	if err != nil {
		log.Error().
			Err(err).
			Str("action", "restore_auth").
			Str("method", cfg.Auth.Method).
			Msg("vault auth failed")
		return err
	}

	// 3) Push snapshot into Vault (Raft)
	restoreStart := time.Now()
	log.Info().
		Str("action", "vault_restore").
		Str("vault_addr", cfg.VaultAddr).
		Str("local", local).
		Bool("force", opt.Force).
		Msg("starting Vault restore")
	if err := vault.RestoreSnapshot(ctx, cfg.VaultAddr, token, local, opt.Force, cfg.RetryOptions()); err != nil {
		log.Error().
			Err(err).
			Str("action", "vault_restore").
			Str("vault_addr", cfg.VaultAddr).
			Str("local", local).
			Dur("elapsed_ms", time.Since(restoreStart)).
			Msg("vault restore failed")
		return fmt.Errorf("vault restore: %w", err)
	}
	log.Info().
		Str("action", "vault_restore").
		Str("vault_addr", cfg.VaultAddr).
		Str("local", local).
		Dur("elapsed_ms", time.Since(restoreStart)).
		Msg("vault restore OK")

	return nil
}
