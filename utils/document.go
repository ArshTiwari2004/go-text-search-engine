package utils

import (
	"compress/gzip"
	"encoding/xml"
	"os"
)

// 1. It defines a struct called document that represents a Wikipedia abstract dump document, with fields for the title, URL, text, and an ID.
// 2. It provides a function called LoadDocuments that takes a file path as input
// 3. The function opens the specified file, creates a gzip reader to read the compressed data, and uses an XML decoder to parse the XML content of the file.
// 4. The decoded data is stored in an anonymous struct that contains a slice of document structs, which is then returned as a slice of documents.
// 5. The function also assigns a unique ID to each document based on its index in the slice, allowing for easy identification and retrieval of documents later on.

// document represents a Wikipedia abstract dump document.
type document struct {
	Title string `xml:"title"`
	URL   string `xml:"url"`
	Text  string `xml:"abstract"`
	ID    int
}

// loadDocuments loads a Wikipedia abstract dump and returns a slice of documents.
// Dump example: https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract1.xml.gz
func LoadDocuments(path string) ([]document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f) // newReader creates a new gzip reader that reads from the provided file, allowing the program to read the compressed data from the file
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	dec := xml.NewDecoder(gz)
	dump := struct {
		Documents []document `xml:"doc"` // to define an anonymous struct to hold the decoded XML data, where Documents is a slice of document structs that will be populated with the data from the XML
	}{}
	if err := dec.Decode(&dump); err != nil {
		return nil, err
	}
	docs := dump.Documents // to assign the decoded documents from the dump to a variable called docs for further processing
	for i := range docs {
		docs[i].ID = i // to assign a unique ID to each document based on its index in the slice
	}
	return docs, nil
}
