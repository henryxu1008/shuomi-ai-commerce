package grsai

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"task-processor/internal/ai"
	productimage "task-processor/internal/product/image"
	"testing"
)

func TestNewSourceImageAdapterInjectsExactRoutePinnedGRSAIClient(t *testing.T) {
	calls, observed := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var wire submitRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&wire))
		require.Equal(t, "gpt-image-2.5", wire.Model)
		_, _ = w.Write([]byte(`{"id":"result","status":"succeeded","results":[{"url":"https://images.example/result.png"}]}`))
	}))
	defer server.Close()
	var pinned ai.RouteBoundImageEditor
	adapter, err := NewSynchronousSourceImageAdapter(SourceImageAdapterConfig{Client: NewClient(Config{Model: "gpt-image-2.5", SubmitURL: server.URL, HTTPClient: server.Client()}), ImageModel: "gpt-image-2.5", RouteReference: "route-a", CredentialReference: "credential-a", ConfigurationVersion: "config-v1", Build: func(client ai.RouteBoundImageEditor) (SourceImageProvider, error) {
		pinned = client
		return &grsaiSourceImageProviderStub{}, nil
	}}, func(context.Context, GenerationObservation) error { observed++; return nil })
	require.NoError(t, err)
	require.NotNil(t, adapter)
	require.NotNil(t, pinned)
	_, err = pinned.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: "gpt-image-2.5", Quality: "auto", Prompt: "approved", Image: []byte("source"), ImageContentType: "image/png"}, ai.ImageRouteSelection{CredentialReference: "credential-a", ConfigurationVersion: "config-v1"})
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, 1, observed)
}
func TestPinnedProductImageClientRejectsRouteDriftBeforeGRSAIDispatch(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	client := NewClient(Config{Model: "gpt-image-2.5", SubmitURL: server.URL, HTTPClient: server.Client()})
	observer := func(context.Context, GenerationObservation) error { return nil }
	pinned, err := newPinnedProductImageClient(client, "gpt-image-2.5", "credential-a", "config-v1", observer)
	require.NoError(t, err)
	_, err = pinned.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: "gpt-image-2.5", Quality: "auto", Prompt: "approved", Image: []byte("source"), ImageContentType: "image/png"}, ai.ImageRouteSelection{CredentialReference: "credential-b", ConfigurationVersion: "config-v1"})
	require.Error(t, err)
	require.Zero(t, calls)
	_, err = newPinnedProductImageClient(client, "gpt-image-2-vip", "credential-a", "config-v1", observer)
	require.Error(t, err)
	require.Zero(t, calls)
}
func TestGRSAISourceImageAdapterRejectsQuoteRouteDrift(t *testing.T) {
	delegate := &grsaiSourceImageProviderStub{quote: productimage.UsageQuote{Operation: "render_scene", RouteReference: "route-b", Model: "gpt-image-2.5", CredentialReference: "credential-a", ConfigurationVersion: "config-v1"}}
	adapter, err := NewSynchronousSourceImageAdapter(SourceImageAdapterConfig{Client: NewClient(Config{Model: "gpt-image-2.5"}), ImageModel: "gpt-image-2.5", RouteReference: "route-a", CredentialReference: "credential-a", ConfigurationVersion: "config-v1", Build: func(ai.RouteBoundImageEditor) (SourceImageProvider, error) { return delegate, nil }}, func(context.Context, GenerationObservation) error { return nil })
	require.NoError(t, err)
	_, err = adapter.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: "render_scene", InputFingerprint: "input-v1", MaximumOutputs: 1})
	require.ErrorIs(t, err, productimage.ErrCapabilityUnsupported)
}
func TestNewSourceImageAdapterRejectsMissingFactoryAndTypedNilDelegate(t *testing.T) {
	base := SourceImageAdapterConfig{Client: NewClient(Config{Model: "gpt-image-2.5"}), ImageModel: "gpt-image-2.5", RouteReference: "route-a", CredentialReference: "credential-a", ConfigurationVersion: "config-v1"}
	observe := func(context.Context, GenerationObservation) error { return nil }
	_, err := NewSynchronousSourceImageAdapter(base, observe)
	require.Error(t, err)
	var provider *grsaiSourceImageProviderStub
	base.Build = func(ai.RouteBoundImageEditor) (SourceImageProvider, error) { return provider, nil }
	_, err = NewSynchronousSourceImageAdapter(base, observe)
	require.Error(t, err)
	base.Client = nil
	_, err = NewSynchronousSourceImageAdapter(base, observe)
	require.Error(t, err)
}

type grsaiSourceImageProviderStub struct{ quote productimage.UsageQuote }

func (*grsaiSourceImageProviderStub) EditSourceWhiteBackground(context.Context, productimage.SourceWhiteBackgroundRequest) (productimage.Candidate, error) {
	return productimage.Candidate{}, nil
}
func (p *grsaiSourceImageProviderStub) QuoteUsage(context.Context, productimage.UsageQuoteRequest) (productimage.UsageQuote, error) {
	return p.quote, nil
}

func (*grsaiSourceImageProviderStub) EditSources(context.Context, productimage.SourceEditRequest) (productimage.Candidate, error) {
	return productimage.Candidate{}, nil
}
