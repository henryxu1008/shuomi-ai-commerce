package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"task-processor/internal/authidentity"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"time"
)

// Local saved-order authority never decrypts a credential or queries a provider.
func (a *OfficialObservationAccess) ReadSavedTemuOrderBinding(ctx context.Context, s storecenter.ObservationSubject, id string) (MerchantBinding, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || a.applications == nil || s.Purpose != storecenter.ObservationPurposeOrders || s.Sync {
		return MerchantBinding{}, o.ErrUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	now := a.now()
	m, e := a.reader.ReadObservation(ctx, s, id, a.authorization, now)
	if e != nil {
		return MerchantBinding{}, observationError(e)
	}
	entry, e := a.applications.resolve(m.Attempt.AppID, m.Attempt.AppVersion, m.Platform, m.Attempt.IdentityPartition)
	if e != nil || entry.application.SupportedModes != storecenter.ModesLocal|storecenter.ModesSemiManaged {
		return MerchantBinding{}, o.ErrConflict
	}
	binding, e := temuOrderBinding(s.OrganizationID, id, m, entry, now)
	if ctx.Err() != nil {
		return MerchantBinding{}, o.ErrUnavailable
	}
	return binding, e
}

func temuOrderBinding(organizationID, storeID string, m storecenter.ProductExecutionMaterial, entry officialApplicationEntry, now time.Time) (MerchantBinding, error) {
	app, attempt, connection := entry.application, m.Attempt, m.Connection
	if m.Platform != storecenter.PlatformTemu || app.Platform != m.Platform || attempt.Platform != m.Platform || connection.Platform != m.Platform || app.AuthorizationMethod != storecenter.AuthorizationManualToken || attempt.AuthorizationMethod != app.AuthorizationMethod || connection.AuthorizationMethod != app.AuthorizationMethod || attempt.IdentityPartition != app.IdentityPartition || connection.IdentityPartition != app.IdentityPartition || attempt.OrganizationID != organizationID || attempt.StoreID != storeID || attempt.AttemptID != connection.AttemptID || attempt.ConnectionVersion != connection.Version || connection.Status != storecenter.ConnectionStatusConnected || attempt.State != "verified" || connection.State != "verified" || attempt.AppID != app.AppID || attempt.AppVersion != app.Version || connection.AppID != app.AppID || connection.AppRevision != app.Version || m.StoreVersion < 1 || connection.Version < 1 || !now.Before(m.ServiceExpiresAt) {
		return MerchantBinding{}, o.ErrConflict
	}
	var facts storecenter.OfficialVerificationFacts
	if len(attempt.VerificationJSON) == 0 || len(attempt.VerificationJSON) > 2*1024*1024 || json.Unmarshal([]byte(attempt.VerificationJSON), &facts) != nil || facts.Platform != app.Platform || facts.IdentityPartition != app.IdentityPartition || facts.Mode != connection.Mode || facts.RegionID != connection.RegionID || connection.CredentialExpiresAt == nil || !facts.ExpiresAt.Equal(*connection.CredentialExpiresAt) || !now.Before(facts.ExpiresAt) || facts.VerifiedAt.IsZero() || facts.VerifiedAt.After(now) || !authidentity.IsBoundedIdentifier(facts.ContractRevision) || facts.Scopes == nil || len(facts.Scopes) > 4096 {
		return MerchantBinding{}, o.ErrConflict
	}
	for _, scope := range facts.Scopes {
		if !authidentity.IsBoundedIdentifier(scope) {
			return MerchantBinding{}, o.ErrConflict
		}
	}
	identity, err := facts.MerchantIdentityHash()
	if err != nil || identity != connection.MerchantIdentityHash {
		return MerchantBinding{}, o.ErrConflict
	}
	scopes := append([]string{}, facts.Scopes...)
	sort.Strings(scopes)
	encoded, _ := json.Marshal(scopes)
	scopesHash := sha256.Sum256(encoded)
	site := "temu-order-us"
	binding := MerchantBinding{Platform: app.Platform, IdentityPartition: app.IdentityPartition, MerchantMode: facts.Mode, RegionID: facts.RegionID, CredentialExpiresAt: facts.ExpiresAt.UTC(), VerificationRevision: facts.ContractRevision, ScopesHash: hex.EncodeToString(scopesHash[:]), OrganizationID: organizationID, StoreID: storeID, Site: site, StoreVersion: m.StoreVersion, ConnectionRevision: connection.Version, ApplicationID: app.AppID, ApplicationRevision: app.Version, SupplierIdentityHash: identity, ServiceExpiresAt: m.ServiceExpiresAt.UTC()}
	if !binding.ValidTemuOrderApplication() {
		return MerchantBinding{}, o.ErrConflict
	}
	return binding, nil
}
