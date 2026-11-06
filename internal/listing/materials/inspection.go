package materials

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	contract "task-processor/internal/marketplace/validator"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/catalog/tools/canonicalinspect"
	"task-processor/internal/product/collection"
)

const MaxInspectionBytes = 512 << 10

var (
	ErrInspectionChanged  = errors.New("platform material inspection no longer matches exact owner facts")
	ErrInspectionTooLarge = errors.New("platform material inspection exceeds disclosure bound")
)

// Inspection is a bounded read projection, never a second Product/Target fact.
// Its evidence ID identifies the entire disclosure, including current rules.
type Inspection struct {
	EvidenceID     string                       `json:"evidenceId"`
	Binding        agent.Binding                `json:"binding"`
	Base           record.TargetRecord          `json:"base"`
	Product        catalog.PublishedSnapshot    `json:"product"`
	Rules          goods.OfficialRuleSnapshot   `json:"rules"`
	Inventory      asset.ApprovedAssetInventory `json:"inventory"`
	InventoryReady bool                         `json:"inventoryReady"`
	computed       contract.DiagnosticResult
	integrity      string
}

// Disclosure excludes tenant/actor authority and approval internals. Only
// server-selected, complete fields needed for this proposal reach the model.
type Disclosure struct {
	EvidenceID         string                     `json:"evidenceId"`
	Binding            agent.Binding              `json:"binding"`
	Revision           int64                      `json:"revision"`
	StoreID            string                     `json:"storeId"`
	Draft              goods.OfficialDraftInput   `json:"draft"`
	Product            catalog.ProductSnapshot    `json:"product"`
	Rules              goods.OfficialRuleSnapshot `json:"rules"`
	Assets             []DisclosedAsset           `json:"assets"`
	Issues             []goods.OfficialIssue      `json:"issues"`
	ComputedDiagnostic contract.DiagnosticResult  `json:"computedDiagnostic"`
	ImageAdvice        asset.ImageAdvice          `json:"imageAdvice"`
}

type DisclosedAsset struct {
	ID                string                        `json:"id"`
	Role              asset.Role                    `json:"role"`
	URL               string                        `json:"url"`
	Width             int                           `json:"width"`
	Height            int                           `json:"height"`
	OfficialPlacement *asset.ImageOfficialPlacement `json:"officialPlacement,omitempty"`
}

func (i Inspection) Disclose() (Disclosure, error) {
	if err := i.verify(); err != nil {
		return Disclosure{}, err
	}
	return i.disclose()
}

func (i Inspection) disclose() (Disclosure, error) {
	// Reuse Catalog's approved disclosure, which removes internal source
	// metadata while retaining supported product facts and trace evidence.
	projected, err := canonicalinspect.Project(i.Product)
	if err != nil {
		return Disclosure{}, ErrInspectionTooLarge
	}
	var canonical canonicalinspect.Output
	if json.Unmarshal(projected, &canonical) != nil {
		return Disclosure{}, ErrInspectionChanged
	}
	d := Disclosure{EvidenceID: i.EvidenceID, Binding: i.Binding, Revision: i.Base.Revision, StoreID: i.Base.Input.StoreID, Draft: i.Base.Input.Draft, Product: canonical.Snapshot, Rules: i.Rules, Assets: []DisclosedAsset{}, Issues: i.Base.Result.Issues}
	d.ComputedDiagnostic = i.computed
	d.ImageAdvice, err = i.imageAdvice()
	if err != nil {
		return Disclosure{}, err
	}
	for _, a := range i.Inventory.Assets {
		d.Assets = append(d.Assets, DisclosedAsset{a.ID, a.Role, a.URL, a.Width, a.Height, a.OfficialPlacement})
	}
	raw, err := json.Marshal(d)
	if err != nil || len(raw) > MaxInspectionBytes {
		return Disclosure{}, ErrInspectionTooLarge
	}
	var copied Disclosure
	if json.Unmarshal(raw, &copied) != nil {
		return Disclosure{}, ErrInspectionChanged
	}
	return copied, nil
}

type InspectionReader interface {
	// Freshly authorizes the original source/member and checks current head,
	// effective Product lineage, Store binding, rules and approved inventory.
	ReadMaterialsInspection(context.Context, agent.Binding) (Inspection, error)
}

func InspectToolRef() commercetool.ToolRef {
	return agentconfig.MaterialsInspectToolRef()
}

func NewInspection(base record.TargetRecord, product catalog.PublishedSnapshot, rules goods.OfficialRuleSnapshot, inventory asset.ApprovedAssetInventory, readAt, evaluatedAt time.Time) (Inspection, error) {
	return NewInspectionWithInventoryAvailability(base, product, rules, inventory, true, readAt, evaluatedAt)
}

// A successful explicit empty read and an inventory not ready for reading are
// different owner observations, even when the original builder needs [].
func NewInspectionWithInventoryAvailability(base record.TargetRecord, product catalog.PublishedSnapshot, rules goods.OfficialRuleSnapshot, inventory asset.ApprovedAssetInventory, ready bool, readAt, evaluatedAt time.Time) (Inspection, error) {
	binding := agent.Binding{ContextKind: "listing-target-record", ContextID: base.ID, ProductKey: product.Identity.ProductKey, CatalogVersion: strconv.FormatUint(product.Version, 10), PublicationID: product.PublicationID, TargetPlatform: "shein"}
	if !binding.Valid() || base.Revision < 1 || !agent.ValidID(base.TargetID) || base.Input.SourceID != base.Source.ID ||
		base.Source.Source.ProductKey != product.Identity.ProductKey || base.EffectiveVersion != product.Version || base.Input.EffectiveVersion != product.Version || base.Input.ApplyReceiptID != base.ApplyReceiptID ||
		base.Merchant.OrganizationID != product.Identity.TenantID || base.Merchant.StoreID != base.Input.StoreID || base.Merchant.Site != "shein-us" ||
		base.ProductHash != collection.Digest(product.Snapshot) || base.RulesHash != collection.Digest(rules) || base.InventoryHash != collection.Digest(inventory) ||
		inventory.Scope != (asset.InventoryScope{TenantID: product.Identity.TenantID, ProductKey: product.Identity.ProductKey, TargetPlatform: "shein", SourceSnapshotVersion: product.Version}) {
		return Inspection{}, ErrInspectionChanged
	}
	value := Inspection{Binding: binding, Base: base, Product: product, Rules: rules, Inventory: inventory, InventoryReady: ready}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > MaxInspectionBytes {
		return Inspection{}, ErrInspectionTooLarge
	}
	var copied Inspection
	if json.Unmarshal(raw, &copied) != nil {
		return Inspection{}, ErrInspectionChanged
	}
	// Bind actual owner facts here, keeping Store/Catalog DTOs out of the
	// compute-only Marketplace adapter. No digest is an authority assertion.
	contextDigest, err := sheinvalidator.OfficialContentDigest(struct {
		Source           any    `json:"source"`
		TargetID         string `json:"target_id"`
		RecordID         string `json:"record_id"`
		ExpectedRevision int64  `json:"expected_revision"`
	}{copied.Base.Source, base.TargetID, base.ID, base.Revision})
	if err != nil {
		return Inspection{}, err
	}
	productDigest, err := sheinvalidator.OfficialContentDigest(struct {
		ProductKey     string                  `json:"product_key"`
		Version        uint64                  `json:"version"`
		PublicationID  string                  `json:"publication_id"`
		ApplyReceiptID string                  `json:"apply_receipt_id"`
		Product        catalog.ProductSnapshot `json:"product"`
	}{product.Identity.ProductKey, product.Version, product.PublicationID, base.ApplyReceiptID, copied.Product.Snapshot})
	if err != nil {
		return Inspection{}, err
	}
	merchantDigest, err := sheinvalidator.OfficialContentDigest(copied.Base.Merchant)
	if err != nil {
		return Inspection{}, err
	}
	identity := sheinvalidator.OfficialIdentity{ContextDigest: contextDigest, ProductReferenceDigest: productDigest, MerchantDigest: merchantDigest}
	assessment := goods.AssessOfficial(copied.Base.Input.Draft, copied.Rules, copied.Inventory, copied.Base.ImageObservations, copied.Base.ProofObservations)
	copied.computed, err = assessment.Diagnostic(contract.BoundRequest[sheinvalidator.OfficialIdentity]{Input: identity, Target: contract.Target{Marketplace: "shein", Site: "shein-us"}, Action: contract.Preview, RuleVersion: sheinvalidator.OfficialDraftRuleVersion, BindingVersion: sheinvalidator.OfficialDraftBindingVersion, ReadAt: readAt, EvaluatedAt: evaluatedAt, Freshness: contract.ExternalFreshness{Status: contract.NotEvaluated}})
	if err != nil {
		return Inspection{}, err
	}
	disclosed, err := copied.disclose()
	if err != nil {
		return Inspection{}, err
	}
	copied.EvidenceID, err = disclosureFingerprint(disclosed)
	if err != nil {
		return Inspection{}, err
	}
	disclosed.EvidenceID = copied.EvidenceID
	output, err := json.Marshal(disclosed)
	if err != nil || len(output) > MaxInspectionBytes {
		return Inspection{}, ErrInspectionTooLarge
	}
	copied.integrity, err = sheinvalidator.OfficialContentDigest(copied)
	if err != nil {
		return Inspection{}, err
	}
	return copied, nil
}

func (i Inspection) verify() error {
	actual, err := sheinvalidator.OfficialContentDigest(i)
	if err != nil || i.integrity == "" || actual != i.integrity {
		return ErrInspectionChanged
	}
	return nil
}

// Only read/evaluation metadata is excluded. Actual content, computed checks,
// scope, algorithm versions and the exact Tool remain in the stable identity.
func disclosureFingerprint(d Disclosure) (string, error) {
	if err := d.ImageAdvice.Verify(); err != nil {
		return "", err
	}
	if err := contract.ValidateBoundInput(d.ComputedDiagnostic.Input, ""); err != nil {
		return "", err
	}
	if err := d.ComputedDiagnostic.Freshness.Validate(d.ComputedDiagnostic.Input.Digest, d.ComputedDiagnostic.Input.EvaluatedAt); err != nil {
		return "", err
	}
	d.EvidenceID = ""
	d.ImageAdvice.Evidence.CallID = ""
	d.ComputedDiagnostic.Input.ReadAt = time.Time{}
	d.ComputedDiagnostic.Input.EvaluatedAt = time.Time{}
	digest, err := sheinvalidator.OfficialContentDigest(struct {
		Tool       commercetool.ToolRef `json:"tool"`
		Disclosure Disclosure           `json:"disclosure"`
	}{InspectToolRef(), d})
	if err != nil {
		return "", err
	}
	return "listing-evidence-v3:" + strings.TrimPrefix(digest, "sha256:"), nil
}
