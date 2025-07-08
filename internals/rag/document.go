package rag

import (
	"fmt"
	"time"
)

type Document struct {
	ID        string    // Unique ID for the document
	Source    string    // e.g., "agency.txt", "FAQ_page.html"
	Content   string    // Full content of the document
	CreatedAt time.Time // Timestamp when the document was added/indexed
}

// Chunk represents a smaller, semantically meaningful piece of a Document.
// These are the units that will be embedded and stored in the vector store.
type Chunk struct {
	ID         string    // Unique ID for the chunk
	DocumentID string    // ID of the parent document
	Content    string    // The text content of the chunk
	Embedding  []float32 // Vector representation of the chunk's content
	Metadata   Metadata  // Additional metadata about the chunk
}

type Metadata struct {
	SourceType string `json:"source_type"`           // e.g., "text_file", "web_page", "database_record"
	PageNumber int    `json:"page_number,omitempty"` // For documents with pages
	Section    string `json:"section,omitempty"`     // For documents with sections
}

func NewChunk(docID, content string, embedding []float32, meta Metadata) Chunk {
	//[TODO] use a UUID generator for IDs.
	// For simplicity, we'll use a basic approach for now.
	chunkID := fmt.Sprintf("%s-%d", docID, time.Now().UnixNano())
	return Chunk{
		ID:         chunkID,
		DocumentID: docID,
		Content:    content,
		Embedding:  embedding,
		Metadata:   meta,
	}
}
