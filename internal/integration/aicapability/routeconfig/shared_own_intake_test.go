package routeconfiguration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func TestOwnIntakeMiniSharedBudget(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		fx, usdInput, cnyInput, cnyOutput, pointLimit int64
		reason                                        string
		wantCost, wantPoints                          int64
	}{
		{"approved profile", 8_000_000, 750_000, 6_000_000, 36_000_000, 30_000_000, "", 288_000, 28_800_000},
		{"both ceilings inclusive", 8_000_000, 812_500, 6_500_000, 36_000_000, 30_000_000, "", 300_000, 30_000_000},
		{"FX drift below absolute ceilings", 8_100_000, 750_000, 6_000_000, 36_000_000, 30_000_000, "COST_BOUND_MISSING", 0, 0},
		{"input rate rounds upward", 8_000_001, 750_000, 6_000_000, 36_000_005, 30_000_000, "COST_BOUND_MISSING", 0, 0},
		{"output rate rounds upward", 8_000_001, 750_000, 6_000_001, 36_000_004, 30_000_000, "COST_BOUND_MISSING", 0, 0},
		{"absolute cost despite higher general budget", 8_000_000, 812_500, 6_500_001, 36_000_000, 30_000_000, "COST_BOUND_MISSING", 0, 0},
		{"missing point limit", 8_000_000, 750_000, 6_000_000, 36_000_000, 0, "POINT_BUDGET_MISSING", 0, 0},
		{"point limit above absolute ceiling", 8_000_000, 750_000, 6_000_000, 36_000_000, 30_000_001, "POINT_BUDGET_EXCEEDED", 0, 0},
		{"points exceed narrower limit", 8_000_000, 750_000, 6_000_000, 36_000_000, 28_799_999, "POINT_BUDGET_EXCEEDED", 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 11, 8, 8, 0, 0, 0, time.UTC)
			ref := d.ModelBindingReference{ID: "mini", Version: "fixture"}
			endpoint := governed.EndpointIdentityDigest("https://api.openai.com/v1")
			p := aicapability.ModelProfile{
				ClientName: ref.ID, ProviderID: "openai", AdapterKind: "openai-compatible", ModelID: aicapability.SingleImageMiniModel,
				EndpointIdentityDigest: endpoint, CredentialVersion: "fixture", RoutingPolicyVersion: "fixture",
				AdapterPolicyVersion: aicapability.SingleImageMiniAdapterPolicy, PromptVersion: agentconfig.OwnIntakePromptVersion,
				OutputSchemaVersion: agentconfig.OwnIntakeActionVersion, UsageMappingVersion: "openai-text-v1", CostPricingVersion: "fixture",
				Currency: "CNY", InputMicrosPerMillion: tc.cnyInput, OutputMicrosPerMillion: tc.cnyOutput,
				MaximumPromptTokens: 24_000, MaximumCompletionTokens: 4_000, MaximumInputBytes: 1 << 20, MaximumOutputBytes: 16 << 10, DeadlineBound: time.Second,
			}
			op := aicapability.OperationOwnProductIntake
			catalog, err := NewCatalog([]Definition{{
				Scope: d.Scope{OrganizationID: "org", Capability: string(op)}, Manufacturer: "openai", Region: "global", ServiceType: "standard-sync",
				Channel: "openai-direct", PriceID: "fixture-mini", ReferenceVersion: ref.Version, BoundEvidenceVersion: "fixture-bound",
				Text:              &governed.RoutePolicy{Profile: p, AdmittedCredentialVersion: "fixture", AdmittedEndpointIdentityDigest: endpoint},
				MaximumCostMicros: 1_000_000, MaximumMicropoints: tc.pointLimit,
			}})
			require.NoError(t, err)
			prices, err := d.FreezePlatformPrices(d.PlatformPricesInput{
				Prices: []d.PublishedModelPrice{{ID: "fixture-mini", Provider: "openai", Model: p.ModelID, Channel: "openai-direct", Region: "global",
					ServiceType: "standard-sync", Currency: "USD", UsageVersion: "openai-text-v1",
					Rates:     aicapability.OfficialModelRates{InputText: tc.usdInput, CachedInputText: 75_000, OutputText: 4_500_000},
					SourceURL: "https://developers.openai.com/api/docs/models/gpt-5.4-mini", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}},
				FX: []d.PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: tc.fx, SourceURL: "https://www.chinamoney.com.cn/chinese/bkccpr/",
					RateDate: "2026-11-08", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}},
			}, "fixture-publication", nil, now)
			require.NoError(t, err)
			metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", EndpointDigest: endpoint, Enabled: true, KeyPresent: true}}
			bindTestChannel(catalog, &metadata.credential)
			shared := &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}
			q, err := shared.QualifyModel(ctx, "org", d.ModelCategoryMultimodal, ref)
			require.NoError(t, err)
			require.Equal(t, tc.reason, q.Reason)
			require.Equal(t, tc.reason == "", q.Ready)
			if !q.Ready {
				return
			}

			profile := q.TextProfiles[string(op)]
			input := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", AgentRunID: "run", Operation: op,
				System: "fixture", Prompt: "x", Profile: profile,
				Image: &aicapability.SingleImageInput{SelectionDigest: strings.Repeat("a", 64), ContentSHA256: strings.Repeat("b", 64), MediaType: "image/png", ByteLength: 100, Width: 1, Height: 1},
			}
			initial, err := aicapability.QuoteText(input)
			require.NoError(t, err)
			input.Prompt += strings.Repeat("x", int(28_000-initial.MaximumTokens))
			quote, err := aicapability.QuoteText(input)
			require.NoError(t, err)
			require.EqualValues(t, 28_000, quote.MaximumTokens)
			require.Equal(t, tc.wantCost, quote.MaximumCostMicros)
			points, err := q.Price.ReserveTextMicropoints(24_000, 4_000)
			require.NoError(t, err)
			require.Equal(t, tc.wantPoints, points)
			input.Prompt += "x"
			_, err = aicapability.QuoteText(input)
			require.ErrorIs(t, err, aicapability.ErrTextEnvelope)
		})
	}
}
