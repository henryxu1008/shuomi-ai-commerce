package acquisitionplan_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	planstore "task-processor/internal/integration/persistence/product/acquisitionplan"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

type transferAuth struct {
	scope  collection.Scope
	denied bool
}

func (a *transferAuth) Authorize(context.Context, string) (collection.Scope, error) {
	if a.denied {
		return collection.Scope{}, collection.ErrForbidden
	}
	return a.scope, nil
}
func (a *transferAuth) AuthorizePlan(context.Context, bool) (sourcing.AcquisitionPlanScope, error) {
	if a.denied {
		return sourcing.AcquisitionPlanScope{}, sourcing.ErrPublicationForbidden
	}
	return sourcing.AcquisitionPlanScope{OrganizationID: a.scope.OrganizationID, ActorID: a.scope.ActorID, MemberID: a.scope.MemberID}, nil
}

type unusedTransferSources struct {
	sourcing.PublishedAcquisitionReader
}
type unusedTransferChild struct {
	productsourcing.PlanAcquisition
}

func TestPlanTransfersActualOriginalBatchesWithDriftAndRevocation(t *testing.T) {
	db := planTransferDB(t)
	ctx := context.Background()
	require.NoError(t, catalogstore.AutoMigrate(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	require.NoError(t, prepstore.InstallSchema(db))
	require.NoError(t, planstore.InstallSchema(db))
	repo, e := planstore.NewRepository(ctx, db)
	require.NoError(t, e)
	collectionRepo, e := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) { return nil, collection.ErrUnavailable })
	require.NoError(t, e)
	authority := &transferAuth{scope: collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}}
	collections, e := collection.NewService(collectionRepo, authority, unusedTransferSources{})
	require.NoError(t, e)
	snapshots, e := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, e)
	collections.WithSnapshots(snapshots)
	prepRepo, e := prepstore.NewRepository(ctx, db)
	require.NoError(t, e)
	preps, e := preparation.NewService(prepRepo, collections, authority)
	require.NoError(t, e)
	service, e := productsourcing.NewAcquisitionPlanService(repo, repo, unusedTransferChild{}, collections, preps, authority, nil)
	require.NoError(t, e)
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: "actor-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Minute)})
	input, e := sourcing.NewAcquisitionPlanInput("product", []string{"123", "456"}, sourcing.AcquisitionRule{}, true)
	require.NoError(t, e)
	p := sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: sourcing.AcquisitionPlanScope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}, Input: input, Revision: 1, DiscoveryIndex: 2, State: "pending", CreatedAt: time.Now().UTC(), Items: []sourcing.AcquisitionPlanItem{}, DiscoveryFailures: []sourcing.DiscoveryFailure{}}
	for _, source := range input.Inputs {
		batchID, itemID, opID := uuid.NewString(), uuid.NewString(), uuid.NewString()
		require.NoError(t, db.Exec("INSERT INTO product_collection_batches(organization_id,actor_id,member_id,id,name,kind,revision,created_at) VALUES('org-a','actor-a','member-a',?,'采集原批次','acquisition',1,now())", batchID).Error)
		for n := 0; n < 2; n++ {
			id := itemID
			if n == 1 {
				id = uuid.NewString()
			}
			product, publication := uuid.NewString(), uuid.NewString()
			raw := []byte(`{"title":"真实持久化测试商品"}`)
			hash := sha256.Sum256(raw)
			require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: "org-a", ProductKey: product, Version: 1, PublicationID: publication, PayloadHash: hex.EncodeToString(hash[:]), SnapshotJSON: raw}).Error)
			require.NoError(t, db.Exec("INSERT INTO product_collection_items(organization_id,actor_id,member_id,id,batch_id,product_key,publication_id,original_version,source_kind,source_operation_id,revision,created_at) VALUES('org-a','actor-a','member-a',?,?,?,?,1,'acquisition',?,1,now())", id, batchID, product, publication, opID).Error)
		}
		p.Items = append(p.Items, sourcing.AcquisitionPlanItem{Source: source, ChildKey: uuid.NewString(), State: "published", OperationID: opID, Evaluation: &sourcing.RuleEvaluation{State: sourcing.RuleMatched}, CollectionItemID: itemID, CollectionBatchID: batchID, CollectionRevision: 1})
	}
	p, e = repo.Create(ctx, p, func(context.Context) error { return nil })
	require.NoError(t, e)
	p, e = service.Advance(ctx, p.ID, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "transfer_pending", p.State)
	frozen := append([]byte{}, p.TransferInputJSON...)
	p, e = service.Advance(ctx, p.ID, p.Revision)
	require.NoError(t, e)
	require.Len(t, p.PreparationIDs, 1)
	require.NoError(t, db.Exec("UPDATE product_collection_batches SET revision=2 WHERE id=?", p.Items[1].CollectionBatchID).Error)
	p, e = service.Advance(ctx, p.ID, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "transfer_pending", p.State)
	require.Len(t, p.PreparationIDs, 1)
	authority.denied = true
	_, e = service.Advance(ctx, p.ID, p.Revision)
	require.ErrorIs(t, e, sourcing.ErrPublicationForbidden)
	authority.denied = false
	require.NoError(t, db.Exec("UPDATE product_collection_batches SET revision=1 WHERE id=?", p.Items[1].CollectionBatchID).Error)
	p, e = service.Advance(ctx, p.ID, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "completed", p.State)
	require.Len(t, p.PreparationIDs, 2)
	require.Equal(t, frozen, p.TransferInputJSON)
	for i, id := range p.PreparationIDs {
		actual, e := preps.Read(ctx, id)
		require.NoError(t, e)
		require.Equal(t, int64(1), actual.Count)
		members, e := preps.ListSources(ctx, id, collection.Query{Limit: 100})
		require.NoError(t, e)
		require.Len(t, members.Items, 1)
		require.Equal(t, p.Items[i].CollectionItemID, members.Items[0].CollectionItemID)
	}
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM product_collection_items WHERE organization_id='org-a'").Scan(&count).Error)
	require.Equal(t, int64(4), count)
}

func planTransferDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ISSUE398_TEST_DSN")
	if dsn == "" {
		t.Skip("SKIP: task-owned PostgreSQL is required")
	}
	require.Contains(t, dsn, "host=127.0.0.1")
	require.Contains(t, dsn, "user=issue398_owner")
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	name := "acqplan_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	db, e := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() {
		pool, _ := db.DB()
		_ = pool.Close()
		require.NoError(t, root.Exec("DROP DATABASE "+name+" WITH (FORCE)").Error)
		rp, _ := root.DB()
		_ = rp.Close()
	})
	return db
}
