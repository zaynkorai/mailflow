package adapter

import (
	"context"
	"mailflow/internals/rag"
	"mailflow/pkg/logging"
	"math"
	"sort"
	"sync"
)

// InMemoryVectorStore is a simple in-memory implementation of the rag.VectorStore interface.
// This is suitable for development and small datasets.
// [TODO] For production, a dedicated vector DB is recommended.
type InMemoryVectorStore struct {
	mu     sync.RWMutex
	chunks map[string]rag.Chunk // Chunk ID to Chunk
}

func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		chunks: make(map[string]rag.Chunk),
	}
}

// Adds a slice of chunks to the in-memory store.
func (s *InMemoryVectorStore) AddChunks(ctx context.Context, chunks []rag.Chunk) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, chunk := range chunks {
		if _, exists := s.chunks[chunk.ID]; exists {
			logging.Debug("Chunk with ID %s already exists, skipping.", chunk.ID)
			continue
		}
		s.chunks[chunk.ID] = chunk
		logging.Debug("Added chunk %s from document %s", chunk.ID, chunk.DocumentID)
	}
	logging.Info("Successfully added %d chunks to in-memory store. Total chunks: %d", len(chunks), len(s.chunks))
	return nil
}

func (s *InMemoryVectorStore) Search(ctx context.Context, queryEmbedding []float32, topN int) ([]rag.Chunk, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	logging.Info("Searching in-memory store for top %d chunks using cosine similarity...", topN)

	if len(s.chunks) == 0 {
		return nil, nil
	}

	type ScoredChunk struct {
		Chunk rag.Chunk
		Score float64
	}
	var scoredChunks []ScoredChunk

	for _, chunk := range s.chunks {
		if len(chunk.Embedding) == 0 || len(chunk.Embedding) != len(queryEmbedding) {
			logging.Error("Skipping chunk %s due to invalid or mismatched embedding length.", chunk.ID)
			continue
		}
		score := cosineSimilarity(queryEmbedding, chunk.Embedding)
		scoredChunks = append(scoredChunks, ScoredChunk{Chunk: chunk, Score: score})
	}

	sort.Slice(scoredChunks, func(i, j int) bool {
		return scoredChunks[i].Score > scoredChunks[j].Score
	})

	var results []rag.Chunk
	for i := 0; i < len(scoredChunks) && i < topN; i++ {
		results = append(results, scoredChunks[i].Chunk)
	}

	logging.Info("Found %d chunks in search (cosine similarity).", len(results))
	return results, nil
}

// Calculates the cosine similarity between two vectors.
// Cosine similarity = (A . B) / (||A|| * ||B||)
func cosineSimilarity(vecA, vecB []float32) float64 {
	if len(vecA) != len(vecB) {
		return 0.0 // Or return an error, depending on desired behavior
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := 0; i < len(vecA); i++ {
		dotProduct += float64(vecA[i] * vecB[i])
		normA += float64(vecA[i] * vecA[i])
		normB += float64(vecB[i] * vecB[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0 // Avoid division by zero
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *InMemoryVectorStore) GetTotalChunks() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.chunks)
}
