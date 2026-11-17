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
	issuer, err := validateIdentityURL("issuerURL", c.Identity.IssuerURL)
	if err != nil || issuer.Scheme != "https" || c.Identity.TenantDirectoryToken == "" || issuer.Port() == "" && c.Identity.AuthorizationAPIURL != c.Identity.IssuerURL {
		return errors.New("manual research requires admitted HTTPS native identity")
	}
	if c.MarketResearch == nil || !c.MarketResearch.SupplierReview || !c.ProductCollections || c.ProductAcquisitionDatabase == nil || c.DataServices == nil {
		return errors.New("manual research requires the existing research, collection and source owners")
	}
	return nil
}
