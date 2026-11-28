package temporal

import (
	"context"
	"fmt"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	productasset "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func (o *workflowEffectOwner) publishImageSet(ctx workflow.Context, input PublishImageSetActivityInput) error {
	return o.execute(ctx, "", func(ownerCtx workflow.Context) error {
		if err := workflow.ExecuteActivity(ownerCtx, activityPublishApprovedImageSet, input).Get(ownerCtx, nil); err != nil {
			return fmt.Errorf("publish selected image set: %w", err)
		}
		return nil
	})
}

func (a *Activities) PublishApprovedImageSet(ctx context.Context, input PublishImageSetActivityInput) error {
	if a.imageSetPublisher == nil {
		return imageagent.ErrCommandBlocked
	}
	var original *imageagent.RunProjection
	if a.imageSetApprovals != nil && a.executionAuthorizer != nil {
		current, err := a.loadOrganizationExecution(ctx, input.RunID, input.Identity)
		if err != nil {
			return err
		}
		if input.PlanRevision != current.Plan.Revision || input.ResultDigest != current.ResultDigest {
			return imageagent.ErrRevisionConflict
		}
		if err := imageagent.ValidateImageSetSelectionIntent(input.RunID, current.Plan, input.ResultDigest, input.Selection.ActionID, &input.Selection); err != nil {
			return err
		}
		commit, committed, err := a.readCommittedImageSetApproval(ctx, current)
		if err != nil {
			return err
		}
		if committed {
			if commit.ActionID != input.Selection.ActionID || commit.ImageSet.Digest != input.Selection.SelectionDigest || commit.ImageSet.RequestDigest != productasset.ImageSetRequestDigest(input.Selection) {
				return imageagent.ErrRevisionConflict
			}
			return nil
		}
		if current.Plan.Set.ColorImageRework() != nil {
			pending := current.PendingCommand
			if pending == nil || pending.ActionID != input.Selection.ActionID || pending.SelectionDigest != input.Selection.SelectionDigest ||
				pending.RequestDigest != productasset.ImageSetRequestDigest(input.Selection) {
				return imageagent.ErrRevisionConflict
			}
			if notSaved, err := a.colorReworkSaveNotCommitted(ctx, current); err != nil {
				return err
			} else if notSaved {
				return colorImageReworkNotSaved()
			}
			original = &current
		}
	}
	releaseCapacity, err := a.configurationCapacity.Acquire(ctx, 1)
	if err != nil {
		return err
	}
	defer releaseCapacity()
	ctx, err = a.restoreExecutionIdentity(ctx, input.RunID, input.Identity)
	if err != nil {
		return err
	}
	_, err = a.imageSetPublisher.PublishApprovedImageSet(ctx, imageagent.PublishImageSetInput{RunID: input.RunID, TenantID: input.Identity.TenantID, UserID: input.Identity.UserID, PlanRevision: input.PlanRevision, ResultDigest: input.ResultDigest, Selection: *imageagent.CloneImageSetCommand(&input.Selection)})
	if err != nil && original != nil {
		releaseCapacity()
		if notSaved, readErr := a.colorReworkSaveNotCommitted(ctx, *original); readErr == nil && notSaved {
			return colorImageReworkNotSaved()
		}
	}
	return err
}

func colorImageReworkNotSaved() error {
	return sdktemporal.NewNonRetryableApplicationError("the original inventory head was superseded", imageagent.ColorImageReworkApprovalNotSaved, nil)
}
