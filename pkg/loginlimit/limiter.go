package loginlimit

import (
	"strings"
	"sync"
	"time"
)

type bucket struct {
	count   int
	resetAt time.Time
}

// Limiter reduces login/logout abuse per IP and per email.
type Limiter struct {
	mu      sync.Mutex
	byIP    map[string]bucket
	byEmail map[string]bucket
}

func New() *Limiter {
	return &Limiter{
		byIP:    make(map[string]bucket),
		byEmail: make(map[string]bucket),
	}
}

func (l *Limiter) Allow(ip, email string) bool {
	ip = strings.TrimSpace(ip)
	email = strings.ToLower(strings.TrimSpace(email))
	if ip == "" {
		ip = "unknown"
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.cleanup(now)

	if !l.consume(l.byIP, "ip:"+ip, 25, 15*time.Minute, now) {
		return false
	}

	if email != "" && !l.consume(l.byEmail, "email:"+email, 15, 15*time.Minute, now) {
		return false
	}

	return true
}

func (l *Limiter) consume(store map[string]bucket, key string, max int, window time.Duration, now time.Time) bool {
	entry, exists := store[key]
	if !exists || now.After(entry.resetAt) {
		store[key] = bucket{count: 1, resetAt: now.Add(window)}
		return true
	}

	if entry.count >= max {
		return false
	}

	entry.count++
	store[key] = entry
	return true
}

func (l *Limiter) cleanup(now time.Time) {
	for key, entry := range l.byIP {
		if now.After(entry.resetAt) {
			delete(l.byIP, key)
		}
	}
	for key, entry := range l.byEmail {
		if now.After(entry.resetAt) {
			delete(l.byEmail, key)
		}
	}
}
