package currentapplication

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	coreconfig "task-processor/internal/core/config"
	"task-processor/internal/integration/shein"
	"task-processor/internal/integration/temu"
	"task-processor/internal/storecenter"
)

type OfficialStoreConnectionConfig struct {
	Platform          storecenter.Platform                  `json:"platform,omitempty"`
	IdentityPartition storecenter.OfficialIdentityPartition `json:"identityPartition,omitempty"`
	Type              storecenter.OfficialApplicationType   `json:"type"`
	AppID             string                                `json:"appId"`
	Version           string                                `json:"version"`
	APIOrigin         string                                `json:"apiOrigin"`
	CallbackURL       string                                `json:"callbackURL"`
	AppSecretFile     string                                `json:"appSecretFile"`
	CredentialKeyFile string                                `json:"credentialKeyFile"`
	CredentialKeyID   string                                `json:"credentialKeyId"`
}

func (c *OfficialStoreConnectionConfig) application() (storecenter.OfficialApplication, error) {
	if !authidentity.IsBoundedIdentifier(c.Version) || strings.Contains(c.Version, "~") {
		return storecenter.OfficialApplication{}, errors.New("official Store application revision invalid")
	}
	// Type is the existing SHEIN configuration discriminator. TEMU requires an
	// explicit platform/partition and never reuses that application's one mode.
	if (c.Platform == "" || c.Platform == storecenter.PlatformShein) && c.Type.Valid() && c.IdentityPartition == "" {
		return storecenter.OfficialApplication{Platform: storecenter.PlatformShein, AuthorizationMethod: storecenter.AuthorizationCallback, AppID: c.AppID, Version: storeapp.BoundOfficialRevision(c.Version, c.Type), CallbackURL: c.CallbackURL}, nil
	}
	if c.Platform != storecenter.PlatformTemu || c.Type != "" || c.APIOrigin != "" || c.CallbackURL != "" {
		return storecenter.OfficialApplication{}, errors.New("official Store application platform contract invalid")
	}
	modes := storecenter.ModesLocal
	switch c.IdentityPartition {
	case storecenter.PartitionCN:
		modes = storecenter.ModesFullyManaged | storecenter.ModesSemiManaged
	case storecenter.PartitionUS, storecenter.PartitionEU, storecenter.PartitionGlobal:
	default:
		return storecenter.OfficialApplication{}, errors.New("official Store application identity partition invalid")
	}
	return storecenter.OfficialApplication{Platform: storecenter.PlatformTemu, IdentityPartition: c.IdentityPartition, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: modes, AppID: c.AppID, Version: storeapp.BoundTemuOfficialRevision(c.Version, c.IdentityPartition)}, nil
}

func (c *OfficialStoreConnectionConfig) validate() error {
	if c == nil {
		return nil
	}
	app, err := c.application()
	if err != nil {
		return err
	}
	if !authidentity.IsBoundedIdentifier(app.Version) {
		return errors.New("official Store bound application revision invalid")
	}
	if !filepath.IsAbs(c.AppSecretFile) || !filepath.IsAbs(c.CredentialKeyFile) || c.AppSecretFile == c.CredentialKeyFile || c.CredentialKeyID == "" || len(c.CredentialKeyID) > 128 {
		return errors.New("official Store connection requires separate private credential files and key identity")
	}
	// Validate protocol/configuration without network calls or secret-file reads.
	if app.Platform == storecenter.PlatformShein {
		_, err = shein.NewOfficialClient(shein.OfficialClientOptions{Application: app, AppSecret: strings.Repeat("0", 32), APIOrigin: c.APIOrigin})
	} else {
		_, err = temu.NewOfficialClient(temu.OfficialClientOptions{Application: app, AppSecret: strings.Repeat("0", 32)})
	}
	if err != nil {
		return errors.New("official Store application configuration invalid")
	}
	return nil
}
func (c *OfficialStoreConnectionConfig) prepare(ctx context.Context) (storecenter.OfficialConnectionProvider, storecenter.OfficialCredentialProtection, error) {
	if c == nil {
		return nil, nil, nil
	}
	if err := c.validate(); err != nil {
		return nil, nil, err
	}
	secret, err := readOfficialPrivateFile(ctx, c.AppSecretFile)
	if err != nil {
		return nil, nil, err
	}
	defer clear(secret)
	keyText, err := readOfficialPrivateFile(ctx, c.CredentialKeyFile)
	if err != nil {
		return nil, nil, err
	}
	defer clear(keyText)
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(keyText)))
	if err != nil {
		return nil, nil, errors.New("official Store credential key must be Base64 encoded")
	}
	defer clear(key)
	app, _ := c.application()
	var provider storecenter.OfficialConnectionProvider
	var protection storecenter.OfficialCredentialProtection
	if app.Platform == storecenter.PlatformShein {
		protection, err = shein.NewCredentialProtection(c.CredentialKeyID, key)
		if err == nil {
			provider, err = shein.NewOfficialClient(shein.OfficialClientOptions{Application: app, AppSecret: strings.TrimSpace(string(secret)), APIOrigin: c.APIOrigin})
		}
	} else {
		protection, err = temu.NewCredentialProtection(c.CredentialKeyID, key)
		if err == nil {
			provider, err = temu.NewOfficialClient(temu.OfficialClientOptions{Application: app, AppSecret: strings.TrimSpace(string(secret))})
		}
	}
	if err != nil {
		return nil, nil, errors.New("official Store application configuration unavailable")
	}
	return provider, protection, nil
}
func validateOfficialApplications(configs []OfficialStoreConnectionConfig) error {
	if len(configs) > 16 {
		return errors.New("at most 16 official Store applications")
	}
	apps, keys, files := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, config := range configs {
		if err := config.validate(); err != nil {
			return err
		}
		app, _ := config.application()
		identity := string(app.Platform) + ":" + string(app.IdentityPartition) + ":" + app.AppID
		if apps[identity] || keys[config.CredentialKeyID] {
			return errors.New("official Store application and credential key identities must be unique")
		}
		apps[identity], keys[config.CredentialKeyID] = true, true
		for _, path := range []string{config.AppSecretFile, config.CredentialKeyFile} {
			path = filepath.Clean(path)
			if runtime.GOOS == "windows" {
				path = strings.ToLower(path)
			}
			if files[path] {
				return errors.New("official Store applications require separate private files")
			}
			files[path] = true
		}
	}
	return nil
}
func prepareOfficialApplications(ctx context.Context, configs []OfficialStoreConnectionConfig) (*storeapp.OfficialApplicationRegistry, error) {
	if err := validateOfficialApplications(configs); err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return nil, nil
	}
	registrations := make([]storeapp.OfficialApplicationRegistration, 0, len(configs))
	for _, config := range configs {
		provider, protection, err := config.prepare(ctx)
		if err != nil {
			return nil, err
		}
		registrations = append(registrations, storeapp.OfficialApplicationRegistration{Provider: provider, Protection: protection, Type: config.Type})
	}
	return storeapp.NewOfficialApplicationRegistry(registrations)
}
func readOfficialPrivateFile(ctx context.Context, path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1024 || info.Size() < 1 {
		return nil, errors.New("official Store credential file unavailable")
	}
	if runtime.GOOS == "windows" {
		if err := coreconfig.VerifyPrivateFiles(ctx, []string{path}); err != nil {
			return nil, errors.New("official Store credential file must be private")
		}
	} else if info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("official Store credential file must be private")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("official Store credential file unavailable")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 1025))
	if err != nil || len(data) > 1024 {
		return nil, errors.New("official Store credential file unavailable")
	}
	return data, nil
}
