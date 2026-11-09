package dataservicesapp

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
)

func TestManualOriginalSinglePublishedProofNoWritesAndFinalRevocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	db := publicationTestDB(t, ctx)
	scope := collection.Scope{OrganizationID: "manual-org", ActorID: "manual-actor", MemberID: "manual-member"}
	project := uuid.NewString()
	checks, revoked := 0, false
	authorize := func(_ context.Context, s collection.Scope, p string, write bool) error {
		checks++
		if s != scope || p != project || revoked && checks > 1 {
			return collection.ErrForbidden
		}
		return nil
	}
	f, err := NewResearchFixture(ctx, db, scope, project, authorize, false)
	require.NoError(t, err)
	// Seed only the one original in this disposable database. No other bundle
	// item is available, and the new reader cannot import or repair anything.
	batch, err := f.importBatch(ctx, scope, project, f.bundleID(), researchFixtureKeyword, f.sources()[1:2], researchFixturePublicationAuthority{f, true})
	require.NoError(t, err)
	entry := batch.Amazon[0]
	b := ResearchManualOriginalBinding{Scope: scope, Project: project, CandidateID: uuid.NewString(), LinkID: uuid.NewString(), ItemID: entry.Item.ID, ItemRevision: 1, PublicationID: entry.Receipt.PublicationID, CatalogVersion: entry.Receipt.CatalogVersion, Checksum: entry.Checksum, EnvelopeHash: entry.Receipt.EnvelopeHash, SnapshotHash: entry.Receipt.SnapshotHash}
	r, err := NewResearchManualOriginalReader(ctx, db, b, authorize)
	require.NoError(t, err)
	in := collection.SelectionInput{ItemID: b.ItemID, ExpectedRevision: 1, OriginalPublicationID: b.PublicationID, OriginalVersion: 1}
	got, err := r.Read(ctx, scope, project, in)
	require.NoError(t, err)
	require.Equal(t, entry, got)
	for name, change := range map[string]func(*collection.SelectionInput){"item": func(i *collection.SelectionInput) { i.ItemID = uuid.NewString() }, "revision": func(i *collection.SelectionInput) { i.ExpectedRevision++ }, "publication": func(i *collection.SelectionInput) { i.OriginalPublicationID = uuid.NewString() }, "version": func(i *collection.SelectionInput) { i.OriginalVersion++ }} {
		t.Run(name, func(t *testing.T) {
			bad := in
			change(&bad)
			_, err := r.Read(ctx, scope, project, bad)
			require.ErrorIs(t, err, collection.ErrConflict)
		})
	}
	for _, change := range []func(*collection.Scope){func(s *collection.Scope) { s.OrganizationID = "other" }, func(s *collection.Scope) { s.ActorID = "other" }, func(s *collection.Scope) { s.MemberID = "other" }} {
		bad := scope
		change(&bad)
		_, err = r.Read(ctx, bad, project, in)
		require.ErrorIs(t, err, collection.ErrForbidden)
	}
	_, err = r.Read(ctx, scope, uuid.NewString(), in)
	require.ErrorIs(t, err, collection.ErrForbidden)
	for _, field := range []string{"envelope", "snapshot"} {
		bad := b
		if field == "envelope" {
			bad.EnvelopeHash = collection.Digest("different")
		} else {
			bad.SnapshotHash = collection.Digest("different")
		}
		reader, e := NewResearchManualOriginalReader(ctx, db, bad, authorize)
		require.NoError(t, e)
		_, e = reader.Read(ctx, scope, project, in)
		require.ErrorIs(t, e, collection.ErrConflict)
	}
	bad := b
	bad.Checksum = "sha256:" + collection.Digest("different")
	require.Error(t, bad.Validate())
	checks = 0
	revoked = true
	_, err = r.Read(ctx, scope, project, in)
	require.ErrorIs(t, err, collection.ErrForbidden)
	revoked = false
	for table, expected := range map[string]int64{"product_source_publications": 1, "product_snapshot_versions": 1, "product_collection_items": 1, "data_acquisition_jobs": 0} {
		var n int64
		require.NoError(t, db.Table(table).Count(&n).Error)
		require.Equal(t, expected, n, table)
	}
	lateChecks := 0
	lateReader, e := NewResearchManualOriginalReader(ctx, db, b, func(context.Context, collection.Scope, string, bool) error {
		lateChecks++
		if lateChecks == 3 {
			return db.Exec("UPDATE product_collection_items SET archived_at=now(), revision=revision+1 WHERE id=?", b.ItemID).Error
		}
		return nil
	})
	require.NoError(t, e)
	_, err = lateReader.Read(ctx, scope, project, in)
	require.Error(t, err)
}
