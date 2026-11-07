package research

import (
	"context"
	"github.com/google/uuid"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
)

const analysisProjectID = "11111111-1111-4111-8111-111111111111"

type analysisRepository struct {
	Repository
	rows map[string]Candidate
}

func (r *analysisRepository) Load(_ context.Context, s collection.Scope, project, id string) (Candidate, error) {
	if s != (collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}) {
		return Candidate{}, ErrForbidden
	}
	v, ok := r.rows[id]
	if !ok || v.ProjectID != project {
		return Candidate{}, ErrNotFound
	}
	return v, nil
}

func (r *analysisRepository) ReadAnalysisEvidence(ctx context.Context, scope collection.Scope, project, candidate, link string) (AnalysisEvidence, error) {
	row, err := r.Load(ctx, scope, project, candidate)
	if err != nil {
		return AnalysisEvidence{}, err
	}
	out := AnalysisEvidence{CandidateRevision: row.Revision, SelectedLinkID: row.SelectedLinkID}
	for _, evidence := range row.Links {
		if evidence.ID == link && !evidence.Removed {
			out.Ref = &evidence.Ref
			break
		}
	}
	return out, nil
}

type analysisProjects struct{ archived, denied bool }

func (p *analysisProjects) Authorize(context.Context, collection.Scope, string, bool) error {
	if p.denied {
		return ErrForbidden
	}
	return nil
}
func (p *analysisProjects) AuthorizeAnalysis(_ context.Context, _ collection.Scope, _ string, active bool) error {
	if p.denied {
		return ErrForbidden
	}
	if active && p.archived {
		return ErrArchived
	}
	return nil
}

type analysisSources struct {
	rows map[string]Observation
	deny string
}

func (s *analysisSources) Observe(_ context.Context, _ collection.Scope, _ string, in Selection) (Observation, error) {
	if in.ItemID == s.deny {
		return Observation{}, ErrForbidden
	}
	return s.rows[in.ItemID], nil
}
func analysisReadFixture() (*Service, collection.Scope, []AnalysisChoice, *analysisRepository, *analysisProjects, *analysisSources) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	repo := &analysisRepository{rows: map[string]Candidate{}}
	projects := &analysisProjects{}
	sources := &analysisSources{rows: map[string]Observation{}}
	choices := []AnalysisChoice{}
	for i := 0; i < 2; i++ {
		id := uuid.NewString()
		sel := Selection{ItemID: id, Revision: 1, PublicationID: uuid.NewString(), Version: 1}
		ref := EvidenceRef{Selection: sel, Source: collection.Source{Kind: "amazon_data", ProductKey: "product-" + id, PublicationID: sel.PublicationID, OperationID: sel.PublicationID, Version: 1}, JobID: uuid.NewString(), Checksum: "checksum"}
		sources.rows[id] = Observation{Ref: ref, Site: "us", ASIN: "B000123456", Title: id}
		link := uuid.NewString()
		candidate := Candidate{ID: uuid.NewString(), ProjectID: analysisProjectID, SelectedLinkID: link, Revision: 2, Links: []Link{{ID: link, Ref: ref}}}
		repo.rows[candidate.ID] = candidate
		choices = append(choices, AnalysisChoice{CandidateID: candidate.ID, ExpectedRevision: 2, LinkID: candidate.SelectedLinkID, EvidenceDigest: ref.Digest(), Selection: sel})
	}
	return New(repo, projects, sources), scope, choices, repo, projects, sources
}
func TestAnalysisReadChecksEveryExactChoiceAndNeverDisclosesPartialEvidence(t *testing.T) {
	s, scope, choices, _, projects, sources := analysisReadFixture()
	rows, err := s.InspectSelection(context.Background(), scope, analysisProjectID, choices)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	changed := append([]AnalysisChoice(nil), choices...)
	changed[1].ExpectedRevision++
	rows, err = s.InspectSelection(context.Background(), scope, analysisProjectID, changed)
	require.ErrorIs(t, err, ErrConflict)
	require.Empty(t, rows)
	sources.deny = choices[1].Selection.ItemID
	rows, err = s.InspectSelection(context.Background(), scope, analysisProjectID, choices)
	require.ErrorIs(t, err, ErrForbidden)
	require.Empty(t, rows)
	sources.deny = ""
	projects.archived = true
	_, err = s.InspectSelection(context.Background(), scope, analysisProjectID, choices)
	require.ErrorIs(t, err, ErrArchived)
}
func TestAnalysisHistorySurvivesManualRevisionButMasksRemovedOrRevokedEvidence(t *testing.T) {
	s, scope, choices, repo, projects, sources := analysisReadFixture()
	c := repo.rows[choices[0].CandidateID]
	c.Revision = 3
	c.State = "APPROVED_FOR_TEST"
	repo.rows[c.ID] = c
	require.NoError(t, s.AuthorizeAnalysisHistory(context.Background(), scope, analysisProjectID, choices))
	projects.archived = true
	require.NoError(t, s.AuthorizeAnalysisHistory(context.Background(), scope, analysisProjectID, choices))
	c.Links[0].Removed = true
	repo.rows[c.ID] = c
	require.ErrorIs(t, s.AuthorizeAnalysisHistory(context.Background(), scope, analysisProjectID, choices), ErrNotFound)
	c.Links[0].Removed = false
	repo.rows[c.ID] = c
	sources.deny = choices[1].Selection.ItemID
	require.ErrorIs(t, s.AuthorizeAnalysisHistory(context.Background(), scope, analysisProjectID, choices), ErrForbidden)
}
