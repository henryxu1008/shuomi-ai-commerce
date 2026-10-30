package currentapplication

import (
	"errors"
	k "task-processor/internal/browserkernel"
)

type BrowserKernelConfig struct {
	Database        DatabaseConfig `json:"database"`
	ArtifactOrigins []string       `json:"artifactOrigins"`
}

func (c *Config) validateBrowserKernel() error {
	if c.BrowserKernel == nil {
		return nil
	}
	d := c.BrowserKernel.Database
	if e := d.validate("browserKernel.database"); e != nil {
		return e
	}
	if d.User != "browser_kernel_runtime" || d.Database != "browser_kernel" || d.MaxConnections > 4 {
		return errors.New("browser kernel requires its dedicated bounded runtime owner")
	}
	if e := k.ValidateOrigins(c.BrowserKernel.ArtifactOrigins); e != nil {
		return errors.New("browser kernel requires exact approved HTTPS artifact origins")
	}
	owners := []*DatabaseConfig{&c.SourceAccountDatabase, c.CommercialOwnerDatabase, c.MoneyOwnerDatabase, c.ProductAcquisitionDatabase, c.NotificationCenterDatabase, c.AgentCustomizationDatabase}
	if c.StoreCenter != nil {
		owners = append(owners, &c.StoreCenter.Database)
	}
	if c.Knowledge != nil {
		owners = append(owners, &c.Knowledge.Database)
	}
	if c.Membership != nil {
		owners = append(owners, &c.Membership.Database)
	}
	if c.Ecoservices != nil {
		owners = append(owners, &c.Ecoservices.Database)
	}
	if c.ToolMarket != nil {
		owners = append(owners, &c.ToolMarket.Database)
	}
	if c.ProjectCenter != nil {
		owners = append(owners, &c.ProjectCenter.Database)
	}
	if c.ReportCenter != nil {
		owners = append(owners, &c.ReportCenter.Database)
	}
	if c.DataServices != nil {
		owners = append(owners, &c.DataServices.Database)
	}
	if c.AIWorkbench != nil {
		owners = append(owners, &c.AIWorkbench.Database)
	}
	if c.ProductAgent != nil {
		owners = append(owners, &c.ProductAgent.Database, &c.ProductAgent.ReviewDatabase, &c.ProductAgent.AssetDatabase)
	}
	if c.ImageAgent != nil {
		owners = append(owners, &c.ImageAgent.Database, &c.ImageAgent.AssetDatabase)
	}
	if c.SupplyChain != nil {
		owners = append(owners, &c.SupplyChain.AssetDatabase)
	}
	if c.POD != nil {
		owners = append(owners, &c.POD.AssetDatabase)
	}
	if c.Referrals.Enabled {
		owners = append(owners, &c.Referrals.Database)
	}
	if c.LocalTrial != nil && c.LocalTrial.Enabled {
		owners = append(owners, &c.LocalTrial.Database)
	}
	if c.AccountAuditUsage != nil {
		owners = append(owners, &c.AccountAuditUsage.Image, &c.AccountAuditUsage.Product)
	}
	if c.ModelRoutePricing != nil {
		owners = append(owners, c.ModelRoutePricing.TextWriter, c.ModelRoutePricing.ImageWriter)
	}
	for _, other := range owners {
		if other != nil && sameDatabaseTarget(d, *other) {
			return errors.New("browser kernel cannot share an existing owner database")
		}
	}
	return nil
}
