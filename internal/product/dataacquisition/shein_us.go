package dataacquisition

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/currency"

	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

const SHEINUSObservationEncodingVersion = "shein-us-observation-json-v1"
const sheinUSObservationVersion = "1"

var (
	sheinGoodsID  = regexp.MustCompile(`^[1-9][0-9]{0,63}$`)
	sheinAmount   = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)
	sheinChecksum = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

type SHEINUSQuerySummary struct {
	Mode        string `json:"mode"`
	CategoryRef string `json:"categoryRef,omitempty"`
	Keyword     string `json:"keyword,omitempty"`
}

// SHEINUSQuery is the frozen query for an already prepared observation.
type SHEINUSQuery struct {
	QueryRef     string              `json:"queryRef"`
	Market       string              `json:"market"`
	SourceRegion string              `json:"sourceRegion"`
	Query        SHEINUSQuerySummary `json:"query"`
}

func (q SHEINUSQuery) valid() bool {
	if !collection.ValidID(q.QueryRef) || q.Market != "US" || q.SourceRegion != "US" {
		return false
	}
	switch q.Query.Mode {
	case "category":
		return sheinGoodsID.MatchString(q.Query.CategoryRef) && q.Query.Keyword == ""
	case "keyword":
		return q.Query.CategoryRef == "" && sheinText(q.Query.Keyword, 800)
	default:
		return false
	}
}

type SHEINUSCardPrice struct {
	Amount        string  `json:"amount"`
	Currency      string  `json:"currency"`
	Basis         string  `json:"basis"`
	PromotionNote *string `json:"promotionNote"`
	TaxIncluded   *bool   `json:"taxIncluded"`
	Unit          *string `json:"unit"`
}

// Field order defines the v1 producer encoding. Readers verify saved bytes.
type SHEINUSObservation struct {
	Platform           string            `json:"platform"`
	Market             string            `json:"market"`
	NativeGoodsID      string            `json:"nativeGoodsId"`
	SourceURL          string            `json:"sourceUrl"`
	JobID              string            `json:"jobId"`
	QueryRef           string            `json:"queryRef"`
	ObservationVersion string            `json:"observationVersion"`
	ParserVersion      string            `json:"parserVersion"`
	Title              *string           `json:"title"`
	CapturedAt         *string           `json:"capturedAt"`
	VariantID          *string           `json:"variantId"`
	Price              *SHEINUSCardPrice `json:"price"`
	Missing            []string          `json:"missing"`
}

type SHEINUSPreparedRef struct {
	JobID, ItemID, ReservationID, Checksum string
}

func (r SHEINUSPreparedRef) Valid() bool {
	return collection.ValidID(r.JobID) && collection.ValidID(r.ItemID) &&
		collection.ValidID(r.ReservationID) && sheinChecksum.MatchString(r.Checksum)
}

type SHEINUSCapture struct {
	Scope                        collection.Scope
	JobID, ItemID, NativeGoodsID string
	Query                        SHEINUSQuery
	Raw                          []byte
	Source                       collection.Source
}

type SHEINUSProvenance struct {
	Source                                                   collection.Source
	JobID, QueryRef                                          string
	QuerySummary                                             SHEINUSQuerySummary
	Observation                                              SHEINUSObservation
	ObservationVersion, ObservationEncodingVersion, Checksum string
}

type SHEINUSCapturedReader interface {
	ReadSHEINUSCaptured(context.Context, collection.Scope, string) (SHEINUSCapture, error)
}

func sheinText(value string, limit int) bool {
	return len(value) <= limit && strings.TrimSpace(value) != "" &&
		utf8.ValidString(value) && !strings.ContainsRune(value, 0)
}

func (c SHEINUSCapture) observation() (SHEINUSObservation, error) {
	var o SHEINUSObservation
	if c.Scope.Validate() != nil || !collection.ValidID(c.JobID) || !collection.ValidID(c.ItemID) ||
		!sheinGoodsID.MatchString(c.NativeGoodsID) || !c.Query.valid() ||
		len(c.Raw) == 0 || len(c.Raw) > sourcing.MaxEncodedEnvelopeBytes || !utf8.Valid(c.Raw) || !json.Valid(c.Raw) {
		return o, ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&o) != nil || o.Platform != "shein" || o.Market != c.Query.Market ||
		o.NativeGoodsID != c.NativeGoodsID || o.JobID != c.JobID || o.QueryRef != c.Query.QueryRef ||
		o.ObservationVersion != sheinUSObservationVersion || !sheinText(o.ParserVersion, 128) || o.VariantID != nil {
		return SHEINUSObservation{}, ErrInvalid
	}
	link, err := url.Parse(o.SourceURL)
	if err != nil || len(o.SourceURL) > 2048 || link.Scheme != "https" || link.Host != "us.shein.com" ||
		link.User != nil || link.Fragment != "" || !strings.HasSuffix(link.EscapedPath(), "-p-"+o.NativeGoodsID+".html") {
		return SHEINUSObservation{}, ErrInvalid
	}
	if o.Title != nil && !sheinText(*o.Title, 2000) {
		return SHEINUSObservation{}, ErrInvalid
	}
	if o.CapturedAt != nil {
		if _, err := time.Parse(time.RFC3339Nano, *o.CapturedAt); err != nil {
			if _, err := time.Parse(time.DateOnly, *o.CapturedAt); err != nil {
				return SHEINUSObservation{}, ErrInvalid
			}
		}
	}
	if o.Price != nil && !o.Price.valid() {
		return SHEINUSObservation{}, ErrInvalid
	}
	if !o.validMissing() {
		return SHEINUSObservation{}, ErrInvalid
	}
	return o, nil
}

func (p SHEINUSCardPrice) valid() bool {
	_, err := currency.ParseISO(p.Currency)
	return err == nil && currencyPattern.MatchString(p.Currency) && len(p.Amount) <= 32 &&
		sheinAmount.MatchString(p.Amount) && p.Basis == "listing_display" &&
		(p.PromotionNote == nil || sheinText(*p.PromotionNote, 512)) &&
		(p.Unit == nil || sheinText(*p.Unit, 128))
}

func (o SHEINUSObservation) validMissing() bool {
	fields := map[string]bool{
		"title": o.Title == nil, "capturedAt": o.CapturedAt == nil,
		"variantId": o.VariantID == nil, "price": o.Price == nil,
	}
	if o.Price != nil {
		fields["price.promotionNote"] = o.Price.PromotionNote == nil
		fields["price.taxIncluded"] = o.Price.TaxIncluded == nil
		fields["price.unit"] = o.Price.Unit == nil
	}
	for _, field := range o.Missing {
		if !fields[field] {
			return false
		}
		delete(fields, field)
	}
	for _, missing := range fields {
		if missing {
			return false
		}
	}
	return true
}

func SHEINUSProducer() sourcing.ProducerDescriptor {
	return sourcing.ProducerDescriptor{Kind: collection.SourceSHEINUSData, Version: "v1"}
}

func (c SHEINUSCapture) productKey() string {
	return "shein-" + collection.StableID(c.Scope.OrganizationID, c.Scope.ActorID, "shein-product", "US", c.NativeGoodsID)
}

func (c SHEINUSCapture) metadata() map[string]string {
	return map[string]string{
		"jobId": c.JobID, "queryRef": c.Query.QueryRef,
		"observationVersion":         sheinUSObservationVersion,
		"observationEncodingVersion": SHEINUSObservationEncodingVersion,
	}
}

func (c SHEINUSCapture) Publication(checksum string) (sourcing.PublicationCommand, error) {
	if !sheinChecksum.MatchString(checksum) || sourcing.RawSnapshotChecksum(string(c.Raw)) != checksum {
		return sourcing.PublicationCommand{}, ErrConflict
	}
	o, err := c.observation()
	if err != nil || o.Title == nil {
		return sourcing.PublicationCommand{}, ErrInvalid
	}
	encoded, err := json.Marshal(o)
	if err != nil || !bytes.Equal(c.Raw, encoded) {
		return sourcing.PublicationCommand{}, ErrInvalid
	}
	envelope := sourcing.SourceEnvelope{
		Identity:         sourcing.SourceIdentity{SourceType: "public_web", SourcePlatform: "shein", SourceID: "US:" + o.NativeGoodsID, SourceURL: o.SourceURL},
		RawReference:     sourcing.RawSourceReference{ReferenceType: "captured_evidence", ReferenceID: c.ItemID, Checksum: checksum, Metadata: c.metadata()},
		ProductCandidate: sourcing.ProductCandidate{Title: *o.Title},
	}
	for _, field := range o.Missing {
		envelope.MissingFacts = append(envelope.MissingFacts, sourcing.MissingFact{Field: field, Reason: "not_observed"})
	}
	return sourcing.PublicationCommand{PublicationID: c.ItemID, Producer: SHEINUSProducer(), ProductKey: c.productKey(), Envelope: envelope}, nil
}

// ReadSHEINUSProvenance checks the exact original publication and current read
// authority. This path has no provider, Catalog-latest, or resource operation.
func ReadSHEINUSProvenance(ctx context.Context, p Principal, source collection.Source, live LiveAccess, captures SHEINUSCapturedReader, publications sourcing.PublicationReadStore) (SHEINUSProvenance, error) {
	if ctx == nil || p.Scope.Validate() != nil || p.CredentialID != "" || p.CredentialRevision != 0 ||
		source.Kind != collection.SourceSHEINUSData || !collection.ValidID(source.PublicationID) ||
		source.OperationID != source.PublicationID || source.ProductKey == "" || source.Version == 0 {
		return SHEINUSProvenance{}, ErrInvalid
	}
	if live == nil || captures == nil || publications == nil {
		return SHEINUSProvenance{}, ErrUnavailable
	}
	if err := live.CheckRead(ctx, p); err != nil {
		return SHEINUSProvenance{}, err
	}
	c, err := captures.ReadSHEINUSCaptured(ctx, p.Scope, source.PublicationID)
	if err != nil {
		return SHEINUSProvenance{}, err
	}
	if c.Scope != p.Scope || c.ItemID != source.OperationID {
		return SHEINUSProvenance{}, ErrNotFound
	}
	if c.Source != source {
		return SHEINUSProvenance{}, ErrConflict
	}
	pub, err := publications.Read(ctx, p.Scope.OrganizationID, source.PublicationID)
	if errors.Is(err, sourcing.ErrSourcePublicationNotFound) {
		return SHEINUSProvenance{}, ErrNotFound
	}
	if err != nil {
		return SHEINUSProvenance{}, ErrUnavailable
	}
	ref := pub.Envelope.RawReference
	receipt := pub.Receipt
	if receipt.OrganizationID != c.Scope.OrganizationID || receipt.ActorID != c.Scope.ActorID ||
		receipt.PublicationID != source.PublicationID || receipt.ProductKey != source.ProductKey ||
		receipt.ProductKey != c.productKey() || receipt.CatalogVersion != source.Version ||
		receipt.Producer != SHEINUSProducer() || ref.ReferenceType != "captured_evidence" ||
		ref.ReferenceID != c.ItemID || !sheinChecksum.MatchString(ref.Checksum) ||
		ref.Checksum != sourcing.RawSnapshotChecksum(string(c.Raw)) {
		return SHEINUSProvenance{}, ErrUnavailable
	}
	for key, value := range c.metadata() {
		if ref.Metadata[key] != value {
			return SHEINUSProvenance{}, ErrUnavailable
		}
	}
	o, err := c.observation()
	identity := pub.Envelope.Identity
	if err != nil || identity.SourceType != "public_web" || identity.SourcePlatform != "shein" ||
		identity.SourceID != "US:"+o.NativeGoodsID || identity.SourceURL != o.SourceURL {
		return SHEINUSProvenance{}, ErrUnavailable
	}
	if err := live.CheckRead(ctx, p); err != nil {
		return SHEINUSProvenance{}, err
	}
	return SHEINUSProvenance{
		Source: source, JobID: c.JobID, QueryRef: c.Query.QueryRef, QuerySummary: c.Query.Query,
		Observation: o, ObservationVersion: o.ObservationVersion,
		ObservationEncodingVersion: SHEINUSObservationEncodingVersion, Checksum: ref.Checksum,
	}, nil
}
