package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache wraps a Redis client and provides typed helpers for GoSearch.
//
// Why Redis?
//   - In-memory key-value store, sub-millisecond reads.
//   - Search queries follow Zipf's law: the top 5% of queries account for
//     ~80% of traffic. Caching them eliminates TF-IDF computation entirely.
//   - TTL-based eviction keeps results fresh without manual invalidation.
//
// Design: The cache is OPTIONAL. If Redis is unavailable the engine falls
// back to normal TF-IDF search and callers check cache.Available() first.

type Cache struct {
	client    *redis.Client
	available bool
	ttl       time.Duration
}

// CachedResult mirrors engine.SearchResult but is self-contained so the
// cache package does not import the engine package (avoids circular deps).
type CachedResult struct {
	DocumentID    int      `json:"doc_id"`
	DocumentTitle string   `json:"title"`
	DocumentURL   string   `json:"url"`
	DocumentText  string   `json:"text"`
	Score         float64  `json:"score"`
	Snippets      []string `json:"snippets"`
	Rank          int      `json:"rank"`
	WordCount     int      `json:"word_count"`
}

// CacheEntry is what we actually store in Redis.
type CacheEntry struct {
	Query      string         `json:"query"`
	Results    []CachedResult `json:"results"`
	TotalCount int            `json:"total_count"`
	CachedAt   time.Time      `json:"cached_at"`
	OriginalMs float64        `json:"original_ms"` // how long the real search took
}

// Config holds Redis connection settings.
type Config struct {
	Addr     string        // e.g. "localhost:6379"
	Password string        // "" for no auth
	DB       int           // Redis DB index (0–15)
	TTL      time.Duration // cache entry lifetime (default 60s)
}

// DefaultConfig returns a config for a local Redis with 60-second TTL.
func DefaultConfig() Config {
	return Config{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
		TTL:      60 * time.Second,
	}
}

// New creates a Cache. It pings Redis; if unreachable it returns a
// degraded Cache where Available() == false so callers can skip caching.

func New(cfg Config) *Cache {
	if cfg.TTL == 0 {
		cfg.TTL = 60 * time.Second
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c := &Cache{client: rdb, ttl: cfg.TTL}

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[cache] Redis not available at %s: %v — running without cache", cfg.Addr, err)
		c.available = false
	} else {
		log.Printf("[cache] Redis connected at %s (TTL=%s)", cfg.Addr, cfg.TTL)
		c.available = true
	}

	return c
}

// Available returns true when Redis is reachable.

func (c *Cache) Available() bool {
	return c.available
}

// key builds a deterministic Redis key from a query string.
//
// Normalisation: lowercase + trim + collapse whitespace so
// "  Machine Learning " and "machine learning" hit the same key.

func (c *Cache) key(query string) string {
	normalised := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(query)), " "))
	sum := sha256.Sum256([]byte(normalised))
	return fmt.Sprintf("gosearch:query:%x", sum[:8]) // 16-char hex suffix
}

// Get retrieves a cached result set. Returns (nil, nil) on cache miss.

func (c *Cache) Get(ctx context.Context, query string) (*CacheEntry, error) {
	if !c.available {
		return nil, nil
	}

	val, err := c.client.Get(ctx, c.key(query)).Bytes()
	if err == redis.Nil {
		return nil, nil // cache miss — not an error
	}
	if err != nil {
		return nil, fmt.Errorf("cache get: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(val, &entry); err != nil {
		return nil, fmt.Errorf("cache unmarshal: %w", err)
	}

	return &entry, nil
}

// Set stores a result set in Redis with the configured TTL.

func (c *Cache) Set(ctx context.Context, entry *CacheEntry) error {
	if !c.available {
		return nil
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("cache marshal: %w", err)
	}

	return c.client.Set(ctx, c.key(entry.Query), data, c.ttl).Err()
}

// Invalidate removes all GoSearch query keys from Redis.
// Call this after a full index rebuild so stale results are evicted.

func (c *Cache) Invalidate(ctx context.Context) error {
	if !c.available {
		return nil
	}

	// SCAN is non-blocking unlike KEYS — safe on production Redis.

	var cursor uint64
	var deleted int64

	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, "gosearch:query:*", 100).Result()
		if err != nil {
			return fmt.Errorf("cache scan: %w", err)
		}
		if len(keys) > 0 {
			n, err := c.client.Del(ctx, keys...).Result()
			if err != nil {
				return fmt.Errorf("cache del: %w", err)
			}
			deleted += n
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	log.Printf("[cache] Invalidated %d cached query keys", deleted)
	return nil
}

// Stats returns hit/miss counters stored in Redis itself.
// We use simple INCR keys so stats survive server restarts.

func (c *Cache) Stats(ctx context.Context) (hits, misses int64, err error) {
	if !c.available {
		return 0, 0, nil
	}

	h, err := c.client.Get(ctx, "gosearch:stats:hits").Int64()
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}
	m, err := c.client.Get(ctx, "gosearch:stats:misses").Int64()
	if err != nil && err != redis.Nil {
		return 0, 0, err
	}

	return h, m, nil
}

// RecordHit increments the hit counter (fire-and-forget).
func (c *Cache) RecordHit(ctx context.Context) {
	if c.available {
		c.client.Incr(ctx, "gosearch:stats:hits")
	}
}

// RecordMiss increments the miss counter (fire-and-forget).
func (c *Cache) RecordMiss(ctx context.Context) {
	if c.available {
		c.client.Incr(ctx, "gosearch:stats:misses")
	}
}

// Close cleanly disconnects the Redis client.
func (c *Cache) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
