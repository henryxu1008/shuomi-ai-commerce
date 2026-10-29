package sourcing

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	BrowserCaptureVersion         = 1
	BrowserCaptureParserVersion   = "1688-browser-dom/v2"
	BrowserCaptureV2Version       = 2
	BrowserCaptureV2ParserVersion = "1688-browser-dom/v3"
	BrowserCaptureV2Contract      = "src2b-browser-capture-v2"
	BrowserCaptureMaxBytes        = 2 * 1024 * 1024
	browserCaptureMaxStringBytes  = 8 * 1024
	browserCaptureMaxCollection   = 256
	browserCaptureMaxAggregate    = 1024
)

// ErrInvalidBrowserCapture deliberately contains no untrusted input.
var ErrInvalidBrowserCapture = errors.New("invalid browser capture")

// BrowserAcquisitionProducerKind was admitted for the Browser implementation
// slice; it does not change the Public descriptor or SRC-1 fact rules.
const BrowserAcquisitionProducerKind = "browser_acquisition"

// BrowserAcquisitionFingerprint binds the action, canonical source and server
// payload digest. The original org/actor/key operation namespace is unchanged.
func BrowserAcquisitionFingerprint(source AcquisitionSource, captureSHA256 string) (string, error) {
	return browserFingerprint(source, captureSHA256, AcquisitionContractVersion)
}

func BrowserCaptureFingerprint(version int, source AcquisitionSource, captureSHA256 string) (string, error) {
	switch version {
	case 1:
		return BrowserAcquisitionFingerprint(source, captureSHA256)
	case 2:
		return browserFingerprint(source, captureSHA256, BrowserCaptureV2Contract)
	default:
		return "", ErrInvalidBrowserCapture
	}
}

func browserFingerprint(source AcquisitionSource, captureSHA256, contract string) (string, error) {
	canonical, err := Canonical1688Source(source.URL)
	if err != nil || canonical != source {
		return "", ErrInvalidBrowserCapture
	}
	raw, err := hex.DecodeString(captureSHA256)
	if err != nil || len(raw) != sha256.Size || hex.EncodeToString(raw) != captureSHA256 {
		return "", ErrInvalidBrowserCapture
	}
	input, _ := json.Marshal([]string{contract, "browser_capture", source.URL, captureSHA256})
	digest := sha256.Sum256(input)
	return hex.EncodeToString(digest[:]), nil
}

// BrowserCaptureCommandVersion uses persisted authority, never trial hashes.
// The frozen v1 descriptor and metadata retain their original bytes.
func BrowserCaptureCommandVersion(command PublicationCommand) (int, error) {
	m := command.Envelope.RawReference.Metadata
	if command.Producer.Kind != BrowserAcquisitionProducerKind || m["channel"] != AcquisitionChannelBrowser || !acquisitionDigest.MatchString(m["capture_sha256"]) {
		return 0, ErrInvalidBrowserCapture
	}
	switch command.Producer.Version {
	case "v1":
		if m["contract_version"] == AcquisitionContractVersion && m["parser_version"] == BrowserCaptureParserVersion {
			for _, variant := range command.Envelope.ProductCandidate.Variants {
				if len(variant.Images) != 0 {
					return 0, ErrInvalidBrowserCapture
				}
			}
			return 1, nil
		}
	case "v2":
		if m["contract_version"] == BrowserCaptureV2Contract && m["parser_version"] == BrowserCaptureV2ParserVersion && command.Envelope.RawReference.ReferenceType == "browser_capture_evidence_v2" {
			return 2, nil
		}
	}
	return 0, ErrInvalidBrowserCapture
}

// BrowserCapture carries a transport receipt and the existing domain evidence.
// PayloadSHA256 is server-computed; the client's contentSHA256 is only a claim.
type BrowserCapture struct {
	Source           AcquisitionSource
	Evidence         AcquisitionEvidence
	CanonicalPayload []byte
	PayloadSHA256    string
}

// ParseBrowserCapture validates and canonicalizes the frozen Browser wire.
func ParseBrowserCapture(body []byte) (BrowserCapture, error) {
	if len(body) == 0 || len(body) > BrowserCaptureMaxBytes {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	// This header selects exactly one strict validator; it never admits fields.
	var header struct {
		CaptureVersion int `json:"captureVersion"`
	}
	if json.Unmarshal(body, &header) != nil {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	switch header.CaptureVersion {
	case 1:
		return parseBrowserCapture[browserVariantWire](body, 1, BrowserCaptureParserVersion)
	case 2:
		return parseBrowserCapture[browserVariantWireV2](body, 2, BrowserCaptureV2ParserVersion)
	default:
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
}

func parseBrowserCapture[V any](body []byte, version int, parser string) (BrowserCapture, error) {
	if len(body) == 0 || len(body) > BrowserCaptureMaxBytes || !utf8.Valid(body) || !browserUnicodeEscapesValid(body) {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	aggregate := 0
	if err := checkBrowserWire(decoder, reflect.TypeFor[browserWire[V]](), &aggregate); err != nil {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	if _, err := decoder.Token(); err != io.EOF {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	var wire browserWire[V]
	if err := json.Unmarshal(body, &wire); err != nil {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	e := &wire.Evidence
	if wire.CaptureVersion != version || e.SchemaVersion != version || e.ParserVersion != parser {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	source, err := Canonical1688Source(e.SourceURL)
	if err != nil || source.OfferID != e.OfferID {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	capturedAt, err := time.Parse(time.RFC3339Nano, e.CapturedAt)
	if err != nil || capturedAt.IsZero() {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	claim, err := hex.DecodeString(e.ContentSHA256)
	if err != nil || len(claim) != sha256.Size {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	e.SourceURL = source.URL
	e.CapturedAt = capturedAt.UTC().Format(time.RFC3339Nano)
	canonical, err := json.Marshal(wire)
	if err != nil || len(canonical) > BrowserCaptureMaxBytes {
		return BrowserCapture{}, ErrInvalidBrowserCapture
	}
	digest := sha256.Sum256(canonical)
	return BrowserCapture{
		Source:           source,
		Evidence:         e.domain(capturedAt.UTC()),
		CanonicalPayload: canonical,
		PayloadSHA256:    hex.EncodeToString(digest[:]),
	}, nil
}

// Go's decoder replaces unpaired UTF-16 escapes with U+FFFD. Reject only those
// malformed pairs before decoding so distinct input facts never collapse.
// Ordinary escaped backslashes and actual U+FFFD remain valid.
func browserUnicodeEscapesValid(body []byte) bool {
	inString := false
	for i := 0; i < len(body); i++ {
		if body[i] == '"' {
			inString = !inString
			continue
		}
		if !inString || body[i] != '\\' {
			continue
		}
		i++
		if i >= len(body) {
			return false
		}
		if body[i] != 'u' {
			continue
		}
		if i+4 >= len(body) {
			return false
		}
		unit, err := strconv.ParseUint(string(body[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			if i+10 >= len(body) || body[i+5] != '\\' || body[i+6] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(body[i+7:i+11]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 10
		} else {
			if unit >= 0xdc00 && unit <= 0xdfff {
				return false
			}
			i += 4
		}
	}
	return true
}

// These private structs are the frozen transport shape, not a second domain
// evidence model. Declaration order fixes canonical bytes. Nothing is omitted.
type browserCaptureWire = browserWire[browserVariantWire]
type browserEvidenceWire = browserEvidence[browserVariantWire]

type browserWire[V any] struct {
	CaptureVersion int                `json:"captureVersion"`
	Evidence       browserEvidence[V] `json:"evidence"`
}

type browserEvidence[V any] struct {
	SchemaVersion int                    `json:"schemaVersion"`
	SourceURL     string                 `json:"sourceURL"`
	OfferID       string                 `json:"offerID"`
	Title         *string                `json:"title"`
	Description   *string                `json:"description"`
	Attributes    []browserAttributeWire `json:"attributes"`
	Variants      []V                    `json:"variants"`
	PriceFacts    []browserPriceWire     `json:"priceFacts"`
	Images        []browserImageWire     `json:"images"`
	CapturedAt    string                 `json:"capturedAt"`
	ContentSHA256 string                 `json:"contentSHA256"`
	ParserVersion string                 `json:"parserVersion"`
	Warnings      []browserWarningWire   `json:"warnings"`
	MissingFacts  []browserMissingWire   `json:"missingFacts"`
}

type browserAttributeWire struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type browserVariantWire struct {
	SourceID   *string                `json:"sourceID"`
	SKU        *string                `json:"sku"`
	Title      *string                `json:"title"`
	Attributes []browserAttributeWire `json:"attributes"`
	Price      *browserPriceWire      `json:"price"`
}

type browserVariantWireV2 struct {
	SourceID   *string                `json:"sourceID"`
	SKU        *string                `json:"sku"`
	Title      *string                `json:"title"`
	Attributes []browserAttributeWire `json:"attributes"`
	Price      *browserPriceWire      `json:"price"`
	Images     []browserImageWire     `json:"images"`
}

type browserPriceWire struct {
	Amount      string  `json:"amount"`
	Currency    *string `json:"currency"`
	MinQuantity *string `json:"minQuantity" browser:"optional"`
}

type browserImageWire struct {
	URL  string `json:"url"`
	Role string `json:"role"`
}

type browserWarningWire struct {
	Code  string `json:"code"`
	Field string `json:"field"`
}

type browserMissingWire struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (e browserEvidence[V]) domain(capturedAt time.Time) AcquisitionEvidence {
	result := AcquisitionEvidence{
		SchemaVersion: e.SchemaVersion, SourceURL: e.SourceURL, OfferID: e.OfferID,
		Title: e.Title, Description: e.Description,
		Attributes: browserDomainAttributes(e.Attributes),
		Variants:   make([]AcquisitionVariant, len(e.Variants)),
		PriceFacts: make([]AcquisitionPrice, len(e.PriceFacts)),
		Images:     make([]AcquisitionImage, len(e.Images)),
		CapturedAt: capturedAt, ContentSHA256: e.ContentSHA256, ParserVersion: e.ParserVersion,
		Warnings:     make([]SourceWarning, len(e.Warnings)),
		MissingFacts: make([]MissingFact, len(e.MissingFacts)),
	}
	for i, item := range e.Variants {
		var v browserVariantWire
		var images []browserImageWire
		switch variant := any(item).(type) {
		case browserVariantWire:
			v = variant
		case browserVariantWireV2:
			v = browserVariantWire{SourceID: variant.SourceID, SKU: variant.SKU, Title: variant.Title, Attributes: variant.Attributes, Price: variant.Price}
			images = variant.Images
		}
		result.Variants[i] = AcquisitionVariant{
			SourceID: v.SourceID, SKU: v.SKU, Title: v.Title,
			Attributes: browserDomainAttributes(v.Attributes),
		}
		if v.Price != nil {
			price := v.Price.domain()
			result.Variants[i].Price = &price
		}
		for _, image := range images {
			result.Variants[i].Images = append(result.Variants[i].Images, AcquisitionImage{URL: image.URL, Role: image.Role})
		}
	}
	for i, p := range e.PriceFacts {
		result.PriceFacts[i] = p.domain()
	}
	for i, v := range e.Images {
		result.Images[i] = AcquisitionImage{URL: v.URL, Role: v.Role}
	}
	for i, v := range e.Warnings {
		result.Warnings[i] = SourceWarning{Code: v.Code, Field: v.Field}
	}
	for i, v := range e.MissingFacts {
		result.MissingFacts[i] = MissingFact{Field: v.Field, Reason: v.Reason}
	}
	return result
}

func browserDomainAttributes(wire []browserAttributeWire) []AcquisitionAttribute {
	result := make([]AcquisitionAttribute, len(wire))
	for i, v := range wire {
		result[i] = AcquisitionAttribute{Name: v.Name, Value: v.Value}
	}
	return result
}

func (p browserPriceWire) domain() AcquisitionPrice {
	return AcquisitionPrice{Amount: p.Amount, Currency: p.Currency, MinQuantity: p.MinQuantity}
}

// encoding/json owns syntax and typed decoding. This narrow token pass adds
// exact-case/duplicate/required-field checks and budgets that Unmarshal lacks.
// Recursion follows only the finite private wire types, never arbitrary JSON.
func checkBrowserWire(decoder *json.Decoder, shape reflect.Type, aggregate *int) error {
	return checkBrowserWireValue(decoder, shape, aggregate, false)
}

func checkBrowserWireValue(decoder *json.Decoder, shape reflect.Type, aggregate *int, nonnull bool) error {
	token, err := decoder.Token()
	if err != nil {
		return ErrInvalidBrowserCapture
	}
	if shape.Kind() == reflect.Pointer {
		if token == nil {
			if nonnull {
				return ErrInvalidBrowserCapture
			}
			return nil
		}
		shape = shape.Elem()
	}
	switch shape.Kind() {
	case reflect.Struct:
		if token != json.Delim('{') {
			return ErrInvalidBrowserCapture
		}
		fields := make(map[string]reflect.StructField, shape.NumField())
		for i := 0; i < shape.NumField(); i++ {
			field := shape.Field(i)
			fields[strings.Split(field.Tag.Get("json"), ",")[0]] = field
		}
		seen := make(map[string]bool, len(fields))
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return ErrInvalidBrowserCapture
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return ErrInvalidBrowserCapture
			}
			field, ok := fields[key]
			if !ok {
				return ErrInvalidBrowserCapture
			}
			seen[key] = true
			option := field.Tag.Get("browser")
			if option == "optional_nonnull_count" {
				*aggregate++
				if *aggregate > browserCaptureMaxAggregate {
					return ErrInvalidBrowserCapture
				}
			}
			if err := checkBrowserWireValue(decoder, field.Type, aggregate, strings.HasPrefix(option, "optional_nonnull")); err != nil {
				return err
			}
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim('}') {
			return ErrInvalidBrowserCapture
		}
		for key, field := range fields {
			if !seen[key] && !strings.HasPrefix(field.Tag.Get("browser"), "optional") {
				return ErrInvalidBrowserCapture
			}
		}
	case reflect.Slice:
		if token != json.Delim('[') {
			return ErrInvalidBrowserCapture
		}
		count := 0
		for decoder.More() {
			count++
			*aggregate += 1
			if count > browserCaptureMaxCollection || *aggregate > browserCaptureMaxAggregate {
				return ErrInvalidBrowserCapture
			}
			if err := checkBrowserWire(decoder, shape.Elem(), aggregate); err != nil {
				return err
			}
		}
		if end, err := decoder.Token(); err != nil || end != json.Delim(']') {
			return ErrInvalidBrowserCapture
		}
	case reflect.String:
		value, ok := token.(string)
		if !ok || len(value) > browserCaptureMaxStringBytes {
			return ErrInvalidBrowserCapture
		}
	case reflect.Int:
		if _, ok := token.(json.Number); !ok {
			return ErrInvalidBrowserCapture
		}
	default:
		return ErrInvalidBrowserCapture
	}
	return nil
}
