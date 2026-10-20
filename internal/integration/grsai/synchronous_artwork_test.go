package grsai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ai"
)

func TestSynchronousArtworkGeneratesWithoutOriginalAndNeverReposts(t *testing.T) {
	for _, route := range []struct{ model, quality string }{{"gpt-image-2.5", "auto"}, {"gpt-image-2-vip", "medium"}} {
		t.Run(route.model, func(t *testing.T) {

			for _, outcome := range []string{"succeeded", "running", "server_error", "observer_failure"} {
				t.Run(outcome, func(t *testing.T) {
					var submits, queries, observed atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost {
							queries.Add(1)
							w.WriteHeader(500)
							return
						}
						submits.Add(1)
						var body map[string]json.RawMessage
						if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
							w.WriteHeader(400)
							return
						}
						assert.NotContains(t, body, "images", "a source-free creative request must not invent an original")
						assert.Len(t, body, 5)
						assert.JSONEq(t, `"`+route.quality+`"`, string(body["quality"]))
						assert.JSONEq(t, `"original forest animal artwork"`, string(body["prompt"]))
						assert.JSONEq(t, `"1024x1024"`, string(body["aspectRatio"]))
						assert.JSONEq(t, `"json"`, string(body["replyType"]))
						if outcome == "server_error" {
							w.WriteHeader(503)
							return
						}
						status := "succeeded"
						if outcome == "running" {
							status = "running"
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"id": "artwork-1", "status": status, "results": []map[string]string{{"url": "https://example.test/artwork.png"}}})
					}))
					defer server.Close()
					client := NewClient(Config{Model: route.model, SubmitURL: server.URL, MaxAttempts: 3, HTTPClient: server.Client()})
					response, err := client.GenerateImageOnce(context.Background(), &ai.ImageGenerateRequest{Model: route.model, Prompt: "original forest animal artwork", Size: "1024x1024", Quality: route.quality, N: 1}, func(_ context.Context, observation GenerationObservation) error {
						observed.Add(1)
						require.Equal(t, "artwork-1", observation.ResponseID)
						require.Len(t, observation.ResultDigest, 64)
						if outcome == "observer_failure" {
							return errors.New("controlled receipt failure")
						}
						return nil
					})
					require.EqualValues(t, 1, submits.Load(), "configured retries cannot create a second paid artwork")
					require.Zero(t, queries.Load(), "the accepted-job protocol cannot be silently polled")
					if outcome == "succeeded" {
						require.NoError(t, err)
						require.Len(t, response.Data, 1)
					} else {
						require.Error(t, err)
						require.Nil(t, response)
					}
					if outcome == "succeeded" || outcome == "observer_failure" {
						require.EqualValues(t, 1, observed.Load())
					} else {
						require.Zero(t, observed.Load())
					}
				})
			}
		})
	}
}
