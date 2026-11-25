package imageagent_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/store"
	productasset "github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/stretchr/testify/require"
)

type setConfigurationFixture struct {
	template       agentconfig.SetTemplate
	snapshot       agentconfig.ImageConfigurationSnapshot
	receipt        *agentconfig.ImageRunAdmissionReceipt
	enabled        bool
	admitACK       error
	admissions     int
	admissionReads int
}

func (c *setConfigurationFixture) PrepareImageConfiguration(_ context.Context, input agentconfig.ImageStartCommand) (agentconfig.ImageConfigurationSnapshot, error) {
	if !c.enabled {
		return agentconfig.ImageConfigurationSnapshot{}, agentconfig.ErrNotEnabled
	}
	c.snapshot = agentconfig.ImageConfigurationSnapshot{ID: "91d39d8d-3819-4a7c-ae7f-04ce91951f9a", Digest: strings.Repeat("a", 64), Scope: input.Scope, MemberID: input.MemberID, RequestKey: input.RequestKey, ContextID: input.ContextID, RunID: input.RunID, TargetPlatform: input.TargetPlatform, SourceDigest: input.SourceDigest, InputDigest: input.InputDigest, Parameters: c.template, ParametersDigest: strings.Repeat("b", 64), Epoch: "1", AgentVersion: agentconfig.ImageAgentVersion, HardLimits: input.HardLimits}
	c.snapshot.ExpectedModeRevision = input.ExpectedModeRevision
	if input.Template != nil {
		c.snapshot.Template = *input.Template
	}
	c.snapshot.ModeSnapshotRef = agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: input.RequestKey, Digest: strings.Repeat("c", 64)}
	return c.snapshot, nil
}
func (c *setConfigurationFixture) LoadImageConfiguration(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.ImageConfigurationSnapshot, error) {
	return c.snapshot, nil
}
func (c *setConfigurationFixture) ReadImageRunAdmission(context.Context, agent.Scope, agent.ConfigurationSnapshotRef) (agentconfig.ImageRunAdmissionReceipt, error) {
	c.admissionReads++
	if c.receipt == nil {
		return agentconfig.ImageRunAdmissionReceipt{}, agentconfig.ErrNotFound
	}
	return *c.receipt, nil
}
func (c *setConfigurationFixture) AdmitImageRun(_ context.Context, command agentconfig.ImageRunAdmissionCommand, _ agentconfig.ImageRunLimits) (agentconfig.ImageRunAdmissionReceipt, error) {
	if !c.enabled {
		return agentconfig.ImageRunAdmissionReceipt{}, agentconfig.ErrNotEnabled
	}
	c.admissions++
	now := time.Now().UTC().Truncate(time.Microsecond)
	receipt := agentconfig.ImageRunAdmissionReceipt{ID: "6be5e768-31c1-434e-a6c0-22a508fb3a35", Command: command, AdmittedAt: now, Deadline: now.Add(time.Duration(command.Limits.ElapsedSeconds) * time.Second)}
	raw, _ := json.Marshal(receipt)
	sum := sha256.Sum256(raw)
	receipt.Digest = hex.EncodeToString(sum[:])
	c.receipt = &receipt
	return receipt, c.admitACK
}

type setContextFixture struct {
	copy            *imageagent.ImageTextReviewCopy
	copyReads       int
	copyErr         error
	preparation     imageagent.ImageSetPreparation
	revalidationErr error
}

func TestImageAdmissionCannotRestoreOmittedPendingStatus(t *testing.T) {
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	confirmed, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.NoError(t, err)
	require.NoError(t, imageagent.ValidateImageSetAdmission(confirmed.Run, confirmed.Plan))
	confirmed.Plan.Slots[0].Status = ""
	require.ErrorIs(t, imageagent.ValidateImageSetAdmission(confirmed.Run, confirmed.Plan), imageagent.ErrRevisionConflict)
}

func (c *setContextFixture) ReadConfirmedImageText(context.Context, imageagent.ExecutionIdentity, imageagent.ImageTextReviewReference) (imageagent.ImageTextReviewCopy, error) {
	c.copyReads++
	if c.copyErr != nil {
		return imageagent.ImageTextReviewCopy{}, c.copyErr
	}
	if c.copy == nil {
		return imageagent.ImageTextReviewCopy{}, imageagent.ErrCommandBlocked
	}
	return *c.copy, nil
}

func (c *setContextFixture) ResolveImageSet(_ context.Context, _ imageagent.ExecutionIdentity, input imageagent.PrepareImageSetInput) (imageagent.ImageSetPreparation, error) {
	result := c.preparation
	if input.ColorRework == nil {
		return result, nil
	}
	original := input.SharedOriginalIDs[0]
	result.Catalog.Assets, result.Observations = nil, nil
	for _, asset := range c.preparation.Catalog.Assets {
		if asset.ID == original {
			result.Catalog.Assets = append(result.Catalog.Assets, asset)
		}
	}
	for _, ref := range c.preparation.Observations {
		if ref.AssetID == original {
			result.Observations = append(result.Observations, ref)
		}
	}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{ProductContext: result.Catalog.ProductContext, Assets: result.Catalog.Assets})
	result.Catalog, result.Source.CatalogHash = catalog, catalog.Manifest.Hash
	return result, err
}
func (c *setContextFixture) RevalidateImageSet(_ context.Context, identity imageagent.ExecutionIdentity, projection imageagent.RunProjection) error {
	if identity.RunID != projection.Run.ID {
		return imageagent.ErrIdentityRequired
	}
	return c.revalidationErr
}

type setQuoteFixture struct {
	quote      imageagent.ImageGenerationQuote
	selections []*imageagent.ImageRouteSelection
	err        error
}

func (q *setQuoteFixture) ReadImageGenerationQuote(_ context.Context, _ imageagent.ExecutionIdentity, selection *imageagent.ImageRouteSelection) (imageagent.ImageGenerationQuote, error) {
	q.selections = append(q.selections, imageagent.CloneImageRoute(selection))
	return q.quote, q.err
}

func TestPrepareImageSetRejectsAnotherSourceOwner(t *testing.T) {
	s, _, _, _, contexts, _, ctx, input := imageSetServiceFixture(t)
	input.ContextKind = imageagent.ImageSourceAcquisition
	contexts.preparation.Source.ContextKind = imageagent.ImageSourceListing
	_, err := s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
}

func TestOwnedGRSaiImageSetConfirmsCompletePlanWithZeroPlatformPoints(t *testing.T) {
	for _, name := range []string{"shared_original", "separate_group_originals", "two_originals_in_same_group"} {
		t.Run(name, func(t *testing.T) {
			s, _, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixture(t)
			input.ContextKind = imageagent.ImageSourceListing
			input.Route = &imageagent.ImageRouteSelection{BindingID: strings.Repeat("b", 64), RevisionID: strings.Repeat("c", 64)}
			config.template.Carousel = append(config.template.Carousel, agentconfig.ContentTask{ID: "scene", Purpose: "usage_scene"})
			contexts.preparation.Source.ContextKind = input.ContextKind
			contexts.preparation.OwnProduct = true
			quotes.quote = imageagent.GenerationProviderMetadata{Provider: "grsai", Model: aicapability.GRSaiProductImageModel,
				Protocol: aicapability.GRSaiProductImageProtocol, BillingMode: aicapability.ModelUsageOnly,
				ConfigurationVersion: input.Route.RevisionID, RouteReference: input.Route.BindingID,
				CredentialReference: "grsai-key", ConnectionID: "grsai-connection"}.ImageQuote()

			if name != "shared_original" {
				config.template.ShareOriginals = false
				preparation := &contexts.preparation
				extra := preparation.Catalog.Assets[0]
				extra.ID, extra.URL = "source-2", "https://images.example.org/detail.png"
				catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{
					ProductContext: preparation.Catalog.ProductContext, Assets: append(preparation.Catalog.Assets, extra),
				})
				require.NoError(t, err)
				preparation.Catalog, preparation.Source.CatalogHash = catalog, catalog.Manifest.Hash
				extraObservation := preparation.Observations[0]
				extraObservation.AssetID = extra.ID
				preparation.Observations = append(preparation.Observations, extraObservation)
				input.SharedOriginalIDs = nil
				input.CarouselOriginalIDs, input.DetailOriginalIDs = []string{"source-1"}, []string{"source-2"}
				if name == "two_originals_in_same_group" {
					input.CarouselOriginalIDs = append(input.CarouselOriginalIDs, extra.ID)
				}
			}

			prepared, err := s.PrepareImageSet(ctx, input)
			if name == "two_originals_in_same_group" {
				require.ErrorIs(t, err, imageagent.ErrValidation)
				require.Empty(t, workflows.starts)
				return
			}
			require.NoError(t, err)
			require.Equal(t, 3, prepared.Images)
			require.Zero(t, prepared.Points)
			require.Empty(t, workflows.starts)
			confirmed, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
			require.NoError(t, err)
			require.NoError(t, imageagent.ValidateImageSetAdmission(confirmed.Run, confirmed.Plan))
			require.Len(t, workflows.starts, 1)
			require.Equal(t, agentconfig.ImageRunLimits{UnitVersion: agentconfig.ProviderImageUsageUnit, Images: 3, ElapsedSeconds: 3600},
				confirmed.Run.ImageAdmission.Command.Limits)
			require.Equal(t, 3, confirmed.Run.Budget.MaxModelCalls)
			require.Equal(t, imageagent.SlotRoleMain, confirmed.Plan.Slots[0].Role)
			require.Equal(t, "carousel", confirmed.Plan.Slots[1].Recipe.Placement.Group)
			require.Equal(t, "detail", confirmed.Plan.Slots[2].Recipe.Placement.Group)
			for _, slot := range confirmed.Plan.Slots[:2] {
				require.Equal(t, []string{"source-1"}, slot.SourceAssetIDs)
			}
			detailOriginal := "source-1"
			if name == "separate_group_originals" {
				detailOriginal = "source-2"
			}
			require.Equal(t, []string{detailOriginal}, confirmed.Plan.Slots[2].SourceAssetIDs)
		})
	}
}

func TestPrepareImageSetRejectsMissingOrDifferentCatalogVersion(t *testing.T) {
	for _, version := range []uint64{0, 2} {
		s, _, _, configuration, contexts, quote, ctx, input := imageSetServiceFixture(t)
		catalog := contexts.preparation.Catalog
		catalog.ProductContext.SourceSnapshotVersion = version
		catalog.Manifest.Hash = ""
		catalog, err := imageagent.NormalizeAssetCatalog(catalog)
		require.NoError(t, err)
		contexts.preparation.Catalog = catalog
		contexts.preparation.Source.CatalogHash = catalog.Manifest.Hash
		_, err = s.PrepareImageSet(ctx, input)
		require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
		require.Empty(t, configuration.snapshot.ID)
		require.Empty(t, quote.selections)
	}
}

func TestPrepareImageSetQuotesOnlyExplicitlySelectedTasks(t *testing.T) {
	s, _, workflows, _, _, _, ctx, input := imageSetServiceFixture(t)
	require.NoError(t, json.Unmarshal([]byte(`{"SelectedTaskIDs":["overview"]}`), &input))
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, 1, prepared.Images, "unselected tasks must not acquire a new generation identity or quote")
	require.Equal(t, "overview", prepared.Projection.Plan.Slots[0].ID)
	require.EqualValues(t, 12, prepared.Points)
	require.Zero(t, workflows.starts)
}

func TestPrepareImageSetFreezesExplicitRouteAndConflictsOnChangedSelection(t *testing.T) {
	s, _, _, _, _, quote, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(&methodFixture{text: "A clean detail photograph."}))
	ref := imageMethodRef()
	input.Method = &ref
	quote.quote.Model, quote.quote.Quality = "gpt-image-2-vip", "medium"
	quote.quote.ConfigurationVersion, quote.quote.PriceVersion = "route-vip", "route-vip"
	quote.quote.Billing = syntheticImageBilling(quote.quote.Model, quote.quote.PriceVersion, quote.quote.Points)
	require.NoError(t, json.Unmarshal([]byte(`{"Route":{"BindingID":"image-vip","RevisionID":"route-vip"}}`), &input))
	p, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	raw, err := json.Marshal(p.Projection.Plan.Set)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &fields))
	require.JSONEq(t, `{"BindingID":"image-vip","RevisionID":"route-vip"}`, string(fields["Route"]))
	require.Equal(t, ref, p.Projection.Plan.Set.Method.Skill)
	_, err = s.ConfirmImagePlan(ctx, confirmSetInput(p))
	require.NoError(t, err)
	require.Len(t, quote.selections, 2)
	require.Equal(t, p.Projection.Plan.Set.Route, quote.selections[0])
	require.Equal(t, quote.selections[0], quote.selections[1])
	quote.err = imageagent.ErrBudgetQuoteUnavailable
	_, err = s.ConfirmImagePlan(ctx, confirmSetInput(p))
	require.NoError(t, err, "original admission restores without choosing a new route")
	require.Len(t, quote.selections, 2)
	require.NoError(t, json.Unmarshal([]byte(`{"Route":{"BindingID":"image-vip","RevisionID":"route-other"}}`), &input))
	_, err = s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
}

func TestNewImageSetPromptContainsVisualFactsWithoutInternalIdentityFields(t *testing.T) {
	s, _, _, _, source, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	for _, slot := range prepared.Projection.Plan.Slots {
		prompt := slot.Recipe.Prompt
		require.NotContains(t, prompt, `"ProductID":`, "internal identity belongs in the frozen plan, not the image content")
		require.NotContains(t, prompt, `"SourceSnapshotVersion":`)
		require.Contains(t, prompt, source.preparation.Catalog.ProductContext.Title)
		require.Contains(t, prompt, slot.Recipe.Purpose)
		require.Contains(t, prompt, slot.Recipe.Background)
		require.Contains(t, prompt, slot.Recipe.Language)
	}
	source.preparation.Catalog.ProductContext.Title = "changed later"
	replay, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, prepared.Projection.Plan, replay.Projection.Plan, "a replay must preserve the original frozen prompt")
}

func TestRegenerationRequiresTheOriginalRunToHaveKnownClosedEffects(t *testing.T) {
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	first, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	input.RequestID = "3b9ef503-8c6c-4a76-8eb8-047213c565cf"
	raw, err := json.Marshal(map[string]any{"RegenerateFromRunID": first.Projection.Run.ID, "SelectedTaskIDs": []string{"overview"}})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &input))
	_, err = s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked, "an unfinished or UNKNOWN original request cannot be replaced by another dispatch")
}

func TestKnownClosedSubsetRegenerationCreatesANewAwaitingConfirmationRun(t *testing.T) {
	s, repo, workflows, config, _, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	parent, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.NoError(t, err)
	parent.Run.Status = imageagent.RunStatusBlocked
	parent.Run.CurrentNode = "approve-results"
	parent.Run.Version++
	mutations := []imageagent.SlotProjectionMutation{}
	for i := range parent.Slots {
		parent.Slots[i].Slot.Status = imageagent.SlotStatusBlocked
		parent.Slots[i].ErrorCode = "budget_exceeded"
		parent.Slots[i].Closure = &imageagent.ImageSlotClosure{UnitVersion: "ai-micropoint-v1", Kind: "not_dispatched"}
		slot := parent.Slots[i]
		mutations = append(mutations, imageagent.SlotProjectionMutation{PlanRevision: parent.Plan.Revision, Result: imageagent.SlotResult{SlotID: slot.Slot.ID, Status: slot.Slot.Status, ErrorCode: slot.ErrorCode, Closure: slot.Closure}, Projection: slot, Attempt: imageagent.StepAttempt{TenantID: parent.Run.TenantID, OwnerUserID: parent.Run.UserID, RunID: parent.Run.ID, PlanRevision: parent.Plan.Revision, SlotID: slot.Slot.ID, Node: "closed", IdempotencyKey: "closed:" + slot.Slot.ID, Outcome: "blocked", ErrorCategory: slot.ErrorCode}})
	}
	closedDigest, err := imageagent.ImageSetClosedEffectsDigest(parent.Plan, parent.Slots, nil)
	require.NoError(t, err)
	_, err = repo.CommitProjection(ctx, imageagent.ProjectionCommit{Scope: imageagent.ScopeForRun(parent.Run), CommitID: "controlled-closed-parent", ExpectedProjectionVersion: parent.ProjectionVersion, ExpectedRunVersion: parent.Run.Version - 1, Snapshot: parent, SlotMutations: mutations, RunMutation: &imageagent.RunMutation{Status: parent.Run.Status, CurrentNode: parent.Run.CurrentNode, ActivePlanRevision: parent.Plan.Revision}, EventType: "run.blocked", EventPayload: json.RawMessage(`{}`)})
	require.NoError(t, err)
	input.RequestID = "3b9ef503-8c6c-4a76-8eb8-047213c565cf"
	input.RegenerateFromRunID = parent.Run.ID
	input.SelectedTaskIDs = []string{"overview"}
	newRun, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.NotEqual(t, parent.Run.ID, newRun.Projection.Run.ID)
	require.Equal(t, imageagent.RunStatusAwaitingPlanApproval, newRun.Projection.Run.Status)
	require.Len(t, newRun.Projection.Plan.Slots, 1)
	require.EqualValues(t, 12, newRun.Points)
	require.Equal(t, &imageagent.ImageSetRegeneration{RunID: parent.Run.ID, ClosedEffectsDigest: closedDigest}, newRun.Projection.Plan.Set.Regeneration)
	require.Len(t, workflows.starts, 1, "the new run cannot start before its own point confirmation")
	require.Equal(t, 1, config.admissions)
	stored, err := repo.GetProjection(ctx, imageagent.ScopeForRun(parent.Run))
	require.NoError(t, err)
	require.Equal(t, parent.ResultDigest, stored.ResultDigest, "original effects and receipts remain untouched")
}

func imageSetServiceFixture(t *testing.T, options ...imageagent.ServiceOption) (*imageagent.Service, imageagent.Repository, *recordingWorkflowClient, *setConfigurationFixture, *setContextFixture, *setQuoteFixture, context.Context, imageagent.PrepareImageSetInput) {
	return imageSetServiceFixtureWithRepository(t, nil, options...)
}
func imageSetServiceFixtureWithRepository(t *testing.T, wrap func(imageagent.Repository) imageagent.Repository, options ...imageagent.ServiceOption) (*imageagent.Service, imageagent.Repository, *recordingWorkflowClient, *setConfigurationFixture, *setContextFixture, *setQuoteFixture, context.Context, imageagent.PrepareImageSetInput) {
	t.Helper()
	config := &setConfigurationFixture{enabled: true, template: agentconfig.SetTemplate{Schema: agentconfig.ImageConfigurationSchema, Mode: "standard", ShareOriginals: true, Background: "white", Language: "zh", Carousel: []agentconfig.ContentTask{{ID: "identity", Purpose: "product_identity"}}, Detail: []agentconfig.ContentTask{{ID: "overview", Purpose: "product_overview"}}}}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{ProductContext: imageagent.ProductContextRef{ProductID: "product", Title: "Test product", SourceSnapshotVersion: 1}, Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://images.example.org/source.png", Width: 1024, Height: 1024}}})
	require.NoError(t, err)
	contexts := &setContextFixture{preparation: imageagent.ImageSetPreparation{Source: imageagent.ImageSourceBinding{ContextKind: imageagent.ImageSourceAcquisition, ProductID: "product", OperationID: "operation", OriginalPublicationID: "publication", OriginalVersion: 1, EffectiveVersion: 1, CatalogHash: catalog.Manifest.Hash}, Target: imageagent.ImageTarget{Platform: "product"}, Catalog: catalog, Observations: []imageagent.ImageSourceObservation{{AssetID: "source-1", SHA256: strings.Repeat("c", 64), Bytes: 10, Width: 1024, Height: 1024, MediaType: "image/png"}}}}
	quotes := &setQuoteFixture{quote: imageagent.ImageGenerationQuote{Billing: syntheticImageBilling("gpt-image-2.5", "price", 12), Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price", Points: 12, RouteReference: "route", CredentialReference: "credential", ConfigurationVersion: "config"}}
	repository := store.NewMemoryRepository()
	if wrap != nil {
		repository = wrap(repository)
	}
	workflows := &recordingWorkflowClient{}
	options = append([]imageagent.ServiceOption{imageagent.WithOrganizationScope(), imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{ConfirmedCopy: contexts, Configuration: config, Contexts: contexts, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 32, Points: 10000, ElapsedSeconds: 3600}})}, options...)
	service, err := imageagent.NewService(repository, workflows, staticCatalogResolver{catalog: catalog}, options...)
	require.NoError(t, err)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member"})
	input := imageagent.PrepareImageSetInput{Text: &imageagent.ImageTextSelection{Mode: "none"}, ExpectedModeRevision: "0", ContextKind: imageagent.ImageSourceAcquisition, RequestID: "c5fd7c43-138c-43dc-8dfb-6f9be74a5e9e", ContextID: "operation", Target: imageagent.ImageTargetSelection{Platform: "product"}, SharedOriginalIDs: []string{"source-1"}}
	return service, repository, workflows, config, contexts, quotes, ctx, input
}

func confirmSetInput(prepared imageagent.PreparedImageSet) imageagent.ConfirmImagePlanInput {
	return imageagent.ConfirmImagePlanInput{RunID: prepared.Projection.Run.ID, ActionID: "b912e7d4-df80-44a5-8510-3cdf91d5b8dd", ExpectedRevision: 1, PlanDigest: prepared.PlanDigest, QuoteDigest: prepared.QuoteDigest}
}

func TestConfirmImageSetRechecksTenantAdmissionButRecoversOriginalReceipts(t *testing.T) {
	for _, stage := range []string{"new", "run_receipt", "configuration_receipt"} {
		t.Run(stage, func(t *testing.T) {
			var wrap func(imageagent.Repository) imageagent.Repository
			if stage == "configuration_receipt" {
				wrap = func(r imageagent.Repository) imageagent.Repository {
					return &confirmCommitFailure{Repository: r, fail: true}
				}
			}
			service, repo, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixtureWithRepository(t, wrap)
			prepared, err := service.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			command := confirmSetInput(prepared)
			if stage != "new" {
				_, err = service.ConfirmImagePlan(ctx, command)
				if stage == "configuration_receipt" {
					require.ErrorContains(t, err, "confirm persistence unavailable")
				} else {
					require.NoError(t, err)
				}
			}
			starts := len(workflows.starts)
			restarted, err := imageagent.NewService(repo, workflows, staticCatalogResolver{catalog: contexts.preparation.Catalog},
				imageagent.WithOrganizationScope(),
				imageagent.WithTenantStartGate(imageagent.TenantAllowlistStartGate{Enabled: true, AllowedTenantIDs: []string{"another-org"}}),
				imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{Configuration: config, Contexts: contexts, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 32, Points: 10000, ElapsedSeconds: 3600}}))
			require.NoError(t, err)
			resumed, err := restarted.ConfirmImagePlan(ctx, command)
			if stage == "new" {
				require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
				require.Zero(t, config.admissions)
				require.Empty(t, workflows.starts)
				stored, readErr := repo.GetProjection(ctx, imageagent.ScopeForRun(prepared.Projection.Run))
				require.NoError(t, readErr)
				require.Equal(t, imageagent.RunStatusAwaitingPlanApproval, stored.Run.Status)
				require.Nil(t, stored.Run.ImageAdmission)
				return
			}
			require.NoError(t, err)
			require.Equal(t, *config.receipt, *resumed.Run.ImageAdmission)
			require.Equal(t, 1, config.admissions)
			require.Len(t, workflows.starts, starts+1)
			require.Equal(t, command.ActionID, workflows.starts[starts].Run.ImageAdmission.Command.ConfirmActionID)
		})
	}
}

func TestPrepareFullImageSetSharesOnlyOriginalsAndRequiresConfirmation(t *testing.T) {
	s, _, workflows, config, _, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, 2, prepared.Images)
	require.EqualValues(t, 24, prepared.Points)
	require.Equal(t, imageagent.RunStatusAwaitingPlanApproval, prepared.Projection.Run.Status)
	require.Empty(t, workflows.starts)
	require.Zero(t, config.admissions)
	require.Equal(t, prepared.Projection.Plan.Slots[0].Recipe.References, prepared.Projection.Plan.Slots[1].Recipe.References)
	require.NotEqual(t, prepared.Projection.Plan.Slots[0].ID, prepared.Projection.Plan.Slots[1].ID)
	replay, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, prepared, replay)
	input.SharedOriginalIDs = []string{"source-2"}
	_, err = s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
}

func TestConfirmImageSetUsesOriginalReceiptAfterAdmissionOrStartACKLoss(t *testing.T) {
	s, repo, workflows, config, _, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	config.admitACK = errors.New("admission ACK lost")
	workflows.startErr = errors.New("Temporal start ACK lost")
	confirmed, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.ErrorContains(t, err, "start ACK lost")
	require.Equal(t, imageagent.RunStatusExecuting, confirmed.Run.Status)
	require.Equal(t, 1, config.admissions)
	original := *confirmed.Run.ImageAdmission
	config.enabled = false
	workflows.startErr = nil
	replayed, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.NoError(t, err)
	require.Equal(t, original, *replayed.Run.ImageAdmission)
	require.Equal(t, 1, config.admissions)
	require.Len(t, workflows.starts, 2)
	require.Equal(t, workflows.starts[0], workflows.starts[1])
	stored, err := repo.GetProjection(ctx, imageagent.ScopeForRun(confirmed.Run))
	require.NoError(t, err)
	require.Equal(t, &original, stored.Run.ImageAdmission)
}

func TestConfirmImageSetRejectsChangedPriceSourceOrDisabledEnterprise(t *testing.T) {
	for _, kind := range []string{"price", "source", "disabled", "member"} {
		t.Run(kind, func(t *testing.T) {
			s, _, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixture(t)
			prepared, err := s.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			switch kind {
			case "price":
				quotes.quote.Points++
			case "source":
				contexts.revalidationErr = imageagent.ErrRevisionConflict
			case "disabled":
				config.enabled = false
			case "member":
				ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "replacement"})
			}
			_, err = s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
			require.Error(t, err)
			require.Empty(t, workflows.starts)
			require.Zero(t, config.admissions)
		})
	}
}

func TestConfirmImageSetRestoresExpiredOriginalStartWithoutExtendingBudget(t *testing.T) {
	for _, kind := range []string{"run_receipt", "config_receipt"} {
		t.Run(kind, func(t *testing.T) {
			service, repo, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixtureWithRepository(t, func(r imageagent.Repository) imageagent.Repository {
				return &confirmCommitFailure{Repository: r, fail: kind == "config_receipt"}
			})
			prepared, err := service.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			command := confirmSetInput(prepared)
			workflows.startErr = errors.New("Temporal unavailable")
			_, err = service.ConfirmImagePlan(ctx, command)
			require.Error(t, err)
			original := *imageagent.CloneImageAdmission(config.receipt)
			config.enabled = false
			quotes.quote.Points++
			workflows.startErr = nil
			restarted, err := imageagent.NewService(repo, workflows, staticCatalogResolver{catalog: contexts.preparation.Catalog}, imageagent.WithOrganizationScope(), imageagent.WithTenantStartGate(imageagent.TenantAllowlistStartGate{Enabled: true, AllowedTenantIDs: []string{"another-org"}}), imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{Configuration: config, Contexts: contexts, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 32, Points: 10000, ElapsedSeconds: 3600}, Now: func() time.Time { return original.Deadline.Add(time.Second) }}))
			require.NoError(t, err)
			resumed, err := restarted.ConfirmImagePlan(ctx, command)
			require.NoError(t, err, "the original owner must get the expired immutable input to close it without dispatch")
			require.Equal(t, original, *resumed.Run.ImageAdmission)
			require.Equal(t, original.AdmittedAt, resumed.Run.StartedAt)
			require.Equal(t, original.Deadline, resumed.Run.StartedAt.Add(resumed.Run.Budget.MaxElapsed))
			require.Equal(t, 1, config.admissions)
			last := workflows.starts[len(workflows.starts)-1]
			require.Equal(t, original, *last.Run.ImageAdmission)
			require.Equal(t, prepared.Projection.Plan, last.Plan)
			if kind == "run_receipt" {
				require.Len(t, workflows.starts, 2)
				require.Equal(t, workflows.starts[0], workflows.starts[1])
			} else {
				require.Len(t, workflows.starts, 1)
			}
		})
	}
}

func TestExpiredImageSetConfirmationDoesNotReviveClosedWorkflowProgress(t *testing.T) {
	for _, status := range []imageagent.RunStatus{imageagent.RunStatusBlocked, imageagent.RunStatusFailed} {
		t.Run(string(status), func(t *testing.T) {
			service, repo, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixture(t)
			prepared, err := service.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			command := confirmSetInput(prepared)
			current, err := service.ConfirmImagePlan(ctx, command)
			require.NoError(t, err)
			next := current
			next.Run.Status, next.Run.Version = status, current.Run.Version+1
			scope := imageagent.ScopeForRun(current.Run)
			original, err := repo.CommitProjection(ctx, imageagent.ProjectionCommit{Scope: scope, CommitID: "workflow-progress", ExpectedProjectionVersion: current.ProjectionVersion, ExpectedRunVersion: current.Run.Version, Snapshot: next, RunMutation: &imageagent.RunMutation{Status: status, CurrentNode: next.Run.CurrentNode, ActivePlanRevision: next.Plan.Revision}, EventType: "run.updated", EventPayload: []byte(`{}`)})
			require.NoError(t, err)
			restarted, err := imageagent.NewService(repo, workflows, staticCatalogResolver{catalog: contexts.preparation.Catalog}, imageagent.WithOrganizationScope(), imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{Configuration: config, Contexts: contexts, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 32, Points: 10000, ElapsedSeconds: 3600}, Now: func() time.Time { return config.receipt.Deadline.Add(time.Second) }}))
			require.NoError(t, err)
			resumed, err := restarted.ConfirmImagePlan(ctx, command)
			require.NoError(t, err)
			require.Equal(t, original, resumed)
			require.Len(t, workflows.starts, 1, "confirmation must not start another execution after workflow progress is already closed or blocked")
		})
	}
}

func TestRestartFailedImageSetRestoresOriginalAdmissionAfterSourceRevocation(t *testing.T) {
	for _, revoked := range []bool{false, true} {
		t.Run(map[bool]string{false: "original_after_deadline", true: "source_revoked"}[revoked], func(t *testing.T) {
			service, repo, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixture(t)
			prepared, err := service.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			current, err := service.ConfirmImagePlan(ctx, confirmSetInput(prepared))
			require.NoError(t, err)
			next := current
			next.Run.Status, next.Run.CurrentNode, next.Run.Version = imageagent.RunStatusFailed, "workflow_failed", current.Run.Version+1
			original, err := repo.CommitProjection(ctx, imageagent.ProjectionCommit{Scope: imageagent.ScopeForRun(current.Run), CommitID: "workflow-failed", ExpectedProjectionVersion: current.ProjectionVersion, ExpectedRunVersion: current.Run.Version, Snapshot: next, RunMutation: &imageagent.RunMutation{Status: next.Run.Status, CurrentNode: next.Run.CurrentNode, ActivePlanRevision: next.Plan.Revision}, EventType: "run.failed", EventPayload: []byte(`{}`)})
			require.NoError(t, err)
			config.enabled = false
			quotes.quote.PriceVersion = "current-price-changed"
			if revoked {
				contexts.revalidationErr = imageagent.ErrRevisionConflict
			}
			restarted, err := imageagent.NewService(repo, workflows, staticCatalogResolver{catalog: contexts.preparation.Catalog}, imageagent.WithOrganizationScope(), imageagent.WithTenantStartGate(imageagent.TenantAllowlistStartGate{Enabled: true, AllowedTenantIDs: []string{"another-org"}}), imageagent.WithImageSetDependencies(imageagent.ImageSetDependencies{Configuration: config, Contexts: contexts, Quotes: quotes, HardLimits: agentconfig.ImageRunLimits{UnitVersion: "ai-micropoint-v1", Images: 32, Points: 10000, ElapsedSeconds: 3600}, Now: func() time.Time { return config.receipt.Deadline.Add(time.Second) }}))
			require.NoError(t, err)
			err = restarted.RestartFailed(ctx, original.Run.ID)
			require.NoError(t, err, "the original workflow must close undispatched slots after its live source check")
			require.Len(t, workflows.starts, 2)
			require.Equal(t, original.Run, workflows.starts[1].Run)
			require.Equal(t, original.Plan, workflows.starts[1].Plan)
			require.Equal(t, original.AssetCatalog, workflows.starts[1].AssetCatalog)
			require.Equal(t, workflows.starts[0].Identity, workflows.starts[1].Identity)
			require.Equal(t, 1, config.admissions)
			stored, err := repo.GetProjection(ctx, imageagent.ScopeForRun(original.Run))
			require.NoError(t, err)
			require.Equal(t, original, stored, "restart only restores the original workflow; its owner controls projection transitions")
		})
	}
}

func TestConfirmAdmittedImageSetRestoresOriginalStartAfterSourceChange(t *testing.T) {
	for _, kind := range []string{"run_receipt", "config_receipt"} {
		t.Run(kind, func(t *testing.T) {
			service, repo, workflows, config, contexts, _, ctx, input := imageSetServiceFixtureWithRepository(t, func(r imageagent.Repository) imageagent.Repository {
				return &confirmCommitFailure{Repository: r, fail: kind == "config_receipt"}
			})
			prepared, err := service.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			command := confirmSetInput(prepared)
			workflows.startErr = errors.New("Temporal start unavailable")
			_, err = service.ConfirmImagePlan(ctx, command)
			require.Error(t, err)
			original := *imageagent.CloneImageAdmission(config.receipt)
			contexts.revalidationErr = imageagent.ErrRevisionConflict
			config.enabled = false
			workflows.startErr = nil
			resumed, err := service.ConfirmImagePlan(ctx, command)
			require.NoError(t, err, "source changes cannot prevent recovery of an already admitted immutable input")
			require.Equal(t, &original, resumed.Run.ImageAdmission)
			require.Equal(t, 1, config.admissions)
			last := workflows.starts[len(workflows.starts)-1]
			require.Equal(t, prepared.Projection.Plan, last.Plan)
			require.Equal(t, prepared.Projection.AssetCatalog, last.AssetCatalog)
			require.Equal(t, original, *last.Run.ImageAdmission)
			require.Equal(t, original.Deadline, last.Run.StartedAt.Add(last.Run.Budget.MaxElapsed))
			stored, err := repo.GetProjection(ctx, imageagent.ScopeForRun(resumed.Run))
			require.NoError(t, err)
			require.Equal(t, &original, stored.Run.ImageAdmission)
		})
	}
}

func TestPrepareImageSetMissingSpecificationsBlocksBeforeGeneration(t *testing.T) {
	s, _, workflows, config, _, _, ctx, input := imageSetServiceFixture(t)
	config.template.Detail = []agentconfig.ContentTask{{ID: "dimensions", Purpose: "specification_dimensions"}}
	_, err := s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrValidation)
	require.ErrorContains(t, err, "specifications evidence")
	require.Empty(t, workflows.starts)
}

func TestPrepareImageSetRejectsAResolvedDifferentStore(t *testing.T) {
	s, _, workflows, config, contexts, _, ctx, input := imageSetServiceFixture(t)
	input.Target = imageagent.ImageTargetSelection{Platform: "shein", RecordID: "record", StoreID: "chosen-store", Site: "shein-us", CategoryID: 1}
	contexts.preparation.Target = imageagent.ImageTarget{Platform: "shein", RecordID: "record", StoreID: "different-store", Site: "shein-us", CategoryID: 1, ProductTypeID: 2, ApplicationID: "application", ApplicationMode: "fully_managed", AttributesDigest: strings.Repeat("d", 64), VariantsDigest: strings.Repeat("e", 64), RequirementDigest: strings.Repeat("f", 64), RequirementVersion: "current"}
	contexts.preparation.OfficialPlacements = map[string]imageagent.OfficialImagePlacement{"identity": {Group: "skc", Type: 1, Sort: 1, Site: "shein-us"}, "overview": {Group: "detail", Type: 7, Sort: 1, Site: "shein-us"}}
	_, err := s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
	require.Empty(t, workflows.starts)
	require.Empty(t, config.snapshot.ID)
}

func TestPrepareImageSetBindsExplicitOfficialPositionsToEachConfiguredTask(t *testing.T) {
	for _, mode := range []string{"exact", "changed", "missing", "extra", "generic_claim"} {
		t.Run(mode, func(t *testing.T) {
			s, _, _, configuration, contexts, _, ctx, input := imageSetServiceFixture(t)
			configuration.template.Detail = nil
			position := imageagent.OfficialImagePlacement{Group: "skc", Type: 1, Sort: 1, Site: "shein-us"}
			input.Target = imageagent.ImageTargetSelection{Platform: "shein", RecordID: "record", StoreID: "store", Site: "shein-us", CategoryID: 1}
			input.OfficialPlacements = map[string]imageagent.OfficialImagePlacement{"identity": position}
			contexts.preparation.Target = imageagent.ImageTarget{Platform: "shein", RecordID: "record", StoreID: "store", Site: "shein-us", ApplicationID: "application", ApplicationMode: "self_operated", CategoryID: 1, ProductTypeID: 2, AttributesDigest: strings.Repeat("a", 64), VariantsDigest: strings.Repeat("b", 64), RequirementDigest: strings.Repeat("c", 64), RequirementVersion: "shein-images-v1"}
			contexts.preparation.OfficialPlacements = map[string]imageagent.OfficialImagePlacement{"identity": position}
			switch mode {
			case "changed":
				changed := position
				changed.Type = 2
				input.OfficialPlacements["identity"] = changed
			case "missing":
				input.OfficialPlacements = nil
			case "extra":
				input.OfficialPlacements["unused"] = position
			case "generic_claim":
				input.Target = imageagent.ImageTargetSelection{Platform: "product"}
				contexts.preparation.Target = imageagent.ImageTarget{Platform: "product"}
			}
			prepared, err := s.PrepareImageSet(ctx, input)
			if mode == "exact" {
				require.NoError(t, err)
				require.Equal(t, &position, prepared.Projection.Plan.Slots[0].Recipe.OfficialPlacement)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestPreparedSetCannotBypassConfirmationThroughOriginalStart(t *testing.T) {
	s, _, workflows, _, _, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	err = s.Start(ctx, imageagent.StartRunInput{RunID: "bypass", BusinessTaskID: "operation", TargetPlatform: "product", ImagePolicyContext: imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}, Mode: imageagent.RunModeManual, IdempotencyKey: "bypass", Plan: prepared.Projection.Plan, Budget: prepared.Projection.Run.Budget})
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
	require.Empty(t, workflows.starts)
}

func TestCancelPreparedSetDoesNotNeedTemporalOrReviveOnConfirmation(t *testing.T) {
	s, repo, workflows, config, _, _, ctx, input := imageSetServiceFixture(t)
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.NoError(t, s.Cancel(ctx, prepared.Projection.Run.ID, 1, "cancel-prepared"))
	stored, err := repo.GetProjection(ctx, imageagent.ScopeForRun(prepared.Projection.Run))
	require.NoError(t, err)
	require.Equal(t, imageagent.RunStatusCancelled, stored.Run.Status)
	require.Empty(t, workflows.cancellations)
	_, err = s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
	require.Zero(t, config.admissions)
}

type confirmCommitFailure struct {
	imageagent.Repository
	fail bool
}

func (r *confirmCommitFailure) CommitProjection(ctx context.Context, commit imageagent.ProjectionCommit) (imageagent.RunProjection, error) {
	if r.fail && strings.HasPrefix(commit.CommitID, "confirm:") {
		r.fail = false
		return imageagent.RunProjection{}, errors.New("confirm persistence unavailable")
	}
	return r.Repository.CommitProjection(ctx, commit)
}
func TestConfirmConsumesTheOriginalAdmissionAfterImageCommitFailureAndCurrentPriceChange(t *testing.T) {
	service, repo, workflows, config, _, quotes, ctx, input := imageSetServiceFixtureWithRepository(t, func(r imageagent.Repository) imageagent.Repository {
		return &confirmCommitFailure{Repository: r, fail: true}
	})
	prepared, err := service.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	command := confirmSetInput(prepared)
	_, err = service.ConfirmImagePlan(ctx, command)
	require.ErrorContains(t, err, "confirm persistence unavailable")
	require.Equal(t, 1, config.admissions)
	require.Empty(t, workflows.starts)
	original := *config.receipt
	projection, err := repo.GetProjection(ctx, imageagent.ScopeForRun(prepared.Projection.Run))
	require.NoError(t, err)
	require.Nil(t, projection.Run.ImageAdmission)
	config.enabled = false
	quotes.quote.Points++
	resumed, err := service.ConfirmImagePlan(ctx, command)
	require.NoError(t, err)
	require.Equal(t, original, *resumed.Run.ImageAdmission)
	require.Equal(t, 1, config.admissions)
	require.Len(t, workflows.starts, 1)
	changed := command
	changed.ActionID = "731f0fcf-39ac-4f7b-9057-16fef9b20cf4"
	_, err = service.ConfirmImagePlan(ctx, changed)
	require.Error(t, err)
	require.Len(t, workflows.starts, 1)
}
func TestOwnedColorImageSetFreezesNineTasksAndAdmission(t *testing.T) {
	assertOwnedColorImageSet(t, nil, 9)
}

func TestOwnedColorImageSetUsesSelectedTemplateTasks(t *testing.T) {
	assertOwnedColorImageSet(t, []string{"overview", "identity", "scene", "closeup"}, 12)
}

func TestColorImageReworkPlanKeepsOneExactSlotAndBindsComposition(t *testing.T) {
	service, _, config, ctx, input := ownedColorSetFixture(t)
	config.template.Detail = append(config.template.Detail, agentconfig.ContentTask{ID: "structure", Purpose: "structure_functions"})
	prepared, err := service.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, 12, prepared.Images)
	parent := prepared.Projection
	slot := parent.Plan.Slots[len(parent.Plan.Slots)-1]
	require.Equal(t, "White", slot.Recipe.Placement.Color)
	require.Equal(t, 2, slot.Recipe.Placement.Order)
	plan := parent.Plan
	plan.Set = imageagent.CloneImageSetPlan(plan.Set)
	plan.Slots = []imageagent.Slot{slot}
	plan.SourceAssetIDs = slot.SourceAssetIDs
	digest := strings.Repeat("a", 64)
	plan.Set.Regeneration = &imageagent.ImageSetRegeneration{RunID: parent.Run.ID, ClosedEffectsDigest: digest, ResultDigest: digest,
		ColorRework: &imageagent.ColorImageReworkBinding{ParentRevision: 1, SlotID: slot.ID, Attempt: 1, CandidateID: "old-candidate",
			ArtifactHash: digest, ReplacedAssetID: "old-selected", Count: 12,
			Head: productasset.ImageInventoryHead{ActionID: "a5dfaa12-be4e-4781-a549-0cdfda388b27", PayloadHash: digest}}}
	plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(plan)
	require.NoError(t, err)
	require.NoError(t, imageagent.ValidateImageSetPlan(plan))
	quote := plan.Set.QuoteDigest
	plan.Slots[0].Recipe = imageagent.CloneImageSlotRecipe(plan.Slots[0].Recipe)
	plan.Slots[0].Recipe.Composition = "Show the white coat without a model."
	plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(plan)
	require.NoError(t, err)
	require.NotEqual(t, quote, plan.Set.QuoteDigest)
}

func assertOwnedColorImageSet(t *testing.T, selected []string, count int) {
	t.Helper()
	service, workflows, config, ctx, input := ownedColorSetFixture(t)
	if selected != nil {
		config.template.Detail = append(config.template.Detail,
			agentconfig.ContentTask{ID: "closeup", Purpose: "detail_closeup"},
			agentconfig.ContentTask{ID: "benefits", Purpose: "key_benefits"})
	}
	input.SelectedTaskIDs = selected
	prepared, err := service.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, count, prepared.Images)
	require.Zero(t, prepared.Points)
	require.Empty(t, workflows.starts)
	for _, slot := range prepared.Projection.Plan.Slots {
		color := slot.Recipe.Placement.Color
		require.Contains(t, slot.Recipe.Prompt, "Selected color: "+color)
		require.NotContains(t, slot.Recipe.Prompt, "Confirmed color: mixed")
		require.Equal(t, []string{input.ColorImages.OriginalFor(color)}, slot.SourceAssetIDs)
		require.NotEqual(t, "key_benefits", slot.Recipe.Purpose)
	}
	if selected != nil {
		for _, group := range input.ColorImages.Groups {
			purposes := []string{}
			for _, slot := range prepared.Projection.Plan.Slots {
				if slot.Recipe.Placement.Color == group.Color {
					purposes = append(purposes, slot.Recipe.Purpose)
				}
			}
			require.Equal(t, []string{"product_identity", "usage_scene", "product_overview", "detail_closeup"}, purposes)
		}
		changed := input
		changed.SelectedTaskIDs = []string{"identity"}
		_, err = service.PrepareImageSet(ctx, changed)
		require.ErrorIs(t, err, imageagent.ErrRevisionConflict, "the same request cannot silently change its selected tasks")
		incomplete := prepared.Projection.Plan
		incomplete.Slots = incomplete.Slots[:len(incomplete.Slots)-1]
		incomplete.Set = imageagent.CloneImageSetPlan(incomplete.Set)
		incomplete.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(incomplete)
		require.NoError(t, err)
		require.ErrorIs(t, imageagent.ValidateImageSetPlan(incomplete), imageagent.ErrValidation)
		mismatched := prepared.Projection.Plan
		mismatched.Slots = append([]imageagent.Slot(nil), mismatched.Slots...)
		last := &mismatched.Slots[len(mismatched.Slots)-1]
		last.Recipe = imageagent.CloneImageSlotRecipe(last.Recipe)
		last.Recipe.Placement.Order++
		require.ErrorIs(t, imageagent.ValidateImageSetPlan(mismatched), imageagent.ErrValidation)
	}
	confirmed, err := service.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.NoError(t, err)
	require.Len(t, workflows.starts, 1)
	require.Equal(t, count, confirmed.Run.Budget.MaxModelCalls)
	require.NoError(t, imageagent.ValidateImageSetAdmission(confirmed.Run, confirmed.Plan))
	for _, slot := range confirmed.Plan.Slots {
		execution, err := imageagent.ImageSetSlotExecutionInput(confirmed, slot.ID, 1)
		require.NoError(t, err)
		require.NoError(t, imageagent.ValidateImageSetExecution(execution))
		digest, err := imageagent.ImageSlotGenerationInputDigest(execution)
		require.NoError(t, err)
		require.Len(t, digest, 64)
		execution.Slot.Recipe.Placement.Color = "Unselected"
		require.Error(t, imageagent.ValidateImageSetExecution(execution))
	}
}

func TestOwnedColorImageSetRejectsInvalidTemplateSelections(t *testing.T) {
	for _, selected := range [][]string{{}, {"unknown"}, {"identity", "identity"}} {
		service, workflows, _, ctx, input := ownedColorSetFixture(t)
		input.SelectedTaskIDs = selected
		_, err := service.PrepareImageSet(ctx, input)
		require.ErrorIs(t, err, imageagent.ErrValidation)
		require.Empty(t, workflows.starts)
	}
	service, workflows, config, ctx, input := ownedColorSetFixture(t)
	config.template.Mode = "custom"
	config.template.Detail = nil
	config.template.Carousel = nil
	for i := 0; i < 11; i++ {
		task := agentconfig.ContentTask{ID: fmt.Sprintf("custom-%d", i), Purpose: "custom", Brief: "Show the product"}
		config.template.Carousel = append(config.template.Carousel, task)
		input.SelectedTaskIDs = append(input.SelectedTaskIDs, task.ID)
	}
	_, err := service.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrValidation)
	require.Empty(t, workflows.starts)
}

func ownedColorSetFixture(t *testing.T) (*imageagent.Service, *recordingWorkflowClient, *setConfigurationFixture, context.Context, imageagent.PrepareImageSetInput) {
	t.Helper()
	service, _, workflows, config, contexts, quotes, ctx, input := imageSetServiceFixture(t)
	input = configureOwnedColorSet(t, config, contexts, quotes, input)
	return service, workflows, config, ctx, input
}

func configureOwnedColorSet(t *testing.T, config *setConfigurationFixture, contexts *setContextFixture, quotes *setQuoteFixture, input imageagent.PrepareImageSetInput) imageagent.PrepareImageSetInput {
	t.Helper()
	input.ContextKind = imageagent.ImageSourceListing
	input.Route = &imageagent.ImageRouteSelection{BindingID: strings.Repeat("b", 64), RevisionID: strings.Repeat("c", 64)}
	input.Template = &agentconfig.TemplateRef{TemplateID: "c5fd7c43-138c-43dc-8dfb-6f9be74a5e9e", Revision: "1"}
	input.SharedOriginalIDs = nil
	config.template.Carousel = append(config.template.Carousel, agentconfig.ContentTask{ID: "scene", Purpose: "usage_scene"})
	p := &contexts.preparation
	p.Source.ContextKind = input.ContextKind
	p.OwnProduct = true
	p.Catalog.ProductContext.Attributes = map[string]string{"color": "mixed"}
	option := imageagent.ImageColorOption{Name: "Shade"}
	selection := imageagent.ColorImageSelection{Option: option.Name}
	for i, name := range []string{"Black", "Gray", "White"} {
		original := p.Catalog.Assets[0]
		observation := p.Observations[0]
		if i > 0 {
			original.ID = "source-" + []string{"unused", "z", "a"}[i]
			original.URL = "https://images.example.org/" + name + ".png"
			observation.AssetID = original.ID
			p.Catalog.Assets = append(p.Catalog.Assets, original)
			p.Observations = append(p.Observations, observation)
		}
		option.Groups = append(option.Groups, imageagent.ImageColorGroup{Value: name, OriginalIDs: []string{original.ID}})
		selection.Groups = append(selection.Groups, imageagent.ColorImageOriginal{Color: name, OriginalID: original.ID})
	}
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{ProductContext: p.Catalog.ProductContext, Assets: p.Catalog.Assets})
	require.NoError(t, err)
	p.Catalog = catalog
	p.Source.CatalogHash = catalog.Manifest.Hash
	p.ColorOptions = []imageagent.ImageColorOption{option}
	input.ColorImages = &selection
	quotes.quote = imageagent.GenerationProviderMetadata{Provider: "grsai", Model: aicapability.GRSaiProductImageModel,
		Protocol: aicapability.GRSaiProductImageProtocol, BillingMode: aicapability.ModelUsageOnly,
		ConfigurationVersion: input.Route.RevisionID, RouteReference: input.Route.BindingID,
		CredentialReference: "grsai-key", ConnectionID: "grsai-connection"}.ImageQuote()
	return input
}
