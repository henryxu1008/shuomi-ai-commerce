package storecenterapp

import (
	"sort"
	"strconv"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
	walmart "task-processor/internal/storecenter/walmart"
)

type OfficialApplicationRegistration struct {
	Provider   storecenter.OfficialConnectionProvider
	Protection storecenter.OfficialCredentialProtection
	Type       storecenter.OfficialApplicationType
}
type officialApplicationEntry struct {
	application storecenter.OfficialApplication
	provider    storecenter.OfficialConnectionProvider
	protection  storecenter.OfficialCredentialProtection
	mode        storecenter.OfficialApplicationType
}
type OfficialApplicationChoice = storecenter.OfficialApplicationChoice
type OfficialApplicationRegistry struct {
	entries           map[officialApplicationKey]officialApplicationEntry
	tiktokProvider    tiktok.SellerAuthorization
	tiktokProtection  tiktok.CredentialProtection
	walmartProvider   walmart.Authorization
	walmartProtection walmart.CredentialProtection
}

// TikTok uses a canonical seller grant and its own credential contract.
// Registration shares composition with other platforms, never their tokens.
func WithTikTokApplication(r *OfficialApplicationRegistry, provider tiktok.SellerAuthorization, protection tiktok.CredentialProtection) (*OfficialApplicationRegistry, error) {
	if provider == nil || protection == nil || (r != nil && r.tiktokProvider != nil) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	app := provider.Application()
	if !authidentity.IsBoundedIdentifier(app.AppKey) || !authidentity.IsBoundedIdentifier(app.Revision) {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	result := &OfficialApplicationRegistry{entries: map[officialApplicationKey]officialApplicationEntry{}, tiktokProvider: provider, tiktokProtection: protection}
	if r != nil {
		result.walmartProvider = r.walmartProvider
		result.walmartProtection = r.walmartProtection
		for key, entry := range r.entries {
			result.entries[key] = entry
		}
	}
	return result, nil
}

func WithWalmartApplication(r *OfficialApplicationRegistry, p walmart.Authorization, protection walmart.CredentialProtection) (*OfficialApplicationRegistry, error) {
	if p == nil || protection == nil || r != nil && r.walmartProvider != nil {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	app := p.Application()
	if !authidentity.IsBoundedIdentifier(app.ClientID) || !authidentity.IsBoundedIdentifier(app.Revision) || app.Environment != "production" && app.Environment != "sandbox" {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	out := &OfficialApplicationRegistry{entries: map[officialApplicationKey]officialApplicationEntry{}, walmartProvider: p, walmartProtection: protection}
	if r != nil {
		out.tiktokProvider = r.tiktokProvider
		out.tiktokProtection = r.tiktokProtection
		for k, e := range r.entries {
			out.entries[k] = e
		}
	}
	return out, nil
}
func (r *OfficialApplicationRegistry) WalmartConfigured() bool {
	return r != nil && r.walmartProvider != nil
}

func (r *OfficialApplicationRegistry) TikTokConfigured() bool {
	return r != nil && r.tiktokProvider != nil
}

func newTikTokConnections(store storecenter.OfficialConnectionStore, registry *OfficialApplicationRegistry) (*TikTokConnections, error) {
	repo, ok := store.(storecenter.TikTokConnectionStore)
	if !ok {
		return nil, nil
	}
	if !registry.TikTokConfigured() {
		return NewTikTokConnections(repo, nil, nil)
	}
	return NewTikTokConnections(repo, registry.tiktokProvider, registry.tiktokProtection)
}

type officialApplicationKey struct {
	platform  storecenter.Platform
	partition storecenter.OfficialIdentityPartition
	id        string
}

// Including mode in the persisted revision makes a configuration type change
// invalidate every previously connected attempt instead of changing its rules.
func BoundOfficialRevision(version string, mode storecenter.OfficialApplicationType) string {
	return version + ":" + string(mode)
}

// The fixed r1 contract includes identity queries and their CN/PA or regional
// routes. Mode is verified per merchant; a CN application supports both modes.
func BoundTemuOfficialRevision(version string, partition storecenter.OfficialIdentityPartition) string {
	modes := storecenter.ModesLocal
	if partition == storecenter.PartitionCN {
		modes = storecenter.ModesSemiManaged | storecenter.ModesFullyManaged
	}
	return version + ":temu:" + string(partition) + ":manual_token:" + strconv.Itoa(int(modes)) + ":r1"
}
func NewOfficialApplicationRegistry(registrations []OfficialApplicationRegistration) (*OfficialApplicationRegistry, error) {
	if len(registrations) < 1 || len(registrations) > 16 {
		return nil, storecenter.ErrOfficialConnectionUnavailable
	}
	r := &OfficialApplicationRegistry{entries: map[officialApplicationKey]officialApplicationEntry{}}
	for _, registration := range registrations {
		if registration.Provider == nil || registration.Protection == nil {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		app := registration.Provider.Application()
		if !validRegisteredApplication(app, registration.Type) {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		key := officialApplicationKey{app.Platform, app.IdentityPartition, app.AppID}
		if _, duplicate := r.entries[key]; duplicate {
			return nil, storecenter.ErrOfficialConnectionUnavailable
		}
		r.entries[key] = officialApplicationEntry{application: app, provider: registration.Provider, protection: registration.Protection, mode: registration.Type}
	}
	return r, nil
}
func validRegisteredApplication(app storecenter.OfficialApplication, mode storecenter.OfficialApplicationType) bool {
	if !authidentity.IsBoundedIdentifier(app.Version) || !authidentity.IsBoundedIdentifier(app.AppID) {
		return false
	}
	if app.Platform == storecenter.PlatformShein {
		base, ok := strings.CutSuffix(app.Version, ":"+string(mode))
		return mode.Valid() && ok && authidentity.IsBoundedIdentifier(base) && app.IdentityPartition == "" && app.AuthorizationMethod == storecenter.AuthorizationCallback && app.SupportedModes == 0
	}
	if app.Platform != storecenter.PlatformTemu || mode != "" || app.AuthorizationMethod != storecenter.AuthorizationManualToken || app.CallbackURL != "" {
		return false
	}
	modes := storecenter.ModesLocal
	switch app.IdentityPartition {
	case storecenter.PartitionCN:
		modes = storecenter.ModesSemiManaged | storecenter.ModesFullyManaged
	case storecenter.PartitionUS, storecenter.PartitionEU, storecenter.PartitionGlobal:
	default:
		return false
	}
	suffix := strings.TrimPrefix(BoundTemuOfficialRevision("v", app.IdentityPartition), "v")
	base, ok := strings.CutSuffix(app.Version, suffix)
	return ok && authidentity.IsBoundedIdentifier(base) && app.SupportedModes == modes
}
func (r *OfficialApplicationRegistry) resolve(id, revision string, platform storecenter.Platform, partition storecenter.OfficialIdentityPartition) (officialApplicationEntry, error) {
	if r == nil {
		return officialApplicationEntry{}, storecenter.ErrOfficialConnectionUnavailable
	}
	entry, ok := r.entries[officialApplicationKey{platform, partition, id}]
	if !ok || entry.application.Version != revision || entry.provider.Application() != entry.application {
		return officialApplicationEntry{}, storecenter.ErrOfficialConnectionUnavailable
	}
	return entry, nil
}
func (r *OfficialApplicationRegistry) selectApplication(id string, platform storecenter.Platform, partition storecenter.OfficialIdentityPartition) (officialApplicationEntry, error) {
	if r == nil {
		return officialApplicationEntry{}, storecenter.ErrOfficialConnectionUnavailable
	}
	entry, ok := r.entries[officialApplicationKey{platform, partition, id}]
	if !ok {
		return officialApplicationEntry{}, storecenter.ErrOfficialConnectionUnavailable
	}
	return r.resolve(id, entry.application.Version, platform, partition)
}
func (r *OfficialApplicationRegistry) Applications() []OfficialApplicationChoice {
	choices := []OfficialApplicationChoice{}
	if r == nil {
		return choices
	}
	for _, entry := range r.entries {
		if _, err := r.resolve(entry.application.AppID, entry.application.Version, entry.application.Platform, entry.application.IdentityPartition); err == nil {
			choices = append(choices, OfficialApplicationChoice{Platform: entry.application.Platform, IdentityPartition: entry.application.IdentityPartition, AuthorizationMethod: entry.application.AuthorizationMethod, SupportedModes: entry.application.SupportedModes, AppID: entry.application.AppID, Revision: entry.application.Version, Type: entry.mode})
		}
	}
	sort.Slice(choices, func(i, j int) bool {
		a, b := choices[i], choices[j]
		if a.Platform != b.Platform {
			return a.Platform < b.Platform
		}
		if a.IdentityPartition != b.IdentityPartition {
			return a.IdentityPartition < b.IdentityPartition
		}
		return a.AppID < b.AppID
	})
	return choices
}
