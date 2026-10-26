package review

import (
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"

	"task-processor/internal/product/catalog"
	reviewproposal "task-processor/internal/product/review/proposal"
)

type ContentValue struct {
	Field string `json:"field"`
	Value string `json:"value"`
}

type ContentDecision struct {
	Action   string         `json:"action"`
	Actor    string         `json:"actor"`
	Revision uint64         `json:"revision"`
	Before   []ContentValue `json:"before"`
	After    []ContentValue `json:"after"`
	At       time.Time      `json:"at"`
}

// ContentPayload is part of the existing Review record, not another owner.
// It cannot contain price/stock/paths or modify the frozen language/field set.
type ContentPayload struct {
	Selection reviewproposal.ContentSelection `json:"selection"`
	Before    []ContentValue                  `json:"before"`
	After     []ContentValue                  `json:"after"`
	Decisions []ContentDecision               `json:"decisions"`
}

func cloneContentValues(values []ContentValue) []ContentValue {
	return append([]ContentValue(nil), values...)
}

func (p *ContentPayload) clone() *ContentPayload {
	if p == nil {
		return nil
	}
	value := *p
	value.Before, value.After = cloneContentValues(p.Before), cloneContentValues(p.After)
	value.Decisions = append([]ContentDecision(nil), p.Decisions...)
	for i := range value.Decisions {
		value.Decisions[i].Before = cloneContentValues(p.Decisions[i].Before)
		value.Decisions[i].After = cloneContentValues(p.Decisions[i].After)
	}
	return &value
}

func orderedContentValues(selection reviewproposal.ContentSelection, values []ContentValue, after bool) ([]ContentValue, error) {
	fields := selection.Fields()
	if !selection.Valid() || len(values) != len(fields) {
		return nil, ErrInvalid
	}
	byField := make(map[string]string, 4)
	for _, value := range values {
		if _, duplicate := byField[value.Field]; duplicate || !utf8.ValidString(value.Value) {
			return nil, ErrInvalid
		}
		byField[value.Field] = value.Value
	}
	ordered := make([]ContentValue, 0, len(fields))
	var changes []reviewproposal.FieldChange
	for _, field := range fields {
		value, found := byField[field]
		if !found {
			return nil, ErrInvalid
		}
		ordered = append(ordered, ContentValue{field, value})
		changes = append(changes, reviewproposal.FieldChange{Field: field, Value: value})
	}
	if after {
		if _, err := applyContentPatch(catalog.ProductSnapshot{Title: "validation"}, selection, changes); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func validContentValues(selection reviewproposal.ContentSelection, values []ContentValue, after bool) bool {
	ordered, err := orderedContentValues(selection, values, after)
	if err != nil {
		return false
	}
	for i := range ordered {
		if ordered[i] != values[i] {
			return false
		}
	}
	return true
}

func (p *ContentPayload) valid(revision uint64, state string, native bool) bool {
	if p == nil || !validContentValues(p.Selection, p.Before, false) || !validContentValues(p.Selection, p.After, true) {
		return false
	}
	previous := uint64(1)
	values := p.After
	if len(p.Decisions) > 0 {
		values = p.Decisions[0].Before
	}
	for _, d := range p.Decisions {
		if (d.Action != "accept" && d.Action != "edit" && d.Action != "reject") || !ValidKey(d.Actor) || d.Revision != previous+1 || d.Revision > revision || d.At.IsZero() || !validContentValues(p.Selection, d.Before, true) || !validContentValues(p.Selection, d.After, true) || !reflect.DeepEqual(values, d.Before) || d.Action != "edit" && !reflect.DeepEqual(d.Before, d.After) {
			return false
		}
		previous = d.Revision
		values = d.After
	}
	if previous != revision || !reflect.DeepEqual(values, p.After) {
		return false
	}
	if len(p.Decisions) == 0 {
		return state == "pending" && !native || state == "applied" && native
	}
	last := p.Decisions[len(p.Decisions)-1].Action
	return !native && (state == "pending" && last == "edit" || state == "accepted" && last == "accept" || state == "rejected" && last == "reject" || state == "applied" && last == "accept")
}

func (r Record) validContentRecord() bool {
	if r.Policy != reviewproposal.ContentPolicyVersion {
		return r.Content == nil && (r.Execution == nil || r.Execution.ContentSelection.Absent())
	}
	if !r.Content.valid(r.Revision, r.State, r.PolicyAdoption != nil) || r.Execution == nil || r.Execution.ContentSelection != r.Content.Selection || len(r.History) != 0 || r.Before != "" || r.Title != "" || r.Execution.Scope.OrganizationID != r.Org || r.Execution.Scope.ActorID != r.Owner || r.Execution.Binding.ProductKey != r.Input.ProductKey || r.Execution.Binding.CatalogVersion != strconv.FormatUint(r.Input.BaseVersion, 10) || r.Execution.Binding.PublicationID != r.BasePublicationID {
		return false
	}
	original, err := originalContentValues(r)
	if err != nil {
		return false
	}
	first := r.Content.After
	if len(r.Content.Decisions) > 0 {
		first = r.Content.Decisions[0].Before
	}
	if !reflect.DeepEqual(original, first) {
		return false
	}
	return r.PolicyAdoption == nil || r.PolicyAdoption.ActionVersion == ContentPolicyAdoptionVersion && len(r.Content.Decisions) == 0 && r.PolicyAdoption.ProposalDigest == proposalDigest(r)
}

func (v View) validContentView() bool {
	if v.Policy != reviewproposal.ContentPolicyVersion {
		return v.Content == nil && (v.Execution == nil || v.Execution.ContentSelection.Absent())
	}
	if !v.Content.valid(v.Revision, v.State, v.PolicyAdoption != nil) || v.Execution == nil || v.Execution.ContentSelection != v.Content.Selection || len(v.History) != 0 || v.Before != "" || v.Title != "" || v.OriginalTitle != "" || v.Execution.Scope.ActorID != v.Owner || v.Execution.Binding.ProductKey != v.Input.ProductKey || v.Execution.Binding.CatalogVersion != strconv.FormatUint(v.Input.BaseVersion, 10) {
		return false
	}
	return v.PolicyAdoption == nil || v.PolicyAdoption.ActionVersion == ContentPolicyAdoptionVersion && len(v.Content.Decisions) == 0
}

func originalContentValues(r Record) ([]ContentValue, error) {
	if r.Content == nil {
		return nil, ErrInvalid
	}
	var original []ContentValue
	for _, change := range r.Original.Changes {
		original = append(original, ContentValue{change.Field, change.Value})
	}
	return orderedContentValues(r.Content.Selection, original, true)
}

func (r *Record) decideContent(actor string, d DecisionInput) error {
	if !r.validContentRecord() || !ValidKey(actor) {
		return ErrInvalid
	}
	if d.ExpectedRevision == 0 || d.ExpectedRevision != r.Revision || r.State == "applied" || r.State == "rejected" || r.Revision >= 100 || d.Action == "edit" && r.Revision >= 99 {
		return ErrConflict
	}
	if (d.Action != "edit" && d.Action != "accept" && d.Action != "reject") || d.Title != "" || d.Action != "edit" && len(d.Content) != 0 {
		return ErrInvalid
	}
	p := r.Content.clone()
	before := cloneContentValues(p.After)
	if d.Action == "edit" {
		after, err := orderedContentValues(p.Selection, d.Content, true)
		if err != nil {
			return err
		}
		p.After = after
	}
	next := *r
	next.Content = p
	next.Revision++
	switch d.Action {
	case "edit":
		next.State = "pending"
	case "accept":
		next.State = "accepted"
	case "reject":
		next.State = "rejected"
	}
	p.Decisions = append(p.Decisions, ContentDecision{d.Action, actor, next.Revision, before, cloneContentValues(p.After), time.Now().UTC()})
	if err := next.ValidateStorage(); err != nil {
		return err
	}
	*r = next
	return nil
}
