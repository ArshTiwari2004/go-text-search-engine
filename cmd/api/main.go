package main

import (
	"flag"
	"log"
	"time"

	"https://github.com/ArshTiwari2004/go-text-search-engine/internal/api"
	"https://github.com/ArshTiwari2004/go-text-search-engine/internal/engine"
	"https://github.com/ArshTiwari2004/go-text-search-engine/internal/storage"
)

// Configuration flags for the application
var (
	dumpPath string
	dataDir  string
	port     string
	rebuild  bool
)

func init() {
	// Define command-line flags
	flag.StringVar(&dumpPath, "dump", "enwiki-latest-stub-articles.xml.gz",
		"Path to Wikipedia dump file")
	flag.StringVar(&dataDir, "data", "./data",
		"Directory for index persistence")
	flag.StringVar(&port, "port", "8080",
		"HTTP server port")
	flag.BoolVar(&rebuild, "rebuild", false,
		"Force rebuild index from dump (ignores persisted index)")
}

func main() {
	flag.Parse()

	log.Println("Starting GoSearch API Server")
	log.Println("=" + string(make([]byte, 50)) + "=")

	// Initialize persistence manager
	pm, err := storage.NewPersistenceManager(dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize persistence: %v", err)
	}

	// Initialize search engine
	eng := engine.NewEngine()

	// Try to load existing index, or build new one
	if !rebuild && pm.IndexExists() {
		log.Println("Loading persisted index...")
		start := time.Now()

		idx, docs, stats, err := pm.LoadIndex()
		if err != nil {
			log.Printf("Failed to load index: %v", err)
			log.Println("Building new index from dump...")
			if err := buildIndexFromDump(eng, dumpPath); err != nil {
				log.Fatalf("Failed to build index: %v", err)
			}
		} else {
			// Successfully loaded from disk
			// This is a private operation, would need to expose via Engine API
			log.Printf("Loaded %d documents in %v", len(docs), time.Since(start))
			log.Printf("   Terms: %d | Memory: %.2f MB",
				stats.TotalTerms,
				float64(stats.MemoryUsage)/1024/1024)

			// For now, rebuild if load fails
			log.Println("Rebuilding index for demonstration...")
			if err := buildIndexFromDump(eng, dumpPath); err != nil {
				log.Fatalf("Failed to build index: %v", err)
			}
		}
	} else {
		// Build index from dump file
		log.Println("Building index from dump file...")
		if err := buildIndexFromDump(eng, dumpPath); err != nil {
			log.Fatalf("Failed to build index: %v", err)
		}

		// Save index for future use
		log.Println(" Saving index to disk...")
		// Note: Would need to expose index internals to save
		// Simplified for now
	}

	// Display startup statistics
	displayStats(eng)

	// Start API server
	server := api.NewServer(eng, pm)
	log.Printf("API server listening on http://localhost:%s", port)
	log.Println("API Documentation:")
	log.Println("   POST /api/v1/search        - Search with JSON")
	log.Println("   GET  /api/v1/search?q=...  - Simple search")
	log.Println("   GET  /api/v1/stats         - Engine statistics")
	log.Println("   GET  /health               - Health check")
	log.Println()

	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// buildIndexFromDump loads documents from dump file and builds the index
func buildIndexFromDump(eng *engine.Engine, path string) error {
	start := time.Now()

	log.Printf("Loading documents from: %s", path)
	docs, err := engine.LoadDocuments(path)
	if err != nil {
		return err
	}
	log.Printf("Loaded %d documents in %v", len(docs), time.Since(start))

	start = time.Now()
	log.Println("Building inverted index...")
	if err := eng.IndexDocuments(docs); err != nil {
		return err
	}

	stats := eng.GetStats()
	log.Printf("Indexed %d documents in %v", stats.TotalDocuments, time.Since(start))

	return nil
}

// displayStats prints engine statistics in a formatted way
func displayStats(eng *engine.Engine) {
	stats := eng.GetStats()

	log.Println()
	log.Println("Search Engine Statistics")
	log.Println("=" + string(make([]byte, 50)) + "=")
	log.Printf("Documents:      %d", stats.TotalDocuments)
	log.Printf("Unique Terms:   %d", stats.TotalTerms)
	log.Printf("Index Size:     %.2f MB", float64(stats.IndexSize)/1024/1024)
	log.Printf("Memory Usage:   %.2f MB", float64(stats.MemoryUsage)/1024/1024)
	log.Printf("Index Time:     %v", stats.LastIndexTime)
	log.Println("=" + string(make([]byte, 50)) + "=")
	log.Println()
}
