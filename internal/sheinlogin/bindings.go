package sheinlogin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"
	"task-processor/internal/core/config"
	"task-processor/internal/storecenter"
)

var ErrInvalidInput = errors.New("invalid SHEIN worker input")
var ErrUnavailable = errors.New("SHEIN worker dependency unavailable")

type Binding struct {
	OrganizationID string `json:"organization_id"`
	StoreID        string `json:"store_id"`
	Revision       string `json:"revision"`
	LoginURL       string `json:"login_url"`
	CredentialFile string `json:"credential_file"`
	ProxyFile      string `json:"proxy_file,omitempty"`
	Headless       bool   `json:"headless"`
}
type AuthConfig struct {
	IssuerURL           string                 `json:"issuer_url"`
	AuthorizationAPIURL string                 `json:"authorization_api_url"`
	ClientID            string                 `json:"client_id"`
	ClientSecretFile    string                 `json:"client_secret_file"`
	ServiceTokenFile    string                 `json:"service_token_file"`
	ProjectID           string                 `json:"project_id"`
	RolePolicyDatabase  *config.DatabaseConfig `json:"role_policy_database,omitempty"`
}
type Config struct {
	InstallationID      string                `json:"installation_id"`
	StoreDatabase       config.DatabaseConfig `json:"store_database"`
	Redis               config.RedisConfig    `json:"redis"`
	ProfileRoot         string                `json:"profile_root"`
	ArtifactRoot        string                `json:"artifact_root"`
	DriverDirectory     string                `json:"driver_directory"`
	ViewportWidth       int                   `json:"viewport_width,omitempty"`
	ViewportHeight      int                   `json:"viewport_height,omitempty"`
	BrowserPath         string                `json:"browser_path"`
	MaxConcurrentLogins int                   `json:"max_concurrent_logins"`
	Auth                AuthConfig            `json:"auth"`
	Bindings            []Binding             `json:"bindings"`
}

func validUUID(v string) bool {
	u, e := uuid.Parse(v)
	return e == nil && u != uuid.Nil && u.String() == v
}
func validOpaque(v string) bool {
	return v != "" && len(v) <= 200 && strings.TrimSpace(v) == v && !strings.ContainsAny(v, "\x00\r\n")
}
func DecodeStrict(data []byte, out any, limit int) error {
	if len(data) == 0 || len(data) > limit {
		return ErrInvalidInput
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil || d.Decode(new(any)) != io.EOF {
		return ErrInvalidInput
	}
	return nil
}

// ReadPrivate is shared by config, auth and executor-only credential reads.
// Errors never include a path or underlying filesystem/provider content.
func ReadPrivate(ctx context.Context, path string, limit int) ([]byte, error) {
	if !filepath.IsAbs(path) || noSymlinkComponents(path) != nil {
		return nil, ErrInvalidInput
	}
	i, e := os.Lstat(path)
	if e != nil || !i.Mode().IsRegular() || i.Size() > int64(limit) {
		return nil, ErrInvalidInput
	}
	if runtime.GOOS != "windows" && i.Mode().Perm()&0077 != 0 {
		return nil, ErrInvalidInput
	}
	if runtime.GOOS == "windows" && config.VerifyPrivateFiles(ctx, []string{path}) != nil {
		return nil, ErrInvalidInput
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, ErrUnavailable
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if e != nil || len(b) > limit {
		return nil, ErrInvalidInput
	}
	return b, nil
}
func LoadConfig(ctx context.Context, path string) (Config, error) {
	var c Config
	b, e := ReadPrivate(ctx, path, 1<<20)
	if e != nil {
		return c, e
	}
	if e = DecodeStrict(b, &c, 1<<20); e != nil {
		return c, e
	}
	return c, c.Validate()
}
func (c Config) Validate() error {
	if !validUUID(c.InstallationID) || !filepath.IsAbs(c.BrowserPath) || !filepath.IsAbs(c.DriverDirectory) || !filepath.IsAbs(c.ProfileRoot) || !filepath.IsAbs(c.ArtifactRoot) || c.ProfileRoot == c.ArtifactRoot || strings.TrimSpace(c.Redis.Host) == "" || c.Redis.Port <= 0 || c.StoreDatabase.User != "shein_login_worker" || c.StoreDatabase.Host == "" || c.StoreDatabase.Port < 1 || c.StoreDatabase.Database == "" || len(c.Bindings) == 0 || len(c.Bindings) > 100 || c.MaxConcurrentLogins < 1 || c.MaxConcurrentLogins > 10 {
		return ErrInvalidInput
	}
	if !validOpaque(c.Auth.ProjectID) || c.Auth.ClientID == "" || !filepath.IsAbs(c.Auth.ClientSecretFile) || !filepath.IsAbs(c.Auth.ServiceTokenFile) {
		return ErrInvalidInput
	}
	for _, v := range []string{c.Auth.IssuerURL, c.Auth.AuthorizationAPIURL} {
		u, e := url.Parse(v)
		if e != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Host == "" || u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost")) {
			return ErrInvalidInput
		}
	}
	seen := map[string]bool{}
	for _, b := range c.Bindings {
		key := b.OrganizationID + "\x00" + b.StoreID
		if !validOpaque(b.OrganizationID) || !validUUID(b.StoreID) || !validUUID(b.Revision) || seen[key] || !filepath.IsAbs(b.CredentialFile) || b.ProxyFile != "" && !filepath.IsAbs(b.ProxyFile) {
			return ErrInvalidInput
		}
		seen[key] = true
		if !validLoginURL(b.LoginURL) {
			return ErrInvalidInput
		}
	}
	return nil
}
func validLoginURL(v string) bool {
	u, e := url.Parse(v)
	return e == nil && u.Scheme == "https" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && (u.Path == "" || u.Path == "/") && (u.Host == "sellerhub.shein.com" || u.Host == "sso.geiwohuo.com")
}
func (c Config) Binding(org, id string) (Binding, error) {
	for _, b := range c.Bindings {
		if b.OrganizationID == org && b.StoreID == id {
			return b, nil
		}
	}
	return Binding{}, ErrInvalidInput
}
func (c Config) Scope(b Binding) storecenter.SheinLoginScope {
	// The fingerprint covers non-secret deployment inputs, not secret contents.
	data, _ := json.Marshal(struct {
		Binding                                                 Binding
		ProfileRoot, ArtifactRoot, BrowserPath, DriverDirectory string
		Width, Height                                           int
	}{b, c.ProfileRoot, c.ArtifactRoot, c.BrowserPath, c.DriverDirectory, c.ViewportWidth, c.ViewportHeight})
	sum := sha256.Sum256(data)
	return storecenter.SheinLoginScope{OrganizationID: b.OrganizationID, StoreID: b.StoreID, InstallationID: c.InstallationID, BindingRevision: b.Revision, ConfigFingerprint: hex.EncodeToString(sum[:])}
}
func (b Binding) LoadAccount(ctx context.Context, s storecenter.SheinLoginScope) (Account, error) {
	// Producer code uses Binding/Scope only and never calls this method.
	if s.OrganizationID != b.OrganizationID || s.StoreID != b.StoreID || s.BindingRevision != b.Revision {
		return Account{}, ErrInvalidInput
	}
	var private struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	data, e := ReadPrivate(ctx, b.CredentialFile, 65536)
	if e != nil {
		return Account{}, e
	}
	if DecodeStrict(data, &private, 65536) != nil || private.Username == "" || private.Password == "" || len(private.Username) > 500 || len(private.Password) > 4096 {
		return Account{}, ErrInvalidInput
	}
	proxy := ""
	if b.ProxyFile != "" {
		data, e = ReadPrivate(ctx, b.ProxyFile, 65536)
		if e != nil {
			return Account{}, e
		}
		proxy = strings.TrimSpace(string(data))
		u, e := url.Parse(proxy)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "socks5") {
			return Account{}, ErrInvalidInput
		}
	}
	return Account{Scope: s, OrganizationID: s.OrganizationID, StoreID: s.StoreID, BindingRevision: s.BindingRevision, Generation: s.Generation, Username: private.Username, Password: private.Password, LoginURL: b.LoginURL, Proxy: proxy}, nil
}
