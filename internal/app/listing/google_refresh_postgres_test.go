package listingapp_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const googleRefreshTableName = "workbench_store_google_oauth_refresh"

var errRefreshAcknowledgement = errors.New("isolated test dropped the PostgreSQL commit acknowledgement")

type googleRefreshFixture struct {
	ctx                       context.Context
	db                        *gorm.DB
	scope                     collection.Scope
	storeID                   string
	repo                      *storecenter.MemberScopedStoreRepository
	application               storeapp.GoogleApplicationRegistration
	hook                      *atomic.Pointer[func(http.ResponseWriter, *http.Request)]
	calls                     *atomic.Int32
	challenge                 *atomic.Value
	initialPlan, originalPlan submission.GooglePlan
	listing                   *listingapp.GooglePublication
}

func (f googleRefreshFixture) connections(t *testing.T, repo storeapp.GoogleConnectionStore, registration storeapp.GoogleApplicationRegistration) *storeapp.GoogleConnections {
	t.Helper()
	service, err := storeapp.NewGoogleConnections(repo, registration)
	require.NoError(t, err)
	return service
}

func (f googleRefreshFixture) reset(t *testing.T) storeapp.GooglePublicationBinding {
	t.Helper()
	require.NoError(t, f.db.Exec("UPDATE workbench_stores SET service_expires_at = clock_timestamp() + interval '1 hour' WHERE id = ?", f.storeID).Error)
	service := f.connections(t, f.repo, f.application)
	attempt := f.begin(t, service)
	f.complete(t, service, attempt)
	binding, err := service.PublicationBinding(f.ctx, f.scope.OrganizationID, f.storeID)
	require.NoError(t, err)
	require.NoError(t, expireGoogleGrant(f.db, f.storeID))
	return binding
}

func (f googleRefreshFixture) begin(t *testing.T, service *storeapp.GoogleConnections) storecenter.OfficialConnectionBegin {
	t.Helper()
	store, err := f.repo.Get(f.ctx, f.scope.OrganizationID, f.storeID)
	require.NoError(t, err)
	attempt, err := service.Begin(f.ctx, f.scope.OrganizationID, f.storeID, store.Version())
	require.NoError(t, err)
	return attempt
}

func (f googleRefreshFixture) complete(t *testing.T, service *storeapp.GoogleConnections, attempt storecenter.OfficialConnectionBegin) {
	t.Helper()
	address, err := url.Parse(attempt.AuthorizationURL)
	require.NoError(t, err)
	f.challenge.Store(address.Query().Get("code_challenge"))
	require.NoError(t, service.Complete(f.ctx, f.scope.OrganizationID, f.storeID, attempt.AttemptID,
		address.Query().Get("state"), "mock-reauthorization-code"))
	bound, err := service.Bind(f.ctx, f.scope.OrganizationID, f.storeID, attempt.AttemptID, storecenter.GoogleAccountSelection{
		AccountID: "123", DataSourceID: "456", ContentLanguage: "en", FeedLabel: "US", DedicatedSourceConfirmed: true,
	})
	require.NoError(t, err)
	require.Equal(t, storecenter.ConnectionStatusConnected, bound.Status)
}

func googleRefreshAuthorized(ctx context.Context) error { return ctx.Err() }

func (f googleRefreshFixture) acquire(service *storeapp.GoogleConnections, binding storeapp.GooglePublicationBinding) error {
	_, err := service.AcquirePublicationCredential(f.ctx, f.scope.OrganizationID, f.storeID, binding, googleRefreshAuthorized)
	return err
}

func (f googleRefreshFixture) refreshState(t *testing.T, binding storeapp.GooglePublicationBinding) storecenter.GoogleCredentialState {
	t.Helper()
	state, err := f.repo.GoogleCredential(f.ctx, storecenter.GoogleCredentialRequest{
		OrganizationID: f.scope.OrganizationID, StoreID: f.storeID, Application: f.application.Application,
		Target: binding.Target, Generation: binding.Generation,
	}, "")
	require.NoError(t, err)
	return state
}

func googleFixtureRenewedToken(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(`{"access_token":"renewed-access","refresh_token":"renewed-refresh","token_type":"Bearer","expires_in":3600}`))
}

func (f googleRefreshFixture) testRecovery(t *testing.T) {
	t.Helper()
	success := googleFixtureRenewedToken
	f.hook.Store(&success)
	t.Cleanup(func() { f.hook.Store(nil) })

	t.Run("postgres_commit_acknowledgements", func(t *testing.T) {
		for _, point := range []string{"claim", "permit", "committed_save", "uncommitted_save"} {
			t.Run(point, func(t *testing.T) {
				binding := f.reset(t)
				fault := &googleRefreshCommitFault{GoogleConnectionStore: f.repo, point: point}
				service := f.connections(t, fault, f.application)
				before := f.calls.Load()
				err := f.acquire(service, binding)
				state := f.refreshState(t, binding)
				if point == "claim" || point == "permit" {
					require.ErrorIs(t, err, storecenter.ErrGoogleRefreshUnavailable)
					require.Equal(t, before, f.calls.Load(), "an unconfirmed claim or permit cannot send")
					require.Equal(t, storecenter.GoogleRefreshIdle, state.Phase)
					require.False(t, state.Usable)
				} else {
					require.NoError(t, err)
					require.Equal(t, before+1, f.calls.Load())
					require.True(t, state.Usable)
					require.EqualValues(t, 2, state.Snapshot.Revision)
				}
				require.True(t, fault.triggered, "the simulated database acknowledgement must be exercised")
			})
		}
	})

	t.Run("original_send_window", func(t *testing.T) {
		for _, point := range []string{"delayed_permit_ack", "paused_sender"} {
			t.Run(point, func(t *testing.T) {
				binding := f.reset(t)
				require.NoError(t, f.db.Exec("UPDATE workbench_stores SET service_expires_at = clock_timestamp() + interval '6 seconds' WHERE id = ?", f.storeID).Error)
				fault := &googleRefreshCommitFault{GoogleConnectionStore: f.repo, point: point}
				registration := f.application
				var paused bool
				if point == "paused_sender" {
					registration.OAuth = googlePausedOAuth{GoogleOAuth: registration.OAuth, paused: &paused}
				}
				service := f.connections(t, fault, registration)
				before := f.calls.Load()
				require.ErrorIs(t, f.acquire(service, binding), storecenter.ErrGoogleRefreshUnavailable)
				require.True(t, fault.triggered || paused)
				require.Equal(t, before, f.calls.Load(), "the original absolute deadline forbids any OAuth POST")
			})
		}
	})

	t.Run("possible_send_never_redeems_again", func(t *testing.T) {
		binding := f.reset(t)
		service := f.connections(t, f.repo, f.application)
		lost := func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) }
		f.hook.Store(&lost)
		defer f.hook.Store(&success)
		before := f.calls.Load()
		require.ErrorIs(t, f.acquire(service, binding), storecenter.ErrGoogleRefreshUnavailable)
		require.Equal(t, storecenter.GoogleRefreshPossible, f.refreshState(t, binding).Phase)
		require.NoError(t, f.db.Exec("UPDATE "+googleRefreshTableName+" SET deadline = clock_timestamp() - interval '1 second' WHERE store_id = ?", f.storeID).Error)
		require.ErrorIs(t, f.acquire(service, binding), storecenter.ErrGoogleReauthorization)
		require.ErrorIs(t, f.acquire(service, binding), storecenter.ErrGoogleReauthorization)
		require.Equal(t, before+1, f.calls.Load())
		state := f.refreshState(t, binding)
		require.Equal(t, storecenter.GoogleRefreshReauthorize, state.Phase)
		require.False(t, state.Usable)
	})

	t.Run("expired_claim_without_send_can_be_replaced", func(t *testing.T) {
		binding := f.reset(t)
		request := storecenter.GoogleCredentialRequest{OrganizationID: f.scope.OrganizationID, StoreID: f.storeID,
			Application: f.application.Application, Target: binding.Target, Generation: binding.Generation}
		prior, err := f.repo.GoogleCredential(f.ctx, request, uuid.NewString())
		require.NoError(t, err)
		require.Equal(t, storecenter.GoogleRefreshClaimed, prior.Phase)
		require.NoError(t, f.db.Exec("UPDATE "+googleRefreshTableName+" SET deadline=clock_timestamp()-interval '1 second' WHERE store_id=?", f.storeID).Error)
		before := f.calls.Load()
		require.NoError(t, f.acquire(f.connections(t, f.repo, f.application), binding))
		_, err = f.repo.AuthorizeGoogleRefresh(f.ctx, request, prior)
		require.ErrorIs(t, err, storecenter.ErrVersionConflict)
		require.Equal(t, before+1, f.calls.Load())
	})

	for _, mutation := range []string{"disconnect", "reauthorization_started", "reauthorization_bound", "member_grant_changed", "live_operation_permission_revoked", "credential_revision_advanced"} {
		t.Run(mutation, func(t *testing.T) { f.testLateReply(t, mutation, success) })
	}

	t.Run("original_target_and_generation", func(t *testing.T) {
		binding := f.reset(t)
		require.Equal(t, f.originalPlan.Target, binding.Target)
		require.Greater(t, binding.Generation, f.originalPlan.Generation)
		before := f.calls.Load()
		_, err := f.listing.Confirm(f.ctx, f.initialPlan.SourceID, f.initialPlan.ID)
		require.ErrorIs(t, err, storecenter.ErrVersionConflict)
		require.Equal(t, before, f.calls.Load(), "a prior generation cannot initiate another product send")
		queried, err := f.listing.Result(f.ctx, f.originalPlan.SourceID, f.originalPlan.ID)
		require.NoError(t, err)
		require.Equal(t, submission.ExecutionSucceeded, queried.Items[0].Status)
		require.NotNil(t, queried.Items[0].Observation)
		require.Equal(t, "accounts/123/dataSources/456", queried.Items[0].Observation.DataSource)
		require.Equal(t, before+1, f.calls.Load(), "the old submission is queried using the current grant for the same target")
	})

	t.Run("ineligible_requests_never_exchange", func(t *testing.T) {
		binding := f.reset(t)
		service := f.connections(t, f.repo, f.application)
		before := f.calls.Load()
		_, err := service.AcquirePublicationCredential(f.ctx, "another-organization", f.storeID, binding, googleRefreshAuthorized)
		require.ErrorIs(t, err, storecenter.ErrNotFound)
		wrong := binding
		wrong.Target.DataSourceID = "789"
		require.ErrorIs(t, f.acquire(service, wrong), storecenter.ErrVersionConflict)
		require.NoError(t, f.db.Exec("UPDATE workbench_stores SET service_expires_at = clock_timestamp() - interval '1 second' WHERE id = ?", f.storeID).Error)
		require.ErrorIs(t, f.acquire(service, binding), storecenter.ErrOfficialConnectionUnavailable)
		require.Equal(t, before, f.calls.Load())
	})
}

func (f googleRefreshFixture) testLateReply(t *testing.T, mutation string, success func(http.ResponseWriter, *http.Request)) {
	t.Helper()
	binding := f.reset(t)
	service := f.connections(t, f.repo, f.application)
	requestService := service
	if mutation == "member_grant_changed" {
		require.NoError(t, f.repo.SetMemberGrant(f.ctx, storecenter.MemberStoreGrantCommand{
			OrganizationID: f.scope.OrganizationID, StoreID: f.storeID, MemberID: "refresh-operator",
			OperationID: uuid.NewString(), Active: true,
		}))
		memberRepo, err := storecenter.NewMemberScopedStoreRepository(f.db, googleRefreshOperator{scope: f.scope})
		require.NoError(t, err)
		requestService = f.connections(t, memberRepo, f.application)
	}
	received, release := make(chan struct{}), make(chan struct{})
	delayed := func(w http.ResponseWriter, r *http.Request) {
		close(received)
		select {
		case <-release:
			success(w, r)
		case <-r.Context().Done():
		}
	}
	f.hook.Store(&delayed)
	defer f.hook.Store(&success)
	var released bool
	defer func() {
		if !released {
			close(release)
		}
	}()
	done := make(chan error, 1)
	var operationRevoked atomic.Bool
	check := func(ctx context.Context) error {
		if operationRevoked.Load() {
			return storecenter.ErrNotFound
		}
		return ctx.Err()
	}
	before := f.calls.Load()
	go func() {
		_, err := requestService.AcquirePublicationCredential(f.ctx, f.scope.OrganizationID, f.storeID, binding, check)
		done <- err
	}()
	awaitGoogleTestSignal(t, received)
	switch mutation {
	case "disconnect":
		store, err := f.repo.Get(f.ctx, f.scope.OrganizationID, f.storeID)
		require.NoError(t, err)
		require.NoError(t, service.Disconnect(f.ctx, f.scope.OrganizationID, f.storeID, store.Version()))
	case "reauthorization_started":
		f.begin(t, service)
	case "reauthorization_bound":
		f.complete(t, service, f.begin(t, service))
	case "member_grant_changed":
		cmd := storecenter.MemberStoreGrantCommand{OrganizationID: f.scope.OrganizationID, StoreID: f.storeID,
			MemberID: "refresh-operator", OperationID: uuid.NewString(), ExpectedVersion: 1, Active: false}
		require.NoError(t, f.repo.SetMemberGrant(f.ctx, cmd))
		cmd.OperationID, cmd.ExpectedVersion, cmd.Active = uuid.NewString(), 2, true
		require.NoError(t, f.repo.SetMemberGrant(f.ctx, cmd))
	case "live_operation_permission_revoked":
		operationRevoked.Store(true)
	case "credential_revision_advanced":
		request := storecenter.GoogleCredentialRequest{OrganizationID: f.scope.OrganizationID, StoreID: f.storeID,
			Application: f.application.Application, Target: binding.Target, Generation: binding.Generation}
		current := f.refreshState(t, binding)
		expiry := time.Now().Add(time.Hour)
		key, ciphertext, err := f.application.Protection.Seal(current.Snapshot.Binding,
			google.TokenSet{AccessToken: "winning-access", RefreshToken: "winning-refresh", ExpiresAt: expiry})
		require.NoError(t, err)
		require.NoError(t, f.repo.FinishGoogleRefresh(f.ctx, request, current, storecenter.GoogleRefreshOutcome{
			KeyID: key, Ciphertext: ciphertext, AccessExpiresAt: expiry,
		}))
	}
	close(release)
	released = true
	err := <-done
	grant, readErr := f.repo.ReadGoogleGrant(f.ctx, f.scope.OrganizationID, f.storeID, false)
	require.NoError(t, readErr)
	require.Equal(t, before+1, f.calls.Load(), "no second exchange follows a late response")
	switch mutation {
	case "disconnect":
		require.Error(t, err)
		require.True(t, grant.Revoked)
		require.Empty(t, grant.Ciphertext)
	case "reauthorization_started":
		require.Error(t, err)
		require.True(t, grant.Revoked)
		require.Equal(t, binding.Generation, grant.Generation)
	case "reauthorization_bound":
		require.Error(t, err)
		require.Equal(t, binding.Generation+1, grant.Generation)
		fallthrough
	case "member_grant_changed", "live_operation_permission_revoked", "credential_revision_advanced":
		credential, openErr := f.application.Protection.Open(grant.CredentialBinding(), grant.KeyID, grant.Ciphertext)
		require.NoError(t, openErr)
		if mutation == "credential_revision_advanced" {
			require.NoError(t, err)
			require.Equal(t, "winning-access", credential.AccessToken)
			require.EqualValues(t, 2, f.refreshState(t, binding).Snapshot.Revision)
			return
		}
		require.Error(t, err)
		require.Equal(t, "pretend-access", credential.AccessToken, "late refresh response did not overwrite the current grant")
	}
}

type googleRefreshOperator struct{ scope collection.Scope }

func (g googleRefreshOperator) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	if org != g.scope.OrganizationID {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: "operator", MemberID: "refresh-operator"}, nil
}

type googleRefreshCommitFault struct {
	storeapp.GoogleConnectionStore
	point     string
	triggered bool
}

func (g *googleRefreshCommitFault) GoogleCredential(ctx context.Context, request storecenter.GoogleCredentialRequest, claimID string) (storecenter.GoogleCredentialState, error) {
	state, err := g.GoogleConnectionStore.GoogleCredential(ctx, request, claimID)
	if err == nil && g.point == "claim" && claimID != "" && state.ClaimID == claimID && !g.triggered {
		g.triggered = true
		return state, errRefreshAcknowledgement
	}
	return state, err
}

func (g *googleRefreshCommitFault) AuthorizeGoogleRefresh(ctx context.Context, request storecenter.GoogleCredentialRequest, state storecenter.GoogleCredentialState) (time.Time, error) {
	deadline, err := g.GoogleConnectionStore.AuthorizeGoogleRefresh(ctx, request, state)
	if err == nil {
		switch g.point {
		case "permit":
			g.triggered = true
			return deadline, errRefreshAcknowledgement
		case "delayed_permit_ack":
			g.triggered = true
			time.Sleep(time.Until(deadline.Add(-storecenter.GoogleRefreshSaveReserve)) + 25*time.Millisecond)
		}
	}
	return deadline, err
}

func (g *googleRefreshCommitFault) FinishGoogleRefresh(ctx context.Context, request storecenter.GoogleCredentialRequest, state storecenter.GoogleCredentialState, result storecenter.GoogleRefreshOutcome) error {
	if !g.triggered && result.Ciphertext != "" {
		switch g.point {
		case "committed_save":
			if err := g.GoogleConnectionStore.FinishGoogleRefresh(ctx, request, state, result); err != nil {
				return err
			}
			g.triggered = true
			return errRefreshAcknowledgement
		case "uncommitted_save":
			g.triggered = true
			return errRefreshAcknowledgement
		}
	}
	return g.GoogleConnectionStore.FinishGoogleRefresh(ctx, request, state, result)
}

type googlePausedOAuth struct {
	storeapp.GoogleOAuth
	paused *bool
}

func (g googlePausedOAuth) Refresh(ctx context.Context, token google.TokenSet) (google.TokenSet, error) {
	*g.paused = true
	<-ctx.Done()
	return g.GoogleOAuth.Refresh(ctx, token)
}
