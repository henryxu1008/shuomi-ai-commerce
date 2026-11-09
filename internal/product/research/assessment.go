package research

import (
	"context"
	"encoding/json"
	"strconv"
	"task-processor/internal/product/collection"
	"time"
)

const SaveAssessment = "save_assessment"

type AssessmentRef struct {
	ID          string `json:"assessmentId"`
	Revision    int64  `json:"revision"`
	ContextHash string `json:"contextHash"`
	InputHash   string `json:"inputHash"`
}
type AssessmentSupplierRef struct {
	Key      string `json:"requestKey"`
	Revision int64  `json:"candidateRevision"`
}
type AssessmentInput struct {
	Cost           CostInput              `json:"cost"`
	SupplierReview *AssessmentSupplierRef `json:"supplierReview,omitempty"`
}
type AssessmentRecord struct {
	MemberID          string                 `json:"memberId"`
	LinkID            string                 `json:"linkId"`
	Market            EvidenceRef            `json:"market"`
	CandidateRevision int64                  `json:"candidateRevision"`
	SupplierReview    *AssessmentSupplierRef `json:"supplierReview,omitempty"`
	Supplier          *SupplierReviewRecord  `json:"supplier,omitempty"`
	Input             CostInput              `json:"input"`
	Intent            AssessmentInput        `json:"intent"`
	Result            CostResult             `json:"result"`
	RuleVersion       string                 `json:"ruleVersion"`
	RequestHash       string                 `json:"requestHash"`
	InputHash         string                 `json:"inputHash"`
	ContextHash       string                 `json:"contextHash"`
}
type AssessmentRow struct {
	ID       string
	Revision int64
	At       time.Time
	Record   AssessmentRecord
}
type AssessmentDetail struct {
	Ref              AssessmentRef          `json:"ref"`
	Input            CostInput              `json:"input"`
	Result           CostResult             `json:"result"`
	RuleVersion      string                 `json:"ruleVersion"`
	Market           Selection              `json:"market"`
	SupplierReview   *AssessmentSupplierRef `json:"supplierReview,omitempty"`
	SupplierBinding  string                 `json:"supplierBinding"`
	SupplierSKU      string                 `json:"supplierSkuId,omitempty"`
	SupplierQuantity int                    `json:"supplierQuantity,omitempty"`
	SupplierUnit     string                 `json:"supplierUnit,omitempty"`
	SupplierVersion  uint64                 `json:"supplierVersion,omitempty,string"`
}
type AssessmentView struct {
	ID            string            `json:"id"`
	Revision      int64             `json:"candidateRevision"`
	At            time.Time         `json:"at"`
	Applicability string            `json:"applicability"`
	Detail        *AssessmentDetail `json:"detail,omitempty"`
}
type AssessmentPage struct {
	Items      []AssessmentView `json:"items"`
	NextBefore int64            `json:"nextBefore,omitempty"`
}
type AssessmentPreview struct {
	Input       CostInput  `json:"input"`
	Result      CostResult `json:"result"`
	RuleVersion string     `json:"ruleVersion"`
}
type AssessmentProof struct {
	Record  AssessmentRecord
	Expires time.Time
	Recheck func(context.Context) error
}
type AssessmentStore interface {
	AssessmentsReady() bool
	CommitAssessment(context.Context, collection.Scope, string, string, Command, AssessmentProof, error) (Receipt, error)
	LoadAssessment(context.Context, collection.Scope, string, string, string) (AssessmentRow, error)
	ListAssessments(context.Context, collection.Scope, string, string, int64) ([]AssessmentRow, error)
	LoadSupplierReview(context.Context, collection.Scope, string, string, string) (SupplierReviewRow, error)
	SupplierReviewLink(context.Context, collection.Scope, string, string, string) (EvidenceRef, error)
}

func ValidAssessmentRef(r AssessmentRef) bool {
	return collection.ValidID(r.ID) && r.Revision == 1 && supplierDigest.MatchString(r.ContextHash) && supplierDigest.MatchString(r.InputHash)
}
func validateAssessmentCommand(c Command) error {
	if !collection.ValidID(c.CandidateID) || c.ExpectedRevision < 1 || !collection.ValidID(c.LinkID) || !supplierDigest.MatchString(c.EvidenceDigest) || c.Assessment == nil || c.AssessmentRef != nil || c.Supplier != nil || len(c.Selections) > 0 || c.State != "" || c.Reason != "" {
		return ErrInvalid
	}
	if ref := c.Assessment.SupplierReview; ref != nil && (!collection.ValidID(ref.Key) || ref.Revision < 1) {
		return ErrInvalid
	}
	_, result := CalculateCost(c.Assessment.Cost)
	if result.Status == "INVALID_INPUT" {
		return &CostInputError{Fields: result.Fields}
	}
	raw, err := json.Marshal(c)
	if err != nil || len(raw) > 24<<10 {
		return ErrInvalid
	}
	return nil
}
func (s *Service) AssessmentsEnabled() bool {
	if s == nil || s.SupplierSources == nil || s.AssessmentExpiry == nil {
		return false
	}
	store, ok := s.Store.(AssessmentStore)
	return ok && store.AssessmentsReady()
}
func (s *Service) assessmentDeadline(ctx context.Context) (context.Context, context.CancelFunc, time.Time, error) {
	expires := s.AssessmentExpiry(ctx)
	maximum := time.Now().Add(5 * time.Second)
	if expires.After(maximum) {
		expires = maximum
	}
	if deadline, ok := ctx.Deadline(); ok && deadline.Before(expires) {
		expires = deadline
	}
	if !time.Now().Before(expires) {
		return ctx, func() {}, expires, ErrForbidden
	}
	bound, cancel := context.WithDeadline(ctx, expires)
	return bound, cancel, expires, nil
}

// checkAssessmentSources never substitutes the latest source or supplier review.
// It is also the final authorization gate executed inside the original Research transaction.
func (s *Service) checkAssessmentSources(ctx context.Context, scope collection.Scope, project, candidate string, rec AssessmentRecord, write bool) error {
	if rec.MemberID != scope.MemberID {
		return ErrNotFound
	}
	if err := s.authorize(ctx, scope, project, write); err != nil {
		return err
	}
	store := s.Store.(AssessmentStore)
	ref, err := store.SupplierReviewLink(ctx, scope, project, candidate, rec.LinkID)
	if err != nil {
		return err
	}
	if ref != rec.Market {
		return ErrConflict
	}
	market, err := s.SupplierSources.ReadSupplierMarket(ctx, scope, project, rec.Market)
	if err != nil {
		return err
	}
	if market.Ref != rec.Market || market.Revision < 1 {
		return ErrConflict
	}
	if rec.SupplierReview != nil {
		row, e := store.LoadSupplierReview(ctx, scope, project, candidate, rec.SupplierReview.Key)
		if e != nil {
			return e
		}
		if row.Revision != rec.SupplierReview.Revision || rec.Supplier == nil || row.Record.MemberID != scope.MemberID || collection.Digest(row.Record) != collection.Digest(*rec.Supplier) || row.Record.Market != rec.Market || row.Record.LinkID != rec.LinkID {
			return ErrConflict
		}
		if !validSupplierInput(&row.Record.Input) {
			return ErrConflict
		}
		offer, e := s.SupplierSources.ReadSupplierOffer(ctx, scope, project, row.Record.Input.Selection, &row.Record.Offer)
		if e != nil {
			return e
		}
		if offer.Original != row.Record.Offer || checkSupplierEvidencePair(market, offer) != nil {
			return ErrConflict
		}
		if _, _, ok := reviewComparisons(market, offer, row.Record.Input); !ok {
			return ErrConflict
		}
	}
	if err = s.authorize(ctx, scope, project, write); err != nil {
		return err
	}
	return ctx.Err()
}
func (r AssessmentRecord) Reference(id string) AssessmentRef {
	return AssessmentRef{ID: id, Revision: 1, ContextHash: r.ContextHash, InputHash: r.InputHash}
}
func (s *Service) PrepareAssessment(ctx context.Context, scope collection.Scope, project string, c Command) (AssessmentProof, error) {
	if err := s.authorize(ctx, scope, project, true); err != nil {
		return AssessmentProof{}, err
	}
	if !s.AssessmentsEnabled() {
		return AssessmentProof{}, ErrUnsupported
	}
	if validation := Validate(c); validation != nil {
		return AssessmentProof{}, validation
	}
	if c.Action != SaveAssessment && c.AssessmentRef == nil {
		return AssessmentProof{}, ErrInvalid
	}
	bounded, cancel, expires, err := s.assessmentDeadline(ctx)
	if err != nil {
		return AssessmentProof{}, err
	}
	defer cancel()
	current, err := s.Store.Load(bounded, scope, project, c.CandidateID)
	if err != nil {
		return AssessmentProof{}, err
	}
	link, err := supplierLink(current)
	if err != nil {
		return AssessmentProof{}, err
	}
	if current.Revision != c.ExpectedRevision || link.Ref.Digest() != c.EvidenceDigest || c.Action == SaveAssessment && link.ID != c.LinkID {
		return AssessmentProof{}, ErrConflict
	}
	var rec AssessmentRecord
	if c.AssessmentRef != nil {
		row, e := s.Store.(AssessmentStore).LoadAssessment(bounded, scope, project, c.CandidateID, c.AssessmentRef.ID)
		if e != nil {
			return AssessmentProof{}, e
		}
		rec = row.Record
		if rec.Reference(row.ID) != *c.AssessmentRef || rec.LinkID != link.ID || rec.Market != link.Ref {
			return AssessmentProof{}, ErrConflict
		}
	} else {
		normalized, result := CalculateCost(c.Assessment.Cost)
		if result.Status == "INVALID_INPUT" {
			return AssessmentProof{}, ErrInvalid
		}
		rec = AssessmentRecord{MemberID: scope.MemberID, LinkID: link.ID, Market: link.Ref, CandidateRevision: current.Revision, Intent: *c.Assessment, Input: normalized, Result: result, RuleVersion: CostRuleVersion, RequestHash: collection.Digest(c), InputHash: collection.Digest(normalized), SupplierReview: c.Assessment.SupplierReview}
		if rec.SupplierReview != nil {
			row, e := s.Store.(AssessmentStore).LoadSupplierReview(bounded, scope, project, c.CandidateID, rec.SupplierReview.Key)
			if e != nil {
				return AssessmentProof{}, e
			}
			if row.Revision != rec.SupplierReview.Revision || row.Record.Market != link.Ref || row.Record.LinkID != link.ID || row.Record.MemberID != scope.MemberID || normalized.Quantity != strconv.Itoa(row.Record.Input.Quantity) || row.Record.Input.Unit != "piece" {
				return AssessmentProof{}, ErrConflict
			}
			rec.Supplier = &row.Record
		}
		rec.ContextHash = collection.Digest(struct {
			Market         EvidenceRef
			Supplier       *AssessmentSupplierRef
			SupplierRecord *SupplierReviewRecord
			Input          CostInput
			Rule           string
		}{rec.Market, rec.SupplierReview, rec.Supplier, rec.Input, rec.RuleVersion})
	}
	if err = s.checkAssessmentSources(bounded, scope, project, c.CandidateID, rec, true); err != nil {
		return AssessmentProof{}, err
	}
	return AssessmentProof{Record: rec, Expires: expires, Recheck: func(gate context.Context) error {
		return s.checkAssessmentSources(gate, scope, project, c.CandidateID, rec, true)
	}}, nil
}
func (s *Service) PreviewAssessment(ctx context.Context, scope collection.Scope, project string, c Command) (AssessmentPreview, error) {
	if c.Action != SaveAssessment {
		return AssessmentPreview{}, ErrInvalid
	}
	proof, err := s.PrepareAssessment(ctx, scope, project, c)
	if err != nil {
		return AssessmentPreview{}, err
	}
	current, err := s.Store.Load(ctx, scope, project, c.CandidateID)
	if err != nil {
		return AssessmentPreview{}, err
	}
	if current.Revision != c.ExpectedRevision {
		return AssessmentPreview{}, ErrConflict
	}
	if !time.Now().Before(proof.Expires) {
		return AssessmentPreview{}, ErrUnavailable
	}
	return AssessmentPreview{Input: proof.Record.Input, Result: proof.Record.Result, RuleVersion: proof.Record.RuleVersion}, nil
}
func (s *Service) assessmentView(ctx context.Context, scope collection.Scope, project string, current Candidate, row AssessmentRow) AssessmentView {
	view := AssessmentView{ID: row.ID, Revision: row.Revision, At: row.At, Applicability: "DETAIL_UNAVAILABLE"}
	rec := row.Record
	if s.checkAssessmentSources(ctx, scope, project, current.ID, rec, false) != nil {
		return view
	}
	view.Applicability = "HISTORICAL"
	for _, link := range current.Links {
		if !link.Removed && link.ID == current.SelectedLinkID && link.ID == rec.LinkID && link.Ref == rec.Market {
			view.Applicability = "CURRENT"
		}
	}
	binding := "UNKNOWN"
	if rec.Supplier != nil {
		binding = rec.Supplier.Input.Status
	}
	view.Detail = &AssessmentDetail{Ref: rec.Reference(row.ID), Input: rec.Input, Result: rec.Result, RuleVersion: rec.RuleVersion, Market: rec.Market.Selection, SupplierReview: rec.SupplierReview, SupplierBinding: binding}
	if rec.Supplier != nil {
		view.Detail.SupplierSKU = rec.Supplier.Input.SKUID
		view.Detail.SupplierQuantity = rec.Supplier.Input.Quantity
		view.Detail.SupplierUnit = rec.Supplier.Input.Unit
		view.Detail.SupplierVersion = rec.Supplier.Offer.Source.Version
	}
	return view
}
func (s *Service) ReadAssessment(ctx context.Context, scope collection.Scope, project, candidate, id string) (AssessmentView, error) {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return AssessmentView{}, err
	}
	if !s.AssessmentsEnabled() {
		return AssessmentView{}, ErrUnsupported
	}
	if !collection.ValidID(candidate) || !collection.ValidID(id) {
		return AssessmentView{}, ErrInvalid
	}
	bounded, cancel, _, err := s.assessmentDeadline(ctx)
	if err != nil {
		return AssessmentView{}, err
	}
	defer cancel()
	current, err := s.Store.Load(bounded, scope, project, candidate)
	if err != nil {
		return AssessmentView{}, err
	}
	row, err := s.Store.(AssessmentStore).LoadAssessment(bounded, scope, project, candidate, id)
	if err != nil {
		return AssessmentView{}, err
	}
	if row.Record.MemberID != scope.MemberID {
		return AssessmentView{}, ErrNotFound
	}
	out := s.assessmentView(bounded, scope, project, current, row)
	if err = s.authorize(bounded, scope, project, false); err != nil {
		return AssessmentView{}, err
	}
	if err = s.checkSupplierReadRevision(bounded, scope, project, current); err != nil {
		return AssessmentView{}, err
	}
	return out, nil
}
func (s *Service) Assessments(ctx context.Context, scope collection.Scope, project, candidate string, before int64) (AssessmentPage, error) {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return AssessmentPage{}, err
	}
	if !s.AssessmentsEnabled() {
		return AssessmentPage{}, ErrUnsupported
	}
	if !collection.ValidID(candidate) || before < 0 {
		return AssessmentPage{}, ErrInvalid
	}
	bounded, cancel, _, err := s.assessmentDeadline(ctx)
	if err != nil {
		return AssessmentPage{}, err
	}
	defer cancel()
	current, err := s.Store.Load(bounded, scope, project, candidate)
	if err != nil {
		return AssessmentPage{}, err
	}
	rows, err := s.Store.(AssessmentStore).ListAssessments(bounded, scope, project, candidate, before)
	if err != nil {
		return AssessmentPage{}, err
	}
	page := AssessmentPage{Items: []AssessmentView{}}
	if len(rows) > 5 {
		rows = rows[:5]
		page.NextBefore = rows[4].Revision
	}
	for _, row := range rows {
		if row.Record.MemberID != scope.MemberID {
			continue
		}
		page.Items = append(page.Items, s.assessmentView(bounded, scope, project, current, row))
	}
	if err = s.authorize(bounded, scope, project, false); err != nil {
		return AssessmentPage{}, err
	}
	if err = s.checkSupplierReadRevision(bounded, scope, project, current); err != nil {
		return AssessmentPage{}, err
	}
	return page, nil
}
