package preparation

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
)

func TestUploadPlanOperationApprovalsBindEverySelectedSourceAndVersion(t *testing.T) {
	id := uuid.NewString()
	in := OperationInput{PreparationID: uuid.NewString(), ExpectedRevision: 1, StoreID: uuid.NewString(), Action: OperationUpload, SourceIDs: []string{id}, PlanID: uuid.NewString(), PlanCommandKey: uuid.NewString(), PlanRequestHash: collection.Digest("request")}
	require.ErrorIs(t, in.Validate(), ErrInvalid, "a plan upload cannot silently bind latest head")
	in.TargetApprovals = []TargetApproval{{SourceID: id, RecordID: uuid.NewString(), Revision: 3}}
	require.NoError(t, in.Validate())
	in.TargetApprovals[0].SourceID = uuid.NewString()
	require.ErrorIs(t, in.Validate(), ErrInvalid)
	in.TargetApprovals[0].SourceID = id
	in.TargetApprovals[0].Revision = 0
	require.ErrorIs(t, in.Validate(), ErrInvalid)
}

func TestUploadPlanMetadataCannotBypassProofThroughPublicOperationCreate(t *testing.T) {
	auth := &operationAuth{scope: Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}}
	service := &OperationService{base: &Service{auth: auth}}
	in := OperationInput{PreparationID: uuid.NewString(), ExpectedRevision: 1, StoreID: uuid.NewString(), Action: OperationAdapt, SourceIDs: []string{uuid.NewString()}, PlanID: uuid.NewString(), PlanCommandKey: uuid.NewString(), PlanRequestHash: collection.Digest("request")}
	_, err := service.Create(planRequestContext(auth.scope), uuid.NewString(), in)
	require.ErrorIs(t, err, ErrInvalid, "only the plan continuation proof can mint a plan child")
}
