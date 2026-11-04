// Package routeconfiguration qualifies existing credential bindings without
// resolving keys or constructing a provider client.
package routeconfiguration

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strings"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/bailian"
	"task-processor/internal/integration/openai"
)

type ImageAdmission struct {
	DeadlineSeconds    int64                             `json:"deadlineSeconds"`
	UnitVersion        string                            `json:"unitVersion"`
	MaximumMicropoints int64                             `json:"maximumMicropoints,string"`
	ClientName         string                            `json:"clientName"`
	Provider           string                            `json:"provider"`
	Model              string                            `json:"model"`
	Endpoint           string                            `json:"endpoint"`
	CredentialVersion  string                            `json:"credentialVersion"`
	Billing            aicapability.OfficialModelBilling `json:"billing"`
}
type Definition struct {
	// Empty means the wire ID is already the manufacturer price ID. A
	// different ID must be explicit; never strip a channel/vendor prefix.
	ManufacturerModel    string                `json:"manufacturerModel,omitempty"`
	ConnectionID         string                `json:"connectionId"`
	ConnectionProvider   string                `json:"connectionProvider"`
	Manufacturer         string                `json:"manufacturer"`
	Region               string                `json:"region"`
	ServiceType          string                `json:"serviceType"`
	Channel              string                `json:"channel"`
	PriceID              string                `json:"priceId"`
	ReferenceVersion     string                `json:"referenceVersion"`
	BoundEvidenceVersion string                `json:"boundEvidenceVersion"`
	Scope                d.Scope               `json:"scope"`
	Text                 *governed.RoutePolicy `json:"text,omitempty"`
	Image                *ImageAdmission       `json:"image,omitempty"`
	MaximumCostMicros    int64                 `json:"maximumCostMicros,omitempty"`
	// Per declared text call. This is a budget, never a fixed charge or hold.
	MaximumMicropoints int64 `json:"maximumMicropoints,omitempty"`
}
type Catalog struct {
	definitions map[d.Scope]map[string]Definition
}

func NewCatalog(defs []Definition) (*Catalog, error) {
	if len(defs) > 128 {
		return nil, d.ErrInvalid
	}
	c := &Catalog{definitions: map[d.Scope]map[string]Definition{}}
	for _, def := range defs {
		if def.ManufacturerModel != "" && !d.ValidFact(def.ManufacturerModel, 128) {
			return nil, d.ErrInvalid
		}
		if def.Image != nil && def.ManufacturerModel != "" && def.ManufacturerModel != def.Image.Model {
			return nil, d.ErrInvalid
		}
		if !def.Scope.Valid() || def.MaximumMicropoints < 0 || (def.Text == nil) == (def.Image == nil) || (def.Image != nil) != (def.Scope.Capability == d.ImageCapability) {
			return nil, d.ErrInvalid
		}
		id := ""
		if def.Text != nil {
			p := *def.Text
			def.Text = &p
			id = p.Profile.ClientName
			if !governed.ValidRouteShape(p.ShapeProfile(), aicapability.Operation(def.Scope.Capability)) || def.MaximumCostMicros <= 0 {
				return nil, d.ErrInvalid
			}
		}
		if def.Image != nil {
			i := *def.Image
			def.Image = &i
			id = i.ClientName
			if !validImage(i) {
				return nil, d.ErrInvalid
			}
		}
		if !d.ValidFact(id, 64) {
			return nil, d.ErrInvalid
		}
		if c.definitions[def.Scope] == nil {
			c.definitions[def.Scope] = map[string]Definition{}
		}
		if _, ok := c.definitions[def.Scope][id]; ok {
			return nil, d.ErrInvalid
		}
		c.definitions[def.Scope][id] = def
	}
	return c, nil
}
func validImage(i ImageAdmission) bool {
	u, e := url.Parse(i.Endpoint)
	if i.UnitVersion != aicapability.AIPointMicrounitV1 || i.MaximumMicropoints <= 0 || e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return false
	}
	if i.Provider == "bailian" {
		_, e = bailian.Endpoint(i.Endpoint)
		return e == nil && bailian.SupportsModel(i.Model)
	}
	if i.Provider == "openai" && i.Model == aicapability.FlareImageModel {
		_, e = openai.FlareImageEndpoint(i.Endpoint)
		return e == nil
	}
	return i.Provider == "grsai" && (i.Model == "gpt-image-2.5" || i.Model == "gpt-image-2-vip") && (strings.TrimSuffix(u.Path, "/") == "" || strings.TrimSuffix(u.Path, "/") == "/v1" || strings.TrimSuffix(u.Path, "/") == "/v1/api/generate")
}
func (c *Catalog) Scopes(org string) []d.Scope {
	out := []d.Scope{}
	for sc := range c.definitions {
		if sc.OrganizationID == org {
			out = append(out, sc)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Capability < out[j].Capability })
	return out
}
func (c *Catalog) Candidates(sc d.Scope) []string {
	out := []string{}
	for id := range c.definitions[sc] {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
func (c *Catalog) CredentialID(scope d.Scope, id string) (string, error) {
	if c == nil {
		return "", d.ErrUnavailable
	}
	def, ok := c.definitions[scope][id]
	if !ok || !d.ValidFact(def.ConnectionID, 128) {
		return "", d.ErrNeedsConfiguration
	}
	return def.ConnectionID, nil
}

func (c *Catalog) Qualify(sc d.Scope, id string, row d.Credential) (d.Binding, error) {
	def, ok := c.definitions[sc][id]
	if !ok || !d.ValidFact(def.ConnectionID, 128) || !d.ValidFact(def.ConnectionProvider, 128) || row.OrganizationID != sc.OrganizationID || row.ConnectionID != def.ConnectionID || row.Provider != def.ConnectionProvider || row.Channel != def.Channel || row.Region != def.Region || !row.Enabled || !row.KeyPresent || !strings.HasPrefix(row.Reference, "channel-credential-v1:") {
		return d.Binding{}, d.ErrNeedsConfiguration
	}
	if def.Text != nil {
		p := def.Text.ShapeProfile()
		if row.EndpointDigest != p.EndpointIdentityDigest {
			return d.Binding{}, d.ErrNeedsConfiguration
		}
		p.ConnectionID, p.CredentialVersion = def.ConnectionID, row.Reference
		return d.Binding{ID: id, CredentialVersion: row.Reference, Provider: p.ProviderID, Model: p.ModelID, Protocol: p.AdapterKind, UsageMappingVersion: p.UsageMappingVersion, Currency: p.Currency, MaximumCostMicros: def.MaximumCostMicros, MaximumInputBytes: p.MaximumInputBytes, MaximumOutputBytes: p.MaximumOutputBytes, MaximumPromptTokens: p.MaximumPromptTokens, MaximumCompletionTokens: p.MaximumCompletionTokens, DeadlineSeconds: int64(p.DeadlineBound.Seconds()), Text: &p}, nil
	}
	i := def.Image
	if row.EndpointDigest != governed.EndpointIdentityDigest(i.Endpoint) || i.DeadlineSeconds < 1 || i.DeadlineSeconds > 300 {
		return d.Binding{}, d.ErrNeedsConfiguration
	}
	protocol := "grsai-json-sync-v1"
	if i.Provider == "openai" {
		protocol = aicapability.FlareImageProtocol
	} else if i.Provider == "bailian" {
		protocol = bailian.Protocol
	} else if i.Provider != "grsai" {
		return d.Binding{}, d.ErrNeedsConfiguration
	}
	return d.Binding{ID: id, CredentialVersion: row.Reference, Provider: i.Provider, Model: i.Model, Protocol: protocol, DeadlineSeconds: i.DeadlineSeconds, Image: &d.ImageProfile{ConnectionID: def.ConnectionID, DeadlineSeconds: i.DeadlineSeconds, Provider: i.Provider, Model: i.Model, Protocol: protocol, EndpointDigest: row.EndpointDigest, CredentialReference: row.Reference, Billing: i.Billing}}, nil
}

type TextReader struct{ Service *d.Service }

func (r TextReader) ReadPolicy(ctx context.Context, key governed.RouteKey) (governed.RoutePolicy, error) {
	current, e := r.Service.Current(ctx, d.Scope{OrganizationID: key.OrganizationID, Capability: string(key.Operation)})
	if errors.Is(e, d.ErrNeedsConfiguration) {
		return governed.RoutePolicy{}, nil
	}
	if e != nil || current.Payload.Text == nil {
		return governed.RoutePolicy{}, d.ErrUnavailable
	}
	p := *current.Payload.Text
	return governed.RoutePolicy{Profile: p, AdmittedCredentialVersion: p.CredentialVersion, AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, nil
}
