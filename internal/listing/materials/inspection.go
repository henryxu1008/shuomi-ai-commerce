package materials

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
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
	EvidenceID string                       `json:"evidenceId"`
	Binding    agent.Binding                `json:"binding"`
	Base       record.TargetRecord          `json:"base"`
	Product    catalog.PublishedSnapshot    `json:"product"`
	Rules      goods.OfficialRuleSnapshot   `json:"rules"`
	Inventory  asset.ApprovedAssetInventory `json:"inventory"`
}

// Disclosure excludes tenant/actor authority and approval internals. Only
// server-selected, complete fields needed for this proposal reach the model.
type Disclosure struct {
	EvidenceID string                     `json:"evidenceId"`
	Binding    agent.Binding              `json:"binding"`
	Revision   int64                      `json:"revision"`
	StoreID    string                     `json:"storeId"`
	Draft      goods.OfficialDraftInput   `json:"draft"`
	Product    catalog.ProductSnapshot    `json:"product"`
	Rules      goods.OfficialRuleSnapshot `json:"rules"`
	Assets     []DisclosedAsset           `json:"assets"`
	Issues     []goods.OfficialIssue      `json:"issues"`
}

type DisclosedAsset struct {
	ID     string     `json:"id"`
	Role   asset.Role `json:"role"`
	URL    string     `json:"url"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
}

func (i Inspection) Disclose() (Disclosure, error) {
	if err := i.verify(); err != nil {
		return Disclosure{}, err
	}
	d := Disclosure{EvidenceID: i.EvidenceID, Binding: i.Binding, Revision: i.Base.Revision, StoreID: i.Base.Input.StoreID, Draft: i.Base.Input.Draft, Product: i.Product.Snapshot, Rules: i.Rules, Assets: []DisclosedAsset{}, Issues: i.Base.Result.Issues}
	for _, a := range i.Inventory.Assets {
		d.Assets = append(d.Assets, DisclosedAsset{a.ID, a.Role, a.URL, a.Width, a.Height})
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

func NewInspection(base record.TargetRecord, product catalog.PublishedSnapshot, rules goods.OfficialRuleSnapshot, inventory asset.ApprovedAssetInventory) (Inspection, error) {
	binding := agent.Binding{ContextKind: "listing-target-record", ContextID: base.ID, ProductKey: product.Identity.ProductKey, CatalogVersion: strconv.FormatUint(product.Version, 10), PublicationID: product.PublicationID, TargetPlatform: "shein"}
	if !binding.Valid() || base.Revision < 1 || !agent.ValidID(base.TargetID) || base.Input.SourceID != base.Source.ID ||
		base.Source.Source.ProductKey != product.Identity.ProductKey || base.EffectiveVersion != product.Version || base.Input.EffectiveVersion != product.Version || base.Input.ApplyReceiptID != base.ApplyReceiptID ||
		base.Merchant.OrganizationID != product.Identity.TenantID || base.Merchant.StoreID != base.Input.StoreID || base.Merchant.Site != "shein-us" ||
		base.ProductHash != collection.Digest(product.Snapshot) || base.RulesHash != collection.Digest(rules) || base.InventoryHash != collection.Digest(inventory) ||
		inventory.Scope != (asset.InventoryScope{TenantID: product.Identity.TenantID, ProductKey: product.Identity.ProductKey, TargetPlatform: "shein", SourceSnapshotVersion: product.Version}) {
		return Inspection{}, ErrInspectionChanged
	}
	value := Inspection{Binding: binding, Base: base, Product: product, Rules: rules, Inventory: inventory}
	value.EvidenceID = "listing-evidence:" + collection.Digest(value)
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > MaxInspectionBytes {
		return Inspection{}, ErrInspectionTooLarge
	}
	var copied Inspection
	if json.Unmarshal(raw, &copied) != nil {
		return Inspection{}, ErrInspectionChanged
	}
	return copied, nil
}

func (i Inspection) verify() error {
	expected, err := NewInspection(i.Base, i.Product, i.Rules, i.Inventory)
	if err != nil {
		return err
	}
	if i.Binding != expected.Binding || i.EvidenceID != expected.EvidenceID {
		return ErrInspectionChanged
	}
	return nil
}
