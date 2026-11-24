package amazon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// CheckGTINUnused admits a new product only after a complete exact lookup
// returns no catalog match. It never returns ASIN data or follows pagination.
func (c *Client) CheckGTINUnused(ctx context.Context, accessToken, gtin string) (bool, error) {
	var identifierType string
	switch len(gtin) {
	case 12:
		identifierType = "UPC"
	case 13:
		identifierType = "EAN"
	case 14:
		identifierType = "GTIN"
	default:
		return false, ErrInvalid
	}
	for _, char := range gtin {
		if char < '0' || char > '9' {
			return false, ErrInvalid
		}
	}
	query := url.Values{"identifiers": {gtin}, "identifiersType": {identifierType}, "marketplaceIds": {USMarketplace}, "pageSize": {"1"}}
	raw, meta, err := c.api(ctx, http.MethodGet, "/catalog/2022-04-01/items", query, accessToken, nil)
	if err != nil {
		return false, err
	}
	if err := responseStatus(meta.status); err != nil {
		return false, err
	}
	var result struct {
		NumberOfResults *int
		Items           []json.RawMessage
		Pagination      struct{ NextToken string }
	}
	if json.Unmarshal(raw, &result) != nil || result.NumberOfResults == nil || result.Items == nil ||
		*result.NumberOfResults < len(result.Items) || *result.NumberOfResults > 0 && len(result.Items) == 0 ||
		len(result.Items) > 1 || result.Pagination.NextToken != "" && *result.NumberOfResults == 0 {
		return false, ErrUnavailable
	}
	return *result.NumberOfResults == 0, nil
}
