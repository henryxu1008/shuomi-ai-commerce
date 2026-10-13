package supplychainapp

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func TestTemuRecoveryPreservesOriginalUnknownAndRejectsSPUOrRevokedAuthority(t *testing.T) {
	s, scope, records, _, _, auth, _ := nativeUploadFixture(t)
	merchant := &nativeUploadMerchant{binding: records.saved.Merchant, fail: true}
	s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
	original, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, original.Status)
	input := ResolveUploadInput{RecordID: records.saved.ID, AttemptID: original.AttemptID}
	recovered, err := s.ResolveUpload(context.Background(), scope, input)
	require.NoError(t, err)
	require.Equal(t, original.AttemptID, recovered.AttemptID)
	require.Equal(t, submission.ExecutionOutcomeUnknown, recovered.Status)
	require.Empty(t, recovered.Product)
	require.Nil(t, recovered.TemuProduct)
	input.SPU = "manufactured-spu"
	_, err = s.ResolveUpload(context.Background(), scope, input)
	require.ErrorIs(t, err, record.ErrInvalid)
	input.SPU = ""
	auth.denied = true
	_, err = s.ResolveUpload(context.Background(), scope, input)
	require.ErrorIs(t, err, record.ErrForbidden)
	require.Equal(t, 1, merchant.productCalls)
	require.Equal(t, 1, merchant.imageCalls)
}

func TestTemuAttemptReadRetainsNativeStatusAndOriginalRecord(t *testing.T) {
	s, scope, records, kernel, official, auth, _ := nativeUploadFixture(t)
	merchant := &nativeUploadMerchant{binding: records.saved.Merchant, fail: true}
	s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
	key := uuid.NewString()
	original, err := s.Upload(context.Background(), scope, key, records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, official.intents[key].RecordHash, collection.Digest(records.saved))
	require.Equal(t, official.intents[key].Target, kernel.attempts[key].Target)
	require.Equal(t, key, kernel.attempts[key].IntentKey)
	got, err := kernel.Get(context.Background(), submission.ExecutionScope{OrganizationID: scope.OrganizationID}, original.AttemptID)
	require.NoError(t, err)
	read, err := official.ReadOfficialIntent(context.Background(), scope, got.IntentKey)
	require.NoError(t, err)
	require.Equal(t, records.saved.ID, read.RecordID)
	stores := &rulesFixture{binding: records.saved.Merchant}
	app := Application{Uploader: s, Authorization: auth, TargetMerchants: stores}
	view, err := app.ReadUploadAttempt(context.Background(), records.saved.ID, original.AttemptID)
	require.NoError(t, err)
	require.Equal(t, "temu", view.Platform)
	require.Equal(t, submission.ExecutionOutcomeUnknown, view.Status)
	require.NotContains(t, view.Message, "SPU")
	_, err = app.ReadUploadAttempt(context.Background(), uuid.NewString(), original.AttemptID)
	require.Error(t, err)
	stores.binding.SupplierIdentityHash = "changed"
	_, err = app.ReadUploadAttempt(context.Background(), records.saved.ID, original.AttemptID)
	require.ErrorIs(t, err, record.ErrForbidden)
}
