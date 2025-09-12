package provider

import "context"

// Provider defines the contract for storage backends used by the operator.
// Paths/keys are plain strings so implementations can decide their own format.
type Provider interface {
	// Backup uploads local data (source) to remote storage (target).
	Backup(ctx context.Context, source, target string) error

	// Restore downloads remote data (source) to a local path (target).
	Restore(ctx context.Context, source, target string) error

	// Name returns the provider identifier (e.g. "azure", "s3").
	Name() string
}
