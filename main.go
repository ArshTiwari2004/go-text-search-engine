package main

import (
	"flag"
	"log"
	"time"

	utils "github.com/ArshTiwari2004/go-text-search-engine/utils" // to import the utils package which contains functions for loading documents and indexing/searching
)

// the technique used in this code is a simple full-text search engine that loads documents from a wiki abstract dump, indexes them for efficient searching, and then performs a search based on a user-provided query.
// The program measures the time taken for loading, indexing, and searching to provide insights into the performance of each step.

func main() {
	var dumpPath, query string
	flag.StringVar(&dumpPath, "p", "enwiki-latest-stub-articles.xml.gz", "wiki abstract dump path") // to specify the path to the wiki abstract dump file
	flag.StringVar(&query, "q", "Small wild cat", "search query")                                   // to specify the search query
	flag.Parse()                                                                                    // to parse the command-line flags so that the values can be used in the program as in dumpPath and query variables

	log.Println("Running Full Text Search")

	start := time.Now()
	docs, err := utils.LoadDocuments(dumpPath) // to load documents from the dump file from document.go and store them in the docs variable, while also measuring the time taken to load the documents
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Loaded %d documents in %v", len(docs), time.Since(start)) // to log the number of documents loaded and the time taken to load them

	start = time.Now()       // 1st time measurement for indexing
	idx := make(utils.Index) // to create an empty index
	idx.Add(docs)            // to add the loaded documents to the index for efficient searching
	log.Printf("Indexed %d documents in %v", len(docs), time.Since(start))

	start = time.Now()                                                                // 2nd time measurement for search
	matchedIDs := idx.Search(query)                                                   // to search the index for documents matching the query
	log.Printf("Search found %d documents in %v", len(matchedIDs), time.Since(start)) // to log the number of documents found and the time taken to search

	for _, id := range matchedIDs {
		doc := docs[id]                      // to retrieve the document corresponding to the matched ID from the loaded documents
		log.Printf("%d\t%s\n", id, doc.Text) // to log the ID and text of each matched document in a tab-separated format
	}
}
