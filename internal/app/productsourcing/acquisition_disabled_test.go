package productsourcing

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
)

type disabledAcquisitionStore struct {
	acquisitionStoreSpy
	starts  int
	missing bool
}

func (s *disabledAcquisitionStore) Start(_ context.Context, request sourcing.AcquisitionOperation) (sourcing.AcquisitionOperation, bool, error) {
	s.starts++
	if s.missing {
		request.State = sourcing.AcquisitionAcquiring
		return request, true, nil
	}
	return s.op, false, nil
}

func (s *disabledAcquisitionStore) ByKey(ctx context.Context, scope sourcing.PublicationScope, key string) (sourcing.AcquisitionOperation, error) {
	if s.missing {
		return sourcing.AcquisitionOperation{}, sourcing.ErrAcquisitionNotFound
	}
	return s.acquisitionStoreSpy.ByKey(ctx, scope, key)
}

func TestAcquisitionWithoutProviderRejectsNewIntentBeforeOperationOrCharge(t *testing.T) {
	scope := sourcing.PublicationScope{OrganizationID: "org", ActorID: "actor"}
	store := &disabledAcquisitionStore{missing: true}
	charges := &acquisitionTestChargeCoordinator{}
	s, err := NewAcquisitionService(store, nil, &acquisitionPublisherSpy{}, nil, acquisitionAuthFunc(func(context.Context) (sourcing.PublicationScope, error) { return scope, nil }), charges)
	require.NoError(t, err)
	_, err = s.Acquire(context.Background(), uuid.NewString(), "123")
	require.ErrorIs(t, err, sourcing.ErrAcquisitionUnavailable)
	require.Zero(t, store.starts, "disabled acquisition must not create or claim operations")
	require.Zero(t, charges.beforeCalls)
}

func TestAcquisitionWithoutProviderStillVerifiesOriginalFrozenIntent(t *testing.T) {
	scope := sourcing.PublicationScope{OrganizationID: "org", ActorID: "actor"}
	key := uuid.NewString()
	source, err := sourcing.Canonical1688Source("123")
	require.NoError(t, err)
	command := sourcing.PublicationCommand{PublicationID: "original-publication", ProductKey: "crawler:1688:123", Envelope: sourcing.SourceEnvelope{ProductCandidate: sourcing.ProductCandidate{Title: "original"}}}
	op := acquisitionOperation(scope, key, source)
	op.State, op.Command = sourcing.AcquisitionPublishing, &command
	store := &disabledAcquisitionStore{acquisitionStoreSpy: acquisitionStoreSpy{op: op}}
	publisher := &acquisitionPublisherSpy{}
	s, err := NewAcquisitionService(store, nil, publisher, nil, acquisitionAuthFunc(func(context.Context) (sourcing.PublicationScope, error) { return scope, nil }), &acquisitionTestChargeCoordinator{})
	require.NoError(t, err)
	_, err = s.Acquire(context.Background(), key, "123")
	require.ErrorIs(t, err, sourcing.ErrAcquisitionUnknown)
	require.Equal(t, 1, publisher.verifies)
	require.Equal(t, command, publisher.command)
	require.Zero(t, store.starts)
	_, err = s.Acquire(context.Background(), key, "124")
	require.ErrorIs(t, err, sourcing.ErrAcquisitionConflict)
	require.Equal(t, 1, publisher.verifies)
	require.Zero(t, store.starts)
}
