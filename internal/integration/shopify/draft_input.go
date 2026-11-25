package shopify

import (
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/httpimage"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/description"
)

const (
	maxDraftOptions  = 3
	maxDraftVariants = 30
	maxDraftImages   = 10
)

var (
	sourceKeyPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)
	pricePattern     = regexp.MustCompile(`^(?:0|[1-9][0-9]{0,11})(?:\.[0-9]{1,4})?$`)
)

// Draft contains the approved product values. Price strings are already
// denominated in the confirmed store currency. Image URLs must be approved by
// the asset owner for transfer to Shopify.
type Draft struct {
	SourceKey       string
	Title           string
	DescriptionHTML string
	Currency        string
	Options         []DraftOption
	Variants        []DraftVariant
	Images          []DraftImage
}

type DraftOption struct {
	Name   string
	Values []string
}

type DraftVariant struct {
	ImageIndex int
	SKU        string
	Price      string
	Values     []string
}

type DraftImage struct {
	URL string
	Alt string
}

type draftInput struct {
	Title           string         `json:"title"`
	DescriptionHTML string         `json:"descriptionHtml,omitempty"`
	Status          string         `json:"status"`
	ProductOptions  []draftOption  `json:"productOptions"`
	Variants        []draftVariant `json:"variants"`
	Files           []draftFile    `json:"files,omitempty"`
	Metafields      []metafield    `json:"metafields"`
}

type draftOption struct {
	Name   string        `json:"name"`
	Values []optionValue `json:"values"`
}

type optionValue struct {
	Name string `json:"name"`
}

type selectedValue struct {
	OptionName string `json:"optionName"`
	Name       string `json:"name"`
}

type draftVariant struct {
	File         *draftFile      `json:"file,omitempty"`
	SKU          string          `json:"sku,omitempty"`
	Price        string          `json:"price"`
	OptionValues []selectedValue `json:"optionValues"`
}

type draftFile struct {
	Filename    string `json:"filename,omitempty"`
	URL         string `json:"originalSource"`
	Alt         string `json:"alt,omitempty"`
	ContentType string `json:"contentType"`
}

type metafield struct {
	Namespace string `json:"namespace"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func sourceMetafieldValue(value string) metafield {
	return metafield{Namespace: "$app", Key: sourceMetafield, Value: value}
}

func (d Draft) input() (draftInput, error) {
	if !sourceKeyPattern.MatchString(d.SourceKey) || !validText(d.Title, 255) ||
		len(d.Options) > maxDraftOptions || len(d.Variants) == 0 || len(d.Variants) > maxDraftVariants || len(d.Images) > maxDraftImages {
		return draftInput{}, ErrInvalidRequest
	}
	result := draftInput{Title: d.Title, DescriptionHTML: d.DescriptionHTML, Status: "DRAFT",
		Metafields: []metafield{sourceMetafieldValue(d.SourceKey)}}
	approvedImages := make([]string, 0, len(d.Images))
	for _, image := range d.Images {
		if !ValidPublicDraftImageURL(image.URL) || image.Alt != "" && !validText(image.Alt, 255) || slices.Contains(approvedImages, image.URL) {
			return draftInput{}, ErrInvalidRequest
		}
		approvedImages = append(approvedImages, image.URL)
		file := draftFile{URL: image.URL, Alt: image.Alt, ContentType: "IMAGE"}
		if parsed, ok := stagedResource(image.URL); ok {
			file.Filename = path.Base(parsed.Path)
		}
		result.Files = append(result.Files, file)
	}
	if description.Validate(d.DescriptionHTML, description.HTML, approvedImages) != nil {
		return draftInput{}, ErrInvalidRequest
	}
	options, variants := d.Options, d.Variants
	if len(options) == 0 {
		if len(variants) != 1 || len(variants[0].Values) != 0 {
			return draftInput{}, ErrInvalidRequest
		}
		options = []DraftOption{{Name: "Title", Values: []string{"Default Title"}}}
		variants = []DraftVariant{{SKU: variants[0].SKU, Price: variants[0].Price, ImageIndex: variants[0].ImageIndex, Values: []string{"Default Title"}}}
	}
	if !result.setOptions(options) || !result.setVariants(variants, options) {
		return draftInput{}, ErrInvalidRequest
	}
	return result, nil
}

func (d *draftInput) setOptions(options []DraftOption) bool {
	names := make(map[string]bool, len(options))
	for _, option := range options {
		name := strings.ToLower(option.Name)
		if !validText(option.Name, 255) || names[name] || len(option.Values) == 0 || len(option.Values) > maxDraftVariants {
			return false
		}
		names[name] = true
		values := make(map[string]bool, len(option.Values))
		item := draftOption{Name: option.Name}
		for _, value := range option.Values {
			key := strings.ToLower(value)
			if !validText(value, 255) || values[key] {
				return false
			}
			values[key] = true
			item.Values = append(item.Values, optionValue{Name: value})
		}
		d.ProductOptions = append(d.ProductOptions, item)
	}
	return true
}

func (d *draftInput) setVariants(variants []DraftVariant, options []DraftOption) bool {
	combinations := make(map[string]bool, len(variants))
	for _, variant := range variants {
		if len(variant.Values) != len(options) || !pricePattern.MatchString(variant.Price) || variant.SKU != "" && !validText(variant.SKU, 128) {
			return false
		}
		if variant.ImageIndex < 0 || variant.ImageIndex > len(d.Files) {
			return false
		}
		item := draftVariant{SKU: variant.SKU, Price: variant.Price}
		if variant.ImageIndex > 0 {
			item.File = &d.Files[variant.ImageIndex-1]
		}
		for i, option := range options {
			value := variant.Values[i]
			if !slices.Contains(option.Values, value) {
				return false
			}
			item.OptionValues = append(item.OptionValues, selectedValue{OptionName: option.Name, Name: value})
		}
		key := strings.Join(variant.Values, "\x00")
		if combinations[key] {
			return false
		}
		combinations[key] = true
		d.Variants = append(d.Variants, item)
	}
	return true
}

func validText(value string, maxRunes int) bool {
	return value != "" && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maxRunes &&
		strings.TrimSpace(value) == value && strings.IndexFunc(value, unicode.IsControl) < 0
}

// ValidPublicDraftImageURL checks if Shopify can be given the original HTTPS reference.
func ValidPublicDraftImageURL(value string) bool {
	if len(value) > 2048 || strings.ContainsAny(value, "\\ \x00\r\n\t") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	_, err = httpimage.ValidatePublicHTTPSURL(value)
	return err == nil
}
