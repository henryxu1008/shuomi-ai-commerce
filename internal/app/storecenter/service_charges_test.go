package storecenterapp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	resourceadapter "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/orgresource"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type serviceContextAccess struct{}

func (serviceContextAccess) AuthorizeStoreMember(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || id.EffectiveOrganizationID != org {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	a := serviceTestAuthorizer()
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: id.UserID, MemberID: id.EffectiveMemberID, Administrator: a.IsTenantAdmin(id.UserID, id.Roles), CanWrite: authz.AllowedOrganization(ctx, a, id.UserID, org, id.Roles, authz.PermissionWorkbenchStoreUpdate)}, nil
}

type delayedStoreSettlement struct{ orgresource.ConsumerChargePort }

func (p delayedStoreSettlement) Reconcile(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	return orgresource.ConsumerChargeReceipt{}, errors.New("synthetic settlement transport failure")
}

func TestDeletedStoreKeepsOriginalPaidProofForResourceRecovery(t *testing.T) {
	f := newServiceChargeFixture(t)
	operation := uuid.NewString()
	_, err := serviceApplication(t, f, delayedStoreSettlement{f.ledger}).Activate(f.member, storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: operation, ExpectedStoreVersion: 1})
	require.Error(t, err)
	proof, err := f.repo.ReadServiceChargeProof(context.Background(), "org-a", operation)
	require.NoError(t, err)
	require.Equal(t, "succeeded", proof.State)
	_, err = f.repo.DeleteRecord(f.admin, storecenter.DeleteStoreRequest{OrganizationID: "org-a", StoreID: f.store.ID(), ActorSubject: "admin", OperationKey: uuid.NewString(), ExpectedVersion: 2}, time.Now().UTC())
	require.NoError(t, err)
	_, err = f.repo.Get(f.member, "org-a", f.store.ID())
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	n, err := f.ledger.RecoverDue(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, n)
	position, err := f.positions.ReadPosition(context.Background(), "org-a", "member-a", orgresource.ResourceStoreRenewalPeriod)
	require.NoError(t, err)
	require.Zero(t, position.Reserved)
	require.EqualValues(t, 1, position.Consumed)
	retained, err := f.repo.ReadServiceChargeProof(context.Background(), "org-a", operation)
	require.NoError(t, err)
	require.Equal(t, proof.EvidenceID, retained.EvidenceID)
	require.Equal(t, proof.Snapshot, retained.Snapshot)
	n, err = f.ledger.RecoverDue(context.Background())
	require.NoError(t, err)
	require.Zero(t, n)
}

type connectedStore struct{}

func (connectedStore) Status(context.Context, storecenter.ConnectionStatusInput) (storecenter.ConnectionStatus, error) {
	return storecenter.ConnectionStatusConnected, nil
}
func serviceContext(user, member, role string) context.Context {
	return authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: user, EffectiveMemberID: member, Roles: []string{role}, OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: "org-a", AuthorizationID: member, Roles: []string{role}}}})
}

type serviceChargeFixture struct {
	repo          *storecenter.MemberScopedStoreRepository
	ledger        *orgresource.ConsumerChargeService
	positions     *resourceadapter.GormMemberAllocationRepository
	resourceDB    *gorm.DB
	store         *storecenter.Store
	shopify       storecenter.ShopifyBinding
	member, admin context.Context
}

func newServiceChargeFixture(t *testing.T) serviceChargeFixture {
	t.Helper()
	return newServiceChargeFixtureForPlatform(t, storecenter.PlatformShein)
}

func newServiceChargeFixtureForPlatform(t *testing.T, platform storecenter.Platform) serviceChargeFixture {
	t.Helper()
	open := func(name string) *gorm.DB {
		db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), name)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		pool.SetMaxOpenConns(1)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	resourceDB := open("resource.db")
	var storeDB *gorm.DB
	if platform == storecenter.PlatformShopify {
		storeDB = newShopifyChargeStoreDatabase(t)
	} else {
		storeDB = open("store.db")
		require.NoError(t, storecenter.AutoMigrateStoreRepository(storeDB))
	}
	require.NoError(t, resourceadapter.AutoMigrate(resourceDB))
	require.NoError(t, resourceDB.Table("saas_organization_resource_buckets").Create(map[string]any{"organization_id": "org-a", "resource_type": "store_renewal_period", "available": 4, "reserved": 0, "consumed": 0, "created_at": time.Now().UTC(), "updated_at": time.Now().UTC()}).Error)
	positions, err := resourceadapter.NewGormMemberAllocationRepository(resourceDB, resourceadapter.TransactionConfig{})
	require.NoError(t, err)
	_, err = positions.Transfer(context.Background(), orgresource.MemberResourceTransfer{OrganizationID: "org-a", MemberID: "member-a", ActorID: "admin", OperationID: "allocate", ResourceType: orgresource.ResourceStoreRenewalPeriod, Action: orgresource.MemberResourceAllocate, Quantity: 3})
	require.NoError(t, err)
	repo, err := storecenter.NewMemberScopedStoreRepository(storeDB, serviceContextAccess{})
	require.NoError(t, err)
	member, admin := serviceContext("operator", "member-a", authz.EnterpriseRoleKey("org-a", 1)), serviceContext("admin", "member-admin", "listingkit_admin")
	external := "external"
	if platform == storecenter.PlatformShopify {
		external = ""
	}
	store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "operator", Name: "Store", Platform: string(platform), Region: "SG", ExternalStoreID: external, CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC().Add(-time.Minute)})
	require.NoError(t, err)
	store, _, err = repo.CreateOrReplay(member, "org-a", store)
	require.NoError(t, err)
	owner, err := NewServiceChargeOwner(repo)
	require.NoError(t, err)
	chargeRepo, err := resourceadapter.NewGormConsumerChargeRepository(resourceDB, resourceadapter.TransactionConfig{})
	require.NoError(t, err)
	ledger, err := orgresource.NewConsumerChargeService(chargeRepo, map[orgresource.ResourceConsumer]orgresource.ConsumerChargeOwner{orgresource.ConsumerStoreService: owner})
	require.NoError(t, err)
	f := serviceChargeFixture{repo: repo, ledger: ledger, positions: positions, resourceDB: resourceDB, store: store, member: member, admin: admin}
	if platform == storecenter.PlatformShopify {
		installation := storecenter.ShopifyInstallation{
			AppID: "gid://shopify/App/812", ShopID: "gid://shopify/Shop/19",
			InstallationID: "gid://shopify/AppInstallation/27", Domain: "fixture.myshopify.com", Currency: "USD",
		}
		f.shopify = pairShopifyChargeStore(t, repo, member, store, installation)
		f.store, err = repo.Get(member, "org-a", store.ID())
		require.NoError(t, err)
	}
	return f
}

func newShopifyChargeStoreDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("shopify_charge"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("isolated-shopify-charge-test"), tcpostgres.BasicWaitStrategies())
	if container != nil {
		t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	}
	require.NoError(t, err)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	return db
}

func pairShopifyChargeStore(t *testing.T, repo *storecenter.MemberScopedStoreRepository, ctx context.Context, store *storecenter.Store, installation storecenter.ShopifyInstallation) storecenter.ShopifyBinding {
	t.Helper()
	session, state := strings.Repeat("a", 64), strings.Repeat("b", 64)
	command := storecenter.OfficialConnectionCommand{OrganizationID: store.OrganizationID(), StoreID: store.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: store.Version()}
	now := time.Now().UTC().Truncate(time.Microsecond)
	begun, err := repo.BeginShopifyPairing(ctx, command, installation.AppID, session, state, now)
	require.NoError(t, err)
	claimed, err := repo.ClaimShopifyPairing(ctx, begun.Key(), session, state, now)
	require.NoError(t, err)
	generation, err := repo.ShopifyPairingGeneration(ctx, claimed.Key(), session, installation, now)
	require.NoError(t, err)
	staged, err := repo.StageShopifyPairing(ctx, claimed.Key(), session, installation, generation, "fixture-key", "fixture-ciphertext", now)
	require.NoError(t, err)
	binding, err := repo.ConfirmShopifyPairing(ctx, staged.Key(), session, installation, now, 0)
	require.NoError(t, err)
	return binding
}
func serviceApplication(t *testing.T, f serviceChargeFixture, port orgresource.ConsumerChargePort) *storecenter.ServiceLifecycleApplication {
	t.Helper()
	return serviceApplicationWithConnections(t, f, port, connectedStore{})
}

func serviceApplicationWithConnections(t *testing.T, f serviceChargeFixture, port orgresource.ConsumerChargePort, connections storecenter.ConnectionStatusProvider) *storecenter.ServiceLifecycleApplication {
	t.Helper()
	executor, err := NewServiceLifecycleExecutor(f.repo, port, connections)
	require.NoError(t, err)
	application, err := storecenter.NewServiceLifecycleApplication(f.repo, executor, connections, serviceTestAuthorizer(), storecenter.Phase1ServiceQuantityPolicy{}, time.Now)
	require.NoError(t, err)
	return application
}
func TestServiceChargesUseSeparateOwnersAndOriginalFundingOnReplay(t *testing.T) {
	f := newServiceChargeFixture(t)
	application := serviceApplication(t, f, f.ledger)
	activation := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: 1}
	first, err := application.Activate(f.member, activation)
	require.NoError(t, err)
	require.Equal(t, "2", first.Snapshot.BalanceAfter)
	renewal := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: 2, Quantity: 2}
	renewed, err := application.Renew(f.member, renewal)
	require.NoError(t, err)
	require.EqualValues(t, 3, renewed.Snapshot.StoreVersion)
	require.Equal(t, "0", renewed.Snapshot.BalanceAfter)
	require.Equal(t, 60*24*time.Hour, renewed.Snapshot.ServiceState.ExpiresAt.Sub(*first.Snapshot.ServiceState.ExpiresAt))
	replay, err := application.Renew(f.member, renewal)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, renewed.Snapshot, replay.Snapshot)
	_, err = application.Renew(f.member, storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: 3, Quantity: 1})
	require.ErrorIs(t, err, orgresource.ErrInsufficientBalance)
	adminRenewed, err := application.Renew(f.admin, storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: 3, Quantity: 1})
	require.NoError(t, err)
	require.EqualValues(t, 4, adminRenewed.Snapshot.StoreVersion)
	position, err := f.positions.ReadPosition(context.Background(), "org-a", "member-a", orgresource.ResourceStoreRenewalPeriod)
	require.NoError(t, err)
	require.Zero(t, position.Free)
	require.Zero(t, position.Reserved)
	require.EqualValues(t, 3, position.Consumed)
	var bucket struct{ Available, Allocated, Reserved, Consumed int64 }
	require.NoError(t, f.resourceDB.Table("saas_organization_resource_buckets").Take(&bucket).Error)
	require.Zero(t, bucket.Available)
	require.Zero(t, bucket.Allocated)
	require.Zero(t, bucket.Reserved)
	require.EqualValues(t, 4, bucket.Consumed)
	promoted := serviceContext("operator", "member-a", "listingkit_admin")
	replayedActivation, err := application.Activate(promoted, activation)
	require.NoError(t, err)
	require.Equal(t, "2", replayedActivation.Snapshot.BalanceAfter, "must return the original funding receipt, even after promotion and other spending")
	_, err = application.Renew(serviceContext("operator", "member-a", "listingkit_viewer"), renewal)
	require.ErrorIs(t, err, storecenter.ErrServiceLifecyclePermissionDenied)
}

type revokeOnReserve struct {
	orgresource.ConsumerChargePort
	once func()
}

func (p *revokeOnReserve) Reserve(ctx context.Context, id orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	receipt, err := p.ConsumerChargePort.Reserve(ctx, id)
	if err == nil && p.once != nil {
		action := p.once
		p.once = nil
		action()
	}
	return receipt, err
}
func TestServiceChargeReleasesOriginalMemberWhenGrantRevokedBeforeBinding(t *testing.T) {
	f := newServiceChargeFixture(t)
	port := &revokeOnReserve{ConsumerChargePort: f.ledger, once: func() {
		require.NoError(t, f.repo.SetMemberGrant(f.admin, storecenter.MemberStoreGrantCommand{OrganizationID: "org-a", StoreID: f.store.ID(), MemberID: "member-a", OperationID: uuid.NewString(), ExpectedVersion: 1}))
	}}
	application := serviceApplication(t, f, port)
	request := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: 1}
	_, err := application.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	position, err := f.positions.ReadPosition(context.Background(), "org-a", "member-a", orgresource.ResourceStoreRenewalPeriod)
	require.NoError(t, err)
	require.EqualValues(t, 3, position.Free)
	require.Zero(t, position.Reserved)
	require.Zero(t, position.Consumed)
	require.NoError(t, f.repo.SetMemberGrant(f.admin, storecenter.MemberStoreGrantCommand{OrganizationID: "org-a", StoreID: f.store.ID(), MemberID: "member-a", OperationID: uuid.NewString(), ExpectedVersion: 2, Active: true}))
	_, err = application.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrNotFound, "restoring assignment cannot revive a failed command")
	store, err := f.repo.Get(f.member, "org-a", f.store.ID())
	require.NoError(t, err)
	require.EqualValues(t, 1, store.Version())
	require.Nil(t, store.Snapshot().ServiceExpiresAt)
}

// The Workbench list still reports connected after the Shopify installation
// can no longer be verified. Paid commands must use the live check.
type shopifyServiceConnectionFixture struct {
	connectedStore
	binding   storecenter.ShopifyBinding
	err       error
	calls     []storecenter.ConnectionStatusInput
	afterRead func()
}

func (f *shopifyServiceConnectionFixture) VerifyShopifyServiceConnection(_ context.Context, input storecenter.ConnectionStatusInput) (storecenter.ShopifyBinding, error) {
	f.calls = append(f.calls, input)
	if len(f.calls) == 2 && f.afterRead != nil {
		f.afterRead()
	}
	return f.binding, f.err
}

func TestShopifyServiceRejectsRevokedAuthorizationBeforeReserving(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	connection := &shopifyServiceConnectionFixture{binding: f.shopify, err: storecenter.ErrConnectionUnavailable}
	reservations := 0
	ledger := &revokeOnReserve{ConsumerChargePort: f.ledger, once: func() { reservations++ }}
	app := serviceApplicationWithConnections(t, f, ledger, connection)
	request := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: f.store.Version()}

	_, err := app.Activate(f.admin, request)
	require.ErrorIs(t, err, storecenter.ErrConnectionNotFresh)
	require.Zero(t, reservations)
	require.Len(t, connection.calls, 1)
	require.Equal(t, storecenter.ConnectionStatusInput{OrganizationID: "org-a", StoreID: f.store.ID(), Platform: storecenter.PlatformShopify, ConnectionRef: f.shopify.ShopKey}, connection.calls[0])
	_, err = f.repo.ReadServiceChargeIntent(f.admin, "org-a", request.OperationID)
	require.ErrorIs(t, err, storecenter.ErrNotFound)
	store, err := f.repo.Get(f.admin, "org-a", request.StoreID)
	require.NoError(t, err)
	require.Equal(t, f.store.Version(), store.Version())

	connection.err = nil
	activation, err := app.Activate(f.member, request)
	require.NoError(t, err)
	require.Equal(t, f.store.Version()+1, activation.Snapshot.StoreVersion)
	require.Equal(t, "2", activation.Snapshot.BalanceAfter)
	require.Len(t, connection.calls, 3, "verify before admission and before applying the service change")
	connection.err = storecenter.ErrConnectionUnavailable

	for _, purchase := range []func(context.Context, storecenter.ServiceLifecycleApplicationRequest) (storecenter.ServiceOperationResult, error){app.Renew, app.Reactivate} {
		newRequest := storecenter.ServiceLifecycleApplicationRequest{StoreID: request.StoreID, OperationID: uuid.NewString(), ExpectedStoreVersion: activation.Snapshot.StoreVersion, Quantity: 1}
		_, err = purchase(f.member, newRequest)
		require.ErrorIs(t, err, storecenter.ErrConnectionNotFresh)
		_, err = f.ledger.Lookup(f.member, storeServiceChargeIdentity(newRequest.OperationID))
		require.ErrorIs(t, err, orgresource.ErrReservationNotFound)
	}
	require.Len(t, connection.calls, 5)
	replayed, err := app.Activate(f.member, request)
	require.NoError(t, err)
	require.True(t, replayed.Replayed)
	require.Equal(t, activation.Snapshot, replayed.Snapshot)
	require.Len(t, connection.calls, 5, "a completed service operation reads its original receipt")
}

func TestShopifyServiceReleasesOriginalReservationWhenVerificationFailsAfterReserve(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	connection := &shopifyServiceConnectionFixture{binding: f.shopify}
	ledger := &revokeOnReserve{ConsumerChargePort: f.ledger, once: func() {
		connection.err = storecenter.ErrConnectionUnavailable
	}}
	app := serviceApplicationWithConnections(t, f, ledger, connection)
	request := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: f.store.Version()}

	_, err := app.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrShopifyServiceChargeRejected)
	require.Len(t, connection.calls, 2)
	receipt, err := f.ledger.Lookup(f.member, storeServiceChargeIdentity(request.OperationID))
	require.NoError(t, err)
	require.Equal(t, orgresource.ReservationReleased, receipt.State)
	position, err := f.positions.ReadPosition(f.member, "org-a", "member-a", orgresource.ResourceStoreRenewalPeriod)
	require.NoError(t, err)
	require.EqualValues(t, 3, position.Free)
	require.Zero(t, position.Reserved)
	require.Zero(t, position.Consumed)

	connection.err = nil
	_, err = app.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrShopifyServiceChargeRejected, "reauthorization cannot revive the failed operation")
	require.Len(t, connection.calls, 2)
	store, err := f.repo.Get(f.member, "org-a", request.StoreID)
	require.NoError(t, err)
	require.Equal(t, f.store.Version(), store.Version())
	require.Nil(t, store.Snapshot().ServiceStartedAt)
}

func TestShopifyServiceRecoversOriginalReservationAfterInterruptedRequest(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	connection := &shopifyServiceConnectionFixture{binding: f.shopify}
	call, cancel := context.WithCancel(f.member)
	defer cancel()
	ledger := &revokeOnReserve{ConsumerChargePort: f.ledger, once: cancel}
	app := serviceApplicationWithConnections(t, f, ledger, connection)
	request := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: f.store.Version()}

	_, err := app.Activate(call, request)
	require.ErrorIs(t, err, storecenter.ErrServiceChargeUnknown)
	original, err := f.ledger.Lookup(f.member, storeServiceChargeIdentity(request.OperationID))
	require.NoError(t, err)
	require.Equal(t, orgresource.ReservationReserved, original.State)
	connection.err = storecenter.ErrConnectionUnavailable
	_, err = app.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrShopifyServiceChargeRejected)
	terminal, err := f.ledger.Lookup(f.member, storeServiceChargeIdentity(request.OperationID))
	require.NoError(t, err)
	require.Equal(t, original.ReservationID, terminal.ReservationID)
	require.Equal(t, orgresource.ReservationReleased, terminal.State)
}

func TestShopifyServiceRejectsSameShopRebindAfterLiveCheck(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	candidate, err := storecenter.NewStore(storecenter.CreateStoreInput{
		ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "operator", Name: "Replacement",
		Platform: string(storecenter.PlatformShopify), Region: "SG", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	replacement, _, err := f.repo.CreateOrReplay(f.member, "org-a", candidate)
	require.NoError(t, err)
	connection := &shopifyServiceConnectionFixture{binding: f.shopify, afterRead: func() {
		newBinding := pairShopifyChargeStore(t, f.repo, f.member, replacement, f.shopify.ShopifyInstallation)
		require.Equal(t, f.shopify.ShopKey, newBinding.ShopKey)
		require.Equal(t, replacement.ID(), newBinding.StoreID)
		require.NotEqual(t, f.shopify.Generation, newBinding.Generation)
	}}
	app := serviceApplicationWithConnections(t, f, f.ledger, connection)
	request := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: f.store.Version()}

	_, err = app.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrShopifyServiceChargeRejected)
	require.Len(t, connection.calls, 2)
	store, err := f.repo.Get(f.member, "org-a", request.StoreID)
	require.NoError(t, err)
	require.Equal(t, f.store.Version(), store.Version(), "rebind leaves the old Store version intact")
	require.Nil(t, store.Snapshot().ServiceStartedAt)
	receipt, err := f.ledger.Lookup(f.member, storeServiceChargeIdentity(request.OperationID))
	require.NoError(t, err)
	require.Equal(t, orgresource.ReservationReleased, receipt.State)
}

func TestShopifyServiceRejectsPreviousVersionAfterReauthorization(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	connection := &shopifyServiceConnectionFixture{binding: f.shopify, afterRead: func() {
		renewed := pairShopifyChargeStore(t, f.repo, f.member, f.store, f.shopify.ShopifyInstallation)
		require.Equal(t, f.shopify.ShopKey, renewed.ShopKey)
		require.Greater(t, renewed.Generation, f.shopify.Generation)
	}}
	app := serviceApplicationWithConnections(t, f, f.ledger, connection)
	request := storecenter.ServiceLifecycleApplicationRequest{StoreID: f.store.ID(), OperationID: uuid.NewString(), ExpectedStoreVersion: f.store.Version()}

	_, err := app.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrShopifyServiceChargeRejected)
	receipt, err := f.ledger.Lookup(f.member, storeServiceChargeIdentity(request.OperationID))
	require.NoError(t, err)
	require.Equal(t, orgresource.ReservationReleased, receipt.State)
	_, err = app.Activate(f.member, request)
	require.ErrorIs(t, err, storecenter.ErrShopifyServiceChargeRejected)
	require.Len(t, connection.calls, 2, "the original terminal result supersedes another verification")
}

type shopifyServiceTransport func(*http.Request) (*http.Response, error)

func (f shopifyServiceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestShopifyServiceVerificationUsesLiveInstallationAndScope(t *testing.T) {
	f := newServiceChargeFixtureForPlatform(t, storecenter.PlatformShopify)
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	for _, tc := range []struct {
		name, scope, installation string
		tokenStatus               int
		wantErr                   error
	}{
		{"uninstalled", "", "", http.StatusUnauthorized, storecenter.ErrConnectionUnavailable},
		{"scope revoked", "read_products", f.shopify.InstallationID, http.StatusOK, storecenter.ErrConnectionUnavailable},
		{"different installation", "write_products", "gid://shopify/AppInstallation/28", http.StatusOK, storecenter.ErrConnectionNotFresh},
		{"current installation", "write_products", f.shopify.InstallationID, http.StatusOK, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var paths []string
			client := &http.Client{Transport: shopifyServiceTransport(func(req *http.Request) (*http.Response, error) {
				require.Equal(t, f.shopify.Domain, req.URL.Host)
				paths = append(paths, req.URL.Path)
				status := tc.tokenStatus
				payload := `{"access_token":"synthetic-token","scope":"write_products","expires_in":3600}`
				if req.URL.Path != "/admin/oauth/access_token" {
					require.Equal(t, "/admin/api/2026-10/graphql.json", req.URL.Path)
					require.Equal(t, "synthetic-token", req.Header.Get("X-Shopify-Access-Token"))
					query, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.Contains(t, string(query), "currentAppInstallation")
					status = http.StatusOK
					payload = fmt.Sprintf(`{"data":{"shop":{"id":%q,"myshopifyDomain":%q,"currencyCode":"USD","plan":{"partnerDevelopment":true}},"currentAppInstallation":{"id":%q,"app":{"id":%q},"accessScopes":[{"handle":%q}]}}}`,
						f.shopify.ShopID, f.shopify.Domain, tc.installation, f.shopify.AppID, tc.scope)
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
			})}
			issuer, err := wire.NewAppTokenIssuer(client, "synthetic-app", "synthetic-key", key)
			require.NoError(t, err)
			connections := &OfficialConnections{shopify: &ShopifyConnections{store: f.repo, app: ShopifyApplication{
				AppID: f.shopify.AppID, Admin: wire.NewDevelopmentAdminClient(client), Issuer: issuer,
			}}}
			binding, err := storecenter.VerifyShopifyServiceConnection(f.member, connections, f.store)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				require.Zero(t, binding)
			} else {
				require.NoError(t, err)
				require.Equal(t, f.shopify, binding)
			}
			if tc.tokenStatus == http.StatusOK {
				require.Equal(t, []string{"/admin/oauth/access_token", "/admin/api/2026-10/graphql.json"}, paths)
			} else {
				require.Equal(t, []string{"/admin/oauth/access_token"}, paths)
			}
		})
	}
}

func storeServiceChargeIdentity(operationID string) orgresource.ConsumerChargeIdentity {
	return orgresource.ConsumerChargeIdentity{OrganizationID: "org-a", Consumer: orgresource.ConsumerStoreService, OperationID: operationID}
}

type serviceRolePolicyFixture struct{}

func (serviceRolePolicyFixture) RoleModules(_ context.Context, org string, keys []string) (map[string][]string, error) {
	result := map[string][]string{}
	for _, key := range keys {
		if key == authz.EnterpriseRoleKey(org, 1) {
			result[key] = []string{"stores"}
		}
	}
	return result, nil
}
func serviceTestAuthorizer() *authz.ListingKitAuthorizer {
	a, err := authz.NewListingKitAuthorizer(nil, nil)
	if err != nil {
		panic(err)
	}
	a.SetRolePolicyReader(serviceRolePolicyFixture{})
	return a
}
