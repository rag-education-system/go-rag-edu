package loginlimit

import (
	"fmt"
	"testing"
)

func TestLimiterAllowsWithinBudget(t *testing.T) {
	limiter := New()

	for i := 0; i < 10; i++ {
		if !limiter.Allow("127.0.0.1", "user@example.com") {
			t.Fatalf("expected attempt %d to be allowed", i+1)
		}
	}
}

func TestLimiterBlocksByEmail(t *testing.T) {
	limiter := New()

	for i := 0; i < 15; i++ {
		_ = limiter.Allow("10.0.0.1", "abuser@example.com")
	}

	if limiter.Allow("10.0.0.2", "abuser@example.com") {
		t.Fatal("expected email-specific limit to block additional logins")
	}
}

func TestLimiterBlocksByIP(t *testing.T) {
	limiter := New()

	for i := 0; i < 25; i++ {
		_ = limiter.Allow("203.0.113.10", fmt.Sprintf("user%d@example.com", i))
	}

	if limiter.Allow("203.0.113.10", "new@example.com") {
		t.Fatal("expected IP-specific limit to block additional logins")
	}
}
