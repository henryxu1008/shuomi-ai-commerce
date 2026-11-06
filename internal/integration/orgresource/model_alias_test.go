package orgresourceadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"task-processor/internal/ledger/orgresource"
)

func TestAliasedModelUsesOriginalResourceReservationAndTerminalPaths(t *testing.T) {
	for _, op := range []aicapability.Operation{aicapability.OperationProductAgentDecision, aicapability.OperationProductImageAnalysis} {
		for _, terminal := range []string{"settled", "held-no-send", "no-hold", "unknown", "recover"} {
			t.Run(string(op)+"/"+terminal, func(t *testing.T) {
				db, recorder, repo, _, fact := modelPointFixture(t, op)
				ctx := context.Background()
				var initial organizationResourceBucketRow
				require.NoError(t, db.Take(&initial).Error)
				fact.InvocationID = "aliased-model"
				fact.ModelID, fact.OfficialTariff.Model = "google/fixture-model", "fixture-model"
				fact.OfficialTariff.WireModel = fact.ModelID
				fact.OfficialTariff.UsageVersion = aicapability.ReferenceTextTotalsV1
				fact.OfficialTariff.Rates.CachedInputText = 0
				fact.MaximumPromptTokens, fact.MaximumCompletionTokens = 10, 10
				_, err := recorder.ClaimInvocation(ctx, fact)
				require.NoError(t, err)
				if terminal != "no-hold" {
					require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
					require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
				}
				if terminal == "unknown" {
					require.ErrorIs(t, repo.ReleaseAIInvocationUsage(ctx, "org", fact.InvocationID), orgresource.ErrOwnerNotTerminal)
				} else {
					fact.UsageKnown, fact.FinishedAt = true, fact.StartedAt.Add(time.Second)
					if terminal == "settled" || terminal == "recover" {
						fact.Outcome = aicapability.InvocationSucceeded
						fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
						fact.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceTextTotalsV1, Complete: true, InputText: 3, OutputText: 2}
					} else {
						fact.Outcome, fact.ErrorCode = aicapability.InvocationFailed, "rejected_before_dispatch"
					}
					if terminal == "recover" {
						recorder.SetUsageSettler(nil)
					}
					require.NoError(t, recorder.RecordInvocation(ctx, fact))
					require.NoError(t, recorder.RecordInvocation(ctx, fact))
					if terminal == "recover" {
						completed, err := repo.RecoverDue(ctx)
						require.NoError(t, err)
						require.Equal(t, 1, completed)
						completed, err = repo.RecoverDue(ctx)
						require.NoError(t, err)
						require.Zero(t, completed)
					}
				}
				var bucket organizationResourceBucketRow
				require.NoError(t, db.Take(&bucket).Error)
				var expectedHeld, expectedConsumed int64
				if terminal == "unknown" {
					expectedHeld = 30
				} else if terminal == "settled" || terminal == "recover" {
					expectedConsumed = 7
				}
				require.Equal(t, expectedHeld, bucket.Reserved)
				require.Equal(t, expectedConsumed, bucket.Consumed)
				require.Equal(t, initial.Available-expectedHeld-expectedConsumed, bucket.Available)
				var count int64
				require.NoError(t, db.Model(&organizationResourceReservationRow{}).Where("owner_attempt_id = ?", fact.InvocationID).Count(&count).Error)
				if terminal == "no-hold" {
					require.Zero(t, count)
				} else {
					require.EqualValues(t, 1, count, "the original key cannot create a second reservation")
				}
			})
		}
	}
}

type aliasOwnerProof struct{ fact aicapability.InvocationRecord }

func (o aliasOwnerProof) ReadModelInvocation(context.Context, string, string) (aicapability.InvocationRecord, error) {
	return o.fact, nil
}

func TestResourceAliasRequiresExactFrozenWireBinding(t *testing.T) {
	for _, wire := range []string{"", "another/wire-model"} {
		t.Run("wire="+wire, func(t *testing.T) {
			db, _, repo, _, fact := modelPointFixture(t)
			fact.ModelID = "google/fixture-model"
			fact.OfficialTariff.WireModel = wire
			repo.owner = aliasOwnerProof{fact}
			require.ErrorIs(t, repo.ReserveAIInvocationUsage(context.Background(), "org", "member", fact.InvocationID, 20, fact.StartedAt), orgresource.ErrInvalidOwnerProof)
			var bucket organizationResourceBucketRow
			require.NoError(t, db.Take(&bucket).Error)
			require.EqualValues(t, 100, bucket.Available)
			require.Zero(t, bucket.Reserved)
		})
	}
}
