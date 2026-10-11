package productsourcing

import (
	"context"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

// PlanAuthority composes existing live permissions; it creates no identity or grant.
type PlanAuthority struct {
	Sourcing     sourcing.PublicationAuthorizer
	Collections  collection.Authorizer
	Preparations preparation.Authorizer
}

func (a PlanAuthority) AuthorizePlan(ctx context.Context, transfer bool) (sourcing.AcquisitionPlanScope, error) {
	var result sourcing.AcquisitionPlanScope
	if a.Sourcing == nil {
		return result, sourcing.ErrPublicationForbidden
	}
	source, e := a.Sourcing.Authorize(ctx)
	if e != nil {
		return result, e
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !authidentity.IsBoundedIdentifier(identity.EffectiveMemberID) || identity.UserID != source.ActorID || identity.EffectiveOrganizationID != source.OrganizationID {
		return result, sourcing.ErrPublicationForbidden
	}
	result = sourcing.AcquisitionPlanScope{OrganizationID: source.OrganizationID, ActorID: source.ActorID, MemberID: identity.EffectiveMemberID}
	if transfer {
		if a.Collections == nil || a.Preparations == nil {
			return result, sourcing.ErrAcquisitionUnavailable
		}
		scope, e := a.Preparations.Authorize(ctx, preparation.PermissionManage)
		if e != nil {
			return result, sourcing.ErrPublicationForbidden
		}
		wanted := collection.Scope{OrganizationID: result.OrganizationID, ActorID: result.ActorID, MemberID: result.MemberID}
		if scope != wanted {
			return result, sourcing.ErrPublicationForbidden
		}
		for _, permission := range []string{collection.PermissionRead, collection.PermissionManage} {
			scope, e := a.Collections.Authorize(ctx, permission)
			if e != nil || scope != wanted {
				return result, sourcing.ErrPublicationForbidden
			}
		}
	}
	return result, nil
}
func (s *AcquisitionPlanService) CanTransfer(ctx context.Context) (bool, error) {
	if _, e := s.auth.AuthorizePlan(ctx, false); e != nil {
		return false, e
	}
	_, e := s.auth.AuthorizePlan(ctx, true)
	return e == nil && s.collections != nil && s.preparations != nil, nil
}
