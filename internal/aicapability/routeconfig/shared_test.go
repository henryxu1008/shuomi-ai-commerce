package routeconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func sharedChoiceFixture() SharedModelSelection {
	return SharedModelSelection{SchemaVersion: 1, UnitVersion: aicapability.AIPointMicrounitV1, DefaultChannel: "openai-direct",
		Text:       ModelBindingReference{ID: "mini", Version: "binding-v1"},
		Image:      ModelBindingReference{ID: "flare", Version: "binding-v1"},
		Multimodal: ModelBindingReference{ID: "mini", Version: "binding-v1"}}
}

type sharedCatalogFixture struct {
	unready ModelCategory
	reads   []ModelCategory
	foreign bool
}

func (*sharedCatalogFixture) ValidateDefaultChannel(_ context.Context, _ string, channel string) error {
	if channel != "openai-direct" {
		return ErrNeedsConfiguration
	}
	return nil
}

func (c *sharedCatalogFixture) QualifyModel(_ context.Context, org string, category ModelCategory, ref ModelBindingReference) (ModelQualification, error) {
	c.reads = append(c.reads, category)
	price := aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "price", FXVersion: "fx", Currency: "USD", CNYMicrosPerCurrencyUnit: 7_000_000, Model: ref.ID, Channel: "openai-direct", UsageVersion: "synthetic", Rates: aicapability.OfficialModelRates{InputText: 750_000, OutputText: 4_500_000}}
	if c.foreign {
		org = "other-org"
	}
	return syntheticQualifiedModel(ModelQualification{OrganizationID: org, Category: category, Reference: ref, Channel: "openai-direct", Model: ref.ID, Ready: category != c.unready, Reason: "BUDGET_BOUND_MISSING", BoundEvidenceVersion: "synthetic-bound", ProfileVersion: "synthetic-profile", Price: price}), nil
}

type sharedRepositoryFixture struct {
	Repository
	current Revision
	commits int
}

func (r *sharedRepositoryFixture) Current(_ context.Context, scope Scope) (Revision, error) {
	if r.current.ID == "" {
		return Revision{}, ErrNotFound
	}
	if r.current.Scope != scope {
		return Revision{}, ErrNotFound
	}
	return r.current, nil
}

func (r *sharedRepositoryFixture) Publish(_ context.Context, cmd Command, build func(CredentialReader, string) (Payload, error)) (Revision, error) {
	payload, err := build(nil, "shared-revision")
	if err != nil {
		return Revision{}, err
	}
	r.commits++
	r.current = Revision{ID: "shared-revision", Scope: cmd.Scope, Payload: payload, SchemaVersion: 1, Fingerprint: cmd.Fingerprint, Key: cmd.Key, ActorID: cmd.ActorID}
	return r.current, nil
}

func TestSharedModelSelectionCannotPublishThreeCategoriesPartially(t *testing.T) {
	ctx := context.Background()
	for _, missing := range []ModelCategory{ModelCategoryText, ModelCategoryImage, ModelCategoryMultimodal} {
		repo, catalog := &sharedRepositoryFixture{}, &sharedCatalogFixture{unready: missing}
		service := SharedModelService{Repository: repo, Catalog: catalog}
		input := SharedModelInput{ExpectedRevision: NoRevision, Selection: sharedChoiceFixture()}
		preview, err := service.Preview(ctx, "org-a", input.Selection)
		require.NoError(t, err)
		require.False(t, preview.Ready)
		require.Len(t, preview.Models, 3, "unready Flare identity must remain visible")
		_, err = service.Publish(ctx, "org-a", "actor", "key", input)
		require.ErrorIs(t, err, ErrNeedsConfiguration)
		require.Zero(t, repo.commits, "rejected selection cannot write partial current/revision")
	}
}

func TestSharedModelSelectionPreservesExistingRepositoryAndReadsOnlyRequiredCategory(t *testing.T) {
	ctx := context.Background()
	repo, catalog := &sharedRepositoryFixture{}, &sharedCatalogFixture{}
	service := SharedModelService{Repository: repo, Catalog: catalog}
	input := SharedModelInput{ExpectedRevision: NoRevision, Selection: sharedChoiceFixture()}
	row, err := service.Publish(ctx, "org-a", "actor", "same-key", input)
	require.NoError(t, err)
	require.Equal(t, SharedModelCapability, row.Scope.Capability)
	require.Equal(t, "org-a", row.Scope.OrganizationID)
	require.Equal(t, input.Selection, *row.Payload.SharedModels)

	catalog.reads = nil
	catalog.unready = ModelCategoryImage
	model, digest, err := service.ReadCategory(ctx, "org-a", ModelCategoryText)
	require.NoError(t, err)
	require.Equal(t, "mini", model.Model)
	require.NotEmpty(t, digest)
	require.Equal(t, []ModelCategory{ModelCategoryText}, catalog.reads)
	_, _, err = service.ReadCategory(ctx, "org-a", ModelCategoryImage)
	require.ErrorIs(t, err, ErrNeedsConfiguration)
	_, _, err = service.ReadCategory(ctx, "org-b", ModelCategoryText)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSharedModelSelectionDigestsIgnoreUnrelatedCategoryAndBindExactVersion(t *testing.T) {
	choice := sharedChoiceFixture()
	before, err := choice.CategoryDigest(ModelCategoryText)
	require.NoError(t, err)
	choice.Image.Version = "new-image-binding"
	after, err := choice.CategoryDigest(ModelCategoryText)
	require.NoError(t, err)
	require.Equal(t, before, after)
	choice.Text.Version = "new-text-binding"
	after, err = choice.CategoryDigest(ModelCategoryText)
	require.NoError(t, err)
	require.NotEqual(t, before, after)
	choice.UnitVersion = "whole-point"
	_, err = choice.CategoryDigest(ModelCategoryText)
	require.Error(t, err)
}

func TestSharedModelSelectionRejectsCrossOrganizationAndMixedPayload(t *testing.T) {
	ctx := context.Background()
	repo, catalog := &sharedRepositoryFixture{}, &sharedCatalogFixture{foreign: true}
	service := SharedModelService{Repository: repo, Catalog: catalog}
	_, err := service.Publish(ctx, "org-a", "actor", "key", SharedModelInput{ExpectedRevision: NoRevision, Selection: sharedChoiceFixture()})
	require.Error(t, err)
	require.Zero(t, repo.commits)

	selection := sharedChoiceFixture()
	repo.current = Revision{ID: "mixed", SchemaVersion: 1, Scope: Scope{OrganizationID: "org-a", Capability: SharedModelCapability}, Payload: Payload{SharedModels: &selection, Text: testBinding().Text}}
	_, _, err = service.ReadCategory(ctx, "org-a", ModelCategoryText)
	require.Error(t, err, "shared selection must not reinterpret an operation profile")
}

func TestSharedModelSelectionDefaultChannelMustBelongToTrustedCatalog(t *testing.T) {
	repo := &sharedRepositoryFixture{}
	service := SharedModelService{Repository: repo, Catalog: &sharedCatalogFixture{}}
	input := SharedModelInput{ExpectedRevision: NoRevision, Selection: sharedChoiceFixture()}
	input.Selection.DefaultChannel = "caller-invented-channel"
	_, err := service.Publish(context.Background(), "org-a", "actor", "key", input)
	require.ErrorIs(t, err, ErrNeedsConfiguration)
	require.Zero(t, repo.commits)
}
