package currentapplication

import (
	"errors"

	"task-processor/internal/product/collection"
)

type ResearchFixtureConfig struct {
	Acknowledgment    string `json:"acknowledgment"`
	OrganizationID    string `json:"organizationId"`
	ActorID           string `json:"actorId"`
	MemberID          string `json:"memberId"`
	ProjectID         string `json:"projectId"`
	PaginationEnabled bool   `json:"paginationEnabled,omitempty"`
}

func (c *Config) validateResearchFixture() error {
	f := c.ResearchFixture
	if f == nil {
		return nil
	}
	scope := collection.Scope{OrganizationID: f.OrganizationID, ActorID: f.ActorID, MemberID: f.MemberID}
	if f.Acknowledgment != "LOCAL_SYNTHETIC_RESEARCH_W1_ONLY" || scope.Validate() != nil || !collection.ValidID(f.ProjectID) {
		return errors.New("research fixture requires one explicit local scope and project")
	}
	issuer, err := validateLoopbackURL("issuerURL", c.Identity.IssuerURL)
	if err != nil || issuer.Scheme != "https" || c.Identity.TenantDirectoryToken == "" {
		return errors.New("research fixture requires loopback HTTPS identity")
	}
	if c.MarketResearch == nil || !c.MarketResearch.SupplierReview || c.DataServices == nil || c.DataServices.ExecutionEnabled ||
		!c.ProductCollections || c.ProductAcquisitionDatabase == nil || c.SupplyChain != nil || c.SheinUSTrialInputDir != nil ||
		c.SourceMaterials != nil || c.POD != nil || c.TitlePilot != nil ||
		c.AIWorkbench != nil && c.AIWorkbench.Enabled || c.ProductAgent != nil && c.ProductAgent.Enabled ||
		c.ImageAgent != nil && c.ImageAgent.Generation.Enabled ||
		c.StoreCenter != nil && len(c.StoreCenter.OfficialApplications) != 0 {
		return errors.New("research fixture requires the current data owners and disabled execution")
	}
	return nil
}
