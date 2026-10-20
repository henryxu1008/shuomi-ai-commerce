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

func TestImageMethodVisualPlanUsesNaturalFactsWithoutDocumentBlocks(t *testing.T) {
	m := &methodFixture{text: "Soft photographic light and balanced negative space."}
	s, _, _, _, _, _, ctx, input := imageSetServiceFixture(t, imageagent.WithImageMethodContext(m))
	ref := imageMethodRef()
	input.Method = &ref
	prepared, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	require.Equal(t, "image-method-visual-brief-v3", prepared.Projection.Plan.Set.Method.PolicyVersion)
	for _, slot := range prepared.Projection.Plan.Slots {
		require.NotContains(t, slot.Recipe.Prompt, `{"Product":`)
		require.NotContains(t, slot.Recipe.Prompt, "ProductID")
		require.NotContains(t, slot.Recipe.Prompt, "SKILL.md")
	}
}

func TestDetailMethodFreezesLiteralCopyInsteadOfDelegatingCopywriting(t *testing.T) {
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
	require.Equal(t, []string{"IKEA DRÖNA 深灰色布艺收纳盒", "产品尺寸：宽33厘米，深38厘米，高33厘米"}, recipe.VisibleText)
	require.Contains(t, prepared.Projection.Plan.Slots[1].Recipe.Prompt, "No table")
	require.NotContains(t, prepared.Projection.Plan.Slots[1].Recipe.Prompt, "Do not render source notes")
	require.Equal(t, ref, prepared.Projection.Plan.Set.Method.Skill)
	confirmed, err := s.ConfirmImagePlan(ctx, confirmSetInput(prepared))
	require.NoError(t, err)
	execution, err := imageagent.ImageSetSlotExecutionInput(confirmed, "overview", 1)
	require.NoError(t, err)
	require.NoError(t, imageagent.ValidateImageSetExecution(execution))
	execution.Slot.Recipe.VisibleText[0] = "unsupported text"
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

type historicalMethodRepository struct {
	imageagent.Repository
	input      imageagent.PrepareImageSetInput
	historical bool
}

func (r *historicalMethodRepository) GetProjection(ctx context.Context, scope imageagent.RunScope) (imageagent.RunProjection, error) {
	p, err := r.Repository.GetProjection(ctx, scope)
	if err == nil && r.historical {
		p.Plan.Set.Method.PolicyVersion = "image-method-prompt-v1"
		raw, _ := json.Marshal(struct {
			Input        imageagent.PrepareImageSetInput
			MethodPolicy string
		}{r.input, "image-method-prompt-v1"})
		sum := sha256.Sum256(raw)
		p.Plan.Set.InputDigest = hex.EncodeToString(sum[:])
		p.Plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(p.Plan)
	}
	return p, err
}

// Install an original v2 snapshot in the isolated repository, rather than
// changing a read view of an immutable v3 plan during confirmation.
type frozenVisualV2Repository struct {
	imageagent.Repository
	input imageagent.PrepareImageSetInput
}

func (r *frozenVisualV2Repository) InitializeRun(ctx context.Context, p imageagent.ProjectionInitialization) (imageagent.RunProjection, error) {
	p.Plan.Set.Method.PolicyVersion = "image-method-visual-brief-v2"
	raw, _ := json.Marshal(struct {
		Input        imageagent.PrepareImageSetInput
		MethodPolicy string
	}{r.input, "image-method-visual-brief-v2"})
	sum := sha256.Sum256(raw)
	p.Plan.Set.InputDigest = hex.EncodeToString(sum[:])
	var err error
	p.Plan.Set.QuoteDigest, err = imageagent.ImageSetQuoteDigest(p.Plan)
	if err != nil {
		return imageagent.RunProjection{}, err
	}
	p.Snapshot.Plan = p.Plan
	return r.Repository.InitializeRun(ctx, p)
}

func TestFrozenVisualV2SameKeyAndOriginalConfirmationKeepTheirBytes(t *testing.T) {
	var original *frozenVisualV2Repository
	wrap := func(repo imageagent.Repository) imageagent.Repository {
		original = &frozenVisualV2Repository{Repository: repo}
		return original
	}
	m := &methodFixture{text: "Original immutable visual brief."}
	s, _, _, cfg, _, _, ctx, input := imageSetServiceFixtureWithRepository(t, wrap, imageagent.WithImageMethodContext(m))
	ref := imageMethodRef()
	input.Method = &ref
	original.input = input
	first, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	old, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	before, err := json.Marshal(old.Projection.Plan.Slots)
	require.NoError(t, err)
	initial, err := json.Marshal(first.Projection.Plan.Slots)
	require.NoError(t, err)
	require.Equal(t, initial, before, "non-detail recipes retain their exact v2 prompt bytes")
	require.Equal(t, ref, old.Projection.Plan.Set.Method.Skill)
	require.NoError(t, imageagent.ValidateImageMethodDispatch(old.Projection.Plan.Set.Method))
	_, err = s.ConfirmImagePlan(ctx, confirmSetInput(old))
	require.NoError(t, err)
	calls := m.calls
	m.unavailable = true
	_, err = s.ConfirmImagePlan(ctx, confirmSetInput(old))
	require.NoError(t, err, "original confirmation receipt recovery keeps its prior admission")
	require.Equal(t, 1, cfg.admissions)
	require.Equal(t, calls, m.calls)
	read, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	after, err := json.Marshal(read.Projection.Plan.Slots)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, old.PlanDigest, read.PlanDigest)
	require.Equal(t, old.QuoteDigest, read.QuoteDigest)
}
func TestHistoricalMethodSameKeyReadSurvivesPolicyChangeButNewConfirmIsBlocked(t *testing.T) {
	var original *historicalMethodRepository
	wrap := func(repo imageagent.Repository) imageagent.Repository {
		original = &historicalMethodRepository{Repository: repo}
		return original
	}
	m := &methodFixture{text: "A product photograph."}
	s, _, _, cfg, _, _, ctx, input := imageSetServiceFixtureWithRepository(t, wrap, imageagent.WithImageMethodContext(m))
	ref := imageMethodRef()
	input.Method = &ref
	original.input = input
	_, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err)
	original.historical = true
	calls := m.calls
	old, err := s.PrepareImageSet(ctx, input)
	require.NoError(t, err, "same stored v1 policy must determine its original hash")
	require.Equal(t, "image-method-prompt-v1", old.Projection.Plan.Set.Method.PolicyVersion)
	require.Equal(t, calls, m.calls)
	_, err = s.ConfirmImagePlan(ctx, confirmSetInput(old))
	require.ErrorIs(t, err, imageagent.ErrCommandBlocked)
	require.Zero(t, cfg.admissions)
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
