package temporal

import (
	"context"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/asset"
)

func (a *Activities) PrepareImageSetPolicyAdoption(ctx context.Context, identity imageagent.ExecutionIdentity) (*asset.ImageSetCommand, error) {
	if a.imageSetPolicyAdopter == nil {
		return nil, imageagent.ErrCommandBlocked
	}
	if _, err := a.loadOrganizationExecution(ctx, identity.RunID, identity); err != nil {
		return nil, err
	}
	ctx, err := a.restoreExecutionIdentity(ctx, identity.RunID, identity)
	if err != nil {
		return nil, err
	}
	return a.imageSetPolicyAdopter.PreparePolicyAdoption(ctx, identity)
}
