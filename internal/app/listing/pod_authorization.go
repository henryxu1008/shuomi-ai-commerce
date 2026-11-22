package listingapp

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/supplymarket"
)

// POD reuses this current original-membership owner. It grants only the three
// frozen design requirements, without accepting arbitrary permission strings.
func (a OrganizationExecutionAuthorizer) AuthorizePODDesign(ctx context.Context, scope collection.Scope) error {
	roles, err := a.current(ctx, scope)
	if err != nil {
		return err
	}
	for _, permission := range []string{collection.PermissionRead, collection.PermissionManage, supplymarket.PermissionDesign} {
		allowed, err := authz.AuthorizeOrganization(ctx, a.Permissions, scope.ActorID, scope.OrganizationID, roles, permission)
		if err != nil {
			return collection.ErrUnavailable
		}
		if !allowed {
			return collection.ErrForbidden
		}
	}
	return nil
}
