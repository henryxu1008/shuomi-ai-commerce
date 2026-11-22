package websearch

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
)

const (
	Connection  = credential.BraveWebSearchConnection
	BaseURL     = "https://api.search.brave.com"
	Platform    = "platform"
	Owned       = "owned"
	Held        = "held"
	Claimed     = "claimed"
	NotSent     = "not_sent"
	Unknown     = "unknown"
	Complete    = "complete"
	NoResults   = "no_results"
	Unavailable = "unavailable"
)

var (
	ErrInvalid        = errors.New("invalid web search")
	ErrConflict       = errors.New("web search changed")
	ErrNotFound       = errors.New("web search not found")
	ErrUnavailable    = errors.New("web search unavailable")
	ErrQuota          = errors.New("web search limit reached")
	ErrOutcomeUnknown = errors.New("web search outcome requires confirmation")
)

type Scope struct{ Organization, Actor, Member, Conversation string }

type Publication struct {
	Revision      int64    `json:"revision"`
	Name          string   `json:"name"`
	Country       string   `json:"country"`
	Language      string   `json:"language"`
	MaxUSDMicros  int64    `json:"maxUsdMicros"`
	TotalLimit    int      `json:"totalLimit"`
	TenantLimit   int      `json:"tenantLimit"`
	Organizations []string `json:"organizations"`
	Published     bool     `json:"published"`
}

type Selection struct {
	Revision int64  `json:"revision"`
	Source   string `json:"source"`
}

type Option struct {
	Source    string `json:"source"`
	Name      string `json:"name"`
	Available bool   `json:"available"`
	Limit     int    `json:"limit"`
	Preview   string `json:"preview,omitempty"`
}

type View struct {
	Selection Selection `json:"selection"`
	Options   []Option  `json:"options"`
}

type Result struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type Receipt struct {
	Key       string    `json:"key"`
	Query     string    `json:"query"`
	Status    string    `json:"status"`
	Source    string    `json:"source"`
	StartedAt time.Time `json:"startedAt"`
	Usage     *int      `json:"usage"`
}

type Response struct {
	Receipt Receipt  `json:"receipt"`
	Results []Result `json:"results,omitempty"`
}

type Route struct {
	Owner, Source, Country, Language, Preview         string
	Credential                                        credential.Metadata
	TotalLimit, TenantLimit                           int
	PublicationRevision, ChoiceRevision, MaxUSDMicros int64
}

type Attempt struct {
	Scope                   Scope
	Route                   Route
	Key, Query, Fingerprint string
	Deadline                time.Time
}

type Repository interface {
	Publication(context.Context) (Publication, error)
	Selection(context.Context, string) (Selection, error)
	SavePublication(context.Context, string, string, Publication) (Publication, error)
	SaveSelection(context.Context, string, string, string, Selection) (Selection, error)
	PublicationReceipt(context.Context, string, string) (Publication, error)
	SelectionReceipt(context.Context, string, string, string) (Selection, error)
	Recent(context.Context, Scope) ([]Receipt, error)
	Find(context.Context, Scope, string, string) (Receipt, error)
	Hold(context.Context, Attempt) (Receipt, bool, time.Time, error)
	Claim(context.Context, Attempt) (bool, error)
	Finish(context.Context, Attempt, string, *int) (Receipt, error)
}

type CredentialWriter interface {
	PublishSearchCredential(context.Context, credential.Command) (credential.Metadata, error)
	SearchCredentialVersionTime(context.Context, string, int64) (time.Time, error)
	Receipt(context.Context, string, string, string) (credential.Metadata, error)
}

type Searcher interface {
	Search(context.Context, string, string, string, string) ([]Result, error)
}

type Service struct {
	Repo           Repository
	Metadata       credential.MetadataReader
	Writer         CredentialWriter
	Secrets        credential.SecretReader
	Provider       Searcher
	FingerprintKey []byte
}

func ValidQuery(s string) bool {
	return s != "" && utf8.ValidString(s) && strings.TrimSpace(s) == s && utf8.RuneCountInString(s) <= 200 &&
		len(strings.Fields(s)) <= 30 && strings.IndexFunc(s, unicode.IsControl) < 0
}

func (p Publication) Validate() error {
	if p.Revision < 0 || !credential.ValidID(p.Name) || (p.Country != "US" && p.Country != "ALL") ||
		(p.Language != "en" && p.Language != "zh-hans") || p.TotalLimit < 1 || p.TotalLimit > 10_000 ||
		p.TenantLimit < 1 || p.TenantLimit > p.TotalLimit || p.MaxUSDMicros < 1 ||
		p.MaxUSDMicros > 1_000_000 || len(p.Organizations) > 64 || p.Published && len(p.Organizations) == 0 {
		return ErrInvalid
	}
	seen := make(map[string]bool)
	for _, org := range p.Organizations {
		if !credential.ValidID(org) || org == credential.PlatformOwner || seen[org] {
			return ErrInvalid
		}
		seen[org] = true
	}
	return nil
}
