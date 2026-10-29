package membership

import (
	"context"
	"slices"
	"strconv"

	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
)

const (
	CommandBusinessRoleGrant  CommandKind    = "business_role_grant"
	CommandBusinessRoleRevoke CommandKind    = "business_role_revoke"
	StepConditionalRole       OperationStep  = "conditional_role"
	EventProviderNoEffect     OperationEvent = "provider_no_effect"
	RoleWriteApplied                         = "APPLIED"
	RoleWriteNoEffect                        = "VERSION_CONFLICT_NO_EFFECT"
)

// NativeAuthorizationRevision is one authoritative provider observation, never
// an authorization cache or a version assembled from a projected member list.
type NativeAuthorizationRevision struct {
	InstanceID      string   `json:"instanceId"`
	OrganizationID  string   `json:"organizationId"`
	ProjectID       string   `json:"projectId"`
	ProjectGrantID  string   `json:"projectGrantId"`
	AuthorizationID string   `json:"authorizationId"`
	UserID          string   `json:"userId"`
	NativeSequence  string   `json:"nativeSequence"`
	Roles           []string `json:"roles"`
	State           string   `json:"state"`
	ChangedAt       string   `json:"changedAt"`
}

type NativeRevisionReader interface {
	ReadAuthorizationRevision(context.Context, string, string) (NativeAuthorizationRevision, error)
}

type ConditionalRolePlan struct {
	Native     NativeAuthorizationRevision `json:"native"`
	Direction  string                      `json:"direction"`
	AfterRoles []string                    `json:"afterRoles"`
}

// RoleWriteEvidence binds a result of exactly one native conditional request.
// DispatchID is correlation only; it is not provider-side retry permission.
type RoleWriteEvidence struct {
	Result           string `json:"result"`
	InstanceID       string `json:"instanceId"`
	OrganizationID   string `json:"organizationId"`
	ProjectID        string `json:"projectId"`
	ProjectGrantID   string `json:"projectGrantId"`
	AuthorizationID  string `json:"authorizationId"`
	UserID           string `json:"userId"`
	ExpectedSequence string `json:"expectedSequence"`
	NativeSequence   string `json:"nativeSequence,omitempty"`
	RoleKey          string `json:"roleKey"`
	Direction        string `json:"direction"`
	DispatchID       string `json:"dispatchId"`
}

type ProviderNoEffectError struct{ Evidence RoleWriteEvidence }

func (e *ProviderNoEffectError) Error() string {
	return "native conditional role conflict with no effect"
}

func businessRoleKind(kind CommandKind) bool {
	return kind == CommandBusinessRoleGrant || kind == CommandBusinessRoleRevoke
}

func nativeSequence(value string) (uint64, bool) {
	n, err := strconv.ParseUint(value, 10, 63)
	return n, err == nil && n != 0 && strconv.FormatUint(n, 10) == value
}

func (r NativeAuthorizationRevision) valid() bool {
	_, sequenceOK := nativeSequence(r.NativeSequence)
	if !sequenceOK || !authidentity.IsBoundedIdentifier(r.InstanceID) || !authidentity.IsBoundedIdentifier(r.OrganizationID) || !authidentity.IsBoundedIdentifier(r.ProjectID) || !authidentity.IsBoundedIdentifier(r.AuthorizationID) || !authidentity.IsBoundedIdentifier(r.UserID) || (r.ProjectGrantID != "" && !authidentity.IsBoundedIdentifier(r.ProjectGrantID)) || r.State != "active" || len(r.Roles) == 0 || len(r.Roles) > 32 {
		return false
	}
	seen := map[string]bool{}
	for _, role := range r.Roles {
		if !authidentity.IsBoundedIdentifier(role) || seen[role] {
			return false
		}
		seen[role] = true
	}
	return true
}

func validConditionalPlan(op Operation) bool {
	if !businessRoleKind(op.Kind) || op.Step != StepConditionalRole || op.ConditionalRole == nil {
		return false
	}
	p := op.ConditionalRole
	r := p.Native
	if !r.valid() || r.AuthorizationID != op.AuthorizationID || r.UserID != op.TargetUserID || r.OrganizationID != op.Scope.OrganizationID || r.ProjectID != op.Scope.ProjectID || !authz.IsEnterpriseRoleKey(op.Scope.OrganizationID, op.Role) {
		return false
	}
	after := slices.Clone(r.Roles)
	if op.Kind == CommandBusinessRoleGrant {
		if p.Direction != "ADD" || slices.Contains(after, op.Role) || len(after) >= 32 {
			return false
		}
		after = append(after, op.Role)
	} else {
		if p.Direction != "REMOVE" || !slices.Contains(after, op.Role) {
			return false
		}
		after = slices.DeleteFunc(after, func(key string) bool { return key == op.Role })
		if len(after) == 0 {
			return false
		}
	}
	return len(after) == len(p.AfterRoles) && sameRoles(after, p.AfterRoles)
}

func (e RoleWriteEvidence) matches(op Operation) bool {
	if !validConditionalPlan(op) || op.DispatchID == "" {
		return false
	}
	r := op.ConditionalRole.Native
	return e.InstanceID == r.InstanceID && e.OrganizationID == r.OrganizationID && e.ProjectID == r.ProjectID && e.ProjectGrantID == r.ProjectGrantID && e.AuthorizationID == r.AuthorizationID && e.UserID == r.UserID && e.ExpectedSequence == r.NativeSequence && e.RoleKey == op.Role && e.Direction == op.ConditionalRole.Direction && e.DispatchID == op.DispatchID
}

func AuditOperation(kind CommandKind) string {
	if businessRoleKind(kind) {
		return string(CommandRole)
	}
	return string(kind)
}

func ConditionalRoleOperationValid(op Operation) bool          { return validConditionalPlan(op) }
func (e RoleWriteEvidence) MatchesOperation(op Operation) bool { return e.matches(op) }
func (r NativeAuthorizationRevision) Valid() bool              { return r.valid() }
