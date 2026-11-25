package shopify

import (
	"errors"
	"html"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/description"
)

var ErrDraft = errors.New("invalid Shopify draft")

var pricePattern = regexp.MustCompile(`^(?:0|[1-9][0-9]{0,11})(?:\.[0-9]{1,4})?$`)

// Input contains explicit prices in the displayed shop currency and references
// to approved images. The saved product supplies the remaining facts.
type Input struct {
	Currency        string   `json:"currency"`
	Prices          []string `json:"prices"`
	ImageIDs        []string `json:"imageIds,omitempty"`
	VariantImageIDs []string `json:"variantImageIds,omitempty"`
}

type Draft struct {
	SourceKey       string         `json:"sourceKey,omitempty"`
	Title           string         `json:"title"`
	DescriptionHTML string         `json:"descriptionHtml,omitempty"`
	Currency        string         `json:"currency"`
	Options         []DraftOption  `json:"options,omitempty"`
	Variants        []DraftVariant `json:"variants"`
	Images          []DraftImage   `json:"images,omitempty"`
}

type DraftOption struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}

type DraftVariant struct {
	SKU        string   `json:"sku,omitempty"`
	Price      string   `json:"price"`
	Values     []string `json:"values,omitempty"`
	ImageIndex int      `json:"imageIndex,omitempty"`
}

type DraftImage struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

type ProductObservation struct {
	ID     string   `json:"id"`
	Status string   `json:"status"`
	Media  []string `json:"media,omitempty"`
}

func (p ProductObservation) Valid() bool {
	if !productGID.MatchString(p.ID) || len(p.Media) > 10 {
		return false
	}
	switch p.Status {
	case "DRAFT", "ACTIVE", "ARCHIVED", "UNLISTED":
	default:
		return false
	}
	for _, status := range p.Media {
		switch status {
		case "UPLOADED", "PROCESSING", "READY", "FAILED", "UNKNOWN":
		default:
			return false
		}
	}
	return true
}

var productGID = regexp.MustCompile(`^gid://shopify/Product/[1-9][0-9]{0,19}$`)

func BuildDraft(input Input, snapshot catalog.ProductSnapshot, inventory asset.ApprovedAssetInventory, currency string) (Draft, error) {
	if currency == "" || input.Currency != currency || len(input.ImageIDs) > 10 || !draftText(snapshot.Title) {
		return Draft{}, ErrDraft
	}
	variants := snapshot.Variants
	if len(variants) == 0 {
		variants = []catalog.Variant{{}}
	}
	if len(variants) != len(input.Prices) || len(variants) > 30 || len(variants[0].Attributes) > 3 {
		return Draft{}, ErrDraft
	}
	if len(input.VariantImageIDs) != 0 && len(input.VariantImageIDs) != len(variants) {
		return Draft{}, ErrDraft
	}
	result := Draft{Title: snapshot.Title, Currency: currency}
	for _, attribute := range variants[0].Attributes {
		if !draftText(attribute.Name) || slices.ContainsFunc(result.Options, func(option DraftOption) bool { return strings.EqualFold(option.Name, attribute.Name) }) {
			return Draft{}, ErrDraft
		}
		result.Options = append(result.Options, DraftOption{Name: attribute.Name})
	}
	seen := make(map[string]bool, len(variants))
	for i, variant := range variants {
		if !pricePattern.MatchString(input.Prices[i]) || len(variant.Attributes) != len(result.Options) ||
			variant.SKU != "" && (!draftText(variant.SKU) || utf8.RuneCountInString(variant.SKU) > 128) {
			return Draft{}, ErrDraft
		}
		optionValues := make([]string, len(result.Options))
		for _, attribute := range variant.Attributes {
			position := slices.IndexFunc(result.Options, func(option DraftOption) bool { return option.Name == attribute.Name })
			if position < 0 || optionValues[position] != "" || !draftText(attribute.Value) {
				return Draft{}, ErrDraft
			}
			optionValues[position] = attribute.Value
			for _, value := range result.Options[position].Values {
				if value != attribute.Value && strings.EqualFold(value, attribute.Value) {
					return Draft{}, ErrDraft
				}
			}
			if !slices.Contains(result.Options[position].Values, attribute.Value) {
				result.Options[position].Values = append(result.Options[position].Values, attribute.Value)
			}
		}
		key := strings.Join(optionValues, "\x00")
		if seen[key] {
			return Draft{}, ErrDraft
		}
		seen[key] = true
		item := DraftVariant{SKU: variant.SKU, Price: input.Prices[i], Values: optionValues}
		if len(input.VariantImageIDs) > 0 && input.VariantImageIDs[i] != "" {
			item.ImageIndex = slices.Index(input.ImageIDs, input.VariantImageIDs[i]) + 1
			if item.ImageIndex == 0 {
				return Draft{}, ErrDraft
			}
		}
		result.Variants = append(result.Variants, item)
	}
	var included []string
	for _, id := range input.ImageIDs {
		index := slices.IndexFunc(inventory.Assets, func(item asset.ApprovedAsset) bool { return id == item.ID })
		if index < 0 || slices.Contains(included, inventory.Assets[index].URL) || !validDraftImageURL(inventory.Assets[index].URL) || inventory.Assets[index].Role == asset.RoleDesign {
			return Draft{}, ErrDraft
		}
		included = append(included, inventory.Assets[index].URL)
		result.Images = append(result.Images, DraftImage{URL: inventory.Assets[index].URL})
	}
	if plain := snapshot.PlainDescription(); plain != "" {
		result.DescriptionHTML = "<p>" + html.EscapeString(plain) + "</p>"
		if description.Validate(result.DescriptionHTML, description.HTML, []string{}) != nil {
			return Draft{}, ErrDraft
		}
	}
	return result, nil
}

func validDraftImageURL(value string) bool {
	if len(value) > 2048 || strings.ContainsAny(value, "\\ \x00\r\n\t") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == ""
}

func draftText(value string) bool {
	return value != "" && utf8.ValidString(value) && strings.TrimSpace(value) == value && utf8.RuneCountInString(value) <= 255 && strings.IndexFunc(value, unicode.IsControl) < 0
}
