package grsai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type productImageTransport func(*http.Request) (*http.Response, error)

func (f productImageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestProductImageAsyncUsesOriginalTaskAndExactReference(t *testing.T) {
	for _, origin := range []string{ProductImageBaseURL, "https://grsai.dakka.com.cn"} {
		t.Run(origin, func(t *testing.T) { testProductImageAsyncUsesOriginalTaskAndExactReference(t, origin) })
	}
}

func testProductImageAsyncUsesOriginalTaskAndExactReference(t *testing.T, origin string) {
	data := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 1}
	var requests []string
	transport := productImageTransport(func(r *http.Request) (*http.Response, error) {
		requests = append(requests, r.Method+" "+r.URL.String())
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		var reply string
		if r.Method == http.MethodPost {
			require.Nil(t, r.GetBody)
			var got submitRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
			require.Equal(t, submitRequest{Model: ProductImageModel, Prompt: "Preserve this coat", Images: []string{"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(data)}, AspectRatio: "1024x1024", Quality: "auto", ReplyType: "async"}, got)
			reply = `{"id":"14-task-one","status":"running"}`
		} else {
			require.Equal(t, "14-task-one", r.URL.Query().Get("id"))
			reply = `{"id":"14-task-one","status":"succeeded","results":[{"url":"https://media.example.test/coat.png"}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(reply))}, nil
	})
	client := NewClient(Config{APIKey: "test-key", Model: ProductImageModel, SubmitURL: origin, HTTPClient: &http.Client{Transport: transport}})
	id, err := client.StartProductImage(t.Context(), "Preserve this coat", data)
	require.NoError(t, err)
	require.Equal(t, []string{"POST " + origin + "/v1/api/generate"}, requests)
	result, err := client.ReadProductImageResult(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, "14-task-one", result.TaskID)
	require.Equal(t, ProductImageSucceeded, result.State)
	require.Equal(t, "https://media.example.test/coat.png", result.ResultURL)
	require.Len(t, result.ResultDigest, 64)
	require.Equal(t, []string{"POST " + origin + "/v1/api/generate", "GET " + origin + "/v1/api/result?id=14-task-one"}, requests)
}

func TestProductImageAsyncRejectsUntrustedRouteAndMismatchedResult(t *testing.T) {
	var called bool
	client := NewClient(Config{APIKey: "test-key", Model: ProductImageModel, SubmitURL: "https://other.example.test/v1", HTTPClient: &http.Client{Transport: productImageTransport(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, context.Canceled
	})}})
	_, err := client.StartProductImage(t.Context(), "Product", []byte{0xff, 0xd8, 0xff})
	require.ErrorIs(t, err, ErrProductImagePreflight)
	_, err = client.ReadProductImageResult(t.Context(), "14-task-one")
	require.Error(t, err)
	require.False(t, called)

	client = NewClient(Config{APIKey: "test-key", Model: ProductImageModel, SubmitURL: ProductImageBaseURL, HTTPClient: &http.Client{Transport: productImageTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"wrong-task","status":"succeeded","results":[{"url":"https://media.example.test/coat.png"}]}`))}, nil
	})}})
	_, err = client.ReadProductImageResult(t.Context(), "14-task-one")
	require.Error(t, err)
}

func TestProductImageAsyncDistinguishesPreflightFromAttemptedTransport(t *testing.T) {
	calls := 0
	client := NewClient(Config{APIKey: "test-key", Model: ProductImageModel, SubmitURL: ProductImageBaseURL, HTTPClient: &http.Client{Transport: productImageTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, context.DeadlineExceeded
	})}})
	source := []byte{0xff, 0xd8, 0xff, 0xe0, 0, 1}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := client.StartProductImage(ctx, "Product", source)
	require.ErrorIs(t, err, ErrProductImagePreflight)
	require.Zero(t, calls)
	_, err = client.StartProductImage(t.Context(), "Product", source)
	require.ErrorIs(t, err, errProductImageOutcome)
	require.Equal(t, 1, calls)
}
