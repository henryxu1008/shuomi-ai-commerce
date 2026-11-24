// Package amazon owns credential-free rules for preparing an Amazon US listing.
package amazon

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	USMarketplace = "ATVPDKIKX0DER"
	maxAttributes = 256 << 10
)

var (
	ErrInvalidDraft = errors.New("invalid Amazon listing details")
	ErrInvalidRules = errors.New("Amazon listing rules unavailable")
	ErrUnmetRules   = errors.New("Amazon listing requirements are not met")
	productTypeName = regexp.MustCompile(`^[A-Z0-9_]{1,128}$`)
)

// DraftInput contains the exact facts confirmed by the seller. Price and
// quantity belong to the target offer, not the saved source product.
type DraftInput struct {
	ProductType string `json:"productType"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Brand       string `json:"brand"`
	GTIN        string `json:"gtin"`
	PriceCents  int    `json:"priceCents"`
	Quantity    *int   `json:"quantity"`
	MainImageID string `json:"mainImageId"`
}

type Draft struct {
	ProductType string `json:"productType"`
	SKU         string `json:"sku"`
	// Encode as base64 for JSONB so saved bytes and their fingerprint agree.
	Attributes []byte `json:"attributes"`
	WireHash   string `json:"wireHash"`
	RuleHash   string `json:"ruleHash"`
}

// SellerSKU reserves a Shuomi namespace for the same saved product and local
// Store. User, revision and Amazon application changes do not affect it.
func SellerSKU(storeID, productKey string) string {
	return "shuomi-" + digest([]byte("amazon-us-sku-v1\x00" + storeID + "\x00" + productKey))[:32]
}

func BuildDraft(in DraftInput, storeID, productKey, imageURL string, rules []byte) (Draft, error) {
	gtinType, ok := identifierType(in.GTIN)
	if !ok || !productTypeName.MatchString(in.ProductType) || storeID == "" || productKey == "" ||
		!boundedText(in.Title, 200) || !boundedText(in.Brand, 100) ||
		in.Description != "" && !boundedText(in.Description, 2000) ||
		!boundedText(in.MainImageID, 128) || !validImageURL(imageURL) ||
		in.PriceCents <= 0 || in.PriceCents > 100_000_000 || in.Quantity == nil || *in.Quantity < 0 || *in.Quantity > 100_000 {
		return Draft{}, ErrInvalidDraft
	}
	values := func(value string, localized bool) []any {
		entry := map[string]string{"value": value, "marketplace_id": USMarketplace}
		if localized {
			entry["language_tag"] = "en_US"
		}
		return []any{entry}
	}
	amount := json.Number(fmt.Sprintf("%d.%02d", in.PriceCents/100, in.PriceCents%100))
	object := map[string]any{
		"item_name":      values(in.Title, true),
		"brand":          values(in.Brand, true),
		"condition_type": values("new_new", false),
		"externally_assigned_product_identifier": []any{map[string]any{
			"marketplace_id": USMarketplace, "type": gtinType, "value": in.GTIN,
		}},
		"main_product_image_locator": []any{map[string]any{
			"marketplace_id": USMarketplace, "media_location": imageURL,
		}},
		"purchasable_offer": []any{map[string]any{
			"marketplace_id": USMarketplace, "currency": "USD",
			"our_price": []any{map[string]any{"schedule": []any{map[string]any{"value_with_tax": amount}}}},
		}},
		"fulfillment_availability": []any{map[string]any{
			"fulfillment_channel_code": "DEFAULT", "quantity": *in.Quantity,
		}},
	}
	if in.Description != "" {
		object["product_description"] = values(in.Description, true)
	}
	wire, err := json.Marshal(object)
	if err != nil || len(wire) > maxAttributes {
		return Draft{}, ErrInvalidDraft
	}
	if err := validateSchema(rules, wire); err != nil {
		return Draft{}, err
	}
	draft := Draft{ProductType: in.ProductType, SKU: SellerSKU(storeID, productKey), Attributes: wire, RuleHash: digest(rules)}
	draft.WireHash = draft.fingerprint()
	return draft, nil
}

func (d Draft) Validate() error {
	attributes := bytes.TrimSpace(d.Attributes)
	if !productTypeName.MatchString(d.ProductType) || !strings.HasPrefix(d.SKU, "shuomi-") ||
		!validDigest(d.SKU[len("shuomi-"):], 16) || !validDigest(d.RuleHash, sha256.Size) ||
		len(attributes) == 0 || len(attributes) > maxAttributes || attributes[0] != '{' || !json.Valid(attributes) ||
		d.WireHash != d.fingerprint() {
		return ErrInvalidDraft
	}
	return nil
}

func (d Draft) fingerprint() string {
	encoded, _ := json.Marshal([]any{d.ProductType, d.SKU, d.RuleHash, d.Attributes})
	return digest(encoded)
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validDigest(value string, size int) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == size && strings.ToLower(value) == value
}

func boundedText(text string, limit int) bool {
	if strings.TrimSpace(text) != text || text == "" || !utf8.ValidString(text) || len(text) > limit {
		return false
	}
	return strings.IndexFunc(text, unicode.IsControl) < 0
}

func validImageURL(raw string) bool {
	if len(raw) > 2048 {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == ""
}

func identifierType(gtin string) (string, bool) {
	var kind string
	switch len(gtin) {
	case 12:
		kind = "upc"
	case 13:
		kind = "ean"
	case 14:
		kind = "gtin"
	default:
		return "", false
	}
	sum := 0
	for i := 0; i < len(gtin); i++ {
		digit := gtin[len(gtin)-1-i]
		if digit < '0' || digit > '9' {
			return "", false
		}
		weight := 1 + 2*(i%2)
		sum += weight * int(digit-'0')
	}
	return kind, sum != 0 && sum%10 == 0
}
