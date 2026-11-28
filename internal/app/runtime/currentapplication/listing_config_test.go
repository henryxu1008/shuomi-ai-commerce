package currentapplication

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	coreconfig "github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"gorm.io/gorm"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSupplyManifestRejectsIncompleteOwnerAssembly(t *testing.T) {
	for _, feature := range []string{
		`{}`, `{ "assetDatabase": {"host":"127.0.0.1","port":5432,"user":"supply_asset_runtime","password":"fixture","database":"assets","maxConnections":2}, "temporalAddress":"127.0.0.1:7233", "temporalNamespace":"default" }`,
	} {
		cfg := acquisitionRuntimeConfig()
		encoded, err := json.Marshal(cfg)
		require.NoError(t, err)
		var manifest map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(encoded, &manifest))
		manifest["supplyChain"] = json.RawMessage(feature)
		encoded, err = json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(encoded, cfg))
		require.Error(t, cfg.validate(), "supply cannot serve without collections, Store applications and live membership credentials")
	}
}

func TestSupplyManifestAdmitsShopifyWithoutUnrelatedStoreApps(t *testing.T) {
	c := supplyRuntimeConfig(t)
	c.StoreCenter.OfficialApplications = nil
	require.ErrorContains(t, c.validateListing(), "official Store applications")
	root := t.TempDir()
	c.StoreCenter.Shopify = &ShopifyConfig{
		ApplicationGID: "gid://shopify/App/123", ClientID: "fixture-app",
		InstallURL: "https://apps.shopify.com/fixture-app", CallbackURL: "https://localhost/api/shopify/callback",
		ClientSecret: filepath.Join(root, "client"), PairingKey: filepath.Join(root, "pairing"),
		SigningKey: filepath.Join(root, "signing"), PairingKeyID: "pairing-1", SigningKeyID: "signing-1",
	}
	require.NoError(t, c.validateListing())
	require.NoError(t, c.validateStoreCenter())
}

func TestShopifyPreviewManifestHasNoExecutionConfiguration(t *testing.T) {
	c := supplyRuntimeConfig(t)
	root := t.TempDir()
	c.StoreCenter.OfficialApplications = nil
	c.StoreCenter.Shopify = &ShopifyConfig{
		ApplicationGID: "gid://shopify/App/123", ClientID: "fixture-app",
		InstallURL: "https://apps.shopify.com/fixture-app", CallbackURL: "https://localhost/api/shopify/callback",
		ClientSecret: filepath.Join(root, "client"), PairingKey: filepath.Join(root, "pairing"),
		SigningKey: filepath.Join(root, "signing"), PairingKeyID: "pairing-1", SigningKeyID: "signing-1",
	}
	c.SupplyChain.TemporalAddress = ""
	c.SupplyChain.TemporalNamespace = ""
	data, err := json.Marshal(c)
	require.NoError(t, err)
	var manifest map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &manifest))
	var supply map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(manifest["supplyChain"], &supply))
	supply["shopifyPreviewArtifacts"] = json.RawMessage(`{"enabled":true,"provider":"s3","publicBase":"https://localhost:19444/image-agent-assets/preview","isolatedTrialGeneratedURLs":true,"s3":{"region":"us-east-1","endpoint":"http://127.0.0.1:9104","bucket":"preview","accessKeyID":"fixture","secretAccessKey":"fixture","usePathStyle":true,"artifactMode":"aws"}}`)
	manifest["supplyChain"], err = json.Marshal(supply)
	require.NoError(t, err)
	data, err = json.Marshal(manifest)
	require.NoError(t, err)
	preview, err := DecodeConfigDocument(data)
	require.NoError(t, err)
	require.NoError(t, preview.validate())
	for _, mutate := range []func(*Config){
		func(c *Config) { c.SupplyChain.TemporalAddress = "127.0.0.1:7233" },
		func(c *Config) { c.ImageAgent = &ImageAgentConfig{} },
		func(c *Config) { c.StoreCenter.Shopify.StartedInstallation = true },
		func(c *Config) { c.Listen.Host = "0.0.0.0" },
	} {
		preview, err := DecodeConfigDocument(data)
		require.NoError(t, err)
		mutate(preview)
		require.Error(t, preview.validateListing())
	}
}

func TestShopifyPairingTrialManifestExcludesNonPairingCapabilities(t *testing.T) {
	root := t.TempDir()
	base := supplyRuntimeConfig(t)
	store := StoreCenterConfig{Enabled: true, Database: base.StoreCenter.Database, Shopify: &ShopifyConfig{
		ApplicationGID: "gid://shopify/App/123", ClientID: "fixture-app",
		InstallURL: "https://admin.shopify.com/store/test/apps/fixture-app", CallbackURL: "https://127.0.0.1:19444/api/shopify/callback",
		ClientSecret: filepath.Join(root, "secret"), PairingKey: filepath.Join(root, "pairing"), SigningKey: filepath.Join(root, "signing"),
		PairingKeyID: "pairing-1", SigningKeyID: "signing-1",
	}}
	cfg := Config{
		ShopifyPairingTrial: true, SchemaVersion: base.SchemaVersion, Listen: base.Listen, Identity: base.Identity,
		SourceAccountDatabase: base.SourceAccountDatabase, CommercialOwnerDatabase: base.CommercialOwnerDatabase, StoreCenter: &store,
	}
	require.NoError(t, cfg.validateShopifyPairingTrial())
	denied := cfg
	denied.SupplyChain = base.SupplyChain
	require.Error(t, denied.validateShopifyPairingTrial())
	store.Shopify.LifecycleNotifications = true
	require.Error(t, cfg.validateShopifyPairingTrial())
	store.Shopify.LifecycleNotifications = false
	store.Shopify.CallbackURL = "https://localhost:19444/api/shopify/callback"
	require.Error(t, cfg.validateShopifyPairingTrial())
}

func TestShopifyMediaTrialManifestPinsFixtureAndRejectsUnsafeComposition(t *testing.T) {
	root := t.TempDir()
	base := supplyRuntimeConfig(t)
	store := StoreCenterConfig{Enabled: true, Database: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "store_center_runtime", Password: "fixture", Database: "media_store", MaxConnections: 2}, Shopify: &ShopifyConfig{
		ApplicationGID: "gid://shopify/App/451916529665", ClientID: "fixture-app",
		InstallURL: "https://admin.shopify.com/store/test/apps/fixture-app", CallbackURL: "https://127.0.0.1:36528/api/shopify/callback",
		ClientSecret: filepath.Join(root, "secret"), PairingKey: filepath.Join(root, "pairing"), SigningKey: filepath.Join(root, "signing"),
		PairingKeyID: "pairing-1", SigningKeyID: "signing-1", LifecycleNotifications: true,
	}}
	productInit := DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "acquisition_owner", Password: "fixture", Database: "media_product", MaxConnections: 2}
	trial := &ShopifyMediaTrialConfig{
		ProductInitializer: productInit,
		ProductServing:     DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "source_acquisition_runtime", Password: "fixture", Database: "media_product", MaxConnections: 4},
		AssetDatabase:      DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "supply_asset_runtime", Password: "fixture", Database: "media_assets", MaxConnections: 4},
		OrganizationID:     "org-test", ActorID: "actor-test", MemberID: "member-test", SourceID: "source-test", ProductKey: "own-test", SourceVersion: 1,
		ApprovedDigest: strings.Repeat("a", 64), StoreID: "store-test", ShopKey: "shop-key-test", Generation: 1,
		ApplicationGID: store.Shopify.ApplicationGID, ShopGID: "gid://shopify/Shop/85113995426", InstallationGID: "gid://shopify/AppInstallation/783089041570",
		Domain: "shuomi-upload-test-1613.myshopify.com", DraftProductGID: "gid://shopify/Product/123456",
		GalleryDigest: strings.Repeat("b", 64), AppendDigest: strings.Repeat("c", 64),
		ServiceStartsAt: time.Now().UTC(), ServiceExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	cfg := Config{SchemaVersion: base.SchemaVersion, Listen: ListenConfig{Host: "127.0.0.1", Port: 36529}, Identity: base.Identity,
		SourceAccountDatabase: base.SourceAccountDatabase, CommercialOwnerDatabase: base.CommercialOwnerDatabase,
		StoreCenter: &store, ShopifyMediaTrial: trial}
	require.NoError(t, cfg.validateShopifyMediaTrial())

	for name, mutate := range map[string]func(*Config){
		"normal modules excluded": func(c *Config) { c.SupplyChain = base.SupplyChain },
		"pairing mode excluded":   func(c *Config) { c.ShopifyPairingTrial = true },
		"source copy rejected":    func(c *Config) { c.ShopifyMediaTrial.SourceID = "" },
		"original coat rejected":  func(c *Config) { c.ShopifyMediaTrial.DraftProductGID = "gid://shopify/Product/10419372589218" },
		"database collision":      func(c *Config) { c.ShopifyMediaTrial.AssetDatabase.Database = c.StoreCenter.Database.Database },
		"expired fixture":         func(c *Config) { c.ShopifyMediaTrial.ServiceExpiresAt = c.ShopifyMediaTrial.ServiceStartsAt },
		"long fixture": func(c *Config) {
			c.ShopifyMediaTrial.ServiceExpiresAt = c.ShopifyMediaTrial.ServiceStartsAt.Add(25 * time.Hour)
		},
		"bad digest": func(c *Config) { c.ShopifyMediaTrial.GalleryDigest = "not-a-digest" },
	} {
		t.Run(name, func(t *testing.T) {
			copy := cfg
			trialCopy := *trial
			copy.ShopifyMediaTrial = &trialCopy
			storeCopy := *cfg.StoreCenter
			shopifyCopy := *cfg.StoreCenter.Shopify
			storeCopy.Shopify = &shopifyCopy
			copy.StoreCenter = &storeCopy
			mutate(&copy)
			require.Error(t, copy.validateShopifyMediaTrial())
		})
	}
}

func TestSupplyRuntimePassesCanonicalPoolsAndClosesWorkflowOnConstructionFailure(t *testing.T) {
	c := supplyRuntimeConfig(t)
	c.Identity.IssuerURL = "https://localhost:18080"
	c.Identity.AuthorizationAPIURL = "https://localhost:18080"
	c.SupplyCompletionDrafts = true
	c.ControlledLocalDraft = localDraftConfig().ControlledLocalDraft
	c.ControlledLocalDraft.AssetDatabase = c.SupplyChain.AssetDatabase
	c.SourceMedia = &coreconfig.ImageAgentArtifactStoreConfig{Enabled: true, Provider: "s3", PublicBase: "https://files.example.org/products", S3: coreconfig.ImageAgentArtifactStoreS3Config{Bucket: "products", Region: "us-east-1", AccessKeyID: "synthetic", SecretAccessKey: "synthetic", ArtifactMode: "aws"}}
	app := c.StoreCenter.OfficialApplications[0]
	require.NoError(t, os.WriteFile(app.AppSecretFile, []byte(strings.Repeat("synthetic", 4)), 0600))
	require.NoError(t, os.WriteFile(app.CredentialKeyFile, []byte(base64.StdEncoding.EncodeToString(make([]byte, 32))), 0600))
	privatizeSyntheticTestFile(t, app.AppSecretFile)
	privatizeSyntheticTestFile(t, app.CredentialKeyFile)
	source, product, owner, store, assets := &gorm.DB{}, &gorm.DB{}, emptyStorageCommercialOwner(t), &gorm.DB{}, &gorm.DB{}
	workflow := &mocks.Client{}
	closed := false
	assetOpens := 0
	var pools []*gorm.DB
	err := Run(context.Background(), c, logrus.New(), Dependencies{
		IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
		OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
		OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil },
		OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return owner, nil },
		OpenStoreCenter:        func(context.Context, DatabaseConfig) (*gorm.DB, error) { return store, nil },
		OpenApprovedAssets: func(_ context.Context, db DatabaseConfig) (*gorm.DB, error) {
			assetOpens++
			require.Equal(t, c.SupplyChain.AssetDatabase, db)
			return assets, nil
		},
		DialProductWorkflow: func(context.Context, string, string) (client.Client, func() error, error) {
			return workflow, func() error { closed = true; return nil }, nil
		},
		CloseDatabase: func(db *gorm.DB) error { pools = append(pools, db); return nil },
		NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
			require.NotNil(t, f.ControlledLocalDraft)
			require.Same(t, assets, f.ApprovedAssetDB)
			require.Same(t, workflow, f.ListingWorkflow)
			require.NotNil(t, f.ListingWorker)
			require.NotNil(t, f.SourceMediaStorage)
			require.Contains(t, f.SourceMediaStorage.PublicURL("fixture"), "https://files.example.org/products/")
			return nil, errors.New("fixture assembly failure")
		},
	})
	require.ErrorContains(t, err, "fixture assembly failure")
	require.True(t, closed)
	require.Equal(t, 1, assetOpens)
	count := 0
	for _, p := range pools {
		if p == assets {
			count++
		}
	}
	require.Equal(t, 1, count)
}

func supplyRuntimeConfig(t *testing.T) *Config {
	c := acquisitionRuntimeConfig()
	c.ProductCollections = true
	c.Identity.TenantDirectoryToken = "fixture-token"
	root := t.TempDir()
	c.StoreCenter = &StoreCenterConfig{Enabled: true, Database: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "store_center_runtime", Password: "fixture", Database: "stores", MaxConnections: 2}, OfficialApplications: []OfficialStoreConnectionConfig{{Type: storecenter.ApplicationSelfOperated, AppID: "fixture-app", Version: "v1", APIOrigin: "https://openapi.sheincorp.com", CallbackURL: "https://localhost/callback", AppSecretFile: filepath.Join(root, "secret"), CredentialKeyFile: filepath.Join(root, "key"), CredentialKeyID: "fixture-key"}}}
	c.SupplyChain = &SupplyChainConfig{AssetDatabase: DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "supply_asset_runtime", Password: "fixture", Database: "assets", MaxConnections: 2}, TemporalAddress: "127.0.0.1:7233", TemporalNamespace: "default"}
	return c
}

func ecoservicesSupplyRuntimeConfig(t *testing.T) *Config {
	c := supplyRuntimeConfig(t)
	eco := ecoservicesTestConfig()
	c.Ecoservices = eco.Ecoservices
	c.Ecoservices.Database.Host = c.SourceAccountDatabase.Host
	c.Ecoservices.Database.Port = c.SourceAccountDatabase.Port
	c.MoneyOwnerDatabase = eco.MoneyOwnerDatabase
	return c
}

func TestSupplyAndEcoservicesCannotDeclareTheSameDatabase(t *testing.T) {
	c := ecoservicesSupplyRuntimeConfig(t)
	require.NoError(t, c.validate())
	c.SupplyChain.AssetDatabase.Database = c.Ecoservices.Database.Database
	require.ErrorContains(t, c.validate(), "supply assets require their independently owned database")
}

func TestSupplyAndEcoservicesRuntimePoolsStayIndependent(t *testing.T) {
	for _, alias := range []bool{false, true} {
		t.Run(map[bool]string{false: "both features", true: "shared injected pool"}[alias], func(t *testing.T) {
			c := ecoservicesSupplyRuntimeConfig(t)
			app := c.StoreCenter.OfficialApplications[0]
			require.NoError(t, os.WriteFile(app.AppSecretFile, []byte(strings.Repeat("synthetic", 4)), 0600))
			require.NoError(t, os.WriteFile(app.CredentialKeyFile, []byte(base64.StdEncoding.EncodeToString(make([]byte, 32))), 0600))
			privatizeSyntheticTestFile(t, app.AppSecretFile)
			privatizeSyntheticTestFile(t, app.CredentialKeyFile)
			source, product, commercial, money, store, assets, eco := &gorm.DB{}, &gorm.DB{}, emptyStorageCommercialOwner(t), &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}
			if alias {
				eco = assets
			}
			workflow := &mocks.Client{}
			constructed, assembled := 0, 0
			closed := map[*gorm.DB]int{}
			stop := errors.New("bounded combination construction stop")
			err := Run(context.Background(), c, logrus.New(), Dependencies{
				IdentityPreflight:      func(context.Context, IdentityConfig) error { return nil },
				OpenSourceAccount:      func(context.Context, DatabaseConfig) (*gorm.DB, error) { return source, nil },
				OpenProductAcquisition: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return product, nil },
				OpenCommercialOwner:    func(context.Context, DatabaseConfig) (*gorm.DB, error) { return commercial, nil },
				OpenMoneyOwner:         func(context.Context, DatabaseConfig) (*gorm.DB, error) { return money, nil },
				OpenStoreCenter:        func(context.Context, DatabaseConfig) (*gorm.DB, error) { return store, nil },
				OpenApprovedAssets:     func(context.Context, DatabaseConfig) (*gorm.DB, error) { return assets, nil },
				DialProductWorkflow: func(context.Context, string, string) (client.Client, func() error, error) {
					return workflow, func() error { return nil }, nil
				},
				OpenEcoservices: func(context.Context, DatabaseConfig) (*gorm.DB, error) { return eco, nil },
				NewEcoservices: func(context.Context, *gorm.DB, *EcoservicesConfig) (*EcoservicesRuntime, error) {
					constructed++
					return &EcoservicesRuntime{DB: eco}, nil
				},
				NewApplicationWithFeatures: func(_ context.Context, _ *gorm.DB, f ApplicationFeatures, _ *coreconfig.Config, _ *logrus.Logger) (*http.Server, error) {
					assembled++
					require.Same(t, eco, f.Ecoservices.DB)
					require.Same(t, assets, f.ApprovedAssetDB)
					require.NotNil(t, f.OfficialStoreApplications)
					require.True(t, f.ProductCollections)
					return nil, stop
				},
				Listen:        func(string, string) (net.Listener, error) { return nil, stop },
				CloseDatabase: func(db *gorm.DB) error { closed[db]++; return nil },
			})
			if alias {
				require.ErrorContains(t, err, "ecoservices requires an independent owner pool")
				require.Zero(t, constructed)
				require.Zero(t, assembled)
			} else {
				require.ErrorIs(t, err, stop)
				require.Equal(t, 1, constructed)
				require.Equal(t, 1, assembled)
			}
			for _, db := range []*gorm.DB{source, product, commercial, money, store, assets, eco} {
				require.Equal(t, 1, closed[db], "each original owner pool must close once")
			}
		})
	}
}
func TestSupplyManifestPinsExistingAssetOwnerAndDedicatedRole(t *testing.T) {
	c := supplyRuntimeConfig(t)
	require.NoError(t, c.validate())
	for name, mutate := range map[string]func(*Config){
		"missing membership":   func(c *Config) { c.Identity.TenantDirectoryToken = "" },
		"missing collections":  func(c *Config) { c.ProductCollections = false },
		"missing applications": func(c *Config) { c.StoreCenter.OfficialApplications = nil },
		"wrong role":           func(c *Config) { c.SupplyChain.AssetDatabase.User = "product_agent_runtime" },
		"shared Product":       func(c *Config) { c.SupplyChain.AssetDatabase.Database = "product" },
		"split Asset fact": func(c *Config) {
			c.ImageAgent = &ImageAgentConfig{Database: DatabaseConfig{Host: "127.0.0.1", Port: 5432, Database: "another_asset"}}
		},
		"unbounded workflow endpoint": func(c *Config) { c.SupplyChain.TemporalAddress = "external.example:7233" },
	} {
		t.Run(name, func(t *testing.T) { c := supplyRuntimeConfig(t); mutate(c); require.Error(t, c.validate()) })
	}
}
