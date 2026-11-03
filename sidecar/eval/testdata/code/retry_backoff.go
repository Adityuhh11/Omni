// Package retry runs a function again after a failure, waiting twice as long each time.
package retry

import (
	"context"
	"time"
)

// Do calls fn up to attempts times, sleeping base, 2*base, 4*base... between failures, and stops early when ctx ends.
func Do(ctx context.Context, attempts int, base time.Duration, fn func() error) error {
	var err error
	delay := base
	for i := 0; i < attempts; i++ {
		if err = fn(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			delay *= 2
		}
	}
	return err
}
