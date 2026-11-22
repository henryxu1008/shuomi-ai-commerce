package orgresourceadapter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
)

func TestSkillAuthoringParameterRejectionSettlesOriginalReservationAtZero(t *testing.T) {
	testWorkbenchParameterRejection(t, aicapability.OperationAIWorkbenchSkillAuthoring)
}
func TestChatIntentParameterRejectionSettlesOriginalReservationAtZero(t *testing.T) {
	testWorkbenchParameterRejection(t, aicapability.OperationAIWorkbenchChatIntent)
}
func testWorkbenchParameterRejection(t *testing.T, operation aicapability.Operation) {
	for _, delayed := range []bool{false, true} {
		name := "inline"
		if delayed {
			name = "recover_after_commercial_rollback"
		}
		t.Run(name, func(t *testing.T) {
			db, recorder, repo, _, fact := modelPointFixture(t, operation)
			confirm := repo.ConfirmSkillAuthoringParameterRejection
			if operation == aicapability.OperationAIWorkbenchChatIntent {
				confirm = repo.ConfirmChatIntentParameterRejection
			}
			ctx := context.Background()
			require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, fact.TenantID, fact.MemberID, fact.InvocationID, 20, fact.StartedAt))
			before, err := repo.InspectModelInvocationReservation(ctx, fact.TenantID, fact.InvocationID)
			require.NoError(t, err)
			require.Equal(t, "reserved", before.State)
			require.Error(t, confirm(ctx, fact.TenantID, fact.InvocationID))

			fact.Outcome = aicapability.InvocationFailed
			fact.ErrorCategory = aicapability.ErrorProviderRejected
			fact.ErrorCode = aicapability.SkillAuthoringOpenAIParameterRejectedV1
			if operation == aicapability.OperationAIWorkbenchChatIntent {
				fact.ErrorCode = aicapability.ChatIntentOpenAIParameterRejectedV1
			}
			fact.EstimatedCostKnown = true
			fact.FinishedAt = fact.StartedAt.Add(time.Second)
			fact.ProviderRequestID = strings.Repeat("a", 64)
			require.Error(t, recorder.ResolveDispatchedInvocation(ctx, fact), "manual recovery cannot mint an inline proof")
			crossPurpose := fact
			if operation == aicapability.OperationAIWorkbenchChatIntent {
				crossPurpose.ErrorCode = aicapability.SkillAuthoringOpenAIParameterRejectedV1
			} else {
				crossPurpose.ErrorCode = aicapability.ChatIntentOpenAIParameterRejectedV1
			}
			require.Error(t, recorder.RecordInvocation(ctx, crossPurpose))
			forged := fact
			forged.UsageKnown = true
			require.Error(t, recorder.RecordInvocation(ctx, forged))

			if delayed {
				require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject_parameter_rejection_audit", func(tx *gorm.DB) {
					if tx.Statement.Table == "saas_organization_resource_audit_logs" {
						tx.AddError(errors.New("synthetic settlement rollback"))
					}
				}))
				require.Error(t, recorder.RecordInvocation(ctx, fact))
				require.Error(t, confirm(ctx, fact.TenantID, fact.InvocationID))
				require.NoError(t, db.Callback().Create().Remove("reject_parameter_rejection_audit"))
				repo.now = func() time.Time { return fact.StartedAt.Add(time.Minute) }
				n, err := repo.RecoverDue(ctx)
				require.NoError(t, err)
				require.Equal(t, 1, n)
			} else {
				require.NoError(t, recorder.RecordInvocation(ctx, fact))
			}
			require.NoError(t, confirm(ctx, fact.TenantID, fact.InvocationID))
			require.Error(t, repo.ConfirmModelInvocationNoDispatchRelease(ctx, fact.TenantID, fact.InvocationID))
			require.NoError(t, recorder.RecordInvocation(ctx, fact))

			after, err := repo.InspectModelInvocationReservation(ctx, fact.TenantID, fact.InvocationID)
			require.NoError(t, err)
			require.Equal(t, before.ReservationID, after.ReservationID)
			require.Equal(t, "committed", after.State)
			var bucket organizationResourceBucketRow
			require.NoError(t, db.Take(&bucket).Error)
			require.EqualValues(t, 100, bucket.Available)
			require.Zero(t, bucket.Reserved)
			require.Zero(t, bucket.Consumed)
			var month memberAIPointMonthRow
			require.NoError(t, db.Take(&month).Error)
			require.Zero(t, month.Reserved)
			require.Zero(t, month.Consumed)
			var events []organizationResourceEventRow
			require.NoError(t, db.Find(&events).Error)
			require.Len(t, events, 2, "replaying the same receipt must not credit again")
			for _, event := range events {
				require.Zero(t, event.ConsumedDelta)
			}
		})
	}
}
