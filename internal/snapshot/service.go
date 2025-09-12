package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/auth"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/vault"
)

// Options controls snapshot output and naming.
type Options struct {
	// LocalPath: destination file for the Vault snapshot (default: ./snapshot.snap).
	LocalPath string
	// RemotePrefix: provider prefix/directory; a timestamped filename is appended (default: vault/snapshots).
	RemotePrefix string
	// TimestampFormat: Go time layout for the filename (default: 2006-01-02T15-04-05Z).
	TimestampFormat string
}

// Result contains the produced snapshot file and the upload key.
type Result struct {
	LocalPath string
	RemoteKey string
	Timestamp time.Time
}

// Create takes a Vault Raft snapshot and returns where to upload it (remote key).
func Create(ctx context.Context, cfg config.Config, opt Options) (Result, error) {
	var res Result

	local := strings.TrimSpace(opt.LocalPath)
	if local == "" {
		local = "./snapshot.snap"
	}
	// Fail if parent dir does not exist.
	if dir := filepath.Dir(local); dir != "" && dir != "." {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			return res, fmt.Errorf("directory %q does not exist", dir)
		} else if err != nil {
			return res, fmt.Errorf("stat %q: %w", dir, err)
		}
	}

	// Acquire Vault token via auth provider (token or kubernetes, depending on cfg).
	token, err := auth.AcquireToken(ctx, cfg)
	if err != nil {
		log.Error().
			Err(err).
			Str("action", "snapshot_auth").
			Str("method", cfg.Auth.Method).
			Msg("vault auth failed")
		return res, err
	}

	start := time.Now()
	log.Info().
		Str("action", "vault_snapshot").
		Str("local", local).
		Msg("starting snapshot")
	if err := vault.SaveSnapshot(ctx, cfg.VaultAddr, token, local, cfg.RetryOptions()); err != nil {
		log.Error().
			Err(err).
			Str("action", "vault_snapshot").
			Str("local", local).
			Dur("elapsed_ms", time.Since(start)).
			Msg("snapshot failed")
		return res, fmt.Errorf("vault snapshot: %w", err)
	}
	log.Info().
		Str("action", "vault_snapshot").
		Str("local", local).
		Dur("elapsed_ms", time.Since(start)).
		Msg("snapshot OK")

	// Build remote key "<prefix>/<timestamp>.snap".
	prefix := strings.Trim(strings.TrimSpace(opt.RemotePrefix), "/")
	if prefix == "" {
		prefix = "vault/snapshots"
	}
	ts := time.Now().UTC()
	layout := strings.TrimSpace(opt.TimestampFormat)
	if layout == "" {
		layout = "2006-01-02T15-04-05Z"
	}
	filename := fmt.Sprintf("%s.snap", ts.Format(layout))
	key := filepath.ToSlash(filepath.Join(prefix, filename))

	res.LocalPath = local
	res.RemoteKey = key
	res.Timestamp = ts

	log.Debug().
		Str("action", "build_key").
		Str("prefix", prefix).
		Str("remote_key", key).
		Msg("generated remote key")

	return res, nil
}
