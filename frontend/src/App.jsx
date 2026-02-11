import React, { useState, useEffect } from 'react';
import './App.css';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';


function App() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [stats, setStats] = useState(null);
  const [searchTime, setSearchTime] = useState(null);

  // Load engine stats on mount
  useEffect(() => {
    fetchStats();
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
    
    if (!query.trim()) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/search`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          query: query,
          max_results: 20,
        }),
      });

      const data = await response.json();
      
      if (data.success) {
        setResults(data.results || []);
        setSearchTime(data.time_taken);
      } else {
        setError(data.message || 'Search failed');
      }
    } catch (err) {
      setError('Failed to connect to search server');
      console.error('Search error:', err);
    } finally {
      setLoading(false);
    }
  };

  const highlightText = (text, query) => {
    if (!query) return text;
    
    const terms = query.toLowerCase().split(' ');
    let highlighted = text;
    
    terms.forEach(term => {
      const regex = new RegExp(`(${term})`, 'gi');
      highlighted = highlighted.replace(regex, '<mark>$1</mark>');
    });
    
    return <span dangerouslySetInnerHTML={{ __html: highlighted }} />;
  };

  return (
    <div className="App">
      <header className="App-header">
        <div className="container">
          <h1>🔍 GoSearch</h1>
          <p className="subtitle">Fast, relevant, full-text search engine</p>
          
          {stats && (
            <div className="stats-bar">
              <span>{stats.total_documents?.toLocaleString()} documents</span>
              <span>•</span>
              <span>{stats.total_terms?.toLocaleString()} terms</span>
              <span>•</span>
              <span>{stats.total_queries?.toLocaleString()} queries</span>
              <span>•</span>
              <span>~{stats.average_query_time}</span>
            </div>
          )}
        </div>
      </header>

      <main className="container">
        <form onSubmit={handleSearch} className="search-box">
          <input
            type="text"
            placeholder="Search for anything... (e.g., 'machine learning', 'climate change')"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="search-input"
            autoFocus
          />
          <button type="submit" className="search-button" disabled={loading}>
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>

        {error && (
          <div className="error-message">
            ⚠️ {error}
          </div>
        )}

        {searchTime && results.length > 0 && (
          <div className="results-info">
            Found <strong>{results.length}</strong> results in <strong>{searchTime}</strong>
          </div>
        )}

        <div className="results-container">
          {results.map((result, index) => (
            <div key={result.document.id} className="result-card">
              <div className="result-rank">#{result.rank}</div>
              <div className="result-content">
                <h3 className="result-title">
                  {highlightText(result.document.title, query)}
                </h3>
                <p className="result-snippet">
                  {highlightText(
                    result.snippets && result.snippets[0] 
                      ? result.snippets[0] 
                      : result.document.text.substring(0, 200) + '...',
                    query
                  )}
                </p>
                <div className="result-meta">
                  <span className="score">Score: {result.score.toFixed(3)}</span>
                  {result.document.url && (
                    <>
                      <span>•</span>
                      <a 
                        href={result.document.url} 
                        target="_blank" 
                        rel="noopener noreferrer"
                        className="result-link"
                      >
                        View Source →
                      </a>
                    </>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>

        {!loading && !error && results.length === 0 && query && (
          <div className="no-results">
            <p>No results found for "<strong>{query}</strong>"</p>
            <p>Try different keywords or check your spelling</p>
          </div>
        )}
      </main>

      <footer className="footer">
        <div className="container">
          <p>
            Built with ❤️ using Go, React, and TF-IDF ranking
          </p>
          <p>
            <a href="https://github.com/ArshTiwari2004/gosearch" target="_blank" rel="noopener noreferrer">
              View on GitHub
            </a>
          </p>
        </div>
      </footer>
    </div>
  );
}

export default App;