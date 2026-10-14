package review

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/sourcing"
)

func candidatePolicy(in CandidateInput) (enrichment.PolicySnapshot, error) {
	fields := []string{"title"}
	switch in.PolicyVersion {
	case "title-review-v1":
		if !in.ContentSelection.Absent() || in.Execution != nil && !in.Execution.ContentSelection.Absent() {
			return enrichment.PolicySnapshot{}, ErrInvalid
		}
	case enrichment.ContentPolicyVersion:
		if !in.ContentSelection.Valid() || in.Execution == nil || !in.Execution.Valid() || in.Execution.ContentSelection != in.ContentSelection || in.Execution.Binding.ProductKey != in.Base.ProductKey || in.Execution.Binding.CatalogVersion != strconv.FormatUint(in.Base.BaseVersion, 10) || in.Execution.Binding.PublicationID != in.PublicationID {
			return enrichment.PolicySnapshot{}, ErrInvalid
		}
		fields = in.ContentSelection.Fields()
	default:
		return enrichment.PolicySnapshot{}, ErrInvalid
	}
	return enrichment.PolicySnapshot{Version: in.PolicyVersion, AllowedFields: fields, RequiredFields: fields}, nil
}

func validateIntakeProposal(ctx context.Context, base catalog.ProductSnapshot, source sourcing.SourceEnvelope, in CandidateInput) (enrichment.Proposal, error) {
	policy, err := candidatePolicy(in)
	if err != nil {
		return enrichment.Proposal{}, err
	}
	if in.PolicyVersion == enrichment.ContentPolicyVersion {
		return validateContentProposal(ctx, base, source, in.ContentSelection, in.Candidate)
	}
	proposal, err := enrichment.ValidateCandidate(ctx, enrichment.Request{Snapshot: base, Source: source, Policy: policy}, in.Candidate)
	if err != nil {
		return proposal, err
	}
	if len(proposal.Changes) != 1 || proposal.Changes[0].Field != "title" || !proposal.Validation.Valid {
		return proposal, ErrInvalid
	}
	return proposal, ValidateTitle(proposal.Changes[0].Value)
}

func snapshotContentValues(base catalog.ProductSnapshot, selection enrichment.ContentSelection) ([]ContentValue, error) {
	values := make([]ContentValue, 0, 4)
	for _, field := range selection.Fields() {
		value := base.Title
		switch field {
		case "description":
			value = base.Description
		case "selling_points", "seo_keywords":
			items := base.SellingPoints
			if field == "seo_keywords" {
				items = base.SEOKeywords
			}
			var raw bytes.Buffer
			encoder := json.NewEncoder(&raw)
			encoder.SetEscapeHTML(false)
			if err := encoder.Encode(items); err != nil {
				return nil, ErrInvalid
			}
			value = strings.TrimSuffix(raw.String(), "\n")
		}
		values = append(values, ContentValue{Field: field, Value: value})
	}
	return orderedContentValues(selection, values, false)
}

func contentPayload(base catalog.ProductSnapshot, selection enrichment.ContentSelection, proposal enrichment.Proposal) (*ContentPayload, error) {
	before, err := snapshotContentValues(base, selection)
	if err != nil {
		return nil, err
	}
	var values []ContentValue
	for _, change := range proposal.Changes {
		values = append(values, ContentValue{Field: change.Field, Value: change.Value})
	}
	after, err := orderedContentValues(selection, values, true)
	if err != nil {
		return nil, err
	}
	return &ContentPayload{Selection: selection, Before: before, After: after}, nil
}

func contentScopeMatches(in CandidateInput, scope Scope) bool {
	return in.PolicyVersion != enrichment.ContentPolicyVersion || in.Execution != nil && in.Execution.Scope.OrganizationID == scope.Org && in.Execution.Scope.ActorID == scope.Actor
}

func validateContentPatch(ctx context.Context, base catalog.PublishedSnapshot, source sourcing.SourceEnvelope, r Record) error {
	if !r.validContentRecord() || base.PublicationID != r.BasePublicationID {
		return ErrConflict
	}
	before, err := snapshotContentValues(base.Snapshot, r.Content.Selection)
	if err != nil || !reflect.DeepEqual(before, r.Content.Before) {
		return ErrConflict
	}
	in := CandidateInput{ContentSelection: r.Content.Selection, Execution: r.Execution, Base: r.Input, PublicationID: r.BasePublicationID, PolicyVersion: r.Policy, Candidate: enrichment.Candidate{Changes: r.Original.Changes}}
	if _, err := validateIntakeProposal(ctx, base.Snapshot, source, in); err != nil {
		return err
	}
	_, err = applyContentPatch(base.Snapshot, r.Content.Selection, contentChanges(r.Content.After))
	return err
}

func contentChanges(values []ContentValue) []enrichment.FieldChange {
	changes := make([]enrichment.FieldChange, 0, len(values))
	for _, value := range values {
		changes = append(changes, enrichment.FieldChange{Field: value.Field, Value: value.Value})
	}
	return changes
}
