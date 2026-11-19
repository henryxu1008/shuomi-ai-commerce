package store

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestModelPointSchemaAndScopedCanonicalFact(t *testing.T) {
	db := newInvocationLedgerDB(t)
	require.NoError(t, VerifyModelPointInvocationSchema(context.Background(), db))
	r := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "canonical", TenantID: "org", UserID: "user", MemberID: "member", AgentRunID: "run", InputHash: "input", Operation: aicapability.OperationProductAgentDecision, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY"}
	_, err := r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	got, err := r.ReadModelInvocation(context.Background(), "org", "canonical")
	require.NoError(t, err)
	require.Equal(t, fact.OfficialTariff, got.OfficialTariff)
	_, err = r.ReadModelInvocation(context.Background(), "other", "canonical")
	require.Error(t, err)
	// Synthetic obsolete schema is rejected without changing it.
	require.NoError(t, db.Exec("ALTER TABLE ai_invocations DROP COLUMN official_tariff_json").Error)
	require.Error(t, VerifyModelPointInvocationSchema(context.Background(), db))
	require.False(t, db.Migrator().HasColumn(&invocationRow{}, "official_tariff_json"))
}

func TestContentPricedInvocationKeepsCanonicalIdentityAndReplay(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "content-invocation", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "content-run", InputHash: "frozen-content-selection", Operation: aicapability.OperationProductContentDecision, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY"}
	acquired, err := ledger.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.True(t, acquired)
	acquired, err = ledger.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.False(t, acquired)
	read, err := ledger.ReadModelInvocation(context.Background(), fact.TenantID, fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, fact.Operation, read.Operation)
	require.Equal(t, fact.AgentRunID, read.AgentRunID)
	foreign := fact
	foreign.Operation = aicapability.OperationProductAgentDecision
	_, err = ledger.ClaimInvocation(context.Background(), foreign)
	require.Error(t, err, "a title dispatch cannot reuse the content ledger identity")
	_, err = ledger.ReadModelInvocation(context.Background(), "foreign", fact.InvocationID)
	require.Error(t, err)
}

func TestAdvisoryAgentsPricedInvocationRetainsExistingScopedLedger(t *testing.T) {
	for _, op := range []aicapability.Operation{aicapability.OperationListingMaterialsDecision, aicapability.OperationProductResearchDecision, aicapability.OperationProductSelectionDecision} {
		t.Run(string(op), func(t *testing.T) {
			db := newInvocationLedgerDB(t)
			ledger := NewGormInvocationRecorder(db)
			fact := aicapability.InvocationRecord{InvocationID: "materials-invocation", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "materials-run", InputHash: "exact-material-disclosure", Operation: op, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY"}
			acquired, err := ledger.ClaimInvocation(context.Background(), fact)
			require.NoError(t, err)
			require.True(t, acquired)
			acquired, err = ledger.ClaimInvocation(context.Background(), fact)
			require.NoError(t, err)
			require.False(t, acquired)
			read, err := ledger.ReadModelInvocation(context.Background(), "org", fact.InvocationID)
			require.NoError(t, err)
			require.Equal(t, fact.AgentRunID, read.AgentRunID)
			require.Equal(t, fact.OfficialTariff, read.OfficialTariff)
			foreign := fact
			foreign.Operation = aicapability.OperationProductContentDecision
			_, err = ledger.ClaimInvocation(context.Background(), foreign)
			require.Error(t, err)
			_, err = ledger.ReadModelInvocation(context.Background(), "other-org", fact.InvocationID)
			require.Error(t, err)

		})
	}
}

func TestSingleImagePricedInvocationUsesOriginalScopedLedgerAndUsage(t *testing.T) {
	db := newInvocationLedgerDB(t)
	l := NewGormInvocationRecorder(db)
	ctx := context.Background()
	fact := aicapability.InvocationRecord{InvocationID: "single-image", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "frozen-image-and-quote", ModelID: aicapability.SingleImageMiniModel, Capability: aicapability.CapabilityProductImageAnalysis, Operation: aicapability.OperationProductImageAnalysis, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 5000, MaximumCompletionTokens: 64,
		OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: aicapability.SingleImageMiniModel, Channel: "openai-direct", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, CachedInputText: 1000, OutputText: 20000}}}
	won, err := l.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	require.True(t, won)
	read, err := l.ReadModelInvocation(ctx, "org", fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, fact.OfficialTariff, read.OfficialTariff)
	terminal := fact
	terminal.Outcome, terminal.UsageKnown, terminal.FinishedAt = aicapability.InvocationUsageObservedFailed, true, time.Now().UTC()
	terminal.PromptTokens, terminal.CompletionTokens, terminal.TotalTokens = 30, 5, 35
	terminal.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 30, CachedInputText: 10, OutputText: 5}
	require.NoError(t, l.RecordInvocation(ctx, terminal))
	read, err = l.ReadModelInvocation(ctx, "org", fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, terminal.BillableUsage, read.BillableUsage)
	won, err = l.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	require.False(t, won)
	_, err = l.ReadModelInvocation(ctx, "other-org", fact.InvocationID)
	require.Error(t, err)
	changed := fact
	changed.Operation = aicapability.OperationProductAgentDecision
	_, err = l.ClaimInvocation(ctx, changed)
	require.Error(t, err, "a text purpose cannot reuse the visual dispatch fact")
}

func TestPlannerPricedInvocationUsesSameScopedOwnerWithoutFakeAgentRun(t *testing.T) {
	db := newInvocationLedgerDB(t)
	r := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{
		InvocationID: "chat-plan-invocation", TenantID: "org", UserID: "actor", MemberID: "member",
		InputHash: "frozen-profile-and-message", Operation: aicapability.OperationAIWorkbenchChatPlan,
		Capability: aicapability.CapabilityAIWorkbenchChatPlanning,
		Outcome:    aicapability.InvocationDispatched, StartedAt: time.Now().UTC(),
		MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY",
	}
	acquired, err := r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.True(t, acquired)
	acquired, err = r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.False(t, acquired)
	got, err := r.ReadModelInvocation(context.Background(), "org", fact.InvocationID)
	require.NoError(t, err)
	require.Empty(t, got.AgentRunID)
	require.Empty(t, got.BusinessTaskID)
	require.Equal(t, fact.OfficialTariff, got.OfficialTariff)
	ctx := context.Background()
	add := func(id string, change func(*aicapability.InvocationRecord)) {
		row := fact
		row.InvocationID = id
		change(&row)
		claimed, err := r.ClaimInvocation(ctx, row)
		require.NoError(t, err)
		require.True(t, claimed)
	}
	add("foreign-user", func(row *aicapability.InvocationRecord) { row.UserID = "other" })
	add("foreign-member", func(row *aicapability.InvocationRecord) { row.MemberID = "other" })
	add("foreign-org", func(row *aicapability.InvocationRecord) { row.TenantID = "other" })
	add("route-call", func(row *aicapability.InvocationRecord) { row.Operation = aicapability.OperationAIWorkbenchStoreRoute })
	imageQuestion := func(row *aicapability.InvocationRecord) {
		row.Operation = aicapability.OperationAIWorkbenchChatImages
		row.ModelID = aicapability.SingleImageMiniModel
		row.OfficialTariff.Model = row.ModelID
		row.OfficialTariff.Channel = "openai-direct"
	}
	add("image-question", imageQuestion)
	add("foreign-image-question", func(row *aicapability.InvocationRecord) {
		imageQuestion(row)
		row.MemberID = "other"
	})
	images, err := r.ReadModelInvocation(ctx, "org", "image-question")
	require.NoError(t, err)
	add("bad-call", func(*aicapability.InvocationRecord) {})
	require.NoError(t, db.Model(&invocationRow{}).Where("invocation_id = ?", "bad-call").Update("maximum_prompt_tokens", 0).Error)
	facts, err := r.ReadModelPurposeInvocations(ctx, "org", "actor", "member", []string{
		fact.InvocationID, "image-question", "foreign-image-question", "foreign-user", "foreign-member", "foreign-org", "route-call", "bad-call", "missing-call",
	})
	require.NoError(t, err)
	require.ElementsMatch(t, []aicapability.InvocationRecord{got, images}, facts)
	_, err = r.ReadModelPurposeInvocations(ctx, "org", "actor", "member", make([]string, 51))
	require.Error(t, err)
	foreign := fact
	foreign.AgentRunID = "invented-run"
	_, err = r.ClaimInvocation(context.Background(), foreign)
	require.Error(t, err)
	foreign = fact
	foreign.BusinessTaskID = "invented-task"
	_, err = r.ClaimInvocation(context.Background(), foreign)
	require.Error(t, err)
}

func TestStoreRoutePricedInvocationHasItsOwnPurpose(t *testing.T) {
	db := newInvocationLedgerDB(t)
	r := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{
		InvocationID: "route-invocation", TenantID: "org", UserID: "actor", MemberID: "member",
		InputHash: "frozen-question", Operation: aicapability.OperationAIWorkbenchStoreRoute,
		Capability: aicapability.CapabilityAIWorkbenchChatPlanning, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(),
		MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY",
	}
	acquired, err := r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.True(t, acquired)
	acquired, err = r.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.False(t, acquired)
	read, err := r.ReadModelInvocation(context.Background(), "org", fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, aicapability.OperationAIWorkbenchStoreRoute, read.Operation)
	fact.Operation = aicapability.OperationAIWorkbenchChatPlan
	_, err = r.ClaimInvocation(context.Background(), fact)
	require.Error(t, err, "a different purpose must not reuse the same invocation")
}

func TestInvocationExistenceChecksGlobalIdentityWithoutHidingConflicts(t *testing.T) {
	r := NewGormInvocationRecorder(newInvocationLedgerDB(t))
	ctx := context.Background()
	fact := aicapability.InvocationRecord{InvocationID: "global-id", TenantID: "other-org", UserID: "actor", MemberID: "member", InputHash: "input", Operation: aicapability.OperationAIWorkbenchStoreRoute, Capability: aicapability.CapabilityAIWorkbenchChatPlanning, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 100, MaximumCompletionTokens: 50, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}, ModelID: "fixture-model", Currency: "CNY"}
	_, err := r.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	found, err := r.HasInvocationID(ctx, "global-id")
	require.NoError(t, err)
	require.True(t, found)
	found, err = r.HasInvocationID(ctx, "not-present")
	require.NoError(t, err)
	require.False(t, found)
}
