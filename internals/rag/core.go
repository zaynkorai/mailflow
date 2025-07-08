package rag

import (
	"context"
	"fmt"
	"mailflow/pkg/logging"
)

const (
	DefaultChunkSize    = 1000
	DefaultChunkOverlap = 200
)

type TextChunker interface {
	Chunk(text string, docID string, metadata Metadata) ([]Chunk, error)
}

type SimpleTextChunker struct {
	ChunkSize    int
	ChunkOverlap int
}

func NewSimpleTextChunker(chunkSize, chunkOverlap int) *SimpleTextChunker {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		chunkOverlap = DefaultChunkOverlap
	}
	return &SimpleTextChunker{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
	}
}

// Chunk breaks down a large text into smaller, overlapping chunks.
// This is a basic implementation; more sophisticated chunking would consider
// sentence boundaries, paragraphs, or semantic meaning.
func (s *SimpleTextChunker) Chunk(text string, docID string, docMetadata Metadata) ([]Chunk, error) {
	logging.Debug("Chunking text for document %s with size %d and overlap %d", docID, s.ChunkSize, s.ChunkOverlap)
	var chunks []Chunk
	runes := []rune(text)
	textLen := len(runes)

	if textLen == 0 {
		return nil, fmt.Errorf("cannot chunk empty text")
	}

	start := 0
	for start < textLen {
		end := start + s.ChunkSize
		if end > textLen {
			end = textLen
		}

		chunkContent := string(runes[start:end])
		chunk := NewChunk(docID, chunkContent, nil, docMetadata) // Embedding is nil initially
		chunks = append(chunks, chunk)

		if end == textLen {
			break
		}

		// Move start pointer for the next chunk, considering overlap
		start += (s.ChunkSize - s.ChunkOverlap)
		if start >= textLen { // Ensure we don't go past the end
			break
		}
	}
	logging.Info("Text chunked into %d pieces for document %s.", len(chunks), docID)
	return chunks, nil
}

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// Orchestrater of chunking, embedding, and storage of documents.
type RAGSystem struct {
	Chunker     TextChunker
	Embedder    Embedder
	VectorStore VectorStore
}

func NewRAGSystem(chunker TextChunker, embedder Embedder, store VectorStore) *RAGSystem {
	return &RAGSystem{
		Chunker:     chunker,
		Embedder:    embedder,
		VectorStore: store,
	}
}

// IndexDocument processes a document by chunking its content, embedding each chunk,
// and adding them to the vector store.
func (r *RAGSystem) IndexDocument(ctx context.Context, doc Document) error {
	logging.Info("Indexing document: %s (Source: %s)", doc.ID, doc.Source)

	chunks, err := r.Chunker.Chunk(doc.Content, doc.ID, Metadata{SourceType: "text_file"}) // Assuming text_file for agency.txt
	if err != nil {
		return fmt.Errorf("failed to chunk document %s: %w", doc.ID, err)
	}

	// Embed each chunk and prepare for storage
	var embeddedChunks []Chunk
	for i, chunk := range chunks {
		logging.Debug("Embedding chunk %d/%d for document %s...", i+1, len(chunks), doc.ID)
		embedding, err := r.Embedder.Embed(ctx, chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to embed chunk %s for document %s: %w", chunk.ID, doc.ID, err)
		}
		chunk.Embedding = embedding
		embeddedChunks = append(embeddedChunks, chunk)
	}

	err = r.VectorStore.AddChunks(ctx, embeddedChunks)
	if err != nil {
		return fmt.Errorf("failed to add chunks to vector store for document %s: %w", doc.ID, err)
	}

	logging.Info("Successfully indexed document: %s. Added %d chunks.", doc.ID, len(chunks))
	return nil
}

func (r *RAGSystem) Retrieve(ctx context.Context, query string, topN int) ([]Chunk, error) {
	logging.Info("Retrieving chunks for query: '%s'", query)

	queryEmbedding, err := r.Embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	relevantChunks, err := r.VectorStore.Search(ctx, queryEmbedding, topN)
	if err != nil {
		return nil, fmt.Errorf("failed to search vector store: %w", err)
	}

	logging.Info("Retrieved %d relevant chunks for query.", len(relevantChunks))
	return relevantChunks, nil
}
