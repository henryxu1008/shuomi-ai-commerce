package einomodel

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/schema"
	"google.golang.org/genai"
	"task-processor/internal/aicapability"
)

func geminiThinkingConfig(mode string) *genai.ThinkingConfig {
	switch mode {
	case "budget-zero":
		budget := int32(0)
		return &genai.ThinkingConfig{ThinkingBudget: &budget}
	case "minimal":
		return &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevelMinimal}
	case "low":
		return &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevelLow}
	}
	return nil
}

func validGeminiModelID(s string) bool {
	if len(s) == 0 || len(s) > 128 || s == "." || s == ".." {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

type geminiInputPart struct {
	Text       string `json:"text,omitempty"`
	InlineData *struct {
		MIME string `json:"mimeType"`
		Data string `json:"data"`
	} `json:"inlineData,omitempty"`
}
type geminiInputContent struct {
	Role  string            `json:"role,omitempty"`
	Parts []geminiInputPart `json:"parts"`
}

func validGeminiWire(raw []byte, cfg GuardConfig) bool {
	var e struct {
		System     *geminiInputContent  `json:"systemInstruction"`
		Contents   []geminiInputContent `json:"contents"`
		Generation struct {
			Max        int      `json:"maxOutputTokens"`
			Candidates int      `json:"candidateCount,omitempty"`
			Modalities []string `json:"responseModalities,omitempty"`
			Thinking   *struct {
				Budget  *int32 `json:"thinkingBudget"`
				Level   string `json:"thinkingLevel"`
				Include bool   `json:"includeThoughts,omitempty"`
			} `json:"thinkingConfig"`
		} `json:"generationConfig"`
	}
	if !decodeClosedSingleImageJSON(raw, &e) || e.System == nil || len(e.System.Parts) != 1 ||
		e.System.Parts[0].Text == "" || e.System.Parts[0].InlineData != nil ||
		len(e.Contents) != 1 || e.Contents[0].Role != "user" ||
		e.Generation.Max != cfg.MaximumOutputTokens || e.Generation.Candidates != 0 && e.Generation.Candidates != 1 ||
		len(e.Generation.Modalities) != 1 || e.Generation.Modalities[0] != "TEXT" ||
		e.Generation.Thinking == nil || e.Generation.Thinking.Include {
		return false
	}
	thinking := geminiThinkingConfig(cfg.ThinkingMode)
	if thinking == nil {
		return false
	}
	if thinking.ThinkingBudget != nil {
		if e.Generation.Thinking.Budget == nil || *e.Generation.Thinking.Budget != *thinking.ThinkingBudget || e.Generation.Thinking.Level != "" {
			return false
		}
	} else if e.Generation.Thinking.Budget != nil || e.Generation.Thinking.Level != string(thinking.ThinkingLevel) {
		return false
	}
	parts := e.Contents[0].Parts
	if len(parts) < 1 || parts[0].Text == "" || parts[0].InlineData != nil {
		return false
	}
	if cfg.SingleImage == nil {
		return len(parts) == 1
	}
	if len(parts) != 2 || parts[0].Text != cfg.SingleImage.Prompt || e.System.Parts[0].Text != cfg.SingleImage.System ||
		parts[1].Text != "" || parts[1].InlineData == nil || parts[1].InlineData.MIME != cfg.SingleImage.Image.MediaType {
		return false
	}
	return validBoundBase64(parts[1].InlineData.Data, cfg.SingleImage.Image)
}

func validBoundBase64(data string, i aicapability.SingleImageInput) bool {
	if len(data) != base64.StdEncoding.EncodedLen(i.ByteLength) {
		return false
	}
	b, err := base64.StdEncoding.Strict().DecodeString(data)
	return err == nil && validSingleImageContent(i, b)
}

func validPurposeChatWire(raw []byte, cfg GuardConfig) bool {
	var e struct {
		Model    string `json:"model"`
		Max      int    `json:"max_tokens"`
		Stream   *bool  `json:"stream,omitempty"`
		Thinking *bool  `json:"enable_thinking,omitempty"`
		Template *struct {
			Thinking *bool `json:"enable_thinking"`
		} `json:"chat_template_kwargs,omitempty"`
		N        int   `json:"n,omitempty"`
		Store    *bool `json:"store,omitempty"`
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if !decodeClosedSingleImageJSON(raw, &e) || e.Model != cfg.ModelID || e.Max != cfg.MaximumOutputTokens ||
		e.Stream != nil && *e.Stream || e.N != 0 && e.N != 1 || e.Store != nil && *e.Store ||
		len(e.Messages) != 2 || e.Messages[0].Role != "system" || e.Messages[1].Role != "user" {
		return false
	}
	if cfg.Adapter == AdapterQwenCompatible {
		if e.Thinking == nil || *e.Thinking || e.Template == nil || e.Template.Thinking == nil || *e.Template.Thinking {
			return false
		}
	} else if e.Thinking != nil || e.Template != nil || e.N != 1 || e.Store == nil {
		return false
	}
	var system string
	if json.Unmarshal(e.Messages[0].Content, &system) != nil || system == "" {
		return false
	}
	if cfg.SingleImage == nil {
		var prompt string
		return json.Unmarshal(e.Messages[1].Content, &prompt) == nil && prompt != ""
	}
	var parts []struct {
		Type  string `json:"type"`
		Text  string `json:"text,omitempty"`
		Image *struct {
			URL    string `json:"url"`
			Detail string `json:"detail,omitempty"`
		} `json:"image_url,omitempty"`
	}
	if !decodeClosedSingleImageJSON(e.Messages[1].Content, &parts) || len(parts) != 2 ||
		system != cfg.SingleImage.System || parts[0].Type != "text" || parts[0].Text != cfg.SingleImage.Prompt || parts[0].Image != nil ||
		parts[1].Type != "image_url" || parts[1].Text != "" || parts[1].Image == nil {
		return false
	}
	detail := ""
	if cfg.Adapter == AdapterOpenAICompatible {
		detail = "high"
	}
	if parts[1].Image.Detail != detail {
		return false
	}
	prefix := "data:" + cfg.SingleImage.Image.MediaType + ";base64,"
	return strings.HasPrefix(parts[1].Image.URL, prefix) && validBoundBase64(strings.TrimPrefix(parts[1].Image.URL, prefix), cfg.SingleImage.Image)
}

// Gemini's pinned component returns the same plain text both in Content and
// typed output parts. Accept that representation without admitting generated
// images, code/tool output or giving opaque signatures to business consumers.
func validComponentTextParts(m *schema.Message, adapter AdapterKind) bool {
	if adapter != AdapterGoogleGenerateContent {
		return len(m.MultiContent) == 0 && len(m.AssistantGenMultiContent) == 0
	}
	if len(m.MultiContent) != 0 {
		var text strings.Builder
		for _, p := range m.MultiContent {
			if p.Type != schema.ChatMessagePartTypeText {
				return false
			}
			text.WriteString(p.Text)
		}
		if text.String() != m.Content {
			return false
		}
	}
	var text strings.Builder
	for _, p := range m.AssistantGenMultiContent {
		if p.Type != schema.ChatMessagePartTypeText || p.Text == "" {
			return false
		}
		for key, value := range p.Extra {
			sig, ok := value.([]byte)
			if key != "gemini_thought_signature" || !ok || len(sig) > 1<<20 {
				return false
			}
		}
		text.WriteString(p.Text)
	}
	return len(m.AssistantGenMultiContent) > 0 && text.String() == m.Content
}
