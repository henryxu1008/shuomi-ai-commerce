package storecenter_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAmazonAuthorizationStorePersistsSingleClaimAndSellerOwnership(t *testing.T) {
	ctx := context.Background()
	db := newTikTokStoreDatabase(t)
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallAmazonAuthorizationSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())

	// Production store admission is not wired yet. Only this disposable
	// database accepts the Amazon platform for the repository tests.
	require.NoError(t, db.Exec("ALTER TABLE workbench_stores DROP CONSTRAINT store_platform, ADD CONSTRAINT store_platform CHECK (platform IN ('shein', 'temu', 'tiktokshop', 'walmart', 'amazon'))").Error)
	member := &storeMemberAuthorizer{member: "first-membership"}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, member)
	require.NoError(t, err)
	storeA := amazonRepositoryStore(t, db, repo, "org-a")
	storeB := amazonRepositoryStore(t, db, repo, "org-b")
	now := time.Now().UTC().Truncate(time.Microsecond)

	begin, step, command := amazonRepositoryFlow("org-a", storeA, "session-A", "app", "FIRSTSELLER", now)
	require.NoError(t, repo.StartAmazonAuthorization(ctx, begin))
	otherSession := step
	otherSession.Scope.SessionDigest = sha256.Sum256([]byte("wrong-browser"))
	require.ErrorIs(t, repo.AdvanceAmazonAuthorization(ctx, otherSession), storecenter.ErrNotFound)
	require.NoError(t, repo.AdvanceAmazonAuthorization(ctx, step))
	require.ErrorIs(t, repo.AdvanceAmazonAuthorization(ctx, step), storecenter.ErrNotFound)
	wrongSeller := command
	wrongSeller.SellerID = "ANOTHERSELLER"
	_, err = repo.ClaimAmazonAuthorization(ctx, wrongSeller)
	require.ErrorIs(t, err, storecenter.ErrNotFound)

	type claimResult struct {
		claim storecenter.AmazonAuthorizationClaim
		err   error
	}
	claimResults := make(chan claimResult, 8)
	var group sync.WaitGroup
	for range cap(claimResults) {
		group.Go(func() {
			claim, err := repo.ClaimAmazonAuthorization(ctx, command)
			claimResults <- claimResult{claim, err}
		})
	}
	group.Wait()
	close(claimResults)
	var winningClaim storecenter.AmazonAuthorizationClaim
	var count int
	for result := range claimResults {
		if errors.Is(result.err, storecenter.ErrOfficialExchangeUnknown) {
			continue
		}
		require.NoError(t, result.err)
		require.False(t, result.claim.Replay)
		winningClaim = result.claim
		count++
	}
	require.Equal(t, 1, count)
	require.ErrorIs(t, repo.BindAmazonAuthorization(ctx, winningClaim, "key-id", "sealed", now.Add(10*time.Minute)), storecenter.ErrNotFound)

	member.member = "replacement-membership"
	require.ErrorIs(t, repo.BindAmazonAuthorization(ctx, winningClaim, "key-id", "sealed", now), storecenter.ErrNotFound)
	member.member = "first-membership"
	require.NoError(t, repo.BindAmazonAuthorization(ctx, winningClaim, "key-id", "sealed", now))

	// New repository instance uses the persisted scope and original receipt.
	reopened, err := storecenter.NewMemberScopedStoreRepository(db, member)
	require.NoError(t, err)
	replay, err := reopened.ClaimAmazonAuthorization(ctx, command)
	require.NoError(t, err)
	require.True(t, replay.Replay)
	require.Equal(t, winningClaim.CredentialAAD(), replay.CredentialAAD())
	foreignCode := command
	foreignCode.CodeDigest = amazonFixtureDigest("different-code")
	_, err = reopened.ClaimAmazonAuthorization(ctx, foreignCode)
	require.ErrorIs(t, err, storecenter.ErrNotFound)

	begin, step, otherEnterprise := amazonRepositoryFlow("org-b", storeB, "session-B", "different-app", "FIRSTSELLER", now)
	require.NoError(t, reopened.StartAmazonAuthorization(ctx, begin))
	require.NoError(t, reopened.AdvanceAmazonAuthorization(ctx, step))
	secondClaim, err := reopened.ClaimAmazonAuthorization(ctx, otherEnterprise)
	require.NoError(t, err)
	require.ErrorIs(t, reopened.BindAmazonAuthorization(ctx, secondClaim, "key-id", "sealed", now), storecenter.ErrAlreadyExists)

	begin, step, renewal := amazonRepositoryFlow("org-a", storeA, "new-browser-session", "new-app", "FIRSTSELLER", now)
	require.NoError(t, reopened.StartAmazonAuthorization(ctx, begin))
	require.NoError(t, reopened.AdvanceAmazonAuthorization(ctx, step))
	renewalClaim, err := reopened.ClaimAmazonAuthorization(ctx, renewal)
	require.NoError(t, err)
	require.Equal(t, winningClaim.Generation+1, renewalClaim.Generation)
	require.NoError(t, reopened.BindAmazonAuthorization(ctx, renewalClaim, "key-id", "sealed-renewal", now))
	_, err = reopened.ClaimAmazonAuthorization(ctx, command)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	t.Run("application and durable Store together", func(t *testing.T) {
		checkAmazonAuthorizationWithApplication(t, db, reopened, now)
	})
	t.Run("concurrent retargeting discards stale claims", func(t *testing.T) {
		checkAmazonAuthorizationConcurrentRetargeting(t, db, reopened, now)
	})
}

func amazonRepositoryStore(t *testing.T, db *gorm.DB, repo *storecenter.MemberScopedStoreRepository, org string) *storecenter.Store {
	t.Helper()
	input := validCreateStoreInput()
	input.ID, input.CreateIdempotencyKey = uuid.NewString(), uuid.NewString()
	input.OrganizationID, input.ActorSubject, input.Region = org, "subject-create", "US"
	input.Name, input.ExternalStoreID = "Fixture "+input.ID, input.ID
	candidate, err := storecenter.NewStore(input)
	require.NoError(t, err)
	persisted, _, err := repo.CreateOrReplay(context.Background(), org, candidate)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE workbench_stores SET platform = 'amazon' WHERE organization_id = ? AND id = ?", org, persisted.ID()).Error)
	return persisted
}

func amazonRepositoryFlow(org string, store *storecenter.Store, session, appID, seller string, now time.Time) (
	storecenter.AmazonAuthorizationBegin, storecenter.AmazonAuthorizationStep, storecenter.AmazonAuthorizationClaimCommand,
) {
	scope := storecenter.AmazonBrowserScope{OrganizationID: org, SessionDigest: sha256.Sum256([]byte(session))}
	app := storecenter.AmazonAuthorizationApplication{ID: appID, Revision: "r1", RedirectURI: "https://workbench.example.org/amazon/redirect"}
	begin := storecenter.AmazonAuthorizationBegin{
		Scope: scope, Application: app, StoreID: store.ID(), AttemptID: uuid.NewString(),
		FlowDigest: amazonFixtureDigest(session + "/cookie"), ExpectedStoreVersion: store.Version(),
		Now: now, ExpiresAt: now.Add(9 * time.Minute),
	}
	step := storecenter.AmazonAuthorizationStep{
		Scope: scope, Application: app, FlowDigest: begin.FlowDigest,
		StateDigest: amazonFixtureDigest(session + "/state"), SellerID: seller, Now: now,
	}
	return begin, step, storecenter.AmazonAuthorizationClaimCommand{AmazonAuthorizationStep: step, CodeDigest: amazonFixtureDigest(session + "/code")}
}

func amazonFixtureDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func checkAmazonAuthorizationWithApplication(t *testing.T, db *gorm.DB, repo *storecenter.MemberScopedStoreRepository, now time.Time) {
	t.Helper()
	ctx := context.Background()
	store := amazonRepositoryStore(t, db, repo, "org-c")
	scope := storecenter.AmazonBrowserScope{OrganizationID: "org-c", SessionDigest: sha256.Sum256([]byte("workbench-session-C"))}
	const appID = "amzn1.sellerapps.app.2eca283f-9f5a-4d13-b16c-474000000057"
	appConfig := storecenter.AmazonAuthorizationApplication{
		ID: appID, Revision: "r1", RedirectURI: "https://shuomi.example.org/amazon/redirect",
	}
	provider := &amazonRepositoryProvider{}
	protection, err := credentialprotection.NewAESGCM("fixture-key", make([]byte, 32))
	require.NoError(t, err)
	clock := func() time.Time { return now }
	app, err := storeapp.NewAmazonAuthorizations(repo, provider, protection, appConfig, clock)
	require.NoError(t, err)
	started, err := app.Begin(ctx, scope, store.ID(), store.Version())
	require.NoError(t, err)
	login := storecenter.AmazonLoginCallback{
		FlowCookie: started.FlowCookie, AmazonState: "synthetic-amazon-state", SellerID: "THIRDSELLER",
		AmazonCallbackURI: "https://amazon.com/apps/authorize/confirm/" + appID,
	}
	confirm, err := app.Continue(ctx, scope, login)
	require.NoError(t, err)
	target, err := url.Parse(confirm)
	require.NoError(t, err)
	callback := storecenter.AmazonRedirectCallback{
		FlowCookie: started.FlowCookie, State: target.Query().Get("state"), SellerID: "THIRDSELLER", Code: "synthetic-code",
	}
	result, err := app.Complete(ctx, scope, callback)
	require.NoError(t, err)
	require.Equal(t, store.ID(), result.StoreID)
	require.EqualValues(t, 1, result.Generation)
	var ciphertext string
	require.NoError(t, db.Raw("SELECT ciphertext FROM workbench_store_amazon_grants WHERE organization_id = ? AND store_id = ?", "org-c", store.ID()).Row().Scan(&ciphertext))
	require.NotEmpty(t, ciphertext)
	require.NotContains(t, ciphertext, "synthetic-refresh-token")

	repoAfterRestart, err := storecenter.NewMemberScopedStoreRepository(db, &storeMemberAuthorizer{member: "first-membership"})
	require.NoError(t, err)
	appAfterRestart, err := storeapp.NewAmazonAuthorizations(repoAfterRestart, provider, protection, appConfig, clock)
	require.NoError(t, err)
	replayed, err := appAfterRestart.Complete(ctx, scope, callback)
	require.NoError(t, err)
	require.Equal(t, result, replayed)
	require.Equal(t, 1, provider.exchanges)
	require.Equal(t, 1, provider.probes)
}

type amazonRepositoryProvider struct {
	exchanges, probes int
}

func (p *amazonRepositoryProvider) ExchangeCode(_ context.Context, code string) (amazon.Tokens, error) {
	p.exchanges++
	if code != "synthetic-code" {
		return amazon.Tokens{}, amazon.ErrInvalid
	}
	return amazon.Tokens{AccessToken: "synthetic-access-token", RefreshToken: "synthetic-refresh-token"}, nil
}

func (p *amazonRepositoryProvider) VerifyUSSeller(_ context.Context, token, sellerID string) error {
	p.probes++
	if token != "synthetic-access-token" || sellerID != "THIRDSELLER" {
		return amazon.ErrUnauthorized
	}
	return nil
}

func checkAmazonAuthorizationConcurrentRetargeting(t *testing.T, db *gorm.DB, repo *storecenter.MemberScopedStoreRepository, now time.Time) {
	t.Helper()
	ctx := context.Background()
	shop1 := amazonRepositoryStore(t, db, repo, "org-d")
	shop2 := amazonRepositoryStore(t, db, repo, "org-d")
	begin1, step1, cmd1 := amazonRepositoryFlow("org-d", shop1, "browser-one", "app", "SELLERONE", now)
	begin2, _, _ := amazonRepositoryFlow("org-d", shop2, "browser-two", "app", "SELLERTWO", now)
	require.NoError(t, repo.StartAmazonAuthorization(ctx, begin1))
	require.NoError(t, repo.StartAmazonAuthorization(ctx, begin2))
	require.NoError(t, repo.AdvanceAmazonAuthorization(ctx, step1))
	staleClaim, err := repo.ClaimAmazonAuthorization(ctx, cmd1)
	require.NoError(t, err)

	// Hold each transaction immediately before its session upsert so the test
	// exercises crossed row locks instead of accidentally running in series.
	require.NoError(t, db.Exec(`
		CREATE FUNCTION delay_amazon_session_upsert() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN
			PERFORM pg_sleep(0.2);
			RETURN NEW;
		END
		$$;
		CREATE TRIGGER delay_amazon_session_upsert BEFORE INSERT ON workbench_store_amazon_attempts
			FOR EACH ROW EXECUTE FUNCTION delay_amazon_session_upsert();
	`).Error)
	first, firstStep, _ := amazonRepositoryFlow("org-d", shop2, "browser-one", "app", "SELLERTWO", now)
	second, secondStep, _ := amazonRepositoryFlow("org-d", shop1, "browser-two", "app", "SELLERONE", now)
	first.FlowDigest = amazonFixtureDigest("browser-one/restarted/cookie")
	second.FlowDigest = amazonFixtureDigest("browser-two/restarted/cookie")
	firstStep.FlowDigest, secondStep.FlowDigest = first.FlowDigest, second.FlowDigest
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, start := range []storecenter.AmazonAuthorizationBegin{first, second} {
		group.Go(func() { results <- repo.StartAmazonAuthorization(ctx, start) })
	}
	group.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	require.ErrorIs(t, repo.BindAmazonAuthorization(ctx, staleClaim, "key", "sealed", now), storecenter.ErrNotFound)
	require.ErrorIs(t, repo.AdvanceAmazonAuthorization(ctx, step1), storecenter.ErrNotFound)
	require.NoError(t, repo.AdvanceAmazonAuthorization(ctx, firstStep))
	require.NoError(t, repo.AdvanceAmazonAuthorization(ctx, secondStep))
}
