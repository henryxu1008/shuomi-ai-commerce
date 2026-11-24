package shopify

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"
)

const (
	sourceMetafield = "product_upload_source"
	productFields   = `id status
  metafield(namespace: "$app", key: "` + sourceMetafield + `") { value }
  media(first: 10) { nodes { status } }`
	createDraftQuery = `mutation CreateShopifyDraft($input: ProductSetInput!) {
  productSet(input: $input, synchronous: true) {
    product { ` + productFields + ` }
    userErrors { code field }
  }
}`
	sourceDefinitionQuery = `query ReadShopifySourceDefinition($identifier: MetafieldDefinitionIdentifierInput!) {
  metafieldDefinition(identifier: $identifier) {
    type { name }
    capabilities { uniqueValues { enabled } }
    access { storefront }
  }
}`
	findProductQuery = `query ReadShopifyDraft($identifier: ProductIdentifierInput!) {
  productByIdentifier(identifier: $identifier) { ` + productFields + ` }
}`
	findByIDQuery = `query ReadKnownShopifyProduct($id: ID!) {
  product(id: $id) { id status media(first: 10) { nodes { status } } }
}`
)

var (
	ErrDraftOutcomeUnknown = errors.New("Shopify draft outcome is unknown")
	ErrDraftUnverified     = errors.New("Shopify draft could not be verified")
)

// DraftWrite holds the exact token used to verify the shop, installation and
// currency. A caller still needs live Store authorization and a Listing permit.
type DraftWrite struct {
	client       *AdminClient
	installation Installation
	token        string
	expiresAt    time.Time
	used         *atomic.Bool
}

func (*DraftWrite) String() string   { return "Shopify draft write [redacted]" }
func (*DraftWrite) GoString() string { return "Shopify draft write [redacted]" }

func (c *AdminClient) VerifyDraftWrite(ctx context.Context, expected Installation, token string) (*DraftWrite, error) {
	if !currencyPattern.MatchString(expected.Currency) {
		return nil, ErrInvalidRequest
	}
	current, err := c.matchInstallation(ctx, expected, token)
	if err != nil {
		return nil, err
	}
	if current.Currency != expected.Currency || !c.shopifySourceDefinitionReady(ctx, expected.Domain, token) {
		return nil, ErrInstallationUnavailable
	}
	return &DraftWrite{client: c, installation: expected, token: token, expiresAt: time.Now().Add(adminTimeout), used: new(atomic.Bool)}, nil
}

func (c *AdminClient) shopifySourceDefinitionReady(ctx context.Context, domain, token string) bool {
	var data struct {
		Definition *struct {
			Type struct {
				Name string `json:"name"`
			} `json:"type"`
			Capabilities struct {
				UniqueValues struct {
					Enabled bool `json:"enabled"`
				} `json:"uniqueValues"`
			} `json:"capabilities"`
			Access struct {
				Storefront string `json:"storefront"`
			} `json:"access"`
		} `json:"metafieldDefinition"`
	}
	identifier := map[string]string{"ownerType": "PRODUCT", "key": sourceMetafield}
	return c.query(ctx, domain, token, sourceDefinitionQuery, map[string]any{"identifier": identifier}, &data) == nil &&
		data.Definition != nil && data.Definition.Type.Name == "single_line_text_field" &&
		data.Definition.Capabilities.UniqueValues.Enabled && data.Definition.Access.Storefront == "NONE"
}

func (c *AdminClient) matchInstallation(ctx context.Context, expected Installation, token string) (Installation, error) {
	if !validGID(expected.ShopID, "Shop") || !validGID(expected.ID, "AppInstallation") {
		return Installation{}, ErrInvalidRequest
	}
	current, err := c.Installation(ctx, expected.Domain, token, expected.AppID)
	if err != nil {
		return Installation{}, err
	}
	if current.ShopID != expected.ShopID || current.ID != expected.ID {
		return Installation{}, ErrInstallationUnavailable
	}
	return current, nil
}

type ProductObservation struct {
	ID     string
	Status string
	Media  []string
}

func (w *DraftWrite) Create(ctx context.Context, draft Draft) (ProductObservation, error) {
	if w == nil || w.used == nil || ctx == nil || ctx.Err() != nil ||
		draft.Currency != w.installation.Currency || !time.Now().Before(w.expiresAt) {
		return ProductObservation{}, ErrInvalidRequest
	}
	input, err := draft.input()
	if err != nil {
		return ProductObservation{}, err
	}
	if ctx.Err() != nil || !time.Now().Before(w.expiresAt) || !w.used.CompareAndSwap(false, true) {
		return ProductObservation{}, ErrInvalidRequest
	}
	var data struct {
		Result *struct {
			Product    *remoteProduct    `json:"product"`
			UserErrors []json.RawMessage `json:"userErrors"`
		} `json:"productSet"`
	}
	variables := map[string]draftInput{"input": input}
	err = w.client.query(ctx, w.installation.Domain, w.token, createDraftQuery, variables, &data)
	if err != nil || data.Result == nil || data.Result.UserErrors == nil || len(data.Result.UserErrors) != 0 {
		return ProductObservation{}, ErrDraftOutcomeUnknown
	}
	if data.Result.Product == nil || data.Result.Product.Metafield.Value != draft.SourceKey {
		return ProductObservation{}, ErrDraftOutcomeUnknown
	}
	product, ok := data.Result.Product.observation()
	if !ok || product.Status != "DRAFT" {
		return ProductObservation{}, ErrDraftOutcomeUnknown
	}
	return product, nil
}

// FindDraftBySource uses a read-only custom ID lookup. An empty result cannot
// prove an earlier write did not happen, including after a Shopify reinstall.
func (c *AdminClient) FindDraftBySource(ctx context.Context, expected Installation, token, sourceKey string) (ProductObservation, error) {
	if !sourceKeyPattern.MatchString(sourceKey) {
		return ProductObservation{}, ErrInvalidRequest
	}
	if _, err := c.matchInstallation(ctx, expected, token); err != nil {
		return ProductObservation{}, err
	}
	var data struct {
		Product *remoteProduct `json:"productByIdentifier"`
	}
	variables := map[string]any{"identifier": map[string]any{"customId": sourceMetafieldValue(sourceKey)}}
	if err := c.query(ctx, expected.Domain, token, findProductQuery, variables, &data); err != nil {
		return ProductObservation{}, ErrDraftUnverified
	}
	if data.Product == nil || data.Product.Metafield.Value != sourceKey {
		return ProductObservation{}, ErrDraftUnverified
	}
	product, ok := data.Product.observation()
	if !ok {
		return ProductObservation{}, ErrDraftUnverified
	}
	return product, nil
}

// FindProductByID reads the known product from a verified receipt for the
// same shop, including when the app-owned metafield was removed.
func (c *AdminClient) FindProductByID(ctx context.Context, expected Installation, token, productID string) (ProductObservation, error) {
	if !validGID(productID, "Product") {
		return ProductObservation{}, ErrInvalidRequest
	}
	if _, err := c.matchInstallation(ctx, expected, token); err != nil {
		return ProductObservation{}, err
	}
	var data struct {
		Product *remoteProduct `json:"product"`
	}
	if err := c.query(ctx, expected.Domain, token, findByIDQuery, map[string]string{"id": productID}, &data); err != nil {
		return ProductObservation{}, ErrDraftUnverified
	}
	product, ok := data.Product.observation()
	if !ok || product.ID != productID {
		return ProductObservation{}, ErrDraftUnverified
	}
	return product, nil
}

type remoteProduct struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Metafield struct {
		Value string `json:"value"`
	} `json:"metafield"`
	Media struct {
		Nodes []struct {
			Status string `json:"status"`
		} `json:"nodes"`
	} `json:"media"`
}

func (p *remoteProduct) observation() (ProductObservation, bool) {
	if p == nil || !validGID(p.ID, "Product") {
		return ProductObservation{}, false
	}
	switch p.Status {
	case "DRAFT", "ACTIVE", "ARCHIVED", "UNLISTED":
	default:
		return ProductObservation{}, false
	}
	result := ProductObservation{ID: p.ID, Status: p.Status}
	if len(p.Media.Nodes) > maxDraftImages {
		return result, true
	}
	for _, image := range p.Media.Nodes {
		switch image.Status {
		case "UPLOADED", "PROCESSING", "READY", "FAILED":
			result.Media = append(result.Media, image.Status)
		default:
			result.Media = append(result.Media, "UNKNOWN")
		}
	}
	return result, true
}
