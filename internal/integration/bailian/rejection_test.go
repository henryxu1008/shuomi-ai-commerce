package bailian

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/ai"
)

func TestFreeTierRefusalIsKnownOnlyForCompleteUnambiguous403(t *testing.T) {
	const id = "070f62f5-244a-9703-85c5-a5f0c0d039e0"
	valid := `{"request_id":"` + id + `","code":"AllocationQuota.FreeTierOnly","message":"quota exhausted"}`
	for _, tc := range []struct {
		name, body, header string
		status             int
		known              bool
	}{
		{"quota", valid, id, 403, true},
		{"other status", valid, id, 500, false},
		{"truncated", valid[:len(valid)-1], id, 403, false},
		{"duplicate", strings.Replace(valid, `"code":`, `"code":"Other","code":`, 1), id, 403, false},
		{"output", strings.TrimSuffix(valid, "}") + `,"output":{}}`, id, 403, false},
		{"usage", strings.TrimSuffix(valid, "}") + `,"usage":{"output_image_count":1}}`, id, 403, false},
		{"task", strings.TrimSuffix(valid, "}") + `,"task_id":"accepted"}`, id, 403, false},
		{"id mismatch", valid, "e78918de-1b27-9652-994f-3270179d5da6", 403, false},
		{"unrecognized code", strings.ReplaceAll(valid, "AllocationQuota.FreeTierOnly", "Forbidden"), id, 403, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			cfg := fixtureConfig(imageTransport(func(*http.Request) (*http.Response, error) {
				calls++
				h := make(http.Header)
				h.Set("X-Request-Id", tc.header)
				return &http.Response{StatusCode: tc.status, Header: h, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
			}))
			client, err := NewClient(cfg, func(context.Context, Observation) error { t.Fatal("refusal cannot observe success"); return nil })
			require.NoError(t, err)
			_, err = client.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Prompt: "one cup", Model: Model}, fixtureRoute())
			require.Error(t, err)
			require.Equal(t, !tc.known, errors.Is(err, errUnknown))
			require.Equal(t, 1, calls)
		})
	}
}
