package research

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"task-processor/internal/product/collection"
)

type pageEvidenceRepository struct {
	Repository
	rows []Candidate
}

func (r pageEvidenceRepository) List(context.Context, collection.Scope, string, string) ([]Candidate, string, error) {
	return r.rows, "", nil
}

type pageEvidenceSources struct {
	*analysisSources
	started chan Selection
	release chan struct{}
}

func (*pageEvidenceSources) CanReadPageEvidenceConcurrently(collection.Scope, string, Selection) bool {
	return true
}

type testPageEvidenceReader struct {
	ctx     context.Context
	sources *pageEvidenceSources
	scope   collection.Scope
	project string
}

func (s *pageEvidenceSources) BindPageEvidence(ctx context.Context, scope collection.Scope, project string) (PageEvidenceReader, error) {
	return testPageEvidenceReader{ctx: ctx, sources: s, scope: scope, project: project}, nil
}

func (r testPageEvidenceReader) Observe(selection Selection) (Observation, error) {
	return r.sources.Observe(r.ctx, r.scope, r.project, selection)
}

func (testPageEvidenceReader) Reauthorize(context.Context) error { return nil }

func (s *pageEvidenceSources) Observe(ctx context.Context, scope collection.Scope, project string, selection Selection) (Observation, error) {
	s.started <- selection
	select {
	case <-s.release:
		return s.analysisSources.Observe(ctx, scope, project, selection)
	case <-ctx.Done():
		return Observation{}, ctx.Err()
	}
}

func TestResearchPageConcurrentEvidenceStillChecksExactCurrentAndHistoricalReferences(t *testing.T) {
	service, scope, choices, repo, _, originals := analysisReadFixture()
	first, second := repo.rows[choices[0].CandidateID], repo.rows[choices[1].CandidateID]
	first.Reason = "current reason"
	otherVersion := first.Links[0].Ref
	otherVersion.Checksum = "different original checksum"
	first.History = []Decision{{Ref: otherVersion, Reason: "hidden historical reason", EvidenceDigest: otherVersion.Digest()}}
	second.Reason = "hidden current reason"
	originals.deny = choices[1].Selection.ItemID
	sources := &pageEvidenceSources{analysisSources: originals, started: make(chan Selection, 2), release: make(chan struct{})}
	service.Sources = sources
	service.Store = pageEvidenceRepository{rows: []Candidate{first, second}}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	type result struct {
		page Page
		err  error
	}
	done := make(chan result, 1)
	go func() {
		page, err := service.List(ctx, scope, analysisProjectID, "")
		done <- result{page: page, err: err}
	}()
	started := make([]Selection, 0, 2)
	for range 2 {
		select {
		case selection := <-sources.started:
			started = append(started, selection)
		case <-ctx.Done():
			t.Fatal("the two admitted originals did not start independently")
		}
	}
	require.ElementsMatch(t, []Selection{choices[0].Selection, choices[1].Selection}, started)
	close(sources.release)
	read := <-done
	require.NoError(t, read.err)
	require.False(t, read.page.Incomplete)
	require.Len(t, read.page.Items, 2)
	require.Equal(t, first.Reason, read.page.Items[0].Reason)
	require.Empty(t, read.page.Items[0].History[0].Reason)
	require.Empty(t, read.page.Items[0].History[0].EvidenceDigest)
	require.Nil(t, read.page.Items[1].Observation)
	require.Empty(t, read.page.Items[1].Reason)
}
