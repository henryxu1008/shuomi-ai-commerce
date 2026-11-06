package dataacquisition

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
	"testing"
)

type provenanceRepo struct {
	Repository
	job  Job
	item Item
}

func (r *provenanceRepo) ReadCaptured(context.Context, collection.Scope, string) (Job, Item, error) {
	return r.job, r.item, nil
}

type provenanceVerifier struct{ verify func() error }

func (v provenanceVerifier) Verify(context.Context, collection.Scope, collection.Source, Evidence) error {
	if v.verify != nil {
		return v.verify()
	}
	return nil
}

func TestProvenanceBindsOriginalSourceAndRechecksReadAuthority(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "owner", MemberID: "original"}
	source := collection.Source{ProductKey: "amazon-product", PublicationID: uuid.NewString(), Version: 3, Kind: "amazon_data"}
	source.OperationID = source.PublicationID
	e := Evidence{Site: "us", ASIN: "B000123456", Title: "controlled evidence", MainImage: "https://m.media-amazon.com/images/I/fixture.jpg", Availability: "available", Price: 10, Currency: "USD", CapturedAt: "2026-11-02T01:00:00Z", ParserVersion: "amazon-v1", Missing: []string{"rating"}}
	require.NoError(t, e.Validate())
	r := &provenanceRepo{job: Job{ID: uuid.NewString(), Scope: scope, Query: Query{Site: "us", Fields: []string{"title"}}}, item: Item{ID: source.PublicationID, ASIN: e.ASIN, State: "SAVED", Source: &source, Evidence: &e}}
	live := &revokedAccess{}
	svc := &Service{repo: r, live: live}
	out, err := svc.Provenance(context.Background(), Principal{Scope: scope}, source, provenanceVerifier{})
	require.NoError(t, err)
	require.Equal(t, "amazon-v1", out.Evidence.ParserVersion)
	require.Equal(t, source, out.Source)
	require.Equal(t, e.CapturedAt, out.Evidence.CapturedAt)
	require.NotEmpty(t, out.Checksum)
	for _, mutate := range []func(){func() { r.job.Scope.ActorID = "other" }, func() { r.job.Scope.MemberID = "rebuilt" }, func() { r.item.State = "PREPARED_EVIDENCE" }, func() { r.item.Source = &collection.Source{} }, func() { r.item.Evidence = nil }} {
		r.job.Scope = scope
		r.item = Item{ID: source.PublicationID, ASIN: e.ASIN, State: "SAVED", Source: &source, Evidence: &e}
		mutate()
		out, err = svc.Provenance(context.Background(), Principal{Scope: scope}, source, provenanceVerifier{})
		require.Error(t, err)
		require.Empty(t, out.JobID)
	}
	r.job.Scope = scope
	r.item = Item{ID: source.PublicationID, ASIN: e.ASIN, State: "SAVED", Source: &source, Evidence: &e}
	_, err = svc.Provenance(context.Background(), Principal{Scope: scope}, source, provenanceVerifier{verify: func() error { live.denied = true; return nil }})
	require.ErrorIs(t, err, ErrForbidden)
}
