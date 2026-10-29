package collection

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/catalog"
	"testing"
	"time"
)

type quoteFixture struct {
	read  func(context.Context, Scope, Source) (SourceQuotes, error)
	calls int
}

func (q *quoteFixture) ReadSourceQuotes(ctx context.Context, scope Scope, source Source) (SourceQuotes, error) {
	q.calls++
	return q.read(ctx, scope, source)
}

func TestDetailQuotesUsePrivateFixedSourceAndKeepKnownProductOnFailure(t *testing.T) {
	scope := Scope{"org-a", "actor-a", "member-a"}
	item := Item{ID: uuid.NewString(), Source: Source{ProductKey: "product", PublicationID: "publication", Version: 3, Kind: "acquisition"}}
	snapshots := &snapshotFixture{snapshot: catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: item.Source.ProductKey}, PublicationID: item.Source.PublicationID, Version: 3, Snapshot: catalog.ProductSnapshot{Title: "known"}}}
	auth := &testAuth{scope: scope}
	service, err := NewService(selectionStore{item: item}, auth, testSources{})
	require.NoError(t, err)
	service.WithSnapshots(snapshots)
	quote := &quoteFixture{read: func(ctx context.Context, actual Scope, source Source) (SourceQuotes, error) {
		require.Equal(t, scope, actual)
		require.Equal(t, item.Source, source)
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), 2*time.Second)
		return SourceQuotes{State: "available", PublicationID: source.PublicationID, Version: source.Version, Prices: []SourcePrice{{Amount: "27.00"}}}, nil
	}}
	service.WithSourceQuotes(quote)
	detail, err := service.ReadItem(context.Background(), item.ID)
	require.NoError(t, err)
	require.Equal(t, "available", detail.SourceQuotes.State)
	quote.read = func(context.Context, Scope, Source) (SourceQuotes, error) {
		return SourceQuotes{}, errors.New("unavailable")
	}
	detail, err = service.ReadItem(context.Background(), item.ID)
	require.NoError(t, err)
	require.Equal(t, "known", detail.Product.Title)
	require.Equal(t, "unavailable", detail.SourceQuotes.State)
	quote.read = func(context.Context, Scope, Source) (SourceQuotes, error) {
		return SourceQuotes{State: "available", PublicationID: "other", Version: 3}, nil
	}
	detail, err = service.ReadItem(context.Background(), item.ID)
	require.NoError(t, err)
	require.Equal(t, "unavailable", detail.SourceQuotes.State)
	previous := quote.calls
	auth.err = ErrForbidden
	_, err = service.ReadItem(context.Background(), item.ID)
	require.ErrorIs(t, err, ErrForbidden)
	require.Equal(t, previous, quote.calls)
	auth.err = nil
	service.WithSourceQuotes(nil)
	detail, err = service.ReadItem(context.Background(), item.ID)
	require.NoError(t, err)
	require.Equal(t, "unavailable", detail.SourceQuotes.State)
	service.WithSourceQuotes(quote)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	detail, err = service.ReadItem(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, "unavailable", detail.SourceQuotes.State)
	require.Equal(t, previous, quote.calls)
	quote.read = func(ctx context.Context, _ Scope, _ Source) (SourceQuotes, error) {
		<-ctx.Done()
		return SourceQuotes{}, ctx.Err()
	}
	ctx2, cancel2 := context.WithTimeout(context.Background(), 750*time.Millisecond)
	defer cancel2()
	started := time.Now()
	detail, err = service.ReadItem(ctx2, item.ID)
	require.NoError(t, err)
	require.Equal(t, "known", detail.Product.Title)
	require.Equal(t, "unavailable", detail.SourceQuotes.State)
	require.NoError(t, ctx2.Err())
	require.Less(t, time.Since(started), 600*time.Millisecond)
	require.Equal(t, previous+1, quote.calls)
}
