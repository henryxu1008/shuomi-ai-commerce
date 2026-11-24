package currentapplication

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	integration "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/googlemerchant"
	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
)

type GoogleMerchantConfig struct {
	ClientID          string `json:"clientId"`
	Revision          string `json:"revision"`
	CallbackURL       string `json:"callbackURL"`
	ClientSecretFile  string `json:"clientSecretFile"`
	CredentialKeyFile string `json:"credentialKeyFile"`
	CredentialKeyID   string `json:"credentialKeyId"`
}

func (c *GoogleMerchantConfig) application() google.Application {
	return google.Application{ID: c.ClientID, Revision: c.Revision, CallbackURL: c.CallbackURL}
}

func (c *GoogleMerchantConfig) validate() error {
	if c == nil {
		return nil
	}
	if !filepath.IsAbs(c.ClientSecretFile) || !filepath.IsAbs(c.CredentialKeyFile) ||
		strings.EqualFold(filepath.Clean(c.ClientSecretFile), filepath.Clean(c.CredentialKeyFile)) ||
		c.CredentialKeyID == "" || len(c.CredentialKeyID) > 128 {
		return errors.New("Google Merchant requires separate private credential files")
	}
	if _, err := integration.NewOAuth(c.application(), "validate-config-only", nil); err != nil {
		return errors.New("Google Merchant OAuth configuration invalid")
	}
	return nil
}

func (c *GoogleMerchantConfig) prepare(ctx context.Context, registry *storeapp.OfficialApplicationRegistry) (*storeapp.OfficialApplicationRegistry, error) {
	if c == nil {
		return registry, nil
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	secret, err := readOfficialPrivateFile(ctx, c.ClientSecretFile)
	if err != nil {
		return nil, err
	}
	defer clear(secret)
	keyText, err := readOfficialPrivateFile(ctx, c.CredentialKeyFile)
	if err != nil {
		return nil, err
	}
	defer clear(keyText)
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyText)))
	if err != nil {
		return nil, errors.New("Google Merchant credential key encoding invalid")
	}
	defer clear(key)
	app := c.application()
	oauth, err := integration.NewOAuth(app, strings.TrimSpace(string(secret)), nil)
	if err != nil {
		return nil, err
	}
	protection, err := integration.NewCredentialProtection(c.CredentialKeyID, key)
	if err != nil {
		return nil, err
	}
	accounts, err := integration.NewClient(nil)
	if err != nil {
		return nil, err
	}
	return storeapp.WithGoogleMerchantApplication(registry, storeapp.GoogleApplicationRegistration{
		Application: app, OAuth: oauth, Accounts: accounts, Protection: protection,
	})
}
