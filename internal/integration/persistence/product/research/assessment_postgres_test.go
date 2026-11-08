package researchpersistence

import (
	"context"
	"encoding/json"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

// All writes belong to this disposable container; no application or retained database is used.
func TestAssessmentTransactionRecoveryReferenceAndPrivacy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	pg, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("assessment1205"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("isolated-assessment1205"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(dsn string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	owner := open(dsn)
	require.NoError(t, InstallSchema(owner))
	require.NoError(t, owner.Exec(`CREATE ROLE product_research_runtime LOGIN PASSWORD 'isolated-assessment-role'; REVOKE CREATE ON SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, GrantRuntime(owner))
	require.NoError(t, InstallSupplierReviewSchema(owner))
	require.NoError(t, InstallAssessmentSchema(owner))
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword(RuntimeRole, "isolated-assessment-role")
	db := open(u.String())
	require.NoError(t, VerifyRuntimePermissions(ctx, db))
	require.NoError(t, VerifyAssessmentSchema(ctx, db))
	repo, err := New(ctx, db)
	require.NoError(t, err)
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	project := &projectFixture{scope: scope, id: uuid.NewString()}
	sel := research.Selection{ItemID: uuid.NewString(), Revision: 1, PublicationID: uuid.NewString(), Version: 1}
	ref := research.EvidenceRef{Selection: sel, Source: collection.Source{ProductKey: "amazon-B000123456", PublicationID: sel.PublicationID, OperationID: sel.PublicationID, Kind: "amazon_data", Version: 1}, JobID: uuid.NewString(), Checksum: collection.Digest(sel)}
	obs := research.Observation{Ref: ref, Site: "us", ASIN: "B000123456", Title: "synthetic assessment item"}
	sources := &sourceFixture{values: map[string]research.Observation{sel.ItemID: obs}}
	original := &supplierSourceFixture{amazon: research.SupplierMarket{Ref: ref, Revision: 1, ASIN: obs.ASIN, Title: obs.Title}}
	service := func() *research.Service {
		s := research.New(repo, project, sources)
		s.SupplierSources = original
		s.AssessmentExpiry = func(context.Context) time.Time { return time.Now().Add(5 * time.Second) }
		return s
	}
	svc := service()
	linked, err := svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "link", Selections: []research.Selection{sel}})
	require.NoError(t, err)
	id := linked.CandidateIDs[0]
	candidate, err := repo.Load(ctx, scope, project.id, id)
	require.NoError(t, err)
	cmd := research.Command{Action: research.SaveAssessment, CandidateID: id, ExpectedRevision: candidate.Revision, LinkID: candidate.SelectedLinkID, EvidenceDigest: ref.Digest(), Assessment: &research.AssessmentInput{Cost: research.CostInput{Rule: "fixed_percent", PurchaseCurrency: "USD"}}}
	preview, err := svc.PreviewAssessment(ctx, scope, project.id, cmd)
	require.NoError(t, err)
	require.Equal(t, "INSUFFICIENT_INPUT", preview.Result.Status)
	key := uuid.NewString()
	var receipts [2]research.Receipt
	var failures [2]error
	var wg sync.WaitGroup
	for i := range receipts {
		wg.Go(func() { receipts[i], failures[i] = svc.Execute(ctx, scope, project.id, key, cmd) })
	}
	wg.Wait()
	require.NoError(t, failures[0])
	require.NoError(t, failures[1])
	require.NotEqual(t, receipts[0].Replayed, receipts[1].Replayed)
	require.Equal(t, receipts[0].AssessmentRef, receipts[1].AssessmentRef)
	require.NotNil(t, receipts[0].AssessmentRef)
	var count int64
	require.NoError(t, owner.Table(assessmentTable).Count(&count).Error)
	require.EqualValues(t, 1, count)
	saved, err := svc.ReadAssessment(ctx, scope, project.id, id, key)
	require.NoError(t, err)
	require.Equal(t, "CURRENT", saved.Applicability)
	require.NotNil(t, saved.Detail)
	require.Equal(t, "INSUFFICIENT_INPUT", saved.Detail.Result.Status)
	replay, err := service().Lookup(ctx, scope, project.id, key)
	require.NoError(t, err)
	require.Equal(t, receipts[0].AssessmentRef, replay.AssessmentRef, "response loss is reconciled by the original key")
	changed := cmd
	payload := *cmd.Assessment
	payload.Cost.Price = "30"
	changed.Assessment = &payload
	_, err = svc.Execute(ctx, scope, project.id, key, changed)
	require.ErrorIs(t, err, research.ErrConflict)
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), cmd)
	require.ErrorIs(t, err, research.ErrConflict, "different keys still use candidate CAS")
	next := receipts[0].Revisions[id]
	reviewed, err := svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "review", CandidateID: id, ExpectedRevision: next, EvidenceDigest: ref.Digest()})
	require.NoError(t, err)
	decision := research.Command{Action: "decide", CandidateID: id, ExpectedRevision: reviewed.Revisions[id], EvidenceDigest: ref.Digest(), State: "APPROVED_FOR_TEST", Reason: "saved missing conditions explicitly considered", AssessmentRef: receipts[0].AssessmentRef}
	bad := *decision.AssessmentRef
	bad.InputHash = collection.Digest("other input")
	invalid := decision
	invalid.AssessmentRef = &bad
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), invalid)
	require.ErrorIs(t, err, research.ErrConflict)
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), decision)
	require.NoError(t, err, "technical revision increments must not invalidate the business context")
	current, err := repo.Load(ctx, scope, project.id, id)
	require.NoError(t, err)
	require.Equal(t, receipts[0].AssessmentRef, current.History[0].AssessmentRef)
	project.archived = true
	saved, err = svc.ReadAssessment(ctx, scope, project.id, id, key)
	require.NoError(t, err)
	require.NotNil(t, saved.Detail)
	_, err = svc.PreviewAssessment(ctx, scope, project.id, cmd)
	require.ErrorIs(t, err, research.ErrArchived)
	project.archived = false
	// Revoked originals produce a placeholder, never protected saved amounts or reasons.
	original.marketDenied = true
	saved, err = svc.ReadAssessment(ctx, scope, project.id, id, key)
	require.NoError(t, err)
	require.Nil(t, saved.Detail)
	require.Equal(t, "DETAIL_UNAVAILABLE", saved.Applicability)
	page, err := svc.List(ctx, scope, project.id, "")
	require.NoError(t, err)
	raw, err := json.Marshal(page)
	require.NoError(t, err)
	require.NotContains(t, string(raw), decision.Reason)
	require.NotContains(t, string(raw), "assessmentRef")
	original.marketDenied = false
	// The final gate must roll back both writes even if checks had passed earlier.
	current, err = repo.Load(ctx, scope, project.id, id)
	require.NoError(t, err)
	cmd.ExpectedRevision = current.Revision
	proof, err := svc.PrepareAssessment(ctx, scope, project.id, cmd)
	require.NoError(t, err)
	original.marketDenied = true
	failedKey := uuid.NewString()
	_, err = repo.CommitAssessment(ctx, scope, project.id, failedKey, cmd, proof, nil)
	require.ErrorIs(t, err, research.ErrForbidden)
	_, err = repo.Lookup(ctx, scope, project.id, failedKey)
	require.ErrorIs(t, err, research.ErrNotFound)
	require.NoError(t, owner.Table(assessmentTable).Count(&count).Error)
	require.EqualValues(t, 1, count)
	original.marketDenied = false
	differentMember := scope
	differentMember.MemberID = "other-member"
	project.scope = differentMember
	_, err = svc.ReadAssessment(ctx, differentMember, project.id, id, key)
	require.ErrorIs(t, err, research.ErrNotFound)
	_, err = svc.Lookup(ctx, differentMember, project.id, key)
	require.ErrorIs(t, err, research.ErrNotFound)
	project.scope = scope
	for _, field := range []string{"org", "actor", "project"} {
		otherScope, otherProject := scope, project.id
		if field == "org" {
			otherScope.OrganizationID = "other-org"
		}
		if field == "actor" {
			otherScope.ActorID = "other-actor"
		}
		if field == "project" {
			otherProject = uuid.NewString()
		}
		_, e := repo.LoadAssessment(ctx, otherScope, otherProject, id, key)
		require.ErrorIs(t, e, research.ErrNotFound)
	}
	// A failed record insert must not consume the candidate CAS or commit a receipt.
	require.NoError(t, owner.Exec("ALTER TABLE research_assessments ADD CONSTRAINT test_reject_new_assessment CHECK (candidate_revision < 5) NOT VALID").Error)
	current, err = repo.Load(ctx, scope, project.id, id)
	require.NoError(t, err)
	cmd.ExpectedRevision = current.Revision
	rejected := uuid.NewString()
	_, err = svc.Execute(ctx, scope, project.id, rejected, cmd)
	require.ErrorIs(t, err, research.ErrUnavailable)
	unchanged, err := repo.Load(ctx, scope, project.id, id)
	require.NoError(t, err)
	require.Equal(t, current.Revision, unchanged.Revision)
	_, err = repo.Lookup(ctx, scope, project.id, rejected)
	require.ErrorIs(t, err, research.ErrNotFound)
	require.NoError(t, owner.Exec("ALTER TABLE research_assessments DROP CONSTRAINT test_reject_new_assessment").Error)
	// Explicit new conditions create a new record, never rewrite the old decision.
	for i := 0; i < 6; i++ {
		current, err = repo.Load(ctx, scope, project.id, id)
		require.NoError(t, err)
		cmd.ExpectedRevision = current.Revision
		payload := *cmd.Assessment
		payload.Cost.Price = "31"
		cmd.Assessment = &payload
		_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), cmd)
		require.NoError(t, err)
	}
	firstPage, err := svc.Assessments(ctx, scope, project.id, id, 0)
	require.NoError(t, err)
	require.Len(t, firstPage.Items, 5)
	require.Positive(t, firstPage.NextBefore)
	secondPage, err := svc.Assessments(ctx, scope, project.id, id, firstPage.NextBefore)
	require.NoError(t, err)
	require.Len(t, secondPage.Items, 2)
	require.Zero(t, secondPage.NextBefore)
	old, err := svc.ReadAssessment(ctx, scope, project.id, id, key)
	require.NoError(t, err)
	require.Equal(t, "", old.Detail.Input.Price)
	current, err = repo.Load(ctx, scope, project.id, id)
	require.NoError(t, err)
	require.Equal(t, decision.AssessmentRef, current.History[0].AssessmentRef)
	_, err = svc.Execute(ctx, scope, project.id, key, research.Command{Action: "reset", CandidateID: id, ExpectedRevision: current.Revision, EvidenceDigest: ref.Digest()})
	require.ErrorIs(t, err, research.ErrConflict)
	// Unlinking masks the original saved detail, without deleting its record.
	_, err = svc.Execute(ctx, scope, project.id, uuid.NewString(), research.Command{Action: "unlink", CandidateID: id, ExpectedRevision: current.Revision, LinkID: current.SelectedLinkID})
	require.NoError(t, err)
	old, err = svc.ReadAssessment(ctx, scope, project.id, id, key)
	require.NoError(t, err)
	require.Nil(t, old.Detail)
	for _, sql := range []string{"UPDATE research_assessments SET record_json=record_json", "DELETE FROM research_assessments", "UPDATE research_commands SET input_hash=input_hash"} {
		require.Error(t, db.Exec(sql).Error)
	}
	require.NoError(t, VerifyRuntimePermissions(ctx, db))
}
