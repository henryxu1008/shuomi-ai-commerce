package researchpersistence

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"sync"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"testing"
	"time"
)

type projectFixture struct {
	scope    collection.Scope
	id       string
	archived bool
}

func (p *projectFixture) Authorize(_ context.Context, s collection.Scope, id string, write bool) error {
	if s != p.scope || id != p.id {
		return research.ErrNotFound
	}
	if write && p.archived {
		return research.ErrArchived
	}
	return nil
}

type sourceFixture struct {
	failure    error
	values     map[string]research.Observation
	denied     bool
	deniedItem string
}

func (p *projectFixture) AuthorizeAnalysis(ctx context.Context, s collection.Scope, id string, active bool) error {
	return p.Authorize(ctx, s, id, active)
}

func (s *sourceFixture) Observe(_ context.Context, _ collection.Scope, in research.Selection) (research.Observation, error) {
	if s.failure != nil {
		return research.Observation{}, s.failure
	}
	if s.denied || s.deniedItem == in.ItemID {
		return research.Observation{}, research.ErrForbidden
	}
	v, ok := s.values[in.ItemID]
	if !ok || v.Ref.Selection != in {
		return research.Observation{}, research.ErrConflict
	}
	return v, nil
}

func TestResearchPersistsExactEvidenceDecisionsReplayAndRevocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("research1072"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("isolated-research1072"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, InstallSchema(db))
	require.NoError(t, db.Exec(`ALTER TABLE research_opportunities DROP CONSTRAINT research_opportunity_identity`).Error)
	require.ErrorIs(t, VerifySchema(ctx, db), research.ErrUnavailable, "missing scoped source identity must fail readiness")
	require.NoError(t, db.Exec(`ALTER TABLE research_opportunities ADD UNIQUE(organization_id,actor_id,project_id,platform,market,native_product_id)`).Error)
	require.NoError(t, db.Exec(`CREATE ROLE product_research_runtime LOGIN PASSWORD 'isolated-research-role'; CREATE TABLE foreign_owner(id integer); REVOKE CREATE ON SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, GrantRuntime(db))
	require.NoError(t, InstallSupplierReviewSchema(db))
	runtimeURL, e := url.Parse(dsn)
	require.NoError(t, e)
	runtimeURL.User = url.UserPassword(RuntimeRole, "isolated-research-role")
	runtimeDB, e := gorm.Open(postgres.Open(runtimeURL.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	runtimePool, e := runtimeDB.DB()
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, runtimePool.Close()) })
	require.NoError(t, VerifyRuntimePermissions(ctx, runtimeDB))
	require.Error(t, runtimeDB.Exec("SELECT * FROM foreign_owner").Error)
	require.Error(t, runtimeDB.Exec("UPDATE research_commands SET input_hash=input_hash").Error)
	require.Error(t, runtimeDB.Exec("DELETE FROM research_opportunities").Error)
	repo, err := New(ctx, runtimeDB)
	require.NoError(t, err)
	scope := collection.Scope{OrganizationID: "org", ActorID: "owner", MemberID: "member"}
	project := &projectFixture{scope: scope, id: uuid.NewString()}
	sources := &sourceFixture{values: map[string]research.Observation{}}
	makeEvidence := func(asin string) research.Selection {
		pub := uuid.NewString()
		selection := research.Selection{ItemID: uuid.NewString(), Revision: 1, PublicationID: pub, Version: 1}
		sources.values[selection.ItemID] = research.Observation{Ref: research.EvidenceRef{Selection: selection, Source: collection.Source{ProductKey: "amazon-" + asin, PublicationID: pub, OperationID: pub, Kind: "amazon_data", Version: 1}, JobID: uuid.NewString(), Checksum: collection.Digest(pub)}, Site: "us", ASIN: asin, Title: "controlled fixture", CapturedAt: "2026-11-02T00:00:00Z", ParserVersion: "amazon-v1", Missing: []string{"rating", "reviewCount"}}
		return selection
	}
	a, b := makeEvidence("B000123456"), makeEvidence("B000123457")
	svc := research.New(repo, project, sources)
	unsupportedKey := uuid.NewString()
	sources.failure = research.ErrUnsupported
	_, err = svc.Execute(ctx, scope, project.id, unsupportedKey, research.Command{Action: "link", Selections: []research.Selection{a}})
	require.ErrorIs(t, err, research.ErrUnsupported)
	_, err = svc.Lookup(ctx, scope, project.id, unsupportedKey)
	require.ErrorIs(t, err, research.ErrNotFound, "definite unsupported source rejection must not persist a command")
	sources.failure = nil

	key := uuid.NewString()
	link := research.Command{Action: "link", Selections: []research.Selection{a, b}}
	receipt, err := svc.Execute(ctx, scope, project.id, key, link)
	require.NoError(t, err)
	require.Len(t, receipt.CandidateIDs, 2)
	replay, err := svc.Execute(ctx, scope, project.id, key, link)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	_, err = svc.Execute(ctx, scope, project.id, key, research.Command{Action: "link", Selections: []research.Selection{a}})
	require.ErrorIs(t, err, research.ErrConflict)
	page, err := svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	current := page.Items[0]
	require.NotNil(t, current.Observation)
	t.Run("saved supplier review", func(t *testing.T) {
		verifySupplierReview(t, ctx, svc, scope, project.id, current)
	})
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	review := research.Command{Action: "review", CandidateID: current.ID, ExpectedRevision: current.Revision, EvidenceDigest: current.EvidenceDigest}
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), review)
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	approve := research.Command{Action: "decide", CandidateID: current.ID, ExpectedRevision: current.Revision, EvidenceDigest: current.EvidenceDigest, State: "APPROVED_FOR_TEST", Reason: "check original samples"}
	approved, err := svc.Execute(ctx, scope, project.id, uuid.NewString(), approve)
	require.NoError(t, err)
	approvedRevision := approved.Revisions[current.ID]
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), approve)
	require.ErrorIs(t, err, research.ErrConflict)
	// Conflicting concurrent decisions use CAS; exactly one consumes the revision.
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	concurrent := research.Command{Action: "reset", CandidateID: page.Items[1].ID, ExpectedRevision: page.Items[1].Revision, EvidenceDigest: page.Items[1].EvidenceDigest}
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, e := svc.Execute(ctx, scope, project.id, uuid.NewString(), concurrent)
			results <- e
		}()
	}
	wg.Wait()
	close(results)
	var success, conflict int
	for e := range results {
		if e == nil {
			success++
		} else if e == research.ErrConflict {
			conflict++
		} else {
			require.NoError(t, e)
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflict)

	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	require.Equal(t, "APPROVED_FOR_TEST", current.State)
	more := makeEvidence(current.Observation.ASIN)
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{more}})
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	require.Equal(t, "APPROVED_FOR_TEST", current.State, "new evidence cannot silently replace approved evidence")
	require.Equal(t, approve.Reason, current.Reason)
	originalItem := current.Observation.Ref.Selection.ItemID
	var next string
	for _, l := range current.Evidence {
		if l.Observation != nil && l.Observation.Ref.Selection == more {
			next = l.ID
		}
	}
	require.NotEmpty(t, next)
	newObservation := sources.values[more.ItemID]
	unlinkOtherKey := uuid.NewString()
	unlinked, err := svc.Execute(ctx, scope, project.id, unlinkOtherKey, research.Command{Action: "unlink", CandidateID: current.ID, ExpectedRevision: current.Revision, LinkID: next})
	require.NoError(t, err)
	unlinkedRevision := unlinked.Revisions[current.ID]
	unlinkRow, err := lookup(db.WithContext(ctx), scope, project.id, unlinkOtherKey)
	require.NoError(t, err)
	var unlinkHistory research.Decision
	require.NoError(t, json.Unmarshal(unlinkRow.DecisionJSON, &unlinkHistory))
	t.Run("unlink records no reason from a different observation", func(t *testing.T) {
		require.Equal(t, newObservation.Ref, unlinkHistory.Ref)
		require.Empty(t, unlinkHistory.Reason)
	})
	// Recreate the stored shape from before the fix to verify it is safe to read as-is.
	unlinkHistory.Reason = approve.Reason
	oldDecisionJSON, err := json.Marshal(unlinkHistory)
	require.NoError(t, err)
	update := db.WithContext(ctx).Exec("UPDATE research_commands SET decision_json=? WHERE organization_id=? AND actor_id=? AND project_id=? AND key=?", string(oldDecisionJSON), scope.OrganizationID, scope.ActorID, project.id, unlinkOtherKey)
	require.NoError(t, update.Error)
	require.EqualValues(t, 1, update.RowsAffected)
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{more}})
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	require.Equal(t, approve.Reason, current.Reason)
	for _, name := range []string{"original access revoked", "original checksum changed"} {
		t.Run(name, func(t *testing.T) {
			original := sources.values[originalItem]
			if name == "original access revoked" {
				sources.deniedItem = originalItem
			} else {
				changed := original
				changed.Ref.Checksum = "different-original-checksum"
				sources.values[originalItem] = changed
			}
			defer func() {
				sources.deniedItem = ""
				sources.values[originalItem] = original
			}()
			masked, err := svc.List(ctx, scope, project.id, "")
			require.NoError(t, err)
			view := masked.Items[0]
			require.Equal(t, current.ID, view.ID)
			require.Equal(t, current.SelectedLinkID, view.SelectedLinkID)
			require.Equal(t, current.State, view.State)
			require.Nil(t, view.Observation)
			require.Empty(t, view.Reason)
			require.Contains(t, view.Evidence, research.EvidenceView{ID: current.SelectedLinkID})
			require.Contains(t, view.Evidence, research.EvidenceView{ID: next, Observation: &newObservation})
			require.NotEmpty(t, view.History)
			foundUnlink := false
			for _, h := range view.History {
				require.Empty(t, h.Reason)
				if h.Revision == unlinkedRevision {
					foundUnlink = true
					require.Equal(t, newObservation.Ref.Digest(), h.EvidenceDigest)
				} else {
					require.Empty(t, h.EvidenceDigest)
				}
			}
			require.True(t, foundUnlink)
			stored, err := repo.Load(ctx, scope, project.id, current.ID)
			require.NoError(t, err)
			require.Equal(t, approve.Reason, stored.Reason)
		})
	}
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "select", CandidateID: current.ID, ExpectedRevision: current.Revision, LinkID: next})
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	require.Equal(t, "TO_RESEARCH", current.State)
	found := false
	for _, h := range current.History {
		if h.Revision == approvedRevision {
			found = true
			require.Equal(t, approve.Reason, h.Reason)
			require.NotEmpty(t, h.EvidenceDigest)
		}
	}
	require.True(t, found)
	started, err := svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "review", CandidateID: current.ID, ExpectedRevision: current.Revision, EvidenceDigest: current.EvidenceDigest})
	require.NoError(t, err)
	const newerReason = "reviewed the newer samples"
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "decide", CandidateID: current.ID, ExpectedRevision: started.Revisions[current.ID], EvidenceDigest: current.EvidenceDigest, State: "REJECTED", Reason: newerReason})
	require.NoError(t, err)
	sources.deniedItem = originalItem
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	current = page.Items[0]
	require.Equal(t, newerReason, current.Reason)
	require.NotNil(t, current.Observation)
	require.Equal(t, more, current.Observation.Ref.Selection)
	var oldDecisionMasked, newDecisionVisible bool
	for _, h := range current.History {
		switch {
		case h.Revision == approvedRevision:
			require.Empty(t, h.Reason)
			require.Empty(t, h.EvidenceDigest)
			oldDecisionMasked = true
		case h.State == "REJECTED":
			require.Equal(t, newerReason, h.Reason)
			require.Equal(t, current.EvidenceDigest, h.EvidenceDigest)
			newDecisionVisible = true
		}
	}
	require.True(t, oldDecisionMasked)
	require.True(t, newDecisionVisible)
	sources.deniedItem = ""
	// Unlink removes only this relationship. A new explicit link can restore it.
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "unlink", CandidateID: current.ID, ExpectedRevision: current.Revision, LinkID: current.SelectedLinkID})
	require.NoError(t, err)
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{more}})
	require.NoError(t, err)
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	require.NotNil(t, page.Items[0].Observation, "explicit relink must restore removed relationship")
	sources.denied = true
	page, err = svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	for _, v := range page.Items {
		require.Nil(t, v.Observation)
		require.Empty(t, v.EvidenceDigest)
		for _, h := range v.History {
			require.Empty(t, h.Reason)
			require.Empty(t, h.EvidenceDigest)
		}
	}
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), review)
	require.Error(t, err)
	again := research.New(repo, project, sources)
	stored, err := again.Lookup(ctx, scope, project.id, key)
	require.NoError(t, err)
	require.Equal(t, receipt.CandidateIDs, stored.CandidateIDs)
	foreign := scope
	foreign.ActorID = "other"
	_, err = svc.List(ctx, foreign, project.id, "")
	require.Error(t, err)
	project.archived = true
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), link)
	require.ErrorIs(t, err, research.ErrArchived)
}
