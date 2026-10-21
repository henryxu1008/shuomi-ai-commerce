package imageagentworker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	openai "task-processor/internal/integration/openai"
)

func TestImageSetQuoteUsesOriginalProviderResolutionWithoutCallingProvider(t *testing.T) {
	for _, route := range []struct{ model, quality string }{{"gpt-image-2.5", "auto"}, {"gpt-image-2-vip", "medium"}} {
		t.Run(route.model, func(t *testing.T) {

			requests := 0
			server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
			defer server.Close()
			resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "db:1:1:image_gpt_image_2", Config: &openai.ClientConfig{APIKey: "controlled-only", APIStyle: "grsai", Model: route.model, BaseURL: server.URL, Timeout: time.Minute}}}
			reader := imageGenerationQuoteReader{factory: generationProviderFactory{resolver: resolver, price: config.ImageAgentGenerationConfig{PriceVersion: "price", PointsPerImage: 12}}}
			identity := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "source"}
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member"})
			quote, err := reader.ReadImageGenerationQuote(ctx, identity)
			require.NoError(t, err)
			require.True(t, quote.Valid())
			require.Equal(t, route.quality, quote.Quality)
			require.Equal(t, route.model, quote.Model)
			require.EqualValues(t, 12, quote.Points)
			_, metadata, err := reader.factory.resolve(ctx)
			require.NoError(t, err)
			require.Equal(t, metadata.ConfigurationVersion, quote.ConfigurationVersion)
			require.Equal(t, metadata.CredentialReference, quote.CredentialReference)
			identity.MemberID = "another-member"
			_, err = reader.ReadImageGenerationQuote(ctx, identity)
			require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
			_, err = reader.ReadImageGenerationQuote(context.Background(), identity)
			require.ErrorIs(t, err, imageagent.ErrIdentityRequired)
			require.Zero(t, requests)
			_, err = NewImageGenerationQuoteReader(nil, reader.factory.price)
			require.Error(t, err)
		})
	}
}

func TestImageSetQuoteAdmitsExplicitBeijingQwenRouteWithoutDispatch(t *testing.T) {
	resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "db:1:1:image_gpt_image_2", Config: &openai.ClientConfig{APIKey: "fixture-only", APIStyle: "dashscope", Model: "qwen-image-3.0", BaseURL: "https://workspace.cn-beijing.maas.aliyuncs.com", Timeout: time.Minute}}}
	reader := imageGenerationQuoteReader{factory: generationProviderFactory{resolver: resolver, price: config.ImageAgentGenerationConfig{PriceVersion: "trial-qwen3-v1", PointsPerImage: 1}}}
	id := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, TenantID: "org", UserID: "actor", MemberID: "member"}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member"})
	q, err := reader.ReadImageGenerationQuote(ctx, id)
	require.NoError(t, err)
	require.True(t, q.Valid())
	require.Equal(t, "bailian", q.Provider)
	require.Equal(t, "qwen-image-3.0", q.Model)
	require.Equal(t, "dashscope-image-sync-v1", q.Protocol)
	resolver.resolved.Config.BaseURL = "https://dashscope-intl.aliyuncs.com"
	_, err = reader.ReadImageGenerationQuote(ctx, id)
	require.Error(t, err)
}
