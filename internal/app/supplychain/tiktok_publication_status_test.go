package supplychainapp

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type statusMerchant struct {
	TikTokRulesMerchant
	binding      storecenter.ProductMerchantBinding
	calls        int
	expected     marketplace.ProductStatusExpectation
	afterRead    func()
	wrongProduct bool
}

func (m *statusMerchant) Binding() storecenter.ProductMerchantBinding { return m.binding }
func (m *statusMerchant) ProductStatus(_ context.Context, expected marketplace.ProductStatusExpectation) (marketplace.ProductStatus, error) {
	m.calls++
	m.expected = expected
	if m.afterRead != nil {
		m.afterRead()
	}
	value := marketplace.ProductStatus{ProductID: expected.ProductID, Status: "PENDING", AuditStatus: "AUDITING", Reasons: []string{}}
	if m.wrongProduct {
		value.ProductID = "different"
	}
	return value, nil
}

type statusStore struct{ merchant *statusMerchant }

func (s statusStore) TikTokMerchant(_ context.Context, _ collection.Scope, _ string, expected *storecenter.ProductMerchantBinding) (TikTokRulesMerchant, error) {
	if expected == nil || *expected != s.merchant.binding {
		return nil, record.ErrConflict
	}
	return s.merchant, nil
}

func TestTikTokPublicationStatusReadsOriginalProductWithoutAnotherMutation(t *testing.T) {
	for _, method := range []marketplace.ListingMethod{marketplace.GlobalPublishing, marketplace.LocalReplication} {
		t.Run(string(method), func(t *testing.T) {
			uploader, scope, records, writer, _, _ := tiktokUploadFixture(t, method)
			stores := &rulesFixture{binding: writer.Binding()}
			reader := &statusMerchant{binding: writer.Binding()}
			app := Application{Sources: uploader.dependencies.Sources.(*preparation.SourceSelector), Authorization: &uploadAuth{scope: scope}, PublicationReceipts: uploader.dependencies.Receipts, TargetMerchants: stores, TikTokStatusStores: statusStore{reader}}
			ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
			_, err := app.TikTokPublicationStatus(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
			require.ErrorIs(t, err, record.ErrNotReady)
			require.Zero(t, reader.calls)
			_, err = uploader.Upload(ctx, scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			mutations := len(writer.calls)
			view, err := app.TikTokPublicationStatus(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
			require.NoError(t, err)
			require.Equal(t, "PENDING", view.TikTokStatus.Status)
			require.Equal(t, "local-1", reader.expected.ProductID)
			require.Equal(t, []string{"local-sku-1"}, reader.expected.LocalSKUIDs)
			if method == marketplace.GlobalPublishing {
				require.Equal(t, "global-1", reader.expected.GlobalProductID)
				require.Len(t, reader.expected.Mappings, 1)
			} else {
				require.Empty(t, reader.expected.GlobalProductID)
			}
			require.Equal(t, mutations, len(writer.calls))
			_, err = app.TikTokPublicationStatus(ctx, uuid.NewString(), records.saved.Merchant.StoreID)
			require.Error(t, err)
			require.Equal(t, 1, reader.calls)
			reader.wrongProduct = true
			_, err = app.TikTokPublicationStatus(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
			require.ErrorIs(t, err, record.ErrConflict)
			reader.wrongProduct = false
			reader.afterRead = func() { stores.binding.TikTok.BindingRevision++ }
			_, err = app.TikTokPublicationStatus(ctx, records.saved.Source.ID, records.saved.Merchant.StoreID)
			require.ErrorIs(t, err, record.ErrForbidden)
			require.Equal(t, mutations, len(writer.calls))
		})
	}
}
