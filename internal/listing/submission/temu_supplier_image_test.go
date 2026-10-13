package submission

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestSupplierImageReceiptSeparatesSourceAndArtifactWithoutInventedExpiry(t *testing.T) {
	b := temuExecutionBinding(storecenter.MerchantFullyManaged)
	scope := collection.Scope{OrganizationID: b.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}
	o := temugoods.ImageObservation{AssetID: "a", SourceURL: "https://source.example.org/a.png", ContentHash: collection.Digest("source"), CatID: 34, Usage: 5, Width: 800, Height: 800, Bytes: 1234, MediaType: "image/png"}
	m := TemuSupplierImageManifest{Image: o, Options: model.SupplierImageOptions{CategoryID: 34}}
	subject, err := TemuImageSubjectID(o, b)
	require.NoError(t, err)
	raw, _ := json.Marshal(m)
	id, _ := uuid.NewV7()
	reserved, err := NewExecutionReservation(AcquireExecutionCommand{Scope: ExecutionScope{b.OrganizationID}, IntentKey: "supplier-image", Target: ExecutionTarget{"temu", b.StoreID, subject}, Action: TemuSupplierImageAction, Payload: raw, ClaimOwnerID: "worker-a", Lease: time.Hour}, id.String(), "claim-token", time.Now().UTC().Add(-time.Minute))
	require.NoError(t, err)
	a := reserved.Attempt
	a.FenceEpoch = 1
	claim := ExecutionClaim{Scope: ExecutionScope{b.OrganizationID}, AttemptID: a.AttemptID, FenceEpoch: 1, OwnerID: a.ClaimOwnerID, Token: "claim-token"}
	artifact := temugoods.ImageArtifact{URL: "https://official.example.org/a.png", ContentHash: collection.Digest("platform reencoded"), Width: 800, Height: 800, Bytes: 2345, MediaType: "image/png"}
	result := model.SupplierImageUploaded{URL: artifact.URL, ImageURL: artifact.URL, URLs: []string{artifact.URL}}
	proof, err := NewTemuSupplierImageCompletion(scope, b, a, claim, m, result, collection.Digest("response"), artifact)
	require.NoError(t, err)
	_, r, err := proof.Read(context.Background())
	require.NoError(t, err)
	require.True(t, r.TemuImage.UseBefore.IsZero())
	require.NotEqual(t, r.TemuImage.Image.ContentHash, r.TemuImage.SupplierArtifact.ContentHash)
	require.NoError(t, ValidateOfficialReceipt(r))
	r.TemuImage.UseBefore = time.Now().Add(time.Hour)
	require.Error(t, ValidateOfficialReceipt(r))
	result.URL = "https://official.example.org/other.png"
	_, err = NewTemuSupplierImageCompletion(scope, b, a, claim, m, result, collection.Digest("response"), artifact)
	require.Error(t, err)
	o.Usage = 3
	reuse, err := TemuImageSubjectID(o, b)
	require.NoError(t, err)
	require.Equal(t, subject, reuse, "same bytes/business type cannot establish another image fence by changing a slot")
	b.MerchantMode = storecenter.MerchantSemiManaged
	b.ConnectionRevision++
	b.ApplicationID = "new-app"
	same, err := TemuImageSubjectID(o, b)
	require.NoError(t, err)
	require.Equal(t, subject, same)
	o.Usage = 6
	outer, err := TemuImageSubjectID(o, b)
	require.NoError(t, err)
	require.NotEqual(t, subject, outer)
}
