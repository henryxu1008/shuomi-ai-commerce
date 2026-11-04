package imageagentapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"reflect"
	"sort"
	"strconv"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/asset/tools/assetinspect"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/collection"
)

// AnalysisImageReader reads only identities returned by the original owners.
// Composition must preserve source/manual/immutable-generated provenance. No
// browser URL, new upload, generation permission or ImageSet workflow is used.
type AnalysisImageReader interface {
	ReadAnalysisSource(context.Context, agent.Binding, catalog.IdentifiedImage, int64) ([]byte, error)
	ReadAnalysisApproved(context.Context, agent.Binding, asset.ApprovedAsset, int64) ([]byte, error)
}

type AnalysisSources struct {
	snapshots catalog.VersionedSnapshotReader
	assets    asset.ApprovedInventoryReader
	authority agent.Authorizer
	images    AnalysisImageReader
}

func NewAnalysisSources(snapshots catalog.VersionedSnapshotReader, assets asset.ApprovedInventoryReader, authority agent.Authorizer, images AnalysisImageReader) (*AnalysisSources, error) {
	for _, v := range []any{snapshots, assets, authority, images} {
		if v == nil || reflect.ValueOf(v).Kind() == reflect.Pointer && reflect.ValueOf(v).IsNil() {
			return nil, agent.ErrUnavailable
		}
	}
	return &AnalysisSources{snapshots: snapshots, assets: assets, authority: authority, images: images}, nil
}

type resolvedAnalysisSource struct {
	snapshot  catalog.PublishedSnapshot
	inventory asset.ApprovedAssetInventory
	source    catalog.IdentifiedImage
	approved  asset.ApprovedAsset
}

func (r *AnalysisSources) resolve(ctx context.Context, b agent.Binding, a agentconfig.ImageAnalysisConfiguration) (resolvedAnalysisSource, error) {
	var out resolvedAnalysisSource
	if r == nil || ctx == nil || ctx.Err() != nil || !agentconfig.ImageAnalysisBinding(b) {
		return out, agent.ErrInvalid
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || i.TenantID != i.EffectiveOrganizationID || !i.TokenExpiresAt.After(time.Now()) || !agent.ValidID(i.EffectiveMemberID) {
		return out, agent.ErrUnavailable
	}
	scope, err := r.authority.Authorize(ctx, b)
	if err != nil {
		return out, err
	}
	if scope != (agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}) {
		return out, agent.ErrInvalid
	}
	version, err := strconv.ParseUint(b.CatalogVersion, 10, 64)
	if err != nil || version == 0 || strconv.FormatUint(version, 10) != b.CatalogVersion {
		return out, agent.ErrInvalid
	}
	snapshot, err := r.snapshots.GetSnapshot(ctx, catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: b.ProductKey}, version)
	if err != nil {
		return out, err
	}
	if snapshot.Identity != (catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: b.ProductKey}) || snapshot.Version != version || snapshot.PublicationID != b.PublicationID {
		return out, agent.ErrConflict
	}
	out.snapshot = snapshot
	if a.Kind == "source" {
		if a.InventoryDigest != "" {
			return out, agent.ErrInvalid
		}
		for _, img := range catalog.IdentifyImages(snapshot.Snapshot.Images) {
			if img.ID == a.ImageID {
				out.source = img
				return out, nil
			}
		}
		return out, agent.ErrInvalid
	}
	if a.Kind != "approved" || !agentconfig.ImageDigest(a.InventoryDigest) {
		return out, agent.ErrInvalid
	}
	expected := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: b.ProductKey, SourceSnapshotVersion: version, TargetPlatform: b.TargetPlatform}
	inventory, err := r.assets.GetApprovedInventory(ctx, expected)
	if err != nil {
		return out, err
	}
	if inventory.Scope != expected || collection.Digest(inventory) != a.InventoryDigest {
		return out, agent.ErrConflict
	}
	if err = asset.ValidateApprovalCommit(asset.ApprovalCommit{TenantID: scope.OrganizationID, ProductKey: b.ProductKey, TargetPlatform: b.TargetPlatform, SourceSnapshotVersion: version, ActionID: "analysis-exact-read", Assets: inventory.Assets}); err != nil {
		return out, err
	}
	out.inventory = asset.CloneApprovedAssetInventory(inventory)
	for _, img := range out.inventory.Assets {
		if img.ID == a.ImageID {
			out.approved = img
			return out, nil
		}
	}
	return out, agent.ErrInvalid
}

func analysisEvidenceHash(raw []byte) string {
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}

func (r *AnalysisSources) checked(ctx context.Context, in agent.ModelInput, a agentconfig.ImageAnalysisConfiguration) (resolvedAnalysisSource, json.RawMessage, error) {
	if !a.Image.Valid() {
		return resolvedAnalysisSource{}, nil, agent.ErrInvalid
	}
	resolved, err := r.resolve(ctx, in.Binding, a)
	if err != nil {
		return resolved, nil, err
	}
	want := 1
	if a.Kind == "approved" {
		want = 2
	}
	if len(in.History) != want {
		return resolved, nil, agent.ErrInvalid
	}
	refs := []commercetool.ToolRef{canonicalinspect.Definition().Ref, assetinspect.Definition().Ref}
	for i, o := range in.History {
		if o.Tool != refs[i] || o.AuditStatus != commercetool.AuditStatusRecorded || !agent.ValidID(o.CallID) || o.Step != i+1 || o.InvocationID != "" || len(o.Output) == 0 || len(o.Output) > canonicalinspect.MaxOutputBytes {
			return resolved, nil, agent.ErrInvalid
		}
	}
	canonical, err := canonicalinspect.Project(resolved.snapshot)
	if err != nil || !bytes.Equal(canonical, in.History[0].Output) {
		return resolved, nil, agent.ErrConflict
	}
	if want == 2 {
		var output assetinspect.Output
		if json.Unmarshal(in.History[1].Output, &output) != nil {
			return resolved, nil, agent.ErrInvalid
		}
		product, err := json.Marshal(output.Output)
		if err != nil || !bytes.Equal(product, canonical) || output.AssetBinding != (assetinspect.AssetBinding{TargetPlatform: in.Binding.TargetPlatform, SourceSnapshotVersion: in.Binding.CatalogVersion}) {
			return resolved, nil, agent.ErrConflict
		}
		sort.Slice(resolved.inventory.Assets, func(i, j int) bool { return resolved.inventory.Assets[i].ID < resolved.inventory.Assets[j].ID })
		if !reflect.DeepEqual(output.ApprovedAssets, resolved.inventory.Assets) {
			return resolved, nil, agent.ErrConflict
		}
	}
	// Finite identity/digests project actual audited observations. Raw product
	// data, URLs and image bytes are not copied into the model prompt/history.
	reads := make([]struct {
		Tool           commercetool.ToolRef
		CallID, SHA256 string
	}, 0, want)
	for _, o := range in.History {
		reads = append(reads, struct {
			Tool           commercetool.ToolRef
			CallID, SHA256 string
		}{o.Tool, o.CallID, analysisEvidenceHash(o.Output)})
	}
	evidence, err := json.Marshal(struct {
		Kind, ImageID string
		Reads         any
	}{a.Kind, a.ImageID, reads})
	return resolved, evidence, err
}

func (r *AnalysisSources) ReadImageAnalysis(ctx context.Context, in agent.ModelInput, a agentconfig.ImageAnalysisConfiguration) ([]byte, json.RawMessage, error) {
	resolved, evidence, err := r.checked(ctx, in, a)
	if err != nil {
		return nil, nil, err
	}
	content, err := r.readSelected(ctx, in.Binding, a.Kind, resolved)
	if err != nil {
		return nil, nil, err
	}
	if err = governed.ValidateSingleImageContent(a.Image, content); err != nil {
		return nil, nil, err
	}
	return append([]byte(nil), content...), evidence, nil
}

func (r *AnalysisSources) readSelected(ctx context.Context, b agent.Binding, kind string, resolved resolvedAnalysisSource) ([]byte, error) {
	if kind == "source" {
		return r.images.ReadAnalysisSource(ctx, b, resolved.source, aicapability.MaximumSingleImageBytes)
	}
	return r.images.ReadAnalysisApproved(ctx, b, resolved.approved, aicapability.MaximumSingleImageBytes)
}

// FreezeImageAnalysis derives content identity from the authorized original
// reader before the existing configuration owner commits Prepare. No provider,
// Run, reserve, image copy or new durable permit is created here.
func (r *AnalysisSources) FreezeImageAnalysis(ctx context.Context, c agentconfig.StartCommand, kind, imageID, inventoryDigest, applyReceiptID string) (agentconfig.StartCommand, error) {
	if ctx == nil || ctx.Err() != nil {
		return agentconfig.StartCommand{}, agent.ErrInvalid
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || c.Scope != (agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}) || c.MemberID != i.EffectiveMemberID {
		return agentconfig.StartCommand{}, agent.ErrUnavailable
	}
	a := agentconfig.ImageAnalysisConfiguration{Kind: kind, ImageID: imageID, InventoryDigest: inventoryDigest, ApplyReceiptID: applyReceiptID}
	resolved, err := r.resolve(ctx, c.Request.Binding, a)
	if err != nil {
		return agentconfig.StartCommand{}, err
	}
	content, err := r.readSelected(ctx, c.Request.Binding, kind, resolved)
	if err != nil {
		return agentconfig.StartCommand{}, err
	}
	if len(content) == 0 || len(content) > aicapability.MaximumSingleImageBytes {
		return agentconfig.StartCommand{}, agent.ErrInvalid
	}
	decoded, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil || format != "png" && format != "jpeg" {
		return agentconfig.StartCommand{}, agent.ErrInvalid
	}
	description := aicapability.SingleImageInput{ContentSHA256: analysisEvidenceHash(content), MediaType: "image/" + format, ByteLength: len(content), Width: decoded.Width, Height: decoded.Height}
	frozen, err := agentconfig.FreezeImageAnalysis(c, kind, imageID, inventoryDigest, applyReceiptID, description)
	if err != nil {
		return agentconfig.StartCommand{}, err
	}
	if err = governed.ValidateSingleImageContent(frozen.Image, content); err != nil {
		return agentconfig.StartCommand{}, err
	}
	c.ImageAnalysis = &frozen
	return c, nil
}

func (r *AnalysisSources) CheckImageAnalysis(ctx context.Context, in agent.ModelInput, a agentconfig.ImageAnalysisConfiguration) error {
	_, _, err := r.checked(ctx, in, a)
	return err
}
