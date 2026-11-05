package currentapplication

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"gorm.io/gorm"
	credential "task-processor/internal/aicapability/channelcredential"
	coreconfig "task-processor/internal/core/config"
	governed "task-processor/internal/integration/aicapability/einomodel"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
)

// ChannelCredentialsConfig is private deployment input. The metadata/config
// lifecycle never needs to instantiate this secret reader or decrypt a Key.
type ChannelCredentialsConfig struct {
	Reader           DatabaseConfig  `json:"reader"`
	ManagementWriter *DatabaseConfig `json:"managementWriter,omitempty"`
	ProtectionKeyID  string          `json:"protectionKeyId"`
	ProtectionKey    string          `json:"protectionKey"`
}

func (c *ChannelCredentialsConfig) protection() (*channelstore.Protection, error) {
	if c == nil {
		return nil, credential.ErrUnavailable
	}
	key, e := base64.StdEncoding.Strict().DecodeString(c.ProtectionKey)
	if e != nil || len(key) != 32 {
		return nil, credential.ErrUnavailable
	}
	return channelstore.NewProtection(c.ProtectionKeyID, key)
}

type ChannelCredentialProvision struct {
	credential.Command
	APIKey         string         `json:"apiKey,omitempty"`
	WriterDatabase DatabaseConfig `json:"writerDatabase"`
}

// This is the original private operator entry, now connection-scoped. It does
// not accept a browser-controlled model, endpoint, member or global credential.
func LoadChannelCredentialProvision(path string) (ChannelCredentialProvision, error) {
	var input ChannelCredentialProvision
	if !filepath.IsAbs(path) {
		return input, credential.ErrInvalid
	}
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() {
		return input, credential.ErrInvalid
	}
	if runtime.GOOS == "windows" {
		if coreconfig.VerifyPrivateFiles(context.Background(), []string{path}) != nil {
			return input, credential.ErrInvalid
		}
	} else if info.Mode().Perm()&0o077 != 0 {
		return input, credential.ErrInvalid
	}
	f, e := os.Open(path)
	if e != nil {
		return input, credential.ErrInvalid
	}
	defer f.Close()
	raw, e := io.ReadAll(io.LimitReader(f, (16<<10)+1))
	if e != nil || len(raw) > 16<<10 || validateJSONShape(raw) != nil {
		return input, credential.ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&input) != nil || ensureJSONEOF(decoder) != nil {
		return ChannelCredentialProvision{}, credential.ErrInvalid
	}
	return input, nil
}

func ValidateChannelCredentialProvision(cfg *Config, input ChannelCredentialProvision) error {
	c := input.Command
	c.Secret = input.APIKey
	if cfg == nil || cfg.ProductAgent == nil || cfg.ModelRoutePricing == nil || cfg.ChannelCredentials == nil || c.Validate() != nil {
		return credential.ErrInvalid
	}
	if input.WriterDatabase.validate("channel credential writer") != nil || !sameDatabaseTarget(input.WriterDatabase, cfg.ProductAgent.Database) || input.WriterDatabase.User != "title_credential_writer" || input.WriterDatabase.User == cfg.ProductAgent.Database.User || input.WriterDatabase.MaxConnections > 2 {
		return credential.ErrInvalid
	}
	allowed := false
	for _, org := range cfg.ProductAgent.AllowedOrganizationIDs {
		allowed = allowed || org == c.OrganizationID
	}
	if !allowed {
		return credential.ErrInvalid
	}
	defs, e := cfg.modelRouteDefinitions()
	if e != nil {
		return credential.ErrInvalid
	}
	if cfg.ProductAgent.hasPolicies() && cfg.ProductAgent.validatePolicyMetadata() != nil {
		return credential.ErrInvalid
	}
	found := false
	for _, d := range defs {
		if d.Scope.OrganizationID != c.OrganizationID || d.ConnectionID != c.ConnectionID {
			continue
		}
		found = true
		if d.ConnectionProvider != c.Provider || d.Channel != c.Channel || d.Region != c.Region {
			return credential.ErrInvalid
		}
		if d.Text != nil {
			p := d.Text.ShapeProfile()
			if governed.EndpointIdentityDigest(c.BaseURL) != p.EndpointIdentityDigest || !governed.ValidAdapterEndpoint(governed.AdapterKind(p.AdapterKind), c.BaseURL) {
				return credential.ErrInvalid
			}
		}
		if d.Image != nil && governed.EndpointIdentityDigest(d.Image.Endpoint) != governed.EndpointIdentityDigest(c.BaseURL) {
			return credential.ErrInvalid
		}
	}
	if !found {
		return credential.ErrInvalid
	}
	return nil
}

func SaveChannelCredential(ctx context.Context, cfg *Config, input ChannelCredentialProvision, db *gorm.DB) (credential.Metadata, error) {
	if ctx == nil || db == nil || ValidateChannelCredentialProvision(cfg, input) != nil {
		return credential.Metadata{}, credential.ErrInvalid
	}
	// Verify the actual fixed writer, not merely the manifest's claimed role.
	var allowed bool
	if db.WithContext(ctx).Raw(`SELECT current_user='title_credential_writer' AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication) AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid OR roleid=r.oid) FROM pg_roles r WHERE rolname=current_user`).Scan(&allowed).Error != nil || !allowed {
		return credential.Metadata{}, credential.ErrUnavailable
	}
	p, e := cfg.ChannelCredentials.protection()
	if e != nil {
		return credential.Metadata{}, e
	}
	store, e := channelstore.New(db, p)
	if e != nil {
		return credential.Metadata{}, e
	}
	c := input.Command
	c.Secret = input.APIKey
	return store.Publish(ctx, c)
}

func (c *Config) validateChannelCredentials() error {
	if c != nil && c.ChannelCredentials != nil && c.ChannelCredentials.ManagementWriter != nil {
		w := c.ChannelCredentials.ManagementWriter
		if c.ProductAgent == nil || c.ModelRoutePricing == nil || w.validate("channel management writer") != nil || w.User != "title_credential_writer" || w.MaxConnections > 2 || !sameDatabaseTarget(*w, c.ProductAgent.Database) {
			return credential.ErrInvalid
		}
		if _, e := c.ChannelCredentials.protection(); e != nil {
			return e
		}
		defs, e := c.modelRouteDefinitions()
		if e != nil {
			return e
		}
		if _, e = channelManagementDefinitions(defs, true); e != nil {
			return e
		}
	}
	if c == nil || c.ChannelCredentials == nil || c.ProductAgent == nil || !c.needsChannelSecrets() {
		return nil
	}
	r := c.ChannelCredentials.Reader
	if r.validate("channel secret reader") != nil || r.User != "ai_channel_secret_reader" || r.MaxConnections > 2 || !sameDatabaseTarget(r, c.ProductAgent.Database) {
		return errors.New("channel secret reader must be a bounded read-only identity on Run DB")
	}
	if _, e := c.ChannelCredentials.protection(); e != nil {
		return errors.New("channel credential protection unavailable")
	}
	return nil
}

func openChannelCredentials(ctx context.Context, c *Config, open func(context.Context, DatabaseConfig) (*gorm.DB, error)) (credential.SecretReader, *gorm.DB, error) {
	if c == nil || c.ChannelCredentials == nil || c.ProductAgent == nil || !c.needsChannelSecrets() {
		return nil, nil, nil
	}
	if open == nil || c.validateChannelCredentials() != nil {
		return nil, nil, credential.ErrUnavailable
	}
	db, e := open(ctx, c.ChannelCredentials.Reader)
	if e != nil || db == nil {
		return nil, db, credential.ErrUnavailable
	}
	if channelstore.VerifySecretReader(ctx, db) != nil {
		return nil, db, credential.ErrUnavailable
	}
	p, e := c.ChannelCredentials.protection()
	if e != nil {
		return nil, db, e
	}
	s, e := channelstore.New(db, p)
	return s, db, e
}

func (c *Config) needsChannelSecrets() bool {
	return c != nil && ((c.ProductAgent != nil && c.ProductAgent.Enabled) || (c.ImageAgent != nil && c.ImageAgent.Generation.Enabled))
}
