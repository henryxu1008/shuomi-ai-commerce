package dataservicesapp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"task-processor/internal/authidentity"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/sourcing"
)

type sheinUSFixture struct {
	Job struct {
		JobID string
		Scope collection.Scope
		dataacquisition.SHEINUSQuery
	}
	Item        struct{ ItemID string }
	Publication struct {
		Source       collection.Source
		RawReference struct{ Checksum string }
	}
	ObservationUTF8 string
}

type sheinUSLiveFixture struct {
	authorizedFixture
	reads  int
	denyAt int
}

func (f *sheinUSLiveFixture) CheckRead(context.Context, dataacquisition.Principal) error {
	f.reads++
	if f.denyAt > 0 && f.reads >= f.denyAt {
		return dataacquisition.ErrForbidden
	}
	return nil
}

type sheinUSCollectionAuth struct{ scope collection.Scope }

func (a sheinUSCollectionAuth) Authorize(context.Context, string) (collection.Scope, error) {
	return a.scope, nil
}

type unusedSHEINUSAcquisition struct {
	sourcing.PublishedAcquisitionReader
}

func TestSHEINUSPublicationAndExactReadback(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db := publicationTestDB(t, ctx)
	raw, err := os.ReadFile("../../../docs/architecture/evidence/shein-us-data-v1-p1-p2.json")
	require.NoError(t, err)
	var fixture struct{ Publications map[string]sheinUSFixture }
	require.NoError(t, json.Unmarshal(raw, &fixture))

	live := &sheinUSLiveFixture{}
	publisher, err := NewProductPublisher(live)
	require.NoError(t, err)
	repo, err := jobstore.NewRepository(ctx, db, live, publisher, transactionResultReader)
	require.NoError(t, err)
	sources, err := sourcestore.NewRepository(db, newCatalogBridge)
	require.NoError(t, err)
	m := &Module{live: live, repo: repo, results: capturedResultReader{sources}}
	p := dataacquisition.Principal{Scope: fixture.Publications["P1"].Job.Scope}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{
		TenantID: p.Scope.OrganizationID, EffectiveOrganizationID: p.Scope.OrganizationID,
		UserID: p.Scope.ActorID, EffectiveMemberID: p.Scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute),
	})
	cr, err := collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) {
		return nil, collection.ErrUnavailable
	})
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	selections, err := collection.NewService(cr, sheinUSCollectionAuth{p.Scope}, unusedSHEINUSAcquisition{})
	require.NoError(t, err)
	selections.WithSnapshots(snapshots)
	sheinPublisher := NewSHEINUSPublisher(live)

	refs := make(map[string]dataacquisition.SHEINUSPreparedRef)
	for _, name := range []string{"P1", "P2"} {
		f := fixture.Publications[name]
		refs[name] = seedSHEINUSPrepared(t, db, f)
		if name == "P1" {
			_, err := repo.PublishSHEINUS(ctx, p, refs[name], func(ctx context.Context, tx *gorm.DB, scope collection.Scope, funding orgresource.ResourceFunding, command sourcing.PublicationCommand, verify bool) (collection.Source, error) {
				_, err := sheinPublisher(ctx, tx, scope, funding, command, verify)
				require.NoError(t, err)
				return collection.Source{}, dataacquisition.ErrUnavailable
			})
			require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
			_, err = sources.Read(ctx, p.Scope.OrganizationID, f.Item.ItemID)
			require.ErrorIs(t, err, sourcing.ErrSourcePublicationNotFound)
		}
		source, err := repo.PublishSHEINUS(ctx, p, refs[name], sheinPublisher)
		require.NoError(t, err, name)
		require.Equal(t, f.Publication.Source, source)
		var count int64
		require.NoError(t, db.Raw("SELECT count(*) FROM product_collection_items WHERE organization_id=? AND actor_id=? AND publication_id=? AND original_version=? AND source_kind=?", p.Scope.OrganizationID, p.Scope.ActorID, source.PublicationID, source.Version, source.Kind).Scan(&count).Error)
		require.EqualValues(t, 1, count)
	}

	p1, p2 := fixture.Publications["P1"], fixture.Publications["P2"]
	for _, f := range []sheinUSFixture{p1, p2} {
		batch, err := collection.NewPublicationBatch(p.Scope, f.Job.JobID, collection.SourceSHEINUSData, "SHEIN · US")
		require.NoError(t, err)
		items, err := cr.ListItems(ctx, p.Scope, batch.ID, collection.Query{Limit: 10})
		require.NoError(t, err)
		require.Len(t, items.Items, 1)
		item := items.Items[0]
		in := collection.SelectionInput{
			ItemID: item.ID, ExpectedRevision: item.Revision,
			OriginalPublicationID: item.Source.PublicationID, OriginalVersion: item.Source.Version,
		}
		proof, err := selections.Select(ctx, in)
		require.NoError(t, err)
		detail, err := proof.Read(ctx)
		require.NoError(t, err)
		selectedScope, err := proof.Scope(ctx)
		require.NoError(t, err)
		observed, err := m.ReadSHEINUSProvenance(ctx, dataacquisition.Principal{Scope: selectedScope}, detail.Item.Source)
		require.NoError(t, err)
		require.Equal(t, f.Publication.Source, observed.Source)
		require.Equal(t, f.Job.JobID, observed.JobID)
		require.Equal(t, f.Job.QueryRef, observed.QueryRef)
		require.Equal(t, f.Job.SHEINUSQuery.Query, observed.QuerySummary)
		require.Equal(t, f.Publication.RawReference.Checksum, observed.Checksum)
		require.Equal(t, "568824474", observed.Observation.NativeGoodsID)
		in.OriginalVersion++
		_, err = selections.Select(ctx, in)
		require.ErrorIs(t, err, collection.ErrConflict)
	}
	replayed, err := repo.PublishSHEINUS(ctx, p, refs["P1"], sheinPublisher)
	require.NoError(t, err)
	require.Equal(t, p1.Publication.Source, replayed)
	conflicting := refs["P1"]
	conflicting.Checksum = refs["P2"].Checksum
	_, err = repo.PublishSHEINUS(ctx, p, conflicting, sheinPublisher)
	require.ErrorIs(t, err, dataacquisition.ErrConflict)
	conflicting = refs["P1"]
	conflicting.ReservationID = refs["P2"].ReservationID
	_, err = repo.PublishSHEINUS(ctx, p, conflicting, sheinPublisher)
	require.ErrorIs(t, err, dataacquisition.ErrConflict)

	stale := p1.Publication.Source
	stale.Version = p2.Publication.Source.Version
	observed, err := m.ReadSHEINUSProvenance(ctx, p, stale)
	require.ErrorIs(t, err, dataacquisition.ErrConflict)
	require.Empty(t, observed)

	other := p
	other.Scope.MemberID = "another-member"
	observed, err = m.ReadSHEINUSProvenance(ctx, other, p1.Publication.Source)
	require.ErrorIs(t, err, dataacquisition.ErrNotFound)
	require.Empty(t, observed)

	live.denyAt = live.reads + 1
	observed, err = m.ReadSHEINUSProvenance(ctx, p, p1.Publication.Source)
	require.ErrorIs(t, err, dataacquisition.ErrForbidden)
	require.Empty(t, observed)
	live.denyAt = live.reads + 2
	observed, err = m.ReadSHEINUSProvenance(ctx, p, p1.Publication.Source)
	require.ErrorIs(t, err, dataacquisition.ErrForbidden)
	require.Empty(t, observed)
	live.denyAt = 0

	_, err = repo.Read(ctx, p.Scope, p1.Job.JobID)
	require.ErrorIs(t, err, dataacquisition.ErrNotFound)
	jobs, err := repo.List(ctx, p.Scope, 20)
	require.NoError(t, err)
	require.Empty(t, jobs)
	_, err = repo.ChargeIntent(ctx, orgresource.ConsumerChargeIdentity{OrganizationID: p.Scope.OrganizationID, Consumer: orgresource.ConsumerAmazonData, OperationID: p1.Item.ItemID})
	require.ErrorIs(t, err, dataacquisition.ErrNotFound)
	usage, err := repo.Usage(ctx, p.Scope)
	require.NoError(t, err)
	require.Zero(t, usage.DayRows)
	require.Zero(t, usage.MonthPendingFen)

	require.NoError(t, db.Exec("UPDATE data_acquisition_items SET evidence_json=? WHERE id=?", []byte(strings.Replace(p1.ObservationUTF8, "38.88", "39.88", 1)), p1.Item.ItemID).Error)
	observed, err = m.ReadSHEINUSProvenance(ctx, p, p1.Publication.Source)
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
	require.Empty(t, observed)
	observed, err = m.ReadSHEINUSProvenance(ctx, p, p2.Publication.Source)
	require.NoError(t, err)
	require.Equal(t, "36.50", observed.Observation.Price.Amount)
}

func seedSHEINUSPrepared(t *testing.T, db *gorm.DB, f sheinUSFixture) dataacquisition.SHEINUSPreparedRef {
	t.Helper()
	q, err := json.Marshal(f.Job.SHEINUSQuery)
	require.NoError(t, err)
	s := f.Job.Scope
	reservation := uuid.NewString()
	require.NoError(t, db.Exec(`INSERT INTO data_acquisition_jobs
		(organization_id,actor_id,member_id,id,source_kind,command_key,input_hash,query_json,funding,state,quota_reserved,day_window,month_window,created_at,deadline)
		VALUES (?,?,?,?,?,?,?,?,'member_allocated','RUNNING',0,CURRENT_DATE,CURRENT_DATE,now(),now()+interval '10 minutes')`,
		s.OrganizationID, s.ActorID, s.MemberID, f.Job.JobID, "shein_us_data", f.Job.JobID, collection.Digest(f.Job.SHEINUSQuery), string(q)).Error)
	require.NoError(t, db.Exec(`INSERT INTO data_acquisition_items
		(organization_id,actor_id,job_id,id,source_kind,native_goods_id,state,intent_json,evidence_json,reservation_id,charge_state,created_at)
		VALUES (?,?,?,?,?,'568824474','PREPARED_EVIDENCE','{}',?,?,'reserved',now())`,
		s.OrganizationID, s.ActorID, f.Job.JobID, f.Item.ItemID, "shein_us_data", []byte(f.ObservationUTF8), reservation).Error)
	return dataacquisition.SHEINUSPreparedRef{JobID: f.Job.JobID, ItemID: f.Item.ItemID, ReservationID: reservation, Checksum: f.Publication.RawReference.Checksum}
}
