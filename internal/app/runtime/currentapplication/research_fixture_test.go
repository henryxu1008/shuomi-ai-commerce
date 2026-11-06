package currentapplication

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResearchFixtureRequiresClosedLocalComposition(t *testing.T) {
	base := func() Config {
		return Config{
			ResearchFixture: &ResearchFixtureConfig{
				Acknowledgment: "LOCAL_SYNTHETIC_RESEARCH_W1_ONLY",
				OrganizationID: "test-org", ActorID: "test-user", MemberID: "test-member",
				ProjectID: "12345678-1234-4234-8234-123456789abc",
			},
			Identity:       IdentityConfig{IssuerURL: "https://localhost:35444", TenantDirectoryToken: "test-only"},
			MarketResearch: &ResearchConfig{SupplierReview: true},
			DataServices:   &DataServicesConfig{}, ProductCollections: true, ProductAcquisitionDatabase: &DatabaseConfig{},
		}
	}
	cfg := base()
	require.NoError(t, cfg.validateResearchFixture())
	for name, open := range map[string]func(*Config){
		"Amazon":          func(c *Config) { c.DataServices.ExecutionEnabled = true },
		"AI":              func(c *Config) { c.AIWorkbench = &AIWorkbenchConfig{Enabled: true} },
		"listing":         func(c *Config) { c.SupplyChain = &SupplyChainConfig{} },
		"nonlocal issuer": func(c *Config) { c.Identity.IssuerURL = "https://identity.example.com:35444" },
	} {
		t.Run(name, func(t *testing.T) {
			cfg := base()
			open(&cfg)
			require.Error(t, cfg.validateResearchFixture())
			cfg.ResearchFixture = nil
			require.NoError(t, cfg.validateResearchFixture())
		})
	}
}
