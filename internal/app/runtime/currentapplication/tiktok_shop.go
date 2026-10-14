package currentapplication

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"

	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/integration/tiktokshop"
	authorization "task-processor/internal/storecenter/tiktokshop"
)

type TikTokShopConfig struct {
	AppKey            string `json:"appKey"`
	ServiceID         string `json:"serviceId"`
	Revision          string `json:"revision"`
	CallbackURL       string `json:"callbackURL"`
	AppSecretFile     string `json:"appSecretFile"`
	CredentialKeyFile string `json:"credentialKeyFile"`
	CredentialKeyID   string `json:"credentialKeyId"`
}

func (c *TikTokShopConfig) application() authorization.Application {
	return authorization.Application{AppKey: c.AppKey, ServiceID: c.ServiceID, Revision: c.Revision, CallbackURL: c.CallbackURL}
}
func (c *TikTokShopConfig) validate() error {
	if c == nil {
		return nil
	}
	if !authidentity.IsBoundedIdentifier(c.Revision) || !authidentity.IsBoundedIdentifier(c.CredentialKeyID) || !filepath.IsAbs(c.AppSecretFile) || !filepath.IsAbs(c.CredentialKeyFile) || strings.EqualFold(filepath.Clean(c.AppSecretFile), filepath.Clean(c.CredentialKeyFile)) {
		return errors.New("TikTok Shop requires separate private credential files and an explicit revision")
	}
	if _, err := tiktokshop.NewOfficialClient(tiktokshop.OfficialClientOptions{Application: c.application(), AppSecret: "validation-only"}); err != nil {
		return errors.New("TikTok Shop application configuration invalid")
	}
	return nil
}
func (c *TikTokShopConfig) prepare(ctx context.Context, registry *storeapp.OfficialApplicationRegistry) (*storeapp.OfficialApplicationRegistry, error) {
	if c == nil {
		return registry, nil
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	secret, err := readOfficialPrivateFile(ctx, c.AppSecretFile)
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
		return nil, errors.New("TikTok Shop credential key must be Base64 encoded")
	}
	defer clear(key)
	protection, err := tiktokshop.NewCredentialProtection(c.CredentialKeyID, key)
	if err != nil {
		return nil, errors.New("TikTok Shop credential protection unavailable")
	}
	provider, err := tiktokshop.NewOfficialClient(tiktokshop.OfficialClientOptions{Application: c.application(), AppSecret: strings.TrimSpace(string(secret))})
	if err != nil {
		return nil, errors.New("TikTok Shop application unavailable")
	}
	return storeapp.WithTikTokApplication(registry, provider, protection)
}
