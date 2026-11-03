package preparation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/product/collection"
)

func planRequestContext(scope Scope) context.Context {
	return authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Hour)})
}

type planRepositoryTest struct {
	UploadPlanRepository
	plan UploadPlan
	hash string
}

func (r planRepositoryTest) FindUploadPlan(_ context.Context, scope Scope, key, digest string) (UploadPlanReceipt, error) {
	if digest != r.hash {
		return UploadPlanReceipt{}, ErrConflict
	}
	return UploadPlanReceipt{Plan: r.plan, Replayed: true}, nil
}

func (r planRepositoryTest) FindUploadPlanAdvance(_ context.Context, scope Scope, id, key, digest string) (UploadPlanAdvanceReceipt, error) {
	if digest != r.hash {
		return UploadPlanAdvanceReceipt{}, ErrConflict
	}
	return UploadPlanAdvanceReceipt{PlanID: id, OperationIDs: []string{"original-operation"}, Replayed: true}, nil
}

type planInputsTest struct {
	UploadPlanInputs
	calls int
}

func (r *planInputsTest) ReadUploadPlanBinding(context.Context, Scope, UploadPlanInput) (string, error) {
	r.calls++
	return "", ErrUnavailable
}

func TestUploadPlanOriginalKeyReplayDoesNotRebindCurrentConfiguration(t *testing.T) {
	auth := &operationAuth{scope: Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}}
	in := UploadPlanInput{PreparationID: uuid.NewString(), StoreID: uuid.NewString(), ExpectedRevision: 1, Rule: "standard", PublicationMode: "platform"}
	reader := &planInputsTest{}
	repo := planRepositoryTest{hash: collection.Digest(in), plan: UploadPlan{ID: uuid.NewString(), Input: in}}
	svc, err := NewUploadPlanService(auth, repo, reader)
	require.NoError(t, err)
	r, err := svc.Create(planRequestContext(auth.scope), uuid.NewString(), in)
	require.NoError(t, err)
	require.True(t, r.Replayed)
	require.Equal(t, 0, reader.calls, "replay must not hydrate a new store binding or mutable quote")
	in.Rule = "invented-rule"
	_, err = svc.Create(planRequestContext(auth.scope), uuid.NewString(), in)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestUploadPlanContinueReplayRetainsOriginalOperationAndChecksCurrentAuthority(t *testing.T) {
	auth := &operationAuth{scope: Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}}
	in := UploadPlanContinueInput{ExpectedRevision: 1, PreviewHash: collection.Digest("original-preview")}
	reader := &planInputsTest{}
	svc, err := NewUploadPlanService(auth, planRepositoryTest{hash: collection.Digest(in)}, reader)
	require.NoError(t, err)
	id, key := uuid.NewString(), uuid.NewString()
	r, err := svc.Continue(planRequestContext(auth.scope), id, key, in)
	require.NoError(t, err)
	require.True(t, r.Replayed)
	require.Equal(t, []string{"original-operation"}, r.OperationIDs)
	require.Zero(t, reader.calls)
	in.PreviewHash = collection.Digest("changed-preview")
	_, err = svc.Continue(planRequestContext(auth.scope), id, key, in)
	require.ErrorIs(t, err, ErrConflict)
	auth.denied = true
	_, err = svc.Continue(planRequestContext(auth.scope), id, key, in)
	require.ErrorIs(t, err, ErrForbidden, "an original key is not permanent execution authority")
}
