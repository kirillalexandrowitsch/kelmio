package ratelimit

import (
	"strings"
	"sync"
	"time"
)

type Clock func() time.Time

type Result struct {
	Allowed    bool
	RetryAfter time.Duration
}

type Limiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	now     Clock
	buckets map[string]bucket
}

type bucket struct {
	count      int
	windowEnds time.Time
}

func NewLimiter(limit int, window time.Duration, now Clock) *Limiter {
	if limit < 1 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	if now == nil {
		now = time.Now
	}

	return &Limiter{
		limit:   limit,
		window:  window,
		now:     now,
		buckets: make(map[string]bucket),
	}
}

func (l *Limiter) Allow(key string) Result {
	key = normalizeKey(key)
	if key == "" {
		return Result{Allowed: true}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	current := l.buckets[key]
	if current.windowEnds.IsZero() || !now.Before(current.windowEnds) {
		current = bucket{
			count:      0,
			windowEnds: now.Add(l.window),
		}
	}

	if current.count >= l.limit {
		retryAfter := current.windowEnds.Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		l.buckets[key] = current
		return Result{
			Allowed:    false,
			RetryAfter: retryAfter,
		}
	}

	current.count++
	l.buckets[key] = current
	return Result{Allowed: true}
}

func (l *Limiter) Reset(key string) {
	key = normalizeKey(key)
	if key == "" {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.buckets, key)
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
