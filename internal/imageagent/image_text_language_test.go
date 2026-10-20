package imageagent_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
)

type frozenLanguageReplayRepository struct {
	imageagent.Repository
	projection *imageagent.RunProjection
}

func (r *frozenLanguageReplayRepository) GetProjection(ctx context.Context, scope imageagent.RunScope) (imageagent.RunProjection, error) {
	if r.projection != nil {
		return *r.projection, nil
	}
	return r.Repository.GetProjection(ctx, scope)
}

func TestHistoricalPrepareWithoutTextReplaysFrozenBytes(t *testing.T) {
	for _, policy := range []string{"", imageagent.FrozenVisualBriefPolicyVersion, imageagent.FrozenLiteralCopyPolicyVersion} {
		t.Run(policy, func(t *testing.T) {
			var repo *frozenLanguageReplayRepository
			service, _, _, _, owner, _, ctx, input := imageSetServiceFixtureWithRepository(t, func(r imageagent.Repository) imageagent.Repository {
				repo = &frozenLanguageReplayRepository{Repository: r}
				return repo
			})
			prepared, err := service.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			frozen := prepared.Projection
			input.Text = nil
			frozen.Plan.Set.Text = nil
			if policy != "" {
				ref := imageMethodRef()
				input.Method = &ref
				frozen.Plan.Set.Method = &imageagent.ImageMethodBinding{Skill: ref, PolicyVersion: policy}
			}
			wire, err := json.Marshal(input)
			require.NoError(t, err)
			require.NotContains(t, string(wire), `"Text"`)
			if policy != "" {
				wire, err = json.Marshal(struct {
					Input        imageagent.PrepareImageSetInput
					MethodPolicy string
				}{input, policy})
				require.NoError(t, err)
			}
			sum := sha256.Sum256(wire)
			frozen.Plan.Set.InputDigest = hex.EncodeToString(sum[:])
			frozen.Plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(frozen.Plan)
			require.NoError(t, err)
			repo.projection = &frozen
			before, err := json.Marshal(frozen)
			require.NoError(t, err)
			for i := 0; i < 2; i++ {
				result, err := service.PrepareImageSet(ctx, input)
				require.NoError(t, err)
				after, err := json.Marshal(result.Projection)
				require.NoError(t, err)
				require.Equal(t, string(before), string(after))
			}
			require.Zero(t, owner.copyReads, "old request recovery cannot translate or fabricate consent")
		})
	}
}

func TestNewImagePreparationRequiresExplicitTextChoice(t *testing.T) {
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	require.NoError(t, json.Unmarshal([]byte(`{"Text":null}`), &input))
	_, err := s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrValidation, "fresh plans must not silently reuse source-language copy")
}

func TestReviewedCopyFreezesTargetLanguageWithoutChangingCatalog(t *testing.T) {
	for _, bad := range []string{"", "language", "source", "publication", "revision", "too_long", "unknown"} {
		t.Run(bad, func(t *testing.T) {
			s, _, _, config, contexts, _, ctx, input := imageSetServiceFixture(t)
			config.template.Language = "en"
			ref := imageagent.ImageTextReviewReference{ID: "a3ce4c41-745a-47bf-a418-c940d3ba4b84", Revision: 2}
			contexts.preparation.EffectivePublicationID = "publication"
			contexts.copy = &imageagent.ImageTextReviewCopy{Review: ref, ProductID: "product", PublicationID: "publication", Version: 1, Language: "en", OriginRunID: "71288b77-f3e0-4cb6-a9ad-c4bf3a7d21e3", ModeSnapshot: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: ref.ID, Digest: strings.Repeat("c", 64)}, Lines: []string{"DRONA fabric box", "Size: 33 x 38 x 33 cm"}}
			input.Text = &imageagent.ImageTextSelection{Mode: "reviewed", Review: &ref, TaskIDs: []string{"overview"}}
			switch bad {
			case "language":
				contexts.copy.Language = "zh"
			case "source":
				contexts.copy.ProductID = "other"
			case "publication":
				contexts.copy.PublicationID = "other"
			case "revision":
				contexts.copy.Review.Revision = 3
			case "too_long":
				contexts.copy.Lines[0] = strings.Repeat("x", 33)
			case "unknown":
				contexts.copyErr = imageagent.ErrCommandBlocked
			}
			originalHash := contexts.preparation.Catalog.Manifest.Hash
			p, err := s.PrepareImageSet(ctx, input)
			if bad != "" {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, originalHash, p.Projection.Plan.Set.Source.CatalogHash)
			require.Equal(t, "en", p.Projection.Plan.Set.Text.Language)
			require.Empty(t, p.Projection.Plan.Slots[0].Recipe.VisibleText)
			require.Equal(t, contexts.copy.Lines, p.Projection.Plan.Slots[1].Recipe.VisibleText)
			firstDigest := p.PlanDigest
			contexts.copy.Lines[0] = "changed later"
			replayed, err := s.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			require.Equal(t, firstDigest, replayed.PlanDigest)
			require.Equal(t, 1, contexts.copyReads, "same-key replay must not translate or re-read new content")
			clone := imageagent.CloneImageSetPlan(p.Projection.Plan.Set)
			clone.Text.Copy.Lines[0] = "changed clone"
			require.Equal(t, "DRONA fabric box", p.Projection.Plan.Set.Text.Copy.Lines[0])
			input.Text = &imageagent.ImageTextSelection{Mode: "none"}
			_, err = s.PrepareImageSet(ctx, input)
			require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
		})
	}
}

func TestReviewedImageTextFailsClosedWithoutConfirmedCopyOwner(t *testing.T) {
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	require.NoError(t, json.Unmarshal([]byte(`{"Text":{"mode":"reviewed","review":{"proposalId":"a3ce4c41-745a-47bf-a418-c940d3ba4b84","revision":"2"},"taskIds":["overview"]}}`), &input))
	_, err := s.PrepareImageSet(ctx, input)
	require.Error(t, err, "a Review identity is not translated-copy authority without its owner")
}

func TestExplicitNoTextDoesNotDelegateLanguageToImageModel(t *testing.T) {
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	require.NoError(t, json.Unmarshal([]byte(`{"Text":{"Mode":"none"}}`), &input))
	p, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	for _, slot := range p.Projection.Plan.Slots {
		require.Contains(t, slot.Recipe.Prompt, "No visible text", "new no-text requests must not retain the old discretionary lettering instruction")
	}
}
