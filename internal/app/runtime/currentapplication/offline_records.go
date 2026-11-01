package currentapplication

import "errors"

// Presence opts in; no worker, provider or retained-database feature upgrade.
type SheinOfflineRecordsConfig struct {
	AssetDatabase DatabaseConfig `json:"assetDatabase"`
}

func (c *Config) validateOfflineRecords() error {
	s := c.SheinOfflineRecords
	if s == nil {
		return nil
	}
	if !c.ProductCollections || c.ProductAcquisitionDatabase == nil || c.ReportCenter == nil || c.StoreCenter == nil || !c.StoreCenter.Enabled || c.Identity.TenantDirectoryToken == "" || c.LocalTrial != nil || c.PrivateDraftTrial != nil {
		return errors.New("offline SHEIN records require normal Product/Collection, Store, Report and live membership")
	}
	d := s.AssetDatabase
	if err := d.validate("sheinOfflineRecords.assetDatabase"); err != nil {
		return err
	}
	if d.User != "supply_asset_runtime" || d.MaxConnections > 8 {
		return errors.New("offline SHEIN requires the bounded current Asset role")
	}
	if err := c.validateAssetOwnerIsolation(d); err != nil {
		return err
	}
	if sameDatabaseTarget(d, c.ReportCenter.Database) {
		return errors.New("offline Asset and Report owners must be independent")
	}
	if c.SourceMaterials != nil && c.SourceMaterials.AssetDatabase != d || c.SupplyChain != nil && c.SupplyChain.AssetDatabase != d || c.ImageAgent != nil && c.ImageAgent.WorkerConfigFile != "" && c.ImageAgent.AssetDatabase != d || c.ProductAgent != nil && c.ProductAgent.Enabled && !c.ProductAgent.ContentTitleOnly && !c.ProductAgent.ContentOnly && !sameDatabaseTarget(d, c.ProductAgent.AssetDatabase) {
		return errors.New("offline SHEIN must use the canonical current Asset owner")
	}
	return nil
}
