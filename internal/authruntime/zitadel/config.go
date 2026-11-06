package zitadel

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var publicIdentityOriginPattern = regexp.MustCompile(`^https://([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)

// IsCanonicalPublicIdentityOrigin admits only a bounded HTTPS DNS origin from
// private operator configuration. It does not grant request-selected authority.
func IsCanonicalPublicIdentityOrigin(raw string) bool {
	return len(raw) <= 261 && publicIdentityOriginPattern.MatchString(raw)
}

type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	ProjectID    string
	HTTPClient   *http.Client
}

type IntrospectionResponse struct {
	Active     bool            `json:"active"`
	Subject    string          `json:"sub"`
	ExpiresAt  int64           `json:"exp"`
	Username   string          `json:"username"`
	UserID     string          `json:"user_id"`
	ResourceID string          `json:"urn:zitadel:iam:user:resourceowner:id"`
	Roles      []string        `json:"-"`
	Extra      json.RawMessage `json:"-"`
}

type discoveryDocument struct {
	IntrospectionEndpoint string `json:"introspection_endpoint"`
}

func normalizeConfig(cfg Config) Config {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	cfg.IssuerURL = strings.TrimRight(strings.TrimSpace(cfg.IssuerURL), "/")
	cfg.ClientID = strings.TrimSpace(cfg.ClientID)
	cfg.ClientSecret = strings.TrimSpace(cfg.ClientSecret)
	cfg.ProjectID = strings.TrimSpace(cfg.ProjectID)
	return cfg
}
