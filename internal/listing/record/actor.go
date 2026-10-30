package record

import "strings"

const (
	maxActorOrganizationBytes = 64
	maxActorUserBytes         = 128
	maxActorRoleBytes         = 128
	maxActorRoles             = 32
	maxOperationIDBytes       = 128
)

// Actor is the verified server identity used for an organization-scoped record
// operation. Identity facts and role authority remain with AuthIdentity/authz.
type Actor struct {
	OrganizationID string
	UserID         string
	Roles          []string
}

func ValidateActor(actor Actor) error {
	if !canonicalActorString(actor.OrganizationID, maxActorOrganizationBytes) ||
		!canonicalActorString(actor.UserID, maxActorUserBytes) ||
		len(actor.Roles) == 0 || len(actor.Roles) > maxActorRoles {
		return ErrInvalid
	}
	for _, role := range actor.Roles {
		if !canonicalActorString(role, maxActorRoleBytes) {
			return ErrInvalid
		}
	}
	return nil
}

func ValidateOperationID(operationID string) error {
	if !canonicalActorString(operationID, maxOperationIDBytes) {
		return ErrInvalid
	}
	return nil
}

func canonicalActorString(value string, maxBytes int) bool {
	return value != "" && value == strings.TrimSpace(value) && len(value) <= maxBytes
}
