package storecenter

import (
	"context"
	"time"
)

type ConnectionStatus string

const (
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusConnected    ConnectionStatus = "connected"
	ConnectionStatusExpired      ConnectionStatus = "expired"
	ConnectionStatusUnavailable  ConnectionStatus = "unavailable"
)

// ConnectionStatusInput is the narrow provider boundary. ConnectionRef is an
// opaque lookup reference and is never a Store Center response field.
type ConnectionStatusInput struct {
	OrganizationID string   `json:"-"`
	StoreID        string   `json:"storeId"`
	Platform       Platform `json:"platform"`
	ConnectionRef  string   `json:"-"`
}

type ConnectionStatusProvider interface {
	Status(context.Context, ConnectionStatusInput) (ConnectionStatus, error)
}

// ShopifyServiceConnectionVerifier checks a current installation for a paid
// service command. A provider's display status does not grant this authority.
type ShopifyServiceConnectionVerifier interface {
	VerifyShopifyServiceConnection(context.Context, ConnectionStatusInput) (ShopifyBinding, error)
}

func VerifyShopifyServiceConnection(ctx context.Context, provider ConnectionStatusProvider, store *Store) (ShopifyBinding, error) {
	verifier, ok := provider.(ShopifyServiceConnectionVerifier)
	if !ok || store.Platform() != PlatformShopify {
		return ShopifyBinding{}, ErrConnectionUnavailable
	}
	input := ConnectionStatusInput{
		OrganizationID: store.OrganizationID(), StoreID: store.ID(), Platform: store.Platform(), ConnectionRef: store.ConnectionRef(),
	}
	call, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	binding, err := verifier.VerifyShopifyServiceConnection(call, input)
	if err != nil {
		return ShopifyBinding{}, err
	}
	if call.Err() != nil || binding.OrganizationID != input.OrganizationID || binding.StoreID != input.StoreID ||
		binding.ShopKey == "" || binding.ShopKey != input.ConnectionRef || binding.Generation < 1 || !binding.ShopifyInstallation.valid() {
		return ShopifyBinding{}, ErrConnectionUnavailable
	}
	return binding, nil
}

func resolveConnectionStatus(ctx context.Context, provider ConnectionStatusProvider, input ConnectionStatusInput, timeout time.Duration) ConnectionStatus {
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	status, err := provider.Status(callCtx, input)
	if err != nil {
		return ConnectionStatusUnavailable
	}
	switch status {
	case ConnectionStatusDisconnected, ConnectionStatusConnected, ConnectionStatusExpired, ConnectionStatusUnavailable:
		return status
	default:
		return ConnectionStatusUnavailable
	}
}
