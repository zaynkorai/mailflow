package llm

type EmbedContentRequest struct {
	Requests []struct {
		Model   string `json:"model"`
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"requests"`
}

type EmbedContentResponse struct {
	Embeddings []struct {
		Values []float32 `json:"values"`
	} `json:"embeddings"`
}

type GenerateContentRequest struct {
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

type GenerationConfig struct {
	ResponseMimeType string   `json:"responseMimeType,omitempty"`
	Temperature      *float32 `json:"temperature,omitempty"`
	TopP             *float32 `json:"topP,omitempty"`
	TopK             *int32   `json:"topK,omitempty"`
	CandidateCount   *int32   `json:"candidateCount,omitempty"`
	MaxOutputTokens  *int32   `json:"maxOutputTokens,omitempty"`
	StopSequences    []string `json:"stopSequences,omitempty"`
}

type GenerateContentResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
		FinishReason  string `json:"finishReason"`
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"candidates"`
	PromptFeedback struct {
		SafetyRatings []struct {
			Category    string `json:"category"`
			Probability string `json:"probability"`
		} `json:"safetyRatings"`
	} `json:"promptFeedback"`
}
