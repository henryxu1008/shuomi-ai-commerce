package storecenterapp

import (
	"context"
	"github.com/google/uuid"
	"testing"
	"time"

	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
)

type tiktokRecordOnlyStore struct {
	storecenter.TikTokConnectionStore
	store *storecenter.Store
}

type tiktokRefreshStore struct {
	storecenter.TikTokConnectionStore
	dispatched bool
	unknown    bool
}

func (*tiktokRefreshStore) ReadTikTokGrant(context.Context, string, string) (tiktok.GrantSnapshot, storecenter.TikTokShopBinding, error) {
	return tiktok.GrantSnapshot{AppKey: "app", AppRevision: "new"}, storecenter.TikTokShopBinding{}, nil
}

func (s *tiktokRefreshStore) ClaimTikTokRefresh(context.Context, string, string, string, int64, string, time.Time) (storecenter.TikTokRefreshClaim, error) {
	dispatch := !s.dispatched
	s.dispatched = true
	return storecenter.TikTokRefreshClaim{Dispatch: dispatch, Grant: tiktok.GrantSnapshot{OrganizationID: "org", ID: uuid.NewString(), AppKey: "app", AppRevision: "new", Generation: 1, KeyID: "key", Ciphertext: "encrypted"}}, nil
}
func (s *tiktokRefreshStore) UnknownTikTokRefresh(context.Context, storecenter.TikTokRefreshClaim, time.Time) error {
	s.unknown = true
	return nil
}
func (s *tiktokRefreshStore) ReadTikTokConnection(context.Context, string, string) (storecenter.OfficialConnectionView, error) {
	return storecenter.OfficialConnectionView{Status: storecenter.ConnectionStatusUnavailable}, nil
}
func (s *tiktokRefreshStore) Get(context.Context, string, string) (*storecenter.Store, error) {
	return tiktokRecordOnlyStoreForTest(), nil
}
func tiktokRecordOnlyStoreForTest() *storecenter.Store {
	s, _ := storecenter.NewStore(storecenter.CreateStoreInput{ID: "00000000-0000-4000-8000-000000000001", OrganizationID: "org", ActorSubject: "actor", Name: "Shop", Platform: "tiktokshop", Region: "US", ExternalStoreID: "USCODE", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	return s
}

type tiktokRefreshProvider struct {
	tiktokRevisionProvider
	calls int
}

func (p *tiktokRefreshProvider) Refresh(context.Context, string) (tiktok.TokenSet, error) {
	p.calls++
	return tiktok.TokenSet{}, tiktok.ErrOutcomeUnknown
}

type tiktokRefreshProtection struct{ tiktok.CredentialProtection }

func (tiktokRefreshProtection) Open(tiktok.CredentialBinding, string, string) (tiktok.TokenSet, error) {
	return tiktok.TokenSet{RefreshToken: "private"}, nil
}
func TestTikTokLostRefreshResponseIsUnknownAndNeverRedispatched(t *testing.T) {
	s := &tiktokRefreshStore{}
	p := &tiktokRefreshProvider{}
	a, err := NewTikTokConnections(s, p, tiktokRefreshProtection{})
	if err != nil {
		t.Fatal(err)
	}
	op := uuid.NewString()
	_, err = a.Refresh(context.Background(), "org", "00000000-0000-4000-8000-000000000001", op, 1)
	if err != storecenter.ErrOfficialExchangeUnknown || !s.unknown || p.calls != 1 {
		t.Fatalf("unknown not retained: %v %v %d", err, s.unknown, p.calls)
	}
	_, _ = a.Refresh(context.Background(), "org", "00000000-0000-4000-8000-000000000001", op, 1)
	if p.calls != 1 {
		t.Fatal("refresh dispatched twice")
	}
}

func (s tiktokRecordOnlyStore) Get(context.Context, string, string) (*storecenter.Store, error) {
	return s.store, nil
}

func TestUnconfiguredTikTokConnectionDoesNotRequireProviderOrReadCredentials(t *testing.T) {
	store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: "00000000-0000-4000-8000-000000000001", OrganizationID: "org-a", ActorSubject: "actor-a", Name: "US Shop", Platform: "tiktokshop", Region: "US", ExternalStoreID: "USCODE", CreateIdempotencyKey: "00000000-0000-4000-8000-000000000002", OccurredAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	app, err := NewTikTokConnections(tiktokRecordOnlyStore{store: store}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	view, err := app.Read(context.Background(), "org-a", store.ID())
	if err != nil || view.Platform != storecenter.PlatformTikTokShop || view.Status != storecenter.ConnectionStatusUnavailable {
		t.Fatalf("unconfigured projection: %v %v", view, err)
	}
	choices, err := app.Applications(context.Background(), "org-a", store.ID())
	if err != nil || len(choices) != 0 {
		t.Fatal("unconfigured application advertised")
	}
	if _, err := app.Begin(context.Background(), storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: store.ID()}); err != storecenter.ErrOfficialConnectionUnavailable {
		t.Fatal("unconfigured provider dispatch permitted")
	}
}

type tiktokRevisionStore struct {
	storecenter.TikTokConnectionStore
}

func (tiktokRevisionStore) ReadTikTokAttempt(context.Context, string, string, string) (storecenter.TikTokConnectionAttempt, error) {
	return storecenter.TikTokConnectionAttempt{AppKey: "app", AppRevision: "old"}, nil
}
func (tiktokRevisionStore) ClaimTikTokExchange(context.Context, storecenter.CompleteTikTokConnection, string, time.Time) (storecenter.TikTokConnectionAttempt, bool, error) {
	panic("changed application must be rejected before a code dispatch claim")
}

type tiktokRevisionProvider struct{ tiktok.SellerAuthorization }

func (tiktokRevisionProvider) Application() tiktok.Application {
	return tiktok.Application{AppKey: "app", Revision: "new"}
}

type tiktokUnusedProtection struct{ tiktok.CredentialProtection }

func TestTikTokChangedApplicationCannotClaimOldCallback(t *testing.T) {
	app, err := NewTikTokConnections(tiktokRevisionStore{}, tiktokRevisionProvider{}, tiktokUnusedProtection{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = app.Complete(context.Background(), storecenter.CompleteTikTokConnection{OrganizationID: "org", StoreID: "store", AttemptID: "attempt", AppKey: "app", State: "state", AuthCode: "code"})
	if err != storecenter.ErrOfficialConnectionUnavailable {
		t.Fatalf("changed revision: %v", err)
	}
}
