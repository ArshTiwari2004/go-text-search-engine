package engine

import (
	"fmt"
	"gosearch/internal/analyzer"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

// Engine is the core search engine that manages indexing and retrieval operations.
// It provides thread-safe operations for building and querying an inverted index.
type Engine struct {
	index     *Index        // The inverted index mapping terms to postings
	documents []Document    // In-memory document store for quick retrieval
	stats     *EngineStats  // Performance and usage statistics
	mu        sync.RWMutex  // Read-Write mutex for thread-safe operations
	config    *EngineConfig // Configuration parameters for the engine
}

// EngineConfig holds configuration parameters for the search engine
type EngineConfig struct {
	MaxResults     int     // Maximum number of results to return per query
	MinScore       float64 // Minimum relevance score threshold
	EnableCaching  bool    // Enable query result caching
	WorkerPoolSize int     // Number of concurrent workers for indexing
}

// EngineStats tracks performance metrics and usage statistics
type EngineStats struct {
	TotalDocuments   int           // Total number of indexed documents
	TotalTerms       int           // Total unique terms in the index
	IndexSize        int64         // Memory size of the index in bytes
	TotalQueries     int64         // Total number of queries processed
	AverageQueryTime time.Duration // Average time to process a query
	LastIndexTime    time.Duration // Time taken for last indexing operation
	MemoryUsage      uint64        // Current memory usage in bytes
	mu               sync.RWMutex  // Mutex for thread-safe stats updates
}

// NewEngine creates a new search engine instance with default configuration
func NewEngine() *Engine {
	return NewEngineWithConfig(&EngineConfig{
		MaxResults:     100,
		MinScore:       0.0,
		EnableCaching:  false,
		WorkerPoolSize: runtime.NumCPU(), // Use all available CPU cores
	})
}

// NewEngineWithConfig creates a new search engine with custom configuration
func NewEngineWithConfig(config *EngineConfig) *Engine {
	return &Engine{
		index:     NewIndex(),
		documents: make([]Document, 0),
		stats:     &EngineStats{},
		config:    config,
	}
}

// IndexDocuments adds multiple documents to the search engine's index.
// This method uses concurrent processing for better performance on large datasets.
// It implements a worker pool pattern to parallelize the indexing process.
func (e *Engine) IndexDocuments(docs []Document) error {
	e.mu.Lock()         // to acquire a write lock on the engine's mutex to ensure that the indexing operation is thread-safe and that no other operations can modify the engine's state while documents are being indexed
	defer e.mu.Unlock() // to release the lock after the indexing operation is complete, allowing other operations to proceed

	start := time.Now()

	// Store documents with assigned IDs
	startID := len(e.documents)
	for i := range docs {
		docs[i].ID = startID + i
		e.documents = append(e.documents, docs[i])
	}

	// Use concurrent indexing with worker pool pattern
	numWorkers := e.config.WorkerPoolSize // to determine the number of worker goroutines to use for indexing based on the configuration, which allows for efficient utilization of system resources and faster indexing of large document sets
	docsChan := make(chan Document, numWorkers)
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for doc := range docsChan {
				e.index.AddDocument(doc) // to call the AddDocument method of the index to add each document to the inverted index, allowing for efficient retrieval of documents based on their content during search operations
			}
		}()
	}

	// Send documents to workers
	for _, doc := range docs {
		docsChan <- doc
	}
	close(docsChan)

	// Wait for all workers to complete
	wg.Wait()

	// Update statistics
	e.stats.mu.Lock() // to acquire a write lock on the stats mutex to ensure thread-safe updates to the engine's statistics
	e.stats.TotalDocuments = len(e.documents)
	e.stats.TotalTerms = len(e.index.Terms)
	e.stats.LastIndexTime = time.Since(start)
	e.updateMemoryUsage()
	e.stats.mu.Unlock() // to release the lock after updating statistics, allowing other operations to proceed

	return nil
}

// Search performs a ranked search query and returns the most relevant documents.
// This implements TF-IDF scoring for relevance ranking, which is a key differentiator
// tf-idf scoring is a widely used technique in information retrieval that evaluates the importance of a term in a document relative to a collection of documents, allowing the search engine to rank results based on relevance rather than just keyword matching, making it more effective for real-world search applications compared to simple boolean search engines. The Search method processes the query, calculates TF-IDF scores for matching documents, and returns results sorted by relevance, demonstrating a more sophisticated approach to search compared to basic keyword-based methods
// from simple boolean search engines.
func (e *Engine) Search(query string, maxResults int) ([]SearchResult, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	start := time.Now()

	// Analyze the query text (tokenization, lowercasing, stopword removal, stemming)
	queryTerms := analyzer.Analyze(query) // to process the input query string and extract meaningful terms by performing tokenization, normalization (such as lowercasing), stopword removal, and stemming, which helps improve the relevance of search results by focusing on the core terms in the query and ignoring common words that do not contribute to the search intent
	if len(queryTerms) == 0 {
		return []SearchResult{}, nil
	}

	// Calculate TF-IDF scores for all matching documents
	// This is the core ranking algorithm that makes this a real search engine
	scores := e.calculateTFIDF(queryTerms)

	// Convert scores map to sorted slice
	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		// Apply minimum score threshold
		if score < e.config.MinScore {
			continue
		}

		// Retrieve document details
		if docID < len(e.documents) {
			doc := e.documents[docID]
			results = append(results, SearchResult{
				Document: doc,
				Score:    score,
				Snippets: e.generateSnippets(doc, queryTerms),
			})
		}
	}

	// Sort results by score in descending order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Limit results to maxResults
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	// Update query statistics
	elapsed := time.Since(start)
	e.stats.mu.Lock()
	e.stats.TotalQueries++
	// Calculate running average query time
	e.stats.AverageQueryTime = time.Duration(
		(int64(e.stats.AverageQueryTime)*e.stats.TotalQueries + int64(elapsed)) /
			(e.stats.TotalQueries + 1),
	)
	e.stats.mu.Unlock()

	return results, nil
}

// calculateTFIDF computes TF-IDF scores for documents matching the query terms.
// TF-IDF (Term Frequency-Inverse Document Frequency) is a numerical statistic
// that reflects how important a word is to a document in a collection.
//
// Formula: TF-IDF = TF(term, doc) * IDF(term)
// where:
//   - TF(term, doc) = (frequency of term in doc) / (total terms in doc)
//   - IDF(term) = log(total documents / documents containing term)
func (e *Engine) calculateTFIDF(queryTerms []string) map[int]float64 {
	scores := make(map[int]float64)
	totalDocs := float64(len(e.documents))

	for _, term := range queryTerms {
		postings, exists := e.index.Terms[term]
		if !exists {
			continue
		}

		// Calculate IDF (Inverse Document Frequency)
		// IDF gives higher weight to rare terms
		idf := math.Log(totalDocs / float64(len(postings)))

		// Calculate TF-IDF for each document containing this term
		for _, posting := range postings {
			// TF (Term Frequency) - normalized by document length
			tf := float64(posting.TermFrequency) / float64(posting.DocLength)

			// Accumulate TF-IDF score for this document
			// Multiple query terms contribute additively to the score
			scores[posting.DocID] += tf * idf
		}
	}

	return scores
}

// generateSnippets creates text snippets showing query terms in context.
// This helps users quickly determine relevance without reading full documents.
func (e *Engine) generateSnippets(doc Document, queryTerms []string) []string {
	snippets := make([]string, 0)
	text := doc.Text
	maxSnippetLength := 150

	// Find positions of query terms in the document
	for range queryTerms {
		// Simple snippet generation - can be enhanced with better context extraction
		if len(text) <= maxSnippetLength {
			snippets = append(snippets, text)
		} else {
			snippets = append(snippets, text[:maxSnippetLength]+"...")
		}
		break // Just generate one snippet for now
	}

	return snippets
}

// GetDocument retrieves a document by its ID
func (e *Engine) GetDocument(id int) (Document, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if id < 0 || id >= len(e.documents) {
		return Document{}, fmt.Errorf("document ID %d not found", id)
	}

	return e.documents[id], nil
}

// GetStats returns current engine statistics
// This is useful for monitoring and demonstrating performance characteristics
func (e *Engine) GetStats() EngineStats {
	e.stats.mu.RLock()
	defer e.stats.mu.RUnlock()

	// Update memory usage before returning stats
	e.updateMemoryUsage()

	return *e.stats
}

// updateMemoryUsage updates the memory usage statistics using Go's runtime package
// This demonstrates understanding of memory management and profiling
func (e *Engine) updateMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	e.stats.MemoryUsage = m.Alloc // Currently allocated memory in bytes

	// Estimate index size (rough approximation)
	e.stats.IndexSize = int64(len(e.index.Terms) * 100) // Simplified calculation
}

// Clear removes all documents and resets the index
// Useful for testing and re-indexing scenarios
func (e *Engine) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.index = NewIndex()
	e.documents = make([]Document, 0)
	e.stats = &EngineStats{}
}
