package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Token-Bucket Rate Limiter
//
// Algorithm: Token Bucket
//   - Each IP address gets its own bucket.
//   - The bucket holds at most `capacity` tokens.
//   - Tokens refill at `refillRate` tokens per `refillEvery` interval.
//   - Each request consumes 1 token.
//   - If the bucket is empty, the request is rejected with 429.
//
// Why token-bucket over leaky-bucket or fixed-window?
//   - Allows short bursts (up to `capacity`) — better UX for humans.
//   - No thundering-herd at window boundary (unlike fixed-window).
//   - O(1) per request; O(active_IPs) memory.
//

// bucket holds the token state for a single IP.
type bucket struct {
	tokens     float64
	capacity   float64
	refillRate float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

// take tries to consume 1 token. Returns true if allowed, false if rate-limited.
func (b *bucket) take() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Refill tokens based on elapsed time since last request.
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens = min64(b.capacity, b.tokens+elapsed*b.refillRate)
	b.lastRefill = now

	if b.tokens < 1 {
		return false // rate-limited
	}
	b.tokens--
	return true
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimiterConfig controls rate limiter behaviour.
type RateLimiterConfig struct {
	RequestsPerSecond float64 // sustained request rate allowed per IP
	BurstSize         int     // max tokens (allows short bursts)
}

// DefaultRateLimiterConfig is permissive enough for development but
// protects against scripted abuse.
func DefaultRateLimiterConfig() RateLimiterConfig {
	return RateLimiterConfig{
		RequestsPerSecond: 10, // 10 req/s sustained
		BurstSize:         30, // allows burst of 30 before throttling
	}
}

// limiterStore manages per-IP buckets.
type limiterStore struct {
	buckets sync.Map // map[string]*bucket
	cfg     RateLimiterConfig
}

func newLimiterStore(cfg RateLimiterConfig) *limiterStore {
	ls := &limiterStore{cfg: cfg}
	// Prune stale IP entries every 5 minutes to prevent unbounded memory growth.
	go ls.pruneLoop()
	return ls
}

func (ls *limiterStore) getBucket(ip string) *bucket {
	if b, ok := ls.buckets.Load(ip); ok {
		return b.(*bucket)
	}
	b := &bucket{
		tokens:     float64(ls.cfg.BurstSize),
		capacity:   float64(ls.cfg.BurstSize),
		refillRate: ls.cfg.RequestsPerSecond,
		lastRefill: time.Now(),
	}
	// Store only if no concurrent goroutine beat us to it.
	actual, _ := ls.buckets.LoadOrStore(ip, b)
	return actual.(*bucket)
}

func (ls *limiterStore) pruneLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		threshold := time.Now().Add(-10 * time.Minute)
		ls.buckets.Range(func(key, value interface{}) bool {
			b := value.(*bucket)
			b.mu.Lock()
			idle := b.lastRefill.Before(threshold)
			b.mu.Unlock()
			if idle {
				ls.buckets.Delete(key)
			}
			return true
		})
	}
}

// RateLimiter returns a Gin middleware that enforces per-IP rate limiting.
//
// Attaches these response headers (like GitHub's API):
//   X-RateLimit-Limit     — max requests per window
//   X-RateLimit-Remaining — tokens left in bucket (approx)
//   Retry-After           — seconds until next token available (on 429)

func RateLimiter(cfg RateLimiterConfig) gin.HandlerFunc {
	store := newLimiterStore(cfg)

	return func(c *gin.Context) {
		// Prefer X-Forwarded-For (set by Nginx/ALB) over direct RemoteAddr.
		ip := c.GetHeader("X-Forwarded-For")
		if ip == "" {
			ip = c.ClientIP()
		}

		b := store.getBucket(ip)

		// Set informational headers on every response.
		c.Header("X-RateLimit-Limit", formatFloat(cfg.RequestsPerSecond))
		c.Header("X-RateLimit-Burst", formatInt(cfg.BurstSize))

		if !b.take() {
			retryAfter := 1.0 / cfg.RequestsPerSecond
			c.Header("Retry-After", formatFloat(retryAfter))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests. Please slow down.",
				"success": false,
			})
			return
		}

		c.Next()
	}
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.0f", f)
}

func formatInt(i int) string {
	return fmt.Sprintf("%d", i)
}

// Request Logger
//
// Structured request logger that replaces Gin's default coloured output.
// Logs: method, path, status, latency, IP.
// Format is designed for easy parsing by log aggregators (Loki, Datadog).

// RequestLogger returns a Gin middleware that logs every request.

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		ip := c.ClientIP()

		if query != "" {
			path = path + "?" + query
		}

		// Simple structured log line — swap for zerolog/zap in production.
		logFn := logInfo
		if status >= 500 {
			logFn = logError
		} else if status >= 400 {
			logFn = logWarn
		}

		logFn("[api] %s %s %d %s %s", c.Request.Method, path, status, latency, ip)
	}
}

func logInfo(format string, args ...interface{})  { fmt.Printf("[INFO]  "+format+"\n", args...) }
func logWarn(format string, args ...interface{})  { fmt.Printf("[WARN]  "+format+"\n", args...) }
func logError(format string, args ...interface{}) { fmt.Printf("[ERROR] "+format+"\n", args...) }

//  CORS Middleware
// Replaces the inline corsMiddleware in handlers.go.
// AllowedOrigins should be restricted to your domain in production.

// CORSConfig holds CORS settings.

type CORSConfig struct {
	AllowedOrigins []string // e.g. ["https://gosearch.example.com"]
}

// CORS returns a Gin middleware for Cross-Origin Resource Sharing.
// In development, pass AllowedOrigins: []string{"*"}.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	origins := cfg.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if the request origin is in the allow-list.
		allowed := false
		for _, o := range origins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400") // 24h preflight cache

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
