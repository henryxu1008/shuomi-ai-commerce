package storecenter_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"task-processor/internal/storecenter"
	tiktok "task-processor/internal/storecenter/tiktokshop"
)

func TestTikTokCallbackClaimIsSingleDispatchAndRetainsOriginalMember(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	access := &storeMemberAuthorizer{member: "original-member"}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, access)
	require.NoError(t, err)
	input := validCreateStoreInput()
	input.Platform = "tiktokshop"
	input.Region = "US"
	input.ExternalStoreID = "USCODE"
	input.OrganizationID = "org-a"
	input.ActorSubject = "subject-create"
	candidate, err := storecenter.NewStore(input)
	require.NoError(t, err)
	store, _, err := repo.CreateOrReplay(context.Background(), "org-a", candidate)
	require.NoError(t, err)
	hash := sha256.Sum256([]byte("one-time-state"))
	stateHash := hex.EncodeToString(hash[:])
	command := storecenter.OfficialConnectionCommand{OrganizationID: "org-a", StoreID: store.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: store.Version()}
	app := tiktok.Application{AppKey: "fixture-app", ServiceID: "fixture-service", Revision: "r1", CallbackURL: "https://example.org/callback"}
	attempt, err := repo.BeginTikTokConnection(context.Background(), command, app, stateHash, time.Now().UTC())
	require.NoError(t, err)
	complete := storecenter.CompleteTikTokConnection{OrganizationID: "org-a", StoreID: store.ID(), AttemptID: attempt.AttemptID, AppKey: app.AppKey, State: "one-time-state", AuthCode: "fixture-code"}
	var group sync.WaitGroup
	dispatches := make(chan bool, 8)
	failures := make(chan error, 8)
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, dispatch, err := repo.ClaimTikTokExchange(context.Background(), complete, stateHash, time.Now().UTC())
			if err != nil {
				failures <- err
			}
			dispatches <- dispatch
		}()
	}
	group.Wait()
	close(dispatches)
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
	count := 0
	for dispatch := range dispatches {
		if dispatch {
			count++
		}
	}
	require.Equal(t, 1, count)
	claimed, err := repo.ReadTikTokAttempt(context.Background(), "org-a", store.ID(), attempt.AttemptID)
	require.NoError(t, err)
	sellerHash := sha256.Sum256([]byte("fixture-seller"))
	require.NoError(t, repo.SaveTikTokCandidate(context.Background(), claimed, "fixture-key", "sealed-candidate", hex.EncodeToString(sellerHash[:]), time.Now().UTC()))
	resumed, dispatch, err := repo.ClaimTikTokExchange(context.Background(), complete, stateHash, time.Now().UTC())
	require.NoError(t, err)
	require.False(t, dispatch)
	require.Equal(t, "credential_received", resumed.State)
	require.Equal(t, "sealed-candidate", resumed.Ciphertext)
	_, found, err := repo.ReadTikTokSellerGrant(context.Background(), resumed, hex.EncodeToString(sellerHash[:]))
	require.NoError(t, err)
	require.False(t, found)
	now := time.Now().UTC()
	grant := tiktok.GrantSnapshot{OrganizationID: "org-a", ID: uuid.NewString(), AppKey: app.AppKey, AppRevision: app.Revision, SellerIdentityHash: hex.EncodeToString(sellerHash[:]), Generation: 1, Revision: 1, Status: tiktok.GrantActive, KeyID: "fixture-key", Ciphertext: "sealed-canonical", AccessExpiresAt: now.Add(time.Hour), RefreshExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}
	intent := storecenter.TikTokBindingIntent{Attempt: resumed, Shop: tiktok.Shop{ID: "actual-us-shop", Code: "USCODE", Region: "US", SellerType: "CROSS_BORDER", Cipher: "actual-provider-cipher"}, Grant: grant, ReplaceCredential: true}
	view, err := repo.BindTikTokConnection(context.Background(), intent, now)
	require.NoError(t, err)
	require.Equal(t, storecenter.ConnectionStatusConnected, view.Status)
	bound, err := repo.ReadTikTokAttempt(context.Background(), "org-a", store.ID(), attempt.AttemptID)
	require.NoError(t, err)
	require.Empty(t, bound.Ciphertext)
	require.Equal(t, "bound", bound.State)
	canonical, binding, err := repo.ReadTikTokGrant(context.Background(), "org-a", store.ID())
	require.NoError(t, err)
	require.Equal(t, grant.ID, canonical.ID)
	require.Equal(t, "actual-us-shop", binding.Shop.ID)
	// An old staging key cannot issue another canonical generation.
	_, err = repo.BindTikTokConnection(context.Background(), intent, now.Add(time.Second))
	require.Error(t, err)
	access.member = "replacement-member"
	access.admin = true
	_, _, err = repo.ClaimTikTokExchange(context.Background(), complete, stateHash, time.Now().UTC())
	require.Error(t, err)
	access.member = "original-member"
	access.admin = false
	event := tiktok.Deauthorization{AppKey: app.AppKey, ShopID: "actual-us-shop", NotificationID: "fixture-event", BodyHash: stateHash, OccurredAt: time.Now().UTC()}
	require.NoError(t, repo.ApplyTikTokDeauthorization(context.Background(), event, time.Now().UTC()))
	closed, err := repo.ReadTikTokConnection(context.Background(), "org-a", store.ID())
	require.NoError(t, err)
	require.NotEqual(t, storecenter.ConnectionStatusConnected, closed.Status)
	require.NoError(t, repo.ApplyTikTokDeauthorization(context.Background(), event, time.Now().UTC()))
	closedAgain, err := repo.ReadTikTokConnection(context.Background(), "org-a", store.ID())
	require.NoError(t, err)
	require.Equal(t, closed.TikTok.BindingRevision, closedAgain.TikTok.BindingRevision)
	// Only an explicit new authorization can rebind the revoked Shop.
	require.NoError(t, db.Exec(`UPDATE workbench_store_tiktok_shop_bindings SET status='active' WHERE organization_id='org-a' AND store_id=?`, store.ID()).Error)
	require.NoError(t, db.Exec(`UPDATE workbench_store_connections SET status='connected' WHERE organization_id='org-a' AND store_id=?`, store.ID()).Error)
	refreshID := uuid.NewString()
	inputHash := sha256.Sum256([]byte("fixture-refresh-input"))
	claim, err := repo.ClaimTikTokRefresh(context.Background(), "org-a", store.ID(), refreshID, 1, hex.EncodeToString(inputHash[:]), time.Now().UTC())
	require.NoError(t, err)
	require.True(t, claim.Dispatch)
	replay, err := repo.ClaimTikTokRefresh(context.Background(), "org-a", store.ID(), refreshID, 1, hex.EncodeToString(inputHash[:]), time.Now().UTC())
	require.NoError(t, err)
	require.False(t, replay.Dispatch)
	_, err = repo.ClaimTikTokRefresh(context.Background(), "org-a", store.ID(), refreshID, 1, stateHash, time.Now().UTC())
	require.Error(t, err)
	current, err := repo.Get(context.Background(), "org-a", store.ID())
	require.NoError(t, err)
	nextCommand := command
	nextCommand.AttemptID = uuid.NewString()
	nextCommand.ExpectedStoreVersion = current.Version()
	_, err = repo.BeginTikTokConnection(context.Background(), nextCommand, app, stateHash, time.Now().UTC())
	require.NoError(t, err)
	// A concurrent explicit reauthorization fences the old refresh completion.
	nextGrant := grant
	nextGrant.Generation = 2
	nextGrant.Revision = claim.Fence.Revision + 1
	nextGrant.Ciphertext = "stale-refresh-cipher"
	require.Error(t, repo.CompleteTikTokRefresh(context.Background(), claim, nextGrant, time.Now().UTC()))
	access.member = "replacement-member"
	access.admin = true
	_, _, err = repo.ClaimTikTokExchange(context.Background(), complete, stateHash, time.Now().UTC())
	require.Error(t, err)
	access.member = "original-member"
	complete.OrganizationID = "org-b"
	_, _, err = repo.ClaimTikTokExchange(context.Background(), complete, stateHash, time.Now().UTC())
	require.Error(t, err)
}
