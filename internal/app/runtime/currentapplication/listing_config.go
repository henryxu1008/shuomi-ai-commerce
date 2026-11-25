package currentapplication

import (
	"errors"
	"net"
	"reflect"
	"strconv"

	coreconfig "github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
)

// Supply uses the current Product and Store owners, the existing Asset owner,
// and the Temporal SDK. Presence is explicit admission; omitted stays closed.
type SupplyChainConfig struct {
	AssetDatabase           DatabaseConfig                            `json:"assetDatabase"`
	TemporalAddress         string                                    `json:"temporalAddress,omitempty"`
	TemporalNamespace       string                                    `json:"temporalNamespace,omitempty"`
	ShopifyPreviewArtifacts *coreconfig.ImageAgentArtifactStoreConfig `json:"shopifyPreviewArtifacts,omitempty"`
}

// SourceMaterials admits only original source approval and Product diagnostics.
// It consumes the existing Asset owner without Store, Agent or workflow workers.
type SourceMaterialsConfig struct {
	AssetDatabase DatabaseConfig `json:"assetDatabase"`
}

func (c *Config) validateSourceMaterials() error {
	s := c.SourceMaterials
	if s == nil {
		return nil
	}
	if !c.ProductCollections || !c.AcquisitionPlans || c.PrivateDraftTrial != nil || c.Identity.TenantDirectoryToken == "" {
		return errors.New("source materials require current collections, acquisition plans and live membership")
	}
	if err := s.AssetDatabase.validate("sourceMaterials.assetDatabase"); err != nil {
		return err
	}
	if s.AssetDatabase.User != "supply_asset_runtime" || s.AssetDatabase.MaxConnections > 8 {
		return errors.New("source materials require bounded supply_asset_runtime Asset access")
	}
	if err := c.validateAssetOwnerIsolation(s.AssetDatabase); err != nil {
		return err
	}
	if c.SupplyChain != nil && c.SupplyChain.AssetDatabase != s.AssetDatabase ||
		c.ImageAgent != nil && !sameDatabaseTarget(s.AssetDatabase, c.ImageAgent.Database) ||
		c.ImageAgent != nil && c.ImageAgent.WorkerConfigFile != "" && c.ImageAgent.AssetDatabase != s.AssetDatabase ||
		c.ProductAgent != nil && c.ProductAgent.Enabled && !c.ProductAgent.ContentTitleOnly && !c.ProductAgent.ContentOnly && !sameDatabaseTarget(s.AssetDatabase, c.ProductAgent.AssetDatabase) {
		return errors.New("source materials must share the current canonical Asset owner")
	}
	return nil
}

func (c *Config) validateListing() error {
	s := c.SupplyChain
	if s == nil {
		return nil
	}
	store := c.StoreCenter
	if !c.ProductCollections || c.ProductAcquisitionDatabase == nil || store == nil || !store.Enabled ||
		(len(store.OfficialApplications) == 0 && store.Shopify == nil) || c.Identity.TenantDirectoryToken == "" {
		return errors.New("supply chain requires collections, official Store applications and current membership authorization")
	}
	if err := s.AssetDatabase.validate("supplyChain.assetDatabase"); err != nil {
		return err
	}
	if s.AssetDatabase.User != "supply_asset_runtime" || s.AssetDatabase.MaxConnections > 8 {
		return errors.New("supply chain requires bounded supply_asset_runtime Asset access")
	}
	if err := c.validateAssetOwnerIsolation(s.AssetDatabase); err != nil {
		return err
	}
	if c.ImageAgent != nil && !sameDatabaseTarget(s.AssetDatabase, c.ImageAgent.Database) || c.ProductAgent != nil && c.ProductAgent.Enabled && !c.ProductAgent.ContentTitleOnly && !c.ProductAgent.ContentOnly && !sameDatabaseTarget(s.AssetDatabase, c.ProductAgent.AssetDatabase) {
		return errors.New("supply must use the current canonical Asset database")
	}
	if s.ShopifyPreviewArtifacts != nil {
		return c.validateShopifyPreview(s, store)
	}
	host, port, err := net.SplitHostPort(s.TemporalAddress)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return errors.New("supply Temporal address must use a literal loopback address")
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 || !boundedValue(s.TemporalNamespace, 128) {
		return errors.New("supply Temporal namespace or port invalid")
	}
	return nil
}

func (c *Config) ShopifyPreviewOnly() bool {
	return c.SupplyChain != nil && c.SupplyChain.ShopifyPreviewArtifacts != nil
}

func (c *Config) validateShopifyPreview(s *SupplyChainConfig, store *StoreCenterConfig) error {
	allowed := Config{
		SchemaVersion: c.SchemaVersion, Listen: c.Listen, Identity: c.Identity,
		SourceAccountDatabase: c.SourceAccountDatabase, CommercialOwnerDatabase: c.CommercialOwnerDatabase,
		StorageCapacity:            c.StorageCapacity,
		ProductAcquisitionDatabase: c.ProductAcquisitionDatabase, ProductCollections: c.ProductCollections,
		StoreCenter: store, SupplyChain: s, Membership: c.Membership, ListingKitAuthorization: c.ListingKitAuthorization,
	}
	if c.StorageCapacity {
		allowed.MoneyOwnerDatabase = c.MoneyOwnerDatabase
	}
	if !reflect.DeepEqual(*c, allowed) || !net.ParseIP(c.Listen.Host).IsLoopback() ||
		s.TemporalAddress != "" || s.TemporalNamespace != "" || !s.ShopifyPreviewArtifacts.IsolatedTrialGeneratedURLs {
		return errors.New("Shopify draft preview requires loopback and excludes workflow execution and unrelated modules")
	}
	shopify := store.Shopify
	allowedStore := StoreCenterConfig{Enabled: true, Database: store.Database, Shopify: shopify}
	if !reflect.DeepEqual(*store, allowedStore) || shopify == nil || shopify.StartedInstallation || shopify.LifecycleNotifications {
		return errors.New("Shopify draft preview requires only an existing Shopify installation")
	}
	return nil
}

func (c *Config) validateAssetOwnerIsolation(database DatabaseConfig) error {
	owners := []*DatabaseConfig{&c.SourceAccountDatabase, c.ProductAcquisitionDatabase, c.CommercialOwnerDatabase, c.MoneyOwnerDatabase, c.NotificationCenterDatabase}
	if c.StoreCenter != nil {
		owners = append(owners, &c.StoreCenter.Database)
	}
	if c.Ecoservices != nil && c.Ecoservices.Enabled {
		owners = append(owners, &c.Ecoservices.Database)
	}
	if c.ProductAgent != nil {
		owners = append(owners, &c.ProductAgent.Database, &c.ProductAgent.ReviewDatabase)
	}
	if c.AIWorkbench != nil {
		owners = append(owners, &c.AIWorkbench.Database)
	}
	if c.Membership != nil {
		owners = append(owners, &c.Membership.Database)
	}
	if c.Referrals.Enabled {
		owners = append(owners, &c.Referrals.Database)
	}
	for _, owner := range owners {
		if owner != nil && sameDatabaseTarget(database, *owner) {
			return errors.New("supply assets require their independently owned database")
		}
	}
	return nil
}
