import React, { useState, useEffect, useRef } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

const SAMPLE_QUERIES = [
  'machine learning algorithms',
  'climate change effects',
  'quantum computing basics',
  'black holes formation',
  'ancient civilizations',
];

function App() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [stats, setStats] = useState(null);
  const [searchTime, setSearchTime] = useState(null);
  const [hasSearched, setHasSearched] = useState(false);
  const [placeholder, setPlaceholder] = useState(SAMPLE_QUERIES[0]);
  const inputRef = useRef(null);
  const placeholderIdx = useRef(0);

  useEffect(() => {
    fetchStats();
    const interval = setInterval(() => {
      placeholderIdx.current = (placeholderIdx.current + 1) % SAMPLE_QUERIES.length;
      setPlaceholder(SAMPLE_QUERIES[placeholderIdx.current]);
    }, 3000);
    return () => clearInterval(interval);
  }, []);

  const fetchStats = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/stats`);
      const data = await response.json();
      setStats(data);
    } catch (err) {
      console.error('Failed to fetch stats:', err);
    }
  };

  const handleSearch = async (e) => {
    e.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setError(null);
    setHasSearched(true);

    try {
      const response = await fetch(`${API_BASE_URL}/search`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query, max_results: 20 }),
      });
      const data = await response.json();
      if (data.success) {
        setResults(data.results || []);
        setSearchTime(data.time_taken);
      } else {
        setError(data.message || 'Search failed');
        setResults([]);
      }
    } catch (err) {
      setError('Cannot reach the GoSearch backend. Run it locally, see the Setup Guide.');
      setResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSampleClick = (q) => {
    setQuery(q);
    inputRef.current?.focus();
  };

  const highlightText = (text, query) => {
    if (!query) return text;
    const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
    let html = text;
    terms.forEach(term => {
      const regex = new RegExp(`(${term.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
      html = html.replace(regex, '<mark>$1</mark>');
    });
    return <span dangerouslySetInnerHTML={{ __html: html }} />;
  };

  const isHomepage = !hasSearched;

  return (
    <div className="gs-root">
      {/* ── Notice Banner ── */}
      <div className="gs-banner">
        <span className="gs-banner-dot" />
        Backend not yet deployed — run locally via the&nbsp;
        <a href="/documentation">Setup Guide</a>.
      </div>

      {/* ── HEADER / HERO ── */}
      <header className={`gs-header ${isHomepage ? 'gs-header--hero' : 'gs-header--compact'}`}>
        <div className="gs-header-inner">
          <a href="/" className="gs-logo-link">
            <img src="/gosearchlogo1.png" alt="GoSearch" className="gs-logo" />
          </a>

          {isHomepage && (
            <p className="gs-tagline">
              A <em>concurrent full-text search engine</em> built from scratch in Go with <br />
              TF-IDF ranking, inverted index, millisecond latency.
            </p>
          )}

          {/* ── Search Bar ── */}
          <form className="gs-search-form" onSubmit={handleSearch}>
            <div className="gs-search-wrap">
              <svg className="gs-search-icon" viewBox="0 0 20 20" fill="none">
                <circle cx="8.5" cy="8.5" r="5.5" stroke="currentColor" strokeWidth="1.8" />
                <path d="M13.5 13.5 L17 17" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" />
              </svg>
              <input
                ref={inputRef}
                type="text"
                className="gs-search-input"
                placeholder={`Try "${placeholder}"`}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                autoFocus
              />
              {query && (
                <button type="button" className="gs-search-clear" onClick={() => { setQuery(''); inputRef.current?.focus(); }}>
                  ×
                </button>
              )}
            </div>
            <button type="submit" className={`gs-search-btn ${loading ? 'gs-search-btn--loading' : ''}`} disabled={loading}>
              {loading
                ? <span className="gs-spinner" />
                : <span>Search</span>}
            </button>
          </form>

          {/* Sample queries */}
          {isHomepage && (
            <div className="gs-samples">
              <span className="gs-samples-label">Try:</span>
              {SAMPLE_QUERIES.map(q => (
                <button key={q} className="gs-pill" onClick={() => handleSampleClick(q)}>{q}</button>
              ))}
            </div>
          )}
        </div>
      </header>

      {/* ── MAIN CONTENT ── */}
      <main className="gs-main">

        {/* Stats row */}
        {stats && isHomepage && (
          <div className="gs-stats-row">
            <StatCard icon="📄" value={stats.total_documents?.toLocaleString()} label="documents" />
            <StatCard icon="🔤" value={stats.total_terms?.toLocaleString()} label="unique terms" />
            <StatCard icon="🔍" value={stats.total_queries?.toLocaleString()} label="queries served" />
            <StatCard icon="⚡" value={`~${stats.average_query_time}`} label="avg latency" />
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="gs-error">
            <svg viewBox="0 0 20 20" className="gs-error-icon"><circle cx="10" cy="10" r="9" stroke="currentColor" strokeWidth="1.5" fill="none" /><path d="M10 6v4M10 13h.01" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" /></svg>
            {error}
          </div>
        )}

        {/* Results meta */}
        {!loading && !error && results.length > 0 && (
          <div className="gs-meta">
            <strong>{results.length}</strong> results for <em>"{query}"</em>
            {searchTime && <> &mdash; <strong>{searchTime}</strong></>}
          </div>
        )}

        {/* Results */}
        <div className="gs-results">
          {results.map((result, i) => (
            <ResultCard key={result.document.id} result={result} query={query} highlightText={highlightText} animDelay={i * 40} />
          ))}
        </div>

        {/* No results */}
        {hasSearched && !loading && !error && results.length === 0 && (
          <div className="gs-empty">
            <div className="gs-empty-icon">🔎</div>
            <p>No results for <strong>"{query}"</strong></p>
            <p className="gs-empty-hint">Try different keywords or broader terms</p>
          </div>
        )}

        {/* Homepage feature cards */}
        {isHomepage && (
          <div className="gs-features">
            {FEATURES.map(f => (
              <div className="gs-feature-card" key={f.title}>
                <div className="gs-feature-icon">{f.icon}</div>
                <h3>{f.title}</h3>
                <p>{f.desc}</p>
              </div>
            ))}
          </div>
        )}
      </main>

      {/*footer  */}
    <footer className="gs-footer">
 <span>
  For any suggestions, reach out to me at{" "}
  <a href="mailto:arshtiwari12345@gmail.com">
    arshtiwari12345@gmail.com
  </a>. I’ll reply within 1 day.
</span>

  <span className="gs-footer-sep">·</span>
  <a href="/documentation">Docs</a>

  <span className="gs-footer-sep">·</span>
  <a
    href="https://github.com/ArshTiwari2004/go-text-search-engine"
    target="_blank"
    rel="noopener noreferrer"
  >
    GitHub ↗
  </a>

  <span className="gs-footer-sep">·</span>

  <a
    href="https://www.linkedin.com/in/arsh-tiwari-072609284/"
    target="_blank"
    rel="noopener noreferrer"
  >
    LinkedIn ↗
  </a>

  <span className="gs-footer-sep">·</span>

  <a
    href="https://x.com/ArshTiwari17"
    target="_blank"
    rel="noopener noreferrer"
  >
    X ↗
  </a>

  <span className="gs-footer-sep">·</span>

  <a
    href="https://arsh-portfolio-delta.vercel.app/"
    target="_blank"
    rel="noopener noreferrer"
  >
    Portfolio ↗
  </a>

  <span className="gs-footer-sep">·</span>

  <a
    href="https://www.youtube.com/@TeamSynapse3"
    target="_blank"
    rel="noopener noreferrer"
  >
    YouTube ↗
  </a>
</footer>
    </div>
  );
}

function StatCard({ icon, value, label }) {
  return (
    <div className="gs-stat-card">
      <span className="gs-stat-icon">{icon}</span>
      <span className="gs-stat-value">{value}</span>
      <span className="gs-stat-label">{label}</span>
    </div>
  );
}

function ResultCard({ result, query, highlightText, animDelay }) {
  const snippet = result.snippets?.[0] || result.document.text?.substring(0, 220) + '...';
  return (
    <div className="gs-result-card" style={{ animationDelay: `${animDelay}ms` }}>
      <div className="gs-result-rank">#{result.rank}</div>
      <div className="gs-result-body">
        <h3 className="gs-result-title">{highlightText(result.document.title, query)}</h3>
        <p className="gs-result-snippet">{highlightText(snippet, query)}</p>
        <div className="gs-result-meta">
          <span className="gs-score-badge">Score {result.score.toFixed(4)}</span>
          {result.document.url && (
            <a className="gs-result-link" href={result.document.url} target="_blank" rel="noopener noreferrer">
              View source ↗
            </a>
          )}
          {result.document.word_count > 0 && (
            <span className="gs-word-count">{result.document.word_count.toLocaleString()} words</span>
          )}
        </div>
      </div>
    </div>
  );
}

const FEATURES = [
  { icon: '⚡', title: 'Sub-millisecond Queries', desc: 'Inverted index enables O(k) lookup with k being documents with the term, not total corpus size.' },
  { icon: '📊', title: 'TF-IDF Ranking', desc: 'Term Frequency × Inverse Document Frequency scoring surfaces the most relevant documents first.' },
  { icon: '🔄', title: 'Concurrent Indexing', desc: 'Worker-pool goroutines parallelize document ingestion, saturating all CPU cores during bulk index builds.' },
  { icon: '💾', title: 'Persistent Storage', desc: 'Gob-encoded index snapshots survive restarts, so no re-indexing on boot, under 1 s startup time.' },
  { icon: '🔍', title: 'Text Analysis Pipeline', desc: 'Tokenize → lowercase → stopword removal → Snowball stemming for higher recall without noise.' },
  { icon: '🌐', title: 'REST API', desc: 'Gin-powered HTTP server with CORS, health checks, stats endpoint and autocomplete stubs.' },
];

export default App;