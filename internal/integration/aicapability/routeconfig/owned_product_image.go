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
		var option d.OwnedProductImageOption
		switch row.Template {
		case credential.GRSaiImageTemplate:
			option = d.OwnedProductImageOption{Provider: "grsai", Model: aicapability.GRSaiProductImageModel,
				Protocol: aicapability.GRSaiProductImageProtocol, Channel: credential.GRSaiImageChannel}
		case credential.ToAPIsImageTemplate:
			option = d.OwnedProductImageOption{Provider: aicapability.ToAPIsProductImageProvider, Model: aicapability.ToAPIsProductImageModel,
				Protocol: aicapability.ToAPIsProductImageProtocol, BaseURL: aicapability.ToAPIsProductImageBaseURL, Channel: credential.ToAPIsImageChannel, Region: "global"}
		default:
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
		if option.Provider == "grsai" {
			option.BaseURL = metadata.BaseURL
			option.Region = aicapability.GRSaiProductImageRegion(metadata.BaseURL)
		}
		if option.Region == "" || metadata.OrganizationID != org || metadata.ConnectionID != row.ConnectionID ||
			metadata.Channel != option.Channel || metadata.Provider != option.Provider ||
			metadata.Region != option.Region || metadata.BaseURL != option.BaseURL ||
			!credential.ValidID(metadata.Reference) || !d.ValidFact(metadata.EndpointDigest, 64) {
			return nil, d.ErrUnavailable
		}
		option.BindingID = ownedProductImageDigest(option.Protocol, org, row.ConnectionID)
		option.RevisionID = ownedProductImageDigest(option.BindingID, metadata.Reference, metadata.EndpointDigest)
		option.ConnectionID, option.ChannelName = row.ConnectionID, row.Name
		option.CredentialVersion, option.EndpointDigest = metadata.Reference, metadata.EndpointDigest
		options = append(options, option)
	}
	return options, nil
}

func ownedProductImageDigest(values ...string) string {
	encoded, _ := json.Marshal(values)
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}
