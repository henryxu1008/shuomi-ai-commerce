package draft

import (
	"regexp"
	"strconv"
	"strings"

	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/publishing/common"
	sheinattribute "task-processor/internal/shein/api/attribute"
	sheinproduct "task-processor/internal/shein/api/product"
)

var (
	quantityPrefixPattern = regexp.MustCompile(`(?i)\b(\d+)\s*[- ]?\s*(piece|pieces|piece\(s\)|pc|pcs|pack|packs|set|sets|pair|pairs)\b`)
	quantityOfPattern     = regexp.MustCompile(`(?i)\b(pack|set|bundle|collection)\s+of\s+(\d+)\b`)
)

func buildLocalPreview(pkg *offlinepackage.Package) *sheinproduct.Product {
	if pkg == nil || pkg.DraftPayload == nil {
		return nil
	}
	productTypeID := pkg.ProductTypeID
	var brandCode *string
	if value := strings.TrimSpace(pkg.Metadata["authorized_brand_code"]); value != "" {
		brandCode = &value
	}
	return &sheinproduct.Product{
		BrandCode:               brandCode,
		SPUName:                 pkg.DraftPayload.SpuName,
		SupplierCode:            pkg.DraftPayload.SupplierCode,
		CategoryID:              pkg.CategoryID,
		CategoryIDList:          append([]int(nil), pkg.CategoryIDList...),
		ProductTypeID:           productTypeID,
		TopCategoryID:           pkg.TopCategoryID,
		SourceSystem:            "listingkit",
		MultiLanguageNameList:   toLanguageContents(pkg.DraftPayload.MultiLanguageNameList),
		MultiLanguageDescList:   toLanguageContents(pkg.DraftPayload.MultiLanguageDescList),
		ProductAttributeList:    buildProductAttributes(pkg),
		SizeAttributeList:       append([]sheinproduct.SizeAttribute(nil), pkg.DraftPayload.SizeAttributeList...),
		ImageInfo:               toImageInfo(pkg.DraftPayload.ImageInfo),
		SiteList:                toSiteInfo(pkg.DraftPayload.SiteList),
		SKCList:                 toSKCs(pkg.DraftPayload.SKCList),
		CustomAttributeRelation: append([]sheinattribute.CustomAttributeRelation(nil), pkg.CustomAttributeRelation...),
		Extra:                   sheinproduct.Extra{SwitchToSPUPic: true, UseCVTransformImage: true},
		ProductCertificateList:  []int{},
		CertificateList:         []int{},
	}
}

func buildProductAttributes(pkg *offlinepackage.Package) []sheinproduct.ProductAttribute {
	return toResolvedAttributes(pkg)
}

func toLanguageContents(items []offlinepackage.LocalizedText) []sheinproduct.LanguageContent {
	if len(items) == 0 {
		return nil
	}
	result := make([]sheinproduct.LanguageContent, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			continue
		}
		lang := strings.TrimSpace(item.Language)
		if lang == "" {
			lang = "en"
		}
		result = append(result, sheinproduct.LanguageContent{Language: lang, Name: item.Name})
	}
	return result
}

func toResolvedAttributes(pkg *offlinepackage.Package) []sheinproduct.ProductAttribute {
	if pkg != nil && len(pkg.ResolvedAttributes) > 0 {
		result := make([]sheinproduct.ProductAttribute, 0, len(pkg.ResolvedAttributes))
		compositionCount := countResolvedCompositionAttributes(pkg.ResolvedAttributes)
		seenAttributeKeys := make(map[string]struct{}, len(pkg.ResolvedAttributes))
		for _, item := range pkg.ResolvedAttributes {
			if item.AttributeID <= 0 {
				continue
			}
			if item.AttributeType == 2 {
				// Numeric display attributes such as length/width/height are
				// submitted through SKU dimension fields. Sending them again as
				// generic product attributes causes SHEIN pre-validation type
				// mismatches for attribute_type=2 templates.
				continue
			}
			extraValue := resolvedAttributeExtraValue(pkg, item, compositionCount)
			dedupKey := resolvedAttributeDedupKey(item, extraValue)
			if _, exists := seenAttributeKeys[dedupKey]; exists {
				continue
			}
			seenAttributeKeys[dedupKey] = struct{}{}
			result = append(result, sheinproduct.ProductAttribute{
				AttributeID:         item.AttributeID,
				AttributeValueID:    item.AttributeValueID,
				AttributeExtraValue: extraValue,
			})
		}
		if len(result) > 0 {
			return result
		}
	}
	if pkg == nil || pkg.DraftPayload == nil {
		return nil
	}
	return toProductAttributes(pkg.DraftPayload.ProductAttributeList)
}

func resolvedAttributeDedupKey(item offlinepackage.ResolvedAttribute, extraValue string) string {
	valueID := 0
	if item.AttributeValueID != nil {
		valueID = *item.AttributeValueID
	}
	return strconv.Itoa(item.AttributeID) + ":" + strconv.Itoa(valueID) + ":" + extraValue
}

func countResolvedCompositionAttributes(items []offlinepackage.ResolvedAttribute) int {
	count := 0
	for _, item := range items {
		if item.AttributeType == 3 {
			count++
		}
	}
	return count
}

func resolvedAttributeExtraValue(pkg *offlinepackage.Package, item offlinepackage.ResolvedAttribute, compositionCount int) string {
	extraValue := strings.TrimSpace(item.AttributeExtraValue)
	if extraValue != "" {
		return extraValue
	}
	if inferred := inferSupplementalAttributeExtraValue(pkg, item); inferred != "" {
		return inferred
	}
	if item.AttributeType != 3 {
		return ""
	}
	if len(parseCompositionItems(item.Value)) > 0 {
		if parsed := parseResolvedCompositionPercent(item.Value); parsed != "" {
			return parsed
		}
	}
	if item.AttributeValueID != nil && compositionCount == 1 {
		return "100"
	}
	return ""
}

func inferSupplementalAttributeExtraValue(pkg *offlinepackage.Package, item offlinepackage.ResolvedAttribute) string {
	if numeric := firstNumericToken(item.Value); numeric != "" {
		return numeric
	}
	if !attributeNeedsSupplementalExtraValue(item) {
		return ""
	}
	if inferred := inferQuantityAttributeExtraValue(pkg, item); inferred != "" {
		return inferred
	}
	return ""
}

func attributeNeedsSupplementalExtraValue(item offlinepackage.ResolvedAttribute) bool {
	if item.AttributeValueID == nil {
		return false
	}
	if strings.TrimSpace(item.AttributeExtraValue) != "" {
		return false
	}
	if item.AttributeID == 1000411 {
		return true
	}
	name := normalizeText(item.Name)
	value := normalizeText(item.Value)
	return strings.Contains(name, "quantity") ||
		strings.Contains(name, "count") ||
		strings.Contains(value, "piece") ||
		strings.Contains(value, "pack") ||
		strings.Contains(value, "set")
}

func inferQuantityAttributeExtraValue(pkg *offlinepackage.Package, item offlinepackage.ResolvedAttribute) string {
	if item.AttributeID != 1000411 && !strings.Contains(normalizeText(item.Name), "quantity") {
		return ""
	}
	for _, candidate := range collectSupplementalAttributeTextCandidates(pkg, item) {
		if value := inferQuantityFromText(candidate); value != "" {
			return value
		}
	}
	return "1"
}

func collectSupplementalAttributeTextCandidates(pkg *offlinepackage.Package, item offlinepackage.ResolvedAttribute) []string {
	candidates := make([]string, 0, 16)
	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			candidates = append(candidates, value)
		}
	}
	appendCandidate(item.Value)
	if pkg == nil {
		return candidates
	}
	appendCandidate(pkg.ProductNameEn)
	appendCandidate(pkg.ProductNameMulti)
	appendCandidate(pkg.SpuName)
	appendCandidate(pkg.Description)
	for _, attr := range pkg.ProductAttributes {
		appendCandidate(attr.Name)
		appendCandidate(attr.Value)
	}
	for _, value := range pkg.Attributes {
		appendCandidate(value)
	}
	if pkg.DraftPayload != nil {
		appendCandidate(pkg.DraftPayload.SpuName)
		for _, text := range pkg.DraftPayload.MultiLanguageNameList {
			appendCandidate(text.Name)
		}
		for _, text := range pkg.DraftPayload.MultiLanguageDescList {
			appendCandidate(text.Name)
		}
		for _, attr := range pkg.DraftPayload.ProductAttributeList {
			appendCandidate(attr.Name)
			appendCandidate(attr.Value)
		}
	}
	return candidates
}

func inferQuantityFromText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if matches := quantityPrefixPattern.FindStringSubmatch(value); len(matches) > 1 {
		return matches[1]
	}
	if matches := quantityOfPattern.FindStringSubmatch(value); len(matches) > 2 {
		return matches[2]
	}
	return ""
}

func firstNumericToken(value string) string {
	if match := numericTokenPattern.FindString(strings.TrimSpace(value)); match != "" {
		if parsed, err := strconv.ParseFloat(match, 64); err == nil && parsed > 0 {
			return common.FormatFloat(parsed)
		}
	}
	return ""
}

func parseResolvedCompositionPercent(value string) string {
	items := parseCompositionItems(value)
	if len(items) != 1 {
		return ""
	}
	return common.FormatFloat(items[0].Percent)
}
