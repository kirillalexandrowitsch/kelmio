package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsWithinLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(2, time.Minute, func() time.Time { return now })

	if result := limiter.Allow("admin"); !result.Allowed {
		t.Fatalf("first Allow() = blocked, want allowed")
	}
	if result := limiter.Allow("admin"); !result.Allowed {
		t.Fatalf("second Allow() = blocked, want allowed")
	}
}

func TestLimiterBlocksAfterLimit(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(1, time.Minute, func() time.Time { return now })

	if result := limiter.Allow("admin"); !result.Allowed {
		t.Fatalf("first Allow() = blocked, want allowed")
	}

	result := limiter.Allow("admin")
	if result.Allowed {
		t.Fatal("second Allow() = allowed, want blocked")
	}
	if result.RetryAfter != time.Minute {
		t.Fatalf("RetryAfter = %s, want 1m", result.RetryAfter)
	}
}

func TestLimiterUsesSeparateIdentifiers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(1, time.Minute, func() time.Time { return now })

	if result := limiter.Allow("admin"); !result.Allowed {
		t.Fatalf("admin first Allow() = blocked, want allowed")
	}
	if result := limiter.Allow("demo_member"); !result.Allowed {
		t.Fatalf("demo first Allow() = blocked, want allowed")
	}
	if result := limiter.Allow("admin"); result.Allowed {
		t.Fatal("admin second Allow() = allowed, want blocked")
	}
}

func TestLimiterResetsIdentifier(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(1, time.Minute, func() time.Time { return now })

	limiter.Allow("admin")
	if result := limiter.Allow("admin"); result.Allowed {
		t.Fatal("second Allow() = allowed, want blocked")
	}

	limiter.Reset("admin")
	if result := limiter.Allow("admin"); !result.Allowed {
		t.Fatal("Allow() after Reset() = blocked, want allowed")
	}
}

func TestLimiterAllowsAfterWindowRollsOver(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(1, time.Minute, func() time.Time { return now })

	limiter.Allow("admin")
	now = now.Add(30 * time.Second)
	blocked := limiter.Allow("admin")
	if blocked.Allowed {
		t.Fatal("Allow() before rollover = allowed, want blocked")
	}
	if blocked.RetryAfter != 30*time.Second {
		t.Fatalf("RetryAfter = %s, want 30s", blocked.RetryAfter)
	}

	now = now.Add(30 * time.Second)
	if result := limiter.Allow("admin"); !result.Allowed {
		t.Fatal("Allow() after rollover = blocked, want allowed")
	}
}

func TestLimiterNormalizesKeys(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.June, 4, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(1, time.Minute, func() time.Time { return now })

	if result := limiter.Allow(" Admin "); !result.Allowed {
		t.Fatal("first Allow() = blocked, want allowed")
	}
	if result := limiter.Allow("admin"); result.Allowed {
		t.Fatal("second Allow() with normalized key = allowed, want blocked")
	}
}
