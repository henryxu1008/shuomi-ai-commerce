package httpapi

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	routestore "task-processor/internal/integration/persistence/routeconfig"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Explicit synthetic shared reader for original isolated consumer regressions.
// Production has no static per-operation fallback.
type syntheticCurrentTextPolicies struct {
	settings *ProductAgentDependencies
	planning map[string]governed.RoutePolicy
}

func (f syntheticCurrentTextPolicies) ReadPolicy(_ context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	var policies map[string]governed.RoutePolicy
	switch key.Operation {
	case aicapability.OperationAIWorkbenchChatPlan:
		policies = f.planning
	case aicapability.OperationProductAgentDecision:
		policies = f.settings.TextPolicies
	case aicapability.OperationProductContentDecision:
		policies = f.settings.ContentTextPolicies
	case aicapability.OperationListingMaterialsDecision:
		policies = f.settings.MaterialsTextPolicies
	case aicapability.OperationProductSelectionDecision:
		policies = f.settings.SelectionTextPolicies
	case aicapability.OperationShopDiagnosisDecision:
		policies = f.settings.DiagnosisTextPolicies
	case aicapability.OperationProductImageAnalysis:
		policies = f.settings.ImageAnalysisPolicies
	}
	p := policies[key.OrganizationID]
	p.Profile.ConnectionID = p.Profile.ClientName
	p.Profile.CredentialVersion = p.AdmittedCredentialVersion
	return p, nil
}

// Only the saved shared-category projection is synthetic. These two business
// regressions use #1078's real SharedTextReader and the original execution and
// resource owners, without a static serving fallback or real configuration.
type syntheticSelectionSharedCategory struct{ settings *ProductAgentDependencies }

// The same native configuration, price and qualification owners used by the
// application. Only database contents/credentials and the provider are synthetic.
func configuredSelectionSharedReader(t *testing.T, settings *ProductAgentDependencies, db *gorm.DB) qualified.SharedTextReader {
	t.Helper()
	ctx, now := context.Background(), time.Now().UTC()
	connection, err := newHTTPChannelFixture(db).owner.Metadata(ctx, "B", "default")
	require.NoError(t, err)
	require.NoError(t, routestore.InstallSchema(ctx, db))
	repo, err := routestore.New(db)
	require.NoError(t, err)
	prices := &d.PlatformPricesService{Repository: repo}
	_, err = prices.Publish(ctx, "offline-platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "offline-reference", Provider: "deepseek", Model: "deepseek-v4.1-flash", Channel: connection.Channel, Region: connection.Region, ServiceType: "standard-sync", Currency: "CNY", UsageVersion: aicapability.ReferenceTextTotalsV1, Rates: aicapability.OfficialModelRates{InputText: 2_000_000, OutputText: 8_000_000}, SourceURL: "https://api-docs.deepseek.com/", VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}})
	require.NoError(t, err)
	defs := []qualified.Definition{}
	for op, policies := range map[aicapability.Operation]map[string]governed.RoutePolicy{aicapability.OperationProductResearchDecision: settings.ResearchTextPolicies, aicapability.OperationProductSelectionDecision: settings.SelectionTextPolicies} {
		if policy, ok := policies["B"]; ok {
			copy := policy
			defs = append(defs, qualified.Definition{ConnectionID: connection.ConnectionID, ConnectionProvider: connection.Provider, Manufacturer: "deepseek", Region: connection.Region, ServiceType: "standard-sync", Channel: connection.Channel, PriceID: "offline-reference", ReferenceVersion: "offline-shared-v1", Scope: d.Scope{OrganizationID: "B", Capability: string(op)}, Text: &copy, MaximumCostMicros: 150000, MaximumMicropoints: 10_000_000})
		}
	}
	catalog, err := qualified.NewCatalog(defs)
	require.NoError(t, err)
	shared := &qualified.SharedCatalog{Catalog: catalog, Credentials: repo, Prices: prices}
	service := &d.SharedModelService{Repository: repo, Catalog: shared}
	selection := d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: connection.Channel, Text: d.ModelBindingReference{ID: "default", Version: "offline-shared-v1"}, Image: d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"}, Multimodal: d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}}
	_, err = service.Publish(ctx, "B", "admin", uuid.NewString(), d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: selection})
	require.NoError(t, err)
	q, _, err := service.ReadCategory(ctx, "B", d.ModelCategoryText)
	require.NoError(t, err)
	require.True(t, q.Ready)
	require.True(t, q.Price.IsReferenceEstimate())
	_, _, err = service.ReadCategory(ctx, "B", d.ModelCategoryImage)
	require.ErrorIs(t, err, d.ErrNeedsConfiguration, "saving choices never makes unopened image generation executable")
	return qualified.SharedTextReader{Service: service}
}

// The Chat and content consumers use the same real current selection, channel
// metadata, published fixture tariff and SharedTextReader; the provider is local.
func configuredChatSharedReader(t *testing.T, settings *ProductAgentDependencies, db *gorm.DB, planner governed.RoutePolicy) qualified.SharedTextReader {
	t.Helper()
	ctx, now := context.Background(), time.Now().UTC()
	connection, err := newHTTPChannelFixture(db).owner.Metadata(ctx, "B", "default")
	require.NoError(t, err)
	require.NoError(t, routestore.InstallSchema(ctx, db))
	repo, err := routestore.New(db)
	require.NoError(t, err)
	prices := &d.PlatformPricesService{Repository: repo}
	_, err = prices.Publish(ctx, "offline-platform", uuid.NewString(), d.PlatformPricesInput{ExpectedRevision: d.NoRevision, Prices: []d.PublishedModelPrice{{ID: "chat-fixture-price", Provider: "openai", Model: planner.Profile.ModelID, Channel: connection.Channel, Region: connection.Region, ServiceType: "standard-sync", Currency: "CNY", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now.Add(-time.Minute), EffectiveUntil: now.Add(time.Hour)}}})
	require.NoError(t, err)
	var defs []qualified.Definition
	for op, policy := range map[aicapability.Operation]governed.RoutePolicy{aicapability.OperationAIWorkbenchChatPlan: planner, aicapability.OperationProductContentDecision: settings.ContentTextPolicies["B"]} {
		copy := policy
		defs = append(defs, qualified.Definition{ConnectionID: connection.ConnectionID, ConnectionProvider: connection.Provider, Manufacturer: "openai", Region: connection.Region, ServiceType: "standard-sync", Channel: connection.Channel, PriceID: "chat-fixture-price", ReferenceVersion: "chat-shared-v1", BoundEvidenceVersion: "fixture-bound", Scope: d.Scope{OrganizationID: "B", Capability: string(op)}, Text: &copy, MaximumCostMicros: settings.Limits.CostMicros})
	}
	catalog, err := qualified.NewCatalog(defs)
	require.NoError(t, err)
	shared := &qualified.SharedCatalog{Catalog: catalog, Credentials: repo, Prices: prices}
	service := &d.SharedModelService{Repository: repo, Catalog: shared}
	selection := d.SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: connection.Channel, Text: d.ModelBindingReference{ID: "default", Version: "chat-shared-v1"}, Image: d.ModelBindingReference{ID: "openai-flare", Version: "approved-default-v1"}, Multimodal: d.ModelBindingReference{ID: "openai-mini", Version: "approved-default-v1"}}
	_, err = service.Publish(ctx, "B", "admin", uuid.NewString(), d.SharedModelInput{ExpectedRevision: d.NoRevision, Selection: selection})
	require.NoError(t, err)
	q, _, err := service.ReadCategory(ctx, "B", d.ModelCategoryText)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.Equal(t, q.TextProfiles[string(aicapability.OperationAIWorkbenchChatPlan)].OfficialTariff, q.TextProfiles[string(aicapability.OperationProductContentDecision)].OfficialTariff)
	_, _, err = service.ReadCategory(ctx, "A", d.ModelCategoryText)
	require.Error(t, err)
	reader := qualified.SharedTextReader{Service: service}
	plan, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationAIWorkbenchChatPlan})
	require.NoError(t, err)
	content, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationProductContentDecision})
	require.NoError(t, err)
	require.Equal(t, plan.Profile.ConnectionID, content.Profile.ConnectionID)
	require.Equal(t, plan.Profile.RoutingPolicyVersion, content.Profile.RoutingPolicyVersion)
	require.Equal(t, plan.Profile.OfficialTariff, content.Profile.OfficialTariff)
	shared.Now = func() time.Time { return now.Add(2 * time.Hour) }
	expired, err := reader.ReadPolicy(ctx, governed.RouteKey{OrganizationID: "B", Operation: aicapability.OperationAIWorkbenchChatPlan})
	require.True(t, err != nil || expired.Profile.Validate() != nil, "expired published price cannot authorize Chat")
	shared.Now = nil
	return reader
}

func (f syntheticSelectionSharedCategory) ReadCategory(_ context.Context, org string, category d.ModelCategory) (d.ModelQualification, string, error) {
	if category != d.ModelCategoryText {
		return d.ModelQualification{}, "", d.ErrNeedsConfiguration
	}
	q := d.ModelQualification{OrganizationID: org, Category: category, Ready: true, TextProfiles: map[string]aicapability.ModelProfile{}}
	for op, policies := range map[aicapability.Operation]map[string]governed.RoutePolicy{
		aicapability.OperationProductResearchDecision:  f.settings.ResearchTextPolicies,
		aicapability.OperationProductSelectionDecision: f.settings.SelectionTextPolicies,
	} {
		if policy, ok := policies[org]; ok {
			p := policy.ShapeProfile()
			q.TextProfiles[string(op)] = p
			q.Text, q.Model, q.Price = &p, p.ModelID, p.OfficialTariff
			q.Reference = d.ModelBindingReference{ID: p.ClientName, Version: "synthetic-shared-v1"}
		}
	}
	if q.Text == nil {
		return d.ModelQualification{}, "", d.ErrNotFound
	}
	return q, "route-v1", nil
}

func assertSelectionMicropointAccounting(t *testing.T, db *gorm.DB, operation aicapability.Operation, zero bool) {
	t.Helper()
	var rows []struct {
		Operation, OfficialTariffJSON, BillableUsageJSON                  string
		MaximumPromptTokens, MaximumCompletionTokens, EstimatedCostMicros int64
		UsageKnown                                                        bool
	}
	require.NoError(t, db.Table("ai_invocations").Where("tenant_id = ?", "B").Find(&rows).Error)
	require.Len(t, rows, 2, "recovery and history must not create more invocations")
	var totalActual, totalReserved, totalCost int64
	for _, row := range rows {
		require.Equal(t, string(operation), row.Operation)
		require.True(t, row.UsageKnown)
		require.Positive(t, row.MaximumPromptTokens)
		require.LessOrEqual(t, row.MaximumPromptTokens, int64(32000))
		require.LessOrEqual(t, row.MaximumCompletionTokens, int64(4000))
		var tariff aicapability.OfficialModelTariff
		var usage aicapability.BillableModelUsage
		require.NoError(t, json.Unmarshal([]byte(row.OfficialTariffJSON), &tariff))
		require.NoError(t, json.Unmarshal([]byte(row.BillableUsageJSON), &usage))
		require.Equal(t, aicapability.AIPointMicrounitV1, tariff.UnitVersion)
		require.True(t, tariff.IsReferenceEstimate())
		require.Equal(t, tariff.UsageVersion, usage.MappingVersion)
		require.Contains(t, tariff.PriceVersion, ":", "the original published reference version is frozen")
		reserved, err := tariff.ReserveTextMicropoints(row.MaximumPromptTokens, row.MaximumCompletionTokens)
		require.NoError(t, err)
		actual, err := tariff.SettlementMicropoints(usage)
		require.NoError(t, err)
		totalReserved, totalActual, totalCost = totalReserved+reserved, totalActual+actual, totalCost+row.EstimatedCostMicros
	}
	// The original bounded executor may tighten each serialized prompt below
	// the admitted 32k maximum; do not require an unnecessarily large hold.
	require.Positive(t, totalReserved)
	require.LessOrEqual(t, totalReserved, int64(19200000))
	require.LessOrEqual(t, totalReserved, int64(20000000))
	expectedActual, expectedCost := int64(24000), int64(240)
	if zero {
		expectedActual, expectedCost = 0, 0
	}
	require.EqualValues(t, expectedActual, totalActual, "charge explicit fixture usage, not the reserved upper bound")
	require.EqualValues(t, expectedCost, totalCost)
	require.LessOrEqual(t, totalCost, int64(300000))
	var bucket struct {
		QuantityUnit                  string
		Available, Reserved, Consumed int64
	}
	require.NoError(t, db.Table("saas_organization_resource_buckets").Where("organization_id = ? AND resource_type = ?", "B", "ai_point").First(&bucket).Error)
	require.Equal(t, aicapability.AIPointMicrounitV1, bucket.QuantityUnit)
	require.Equal(t, totalActual, bucket.Consumed)
	require.Zero(t, bucket.Reserved)
	require.EqualValues(t, 10000000-totalActual, bucket.Available)
}
