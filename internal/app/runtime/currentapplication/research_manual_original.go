package currentapplication

import (
	"errors"
	dataservicesapp "task-processor/internal/app/dataservices"
)

type ResearchManualOriginalConfig struct {
	Acknowledgment string                                        `json:"acknowledgment"`
	Binding        dataservicesapp.ResearchManualOriginalBinding `json:"binding"`
}

func (c *Config) validateResearchManualOriginal() error {
	f := c.ResearchManualOriginal
	if f == nil {
		return nil
	}
	if f.Acknowledgment != "LOCAL_EXISTING_B000123457_MANUAL_ONLY" || f.Binding.Validate() != nil || c.ResearchFixture != nil {
		return errors.New("manual research requires one frozen B000123457 binding without full fixture mode")
	}
	issuer, err := validateLoopbackURL("issuerURL", c.Identity.IssuerURL)
	if err != nil || issuer.Scheme != "https" || c.Identity.TenantDirectoryToken == "" {
		return errors.New("manual research requires loopback HTTPS native identity")
	}
	if c.MarketResearch == nil || !c.MarketResearch.SupplierReview || !c.ProductCollections || c.ProductAcquisitionDatabase == nil || c.DataServices == nil {
		return errors.New("manual research requires the existing research, collection and source owners")
	}
	return nil
}
