import React from "react";

export default function Documentation() {
  return (
    
    <div className="min-h-screen bg-slate-50 text-slate-900 px-6 md:px-16 py-12">
      
      <div className="max-w-5xl mx-auto space-y-16">

        {/* Title */}
        <section>
              <img
          src="/gosearchlogo1.png"
          alt="GoSearch Logo"
          className="mx-auto w-40 md:w-52 lg:w-60 object-contain mb-6"
        />
          <h1 className="text-4xl font-bold mb-6">GoSearch Documentation</h1>
          <p className="mb-4">
  You can view the GitHub of GoSearch:{" "}
  <a
    href="https://github.com/ArshTiwari2004/go-text-search-engine"
    target="_blank"
    rel="noopener noreferrer"
    className="text-blue-600 hover:underline font-medium"
  >
    here
  </a>
</p>

          <p className="text-lg text-slate-600 leading-relaxed">
          GoSearch is a fast full-text search engine built from scratch in Go. It indexes documents using an inverted index and ranks search results with TF-IDF, providing relevant results in milliseconds. Designed for speed, it supports concurrent indexing, persistent storage for quick startup, and a RESTful API making it an efficient, cost-free alternative to heavier search systems like Elasticsearch for datasets under 10 million documents.
          </p>
        </section>

        {/* Installation */}
        <section>
          <h2 className="text-2xl font-semibold mb-4">Installation Guide</h2>

          <div className="bg-white shadow-md rounded-xl p-6 space-y-6">
            
            <div>
              <h3 className="font-semibold mb-2">Prerequisites</h3>
              <ul className="list-disc ml-6 text-slate-600 space-y-1">
                <li>Go 1.21+</li>
                <li>Git</li>
                <li>4GB+ RAM (for large indexing)</li>
              </ul>
            </div>

            <div>
              <h3 className="font-semibold mb-2">Quick Start</h3>
              <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`git clone https://github.com/ArshTiwari2004/go-text-search-engine.git
cd gosearch
go mod download

# Download dataset (optional)
wget https://dumps.wikimedia.org/simplewiki/latest/simplewiki-latest-pages-articles.xml.bz2

go build -o gosearch ./cmd/api
./gosearch -dump simplewiki-latest-pages-articles.xml.bz2 -port 8080`}
              </pre>
            </div>

             <div>
              <h3 className="font-semibold mb-2">Backend setup, run the Go server</h3>
              <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`git clone https://github.com/ArshTiwari2004/go-text-search-engine.git
cd gosearch
cd cmd/api

# Place dataset here:
# simplewiki-latest-pages-articles.xml.bz2

go run main.go

API will start at:
http://localhost:8080 , and you will see a message Starting GoSearch API Server
`}
              </pre>
            </div>


            <div>
              <h3 className="font-semibold mb-2">Frontend Setup (Optional)</h3>
              <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`cd frontend
npm install
npm start`}
              </pre>
            </div>

          </div>
        </section>

        {/* Usage */}
        <section>
          <h2 className="text-2xl font-semibold mb-4">Usage</h2>
          <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`# First run builds index
./gosearch -dump wiki-dump.xml.gz

# Subsequent runs load persisted index
./gosearch

# Force rebuild
./gosearch -rebuild`}
          </pre>
        </section>

        {/* API Documentation */}
        <section>
          <h2 className="text-2xl font-semibold mb-4">API Documentation</h2>

          <div className="bg-white shadow-md rounded-xl p-6 space-y-6">
            
            <div>
              <h3 className="font-semibold">Base URL</h3>
              <p className="text-slate-600">http://localhost:8080/api/v1</p>
            </div>

            <div>
              <h3 className="font-semibold mb-2">POST /search</h3>
              <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`{
  "query": "golang concurrency",
  "max_results": 10
}`}
              </pre>
            </div>

            <div>
              <h3 className="font-semibold mb-2">GET /stats</h3>
              <p className="text-slate-600">
                Returns real-time statistics including document count, term count,
                query count, memory usage, and uptime.
              </p>
            </div>

          </div>
        </section>

        {/* Features */}
        <section>
          <h2 className="text-2xl font-semibold mb-4">Core Features</h2>
          <ul className="list-disc ml-6 text-slate-600 space-y-2">
            <li>Inverted Index for fast lookups</li>
            <li>TF-IDF relevance ranking</li>
            <li>Concurrent indexing (worker pool)</li>
            <li>Persistent storage</li>
            <li>REST API with Gin</li>
            <li>CORS enabled</li>
            <li>Health checks & statistics endpoint</li>
          </ul>
        </section>

        {/* Multi-language Support */}
        <section>
          <h2 className="text-2xl font-semibold mb-4">Can be integrated with any programming language</h2>
          <p className="text-slate-600 mb-4">
            GoSearch exposes a RESTful JSON API. Any programming language that
            supports HTTP requests can integrate with it.
          </p>

          <div className="space-y-6">
            
            <div>
              <h3 className="font-semibold mb-2">Python Example</h3>
              <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`import requests

response = requests.post(
  "http://localhost:8080/api/v1/search",
  json={"query": "golang", "max_results": 5}
)

print(response.json())`}
              </pre>
            </div>

            <div>
              <h3 className="font-semibold mb-2">NodeJS Example</h3>
              <pre className="bg-slate-900 text-green-400 p-4 rounded-lg overflow-x-auto text-sm">
{`const axios = require("axios");

axios.post("http://localhost:8080/api/v1/search", {
  query: "golang",
  max_results: 5
}).then(res => console.log(res.data));`}
              </pre>
            </div>

          </div>
        </section>

        {/* Performance Example */}
        <section>
          <h2 className="text-2xl font-semibold mb-4">Performance Example</h2>
          <p className="text-slate-600">
      While performing testing with simplewiki-latest-pages-articles.xml.bz2 dump file, the search engine had 1,000 documents( limit set intentionally ) indexed with 52,366 unique terms, when searching for "go programming", the frontend sends a POST request to /api/v1/search, and the Go backend performs TF-IDF ranking to return the top results. The query returned 20 results in 1.777208 ms (~1.7 ms), demonstrating very fast processing and low-latency performance.


          </p>
        </section>

      </div>
    </div>
  );
}
