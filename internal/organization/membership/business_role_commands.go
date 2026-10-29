package membership

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
)

func (s *Service) businessRoleEligible(ctx context.Context, member Member, kind CommandKind, key string) bool {
	if !businessRoleKind(kind) || member.State != "active" || s.roleStore == nil || !authz.IsEnterpriseRoleKey(member.OrganizationID, key) || member.ProjectID != s.projectID || len(member.Roles) == 0 || len(member.Roles) > 32 {
		return false
	}
	definitions, _, err := s.roleStore.Roles(ctx, member.OrganizationID)
	if err != nil {
		return false
	}
	known := map[string]bool{"listingkit_admin": true}
	for role, protected := range s.protectedRoles {
		if protected {
			known[role] = true
		}
	}
	var selected *RoleDefinition
	for i := range definitions {
		role := &definitions[i]
		if !role.System && authz.IsEnterpriseRoleKey(member.OrganizationID, role.ID) {
			known[role.ID] = true
			if role.ID == key {
				selected = role
			}
		}
	}
	if selected == nil {
		return false
	}
	protected := s.authorizer.Authorize(member.UserID, member.Roles, authz.PermissionListingKitPlatformAdm) || slices.Contains(member.Roles, "listingkit_admin")
	seen := map[string]bool{}
	for _, role := range member.Roles {
		if !known[role] || seen[role] {
			return false
		}
		seen[role] = true
		protected = protected || s.protectedRoles[role]
	}
	if !protected {
		return false
	}
	if kind == CommandBusinessRoleGrant {
		return len(selected.Modules) == 1 && selected.Modules[0] == "store-browser" && !seen[key] && len(member.Roles) < 32
	}
	return seen[key] && len(member.Roles) > 1
}

func (c *Commands) executeBusinessRole(ctx context.Context, identity authidentity.AuthenticatedIdentity, key string, input CommandInput) (Operation, error) {
	if input.Invitation != nil || !authidentity.IsBoundedIdentifier(input.AuthorizationID) || !versionPattern.MatchString(input.ExpectedVersion) || !authz.IsEnterpriseRoleKey(identity.EffectiveOrganizationID, input.Role) {
		return Operation{}, ErrInvalidRequest
	}
	scope := OperationScope{ProjectID: c.service.projectID, OrganizationID: identity.EffectiveOrganizationID, ActorID: identity.UserID}
	payload, _ := json.Marshal(input)
	digest := sha256.Sum256(payload)
	fingerprint := hex.EncodeToString(digest[:])
	existing, err := c.store.Read(ctx, scope, key)
	if err == nil {
		if existing.Fingerprint != fingerprint {
			return Operation{}, ErrConflict
		}
		return c.resumeBusinessRole(ctx, existing)
	}
	if !errors.Is(err, ErrNotFound) {
		return Operation{}, ErrUnavailable
	}
	member, err := c.readTarget(ctx, input.AuthorizationID)
	if err != nil {
		return Operation{}, err
	}
	op, err := c.store.Begin(ctx, Operation{Scope: scope, Key: key, Fingerprint: fingerprint, Kind: input.Kind, TargetUserID: member.UserID, AuthorizationID: member.ID, ExpectedVersion: input.ExpectedVersion, Role: input.Role, Step: StepConditionalRole, Phase: PhaseReady, Revision: 1})
	if err != nil {
		return Operation{}, err
	}
	return c.resumeBusinessRole(ctx, op)
}

func (c *Commands) rejectBusinessRole(ctx context.Context, op Operation) (Operation, error) {
	rejected, err := c.store.Apply(ctx, op.Scope, op.Key, op.Revision, OperationChange{Event: EventReject})
	if err != nil {
		return op, err
	}
	return rejected, ErrConflict
}

func (c *Commands) resumeBusinessRole(ctx context.Context, op Operation) (Operation, error) {
	if op.Phase != PhaseReady {
		return op, nil
	}
	ctx, identity, err := c.current(ctx)
	if err != nil {
		return op, err
	}
	if op.Scope != (OperationScope{ProjectID: c.service.projectID, OrganizationID: identity.EffectiveOrganizationID, ActorID: identity.UserID}) {
		return op, ErrPermission
	}
	member, err := c.readTarget(ctx, op.AuthorizationID)
	if errors.Is(err, ErrNotFound) {
		return c.rejectBusinessRole(ctx, op)
	}
	if err != nil {
		return op, err
	}
	if member.UserID != op.TargetUserID || observedVersion(member) != op.ExpectedVersion || !c.service.businessRoleEligible(ctx, member, op.Kind, op.Role) {
		return c.rejectBusinessRole(ctx, op)
	}
	reader, ok := c.service.directory.(NativeRevisionReader)
	if !ok {
		return op, ErrUnavailable
	}
	native, err := reader.ReadAuthorizationRevision(ctx, op.Scope.OrganizationID, op.AuthorizationID)
	if err != nil {
		return op, err
	}
	if !native.valid() || native.OrganizationID != member.OrganizationID || native.ProjectID != member.ProjectID || native.AuthorizationID != member.ID || native.UserID != member.UserID {
		return op, ErrInvalidResponse
	}
	projectedTime, parseProjected := time.Parse(time.RFC3339Nano, member.ChangedAt)
	nativeTime, parseNative := time.Parse(time.RFC3339Nano, native.ChangedAt)
	if native.State != member.State || !sameRoles(native.Roles, member.Roles) || parseProjected != nil || parseNative != nil || !projectedTime.Equal(nativeTime) {
		return c.rejectBusinessRole(ctx, op)
	}
	plan := &ConditionalRolePlan{Native: native, Direction: "ADD", AfterRoles: append(slices.Clone(native.Roles), op.Role)}
	if op.Kind == CommandBusinessRoleRevoke {
		plan.Direction = "REMOVE"
		plan.AfterRoles = slices.DeleteFunc(slices.Clone(native.Roles), func(key string) bool { return key == op.Role })
	}
	ctx, identity, err = c.current(ctx)
	if err != nil {
		return op, err
	}
	if identity.UserID != op.Scope.ActorID || identity.EffectiveOrganizationID != op.Scope.OrganizationID || dispatchAllowed(ctx) != nil {
		return op, ErrPermission
	}
	dispatchID := uuid.NewString()
	dispatched, err := c.store.Apply(ctx, op.Scope, op.Key, op.Revision, OperationChange{Event: EventDispatch, DispatchID: dispatchID, ConditionalRole: plan})
	if err != nil {
		return op, err
	}
	ctx, identity, err = c.current(ctx)
	if err != nil || identity.UserID != op.Scope.ActorID || identity.EffectiveOrganizationID != op.Scope.OrganizationID || dispatchAllowed(ctx) != nil || !c.service.businessRoleEligible(ctx, member, op.Kind, op.Role) {
		return dispatched, nil
	}
	ack, writeErr := c.writer.Write(ctx, dispatched)
	if writeErr != nil {
		var noEffect *ProviderNoEffectError
		if errors.As(writeErr, &noEffect) && noEffect != nil {
			rejected, saveErr := c.store.Apply(ctx, op.Scope, op.Key, dispatched.Revision, OperationChange{Event: EventProviderNoEffect, DispatchID: dispatchID, NoEffect: &noEffect.Evidence})
			if saveErr == nil {
				return rejected, ErrConflict
			}
		}
		return dispatched, nil
	}
	if ack.ID != op.AuthorizationID {
		return dispatched, nil
	}
	if _, err = time.Parse(time.RFC3339Nano, ack.At); err != nil {
		return dispatched, nil
	}
	completed, err := c.store.Apply(ctx, op.Scope, op.Key, dispatched.Revision, OperationChange{Event: EventAcknowledge, DispatchID: dispatchID, Acknowledgment: &ack})
	if err != nil {
		return dispatched, nil
	}
	return completed, nil
}
