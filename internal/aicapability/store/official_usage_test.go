package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestOfficialUsageMetadataSurvivesCanonicalLedgerAndCannotChangeOnReplay(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	ctx := context.Background()
	fact := aicapability.InvocationRecord{InvocationID: "official-metered", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched, Operation: aicapability.OperationProductAgentDecision}
	acquired, err := ledger.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	require.True(t, acquired)
	terminal := fact
	terminal.Outcome, terminal.UsageKnown, terminal.FinishedAt = aicapability.InvocationSucceeded, true, time.Now().UTC()
	terminal.PromptTokens, terminal.CompletionTokens, terminal.TotalTokens = 5, 2, 7
	terminal.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 5, CachedInputText: 2, OutputText: 2}
	require.NoError(t, ledger.RecordInvocation(ctx, terminal))
	got, found, err := ledger.FindInvocation(ctx, "org", "member", fact.InvocationID, "input")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, terminal.BillableUsage, got.BillableUsage)
	require.NoError(t, ledger.RecordInvocation(ctx, terminal))
	changedUsage := *terminal.BillableUsage
	changedUsage.CachedInputText = 1
	terminal.BillableUsage = &changedUsage
	require.Error(t, ledger.RecordInvocation(ctx, terminal), "terminal usage is immutable, including cache partitions")

	require.NoError(t, db.Exec("ALTER TABLE ai_invocations DROP COLUMN billable_usage_json").Error)
	require.Error(t, VerifyModelPointInvocationSchema(ctx, db), "startup must reject the missing field without changing the database")
	require.False(t, db.Migrator().HasColumn(&invocationRow{}, "billable_usage_json"))
}

func TestOfficialUsageMetadataIsNotAClaimOrNoSendProof(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "premature-usage", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched, Operation: aicapability.OperationProductAgentDecision,
		BillableUsage: &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true}}
	_, err := ledger.ClaimInvocation(context.Background(), fact)
	require.Error(t, err)
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, time.Now().UTC()
	fact.PromptTokens, fact.TotalTokens = 5, 5
	require.Error(t, ledger.RecordInvocation(context.Background(), fact), "a complete-looking but inconsistent partition is rejected")
}

func TestOfficialPriceCannotBeClaimedForAnotherModel(t *testing.T) {
	db := newInvocationLedgerDB(t)
	ledger := NewGormInvocationRecorder(db)
	fact := aicapability.InvocationRecord{InvocationID: "wrong-official-model", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", ModelID: "model", Operation: aicapability.OperationProductAgentDecision, Outcome: aicapability.InvocationDispatched, StartedAt: time.Now().UTC(), MaximumPromptTokens: 10, MaximumCompletionTokens: 10,
		OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialTokenRates{InputText: 10000, OutputText: 20000}}}
	fact.OfficialTariff.Model = "another-model"
	_, err := ledger.ClaimInvocation(context.Background(), fact)
	require.Error(t, err)
	var count int64
	require.NoError(t, db.Model(&invocationRow{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestInvocationInstallerRefusesOldOrDualPriceSchemaWithoutRepair(t *testing.T) {
	for _, kind := range []string{"missing-official-price", "independent-price"} {
		t.Run(kind, func(t *testing.T) {
			db := newInvocationLedgerDB(t)
			if kind == "missing-official-price" {
				require.NoError(t, db.Exec("ALTER TABLE ai_invocations DROP COLUMN official_tariff_json").Error)
			} else {
				require.NoError(t, db.Exec("ALTER TABLE ai_invocations ADD COLUMN input_points_per_million_tokens bigint NOT NULL DEFAULT 0").Error)
			}
			require.Error(t, AutoMigrateInvocationLedger(db), "installer cannot upgrade an existing ledger or preserve a second pricing column")
			require.Error(t, VerifyModelPointInvocationSchema(context.Background(), db))
			if kind == "missing-official-price" {
				require.False(t, db.Migrator().HasColumn(&invocationRow{}, "official_tariff_json"))
			} else {
				require.True(t, db.Migrator().HasColumn(&invocationRow{}, "input_points_per_million_tokens"))
			}
		})
	}
}
