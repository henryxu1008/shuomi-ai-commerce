package preparation

import (
	"context"
	"errors"
	"sort"
	"time"

	"task-processor/internal/product/collection"
)

type UploadPlanContinueInput struct {
	ExpectedRevision int64    `json:"expectedRevision"`
	PreviewHash      string   `json:"previewHash"`
	SourceIDs        []string `json:"sourceIds,omitempty"`
}

func (in UploadPlanContinueInput) Validate() error {
	if !validPlanHash(in.PreviewHash) {
		return ErrInvalid
	}
	return in.validateSelection()
}
func (in UploadPlanContinueInput) validateSelection() error {
	if in.ExpectedRevision < 1 || len(in.SourceIDs) > 1000 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range in.SourceIDs {
		if !collection.ValidID(id) || seen[id] {
			return ErrInvalid
		}
		seen[id] = true
	}
	return nil
}
func validPlanHash(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func (in UploadPlanContinueInput) normalized() UploadPlanContinueInput {
	in.SourceIDs = append([]string(nil), in.SourceIDs...)
	sort.Strings(in.SourceIDs)
	return in
}

type UploadPlanAdvanceReceipt struct {
	PlanID       string   `json:"planId"`
	OperationIDs []string `json:"operationIds"`
	Replayed     bool     `json:"replayed"`
}

type UploadPlanPreview struct {
	Plan       UploadPlan           `json:"plan"`
	Items      []UploadPlanItemView `json:"items"`
	Count      int64                `json:"count"`
	Actionable int64                `json:"actionable"`
	Counts     map[string]int64     `json:"counts"`
	Hash       string               `json:"hash"`
}

type UploadPlanInputs interface {
	ReadUploadPlanBinding(context.Context, Scope, UploadPlanInput) (string, error)
	ReadUploadPlanFacts(context.Context, UploadPlan, UploadPlanItem) (UploadPlanFacts, error)
	AuthorizeUploadPlanStep(context.Context, UploadPlan, OperationInput) error
}

type UploadPlanRepository interface {
	FindUploadPlan(context.Context, Scope, string, string) (UploadPlanReceipt, error)
	CreateUploadPlan(context.Context, UploadPlanCommit) (UploadPlanReceipt, error)
	ReadUploadPlan(context.Context, Scope, string) (UploadPlan, error)
	ListUploadPlans(context.Context, Scope, Query) (collection.Page[UploadPlan], error)
	ListUploadPlanItems(context.Context, Scope, string, Query) (collection.Page[UploadPlanItem], error)
	FindUploadPlanAdvance(context.Context, Scope, string, string, string) (UploadPlanAdvanceReceipt, error)
	AdvanceUploadPlan(context.Context, UploadPlanAdvanceCommit) (UploadPlanAdvanceReceipt, error)
}

type UploadPlanService struct {
	auth       Authorizer
	repository UploadPlanRepository
	inputs     UploadPlanInputs
}

func NewUploadPlanService(auth Authorizer, repository UploadPlanRepository, inputs UploadPlanInputs) (*UploadPlanService, error) {
	if auth == nil || repository == nil || inputs == nil {
		return nil, ErrUnavailable
	}
	return &UploadPlanService{auth, repository, inputs}, nil
}

func (s *UploadPlanService) authorize(ctx context.Context, permission string) (Scope, error) {
	if ctx == nil || s == nil || s.auth == nil {
		return Scope{}, ErrForbidden
	}
	return (&Service{auth: s.auth}).authorize(ctx, permission)
}

func (s *UploadPlanService) Create(ctx context.Context, key string, in UploadPlanInput) (UploadPlanReceipt, error) {
	if ctx == nil {
		return UploadPlanReceipt{}, ErrForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, PermissionManage)
	if err != nil {
		return UploadPlanReceipt{}, err
	}
	if !collection.ValidID(key) || in.Validate() != nil {
		return UploadPlanReceipt{}, ErrInvalid
	}
	in = in.normalized()
	prior, err := s.repository.FindUploadPlan(ctx, scope, key, collection.Digest(in))
	if err == nil {
		return prior, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return UploadPlanReceipt{}, err
	}
	binding, err := s.inputs.ReadUploadPlanBinding(ctx, scope, in)
	if err != nil {
		return UploadPlanReceipt{}, err
	}
	if !validPlanHash(binding) {
		return UploadPlanReceipt{}, ErrUnavailable
	}
	return s.repository.CreateUploadPlan(ctx, UploadPlanCommit{service: s, scope: scope, key: key, input: in, binding: binding})
}

func (s *UploadPlanService) Read(ctx context.Context, id string) (UploadPlan, error) {
	if !collection.ValidID(id) {
		return UploadPlan{}, ErrInvalid
	}
	scope, err := s.authorize(ctx, PermissionRead)
	if err != nil {
		return UploadPlan{}, err
	}
	return s.repository.ReadUploadPlan(ctx, scope, id)
}
func (s *UploadPlanService) List(ctx context.Context, q Query) (collection.Page[UploadPlan], error) {
	if q.Validate() != nil {
		return collection.Page[UploadPlan]{}, ErrInvalid
	}
	scope, err := s.authorize(ctx, PermissionRead)
	if err != nil {
		return collection.Page[UploadPlan]{}, err
	}
	return s.repository.ListUploadPlans(ctx, scope, q)
}

// Preview streams every fixed member into one digest. Only the first 100 are
// returned for display; the hash and counts never truncate whole selection.
func (s *UploadPlanService) Preview(ctx context.Context, id string, in UploadPlanContinueInput) (UploadPlanPreview, error) {
	out := UploadPlanPreview{Items: []UploadPlanItemView{}, Counts: map[string]int64{}}
	if ctx == nil || !collection.ValidID(id) {
		return out, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 18*time.Second)
	defer cancel()
	if in.validateSelection() != nil {
		return out, ErrInvalid
	}
	in = in.normalized()
	plan, err := s.Read(ctx, id)
	if err != nil {
		return out, err
	}
	if plan.Revision != in.ExpectedRevision {
		return out, ErrConflict
	}
	if err = s.checkBinding(ctx, plan); err != nil {
		return out, err
	}
	out.Plan = plan
	digest := NewUploadPlanPreviewDigest(plan)
	selected := map[string]bool{}
	for _, v := range in.SourceIDs {
		selected[v] = true
	}
	last := ""
	var scanned int64
	for {
		page, e := s.repository.ListUploadPlanItems(ctx, plan.Owner, id, Query{Limit: 100, After: last})
		if e != nil {
			return out, e
		}
		for _, item := range page.Items {
			if item.SourceID <= last {
				return out, ErrUnavailable
			}
			last = item.SourceID
			scanned++
			if len(selected) > 0 && !selected[item.SourceID] {
				continue
			}
			view, e := s.resolve(ctx, plan, item)
			if e != nil {
				return out, e
			}
			digest.Add(view)
			out.Count++
			out.Counts[view.Step]++
			if IsUploadPlanAction(view.Step) {
				out.Actionable++
			}
			if len(out.Items) < 100 {
				out.Items = append(out.Items, view)
			}
		}
		if page.NextCursor == "" {
			break
		}
		if page.NextCursor != last {
			return out, ErrUnavailable
		}
	}
	if scanned != plan.Count || len(selected) > 0 && out.Count != int64(len(selected)) {
		return out, ErrConflict
	}
	if err = s.checkBinding(ctx, plan); err != nil {
		return out, err
	}
	out.Hash = digest.Sum()
	return out, nil
}

func (s *UploadPlanService) Continue(ctx context.Context, id, key string, in UploadPlanContinueInput) (UploadPlanAdvanceReceipt, error) {
	if ctx == nil {
		return UploadPlanAdvanceReceipt{}, ErrForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, PermissionManage)
	if err != nil {
		return UploadPlanAdvanceReceipt{}, err
	}
	if !collection.ValidID(id) || !collection.ValidID(key) || in.Validate() != nil {
		return UploadPlanAdvanceReceipt{}, ErrInvalid
	}
	in = in.normalized()
	prior, err := s.repository.FindUploadPlanAdvance(ctx, scope, id, key, collection.Digest(in))
	if err == nil {
		return prior, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return UploadPlanAdvanceReceipt{}, err
	}
	plan, err := s.repository.ReadUploadPlan(ctx, scope, id)
	if err != nil {
		return UploadPlanAdvanceReceipt{}, err
	}
	if plan.Revision != in.ExpectedRevision {
		return UploadPlanAdvanceReceipt{}, ErrConflict
	}
	if err = s.checkBinding(ctx, plan); err != nil {
		return UploadPlanAdvanceReceipt{}, err
	}
	return s.repository.AdvanceUploadPlan(ctx, UploadPlanAdvanceCommit{service: s, plan: plan, key: key, input: in})
}

func (s *UploadPlanService) checkBinding(ctx context.Context, plan UploadPlan) error {
	scope, err := s.authorize(ctx, PermissionManage)
	if err != nil {
		return err
	}
	if scope != plan.Owner {
		return ErrForbidden
	}
	binding, err := s.inputs.ReadUploadPlanBinding(ctx, scope, plan.Input)
	if err != nil {
		return err
	}
	if binding != plan.StoreBindingHash {
		return ErrConflict
	}
	return nil
}

func (s *UploadPlanService) resolve(ctx context.Context, plan UploadPlan, item UploadPlanItem) (UploadPlanItemView, error) {
	facts, err := s.inputs.ReadUploadPlanFacts(ctx, plan, item)
	if err != nil {
		return UploadPlanItemView{}, err
	}
	if facts.HasTarget && (!facts.Target.Valid() || facts.Target.SourceID != item.SourceID) {
		return UploadPlanItemView{}, ErrConflict
	}
	view := UploadPlanItemView{Item: item, Facts: facts, Step: NextUploadPlanStep(plan.Input, item, facts)}
	if view.Step == OperationUpload && facts.PublicationMode != plan.Input.PublicationMode {
		return view, ErrConflict
	}
	return view, nil
}

func IsUploadPlanAction(step string) bool {
	return step == OperationAdapt || step == OperationOptimize || step == OperationUpload
}

// Proofs are minted only after live authorization and cannot be constructed by
// HTTP or persistence consumers. Resolve supplies owner facts during the UoW.
type UploadPlanCommit struct {
	service *UploadPlanService
	scope   Scope
	key     string
	input   UploadPlanInput
	binding string
}

func (p UploadPlanCommit) Read(ctx context.Context) (Scope, string, UploadPlanInput, string, error) {
	if p.service == nil || p.input.Validate() != nil || !collection.ValidID(p.key) {
		return Scope{}, "", UploadPlanInput{}, "", ErrForbidden
	}
	plan := UploadPlan{Owner: p.scope, Input: p.input, StoreBindingHash: p.binding}
	if err := p.service.checkBinding(ctx, plan); err != nil {
		return Scope{}, "", UploadPlanInput{}, "", err
	}
	return p.scope, p.key, p.input.normalized(), p.binding, nil
}

type UploadPlanAdvanceCommit struct {
	service *UploadPlanService
	plan    UploadPlan
	key     string
	input   UploadPlanContinueInput
}

func (p UploadPlanAdvanceCommit) Read(ctx context.Context) (UploadPlan, string, UploadPlanContinueInput, error) {
	if p.service == nil || p.input.Validate() != nil || !collection.ValidID(p.key) {
		return UploadPlan{}, "", UploadPlanContinueInput{}, ErrForbidden
	}
	if err := p.service.checkBinding(ctx, p.plan); err != nil {
		return UploadPlan{}, "", UploadPlanContinueInput{}, err
	}
	return p.plan, p.key, p.input.normalized(), nil
}
func (p UploadPlanAdvanceCommit) Resolve(ctx context.Context, item UploadPlanItem) (UploadPlanItemView, error) {
	if p.service == nil {
		return UploadPlanItemView{}, ErrForbidden
	}
	return p.service.resolve(ctx, p.plan, item)
}
func (p UploadPlanAdvanceCommit) AuthorizeStep(ctx context.Context, in OperationInput) error {
	if p.service == nil || in.PreparationID != p.plan.Input.PreparationID || in.StoreID != p.plan.Input.StoreID {
		return ErrForbidden
	}
	if in.Action == OperationUpload {
		scope, err := p.service.authorize(ctx, PermissionSubmit)
		if err != nil || scope != p.plan.Owner {
			return ErrForbidden
		}
	}
	return p.service.inputs.AuthorizeUploadPlanStep(ctx, p.plan, in)
}
