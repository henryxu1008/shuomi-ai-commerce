package routeconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func TestChooseTextOnlyUsesOriginalCurrentAndPreservesOtherPurposes(t *testing.T) {
	ctx := context.Background()
	repo := &sharedRepositoryFixture{}
	catalog := &sharedCatalogFixture{}
	service := &SharedModelService{Repository: repo, Catalog: catalog}
	ref := sharedChoiceFixture().Text
	q, err := service.qualify(ctx, "org-a", ModelCategoryText, ref)
	require.NoError(t, err)
	token, err := q.TextChoiceToken()
	require.NoError(t, err)
	revision, err := service.ChooseText(ctx, "org-a", "actor", "select", TextChoiceInput{ExpectedRevision: NoRevision, Reference: ref, Quote: token})
	require.NoError(t, err)
	require.Equal(t, ref, revision.Payload.SharedModels.Text)
	require.Empty(t, revision.Payload.SharedModels.Image)
	require.Empty(t, revision.Payload.SharedModels.Multimodal)
	_, _, err = service.ReadCategory(ctx, "org-a", ModelCategoryImage)
	require.ErrorIs(t, err, ErrNotFound)

	original := sharedChoiceFixture()
	before, err := original.CategoryDigest(ModelCategoryMultimodal)
	require.NoError(t, err)
	repo.current.Payload.SharedModels = &original
	revision, err = service.ChooseText(ctx, "org-a", "actor", "choose-again", TextChoiceInput{ExpectedRevision: revision.ID, Reference: ref, Quote: token})
	require.NoError(t, err)
	after, err := revision.Payload.SharedModels.CategoryDigest(ModelCategoryMultimodal)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, original.Image, revision.Payload.SharedModels.Image)
	require.Equal(t, original.DefaultChannel, revision.Payload.SharedModels.DefaultChannel)
	_, err = service.ChooseText(ctx, "org-a", "actor", "bad-quote", TextChoiceInput{ExpectedRevision: revision.ID, Reference: ref, Quote: "stale-quote"})
	require.ErrorIs(t, err, ErrConflict)
}

func TestPurposeSelectionKeepsImageFirstSetupAvailableWithoutText(t *testing.T) {
	ctx := context.Background()
	repo := &sharedRepositoryFixture{}
	service := &SharedModelService{Repository: repo, Catalog: &sharedCatalogFixture{unready: ModelCategoryText}, PurposeSelectionOnly: true}
	image := SharedModelSelection{SchemaVersion: 2, UnitVersion: aicapability.AIPointMicrounitV1, Image: sharedChoiceFixture().Image}
	r, err := service.Publish(ctx, "org-a", "actor", "image-first", SharedModelInput{ExpectedRevision: NoRevision, Selection: image})
	require.NoError(t, err)
	require.Empty(t, r.Payload.SharedModels.Text)
	_, _, err = service.ReadCategory(ctx, "org-a", ModelCategoryText)
	require.ErrorIs(t, err, ErrNotFound)
	q, _, err := service.ReadCategory(ctx, "org-a", ModelCategoryImage)
	require.NoError(t, err)
	require.True(t, q.Ready)
}
