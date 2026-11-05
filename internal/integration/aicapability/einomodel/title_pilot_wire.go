package einomodel

const (
	TitlePilotOpenAIAdapterPolicy = "openai-title-pilot-v1"
	TitlePilotOpenAIEndpoint      = "https://api.openai.com/v1"
	TitlePilotPromptTokens        = 16_384
	TitlePilotCompletionTokens    = 2_048
)

func validTitlePilotWire(raw []byte, modelID string) bool {
	var body struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		MaxCompletionTokens int    `json:"max_completion_tokens"`
		ReasoningEffort     string `json:"reasoning_effort"`
		ServiceTier         string `json:"service_tier"`
		N                   int    `json:"n"`
		Store               *bool  `json:"store"`
		Stream              *bool  `json:"stream"`
	}
	return decodeClosedSingleImageJSON(raw, &body) &&
		body.Model == modelID && body.MaxCompletionTokens == TitlePilotCompletionTokens &&
		body.ReasoningEffort == "none" && body.ServiceTier == "default" && body.N == 1 &&
		body.Store != nil && !*body.Store && (body.Stream == nil || !*body.Stream) &&
		len(body.Messages) == 2 && body.Messages[0].Role == "system" && body.Messages[0].Content != "" &&
		body.Messages[1].Role == "user" && body.Messages[1].Content != ""
}
