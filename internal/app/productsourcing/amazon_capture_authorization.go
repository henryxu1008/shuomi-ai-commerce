package productsourcing

import (
	"context"
	"task-processor/internal/authz"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

type amazonContextAuthority struct {
	write        sourcing.PublicationAuthorizer
	manage, read collection.Authorizer
	readAccess   sourcing.LiveOrganizationAccess
	permissions  *authz.ListingKitAuthorizer
}

func NewAmazonCaptureContextAuthorizer(writeAccess, readAccess sourcing.LiveOrganizationAccess, permissions *authz.ListingKitAuthorizer) (AmazonCaptureAuthorizer, error) {
	write, err := sourcing.NewContextAuthorizer(writeAccess, permissions)
	if err != nil {
		return nil, err
	}
	manage, err := collection.NewContextAuthorizer(writeAccess, permissions)
	if err != nil {
		return nil, err
	}
	read, err := collection.NewContextAuthorizer(readAccess, permissions)
	if err != nil {
		return nil, err
	}
	return amazonContextAuthority{write, manage, read, readAccess, permissions}, nil
}
func (a amazonContextAuthority) Authorize(ctx context.Context, write bool) (collection.Scope, error) {
	if write {
		source, err := a.write.Authorize(ctx)
		if err != nil {
			return collection.Scope{}, err
		}
		scope, err := a.manage.Authorize(ctx, collection.PermissionManage)
		if err != nil {
			return scope, err
		}
		if scope.OrganizationID != source.OrganizationID || scope.ActorID != source.ActorID {
			return collection.Scope{}, sourcing.ErrPublicationForbidden
		}
		return scope, nil
	}
	scope, err := a.read.Authorize(ctx, collection.PermissionRead)
	if err != nil {
		return scope, err
	}
	roles, err := a.readAccess.ResolveLiveRoles(ctx, scope.OrganizationID, scope.ActorID)
	if err != nil {
		return collection.Scope{}, err
	}
	allowed, err := authz.AuthorizeOrganization(ctx, a.permissions, scope.ActorID, scope.OrganizationID, roles, authz.PermissionWorkbenchSupplyRead)
	if err != nil || !allowed {
		return collection.Scope{}, sourcing.ErrPublicationForbidden
	}
	return scope, nil
}
