package currentapplication

import (
	"context"

	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	governed "task-processor/internal/integration/aicapability/einomodel"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
)

func channelManagementDefinitions(definitions []qualified.Definition, writable bool) ([]credential.ConnectionDefinition, error) {
	connections := []credential.ConnectionDefinition{}
	positions := map[string]int{}
	origins := map[string]string{}
	for _, d := range definitions {
		if d.ConnectionID == "" || d.ReferenceVersion == "" {
			continue
		}
		key := d.Scope.OrganizationID + "\x00" + d.ConnectionID
		candidates := []string{d.ConnectionEndpoint}
		if d.Image != nil {
			candidates = append(candidates, d.Image.Endpoint)
		}
		for _, origin := range candidates {
			if origin == "" {
				continue
			}
			if original := origins[key]; original != "" && original != origin {
				return nil, credential.ErrInvalid
			}
			origins[key] = origin
		}
	}
	for _, d := range definitions {
		// Unqualified deployment anchors are not new editable connections.
		if d.ConnectionID == "" || d.ReferenceVersion == "" {
			continue
		}
		endpoint := origins[d.Scope.OrganizationID+"\x00"+d.ConnectionID]
		model, category, id := "", "text", ""
		if d.Text != nil {
			p := d.Text.ShapeProfile()
			model, id = p.ModelID, p.ClientName
			if d.Scope.Capability == string(aicapability.OperationProductImageAnalysis) {
				category = "multimodal"
			}
			if endpoint != "" && (governed.EndpointIdentityDigest(endpoint) != p.EndpointIdentityDigest || !governed.ValidAdapterEndpoint(governed.AdapterKind(p.AdapterKind), endpoint)) {
				return nil, credential.ErrInvalid
			}
		} else if d.Image != nil {
			model, category, id = d.Image.Model, "image-generation", d.Image.ClientName
			if endpoint == "" {
				endpoint = d.Image.Endpoint
			}
			if endpoint != d.Image.Endpoint {
				return nil, credential.ErrInvalid
			}
		}
		if writable && endpoint == "" {
			return nil, credential.ErrInvalid
		}
		key := d.Scope.OrganizationID + "\x00" + d.ConnectionID
		index, exists := positions[key]
		if !exists {
			index = len(connections)
			positions[key] = index
			connections = append(connections, credential.ConnectionDefinition{OrganizationID: d.Scope.OrganizationID, ConnectionID: d.ConnectionID, Provider: d.ConnectionProvider, Channel: d.Channel, Region: d.Region, BaseURL: endpoint, Models: []credential.ManagedModel{}})
		} else {
			original := connections[index]
			if original.Provider != d.ConnectionProvider || original.Channel != d.Channel || original.Region != d.Region || original.BaseURL != endpoint {
				return nil, credential.ErrInvalid
			}
		}
		entry := credential.ManagedModel{ID: id, Version: d.ReferenceVersion, Model: model, Category: category}
		duplicate := false
		for _, m := range connections[index].Models {
			duplicate = duplicate || m == entry
		}
		if !duplicate {
			connections[index].Models = append(connections[index].Models, entry)
		}
	}
	return connections, nil
}

func prepareChannelManagement(ctx context.Context, c *Config, reader, writer *gorm.DB, definitions []qualified.Definition) (*credential.Management, error) {
	if reader == nil {
		return nil, credential.ErrUnavailable
	}
	if channelstore.VerifyMetadataProjection(ctx, reader) != nil {
		return nil, credential.ErrUnavailable
	}
	metadata, e := channelstore.New(reader, nil)
	if e != nil {
		return nil, e
	}
	var publisher credential.ManagementWriter
	if writer != nil {
		if !c.channelManagementEnabled() || writer == reader || channelstore.VerifyManagementWriter(ctx, writer) != nil {
			return nil, credential.ErrUnavailable
		}
		protection, e := c.ChannelCredentials.protection()
		if e != nil {
			return nil, e
		}
		publisher, e = channelstore.New(writer, protection)
		if e != nil {
			return nil, e
		}
	}
	connections, e := channelManagementDefinitions(definitions, publisher != nil)
	if e != nil {
		return nil, e
	}
	return credential.NewManagement(metadata, publisher, connections)
}
