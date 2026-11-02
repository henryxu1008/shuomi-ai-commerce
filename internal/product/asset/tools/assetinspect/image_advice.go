package assetinspect

import (
	"encoding/json"
	"strconv"

	"github.com/invopop/jsonschema"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
)

// ImageAdviceDefinition adds one bounded compute projection, reusing the
// current exact readers and Registry. It has no execution/approval interface.
func ImageAdviceDefinition() commercetool.Definition {
	d := Definition()
	d.Ref = commercetool.ToolRef{ID: "product.image.analyze", Version: "v1.0.0"}
	d.Description = "Check already-approved image facts and give evidence-bound, non-executing guidance. No Store requirement, image reads, generation or mutation."
	d.Risk = commercetool.RiskCompute
	d.OutputSchema = ImageAdviceOutputSchema()
	return d
}
func ImageAdviceOutputSchema() json.RawMessage {
	r := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	raw, err := json.Marshal(r.Reflect(asset.ImageAdvice{}))
	if err != nil {
		return nil
	}
	return raw
}
func NewImageAdviceInvoker(snapshots catalog.VersionedSnapshotReader, assets asset.ApprovedInventoryReader, fresh FreshPrincipalResolver, agent commercetool.AgentDefinition, deps commercetool.InvocationDependencies) (*Invoker, error) {
	e, err := NewExecutor(snapshots, assets)
	if err != nil {
		return nil, err
	}
	e.imageAdvice = true
	return bindInvoker(e, ImageAdviceDefinition(), fresh, agent, deps)
}
func projectImageAdvice(published catalog.PublishedSnapshot, inventory asset.ApprovedAssetInventory, ready bool, envelope commercetool.ExecutionEnvelope) (json.RawMessage, error) {
	binding := asset.ImageAdviceBinding{ProductKey: published.Identity.ProductKey, CatalogVersion: strconv.FormatUint(published.Version, 10), PublicationID: published.PublicationID, TargetPlatform: inventory.Scope.TargetPlatform, InventoryDigest: collection.Digest(inventory)}
	a, err := asset.AnalyzeImageFacts(binding, inventory, ready)
	if err != nil {
		return nil, err
	}
	if err = a.Seal(envelope.Tool().ID, envelope.Tool().Version, envelope.Metadata().CallID); err != nil {
		return nil, err
	}
	return json.Marshal(a)
}
