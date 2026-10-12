package storecenterapp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
	"time"
	"unicode/utf8"
)

// TemuRulesProvider is a named read-only capability of the original registered
// official application. It cannot create products or accept arbitrary API types.
type TemuRulesProvider interface {
	QuerySupplierCategories(context.Context, storecenter.OfficialMerchantCredential, model.SupplierCategoryQuery) ([]model.SupplierCategory, error)
	QueryLocalCategories(context.Context, storecenter.OfficialMerchantCredential, model.LocalCategoryQuery) ([]model.LocalCategory, error)
	QuerySupplierAttributes(context.Context, storecenter.OfficialMerchantCredential, int64) (model.SupplierAttributeTemplate, error)
	QueryLocalCategoryRules(context.Context, storecenter.OfficialMerchantCredential, model.CategoryRuleQuery) (model.LocalCategoryRules, error)
	QueryLocalAttributes(context.Context, storecenter.OfficialMerchantCredential, model.LocalAttributeQuery) (model.LocalAttributeTemplate, error)
	QueryLocalFreightTemplates(context.Context, storecenter.OfficialMerchantCredential) ([]model.LocalFreightTemplate, error)
	QueryLocalTrademarks(context.Context, storecenter.OfficialMerchantCredential, model.LocalTrademarkQuery) (model.LocalTrademarkPage, error)
	QueryLocalVariations(context.Context, storecenter.OfficialMerchantCredential, model.CategoryRuleQuery) (model.LocalVariationTemplate, error)
}

// TemuRulesHandle shares the current Store execution owner. Only this process,
// original caller and exact private material can use its five-second proof.
type TemuRulesHandle struct {
	owner                         *OfficialProductAccess
	entry                         officialApplicationEntry
	subject                       storecenter.ProductExecutionSubject
	binding                       MerchantBinding
	connectionRef, credentialHash string
	expiresAt                     time.Time
}

func (a *OfficialProductAccess) AuthorizeTemuRules(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string, expected *MerchantBinding) (*TemuRulesHandle, error) {
	if a == nil || ctx == nil || ctx.Err() != nil || subject.Purpose != storecenter.ProductPurposeRules {
		return nil, ErrProductAccessChanged
	}
	ctx, cancel := context.WithDeadline(ctx, a.now().Add(5*time.Second))
	defer cancel()
	material, err := a.reader.ReadProductExecution(ctx, subject, storeID, a.authorization, a.now())
	if err != nil || material.Platform != storecenter.PlatformTemu {
		return nil, ErrProductAccessChanged
	}
	entry, err := a.applications.resolve(material.Attempt.AppID, material.Attempt.AppVersion, material.Platform, material.Attempt.IdentityPartition)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	if _, ok := entry.provider.(TemuRulesProvider); !ok {
		return nil, ErrProductAccessChanged
	}
	credential, err := entry.protection.Open(material.Attempt, material.Attempt.KeyID, material.Attempt.Ciphertext)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	binding, ok := temuMerchantBinding(subject, storeID, material, credential, entry, a.now())
	if !ok || expected != nil && *expected != binding {
		return nil, ErrProductAccessChanged
	}
	// An old connected flag cannot replace current official identity/expiry.
	info, err := entry.provider.QueryStore(ctx, credential)
	if err != nil || info.Verification == nil {
		return nil, ErrProductAccessChanged
	}
	current := material
	encoded, err := json.Marshal(info.Verification)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	current.Attempt.VerificationJSON = string(encoded)
	verified, ok := temuMerchantBinding(subject, storeID, current, credential, entry, a.now())
	if !ok || verified != binding {
		return nil, ErrProductAccessChanged
	}
	// Re-read live membership/grant/connection after the official HTTP read.
	fresh, err := a.reader.ReadProductExecution(ctx, subject, storeID, a.authorization, a.now())
	if err != nil || privateCredentialHash(fresh) != privateCredentialHash(material) || fresh.Connection.AttemptID != material.Connection.AttemptID {
		return nil, ErrProductAccessChanged
	}
	freshBinding, ok := temuMerchantBinding(subject, storeID, fresh, credential, entry, a.now())
	if !ok || freshBinding != binding {
		return nil, ErrProductAccessChanged
	}
	expiresAt, _ := ctx.Deadline()
	if ctx.Err() != nil || !a.now().Before(expiresAt) {
		return nil, ErrProductAccessChanged
	}
	return &TemuRulesHandle{owner: a, entry: entry, subject: subject, binding: binding, connectionRef: material.Connection.AttemptID, credentialHash: privateCredentialHash(material), expiresAt: expiresAt}, nil
}

func (h *TemuRulesHandle) Binding() MerchantBinding {
	if h == nil {
		return MerchantBinding{}
	}
	return h.binding
}

func (h *TemuRulesHandle) credential(ctx context.Context) (storecenter.OfficialMerchantCredential, error) {
	if h == nil || h.owner == nil || ctx == nil || ctx.Err() != nil || !h.owner.now().Before(h.expiresAt) || h.subject.Purpose != storecenter.ProductPurposeRules {
		return storecenter.OfficialMerchantCredential{}, ErrProductAccessChanged
	}
	material, err := h.owner.reader.ReadProductExecution(ctx, h.subject, h.binding.StoreID, h.owner.authorization, h.owner.now())
	if err != nil || material.Connection.AttemptID != h.connectionRef || privateCredentialHash(material) != h.credentialHash {
		return storecenter.OfficialMerchantCredential{}, ErrProductAccessChanged
	}
	entry, err := h.owner.applications.resolve(material.Attempt.AppID, material.Attempt.AppVersion, material.Platform, material.Attempt.IdentityPartition)
	if err != nil || entry.application != h.entry.application || entry.mode != h.entry.mode {
		return storecenter.OfficialMerchantCredential{}, ErrProductAccessChanged
	}
	credential, err := entry.protection.Open(material.Attempt, material.Attempt.KeyID, material.Attempt.Ciphertext)
	if err != nil {
		return storecenter.OfficialMerchantCredential{}, ErrProductAccessChanged
	}
	binding, ok := temuMerchantBinding(h.subject, h.binding.StoreID, material, credential, entry, h.owner.now())
	if !ok || binding != h.binding || !h.owner.now().Before(h.expiresAt) {
		return storecenter.OfficialMerchantCredential{}, ErrProductAccessChanged
	}
	return credential, nil
}

func temuMerchantBinding(subject storecenter.ProductExecutionSubject, storeID string, m storecenter.ProductExecutionMaterial, credential storecenter.OfficialMerchantCredential, entry officialApplicationEntry, now time.Time) (MerchantBinding, bool) {
	app, attempt, connection := entry.application, m.Attempt, m.Connection
	if subject.Purpose != storecenter.ProductPurposeRules || m.Platform != storecenter.PlatformTemu || app.Platform != m.Platform || attempt.Platform != m.Platform || connection.Platform != m.Platform || app.AuthorizationMethod != storecenter.AuthorizationManualToken || attempt.AuthorizationMethod != app.AuthorizationMethod || connection.AuthorizationMethod != app.AuthorizationMethod || attempt.IdentityPartition != app.IdentityPartition || connection.IdentityPartition != app.IdentityPartition || attempt.OrganizationID != subject.OrganizationID || attempt.StoreID != storeID || attempt.AttemptID != connection.AttemptID || attempt.ConnectionVersion != connection.Version || connection.Status != storecenter.ConnectionStatusConnected || attempt.State != "verified" || connection.State != "verified" || attempt.AppID != app.AppID || attempt.AppVersion != app.Version || connection.AppID != app.AppID || connection.AppRevision != app.Version || m.StoreVersion < 1 || connection.Version < 1 || !now.Before(m.ServiceExpiresAt) || credential.AppID != app.AppID || !validTemuPrivateCredential(credential) {
		return MerchantBinding{}, false
	}
	var facts storecenter.OfficialVerificationFacts
	if len(attempt.VerificationJSON) == 0 || len(attempt.VerificationJSON) > 2*1024*1024 || json.Unmarshal([]byte(attempt.VerificationJSON), &facts) != nil || facts.Platform != app.Platform || facts.IdentityPartition != app.IdentityPartition || facts.Mode != connection.Mode || facts.RegionID != connection.RegionID || connection.CredentialExpiresAt == nil || !facts.ExpiresAt.Equal(*connection.CredentialExpiresAt) || !now.Before(facts.ExpiresAt) || facts.VerifiedAt.IsZero() || facts.VerifiedAt.After(now) || !authidentity.IsBoundedIdentifier(facts.ContractRevision) || facts.Scopes == nil || len(facts.Scopes) > 4096 {
		return MerchantBinding{}, false
	}
	for _, scope := range facts.Scopes {
		if !authidentity.IsBoundedIdentifier(scope) {
			return MerchantBinding{}, false
		}
	}
	identity, err := facts.MerchantIdentityHash()
	if err != nil || identity != connection.MerchantIdentityHash {
		return MerchantBinding{}, false
	}
	scopes := append([]string{}, facts.Scopes...)
	sort.Strings(scopes)
	encoded, _ := json.Marshal(scopes)
	scopesHash := sha256.Sum256(encoded)
	site := "temu-supplier"
	if facts.Mode == storecenter.MerchantLocal {
		site = "temu-local"
	}
	binding := MerchantBinding{Platform: app.Platform, IdentityPartition: app.IdentityPartition, MerchantMode: facts.Mode, RegionID: facts.RegionID, CredentialExpiresAt: facts.ExpiresAt.UTC(), VerificationRevision: facts.ContractRevision, ScopesHash: hex.EncodeToString(scopesHash[:]), OrganizationID: subject.OrganizationID, StoreID: storeID, Site: site, StoreVersion: m.StoreVersion, ConnectionRevision: connection.Version, ApplicationID: app.AppID, ApplicationRevision: app.Version, SupplierIdentityHash: identity, ServiceExpiresAt: m.ServiceExpiresAt.UTC()}
	return binding, binding.ValidTemuApplication()
}

func validTemuPrivateCredential(c storecenter.OfficialMerchantCredential) bool {
	if c.Temu == nil || c.OpenKeyID != "" || c.SecretKey != "" || c.SupplierID != "" {
		return false
	}
	token := c.Temu.AccessToken
	return token != "" && len(token) <= 4096 && utf8.ValidString(token) && strings.TrimSpace(token) == token && !strings.ContainsAny(token, "\r\n\x00")
}

func callTemuRule[T any](ctx context.Context, h *TemuRulesHandle, local bool, read func(context.Context, storecenter.OfficialMerchantCredential) (T, error)) (T, error) {
	var zero T
	if ctx == nil || h == nil || h.owner == nil || (h.binding.MerchantMode == storecenter.MerchantLocal) != local {
		return zero, ErrProductAccessChanged
	}
	ctx, cancel := context.WithDeadline(ctx, h.expiresAt)
	defer cancel()
	credential, err := h.credential(ctx)
	if err != nil {
		return zero, err
	}
	value, err := read(ctx, credential)
	if err != nil {
		return zero, err
	}
	if _, err = h.credential(ctx); err != nil {
		return zero, err
	}
	return value, nil
}

func (h *TemuRulesHandle) SupplierCategories(ctx context.Context, query model.SupplierCategoryQuery) ([]model.SupplierCategory, error) {
	return callTemuRule(ctx, h, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) ([]model.SupplierCategory, error) {
		return h.entry.provider.(TemuRulesProvider).QuerySupplierCategories(ctx, c, query)
	})
}
func (h *TemuRulesHandle) LocalCategories(ctx context.Context, query model.LocalCategoryQuery) ([]model.LocalCategory, error) {
	return callTemuRule(ctx, h, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) ([]model.LocalCategory, error) {
		return h.entry.provider.(TemuRulesProvider).QueryLocalCategories(ctx, c, query)
	})
}
func (h *TemuRulesHandle) SupplierAttributes(ctx context.Context, catID int64) (model.SupplierAttributeTemplate, error) {
	return callTemuRule(ctx, h, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.SupplierAttributeTemplate, error) {
		return h.entry.provider.(TemuRulesProvider).QuerySupplierAttributes(ctx, c, catID)
	})
}
func (h *TemuRulesHandle) LocalCategoryRules(ctx context.Context, query model.CategoryRuleQuery) (model.LocalCategoryRules, error) {
	return callTemuRule(ctx, h, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.LocalCategoryRules, error) {
		return h.entry.provider.(TemuRulesProvider).QueryLocalCategoryRules(ctx, c, query)
	})
}

func (h *TemuRulesHandle) LocalAttributes(ctx context.Context, query model.LocalAttributeQuery) (model.LocalAttributeTemplate, error) {
	return callTemuRule(ctx, h, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.LocalAttributeTemplate, error) {
		return h.entry.provider.(TemuRulesProvider).QueryLocalAttributes(ctx, c, query)
	})
}

func (h *TemuRulesHandle) LocalFreightTemplates(ctx context.Context) ([]model.LocalFreightTemplate, error) {
	return callTemuRule(ctx, h, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) ([]model.LocalFreightTemplate, error) {
		return h.entry.provider.(TemuRulesProvider).QueryLocalFreightTemplates(ctx, c)
	})
}
func (h *TemuRulesHandle) LocalTrademarks(ctx context.Context, query model.LocalTrademarkQuery) (model.LocalTrademarkPage, error) {
	return callTemuRule(ctx, h, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.LocalTrademarkPage, error) {
		return h.entry.provider.(TemuRulesProvider).QueryLocalTrademarks(ctx, c, query)
	})
}

func (h *TemuRulesHandle) LocalVariations(ctx context.Context, query model.CategoryRuleQuery) (model.LocalVariationTemplate, error) {
	return callTemuRule(ctx, h, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.LocalVariationTemplate, error) {
		return h.entry.provider.(TemuRulesProvider).QueryLocalVariations(ctx, c, query)
	})
}
