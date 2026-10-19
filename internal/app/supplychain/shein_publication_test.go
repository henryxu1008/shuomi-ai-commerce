package supplychainapp

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/listing/submission"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type shelfOfficialFixture struct{ *uploadOfficial }

func (r shelfOfficialFixture) PrepareShelfIntent(ctx context.Context, p submission.ShelfIntentCommit) (submission.OfficialIntent, error) {
	v, e := p.Read(ctx)
	if e == nil {
		r.intents[v.Key] = v
	}
	return v, e
}

type shelfAttemptsFixture struct{ *uploadKernel }

func (r shelfAttemptsFixture) ReadShelfAttempt(ctx context.Context, scope submission.ExecutionScope, target submission.ExecutionTarget) (submission.ExecutionAttempt, error) {
	for _, v := range r.attempts {
		if v.OrganizationID == scope.OrganizationID && v.Target == target {
			return v, nil
		}
	}
	return submission.ExecutionAttempt{}, submission.ErrExecutionNotFound
}

type shelfMerchantFixture struct {
	binding storecenter.ProductMerchantBinding
	state   int
	audit   int
	calls   int
	lose    bool
	forbid  bool
}

func (m *shelfMerchantFixture) Binding() storecenter.ProductMerchantBinding { return m.binding }
func (m *shelfMerchantFixture) QueryDocumentState(_ context.Context, spu, version string) (model.DocumentAudit, error) {
	return model.DocumentAudit{SPU: spu, Version: version, SKCs: []model.AuditSKC{{Name: "skc-a", State: &m.audit}}}, nil
}
func (m *shelfMerchantFixture) QueryShelfProduct(_ context.Context, spu string) (model.ShelfProduct, error) {
	one := 1
	return model.ShelfProduct{SPU: spu, SKCs: []model.ShelfSKC{{Name: "skc-a", Sites: []model.ShelfSite{{Site: "shein-us", State: &m.state}}, SKUs: []model.SaleSKU{{Code: "sku-code-a", State: &one}}}}}, nil
}
func (m *shelfMerchantFixture) Warehouses(context.Context) ([]model.Warehouse, error) {
	return []model.Warehouse{{Code: "us-w", SaleCountries: []string{"US"}}}, nil
}
func (m *shelfMerchantFixture) QuerySKUInventory(context.Context, []string, string) (model.InventorySnapshot, error) {
	n := int64(0)
	return model.InventorySnapshot{Type: "VI", Goods: []model.GoodsInventory{{SPU: "spu-a", SKC: "skc-a", SKUs: []model.SKUInventory{{Code: "sku-code-a", Usable: &n, Warehouses: []model.WarehouseInventory{{Code: "us-w", Usable: &n}}}}}}}, nil
}
func (m *shelfMerchantFixture) ShelfOn(context.Context, string) (model.ShelfResult, error) {
	m.calls++
	m.state = 1
	if m.lose {
		return model.ShelfResult{}, errors.New("lost")
	}
	return model.ShelfResult{SKC: "skc-a", Site: "shein-us", ResponseHash: collection.Digest("response")}, nil
}
func (m *shelfMerchantFixture) PublicationMerchant(_ context.Context, _ collection.Scope, _ string, _ string, expected *storecenter.ProductMerchantBinding) (SheinPublicationMerchant, error) {
	if m.forbid || expected != nil && *expected != m.binding {
		return nil, errors.New("revoked")
	}
	return m, nil
}
func TestSheinShelfConfirmationReadsBackWithoutResolvingUnknown(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "response lost"}[lost], func(t *testing.T) {
			u, scope, records, merchant, kernel, official, auth := uploadFixture(t)
			_, e := u.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, e)
			var original submission.OfficialReceipt
			for id, r := range official.receipts {
				if r.Kind == "publish" {
					r.Product.Version = "v1"
					official.receipts[id] = r
					original = r
				}
			}
			m := &shelfMerchantFixture{binding: merchant.Binding(), audit: 2, lose: lost}
			svc, e := NewSheinPublicationService(SheinPublicationDependencies{Sources: u.dependencies.Sources.(*preparation.SourceSelector), Records: records, Authorization: auth, ExecutionAuthorization: auth, Receipts: official, Intents: shelfOfficialFixture{official}, Kernel: kernel, Attempts: shelfAttemptsFixture{kernel}, Stores: m})
			require.NoError(t, e)
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Hour)})
			in := SheinShelfInput{SourceID: records.saved.Source.ID, StoreID: records.saved.Merchant.StoreID, ReceiptID: original.ID, SKC: "skc-a"}
			key := uuid.NewString()
			confirm, e := svc.Prepare(ctx, key, in)
			require.NoError(t, e)
			require.Equal(t, 0, m.calls)
			require.Len(t, kernel.attempts, 4) // original image and product attempts only
			in.Token = confirm.Token
			result, e := svc.Confirm(ctx, key, in)
			require.NoError(t, e)
			require.Equal(t, 1, m.calls)
			expected := submission.ExecutionSucceeded
			if lost {
				expected = submission.ExecutionOutcomeUnknown
			}
			require.Equal(t, expected, result.Status)
			_, e = svc.Confirm(ctx, key, in)
			require.NoError(t, e)
			require.Equal(t, 1, m.calls)
			observation, e := svc.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, 0)
			require.NoError(t, e)
			require.Equal(t, 1, *observation.SKCs[0].ShelfState)
			require.False(t, observation.SKCs[0].CanShelf)
			require.Equal(t, int64(0), *observation.SKCs[0].SKUs[0].Usable)
			if lost {
				require.Equal(t, submission.ExecutionOutcomeUnknown, observation.SKCs[0].Attempt.Status)
			}
			_, e = svc.Prepare(ctx, uuid.NewString(), in)
			require.Error(t, e)
			require.Equal(t, 1, m.calls)
			m.forbid = true
			_, e = svc.Confirm(ctx, key, in)
			require.Error(t, e)
			require.Equal(t, 1, m.calls)
		})
	}
}
