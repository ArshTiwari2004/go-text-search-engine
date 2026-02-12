package engine

import (
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"gosearch/internal/analyzer"
	"os"
	"time"
)

// Document represents a searchable document in the search engine.
// It contains the content and metadata required for indexing and retrieval.
type Document struct {
	ID        int       `json:"id"`                  // Unique identifier assigned during indexing
	Title     string    `json:"title" xml:"title"`   // Document title
	URL       string    `json:"url" xml:"url"`       // Source URL (if applicable)
	Text      string    `json:"text" xml:"abstract"` // Full text content for indexing
	Timestamp time.Time `json:"timestamp"`           // When document was indexed
	WordCount int       `json:"word_count"`          // Total words in document
	TermCount int       `json:"term_count"`          // Unique terms after analysis
}

// SearchResult represents a single result from a search query.
// It includes the document, relevance score, and highlighted snippets.
type SearchResult struct {
	Document Document `json:"document"` // The matched document
	Score    float64  `json:"score"`    // Relevance score (TF-IDF)
	Snippets []string `json:"snippets"` // Text snippets showing query terms in context
	Rank     int      `json:"rank"`     // Position in result list (1-indexed)
}

// LoadDocuments loads documents from a Wikipedia abstract dump file.
// The dump is a compressed XML file containing Wikipedia abstracts.
//
// File format: gzip-compressed XML with structure:
// <feed>
//
//	<doc>
//	  <title>Article Title</title>
//	  <url>https://...</url>
//	  <abstract>Article text...</abstract>
//	</doc>
//	...
//
// </feed>
//
// This function demonstrates:
// - File I/O operations
// - Compression handling (gzip)
// - XML parsing
// - Memory-efficient processing of large datasets
func LoadDocuments(path string) ([]Document, error) {
	// Open the compressed dump file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	// Create a gzip reader to decompress the file on-the-fly
	// This is memory-efficient as it streams decompression
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	// Create an XML decoder that reads from the gzip stream
	dec := xml.NewDecoder(gz)

	// Define the XML structure to unmarshal into
	// This anonymous struct matches the Wikipedia dump format
	dump := struct {
		Documents []Document `xml:"doc"`
	}{}

	// Decode the entire XML structure into memory
	// For very large files (>1GB decompressed), consider streaming XML parsing
	if err := dec.Decode(&dump); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	docs := dump.Documents
	timestamp := time.Now()

	// Process each document to calculate statistics
	for i := range docs {
		docs[i].ID = i
		docs[i].Timestamp = timestamp

		// Calculate word count for TF-IDF normalization
		tokens := analyzer.Analyze(docs[i].Text)
		docs[i].TermCount = len(tokens)

		// Approximate word count (more accurate than term count due to stopword removal)
		docs[i].WordCount = len(analyzer.Tokenize(docs[i].Text))
	}

	return docs, nil
}

// ValidateDocument checks if a document has the minimum required fields
func (d *Document) ValidateDocument() error {
	if d.Text == "" {
		return fmt.Errorf("document text cannot be empty")
	}
	if d.Title == "" {
		return fmt.Errorf("document title cannot be empty")
	}
	return nil
}

// GetPreview returns a preview of the document text (first N characters)
func (d *Document) GetPreview(maxLength int) string {
	if len(d.Text) <= maxLength {
		return d.Text
	}
	return d.Text[:maxLength] + "..."
}
