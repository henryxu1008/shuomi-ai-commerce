package imageagent

import "context"

type PublishArtworkInput struct {
	Identity               ExecutionIdentity
	PlanRevision           int64
	ResultDigest, ActionID string
}

// Saving a creative work is distinct from approving a RoleDesign asset.
type ArtworkAdoptionReceipt struct {
	RunID, SlotID, ArtifactHash, OperationID, BatchID, ItemID, ProductKey, PublicationID string
	PlanRevision                                                                         int64
	SourceVersion                                                                        uint64
}
type ArtworkPublisher interface {
	PublishArtwork(context.Context, PublishArtworkInput) (ArtworkAdoptionReceipt, error)
}

func (s *Service) AdoptArtwork(ctx context.Context, runID string, revision int64, resultDigest, actionID string) error {
	id, err := s.commandIdentity(ctx, runID, revision, actionID)
	if err != nil {
		return err
	}
	if s.artworks == nil {
		return ErrCommandBlocked
	}
	p, err := s.repository.GetProjection(ctx, RunScope{TenantID: id.TenantID, OwnerUserID: id.UserID, RunID: runID})
	if err != nil {
		return err
	}
	if ValidateArtworkAdmission(p.Run, p.Plan) != nil || p.Plan.Revision != revision || p.ResultDigest != resultDigest {
		return ErrRevisionConflict
	}
	if digest, err := ArtworkResultDigest(p.Plan, p.Slots, p.RecoverableEffects); err != nil || digest != resultDigest {
		return ErrRevisionConflict
	}
	return s.workflows.ApproveResults(ctx, ApproveResultsCommand{RunID: runID, PlanRevision: revision, ResultDigest: resultDigest, ActorID: id.UserID, ActionID: actionID, Identity: id})
}
