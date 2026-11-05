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

	pool, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, jobstore.InstallSHEINUSTrialSchema(ctx, pool))
	refs := make(map[string]dataacquisition.SHEINUSPreparedRef)
	approvals := make(map[string]dataacquisition.SHEINUSTrialApproval)
	for _, name := range []string{"P1", "P2"} {
		f := fixture.Publications[name]
		approvals[name], refs[name] = prepareSHEINUSFixture(t, ctx, db, repo, &f)
		fixture.Publications[name] = f
		if name == "P1" {
			_, err := repo.PublishSHEINUS(ctx, p, approvals[name], refs[name], func(ctx context.Context, tx *gorm.DB, scope collection.Scope, funding orgresource.ResourceFunding, command sourcing.PublicationCommand, verify bool) (collection.Source, error) {
				_, err := sheinPublisher(ctx, tx, scope, funding, command, verify)
				require.NoError(t, err)
				return collection.Source{}, dataacquisition.ErrUnavailable
			})
			require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
			_, err = sources.Read(ctx, p.Scope.OrganizationID, f.Item.ItemID)
			require.ErrorIs(t, err, sourcing.ErrSourcePublicationNotFound)
			tampered := strings.Replace(f.ObservationUTF8, "38.88", "39.88", 1)
			require.NoError(t, db.Exec("UPDATE data_acquisition_items SET evidence_json=? WHERE id=?", []byte(tampered), f.Item.ItemID).Error)
			_, err = repo.PublishSHEINUS(ctx, p, approvals[name], refs[name], sheinPublisher)
			require.ErrorIs(t, err, dataacquisition.ErrConflict)
			require.NoError(t, db.Exec("UPDATE data_acquisition_items SET evidence_json=? WHERE id=?", []byte(f.ObservationUTF8), f.Item.ItemID).Error)
		}
		source, err := repo.PublishSHEINUS(ctx, p, approvals[name], refs[name], sheinPublisher)
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
		require.Equal(t, "after coupon", *observed.Observation.Price.PromotionNote)
		in.OriginalVersion++
		_, err = selections.Select(ctx, in)
		require.ErrorIs(t, err, collection.ErrConflict)
	}
	replayed, err := repo.PublishSHEINUS(ctx, p, approvals["P1"], refs["P1"], sheinPublisher)
	require.NoError(t, err)
	require.Equal(t, p1.Publication.Source, replayed)
	conflicting := refs["P1"]
	conflicting.Checksum = refs["P2"].Checksum
	_, err = repo.PublishSHEINUS(ctx, p, approvals["P1"], conflicting, sheinPublisher)
	require.ErrorIs(t, err, dataacquisition.ErrConflict)
	wrongApproval := approvals["P1"]
	wrongApproval.KeyRef = approvals["P2"].KeyRef
	_, err = repo.PublishSHEINUS(ctx, p, wrongApproval, refs["P1"], sheinPublisher)
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

func prepareSHEINUSFixture(t *testing.T, ctx context.Context, db *gorm.DB, repo *jobstore.Repository, f *sheinUSFixture) (dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSPreparedRef) {
	t.Helper()
	var o dataacquisition.SHEINUSObservation
	require.NoError(t, json.Unmarshal([]byte(f.ObservationUTF8), &o))
	a := dataacquisition.SHEINUSTrialApproval{
		ID: f.Job.JobID, ProjectID: "00000000-0000-4000-8000-000000000501",
		Scope: f.Job.Scope, AccountRef: uuid.NewString(), KeyRef: uuid.NewString(), Funding: orgresource.FundingMember,
		ExpiresAt: time.Now().Add(time.Minute),
	}
	budget := dataacquisition.SHEINUSTrialBudget{
		Binding: a.Binding(), AvailableCredits: 25, MaximumRequestCredits: 25, ZeroIncrementalCash: true, ExclusiveKeyWindow: true,
	}
	pool, err := db.DB()
	require.NoError(t, err)
	svc, err := NewSHEINUSTrial(pool, trialAuthority{approval: a, budget: budget},
		func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			credits := 5
			return dataacquisition.SHEINUSTrialSubmission{
				SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding(), RequestID: a.ID, Credits: &credits},
				Cards:             []dataacquisition.SHEINUSTrialCard{{Position: 1, GoodsID: o.NativeGoodsID, Title: *o.Title, URL: o.SourceURL, Amount: o.Price.Amount, Currency: o.Price.Currency, PromotionNote: "after coupon"}},
			}, nil
		},
		func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest, dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
			return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrNotFound
		})
	require.NoError(t, err)
	p := dataacquisition.Principal{Scope: a.Scope}
	attempt, err := svc.Send(ctx, p, a.ID)
	require.NoError(t, err)
	require.Equal(t, 5, *attempt.Credits)
	approved, err := svc.PublicationApproval(ctx, p, a.ID)
	require.NoError(t, err)
	refs, err := repo.SHEINUSTrialPreparedRefs(ctx, approved)
	require.NoError(t, err)
	require.Len(t, refs, 1)

	f.Job.SHEINUSQuery = a.PageQuery()
	f.Item.ItemID = collection.StableID("shein-us-trial-item", a.ID, o.NativeGoodsID)
	f.Publication.Source.PublicationID = f.Item.ItemID
	f.Publication.Source.OperationID = f.Item.ItemID
	o.QueryRef = f.Job.QueryRef
	o.ParserVersion = "zenrows-category-page-v1"
	note := "after coupon"
	o.Price.PromotionNote = &note
	o.Missing = []string{"capturedAt", "variantId", "price.taxIncluded", "price.unit"}
	expected, err := json.Marshal(o)
	require.NoError(t, err)
	f.ObservationUTF8 = string(expected)
	f.Publication.RawReference.Checksum = sourcing.RawSnapshotChecksum(f.ObservationUTF8)
	require.Equal(t, dataacquisition.SHEINUSPreparedRef{JobID: a.ID, ItemID: f.Item.ItemID, Checksum: f.Publication.RawReference.Checksum}, refs[0])
	return approved, refs[0]
}
