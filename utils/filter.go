package utils

import (
	"strings"

	snowballeng "github.com/kljensen/snowball/english"
)

// lowercaseFilter returns a slice of tokens normalized to lower case.
func lowercaseFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = strings.ToLower(token) // convert each token to lowercase using the strings.ToLower function, which ensures that the search engine treats tokens in a case-insensitive manner, allowing for more flexible and user-friendly search results
	}
	return r
}

// stopwordFilter returns a slice of tokens with stop words removed.
func stopwordFilter(tokens []string) []string {
	var stopwords = map[string]struct{}{
		"a": {}, "and": {}, "be": {}, "have": {}, "i": {},
		"in": {}, "of": {}, "that": {}, "the": {}, "to": {},
	}
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := stopwords[token]; !ok {
			r = append(r, token) // to check if the current token is not a stop word by looking it up in the stopwords map, and if it is not found (i.e., ok is false), the token is appended to the result slice r, effectively filtering out common stop words from the list of tokens that will be used for indexing and searching in the full-text search engine
		}
	}
	return r
}

// stemmerFilter returns a slice of stemmed tokens.
func stemmerFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = snowballeng.Stem(token, false) // to apply stemming to each token using the snowballeng.Stem function from the kljensen/snowball package, which reduces words to their root form (e.g., "running" becomes "run"), allowing for more effective matching of related terms in the search engine and improving search relevance by treating different forms of a word as equivalent
	}
	return r
}
