package storecenterapp

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	"testing"
)

type registryProvider struct {
	storecenter.OfficialConnectionProvider
	app storecenter.OfficialApplication
}

func TestOfficialRegistrySeparatesSameAppIDAcrossPlatformAndIdentityPartitions(t *testing.T) {
	entries := []OfficialApplicationRegistration{{Provider: &registryProvider{app: storecenter.OfficialApplication{Platform: storecenter.PlatformShein, AuthorizationMethod: storecenter.AuthorizationCallback, AppID: "same-id", Version: BoundOfficialRevision("v1", storecenter.ApplicationSelfOperated)}}, Protection: productSecretProtection{}, Type: storecenter.ApplicationSelfOperated}}
	for _, partition := range []storecenter.OfficialIdentityPartition{storecenter.PartitionCN, storecenter.PartitionUS, storecenter.PartitionEU, storecenter.PartitionGlobal} {
		modes := storecenter.ModesLocal
		if partition == storecenter.PartitionCN {
			modes = storecenter.ModesFullyManaged | storecenter.ModesSemiManaged
		}
		entries = append(entries, OfficialApplicationRegistration{Provider: &registryProvider{app: storecenter.OfficialApplication{Platform: storecenter.PlatformTemu, IdentityPartition: partition, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: modes, AppID: "same-id", Version: BoundTemuOfficialRevision("v1", partition)}}, Protection: productSecretProtection{}})
	}
	r, err := NewOfficialApplicationRegistry(entries)
	require.NoError(t, err)
	require.Len(t, r.Applications(), 5)
	for _, registration := range entries {
		a := registration.Provider.Application()
		entry, err := r.resolve(a.AppID, a.Version, a.Platform, a.IdentityPartition)
		require.NoError(t, err)
		require.Equal(t, registration.Provider, entry.provider)
	}
	_, err = r.resolve("same-id", BoundTemuOfficialRevision("v1", storecenter.PartitionEU), storecenter.PlatformTemu, storecenter.PartitionUS)
	require.Error(t, err)
	_, err = NewOfficialApplicationRegistry(append(entries, entries[1]))
	require.Error(t, err)
}

func (p *registryProvider) Application() storecenter.OfficialApplication { return p.app }
func TestOfficialRegistryPinsApplicationTypeAndDoesNotExposeProviderConfiguration(t *testing.T) {
	entries := []OfficialApplicationRegistration{}
	providers := []*registryProvider{}
	for _, mode := range []storecenter.OfficialApplicationType{storecenter.ApplicationSelfOperated, storecenter.ApplicationSemiManaged, storecenter.ApplicationFullyManaged} {
		p := &registryProvider{app: storecenter.OfficialApplication{Platform: storecenter.PlatformShein, AuthorizationMethod: storecenter.AuthorizationCallback, AppID: string(mode), Version: BoundOfficialRevision("v1", mode), CallbackURL: "https://private.example/callback"}}
		providers = append(providers, p)
		entries = append(entries, OfficialApplicationRegistration{Provider: p, Protection: productSecretProtection{}, Type: mode})
	}
	registry, err := NewOfficialApplicationRegistry(entries)
	require.NoError(t, err)
	public, err := json.Marshal(registry.Applications())
	require.NoError(t, err)
	require.NotContains(t, string(public), "private.example")
	require.NotContains(t, string(public), "Protection")
	for _, p := range providers {
		entry, err := registry.resolve(p.app.AppID, p.app.Version, p.app.Platform, p.app.IdentityPartition)
		require.NoError(t, err)
		require.Equal(t, p, entry.provider)
	}
	_, err = registry.resolve(providers[0].app.AppID, providers[1].app.Version, storecenter.PlatformShein, "")
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	providers[0].app.Version = BoundOfficialRevision("v1", storecenter.ApplicationFullyManaged)
	_, err = registry.resolve(string(storecenter.ApplicationSelfOperated), BoundOfficialRevision("v1", storecenter.ApplicationSelfOperated), storecenter.PlatformShein, "")
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	_, err = NewOfficialApplicationRegistry(append(entries, entries[1]))
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	unbound := &registryProvider{app: storecenter.OfficialApplication{AppID: "bad", Version: "v1"}}
	_, err = NewOfficialApplicationRegistry([]OfficialApplicationRegistration{{Provider: unbound, Protection: productSecretProtection{}, Type: storecenter.ApplicationSelfOperated}})
	require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
}
