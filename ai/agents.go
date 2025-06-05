package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/zaynkorai/mailflow/prompts"
)

type Agents struct {
	llmService LLMService
	jsonParser *jsonResponseParser
	textParser *textResponseParser
}

func NewAgents(ctx context.Context, googleAPIKey string) (*Agents, error) {
	geminiService, err := NewGeminiService(ctx, googleAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Gemini service: %w", err)
	}

	return &Agents{
		llmService: geminiService,
		jsonParser: NewJSONResponseParser(),
		textParser: NewTextResponseParser(),
	}, nil
}

func (a *Agents) CategorizeEmail(ctx context.Context, emailBody string) (*CategorizeEmailOutput, error) {
	prompt := fmt.Sprintf(prompts.CATEGORIZE_EMAIL, emailBody)
	output, err := callLLMWithStructuredOutput[CategorizeEmailOutput](ctx, a.llmService, prompt, a.jsonParser)
	if err != nil {
		return nil, fmt.Errorf("failed to categorize email: %w", err)
	}
	return output, nil
}

func (a *Agents) DesignRAGQueries(ctx context.Context, emailBody string) (*RAGQueriesOutput, error) {
	prompt := fmt.Sprintf(prompts.GENERATE_RAG_QUERIES, emailBody)
	output, err := callLLMWithStructuredOutput[RAGQueriesOutput](ctx, a.llmService, prompt, a.jsonParser)
	if err != nil {
		return nil, fmt.Errorf("failed to design RAG queries: %w", err)
	}
	return output, nil
}

func (a *Agents) GenerateRAGAnswer(ctx context.Context, contextStr, question string) (string, error) {
	prompt := fmt.Sprintf(prompts.GENERATE_RAG_ANSWER, contextStr, question)
	answer, err := callLLMForTextOutput(ctx, a.llmService, prompt, a.textParser)
	if err != nil {
		return "", fmt.Errorf("failed to generate RAG answer: %w", err)
	}
	return answer, nil
}

func (a *Agents) EmailWriter(ctx context.Context, emailInformation string, history []string) (*WriterOutput, error) {
	fullPrompt := prompts.EMAIL_WRITER + "\n\n"
	if len(history) > 0 {
		fullPrompt += "History of previous drafts and feedback:\n" + strings.Join(history, "\n") + "\n\n"
	}
	fullPrompt += "Instructions:\n" + emailInformation

	output, err := callLLMWithStructuredOutput[WriterOutput](ctx, a.llmService, fullPrompt, a.jsonParser)
	if err != nil {
		return nil, fmt.Errorf("failed to write email draft: %w", err)
	}
	return output, nil
}

func (a *Agents) EmailProofreader(ctx context.Context, initialEmail, generatedEmail string) (*ProofReaderOutput, error) {
	prompt := fmt.Sprintf(prompts.EMAIL_PROOFREADER, initialEmail, generatedEmail)
	output, err := callLLMWithStructuredOutput[ProofReaderOutput](ctx, a.llmService, prompt, a.jsonParser)
	if err != nil {
		return nil, fmt.Errorf("failed to proofread email: %w", err)
	}
	return output, nil
}
