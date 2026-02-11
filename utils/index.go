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

// intersection returns the set intersection between a and b.
// a and b have to be sorted in ascending order and contain no duplicates.
// kind of a merge step in merge sort, but instead of merging, we only keep the common elements.
func Intersection(a []int, b []int) []int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	r := make([]int, 0, maxLen)
	var i, j int
	for i < len(a) && j < len(b) { // to iterate through both sorted slices a and b simultaneously, using two pointers (i and j) to track the current position in each slice, and comparing the elements at those positions to find common elements (the intersection) between the two slices
		if a[i] < b[j] {
			i++
		} else if a[i] > b[j] {
			j++
		} else {
			r = append(r, a[i])
			i++
			j++
		}
	}
	return r
}

// search queries the Index for the given text.
func (idx Index) Search(text string) []int {
	var r []int
	for _, token := range analyze(text) {
		if ids, ok := idx[token]; ok { // to check if the current token from the search query exists in the index, and if it does, the associated list of document IDs is retrieved for further processing
			if r == nil {
				r = ids
			} else {
				r = Intersection(r, ids) // to compute the intersection of the current result set r with the list of document IDs associated with the current token, which ensures that only documents containing all tokens in the search query are returned as results
			}
		} else {
			// Token doesn't exist.
			return nil
		}
	}
	return r
}
