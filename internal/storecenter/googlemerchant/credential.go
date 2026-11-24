package googlemerchant

import (
	"encoding/json"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
)

type CredentialPurpose string

const (
	AttemptCredentials CredentialPurpose = "attempt"
	GrantCredentials   CredentialPurpose = "grant"
)

type CredentialBinding struct {
	OrganizationID, StoreID, ApplicationID, ApplicationRevision, ID string
	Purpose                                                         CredentialPurpose
	Generation                                                      int64
}

func (b CredentialBinding) AssociatedData() ([]byte, error) {
	if !authidentity.IsBoundedIdentifier(b.OrganizationID) || !authidentity.IsBoundedIdentifier(b.StoreID) ||
		!authidentity.IsBoundedIdentifier(b.ApplicationID) || !authidentity.IsBoundedIdentifier(b.ApplicationRevision) ||
		!authidentity.IsBoundedIdentifier(b.ID) {
		return nil, ErrInvalidGrantInput
	}
	switch b.Purpose {
	case AttemptCredentials:
		if b.Generation != 0 {
			return nil, ErrInvalidGrantInput
		}
	case GrantCredentials:
		if b.Generation < 1 {
			return nil, ErrInvalidGrantInput
		}
	default:
		return nil, ErrInvalidGrantInput
	}
	return json.Marshal([]any{"task-processor/google-merchant/credential/v1", b.OrganizationID, b.StoreID,
		b.ApplicationID, b.ApplicationRevision, b.ID, b.Purpose, b.Generation})
}

type CredentialProtection interface {
	Seal(CredentialBinding, TokenSet) (keyID, ciphertext string, err error)
	Open(CredentialBinding, string, string) (TokenSet, error)
}

func (TokenSet) String() string   { return "Google Merchant credential [redacted]" }
func (TokenSet) GoString() string { return "Google Merchant credential [redacted]" }
