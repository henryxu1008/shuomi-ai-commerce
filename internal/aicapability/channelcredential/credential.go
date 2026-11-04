// Package channelcredential owns organization channel credentials. A credential
// authorizes a connection; it never selects a model or owns prices or balances.
package channelcredential

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalid        = errors.New("invalid channel credential")
	ErrUnavailable    = errors.New("channel credential unavailable")
	ErrNotFound       = errors.New("channel credential not found")
	ErrConflict       = errors.New("channel credential command conflict")
	ErrChanged        = errors.New("channel credential changed")
	ErrOutcomeUnknown = errors.New("channel credential commit outcome unknown")
)

type Metadata struct {
	OrganizationID string `json:"organizationId"`
	ConnectionID   string `json:"connectionId"`
	Provider       string `json:"provider"`
	Channel        string `json:"channel"`
	Account        string `json:"account"`
	Region         string `json:"region"`
	BaseURL        string `json:"-"`
	Revision       int64  `json:"revision"`
	Enabled        bool   `json:"enabled"`
	KeyPresent     bool   `json:"keyPresent"`
	Reference      string `json:"reference"`
	EndpointDigest string `json:"endpointDigest"`
}

type Command struct {
	OrganizationID   string `json:"organizationId"`
	ActorID          string `json:"actorId"`
	Key              string `json:"requestKey"`
	ConnectionID     string `json:"connectionId"`
	ExpectedRevision int64  `json:"expectedRevision"`
	Provider         string `json:"provider"`
	Channel          string `json:"channel"`
	Account          string `json:"account"`
	Region           string `json:"region"`
	BaseURL          string `json:"baseURL"`
	Secret           string `json:"-"`
	Enabled          bool   `json:"enabled"`
}

type Secret struct {
	Metadata Metadata
	APIKey   string `json:"-"`
}

type MetadataReader interface {
	Metadata(context.Context, string, string) (Metadata, error)
}
type SecretReader interface {
	Resolve(context.Context, string, string, string) (Secret, error)
}

func ValidID(v string) bool {
	return v != "" && len(v) <= 128 && utf8.ValidString(v) && strings.TrimSpace(v) == v && strings.IndexFunc(v, unicode.IsControl) < 0
}

func ValidOrigin(v string) bool {
	u, e := url.Parse(v)
	if e != nil || len(v) > 2048 || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || strings.TrimSpace(v) != v {
		return false
	}
	if u.Scheme == "https" {
		return true
	}
	// EXTRACT the existing guarded transport's loopback-only HTTP contract.
	// The trusted model definition must still admit this exact origin.
	ip := net.ParseIP(u.Hostname())
	return u.Scheme == "http" && (strings.EqualFold(u.Hostname(), "localhost") || ip != nil && ip.IsLoopback())
}

func (c Command) Validate() error {
	for _, v := range []string{c.OrganizationID, c.ActorID, c.Key, c.ConnectionID, c.Provider, c.Channel, c.Account, c.Region} {
		if !ValidID(v) {
			return ErrInvalid
		}
	}
	if c.ExpectedRevision < 0 || c.ExpectedRevision >= 1<<62 || !ValidOrigin(c.BaseURL) || len(c.Secret) > 4096 || strings.TrimSpace(c.Secret) != c.Secret || (!c.Enabled && c.Secret != "") || c.Enabled && c.Secret == "" {
		return ErrInvalid
	}
	return nil
}

func (m Metadata) WithIdentity() Metadata {
	endpoint := sha256.Sum256([]byte(m.BaseURL))
	m.EndpointDigest = hex.EncodeToString(endpoint[:])
	// All immutable connection facts and revision participate; no secret or model
	// is included. Rotating A->B->A never revives a prior reference.
	raw, _ := json.Marshal(struct {
		Org, ID, Provider, Channel, Account, Region, Origin string
		Revision                                            int64
	}{m.OrganizationID, m.ConnectionID, m.Provider, m.Channel, m.Account, m.Region, m.BaseURL, m.Revision})
	h := sha256.Sum256(raw)
	m.Reference = "channel-credential-v1:" + hex.EncodeToString(h[:])
	return m
}
