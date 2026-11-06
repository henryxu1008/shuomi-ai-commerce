package currentapplication

import "errors"

type ResearchConfig struct {
	SupplierReview bool           `json:"supplierReview,omitempty"`
	Database       DatabaseConfig `json:"database"`
}

func (c *Config) validateResearch() error {
	d := c.MarketResearch
	if d == nil {
		return nil
	}
	if c.ProjectCenter == nil || c.DataServices == nil || !c.ProductCollections || c.ProductAcquisitionDatabase == nil {
		return errors.New("market research requires projects and current Amazon collections")
	}
	if e := d.Database.validate("marketResearch.database"); e != nil {
		return e
	}
	if d.Database.User != "product_research_runtime" || d.Database.MaxConnections > 4 || !sameDatabaseTarget(d.Database, *c.ProductAcquisitionDatabase) {
		return errors.New("market research requires its restricted pool in the current Product database")
	}
	return nil
}
