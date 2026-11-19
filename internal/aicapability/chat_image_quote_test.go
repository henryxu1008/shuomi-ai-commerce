package aicapability

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func chatImageQuoteInput(t *testing.T) TextInputIdentity {
	t.Helper()
	in := singleImageQuoteInput(t, nil)
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(raw, &wire))
	wire["AgentRunID"] = ""
	wire["Operation"] = "aiworkbench_chat_image_question"
	wire["Profile"].(map[string]any)["AdapterPolicyVersion"] = "openai-mini-chat-vision-v1"
	wire["Profile"].(map[string]any)["MaximumInputBytes"] = 3 << 20
	wire["Profile"].(map[string]any)["MaximumPromptTokens"] = 30000
	second := map[string]any{"SelectionDigest": strings.Repeat("c", 64), "ContentSHA256": strings.Repeat("d", 64), "MediaType": "image/jpeg", "ByteLength": 512, "Width": 128, "Height": 64}
	wire["Images"] = []any{wire["Image"], second}
	delete(wire, "Image")
	raw, err = json.Marshal(wire)
	require.NoError(t, err)
	var group TextInputIdentity
	require.NoError(t, json.Unmarshal(raw, &group))
	return group
}

func TestChatImageQuoteBindsOrderAndReservesWholeGroup(t *testing.T) {
	in := chatImageQuoteInput(t)
	q, err := QuoteText(in)
	require.NoError(t, err)
	require.Greater(t, q.MaximumTokens-in.Profile.MaximumCompletionTokens, 2*SingleImagePromptTokenBound)

	in.Images[0], in.Images[1] = in.Images[1], in.Images[0]
	reordered, err := QuoteText(in)
	require.NoError(t, err)
	require.NotEqual(t, q.InputHash, reordered.InputHash)
	require.Equal(t, q.MaximumTokens, reordered.MaximumTokens)

	in.Profile.MaximumPromptTokens = q.MaximumTokens - in.Profile.MaximumCompletionTokens - 1
	_, err = QuoteText(in)
	require.ErrorIs(t, err, ErrTextInputLimit)
}

func TestChatImageQuoteRejectsPartialAdmission(t *testing.T) {
	for _, change := range []func(*TextInputIdentity){
		func(in *TextInputIdentity) { in.Operation = OperationProductImageAnalysis },
		func(in *TextInputIdentity) { in.Images = nil },
		func(in *TextInputIdentity) { in.Image = &in.Images[0] },
		func(in *TextInputIdentity) { in.AgentRunID = "foreign-product-run" },
		func(in *TextInputIdentity) { in.Profile.AdapterPolicyVersion = SingleImageMiniAdapterPolicy },
		func(in *TextInputIdentity) {
			in.Images = append(in.Images, in.Images...)
			in.Images = append(in.Images, in.Images[0])
		},
		func(in *TextInputIdentity) {
			in.Images = append(in.Images, in.Images[0])
			for i := range in.Images {
				in.Images[i].ByteLength = MaximumSingleImageBytes
			}
		},
	} {
		in := chatImageQuoteInput(t)
		change(&in)
		_, err := QuoteText(in)
		require.Error(t, err)
	}
}
