package zitadel

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

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
