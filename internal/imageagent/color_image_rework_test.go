package imageagent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/store"
	productasset "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type colorReworkAssets struct {
	inventory productasset.ImageSetInventory
	commit    productasset.ApprovalCommit
}

func (f *colorReworkAssets) ReadImageSetInventory(_ context.Context, scope productasset.InventoryScope) (productasset.ImageSetInventory, error) {
	if scope != f.inventory.Scope {
		return productasset.ImageSetInventory{}, productasset.ErrApprovedAssetsNotReady
	}
	return f.inventory, nil
}

func (f *colorReworkAssets) ReadApprovalCommit(_ context.Context, tenant, action string) (productasset.ApprovalCommit, error) {
	if tenant != f.commit.TenantID || action != f.commit.ActionID {
		return productasset.ApprovalCommit{}, productasset.ErrApprovedAssetsNotReady
	}
	return f.commit, nil
}

func TestColorImageReworkPreparesConfirmsAndRetainsTheOriginalTwelve(t *testing.T) {
	for _, backend := range []string{"memory", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			var wrap func(imageagent.Repository) imageagent.Repository
			if backend == "postgres" {
				repo := colorReworkPostgres(t)
				wrap = func(imageagent.Repository) imageagent.Repository { return repo }
			}
			original, repo, _, config, contexts, quotes, ctx, input := imageSetServiceFixtureWithRepository(t, wrap)
			input = configureOwnedColorSet(t, config, contexts, quotes, input)
			config.template.Detail = append(config.template.Detail, agentconfig.ContentTask{ID: "structure", Purpose: "structure_functions"})
			initial, err := original.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			parent, err := original.ConfirmImagePlan(ctx, confirmSetInput(initial))
			require.NoError(t, err)
			parent, baseline := completeColorReworkParent(t, ctx, repo, parent)
			require.Len(t, baseline.commit.Assets, 12)
			target := parent.Plan.Slots[len(parent.Plan.Slots)-1]
			require.Equal(t, "White", target.Recipe.Placement.Color)
			require.Equal(t, 2, target.Recipe.Placement.Order)
			svc := func(configuration *setConfigurationFixture, quote *setQuoteFixture, workflows *recordingWorkflowClient) *imageagent.Service {
				s, err := imageagent.NewService(repo, workflows, staticCatalogResolver{catalog: parent.AssetCatalog},
					imageagent.WithOrganizationScope(), imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{
						Configuration: configuration, Contexts: contexts, Quotes: quote,
						ColorReworkInventory: baseline, ColorReworkApprovals: baseline,
						HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 32, Points: 10000, ElapsedSeconds: 3600},
					}))
				require.NoError(t, err)
				return s
			}
			prepare := input
			prepare.RequestID, prepare.RegenerateFromRunID, prepare.EffectiveCatalogVersion = uuid.NewString(), parent.Run.ID, 1
			prepare.ColorImages = nil
			prepare.ColorRework = &imageagent.ColorImageReworkRequest{SlotID: target.ID, Head: baseline.inventory.Head}
			preparer := svc(config, quotes, &recordingWorkflowClient{})
			blank, err := preparer.PrepareImageSet(ctx, prepare)
			require.NoError(t, err)
			require.Equal(t, target.Recipe.Prompt, blank.Projection.Plan.Slots[0].Recipe.Prompt)
			require.NoError(t, preparer.Cancel(ctx, blank.Projection.Run.ID, 1, uuid.NewString()))
			const note = "Show the white coat without a model."
			prepare.RequestID = uuid.NewString()
			prepare.ColorRework.Composition = " " + note + " "
			first, err := preparer.PrepareImageSet(ctx, prepare)
			require.NoError(t, err)
			require.Equal(t, 1, first.Images)
			require.Zero(t, first.Points)
			slot := first.Projection.Plan.Slots[0]
			require.Equal(t, target.ID, slot.ID)
			require.Equal(t, target.Recipe.Placement, slot.Recipe.Placement)
			require.Equal(t, target.Recipe.References, slot.Recipe.References)
			require.Equal(t, note, slot.Recipe.Composition)
			require.Contains(t, slot.Recipe.Prompt, note)
			require.NotEqual(t, blank.QuoteDigest, first.QuoteDigest)
			prepare.ColorRework.Composition = note
			replay, err := preparer.PrepareImageSet(ctx, prepare)
			require.NoError(t, err)
			require.Equal(t, first.PlanDigest, replay.PlanDigest)
			prepare.ColorRework.Composition = "different"
			_, err = preparer.PrepareImageSet(ctx, prepare)
			require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
			prepare.RequestID = uuid.NewString()
			competing, err := preparer.PrepareImageSet(ctx, prepare)
			require.NoError(t, err)

			type result struct {
				projection imageagent.RunProjection
				err        error
			}
			start := make(chan struct{})
			finished := make(chan result, 2)
			runs := []imageagent.PreparedImageSet{first, competing}
			services := make([]*imageagent.Service, len(runs))
			configurations := make([]*setConfigurationFixture, len(runs))
			clients := make([]*recordingWorkflowClient, len(runs))
			for i := range runs {
				configurations[i] = &setConfigurationFixture{enabled: true}
				clients[i] = &recordingWorkflowClient{}
				services[i] = svc(configurations[i], &setQuoteFixture{quote: quotes.quote}, clients[i])
				go func() {
					<-start
					p, err := services[i].ConfirmImagePlan(ctx, confirmSetInput(runs[i]))
					finished <- result{p, err}
				}()
			}
			close(start)
			a, b := <-finished, <-finished
			if a.err != nil {
				a, b = b, a
			}
			require.NoError(t, a.err)
			require.ErrorIs(t, b.err, imageagent.ErrColorImageReworkConfirmed)
			require.Equal(t, a.projection.Run.ID, b.projection.Run.ID)
			wins := 0
			for i, prepared := range runs {
				require.Equal(t, 1, configurations[i].admissions)
				persisted, err := repo.GetProjection(ctx, imageagent.ScopeForRun(prepared.Projection.Run))
				require.NoError(t, err)
				if persisted.Run.ID == a.projection.Run.ID {
					require.NotNil(t, persisted.Run.ImageAdmission)
					require.Len(t, clients[i].starts, 1)
					_, err = services[i].ConfirmImagePlan(ctx, confirmSetInput(prepared))
					require.NoError(t, err)
					require.Equal(t, 1, configurations[i].admissions)
					wins++
				} else {
					require.Nil(t, persisted.Run.ImageAdmission)
					require.Empty(t, clients[i].starts)
				}
			}
			require.Equal(t, 1, wins)
			unchanged, err := repo.GetProjection(ctx, imageagent.ScopeForRun(parent.Run))
			require.NoError(t, err)
			require.Equal(t, parent.ResultDigest, unchanged.ResultDigest)
			require.Equal(t, parent.Slots, unchanged.Slots)
			require.Equal(t, baseline.commit.ActionID, baseline.inventory.Head.ActionID)
		})
	}
}

func completeColorReworkParent(t *testing.T, ctx context.Context, repo imageagent.Repository, parent imageagent.RunProjection) (imageagent.RunProjection, *colorReworkAssets) {
	t.Helper()
	parent.Run.Version++
	parent.Run.Status, parent.Run.CurrentNode = imageagent.RunStatusCompleted, "complete"
	proofHash := strings.Repeat("a", 64)
	scope := imageagent.ScopeForRun(parent.Run)
	ownerKey, err := imageagent.ArtifactOwnerKey(scope.OwnerUserID)
	require.NoError(t, err)
	mutations := make([]imageagent.SlotProjectionMutation, 0, len(parent.Slots))
	assets := make([]productasset.ApprovedAsset, 0, len(parent.Slots))
	for i := range parent.Slots {
		slot := &parent.Slots[i]
		slot.Slot.Status, slot.Attempt = imageagent.SlotStatusAccepted, 1
		slot.Closure = &imageagent.ImageSlotClosure{UnitVersion: agentconfig.ProviderImageUsageUnit, Kind: "provider_usage",
			IntentID: proofHash, Fingerprint: proofHash, ProviderUsageProofDigest: proofHash}
		id := slot.Slot.ID
		asset := imageagent.AssetCandidate{AssetID: id + "-generated", SourceAssetID: slot.Slot.SourceAssetIDs[0], Width: 1024, Height: 1024,
			DurableAsset: imageagent.DurableAssetIdentity{ObjectKey: fmt.Sprintf("%s/%s/%s/%s/1/%s/1/0-%s.png", imageagent.PublishedArtifactPrefix, scope.TenantID, ownerKey, scope.RunID, id, proofHash), SHA256: proofHash},
			Operations:   []string{"render_source_edit"}, GenerationProof: slot.Closure.CandidateProof()}
		slot.Candidates = []imageagent.AssetCandidate{asset}
		mutations = append(mutations, imageagent.SlotProjectionMutation{PlanRevision: 1, Projection: *slot,
			Result: imageagent.SlotResult{SlotID: id, Attempt: 1, Status: slot.Slot.Status, CandidateAssetIDs: []string{asset.AssetID}, Closure: slot.Closure},
			Attempt: imageagent.StepAttempt{TenantID: scope.TenantID, OwnerUserID: scope.OwnerUserID, RunID: scope.RunID,
				PlanRevision: 1, SlotID: id, Node: "execute_slot_v3", Attempt: 1, Outcome: "accepted", IdempotencyKey: id + "-attempt"}})
		p := slot.Slot.Recipe.Placement
		assets = append(assets, productasset.ApprovedAsset{ID: id + "-approved", RunID: scope.RunID, PlanRevision: 1, SlotID: id, Attempt: 1,
			SourceAssetID: asset.SourceAssetID, URL: "https://images.example.org/" + id + ".png", Width: 1024, Height: 1024,
			Role: productasset.RoleGallery, Presentation: &productasset.ImagePresentation{Color: p.Color, Group: p.Group, Order: p.Order},
			GenerationEvidence: &productasset.GenerationEvidence{IntentID: proofHash, Fingerprint: proofHash, ProviderUsageProofDigest: proofHash, ArtifactHash: proofHash}})
	}
	parent.ResultDigest, err = imageagent.ImageSetResultDigest(parent.Plan, parent.Slots, nil)
	require.NoError(t, err)
	parent, err = repo.CommitProjection(ctx, imageagent.ProjectionCommit{Scope: scope, CommitID: "fixture-closed", ExpectedProjectionVersion: parent.ProjectionVersion,
		ExpectedRunVersion: parent.Run.Version - 1, Snapshot: parent, SlotMutations: mutations,
		RunMutation: &imageagent.RunMutation{Status: parent.Run.Status, CurrentNode: parent.Run.CurrentNode, ActivePlanRevision: 1},
		EventType:   "run.completed", EventPayload: json.RawMessage(`{}`)})
	require.NoError(t, err)
	source := parent.Plan.Set.Source
	head := productasset.ImageInventoryHead{ActionID: uuid.NewString(), PayloadHash: proofHash}
	inventoryScope := productasset.InventoryScope{TenantID: scope.TenantID, ProductKey: source.ProductID, TargetPlatform: "product", SourceSnapshotVersion: source.EffectiveVersion}
	commit := productasset.ApprovalCommit{TenantID: scope.TenantID, ProductKey: source.ProductID, TargetPlatform: "product",
		SourceSnapshotVersion: source.EffectiveVersion, ActionID: head.ActionID, Assets: assets,
		ImageSet: &productasset.ImageSetSelection{Source: productasset.SourceSelectionRequest{ContextKind: string(source.ContextKind), ItemID: source.OperationID,
			OriginalPublicationID: source.OriginalPublicationID, OriginalSnapshotVersion: source.OriginalVersion, EffectiveCatalogVersion: source.EffectiveVersion, TargetPlatform: "product"},
			Results: []productasset.ImageSetResultBinding{{RunID: scope.RunID, PlanRevision: 1, ResultDigest: parent.ResultDigest}}}}
	return parent, &colorReworkAssets{inventory: productasset.ImageSetInventory{Scope: inventoryScope, Head: head, ApprovalActionID: head.ActionID, Assets: assets}, commit: commit}
}

func colorReworkPostgres(t *testing.T) imageagent.Repository {
	t.Helper()
	dsn := os.Getenv("ISSUE487_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL ISSUE487_TEST_DSN")
	}
	require.Contains(t, dsn, "host=127.0.0.1")
	require.Equal(t, "ISOLATED_TRIAL_ONLY", os.Getenv("ISSUE487_EXCLUSIVE_POSTGRES"))
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	schema := "image_rework_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE SCHEMA "+schema).Error)
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		pool, _ := db.DB()
		_ = pool.Close()
		require.NoError(t, root.Exec("DROP SCHEMA "+schema+" CASCADE").Error)
		rootPool, _ := root.DB()
		_ = rootPool.Close()
	})
	require.NoError(t, store.AutoMigrateOrganizationScope(db))
	return store.NewOrganizationRepository(db)
}
