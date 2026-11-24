package routeconfiguration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
)

func (c *SharedCatalog) ReadOwnedProductImages(ctx context.Context, org string) ([]d.OwnedProductImageOption, error) {
	if c == nil || c.Managed == nil || c.Managed.Registrations == nil || c.Managed.Metadata == nil {
		return nil, nil
	}
	if !credential.ValidID(org) || org == credential.PlatformOwner {
		return nil, d.ErrInvalid
	}
	registrations, err := c.Managed.Registrations.Registrations(ctx, org)
	if err != nil {
		return nil, err
	}
	options := make([]d.OwnedProductImageOption, 0)
	for _, row := range registrations {
		if row.Template != credential.GRSaiImageTemplate {
			continue
		}
		if row.OrganizationID != org || !credential.IsRegisteredConnection(row.ConnectionID) {
			return nil, d.ErrUnavailable
		}
		metadata, err := c.Managed.Metadata.Metadata(ctx, org, row.ConnectionID)
		if errors.Is(err, credential.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !metadata.Enabled || !metadata.KeyPresent {
			continue
		}
		if metadata.OrganizationID != org || metadata.ConnectionID != row.ConnectionID ||
			metadata.Channel != credential.GRSaiImageChannel || metadata.Provider != "grsai" ||
			metadata.Region != "global" || metadata.BaseURL != aicapability.GRSaiProductImageBaseURL ||
			!credential.ValidID(metadata.Reference) || !d.ValidFact(metadata.EndpointDigest, 64) {
			return nil, d.ErrUnavailable
		}
		binding := ownedProductImageDigest(aicapability.GRSaiProductImageProtocol, org, row.ConnectionID)
		options = append(options, d.OwnedProductImageOption{BindingID: binding,
			RevisionID: ownedProductImageDigest(binding, metadata.Reference, metadata.EndpointDigest), ConnectionID: row.ConnectionID,
			CredentialVersion: metadata.Reference, EndpointDigest: metadata.EndpointDigest, ChannelName: row.Name})
	}
	return options, nil
}

func ownedProductImageDigest(values ...string) string {
	encoded, _ := json.Marshal(values)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
