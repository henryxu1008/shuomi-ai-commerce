package einomodel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"unicode/utf8"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	strict "sigs.k8s.io/json"
	"task-processor/internal/aicapability"
)

// SingleImageRequest binds the final serialized request to the original quote.
// It is transient transport state, never a new persisted fact or permission.
type SingleImageRequest struct {
	Image          aicapability.SingleImageInput
	System, Prompt string
}

func validSingleImageContent(i aicapability.SingleImageInput, content []byte) bool {
	if !i.Valid() || len(content) != i.ByteLength {
		return false
	}
	h := sha256.Sum256(content)
	if hex.EncodeToString(h[:]) != i.ContentSHA256 {
		return false
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || cfg.Width != i.Width || cfg.Height != i.Height ||
		format != "jpeg" && format != "png" || "image/"+format != i.MediaType {
		return false
	}
	if format == "png" && !singleFramePNG(content) {
		return false
	}
	_, decodedFormat, err := image.Decode(bytes.NewReader(content))
	return err == nil && decodedFormat == format
}

// The standard decoder verifies PNG/CRC/pixels. This bounded chunk walk only
// excludes APNG animation, which the static decoder otherwise reads as frame 1.
func singleFramePNG(raw []byte) bool {
	if len(raw) < 8 || !bytes.Equal(raw[:8], []byte("\x89PNG\r\n\x1a\n")) {
		return false
	}
	for pos := 8; pos+12 <= len(raw); {
		n := uint64(binary.BigEndian.Uint32(raw[pos : pos+4]))
		if n > uint64(len(raw)-pos-12) {
			return false
		}
		kind := string(raw[pos+4 : pos+8])
		if kind == "acTL" || kind == "fcTL" || kind == "fdAT" {
			return false
		}
		pos += int(n) + 12
		if kind == "IEND" {
			return n == 0 && pos == len(raw)
		}
	}
	return false
}

func validSingleImageWire(raw []byte, cfg GuardConfig) bool {
	if cfg.SingleImage == nil || !cfg.SingleImage.Image.Valid() {
		return false
	}
	var envelope struct {
		Model           string `json:"model"`
		ReasoningEffort string `json:"reasoning_effort"`
		ServiceTier     string `json:"service_tier"`
		Max             int    `json:"max_completion_tokens"`
		N               int    `json:"n"`
		Store           *bool  `json:"store"`
		Stream          *bool  `json:"stream"`
		Messages        []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if !decodeClosedSingleImageJSON(raw, &envelope) ||
		envelope.Model != cfg.ModelID || envelope.Max != cfg.MaximumOutputTokens || envelope.N != 1 ||
		envelope.Store == nil || *envelope.Store || envelope.Stream != nil && *envelope.Stream ||
		envelope.ReasoningEffort != "none" || envelope.ServiceTier != "default" || len(envelope.Messages) != 2 ||
		envelope.Messages[0].Role != "system" || envelope.Messages[1].Role != "user" {
		return false
	}
	var system string
	var parts []struct {
		Type     string `json:"type"`
		Text     string `json:"text"`
		ImageURL *struct {
			URL    string `json:"url"`
			Detail string `json:"detail"`
		} `json:"image_url"`
	}
	if json.Unmarshal(envelope.Messages[0].Content, &system) != nil || system != cfg.SingleImage.System ||
		!decodeClosedSingleImageJSON(envelope.Messages[1].Content, &parts) || len(parts) != 2 ||
		parts[0].Type != "text" || parts[0].Text != cfg.SingleImage.Prompt || parts[0].ImageURL != nil ||
		parts[1].Type != "image_url" || parts[1].Text != "" || parts[1].ImageURL == nil || parts[1].ImageURL.Detail != "high" {
		return false
	}
	i := cfg.SingleImage.Image
	prefix := "data:" + i.MediaType + ";base64,"
	url := parts[1].ImageURL.URL
	if len(url) != len(prefix)+base64.StdEncoding.EncodedLen(i.ByteLength) || len(url) < len(prefix) || url[:len(prefix)] != prefix {
		return false
	}
	content, err := base64.StdEncoding.Strict().DecodeString(url[len(prefix):])
	return err == nil && validSingleImageContent(i, content)
}

func decodeClosedSingleImageJSON(raw []byte, target any) bool {
	if len(raw) == 0 || len(raw) > 1<<20 {
		return false
	}
	violations, err := strict.UnmarshalStrict(raw, target, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	return err == nil && len(violations) == 0
}

func generateSingleImage(ctx context.Context, component model.BaseChatModel, input aicapability.TextInputIdentity, guard *GuardedClient) (TextOutput, error) {
	if input.Image == nil || !validSingleImageContent(*input.Image, input.ImageContent) || guard.cfg.SingleImage == nil {
		return TextOutput{}, ErrNotDispatched
	}
	url := "data:" + input.Image.MediaType + ";base64," + base64.StdEncoding.EncodeToString(input.ImageContent)
	user := &schema.Message{Role: schema.User, UserInputMultiContent: []schema.MessageInputPart{
		{Type: schema.ChatMessagePartTypeText, Text: input.Prompt},
		{Type: schema.ChatMessagePartTypeImageURL, Image: &schema.MessageInputImage{MessagePartCommon: schema.MessagePartCommon{URL: &url}, Detail: schema.ImageURLDetailHigh}},
	}}
	return generateMessages(ctx, component, []*schema.Message{schema.SystemMessage(input.System), user}, guard)
}

func validSingleImageRequest(cfg GuardConfig) bool {
	return cfg.Adapter == AdapterOpenAICompatible && cfg.ProviderID == "openai" &&
		cfg.ModelID == aicapability.SingleImageMiniModel && cfg.SingleImage != nil && cfg.SingleImage.Image.Valid() &&
		cfg.SingleImage.System != "" && cfg.SingleImage.Prompt != "" && utf8.ValidString(cfg.SingleImage.System) && utf8.ValidString(cfg.SingleImage.Prompt) &&
		(cfg.Endpoint == "https://api.openai.com/v1" || cfg.Endpoint == "https://api.openai.com/v1/")
}

func validSingleImageResponseUsage(raw []byte, expectedModel string) bool {
	var response struct {
		Model string `json:"model"`
		Usage *struct {
			PromptDetails *struct {
				Audio *int64 `json:"audio_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionDetails *struct {
				Audio    *int64 `json:"audio_tokens"`
				Accepted *int64 `json:"accepted_prediction_tokens"`
				Rejected *int64 `json:"rejected_prediction_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
	}
	if !decodeOfficialUsage(raw, &response) || response.Model != expectedModel || response.Usage == nil {
		return false
	}
	zero := func(n *int64) bool { return n == nil || *n == 0 }
	if d := response.Usage.PromptDetails; d != nil && !zero(d.Audio) {
		return false
	}
	if d := response.Usage.CompletionDetails; d != nil && (!zero(d.Audio) || !zero(d.Accepted) || !zero(d.Rejected)) {
		return false
	}
	return true
}
