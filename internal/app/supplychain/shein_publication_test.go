package supplychainapp

import (
	"context"
	"errors"
	"fmt"
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
	binding       storecenter.ProductMerchantBinding
	state         int
	audit         int
	calls         int
	lose          bool
	forbid        bool
	stockOverride *model.InventorySnapshot
	stockCodes    []string
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
func (m *shelfMerchantFixture) QuerySKUInventory(_ context.Context, ids []string, _ string) (model.InventorySnapshot, error) {
	m.stockCodes = ids
	if m.stockOverride != nil {
		return *m.stockOverride, nil
	}
	n := int64(0)
	return model.InventorySnapshot{Type: "VI", Goods: []model.GoodsInventory{{SPU: "spu-a", SKC: "skc-a", SKUs: []model.SKUInventory{{Code: "sku-code-a", Usable: &n, Warehouses: []model.WarehouseInventory{{Code: "us-w", Usable: &n}}}}}}}, nil
}

func shelfFixture(t *testing.T) (*SheinPublicationService, context.Context, SheinShelfInput, *shelfMerchantFixture, *uploadKernel, *uploadOfficial, *uploadRecords, *uploadAuth) {
	t.Helper()
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
	m := &shelfMerchantFixture{binding: merchant.Binding(), audit: 2}
	svc, e := NewSheinPublicationService(SheinPublicationDependencies{Sources: u.dependencies.Sources.(*preparation.SourceSelector), Records: records, Authorization: auth, ExecutionAuthorization: auth, Receipts: official, Intents: shelfOfficialFixture{official}, Kernel: kernel, Attempts: shelfAttemptsFixture{kernel}, Stores: m})
	require.NoError(t, e)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Hour)})
	in := SheinShelfInput{SourceID: records.saved.Source.ID, StoreID: records.saved.Merchant.StoreID, ReceiptID: original.ID, SKC: "skc-a"}
	return svc, ctx, in, m, kernel, official, records, auth
}
func TestSheinStockMissingIsUnknownAndPagesAreExplicit(t *testing.T) {
	svc, ctx, in, m, _, official, records, _ := shelfFixture(t)
	m.stockOverride = &model.InventorySnapshot{Type: "VI"}
	got, e := svc.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, 0)
	require.NoError(t, e)
	require.False(t, got.StockComplete, "successful empty response cannot mean complete inventory")
	require.Nil(t, got.SKCs[0].SKUs[0].Usable)
	original := official.receipts[in.ReceiptID]
	original.Product.SKCs[0].SKUs = nil
	local := records.saved.Result.Product.SKCs[0].SKUs[0]
	records.saved.Result.Product.SKCs[0].SKUs = nil
	for i := 0; i < 102; i++ {
		code := fmt.Sprintf("sku-%d", i)
		supplier := fmt.Sprintf("supplier-%d", i)
		original.Product.SKCs[0].SKUs = append(original.Product.SKCs[0].SKUs, model.PublishedSKU{SKUCode: code, SupplierSKU: supplier})
		local.SupplierSKU = supplier
		records.saved.Result.Product.SKCs[0].SKUs = append(records.saved.Result.Product.SKCs[0].SKUs, local)
	}
	official.receipts[in.ReceiptID] = original
	first, e := svc.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, 0)
	require.NoError(t, e)
	require.Len(t, m.stockCodes, 100)
	require.Len(t, first.SKCs[0].SKUs, 100)
	require.Equal(t, 100, *first.NextStockOffset)
	require.False(t, first.StockComplete)
	last, e := svc.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, 100)
	require.NoError(t, e)
	require.Equal(t, []string{"sku-100", "sku-101"}, m.stockCodes)
	require.Len(t, last.SKCs[0].SKUs, 2)
	require.Nil(t, last.NextStockOffset)
	require.False(t, last.StockComplete)
	_, e = svc.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, 1)
	require.Error(t, e)
}
func TestSheinShelfRejectsExpiredChangedAndRevokedConfirmation(t *testing.T) {
	for _, scenario := range []string{"expired", "wrong token", "changed mode", "revoked after permit", "completion lost", "read original", "foreign store"} {
		t.Run(scenario, func(t *testing.T) {
			svc, ctx, in, m, kernel, official, _, _ := shelfFixture(t)
			key := uuid.NewString()
			confirm, e := svc.Prepare(ctx, key, in)
			require.NoError(t, e)
			in.Token = confirm.Token
			switch scenario {
			case "expired":
				svc.now = func() time.Time { return confirm.ExpiresAt }
			case "wrong token":
				in.Token = uuid.NewString()
			case "changed mode":
				m.binding.ApplicationType = storecenter.ApplicationFullyManaged
			case "revoked after permit":
				kernel.onAcquire = func(submission.AcquireExecutionCommand) { m.forbid = true }
			case "completion lost":
				official.failCommit = true
			case "read original":
				_, e = svc.Read(ctx, key, in)
				require.ErrorIs(t, e, submission.ErrExecutionNotFound)
				require.Zero(t, m.calls)
				return
			case "foreign store":
				in.StoreID = uuid.NewString()
			}
			got, e := svc.Confirm(ctx, key, in)
			if scenario == "completion lost" || scenario == "revoked after permit" {
				require.NoError(t, e)
				require.Equal(t, submission.ExecutionOutcomeUnknown, got.Status)
				if scenario == "completion lost" {
					require.Equal(t, 1, m.calls)
				} else {
					require.Zero(t, m.calls)
				}
				return
			}
			require.Error(t, e)
			require.Zero(t, m.calls)
		})
	}
}
func TestSheinShelfModesUseOriginalPlanAndNoHead(t *testing.T) {
	for _, scenario := range []string{"full", "semi future", "semi automatic", "semi due", "audit rejected", "audit appeal", "already on"} {
		t.Run(scenario, func(t *testing.T) {
			svc, ctx, in, m, _, official, records, _ := shelfFixture(t)
			switch scenario {
			case "full":
				m.binding.ApplicationType = storecenter.ApplicationFullyManaged
			case "semi future", "semi automatic", "semi due":
				m.binding.ApplicationType = storecenter.ApplicationSemiManaged
				plan := &records.saved.Result.Product.SKCs[0]
				plan.ShelfWay = "2"
				plan.HopeOnSaleDate = "2099-01-01 08:00:00"
				if scenario == "semi automatic" {
					plan.ShelfWay = "1"
					plan.HopeOnSaleDate = ""
				}
				if scenario == "semi due" {
					plan.HopeOnSaleDate = "2020-01-01 08:00:00"
				}
			case "audit rejected":
				m.audit = 3
			case "audit appeal":
				m.audit = 5
			case "already on":
				m.state = 1
			}
			m.binding.ApplicationRevision = "v1:" + string(m.binding.ApplicationType)
			original := official.receipts[in.ReceiptID]
			original.Binding = m.binding
			records.saved.Merchant = m.binding
			official.receipts[in.ReceiptID] = original
			got, e := svc.Status(ctx, in.SourceID, in.StoreID, in.ReceiptID, 0)
			require.NoError(t, e)
			allowed := scenario == "semi automatic" || scenario == "semi due"
			require.Equal(t, allowed, got.SKCs[0].CanShelf)
			_, e = svc.Prepare(ctx, uuid.NewString(), in)
			if allowed {
				require.NoError(t, e)
			} else {
				require.Error(t, e)
			}
			require.Zero(t, m.calls)
		})
	}
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
