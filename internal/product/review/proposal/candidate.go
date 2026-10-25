package proposal

import "context"

// ValidateCandidate applies the same deterministic rules to an existing or
// repaired candidate without generating again or writing any Product facts.
// The caller must supply the authorized current base/source and frozen policy;
// model-authored evidence, quality or earlier validation is not authoritative.
func ValidateCandidate(ctx context.Context, request Request, candidate Candidate) (Proposal, error) {
	if ctx == nil {
		return Proposal{}, ErrInputInvalid
	}
	if err := ctx.Err(); err != nil {
		return Proposal{}, err
	}
	working := cloneRequest(request)
	if err := validateRequest(working); err != nil {
		return Proposal{}, err
	}
	candidate = cloneCandidate(candidate)

	evidence, err := validateEvidence(working.Source, candidate.Changes)
	if err != nil {
		return Proposal{}, err
	}
	quality := scoreCandidate(candidate, working.Policy)
	proposal := Proposal{
		Changes:    cloneFieldChanges(candidate.Changes),
		Evidence:   cloneEvidence(evidence),
		Quality:    quality,
		Warnings:   stableWarnings(candidate.Warnings),
		Rejections: stableRejections(candidate.Rejections),
	}
	proposal.Validation = ValidationResult{
		Valid:            len(proposal.Rejections) == 0,
		EvaluatedChanges: len(proposal.Changes),
	}
	if err := validateProposal(&proposal, working.Policy); err != nil {
		return proposal, err
	}
	return proposal, nil
}
