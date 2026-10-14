package collection

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

type historyFixture struct {
	Repository
	scope    Scope
	item     Item
	archived bool
}

func (s *historyFixture) ReadItem(_ context.Context, scope Scope, id string) (Item, error) {
	if scope != s.scope || id != s.item.ID || s.archived {
		return Item{}, ErrNotFound
	}
	return s.item, nil
}
func (s *historyFixture) ReadBatch(_ context.Context, scope Scope, id string) (Batch, error) {
	if scope != s.scope || id != s.item.BatchID || s.archived {
		return Batch{}, ErrNotFound
	}
	return Batch{ID: id}, nil
}

type historyEvidence struct {
	publications map[string]sourcing.PersistedPublication
	refs         []sourcing.OwnHistoryReference
	after        func()
}

func (s *historyEvidence) Read(_ context.Context, org, id string) (sourcing.PersistedPublication, error) {
	p, ok := s.publications[id]
	if !ok || p.Receipt.OrganizationID != org {
		return p, sourcing.ErrSourcePublicationNotFound
	}
	return p, nil
}
func (s *historyEvidence) ListOwnVersions(_ context.Context, _ sourcing.PublicationScope, _ string, _ uint64, q sourcing.OwnHistoryQuery) ([]sourcing.OwnHistoryReference, error) {
	if s.after != nil {
		s.after()
	}
	if q.Version != 0 {
		refs := []sourcing.OwnHistoryReference{}
		for _, r := range s.refs {
			if r.Version == q.Version {
				refs = append(refs, r)
			}
		}
		return refs, nil
	}
	return s.refs, nil
}
func historyPublication(t *testing.T, scope Scope, id, key string, version uint64, parent string) sourcing.PersistedPublication {
	t.Helper()
	input := OwnProduct{Title: "version " + id, Images: []string{}}
	var env sourcing.SourceEnvelope
	var err error
	zero := uint64(0)
	base := &zero
	if parent == "" {
		env, err = OwnEnvelope(id, input)
	} else {
		env, err = OwnRevisionEnvelope(id, input, parent, 1)
		base = nil
	}
	require.NoError(t, err)
	env, err = sourcing.NormalizePublicationEnvelope(env)
	require.NoError(t, err)
	product, err := sourcing.ToSnapshot(env)
	require.NoError(t, err)
	raw, _ := json.Marshal(env)
	snap, _ := json.Marshal(product)
	producer := sourcing.ProducerDescriptor{Kind: "own_product", Version: "v1"}
	hash, err := sourcing.CanonicalPublicationInputHash(producer, env, raw, key, base)
	require.NoError(t, err)
	digest := func(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
	return sourcing.PersistedPublication{Envelope: env, Snapshot: product, Receipt: sourcing.PublicationReceipt{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, PublicationID: id, CatalogPublicationID: id, ProductKey: key, CatalogVersion: version, ExpectedBaseVersion: base, Producer: producer, InputHash: hash, EnvelopeHash: digest(raw), SnapshotHash: digest(snap), PublishedAt: time.Now().UTC()}}
}
func TestOwnHistoryRechecksAuthorityAndRejectsMismatchedEvidence(t *testing.T) {
	scope := Scope{"org", "actor", "member"}
	p1, p3 := uuid.NewString(), uuid.NewString()
	key := "own-" + p1
	store := &historyFixture{scope: scope, item: Item{ID: uuid.NewString(), BatchID: uuid.NewString(), Source: Source{Kind: "own", ProductKey: key, Version: 3, PublicationID: p3}}}
	auth := &testAuth{scope: scope}
	service, err := NewService(store, auth, testSources{})
	require.NoError(t, err)
	evidence := &historyEvidence{publications: map[string]sourcing.PersistedPublication{p1: historyPublication(t, scope, p1, key, 1, ""), p3: historyPublication(t, scope, p3, key, 3, p1)}, refs: []sourcing.OwnHistoryReference{{PublicationID: p3, Version: 3}, {PublicationID: p1, Version: 1}}}
	service.WithOwnHistory(evidence)
	page, err := service.ListOwnHistory(context.Background(), store.item.ID, 0, 20)
	require.NoError(t, err)
	require.Len(t, page.Versions, 2)
	require.EqualValues(t, 3, page.Versions[0].Version)
	old, err := service.ReadOwnVersion(context.Background(), store.item.ID, 1)
	require.NoError(t, err)
	require.Equal(t, evidence.publications[p1].Snapshot, old.Product)
	evidence.refs = []sourcing.OwnHistoryReference{{PublicationID: p1, Version: 3}}
	_, err = service.ListOwnHistory(context.Background(), store.item.ID, 0, 20)
	require.ErrorIs(t, err, ErrUnavailable)
	evidence.refs = []sourcing.OwnHistoryReference{{PublicationID: p3, Version: 3}, {PublicationID: p1, Version: 1}}
	evidence.after = func() { auth.err = ErrForbidden }
	_, err = service.ListOwnHistory(context.Background(), store.item.ID, 0, 20)
	require.ErrorIs(t, err, ErrForbidden)
	auth.err = nil
	evidence.after = func() { store.archived = true }
	_, err = service.ListOwnHistory(context.Background(), store.item.ID, 0, 20)
	require.ErrorIs(t, err, ErrNotFound)
	for _, permission := range auth.calls {
		require.Equal(t, PermissionRead, permission)
	}
	store.archived = false
	evidence.after = nil
	for _, foreign := range []Scope{{"org", "another", "member"}, {"other-org", "actor", "member"}} {
		auth.scope = foreign
		_, err = service.ListOwnHistory(context.Background(), store.item.ID, 0, 20)
		require.ErrorIs(t, err, ErrNotFound)
	}
	auth.scope = scope
	original := evidence.publications[p1]
	bad := original
	bad.Receipt.ActorID = "another"
	evidence.publications[p1] = bad
	_, err = service.ReadOwnVersion(context.Background(), store.item.ID, 1)
	require.ErrorIs(t, err, ErrUnavailable)
	bad = original
	bad.Receipt.SnapshotHash = "corrupt"
	evidence.publications[p1] = bad
	_, err = service.ReadOwnVersion(context.Background(), store.item.ID, 1)
	require.ErrorIs(t, err, ErrUnavailable)
}
