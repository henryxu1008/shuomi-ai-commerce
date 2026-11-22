package temporal

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
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
