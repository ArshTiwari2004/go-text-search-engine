import React, { useState } from 'react';
import './doc.css';

const SECTIONS = [
  { id: 'overview',      label: 'Overview' },
  { id: 'quickstart',   label: 'Quick Start' },
  { id: 'architecture', label: 'Architecture' },
  { id: 'api',          label: 'API Reference' },
  { id: 'docker',       label: 'Docker' },
  { id: 'features',     label: 'Features' },
  { id: 'perf',         label: 'Performance' },
];

function CodeBlock({ code, lang = 'bash' }) {
  const [copied, setCopied] = useState(false);
  const copy = () => {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };
  return (
    <div className="doc-code-wrap">
      <div className="doc-code-header">
        <span className="doc-code-lang">{lang}</span>
        <button className="doc-copy-btn" onClick={copy}>{copied ? '✓ copied' : 'copy'}</button>
      </div>
      <pre className="doc-pre"><code>{code}</code></pre>
    </div>
  );
}

export default function Documentation() {
  const [active, setActive] = useState('overview');

  const scrollTo = (id) => {
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    setActive(id);
  };

  return (
    <div className="doc-root">
      {/* ── Sidebar ── */}
      <aside className="doc-sidebar">
        <a href="/" className="doc-back">← Home</a>
        <img src="/gosearchlogo1.png" alt="GoSearch" className="doc-sidebar-logo" />
        <nav className="doc-nav">
          {SECTIONS.map(s => (
            <button key={s.id} className={`doc-nav-btn ${active === s.id ? 'doc-nav-btn--active' : ''}`} onClick={() => scrollTo(s.id)}>
              {s.label}
            </button>
          ))}
        </nav>
        <a className="doc-gh-link" href="https://github.com/ArshTiwari2004/go-text-search-engine" target="_blank" rel="noopener noreferrer">
          GitHub ↗
        </a>
      </aside>

      {/* ── Content ── */}
      <main className="doc-content">

        {/* ── Overview ── */}
        <section id="overview" className="doc-section">
          <span className="doc-label">Overview</span>
          <h1 className="doc-h1">GoSearch</h1>
          <p className="doc-lead">
            A lightweight, concurrent full-text search engine built from scratch in Go. It indexes documents using an inverted index and ranks results with TF-IDF, returning relevant results in milliseconds, with optional Redis caching and PostgreSQL-backed analytics for a more production-ready setup.
          </p>
          <div className="doc-badge-row">
            <span className="doc-badge doc-badge--go">Go 1.21+</span>
            <span className="doc-badge doc-badge--react">React 18</span>
            <span className="doc-badge doc-badge--gin">Gin HTTP</span>
            <span className="doc-badge doc-badge--tfidf">TF-IDF</span>
          </div>
        </section>

        {/* ── Quick Start ── */}
        <section id="quickstart" className="doc-section">
          <span className="doc-label">Setup</span>
          <h2 className="doc-h2">Quick Start</h2>

          <h3 className="doc-h3">Prerequisites</h3>
          <ul className="doc-list">
            <li>Go 1.21+</li>
            <li>Git</li>
            <li>Docker Desktop (recommended for Redis + PostgreSQL)</li>
            <li>4 GB RAM (for large Wikipedia dump indexing)</li>
          </ul>

          <h3 className="doc-h3">Backend</h3>
          <CodeBlock lang="bash" code={`git clone https://github.com/ArshTiwari2004/go-text-search-engine.git
cd gosearch
go mod download

# Optional: download Simple Wikipedia dump (~200 MB)
wget https://dumps.wikimedia.org/simplewiki/latest/simplewiki-latest-pages-articles.xml.bz2
mv simplewiki-latest-pages-articles.xml.bz2 cmd/api/

# Start backend only (search engine + API, no Redis / DB)
cd cmd/api
go run main.go

# Backend runs at:
# http://localhost:8080`} />

          <h3 className="doc-h3">Backend with Redis + PostgreSQL</h3>
          <CodeBlock lang="bash" code={`# From project root, start infrastructure first
docker compose up -d redis postgres

# Then start the backend locally
cd cmd/api
go run main.go \\
  -redis localhost:6379 \\
  -dbdsn "postgres://gosearch:gosearch@localhost:5432/gosearch?sslmode=disable"`} />

          <h3 className="doc-h3">Frontend</h3>
          <CodeBlock lang="bash" code={`cd frontend
npm install
npm run dev

# UI available at:
# http://localhost:5173`} />

          <h3 className="doc-h3">Useful Local API Checks</h3>
          <CodeBlock lang="bash" code={`# Health check
curl http://localhost:8080/health

# Search
curl -X POST http://localhost:8080/api/v1/search \\
  -H "Content-Type: application/json" \\
  -d '{"query":"machine learning","max_results":10}'

# Stats
curl http://localhost:8080/api/v1/stats`} />

          <h3 className="doc-h3">Subsequent Runs</h3>
          <CodeBlock lang="bash" code={`# From cmd/api
go run main.go

# Force full rebuild
go run main.go -rebuild

# With Redis + PostgreSQL again
go run main.go \\
  -redis localhost:6379 \\
  -dbdsn "postgres://gosearch:gosearch@localhost:5432/gosearch?sslmode=disable"`} />
        </section>

        {/* ── Architecture ── */}
        <section id="architecture" className="doc-section">
          <span className="doc-label">Internals</span>
          <h2 className="doc-h2">Architecture</h2>

          <h3 className="doc-h3">Request Flow</h3>
          <div className="doc-flow">
            {['HTTP Request', 'Gin Router', 'Cache Check', 'Query Analyzer', 'Inverted Index Lookup', 'TF-IDF Scorer', 'Sort & Trim', 'JSON Response'].map((s, i, arr) => (
              <React.Fragment key={s}>
                <span className="doc-flow-step">{s}</span>
                {i < arr.length - 1 && <span className="doc-flow-arrow">→</span>}
              </React.Fragment>
            ))}
          </div>

          <h3 className="doc-h3">Text Analysis Pipeline</h3>
          <div className="doc-pipeline">
            {[
              { step: '1', name: 'Tokenize', detail: 'Split on non-letter/non-number runes' },
              { step: '2', name: 'Lowercase', detail: 'Normalize case for case-insensitive search' },
              { step: '3', name: 'Stopwords', detail: 'Remove high-frequency English words' },
              { step: '4', name: 'Stem', detail: 'Reduce related word forms to a common root' },
            ].map(p => (
              <div className="doc-pipeline-step" key={p.step}>
                <span className="doc-pipeline-num">{p.step}</span>
                <div>
                  <strong>{p.name}</strong>
                  <span className="doc-pipeline-detail">{p.detail}</span>
                </div>
              </div>
            ))}
          </div>

          <h3 className="doc-h3">Concurrency Model</h3>
          <p>Indexing uses a <strong>worker-pool</strong> pattern — <code>runtime.NumCPU()</code> goroutines drain a buffered channel of documents. The inverted index is protected by a <code>sync.RWMutex</code>: multiple readers can query simultaneously, while writers take an exclusive lock. Optional Redis caching reduces repeated TF-IDF work for hot queries, and PostgreSQL logs query analytics without blocking request handling.</p>

          <h3 className="doc-h3">TF-IDF Formula</h3>
          <CodeBlock lang="text" code={`score(term, doc) = TF(term,doc) × IDF(term)

TF  = termFrequency / totalDocTerms
IDF = log(totalDocs / docsContainingTerm)

Final score = Σ TF-IDF for all query terms in doc`} />
        </section>

        {/* ── API Reference ── */}
        <section id="api" className="doc-section">
          <span className="doc-label">Reference</span>
          <h2 className="doc-h2">API Reference</h2>
          <p className="doc-base-url">Base URL: <code>http://localhost:8080/api/v1</code></p>

          <div className="doc-endpoint">
            <div className="doc-endpoint-head">
              <span className="doc-method doc-method--post">POST</span>
              <span className="doc-path">/search</span>
            </div>
            <p>Execute a full-text search with TF-IDF ranking. In the Redis-enabled setup, repeated queries can be served directly from cache.</p>
            <CodeBlock lang="json" code={`// Request
{
  "query": "golang concurrency",
  "max_results": 10,
  "min_score": 0.0
}

// Response
{
  "query": "golang concurrency",
  "results": [
    {
      "document": { "id": 42, "title": "...", "url": "...", "word_count": 312 },
      "score": 0.8421,
      "snippets": ["...highlighted excerpt..."],
      "rank": 1
    }
  ],
  "total_results": 10,
  "time_taken": "1.77ms",
  "success": true,
  "cache_hit": false
}`} />
          </div>

          <div className="doc-endpoint">
            <div className="doc-endpoint-head">
              <span className="doc-method doc-method--get">GET</span>
              <span className="doc-path">/search?q=golang&limit=10</span>
            </div>
            <p>Query-parameter variant — convenient for browser testing.</p>
          </div>

          <div className="doc-endpoint">
            <div className="doc-endpoint-head">
              <span className="doc-method doc-method--get">GET</span>
              <span className="doc-path">/stats</span>
            </div>
            <CodeBlock lang="json" code={`{
  "total_documents": 1000,
  "total_terms": 52366,
  "total_queries": 247,
  "average_query_time": "1.8ms",
  "memory_usage_mb": 84.2,
  "index_size_kb": 5236.6,
  "uptime": "3h2m",
  "cache_available": true,
  "db_available": true
}`} />
          </div>

          <div className="doc-endpoint">
            <div className="doc-endpoint-head">
              <span className="doc-method doc-method--get">GET</span>
              <span className="doc-path">/analytics</span>
            </div>
            <p>Returns query-level analytics when PostgreSQL is enabled.</p>
          </div>

          <div className="doc-endpoint">
            <div className="doc-endpoint-head">
              <span className="doc-method doc-method--get">GET</span>
              <span className="doc-path">/health</span>
            </div>
            <p>Liveness check for load-balancers and container orchestrators.</p>
          </div>

          <h3 className="doc-h3" style={{marginTop:'32px'}}>SDK Examples</h3>
          <CodeBlock lang="python" code={`import requests

r = requests.post(
    "http://localhost:8080/api/v1/search",
    json={"query": "quantum computing", "max_results": 5}
)
for hit in r.json()["results"]:
    print(f"#{hit['rank']} {hit['document']['title']} (score={hit['score']:.4f})")`} />
          <CodeBlock lang="javascript" code={`const res = await fetch("http://localhost:8080/api/v1/search", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ query: "machine learning", max_results: 10 }),
});
const { results } = await res.json();
results.forEach(r => console.log(r.rank, r.document.title, r.score));`} />
        </section>

        {/* ── Docker ── */}
        <section id="docker" className="doc-section">
          <span className="doc-label">Deployment</span>
          <h2 className="doc-h2">Docker & Docker Compose</h2>

          <h3 className="doc-h3">Backend Dockerfile</h3>
          <CodeBlock lang="dockerfile" code={`# ── Stage 1: Build ────────────────────────────────────────────────────────
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \\
    -ldflags="-s -w" \\
    -o gosearch \\
    ./cmd/api

# ── Stage 2: Runtime ──────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /app/gosearch .

RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./gosearch"]`} />

          <h3 className="doc-h3">docker-compose.yml</h3>
          <CodeBlock lang="yaml" code={`version: "3.9"

services:
  backend:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - DATA_DIR=/app/data
      - GIN_MODE=release
    command: >
      ./gosearch
        -port 8080
        -data /app/data
        -redis redis:6379
        -dbdsn postgres://gosearch:gosearch@postgres:5432/gosearch?sslmode=disable
        -limit 1000
    volumes:
      - index_data:/app/data
    depends_on:
      redis:
        condition: service_healthy
      postgres:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 15s
      timeout: 5s
      retries: 3

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:80"
    environment:
      - VITE_API_URL=http://localhost:8080/api/v1
    depends_on:
      - backend
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 3s
      retries: 3

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: gosearch
      POSTGRES_PASSWORD: gosearch
      POSTGRES_DB: gosearch
    volumes:
      - pg_data:/var/lib/postgresql/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U gosearch -d gosearch"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  index_data:
    driver: local
  redis_data:
    driver: local
  pg_data:
    driver: local`} />

          <h3 className="doc-h3">Useful Docker Commands</h3>
          <CodeBlock lang="bash" code={`# Build and start everything
docker compose up --build

# Run in background
docker compose up --build -d

# Start only infra
docker compose up -d redis postgres

# Start backend only
docker compose up backend

# Backend + Redis + PostgreSQL
docker compose up backend redis postgres

# View logs
docker compose logs -f backend

# Stop everything
docker compose down

# Stop and remove volumes
docker compose down -v`} />
        </section>

        {/* ── Features ── */}
        <section id="features" className="doc-section">
          <span className="doc-label">Capabilities</span>
          <h2 className="doc-h2">Feature Set</h2>
          <div className="doc-feature-grid">
            {[
              ['⚡','Inverted Index','O(k) lookups where k = matching documents, not total corpus'],
              ['📊','TF-IDF Ranking','Normalized term frequency × log-IDF scoring'],
              ['🔄','Concurrent Indexing','Worker-pool goroutines, one per CPU core'],
              ['💾','Persistent Index','Saved index data enables faster restarts'],
              ['🔍','NLP Pipeline','Tokenize → lowercase → stopwords → stemming'],
              ['🌐','REST API','Gin router, CORS, health checks, stats, analytics'],
              ['🐋','Docker Ready','Multi-stage Dockerfile + Docker Compose setup'],
              ['📈','Metrics','Query count, latency, cache hit data, memory & index size'],
            ].map(([icon, title, desc]) => (
              <div className="doc-feat" key={title}>
                <span className="doc-feat-icon">{icon}</span>
                <div>
                  <strong>{title}</strong>
                  <p>{desc}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* ── Performance ── */}
        <section id="perf" className="doc-section">
          <span className="doc-label">Benchmarks</span>
          <h2 className="doc-h2">Performance</h2>
          <p>Tested with <code>simplewiki-latest-pages-articles.xml.bz2</code> on Apple Silicon with 8 GB RAM:</p>
          <table className="doc-table">
            <thead>
              <tr><th>Metric</th><th>Value</th></tr>
            </thead>
            <tbody>
              <tr><td>Documents indexed</td><td>1,000 (limit set)</td></tr>
              <tr><td>Unique terms</td><td>52,366</td></tr>
              <tr><td>Query latency</td><td><strong>~1–4 ms</strong></td></tr>
              <tr><td>Index build time (1k docs)</td><td>~2 s including document load</td></tr>
              <tr><td>Index load time (persisted path)</td><td>sub-second target</td></tr>
              <tr><td>Memory (1k docs)</td><td>tens of MB locally</td></tr>
            </tbody>
          </table>
        </section>

      </main>
    </div>
  );
}