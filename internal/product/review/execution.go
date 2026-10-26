package review

import (
	"context"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/catalog"
	reviewproposal "task-processor/internal/product/review/proposal"
	"time"
)

type CandidateExecutionScope struct{ OrganizationID, ActorID, MemberID string }
type ExecutionCandidateService struct{ service *Service }

func (s *ExecutionCandidateService) WithExecutionPolicy(resolve ExecutionPolicyResolver) *ExecutionCandidateService {
	s.service.WithExecutionPolicy(resolve)
	return s
}

// The tokenless surface can only validate/intake candidates. Human decisions
// and Apply remain on the original request-authorized Service.
func NewExecutionCandidateService(reader catalog.VersionedSnapshotReader, source SourcePublicationGateway, store Store, auth Authorizer, resolve func(context.Context) (CandidateExecutionScope, error)) (*ExecutionCandidateService, error) {
	if resolve == nil {
		return nil, ErrUnavailable
	}
	service, err := NewCandidateService(reader, source, store, auth)
	if err != nil {
		return nil, err
	}
	service.executionResolver = resolve
	return &ExecutionCandidateService{service}, nil
}
func (s *ExecutionCandidateService) ValidateCandidate(ctx context.Context, in CandidateInput) (reviewproposal.Proposal, error) {
	return s.service.ValidateCandidate(ctx, in)
}

func (s *ExecutionCandidateService) ValidateContentCandidate(ctx context.Context, in ContentValidationInput) (reviewproposal.Proposal, error) {
	return s.service.ValidateContentCandidate(ctx, in)
}
func (s *ExecutionCandidateService) CreateFromCandidate(ctx context.Context, key string, in CandidateInput) (View, error) {
	return s.service.CreateFromCandidate(ctx, key, in)
}
func (s *Service) executionScope(ctx context.Context, admin bool) (Scope, error) {
	if admin || s.executionResolver == nil {
		return Scope{}, ErrForbidden
	}
	_, authenticated := authidentity.AuthenticatedIdentityFromContext(ctx)
	deadline, bounded := ctx.Deadline()
	if authenticated || !bounded || !time.Now().Before(deadline) {
		return Scope{}, ErrForbidden
	}
	subject, err := s.executionResolver(ctx)
	if err != nil || !authidentity.IsBoundedIdentifier(subject.OrganizationID) || !authidentity.IsBoundedIdentifier(subject.ActorID) || !authidentity.IsBoundedIdentifier(subject.MemberID) || ctx.Err() != nil {
		return Scope{}, ErrForbidden
	}
	return Scope{Org: subject.OrganizationID, Actor: subject.ActorID}, nil
}
