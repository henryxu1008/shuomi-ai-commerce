package channelcredential

import (
	"context"
	"errors"
	"sort"
)

// ManagementWriter deliberately has no Resolve method. The browser management
// consumer shares the original owner without obtaining an execution secret reader.
type ManagementWriter interface {
	Publish(context.Context, Command) (Metadata, error)
	Receipt(context.Context, string, string, string) (Metadata, error)
}

type ManagedModel struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Model    string `json:"model"`
	Category string `json:"category"`
}

// ConnectionDefinition is trusted server input, never an HTTP request DTO.
type ConnectionDefinition struct {
	OrganizationID, ConnectionID, Provider, Channel, Region, BaseURL string
	Models                                                           []ManagedModel
}

// ManagedConnection is a response whitelist. Neither the encrypted row nor
// the internal provision command can be serialized through this contract.
type ManagedConnection struct {
	OrganizationID string         `json:"organizationId"`
	ConnectionID   string         `json:"connectionId"`
	Provider       string         `json:"provider"`
	Channel        string         `json:"channel"`
	Region         string         `json:"region"`
	Account        string         `json:"account"`
	Revision       int64          `json:"revision,string"`
	Enabled        bool           `json:"enabled"`
	KeyPresent     bool           `json:"keyPresent"`
	Models         []ManagedModel `json:"models"`
}

type ManagementInput struct {
	Action           string `json:"action"`
	ExpectedRevision int64  `json:"expectedRevision,string"`
	Account          string `json:"account,omitempty"`
	APIKey           string `json:"apiKey,omitempty"`
}

type ManagementResult struct {
	OrganizationID   string            `json:"organizationId"`
	ActorID          string            `json:"actorId"`
	RequestKey       string            `json:"requestKey"`
	ConnectionID     string            `json:"connectionId"`
	ExpectedRevision int64             `json:"expectedRevision,string"`
	Action           string            `json:"action"`
	Connection       ManagedConnection `json:"connection"`
}

type Management struct {
	reader      MetadataReader
	writer      ManagementWriter
	connections []ConnectionDefinition
}

func NewManagement(reader MetadataReader, writer ManagementWriter, definitions []ConnectionDefinition) (*Management, error) {
	if reader == nil || len(definitions) > 128 {
		return nil, ErrUnavailable
	}
	m := &Management{reader: reader, writer: writer, connections: []ConnectionDefinition{}}
	seen := map[string]bool{}
	modelCount := 0
	for _, d := range definitions {
		for _, s := range []string{d.OrganizationID, d.ConnectionID, d.Provider, d.Channel, d.Region} {
			if !ValidID(s) {
				return nil, ErrInvalid
			}
		}
		if (d.BaseURL != "" && !ValidOrigin(d.BaseURL)) || writer != nil && d.BaseURL == "" {
			return nil, ErrInvalid
		}
		key := d.OrganizationID + "\x00" + d.ConnectionID
		if seen[key] {
			return nil, ErrInvalid
		}
		seen[key] = true
		d.Models = append([]ManagedModel{}, d.Models...)
		modelCount += len(d.Models)
		if modelCount > 128 {
			return nil, ErrInvalid
		}
		for _, model := range d.Models {
			if !ValidID(model.ID) || !ValidID(model.Version) || !ValidID(model.Model) || (model.Category != "text" && model.Category != "multimodal" && model.Category != "image-generation") {
				return nil, ErrInvalid
			}
		}
		m.connections = append(m.connections, d)
	}
	sort.Slice(m.connections, func(i, j int) bool {
		a, b := m.connections[i], m.connections[j]
		return a.OrganizationID < b.OrganizationID || a.OrganizationID == b.OrganizationID && a.ConnectionID < b.ConnectionID
	})
	return m, nil
}

func (m *Management) CanManage() bool { return m != nil && m.writer != nil }

func publicConnection(v Metadata, models []ManagedModel) ManagedConnection {
	return ManagedConnection{OrganizationID: v.OrganizationID, ConnectionID: v.ConnectionID, Provider: v.Provider, Channel: v.Channel, Region: v.Region, Account: v.Account, Revision: v.Revision, Enabled: v.Enabled, KeyPresent: v.KeyPresent, Models: append([]ManagedModel{}, models...)}
}

func (m *Management) List(ctx context.Context, org string) ([]ManagedConnection, error) {
	if m == nil || ctx == nil || ctx.Err() != nil || !ValidID(org) {
		return nil, ErrUnavailable
	}
	out := []ManagedConnection{}
	for _, d := range m.connections {
		if d.OrganizationID != org {
			continue
		}
		v, e := m.reader.Metadata(ctx, org, d.ConnectionID)
		if errors.Is(e, ErrNotFound) {
			v = Metadata{OrganizationID: org, ConnectionID: d.ConnectionID, Provider: d.Provider, Channel: d.Channel, Region: d.Region}
		} else if e != nil {
			return nil, ErrUnavailable
		}
		if v.OrganizationID != org || v.ConnectionID != d.ConnectionID {
			return nil, ErrUnavailable
		}
		out = append(out, publicConnection(v, d.Models))
	}
	return out, nil
}

func (m *Management) Publish(ctx context.Context, org, actor, id, key string, input ManagementInput) (ManagementResult, error) {
	if m == nil || ctx == nil || ctx.Err() != nil || m.writer == nil {
		return ManagementResult{}, ErrUnavailable
	}
	for _, s := range []string{org, actor, id, key} {
		if !ValidID(s) {
			return ManagementResult{}, ErrInvalid
		}
	}
	if input.Action != "set-key" && input.Action != "disable" || input.ExpectedRevision < 0 || input.ExpectedRevision >= 1<<62 || input.Action == "disable" && (input.APIKey != "" || input.Account != "") {
		return ManagementResult{}, ErrInvalid
	}
	var definition *ConnectionDefinition
	for i := range m.connections {
		if m.connections[i].OrganizationID == org && m.connections[i].ConnectionID == id {
			definition = &m.connections[i]
			break
		}
	}
	if definition == nil {
		return ManagementResult{}, ErrNotFound
	}
	d := *definition
	current, e := m.reader.Metadata(ctx, org, id)
	if e != nil && !errors.Is(e, ErrNotFound) {
		return ManagementResult{}, ErrUnavailable
	}
	account := input.Account
	if e == nil {
		if current.OrganizationID != org || current.ConnectionID != id || current.Provider != d.Provider || current.Channel != d.Channel || current.Region != d.Region || current.BaseURL != d.BaseURL || account != "" && account != current.Account {
			return ManagementResult{}, ErrConflict
		}
		account = current.Account
	}
	command := Command{OrganizationID: org, ActorID: actor, Key: key, ConnectionID: id, ExpectedRevision: input.ExpectedRevision, Provider: d.Provider, Channel: d.Channel, Region: d.Region, BaseURL: d.BaseURL, Account: account, Secret: input.APIKey, Enabled: input.Action == "set-key"}
	if command.Validate() != nil {
		return ManagementResult{}, ErrInvalid
	}
	v, e := m.writer.Publish(ctx, command)
	if e != nil {
		return ManagementResult{}, e
	}
	return managementResult(org, actor, id, key, v)
}

func managementResult(org, actor, id, key string, v Metadata) (ManagementResult, error) {
	if v.OrganizationID != org || v.ConnectionID != id || v.Revision < 1 || !v.KeyPresent {
		return ManagementResult{}, ErrUnavailable
	}
	action := "disable"
	if v.Enabled {
		action = "set-key"
	}
	return ManagementResult{OrganizationID: org, ActorID: actor, ConnectionID: id, RequestKey: key, ExpectedRevision: v.Revision - 1, Action: action, Connection: publicConnection(v, nil)}, nil
}

func (m *Management) Receipt(ctx context.Context, org, actor, id, key string) (ManagementResult, error) {
	if m == nil || ctx == nil || ctx.Err() != nil || m.writer == nil {
		return ManagementResult{}, ErrUnavailable
	}
	for _, s := range []string{org, actor, id, key} {
		if !ValidID(s) {
			return ManagementResult{}, ErrInvalid
		}
	}
	v, e := m.writer.Receipt(ctx, org, actor, key)
	if e != nil {
		return ManagementResult{}, e
	}
	if v.OrganizationID != org || v.ConnectionID != id {
		return ManagementResult{}, ErrNotFound
	}
	// The receipt is frozen original metadata. Do not hydrate from the current
	// connection, definitions, Key, default selection, or a provider request.
	return managementResult(org, actor, id, key, v)
}
