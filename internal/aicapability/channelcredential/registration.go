package channelcredential

import (
	"context"
	"net/netip"
	"net/url"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
)

const BraveWebSearchConnection = "brave-web-search-v1"

const (
	OfficialMiniTemplate    = "openai-mini-v1"
	CompatibleMiniTemplate  = "openai-compatible-mini-v1"
	OfficialOpenAIBaseURL   = "https://api.openai.com/v1"
	OfficialOpenAIChannel   = "openai-direct"
	CompatibleOpenAIChannel = "openai-compatible"
	GRSaiImageTemplate      = "grsai-product-image-v1"
	GRSaiImageChannel       = "grsai-product-image"
)

// PlatformOwner is reserved for platform-owned channels, never an IdP organization.
const PlatformOwner = "platform-model-channel-owner"

type RegistrationInput struct {
	Template string `json:"template"`
	Name     string `json:"name"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Account  string `json:"account,omitempty"`
	APIKey   string `json:"apiKey"`
}

type Registration struct {
	OrganizationID string `json:"organizationId"`
	ConnectionID   string `json:"connectionId"`
	Template       string `json:"template"`
	Name           string `json:"name"`
	Revision       int64  `json:"revision,string"`
}

type RegistrationCommand struct {
	Command
	Registration Registration
}

type RegistrationResult struct {
	Registration Registration      `json:"registration"`
	Connection   ManagedConnection `json:"connection"`
}

type RegistrationReader interface {
	Registration(context.Context, string, string) (Registration, error)
	Registrations(context.Context, string) ([]Registration, error)
}

type RegistrationWriter interface {
	CreateRegistration(context.Context, RegistrationCommand) (RegistrationResult, error)
	RegistrationReceipt(context.Context, string, string, string) (RegistrationResult, error)
	RenameRegistration(context.Context, RenameRegistrationCommand) (Registration, error)
	RenameReceipt(context.Context, string, string, string) (Registration, error)
}

type RenameRegistrationInput struct {
	Name             string `json:"name"`
	ExpectedRevision int64  `json:"expectedRevision,string"`
}

type RenameRegistrationCommand struct {
	OrganizationID, ActorID, RequestKey, ConnectionID string
	Input                                             RenameRegistrationInput
}

func NewRegistrationResult(registration Registration, metadata Metadata) RegistrationResult {
	connection := publicConnection(metadata, nil)
	connection.Registration = &registration
	connection.BaseURL = metadata.BaseURL
	return RegistrationResult{Registration: registration, Connection: connection}
}

func (cmd RegistrationCommand) Validate() error {
	input := RegistrationInput{
		Template: cmd.Registration.Template, Name: cmd.Registration.Name,
		Account: cmd.Account, APIKey: cmd.Secret,
	}
	if input.Template == CompatibleMiniTemplate {
		input.BaseURL = cmd.BaseURL
	}
	expected, err := NewRegistrationCommand(cmd.OrganizationID, cmd.ActorID, cmd.Key, input)
	if err != nil || expected != cmd {
		return ErrInvalid
	}
	return nil
}

func IsRegisteredConnection(id string) bool {
	return aicapability.IsScopedCommandID("uc-", id)
}

func NewRegistrationCommand(org, actor, key string, input RegistrationInput) (RegistrationCommand, error) {
	if !ValidID(org) || !ValidID(actor) || !ValidID(key) || !ValidID(input.Name) {
		return RegistrationCommand{}, ErrInvalid
	}
	id := aicapability.ScopedCommandID("uc-", "create-channel-v1", org, actor, key)
	base, channel := OfficialOpenAIBaseURL, OfficialOpenAIChannel
	provider := "openai"
	switch input.Template {
	case OfficialMiniTemplate:
		if input.BaseURL != "" {
			return RegistrationCommand{}, ErrInvalid
		}
	case CompatibleMiniTemplate:
		base, channel = CompatibleBaseURL(input.BaseURL), CompatibleOpenAIChannel
		if base == "" {
			return RegistrationCommand{}, ErrInvalid
		}
	case GRSaiImageTemplate:
		if input.BaseURL != "" || org == PlatformOwner {
			return RegistrationCommand{}, ErrInvalid
		}
		base, channel, provider = aicapability.GRSaiProductImageBaseURL, GRSaiImageChannel, "grsai"
	default:
		return RegistrationCommand{}, ErrInvalid
	}
	account := input.Account
	if account == "" {
		account = "default"
	}
	cmd := Command{OrganizationID: org, ActorID: actor, Key: key, ConnectionID: id,
		Provider: provider, Channel: channel, Account: account, Region: "global",
		BaseURL: base, Secret: input.APIKey, Enabled: true}
	if err := cmd.Validate(); err != nil {
		return RegistrationCommand{}, err
	}
	return RegistrationCommand{
		Command: cmd,
		Registration: Registration{OrganizationID: org, ConnectionID: id, Template: input.Template,
			Name: input.Name, Revision: 1},
	}, nil
}

// CompatibleBaseURL checks the fixed Chat Completions template's URL shape.
// The transport separately qualifies every resolved address at dial time.
func CompatibleBaseURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || !ValidOrigin(raw) || u.Scheme != "https" || u.ForceQuery ||
		strings.ContainsAny(raw, "?#\\%") || u.Port() != "" && u.Port() != "443" ||
		u.Path != "/v1" && u.Path != "/v1/" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if addr, err := netip.ParseAddr(host); err == nil {
		if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.Is4In6() || addr.Zone() != "" {
			return ""
		}
		if addr.Is6() {
			host = "[" + addr.String() + "]"
		}
	} else if !validPublicDNSName(host) {
		return ""
	}
	return "https://" + host + "/v1"
}

func validPublicDNSName(host string) bool {
	if !strings.Contains(host, ".") || len(host) > 253 ||
		strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") {
		return false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, ch := range label {
			if !(ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' || ch == '-') {
				return false
			}
		}
	}
	return true
}
