package tiktokshop

import (
	"strconv"
	"strings"

	"github.com/google/uuid"

	"task-processor/internal/authidentity"
)

// CredentialBinding is immutable server-owned AAD. Attempt staging and a
// canonical grant generation are different namespaces, even for equal UUIDs.
type CredentialBinding struct {
	OrganizationID, AppKey, AppRevision, ID, Purpose string
	Generation                                       int64
}

func (b CredentialBinding) AssociatedData() ([]byte, error) {
	if !authidentity.IsBoundedIdentifier(b.OrganizationID) || !authidentity.IsBoundedIdentifier(b.AppKey) || !authidentity.IsBoundedIdentifier(b.AppRevision) || uuid.Validate(b.ID) != nil ||
		b.Purpose != "attempt" && b.Purpose != "grant" || b.Purpose == "attempt" && b.Generation != 0 || b.Purpose == "grant" && b.Generation <= 0 {
		return nil, ErrInvalid
	}
	parts := []string{"task-processor/tiktokshop/credential/v1", b.OrganizationID, b.AppKey, b.AppRevision, b.Purpose, b.ID, strconv.FormatInt(b.Generation, 10)}
	var result strings.Builder
	for _, part := range parts {
		result.WriteString(strconv.Itoa(len(part)))
		result.WriteByte(':')
		result.WriteString(part)
	}
	return []byte(result.String()), nil
}

type CredentialProtection interface {
	Seal(CredentialBinding, TokenSet) (keyID, ciphertext string, err error)
	Open(CredentialBinding, string, string) (TokenSet, error)
}
