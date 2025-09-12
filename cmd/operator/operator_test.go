package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"testing"
	"time"

	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/config"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/provider"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/restore"
	"github.com/Chapsvision-dev/vault-raft-backup-restore/internal/snapshot"
)

/* ----------------------------- test harness ----------------------------- */

type exitPanic struct{ code int }

func patchExit(t *testing.T) func() {
	t.Helper()
	prev := exit
	exit = func(code int) { panic(exitPanic{code}) }
	return func() { exit = prev }
}

func mustExitCode(t *testing.T, fn func()) (code int) {
	t.Helper()
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected os.Exit interception, got no panic")
		}
		if ep, ok := r.(exitPanic); ok {
			code = ep.code
			return
		}
		t.Fatalf("unexpected panic: %#v", r)
	}()
	fn()
	return 0
}

func withArgs(t *testing.T, args []string) func() {
	t.Helper()
	prev := os.Args
	os.Args = append([]string{prev[0]}, args...)
	return func() { os.Args = prev }
}

func withEnv(t *testing.T, kv map[string]string) func() {
	t.Helper()
	prev := map[string]*string{}
	for k, v := range kv {
		if old, ok := os.LookupEnv(k); ok {
			tmp := old
			prev[k] = &tmp
		} else {
			prev[k] = nil
		}
		if err := os.Setenv(k, v); err != nil {
			t.Fatalf("setenv %s: %v", k, err)
		}
	}
	return func() {
		for k, v := range prev {
			if v == nil {
				_ = os.Unsetenv(k)
			} else {
				_ = os.Setenv(k, *v)
			}
		}
	}
}

func captureStdout(t *testing.T) func() string {
	t.Helper()
	old := os.Stdout
	var buf bytes.Buffer
	r, w, _ := os.Pipe()
	os.Stdout = w

	done := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(r)
		close(done)
	}()

	return func() string {
		_ = w.Close()
		<-done
		os.Stdout = old
		return buf.String()
	}
}

func resetSeams() {
	loadConfig = config.Load
	newProvider = provider.New
	snapCreate = snapshot.Create
	restoreRun = restore.Run
}

/* --------------------------------- tests -------------------------------- */

// 1) No args -> prints usage, exit code 2
func TestUsage_NoArgs(t *testing.T) {
	resetSeams()
	defer patchExit(t)()
	defer withArgs(t, []string{})()

	restoreOut := captureStdout(t)
	code := mustExitCode(t, func() { main() })
	out := restoreOut()

	if code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
	if !strings.Contains(out, "Usage:") {
		t.Fatalf("expected usage on stdout, got: %q", out)
	}
}

// 2) Backup: precedence Arg > Env > Default, and options are passed to snapshot.Create
func TestBackup_ArgOverridesEnvAndDefault(t *testing.T) {
	resetSeams()
	defer patchExit(t)()
	defer withArgs(t, []string{"backup", "SRC_ARG", "PFX_ARG"})()
	defer withEnv(t, map[string]string{
		"BACKUP_SOURCE": "SRC_ENV",
		"BACKUP_TARGET": "PFX_ENV",
	})()

	// stub config (with different defaults to detect precedence)
	loadConfig = func() (config.Config, error) {
		return config.Config{
			Provider:              "azure",
			BackupSource:          "SRC_DEF",
			BackupTarget:          "PFX_DEF",
			BackupTimestampFormat: "20060102-150405",
		}, nil
	}

	// dummy provider (won't be used because we stop on snapshot error)
	newProvider = func(_ string, _ any) (provider.Provider, error) {
		return dummyProvider{}, nil
	}

	var gotOpts snapshot.Options
	snapCreate = func(ctx context.Context, cfg config.Config, opts snapshot.Options) (snapshot.Result, error) {
		gotOpts = opts
		// stop execution after capturing
		return snapshot.Result{}, errors.New("stop")
	}

	code := mustExitCode(t, func() { main() })
	if code != 1 {
		t.Fatalf("want exit 1 due to injected snapshot error, got %d", code)
	}
	if gotOpts.LocalPath != "SRC_ARG" || gotOpts.RemotePrefix != "PFX_ARG" {
		t.Fatalf("opts mismatch: got LocalPath=%q RemotePrefix=%q", gotOpts.LocalPath, gotOpts.RemotePrefix)
	}
}

// 3) Restore: uses ENV when no args; values are passed to restore.Run
func TestRestore_UsesEnvWhenNoArgs(t *testing.T) {
	resetSeams()
	defer patchExit(t)()
	defer withArgs(t, []string{"restore"})()
	defer withEnv(t, map[string]string{
		"RESTORE_SOURCE": "RK_ENV",
		"RESTORE_TARGET": "LF_ENV",
	})()

	loadConfig = func() (config.Config, error) {
		return config.Config{
			Provider:      "azure",
			RestoreSource: "RK_DEF",
			RestoreTarget: "LF_DEF",
		}, nil
	}
	newProvider = func(_ string, _ any) (provider.Provider, error) {
		return dummyProvider{}, nil
	}

	var got restore.Options
	restoreRun = func(ctx context.Context, cfg config.Config, p provider.Provider, opts restore.Options) error {
		got = opts
		return errors.New("stop")
	}

	code := mustExitCode(t, func() { main() })
	if code != 1 {
		t.Fatalf("want exit 1 due to injected restore error, got %d", code)
	}
	if got.RemoteKey != "RK_ENV" || got.LocalPath != "LF_ENV" {
		t.Fatalf("opts mismatch: got RemoteKey=%q LocalPath=%q", got.RemoteKey, got.LocalPath)
	}
}

// 4) pickArgOrEnv: precedence Arg > Env > Default
func TestPickArgOrEnv_Precedence(t *testing.T) {
	// Build synthetic argv: program, subcmd, ARGVAL
	defer withArgs(t, []string{"subcmd", "ARGVAL"})() // <-- don't include "operator"
	defer withEnv(t, map[string]string{"MY_ENV": "ENVVAL"})()

	got := pickArgOrEnv(2, "MY_ENV", "DEFVAL")
	if got != "ARGVAL" {
		t.Fatalf("want ARGVAL, got %q", got)
	}

	// Without arg -> gets ENV
	defer withArgs(t, []string{"subcmd"})()
	got = pickArgOrEnv(2, "MY_ENV", "DEFVAL")
	if got != "ENVVAL" {
		t.Fatalf("want ENVVAL, got %q", got)
	}

	// Without arg and env -> default
	defer withEnv(t, map[string]string{"MY_ENV": ""})()
	got = pickArgOrEnv(2, "MY_ENV", "DEFVAL")
	if got != "DEFVAL" {
		t.Fatalf("want DEFVAL, got %q", got)
	}
}

// 5) withSignals: cancels context on SIGTERM
func TestWithSignals_CancelsOnInterrupt(t *testing.T) {
	ctx := withSignals(context.Background())

	// Send SIGINT after a short delay to ensure signal.Notify has been registered.
	time.AfterFunc(100*time.Millisecond, func() {
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt) // ignore error, should work on Linux
	})

	select {
	case <-ctx.Done():
		// context was canceled as expected
	case <-time.After(2 * time.Second): // allow more time in CI
		t.Fatal("context not canceled after os.Interrupt")
	}

	// Reset signal handling for cleanliness
	signal.Reset(os.Interrupt)
}

/* ------------------------------- test fakes ------------------------------ */

type dummyProvider struct{}

func (dummyProvider) Name() string                                            { return "dummy" }
func (dummyProvider) Backup(ctx context.Context, local, remote string) error  { return nil }
func (dummyProvider) Restore(ctx context.Context, remote, local string) error { return nil }
