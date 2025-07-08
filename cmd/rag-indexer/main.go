package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"mailflow/internals/config"
	"mailflow/internals/llm"
	"mailflow/internals/rag"
	"mailflow/internals/rag/adapter"
	"mailflow/pkg/logging"
	"math"
	"time"
)

const agencyDataPath = "internals/data/agency.txt"

func main() {
	logging.InitLogger()
	logging.Info("Starting RAG Indexer...")

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Fatal("Failed to load configuration: %v", err)
	}
	logging.Info("Configuration loaded successfully. Google API Key: %s (first 5 chars)", cfg.GoogleAPIKey[:5])

	agencyContent, err := ioutil.ReadFile(agencyDataPath)
	if err != nil {
		logging.Fatal("Failed to read agency data file '%s': %v", agencyDataPath, err)
	}
	logging.Info("Successfully read %d bytes from %s.", len(agencyContent), agencyDataPath)

	chunker := rag.NewSimpleTextChunker(rag.DefaultChunkSize, rag.DefaultChunkOverlap)

	geminiEmbedder := llm.NewGeminiEmbedder(cfg.GoogleAPIKey)
	vectorStore := adapter.NewInMemoryVectorStore()

	ragSystem := rag.NewRAGSystem(chunker, geminiEmbedder, vectorStore)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	doc := rag.Document{
		ID:        "agency-knowledge-base-v1",
		Source:    agencyDataPath,
		Content:   string(agencyContent),
		CreatedAt: time.Now(),
	}

	err = ragSystem.IndexDocument(ctx, doc)
	if err != nil {
		logging.Fatal("Failed to index agency document: %v", err)
	}

	logging.Info("RAG Indexer finished indexing data. Total chunks in store: %d", vectorStore.GetTotalChunks())

	fmt.Println("\n--- Demonstrating RAG Retrieval ---")
	query := "What services does the agency provide?"
	retrievedChunks, err := ragSystem.Retrieve(ctx, query, 3)
	if err != nil {
		logging.Error("Failed to retrieve chunks: %v", err)
	} else {
		logging.Info("Retrieved %d chunks for query '%s':", len(retrievedChunks), query)
		for i, chunk := range retrievedChunks {
			fmt.Printf("Chunk %d (ID: %s, Score: %.4f):\n---\n%s\n---\n", i+1, chunk.ID, calculateSimilarity(query, chunk.Content, geminiEmbedder), chunk.Content)
		}
	}
}

// calculateSimilarity is a helper function to demonstrate similarity for display purposes.
// In a real scenario, the score would come directly from the vector store search.
func calculateSimilarity(text1, text2 string, embedder rag.Embedder) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	emb1, err := embedder.Embed(ctx, text1)
	if err != nil {
		logging.Error("Error embedding text1 for similarity calculation: %v", err)
		return 0.0
	}
	emb2, err := embedder.Embed(ctx, text2)
	if err != nil {
		logging.Error("Error embedding text2 for similarity calculation: %v", err)
		return 0.0
	}

	return cosineSimilarity(emb1, emb2)
}

// cosineSimilarity calculates the cosine similarity between two vectors.
// This is duplicated from inmemory.go for demonstration in main.
func cosineSimilarity(vecA, vecB []float32) float64 {
	if len(vecA) != len(vecB) {
		return 0.0
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
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
