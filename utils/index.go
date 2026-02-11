package utils

// Index is an inverted index. It maps tokens to document IDs.
type Index map[string][]int

// this is a type definition for an Index, which is a map that associates strings (tokens) with slices of integers (document IDs).
// This structure allows for efficient retrieval of document IDs based on the tokens they contain, enabling fast search operations in the full-text search engine.

// add adds documents to the Index.
func (idx Index) Add(docs []document) {
	for _, doc := range docs {
		for _, token := range analyze(doc.Text) { // analyze is a function that processes the text of a document and returns a slice of tokens (words or terms) extracted from the text
			ids := idx[token]                            // to retrieve the current list of document IDs associated with the token from the index
			if ids != nil && ids[len(ids)-1] == doc.ID { // to check if the current document ID is already the last entry in the list of IDs for that token, which helps to avoid adding the same document ID multiple times for the same token
				// Don't add same ID twice.
				continue
			}
			idx[token] = append(ids, doc.ID) // to append the current document ID to the list of IDs for that token in the index, effectively updating the index to include the new document for that token
		}
	}
}

// this file defines the Index type and its Add method, which is responsible for adding documents to the index by analyzing their text and associating tokens with document IDs.
// This is a crucial part of building the inverted index for the full-text search engine.
