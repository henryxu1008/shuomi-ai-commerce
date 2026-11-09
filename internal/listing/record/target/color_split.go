package target

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	sigjson "sigs.k8s.io/json"
	"sort"
	"strconv"
	"time"

	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
)

var ErrSplitExists = errors.New("color split family already exists")

type ColorSKUMapping struct {
	SourceIndex int `json:"sourceIndex"`
	SKCIndex    int `json:"skcIndex"`
	SKUIndex    int `json:"skuIndex"`
}
type ColorSharedImage struct {
	SlotIndex     int     `json:"slotIndex"`
	ColorValueIDs []int64 `json:"colorValueIds"`
}

func strictColorJSON(raw []byte, out any, required ...string) error {
	problems, err := sigjson.UnmarshalStrict(raw, out, sigjson.DisallowDuplicateFields, sigjson.DisallowUnknownFields)
	if err != nil || len(problems) > 0 {
		return ErrInvalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return ErrInvalid
	}
	for _, k := range required {
		v := bytes.TrimSpace(fields[k])
		if len(v) == 0 || bytes.Equal(v, []byte("null")) {
			return ErrInvalid
		}
	}
	return nil
}
func (m *ColorSKUMapping) UnmarshalJSON(raw []byte) error {
	type plain ColorSKUMapping
	var p plain
	if err := strictColorJSON(raw, &p, "sourceIndex", "skcIndex", "skuIndex"); err != nil {
		return err
	}
	*m = ColorSKUMapping(p)
	return nil
}
func (m *ColorSharedImage) UnmarshalJSON(raw []byte) error {
	type plain ColorSharedImage
	var p plain
	if err := strictColorJSON(raw, &p, "slotIndex", "colorValueIds"); err != nil {
		return err
	}
	*m = ColorSharedImage(p)
	return nil
}

type ColorSplitSelection struct {
	ColorAttributeID int64              `json:"colorAttributeId"`
	SKUs             []ColorSKUMapping  `json:"skus"`
	SharedImages     []ColorSharedImage `json:"sharedImages"`
}
type ColorSplitRequest struct {
	SourceID        string                   `json:"sourceId"`
	BaseRecordID    string                   `json:"baseRecordId"`
	CompletionDraft CompletionDraftSelection `json:"completionDraft"`
	Selection       ColorSplitSelection      `json:"selection"`
}
type ColorMember struct {
	SourceIndex    int                    `json:"sourceIndex"`
	SupplierSKU    string                 `json:"supplierSku"`
	SaleAttributes []model.AttributeValue `json:"saleAttributes"`
}
type ColorPartition struct {
	FamilyID         string                   `json:"familyId"`
	PartitionID      string                   `json:"partitionId"`
	ColorAttributeID int64                    `json:"colorAttributeId"`
	ColorValueID     int64                    `json:"colorValueId"`
	BaseRecordID     string                   `json:"baseRecordId"`
	CompletionDraft  CompletionDraftSelection `json:"completionDraft"`
	Members          []ColorMember            `json:"members"`
	LocalOnly        bool                     `json:"localOnly"`
}
type ColorSplitGroup struct {
	ColorValueID  int64                 `json:"colorValueId"`
	Input         TargetInput           `json:"input"`
	RulesHash     string                `json:"rulesHash"`
	MerchantHash  string                `json:"merchantHash"`
	InventoryHash string                `json:"inventoryHash"`
	Issues        []goods.OfficialIssue `json:"issues"`
}
type ColorSplitPlan struct {
	Request  ColorSplitRequest `json:"request"`
	FamilyID string            `json:"familyId"`
	Groups   []ColorSplitGroup `json:"groups"`
	PlanHash string            `json:"planHash"`
}
type ColorSplitReceipt struct {
	FamilyID string         `json:"familyId"`
	Records  []TargetRecord `json:"records"`
	Replayed bool           `json:"replayed,omitempty"`
}
type ColorSplitRepository interface {
	ListColorSplitFamilies(context.Context, collection.Scope, string, string) ([]ColorSplitReceipt, error)
	FindColorSplitCommand(context.Context, collection.Scope, string, string) (ColorSplitReceipt, error)
	ReadColorSplitCommand(context.Context, collection.Scope, string) (ColorSplitReceipt, error)
	ReadColorSplitFamily(context.Context, collection.Scope, string) (ColorSplitReceipt, error)
	SaveColorSplit(context.Context, ColorSplitPrepared) (ColorSplitReceipt, error)
}
type ColorSplitPrepared struct {
	scope     collection.Scope
	key, hash string
	plan      ColorSplitPlan
	records   []TargetRecord
	check     func(context.Context) error
}

func (p ColorSplitPrepared) Read(ctx context.Context) (collection.Scope, string, string, ColorSplitPlan, []TargetRecord, error) {
	if ctx == nil || p.check == nil || p.scope.Validate() != nil || !collection.ValidID(p.key) || len(p.records) != 2 {
		return collection.Scope{}, "", "", ColorSplitPlan{}, nil, ErrForbidden
	}
	if err := p.check(ctx); err != nil {
		return collection.Scope{}, "", "", ColorSplitPlan{}, nil, err
	}
	raw, err := json.Marshal(p.records)
	if err != nil || len(raw) > MaxPayloadBytes {
		return collection.Scope{}, "", "", ColorSplitPlan{}, nil, ErrTooLarge
	}
	var copied []TargetRecord
	if json.Unmarshal(raw, &copied) != nil {
		return collection.Scope{}, "", "", ColorSplitPlan{}, nil, ErrUnavailable
	}
	return p.scope, p.key, p.hash, p.plan, copied, nil
}
func ColorFamilyIdentity(s collection.Scope, source, store string, attribute int64) string {
	return collection.StableID(s.OrganizationID, s.ActorID, "shein-color-family-v1", source, store, "shein-us", strconv.FormatInt(attribute, 10))
}
func ColorPartitionIdentity(family string, attribute, value int64) string {
	return collection.StableID(family, "shein-color-partition-v1", strconv.FormatInt(attribute, 10), strconv.FormatInt(value, 10))
}
func ColorTargetIdentity(s collection.Scope, family, partition string) string {
	return collection.StableID(s.OrganizationID, s.ActorID, "shein-color-target-v1", family, partition)
}
func RecordIdentity(s collection.Scope, in TargetInput) string {
	if p := in.ColorPartition; p != nil {
		return ColorTargetIdentity(s, p.FamilyID, p.PartitionID)
	}
	return TargetIdentity(s, in.SourceID, in.StoreID)
}
func RecordPurpose(in TargetInput) (string, string, string) {
	if p := in.ColorPartition; p != nil {
		return "shein_color_local", p.PartitionID, p.FamilyID
	}
	return "standard", "", ""
}
func ValidColorPartition(s collection.Scope, in TargetInput) bool {
	p := in.ColorPartition
	if p == nil {
		return in.TargetID == "" || in.TargetID == TargetIdentity(s, in.SourceID, in.StoreID)
	}
	if !p.LocalOnly || p.ColorAttributeID <= 0 || p.ColorValueID <= 0 || p.FamilyID != ColorFamilyIdentity(s, in.SourceID, in.StoreID, p.ColorAttributeID) || p.PartitionID != ColorPartitionIdentity(p.FamilyID, p.ColorAttributeID, p.ColorValueID) || in.TargetID != RecordIdentity(s, in) || !collection.ValidID(p.BaseRecordID) || !collection.ValidID(p.CompletionDraft.RecordID) || p.CompletionDraft.Revision < 1 || in.SheinImport != nil || in.CompletionDraft != nil || in.Temu != nil || in.TikTok != nil || in.Walmart != nil || len(in.Draft.Product.SKCs) != 1 {
		return false
	}
	skc := in.Draft.Product.SKCs[0]
	if skc.SaleAttribute.AttributeID != p.ColorAttributeID || skc.SaleAttribute.AttributeValueID == nil || *skc.SaleAttribute.AttributeValueID != p.ColorValueID || len(skc.SKUs) != len(p.Members) || len(p.Members) == 0 || len(p.Members) > 256 {
		return false
	}
	seen := map[int]bool{}
	codes := map[string]bool{}
	for i, m := range p.Members {
		if m.SourceIndex < 0 || seen[m.SourceIndex] || m.SupplierSKU == "" || codes[m.SupplierSKU] || skc.SKUs[i].SupplierSKU != m.SupplierSKU || !reflect.DeepEqual(skc.SKUs[i].SaleAttributes, m.SaleAttributes) {
			return false
		}
		seen[m.SourceIndex] = true
		codes[m.SupplierSKU] = true
	}
	return true
}

// ValidateColorEdit consumes the immutable creation pair, never a caller-supplied ownership map.
func ValidateColorEdit(initial ColorSplitReceipt, in TargetInput, base TargetRecord) error {
	var own *TargetRecord
	for i := range initial.Records {
		v := &initial.Records[i]
		if v.TargetID == in.TargetID {
			own = v
		}
	}
	if own == nil || in.ExpectedRevision < 1 || !reflect.DeepEqual(own.Input.ColorPartition, in.ColorPartition) || len(in.Draft.Product.SKCs) != 1 || !reflect.DeepEqual(own.Input.Draft.Product.SKCs[0].SaleAttribute, in.Draft.Product.SKCs[0].SaleAttribute) {
		return ErrForbidden
	}
	if base.ID != in.ColorPartition.BaseRecordID || base.Input.ColorPartition != nil || base.Input.SourceID != in.SourceID || base.Input.StoreID != in.StoreID {
		return ErrForbidden
	}
	// Resolve structural ownership from the immutable parent, including positions
	// intentionally omitted from the children (for example multi-color swatches).
	parentSlots := map[string][]goods.OfficialImageSlot{}
	for _, slot := range base.Input.Draft.Images {
		parentSlots[slot.AssetID] = append(parentSlots[slot.AssetID], slot)
	}
	for _, slot := range in.Draft.Images {
		positions, known := parentSlots[slot.AssetID]
		if !known {
			continue
		} // New approved inventory is still checked by the existing builder.
		allowed := false
		for _, original := range positions {
			if original.Group == "spu" {
				allowed = true
				break
			}
			if original.SKC < 0 || original.SKC >= len(base.Input.Draft.Product.SKCs) {
				return ErrForbidden
			}
			skc := base.Input.Draft.Product.SKCs[original.SKC]
			if skc.SaleAttribute.AttributeID != in.ColorPartition.ColorAttributeID || skc.SaleAttribute.AttributeValueID == nil || *skc.SaleAttribute.AttributeValueID != in.ColorPartition.ColorValueID {
				continue
			}
			if original.Group != "sku" {
				allowed = true
				break
			}
			if original.SKU < 0 || original.SKU >= len(skc.SKUs) {
				return ErrForbidden
			}
			if slot.Group == "sku" && slot.SKU >= 0 && slot.SKU < len(in.ColorPartition.Members) && in.ColorPartition.Members[slot.SKU].SupplierSKU == skc.SKUs[original.SKU].SupplierSKU {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrForbidden
		}
	}
	ownAssets := map[string]bool{}
	otherAssets := map[string]bool{}
	for _, v := range initial.Records {
		for _, slot := range v.Input.Draft.Images {
			if v.TargetID == in.TargetID {
				ownAssets[slot.AssetID] = true
			} else {
				otherAssets[slot.AssetID] = true
			}
		}
	}
	for _, slot := range in.Draft.Images {
		if otherAssets[slot.AssetID] && !ownAssets[slot.AssetID] {
			return ErrForbidden
		}
		if slot.SKC != 0 || slot.Group == "sku" && (slot.SKU < 0 || slot.SKU >= len(in.ColorPartition.Members)) || slot.Group != "sku" && slot.SKU != 0 {
			return ErrInvalid
		}
		if slot.Group != "spu" && slot.Group != "skc" && slot.Group != "detail" && slot.Group != "sku" {
			return ErrInvalid
		}
	}
	return nil
}

type colorDerived struct {
	ColorValueID int64
	OriginalSKC  int
	Draft        goods.OfficialDraftInput
	Members      []ColorMember
}

func deriveColorDrafts(base goods.OfficialDraftInput, candidate completiondraft.Content, sel ColorSplitSelection) ([]colorDerived, error) {
	if sel.ColorAttributeID <= 0 || len(base.Product.SKCs) != 2 || candidate.Structured == nil || len(sel.SKUs) == 0 || len(sel.SKUs) > 256 || len(sel.SKUs) != len(candidate.Structured.Variants) || len(sel.SharedImages) > len(base.Images) {
		return nil, ErrInvalid
	}
	for _, slot := range base.Images {
		switch slot.Group {
		case "spu":
			if slot.SKC != 0 || slot.SKU != 0 {
				return nil, ErrInvalid
			}
		case "skc", "detail", "sku":
			if slot.SKC < 0 || slot.SKC >= len(base.Product.SKCs) || slot.SKU < 0 || slot.Group != "sku" && slot.SKU != 0 || slot.Group == "sku" && slot.SKU >= len(base.Product.SKCs[slot.SKC].SKUs) {
				return nil, ErrInvalid
			}
		default:
			return nil, ErrInvalid
		}
	}
	raw, err := json.Marshal(base)
	if err != nil {
		return nil, ErrInvalid
	}
	var clone goods.OfficialDraftInput
	if json.Unmarshal(raw, &clone) != nil {
		return nil, ErrInvalid
	}
	groups := make([]colorDerived, 2)
	colorIndex := map[int64]int{}
	rowGroup := map[string]int{}
	total := 0
	for i, skc := range clone.Product.SKCs {
		if skc.SaleAttribute.AttributeID != sel.ColorAttributeID || skc.SaleAttribute.AttributeValueID == nil || *skc.SaleAttribute.AttributeValueID <= 0 || len(skc.SKUs) == 0 {
			return nil, ErrInvalid
		}
		value := *skc.SaleAttribute.AttributeValueID
		if _, ok := colorIndex[value]; ok {
			return nil, ErrInvalid
		}
		colorIndex[value] = i
		groups[i] = colorDerived{ColorValueID: value, OriginalSKC: i, Draft: goods.OfficialDraftInput{Product: clone.Product, Images: []goods.OfficialImageSlot{}}}
		groups[i].Draft.Product.SKCs = []model.ProductSKC{skc}
		total += len(skc.SKUs)
	}
	if total != len(sel.SKUs) {
		return nil, ErrInvalid
	}
	sources := map[int]bool{}
	codes := map[string]bool{}
	// Membership is recorded in the preserved baseline SKU order, independently of request order.
	memberAt := map[string]ColorMember{}
	for _, m := range sel.SKUs {
		if m.SourceIndex < 0 || m.SourceIndex >= len(candidate.Structured.Variants) || sources[m.SourceIndex] || m.SKCIndex < 0 || m.SKCIndex > 1 || m.SKUIndex < 0 || m.SKUIndex >= len(base.Product.SKCs[m.SKCIndex].SKUs) {
			return nil, ErrInvalid
		}
		key := fmt.Sprintf("%d/%d", m.SKCIndex, m.SKUIndex)
		if _, ok := rowGroup[key]; ok {
			return nil, ErrInvalid
		}
		sources[m.SourceIndex] = true
		rowGroup[key] = m.SKCIndex
		sku := base.Product.SKCs[m.SKCIndex].SKUs[m.SKUIndex]
		if sku.SupplierSKU == "" || codes[sku.SupplierSKU] {
			return nil, ErrInvalid
		}
		codes[sku.SupplierSKU] = true
		v := candidate.Structured.Variants[m.SourceIndex]
		if v.SourceIndex == nil || *v.SourceIndex != m.SourceIndex {
			return nil, ErrInvalid
		}
		if v.Stock == nil && len(sku.Stock) != 0 || v.Stock != nil && (len(sku.Stock) != 1 || int64(sku.Stock[0].Quantity) != *v.Stock) {
			return nil, fmt.Errorf("%w: SKU %s stock conflicts with fixed candidate", ErrConflict, sku.SupplierSKU)
		}
		memberAt[key] = ColorMember{SourceIndex: m.SourceIndex, SupplierSKU: sku.SupplierSKU, SaleAttributes: sku.SaleAttributes}
	}
	shared := map[int]map[int64]bool{}
	for _, v := range sel.SharedImages {
		if v.SlotIndex < 0 || v.SlotIndex >= len(base.Images) || base.Images[v.SlotIndex].Group != "spu" || len(v.ColorValueIDs) > 2 {
			return nil, ErrInvalid
		}
		if _, ok := shared[v.SlotIndex]; ok {
			return nil, ErrInvalid
		}
		shared[v.SlotIndex] = map[int64]bool{}
		for _, c := range v.ColorValueIDs {
			if _, ok := colorIndex[c]; !ok || shared[v.SlotIndex][c] {
				return nil, ErrInvalid
			}
			shared[v.SlotIndex][c] = true
		}
	}
	for i := range groups {
		g := &groups[i]
		for j := range g.Draft.Product.SKCs[0].SKUs {
			g.Members = append(g.Members, memberAt[fmt.Sprintf("%d/%d", i, j)])
		}
		g.Draft.Product.SizeAttributes = nil
		for _, v := range clone.Product.SizeAttributes {
			if v.RelatedSaleAttributeID == nil && v.RelatedSaleAttributeValueID == nil {
				g.Draft.Product.SizeAttributes = append(g.Draft.Product.SizeAttributes, v)
				continue
			}
			if v.RelatedSaleAttributeID == nil || v.RelatedSaleAttributeValueID == nil {
				return nil, ErrInvalid
			}
			for _, sku := range g.Draft.Product.SKCs[0].SKUs {
				found := false
				for _, a := range sku.SaleAttributes {
					found = found || a.AttributeID == *v.RelatedSaleAttributeID && a.AttributeValueID != nil && *a.AttributeValueID == *v.RelatedSaleAttributeValueID
				}
				if found {
					g.Draft.Product.SizeAttributes = append(g.Draft.Product.SizeAttributes, v)
					break
				}
			}
		}
		for n, slot := range clone.Images {
			if slot.Group == "spu" {
				if shared[n][g.ColorValueID] {
					g.Draft.Images = append(g.Draft.Images, slot)
				}
				continue
			}
			if slot.SKC != i {
				continue
			}
			if slot.Group != "skc" && slot.Group != "sku" && slot.Group != "detail" {
				return nil, ErrInvalid
			}
			if slot.Group == "sku" && (slot.SKU < 0 || slot.SKU >= len(g.Members)) {
				return nil, ErrInvalid
			}
			if slot.Type == 6 {
				continue
			}
			slot.SKC = 0
			g.Draft.Images = append(g.Draft.Images, slot)
		}
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].ColorValueID < groups[j].ColorValueID })
	return groups, nil
}

func colorPlanHash(p ColorSplitPlan) string {
	p.PlanHash = ""
	raw, _ := json.Marshal(p)
	var copy ColorSplitPlan
	_ = json.Unmarshal(raw, &copy)
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				visit(v.Field(i))
			}
		case reflect.Pointer:
			if !v.IsNil() {
				visit(v.Elem())
			}
		case reflect.Slice:
			if v.Len() == 0 {
				v.Set(reflect.Zero(v.Type()))
			} else {
				for i := 0; i < v.Len(); i++ {
					visit(v.Index(i))
				}
			}
		}
	}
	visit(reflect.ValueOf(&copy).Elem())
	return collection.Digest(copy)
}
func (s *TargetService) colorRepo() (ColorSplitRepository, error) {
	r, ok := s.dependencies.Records.(ColorSplitRepository)
	if !ok {
		return nil, ErrUnavailable
	}
	return r, nil
}
func (s *TargetService) prepareColorSplit(ctx context.Context, scope collection.Scope, in ColorSplitRequest) (ColorSplitPlan, []TargetRecord, error) {
	raw, e := json.Marshal(in)
	if e != nil || len(raw) > MaxPayloadBytes {
		return ColorSplitPlan{}, nil, ErrTooLarge
	}
	var own ColorSplitRequest
	if json.Unmarshal(raw, &own) != nil {
		return ColorSplitPlan{}, nil, ErrInvalid
	}
	in = own

	selected, err := s.dependencies.Sources.Select(ctx, in.SourceID)
	if err != nil {
		return ColorSplitPlan{}, nil, err
	}
	owner, source, _, err := selected.Read(ctx)
	if err != nil || owner != scope {
		return ColorSplitPlan{}, nil, ErrForbidden
	}
	base, candidate, err := s.readImport(ctx, scope, SheinImportRequest{SourceID: in.SourceID, BaseRecordID: in.BaseRecordID, CompletionDraft: in.CompletionDraft}, source)
	if err != nil {
		return ColorSplitPlan{}, nil, err
	}
	if base.Input.ColorPartition != nil || base.Input.Walmart != nil || base.Merchant.ApplicationType != "self_operated" {
		return ColorSplitPlan{}, nil, ErrInvalid
	}
	head, err := s.dependencies.Records.ReadTargetHead(ctx, scope, base.TargetID)
	if err != nil {
		return ColorSplitPlan{}, nil, err
	}
	if head.ID != base.ID || head.Revision != base.Revision {
		return ColorSplitPlan{}, nil, ErrConflict
	}
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, selected, base.EffectiveVersion, base.ApplyReceiptID)
	if err != nil {
		return ColorSplitPlan{}, nil, err
	}
	derived, err := deriveColorDrafts(base.Input.Draft, candidate.Content, in.Selection)
	if err != nil {
		return ColorSplitPlan{}, nil, err
	}
	if len(product.Snapshot.Variants) != len(in.Selection.SKUs) {
		return ColorSplitPlan{}, nil, ErrConflict
	}
	for _, g := range derived {
		for _, m := range g.Members {
			if product.Snapshot.Variants[m.SourceIndex].SKU != m.SupplierSKU {
				return ColorSplitPlan{}, nil, ErrConflict
			}
		}
	}
	inventoryScope := asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: source.Source.ProductKey, TargetPlatform: "shein", SourceSnapshotVersion: product.Version}
	inventory, err := s.dependencies.Assets.GetApprovedInventory(ctx, inventoryScope)
	if errors.Is(err, asset.ErrApprovedAssetsNotReady) {
		inventory = asset.ApprovedAssetInventory{Scope: inventoryScope, Assets: []asset.ApprovedAsset{}}
	} else if err != nil {
		return ColorSplitPlan{}, nil, ErrUnavailable
	}
	if inventory.Scope != inventoryScope {
		return ColorSplitPlan{}, nil, ErrForbidden
	}
	family := ColorFamilyIdentity(scope, in.SourceID, base.Input.StoreID, in.Selection.ColorAttributeID)
	// Canonical selection order has no effect on membership or command hashes.
	sort.Slice(in.Selection.SKUs, func(i, j int) bool { return in.Selection.SKUs[i].SourceIndex < in.Selection.SKUs[j].SourceIndex })
	sort.Slice(in.Selection.SharedImages, func(i, j int) bool {
		return in.Selection.SharedImages[i].SlotIndex < in.Selection.SharedImages[j].SlotIndex
	})
	for i := range in.Selection.SharedImages {
		sort.Slice(in.Selection.SharedImages[i].ColorValueIDs, func(a, b int) bool {
			return in.Selection.SharedImages[i].ColorValueIDs[a] < in.Selection.SharedImages[i].ColorValueIDs[b]
		})
	}
	plan := ColorSplitPlan{Request: in, FamilyID: family, Groups: []ColorSplitGroup{}}
	var records []TargetRecord
	assets := map[string]bool{}
	for _, a := range inventory.Assets {
		assets[a.ID] = true
	}
	for _, g := range derived {
		for _, slot := range g.Draft.Images {
			if !assets[slot.AssetID] {
				return ColorSplitPlan{}, nil, ErrNotReady
			}
		}
		partition := &ColorPartition{FamilyID: family, PartitionID: ColorPartitionIdentity(family, in.Selection.ColorAttributeID, g.ColorValueID), ColorAttributeID: in.Selection.ColorAttributeID, ColorValueID: g.ColorValueID, BaseRecordID: base.ID, CompletionDraft: in.CompletionDraft, Members: g.Members, LocalOnly: true}
		input := TargetInput{SourceID: source.ID, StoreID: base.Input.StoreID, EffectiveVersion: base.EffectiveVersion, ApplyReceiptID: base.ApplyReceiptID, Draft: g.Draft, ColorPartition: partition}
		input.TargetID = RecordIdentity(scope, input)
		prepared, e := s.preparePlatform(ctx, scope, input, inventory, product)
		if e != nil {
			return ColorSplitPlan{}, nil, e
		}
		if !validImportMerchant(scope, input.StoreID, prepared.merchant, prepared.sheinRules) || prepared.merchant.ApplicationType != "self_operated" {
			return ColorSplitPlan{}, nil, ErrNotReady
		}
		colorOK := false
		for _, a := range prepared.sheinRules.Attributes.Attributes {
			if a.ID == in.Selection.ColorAttributeID && a.Type != nil && *a.Type == 1 && a.MainLabel != nil && *a.MainLabel == 1 && a.Show != nil && *a.Show == 1 && a.Status != nil && *a.Status != 1 {
				for _, v := range a.Options {
					colorOK = colorOK || v.ID == g.ColorValueID && v.Show != nil && *v.Show == 1
				}
			}
		}
		if !colorOK {
			return ColorSplitPlan{}, nil, ErrNotReady
		}
		requirement, e := goods.ResolveOfficialImageRequirements(input.Draft.Product, prepared.sheinRules.Fill, input.Draft.Images)
		if e != nil {
			return ColorSplitPlan{}, nil, ErrNotReady
		}
		for _, slot := range input.Draft.Images {
			if !requirement.AllowsSlot(slot) {
				return ColorSplitPlan{}, nil, ErrInvalid
			}
		}
		group := ColorSplitGroup{ColorValueID: g.ColorValueID, Input: input, RulesHash: prepared.rulesHash, MerchantHash: collection.Digest(prepared.merchant), InventoryHash: collection.Digest(inventory), Issues: prepared.shein.Issues}
		plan.Groups = append(plan.Groups, group)
		records = append(records, TargetRecord{TargetID: input.TargetID, Revision: 1, Source: source, EffectiveVersion: input.EffectiveVersion, ApplyReceiptID: input.ApplyReceiptID, Input: input, ProductHash: collection.Digest(product.Snapshot), InventoryHash: group.InventoryHash, RulesHash: group.RulesHash, Merchant: prepared.merchant, Result: prepared.shein, ImageObservations: prepared.images, ProofObservations: prepared.proofs})
	}
	plan.PlanHash = colorPlanHash(plan)
	raw, _ = json.Marshal(plan)
	if len(raw) > MaxPayloadBytes {
		return ColorSplitPlan{}, nil, ErrTooLarge
	}
	return plan, records, nil
}
func (s *TargetService) PreviewColorSplit(ctx context.Context, in ColorSplitRequest) (ColorSplitPlan, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorizeTarget(ctx, preparation.PermissionManage)
	if err != nil {
		return ColorSplitPlan{}, err
	}
	p, _, err := s.prepareColorSplit(ctx, scope, in)
	if err == nil {
		_, err = s.authorizeTarget(ctx, preparation.PermissionManage)
	}
	return p, err
}
func (s *TargetService) CreateColorSplit(ctx context.Context, key string, plan ColorSplitPlan) (ColorSplitReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorizeTarget(ctx, preparation.PermissionManage)
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	repo, err := s.colorRepo()
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	raw, e := json.Marshal(plan)
	if e != nil || len(raw) > MaxPayloadBytes || !collection.ValidID(key) || plan.PlanHash != colorPlanHash(plan) {
		return ColorSplitReceipt{}, ErrInvalid
	}
	hash := colorPlanHash(plan)
	if prior, e := repo.FindColorSplitCommand(ctx, scope, key, hash); e == nil {
		return prior, nil
	} else if !errors.Is(e, ErrNotFound) {
		return ColorSplitReceipt{}, e
	}
	actual, records, err := s.prepareColorSplit(ctx, scope, plan.Request)
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	if colorPlanHash(plan) != colorPlanHash(actual) {
		return ColorSplitReceipt{}, ErrConflict
	}
	// Slow rule/image preparation must not consume an expired initial Source proof.
	fresh, err := s.dependencies.Sources.Select(ctx, plan.Request.SourceID)
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	owner, source, _, err := fresh.Read(ctx)
	if err != nil || owner != scope || source != records[0].Source {
		return ColorSplitReceipt{}, ErrForbidden
	}
	check := func(c context.Context) error {
		current, e := s.authorizeTarget(c, preparation.PermissionManage)
		if e != nil || current != scope {
			return ErrForbidden
		}
		o, src, _, e := fresh.Read(c)
		if e != nil || o != scope || src != source {
			return ErrForbidden
		}
		candidate, e := s.dependencies.CompletionVersions.ReadVersion(c, scope, plan.Request.CompletionDraft.RecordID)
		if e != nil {
			return e
		}
		if candidate.Source != source || candidate.Revision != plan.Request.CompletionDraft.Revision {
			return ErrConflict
		}
		return nil
	}
	for i := range records {
		records[i].ID = collection.StableID(scope.OrganizationID, scope.ActorID, "shein-color-record-v1", key, records[i].Input.ColorPartition.PartitionID)
		records[i].CreatedAt = time.Now().UTC()
	}
	return repo.SaveColorSplit(ctx, ColorSplitPrepared{scope: scope, key: key, hash: hash, plan: actual, records: records, check: check})
}
func (s *TargetService) ReadColorSplitCommand(ctx context.Context, key string) (ColorSplitReceipt, error) {
	scope, err := s.authorizeTarget(ctx, preparation.PermissionRead)
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	r, err := s.colorRepo()
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	return r.ReadColorSplitCommand(ctx, scope, key)
}
func (s *TargetService) ReadColorSplitFamily(ctx context.Context, source, family string) (ColorSplitReceipt, error) {
	scope, err := s.authorizeTarget(ctx, preparation.PermissionRead)
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	selected, err := s.dependencies.Sources.Select(ctx, source)
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	o, src, _, err := selected.Read(ctx)
	if err != nil || o != scope || src.ID != source {
		return ColorSplitReceipt{}, ErrForbidden
	}
	r, err := s.colorRepo()
	if err != nil {
		return ColorSplitReceipt{}, err
	}
	value, err := r.ReadColorSplitFamily(ctx, scope, family)
	if err != nil {
		return value, err
	}
	for _, v := range value.Records {
		if v.Source != src {
			return ColorSplitReceipt{}, ErrForbidden
		}
	}
	return value, nil
}

func (s *TargetService) ListColorSplitFamilies(ctx context.Context, source, store string) ([]ColorSplitReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorizeTarget(ctx, preparation.PermissionRead)
	if err != nil {
		return nil, err
	}
	if !collection.ValidID(source) || !collection.ValidID(store) {
		return nil, ErrInvalid
	}
	sel, err := s.dependencies.Sources.Select(ctx, source)
	if err != nil {
		return nil, err
	}
	o, src, _, err := sel.Read(ctx)
	if err != nil || o != scope || src.ID != source {
		return nil, ErrForbidden
	}
	r, err := s.colorRepo()
	if err != nil {
		return nil, err
	}
	values, err := r.ListColorSplitFamilies(ctx, scope, source, store)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		for _, child := range v.Records {
			if child.Source != src || child.Input.StoreID != store {
				return nil, ErrForbidden
			}
		}
	}
	return values, nil
}
