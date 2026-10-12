package supplychainapp

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/submission"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type nativeUploadMerchant struct {
	binding                  storecenter.ProductMerchantBinding
	imageCalls, productCalls int
	fail                     bool
}

func (m *nativeUploadMerchant) Binding() storecenter.ProductMerchantBinding { return m.binding }
func (m *nativeUploadMerchant) UploadLocalImage(context.Context, model.LocalImageUpload) (model.LocalImageUploaded, string, error) {
	m.imageCalls++
	return model.LocalImageUploaded{Images: []model.LocalUploadedImage{{URL: "https://images.example.org/temu.png"}}}, collection.Digest("response"), nil
}
func (m *nativeUploadMerchant) CreateLocalGoods(_ context.Context, in model.LocalGoodsV2) (temugoods.CreatedGoods, error) {
	m.productCalls++
	if m.fail {
		return temugoods.CreatedGoods{}, submission.ErrExecutionOutcomeUnknown
	}
	created := model.LocalGoodsCreated{GoodsID: uploadPointer(int64(9007199254740993)), ProductType: uploadPointer(int64(1))}
	for _, sku := range in.SKUs {
		specs := append([]model.LocalSpecificationValue(nil), sku.Specifications...)
		for i := range specs {
			if specs[i].SpecID == nil {
				specs[i].SpecID = uploadPointer(int64(2000 + i))
			}
		}
		created.SKUs = append(created.SKUs, model.LocalCreatedSKU{SKUID: uploadPointer(int64(1000 + len(created.SKUs))), ExternalSKUSN: sku.ExternalSKUID, Specifications: specs})
	}
	return temugoods.CreatedGoods{Local: &created, ResponseHash: collection.Digest("created")}, nil
}

type nativeUploadStore struct {
	merchant *nativeUploadMerchant
	auth     *uploadAuth
}

type expiringNativeKernel struct {
	*uploadKernel
	action string
}

func (k expiringNativeKernel) Acquire(ctx context.Context, command submission.AcquireExecutionCommand) (submission.ExecutionAcquisition, error) {
	result, err := k.uploadKernel.Acquire(ctx, command)
	if err == nil && result.Permit != nil && command.Action == k.action {
		result.Permit.LeaseExpiresAt = time.Now().Add(-time.Second)
		result.Attempt.LeaseExpiresAt = result.Permit.LeaseExpiresAt
		k.attempts[command.IntentKey] = result.Attempt
	}
	return result, err
}

func TestTemuUploadExpiredOriginalPermitStopsActualSend(t *testing.T) {
	for _, action := range []string{submission.TemuLocalImageAction, submission.TemuPublishAction} {
		t.Run(action, func(t *testing.T) {
			s, scope, records, kernel, _, auth, _ := nativeUploadFixture(t)
			merchant := &nativeUploadMerchant{binding: records.saved.Merchant}
			s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
			s.dependencies.Kernel = expiringNativeKernel{kernel, action}
			result, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
			require.Zero(t, merchant.productCalls)
			if action == submission.TemuLocalImageAction {
				require.Zero(t, merchant.imageCalls)
			} else {
				require.Equal(t, 1, merchant.imageCalls)
			}
		})
	}
}

func TestTemuUploadCurrentRecordOrApprovalChangingDuringFinalProbePreventsSend(t *testing.T) {
	for _, change := range []string{"record", "approval"} {
		t.Run(change, func(t *testing.T) {
			s, scope, records, kernel, _, auth, probe := nativeUploadFixture(t)
			merchant := &nativeUploadMerchant{binding: records.saved.Merchant}
			s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
			kernel.onAcquire = func(c submission.AcquireExecutionCommand) {
				if c.Action != submission.TemuLocalImageAction {
					return
				}
				probe.afterProbe = func() {
					probe.afterProbe = nil
					if change == "record" {
						records.saved.Revision++
						return
					}
					inventory, err := s.dependencies.Assets.GetApprovedInventory(context.Background(), asset.InventoryScope{})
					require.NoError(t, err)
					inventory.Assets = nil
					s.dependencies.Assets = uploadAssets{inventory}
				}
			}
			result, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
			require.Zero(t, merchant.imageCalls)
			require.Zero(t, merchant.productCalls)
			for _, attempt := range kernel.attempts {
				require.Equal(t, submission.ExecutionOutcomeUnknown, attempt.Status)
			}
		})
	}
}

func (s nativeUploadStore) TemuExecutionMerchant(_ context.Context, scope collection.Scope, storeID, purpose string, expected *storecenter.ProductMerchantBinding) (TemuExecutionMerchant, error) {
	if s.auth.denied || scope != s.auth.scope || storeID != s.merchant.binding.StoreID || expected == nil || *expected != s.merchant.binding || purpose != storecenter.ProductPurposeImage && purpose != storecenter.ProductPurposePublish {
		return nil, submission.ErrExecutionInvalid
	}
	return s.merchant, nil
}

func TestTemuUploadUsesOriginalReceiptsAndDoesNotResendOnReplayOrUnknown(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(fmt.Sprint(lost), func(t *testing.T) {
			s, scope, records, _, official, auth, _ := nativeUploadFixture(t)
			merchant := &nativeUploadMerchant{binding: records.saved.Merchant, fail: lost}
			s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
			key := uuid.NewString()
			first, err := s.Upload(context.Background(), scope, key, records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, 1, merchant.imageCalls)
			require.Equal(t, 1, merchant.productCalls)
			if lost {
				require.Equal(t, submission.ExecutionOutcomeUnknown, first.Status)
			} else {
				require.Equal(t, submission.ExecutionSucceeded, first.Status)
				require.Equal(t, int64(9007199254740993), *first.TemuProduct.Local.GoodsID)
				require.Len(t, official.receipts, 2)
			}
			replay, err := s.Upload(context.Background(), scope, key, records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, first.Status, replay.Status)
			require.Equal(t, 1, merchant.imageCalls)
			require.Equal(t, 1, merchant.productCalls)
			other, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, first.Status, other.Status)
			require.Equal(t, 1, merchant.imageCalls)
			require.Equal(t, 1, merchant.productCalls)
		})
	}
}

func TestTemuUploadDriftAfterPermitMakesNoProviderCallAndRetainsFence(t *testing.T) {
	for _, phase := range []string{"image", "publish"} {
		t.Run(phase, func(t *testing.T) {
			s, scope, records, kernel, _, auth, probe := nativeUploadFixture(t)
			merchant := &nativeUploadMerchant{binding: records.saved.Merchant}
			s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
			kernel.onAcquire = func(c submission.AcquireExecutionCommand) {
				if phase == "image" && c.Action == submission.TemuLocalImageAction || phase == "publish" && c.Action == submission.TemuPublishAction {
					probe.drift = true
				}
			}
			result, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
			require.Zero(t, merchant.productCalls)
			if phase == "image" {
				require.Zero(t, merchant.imageCalls)
			} else {
				require.Equal(t, 1, merchant.imageCalls)
			}
			probe.drift = false
			kernel.onAcquire = nil
			replay, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionOutcomeUnknown, replay.Status)
			require.Zero(t, merchant.productCalls)
			if phase == "image" {
				require.Zero(t, merchant.imageCalls)
			}
		})
	}
}
