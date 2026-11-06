package imageagent

import (
	"context"
	"github.com/google/uuid"
	"strings"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authidentity"
)

// RunReader exposes the original scoped history without execution dependencies.
// It owns no facts and cannot create a run, send a command or reserve resources.
type RunReader struct{ repository Repository }

func NewRunReader(repository Repository) (*RunReader, error) {
	if repository == nil {
		return nil, ErrValidation
	}
	return &RunReader{repository: repository}, nil
}

func (s *RunReader) Get(ctx context.Context, runID string) (RunProjection, error) {
	identity, err := organizationExecutionIdentity(ctx)
	if err != nil {
		return RunProjection{}, err
	}
	scope := RunScope{TenantID: identity.TenantID, OwnerUserID: identity.UserID, RunID: strings.TrimSpace(runID)}
	projection, err := s.repository.GetProjection(ctx, scope)
	if err != nil {
		return RunProjection{}, err
	}

	if _, err := identityForRun(identity, projection.Run); err != nil {
		return RunProjection{}, err
	}

	return projection, nil
}

func (s *RunReader) GetPreparedImageSet(ctx context.Context, kind ImageSourceContextKind, contextID, requestID string) (RunProjection, error) {
	identity, err := organizationExecutionIdentity(ctx)
	if err != nil {
		return RunProjection{}, err
	}
	if !kind.Valid() || !agentconfig.UUID(contextID) || !agentconfig.UUID(requestID) {
		return RunProjection{}, ErrValidation
	}
	identity.BusinessTaskID = contextID
	return s.Get(ctx, imageSetRunID(identity, PrepareImageSetInput{ContextKind: kind, ContextID: contextID, RequestID: requestID}))
}

func (s *RunReader) GetPreparedArtwork(ctx context.Context, runID string) (PreparedArtwork, error) {
	id, err := organizationExecutionIdentity(ctx)
	if err != nil {
		return PreparedArtwork{}, err
	}
	if !agentconfig.UUID(runID) {
		return PreparedArtwork{}, ErrValidation
	}
	p, err := s.repository.GetProjection(ctx, RunScope{TenantID: id.TenantID, OwnerUserID: id.UserID, RunID: runID})
	if err != nil {
		return PreparedArtwork{}, err
	}
	if _, err = identityForRun(id, p.Run); err != nil {
		return PreparedArtwork{}, err
	}
	return preparedArtwork(p)
}

func (s *RunReader) GetPreparedArtworkByKey(ctx context.Context, contextID, requestID string) (PreparedArtwork, error) {
	id, err := organizationExecutionIdentity(ctx)
	if err != nil {
		return PreparedArtwork{}, err
	}
	if !canonicalImageValue(contextID) || len(contextID) > 64 || !agentconfig.UUID(requestID) {
		return PreparedArtwork{}, ErrValidation
	}
	runID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(ArtworkSchema+":"+id.TenantID+":"+id.UserID+":"+contextID+":"+requestID)).String()
	prepared, err := s.GetPreparedArtwork(ctx, runID)
	if err == nil && (prepared.Projection.Run.BusinessTaskID != contextID || prepared.Projection.Run.IdempotencyKey != requestID) {
		return PreparedArtwork{}, ErrRevisionConflict
	}
	return prepared, err
}

func identityForRun(identity ExecutionIdentity, run Run) (ExecutionIdentity, error) {
	identity.BusinessTaskID = run.BusinessTaskID

	if run.ScopeProtocol != OrganizationScopeProtocol || run.TenantID != identity.TenantID || run.UserID != identity.UserID || run.MemberID == "" || run.MemberID != identity.MemberID {
		return ExecutionIdentity{}, ErrIdentityRequired
	}
	identity.RunID = run.ID
	if err := ValidateOrganizationExecution(identity, run.ID); err != nil {
		return ExecutionIdentity{}, err
	}

	return identity, nil
}

func organizationExecutionIdentity(ctx context.Context) (ExecutionIdentity, error) {
	identity, err := verifiedExecutionIdentity(ctx)
	if err != nil {
		return identity, err
	}
	verified, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || verified.EffectiveOrganizationID == "" || verified.TenantID != verified.EffectiveOrganizationID {
		return ExecutionIdentity{}, ErrIdentityRequired
	}
	identity.ScopeProtocol = OrganizationScopeProtocol
	return identity, nil
}
