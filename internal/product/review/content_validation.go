package review

import (
	"context"
	"encoding/json"

	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
	"task-processor/internal/product/sourcing"
)

// This read/compute-only port cannot save, approve or apply a proposal. Intake
// still requires the real execution provenance through CandidateInput.
type ContentValidationInput struct {
	Selection     enrichment.ContentSelection
	Base          CreateInput
	PublicationID string
	Candidate     enrichment.Candidate
}

func (s *Service) ValidateContentCandidate(ctx context.Context, input ContentValidationInput) (enrichment.Proposal, error) {
	if s == nil || ctx == nil || !input.Selection.Valid() || !ValidKey(input.PublicationID) || !ValidKey(input.Base.ProductKey) || input.Base.BaseVersion == 0 || input.Base.BaseVersion > 1<<63-1 {
		return enrichment.Proposal{}, ErrInvalid
	}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > MaxRecordBytes {
		return enrichment.Proposal{}, ErrTooLarge
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, false, false)
	if err != nil {
		return enrichment.Proposal{}, err
	}
	ctx, err = s.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return enrichment.Proposal{}, mapSourceReadError(err)
	}
	base, source, err := s.source(ctx, s.reader, s.sourceReader, scope, input.Base)
	if err != nil {
		return enrichment.Proposal{}, err
	}
	if base.PublicationID != input.PublicationID {
		return enrichment.Proposal{}, ErrConflict
	}
	return validateContentProposal(ctx, base.Snapshot, source, input.Selection, input.Candidate)
}

func validateContentProposal(ctx context.Context, base catalog.ProductSnapshot, source sourcing.SourceEnvelope, selection enrichment.ContentSelection, candidate enrichment.Candidate) (enrichment.Proposal, error) {
	fields := selection.Fields()
	proposal, err := enrichment.ValidateCandidate(ctx, enrichment.Request{Snapshot: base, Source: source, Policy: enrichment.PolicySnapshot{Version: enrichment.ContentPolicyVersion, AllowedFields: fields, RequiredFields: fields}}, candidate)
	if err != nil {
		return proposal, err
	}
	_, err = applyContentPatch(base, selection, proposal.Changes)
	return proposal, err
}
