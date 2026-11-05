package workspace

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	shein "task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	"task-processor/internal/marketplace/shein/publishing"
)

const maxShownSKUs = 4

// BuildOriginalSKUReadinessChecks reads the immutable draft and its preview before
// submit preparation. Defaults cannot supply business evidence.
func BuildOriginalSKUReadinessChecks(pkg *shein.Package) []ReadinessCheckSpec {
	var draft []shein.SKCRequestDraft
	var preview []previewdata.SKC
	if pkg.DraftPayload != nil {
		draft = pkg.DraftPayload.SKCList
	}
	if pkg.PreviewPayload != nil {
		preview = pkg.PreviewPayload.SKCList
	}
	var gaps, known, paths []string
	missing, declared := 0, 0
	for skc := range max(len(draft), len(preview)) {
		var rows []shein.SKUDraft
		var previews []previewdata.SKU
		if skc < len(draft) {
			rows = draft[skc].SKUList
		}
		if skc < len(preview) {
			previews = preview[skc].SKUS
		}
		skcMatches := skc < len(draft) && skc < len(preview)
		if skcMatches && draft[skc].SupplierCode != "" && preview[skc].SupplierCode != nil {
			skcMatches = draft[skc].SupplierCode == *preview[skc].SupplierCode
		}
		for sku := range max(len(rows), len(previews)) {
			name := ""
			if sku < len(rows) {
				name = rows[sku].SupplierSKU
			} else {
				name = previews[sku].SupplierSKU
			}
			label := skuLabel(skc, sku, name)
			var fields []string
			if !skcMatches || sku >= len(rows) || sku >= len(previews) || rows[sku].SupplierSKU != previews[sku].SupplierSKU {
				fields = []string{"无法核对草稿与预览中的规格"}
			} else {
				source, original := rows[sku], previews[sku]
				switch {
				case !source.OwnStockDeclared:
					fields = append(fields, "库存待补")
				case !matchingDeclaredStock(source, original):
					fields = append(fields, "库存声明与规格资料不一致")
				default:
					declared++
					if len(known) < maxShownSKUs {
						known = append(known, fmt.Sprintf("%s 库存 %d", label, source.StockCount))
					}
				}
				fields = append(fields, packagingGaps(original)...)
				if source.Weight != original.Weight || source.WeightUnit != original.WeightUnit ||
					source.Length != original.Length || source.Width != original.Width ||
					source.Height != original.Height || source.LengthUnit != original.LengthUnit {
					fields = append(fields, "草稿与预览的包装资料不一致")
				}
			}
			if len(fields) == 0 {
				continue
			}
			missing++
			if len(gaps) < maxShownSKUs {
				gaps = append(gaps, label+" "+strings.Join(fields, "、"))
				paths = append(paths, fmt.Sprintf("shein.preview_payload.skc_list[%d].sku_list[%d]", skc, sku))
			}
		}
	}
	if missing == 0 && declared == 0 {
		return nil
	}
	message := summarizedSKUs(gaps, missing)
	if len(known) > 0 {
		if message != "" {
			message += "。"
		}
		message += summarizedSKUs(known, declared)
	}
	return []ReadinessCheckSpec{BuildSubmitReadinessCheck(
		"sku_facts", "SKU 库存和包装", missing == 0, message, paths, "核对 SKU 库存及包装资料", false,
	)}
}

func matchingDeclaredStock(draft shein.SKUDraft, preview previewdata.SKU) bool {
	return draft.StockCount >= 0 && int64(draft.StockCount) <= 9007199254740991 &&
		len(draft.StockInfoList) == 1 && len(preview.StockInfoList) == 1 &&
		preview.StockCount != nil && *preview.StockCount == draft.StockCount &&
		draft.StockInfoList[0].InventoryNum == draft.StockCount &&
		preview.StockInfoList[0].InventoryNum == draft.StockCount &&
		strings.TrimSpace(draft.StockInfoList[0].WarehouseCode) != "" &&
		draft.StockInfoList[0].WarehouseCode == preview.StockInfoList[0].MerchantWarehouseCode
}

func packagingGaps(sku previewdata.SKU) []string {
	var result []string
	grams, knownUnit := publishing.SKUWeightGrams(sku.Weight, sku.WeightUnit)
	switch {
	case sku.Weight == 0 && strings.TrimSpace(sku.WeightUnit) == "":
		result = append(result, "重量待补")
	case sku.Weight <= 0 || math.IsNaN(sku.Weight) || math.IsInf(sku.Weight, 0):
		result = append(result, "重量无效")
	case strings.TrimSpace(sku.WeightUnit) == "":
		result = append(result, "重量单位待补")
	case !knownUnit:
		result = append(result, "重量单位无效")
	case !publishing.ValidSKUWeightGrams(grams):
		result = append(result, "重量超出范围")
	}
	for _, dimension := range []struct{ label, value string }{
		{"长", sku.Length}, {"宽", sku.Width}, {"高", sku.Height},
	} {
		value := strings.TrimSpace(dimension.value)
		if value == "" {
			result = append(result, dimension.label+"待补")
		} else if number, err := strconv.ParseFloat(value, 64); err != nil || strings.ContainsAny(value, "xXpP_") || number <= 0 || math.IsNaN(number) || math.IsInf(number, 0) {
			result = append(result, dimension.label+"无效")
		}
	}
	switch strings.ToLower(strings.TrimSpace(sku.LengthUnit)) {
	case "":
		result = append(result, "尺寸单位待补")
	case "in", "inch", "inches", "cm", "mm", "m", "ft":
	default:
		result = append(result, "尺寸单位无效")
	}
	return result
}

func skuLabel(skc, sku int, name string) string {
	label := fmt.Sprintf("SKC %d SKU %d", skc+1, sku+1)
	visible := []rune(strings.TrimSpace(strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)))
	if len(visible) > 24 {
		visible = append(visible[:24], '…')
	}
	if len(visible) > 0 {
		label += "（" + string(visible) + "）"
	}
	return label
}

func summarizedSKUs(shown []string, total int) string {
	message := strings.Join(shown, "；")
	if total > len(shown) {
		message += fmt.Sprintf("；共 %d 个 SKU", total)
	}
	return message
}
