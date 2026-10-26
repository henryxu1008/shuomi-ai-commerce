package review

import (
	"context"
	"encoding/json"

	"task-processor/internal/product/catalog"
	reviewproposal "task-processor/internal/product/review/proposal"
	"task-processor/internal/product/sourcing"
)

// This read/compute-only port cannot save, approve or apply a proposal. Intake
// still requires the real execution provenance through CandidateInput.
type ContentValidationInput struct {
	Selection     reviewproposal.ContentSelection
	Base          CreateInput
	PublicationID string
	Candidate     reviewproposal.Candidate
}

func (s *Service) ValidateContentCandidate(ctx context.Context, input ContentValidationInput) (reviewproposal.Proposal, error) {
	if s == nil || ctx == nil || !input.Selection.Valid() || !ValidKey(input.PublicationID) || !ValidKey(input.Base.ProductKey) || input.Base.BaseVersion == 0 || input.Base.BaseVersion > 1<<63-1 {
		return reviewproposal.Proposal{}, ErrInvalid
	}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > MaxRecordBytes {
		return reviewproposal.Proposal{}, ErrTooLarge
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, false, false)
	if err != nil {
		return reviewproposal.Proposal{}, err
	}
	ctx, err = s.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return reviewproposal.Proposal{}, mapSourceReadError(err)
	}
	base, source, err := s.source(ctx, s.reader, s.sourceReader, scope, input.Base)
	if err != nil {
		return reviewproposal.Proposal{}, err
	}
	if base.PublicationID != input.PublicationID {
		return reviewproposal.Proposal{}, ErrConflict
	}
	return validateContentProposal(ctx, base.Snapshot, source, input.Selection, input.Candidate)
}

func validateContentProposal(ctx context.Context, base catalog.ProductSnapshot, source sourcing.SourceEnvelope, selection reviewproposal.ContentSelection, candidate reviewproposal.Candidate) (reviewproposal.Proposal, error) {
	fields := selection.Fields()
	proposal, err := reviewproposal.ValidateCandidate(ctx, reviewproposal.Request{Snapshot: base, Source: source, Policy: reviewproposal.PolicySnapshot{Version: reviewproposal.ContentPolicyVersion, AllowedFields: fields, RequiredFields: fields}}, candidate)
	if err != nil {
		return proposal, err
	}
	_, err = applyContentPatch(base, selection, proposal.Changes)
	return proposal, err
}
