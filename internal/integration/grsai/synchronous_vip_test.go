package grsai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/ai"
)

func TestVIPSingleDispatchUsesMediumAndRejectsWrongQuality(t *testing.T) {
	for _, artwork := range []bool{false, true} {
		t.Run(map[bool]string{false: "edit", true: "artwork"}[artwork], func(t *testing.T) {
			calls, observations := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "/v1/api/generate", r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "gpt-image-2-vip", body["model"])
				require.Equal(t, "medium", body["quality"])
				require.Equal(t, "1024x1024", body["aspectRatio"])
				require.Equal(t, "json", body["replyType"])
				require.NotContains(t, body, "size")
				_, _ = w.Write([]byte(`{"id":"vip-result","status":"succeeded","results":[{"url":"https://output.example/original.png"}]}`))
			}))
			defer server.Close()
			client := NewClient(Config{Model: "gpt-image-2-vip", SubmitURL: server.URL, HTTPClient: server.Client(), MaxAttempts: 4})
			observer := func(context.Context, GenerationObservation) error { observations++; return nil }
			call := func(quality string) error {
				if artwork {
					_, err := client.GenerateImageOnce(context.Background(), &ai.ImageGenerateRequest{Model: "gpt-image-2-vip", Prompt: "approved copy", Size: "1024x1024", Quality: quality, N: 1}, observer)
					return err
				}
				_, err := client.EditImageOnce(context.Background(), &ai.ImageEditRequest{Model: "gpt-image-2-vip", Prompt: "approved copy", Image: []byte("exact reference"), ImageContentType: "image/png", Size: "1024x1024", Quality: quality, N: 1}, observer)
				return err
			}
			require.NoError(t, call("medium"))
			require.Equal(t, 1, calls)
			require.Equal(t, 1, observations)
			for _, wrong := range []string{"auto", "high"} {
				require.Error(t, call(wrong))
			}
			require.Equal(t, 1, calls, "wrong quality must never dispatch")
		})
	}
}
