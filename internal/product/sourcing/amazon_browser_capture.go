package sourcing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"task-processor/internal/product/amazonidentity"
)

const AmazonBrowserProducerKind = "amazon_browser_capture"
const AmazonBrowserContract = "amazon-browser-capture-v1"

// Private transport types deliberately retain ordered arrays and nullable facts.
// They never carry scope, producer, billing, target or asset approval authority.
type amazonCaptureWire struct {
	CaptureVersion int                `json:"captureVersion"`
	Source         string             `json:"source"`
	Evidence       amazonEvidenceWire `json:"evidence"`
}
type amazonEvidenceWire struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Site          string                 `json:"site"`
	SourceID      string                 `json:"sourceID"`
	SourceURL     string                 `json:"sourceURL"`
	Title         *string                `json:"title"`
	Description   *string                `json:"description"`
	Attributes    []browserAttributeWire `json:"attributes"`
	Variants      []amazonVariantWire    `json:"variants"`
	PriceFacts    []browserPriceWire     `json:"priceFacts"`
	Images        []browserImageWire     `json:"images"`
	Observation   amazonObservationWire  `json:"observation"`
	CapturedAt    string                 `json:"capturedAt"`
	ContentSHA256 string                 `json:"contentSHA256"`
	ParserVersion string                 `json:"parserVersion"`
	Warnings      []browserWarningWire   `json:"warnings"`
	MissingFacts  []browserMissingWire   `json:"missingFacts"`
}
type amazonVariantWire struct {
	SourceID   *string                `json:"sourceID"`
	SKU        *string                `json:"sku"`
	Title      *string                `json:"title"`
	Attributes []browserAttributeWire `json:"attributes"`
	Price      *browserPriceWire      `json:"price"`
	Images     []browserImageWire     `json:"images"`
}
type amazonObservationWire struct {
	Scope      string  `json:"scope"`
	ParentASIN *string `json:"parentASIN"`
}
type AmazonBrowserCapture struct {
	Site, ASIN, PayloadSHA256 string
	CanonicalPayload          []byte
	wire                      amazonCaptureWire
}

func ParseAmazonBrowserCapture(body []byte) (AmazonBrowserCapture, error) {
	invalid := func() (AmazonBrowserCapture, error) { return AmazonBrowserCapture{}, ErrInvalidBrowserCapture }
	if len(body) == 0 || len(body) > BrowserCaptureMaxBytes || !utf8.Valid(body) || !browserUnicodeEscapesValid(body) {
		return invalid()
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	aggregate := 0
	if checkBrowserWire(decoder, reflect.TypeFor[amazonCaptureWire](), &aggregate) != nil {
		return invalid()
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid()
	}
	var wire amazonCaptureWire
	if json.Unmarshal(body, &wire) != nil {
		return invalid()
	}
	if !amazonWireTextSafe(reflect.ValueOf(wire)) {
		return invalid()
	}
	e := &wire.Evidence
	if wire.CaptureVersion != 2 || wire.Source != "amazon" || e.SchemaVersion != 2 || e.ParserVersion != "amazon-browser-dom/v1" || !acquisitionDigest.MatchString(e.ContentSHA256) {
		return invalid()
	}
	switch e.Site {
	case "us", "ca", "uk", "de", "jp":
	default:
		return invalid()
	}
	site, err := amazonidentity.ResolveSite(e.Site)
	if err != nil {
		return invalid()
	}
	asin, err := amazonidentity.NormalizeASIN(site, e.SourceID)
	if err != nil || asin != e.SourceID || e.SourceURL != "https://"+site.Domain+"/dp/"+asin {
		return invalid()
	}
	at, err := time.Parse(time.RFC3339Nano, e.CapturedAt)
	if err != nil || at.IsZero() {
		return invalid()
	}
	// Keep original time spelling in the digest; all accepted facts are bound.
	switch e.Observation.Scope {
	case "selected_product":
		if e.Observation.ParentASIN != nil {
			return invalid()
		}
	case "partial_family":
		if e.Observation.ParentASIN == nil {
			return invalid()
		}
		p, err := amazonidentity.NormalizeASIN(site, *e.Observation.ParentASIN)
		if err != nil || p != *e.Observation.ParentASIN || p == asin {
			return invalid()
		}
	default:
		return invalid()
	}
	canonical, err := json.Marshal(wire)
	if err != nil || len(canonical) > BrowserCaptureMaxBytes {
		return invalid()
	}
	digest := sha256.Sum256(canonical)
	result := AmazonBrowserCapture{Site: e.Site, ASIN: asin, PayloadSHA256: hex.EncodeToString(digest[:]), CanonicalPayload: canonical, wire: wire}
	if _, err = result.Envelope("validation"); err != nil {
		return invalid()
	}
	return result, nil
}

// The finite private wire has no arbitrary maps or interface values. Match the
// current Catalog/Collection text contract before accepting a durable capture.
func amazonWireTextSafe(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Pointer:
		return value.IsNil() || amazonWireTextSafe(value.Elem())
	case reflect.String:
		return !strings.ContainsRune(value.String(), 0)
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if !amazonWireTextSafe(value.Field(i)) {
				return false
			}
		}
	case reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			if !amazonWireTextSafe(value.Index(i)) {
				return false
			}
		}
	}
	return true
}

func (c AmazonBrowserCapture) Envelope(captureID string) (SourceEnvelope, error) {
	e := c.wire.Evidence
	if e.Title == nil || strings.TrimSpace(*e.Title) == "" || len(*e.Title) > 2000 || captureID == "" {
		return SourceEnvelope{}, ErrInvalidBrowserCapture
	}
	attrs, err := acquisitionAttributes(browserDomainAttributes(e.Attributes))
	if err != nil {
		return SourceEnvelope{}, err
	}
	at, _ := time.Parse(time.RFC3339Nano, e.CapturedAt)
	observation, _ := json.Marshal(e.Observation)
	env := SourceEnvelope{
		Identity:         SourceIdentity{SourceType: SourceTypeCrawler, SourcePlatform: "amazon", SourceID: e.Site + ":" + e.SourceID, SourceURL: e.SourceURL},
		RawReference:     RawSourceReference{ReferenceType: AmazonBrowserContract, ReferenceID: e.SourceID, URL: e.SourceURL, Checksum: "sha256:" + e.ContentSHA256, CapturedAt: at.UTC(), Metadata: map[string]string{"channel": "browser_capture", "contract_version": AmazonBrowserContract, "parser_version": e.ParserVersion, "site": e.Site, "asin": e.SourceID, "observation": string(observation), "capture_sha256": c.PayloadSHA256, "funding": "plugin_free"}},
		ProductCandidate: ProductCandidate{Title: *e.Title, Description: acquisitionRawValue(e.Description), Attributes: attrs},
		Trace:            SourceTrace{SourceRunID: "amazon-browser-capture:" + captureID},
	}
	missing := func(field, reason string) {
		env.MissingFacts = append(env.MissingFacts, MissingFact{Field: field, Reason: reason})
		env.Warnings = append(env.Warnings, SourceWarning{Code: "missing_source_fact", Field: field, Message: reason})
	}
	images := func(input []browserImageWire) ([]AssetCandidate, error) {
		out := []AssetCandidate{}
		seen := map[string]bool{}
		for _, im := range input {
			if len(im.URL) > 2048 || !validAcquisitionImageURL(im.URL) || (im.Role != "main" && im.Role != "detail" && im.Role != "variant" && im.Role != "gallery") {
				return nil, ErrInvalidBrowserCapture
			}
			if !seen[im.URL] {
				out = append(out, AssetCandidate{URL: im.URL, Role: im.Role, MediaType: "image"})
				seen[im.URL] = true
			}
		}
		return out, nil
	}
	env.AssetCandidates, err = images(e.Images)
	if err != nil {
		return SourceEnvelope{}, err
	}
	seenIDs, seenAttrs, seenSKU := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for i, v := range e.Variants {
		if len(acquisitionRawValue(v.Title)) > 2000 || len(acquisitionRawValue(v.SKU)) > 200 {
			return SourceEnvelope{}, ErrInvalidBrowserCapture
		}
		a, err := acquisitionAttributes(browserDomainAttributes(v.Attributes))
		if err != nil {
			return SourceEnvelope{}, err
		}
		encoded, _ := json.Marshal(a)
		id := acquisitionValue(v.SourceID)
		sku := acquisitionValue(v.SKU)
		if id != "" {
			site, _ := amazonidentity.ResolveSite(e.Site)
			normal, err := amazonidentity.NormalizeASIN(site, id)
			if err != nil || normal != id || seenIDs[id] {
				return SourceEnvelope{}, ErrInvalidBrowserCapture
			}
			seenIDs[id] = true
		}
		if e.Observation.Scope == "selected_product" && (len(e.Variants) > 1 || id != "" && id != e.SourceID) {
			return SourceEnvelope{}, ErrInvalidBrowserCapture
		}
		if len(e.Variants) > 1 && (len(a) == 0 || seenAttrs[string(encoded)]) || id == "" && len(a) == 0 || sku != "" && seenSKU[sku] {
			return SourceEnvelope{}, ErrInvalidBrowserCapture
		}
		seenAttrs[string(encoded)] = true
		seenSKU[sku] = true
		candidate := ProductVariantCandidate{SourceID: id, SKU: sku, Title: acquisitionValue(v.Title), Attributes: a}
		candidate.Images, err = images(v.Images)
		if err != nil {
			return SourceEnvelope{}, err
		}
		if len(v.Images) == 0 {
			missing(fmt.Sprintf("variants.%d.images", i), "source did not associate variant images")
		}
		if id == "" {
			missing(fmt.Sprintf("variants.%d.sourceID", i), "source variant identifier absent")
		}
		if v.Price != nil {
			p := v.Price.domain()
			value, err := validateAcquisitionPrice(p)
			if err != nil {
				return SourceEnvelope{}, err
			}
			raw, _ := json.Marshal(p)
			a["source.price.fact"] = string(raw)
			a["source.price.amount"] = p.Amount
			original, ok := new(big.Rat).SetString(p.Amount)
			projected, ok2 := new(big.Rat).SetString(strconv.FormatFloat(value, 'f', -1, 64))
			if !ok || !ok2 || original.Cmp(projected) != 0 {
				missing(fmt.Sprintf("variants.%d.price", i), "exact source decimal cannot be projected as a Catalog price")
				env.Warnings = append(env.Warnings, SourceWarning{Code: "source_price_precision", Field: fmt.Sprintf("variants.%d.price", i), Message: "original decimal retained"})
			} else if p.Currency != nil && value > 0 {
				candidate.Price = value
				candidate.Currency = *p.Currency
			} else {
				missing(fmt.Sprintf("variants.%d.price", i), "price currency or positive amount absent")
			}
		} else {
			missing(fmt.Sprintf("variants.%d.price", i), "source price absent")
		}
		env.ProductCandidate.Variants = append(env.ProductCandidate.Variants, candidate)
	}
	if len(e.Variants) == 0 {
		missing("variants", "source did not supply variants")
	}
	if len(e.Images) == 0 {
		missing("images", "source did not supply product images")
	}
	if e.Description == nil {
		missing("description", "source description absent")
	}
	missing("observation", "observed scope: "+e.Observation.Scope)
	env.SupplierOrCostFacts.Facts = map[string]string{}
	for i, p := range e.PriceFacts {
		price := p.domain()
		if _, err := validateAcquisitionPrice(price); err != nil {
			return SourceEnvelope{}, err
		}
		raw, _ := json.Marshal(price)
		env.SupplierOrCostFacts.Facts[fmt.Sprintf("price_fact.%d", i)] = string(raw)
	}
	for _, m := range e.MissingFacts {
		missing(m.Field, m.Reason)
	}
	for _, w := range e.Warnings {
		env.Warnings = append(env.Warnings, SourceWarning{Code: w.Code, Field: w.Field, Message: "source reported " + w.Code})
	}
	foldSourceEnvelopeRepeats(&env)
	// The current Collection exact-detail contract is part of this path. Reject
	// before admission if derived diagnostics cannot be read back; never truncate.
	snapshot, err := ToSnapshot(env)
	if err != nil {
		return SourceEnvelope{}, err
	}
	if len(snapshot.Warnings) > 256 || snapshot.Review != nil && len(snapshot.Review.Reasons) > 128 {
		return SourceEnvelope{}, ErrInvalidBrowserCapture
	}
	for _, warning := range snapshot.Warnings {
		if len(warning.Code) > 128 || len(warning.Field) > 8192 || len(warning.Message) > 8192 {
			return SourceEnvelope{}, ErrInvalidBrowserCapture
		}
	}
	return NormalizePublicationEnvelope(env)
}
