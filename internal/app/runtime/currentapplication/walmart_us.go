package currentapplication

import (
	"context"
	"encoding/base64"
	"errors"
	"path/filepath"
	"strings"

	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	integration "task-processor/internal/integration/walmart"
	auth "task-processor/internal/storecenter/walmart"
)

type WalmartUSConfig struct {
	ClientID          string `json:"clientId"`
	Revision          string `json:"revision"`
	Environment       string `json:"environment"`
	CallbackURL       string `json:"callbackURL"`
	ClientSecretFile  string `json:"clientSecretFile"`
	CredentialKeyFile string `json:"credentialKeyFile"`
	CredentialKeyID   string `json:"credentialKeyId"`
	FullSpecVersion   string `json:"fullSpecVersion"`
	MatchSpecVersion  string `json:"matchSpecVersion"`
}

func (c *WalmartUSConfig) application() auth.Application {
	return auth.Application{ClientID: c.ClientID, Revision: c.Revision, Environment: c.Environment, CallbackURL: c.CallbackURL}
}
func (c *WalmartUSConfig) validate() error {
	if c == nil {
		return nil
	}
	if !authidentity.IsBoundedIdentifier(c.ClientID) || !authidentity.IsBoundedIdentifier(c.Revision) || !authidentity.IsBoundedIdentifier(c.CredentialKeyID) || !filepath.IsAbs(c.ClientSecretFile) || !filepath.IsAbs(c.CredentialKeyFile) || strings.EqualFold(filepath.Clean(c.ClientSecretFile), filepath.Clean(c.CredentialKeyFile)) || len(c.FullSpecVersion) == 0 || len(c.FullSpecVersion) > 128 || len(c.MatchSpecVersion) == 0 || len(c.MatchSpecVersion) > 128 {
		return errors.New("Walmart US requires separate private credential files, revision and pinned spec versions")
	}
	_, e := integration.NewOfficialClient(integration.Options{Application: c.application(), ClientSecret: "validation-only", FullVersion: c.FullSpecVersion, MatchVersion: c.MatchSpecVersion})
	return e
}
func (c *WalmartUSConfig) prepare(ctx context.Context, r *storeapp.OfficialApplicationRegistry) (*storeapp.OfficialApplicationRegistry, error) {
	if c == nil {
		return r, nil
	}
	if e := c.validate(); e != nil {
		return nil, e
	}
	secret, e := readOfficialPrivateFile(ctx, c.ClientSecretFile)
	if e != nil {
		return nil, e
	}
	defer clear(secret)
	keyText, e := readOfficialPrivateFile(ctx, c.CredentialKeyFile)
	if e != nil {
		return nil, e
	}
	defer clear(keyText)
	key, e := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyText)))
	if e != nil {
		return nil, errors.New("Walmart credential key must be Base64 encoded")
	}
	defer clear(key)
	protection, e := integration.NewCredentialProtection(c.CredentialKeyID, key)
	if e != nil {
		return nil, e
	}
	provider, e := integration.NewOfficialClient(integration.Options{Application: c.application(), ClientSecret: strings.TrimSpace(string(secret)), FullVersion: c.FullSpecVersion, MatchVersion: c.MatchSpecVersion})
	if e != nil {
		return nil, e
	}
	return storeapp.WithWalmartApplication(r, provider, protection)
}
