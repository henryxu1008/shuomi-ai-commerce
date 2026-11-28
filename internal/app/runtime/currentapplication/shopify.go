package currentapplication

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"net"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
)

type ShopifyConfig struct {
	ApplicationGID         string `json:"applicationGid"`
	ClientID               string `json:"clientId"`
	InstallURL             string `json:"installURL"`
	CallbackURL            string `json:"callbackURL"`
	ClientSecret           string `json:"clientSecretFile"`
	PairingKey             string `json:"pairingKeyFile"`
	PairingKeyID           string `json:"pairingKeyId"`
	SigningKey             string `json:"signingKeyFile"`
	SigningKeyID           string `json:"signingKeyId"`
	StartedInstallation    bool   `json:"startedInstallation"`
	LifecycleNotifications bool   `json:"lifecycleNotifications"`
}

// ShopifyMediaTrialConfig admits one fixed, loopback-only development-store
// media synchronization trial. Every database owner is independently named.
type ShopifyMediaTrialConfig struct {
	ProductInitializer DatabaseConfig `json:"productInitializer"`
	ProductServing     DatabaseConfig `json:"productServing"`
	AssetDatabase      DatabaseConfig `json:"assetDatabase"`
	OrganizationID     string         `json:"organizationId"`
	ActorID            string         `json:"actorId"`
	MemberID           string         `json:"memberId"`
	SourceID           string         `json:"sourceId"`
	ProductKey         string         `json:"productKey"`
	SourceVersion      int64          `json:"sourceVersion"`
	ApprovedDigest     string         `json:"approvedDigest"`
	StoreID            string         `json:"storeId"`
	ShopKey            string         `json:"shopKey"`
	Generation         int64          `json:"generation"`
	ApplicationGID     string         `json:"applicationGid"`
	ShopGID            string         `json:"shopGid"`
	InstallationGID    string         `json:"installationGid"`
	Domain             string         `json:"domain"`
	DraftProductGID    string         `json:"draftProductGid"`
	GalleryDigest      string         `json:"galleryDigest"`
	AppendDigest       string         `json:"appendDigest"`
	ServiceStartsAt    time.Time      `json:"serviceStartsAt"`
	ServiceExpiresAt   time.Time      `json:"serviceExpiresAt"`
}

func (cfg *Config) validateShopifyMediaTrial() error {
	if cfg.ShopifyMediaTrial == nil {
		return nil
	}
	const message = "Shopify media trial requires a fixed loopback fixture and independent least-privilege owners"
	trial := cfg.ShopifyMediaTrial
	store := cfg.StoreCenter
	if cfg.ShopifyPairingTrial || store == nil || !store.Enabled || store.Shopify == nil ||
		!net.ParseIP(cfg.Listen.Host).IsLoopback() || cfg.Identity.TenantDirectoryToken == "" ||
		trial.SourceVersion < 1 || trial.Generation < 1 || !trial.ServiceExpiresAt.After(trial.ServiceStartsAt) ||
		trial.ServiceExpiresAt.Sub(trial.ServiceStartsAt) > 24*time.Hour ||
		trial.DraftProductGID == "gid://shopify/Product/10419372589218" {
		return errors.New(message)
	}
	for _, db := range []DatabaseConfig{trial.ProductInitializer, trial.ProductServing, trial.AssetDatabase, store.Database} {
		if err := db.validate("shopifyMediaTrial.database"); err != nil || !net.ParseIP(db.Host).IsLoopback() {
			return errors.New(message)
		}
	}
	if !sameDatabaseTarget(trial.ProductInitializer, trial.ProductServing) ||
		trial.ProductInitializer.User == trial.ProductServing.User || trial.ProductServing.User != "source_acquisition_runtime" ||
		trial.ProductInitializer.Database == trial.AssetDatabase.Database || trial.ProductServing.Database == store.Database.Database ||
		trial.ProductInitializer.Database == store.Database.Database || trial.AssetDatabase.Database == store.Database.Database ||
		trial.AssetDatabase.Database == trial.ProductServing.Database || trial.ProductServing.MaxConnections > 8 ||
		trial.AssetDatabase.User != "supply_asset_runtime" ||
		trial.ProductInitializer.User != "acquisition_owner" || !validTrialDigest(trial.ApprovedDigest) ||
		!validTrialDigest(trial.GalleryDigest) || !validTrialDigest(trial.AppendDigest) ||
		!strings.HasPrefix(trial.ApplicationGID, "gid://shopify/App/") ||
		!strings.HasPrefix(trial.ShopGID, "gid://shopify/Shop/") ||
		!strings.HasPrefix(trial.InstallationGID, "gid://shopify/AppInstallation/") ||
		!strings.HasPrefix(trial.DraftProductGID, "gid://shopify/Product/") ||
		!strings.HasSuffix(trial.Domain, ".myshopify.com") ||
		store.Shopify.ApplicationGID != trial.ApplicationGID || store.Shopify.LifecycleNotifications != true ||
		!boundedValue(trial.OrganizationID, 128) || !boundedValue(trial.ActorID, 128) ||
		!boundedValue(trial.MemberID, 128) || !boundedValue(trial.SourceID, 128) ||
		!boundedValue(trial.ProductKey, 256) || !boundedValue(trial.StoreID, 128) || !boundedValue(trial.ShopKey, 128) {
		return errors.New(message)
	}
	allowed := Config{SchemaVersion: cfg.SchemaVersion, Listen: cfg.Listen, Identity: cfg.Identity,
		SourceAccountDatabase: cfg.SourceAccountDatabase, CommercialOwnerDatabase: cfg.CommercialOwnerDatabase,
		StoreCenter: store, ShopifyMediaTrial: trial}
	if !reflect.DeepEqual(*cfg, allowed) {
		return errors.New(message)
	}
	return nil
}

func validTrialDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}

func (cfg *Config) validateShopifyPairingTrial() error {
	if !cfg.ShopifyPairingTrial {
		return nil
	}
	const message = "Shopify pairing trial requires a dedicated loopback Store with only pairing configured"
	store := cfg.StoreCenter
	if store == nil || !store.Enabled || store.Shopify == nil || cfg.Identity.TenantDirectoryToken == "" ||
		!net.ParseIP(cfg.Listen.Host).IsLoopback() || !net.ParseIP(store.Database.Host).IsLoopback() {
		return errors.New(message)
	}
	shopify := store.Shopify
	callback, err := url.Parse(shopify.CallbackURL)
	if err != nil || !net.ParseIP(callback.Hostname()).IsLoopback() || callback.Port() == "" ||
		callback.Path != "/api/shopify/callback" || shopify.StartedInstallation || shopify.LifecycleNotifications ||
		!reflect.DeepEqual(*store, StoreCenterConfig{Enabled: true, Database: store.Database, Shopify: shopify}) {
		return errors.New(message)
	}
	allowed := Config{
		SchemaVersion: cfg.SchemaVersion, Listen: cfg.Listen, Identity: cfg.Identity,
		SourceAccountDatabase: cfg.SourceAccountDatabase, CommercialOwnerDatabase: cfg.CommercialOwnerDatabase,
		Membership: cfg.Membership, ListingKitAuthorization: cfg.ListingKitAuthorization,
		StoreCenter: store, ShopifyPairingTrial: true, StorageCapacity: cfg.StorageCapacity,
	}
	if cfg.StorageCapacity {
		allowed.MoneyOwnerDatabase = cfg.MoneyOwnerDatabase
	}
	if !reflect.DeepEqual(*cfg, allowed) {
		return errors.New(message)
	}
	return nil
}

func (c *ShopifyConfig) validate() error {
	if c == nil {
		return nil
	}
	const bad = "Shopify requires a fixed application and separate private credential files"
	install, err := url.Parse(c.InstallURL)
	if err != nil || install.Scheme != "https" || install.Fragment != "" || install.User != nil ||
		(install.Host != "apps.shopify.com" && install.Host != "admin.shopify.com") || install.Path == "" ||
		!strings.HasPrefix(c.ApplicationGID, "gid://shopify/App/") || c.SigningKeyID == "" || c.PairingKeyID == "" {
		return errors.New(bad)
	}
	paths := map[string]bool{}
	for _, path := range []string{c.ClientSecret, c.PairingKey, c.SigningKey} {
		canonical := strings.ToLower(filepath.Clean(path))
		if !filepath.IsAbs(path) || paths[canonical] {
			return errors.New(bad)
		}
		paths[canonical] = true
	}
	_, err = wire.NewOAuthClient(nil, c.ClientID, "validation-only", c.CallbackURL)
	if err != nil {
		return errors.New(bad)
	}
	return nil
}

func (c *ShopifyConfig) prepare(ctx context.Context, registry *storeapp.OfficialApplicationRegistry, pairingTrial bool) (*storeapp.OfficialApplicationRegistry, error) {
	if c == nil {
		return registry, nil
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	secret, err := readOfficialPrivateFile(ctx, c.ClientSecret)
	if err != nil {
		return nil, err
	}
	defer clear(secret)
	var oauth *wire.OAuthClient
	var admin *wire.AdminClient
	if pairingTrial {
		oauth, admin, err = wire.NewDevelopmentPairingClients(c.ClientID, strings.TrimSpace(string(secret)), c.CallbackURL)
	} else {
		oauth, err = wire.NewOAuthClient(nil, c.ClientID, strings.TrimSpace(string(secret)), c.CallbackURL)
		admin = wire.NewDevelopmentAdminClient(nil)
	}
	if err != nil {
		return nil, err
	}
	encoded, err := readOfficialPrivateFile(ctx, c.PairingKey)
	if err != nil {
		return nil, err
	}
	defer clear(encoded)
	decoded, err := base64.StdEncoding.Strict().DecodeString(strings.TrimSpace(string(encoded)))
	if err != nil {
		return nil, errors.New("Shopify pairing key is invalid")
	}
	defer clear(decoded)
	protection, err := wire.NewPairingProtection(c.PairingKeyID, decoded)
	if err != nil {
		return nil, err
	}
	pemBytes, err := readOfficialPrivateFile(ctx, c.SigningKey)
	if err != nil {
		return nil, err
	}
	defer clear(pemBytes)
	private, err := parseShopifySigningKey(pemBytes)
	if err != nil {
		return nil, err
	}
	issuer, err := wire.NewAppTokenIssuer(nil, c.ClientID, c.SigningKeyID, private)
	if err != nil {
		return nil, err
	}
	return storeapp.WithShopifyApplication(registry, storeapp.ShopifyApplication{AppID: c.ApplicationGID, InstallURL: c.InstallURL,
		OAuth: oauth, Admin: admin, Issuer: issuer, Protection: protection, StartedInstallation: c.StartedInstallation, LifecycleNotifications: c.LifecycleNotifications})
}

func parseShopifySigningKey(raw []byte) (*ecdsa.PrivateKey, error) {
	block, remainder := pem.Decode(raw)
	if block != nil && len(strings.TrimSpace(string(remainder))) == 0 {
		defer clear(block.Bytes)
		if block.Type == "PRIVATE KEY" {
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if ec, ok := key.(*ecdsa.PrivateKey); err == nil && ok {
				return ec, nil
			}
		}
	}
	return nil, errors.New("Shopify signing key must be a valid PKCS#8 EC private key")
}
