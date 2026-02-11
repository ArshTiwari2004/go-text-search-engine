package utils

import (
	"strings"
	"unicode"
)

// tokenize returns a slice of tokens for the given text.
func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { // rune is a type that represents a Unicode code point, and the FieldsFunc function is used to split the input text into tokens based on a custom function that determines the delimiters for tokenization
		// Split on any character that is not a letter or a number.
		return !unicode.IsLetter(r) && !unicodeIsNumber(r)
	})
}

// analyze analyzes the text and returns a slice of tokens.
func analyze(text string) []string {
	tokens := tokenize(text)         // to call the tokenize function to split the input text into individual tokens based on non-letter and non-number characters as delimiters
	tokens = lowercaseFilter(tokens) // to call the lowercaseFilter function to convert all tokens to lowercase, ensuring that the search is case-insensitive
	tokens = stopwordFilter(tokens)  // to call the stopwordFilter function to remove common stop words from the list of tokens, which helps to improve search relevance by eliminating words that do not carry significant meaning
	tokens = stemmerFilter(tokens)   // to call the stemmerFilter function to apply stemming to the tokens, which reduces words to their root form (e.g., "running" becomes "run"), allowing for more effective matching of related terms in the search engine
	return tokens                    // to return the final list of processed tokens that can be used for indexing and searching in the full-text search engine
}
