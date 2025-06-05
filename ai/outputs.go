package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/generative-ai-go/genai"
)

type EmailCategory string

const (
	ProductEnquiry    EmailCategory = "PRODUCT_ENQUIRY"
	CustomerComplaint EmailCategory = "CUSTOMER_COMPLAINT"
	CustomerFeedback  EmailCategory = "CUSTOMER_FEEDBACK"
	Unrelated         EmailCategory = "UNRELATED"
)

type CategorizeEmailOutput struct {
	Category EmailCategory `json:"category"`
}

type RAGQueriesOutput struct {
	Queries []string `json:"queries"`
}

type WriterOutput struct {
	Email string `json:"email_content"`
}

type ProofReaderOutput struct {
	Feedback string `json:"feedback"`
	Send     bool   `json:"send"`
}

func callLLMWithStructuredOutput[T any](ctx context.Context, service LLMService, prompt string, parser *jsonResponseParser) (*T, error) {
	resp, err := service.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate LLM content for structured output: %w", err)
	}

	cleanedResponse, err := parser.Parse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	var output T
	err = json.Unmarshal([]byte(cleanedResponse), &output)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal LLM response to struct: %w. Cleaned Response: %s", err, cleanedResponse)
	}

	return &output, nil
}

func callLLMForTextOutput(ctx context.Context, service LLMService, prompt string, parser *textResponseParser) (string, error) {
	resp, err := service.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate LLM content for text output: %w", err)
	}

	textOutput, err := parser.Parse(resp)
	if err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}
	return textOutput, nil
}
