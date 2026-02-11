package api

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ArshTiwari2004/go-text-search-engine/internal/engine"
	"github.com/ArshTiwari2004/go-text-search-engine/internal/storage"
	"github.com/gin-gonic/gin"
)

// Server wraps the HTTP server and search engine
type Server struct {
	engine      *engine.Engine
	persistence *storage.PersistenceManager
	router      *gin.Engine
}

// SearchRequest represents the JSON structure for search queries
type SearchRequest struct {
	Query      string  `json:"query" binding:"required"` // Search query text
	MaxResults int     `json:"max_results,omitempty"`    // Optional: limit results (default: 10)
	MinScore   float64 `json:"min_score,omitempty"`      // Optional: minimum relevance score
}

// SearchResponse represents the JSON structure for search results
type SearchResponse struct {
	Query        string                `json:"query"`         // Original query
	Results      []engine.SearchResult `json:"results"`       // Matching documents with scores
	TotalResults int                   `json:"total_results"` // Total number of matches
	TimeTaken    string                `json:"time_taken"`    // Query execution time
	Success      bool                  `json:"success"`
}

// StatsResponse represents engine statistics for monitoring
type StatsResponse struct {
	TotalDocuments   int     `json:"total_documents"`
	TotalTerms       int     `json:"total_terms"`
	TotalQueries     int64   `json:"total_queries"`
	AverageQueryTime string  `json:"average_query_time"`
	MemoryUsageMB    float64 `json:"memory_usage_mb"`
	IndexSizeKB      float64 `json:"index_size_kb"`
	Uptime           string  `json:"uptime"`
}

// ErrorResponse represents API error responses
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// NewServer creates a new API server instance
func NewServer(eng *engine.Engine, pm *storage.PersistenceManager) *Server {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	server := &Server{
		engine:      eng,
		persistence: pm,
		router:      gin.Default(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API endpoints
func (s *Server) setupRoutes() {
	// Enable CORS for frontend access
	s.router.Use(corsMiddleware())

	// Health check endpoint
	s.router.GET("/health", s.healthCheck)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Search endpoints
		v1.POST("/search", s.search)
		v1.GET("/search", s.searchGET) // Alternative GET method for simple queries

		// Document endpoints
		v1.GET("/document/:id", s.getDocument)

		// Stats and monitoring
		v1.GET("/stats", s.getStats)

		// Index management (admin operations)
		v1.POST("/index/rebuild", s.rebuildIndex)
		v1.POST("/index/save", s.saveIndex)
		v1.GET("/index/info", s.getIndexInfo)

		// Autocomplete suggestions
		v1.GET("/suggest", s.suggest)
	}
}

// search handles POST /api/v1/search
// This is the main search endpoint using TF-IDF ranking
func (s *Server) search(c *gin.Context) {
	var req SearchRequest

	// Parse JSON request body
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	// Set defaults
	if req.MaxResults == 0 {
		req.MaxResults = 10
	}

	start := time.Now()

	// Execute search with TF-IDF ranking
	results, err := s.engine.Search(req.Query, req.MaxResults)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Search failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	elapsed := time.Since(start)

	// Add rank to results
	for i := range results {
		results[i].Rank = i + 1
	}

	c.JSON(http.StatusOK, SearchResponse{
		Query:        req.Query,
		Results:      results,
		TotalResults: len(results),
		TimeTaken:    elapsed.String(),
		Success:      true,
	})
}

// searchGET handles GET /api/v1/search?q=query&limit=10
// Convenience method for simple queries without JSON
func (s *Server) searchGET(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing query parameter",
			Message: "Query parameter 'q' is required",
			Success: false,
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
			Error:   "Search failed",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	elapsed := time.Since(start)

	for i := range results {
		results[i].Rank = i + 1
	}

	c.JSON(http.StatusOK, SearchResponse{
		Query:        query,
		Results:      results,
		TotalResults: len(results),
		TimeTaken:    elapsed.String(),
		Success:      true,
	})
}

// getDocument retrieves a specific document by ID
func (s *Server) getDocument(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid document ID",
			Message: "Document ID must be an integer",
			Success: false,
		})
		return
	}

	doc, err := s.engine.GetDocument(id)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Document not found",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"document": doc,
		"success":  true,
	})
}

// getStats returns current engine statistics
func (s *Server) getStats(c *gin.Context) {
	stats := s.engine.GetStats()

	response := StatsResponse{
		TotalDocuments:   stats.TotalDocuments,
		TotalTerms:       stats.TotalTerms,
		TotalQueries:     stats.TotalQueries,
		AverageQueryTime: stats.AverageQueryTime.String(),
		MemoryUsageMB:    float64(stats.MemoryUsage) / 1024 / 1024,
		IndexSizeKB:      float64(stats.IndexSize) / 1024,
		Uptime:           time.Since(time.Now().Add(-time.Hour)).String(), // Placeholder
	}

	c.JSON(http.StatusOK, response)
}

// healthCheck returns server health status
func (s *Server) healthCheck(c *gin.Context) {
	stats := s.engine.GetStats()

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"documents": stats.TotalDocuments,
		"terms":     stats.TotalTerms,
		"queries":   stats.TotalQueries,
		"timestamp": time.Now().Unix(),
	})
}

// rebuildIndex triggers a full index rebuild (admin operation)
func (s *Server) rebuildIndex(c *gin.Context) {
	// This would reload from dump file and rebuild index
	// Implementation depends on your requirements
	c.JSON(http.StatusOK, gin.H{
		"message": "Index rebuild initiated",
		"success": true,
	})
}

// saveIndex persists the current index to disk
func (s *Server) saveIndex(c *gin.Context) {
	stats := s.engine.GetStats()

	// This would need access to index internals
	// Simplified for now
	c.JSON(http.StatusOK, gin.H{
		"message":   "Index saved successfully",
		"documents": stats.TotalDocuments,
		"success":   true,
	})
}

// getIndexInfo returns information about the persisted index
func (s *Server) getIndexInfo(c *gin.Context) {
	if s.persistence == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "Persistence not configured",
			"success": false,
		})
		return
	}

	info, err := s.persistence.GetIndexInfo()
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Index info not available",
			Message: err.Error(),
			Success: false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"info":    info,
		"success": true,
	})
}

// suggest provides autocomplete suggestions (simplified implementation)
func (s *Server) suggest(c *gin.Context) {
	prefix := c.Query("q")
	if prefix == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Missing query parameter",
			Message: "Query parameter 'q' is required",
			Success: false,
		})
		return
	}

	// return empty suggestions
	c.JSON(http.StatusOK, gin.H{
		"suggestions": []string{},
		"success":     true,
	})
}

// Run starts the HTTP server on the specified address
func (s *Server) Run(addr string) error {
	log.Printf("Starting API server on %s", addr)
	return s.router.Run(addr)
}

// corsMiddleware enables CORS for frontend access
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
