// Package ratelimit limits how often an action may happen with a token bucket.
package ratelimit

import (
	"sync"
	"time"
)

// Bucket refills at rate tokens per second up to capacity; Allow spends one token.
type Bucket struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	rate     float64
	last     time.Time
}

func New(capacity, rate float64) *Bucket {
	return &Bucket{tokens: capacity, capacity: capacity, rate: rate, last: time.Now()}
}

func (b *Bucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	b.tokens = min(b.capacity, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
