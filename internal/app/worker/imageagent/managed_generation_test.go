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

type optionalImagePolicyFixture struct {
	*imagePolicyFixture
	option d.Revision
}

func (f *optionalImagePolicyFixture) Optional(_ context.Context, sc d.Scope, binding string) (d.Revision, error) {
	if sc.OrganizationID != "org-a" || binding != f.option.Payload.BindingID {
		return d.Revision{}, d.ErrNotFound
	}
	return f.option, nil
}
func (f *optionalImagePolicyFixture) Options(context.Context, d.Scope) ([]d.Revision, error) {
	return []d.Revision{f.option}, nil
}

func TestExplicitVIPResolutionAndFinalCheckDoNotReadDefault(t *testing.T) {
	endpoint := "https://provider.example.test/v1"
	policy := &optionalImagePolicyFixture{imagePolicyFixture: &imagePolicyFixture{err: errors.New("default deliberately unavailable")}, option: d.Revision{ID: "vip-r1", Payload: d.Payload{Optional: true, BindingID: "image-vip", Image: &d.ImageProfile{Provider: "grsai", Model: "gpt-image-2-vip", Protocol: "grsai-json-sync-v1", EndpointDigest: governed.EndpointIdentityDigest(endpoint), CredentialReference: "db:2:3:image-vip", ConfigurationVersion: "vip-r1", PriceVersion: "vip-r1", Points: 1}}}}
	resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "db:2:3:image-vip", Config: &openai.ClientConfig{APIKey: "fixture", APIStyle: "grsai", Model: "gpt-image-2-vip", BaseURL: endpoint, Timeout: time.Second}}}
	f := generationProviderFactory{resolver: resolver, current: policy}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{UserID: "member", TenantID: "org-a", EffectiveOrganizationID: "org-a"})
	choice := &imageagent.ImageRouteSelection{BindingID: "image-vip", RevisionID: "vip-r1"}
	_, m, e := f.resolve(ctx, choice)
	require.NoError(t, e)
	require.Equal(t, "gpt-image-2-vip", m.Model)
	require.Equal(t, "vip-r1", m.PriceVersion)
	input := imageagent.SlotExecutionInput{ImageSet: &imageagent.ImageSetPlan{Route: choice}}
	require.NoError(t, f.revalidate(ctx, input, m))
	policy.option.ID = "vip-r2"
	require.Error(t, f.revalidate(ctx, input, m), "option replacement must reject the old version before dispatch")
	policy.option.ID = "vip-r1"
	resolver.resolved.CacheKey = "rotated"
	require.Error(t, f.revalidate(ctx, input, m), "credential drift cannot fall back to default")
	_, _, e = f.resolve(ctx)
	require.Error(t, e, "default consumers retain their own gate")
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
			require.ErrorIs(t, f.revalidate(ctx, imageagent.SlotExecutionInput{}, first), imageagent.ErrRevisionConflict)
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
