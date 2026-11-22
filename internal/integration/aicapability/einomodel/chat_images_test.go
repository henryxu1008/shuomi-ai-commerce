package einomodel

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/stretchr/testify/require"
)

func chatImageExecutionInput(t *testing.T) aicapability.TextInputIdentity {
	t.Helper()
	in := singleImageExecutionInput(t)
	in.AgentRunID = ""
	in.Operation = aicapability.OperationAIWorkbenchChatImages
	in.Profile.AdapterPolicyVersion = aicapability.ChatImageAdapterPolicy
	in.Profile.MaximumInputBytes = aicapability.MaximumChatImageWire
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 3, 3))))
	second := b.Bytes()
	h := sha256.Sum256(second)
	in.Images = []aicapability.SingleImageInput{*in.Image, {
		SelectionDigest: strings.Repeat("b", 64), ContentSHA256: hex.EncodeToString(h[:]), MediaType: "image/png", ByteLength: len(second), Width: 3, Height: 3,
	}}
	in.ImageContents = [][]byte{in.ImageContent, second}
	in.Image, in.ImageContent = nil, nil
	return in
}

func TestChatImagesExecutorSendsBothInOrderOnce(t *testing.T) {
	in := chatImageExecutionInput(t)
	q, err := aicapability.QuoteText(in)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	sends := 0
	e := singleImageExecutor(in, ledger, imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
		sends++
		var wire struct {
			Messages []struct{ Content json.RawMessage }
		}
		body, readErr := io.ReadAll(r.Body)
		require.NoError(t, readErr)
		require.NoError(t, json.Unmarshal(body, &wire))
		require.Len(t, wire.Messages, 2)
		var parts []struct {
			Text     string
			ImageURL *struct{ URL, Detail string } `json:"image_url"`
		}
		require.NoError(t, json.Unmarshal(wire.Messages[1].Content, &parts))
		require.Len(t, parts, 3)
		require.Equal(t, in.Prompt, parts[0].Text)
		for idx, picture := range in.ImageContents {
			require.Equal(t, "high", parts[idx+1].ImageURL.Detail)
			require.Equal(t, "data:image/png;base64,"+base64.StdEncoding.EncodeToString(picture), parts[idx+1].ImageURL.URL)
		}
		return singleImageResponse(), nil
	}))
	changed := in
	changed.ImageContents = [][]byte{in.ImageContents[1], in.ImageContents[0]}
	_, err = e.Generate(context.Background(), changed, q, nil)
	require.ErrorIs(t, err, ErrNotDispatched)
	require.Empty(t, ledger.records)
	require.Zero(t, sends)

	_, err = e.Generate(context.Background(), in, q, nil)
	require.NoError(t, err)
	require.Equal(t, 1, sends)
	require.Len(t, ledger.records, 2)
	require.Equal(t, aicapability.OperationAIWorkbenchChatImages, ledger.records[0].Operation)
	require.EqualValues(t, 30, ledger.records[1].BillableUsage.InputText)
	_, err = e.Generate(context.Background(), in, q, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.Equal(t, 1, sends)
}
