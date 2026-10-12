package productsourcing

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	s3 "task-processor/internal/integration/s3"
	"task-processor/internal/ledger/storagecapacity"
	"task-processor/internal/product/collection"
	"testing"
)

type mediaCapacitySpy struct {
	storagecapacity.Writer
	deny               bool
	reserves, confirms int
}

func (s *mediaCapacitySpy) ReserveWrite(_ context.Context, i storagecapacity.WriteIntent) (storagecapacity.Reservation, error) {
	s.reserves++
	if s.deny {
		return storagecapacity.Reservation{}, storagecapacity.ErrExceeded
	}
	return storagecapacity.Reservation{ID: "reservation", Intent: i}, nil
}
func (s *mediaCapacitySpy) BindObjects(context.Context, string, string, []storagecapacity.Object) error {
	return nil
}
func (s *mediaCapacitySpy) ClaimDispatch(context.Context, string, string) error { return nil }
func (s *mediaCapacitySpy) ConfirmStored(context.Context, string, string, string, string, string, int64) error {
	s.confirms++
	return nil
}
func TestSourceMediaCapacityDenialPrecedesPutAndReadDoesNotAccount(t *testing.T) {
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	raw := b.Bytes()
	id := collection.MediaIdentity{Hash: storagecapacity.Digest(string(raw)), Bytes: int64(len(raw))}
	storage := &mediaStorage{values: map[string]s3.ImmutableObjectPut{}, lost: true}
	capacity := &mediaCapacitySpy{deny: true}
	service := SourceMedia{Storage: storage, Authorization: &mediaAuth{scope: collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}}, Capacity: &storagecapacity.Meter{Writer: capacity, Namespace: "namespace"}}
	_, e := service.Upload(context.Background(), id, raw)
	require.ErrorIs(t, e, collection.ErrStorageExceeded)
	require.Zero(t, storage.puts)
	capacity.deny = false
	_, e = service.Upload(context.Background(), id, raw)
	require.NoError(t, e)
	require.Equal(t, 1, capacity.confirms)
	_, e = service.Read(context.Background(), id)
	require.NoError(t, e)
	require.Equal(t, 2, capacity.reserves)
	require.Equal(t, 1, capacity.confirms)
}
