package acquisition

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
)

func TestCapacityRejectionFencesBoundOriginalAndClaimsOnce(t *testing.T) {
	db := isolatedDatabase(t)
	require.NoError(t, InstallSchema(db))
	ctx := context.Background()
	r, err := NewRepository(ctx, db)
	require.NoError(t, err)
	req := requestedOperation(t, "org-capacity", "actor", uuid.NewString())
	old, claim, err := r.Start(ctx, req)
	require.NoError(t, err)
	require.True(t, claim)
	require.Error(t, r.RecordCapacityRejection(ctx, old), "an unbound request cannot supply this bound-rejection proof")
	intent := sourcing.AcquisitionChargeIntent{Scope: old.Scope, OperationID: old.ID, MemberID: "member", Funding: sourcing.AcquisitionFundingMember, Fingerprint: old.Fingerprint, Source: old.Source}
	reservation := uuid.NewString()
	require.NoError(t, r.RecordChargeIntent(ctx, old, intent))
	require.NoError(t, r.BindChargeReservation(ctx, old, intent, reservation))
	require.NoError(t, r.RecordCapacityRejection(ctx, old))
	marked, err := r.ByKey(ctx, old.Scope, old.Key)
	require.NoError(t, err)
	require.Equal(t, old.Fence+1, marked.Fence)
	for _, blocked := range []sourcing.AcquisitionOperation{old, marked} {
		require.ErrorIs(t, r.BindChargeReservation(ctx, blocked, intent, reservation), sourcing.ErrAcquisitionFence)
		_, err = r.Prepare(ctx, blocked, originalCommand(t, blocked))
		require.ErrorIs(t, err, sourcing.ErrAcquisitionFence)
		require.ErrorIs(t, r.Finish(ctx, blocked, sourcing.AcquisitionFailed, "SOURCE_UNAVAILABLE"), sourcing.ErrAcquisitionFence)
	}
	proof, err := r.ReadChargeProof(ctx, old.Scope.OrganizationID, old.ID)
	require.NoError(t, err)
	require.Equal(t, "unknown", proof.State)
	require.Equal(t, reservation, proof.ReservationID)
	var claims atomic.Int32
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, claimed, err := r.Start(ctx, req)
			require.NoError(t, err)
			if claimed {
				claims.Add(1)
			}
		}()
	}
	wg.Wait()
	require.EqualValues(t, 1, claims.Load())
	current, err := r.ByKey(ctx, old.Scope, old.Key)
	require.NoError(t, err)
	require.Equal(t, old.ID, current.ID)
	require.Equal(t, old.Fence+2, current.Fence)
	require.Empty(t, current.FailureCode)
	require.NoError(t, r.BindChargeReservation(ctx, current, intent, reservation))
	require.ErrorIs(t, r.RecordCapacityRejection(ctx, old), sourcing.ErrAcquisitionFence)
	require.ErrorIs(t, r.Finish(ctx, old, sourcing.AcquisitionFailed, "SOURCE_UNAVAILABLE"), sourcing.ErrAcquisitionFence)
	require.NoError(t, db.Exec("UPDATE product_acquisition_operations SET lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=?", current.ID).Error)
	_, claimed, err := r.Start(ctx, req)
	require.NoError(t, err)
	require.False(t, claimed, "a bound request without the marker stays UNKNOWN even after expiry")
	require.NoError(t, r.Finish(ctx, current, sourcing.AcquisitionFailed, "SOURCE_UNAVAILABLE"))
	_, claimed, err = r.Start(ctx, req)
	require.NoError(t, err)
	require.False(t, claimed, "terminal Failed never resets")
	require.ErrorIs(t, r.RecordCapacityRejection(ctx, current), sourcing.ErrAcquisitionFence)
}
