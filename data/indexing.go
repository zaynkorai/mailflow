package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

const RAG_SEARCH_PROMPT_TEMPLATE = `
	Using the following pieces of retrieved context, answer the question comprehensively and concisely.
	Ensure your response fully addresses the question based on the given context.

	**IMPORTANT:**
	Just provide the answer and never mention or refer to having access to the external context or information in your answer.
	If you are unable to determine the answer from the provided context, state 'I don't know.'

	Question: %s
	Context: %s
	`

// Document represents a chunk of text from a document.
type Document struct {
	Content   string
	Embedding []float32
}

// SimpleVectorStore is a basic in-memory vector store for demonstration.
// In a production environment, you'd use a dedicated vector database like ChromaDB, Pinecone, Weaviate, etc.
type SimpleVectorStore struct {
	docs           []Document
	embeddingModel *genai.EmbeddingModel
	genaiClient    *genai.Client
}

// Creates a new in-memory vector store.
func NewSimpleVectorStore(ctx context.Context, geminiAPIKey string) (*SimpleVectorStore, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %w", err)
	}

	embeddingModel := client.EmbeddingModel("models/text-embedding-004")
	return &SimpleVectorStore{
		docs:           []Document{},
		embeddingModel: embeddingModel,
		genaiClient:    client,
	}, nil
}

func (s *SimpleVectorStore) Close() {
	if s.genaiClient != nil {
		s.genaiClient.Close() // Closes the underlying GenAI client used by the embedding model.

	}
}

// AddDocument adds a document (chunk) to the vector store after embedding its content.
func (s *SimpleVectorStore) AddDocument(ctx context.Context, content string) error {
	resp, err := s.embeddingModel.EmbedContent(ctx, genai.Text(content)) // Use embeddingModel
	if err != nil {
		return fmt.Errorf("failed to embed content: %w", err)
	}

	if resp.Embedding == nil || len(resp.Embedding.Values) == 0 {
		return fmt.Errorf("no embedding values returned for content")
	}
	embedding := resp.Embedding.Values
	s.docs = append(s.docs, Document{Content: content, Embedding: embedding})
	return nil
}

// CosineSimilarity calculates the cosine similarity between two vectors.
func CosineSimilarity(vec1, vec2 []float32) float64 {
	dotProduct := 0.0
	magnitude1 := 0.0
	magnitude2 := 0.0

	for i := 0; i < len(vec1); i++ {
		dotProduct += float64(vec1[i] * vec2[i])
		magnitude1 += float64(vec1[i] * vec1[i])
		magnitude2 += float64(vec2[i] * vec2[i])
	}

	if magnitude1 == 0 || magnitude2 == 0 {
		return 0.0 // Avoid division by zero
	}

	return dotProduct / (math.Sqrt(magnitude1) * math.Sqrt(magnitude2))
}

// Retrieve performs a semantic search on the vector store and returns the topN most similar documents.
func (s *SimpleVectorStore) Retrieve(ctx context.Context, query string, topN int) ([]Document, error) {
	queryResp, err := s.embeddingModel.EmbedContent(ctx, genai.Text(query)) // Use embeddingModel
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	if queryResp.Embedding == nil || len(queryResp.Embedding.Values) == 0 {
		return nil, fmt.Errorf("no embedding values returned for query")
	}
	queryEmbedding := queryResp.Embedding.Values

	type ScoredDocument struct {
		Document Document
		Score    float64
	}
	var scoredDocs []ScoredDocument
	for _, doc := range s.docs {
		score := CosineSimilarity(queryEmbedding, doc.Embedding)
		scoredDocs = append(scoredDocs, ScoredDocument{Document: doc, Score: score})
	}

	// Sort by score (descending) - simple bubble sort for small N,
	// for larger datasets, use sort.Sort interface.
	for i := 0; i < len(scoredDocs)-1; i++ {
		for j := i + 1; j < len(scoredDocs); j++ {
			if scoredDocs[i].Score < scoredDocs[j].Score {
				scoredDocs[i], scoredDocs[j] = scoredDocs[j], scoredDocs[i]
			}
		}
	}

	if len(scoredDocs) > topN {
		returnDocs := make([]Document, topN)
		for i := 0; i < topN; i++ {
			returnDocs[i] = scoredDocs[i].Document
		}
		return returnDocs, nil
	}

	returnDocs := make([]Document, len(scoredDocs))
	for i, sd := range scoredDocs {
		returnDocs[i] = sd.Document
	}
	return returnDocs, nil
}

func TextChunker(text string, chunkSize, chunkOverlap int) []string {
	var chunks []string
	if len(text) == 0 {
		return chunks
	}

	// A simple character-based chunking. For more robust splitting (e.g., by sentences, paragraphs),
	// a more sophisticated logic or a dedicated library would be needed.
	for i := 0; i < len(text); i += (chunkSize - chunkOverlap) {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[i:end])
		if end == len(text) {
			break
		}
	}
	return chunks
}

func main() {

	geminiAPIKey := os.Getenv("GOOGLE_API_KEY")
	if geminiAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable not set.")
	}

	ctx := context.Background()

	llmClient, err := genai.NewClient(ctx, option.WithAPIKey(geminiAPIKey))
	if err != nil {
		log.Fatalf("Failed to create GenAI client for LLM: %v", err)
	}
	defer llmClient.Close()
	llm := llmClient.GenerativeModel("gemini-2.0-flash")
	llm.SetTemperature(0.1)

	vectorstore, err := NewSimpleVectorStore(ctx, geminiAPIKey)
	if err != nil {
		log.Fatalf("Failed to create simple vector store: %v", err)
	}
	defer vectorstore.Close()

	fmt.Println("Loading & Chunking Docs...")
	docContent, err := os.ReadFile("./data/agency.txt")
	if err != nil {
		log.Fatalf("Failed to read document file: %v", err)
	}

	docChunks := TextChunker(string(docContent), 300, 50)
	fmt.Printf("Split into %d chunks.\n", len(docChunks))

	fmt.Println("Creating vector embeddings & storing in memory...")
	// Add chunks to the vector store (embed and store)
	for i, chunk := range docChunks {
		err := vectorstore.AddDocument(ctx, chunk)
		if err != nil {
			log.Fatalf("Failed to add chunk %d to vector store: %v", i, err)
		}
	}
	fmt.Println("Vector embeddings created and stored.")

	// Test RAG chain
	fmt.Println("Test RAG chain...")
	query := "What are your pricing options?"

	// 1. Retrieve relevant context
	retrievedDocs, err := vectorstore.Retrieve(ctx, query, 3) // Retrieve top 3 documents
	if err != nil {
		log.Fatalf("Failed to retrieve documents: %v", err)
	}

	var contextBuilder strings.Builder
	for i, doc := range retrievedDocs {
		contextBuilder.WriteString(fmt.Sprintf("--- Chunk %d ---\n%s\n", i+1, doc.Content))
	}
	context := contextBuilder.String()

	promptText := fmt.Sprintf(RAG_SEARCH_PROMPT_TEMPLATE, query, context)

	resp, err := llm.GenerateContent(ctx, genai.Text(promptText))
	if err != nil {
		log.Fatalf("Failed to generate content from LLM: %v", err)
	}

	var answer string
	if resp != nil && len(resp.Candidates) > 0 && resp.Candidates[0].Content != nil && len(resp.Candidates[0].Content.Parts) > 0 {
		if text, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
			answer = string(text)
		} else {
			answer = "I don't know (LLM response part is not text)."
		}
	} else {
		answer = "I don't know (no valid response from LLM)."
	}

	fmt.Printf("Question: %s\n", query)
	fmt.Printf("Answer: %s\n", answer)
}
