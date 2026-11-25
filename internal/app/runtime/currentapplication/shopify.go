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
