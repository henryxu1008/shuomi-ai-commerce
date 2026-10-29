package temporal

import (
	"context"
	"fmt"

	sdkactivity "go.temporal.io/sdk/activity"

	"task-processor/internal/imageagent"
)

// PublishApprovedV3 is additive and intentionally not registered here. Task 6
// owns selecting and registering imageagent.publish_approved.v3.
func (a *Activities) PublishApprovedV3(ctx context.Context, input PublishApprovedV3ActivityInput) error {
	current, readErr := a.repository.GetProjection(ctx, imageagent.RunScope{TenantID: input.Identity.TenantID, OwnerUserID: input.Identity.UserID, RunID: input.RunID})
	if readErr != nil {
		return readErr
	}
	if current.Plan.Artwork != nil {
		if a.artworkPublisher == nil || a.executionAuthorizer == nil || current.PendingCommand == nil || current.PendingCommand.PlanRevision != input.PlanRevision || current.PendingCommand.ResultDigest != current.ResultDigest {
			return imageagent.ErrCommandBlocked
		}
		loaded, err := a.loadOrganizationExecution(ctx, input.RunID, input.Identity)
		if err != nil {
			return err
		}
		ctx = withOrganizationActivityIdentity(ctx, loaded.Run, input.Identity.TraceID)
		_, err = a.artworkPublisher.PublishArtwork(ctx, imageagent.PublishArtworkInput{Identity: input.Identity, PlanRevision: input.PlanRevision, ResultDigest: current.ResultDigest, ActionID: current.PendingCommand.ActionID})
		return err
	}
	if a.publisherV3 == nil {
		return fmt.Errorf("image agent v3 approved asset publisher is required")
	}
	ctx, err := a.restoreExecutionIdentity(ctx, input.RunID, input.Identity)
	if err != nil {
		return err
	}
	_, err = a.publisherV3.PublishApprovedV3(ctx, imageagent.PublishApprovedV3Input{
		RunID: input.RunID, TenantID: input.Identity.TenantID, UserID: input.Identity.UserID,
		PlanRevision: input.PlanRevision, CandidateAssetIDs: append([]string(nil), input.CandidateAssetIDs...),
		IdempotencyKey: input.IdempotencyKey,
	})
	return err
}

type activityRegistrar interface {
	RegisterActivityWithOptions(interface{}, sdkactivity.RegisterOptions)
}

func RegisterActivitiesForMode(registrar activityRegistrar, activities *Activities, mode WorkerWireMode) error {
	if registrar == nil {
		return fmt.Errorf("temporal activity registrar is required")
	}
	if activities == nil {
		return fmt.Errorf("image agent activities are required")
	}
	if err := validateActivityMode(activities, mode); err != nil {
		return err
	}
	if err := validateWorkerWireMode(mode); err != nil {
		return err
	}

	registrar.RegisterActivityWithOptions(activities.PersistRunState, sdkactivity.RegisterOptions{Name: activityPersistRunState})
	registrar.RegisterActivityWithOptions(activities.PersistWorkflowFailure, sdkactivity.RegisterOptions{Name: activityPersistWorkflowFailure})
	registrar.RegisterActivityWithOptions(activities.PersistWorkflowFailureV2, sdkactivity.RegisterOptions{Name: activityPersistWorkflowFailureV2})
	registrar.RegisterActivityWithOptions(activities.PersistPlanRevision, sdkactivity.RegisterOptions{Name: activityPersistPlanRevision})
	registrar.RegisterActivityWithOptions(activities.PersistPendingCommand, sdkactivity.RegisterOptions{Name: activityPersistPendingCommand})

	registrar.RegisterActivityWithOptions(activities.ExecuteSlotV3, sdkactivity.RegisterOptions{Name: activityExecuteSlotV3})
	registrar.RegisterActivityWithOptions(activities.ReviewStagedSlotV3, sdkactivity.RegisterOptions{Name: activityReviewStagedSlotV3})
	registrar.RegisterActivityWithOptions(activities.StartEffectRecoveryV3, sdkactivity.RegisterOptions{Name: activityStartEffectRecoveryV3})
	registrar.RegisterActivityWithOptions(activities.RecoverEffectV3, sdkactivity.RegisterOptions{Name: activityRecoverEffectV3})
	registrar.RegisterActivityWithOptions(activities.PersistRecoveryBlockedEffectV3, sdkactivity.RegisterOptions{Name: activityPersistRecoveryBlockedV3})
	registrar.RegisterActivityWithOptions(activities.ReconcileEffectRecoveryV3, sdkactivity.RegisterOptions{Name: activityReconcileEffectRecoveryV3})
	registrar.RegisterActivityWithOptions(activities.PersistSlotResultV3, sdkactivity.RegisterOptions{Name: activityPersistSlotResultV3})

	registrar.RegisterActivityWithOptions(activities.PrepareImageSetPolicyAdoption, sdkactivity.RegisterOptions{Name: activityPrepareImageSetPolicyAdoption})
	registrar.RegisterActivityWithOptions(activities.PublishApprovedImageSet, sdkactivity.RegisterOptions{Name: activityPublishApprovedImageSet})
	registrar.RegisterActivityWithOptions(activities.PersistImageSetSlotResult, sdkactivity.RegisterOptions{Name: activityPersistImageSetSlotResult})

	registrar.RegisterActivityWithOptions(activities.PublishApprovedV3, sdkactivity.RegisterOptions{Name: activityPublishApprovedV3})

	return nil
}
