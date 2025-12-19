package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/retry"
)

// Vault Raft endpoints.
const (
	pathSnapshotGet   = "/v1/sys/storage/raft/snapshot"
	pathSnapshotPost  = "/v1/sys/storage/raft/snapshot"
	pathSnapshotForce = "/v1/sys/storage/raft/snapshot-force"
)

type httpStatusError struct {
	StatusCode int
	RetryAfter time.Duration
}

func (e httpStatusError) Error() string { return fmt.Sprintf("http status %d", e.StatusCode) }

// parseRetryAfter supports seconds and HTTP-date.
func parseRetryAfter(resp *http.Response) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if s, err := strconv.Atoi(v); err == nil {
			return time.Duration(s) * time.Second
		}
		if t, err := http.ParseTime(v); err == nil {
			return time.Until(t)
		}
	}
	return 0
}

// resolveRedirectURL resolves a Location header (absolute or relative) against the base request URL.
func resolveRedirectURL(base *url.URL, loc string) string {
	if strings.TrimSpace(loc) == "" || base == nil {
		return ""
	}
	u, err := url.Parse(loc)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return u.String()
	}
	return base.ResolveReference(u).String()
}

// discoverLeader queries /v1/sys/leader and returns the leader's API address.
// Falls back to the provided addr if discovery fails or address is empty.
func discoverLeader(ctx context.Context, addr string, client *http.Client) string {
	base := strings.TrimRight(addr, "/")
	u := base + "/v1/sys/leader"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return addr
	}
	resp, err := client.Do(req)
	if err != nil {
		return addr
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return addr
	}

	// Vault can return either flat or wrapped responses depending on context/wrapping.
	var flat struct {
		LeaderAddress string `json:"leader_address"`
		HAEnabled     bool   `json:"ha_enabled"`
		IsSelf        bool   `json:"is_self"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&flat); err == nil && strings.TrimSpace(flat.LeaderAddress) != "" {
		return flat.LeaderAddress
	}

	// If first decode failed or didn't contain leader_address, try the wrapped form.
	req2, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return addr
	}
	resp2, err := client.Do(req2)
	if err != nil {
		return addr
	}
	defer func() { _ = resp2.Body.Close() }()

	if resp2.StatusCode != http.StatusOK {
		return addr
	}
	var wrapped struct {
		Data struct {
			LeaderAddress string `json:"leader_address"`
			HAEnabled     bool   `json:"ha_enabled"`
			IsSelf        bool   `json:"is_self"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&wrapped); err == nil {
		if la := strings.TrimSpace(wrapped.Data.LeaderAddress); la != "" {
			return la
		}
	}
	return addr
}

// isSnapshotRetryable returns true if the error should be retried.
func isSnapshotRetryable(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	var se httpStatusError
	if errors.As(err, &se) {
		// 429/408/5xx/307/308 retryable; 503 (sealed/standby) often transient.
		if se.StatusCode == http.StatusTooManyRequests ||
			se.StatusCode == http.StatusRequestTimeout ||
			se.StatusCode == http.StatusServiceUnavailable ||
			se.StatusCode == http.StatusTemporaryRedirect ||
			se.StatusCode == http.StatusPermanentRedirect ||
			(se.StatusCode >= 500 && se.StatusCode <= 599) {
			return true
		}
	}
	return false
}

// handleSnapshotRedirect handles HTTP redirects to the leader.
func handleSnapshotRedirect(resp *http.Response, urlStr *string, attempt int) error {
	if resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		loc := resolveRedirectURL(resp.Request.URL, resp.Header.Get("Location"))
		if loc != "" {
			log.Debug().
				Int("status", resp.StatusCode).
				Str("location", loc).
				Str("action", "vault_snapshot_get").
				Int("attempt", attempt).
				Msg("redirect to leader")
			*urlStr = loc
			return httpStatusError{StatusCode: resp.StatusCode}
		}
	}
	return nil
}

// writeSnapshotToFile writes the response body to a temp file and renames it.
func writeSnapshotToFile(localFile string, body io.Reader, attempt int) error {
	tmp := localFile + ".part"
	out, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil {
			log.Warn().Err(cerr).Str("action", "vault_snapshot_write").Str("file", tmp).Msg("close file failed")
		}
	}()

	if _, err = io.Copy(out, body); err != nil {
		log.Debug().Err(err).Str("action", "vault_snapshot_write").Int("attempt", attempt).Msg("stream copy error")
		return err
	}
	return os.Rename(tmp, localFile)
}

// SaveSnapshot downloads a Vault Raft snapshot to localFile.
func SaveSnapshot(ctx context.Context, addr, token, localFile string, opts retry.Options) error {
	if strings.TrimSpace(addr) == "" {
		addr = "http://vault-hashicorp.localhost"
	}

	if err := ensureParentDir(localFile); err != nil {
		return err
	}

	startTotal := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Minute}
	addr = discoverLeader(ctx, addr, client)
	urlStr := strings.TrimRight(addr, "/") + pathSnapshotGet

	attempt := 0
	doOnce := func(ctx context.Context) error {
		attempt++
		return executeSnapshotGet(ctx, client, &urlStr, token, localFile, attempt, startTotal)
	}

	err := retry.Do(ctx, opts, isSnapshotRetryable, func(ctx context.Context) error {
		return handleRetryAfter(ctx, doOnce)
	})
	if err != nil {
		log.Error().Err(err).Str("action", "vault_snapshot_get").Int("attempts", attempt).
			Dur("total_elapsed_ms", time.Since(startTotal)).Msg("snapshot download failed")
		return err
	}

	log.Debug().Str("action", "vault_snapshot_get").Int("attempts", attempt).
		Dur("total_elapsed_ms", time.Since(startTotal)).Str("local", localFile).Msg("snapshot download OK")
	return nil
}

// ensureParentDir creates the parent directory if it doesn't exist.
func ensureParentDir(localFile string) error {
	if dir := filepath.Dir(localFile); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// executeSnapshotGet performs a single snapshot GET request.
func executeSnapshotGet(ctx context.Context, client *http.Client, urlStr *string, token, localFile string, attempt int, startTotal time.Time) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, *urlStr, http.NoBody)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Debug().Err(err).Str("action", "vault_snapshot_get").Int("attempt", attempt).Msg("request error")
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := handleSnapshotRedirect(resp, urlStr, attempt); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		retryAfter := parseRetryAfter(resp)
		log.Debug().Int("status", resp.StatusCode).Dur("retry_after", retryAfter).
			Str("action", "vault_snapshot_get").Int("attempt", attempt).Msg("non-200 response")
		return httpStatusError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}

	if err := writeSnapshotToFile(localFile, resp.Body, attempt); err != nil {
		return err
	}

	log.Debug().Str("action", "vault_snapshot_get").Int("attempt", attempt).
		Dur("elapsed_ms", time.Since(startTotal)).Msg("attempt succeeded")
	return nil
}

// handleRetryAfter handles Retry-After header by sleeping before retry.
func handleRetryAfter(ctx context.Context, fn func(context.Context) error) error {
	err := fn(ctx)
	var se httpStatusError
	if errors.As(err, &se) && se.RetryAfter > 0 {
		timer := time.NewTimer(se.RetryAfter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

// RestoreSnapshot uploads a snapshot to Vault Raft.
// If force is true, uses /snapshot-force (optional for DR tests).
func RestoreSnapshot(ctx context.Context, addr, token, localFile string, force bool, opts retry.Options) error {
	if strings.TrimSpace(addr) == "" {
		addr = "http://vault-hashicorp.localhost"
	}

	startTotal := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Minute}
	addr = discoverLeader(ctx, addr, client)

	path := pathSnapshotPost
	if force {
		path = pathSnapshotForce
	}
	urlStr := strings.TrimRight(addr, "/") + path

	attempt := 0
	doOnce := func(ctx context.Context) error {
		attempt++
		return executeSnapshotPost(ctx, client, &urlStr, token, localFile, attempt, startTotal)
	}

	err := retry.Do(ctx, opts, isSnapshotRetryable, func(ctx context.Context) error {
		return handleRetryAfter(ctx, doOnce)
	})
	if err != nil {
		log.Error().Err(err).Str("action", "vault_snapshot_post").Int("attempts", attempt).
			Dur("total_elapsed_ms", time.Since(startTotal)).Msg("vault restore failed")
		return err
	}

	log.Debug().Str("action", "vault_snapshot_post").Int("attempts", attempt).
		Dur("total_elapsed_ms", time.Since(startTotal)).Str("local", localFile).Msg("vault restore OK")
	return nil
}

// executeSnapshotPost performs a single snapshot POST request.
func executeSnapshotPost(ctx context.Context, client *http.Client, urlStr *string, token, localFile string, attempt int, startTotal time.Time) error {
	f, err := os.Open(localFile)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, *urlStr, f)
	if err != nil {
		_ = f.Close()
		return err
	}
	if token != "" {
		req.Header.Set("X-Vault-Token", token)
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	resp, err := client.Do(req)
	_ = f.Close()

	if err != nil {
		log.Debug().Err(err).Str("action", "vault_snapshot_post").Int("attempt", attempt).Msg("request error")
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := handleRestoreRedirect(resp, urlStr, attempt); err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		retryAfter := parseRetryAfter(resp)
		log.Debug().Int("status", resp.StatusCode).Dur("retry_after", retryAfter).
			Str("action", "vault_snapshot_post").Int("attempt", attempt).Msg("non-200/204 response")
		return httpStatusError{StatusCode: resp.StatusCode, RetryAfter: retryAfter}
	}

	log.Debug().Str("action", "vault_snapshot_post").Int("attempt", attempt).
		Dur("elapsed_ms", time.Since(startTotal)).Msg("attempt succeeded")
	return nil
}

// handleRestoreRedirect handles redirects during snapshot restore.
func handleRestoreRedirect(resp *http.Response, urlStr *string, attempt int) error {
	if resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
		loc := resolveRedirectURL(resp.Request.URL, resp.Header.Get("Location"))
		if loc != "" {
			log.Debug().
				Int("status", resp.StatusCode).
				Str("location", loc).
				Str("action", "vault_snapshot_post").
				Int("attempt", attempt).
				Msg("redirect to leader")
			*urlStr = loc
			return httpStatusError{StatusCode: resp.StatusCode}
		}
	}
	return nil
}
