package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"mailflow/pkg/logging"
	"net/http"
	"time"
)

const (
	geminiEmbedAPIEndpoint    = "https://generativelanguage.googleapis.com/v1beta/models/embedding-001:batchEmbedContents?key="
	geminiGenerateAPIEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key="
	embeddingModel            = "models/embedding-001"
	generationModel           = "gemini-2.5-flash"
	defaultHTTPTimeout        = 30 * time.Second
)

type GeminiEmbedder struct {
	apiKey string
	client *http.Client
}

func NewGeminiEmbedder(apiKey string) *GeminiEmbedder {
	return &GeminiEmbedder{
		apiKey: apiKey,
		client: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (ge *GeminiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	logging.Debug("Calling Gemini API for embedding text (length: %d)", len(text))

	if ge.apiKey == "" {
		return nil, fmt.Errorf("Google API key is not set for GeminiEmbedder")
	}

	requestBody := EmbedContentRequest{
		Requests: []struct {
			Model   string `json:"model"`
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{
			{
				Model: embeddingModel,
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: text},
					},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", geminiEmbedAPIEndpoint+ge.apiKey, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := ge.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send embedding request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedding response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API returned non-OK status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var response EmbedContentResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal embedding response: %w", err)
	}

	if len(response.Embeddings) == 0 || len(response.Embeddings[0].Values) == 0 {
		return nil, fmt.Errorf("no embedding values returned from Gemini API")
	}

	logging.Debug("Successfully generated embedding of size %d", len(response.Embeddings[0].Values))
	return response.Embeddings[0].Values, nil
}

type GeminiGenerator struct {
	apiKey string
	client *http.Client
}

func NewGeminiGenerator(apiKey string) *GeminiGenerator {
	return &GeminiGenerator{
		apiKey: apiKey,
		client: &http.Client{Timeout: defaultHTTPTimeout},
	}
}

func (gg *GeminiGenerator) GenerateContent(ctx context.Context, prompt string, genConfig *GenerationConfig) (string, error) {
	logging.Debug("Calling Gemini API for content generation (prompt length: %d)", len(prompt))

	if gg.apiKey == "" {
		return "", fmt.Errorf("Google API key is not set for GeminiGenerator")
	}

	requestBody := GenerateContentRequest{
		Contents: []struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			{
				Role: "user",
				Parts: []struct {
					Text string `json:"text"`
				}{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: genConfig,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal generation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", geminiGenerateAPIEndpoint+gg.apiKey, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create generation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := gg.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send generation request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read generation response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned non-OK status: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var response GenerateContentResponse
	err = json.Unmarshal(bodyBytes, &response)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal generation response: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated from Gemini API")
	}

	generatedText := response.Candidates[0].Content.Parts[0].Text
	logging.Debug("Successfully generated content (length: %d)", len(generatedText))
	return generatedText, nil
}
