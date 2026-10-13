package storecenterapp

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
)

func (p *temuRulesProvider) QueryObservedTemuProducts(context.Context, storecenter.OfficialMerchantCredential, o.ProductQuery) (o.ProductPage, error) {
	p.metadataReads++
	if p.afterMetadata != nil {
		p.afterMetadata()
	}
	return o.ProductPage{Total: 0, TemuItems: nil}, nil
}

func TestTemuObservationUsesReadonlyPurposeAndDropsRevokedResults(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged, storecenter.MerchantLocal} {
		partition := storecenter.PartitionCN
		if mode == storecenter.MerchantLocal {
			partition = storecenter.PartitionGlobal
		}
		product, original, p, _ := newTemuRulesAccess(t, partition, mode)
		reader := &observationMaterialReader{material: original.material}
		access, err := NewOfficialObservationAccess(reader, observationLiveAccess{}, product.applications)
		require.NoError(t, err)
		access.now = product.now
		subject := storecenter.ObservationSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: storecenter.ObservationPurposeProducts, Sync: true}
		handle, err := access.Authorize(context.Background(), subject, "store-a", nil)
		require.NoError(t, err)
		require.Equal(t, subject, reader.subject)
		raw, err := json.Marshal(handle)
		require.NoError(t, err)
		require.JSONEq(t, `{}`, string(raw))
		_, err = handle.TemuProducts(context.Background(), o.ProductQuery{Page: 1})
		require.NoError(t, err)
		require.Equal(t, 1, p.metadataReads)
		_, err = handle.Products(context.Background(), 1)
		require.ErrorIs(t, err, o.ErrUnsupported)
		p.afterMetadata = func() { reader.err = storecenter.ErrNotFound }
		page, err := handle.TemuProducts(context.Background(), o.ProductQuery{Page: 2})
		require.ErrorIs(t, err, o.ErrForbidden)
		require.Empty(t, page.TemuItems)
		reader.err = nil
		subject.Purpose = storecenter.ObservationPurposeOrders
		_, err = access.Authorize(context.Background(), subject, "store-a", nil)
		require.ErrorIs(t, err, o.ErrUnsupported)
		subject.Purpose = storecenter.ProductPurposePublish
		_, err = access.Authorize(context.Background(), subject, "store-a", nil)
		require.ErrorIs(t, err, o.ErrForbidden)
	}
}
