package amazon

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/url"
	"path"
	"strings"
)

const (
	productTypesPath         = "/definitions/2020-09-01/productTypes/"
	usDefinitionDownloadHost = "selling-partner-definitions-prod-iad.s3.amazonaws.com"
)

type ProductTypeDefinition struct {
	Version     string            `json:"-"`
	Schema      json.RawMessage   `json:"-"`
	Fingerprint [sha256.Size]byte `json:"-"`
}

type productTypeDefinitionResponse struct {
	ProductType          string   `json:"productType"`
	Locale               string   `json:"locale"`
	Requirements         string   `json:"requirements"`
	RequirementsEnforced string   `json:"requirementsEnforced"`
	MarketplaceIDs       []string `json:"marketplaceIds"`
	ProductTypeVersion   struct {
		Version          string `json:"version"`
		Latest           bool   `json:"latest"`
		ReleaseCandidate bool   `json:"releaseCandidate"`
	} `json:"productTypeVersion"`
	Schema struct {
		Link struct {
			Resource string `json:"resource"`
			Verb     string `json:"verb"`
		} `json:"link"`
		Checksum string `json:"checksum"`
	} `json:"schema"`
}

// GetProductTypeDefinition retrieves the seller-specific US requirements for
// creating a standalone listing. The schema download never carries an LWA token.
func (c *Client) GetProductTypeDefinition(ctx context.Context, accessToken, sellerID, productType string) (ProductTypeDefinition, error) {
	if !validIdentifier(sellerID, 64) || !validIdentifier(productType, 128) {
		return ProductTypeDefinition{}, ErrInvalid
	}
	query := url.Values{
		"sellerId": {sellerID}, "marketplaceIds": {USMarketplace}, "locale": {"en_US"},
		"productTypeVersion": {"LATEST"}, "requirements": {"LISTING"},
		"requirementsEnforced": {"ENFORCED"}, "parentageLevel": {"NONE"},
	}
	raw, metadata, err := c.api(ctx, http.MethodGet, productTypesPath+productType, query, accessToken, nil)
	if err != nil {
		return ProductTypeDefinition{}, err
	}
	if err := responseStatus(metadata.status); err != nil {
		return ProductTypeDefinition{}, err
	}
	var definition productTypeDefinitionResponse
	if json.Unmarshal(raw, &definition) != nil || definition.ProductType != productType ||
		definition.Locale != "en_US" || definition.Requirements != "LISTING" || definition.RequirementsEnforced != "ENFORCED" ||
		len(definition.MarketplaceIDs) != 1 || definition.MarketplaceIDs[0] != USMarketplace ||
		!definition.ProductTypeVersion.Latest || definition.ProductTypeVersion.ReleaseCandidate ||
		!visibleASCII(definition.ProductTypeVersion.Version, 128) || definition.Schema.Link.Verb != http.MethodGet {
		return ProductTypeDefinition{}, ErrUnavailable
	}
	checksum, err := base64.StdEncoding.Strict().DecodeString(definition.Schema.Checksum)
	if err != nil || len(checksum) != md5.Size || !validDefinitionDownloadURL(definition.Schema.Link.Resource) {
		return ProductTypeDefinition{}, ErrUnavailable
	}
	schema, download, err := c.request(ctx, http.MethodGet, definition.Schema.Link.Resource, "", "", nil)
	if err != nil || download.status != http.StatusOK {
		return ProductTypeDefinition{}, ErrUnavailable
	}
	// Amazon's checksum is Base64 MD5; use SHA-256 as our rule fingerprint.
	md5Sum := md5.Sum(schema)
	trimmed := bytes.TrimSpace(schema)
	if !bytes.Equal(checksum, md5Sum[:]) || len(trimmed) == 0 || trimmed[0] != '{' || !json.Valid(trimmed) {
		return ProductTypeDefinition{}, ErrUnavailable
	}
	return ProductTypeDefinition{
		Version: definition.ProductTypeVersion.Version, Schema: schema, Fingerprint: sha256.Sum256(schema),
	}, nil
}

func validDefinitionDownloadURL(raw string) bool {
	if !visibleASCII(raw, 8192) {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != usDefinitionDownloadHost ||
		u.User != nil || u.Fragment != "" || !strings.HasPrefix(u.Path, "/schema/") || path.Clean(u.Path) != u.Path {
		return false
	}
	query, err := url.ParseQuery(u.RawQuery)
	signature := query["X-Amz-Signature"]
	if err != nil || len(signature) != 1 || len(signature[0]) != hex.EncodedLen(sha256.Size) {
		return false
	}
	_, err = hex.DecodeString(signature[0])
	// Preserve the original signed URL. S3 verifies the signature.
	return err == nil
}
