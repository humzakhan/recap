package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateBucket tracks token-bucket state for a single client IP.
type rateBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter returns a Gin middleware that enforces a per-IP token-bucket
// rate limit. Each IP gets requestsPerMinute tokens per minute. Tokens are
// refilled continuously based on elapsed time.
func RateLimiter(requestsPerMinute int) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := make(map[string]*rateBucket)
	rate := float64(requestsPerMinute) / 60.0 // tokens per second

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()

		b, exists := buckets[ip]
		if !exists {
			b = &rateBucket{
				tokens:     float64(requestsPerMinute),
				lastRefill: now,
			}
			buckets[ip] = b
		}

		// Refill tokens based on elapsed time.
		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens += elapsed * rate
		b.lastRefill = now

		// Cap tokens at the maximum.
		max := float64(requestsPerMinute)
		if b.tokens > max {
			b.tokens = max
		}

		if b.tokens < 1.0 {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, errorResponse{
				Error: "rate limit exceeded; try again later",
				Code:  "INVALID_REQUEST",
			})
			return
		}

		b.tokens -= 1.0
		mu.Unlock()

		c.Next()
	}
}
