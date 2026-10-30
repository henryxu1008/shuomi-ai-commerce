package membership

const (
	CommandBusinessRoleGrant  CommandKind   = "business_role_grant"
	CommandBusinessRoleRevoke CommandKind   = "business_role_revoke"
	StepConditionalRole       OperationStep = "conditional_role"
)

// NativeAuthorizationRevision preserves the original stored receipt shape only.
// Official-provider code does not read or execute the retired native extension.
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

// businessRoleKind identifies persisted operations from the retired extension.
// These kinds remain readable; they are never accepted as executable commands.
func businessRoleKind(kind CommandKind) bool {
	return kind == CommandBusinessRoleGrant || kind == CommandBusinessRoleRevoke
}
