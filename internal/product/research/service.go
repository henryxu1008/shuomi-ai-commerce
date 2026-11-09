package research

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"task-processor/internal/product/collection"
	"time"
	"unicode/utf8"
)

func ValidSelection(s Selection) bool {
	return collection.ValidID(s.ItemID) && s.Revision > 0 && collection.ValidID(s.PublicationID) && s.Version > 0
}

var sheinGoodsID = regexp.MustCompile(`^[1-9][0-9]{0,63}$`)
var sheinChecksum = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func ValidObservation(v Observation, s Selection) bool {
	r := v.Ref
	if r.Selection != s || !ValidSelection(s) || r.Source.PublicationID != s.PublicationID || r.Source.OperationID != s.PublicationID || r.Source.Version != s.Version || r.Source.ProductKey == "" || !collection.ValidID(r.JobID) || len(v.Title) > 2000 || !utf8.ValidString(v.Title) {
		return false
	}
	switch r.Source.Kind {
	case "amazon_data":
		return v.SHEIN == nil && len(r.Checksum) > 0 && len(r.Checksum) <= 256 && len(v.Site) <= 4 && v.Site != "" && len(v.ASIN) == 10
	case collection.SourceSHEINUSData:
		return v.SHEIN != nil && v.SHEIN.Market == "US" && sheinGoodsID.MatchString(v.SHEIN.NativeGoodsID) &&
			sheinChecksum.MatchString(r.Checksum) && collection.ValidID(r.QueryRef) && r.ObservationVersion == "1" && r.ObservationEncodingVersion != "" &&
			v.Item.ID == s.ItemID && v.Item.Revision == s.Revision && v.Item.Source == r.Source && v.Item.BatchID == v.BatchID && v.Site == "" && v.ASIN == ""
	default:
		return false
	}
}
func Validate(c Command) error {
	if c.Action == SaveAssessment {
		return validateAssessmentCommand(c)
	}
	if c.Assessment != nil || c.AssessmentRef != nil && (c.Action != "decide" || !ValidAssessmentRef(*c.AssessmentRef)) {
		return ErrInvalid
	}
	if c.Action == RecordSupplierReview {
		if !collection.ValidID(c.CandidateID) || c.ExpectedRevision < 1 || len(c.Selections) != 0 ||
			!collection.ValidID(c.LinkID) || !supplierDigest.MatchString(c.EvidenceDigest) ||
			c.State != "" || c.Reason != "" || !validSupplierInput(c.Supplier) {
			return ErrInvalid
		}
		return nil
	}
	if c.Supplier != nil {
		return ErrInvalid
	}
	if !utf8.ValidString(c.Reason) || len(c.Reason) > 4096 || strings.ContainsRune(c.Reason, 0) {
		return ErrInvalid
	}
	if c.Action == "link" {
		if len(c.Selections) < 1 || len(c.Selections) > 200 || c.CandidateID != "" || c.ExpectedRevision != 0 || c.LinkID != "" || c.State != "" || c.EvidenceDigest != "" || c.Reason != "" {
			return ErrInvalid
		}
		seen := map[string]bool{}
		for _, s := range c.Selections {
			if !ValidSelection(s) || seen[s.ItemID] {
				return ErrInvalid
			}
			seen[s.ItemID] = true
		}
		return nil
	}
	if !collection.ValidID(c.CandidateID) || c.ExpectedRevision < 1 || len(c.Selections) != 0 {
		return ErrInvalid
	}
	switch c.Action {
	case "select", "unlink":
		if !collection.ValidID(c.LinkID) || c.State != "" || c.EvidenceDigest != "" || c.Reason != "" {
			return ErrInvalid
		}
	case "review", "reset":
		if c.LinkID != "" || c.State != "" || c.Reason != "" || len(c.EvidenceDigest) != 64 {
			return ErrInvalid
		}
	case "decide":
		if c.LinkID != "" || (c.State != "APPROVED_FOR_TEST" && c.State != "REJECTED") || strings.TrimSpace(c.Reason) == "" || len(c.EvidenceDigest) != 64 {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}
func (s *Service) authorize(ctx context.Context, scope collection.Scope, project string, write bool) error {
	if s == nil || s.Store == nil || s.Projects == nil || s.Sources == nil {
		return ErrUnavailable
	}
	if ctx == nil || ctx.Err() != nil || scope.Validate() != nil || !collection.ValidID(project) {
		return ErrInvalid
	}
	return s.Projects.Authorize(ctx, scope, project, write)
}
func (s *Service) Lookup(ctx context.Context, scope collection.Scope, project, key string) (Receipt, error) {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return Receipt{}, err
	}
	if !collection.ValidID(key) {
		return Receipt{}, ErrInvalid
	}
	return s.Store.Lookup(ctx, scope, project, key)
}
func (s *Service) Execute(ctx context.Context, scope collection.Scope, project, key string, c Command) (Receipt, error) {
	if err := s.authorize(ctx, scope, project, true); err != nil {
		return Receipt{}, err
	}
	if !collection.ValidID(key) {
		return Receipt{}, ErrInvalid
	}
	if validation := Validate(c); validation != nil {
		return Receipt{}, validation
	}
	if c.Action == RecordSupplierReview && !s.SupplierReviewsEnabled() {
		return Receipt{}, ErrUnsupported
	}
	if (c.Action == SaveAssessment || c.AssessmentRef != nil) && !s.AssessmentsEnabled() {
		return Receipt{}, ErrUnsupported
	}
	c.Selections = append([]Selection(nil), c.Selections...)
	sort.Slice(c.Selections, func(i, j int) bool { return c.Selections[i].ItemID < c.Selections[j].ItemID })
	if out, found, err := s.Store.Replay(ctx, scope, project, key, c); err != nil || found {
		return out, err
	}
	if c.Action == RecordSupplierReview {
		proof, validation := s.prepareSupplier(ctx, scope, project, c)
		return s.Store.(SupplierReviewStore).CommitSupplier(ctx, scope, project, key, c, proof, validation)
	}
	if c.Action == SaveAssessment || c.AssessmentRef != nil {
		proof, validation := s.PrepareAssessment(ctx, scope, project, c)
		return s.Store.(AssessmentStore).CommitAssessment(ctx, scope, project, key, c, proof, validation)
	}
	var validation error
	observations := []Observation{}
	if c.Action == "link" {
		for _, in := range c.Selections {
			v, err := s.Sources.Observe(ctx, scope, project, in)
			if err != nil {
				validation = err
				break
			}
			if !ValidObservation(v, in) {
				validation = ErrUnavailable
				break
			}
			observations = append(observations, v)
		}
	} else if c.Action != "unlink" {
		current, err := s.Store.Load(ctx, scope, project, c.CandidateID)
		validation = err
		if err == nil {
			linkID := current.SelectedLinkID
			if c.Action == "select" {
				linkID = c.LinkID
			}
			var bound *EvidenceRef
			for _, link := range current.Links {
				if link.ID == linkID && !link.Removed {
					r := link.Ref
					bound = &r
					break
				}
			}
			if bound == nil {
				validation = ErrConflict
			} else {
				v, e := s.Sources.Observe(ctx, scope, project, bound.Selection)
				if e != nil {
					validation = e
				} else if !ValidObservation(v, bound.Selection) || v.Ref != *bound {
					validation = ErrConflict
				} else if c.Action != "select" && v.Ref.Source.Kind != "amazon_data" {
					validation = ErrUnsupported
				}
			}
		}
	}
	return s.Store.Commit(ctx, scope, project, key, c, observations, validation)
}
func (s *Service) List(ctx context.Context, scope collection.Scope, project, after string) (Page, error) {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return Page{}, err
	}
	if after != "" && !collection.ValidID(after) {
		return Page{}, ErrInvalid
	}
	rows, next, err := s.Store.List(ctx, scope, project, after)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: []CandidateView{}, NextCursor: next}
	sourceCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	prepared := s.preparePageEvidence(ctx, sourceCtx, scope, project, rows)
	cache := map[string]*Observation{}
	read := func(ref EvidenceRef) *Observation {
		digest := ref.Digest()
		if v, ok := cache[digest]; ok {
			return v
		}
		v, prefetched := prepared[ref.Selection]
		if !prefetched {
			v = s.observePageSelection(sourceCtx, scope, project, ref.Selection)
		}
		if v == nil || v.Ref != ref {
			cache[digest] = nil
			if sourceCtx.Err() != nil {
				page.Incomplete = true
			}
			return nil
		}
		cache[digest] = v
		return v
	}
	assessmentAccess := map[string]bool{}
	canReadAssessment := func(candidate string, ref *AssessmentRef) bool {
		if ref == nil {
			return true
		}
		key := candidate + ":" + collection.Digest(ref)
		if allowed, found := assessmentAccess[key]; found {
			return allowed
		}
		view, e := s.ReadAssessment(sourceCtx, scope, project, candidate, ref.ID)
		allowed := e == nil && view.Detail != nil && view.Detail.Ref == *ref
		assessmentAccess[key] = allowed
		return allowed
	}
	for _, c := range rows {
		v := CandidateView{ID: c.ID, State: c.State, Revision: c.Revision, SelectedLinkID: c.SelectedLinkID, Evidence: []EvidenceView{}, History: []DecisionView{}}
		v.SupplierReview = s.SupplierReviewsEnabled() && c.Identity.Platform == "amazon" && c.Identity.Market == "us"
		v.Assessment = s.AssessmentsEnabled() && c.Identity.Platform == "amazon" && c.Identity.Market == "us"
		for _, link := range c.Links {
			if link.Removed {
				continue
			}
			observation := read(link.Ref)
			v.Evidence = append(v.Evidence, EvidenceView{ID: link.ID, Observation: observation})
			if link.ID == c.SelectedLinkID && observation != nil {
				v.Observation = observation
				v.EvidenceDigest = link.Ref.Digest()
			}
		}
		if owner := c.ReasonDecision; owner != nil && v.Observation != nil && owner.Reason == c.Reason && owner.Ref.Digest() == v.EvidenceDigest && read(owner.Ref) != nil && canReadAssessment(c.ID, owner.AssessmentRef) {
			v.Reason = owner.Reason
		}
		for _, h := range c.History {
			d := DecisionView{State: h.State, Revision: h.Revision, At: h.At}
			allowed := read(h.Ref) != nil
			if h.AssessmentRef != nil {
				allowed = allowed && canReadAssessment(c.ID, h.AssessmentRef)
				if allowed {
					d.AssessmentRef = h.AssessmentRef
				}
			}
			if allowed {
				d.EvidenceDigest = h.EvidenceDigest
				d.Reason = h.Reason
			}
			v.History = append(v.History, d)
		}
		page.Items = append(page.Items, v)
	}
	if err = s.authorize(ctx, scope, project, false); err != nil {
		return Page{}, err
	}
	return page, nil
}
