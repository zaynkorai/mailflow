package rag

import "context"

// VectorStore defines the interface for interacting with a vector database or knowledge store.
// It allows adding chunks (with their embeddings) and searching for relevant chunks.
type VectorStore interface {
	// AddChunks adds a slice of chunks to the store.
	AddChunks(ctx context.Context, chunks []Chunk) error

	// Search searches the store for the top-N most relevant chunks to a given query embedding.
	// It returns the relevant chunks and any error encountered.
	Search(ctx context.Context, queryEmbedding []float32, topN int) ([]Chunk, error)
}
