package storecenterapp

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func (p *temuRulesProvider) QueryObservedTemuProducts(ctx context.Context, _ storecenter.OfficialMerchantCredential, _ o.ProductQuery) (o.ProductPage, error) {
	p.metadataReads++
	p.deadline, _ = ctx.Deadline()
	if p.afterMetadata != nil {
		p.afterMetadata()
	}
	return o.ProductPage{Total: 0, TemuItems: nil}, nil
}

func TestTemuObservationCompletesSequentialReadsWithinOriginalWorkerBudget(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged, storecenter.MerchantLocal} {
		t.Run(string(mode), func(t *testing.T) {
			partition := storecenter.PartitionCN
			if mode == storecenter.MerchantLocal {
				partition = storecenter.PartitionGlobal
			}
			product, original, p, _ := newTemuRulesAccess(t, partition, mode)
			reader := &observationMaterialReader{material: original.material}
			access, err := NewOfficialObservationAccess(reader, observationLiveAccess{}, product.applications)
			require.NoError(t, err)
			start := time.Now()
			now := start
			access.now = func() time.Time { return now }
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p.afterIdentity = func() { now = now.Add(3 * time.Second) }
			subject := storecenter.ObservationSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: storecenter.ObservationPurposeProducts, Sync: true}
			handle, err := access.Authorize(ctx, subject, "store-a", nil)
			require.NoError(t, err)
			p.afterMetadata = func() { now = now.Add(9 * time.Second) }
			_, err = handle.TemuProducts(ctx, o.ProductQuery{Page: 1})
			require.NoError(t, err, "valid sequential reads must not lose the admitted worker budget after ten seconds")
			require.Equal(t, start.Add(30*time.Second), p.deadline)
			_, err = handle.TemuProducts(ctx, o.ProductQuery{Page: 2})
			require.NoError(t, err)
			page, err := handle.TemuProducts(ctx, o.ProductQuery{Page: 3})
			require.ErrorIs(t, err, o.ErrUnavailable, "the original budget cannot renew between pages")
			require.Empty(t, page.TemuItems)
			_, err = handle.TemuProducts(ctx, o.ProductQuery{Page: 4})
			require.ErrorIs(t, err, o.ErrUnavailable)
			require.Equal(t, 3, p.metadataReads, "expired handles must not call the provider again")
		})
	}
}

func TestTemuObservationKeepsEarlierCallerDeadlineAndCancellation(t *testing.T) {
	for _, end := range []string{"deadline", "cancel"} {
		t.Run(end, func(t *testing.T) {
			product, original, p, _ := newTemuRulesAccess(t, storecenter.PartitionGlobal, storecenter.MerchantLocal)
			access, err := NewOfficialObservationAccess(&observationMaterialReader{material: original.material}, observationLiveAccess{}, product.applications)
			require.NoError(t, err)
			start := time.Now()
			now := start
			access.now = func() time.Time { return now }
			ctx, cancel := context.WithDeadline(context.Background(), start.Add(8*time.Second))
			defer cancel()
			subject := storecenter.ObservationSubject{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a", Purpose: storecenter.ObservationPurposeProducts, Sync: true}
			handle, err := access.Authorize(ctx, subject, "store-a", nil)
			require.NoError(t, err)
			p.afterMetadata = func() {
				if end == "cancel" {
					cancel()
				} else {
					now = start.Add(8 * time.Second)
				}
			}
			page, err := handle.TemuProducts(ctx, o.ProductQuery{Page: 1})
			require.ErrorIs(t, err, o.ErrUnavailable)
			require.Empty(t, page.TemuItems)
			require.Equal(t, start.Add(8*time.Second), p.deadline)
		})
	}
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
