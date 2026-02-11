package engine

import (
	"sort"
	"sync"
)

// Posting represents a single occurrence of a term in a document.
// It stores not just the document ID, but also term frequency information
// which is crucial for TF-IDF ranking.
type Posting struct {
	DocID         int // Document identifier
	TermFrequency int // Number of times the term appears in this document
	DocLength     int // Total number of terms in the document (for normalization)
}

// Index is an inverted index that maps terms to their posting lists.
// An inverted index is the fundamental data structure in information retrieval.
//
// Why "inverted"?
// - Normal index: Document -> Terms it contains
// - Inverted index: Term -> Documents containing it
//
// This inversion enables O(k) lookup time where k is the number of documents
// containing the term, rather than O(n) where n is total documents.
//
// Example:
// Doc1: "go is fast"
// Doc2: "go is simple"
// Doc3: "fast and simple"
//
// Inverted Index:
// "go"     -> [{DocID: 1, TF: 1}, {DocID: 2, TF: 1}]
// "is"     -> [{DocID: 1, TF: 1}, {DocID: 2, TF: 1}]
// "fast"   -> [{DocID: 1, TF: 1}, {DocID: 3, TF: 1}]
// "simple" -> [{DocID: 2, TF: 1}, {DocID: 3, TF: 1}]
// "and"    -> [{DocID: 3, TF: 1}]
type Index struct {
	Terms map[string][]Posting // Maps each term to its posting list
	mu    sync.RWMutex         // Protects concurrent access to the index
}

// NewIndex creates a new empty inverted index
func NewIndex() *Index {
	return &Index{
		Terms: make(map[string][]Posting),
	}
}

// AddDocument adds a single document to the inverted index.
// This method:
// 1. Analyzes the document text (tokenization, normalization, stopword removal, stemming)
// 2. Counts term frequencies
// 3. Updates the inverted index with new postings
//
// Thread-safety: This method uses a write lock to ensure safe concurrent access
func (idx *Index) AddDocument(doc Document) {
	// Analyze document text to get normalized terms
	terms := Analyze(doc.Text)

	// Count term frequencies in this document
	// This is essential for TF-IDF calculation
	termFreqs := make(map[string]int)
	for _, term := range terms {
		termFreqs[term]++
	}

	docLength := len(terms) // Total terms in document (after analysis)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Add posting for each unique term
	for term, freq := range termFreqs {
		posting := Posting{
			DocID:         doc.ID,
			TermFrequency: freq,
			DocLength:     docLength,
		}

		// Append to posting list for this term
		idx.Terms[term] = append(idx.Terms[term], posting)
	}
}

// Add is a batch version of AddDocument for backward compatibility
func (idx *Index) Add(docs []Document) {
	for _, doc := range docs {
		idx.AddDocument(doc)
	}
}

// GetPostings retrieves the posting list for a given term.
// Returns nil if the term doesn't exist in the index.
func (idx *Index) GetPostings(term string) []Posting {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return idx.Terms[term]
}

// Search performs a boolean AND search across multiple query terms.
// This is the original search method that returns document IDs without ranking.
//
// Algorithm:
// 1. For each query term, get its posting list
// 2. Find the intersection of all posting lists
// 3. Return document IDs that contain ALL query terms
//
// Time Complexity: O(n * k) where n is average posting list size, k is number of terms
// Space Complexity: O(min posting list size)
func (idx *Index) Search(text string) []int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	queryTerms := Analyze(text)
	if len(queryTerms) == 0 {
		return nil
	}

	// Start with posting list of first term
	var resultIDs []int
	firstTerm := queryTerms[0]

	postings, exists := idx.Terms[firstTerm]
	if !exists {
		return nil // First term not found, no results possible
	}

	// Extract document IDs from first posting list
	resultIDs = make([]int, len(postings))
	for i, p := range postings {
		resultIDs[i] = p.DocID
	}

	// Intersect with remaining terms
	for _, term := range queryTerms[1:] {
		postings, exists := idx.Terms[term]
		if !exists {
			return nil // Any term not found means no results
		}

		// Extract doc IDs from this term's postings
		termIDs := make([]int, len(postings))
		for i, p := range postings {
			termIDs[i] = p.DocID
		}

		// Compute intersection
		resultIDs = intersection(resultIDs, termIDs)

		if len(resultIDs) == 0 {
			return nil // Early termination if no common documents
		}
	}

	return resultIDs
}

// intersection computes the set intersection of two sorted integer slices.
// Both input slices must be sorted in ascending order.
//
// This uses a two-pointer merge algorithm similar to merge sort:
// - Time Complexity: O(n + m) where n, m are slice lengths
// - Space Complexity: O(min(n, m)) for the result
//
// Why this is efficient:
// - Posting lists are naturally sorted by DocID during indexing
// - Two pointer merge is cache-friendly and branch-predictor friendly
// - Much faster than hash-based intersection for sorted data
func intersection(a, b []int) []int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}

	result := make([]int, 0, maxLen)
	i, j := 0, 0

	// Two-pointer merge algorithm
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			i++ // Element in 'a' only, skip it
		} else if a[i] > b[j] {
			j++ // Element in 'b' only, skip it
		} else {
			// Found common element
			result = append(result, a[i])
			i++
			j++
		}
	}

	return result
}

// GetTermCount returns the number of unique terms in the index
func (idx *Index) GetTermCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	return len(idx.Terms)
}

// GetDocumentFrequency returns how many documents contain a given term
// This is used in IDF calculation: IDF = log(N / DF)
func (idx *Index) GetDocumentFrequency(term string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	postings, exists := idx.Terms[term]
	if !exists {
		return 0
	}

	return len(postings)
}

// GetTopTerms returns the N most common terms in the index
// Useful for analytics and understanding the corpus
func (idx *Index) GetTopTerms(n int) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	type termFreq struct {
		term string
		freq int
	}

	terms := make([]termFreq, 0, len(idx.Terms))
	for term, postings := range idx.Terms {
		// Sum up term frequencies across all documents
		totalFreq := 0
		for _, p := range postings {
			totalFreq += p.TermFrequency
		}
		terms = append(terms, termFreq{term, totalFreq})
	}

	// Sort by frequency descending
	sort.Slice(terms, func(i, j int) bool {
		return terms[i].freq > terms[j].freq
	})

	// Return top N
	if len(terms) > n {
		terms = terms[:n]
	}

	result := make([]string, len(terms))
	for i, tf := range terms {
		result[i] = tf.term
	}

	return result
}

// Clear removes all terms from the index
func (idx *Index) Clear() {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	idx.Terms = make(map[string][]Posting)
}
