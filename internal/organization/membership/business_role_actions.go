package membership

import (
	"context"
	"time"
)

func nativeMatchesMember(native NativeAuthorizationRevision, member Member) bool {
	projectedTime, pErr := time.Parse(time.RFC3339Nano, member.ChangedAt)
	nativeTime, nErr := time.Parse(time.RFC3339Nano, native.ChangedAt)
	return native.valid() && native.OrganizationID == member.OrganizationID && native.ProjectID == member.ProjectID && native.AuthorizationID == member.ID && native.UserID == member.UserID && native.State == member.State && sameRoles(native.Roles, member.Roles) && pErr == nil && nErr == nil && projectedTime.Equal(nativeTime)
}

func (s *Service) businessRoleActions(ctx context.Context, member Member, manage bool, definitions []RoleDefinition) *BusinessRoleActions {
	actions := &BusinessRoleActions{GrantableRoleIDs: []string{}, RevocableRoleIDs: []string{}, Reason: "not_allowed"}
	if !manage {
		return actions
	}
	for _, role := range definitions {
		if s.businessRoleEligibleWithDefinitions(member, CommandBusinessRoleGrant, role.ID, definitions) {
			actions.GrantableRoleIDs = append(actions.GrantableRoleIDs, role.ID)
		}
		if s.businessRoleEligibleWithDefinitions(member, CommandBusinessRoleRevoke, role.ID, definitions) {
			actions.RevocableRoleIDs = append(actions.RevocableRoleIDs, role.ID)
		}
	}
	if len(actions.GrantableRoleIDs)+len(actions.RevocableRoleIDs) == 0 {
		actions.Reason = "no_eligible_role"
		return actions
	}
	reader, ok := s.directory.(NativeRevisionReader)
	if ok {
		native, err := reader.ReadAuthorizationRevision(ctx, member.OrganizationID, member.ID)
		if err == nil && nativeMatchesMember(native, member) {
			actions.Reason = "ready"
			return actions
		}
		actions.Reason = "state_changed"
		if err != nil {
			actions.Reason = "provider_unavailable"
		}
	} else {
		actions.Reason = "provider_unavailable"
	}
	actions.GrantableRoleIDs, actions.RevocableRoleIDs = []string{}, []string{}
	return actions
}
