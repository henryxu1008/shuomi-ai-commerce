package httpapi

import (
	"context"
	"encoding/json"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Explicit synthetic shared reader for original isolated consumer regressions.
// Production has no static per-operation fallback.
type syntheticCurrentTextPolicies struct{ settings *ProductAgentDependencies }

func (f syntheticCurrentTextPolicies) ReadPolicy(_ context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	var policies map[string]governed.RoutePolicy
	switch key.Operation {
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
	}
	return policies[key.OrganizationID], nil
}

// Only the saved shared-category projection is synthetic. These two business
// regressions use #1078's real SharedTextReader and the original execution and
// resource owners, without a static serving fallback or real configuration.
type syntheticSelectionSharedCategory struct{ settings *ProductAgentDependencies }

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

func assertSelectionMicropointAccounting(t *testing.T, db *gorm.DB, operation aicapability.Operation) {
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
	require.EqualValues(t, 24000, totalActual, "charge complete fixture usage, not the reserved upper bound")
	require.EqualValues(t, 240, totalCost)
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
