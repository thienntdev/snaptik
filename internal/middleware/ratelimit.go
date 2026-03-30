package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thienntdev/snaptiktok/internal/config"
)

// visitor tracks rate limit state for a single IP
type visitor struct {
	tokens    float64
	lastSeen  time.Time
}

// RateLimiter implements a token bucket rate limiter per IP
type RateLimiter struct {
	visitors  map[string]*visitor
	mu        sync.RWMutex
	maxTokens float64
	refillRate float64 // tokens per second
	cleanup    time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(cfg *config.Config) *RateLimiter {
	rl := &RateLimiter{
		visitors:   make(map[string]*visitor),
		maxTokens:  float64(cfg.RateLimitMax),
		refillRate: float64(cfg.RateLimitMax) / cfg.RateLimitWindow.Seconds(),
		cleanup:    3 * time.Minute,
	}

	// Cleanup stale entries periodically
	go rl.cleanupLoop()

	return rl
}

// getVisitor retrieves or creates a visitor entry
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{
			tokens:   rl.maxTokens,
			lastSeen: time.Now(),
		}
		rl.visitors[ip] = v
		return v
	}

	// Refill tokens based on elapsed time
	elapsed := time.Since(v.lastSeen).Seconds()
	v.tokens += elapsed * rl.refillRate
	if v.tokens > rl.maxTokens {
		v.tokens = rl.maxTokens
	}
	v.lastSeen = time.Now()

	return v
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	v := rl.getVisitor(ip)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v.tokens >= 1 {
		v.tokens--
		return true
	}

	return false
}

// Middleware returns a Fiber middleware function for rate limiting
func (rl *RateLimiter) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()

		if !rl.Allow(ip) {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error":   "Too many requests. Please wait a moment and try again.",
			})
		}

		return c.Next()
	}
}

// cleanupLoop removes stale visitor entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastSeen) > 5*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
