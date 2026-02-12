package engine

import (
	"compress/bzip2"
	"encoding/xml"
	"fmt"
	"gosearch/internal/analyzer"
	"io"
	"os"
	"strings"
	"time"
)

// ==========================
// Core Data Structures
// ==========================

// Document represents a searchable document in the search engine.
type Document struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	WordCount int       `json:"word_count"`
	TermCount int       `json:"term_count"`
}

// SearchResult represents a ranked search result.
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"`
	Snippets []string `json:"snippets"`
	Rank     int      `json:"rank"`
}

// ==========================
// Internal Wiki XML Struct
// ==========================

type wikiPage struct {
	Title    string `xml:"title"`
	Revision struct {
		Text string `xml:"text"`
	} `xml:"revision"`
}

// ==========================
// Streaming Wikipedia Loader
// ==========================

// LoadDocuments loads Wikipedia pages using streaming XML parsing.
// It supports .bz2 dumps (real Wikipedia format).
//
// IMPORTANT:
// Use simplewiki dump for testing:
// simplewiki-latest-pages-articles.xml.bz2
//
// The limit parameter prevents loading entire Wikipedia.
func LoadDocuments(path string, limit int) ([]Document, error) {

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Wikipedia dumps are .bz2 compressed
	reader := bzip2.NewReader(file)
	decoder := xml.NewDecoder(reader)

	var documents []Document
	docID := 0
	timestamp := time.Now()

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("XML parsing error: %w", err)
		}

		// Detect <page> elements
		switch start := token.(type) {
		case xml.StartElement:
			if start.Name.Local == "page" {

				var page wikiPage
				if err := decoder.DecodeElement(&page, &start); err != nil {
					return nil, err
				}

				text := strings.TrimSpace(page.Revision.Text)
				if text == "" {
					continue
				}

				// Analyze text
				tokens := analyzer.Analyze(text)

				doc := Document{
					ID:        docID,
					Title:     page.Title,
					URL:       "https://en.wikipedia.org/wiki/" + strings.ReplaceAll(page.Title, " ", "_"),
					Text:      text,
					Timestamp: timestamp,
					WordCount: len(analyzer.Tokenize(text)),
					TermCount: len(tokens),
				}

				documents = append(documents, doc)
				docID++

				// Stop early if limit reached
				if limit > 0 && docID >= limit {
					return documents, nil
				}
			}
		}
	}

	return documents, nil
}

// ==========================
// Utility Methods
// ==========================

// ValidateDocument ensures required fields exist.
func (d *Document) ValidateDocument() error {
	if strings.TrimSpace(d.Text) == "" {
		return fmt.Errorf("document text cannot be empty")
	}
	if strings.TrimSpace(d.Title) == "" {
		return fmt.Errorf("document title cannot be empty")
	}
	return nil
}

// GetPreview returns the first N characters of text.
func (d *Document) GetPreview(maxLength int) string {
	if len(d.Text) <= maxLength {
		return d.Text
	}
	return d.Text[:maxLength] + "..."
}
