package submission

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	temugoods "task-processor/internal/marketplace/temu/goods"
	temumodel "task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestTemuImageReceiptCutoffComesFromOriginalAttemptAndCannotBeExtended(t *testing.T) {
	b := temuExecutionBinding(storecenter.MerchantLocal)
	scope := collection.Scope{OrganizationID: b.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}
	in := temumodel.LocalImageUpload{FileURL: "https://images.example.org/approved.png", CatID: 42, Usage: 3}
	observed := temugoods.ImageObservation{AssetID: "asset-a", SourceURL: in.FileURL, CatID: 42, Usage: 3, ContentHash: collection.Digest("exact PNG"), Width: 900, Height: 900, Bytes: 1234, MediaType: "image/png"}
	subject, err := TemuLocalImageSubjectID(observed.ContentHash, in.CatID, in.Usage, b)
	require.NoError(t, err)
	raw, _ := json.Marshal(in)
	now := time.Now().UTC().Add(-time.Minute).Truncate(time.Microsecond)
	id, _ := uuid.NewV7()
	reserved, err := NewExecutionReservation(AcquireExecutionCommand{Scope: ExecutionScope{b.OrganizationID}, IntentKey: "image-a", Target: ExecutionTarget{"temu", b.StoreID, subject}, Action: TemuLocalImageAction, Payload: raw, ClaimOwnerID: "worker-a", Lease: time.Hour}, id.String(), "claim-token", now)
	require.NoError(t, err)
	a := reserved.Attempt
	a.FenceEpoch = 1
	claim := ExecutionClaim{Scope: ExecutionScope{b.OrganizationID}, AttemptID: a.AttemptID, FenceEpoch: 1, OwnerID: a.ClaimOwnerID, Token: "claim-token"}
	result := temumodel.LocalImageUploaded{Images: []temumodel.LocalUploadedImage{{URL: "https://temu-image.example.org/1.png"}, {URL: "https://temu-image.example.org/2.png"}}}
	proof, err := NewTemuImageCompletion(scope, b, a, claim, in, observed, result, collection.Digest("response"))
	require.NoError(t, err)
	_, receipt, err := proof.Read(context.Background())
	require.NoError(t, err)
	require.Equal(t, now.Add(7*24*time.Hour), receipt.TemuImage.UseBefore)
	require.Len(t, receipt.TemuImage.Image.RemoteURLs, 2)
	receipt.TemuImage.UseBefore = receipt.TemuImage.UseBefore.Add(time.Second)
	require.Error(t, ValidateOfficialReceipt(receipt))
	observed.CatID++
	_, err = NewTemuImageCompletion(scope, b, a, claim, in, observed, result, collection.Digest("response"))
	require.Error(t, err)
}
