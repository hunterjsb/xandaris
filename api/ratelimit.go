//go:build !js

package api

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements per-key token bucket rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	// Config
	readRate   float64 // tokens per second for GET
	writeRate  float64 // tokens per second for POST
	burstRead  int     // max burst for GET
	burstWrite int     // max burst for POST
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
	capacity  int
	rate      float64
}

// NewRateLimiter creates a rate limiter.
// readRate/writeRate = requests per second. burst = max burst size.
func NewRateLimiter(readRate, writeRate float64, burstRead, burstWrite int) *RateLimiter {
	rl := &RateLimiter{
		buckets:    make(map[string]*bucket),
		readRate:   readRate,
		writeRate:  writeRate,
		burstRead:  burstRead,
		burstWrite: burstWrite,
	}
	// Cleanup stale buckets every 5 minutes
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

// Allow checks if a request should be allowed.
func (rl *RateLimiter) Allow(key string, isWrite bool) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucketKey := key
	if isWrite {
		bucketKey += ":w"
	} else {
		bucketKey += ":r"
	}

	b, exists := rl.buckets[bucketKey]
	if !exists {
		rate := rl.readRate
		cap := rl.burstRead
		if isWrite {
			rate = rl.writeRate
			cap = rl.burstWrite
		}
		b = &bucket{
			tokens:    float64(cap),
			lastCheck: time.Now(),
			capacity:  cap,
			rate:      rate,
		}
		rl.buckets[bucketKey] = b
	}

	// Refill tokens
	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * b.rate
	if b.tokens > float64(b.capacity) {
		b.tokens = float64(b.capacity)
	}
	b.lastCheck = now

	// Try to consume
	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// Middleware wraps an http.Handler with rate limiting.
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for OPTIONS (CORS preflight)
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		// Key: API key or IP address
		key := r.Header.Get("X-API-Key")
		if key == "" {
			key = r.RemoteAddr
		}

		isWrite := r.Method == http.MethodPost

		if !rl.Allow(key, isWrite) {
			w.Header().Set("Retry-After", "1")
			writeErr(w, http.StatusTooManyRequests, "rate limit exceeded")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-10 * time.Minute)
	for key, b := range rl.buckets {
		if b.lastCheck.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}
