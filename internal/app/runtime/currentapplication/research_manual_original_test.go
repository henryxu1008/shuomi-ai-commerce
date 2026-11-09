package currentapplication

import (
	"github.com/stretchr/testify/require"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/product/collection"
	"testing"
)

func TestResearchManualOriginalDefaultOffAndScopedAlongsideExecution(t *testing.T) {
	base := func() Config {
		return Config{
			ResearchManualOriginal: &ResearchManualOriginalConfig{Acknowledgment: "LOCAL_EXISTING_B000123457_MANUAL_ONLY", Binding: dataservicesapp.ResearchManualOriginalBinding{
				Scope: collection.Scope{OrganizationID: "394513989062623234", ActorID: "394513839896395778", MemberID: "394513991663091714"}, Project: "e83d6399-1e20-4fac-ab34-93b698203602", CandidateID: "6768ffeb-490f-5616-9905-bac1f0530d8e", LinkID: "5fa57a3f-f91a-51e0-93a2-e017c8aac2e9",
				ItemID: "32f9a7c2-4345-5c28-a72e-cb75f3525c9d", ItemRevision: 1, PublicationID: "ae0aeb8c-75c7-51c4-bfdc-b696ea62c62b", CatalogVersion: 1,
				Checksum: "sha256:898b11ebf7e631565d56d3f73fbf5ed1739397f64109623b338398a44f8afaf2", EnvelopeHash: "20276049f16dc63b5c1e5957f3a09e980046ea29ad9ee8f1545b5a8bfd1462e1", SnapshotHash: "01cf35754a0d5b17b79f21a79878ec3bfb18736b0ce4a88d8cf525342e0216f5",
			}},
			Identity:       IdentityConfig{IssuerURL: "https://localhost:35444", TenantDirectoryToken: "synthetic-only"},
			MarketResearch: &ResearchConfig{SupplierReview: true}, DataServices: &DataServicesConfig{ExecutionEnabled: true}, ProductCollections: true, ProductAcquisitionDatabase: &DatabaseConfig{},
			ProductAgent: &ProductAgentConfig{Enabled: true},
		}
	}
	c := base()
	require.NoError(t, c.validateResearchManualOriginal())
	for name, change := range map[string]func(*Config){
		"missing acknowledgment": func(c *Config) { c.ResearchManualOriginal.Acknowledgment = "" },
		"wrong member":           func(c *Config) { c.ResearchManualOriginal.Binding.Scope.MemberID = "other" },
		"wrong source": func(c *Config) {
			c.ResearchManualOriginal.Binding.PublicationID = "ae0aeb8c-75c7-51c4-bfdc-b696ea62c62c"
		},
		"missing checksum":         func(c *Config) { c.ResearchManualOriginal.Binding.Checksum = "" },
		"missing hash":             func(c *Config) { c.ResearchManualOriginal.Binding.SnapshotHash = "" },
		"nonlocal":                 func(c *Config) { c.Identity.IssuerURL = "https://id.example.com" },
		"without native directory": func(c *Config) { c.Identity.TenantDirectoryToken = "" },
		"without manual domain":    func(c *Config) { c.MarketResearch.SupplierReview = false },
		"without source pool":      func(c *Config) { c.ProductAcquisitionDatabase = nil },
		"full fixture":             func(c *Config) { c.ResearchFixture = &ResearchFixtureConfig{} },
	} {
		t.Run(name, func(t *testing.T) {
			c := base()
			change(&c)
			require.Error(t, c.validateResearchManualOriginal())
			c.ResearchManualOriginal = nil
			require.NoError(t, c.validateResearchManualOriginal())
		})
	}
}
