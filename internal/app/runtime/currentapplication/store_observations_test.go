package currentapplication

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStoreObservationsConfigDoesNotRequireSupply(t *testing.T) {
	c := storeRuntimeConfig()
	// Validate only this opt-in contract; official material preparation remains
	// bounded by the existing independently tested application registry.
	c.Identity.TenantDirectoryToken = "synthetic-service-token"
	c.StoreCenter.OfficialApplications = []OfficialStoreConnectionConfig{{}}
	c.StoreCenter.Observations = &StoreObservationsConfig{TemporalAddress: "127.0.0.1:7233", TemporalNamespace: "default"}
	require.Nil(t, c.SupplyChain)
	require.NoError(t, c.validateStoreObservations())
	for _, mutate := range []func(*Config){
		func(c *Config) { c.StoreCenter.Enabled = false },
		func(c *Config) { c.StoreCenter.OfficialApplications = nil },
		func(c *Config) { c.Identity.TenantDirectoryToken = "" },
		func(c *Config) { c.StoreCenter.Observations.TemporalAddress = "example.com:7233" },
		func(c *Config) { c.StoreCenter.Observations.TemporalNamespace = "" },
	} {
		copy := *c
		store := *c.StoreCenter
		settings := *store.Observations
		copy.StoreCenter = &store
		store.Observations = &settings
		mutate(&copy)
		require.Error(t, copy.validateStoreObservations())
	}
	encoded, err := json.Marshal(c.StoreCenter)
	require.NoError(t, err)
	var decoded StoreCenterConfig
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, c.StoreCenter.Observations, decoded.Observations)
}

func TestStoreObservationsPreparationRequiresExplicitProviderFreeMode(t *testing.T) {
	c := storeRuntimeConfig()
	c.Identity.TenantDirectoryToken = "synthetic-service-token"
	require.NoError(t, json.Unmarshal([]byte(`{"temporalAddress":"127.0.0.1:7233","temporalNamespace":"default","preparationOnly":true}`), &c.StoreCenter.Observations))
	require.NoError(t, c.validateStoreObservations(), "explicit preparation admits no official application")
	c.StoreCenter.OfficialApplications = []OfficialStoreConnectionConfig{{}}
	require.Error(t, c.validateStoreObservations(), "configured apps cannot silently downgrade")
	c.StoreCenter.OfficialApplications = nil
	c.StoreCenter.Enabled = false
	require.Error(t, c.validateStoreObservations())
	c.StoreCenter.Enabled = true
	c.Identity.TenantDirectoryToken = ""
	require.Error(t, c.validateStoreObservations())
}
