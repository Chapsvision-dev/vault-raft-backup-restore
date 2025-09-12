package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Options configures exponential backoff for retries.
type Options struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// Default backoff settings used when opts are zero/invalid.
var Default = Options{
	MaxAttempts:  5,
	InitialDelay: 300 * time.Millisecond,
	MaxDelay:     8 * time.Second,
	Multiplier:   2.0,
	Jitter:       true,
}

type IsRetryableFunc func(error) bool

// Do executes fn with retries and exponential backoff until it succeeds,
// context is done, or attempts are exhausted. Returns the last error.
func Do(ctx context.Context, opts Options, isRetryable IsRetryableFunc, fn func(context.Context) error) error {
	if opts.MaxAttempts <= 0 {
		opts = Default
	}
	attempt := 0
	backoff := opts.InitialDelay
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		attempt++
		err := fn(ctx)
		if err == nil {
			return nil
		}
		// Stop if not retryable or attempts exhausted.
		if isRetryable != nil && !isRetryable(err) {
			return err
		}
		if attempt >= opts.MaxAttempts {
			return err
		}

		// Compute sleep with optional jitter.
		sleep := backoff
		if opts.Jitter {
			// +/-20% jitter.
			delta := float64(backoff) * 0.2
			j := (rng.Float64()*2 - 1) * delta
			sleep = time.Duration(math.Max(0, float64(backoff)+j))
		}
		// Cap delay.
		if sleep > opts.MaxDelay {
			sleep = opts.MaxDelay
		}

		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}

		// Next backoff with overflow guard and cap.
		next := time.Duration(float64(backoff) * opts.Multiplier)
		if next < backoff {
			next = backoff
		}
		backoff = next
		if backoff > opts.MaxDelay {
			backoff = opts.MaxDelay
		}
	}
}
