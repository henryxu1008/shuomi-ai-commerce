// Package tiktokshop contains the Store-owned TikTok seller authorization port.
// Token generation, binding transactions and dispatch claims belong to Store;
// this port does not authorize a merchant operation or claim an OAuth exchange.
package tiktokshop

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalid               = errors.New("invalid TikTok Shop authorization request")
	ErrUnavailable           = errors.New("TikTok Shop authorization unavailable")
	ErrAuthorizationRejected = errors.New("TikTok Shop seller authorization rejected")
	ErrOutcomeUnknown        = errors.New("TikTok Shop token operation outcome unknown")
	ErrShopIdentity          = errors.New("expected US cross-border TikTok Shop identity not verified")
)

type Application struct {
	AppKey, ServiceID, Revision, CallbackURL string
}

// TokenSet is server-only. Ordinary JSON and formatting omit both credentials;
// the Store repository must explicitly seal them with grant/generation AAD.
type TokenSet struct {
	AccessToken, RefreshToken         string `json:"-"`
	OpenID                            string
	SellerBaseRegion                  string
	UserType                          int
	GrantedScopes                     []string
	AccessExpiresAt, RefreshExpiresAt time.Time
}

func (TokenSet) String() string   { return "TikTok Shop seller credential [redacted]" }
func (TokenSet) GoString() string { return "TikTok Shop seller credential [redacted]" }

type Shop struct {
	ID, Code, Region, SellerType string
	Cipher                       string `json:"-"`
}

func (Shop) String() string   { return "TikTok Shop authorized shop [protected]" }
func (Shop) GoString() string { return "TikTok Shop authorized shop [protected]" }

// SelectUSCrossBorderShop matches the Store's explicit Seller Center code.
// Internal Shop ID, seller_base_region and list order are not that identity.
func SelectUSCrossBorderShop(shops []Shop, expectedCode string) (Shop, error) {
	var selected Shop
	count := 0
	for _, shop := range shops {
		if expectedCode != "" && shop.Code == expectedCode && shop.Region == "US" && shop.SellerType == "CROSS_BORDER" && shop.ID != "" && shop.Cipher != "" {
			selected = shop
			count++
		}
	}
	if count != 1 {
		return Shop{}, ErrShopIdentity
	}
	return selected, nil
}

// SellerAuthorization performs one protocol request per invocation. Callers
// must acquire a durable Store dispatch claim before Exchange or Refresh;
// ErrOutcomeUnknown must never lead to an automatic retry or old-token fallback.
type SellerAuthorization interface {
	Application() Application
	AuthorizationURL(state string) (string, error)
	Exchange(context.Context, string) (TokenSet, error)
	Refresh(context.Context, string) (TokenSet, error)
	AuthorizedShops(context.Context, string) ([]Shop, error)
}

// Deauthorization is evidence for the owning Store transaction, never authority
// to bind a Shop. No provider message, raw body or caller-supplied org is kept.
type Deauthorization struct {
	AppKey, ShopID, NotificationID, BodyHash string
	OccurredAt                               time.Time
}
