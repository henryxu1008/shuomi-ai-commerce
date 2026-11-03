package imageagentapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/collection"
)

type analysisFacts struct {
	snapshot  catalog.PublishedSnapshot
	inventory asset.ApprovedAssetInventory
	deny      bool
	reads     int
	content   []byte
}

func (f *analysisFacts) Authorize(context.Context, agent.Binding) (agent.Scope, error) {
	if f.deny {
		return agent.Scope{}, agent.ErrUnavailable
	}
	return agent.Scope{OrganizationID: "org", ActorID: "actor"}, nil
}
func (f *analysisFacts) GetSnapshot(context.Context, catalog.SnapshotIdentity, uint64) (catalog.PublishedSnapshot, error) {
	return f.snapshot, nil
}
func (f *analysisFacts) GetApprovedInventory(context.Context, asset.InventoryScope) (asset.ApprovedAssetInventory, error) {
	return f.inventory, nil
}
func (f *analysisFacts) ReadAnalysisSource(_ context.Context, _ agent.Binding, img catalog.IdentifiedImage, maximum int64) ([]byte, error) {
	f.reads++
	if img.ID != "catalog-image-1" || maximum != aicapability.MaximumSingleImageBytes {
		return nil, agent.ErrInvalid
	}
	return f.content, nil
}
func (f *analysisFacts) ReadAnalysisApproved(_ context.Context, _ agent.Binding, img asset.ApprovedAsset, maximum int64) ([]byte, error) {
	f.reads++
	if img.ID != "approved-image" || maximum != aicapability.MaximumSingleImageBytes {
		return nil, agent.ErrInvalid
	}
	return f.content, nil
}

func analysisSourceFixture(t *testing.T, approved bool) (*AnalysisSources, *analysisFacts, context.Context, agent.ModelInput, agentconfig.ImageAnalysisConfiguration) {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	f := &analysisFacts{content: b.Bytes(), snapshot: catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org", ProductKey: "product"}, PublicationID: "publication", Version: 1, Snapshot: catalog.ProductSnapshot{Images: []catalog.Image{{URL: "https://source.example/exact.png"}}}}}
	f.inventory = asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: "org", ProductKey: "product", SourceSnapshotVersion: 1, TargetPlatform: "product"}, Assets: []asset.ApprovedAsset{{ID: "approved-image", RunID: "run", PlanRevision: 1, SlotID: "slot", Attempt: 1, Role: asset.RoleMain, URL: "https://assets.example/exact.png"}}}
	r, err := NewAnalysisSources(f, f, f, f)
	require.NoError(t, err)
	i := authidentity.AuthenticatedIdentity{TenantID: "org", EffectiveOrganizationID: "org", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), i)
	binding := agent.Binding{ContextKind: "own_product", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "product"}
	sum := sha256.Sum256(f.content)
	a := agentconfig.ImageAnalysisConfiguration{Kind: "source", ImageID: "catalog-image-1", Image: aicapability.SingleImageInput{SelectionDigest: strings.Repeat("a", 64), ContentSHA256: hex.EncodeToString(sum[:]), MediaType: "image/png", ByteLength: len(f.content), Width: 2, Height: 2, Detail: "high", MeteringVersion: aicapability.SingleImageMiniMetering}}
	canonical, err := canonicalinspect.Project(f.snapshot)
	require.NoError(t, err)
	in := agent.ModelInput{Binding: binding, History: []agent.Observation{{Step: 1, Tool: canonicalinspect.Definition().Ref, CallID: uuid.NewString(), AuditStatus: commercetool.AuditStatusRecorded, Output: canonical}}}
	if approved {
		a.Kind, a.ImageID, a.InventoryDigest = "approved", "approved-image", collection.Digest(f.inventory)
		var c canonicalinspect.Output
		require.NoError(t, json.Unmarshal(canonical, &c))
		out, err := json.Marshal(assetinspect.Output{Output: c, AssetBinding: assetinspect.AssetBinding{TargetPlatform: "product", SourceSnapshotVersion: "1"}, ApprovedAssets: f.inventory.Assets})
		require.NoError(t, err)
		in.History = append(in.History, agent.Observation{Step: 2, Tool: assetinspect.Definition().Ref, CallID: uuid.NewString(), AuditStatus: commercetool.AuditStatusRecorded, Output: out})
	}
	return r, f, ctx, in, a
}

func TestAnalysisSourceUsesExactOriginalSourceOrApprovedInventory(t *testing.T) {
	for _, approved := range []bool{false, true} {
		r, f, ctx, in, a := analysisSourceFixture(t, approved)
		raw, evidence, err := r.ReadImageAnalysis(ctx, in, a)
		require.NoError(t, err)
		require.Equal(t, f.content, raw)
		require.Equal(t, 1, f.reads)
		require.NotContains(t, string(evidence), "https://")
		require.Contains(t, string(evidence), a.ImageID)
		require.NoError(t, r.CheckImageAnalysis(ctx, in, a))
		require.Equal(t, 1, f.reads, "handoff checks current authority and sends the already hashed buffer")
	}
}
func TestAnalysisSourceRejectsChangedFactsPermissionAndContent(t *testing.T) {
	for _, scenario := range []string{"publication", "tool binding", "tool ref", "unrecorded audit", "invented invocation", "inventory", "permission", "bytes", "dimensions"} {
		t.Run(scenario, func(t *testing.T) {
			r, f, ctx, in, a := analysisSourceFixture(t, true)
			switch scenario {
			case "publication":
				f.snapshot.PublicationID = "different"
			case "tool binding":
				in.History[0].Output = json.RawMessage(`{"product_key":"other"}`)
			case "tool ref":
				in.History[0].Tool = commercetool.ToolRef{ID: "other.tool", Version: "v1.0.0"}
			case "unrecorded audit":
				in.History[0].AuditStatus = ""
			case "invented invocation":
				in.History[0].InvocationID = "fake-model-invocation"
			case "inventory":
				a.InventoryDigest = strings.Repeat("b", 64)
			case "permission":
				f.deny = true
			case "bytes":
				f.content = []byte("replacement")
			case "dimensions":
				a.Image.Width++
			}
			_, _, err := r.ReadImageAnalysis(ctx, in, a)
			require.Error(t, err)
			if scenario != "bytes" && scenario != "dimensions" {
				require.Zero(t, f.reads, "reject before any image download")
			}
		})
	}
}
