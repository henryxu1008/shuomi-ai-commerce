package routeconfig

import "context"

// OwnedProductImageOption is server-side credential metadata. Keep it out of
// provider-neutral public model and channel responses.
type OwnedProductImageOption struct {
	BindingID, RevisionID                               string
	ConnectionID, CredentialVersion                     string
	EndpointDigest, ChannelName                         string
	Provider, Model, Protocol, BaseURL, Channel, Region string
}

type OwnedProductImageReader interface {
	ReadOwnedProductImages(context.Context, string) ([]OwnedProductImageOption, error)
}

func (s *SharedModelService) ReadOwnedProductImages(ctx context.Context, org string) ([]OwnedProductImageOption, error) {
	if s == nil {
		return nil, ErrUnavailable
	}
	catalog, ok := s.Catalog.(OwnedProductImageReader)
	if !ok {
		return nil, nil
	}
	return catalog.ReadOwnedProductImages(ctx, org)
}
