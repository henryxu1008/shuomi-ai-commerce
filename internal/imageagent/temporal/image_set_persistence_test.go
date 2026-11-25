package temporal

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/assetpublication"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/store"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/storagecapacity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func imageSetPersistenceFixture(t *testing.T) (*Activities, imageagent.Repository, ExecuteSlotV3ActivityInput) {
	t.Helper()
	return imageSetPersistenceFixtureWithSource(t, nil, false, imageagent.GenerationProviderMetadata{})
}

func imageSetPersistenceFixtureWithSource(t *testing.T, ownedSource []byte, includeDetail bool, metadata imageagent.GenerationProviderMetadata) (*Activities, imageagent.Repository, ExecuteSlotV3ActivityInput) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "image.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, store.AutoMigrateOrganizationScope(db))
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	repo := store.NewOrganizationRepository(db)
	hash := strings.Repeat("a", 64)
	run := imageagent.Run{ID: "392ed0a2-0f01-4c94-9aa4-eb50271fae9c", TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "source-operation", ScopeProtocol: imageagent.OrganizationScopeProtocol, TargetPlatform: "product", Mode: imageagent.RunModeManual, IdempotencyKey: "prepare", Status: imageagent.RunStatusPlanning, CurrentNode: "plan", Version: 1, ActivePlanRevision: 1, MaxConcurrentSlots: 1}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{ProductContext: imageagent.ProductContextRef{ProductID: "product"}, Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://images.example.org/source.png"}}})
	require.NoError(t, err)
	slot := imageagent.Slot{ID: "overview", Role: imageagent.SlotRoleDetail, SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "overview", Status: imageagent.SlotStatusPending, Recipe: &imageagent.ImageSlotRecipe{Purpose: "product_overview", Background: "white", Language: "zh", Placement: imageagent.ImagePlacement{Group: "detail", Order: 1}, PromptVersion: imageagent.ImageSetSchema, Prompt: "Create one finished commercial product photograph from the reference photos. Preserve the actual product, shape, color, material and proportions. Product grounding (never render this as text): . Product type: . Visual task: product_overview; direction: . Background: white. Target text language: zh. One large product, generous whitespace. No table, grid, document, information panel or leader-line labels. No visible text: no letters, numbers, headings, labels or symbols. These text and composition rules take priority over the visual direction and method guidance. Product facts and instructions are not visible copy. Output only the finished image.", References: []imageagent.ImageSourceObservation{{AssetID: "source-1", SHA256: hash, Bytes: 100, Width: 1024, Height: 1024, MediaType: "image/png"}}, Quote: imageagent.ImageGenerationQuote{Billing: syntheticImageBilling("gpt-image-2.5", "price", 12), Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price", Points: 12, RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "config"}}}
	plan := imageagent.Plan{Revision: 1, IdempotencyKey: "plan", SourceAssetIDs: []string{"source-1"}, CreatedBy: "actor", Slots: []imageagent.Slot{slot}, Set: &imageagent.ImageSetPlan{Schema: imageagent.ImageSetSchema, Text: &imageagent.ImageTextBinding{PolicyVersion: imageagent.ImageTextPolicyVersion, Mode: "none", Language: "zh"}, Source: imageagent.ImageSourceBinding{ContextKind: imageagent.ImageSourceAcquisition, ProductID: "product", OperationID: run.BusinessTaskID, OriginalPublicationID: "publication", OriginalVersion: 1, EffectiveVersion: 1, CatalogHash: catalog.Manifest.Hash}, Target: imageagent.ImageTarget{Platform: "product"}, Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: "9e7afaa9-a9f9-48ba-a11a-b5bb377f08e9", Digest: hash}, ConfigurationEpoch: "1", ParametersDigest: hash, InputDigest: hash, MaxPoints: 12}}
	limits := agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 1, Points: 12, ElapsedSeconds: 3600}
	if ownedSource != nil {
		slot.Role = imageagent.SlotRoleMain
		slot.Recipe.Prompt = strings.Replace(slot.Recipe.Prompt, "product_overview", "product_identity", 1)
		slot.Recipe.Purpose = "product_identity"
		slot.Recipe.Placement = imageagent.ImagePlacement{Group: "carousel", Order: 1}
		sum := sha256.Sum256(ownedSource)
		slot.Recipe.References[0].SHA256 = hex.EncodeToString(sum[:])
		slot.Recipe.References[0].Bytes = int64(len(ownedSource))
		slot.Recipe.Quote = metadata.ImageQuote()
		plan.Slots[0] = slot
		if includeDetail {
			detail := slot
			detail.ID, detail.IdempotencyKey, detail.Role = "detail", "detail", imageagent.SlotRoleDetail
			detail.Recipe = imageagent.CloneImageSlotRecipe(slot.Recipe)
			detail.Recipe.Purpose = "product_overview"
			detail.Recipe.Prompt = strings.Replace(detail.Recipe.Prompt, "product_identity", detail.Recipe.Purpose, 1)
			detail.Recipe.Placement = imageagent.ImagePlacement{Group: "detail", Order: 1}
			plan.Slots = append(plan.Slots, detail)
			limits.Images = 2
		}
		plan.Set.Source.ContextKind = imageagent.ImageSourceListing
		plan.Set.Source.OwnProduct = true
		plan.Set.Route = &imageagent.ImageRouteSelection{BindingID: slot.Recipe.Quote.RouteReference, RevisionID: slot.Recipe.Quote.ConfigurationVersion}
		plan.Set.MaxPoints = 0
		limits.UnitVersion, limits.Points = agentconfig.ProviderImageUsageUnit, 0
	}
	plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(plan)
	require.NoError(t, err)
	run.Budget = imageagent.ImageSetBudget(limits)
	run.StartedAt = time.Now().UTC().Truncate(time.Microsecond)
	planDigest, err := imageagent.ImageSetPlanDigest(plan)
	require.NoError(t, err)
	receipt := agentconfig.ImageRunAdmissionReceipt{ID: "74e01be8-f4e6-461f-8acb-18fe37329f1c", AdmittedAt: run.StartedAt, Deadline: run.StartedAt.Add(time.Hour), Command: agentconfig.ImageRunAdmissionCommand{Scope: agent.Scope{OrganizationID: run.TenantID, ActorID: run.UserID}, Snapshot: plan.Set.Configuration, MemberID: run.MemberID, RunID: run.ID, ConfirmActionID: "1b912e40-d50a-48f5-a12b-62c73a42e4b8", SourceDigest: imageagent.ImageSetSourceDigest(plan.Set.Source, plan), InputDigest: plan.Set.InputDigest, PlanDigest: planDigest, QuoteDigest: plan.Set.QuoteDigest, Limits: limits}}
	receiptBytes, err := json.Marshal(receipt)
	require.NoError(t, err)
	receiptSum := sha256.Sum256(receiptBytes)
	receipt.Digest = hex.EncodeToString(receiptSum[:])
	run.ImageAdmission = &receipt
	projection, err := repo.InitializeRun(context.Background(), imageagent.ProjectionInitialization{Scope: imageagent.ScopeForRun(run), Run: run, Plan: plan, Catalog: catalog, Snapshot: imageagent.RunProjection{Run: run, Plan: plan}, CommitID: "prepare", EventType: "run.initialized", EventPayload: []byte(`{}`)})
	require.NoError(t, err)
	input := ExecuteSlotV3ActivityInput{RunID: run.ID, Identity: imageagent.ExecutionIdentity{RunID: run.ID, ScopeProtocol: run.ScopeProtocol, TenantID: run.TenantID, UserID: run.UserID, MemberID: run.MemberID, BusinessTaskID: run.BusinessTaskID}, TargetPlatform: "product", PlanRevision: 1, Slot: slot, Attempt: 1, ImageSet: plan.Set, AssetCatalog: projection.AssetCatalog, IdempotencyKey: slotAttemptKey(1, slot, 1)}
	activities := newV3Activities(t, repo, repo.(imageagent.SlotExternalEffectV3Repository), &recordingStagedExecutor{}, &recordingArtifactStore{})
	activities.executionAuthorizer = allowGenerationExecution{}
	return activities, repo, input
}

func ownedImageMetadata() imageagent.GenerationProviderMetadata {
	return imageagent.GenerationProviderMetadata{Provider: "grsai", Model: aicapability.GRSaiProductImageModel,
		Protocol: aicapability.GRSaiProductImageProtocol, RouteReference: strings.Repeat("b", 64), CredentialReference: "credential",
		ConfigurationVersion: strings.Repeat("c", 64), PromptVersion: imageagent.ImageSetSchema, BillingMode: aicapability.ModelUsageOnly,
		ConnectionID: "connection"}
}

type isolatedOwnedImageTasks struct {
	start  func(string, []byte) (string, error)
	lookup func(string) (imageagent.ProviderUsageImageResult, error)
}

type isolatedApprovalProjection imageagent.RunProjection

func (p isolatedApprovalProjection) GetProjection(_ context.Context, scope imageagent.RunScope) (imageagent.RunProjection, error) {
	if scope != imageagent.ScopeForRun(p.Run) {
		return imageagent.RunProjection{}, imageagent.ErrRunNotFound
	}
	return imageagent.RunProjection(p), nil
}

type isolatedOwnedImageExecutor struct {
	*imageagent.GenerationExecution
	staged *recordingStagedExecutor
}

type isolatedOwnedStorage struct {
	storagecapacity.Writer
	reservation storagecapacity.Reservation
	proof       storagecapacity.CloseProof
	failClose   bool
}

func (s *isolatedOwnedStorage) ReserveWrite(_ context.Context, intent storagecapacity.WriteIntent) (storagecapacity.Reservation, error) {
	s.reservation = storagecapacity.Reservation{ID: "isolated-write", Intent: intent, State: "RESERVED"}
	return s.reservation, nil
}

func (s *isolatedOwnedStorage) ClaimDispatch(context.Context, string, string) error {
	s.reservation.State = "DISPATCHED"
	return nil
}

func (s *isolatedOwnedStorage) FindWrite(context.Context, string, string, string) (storagecapacity.Reservation, error) {
	return s.reservation, nil
}

func (s *isolatedOwnedStorage) CloseWrite(_ context.Context, _, _ string, proof storagecapacity.CloseProof) error {
	if s.failClose {
		return storagecapacity.ErrUnavailable
	}
	s.proof = proof
	s.reservation.State = "CLOSED"
	return nil
}

func (e isolatedOwnedImageExecutor) GenerateSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	return imageagent.SlotGeneratedOutput{}, imageagent.ErrCommandBlocked
}

func (e isolatedOwnedImageExecutor) BuildSlotResult(ctx context.Context, input imageagent.SlotExecutionInput, output imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	return e.staged.BuildSlotResult(ctx, input, output)
}

func (p isolatedOwnedImageTasks) StartProductImage(_ context.Context, prompt string, source []byte) (string, error) {
	return p.start(prompt, source)
}

func (p isolatedOwnedImageTasks) LookupProductImage(_ context.Context, id string) (imageagent.ProviderUsageImageResult, error) {
	return p.lookup(id)
}

func TestOwnedGRSaiAuthorizedActivityKeepsOriginalTaskAndUsageProof(t *testing.T) {
	testOwnedImageAuthorizedActivity(t, ownedImageMetadata())
}

func TestOwnedToAPIsAuthorizedActivityKeepsOriginalTaskAndUsageProof(t *testing.T) {
	metadata := ownedImageMetadata()
	metadata.Provider = aicapability.ToAPIsProductImageProvider
	metadata.Model = aicapability.ToAPIsProductImageModel
	metadata.Protocol = aicapability.ToAPIsProductImageProtocol
	testOwnedImageAuthorizedActivity(t, metadata)
}

func testOwnedImageAuthorizedActivity(t *testing.T, metadata imageagent.GenerationProviderMetadata) {
	t.Helper()
	for _, name := range []string{"first_lookup_success", "lookup_timeout_then_recover_original", "detail_succeeds_while_main_needs_recovery", "detail_succeeds_main_later_fails", "invalid_result"} {
		t.Run(name, func(t *testing.T) {
			mainFails := name == "detail_succeeds_main_later_fails"
			includeDetail := name == "detail_succeeds_while_main_needs_recovery" || mainFails
			recoverOriginal := name == "lookup_timeout_then_recover_original" || includeDetail
			invalidResult := name == "invalid_result"
			ctx := context.Background()
			var photo bytes.Buffer
			require.NoError(t, png.Encode(&photo, image.NewNRGBA(image.Rect(0, 0, 1024, 1024))))
			a, repo, input := imageSetPersistenceFixtureWithSource(t, photo.Bytes(), includeDetail, metadata)
			input.BudgetAuthorization = true
			runScope := slotEffectReservationV3(slotExecutionInputV3(input)).Identity.RunScope
			initial, err := repo.GetProjection(ctx, runScope)
			require.NoError(t, err)
			policy, err := initial.Run.Budget.Policy()
			require.NoError(t, err)
			input.BudgetPolicy = policy
			execution := slotExecutionInputV3(input)
			facts := repo.(imageagent.GenerationFactRepository)
			id := slotEffectReservationV3(execution).Identity
			const task = "14-task-1"
			const detailTask = "14-task-2"
			var detailInput ExecuteSlotV3ActivityInput
			var detailID imageagent.SlotExternalEffectIdentity
			if includeDetail {
				detailInput = input
				detailInput.Slot = initial.Plan.Slots[1]
				detailInput.IdempotencyKey = slotAttemptKey(1, detailInput.Slot, 1)
				detailID = slotEffectReservationV3(slotExecutionInputV3(detailInput)).Identity
			}
			submits, lookups := 0, 0
			detailSubmits, detailLookups := 0, 0
			provider := isolatedOwnedImageTasks{
				start: func(prompt string, source []byte) (string, error) {
					require.Equal(t, photo.Bytes(), source)
					if includeDetail && prompt == detailInput.Slot.Recipe.Prompt {
						detailSubmits++
						return detailTask, nil
					}
					submits++
					require.Equal(t, input.Slot.Recipe.Prompt, prompt)
					return task, nil
				},
				lookup: func(taskID string) (imageagent.ProviderUsageImageResult, error) {
					if taskID == detailTask {
						detailLookups++
						fact, err := facts.ReadGenerationFact(ctx, detailID)
						require.NoError(t, err)
						require.Equal(t, detailTask, fact.ProviderTask.ID)
						return imageagent.ProviderUsageImageResult{TaskID: detailTask, Status: "succeeded", ResultURL: "https://images.example.org/detail.png", Digest: strings.Repeat("e", 64)}, nil
					}
					lookups++
					fact, err := facts.ReadGenerationFact(ctx, id)
					require.NoError(t, err)
					require.Equal(t, task, fact.ProviderTask.ID, "the task must be durable before the first query")
					require.Equal(t, task, taskID)
					if recoverOriginal && lookups == 1 {
						return imageagent.ProviderUsageImageResult{}, context.DeadlineExceeded
					}
					if mainFails {
						if lookups == 2 {
							return imageagent.ProviderUsageImageResult{TaskID: task, Status: "running"}, nil
						}
						return imageagent.ProviderUsageImageResult{TaskID: task, Status: "failed", Digest: strings.Repeat("d", 64)}, nil
					}
					return imageagent.ProviderUsageImageResult{TaskID: task, Status: "succeeded", ResultURL: "https://images.example.org/result.png", Digest: strings.Repeat("d", 64)}, nil
				},
			}
			readOutput := func(_ context.Context, slot imageagent.SlotExecutionInput, fact imageagent.GenerationFact) (imageagent.SlotGeneratedOutput, error) {
				require.Equal(t, imageagent.GenerationSucceeded, fact.State)
				taskID := task
				if slot.Slot.ID == detailID.SlotID {
					taskID = detailTask
				}
				require.Equal(t, taskID, fact.Success.RequestID)
				if invalidResult {
					require.Equal(t, "https://images.example.org/result.png", fact.Success.ResultURL)
					return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
				}
				return imageagent.SlotGeneratedOutput{SlotID: slot.Slot.ID, Attempt: 1, SourceAssetID: "source-1",
					Assets: []imageagent.GeneratedAsset{{Bytes: photo.Bytes(), ContentType: "image/png", Width: 1024, Height: 1024,
						Operations: []string{"render_source_edit"}, ProviderReceiptID: taskID}}}, nil
			}
			generation, err := imageagent.NewGenerationExecution(imageagent.GenerationExecutionDependencies{
				Facts: facts, Resources: struct{ imageagent.GenerationResources }{}, Authorizer: allowGenerationExecution{},
				ReadMemberLimit: func(context.Context, string, string) (imageagent.GenerationMemberLimit, error) {
					t.Fatal("enterprise-owned images must not read platform member point limits")
					return imageagent.GenerationMemberLimit{}, nil
				},
				ReadSourceBytes: func(context.Context, imageagent.SlotExecutionInput) ([]byte, error) { return photo.Bytes(), nil },
				PrepareProvider: func(context.Context, imageagent.SlotExecutionInput, func(context.Context, imageagent.GenerationSuccess) error) (imageagent.PreparedGenerationProvider, error) {
					return imageagent.PreparedGenerationProvider{Metadata: metadata, ProviderUsage: provider}, nil
				},
				RevalidateProvider: func(context.Context, imageagent.SlotExecutionInput, imageagent.GenerationProviderMetadata) error {
					return nil
				},
				MaxSourceBytes:     3 << 20,
				ReadProviderOutput: readOutput,
			})
			require.NoError(t, err)
			capacity := &isolatedOwnedStorage{failClose: invalidResult}
			recovery, err := imageagent.NewGenerationRecovery(repo.(imageagent.GenerationRecoveryFacts), struct {
				imageagent.GenerationFinalizationResource
			}{})
			require.NoError(t, err)
			a.generationRecovery = recovery
			if invalidResult {
				generation.ConfigureStorageCapacity(capacity, 3<<20)
				imageagent.ConfigureGenerationRecoveryStorage(recovery, capacity)
			}
			recorder := &recordingStagedExecutor{mutateResult: func(result *imageagent.SlotExecutionResult) {
				for i := range result.Candidates {
					result.Candidates[i].AssetID = result.SlotID + "-candidate"
					result.Candidates[i].Width, result.Candidates[i].Height = 1024, 1024
					result.Candidates[i].Operations = []string{"render_source_edit"}
				}
			}}
			a.stagedSlotExecutor = isolatedOwnedImageExecutor{generation, recorder}
			a.generationOutputRecovery = generation.ReadGenerationOutput
			published, err := a.ExecuteSlotV3(ctx, input)
			if recoverOriginal {
				require.Error(t, err)
				fact, readErr := facts.ReadGenerationFact(ctx, id)
				require.NoError(t, readErr)
				require.Equal(t, imageagent.GenerationUnknown, fact.State)
				if includeDetail {
					detail, detailErr := a.ExecuteSlotV3(ctx, detailInput)
					require.NoError(t, detailErr)
					detailProjection, detailErr := a.PersistImageSetSlotResult(ctx, PersistSlotResultV3ActivityInput{
						RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: detailInput.IdempotencyKey,
						Result: SlotWorkflowV3Result{Published: detail, Status: imageagent.SlotStatusAccepted, EffectPhase: imageagent.SlotEffectV3PublicationComplete},
					})
					require.NoError(t, detailErr)
					require.Len(t, detailProjection.Candidates, 1)
					detailFact, detailErr := facts.ReadGenerationFact(ctx, detailID)
					require.NoError(t, detailErr)
					require.EqualValues(t, 1, detailFact.Success.ProviderImageCount)
					require.Equal(t, detailFact.TerminalProofDigest(), detailProjection.Candidates[0].GenerationProof.ProviderUsageProofDigest)
					require.Empty(t, detailFact.Reservation)
					require.Empty(t, detailFact.Settlement)
					require.Equal(t, 1, detailSubmits)
					require.Equal(t, 1, detailLookups)
				}
				recoveryInput := EffectRecoveryWorkflowInput{RunID: input.RunID,
					Identity: input.Identity, PlanRevision: 1, Attempt: 1, Slot: input.Slot, ImageSet: input.ImageSet,
					AssetCatalog: input.AssetCatalog, TargetPlatform: "product", ActionID: "original-task-check"}
				if mainFails {
					_, err := a.PersistImageSetSlotResult(ctx, PersistSlotResultV3ActivityInput{RunID: input.RunID,
						Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey,
						Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1},
							Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.SlotProviderOutcomeUnknownCode, EffectPhase: imageagent.SlotEffectV3ProviderUnknown}})
					require.NoError(t, err)
					current, err := repo.GetProjection(ctx, runScope)
					require.NoError(t, err)
					unknown := imageagent.SlotProviderOutcomeUnknownCode
					require.NoError(t, a.PersistRunState(ctx, PersistRunStateActivityInput{RunID: input.RunID, Identity: input.Identity,
						PlanRevision: 1, CommitID: "blocked", CurrentNode: "retry_slot", Projection: WorkflowResult{Status: imageagent.RunStatusBlocked,
							Block: &imageagent.Block{Code: unknown, Message: "waiting for supplier result", SlotID: input.Slot.ID},
							Plan:  current.Plan, Slots: current.Slots, RecoverableEffects: []imageagent.RecoverableEffect{{SlotID: input.Slot.ID, Attempt: 1, Code: unknown}}}}))
					_, err = a.RecoverEffectV3(ctx, recoveryInput)
					require.NoError(t, err)
					waiting, err := a.ReconcileEffectRecoveryV3(ctx, recoveryInput)
					require.NoError(t, err)
					require.Nil(t, waiting.Closure)
					waitingProjection, err := repo.GetProjection(ctx, runScope)
					require.NoError(t, err)
					require.Equal(t, unknown, waitingProjection.Run.Block.Message)
					require.Len(t, waitingProjection.RecoverableEffects, 1)
					_, err = a.ReconcileEffectRecoveryV3(ctx, recoveryInput)
					require.NoError(t, err, "repeated readback while still running is a no-op")

					recoveryInput.ActionID = "original-task-check-again"
					_, err = a.RecoverEffectV3(ctx, recoveryInput)
					require.NoError(t, err)
					failed, err := a.ReconcileEffectRecoveryV3(ctx, recoveryInput)
					require.NoError(t, err, "later evidence must close the same unknown attempt")
					require.Equal(t, imageagent.ImageProviderFailedCode, failed.BlockedCode)
					require.Equal(t, "provider_failed", failed.Closure.Kind)
					again, err := a.ReconcileEffectRecoveryV3(ctx, recoveryInput)
					require.NoError(t, err)
					require.Equal(t, failed, again)
					closed, err := repo.GetProjection(ctx, runScope)
					require.NoError(t, err)
					require.Empty(t, closed.RecoverableEffects)
					_, err = imageagent.ImageSetCandidateResultDigest(closed)
					require.NoError(t, err, "the successful detail is eligible for manual adoption")
					require.Equal(t, 1, submits)
					require.Equal(t, 3, lookups)
					return
				}
				result, recoverErr := a.RecoverEffectV3(ctx, recoveryInput)
				require.NoError(t, recoverErr)
				require.Equal(t, EffectRecoveryOutcomePublished, result.Outcome)
				published = result.Published
				require.Equal(t, 2, lookups)
			} else if invalidResult {
				require.Error(t, err)
				published = imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, lookups)
			}
			require.Equal(t, 1, submits)
			fact, err := facts.ReadGenerationFact(ctx, id)
			require.NoError(t, err)
			require.EqualValues(t, 1, fact.Success.ProviderImageCount)
			require.Empty(t, fact.Reservation)
			require.Empty(t, fact.Settlement)
			status, phase := imageagent.SlotStatusAccepted, imageagent.SlotEffectV3PublicationComplete
			if invalidResult {
				status, phase = imageagent.SlotStatusBlocked, imageagent.SlotEffectV3ProviderUnknown
				require.True(t, fact.ProviderOutputInvalid)
				require.Equal(t, "DISPATCHED", capacity.reservation.State)
			}
			persist := PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity,
				PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: published,
					Status: status, EffectPhase: phase}}
			if invalidResult {
				_, err := a.PersistImageSetSlotResult(ctx, persist)
				require.ErrorIs(t, err, storagecapacity.ErrUnavailable, "cannot close the projection before the capacity close")
				capacity.failClose = false
			}
			projection, err := a.PersistImageSetSlotResult(ctx, persist)
			require.NoError(t, err)
			if invalidResult {
				require.Empty(t, projection.Candidates)
				require.Equal(t, imageagent.InvalidGeneratedOutputCode, projection.ErrorCode)
				require.Equal(t, fact.TerminalProofDigest(), projection.Closure.ProviderUsageProofDigest)
				require.Equal(t, "CLOSED", capacity.reservation.State)
				require.Equal(t, "invalid-generation:"+fact.TerminalProofDigest(), capacity.proof.TerminalReceipt)
				return
			}
			require.Len(t, projection.Candidates, 1)
			require.Equal(t, fact.TerminalProofDigest(), projection.Candidates[0].GenerationProof.ProviderUsageProofDigest)
			require.Empty(t, projection.Candidates[0].GenerationProof.SettlementProofDigest)
			approval, err := repo.GetProjection(ctx, id.RunScope)
			require.NoError(t, err)
			approval.Run.Status = imageagent.RunStatusAwaitingFinalApproval
			approval.ResultDigest, err = imageagent.ImageSetResultDigest(approval.Plan, approval.Slots, nil)
			require.NoError(t, err)
			reader, err := assetpublication.NewImageSetCandidateReader(isolatedApprovalProjection(approval), facts, a.slotEffectsV3, a.artifactStore.(*recordingArtifactStore))
			require.NoError(t, err)
			bound := approval.Plan.Set.Source
			source := asset.SourceSelection{ContextKind: string(bound.ContextKind), TenantID: input.Identity.TenantID,
				ActorID: input.Identity.UserID, MemberID: input.Identity.MemberID, ItemID: bound.OperationID,
				ProductKey: bound.ProductID, OriginalPublicationID: bound.OriginalPublicationID, OriginalSnapshotVersion: bound.OriginalVersion,
				EffectiveCatalogVersion: bound.EffectiveVersion, TargetPlatform: "product"}
			binding := asset.ImageSetResultBinding{RunID: input.RunID, PlanRevision: 1, ResultDigest: approval.ResultDigest}
			choice := asset.ImageSetChoice{Kind: "generated", RunID: input.RunID, PlanRevision: 1, ResultDigest: approval.ResultDigest,
				SlotID: input.Slot.ID, Attempt: 1, AssetID: projection.Candidates[0].AssetID, Presentation: asset.ImagePresentation{Group: "carousel", Order: 1}}
			approved, err := reader.ReadImageSetCandidate(ctx, source, binding, choice)
			require.NoError(t, err)
			require.Equal(t, fact.TerminalProofDigest(), approved.Asset.GenerationEvidence.ProviderUsageProofDigest)
			require.Equal(t, projection.Candidates[0].DurableAsset.SHA256, approved.Asset.GenerationEvidence.ArtifactHash)
			if includeDetail {
				for _, slot := range approval.Slots {
					if slot.Slot.ID == detailInput.Slot.ID {
						choice.SlotID, choice.AssetID = slot.Slot.ID, slot.Candidates[0].AssetID
					}
				}
				choice.Presentation.Group = "detail"
				approvedDetail, err := reader.ReadImageSetCandidate(ctx, source, binding, choice)
				require.NoError(t, err)
				require.NotEqual(t, approved.Asset.GenerationEvidence.ProviderUsageProofDigest, approvedDetail.Asset.GenerationEvidence.ProviderUsageProofDigest)
			}
			source.TenantID = "another-enterprise"
			_, err = reader.ReadImageSetCandidate(ctx, source, binding, choice)
			require.Error(t, err)
		})
	}
}

type settledSetExecutor struct {
	*recordingStagedExecutor
	onGenerate func(context.Context, imageagent.SlotExecutionInput)
}

func (e *settledSetExecutor) GenerateSlot(ctx context.Context, input imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	e.onGenerate(ctx, input)
	return e.recordingStagedExecutor.GenerateSlot(ctx, input)
}

func persistedAcceptedImageSetFixture(t *testing.T) (*Activities, imageagent.Repository, ExecuteSlotV3ActivityInput, imageagent.SlotEffectV3PublishedResult, imageagent.GenerationFact, *recordingStagedExecutor) {
	t.Helper()
	a, repo, input := imageSetPersistenceFixture(t)
	var pngBytes bytes.Buffer
	require.NoError(t, png.Encode(&pngBytes, image.NewNRGBA(image.Rect(0, 0, 1024, 1024))))
	recorder := &recordingStagedExecutor{generated: imageagent.SlotGeneratedOutput{SlotID: input.Slot.ID, Attempt: 1, SourceAssetID: "source-1", Assets: []imageagent.GeneratedAsset{{Bytes: pngBytes.Bytes(), ContentType: "image/png", Width: 1024, Height: 1024, Operations: []string{"render_source_edit"}, ProviderReceiptID: "response"}}}}
	recorder.mutateResult = func(result *imageagent.SlotExecutionResult) {
		for index := range result.Candidates {
			result.Candidates[index].Width, result.Candidates[index].Height = 1024, 1024
			result.Candidates[index].Operations = []string{"render_source_edit"}
		}
	}
	var originalFact imageagent.GenerationFact
	a.stagedSlotExecutor = &settledSetExecutor{recordingStagedExecutor: recorder, onGenerate: func(ctx context.Context, execution imageagent.SlotExecutionInput) {
		quote := execution.Slot.Recipe.Quote
		digest, err := imageagent.ImageSlotGenerationInputDigest(execution)
		require.NoError(t, err)
		intent := imageagent.GenerationIntent{Billing: syntheticImageBilling(quote.Model, quote.PriceVersion, quote.Points), Identity: slotEffectReservationV3(execution).Identity, MemberID: input.Identity.MemberID, CatalogHash: execution.AssetCatalog.Manifest.Hash, SourceDigest: imageagent.ImageSourceBundleDigest(execution.Slot.Recipe.References), PromptVersion: execution.Slot.Recipe.PromptVersion, InputProtocol: imageagent.ImageSetSchema, InputDigest: digest, RouteReference: quote.RouteReference, CredentialReference: quote.CredentialReference, ConfigurationVersion: quote.ConfigurationVersion, Provider: quote.Provider, Model: quote.Model, Protocol: quote.Protocol, Resolution: quote.Resolution, Quality: quote.Quality, PriceVersion: quote.PriceVersion, Points: quote.Points, LimitVersion: 1, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
		facts := repo.(imageagent.GenerationFactRepository)
		fact, err := facts.PrepareGenerationIntent(ctx, intent)
		require.NoError(t, err)
		fact, err = facts.BindGenerationReservation(ctx, intent, imageagent.GenerationReservationReceipt{UnitVersion: "ai-micropoint-v1", IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: input.Identity.TenantID, MemberID: input.Identity.MemberID, OperationID: "image-reserve:" + fact.IntentID, ReservationID: "reservation", ResourceType: "ai_point", Points: intent.Points, PriceVersion: intent.PriceVersion, LimitVersion: intent.LimitVersion, MonthStart: intent.MonthStart})
		require.NoError(t, err)
		_, won, err := facts.BeginGenerationDispatch(ctx, intent)
		require.NoError(t, err)
		require.True(t, won)
		fact, err = facts.RecordGenerationSuccess(ctx, intent, imageagent.GenerationSuccess{BillableUsage: syntheticImageUsage(100, 100), UsageKnown: true, InputTokens: 100, OutputTokens: 100, TotalTokens: 200, ResponseID: "response", ResultDigest: strings.Repeat("b", 64), ResultURL: "https://images.example.org/output.png"})
		require.NoError(t, err)
		originalFact, err = facts.BindGenerationSettlement(ctx, intent, imageagent.GenerationSettlementReceipt{UnitVersion: "ai-micropoint-v1", ReservedMicropoints: intent.Points, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: "reservation", State: "committed", Points: intent.Points, ProofDigest: fact.TerminalProofDigest()})
		require.NoError(t, err)
	}}
	published, err := a.ExecuteSlotV3(context.Background(), input)
	require.NoError(t, err)
	projection, err := a.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: published, Status: imageagent.SlotStatusAccepted, EffectPhase: imageagent.SlotEffectV3PublicationComplete}})
	require.NoError(t, err)
	require.Len(t, projection.Candidates, 1)
	require.Equal(t, &imageagent.ImageGenerationProof{UnitVersion: "ai-micropoint-v1", IntentID: originalFact.IntentID, Fingerprint: originalFact.Fingerprint, SettlementProofDigest: originalFact.TerminalProofDigest(), Points: 12}, projection.Candidates[0].GenerationProof)
	return a, repo, input, published, originalFact, recorder
}

func TestPersistImageSetAcceptedOutputKeepsOriginalGenerationProof(t *testing.T) {
	a, repo, input, published, originalFact, recorder := persistedAcceptedImageSetFixture(t)
	projection, err := repo.GetProjection(context.Background(), originalFact.Intent.Identity.RunScope)
	require.NoError(t, err)
	accepted := projection.Slots[0]
	replay, err := a.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: published, Status: imageagent.SlotStatusAccepted, EffectPhase: imageagent.SlotEffectV3PublicationComplete}})
	require.NoError(t, err)
	require.Equal(t, accepted, replay)
	expired, err := a.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.BudgetElapsedCode, EffectPhase: imageagent.SlotEffectV3ProviderNotDispatched}})
	require.NoError(t, err)
	require.Equal(t, accepted, expired, "expiration must not overwrite original settled materialized success")
	require.Equal(t, 1, recorder.GenerateCalls())
	current, err := repo.GetProjection(context.Background(), originalFact.Intent.Identity.RunScope)
	require.NoError(t, err)
	_, err = imageagent.ImageSetResultDigest(current.Plan, current.Slots, nil)
	require.NoError(t, err, "the retained set must bind the materialized output and original settlement")
	quota, err := a.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.StorageLimitExceededCode, EffectPhase: imageagent.SlotEffectV3ProviderNotDispatched}})
	require.NoError(t, err)
	require.Equal(t, accepted, quota, "a later quota denial cannot overwrite the original settled success")
}

func TestPersistImageSetStorageLimitKeepsNativeClosureAndPreservesUnknown(t *testing.T) {
	for _, mode := range []string{"known_no_generation", "original_unknown", "native_unknown"} {
		t.Run(mode, func(t *testing.T) {
			unknown := mode != "known_no_generation"
			a, repo, input := imageSetPersistenceFixture(t)
			ctx := context.Background()
			execution := slotExecutionInputV3(input)
			reservation := slotEffectReservationV3(execution)
			effects := repo.(imageagent.SlotExternalEffectV3Repository)
			_, won, err := effects.ReserveSlotProviderV3(ctx, reservation)
			require.NoError(t, err)
			require.True(t, won)
			digest, err := imageagent.ImageSlotGenerationInputDigest(execution)
			require.NoError(t, err)
			quote := input.Slot.Recipe.Quote
			intent := imageagent.GenerationIntent{Billing: syntheticImageBilling(quote.Model, quote.PriceVersion, quote.Points), Identity: reservation.Identity, MemberID: input.Identity.MemberID, CatalogHash: input.AssetCatalog.Manifest.Hash, SourceDigest: imageagent.ImageSourceBundleDigest(input.Slot.Recipe.References), PromptVersion: input.Slot.Recipe.PromptVersion, InputProtocol: imageagent.ImageSetSchema, InputDigest: digest, RouteReference: quote.RouteReference, CredentialReference: quote.CredentialReference, ConfigurationVersion: quote.ConfigurationVersion, Provider: quote.Provider, Model: quote.Model, Protocol: quote.Protocol, Resolution: quote.Resolution, Quality: quote.Quality, PriceVersion: quote.PriceVersion, Points: quote.Points, LimitVersion: 1, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
			facts := repo.(imageagent.GenerationFactRepository)
			fact, err := facts.PrepareGenerationIntent(ctx, intent)
			require.NoError(t, err)
			if unknown {
				fact, err = facts.BindGenerationReservation(ctx, intent, imageagent.GenerationReservationReceipt{UnitVersion: "ai-micropoint-v1", IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: input.Identity.TenantID, MemberID: input.Identity.MemberID, OperationID: "image-reserve:" + fact.IntentID, ReservationID: "reservation", ResourceType: "ai_point", Points: intent.Points, PriceVersion: intent.PriceVersion, LimitVersion: intent.LimitVersion, MonthStart: intent.MonthStart})
				require.NoError(t, err)
				_, won, err = facts.BeginGenerationDispatch(ctx, intent)
				require.NoError(t, err)
				require.True(t, won)
				fact, err = facts.MarkGenerationUnknown(ctx, intent)
				require.NoError(t, err)
				if mode == "native_unknown" {
					_, err = effects.RecordSlotProviderNotDispatchedV3(ctx, reservation)
				} else {
					_, err = effects.BlockSlotEffectV3(ctx, imageagent.SlotEffectV3BlockTransition{Reservation: reservation, Phase: imageagent.SlotEffectV3ProviderUnknown, Code: imageagent.SlotProviderOutcomeUnknownCode})
				}
				require.NoError(t, err)
			} else {
				fact, err = facts.RecordGenerationNoEffect(ctx, intent)
				require.NoError(t, err)
				fact, err = facts.BindGenerationSettlement(ctx, intent, imageagent.GenerationSettlementReceipt{UnitVersion: "ai-micropoint-v1", ReservedMicropoints: intent.Points, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, State: "no_reservation", Points: 0, ProofDigest: fact.TerminalProofDigest()})
				require.NoError(t, err)
				_, err = effects.RecordSlotProviderNotDispatchedV3(ctx, reservation)
				require.NoError(t, err)
			}
			in := PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.StorageLimitExceededCode, EffectPhase: imageagent.SlotEffectV3ProviderNotDispatched}}
			result, err := a.PersistImageSetSlotResult(ctx, in)
			require.NoError(t, err)
			if unknown {
				require.Equal(t, imageagent.SlotProviderOutcomeUnknownCode, result.ErrorCode)
				require.Nil(t, result.Closure)
			} else {
				require.Equal(t, imageagent.StorageLimitExceededCode, result.ErrorCode)
				closure, err := imageagent.ImageGenerationClosure(fact)
				require.NoError(t, err)
				require.Equal(t, closure, result.Closure)
				replay, err := a.PersistImageSetSlotResult(ctx, in)
				require.NoError(t, err)
				require.Equal(t, result, replay)
			}
			retained, err := facts.ReadGenerationFact(ctx, reservation.Identity)
			require.NoError(t, err)
			require.Equal(t, fact, retained)
			require.Zero(t, a.stagedSlotExecutor.(*recordingStagedExecutor).GenerateCalls())
		})
	}
}

func TestPersistImageSetSlotResultDistinguishesKnownUnstartedFromUnknown(t *testing.T) {
	for _, mode := range []string{"known_unstarted", "unknown", "expired_unknown"} {
		t.Run(mode, func(t *testing.T) {
			activities, repo, input := imageSetPersistenceFixture(t)
			code, phase := imageagent.BudgetElapsedCode, imageagent.SlotEffectV3ProviderNotDispatched
			if mode != "known_unstarted" {
				_, won, err := repo.(imageagent.SlotExternalEffectV3Repository).ReserveSlotProviderV3(context.Background(), slotEffectReservationV3(slotExecutionInputV3(input)))
				require.NoError(t, err)
				require.True(t, won)
				code, phase = imageagent.SlotProviderOutcomeUnknownCode, imageagent.SlotEffectV3ProviderUnknown
				if mode == "expired_unknown" {
					code, phase = imageagent.BudgetElapsedCode, imageagent.SlotEffectV3ProviderNotDispatched
				}
			}
			result, err := activities.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: code, EffectPhase: phase}})
			require.NoError(t, err)
			if mode == "known_unstarted" {
				require.Zero(t, result.Attempt)
				require.Equal(t, &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "not_dispatched"}, result.Closure)
			} else {
				require.Equal(t, 1, result.Attempt)
				require.Nil(t, result.Closure)
				require.Equal(t, imageagent.SlotProviderOutcomeUnknownCode, result.ErrorCode, "a later budget denial cannot replace the original UNKNOWN owner")
			}
		})
	}
}

func TestExpiredImageSetProjectionPreservesPersistedBlockedEffectPolicy(t *testing.T) {
	for _, policy := range []struct {
		phase imageagent.SlotEffectV3Phase
		code  string
	}{
		{imageagent.SlotEffectV3ProviderUnknown, imageagent.SlotProviderOutcomeUnknownCode},
		{imageagent.SlotEffectV3StagingUnknown, imageagent.SlotStagingOutcomeUnknownCode},
		{imageagent.SlotEffectV3PublicationUnknown, imageagent.SlotPublicationOutcomeUnknownCode},
		{imageagent.SlotEffectV3RecoveryBlocked, imageagent.SlotRecoveryBlockedCode},
	} {
		t.Run(string(policy.phase), func(t *testing.T) {
			a, repo, input := imageSetPersistenceFixture(t)
			ctx := context.Background()
			effects := repo.(imageagent.SlotExternalEffectV3Repository)
			execution := slotExecutionInputV3(input)
			reservation := slotEffectReservationV3(execution)
			_, won, err := effects.ReserveSlotProviderV3(ctx, reservation)
			require.NoError(t, err)
			require.True(t, won)
			transition := imageagent.SlotEffectV3BlockTransition{Reservation: reservation, Phase: policy.phase, Code: policy.code}
			if policy.phase == imageagent.SlotEffectV3StagingUnknown || policy.phase == imageagent.SlotEffectV3PublicationUnknown {
				manifest := v3StagingManifest(input, tinyPNGBytes(t))
				_, err = effects.PrepareSlotStagingV3(ctx, reservation, manifest)
				require.NoError(t, err)
				if policy.phase == imageagent.SlotEffectV3PublicationUnknown {
					fingerprint, err := imageagent.StagingManifestFingerprint(manifest)
					require.NoError(t, err)
					_, err = effects.CommitSlotStagedV3(ctx, reservation, fingerprint)
					require.NoError(t, err)
					final, err := expectedFinalManifestV3(execution, manifest)
					require.NoError(t, err)
					fingerprint, err = imageagent.FinalManifestFingerprint(final)
					require.NoError(t, err)
					_, claim, won, err := effects.ClaimSlotPublicationV3(ctx, imageagent.PublicationClaimRequest{Reservation: reservation, Owner: "original-owner", LeaseDuration: time.Minute, PublicationFingerprint: fingerprint, FinalManifest: final})
					require.NoError(t, err)
					require.True(t, won)
					transition.Owner, transition.Fence = claim.Owner, claim.Fence
				}
			}
			original, err := effects.BlockSlotEffectV3(ctx, transition)
			require.NoError(t, err)
			result, err := a.PersistImageSetSlotResult(ctx, PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.BudgetElapsedCode, EffectPhase: imageagent.SlotEffectV3ProviderNotDispatched}})
			require.NoError(t, err)
			require.Equal(t, policy.code, result.ErrorCode)
			require.Equal(t, 1, result.Attempt)
			require.Nil(t, result.Closure)
			wantPolicy, err := imageagent.SlotEffectV3BlockedPolicyFor(original.Phase, original.BlockedCode)
			require.NoError(t, err)
			gotPolicy, ok := imageagent.SlotEffectV3BlockedPolicyForCode(result.ErrorCode)
			require.True(t, ok)
			require.Equal(t, wantPolicy, gotPolicy, "expiration must retain the original permitted actions")
			retained, err := effects.GetSlotExternalEffectV3(ctx, reservation.Identity)
			require.NoError(t, err)
			require.Equal(t, original, retained)
			require.Zero(t, a.stagedSlotExecutor.(*recordingStagedExecutor).GenerateCalls())
		})
	}
}

func TestSetSlotClosureBindsTheOriginalSettledEconomics(t *testing.T) {
	for _, mode := range []string{"invalid_locator", "invalid_locator_execute", "invalid_bytes", "invalid_bytes_execute", "transient", "unsettled_invalid_bytes", "price_drift"} {
		t.Run(mode, func(t *testing.T) {
			drift := mode == "price_drift"
			a, repo, input := imageSetPersistenceFixture(t)
			execution := slotExecutionInputV3(input)
			reservation := slotEffectReservationV3(execution)
			_, won, err := repo.(imageagent.SlotExternalEffectV3Repository).ReserveSlotProviderV3(context.Background(), reservation)
			require.NoError(t, err)
			require.True(t, won)
			digest, err := imageagent.ImageSlotGenerationInputDigest(execution)
			require.NoError(t, err)
			quote := input.Slot.Recipe.Quote
			intent := imageagent.GenerationIntent{Billing: syntheticImageBilling(quote.Model, quote.PriceVersion, quote.Points), Identity: reservation.Identity, MemberID: input.Identity.MemberID, CatalogHash: input.AssetCatalog.Manifest.Hash, SourceDigest: imageagent.ImageSourceBundleDigest(input.Slot.Recipe.References), PromptVersion: input.Slot.Recipe.PromptVersion, InputProtocol: imageagent.ImageSetSchema, InputDigest: digest, RouteReference: quote.RouteReference, CredentialReference: quote.CredentialReference, ConfigurationVersion: quote.ConfigurationVersion, Provider: quote.Provider, Model: quote.Model, Protocol: quote.Protocol, Resolution: quote.Resolution, Quality: quote.Quality, PriceVersion: quote.PriceVersion, Points: quote.Points, LimitVersion: 1, MonthStart: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)}
			if drift {
				intent.PriceVersion = "different-price"
				intent.Billing = syntheticImageBilling(intent.Model, intent.PriceVersion, intent.Points)
			}
			facts := repo.(imageagent.GenerationFactRepository)
			fact, err := facts.PrepareGenerationIntent(context.Background(), intent)
			require.NoError(t, err)
			fact, err = facts.BindGenerationReservation(context.Background(), intent, imageagent.GenerationReservationReceipt{UnitVersion: "ai-micropoint-v1", IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OrganizationID: input.Identity.TenantID, MemberID: input.Identity.MemberID, OperationID: "image-reserve:" + fact.IntentID, ReservationID: "reservation", ResourceType: "ai_point", Points: intent.Points, PriceVersion: intent.PriceVersion, LimitVersion: intent.LimitVersion, MonthStart: intent.MonthStart})
			require.NoError(t, err)
			_, won, err = facts.BeginGenerationDispatch(context.Background(), intent)
			require.NoError(t, err)
			require.True(t, won)
			if !drift {
				_, err = repo.(imageagent.SlotExternalEffectV3Repository).BlockSlotEffectV3(context.Background(), imageagent.SlotEffectV3BlockTransition{Reservation: reservation, Phase: imageagent.SlotEffectV3ProviderUnknown, Code: imageagent.SlotProviderOutcomeUnknownCode})
				require.NoError(t, err)
				_, err = a.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.SlotProviderOutcomeUnknownCode, EffectPhase: imageagent.SlotEffectV3ProviderUnknown}})
				require.NoError(t, err)
				current, err := repo.GetProjection(context.Background(), reservation.Identity.RunScope)
				require.NoError(t, err)
				block := &imageagent.Block{Code: imageagent.SlotProviderOutcomeUnknownCode, Message: imageagent.SlotProviderOutcomeUnknownCode, SlotID: input.Slot.ID}
				err = a.PersistRunState(context.Background(), PersistRunStateActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, CommitID: "blocked", CurrentNode: "retry_slot", Projection: WorkflowResult{Status: imageagent.RunStatusBlocked, Block: block, Plan: current.Plan, Slots: current.Slots, RecoverableEffects: []imageagent.RecoverableEffect{{SlotID: input.Slot.ID, Attempt: 1, Code: imageagent.SlotProviderOutcomeUnknownCode}}}})
				require.NoError(t, err)
			}
			proof := imageagent.GenerationSuccess{BillableUsage: syntheticImageUsage(100, 100), UsageKnown: true, InputTokens: 100, OutputTokens: 100, TotalTokens: 200, ResponseID: "response", ResultDigest: strings.Repeat("b", 64), ResultUnavailable: "invalid_result"}
			if mode == "invalid_bytes" || mode == "invalid_bytes_execute" || mode == "transient" || mode == "unsettled_invalid_bytes" {
				proof = proof.WithResultLocator("https://output.example/image.png", "")
			}
			fact, err = facts.RecordGenerationSuccess(context.Background(), intent, proof)
			require.NoError(t, err)
			if mode != "unsettled_invalid_bytes" {
				fact, err = facts.BindGenerationSettlement(context.Background(), intent, imageagent.GenerationSettlementReceipt{UnitVersion: "ai-micropoint-v1", ReservedMicropoints: intent.Points, IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, OperationID: "image-finalize:" + fact.IntentID, ReservationID: "reservation", State: "committed", Points: intent.Points, ProofDigest: fact.TerminalProofDigest()})
				require.NoError(t, err)
			}
			if drift {
				_, err := a.PersistImageSetSlotResult(context.Background(), PersistSlotResultV3ActivityInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, AttemptKey: input.IdempotencyKey, Result: SlotWorkflowV3Result{Published: imageagent.SlotEffectV3PublishedResult{SlotID: input.Slot.ID, Attempt: 1}, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.SlotProviderOutcomeUnknownCode, EffectPhase: imageagent.SlotEffectV3ProviderUnknown}})
				require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
				return
			}
			recovery := EffectRecoveryWorkflowInput{RunID: input.RunID, Identity: input.Identity, PlanRevision: 1, Slot: input.Slot, Attempt: 1, ImageSet: input.ImageSet, TargetPlatform: input.TargetPlatform, AssetCatalog: input.AssetCatalog}
			gets := 0
			if proof.ResultURL != "" || mode == "invalid_locator" || mode == "invalid_locator_execute" {
				a.generationOutputRecovery = func(_ context.Context, original imageagent.SlotExecutionInput, persisted imageagent.GenerationFact) (imageagent.SlotGeneratedOutput, error) {
					gets++
					require.Equal(t, input.Slot.ID, original.Slot.ID)
					require.Equal(t, fact, persisted)
					if mode == "transient" {
						return imageagent.SlotGeneratedOutput{}, errors.New("download timed out")
					}
					return imageagent.SlotGeneratedOutput{}, imageagent.ErrInvalidGeneratedOutput
				}
				if mode == "invalid_bytes_execute" || mode == "invalid_locator_execute" {
					_, err = a.ExecuteSlotV3(context.Background(), input)
					require.ErrorContains(t, err, imageagent.SlotProviderOutcomeUnknownCode)
				} else if mode == "transient" {
					_, err = a.RecoverEffectV3(context.Background(), recovery)
					require.Error(t, err)
				} else {
					_, err = a.RecoverEffectV3(context.Background(), recovery)
					require.NoError(t, err, "known invalid output must reach original durable reconciliation")
				}
				if mode == "invalid_locator" || mode == "invalid_locator_execute" {
					require.Equal(t, 1, gets, "empty immutable locator must reach the bound output recovery")
				}
			}
			result, err := a.ReconcileEffectRecoveryV3(context.Background(), recovery)
			require.NoError(t, err)
			if mode == "transient" || mode == "unsettled_invalid_bytes" {
				require.Nil(t, result.Closure)
				current, err := repo.GetProjection(context.Background(), reservation.Identity.RunScope)
				require.NoError(t, err)
				require.Len(t, current.RecoverableEffects, 1)
				return
			}
			require.Equal(t, &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "settled", IntentID: fact.IntentID, Fingerprint: fact.Fingerprint, SettlementProofDigest: fact.TerminalProofDigest(), Points: intent.Points}, result.Closure)
			require.Equal(t, imageagent.InvalidGeneratedOutputCode, result.BlockedCode)
			current, err := repo.GetProjection(context.Background(), reservation.Identity.RunScope)
			require.NoError(t, err)
			require.Equal(t, result.Closure, current.Slots[0].Closure)
			require.Empty(t, current.RecoverableEffects)
			replay, err := a.ReconcileEffectRecoveryV3(context.Background(), recovery)
			require.NoError(t, err, "lost acknowledgement must read the original reconciled result")
			require.Equal(t, result, replay)
			if proof.ResultURL != "" || mode == "invalid_locator" || mode == "invalid_locator_execute" {
				closedGets := gets
				_, err = a.RecoverEffectV3(context.Background(), recovery)
				require.NoError(t, err)
				_, err = a.ReconcileEffectRecoveryV3(context.Background(), recovery)
				require.NoError(t, err)
				require.Equal(t, closedGets, gets, "durable invalid closure must not repeatedly download or generate")
			}
			storedFact, err := facts.ReadGenerationFact(context.Background(), reservation.Identity)
			require.NoError(t, err)
			require.Equal(t, fact, storedFact, "invalid output cannot rewrite the original economic success or settlement")
			require.Zero(t, a.stagedSlotExecutor.(*recordingStagedExecutor).GenerateCalls(), "known bad output never redispatches the provider")
			_, err = imageagent.ImageSetClosedEffectsDigest(current.Plan, current.Slots, current.RecoverableEffects)
			require.NoError(t, err, "known charged failure must permit review/subset preparation")
		})
	}
}
