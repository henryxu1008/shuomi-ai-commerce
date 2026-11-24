package shopify

import (
	"context"
	"errors"
	"slices"
)

var ErrInstallationUnavailable = errors.New("Shopify installation could not be verified")

const installationQuery = `query ConfirmInstallation {
  shop { id myshopifyDomain currencyCode plan { partnerDevelopment } }
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
	installation, writeProducts, err := c.ReadInstallationAccess(ctx, shop, token, appID)
	if err != nil {
		return Installation{}, err
	}
	if !writeProducts {
		return Installation{}, ErrInstallationUnavailable
	}
	return installation, nil
}

// ReadInstallationAccess exposes an explicit lack of product scope only when
// the response otherwise verifies the exact app, Shop and current installation.
func (c *AdminClient) ReadInstallationAccess(ctx context.Context, shop, token, appID string) (Installation, bool, error) {
	if !validGID(appID, "App") {
		return Installation{}, false, ErrInvalidRequest
	}
	var data struct {
		Shop struct {
			ID              string `json:"id"`
			MyshopifyDomain string `json:"myshopifyDomain"`
			CurrencyCode    string `json:"currencyCode"`
			Plan            struct {
				PartnerDevelopment bool `json:"partnerDevelopment"`
			} `json:"plan"`
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
			return Installation{}, false, err
		}
		return Installation{}, false, ErrInstallationUnavailable
	}
	if !validGID(data.Shop.ID, "Shop") || data.Shop.MyshopifyDomain != shop ||
		c.developmentStoresOnly && !data.Shop.Plan.PartnerDevelopment || !currencyPattern.MatchString(data.Shop.CurrencyCode) || !validGID(data.Current.ID, "AppInstallation") ||
		data.Current.App.ID != appID || data.Current.Scopes == nil {
		return Installation{}, false, ErrInstallationUnavailable
	}
	writeProducts := slices.ContainsFunc(data.Current.Scopes, func(scope struct {
		Handle string `json:"handle"`
	}) bool {
		return scope.Handle == productWriteScope
	})
	for _, scope := range data.Current.Scopes {
		if !oauthParameterPattern.MatchString(scope.Handle) {
			return Installation{}, false, ErrInstallationUnavailable
		}
	}
	return Installation{ShopID: data.Shop.ID, Domain: shop, Currency: data.Shop.CurrencyCode, ID: data.Current.ID, AppID: appID}, writeProducts, nil
}
