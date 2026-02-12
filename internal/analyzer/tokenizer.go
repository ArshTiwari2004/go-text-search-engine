package analyzer

import (
	"strings"
	"unicode"

	snowballeng "github.com/kljensen/snowball/english"
)

// Tokenize splits text into individual tokens (words).
// It splits on any character that is not a letter or number.
//
// Why this matters:
// - Tokenization is the first step in text analysis
// - Affects what can be searched and how accurately
// - Must handle punctuation, special characters, etc.
//
// Example:
// Input:  "Hello, world! How's it going?"
// Output: ["Hello", "world", "How", "s", "it", "going"]
//
// Note: Contractions like "How's" become ["How", "s"] which could be improved
// with a more sophisticated tokenizer for production use.
func Tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		// Split on any character that is not a letter or a number
		// This handles most punctuation, whitespace, and special characters
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

// lowercaseFilter normalizes all tokens to lowercase.
// This ensures case-insensitive searching.
//
// Why lowercase normalization:
// - "Apple", "apple", and "APPLE" should match the same documents
// - Reduces index size by eliminating case variations
// - Standard practice in information retrieval
//
// Example:
// Input:  ["Hello", "World", "SEARCH"]
// Output: ["hello", "world", "search"]
func lowercaseFilter(tokens []string) []string {
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = strings.ToLower(token)
	}
	return result
}

// stopwordFilter removes common words that don't add semantic value.
// These are called "stopwords" in information retrieval.
//
// Why remove stopwords:
// - They appear in almost every document ("the", "a", "is")
// - Don't help distinguish documents from each other
// - Reduce index size by ~30-40%
// - Speed up queries (fewer terms to process)
// - Improve relevance (focus on meaningful terms)
//
// Trade-off:
// - Phrases like "to be or not to be" lose meaning
// - For phrase search, stopwords matter
//
// Current list: Basic English stopwords (can be expanded)
// Production systems use lists of 150-500+ stopwords
func stopwordFilter(tokens []string) []string {
	// Stopwords map for O(1) lookup
	// Using struct{} as value type consumes zero memory per entry
	stopwords := map[string]struct{}{
		"a": {}, "and": {}, "be": {}, "have": {}, "i": {},
		"in": {}, "of": {}, "that": {}, "the": {}, "to": {},
		"is": {}, "it": {}, "you": {}, "for": {}, "on": {},
		"with": {}, "as": {}, "at": {}, "by": {}, "an": {},
		"are": {}, "was": {}, "were": {}, "been": {}, "from": {},
	}

	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		// Only keep tokens that are NOT stopwords
		if _, isStopword := stopwords[token]; !isStopword {
			result = append(result, token)
		}
	}
	return result
}

// stemmerFilter reduces words to their root form (stem).
// This uses the Snowball (Porter2) stemming algorithm for English.
//
// Why stemming:
// - "running", "runs", "ran" all become "run"
// - "searching", "searched", "searches" all become "search"
// - Increases recall (finds more relevant documents)
// - Query "running" will match documents with "run", "runs", etc.
//
// Stemming Example:
// - "connection" -> "connect"
// - "connections" -> "connect"
// - "connected" -> "connect"
// - "connecting" -> "connect"
//
// Algorithm: Snowball/Porter2 Stemmer
// - Industry standard for English text
// - Rule-based approach (fast, no ML required)
// - Handles suffixes (-ing, -ed, -s, -tion, etc.)
//
// Trade-off:
// - Can be too aggressive: "university" -> "univers"
// - May merge different words: "generic" and "general" -> "gener"
// - Alternative: Lemmatization (needs dictionary, slower but more accurate)
func stemmerFilter(tokens []string) []string {
	result := make([]string, len(tokens))
	for i, token := range tokens {
		// Snowball stemmer returns the stemmed form
		// Second parameter (false) = don't lowercase (already done)
		result[i] = snowballeng.Stem(token, false)
	}
	return result
}

// Analyze is the main text analysis pipeline.
// It applies all filters in sequence to transform raw text into
// normalized, searchable terms.
//
// Pipeline stages:
// 1. Tokenization: Split into words
// 2. Lowercasing: Normalize case
// 3. Stopword removal: Remove common words
// 4. Stemming: Reduce to root forms
//
// This is the SAME pipeline used for:
// - Indexing documents (building the index)
// - Processing queries (searching the index)
//
// Consistency is crucial: If indexing and querying use different
// analysis, searches won't work correctly!
//
// Example full pipeline:
// Input:  "The cats are running quickly"
// Step 1: ["The", "cats", "are", "running", "quickly"]
// Step 2: ["the", "cats", "are", "running", "quickly"]
// Step 3: ["cats", "running", "quickly"]
// Step 4: ["cat", "run", "quick"]
//
// Performance note:
// - Each filter creates a new slice (immutable approach)
// - For high-volume systems, consider in-place filtering
// - Current approach prioritizes clarity and correctness
func Analyze(text string) []string {
	tokens := Tokenize(text)
	tokens = lowercaseFilter(tokens)
	tokens = stopwordFilter(tokens)
	tokens = stemmerFilter(tokens)
	return tokens
}

// AnalyzeWithoutStemming provides analysis without stemming.
// Useful for exact phrase matching or when stemming is too aggressive.
func AnalyzeWithoutStemming(text string) []string {
	tokens := Tokenize(text)
	tokens = lowercaseFilter(tokens)
	tokens = stopwordFilter(tokens)
	return tokens
}

// GetTermCount returns the number of terms after analysis
// Useful for document length normalization
func GetTermCount(text string) int {
	return len(Analyze(text))
}
