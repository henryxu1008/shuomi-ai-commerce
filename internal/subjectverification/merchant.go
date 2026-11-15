package subjectverification

import (
	"context"
	"errors"
	"task-processor/internal/authidentity"
)

// VerifiedCompanyName is a narrow internal consumer of the verification owner.
// It exposes no application, applicant, credit code, credentials or provider URL.
func (s *Service) VerifiedCompanyName(ctx context.Context, organizationID string) (string, error) {
	if ctx == nil || s == nil || s.Store == nil || !authidentity.IsBoundedIdentifier(s.Scope) || !authidentity.IsBoundedIdentifier(organizationID) {
		return "", ErrUnavailable
	}
	a, err := s.Store.Read(ctx, organizationID)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if a.Scope != s.Scope || a.OrganizationID != organizationID {
		return "", ErrConflict
	}
	if a.State != Verified {
		return "", nil
	}
	if !validText(a.CompanyName, 256) || a.ProviderVerifiedAt.IsZero() || !authidentity.IsBoundedIdentifier(a.ProviderOrganizationID) || !authidentity.IsBoundedIdentifier(a.ProviderAdminID) {
		return "", ErrUnavailable
	}
	return a.CompanyName, nil
}
