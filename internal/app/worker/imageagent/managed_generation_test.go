package imageagentworker

import (
	"context"
	"errors"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/imageagent"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type imagePolicyFixture struct {
	revision d.Revision
	err      error
}

func (f *imagePolicyFixture) Current(_ context.Context, sc d.Scope) (d.Revision, error) {
	if sc.OrganizationID != "org-a" {
		return d.Revision{}, d.ErrUnavailable
	}
	return f.revision, f.err
}
func TestManagedImageFactoryReadsFreshPriceAndRejectsDriftWithoutFallback(t *testing.T) {
	for _, route := range []struct{ model, quality string }{{"gpt-image-2.5", "auto"}, {"gpt-image-2-vip", "medium"}} {
		t.Run(route.model, func(t *testing.T) {

			endpoint := "https://provider.example.test/v1"
			current := &imagePolicyFixture{revision: d.Revision{ID: "route-a", Payload: d.Payload{BindingID: "image-b", Image: &d.ImageProfile{Provider: "grsai", Model: route.model, Protocol: "grsai-json-sync-v1", EndpointDigest: governed.EndpointIdentityDigest(endpoint), CredentialReference: "db:2:3:image-b", ConfigurationVersion: "route-a", PriceVersion: "route-a", Points: 3}}}}
			resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "db:2:3:image-b", Config: &openai.ClientConfig{APIKey: "fixture", APIStyle: "grsai", Model: route.model, BaseURL: endpoint, Timeout: time.Second}}}
			f := generationProviderFactory{resolver: resolver, current: current}
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "member", TenantID: "org-a", EffectiveOrganizationID: "org-a"})
			_, first, e := f.resolve(ctx)
			require.NoError(t, e)
			require.EqualValues(t, 3, first.Points)
			current.revision.Payload.Image.PriceVersion = "route-b"
			current.revision.Payload.Image.ConfigurationVersion = "route-b"
			current.revision.Payload.Image.Points = 5
			require.ErrorIs(t, f.revalidate(ctx, first), imageagent.ErrRevisionConflict)
			_, latest, e := f.resolve(ctx)
			require.NoError(t, e)
			require.EqualValues(t, 5, latest.Points)
			current.err = errors.New("database unavailable")
			_, _, e = f.resolve(ctx)
			require.Error(t, e)
			current.err = nil
			resolver.resolved.CacheKey = "rotated"
			_, _, e = f.resolve(ctx)
			require.Error(t, e)
		})
	}
}
