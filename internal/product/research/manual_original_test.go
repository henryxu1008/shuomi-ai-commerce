package research

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/collection"
	"testing"
)

// The only new source capability is a server-bound manual original, never a client flag.
type manualAnalysisSources struct {
	*analysisSources
	scope   collection.Scope
	project string
	choice  AnalysisChoice
}

func (s *manualAnalysisSources) ManualOriginal(scope collection.Scope, project string) (string, string, Selection, bool) {
	return s.choice.CandidateID, s.choice.LinkID, s.choice.Selection, scope == s.scope && project == s.project
}
func TestManualOriginalNeverEntersAnalysisIncludingMixedSelection(t *testing.T) {
	s, scope, choices, _, _, sources := analysisReadFixture()
	s.Sources = &manualAnalysisSources{sources, scope, analysisProjectID, choices[0]}
	_, err := s.InspectSelection(context.Background(), scope, analysisProjectID, choices)
	require.ErrorIs(t, err, ErrUnsupported)
	require.ErrorIs(t, s.AuthorizeAnalysisHistory(context.Background(), scope, analysisProjectID, choices), ErrUnsupported)
	// The capability remains enabled while an entirely normal pair still works.
	_, _, normal, otherRepo, _, otherSources := analysisReadFixture()
	for id, row := range otherRepo.rows {
		s.Store.(*analysisRepository).rows[id] = row
	}
	for id, row := range otherSources.rows {
		sources.rows[id] = row
	}
	rows, err := s.InspectSelection(context.Background(), scope, analysisProjectID, normal)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	// An unbound source keeps the normal current contract.
	s.Sources = sources
	rows, err = s.InspectSelection(context.Background(), scope, analysisProjectID, choices)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}

type manualCommandRepository struct {
	*analysisRepository
	commits int
	replay  *Receipt
}

func (r *manualCommandRepository) Replay(context.Context, collection.Scope, string, string, Command) (Receipt, bool, error) {
	if r.replay != nil {
		return *r.replay, true, nil
	}
	return Receipt{}, false, nil
}
func (r *manualCommandRepository) Commit(_ context.Context, _ collection.Scope, _ string, key string, _ Command, _ []Observation, validation error) (Receipt, error) {
	r.commits++
	return Receipt{Key: key}, validation
}
func TestManualOriginalRestrictsNewCommandsWithoutBreakingOriginalReceipt(t *testing.T) {
	s, scope, choices, repo, _, sources := analysisReadFixture()
	choice := choices[0]
	s.Sources = &manualAnalysisSources{sources, scope, analysisProjectID, choice}
	r := &manualCommandRepository{analysisRepository: repo}
	s.Store = r
	for _, action := range []string{"reset", "select", "unlink", "decide"} {
		t.Run(action, func(t *testing.T) {
			command := Command{Action: action, CandidateID: choice.CandidateID, ExpectedRevision: choice.ExpectedRevision}
			if action == "select" || action == "unlink" {
				command.LinkID = choice.LinkID
			}
			if action == "reset" || action == "decide" {
				command.EvidenceDigest = choice.EvidenceDigest
			}
			if action == "decide" {
				command.State = "APPROVED_FOR_TEST"
				command.Reason = "synthetic acceptance"
			}
			_, err := s.Execute(context.Background(), scope, analysisProjectID, uuid.NewString(), command)
			require.ErrorIs(t, err, ErrUnsupported)
		})
	}
	require.Zero(t, r.commits)
	key := uuid.NewString()
	r.replay = &Receipt{Key: key, Replayed: true}
	receipt, err := s.Execute(context.Background(), scope, analysisProjectID, key, Command{Action: "reset", CandidateID: choice.CandidateID, ExpectedRevision: 2, EvidenceDigest: choice.EvidenceDigest})
	require.NoError(t, err)
	require.Equal(t, key, receipt.Key)
	require.True(t, receipt.Replayed)
}

func TestManualOriginalCannotBeRelinkedOrAliasedButNormalCommandsRemain(t *testing.T) {
	s, scope, choices, repo, _, sources := analysisReadFixture()
	choice := choices[0]
	s.Sources = &manualAnalysisSources{sources, scope, analysisProjectID, choice}
	r := &manualCommandRepository{analysisRepository: repo}
	s.Store = r
	_, err := s.Execute(context.Background(), scope, analysisProjectID, uuid.NewString(), Command{Action: "link", Selections: []Selection{choice.Selection}})
	require.ErrorIs(t, err, ErrUnsupported)
	for _, state := range []string{"", "REJECTED"} {
		command := Command{Action: "review", CandidateID: choice.CandidateID, ExpectedRevision: 2, EvidenceDigest: choice.EvidenceDigest}
		if state != "" {
			command.Action = "decide"
			command.State = state
			command.Reason = "synthetic manual decision"
		}
		_, err = s.Execute(context.Background(), scope, analysisProjectID, uuid.NewString(), command)
		require.NoError(t, err)
	}
	// Binding an equivalent source to another candidate does not grant it access.
	other := repo.rows[choices[1].CandidateID]
	other.SelectedLinkID = choice.LinkID
	other.Links = repo.rows[choice.CandidateID].Links
	repo.rows[other.ID] = other
	_, err = s.Execute(context.Background(), scope, analysisProjectID, uuid.NewString(), Command{Action: "review", CandidateID: other.ID, ExpectedRevision: 2, EvidenceDigest: choice.EvidenceDigest})
	require.ErrorIs(t, err, ErrUnsupported)
	require.False(t, s.manualEvidenceAllowed(scope, analysisProjectID, other.ID, choice.LinkID, choice.Selection))
	changed := choice.Selection
	changed.Version++
	require.False(t, s.manualEvidenceAllowed(scope, analysisProjectID, choice.CandidateID, choice.LinkID, changed))
	require.Equal(t, 2, r.commits)
}
