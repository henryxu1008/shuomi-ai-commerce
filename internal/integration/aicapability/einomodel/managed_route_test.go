package einomodel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"task-processor/internal/aicapability"
	"task-processor/internal/integration/openai"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type managedFixtureTransport func(*http.Request) (*http.Response, error)

func (f managedFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestManagedPublicationAtFinalGateAndDuringSendPreservesFrozenEconomics(t *testing.T) {
	for _, stage := range []string{"final-gate", "in-flight", "unknown"} {
		t.Run(stage, func(t *testing.T) {
			db, e := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
			require.NoError(t, e)
			require.NoError(t, db.AutoMigrate(&openai.AIClientCredential{}))
			ctx := context.Background()
			writer := openai.NewGormCredentialResolver(db)
			require.NoError(t, writer.SaveCredential(ctx, openai.AIClientCredential{TenantID: "org", ClientName: "org-planning", APIKey: "fixture", BaseURL: "https://provider.example.test/v1", Model: "synthetic-model", APIStyle: "openai-compatible", TimeoutSecond: 5, Enabled: true}))
			row, e := writer.GetCredential(ctx, "org", "", "org-planning")
			require.NoError(t, e)
			p := testTextProfile(row.BaseURL)
			p.CredentialVersion = CredentialVersion(*row)
			current := &changingPolicies{policy: RoutePolicy{Profile: p, AdmittedCredentialVersion: p.CredentialVersion, AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}}
			resolver, e := NewManagedOrganizationRouteResolver(writer, current)
			require.NoError(t, e)
			publish := func() {
				current.policy.Profile.RoutingPolicyVersion = "r2"
				current.policy.Profile.CostPricingVersion = "r2"
				current.policy.Profile.PointTariff.PriceVersion = "r2"
				current.policy.Profile.InputMicrosPerMillion = 3000000
				current.policy.Profile.PointTariff.InputPointsPerMillionTokens = 3000000
			}
			ledger := &recordingLedger{}
			sends, checks := 0, 0
			executor := Executor{Ledger: ledger, Admission: NewBoundedAdmission(), Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }, Resolve: func(ctx context.Context, in aicapability.TextInputIdentity) (QualifiedRoute, error) {
				checks++
				if stage == "final-gate" && checks == 2 {
					publish()
				}
				return resolver.Resolve(ctx, in)
			}, BaseTransport: managedFixtureTransport(func(*http.Request) (*http.Response, error) {
				sends++
				publish()
				if stage == "unknown" {
					return nil, errors.New("fixture response lost")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"fixture","object":"chat.completion","created":1,"model":"synthetic-model","choices":[{"index":0,"message":{"role":"assistant","content":"READY"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`))}, nil
			})}
			input := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", InvocationID: "one", Operation: aicapability.OperationAIWorkbenchChatPlan, System: "system", Prompt: "prompt", Profile: p}
			quote, e := aicapability.QuoteText(input)
			require.NoError(t, e)
			_, e = executor.Generate(ctx, input, quote, nil)
			switch stage {
			case "final-gate":
				require.ErrorIs(t, e, ErrNotDispatched)
				require.Zero(t, sends)
			case "unknown":
				require.ErrorIs(t, e, ErrOutcomeUnknown)
				require.Len(t, ledger.records, 1)
				require.True(t, ledger.reserved)
			case "in-flight":
				require.NoError(t, e)
				last := ledger.records[len(ledger.records)-1]
				require.Equal(t, p.PointTariff, last.PointTariff)
				require.Equal(t, p.CostPricingVersion, last.PolicyVersion)
				cost, _ := p.CostFor(10, 5)
				require.Equal(t, cost, last.EstimatedCostMicros)
			}
			oldRecords := append([]aicapability.InvocationRecord(nil), ledger.records...)
			input.InvocationID = "next"
			_, e = executor.Generate(ctx, input, quote, nil)
			require.ErrorIs(t, e, ErrNotDispatched)
			require.Equal(t, oldRecords, ledger.records, "next call cannot rewrite historical/UNKNOWN facts")
		})
	}
}

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
