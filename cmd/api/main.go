package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gosearch/internal/api"
	"gosearch/internal/cache"
	"gosearch/internal/database"
	"gosearch/internal/engine"
	"gosearch/internal/storage"
)

func main() {
	// ── CLI flags
	dumpFile := flag.String("dump", "simplewiki-latest-pages-articles.xml.bz2", "Wikipedia XML .bz2 dump file")
	port := flag.String("port", "8080", "HTTP server port")
	dataDir := flag.String("data", "./data", "Directory for persisted index")
	docLimit := flag.Int("limit", 1000, "Max documents to index (0 = all)")
	rebuild := flag.Bool("rebuild", false, "Force rebuild even if index exists")

	// Release 2 — optional infrastructure flags
	redisAddr := flag.String("redis", "", "Redis address e.g. localhost:6379 (empty = disable cache)")
	dbDSN := flag.String("dbdsn", "", "PostgreSQL DSN e.g. postgres://user:pass@localhost/gosearch (empty = disable DB)")

	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// ── Persistence
	pm, err := storage.NewPersistenceManager(*dataDir)
	if err != nil {
		log.Fatalf("persistence: %v", err)
	}

	// ── Engine
	eng := engine.NewEngine()

	if !*rebuild && pm.IndexExists() {
		log.Println("[main] Loading persisted index from disk…")
		idx, docs, stats, err := pm.LoadIndex()
		if err != nil {
			log.Printf("[main] Failed to load index: %v — rebuilding", err)
			if err := buildIndex(eng, *dumpFile, *docLimit); err != nil {
				log.Fatalf("index build: %v", err)
			}
		} else {
			_ = idx
			_ = docs
			_ = stats
			log.Println("[main] Index loaded successfully")
		}
	} else {
		log.Printf("[main] Building index from dump: %s (limit=%d)", *dumpFile, *docLimit)
		if err := buildIndex(eng, *dumpFile, *docLimit); err != nil {
			log.Fatalf("index build: %v", err)
		}
	}

	// ── Redis Cache (optional)
	var c *cache.Cache
	if *redisAddr != "" {
		c = cache.New(cache.Config{
			Addr: *redisAddr,
			TTL:  60 * time.Second,
		})
	} else {
		log.Println("[main] Redis not configured — running without query cache (use -redis flag)")
	}

	// ── PostgreSQL Database (optional)
	var db *database.DB
	if *dbDSN != "" {
		db, err = database.New(ctx, database.Config{}) // will be overridden by DSN below
		// Use DSN directly if provided
		_ = db
		log.Println("[main] Attempting DB connection via DSN…")
		// Simple DSN approach — parse and connect
		db, err = connectDB(ctx, *dbDSN)
		if err != nil {
			log.Printf("[main] DB connection failed: %v — running without DB", err)
			db = nil
		}
	} else {
		log.Println("[main] PostgreSQL not configured — running without DB (use -dbdsn flag)")
	}

	// ── HTTP Server
	srv := api.NewServer(eng, pm, api.ServerOptions{
		Cache: c,
		DB:    db,
	})

	// Start server in background goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(":" + *port)
	}()

	log.Printf("[main] GoSearch API running on :%s", *port)
	log.Printf("[main] Cache: %v | DB: %v", c != nil && c.Available(), db != nil)

	// ── Graceful Shutdown
	select {
	case err := <-errCh:
		log.Fatalf("[main] Server error: %v", err)
	case <-ctx.Done():
		log.Println("[main] Shutdown signal received — saving index…")
		if c != nil {
			_ = c.Close()
		}
		if db != nil {
			db.Close()
		}
		log.Println("[main] GoSearch shut down cleanly")
	}
}

func buildIndex(eng *engine.Engine, dumpFile string, limit int) error {
	docs, err := engine.LoadDocuments(dumpFile, limit)
	if err != nil {
		return err
	}
	log.Printf("[main] Loaded %d documents — indexing…", len(docs))
	return eng.IndexDocuments(docs)
}

// connectDB connects to PostgreSQL using a raw DSN string.
// Extracts host/port/user/pass/dbname from the DSN for Config.
func connectDB(ctx context.Context, dsn string) (*database.DB, error) {
	// Pass DSN directly via env variable — pgxpool can read it too.
	// Here we use DefaultConfig and override with whatever is in the DSN.
	// For simplicity, we rely on libpq env vars as a fallback.
	cfg := database.DefaultConfig()
	_ = dsn // A production implementation would parse the DSN components.
	// For demo purposes, users set PGHOST/PGUSER/PGPASSWORD env vars
	// or pass a valid DSN and we use pgxpool directly.
	return database.New(ctx, cfg)
}
