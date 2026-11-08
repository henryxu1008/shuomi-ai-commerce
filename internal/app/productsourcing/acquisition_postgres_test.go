package productsourcing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/sourcing"
)

func acquisitionDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ISSUE398_TEST_DSN")
	if dsn == "" {
		t.Skip("result=SKIP: ISSUE398_TEST_DSN must target task-owned PostgreSQL")
	}
	require.Contains(t, dsn, "host=127.0.0.1")
	require.Contains(t, dsn, "user=issue398_owner")
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	rootPool, err := root.DB()
	require.NoError(t, err)
	rootPool.SetMaxOpenConns(2)
	name := "issue398_app_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	db, err := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() {
		require.NoError(t, pool.Close())
		require.NoError(t, root.Exec("DROP DATABASE "+name+" WITH (FORCE)").Error)
		require.NoError(t, rootPool.Close())
	})
	return db
}

// Synthetic typed evidence keeps publication, authorization and recovery tests
// independent of the retired static-HTML provider.
type acquisitionFixtureProvider func(context.Context, sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error)

func (f acquisitionFixtureProvider) Acquire(ctx context.Context, source sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error) {
	return f(ctx, source)
}

func acquisitionFixture(t *testing.T) (sourcing.PublicAcquirer, *atomic.Int32, func(string)) {
	t.Helper()
	var title atomic.Value
	title.Store("Public fixture bottle")
	count := &atomic.Int32{}
	provider := acquisitionFixtureProvider(func(_ context.Context, source sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error) {
		count.Add(1)
		require.Equal(t, "https://detail.1688.com/offer/981645030344.html", source.URL)
		text := title.Load().(string)
		digest := sha256.Sum256([]byte(text))
		return sourcing.AcquisitionEvidence{
			SchemaVersion: 1, SourceURL: source.URL, OfferID: source.OfferID, Title: &text,
			Images:     []sourcing.AcquisitionImage{{URL: "https://cbu01.alicdn.com/fixture-bottle.jpg", Role: "primary"}},
			CapturedAt: time.Date(2026, 9, 12, 0, 0, 0, 0, time.UTC), ContentSHA256: hex.EncodeToString(digest[:]), ParserVersion: "synthetic-evidence-v1",
		}, nil
	})
	return provider, count, func(text string) { title.Store(text) }
}

func acquisitionIdentity(org, actor string) context.Context {
	return authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: org, EffectiveOrganizationID: org, EffectiveMemberID: "membership:" + actor, UserID: actor, TokenExpiresAt: time.Now().Add(time.Hour)})
}

func TestAcquisitionBusinessChainPublishesExactVersionsAndObservesRevocation(t *testing.T) {
	db := acquisitionDatabase(t)
	provider, fetches, changeTitle := acquisitionFixture(t)
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(acquisitionTestRolePolicy{})
	var revoked atomic.Bool
	live := liveRolesFunc(func(_ context.Context, org, actor string) ([]string, error) {
		if revoked.Load() {
			return []string{"listingkit_viewer"}, nil
		}
		return []string{authz.EnterpriseRoleKey(org, 1)}, nil
	})
	_, err = NewPublicAcquisition(context.Background(), db, live, permissions, provider, newTestResourceChargePort(t, db))
	require.Error(t, err)
	var tables int64
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public'").Scan(&tables).Error)
	require.Zero(t, tables, "ordinary construction never installs schema")
	require.NoError(t, InstallAcquisitionSchema(db))
	service, err := NewPublicAcquisition(context.Background(), db, live, permissions, provider, newTestResourceChargePort(t, db))
	require.NoError(t, err)
	ctx := acquisitionIdentity("org-a", "actor-a")
	key := uuid.NewString()
	first, err := service.Acquire(ctx, key, "981645030344")
	require.NoError(t, err)
	require.NotNil(t, first.Publication)
	require.Equal(t, "Public fixture bottle", first.Publication.Snapshot.Title)
	require.Equal(t, "crawler:1688:981645030344", first.Publication.Receipt.ProductKey)
	require.EqualValues(t, 1, first.Publication.Receipt.CatalogVersion)
	require.Equal(t, sourcing.AcquisitionProducerKind, first.Publication.Receipt.Producer.Kind)
	require.Equal(t, "source-run:acquisition:"+first.Operation.ID, first.Publication.Receipt.PublicationID)
	require.NotEmpty(t, first.Publication.Envelope.MissingFacts)
	require.Len(t, first.Publication.Snapshot.Images, 1)
	changeTitle("Second fixture revision")
	second, err := service.Acquire(ctx, uuid.NewString(), "981645030344")
	require.NoError(t, err)
	require.EqualValues(t, 2, second.Publication.Receipt.CatalogVersion)
	require.Equal(t, "Second fixture revision", second.Publication.Snapshot.Title)
	replayed, err := service.Acquire(ctx, key, "http://DETAIL.1688.COM/offer/981645030344.html?tracking=discarded#discarded")
	require.NoError(t, err)
	require.True(t, replayed.Replayed)
	require.Equal(t, first.Publication, replayed.Publication, "replay reads exact old publication, never latest")
	require.EqualValues(t, 2, fetches.Load())
	_, err = service.Acquire(ctx, key, "12345")
	require.ErrorIs(t, err, sourcing.ErrAcquisitionConflict)
	_, err = service.Read(acquisitionIdentity("org-b", "actor-a"), first.Operation.ID)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionNotFound)
	_, err = service.Read(acquisitionIdentity("org-a", "actor-b"), first.Operation.ID)
	require.ErrorIs(t, err, sourcing.ErrAcquisitionNotFound)
	revoked.Store(true)
	_, err = service.Acquire(ctx, uuid.NewString(), "981645030344")
	require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
	_, err = service.Acquire(ctx, key, "981645030344")
	require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
	_, err = service.Verify(ctx, key, "981645030344")
	require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
	_, err = service.Read(ctx, first.Operation.ID)
	require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
	require.EqualValues(t, 2, fetches.Load())
	var versions, publications, receipts, operations int64
	require.NoError(t, db.Table("product_snapshot_versions").Count(&versions).Error)
	require.NoError(t, db.Table("product_source_publications").Count(&publications).Error)
	require.NoError(t, db.Table("product_source_publication_receipts").Count(&receipts).Error)
	require.NoError(t, db.Table("product_acquisition_operations").Count(&operations).Error)
	require.EqualValues(t, 2, versions)
	require.Equal(t, versions, publications)
	require.Equal(t, versions, receipts)
	require.Equal(t, versions, operations)
}

type acquisitionLostPublicationResponse struct {
	sourcing.AcquisitionPublisher
	publishes, verifies int
	command             []byte
}

func (p *acquisitionLostPublicationResponse) Publish(ctx context.Context, command sourcing.PublicationCommand) (sourcing.PublicationReceipt, error) {
	p.publishes++
	p.command, _ = json.Marshal(command)
	_, err := p.AcquisitionPublisher.Publish(ctx, command)
	if err != nil {
		return sourcing.PublicationReceipt{}, err
	}
	return sourcing.PublicationReceipt{}, sourcing.ErrSourcePublicationOutcomeUnknown
}
func (p *acquisitionLostPublicationResponse) Verify(ctx context.Context, command sourcing.PublicationCommand) (sourcing.PublicationReceipt, error) {
	p.verifies++
	return p.AcquisitionPublisher.Verify(ctx, command)
}

func TestAcquisitionBusinessChainRecoversLostResponseUsingFrozenCommand(t *testing.T) {
	db := acquisitionDatabase(t)
	require.NoError(t, InstallAcquisitionSchema(db))
	provider, fetches, _ := acquisitionFixture(t)
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(acquisitionTestRolePolicy{})
	live := liveRolesFunc(func(_ context.Context, org, _ string) ([]string, error) {
		return []string{authz.EnterpriseRoleKey(org, 1)}, nil
	})
	service, err := NewPublicAcquisition(context.Background(), db, live, permissions, provider, newTestResourceChargePort(t, db))
	require.NoError(t, err)
	lost := &acquisitionLostPublicationResponse{AcquisitionPublisher: service.publisher}
	service.publisher = lost
	ctx := acquisitionIdentity("org-recovery", "actor")
	key := uuid.NewString()
	_, err = service.Acquire(ctx, key, "981645030344")
	require.ErrorIs(t, err, sourcing.ErrAcquisitionUnknown)
	op, err := service.operations.ByKey(ctx, sourcing.PublicationScope{OrganizationID: "org-recovery", ActorID: "actor"}, key)
	require.NoError(t, err)
	require.Equal(t, sourcing.AcquisitionPublished, op.State)
	raw, err := json.Marshal(op.Command)
	require.NoError(t, err)
	require.Equal(t, lost.command, raw)
	// Rebuild all application/persistence owners. There is no provider or base
	// reader available to the recovery call, and no final table was seeded.
	rebuilt, err := NewPublicAcquisition(context.Background(), db, live, permissions, provider, newTestResourceChargePort(t, db))
	require.NoError(t, err)
	rebuilt.provider = nil
	rebuilt.reader = nil
	proof := &acquisitionLostPublicationResponse{AcquisitionPublisher: rebuilt.publisher}
	rebuilt.publisher = proof
	result, err := rebuilt.Verify(ctx, key, "https://detail.1688.com/offer/981645030344.html?discard=1")
	require.NoError(t, err)
	require.Equal(t, sourcing.AcquisitionPublished, result.Operation.State)
	require.Equal(t, "Public fixture bottle", result.Publication.Snapshot.Title)
	require.EqualValues(t, 1, result.Publication.Receipt.CatalogVersion)
	require.Equal(t, op.CommandHash, result.Operation.CommandHash)
	replay, err := rebuilt.Acquire(ctx, key, "981645030344")
	require.NoError(t, err)
	require.Equal(t, result.Publication, replay.Publication)
	require.Equal(t, 1, lost.publishes)
	require.Zero(t, proof.publishes)
	require.Equal(t, 2, proof.verifies)
	require.EqualValues(t, 1, fetches.Load())
	var versions int64
	require.NoError(t, db.Table("product_snapshot_versions").Count(&versions).Error)
	require.EqualValues(t, 1, versions)
}
