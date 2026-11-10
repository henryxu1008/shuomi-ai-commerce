package currentapplication

import (
	"github.com/stretchr/testify/require"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/product/collection"
	"testing"
)

func localDraftConfig() *Config {
	c := acquisitionRuntimeConfig()
	c.Identity.IssuerURL = "https://localhost:18080"
	c.Identity.TenantDirectoryToken = "fixture"
	c.ProductCollections = true
	c.SupplyCompletionDrafts = true
	c.StoreCenter = &StoreCenterConfig{Enabled: true}
	c.ControlledLocalDraft = &ControlledLocalDraftConfig{Acknowledgment: "CONTROLLED_LOCAL_DRAFT_NO_EXECUTION", OrganizationID: "394513989062623234", ActorID: "394513839896395778", MemberID: "394513991663091714", SourceID: "41cb802a-fa17-523e-9444-12044f47e5dc", ProductKey: "own-27ef901c-30e0-57f9-81d9-56032b0efa61", CompletionRecordID: "f7853e11-353a-5214-a7c7-75a8ded6e278", CompletionRevision: 1, StoreID: "11111111-1111-4111-8111-111111111111", FixtureVersion: "issue1158-local-draft-v1", FixtureHash: collection.Digest(listingapp.ControlledLocalRules()), AssetDatabase: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "supply_asset_runtime", Password: "fixture", Database: "image_agent", MaxConnections: 2}}
	return c
}
func TestControlledLocalDraftAdmissionAndDisabledReads(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		c := localDraftConfig()
		c.ControlledLocalDraft.Enabled = enabled
		require.NoError(t, c.validateControlledLocalDraft())
		// These existing features must not become mutually exclusive with local Target.
		c.ImageAgent = &ImageAgentConfig{Database: c.ControlledLocalDraft.AssetDatabase}
		c.SupplyMarket = &SupplyMarketConfig{}
		require.NoError(t, c.validateControlledLocalDraft())
		c.SupplyChain = &SupplyChainConfig{AssetDatabase: c.ControlledLocalDraft.AssetDatabase}
		require.NoError(t, c.validateControlledLocalDraft(), "coexistence retains one canonical Asset pool")
		c.SupplyChain.AssetDatabase.Password = "different"
		require.Error(t, c.validateControlledLocalDraft(), "shared pool requires the same narrow credentials/config")
	}
	for name, mutate := range map[string]func(*Config){
		"ack":            func(c *Config) { c.ControlledLocalDraft.Acknowledgment = "" },
		"org":            func(c *Config) { c.ControlledLocalDraft.OrganizationID = "other" },
		"source":         func(c *Config) { c.ControlledLocalDraft.SourceID = "11111111-1111-4111-8111-111111111111" },
		"R1":             func(c *Config) { c.ControlledLocalDraft.CompletionRevision = 2 },
		"cockpit store":  func(c *Config) { c.ControlledLocalDraft.StoreID = "b6e4504f-c3db-4e1b-9c5f-2605dda4aea1" },
		"fixture":        func(c *Config) { c.ControlledLocalDraft.FixtureHash = "a" },
		"remote":         func(c *Config) { c.Identity.IssuerURL = "https://example.org" },
		"readonly other": func(c *Config) { c.PrivateDraftTrial = &PrivateDraftTrialConfig{} },
		"offline":        func(c *Config) { c.SheinOfflineRecords = &SheinOfflineRecordsConfig{} },
		"owner":          func(c *Config) { c.ControlledLocalDraft.AssetDatabase.User = "postgres" },
		"completion":     func(c *Config) { c.SupplyCompletionDrafts = false },
	} {
		t.Run(name, func(t *testing.T) {
			c := localDraftConfig()
			mutate(c)
			require.Error(t, c.validateControlledLocalDraft())
		})
	}
}
