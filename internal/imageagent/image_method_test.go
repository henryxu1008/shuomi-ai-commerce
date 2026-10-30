package imageagent_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/imageagent"
)

func TestImageMethodBindingRejectsRetiredPolicies(t *testing.T) {
	ref := imageMethodRef()
	for _, policy := range []string{"image-method-prompt-v1", "image-method-visual-brief-v2", "image-method-visual-brief-v3"} {
		t.Run(policy, func(t *testing.T) {
			binding := &imageagent.ImageMethodBinding{Skill: ref, PolicyVersion: policy}
			require.False(t, binding.Valid(), "a retired policy cannot remain a readable current binding")
			require.ErrorIs(t, imageagent.ValidateImageMethodDispatch(binding), imageagent.ErrCommandBlocked)
		})
	}
	require.True(t, (imageagent.ImageMethodBinding{Skill: ref, PolicyVersion: imageagent.ImageMethodPolicyVersion}).Valid())
	require.NoError(t, imageagent.ValidateImageMethodDispatch(nil))
}

func TestImageMethodVisualPlanUsesNaturalFactsWithoutDocumentBlocks(t *testing.T) {
	m := &methodFixture{text: "Soft photographic light and balanced negative space."}
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(m))
	ref := imageMethodRef()
	input.Method = &ref
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, "image-method-visual-brief-v4", prepared.Projection.Plan.Set.Method.PolicyVersion)
	for _, slot := range prepared.Projection.Plan.Slots {
		require.NotContains(t, slot.Recipe.Prompt, `{"Product":`)
		require.NotContains(t, slot.Recipe.Prompt, "ProductID")
		require.NotContains(t, slot.Recipe.Prompt, "SKILL.md")
	}
}

func TestNewDetailMethodDoesNotSilentlyUseSourceLanguageCopy(t *testing.T) {
	m := &methodFixture{text: "A refined product detail graphic with a small number of supported labels."}
	s, _, _, cfg, source, _, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(m))
	cfg.template.Detail[0].Purpose = "key_benefits"
	cfg.template.Detail[0].Brief = "围绕商品事实编排详情图，可使用至多两条有依据的简短中文标签。"
	source.preparation.Catalog.ProductContext.Title = "IKEA DRÖNA 深灰色布艺收纳盒"
	source.preparation.Catalog.ProductContext.Attributes = map[string]string{"产品尺寸": "宽33厘米，深38厘米，高33厘米", "internal_note": "Do not render source notes"}
	source.preparation.Catalog.Manifest = imageagent.CatalogManifest{}
	var err error
	source.preparation.Catalog, err = imageagent.NormalizeAssetCatalog(source.preparation.Catalog)
	require.NoError(t, err)
	source.preparation.Source.CatalogHash = source.preparation.Catalog.Manifest.Hash
	ref := imageMethodRef()
	input.Method = &ref
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	raw, err := json.Marshal(prepared.Projection.Plan.Slots[1].Recipe)
	require.NoError(t, err)
	var recipe struct{ VisibleText []string }
	require.NoError(t, json.Unmarshal(raw, &recipe))
	require.Empty(t, recipe.VisibleText, "source facts cannot replace confirmed target-language copy")
	require.Contains(t, prepared.Projection.Plan.Slots[1].Recipe.Prompt, "No visible text")
	require.Contains(t, prepared.Projection.Plan.Slots[1].Recipe.Prompt, "No table")
	require.NotContains(t, prepared.Projection.Plan.Slots[1].Recipe.Prompt, "Do not render source notes")
	require.Equal(t, ref, prepared.Projection.Plan.Set.Method.Skill)
	confirmed, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.NoError(t, err)
	execution, err := imageagent.ImageSetSlotExecutionInput(confirmed, "overview", 1)
	require.NoError(t, err)
	require.NoError(t, imageagent.ValidateImageSetExecution(execution))
	execution.Slot.Recipe.VisibleText = []string{"unsupported text"}
	require.ErrorIs(t, imageagent.ValidateImageSetExecution(execution), imageagent.ErrRevisionConflict)
	execution, err = imageagent.ImageSetSlotExecutionInput(confirmed, "overview", 1)
	require.NoError(t, err)
	execution.ProductContext.Title = "different product"
	require.ErrorIs(t, imageagent.ValidateImageSetExecution(execution), imageagent.ErrRevisionConflict)
}

func TestVisualMethodIncludesOnlyEvidenceRequiredByTheCurrentTask(t *testing.T) {
	for _, purpose := range []string{"product_identity", "purchase_specs"} {
		t.Run(purpose, func(t *testing.T) {
			m := &methodFixture{text: "A clean product picture."}
			s, _, _, cfg, contexts, _, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(m))
			cfg.template.Carousel[0].Purpose = purpose
			contexts.preparation.Evidence = map[string]string{"description": "INTERNAL_SOURCE_NOTE https://source.example/trace?internal=note", "brand": "UNRELATED_EVIDENCE", "specifications": "24 cl; feldspar porcelain"}
			ref := imageMethodRef()
			input.Method = &ref
			prepared, err := s.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			for _, slot := range prepared.Projection.Plan.Slots {
				require.NotContains(t, slot.Recipe.Prompt, "INTERNAL_SOURCE_NOTE")
				require.NotContains(t, slot.Recipe.Prompt, "https://source.example")
				require.NotContains(t, slot.Recipe.Prompt, "UNRELATED_EVIDENCE")
				if slot.Recipe.Purpose == "purchase_specs" {
					require.Contains(t, slot.Recipe.Prompt, "24 cl; feldspar porcelain")
					require.NotEmpty(t, slot.Recipe.EvidenceDigest)
				} else {
					require.NotContains(t, slot.Recipe.Prompt, "24 cl; feldspar porcelain")
				}
			}
		})
	}
}

func imageMethodRef() agent.SkillVersionRef {
	return agent.SkillVersionRef{Kind: "skill-version-v1", ID: "2bb819bd-af13-4e68-95e0-1cc86130fb1f", OwnerKind: "PLATFORM", OwnerID: "official", Version: 1, Digest: strings.Repeat("f", 64), PolicyVersion: "skill-method-v1"}
}

type methodFixture struct {
	text              string
	unavailable, held bool
	calls             int
}

func (m *methodFixture) AcquireImageMethod(context.Context, imageagent.ExecutionIdentity, agent.SkillVersionRef, string) (string, func(), error) {
	m.calls++
	if m.unavailable {
		return "", nil, imageagent.ErrCommandBlocked
	}
	m.held = true
	return m.text, func() { m.held = false }, nil
}

func TestImageMethodBindsPlanQuoteAndInputWithoutPersistingMaterial(t *testing.T) {
	plain, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	original, err := plain.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	m := &methodFixture{text: "private-method-material-only-provider"}
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(m))
	ref := imageMethodRef()
	input.Method = &ref
	selected, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.False(t, m.held)
	require.Equal(t, ref, selected.Projection.Plan.Set.Method.Skill)
	require.NotEqual(t, original.QuoteDigest, selected.QuoteDigest)
	require.NotEqual(t, original.PlanDigest, selected.PlanDigest)
	require.NotEqual(t, original.Projection.Plan.Set.InputDigest, selected.Projection.Plan.Set.InputDigest)
	require.Equal(t, original.Points, selected.Points)
	require.Equal(t, original.Images, selected.Images)
	raw, err := json.Marshal(selected.Projection)
	require.NoError(t, err)
	require.NotContains(t, string(raw), m.text)
	input.Method = nil
	_, err = s.PrepareImageSet(ctx, input)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
}

func TestImageMethodRejectsOversizeBeforeCreatingRun(t *testing.T) {
	m := &methodFixture{text: strings.Repeat("x", 16<<10)}
	s, repo, _, _, _, _, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(m))
	ref := imageMethodRef()
	input.Method = &ref
	_, err := s.PrepareImageSet(ctx, input)
	require.Error(t, err)
	require.False(t, m.held)
	input.Method = nil
	_, err = s.PrepareImageSet(ctx, input)
	require.NoError(t, err, "oversize material must not leave a frozen run bound to the request key")
	require.NotNil(t, repo)
}

type methodFenceCommit struct {
	imageagent.Repository
	m *methodFixture
	t *testing.T
}

func (r *methodFenceCommit) CommitProjection(ctx context.Context, c imageagent.ProjectionCommit) (imageagent.RunProjection, error) {
	if c.EventType == "run.confirmed" {
		require.True(r.t, r.m.held, "method fence must span the confirmation CAS")
	}
	return r.Repository.CommitProjection(ctx, c)
}

func TestNewImageConfirmationHoldsMethodFenceAndOriginalReceiptRecoveryDoesNotReacquire(t *testing.T) {
	for _, lostCAS := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "original_receipt"}[lostCAS], func(t *testing.T) {
			m := &methodFixture{text: "bounded method"}
			wrap := func(r imageagent.Repository) imageagent.Repository {
				if lostCAS {
					return &confirmCommitFailure{Repository: r, fail: true}
				}
				return &methodFenceCommit{Repository: r, m: m, t: t}
			}
			s, _, _, cfg, _, _, ctx, input := imageSetServiceFixtureWithRepository(t, wrap, imageagent.WithImageMethodContext(m))
			ref := imageMethodRef()
			input.Method = &ref
			prepared, err := s.PrepareImageSet(ctx, input)
			require.NoError(t, err)
			_, err = s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
			if lostCAS {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, m.held)
			require.Equal(t, 1, cfg.admissions)
			calls := m.calls
			m.unavailable = true
			_, err = s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
			require.NoError(t, err)
			require.Equal(t, calls, m.calls)
			require.Equal(t, 1, cfg.admissions)
		})
	}
}

func TestImageMethodBindingRequiresWiredPort(t *testing.T) {
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t)
	raw, err := json.Marshal(map[string]any{"Method": imageMethodRef()})
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &input))
	_, err = s.PrepareImageSet(ctx, input)
	require.Error(t, err, "a selected method must not silently execute without its use/material port")
}
