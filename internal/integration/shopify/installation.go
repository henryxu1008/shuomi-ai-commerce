package shopify

import (
	"context"
	"errors"
	"slices"
)

var ErrInstallationUnavailable = errors.New("Shopify installation could not be verified")

const installationQuery = `query ConfirmInstallation {
  shop { id myshopifyDomain currencyCode }
  currentAppInstallation { id app { id } accessScopes { handle } }
}`

type Installation struct {
	ShopID   string
	Domain   string
	Currency string
	ID       string
	AppID    string
}

// Installation reads the scope, shop, and application represented by the
// supplied token. Store Center checks the returned installation against the
// original merchant authorization before granting a write or read purpose.
func (c *AdminClient) Installation(ctx context.Context, shop, token, appID string) (Installation, error) {
	if !validGID(appID, "App") {
		return Installation{}, ErrInvalidRequest
	}
	var data struct {
		Shop struct {
			ID              string `json:"id"`
			MyshopifyDomain string `json:"myshopifyDomain"`
			CurrencyCode    string `json:"currencyCode"`
		} `json:"shop"`
		Current struct {
			ID  string `json:"id"`
			App struct {
				ID string `json:"id"`
			} `json:"app"`
			Scopes []struct {
				Handle string `json:"handle"`
			} `json:"accessScopes"`
		} `json:"currentAppInstallation"`
	}
	if err := c.query(ctx, shop, token, installationQuery, nil, &data); err != nil {
		if errors.Is(err, ErrInvalidRequest) {
			return Installation{}, err
		}
		return Installation{}, ErrInstallationUnavailable
	}
	if !validGID(data.Shop.ID, "Shop") || data.Shop.MyshopifyDomain != shop ||
		!currencyPattern.MatchString(data.Shop.CurrencyCode) || !validGID(data.Current.ID, "AppInstallation") ||
		data.Current.App.ID != appID || !slices.ContainsFunc(data.Current.Scopes, func(scope struct {
		Handle string `json:"handle"`
	}) bool {
		return scope.Handle == productWriteScope
	}) {
		return Installation{}, ErrInstallationUnavailable
	}
	return Installation{ShopID: data.Shop.ID, Domain: shop, Currency: data.Shop.CurrencyCode, ID: data.Current.ID, AppID: appID}, nil
}
