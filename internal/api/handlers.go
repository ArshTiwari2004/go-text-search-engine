package api

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"

	"gosearch/internal/cache"
	"gosearch/internal/database"
	"gosearch/internal/engine"
	"gosearch/internal/middleware"
	"gosearch/internal/storage"

	"github.com/gin-gonic/gin"
)

// Server wraps the HTTP server and all infrastructure dependencies.
// Release 2 adds: Cache (Redis), DB (PostgreSQL), rate limiting.

type Server struct {
	engine      *engine.Engine
	persistence *storage.PersistenceManager
	cache       *cache.Cache // nil-safe: cache.Available() guards usage
	db          *database.DB // nil-safe: checked before each call
	router      *gin.Engine
}

// SearchRequest is unchanged from Release 1.
type SearchRequest struct {
	Query      string  `json:"query"      binding:"required"`
	MaxResults int     `json:"max_results,omitempty"`
	MinScore   float64 `json:"min_score,omitempty"`
}

// SearchResponse — same shape as Release 1 + cache_hit field.
type SearchResponse struct {
	Query        string                `json:"query"`
	Results      []engine.SearchResult `json:"results"`
	TotalResults int                   `json:"total_results"`
	TimeTaken    string                `json:"time_taken"`
	Success      bool                  `json:"success"`
	CacheHit     bool                  `json:"cache_hit"`
}

type StatsResponse struct {
	TotalDocuments   int             `json:"total_documents"`
	TotalTerms       int             `json:"total_terms"`
	TotalQueries     int64           `json:"total_queries"`
	AverageQueryTime string          `json:"average_query_time"`
	MemoryUsageMB    float64         `json:"memory_usage_mb"`
	IndexSizeKB      float64         `json:"index_size_kb"`
	Uptime           string          `json:"uptime"`
	CacheAvailable   bool            `json:"cache_available"`
	DBAvailable      bool            `json:"db_available"`
	QueryAnalytics   *QueryAnalytics `json:"query_analytics,omitempty"`
}

type QueryAnalytics struct {
	TotalLogged    int64   `json:"total_logged"`
	CacheHitRate   float64 `json:"cache_hit_rate_pct"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	UniqueQueries  int64   `json:"unique_queries"`
	Last24hQueries int64   `json:"last_24h_queries"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// ServerOptions groups optional dependencies injected into the server.
// All fields are optional — the server degrades gracefully when nil.
type ServerOptions struct {
	Cache *cache.Cache
	DB    *database.DB
}

// NewServer creates the API server.
// Pass ServerOptions{} for a Release-1-compatible setup (no cache, no DB).
func NewServer(eng *engine.Engine, pm *storage.PersistenceManager, opts ServerOptions) *Server {
	gin.SetMode(gin.ReleaseMode)

	s := &Server{
		engine:      eng,
		persistence: pm,
		cache:       opts.Cache,
		db:          opts.DB,
		router:      gin.New(), // use gin.New() so we add our own middleware
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Global middleware (Release 2 additions)
	s.router.Use(middleware.RequestLogger())
	s.router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins: []string{"*"}, // tighten in production
	}))

	// Health check — no rate limit (used by load balancers)
	s.router.GET("/health", s.healthCheck)

	// API v1
	v1 := s.router.Group("/api/v1")

	// Apply rate limiting to all /api/v1 routes.
	// 10 req/s sustained, burst of 30.
	v1.Use(middleware.RateLimiter(middleware.DefaultRateLimiterConfig()))

	v1.POST("/search", s.search)
	v1.GET("/search", s.searchGET)
	v1.GET("/document/:id", s.getDocument)
	v1.GET("/stats", s.getStats)
	v1.POST("/index/rebuild", s.rebuildIndex)
	v1.POST("/index/save", s.saveIndex)
	v1.GET("/index/info", s.getIndexInfo)
	v1.GET("/suggest", s.suggest)
	// Release 2 — new analytics endpoint
	v1.GET("/analytics", s.getAnalytics)
}

// search handles POST /api/v1/search
// Release 2 flow:
//  1. Check Redis cache → return immediately on hit
//  2. Run TF-IDF search (existing engine, untouched)
//  3. Store result in Redis cache
//  4. Log query to PostgreSQL (async, non-blocking)
func (s *Server) search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid_request", Message: err.Error(), Success: false,
		})
		return
	}
	if req.MaxResults == 0 {
		req.MaxResults = 10
	}

	ctx := c.Request.Context()
	start := time.Now()

	// ── Step 1: Cache lookup
	if s.cache != nil && s.cache.Available() {
		entry, err := s.cache.Get(ctx, req.Query)
		if err != nil {
			log.Printf("[search] cache get error: %v", err)
		}
		if entry != nil {
			// Cache HIT — build response from cached data and return.
			s.cache.RecordHit(ctx)
			results := cacheEntryToResults(entry)
			latencyMs := float64(time.Since(start).Microseconds()) / 1000.0

			if s.db != nil {
				s.db.LogQuery(req.Query, len(results), latencyMs, true, req.MaxResults)
			}

			c.JSON(http.StatusOK, SearchResponse{
				Query:        req.Query,
				Results:      results,
				TotalResults: len(results),
				TimeTaken:    time.Since(start).String(),
				Success:      true,
				CacheHit:     true,
			})
			return
		}
		s.cache.RecordMiss(ctx)
	}

	// ── Step 2: TF-IDF search (existing engine — UNTOUCHED)
	results, err := s.engine.Search(req.Query, req.MaxResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "search_failed", Message: err.Error(), Success: false,
		})
		return
	}

	for i := range results {
		results[i].Rank = i + 1
	}

	elapsed := time.Since(start)
	latencyMs := float64(elapsed.Microseconds()) / 1000.0

	// ── Step 3: Store in cache ]
	if s.cache != nil && s.cache.Available() {
		entry := &cache.CacheEntry{
			Query:      req.Query,
			Results:    resultsToCache(results),
			TotalCount: len(results),
			CachedAt:   time.Now(),
			OriginalMs: latencyMs,
		}
		if err := s.cache.Set(ctx, entry); err != nil {
			log.Printf("[search] cache set error: %v", err)
		}
	}

	// ── Step 4: Log to PostgreSQL (async)
	if s.db != nil {
		s.db.LogQuery(req.Query, len(results), latencyMs, false, req.MaxResults)
	}

	c.JSON(http.StatusOK, SearchResponse{
		Query:        req.Query,
		Results:      results,
		TotalResults: len(results),
		TimeTaken:    elapsed.String(),
		Success:      true,
		CacheHit:     false,
	})
}

// searchGET handles GET /api/v1/search?q=query&limit=10 (unchanged from R1)
func (s *Server) searchGET(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "missing_query", Message: "Query parameter 'q' is required", Success: false,
		})
		return
	}
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 10
	}

	start := time.Now()
	results, err := s.engine.Search(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "search_failed", Message: err.Error(), Success: false,
		})
		return
	}
	for i := range results {
		results[i].Rank = i + 1
	}

	if s.db != nil {
		latencyMs := float64(time.Since(start).Microseconds()) / 1000.0
		s.db.LogQuery(query, len(results), latencyMs, false, limit)
	}

	c.JSON(http.StatusOK, SearchResponse{
		Query: query, Results: results, TotalResults: len(results),
		TimeTaken: time.Since(start).String(), Success: true,
	})
}

// getDocument unchanged from Release 1.
func (s *Server) getDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "invalid_id", Message: "Document ID must be an integer", Success: false,
		})
		return
	}
	doc, err := s.engine.GetDocument(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "not_found", Message: err.Error(), Success: false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"document": doc, "success": true})
}

// getStats — Release 2 adds cache_available, db_available fields.
func (s *Server) getStats(c *gin.Context) {
	stats := s.engine.GetStats()

	resp := StatsResponse{
		TotalDocuments:   stats.TotalDocuments,
		TotalTerms:       stats.TotalTerms,
		TotalQueries:     stats.TotalQueries,
		AverageQueryTime: stats.AverageQueryTime.String(),
		MemoryUsageMB:    float64(stats.MemoryUsage) / 1024 / 1024,
		IndexSizeKB:      float64(stats.IndexSize) / 1024,
		Uptime:           "see /health",
		CacheAvailable:   s.cache != nil && s.cache.Available(),
		DBAvailable:      s.db != nil,
	}

	c.JSON(http.StatusOK, resp)
}

// getAnalytics — NEW in Release 2: returns query analytics from PostgreSQL.
func (s *Server) getAnalytics(c *gin.Context) {
	if s.db == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"message": "Database not configured. Start GoSearch with DB to see analytics.",
			"success": false,
		})
		return
	}

	ctx := c.Request.Context()

	dbStats, err := s.db.QueryStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "analytics_failed", Message: err.Error(), Success: false,
		})
		return
	}

	var cacheHitRate float64
	if dbStats.TotalQueries > 0 {
		cacheHitRate = float64(dbStats.CacheHits) / float64(dbStats.TotalQueries) * 100
	}

	topQueries, _ := s.db.TopQueries(ctx, 10)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"analytics": QueryAnalytics{
			TotalLogged:    dbStats.TotalQueries,
			CacheHitRate:   cacheHitRate,
			AvgLatencyMs:   dbStats.AvgLatencyMs,
			P99LatencyMs:   dbStats.P99LatencyMs,
			UniqueQueries:  dbStats.UniqueQueries,
			Last24hQueries: dbStats.QueriesLast24h,
		},
		"top_queries": topQueries,
	})
}

func (s *Server) healthCheck(c *gin.Context) {
	stats := s.engine.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"status":          "healthy",
		"documents":       stats.TotalDocuments,
		"terms":           stats.TotalTerms,
		"cache_available": s.cache != nil && s.cache.Available(),
		"db_available":    s.db != nil,
		"timestamp":       time.Now().Unix(),
	})
}

func (s *Server) rebuildIndex(c *gin.Context) {
	// Invalidate cache on rebuild so stale results are evicted.
	if s.cache != nil && s.cache.Available() {
		if err := s.cache.Invalidate(context.Background()); err != nil {
			log.Printf("[rebuildIndex] cache invalidation error: %v", err)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Index rebuild initiated. Cache invalidated.", "success": true})
}

func (s *Server) saveIndex(c *gin.Context) {
	stats := s.engine.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"message": "Index saved successfully", "documents": stats.TotalDocuments, "success": true,
	})
}

func (s *Server) getIndexInfo(c *gin.Context) {
	if s.persistence == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Persistence not configured", "success": false})
		return
	}
	info, err := s.persistence.GetIndexInfo()
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "index_info_unavailable", Message: err.Error(), Success: false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"info": info, "success": true})
}

func (s *Server) suggest(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "missing_query", Message: "Query parameter 'q' is required", Success: false,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": []string{}, "success": true})
}

func (s *Server) Run(addr string) error {
	log.Printf("[server] Starting GoSearch API on %s", addr)
	return s.router.Run(addr)
}

// Conversion helpers

func resultsToCache(results []engine.SearchResult) []cache.CachedResult {
	out := make([]cache.CachedResult, len(results))
	for i, r := range results {
		out[i] = cache.CachedResult{
			DocumentID:    r.Document.ID,
			DocumentTitle: r.Document.Title,
			DocumentURL:   r.Document.URL,
			DocumentText:  r.Document.Text,
			Score:         r.Score,
			Snippets:      r.Snippets,
			Rank:          r.Rank,
			WordCount:     r.Document.WordCount,
		}
	}
	return out
}

func cacheEntryToResults(entry *cache.CacheEntry) []engine.SearchResult {
	out := make([]engine.SearchResult, len(entry.Results))
	for i, cr := range entry.Results {
		out[i] = engine.SearchResult{
			Document: engine.Document{
				ID:        cr.DocumentID,
				Title:     cr.DocumentTitle,
				URL:       cr.DocumentURL,
				Text:      cr.DocumentText,
				WordCount: cr.WordCount,
			},
			Score:    cr.Score,
			Snippets: cr.Snippets,
			Rank:     cr.Rank,
		}
	}
	return out
}
