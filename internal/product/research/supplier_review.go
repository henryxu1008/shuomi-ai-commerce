package research

import (
	"context"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"task-processor/internal/product/collection"
)

const RecordSupplierReview = "record_supplier_review"

type SupplierInput struct {
	Selection   Selection            `json:"selection"`
	SKUID       string               `json:"skuId"`
	Quantity    int                  `json:"quantity"`
	Unit        string               `json:"unit"`
	Status      string               `json:"status"`
	Reason      string               `json:"reason"`
	Comparisons []SupplierComparison `json:"comparisons"`
}

type SupplierComparison struct {
	MarketKey     string `json:"marketKey"`
	SupplierKey   string `json:"supplierKey"`
	SupplierLevel string `json:"supplierLevel"`
	Result        string `json:"result"`
}

type SupplierOriginal struct {
	ItemID                                                   string
	Source                                                   collection.Source
	OfferID, CatalogPublicationID, EnvelopeHash, RawChecksum string
}

type SupplierMarket struct {
	Ref            EvidenceRef       `json:"-"`
	Revision       int64             `json:"-"`
	ASIN           string            `json:"asin"`
	Title          string            `json:"title"`
	Attributes     map[string]string `json:"attributes"`
	FixtureVersion string            `json:"fixtureVersion,omitempty"`
}

type SupplierOffer struct {
	Original       SupplierOriginal  `json:"-"`
	Revision       int64             `json:"-"`
	Title          string            `json:"title"`
	Attributes     map[string]string `json:"attributes"`
	Variants       []SupplierVariant `json:"variants"`
	FixtureVersion string            `json:"fixtureVersion,omitempty"`
}

type SupplierVariant struct {
	ID         string            `json:"id"`
	Label      string            `json:"label"`
	Attributes map[string]string `json:"attributes"`
}

type SupplierSources interface {
	ReadSupplierMarket(context.Context, collection.Scope, string, EvidenceRef) (SupplierMarket, error)
	ReadSupplierOffer(context.Context, collection.Scope, string, Selection, *SupplierOriginal) (SupplierOffer, error)
}

type SupplierReviewRecord struct {
	MemberID, LinkID, EvidenceDigest string
	Market                           EvidenceRef
	Offer                            SupplierOriginal
	MarketRevision                   int64
	Input                            SupplierInput
}

type SupplierReviewProof struct {
	Record  SupplierReviewRecord
	Expires time.Time
}

type SupplierReviewRow struct {
	Revision int64
	At       time.Time
	Record   SupplierReviewRecord
}

type SupplierReviewStore interface {
	CommitSupplier(context.Context, collection.Scope, string, string, Command, SupplierReviewProof, error) (Receipt, error)
	ListSupplierReviews(context.Context, collection.Scope, string, string, int64) ([]SupplierReviewRow, error)
	SupplierReviewLink(context.Context, collection.Scope, string, string, string) (EvidenceRef, error)
}

type SupplierReviewPreview struct {
	Revision       int64          `json:"revision"`
	LinkID         string         `json:"linkId"`
	EvidenceDigest string         `json:"evidenceDigest"`
	Market         SupplierMarket `json:"market"`
	Offer          SupplierOffer  `json:"offer"`
}

type SupplierReviewPage struct {
	Items      []SupplierReviewView `json:"items"`
	NextBefore int64                `json:"nextBefore,omitempty"`
}

type SupplierReviewView struct {
	Current bool                  `json:"current"`
	Detail  *SupplierReviewDetail `json:"detail,omitempty"`
}

type SupplierReviewDetail struct {
	MarketTitle            string                   `json:"marketTitle"`
	ASIN                   string                   `json:"asin"`
	SupplierTitle          string                   `json:"supplierTitle"`
	OfferID                string                   `json:"offerId"`
	MarketVersion          uint64                   `json:"marketVersion,string"`
	SupplierVersion        uint64                   `json:"supplierVersion,string"`
	MarketFixtureVersion   string                   `json:"marketFixtureVersion,omitempty"`
	SupplierFixtureVersion string                   `json:"supplierFixtureVersion,omitempty"`
	SKUID                  string                   `json:"skuId"`
	SKULabel               string                   `json:"skuLabel"`
	Quantity               int                      `json:"quantity"`
	Unit                   string                   `json:"unit"`
	Status                 string                   `json:"status"`
	Reason                 string                   `json:"reason"`
	At                     time.Time                `json:"at"`
	Comparisons            []SupplierComparisonView `json:"comparisons"`
}

type SupplierComparisonView struct {
	SupplierComparison
	MarketValue   string `json:"marketValue,omitempty"`
	SupplierValue string `json:"supplierValue,omitempty"`
}

var supplierDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)

func supplierText(value string, limit int) bool {
	return strings.TrimSpace(value) != "" && len(value) <= limit && utf8.ValidString(value) && !strings.ContainsRune(value, 0)
}

func ValidSupplierSelection(in Selection) bool {
	operation, ok := strings.CutPrefix(in.PublicationID, "source-run:acquisition:")
	_, fixture := collection.ResearchFixtureSupplierOperation(in.PublicationID)
	return (fixture || ok && collection.ValidID(operation)) && collection.ValidID(in.ItemID) && in.Revision > 0 &&
		in.Version > 0 && in.Version <= math.MaxInt64
}

func validSupplierInput(in *SupplierInput) bool {
	if in == nil || !ValidSupplierSelection(in.Selection) || !supplierText(in.SKUID, 128) || in.Quantity < 1 ||
		in.Quantity > 1_000_000 || in.Unit != "piece" || !supplierText(in.Reason, 2048) ||
		(in.Status != "CONFIRMED" && in.Status != "PENDING_REVIEW") || len(in.Comparisons) < 1 || len(in.Comparisons) > 5 {
		return false
	}
	seen := make(map[[3]string]bool, len(in.Comparisons))
	for _, c := range in.Comparisons {
		key := [3]string{c.MarketKey, c.SupplierKey, c.SupplierLevel}
		if !supplierText(c.MarketKey, 128) || !supplierText(c.SupplierKey, 128) || seen[key] ||
			(c.SupplierLevel != "sku" && c.SupplierLevel != "product") ||
			(c.Result != "SAME" && c.Result != "DIFFERENT" && (c.Result != "PENDING" || in.Status != "PENDING_REVIEW")) {
			return false
		}
		seen[key] = true
	}
	return true
}

func (s *Service) SupplierReviewsEnabled() bool {
	_, ok := s.Store.(SupplierReviewStore)
	return ok && s.SupplierSources != nil && s.SupplierExpiry != nil
}

func supplierLink(c Candidate) (Link, error) {
	if c.Identity.Platform != "amazon" || c.Identity.Market != "us" {
		return Link{}, ErrUnsupported
	}
	for _, l := range c.Links {
		if l.ID == c.SelectedLinkID && !l.Removed && l.Ref.Source.Kind == "amazon_data" {
			return l, nil
		}
	}
	return Link{}, ErrConflict
}

func (s *Service) supplierMarket(ctx context.Context, scope collection.Scope, project string, c Candidate, ref EvidenceRef) (SupplierMarket, error) {
	market, err := s.SupplierSources.ReadSupplierMarket(ctx, scope, project, ref)
	if err != nil {
		return SupplierMarket{}, err
	}
	if market.Ref != ref || market.ASIN != c.Identity.NativeProductID || market.Revision <= 0 {
		return SupplierMarket{}, ErrConflict
	}
	return market, nil
}

func (s *Service) SupplierPreview(ctx context.Context, scope collection.Scope, project, candidate string, in Selection) (SupplierReviewPreview, error) {
	if err := s.authorize(ctx, scope, project, true); err != nil {
		return SupplierReviewPreview{}, err
	}
	if !s.SupplierReviewsEnabled() {
		return SupplierReviewPreview{}, ErrUnsupported
	}
	if !collection.ValidID(candidate) || !ValidSupplierSelection(in) {
		return SupplierReviewPreview{}, ErrInvalid
	}
	c, err := s.Store.Load(ctx, scope, project, candidate)
	if err != nil {
		return SupplierReviewPreview{}, err
	}
	link, err := supplierLink(c)
	if err != nil {
		return SupplierReviewPreview{}, err
	}
	market, err := s.supplierMarket(ctx, scope, project, c, link.Ref)
	if err != nil {
		return SupplierReviewPreview{}, err
	}
	offer, err := s.SupplierSources.ReadSupplierOffer(ctx, scope, project, in, nil)
	if err != nil {
		return SupplierReviewPreview{}, err
	}
	if err = checkSupplierEvidencePair(market, offer); err != nil {
		return SupplierReviewPreview{}, err
	}
	if err = s.authorize(ctx, scope, project, true); err != nil {
		return SupplierReviewPreview{}, err
	}
	if err = s.checkSupplierReadRevision(ctx, scope, project, c); err != nil {
		return SupplierReviewPreview{}, err
	}
	return SupplierReviewPreview{Revision: c.Revision, LinkID: link.ID, EvidenceDigest: link.Ref.Digest(), Market: market, Offer: offer}, nil
}

func (s *Service) prepareSupplier(ctx context.Context, scope collection.Scope, project string, cmd Command) (SupplierReviewProof, error) {
	c, err := s.Store.Load(ctx, scope, project, cmd.CandidateID)
	if err != nil {
		return SupplierReviewProof{}, err
	}
	link, err := supplierLink(c)
	if err != nil {
		return SupplierReviewProof{}, err
	}
	if c.Revision != cmd.ExpectedRevision || link.ID != cmd.LinkID || link.Ref.Digest() != cmd.EvidenceDigest {
		return SupplierReviewProof{}, ErrConflict
	}
	expires := s.SupplierExpiry(ctx)
	if !time.Now().Before(expires) {
		return SupplierReviewProof{}, ErrForbidden
	}
	ctx, cancel := context.WithDeadline(ctx, expires)
	defer cancel()
	market, err := s.supplierMarket(ctx, scope, project, c, link.Ref)
	if err != nil {
		return SupplierReviewProof{}, err
	}
	offer, err := s.SupplierSources.ReadSupplierOffer(ctx, scope, project, cmd.Supplier.Selection, nil)
	if err != nil {
		return SupplierReviewProof{}, err
	}
	if err = checkSupplierEvidencePair(market, offer); err != nil {
		return SupplierReviewProof{}, err
	}
	if _, _, ok := reviewComparisons(market, offer, *cmd.Supplier); !ok {
		return SupplierReviewProof{}, ErrInvalid
	}
	if err = s.authorize(ctx, scope, project, true); err != nil {
		return SupplierReviewProof{}, err
	}
	if ctx.Err() != nil {
		return SupplierReviewProof{}, ErrUnavailable
	}
	return SupplierReviewProof{
		Expires: expires,
		Record:  SupplierReviewRecord{MemberID: scope.MemberID, LinkID: link.ID, EvidenceDigest: link.Ref.Digest(), Market: link.Ref, Offer: offer.Original, MarketRevision: market.Revision, Input: *cmd.Supplier},
	}, nil
}

func checkSupplierEvidencePair(market SupplierMarket, offer SupplierOffer) error {
	if collection.IsResearchFixtureProduct(market.Ref.Source.ProductKey) != collection.IsResearchFixtureProduct(offer.Original.Source.ProductKey) {
		return ErrUnsupported
	}
	return nil
}

func reviewComparisons(market SupplierMarket, offer SupplierOffer, in SupplierInput) (SupplierVariant, []SupplierComparisonView, bool) {
	var selected SupplierVariant
	for _, v := range offer.Variants {
		if v.ID == in.SKUID {
			if selected.ID != "" {
				return SupplierVariant{}, nil, false
			}
			selected = v
		}
	}
	if selected.ID == "" {
		return SupplierVariant{}, nil, false
	}
	result := make([]SupplierComparisonView, 0, len(in.Comparisons))
	for _, choice := range in.Comparisons {
		supplier := selected.Attributes
		if choice.SupplierLevel == "product" {
			supplier = offer.Attributes
		}
		mv, sv := market.Attributes[choice.MarketKey], supplier[choice.SupplierKey]
		if (mv == "" && sv == "") || choice.Result != "PENDING" && (!supplierText(mv, 512) || !supplierText(sv, 512)) {
			return SupplierVariant{}, nil, false
		}
		result = append(result, SupplierComparisonView{SupplierComparison: choice, MarketValue: mv, SupplierValue: sv})
	}
	return selected, result, true
}

func (s *Service) SupplierReviews(ctx context.Context, scope collection.Scope, project, candidate string, before int64) (SupplierReviewPage, error) {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return SupplierReviewPage{}, err
	}
	if !s.SupplierReviewsEnabled() {
		return SupplierReviewPage{}, ErrUnsupported
	}
	if !collection.ValidID(candidate) || before < 0 {
		return SupplierReviewPage{}, ErrInvalid
	}
	c, err := s.Store.Load(ctx, scope, project, candidate)
	if err != nil {
		return SupplierReviewPage{}, err
	}
	if c.Identity.Platform != "amazon" || c.Identity.Market != "us" {
		return SupplierReviewPage{}, ErrUnsupported
	}
	store := s.Store.(SupplierReviewStore)
	rows, err := store.ListSupplierReviews(ctx, scope, project, candidate, before)
	if err != nil {
		return SupplierReviewPage{}, err
	}
	page := SupplierReviewPage{Items: make([]SupplierReviewView, 0, len(rows))}
	if len(rows) > 5 {
		rows = rows[:5]
		page.NextBefore = rows[4].Revision
	}
	for i, row := range rows {
		item := SupplierReviewView{Current: before == 0 && i == 0 && row.Record.LinkID == c.SelectedLinkID}
		item.Detail = s.readSupplierDetail(ctx, scope, project, c, row, store)
		page.Items = append(page.Items, item)
	}
	if err = s.authorize(ctx, scope, project, false); err != nil {
		return SupplierReviewPage{}, err
	}
	if err = s.checkSupplierReadRevision(ctx, scope, project, c); err != nil {
		return SupplierReviewPage{}, err
	}
	return page, nil
}

func (s *Service) checkSupplierReadRevision(ctx context.Context, scope collection.Scope, project string, original Candidate) error {
	current, err := s.Store.Load(ctx, scope, project, original.ID)
	if err != nil {
		return err
	}
	if current.Revision != original.Revision || current.SelectedLinkID != original.SelectedLinkID {
		return ErrConflict
	}
	return nil
}

func (s *Service) readSupplierDetail(ctx context.Context, scope collection.Scope, project string, c Candidate, row SupplierReviewRow, store SupplierReviewStore) *SupplierReviewDetail {
	record := row.Record
	if record.MemberID != scope.MemberID || !validSupplierInput(&record.Input) {
		return nil
	}
	ref, err := store.SupplierReviewLink(ctx, scope, project, c.ID, record.LinkID)
	if err != nil || ref != record.Market || ref.Digest() != record.EvidenceDigest {
		return nil
	}
	market, err := s.supplierMarket(ctx, scope, project, c, ref)
	if err != nil {
		return nil
	}
	offer, err := s.SupplierSources.ReadSupplierOffer(ctx, scope, project, record.Input.Selection, &record.Offer)
	if err != nil || offer.Original != record.Offer || checkSupplierEvidencePair(market, offer) != nil {
		return nil
	}
	variant, compared, ok := reviewComparisons(market, offer, record.Input)
	if !ok {
		return nil
	}
	refAfter, err := store.SupplierReviewLink(ctx, scope, project, c.ID, record.LinkID)
	if err != nil || refAfter != ref {
		return nil
	}
	return &SupplierReviewDetail{
		MarketTitle: market.Title, ASIN: market.ASIN, SupplierTitle: offer.Title, OfferID: offer.Original.OfferID,
		MarketVersion: ref.Selection.Version, SupplierVersion: offer.Original.Source.Version, SKUID: variant.ID, SKULabel: variant.Label,
		MarketFixtureVersion: market.FixtureVersion, SupplierFixtureVersion: offer.FixtureVersion,
		Quantity: record.Input.Quantity, Unit: record.Input.Unit, Status: record.Input.Status, Reason: record.Input.Reason, At: row.At, Comparisons: compared,
	}
}
