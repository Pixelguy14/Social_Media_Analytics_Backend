package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket limits the rate at which operations can be performed.
type TokenBucket struct {
	capacity  int
	tokens    float64
	rate      float64 // tokens per second
	lastValid time.Time
	mu        sync.Mutex
}

func NewTokenBucket(capacity int, rate float64) *TokenBucket {
	return &TokenBucket{
		capacity:  capacity,
		tokens:    float64(capacity),
		rate:      rate,
		lastValid: time.Now(),
	}
}

// Allow checks if one operation a token is available. If so, consumes it and returns true.
func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastValid).Seconds()
	
	// Refill based on elapsed time and tokens per second
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}
	tb.lastValid = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Manager holds buckets for different users/IPs
type Manager struct {
	buckets map[string]*TokenBucket
	mu      sync.Mutex
	cap     int
	rate    float64
	drops   int // Count total rate-limited requests
}

func NewManager(capacity int, rate float64) *Manager {
	return &Manager{
		buckets: make(map[string]*TokenBucket),
		cap:     capacity,
		rate:    rate,
		drops:   0,
	}
}

func (m *Manager) Allow(key string) bool {
	m.mu.Lock()
	bucket, exists := m.buckets[key]
	if !exists {
		bucket = NewTokenBucket(m.cap, m.rate)
		m.buckets[key] = bucket
	}
	m.mu.Unlock()

	allowed := bucket.Allow()
	if !allowed {
		m.mu.Lock()
		m.drops++
		m.mu.Unlock()
	}
	return allowed
}

// GetDrops returns the total number of blocked requests
func (m *Manager) GetDrops() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.drops
}
