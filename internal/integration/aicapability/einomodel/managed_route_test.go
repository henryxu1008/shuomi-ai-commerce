package einomodel

import (
	"context"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	"task-processor/internal/integration/openai"
	"testing"
)

type changingPolicies struct {
	policy  RoutePolicy
	missing bool
}

func (p *changingPolicies) ReadPolicy(context.Context, RouteKey) (RoutePolicy, error) {
	if p.missing {
		return RoutePolicy{}, ErrNotDispatched
	}
	return p.policy, nil
}
func TestManagedResolverReadsFreshPolicyAndNeverFallsBack(t *testing.T) {
	db, e := gorm.Open(sqlite.Open("file:managed-route?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, e)
	require.NoError(t, db.AutoMigrate(&openai.AIClientCredential{}))
	writer := openai.NewGormCredentialResolver(db)
	ctx := context.Background()
	require.NoError(t, writer.SaveCredential(ctx, openai.AIClientCredential{TenantID: "org", ClientName: "default", APIKey: "fixture-only", BaseURL: "https://example.org/v1", Model: "model", APIStyle: "openai-compatible", TimeoutSecond: 5, Enabled: true}))
	row, e := writer.GetCredential(ctx, "org", "", "default")
	require.NoError(t, e)
	p := testTextProfile(row.BaseURL)
	p.ClientName = row.ClientName
	p.ModelID = row.Model
	p.CredentialVersion = CredentialVersion(*row)
	current := &changingPolicies{policy: RoutePolicy{Profile: p, AdmittedCredentialVersion: p.CredentialVersion, AdmittedEndpointIdentityDigest: endpointDigest(row.BaseURL)}}
	r, e := NewManagedOrganizationRouteResolver(writer, current)
	require.NoError(t, e)
	input := aicapability.TextInputIdentity{OrganizationID: "org", Operation: aicapability.OperationProductAgentDecision}
	first, e := r.Resolve(ctx, input)
	require.NoError(t, e)
	current.policy.Profile.RoutingPolicyVersion = "new-publication"
	next, e := r.Resolve(ctx, input)
	require.NoError(t, e)
	require.NotEqual(t, first.Profile, next.Profile)
	current.missing = true
	_, e = r.Resolve(ctx, input)
	require.ErrorIs(t, e, ErrNotDispatched)
}
