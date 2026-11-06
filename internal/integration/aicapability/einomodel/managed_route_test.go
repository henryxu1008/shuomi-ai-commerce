package einomodel

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"task-processor/internal/aicapability"
	"testing"

	"github.com/stretchr/testify/require"
)

type managedFixtureTransport func(*http.Request) (*http.Response, error)

func (f managedFixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestManagedPublicationAtFinalGateAndDuringSendPreservesFrozenEconomics(t *testing.T) {
	for _, stage := range []string{"final-gate", "in-flight", "unknown"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			writer := testChannelReader("org", "shared-channel", "https://provider.example.test/v1")
			p := fixtureChannelPolicy(testTextProfile(writer.secret.Metadata.BaseURL), writer).Profile
			current := &changingPolicies{policy: fixtureChannelPolicy(p, writer)}
			resolver, e := NewManagedOrganizationRouteResolver(writer, current)
			require.NoError(t, e)
			publish := func() {
				current.policy.Profile.RoutingPolicyVersion = "r2"
				current.policy.Profile.CostPricingVersion = "r2"
				current.policy.Profile.OfficialTariff.PriceVersion = "r2"
				current.policy.Profile.InputMicrosPerMillion = 3000000
				current.policy.Profile.OfficialTariff.Rates.InputText = 3000000
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
				require.Equal(t, p.OfficialTariff, last.OfficialTariff)
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

func TestChannelRotationAtFinalGateInFlightAndUnknownDoesNotRepriceOrResend(t *testing.T) {
	for _, stage := range []string{"final-gate", "in-flight", "unknown"} {
		t.Run(stage, func(t *testing.T) {
			ctx := context.Background()
			keys := testChannelReader("org", "shared-channel", "https://provider.example.test/v1")
			original := fixtureChannelPolicy(testTextProfile(keys.secret.Metadata.BaseURL), keys)
			current := &changingPolicies{policy: original}
			resolver, e := NewManagedOrganizationRouteResolver(keys, current)
			require.NoError(t, e)
			rotate := func() {
				keys.secret.Metadata.Revision++
				keys.secret.Metadata = keys.secret.Metadata.WithIdentity()
				keys.secret.APIKey = "rotated-synthetic-key"
				current.policy = fixtureChannelPolicy(original.Profile, keys)
			}
			ledger := &recordingLedger{}
			sends, checks := 0, 0
			executor := Executor{Ledger: ledger, Admission: NewBoundedAdmission(), Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }, Resolve: func(ctx context.Context, in aicapability.TextInputIdentity) (QualifiedRoute, error) {
				checks++
				if stage == "final-gate" && checks == 2 {
					rotate()
				}
				return resolver.Resolve(ctx, in)
			}, BaseTransport: managedFixtureTransport(func(r *http.Request) (*http.Response, error) {
				sends++
				require.Equal(t, "Bearer synthetic-organization-key", r.Header.Get("Authorization"))
				rotate()
				if stage == "unknown" {
					return nil, errors.New("synthetic lost response")
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"id":"one","object":"chat.completion","created":1,"model":"synthetic-model","choices":[{"index":0,"message":{"role":"assistant","content":"READY"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`))}, nil
			})}
			input := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", InvocationID: "one", Operation: aicapability.OperationAIWorkbenchChatPlan, System: "system", Prompt: "prompt", Profile: original.Profile}
			q, e := aicapability.QuoteText(input)
			require.NoError(t, e)
			_, e = executor.Generate(ctx, input, q, nil)
			if stage == "final-gate" {
				require.ErrorIs(t, e, ErrNotDispatched)
				require.Zero(t, sends)
			} else if stage == "unknown" {
				require.ErrorIs(t, e, ErrOutcomeUnknown)
				require.True(t, ledger.reserved)
				require.Equal(t, 1, sends)
			} else {
				require.NoError(t, e)
				require.Equal(t, 1, sends)
				last := ledger.records[len(ledger.records)-1]
				require.Equal(t, original.Profile.CredentialVersion, last.CredentialReference)
				require.Equal(t, original.Profile.OfficialTariff, last.OfficialTariff)
			}
			history := append([]aicapability.InvocationRecord(nil), ledger.records...)
			_, e = executor.Generate(ctx, input, q, nil)
			require.Error(t, e)
			require.Equal(t, history, ledger.records)
			if stage != "final-gate" {
				require.Equal(t, 1, sends)
			}
		})
	}
}

func (p *changingPolicies) ReadPolicy(context.Context, RouteKey) (RoutePolicy, error) {
	if p.missing {
		return RoutePolicy{}, ErrNotDispatched
	}
	return p.policy, nil
}
func TestManagedResolverReadsFreshPolicyAndNeverFallsBack(t *testing.T) {
	writer := testChannelReader("org", "shared-channel", "https://example.org/v1")
	ctx := context.Background()
	p := fixtureChannelPolicy(testTextProfile(writer.secret.Metadata.BaseURL), writer).Profile
	current := &changingPolicies{policy: fixtureChannelPolicy(p, writer)}
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
