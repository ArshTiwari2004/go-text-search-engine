import React, { useState, useEffect } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api/v1';

function App() {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [stats, setStats] = useState(null);
  const [searchTime, setSearchTime] = useState(null);

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
    if (!query.trim()) return;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${API_BASE_URL}/search`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: query, max_results: 20 }),
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
    <div className="min-h-screen flex flex-col bg-slate-50 text-slate-900">

      {/* Header */}
      <header className="  py-16 px-4 shadow-lg">
        <div className="max-w-6xl mx-auto text-center">
     <div className="flex flex-col items-center text-center">
  <img
    src="/gosearchlogo1.png"
    alt="GoSearch Logo"
    className="w-40 md:w-56 lg:w-64 object-contain"
  />

  <p className="mt-4 max-w-2xl text-lg opacity-90">
    GoSearch is a lightweight, concurrent full-text search engine built from
    scratch in Golang that indexes large documents and returns relevance-ranked
    results.
  </p>
</div>


          {stats && (
            <div className="mt-6 flex flex-wrap justify-center gap-3 text-sm md:text-base opacity-90">
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

      {/* Main */}
      <main className="flex-1 w-full max-w-6xl mx-auto px-4 md:px-8 -mt-10">

        {/* Search Box */}
        <form
          onSubmit={handleSearch}
          className="bg-white shadow-2xl rounded-2xl p-3 md:p-4 flex flex-col md:flex-row gap-3 border border-slate-200"
        >
          <input
            type="text"
            placeholder="Search for anything... (e.g., 'machine learning', 'climate change')"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="flex-1 px-5 py-4 rounded-xl border border-slate-200 focus:outline-none focus:ring-2 focus:ring-indigo-500 text-base"
            autoFocus
          />
          <button
            type="submit"
            disabled={loading}
            className={`px-8 py-4 rounded-xl font-semibold text-white transition-all duration-300 
              ${loading 
                ? 'bg-indigo-400 animate-pulse cursor-not-allowed' 
                : 'bg-indigo-600 hover:bg-indigo-700 hover:shadow-lg hover:-translate-y-0.5 cursor-pointer'
              }`}
          >
            {loading ? 'Searching...' : 'Search'}
          </button>
        </form>

        {/* Error */}
        {error && (
          <div className="mt-6 bg-red-50 border border-red-200 text-red-600 p-4 rounded-xl text-center">
            ⚠️ {error}
          </div>
        )}

        {/* Results Info */}
        {searchTime && results.length > 0 && (
          <div className="mt-6 text-slate-600">
            Found <strong>{results.length}</strong> results in{' '}
            <strong>{searchTime}</strong>
          </div>
        )}

        {/* Results */}
        <div className="mt-8 space-y-6 pb-16">
          {results.map((result) => (
            <div
              key={result.document.id}
              className="bg-white border border-slate-200 rounded-1xl p-6 "
            >
              <div className="flex flex-col md:flex-row md:items-start gap-4">

                <div className="text-indigo-600 font-bold text-lg md:text-xl min-w-[50px]">
                  #{result.rank}
                </div>

                <div className="flex-1">
                  <h3 className="text-xl md:text-2xl font-semibold mb-2">
                    {highlightText(result.document.title, query)}
                  </h3>

                  <p className="text-slate-600 leading-relaxed mb-4">
                    {highlightText(
                      result.snippets && result.snippets[0]
                        ? result.snippets[0]
                        : result.document.text.substring(0, 200) + '...',
                      query
                    )}
                  </p>

                  <div className="flex flex-wrap items-center gap-3 text-sm text-slate-500">
                    <span className="bg-slate-100 px-3 py-1 rounded-full font-medium">
                      Score: {result.score.toFixed(3)}
                    </span>

                    {result.document.url && (
                      <a
                        href={result.document.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-indigo-600 font-medium hover:underline"
                      >
                        View Source →
                      </a>
                    )}
                  </div>
                </div>

              </div>
            </div>
          ))}
        </div>

        {/* No Results */}
        {!loading && !error && results.length === 0 && query && (
          <div className="text-center mt-16 text-slate-500">
            <p className="text-lg">
              No results found for "<strong>{query}</strong>"
            </p>
            <p className="mt-2">Try different keywords or check your spelling</p>
          </div>
        )}
      </main>

      {/* Footer */}
      <footer className="bg-white border-t border-slate-200 py-8 text-center text-slate-500">
        <a
          href="https://github.com/ArshTiwari2004/go-text-search-engine"
          target="_blank"
          rel="noopener noreferrer"
          className="text-indigo-600 font-medium hover:underline"
        >
          View on GitHub
        </a>
      </footer>
    </div>
  );
}

export default App;
