package target

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
)

const SheinSalesMappingVersion = "shein-sales-mapping-v1"

type SheinSalesDimension struct {
	SourceName  string `json:"sourceName"`
	AttributeID int64  `json:"attributeId"`
	Main        bool   `json:"main"`
}
type SheinSalesValue struct {
	SourceName  string `json:"sourceName"`
	SourceValue string `json:"sourceValue"`
	OptionID    int64  `json:"optionId"`
}
type SheinSalesRow struct {
	SourceIndex int `json:"sourceIndex"`
	SKCIndex    int `json:"skcIndex"`
	SKUIndex    int `json:"skuIndex"`
}
type SheinSalesImage struct {
	SlotIndex int `json:"slotIndex"`
}

func (r *SheinSalesRow) UnmarshalJSON(raw []byte) error {
	type plain SheinSalesRow
	var v plain
	if err := strictColorJSON(raw, &v, "sourceIndex", "skcIndex", "skuIndex"); err != nil {
		return err
	}
	*r = SheinSalesRow(v)
	return nil
}
func (r *SheinSalesImage) UnmarshalJSON(raw []byte) error {
	type plain SheinSalesImage
	var v plain
	if err := strictColorJSON(raw, &v, "slotIndex"); err != nil {
		return err
	}
	*r = SheinSalesImage(v)
	return nil
}

type SheinSalesSelection struct {
	Dimensions []SheinSalesDimension `json:"dimensions"`
	Values     []SheinSalesValue     `json:"values"`
	Rows       []SheinSalesRow       `json:"rows"`
	Images     []SheinSalesImage     `json:"images"`
}
type SheinSalesRequest struct {
	SourceID        string                   `json:"sourceId"`
	BaseRecordID    string                   `json:"baseRecordId"`
	CompletionDraft CompletionDraftSelection `json:"completionDraft"`
	Selection       SheinSalesSelection      `json:"selection"`
}
type SheinSalesMapping struct {
	Version         string                   `json:"version"`
	TargetID        string                   `json:"targetId"`
	BaseRecordID    string                   `json:"baseRecordId"`
	CompletionDraft CompletionDraftSelection `json:"completionDraft"`
	Selection       SheinSalesSelection      `json:"selection"`
	RulesHash       string                   `json:"rulesHash"`
	MerchantHash    string                   `json:"merchantHash"`
	InventoryHash   string                   `json:"inventoryHash"`
}
type SheinSalesPreview struct {
	Request    SheinSalesRequest        `json:"request"`
	Attributes *model.AttributeTemplate `json:"attributes,omitempty"`
	Input      *TargetInput             `json:"input,omitempty"`
	Issues     []goods.OfficialIssue    `json:"issues"`
}

// Safe field diagnostics are returned only after scoped immutable facts have
// been read. They never expose a repository/credential/provider error.
type SheinSalesFieldError struct {
	Field   string
	Message string
}

func (e *SheinSalesFieldError) Error() string { return e.Field + ": " + e.Message }
func (e *SheinSalesFieldError) Unwrap() error { return ErrInvalid }
func salesField(field, message string) error  { return &SheinSalesFieldError{field, message} }
func salesRowField(index int, code, field, message string) error {
	label := map[string]string{"stock": "库存", "saleAttributes": "销售规格", "attributes": "来源规格"}[field]
	if label == "" {
		label = field
	}
	return salesField(fmt.Sprintf("source.variants.%d.%s", index, field), fmt.Sprintf("SKU %d（%s）/ %s：%s", index+1, code, label, message))
}

func normalizeSalesSelection(s *SheinSalesSelection) {
	sort.Slice(s.Dimensions, func(i, j int) bool { return s.Dimensions[i].SourceName < s.Dimensions[j].SourceName })
	sort.Slice(s.Values, func(i, j int) bool {
		a, b := s.Values[i], s.Values[j]
		if a.SourceName != b.SourceName {
			return a.SourceName < b.SourceName
		}
		return a.SourceValue < b.SourceValue
	})
	sort.Slice(s.Rows, func(i, j int) bool { return s.Rows[i].SourceIndex < s.Rows[j].SourceIndex })
	sort.Slice(s.Images, func(i, j int) bool { return s.Images[i].SlotIndex < s.Images[j].SlotIndex })
}

func deriveSheinSales(base goods.OfficialDraftInput, source catalog.ProductSnapshot, candidate completiondraft.Content, selection SheinSalesSelection, rules goods.OfficialRuleSnapshot, inventory asset.ApprovedAssetInventory) (goods.OfficialDraftInput, error) {
	fail := func(field, msg string) (goods.OfficialDraftInput, error) {
		return goods.OfficialDraftInput{}, salesField(field, msg)
	}
	n := len(source.Variants)
	if rules.ApplicationMode != model.ModeSelfOperated {
		return fail("application_type", "销售属性映射目前仅支持 SHEIN US 自营草稿")
	}
	if n == 0 || n > 256 || len(selection.Rows) != n || candidate.Structured == nil || len(candidate.Structured.Variants) != n {
		return fail("rows", "请完整对应固定来源与候选的全部 SKU（最多256行）")
	}
	if len(selection.Dimensions) < 1 || len(selection.Dimensions) > 3 || len(selection.Values) > 768 || len(selection.Images) != len(base.Images) {
		return fail("selection", "请完整确认销售维度、值及已有图片归属")
	}
	attrs := map[int64]model.Attribute{}
	for _, a := range rules.Attributes.Attributes {
		attrs[a.ID] = a
	}
	dims := map[string]SheinSalesDimension{}
	ids := map[int64]bool{}
	mainName := ""
	for _, d := range selection.Dimensions {
		a, ok := attrs[d.AttributeID]
		if d.SourceName == "" || len(d.SourceName) > 200 || dims[d.SourceName].SourceName != "" || ids[d.AttributeID] || !ok || a.Type == nil || *a.Type != 1 || a.Status == nil || *a.Status == 1 || a.Show == nil || *a.Show != 1 || a.Mode == nil || *a.Mode < 1 || *a.Mode > 3 {
			return fail("dimensions", "请选择唯一、可用的店铺销售属性与已有枚举值")
		}
		if d.Main {
			if mainName != "" || a.MainLabel == nil || *a.MainLabel != 1 {
				return fail("dimensions", "必须且只能选择一个官方主销售属性")
			}
			mainName = d.SourceName
		}
		dims[d.SourceName] = d
		ids[d.AttributeID] = true
	}
	if mainName == "" {
		return fail("dimensions", "请选择一个主销售属性")
	}
	type valueKey struct{ name, value string }
	values := map[valueKey]int64{}
	for _, v := range selection.Values {
		d, ok := dims[v.SourceName]
		key := valueKey{v.SourceName, v.SourceValue}
		if !ok || v.SourceValue == "" || len(v.SourceValue) > 8192 || values[key] != 0 || v.OptionID <= 0 {
			return fail("values", "每个来源规格值只对应一个明确的平台值")
		}
		found := false
		for _, o := range attrs[d.AttributeID].Options {
			if o.ID == v.OptionID && o.Show != nil && *o.Show == 1 {
				found = true
			}
		}
		if !found {
			return fail("values", fmt.Sprintf("%s / %s：所选值已停用或不属于当前店铺属性", v.SourceName, v.SourceValue))
		}
		values[key] = v.OptionID
	}
	raw, err := json.Marshal(base)
	if err != nil {
		return fail("draft", "草稿内容无效")
	}
	var out goods.OfficialDraftInput
	if json.Unmarshal(raw, &out) != nil {
		return fail("draft", "草稿内容无效")
	}
	type position struct{ skc, sku int }
	oldRows := map[int]position{}
	used := map[position]bool{}
	codes := map[string]bool{}
	total := 0
	for _, g := range base.Product.SKCs {
		total += len(g.SKUs)
	}
	if total != n {
		return fail("rows", "目标与固定来源的 SKU 数量不同，请回原编辑流程核对")
	}
	for _, r := range selection.Rows {
		pos := position{r.SKCIndex, r.SKUIndex}
		if r.SourceIndex < 0 || r.SourceIndex >= n || r.SKCIndex < 0 || r.SKCIndex >= len(base.Product.SKCs) || r.SKUIndex < 0 || r.SKUIndex >= len(base.Product.SKCs[r.SKCIndex].SKUs) || used[pos] {
			return fail("rows", "来源与目标位置必须完整一对一，不可遗漏或重复")
		}
		if _, ok := oldRows[r.SourceIndex]; ok {
			return fail("rows", "来源行不可重复")
		}
		original := source.Variants[r.SourceIndex]
		sku := base.Product.SKCs[r.SKCIndex].SKUs[r.SKUIndex]
		if original.SKU == "" || sku.SupplierSKU != original.SKU || codes[sku.SupplierSKU] {
			return fail(fmt.Sprintf("source.variants.%d.sku", r.SourceIndex), "请核对准确来源编码；编码不能为空、重复或与目标不符")
		}
		c := candidate.Structured.Variants[r.SourceIndex]
		if c.SourceIndex == nil || *c.SourceIndex != r.SourceIndex {
			return fail("completionDraft", "固定候选的 SKU 身份不完整")
		}
		if c.Stock == nil && len(sku.Stock) != 0 || c.Stock != nil && (len(sku.Stock) != 1 || int64(sku.Stock[0].Quantity) != *c.Stock) {
			return goods.OfficialDraftInput{}, salesRowField(r.SourceIndex, sku.SupplierSKU, "stock", "候选与目标库存不同或仓库有歧义，请先完成现有数值带入")
		}
		oldRows[r.SourceIndex] = pos
		used[pos] = true
		codes[sku.SupplierSKU] = true
	}
	out.Product.SKCs = nil
	dest := map[position]position{}
	mainGroups := map[int64]int{}
	oldGroups := map[int]map[int]bool{}
	newGroups := map[int]map[int]bool{}
	usedValues := map[valueKey]bool{}
	for i, original := range source.Variants {
		pos, ok := oldRows[i]
		if !ok {
			return fail("rows", "来源行未完整覆盖")
		}
		specs := map[string]string{}
		for _, a := range original.Attributes {
			if _, duplicate := specs[a.Name]; duplicate {
				return goods.OfficialDraftInput{}, salesRowField(i, original.SKU, "attributes", "来源规格名称重复")
			}
			specs[a.Name] = a.Value
		}
		var main model.AttributeValue
		secondary := []model.AttributeValue{}
		for _, d := range selection.Dimensions {
			key := valueKey{d.SourceName, specs[d.SourceName]}
			option := values[key]
			if option == 0 {
				return goods.OfficialDraftInput{}, salesRowField(i, original.SKU, d.SourceName, "请确认此来源值对应的平台枚举值")
			}
			usedValues[key] = true
			v := model.AttributeValue{AttributeID: d.AttributeID, AttributeValueID: &option}
			if d.Main {
				main = v
			} else {
				secondary = append(secondary, v)
			}
		}
		sort.Slice(secondary, func(i, j int) bool { return secondary[i].AttributeID < secondary[j].AttributeID })
		g, exists := mainGroups[*main.AttributeValueID]
		if !exists {
			g = len(out.Product.SKCs)
			mainGroups[*main.AttributeValueID] = g
			out.Product.SKCs = append(out.Product.SKCs, model.ProductSKC{SaleAttribute: main, SKUs: []model.ProductSKU{}})
		}
		if oldGroups[pos.skc] == nil {
			oldGroups[pos.skc] = map[int]bool{}
		}
		oldGroups[pos.skc][g] = true
		if newGroups[g] == nil {
			newGroups[g] = map[int]bool{}
		}
		newGroups[g][pos.skc] = true
		sku := outSKUCopy(base.Product.SKCs[pos.skc].SKUs[pos.sku])
		sku.SaleAttributes = secondary
		for _, existing := range out.Product.SKCs[g].SKUs {
			if reflect.DeepEqual(existing.SaleAttributes, secondary) {
				return goods.OfficialDraftInput{}, salesRowField(i, original.SKU, "saleAttributes", "映射后与同一颜色组的另一 SKU 规格重复")
			}
		}
		next := position{g, len(out.Product.SKCs[g].SKUs)}
		dest[pos] = next
		out.Product.SKCs[g].SKUs = append(out.Product.SKCs[g].SKUs, sku)
	}
	if len(usedValues) != len(values) {
		return fail("values", "存在不属于固定来源的额外规格值")
	}
	if len(out.Product.SKCs) > 40 || rules.Attributes.MainAttributeStatus != nil && *rules.Attributes.MainAttributeStatus == 1 && len(out.Product.SKCs) != 1 {
		return fail("dimensions", "当前类目不允许这些主销售属性分组")
	}
	for old, groups := range oldGroups {
		metadata := base.Product.SKCs[old]
		metadata.SaleAttribute = model.AttributeValue{}
		metadata.SKUs = nil
		if !reflect.DeepEqual(metadata, model.ProductSKC{}) && !emptySKCMetadata(metadata) {
			if len(groups) != 1 {
				return fail(fmt.Sprintf("skc_list.%d", old), "非空颜色组资料不能自动分配给多个新组")
			}
			for g := range groups {
				if len(newGroups[g]) != 1 {
					return fail(fmt.Sprintf("skc_list.%d", old), "多个颜色组的非空资料不能自动合并")
				}
				skus, sale := out.Product.SKCs[g].SKUs, out.Product.SKCs[g].SaleAttribute
				out.Product.SKCs[g] = metadata
				out.Product.SKCs[g].SKUs = skus
				out.Product.SKCs[g].SaleAttribute = sale
			}
		}
	}
	seenImages := map[int]bool{}
	assets := map[string]asset.ApprovedAsset{}
	for _, a := range inventory.Assets {
		assets[a.ID] = a
	}
	out.Images = nil
	for _, choice := range selection.Images {
		if choice.SlotIndex < 0 || choice.SlotIndex >= len(base.Images) || seenImages[choice.SlotIndex] {
			return fail("images", "已有图片槽必须逐个确认且不能重复")
		}
		seenImages[choice.SlotIndex] = true
		slot := base.Images[choice.SlotIndex]
		a, ok := assets[slot.AssetID]
		if !ok {
			return fail(fmt.Sprintf("images.%d", choice.SlotIndex), "图片不在本次准确已批准素材集合中")
		}
		sourceIndex := -1
		switch slot.Group {
		case "spu":
			if slot.SKC != 0 || slot.SKU != 0 {
				return fail("images", "公共图片位置无效")
			}
		case "sku":
			next, ok := dest[position{slot.SKC, slot.SKU}]
			if !ok {
				return fail("images", "SKU 图片原位置不存在")
			}
			for i, p := range oldRows {
				if p == (position{slot.SKC, slot.SKU}) {
					sourceIndex = i
				}
			}
			slot.SKC, slot.SKU = next.skc, next.sku
		case "skc", "detail":
			groups := oldGroups[slot.SKC]
			if slot.SKU != 0 || len(groups) != 1 {
				return fail(fmt.Sprintf("images.%d", choice.SlotIndex), "颜色组图片不能自动分配给多个颜色；请先明确原图片归属")
			}
			for g := range groups {
				slot.SKC = g
			}
		default:
			return fail("images", "图片用途未开放")
		}
		if a.OfficialPlacement != nil {
			p := a.OfficialPlacement
			if p.Site != "shein-us" || p.Group != slot.Group || p.SKC != slot.SKC || p.SKU != slot.SKU || p.Type != slot.Type || p.Sort != slot.Sort {
				return fail(fmt.Sprintf("images.%d", choice.SlotIndex), "新位置与原素材批准位置不一致，请回图片流程处理")
			}
		}
		if a.Presentation != nil && a.Presentation.VariantID != "" && (sourceIndex < 0 || source.Variants[sourceIndex].SourceID == "" || a.Presentation.VariantID != source.Variants[sourceIndex].SourceID) {
			return fail(fmt.Sprintf("images.%d", choice.SlotIndex), "图片批准的专属 SKU 与本次行归属不符")
		}
		out.Images = append(out.Images, slot)
	}
	// References outside the reordered arrays must still describe an actual row.
	has := func(attribute, value int64) bool {
		for _, g := range out.Product.SKCs {
			for _, sku := range g.SKUs {
				for _, v := range sku.SaleAttributes {
					if v.AttributeID == attribute && v.AttributeValueID != nil && *v.AttributeValueID == value {
						return true
					}
				}
			}
		}
		return false
	}
	for i, size := range out.Product.SizeAttributes {
		if (size.RelatedSaleAttributeID == nil) != (size.RelatedSaleAttributeValueID == nil) || size.RelatedSaleAttributeID != nil && !has(*size.RelatedSaleAttributeID, *size.RelatedSaleAttributeValueID) {
			return fail(fmt.Sprintf("size_attribute_list.%d", i), "尺码表关联未对应实际映射后的规格")
		}
	}
	if sample := out.Product.Sample; sample != nil {
		matched := false
		for _, g := range out.Product.SKCs {
			if g.SaleAttribute.AttributeID != sample.Spec.Main.AttributeID || g.SaleAttribute.AttributeValueID == nil || *g.SaleAttribute.AttributeValueID != sample.Spec.Main.ValueID {
				continue
			}
			for _, sku := range g.SKUs {
				all := true
				for _, sub := range sample.Spec.Sub {
					found := false
					for _, v := range sku.SaleAttributes {
						if strconv.FormatInt(v.AttributeID, 10) == sub.AttributeID && v.AttributeValueID != nil && strconv.FormatInt(*v.AttributeValueID, 10) == sub.ValueID {
							found = true
						}
					}
					all = all && found
				}
				matched = matched || all
			}
		}
		if !matched {
			return fail("sample_info", "样品关联未对应实际映射后的 SKU")
		}
	}
	return out, nil
}
func outSKUCopy(s model.ProductSKU) model.ProductSKU {
	raw, _ := json.Marshal(s)
	var copy model.ProductSKU
	_ = json.Unmarshal(raw, &copy)
	return copy
}
func emptySKCMetadata(s model.ProductSKC) bool {
	raw, _ := json.Marshal(s)
	var fields map[string]any
	_ = json.Unmarshal(raw, &fields)
	delete(fields, "sale_attribute")
	delete(fields, "sku_list")
	delete(fields, "image_info")
	return len(fields) == 0 && len(s.ImageInfo.Images) == 0
}
