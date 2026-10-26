package offlinepackage

import (
	"fmt"
	"strconv"
	"strings"

	sheinmarketpub "task-processor/internal/marketplace/shein/publishing"
	sheinproduct "task-processor/internal/shein/api/product"
)

// SecondarySaleAttributeRequired reports whether SKU-level sale attributes are required for submit.
func secondarySaleAttributeRequired(pkg *Package) bool {
	if pkg == nil || pkg.SaleAttributeResolution == nil {
		return false
	}
	if !hasMultiSKUWithinSingleSKC(pkg) {
		return false
	}
	if !hasSecondarySourceVariation(pkg) {
		return false
	}
	return hasSecondaryTemplateCandidate(pkg.SaleAttributeResolution)
}

// HasMultiSKUWithinSingleSKC reports whether any SKC has multiple SKUs in draft, preview, or package state.
func hasMultiSKUWithinSingleSKC(pkg *Package) bool {
	if pkg == nil {
		return false
	}
	if pkg.DraftPayload != nil {
		for _, skc := range pkg.DraftPayload.SKCList {
			if len(skc.SKUList) > 1 {
				return true
			}
		}
	}
	if pkg.PreviewPayload != nil {
		for _, skc := range pkg.PreviewPayload.SKCList {
			if len(skc.SKUS) > 1 {
				return true
			}
		}
	}
	for _, skc := range pkg.SkcList {
		if len(skc.SKUs) > 1 {
			return true
		}
	}
	return false
}

// HasSecondarySourceVariation reports whether source data contains a secondary sale-attribute dimension.
func hasSecondarySourceVariation(pkg *Package) bool {
	if pkg == nil || pkg.SaleAttributeResolution == nil {
		return false
	}
	resolution := pkg.SaleAttributeResolution
	if resolution.SecondaryAttributeID > 0 || len(resolution.SKUAttributes) > 0 || len(resolution.SKUValueAssignments) > 0 {
		return true
	}
	sourceName := strings.TrimSpace(resolution.SecondarySourceDimension)
	if sourceName == "" {
		return false
	}
	for _, dimension := range resolution.SourceDimensions {
		if saleDimensionMatches(sourceName, dimension.Name) && dimension.DistinctCount > 1 {
			return true
		}
	}
	for _, skc := range pkg.SkcList {
		values := map[string]struct{}{}
		for _, sku := range skc.SKUs {
			value := ""
			for key, attrValue := range sku.Attributes {
				if saleDimensionMatches(sourceName, key) {
					value = strings.TrimSpace(attrValue)
					break
				}
			}
			if value == "" {
				continue
			}
			values[value] = struct{}{}
			if len(values) > 1 {
				return true
			}
		}
	}
	return false
}

// HasSecondaryTemplateCandidate reports whether SHEIN templates expose a candidate for the secondary dimension.
func hasSecondaryTemplateCandidate(resolution *SaleAttributeResolution) bool {
	if resolution == nil {
		return false
	}
	sourceName := strings.TrimSpace(resolution.SecondarySourceDimension)
	if sourceName == "" {
		return false
	}
	for _, candidate := range resolution.Candidates {
		if candidate.AttributeID <= 0 || candidate.AttributeID == resolution.PrimaryAttributeID || candidate.SKCScope {
			continue
		}
		if saleDimensionMatches(sourceName, candidate.SourceDimension) || saleDimensionMatches(sourceName, candidate.Name) {
			return true
		}
	}
	for _, option := range resolution.TemplateOptions {
		if option.AttributeID <= 0 || option.AttributeID == resolution.PrimaryAttributeID || option.SKCScope {
			continue
		}
		if saleDimensionMatches(sourceName, option.Name) || saleDimensionMatches(sourceName, option.NameEn) {
			return true
		}
	}
	return false
}

// SaleDimensionMatches reports whether two source/template sale dimensions are equivalent.
func saleDimensionMatches(left, right string) bool {
	return sheinmarketpub.SaleDimensionMatches(left, right)
}

// HasBlockingPendingAttributes reports whether required or manually pending attributes still block submit.
func HasBlockingPendingAttributes(pkg *Package) bool {
	if pkg == nil || pkg.AttributeResolution == nil {
		return true
	}
	for _, candidate := range pkg.AttributeResolution.PendingAttributeCandidates {
		if candidate.Required {
			return true
		}
	}
	for _, attr := range pkg.AttributeResolution.PendingAttributes {
		if strings.TrimSpace(attr.Name) != "" || strings.TrimSpace(attr.Value) != "" {
			return true
		}
	}
	return false
}

// SaleAttributesReadyForSubmit reports whether sale attributes are complete enough for submit.
func SaleAttributesReadyForSubmit(pkg *Package) bool {
	return len(saleAttributesReadinessFailureReasons(pkg)) == 0
}

// SaleAttributesReadinessFailureReasons returns sale-attribute submit blockers.
func saleAttributesReadinessFailureReasons(pkg *Package) []string {
	var reasons []string
	if !saleAttributeStatusResolved(pkg) {
		reasons = append(reasons, "sale attribute status is not resolved or primary attribute id is missing")
	}
	if SaleAttributeReviewPending(pkg) {
		reasons = append(reasons, "sale attribute category review is still pending")
	}
	if pkg == nil || pkg.DraftPayload == nil || len(pkg.DraftPayload.SKCList) == 0 {
		reasons = append(reasons, "draft payload skc_list is empty")
		return uniqueNonEmptySubmitStrings(reasons)
	}
	for _, skc := range pkg.DraftPayload.SKCList {
		if !resolvedSaleAttributeReady(skc.SaleAttribute) {
			reasons = append(reasons, fmt.Sprintf("skc %q is missing a resolved sale attribute value id", skc.SupplierCode))
		}
		requireSKUAttributes := len(skc.SKUList) > 1 && secondarySaleAttributeRequired(pkg)
		if !requireSKUAttributes && pkg.SaleAttributeResolution != nil {
			requireSKUAttributes = pkg.SaleAttributeResolution.SecondaryAttributeID > 0 || len(pkg.SaleAttributeResolution.SKUAttributes) > 0
		}
		if requireSKUAttributes {
			if len(skc.SKUList) == 0 {
				reasons = append(reasons, fmt.Sprintf("skc %q is missing sku_list while sku sale attributes are required", skc.SupplierCode))
				continue
			}
			for _, sku := range skc.SKUList {
				if len(sku.SaleAttributes) == 0 {
					reasons = append(reasons, fmt.Sprintf("sku %q is missing sale_attributes", sku.SupplierSKU))
					continue
				}
				for _, attr := range sku.SaleAttributes {
					if !resolvedSaleAttributeValueReady(attr) {
						reasons = append(reasons, fmt.Sprintf(
							"sku %q has unresolved sale attribute %q (attribute_id=%d, attribute_value_id=%v)",
							sku.SupplierSKU,
							attr.Name,
							attr.AttributeID,
							attr.AttributeValueID,
						))
					}
				}
			}
		}
	}
	return uniqueNonEmptySubmitStrings(reasons)
}

// SaleAttributeReviewPending reports whether category review still blocks sale-attribute submit.
func SaleAttributeReviewPending(pkg *Package) bool {
	return pkg != nil && pkg.SaleAttributeResolution != nil && pkg.SaleAttributeResolution.RecommendCategoryReview
}

// SaleAttributeStatusResolved reports whether sale-attribute resolution has a resolved primary attribute.
func saleAttributeStatusResolved(pkg *Package) bool {
	if pkg == nil || pkg.SaleAttributeResolution == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(pkg.SaleAttributeResolution.Status), "resolved") &&
		pkg.SaleAttributeResolution.PrimaryAttributeID > 0
}

// ResolvedSaleAttributeReady reports whether a resolved sale attribute pointer has usable IDs.
func resolvedSaleAttributeReady(attr *ResolvedSaleAttribute) bool {
	return attr != nil && resolvedSaleAttributeValueReady(*attr)
}

// ResolvedSaleAttributeValueReady reports whether a resolved sale attribute value has usable IDs.
func resolvedSaleAttributeValueReady(attr ResolvedSaleAttribute) bool {
	return sheinmarketpub.ResolvedSaleAttributeValueReady(attr.AttributeID, attr.AttributeValueID)
}

// HasAnySubmitSKU reports whether the canonical draft contains at least one SKU.
func HasAnySubmitSKU(pkg *Package) bool {
	draft := DraftPayloadOf(pkg)
	if draft == nil {
		return false
	}
	for _, skc := range draft.SKCList {
		if len(skc.SKUList) > 0 {
			return true
		}
	}
	return false
}

// FinalSubmitImagesReady reports whether final submit images are ready for an action.
func FinalSubmitImagesReady(pkg *Package, action string) (bool, string) {
	input := sheinmarketpub.FinalSubmitImageReadinessInput{
		HasFinalDraft: pkg != nil && pkg.FinalSubmissionDraft != nil,
		RequiresSKC:   sheinmarketpub.FinalSubmitImagesRequireSKC(action),
	}
	if input.HasFinalDraft {
		main := strings.TrimSpace(pkg.FinalSubmissionDraft.MainImageURL)
		if main == "" && pkg.DraftPayload != nil && pkg.DraftPayload.ImageInfo != nil {
			main = strings.TrimSpace(pkg.DraftPayload.ImageInfo.MainImage)
		}
		input.HasMainImage = main != ""
		input.HasGallery = hasFinalGalleryImage(pkg)
		input.HasSKCImage = hasSKCImage(pkg)
		input.HasSwatchRole = hasSwatchRole(pkg)
	}
	return sheinmarketpub.FinalSubmitImagesReady(action, input)
}

// HasFinalGalleryImage reports whether the final draft has at least one gallery-capable image.
func hasFinalGalleryImage(pkg *Package) bool {
	if pkg == nil || pkg.DraftPayload == nil || pkg.DraftPayload.ImageInfo == nil {
		return false
	}
	return sheinmarketpub.SubmitImageURLSliceHasImage(append([]string{pkg.DraftPayload.ImageInfo.MainImage}, pkg.DraftPayload.ImageInfo.Gallery...))
}

// HasSKCImage reports whether the package has an SKC or fallback single-SKC image.
func hasSKCImage(pkg *Package) bool {
	if pkg == nil {
		return false
	}
	if pkg.DraftPayload != nil {
		for _, skc := range pkg.DraftPayload.SKCList {
			if imageDraftHasImage(skc.ImageInfo) {
				return true
			}
		}
	}
	if pkg.PreviewPayload != nil {
		for _, skc := range pkg.PreviewPayload.SKCList {
			if productImageInfoHasImage(&skc.ImageInfo) {
				return true
			}
		}
	}
	if hasSingleSKC(pkg) && hasFinalMainImage(pkg) {
		return true
	}
	return false
}

// HasSwatchRole reports whether final image roles include an explicit swatch/SKC role or an SKC image fallback exists.
func hasSwatchRole(pkg *Package) bool {
	if pkg == nil || pkg.FinalSubmissionDraft == nil {
		return true
	}
	for _, role := range pkg.FinalSubmissionDraft.ImageRoleOverrides {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "swatch", "skc":
			return true
		}
	}
	return hasSKCImage(pkg)
}

// HasSingleSKC reports whether the package has exactly one SKC across draft/package/preview data.
func hasSingleSKC(pkg *Package) bool {
	if pkg == nil {
		return false
	}
	count := 0
	if pkg.DraftPayload != nil && len(pkg.DraftPayload.SKCList) > 0 {
		count = len(pkg.DraftPayload.SKCList)
	} else if len(pkg.SkcList) > 0 {
		count = len(pkg.SkcList)
	} else if pkg.PreviewPayload != nil && len(pkg.PreviewPayload.SKCList) > 0 {
		count = len(pkg.PreviewPayload.SKCList)
	}
	return count == 1
}

// HasFinalMainImage reports whether any final-submit main image source is present.
func hasFinalMainImage(pkg *Package) bool {
	if pkg == nil {
		return false
	}
	if pkg.FinalSubmissionDraft != nil && strings.TrimSpace(pkg.FinalSubmissionDraft.MainImageURL) != "" {
		return true
	}
	if pkg.DraftPayload != nil && pkg.DraftPayload.ImageInfo != nil && strings.TrimSpace(pkg.DraftPayload.ImageInfo.MainImage) != "" {
		return true
	}
	if pkg.Images != nil && strings.TrimSpace(pkg.Images.MainImage) != "" {
		return true
	}
	return false
}

// SubmitPricingReady reports whether draft SKU prices are complete and positive.
func SubmitPricingReady(pkg *Package) bool {
	if pkg == nil || pkg.DraftPayload == nil {
		return false
	}
	hasSKU := false
	for _, skc := range pkg.DraftPayload.SKCList {
		for _, sku := range skc.SKUList {
			hasSKU = true
			if !positiveSubmitMoney(sku.BasePrice) || len(sku.SitePriceList) == 0 {
				return false
			}
			for _, sitePrice := range sku.SitePriceList {
				if !positiveSubmitMoney(sitePrice.BasePrice) {
					return false
				}
			}
		}
	}
	return hasSKU
}

func positiveSubmitMoney(value string) bool {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && parsed > 0
}

// FinalReviewReady reports whether final review confirmation allows the submit action to continue.
func FinalReviewReady(pkg *Package, action string) bool {
	if !sheinmarketpub.FinalReviewRequired(action) {
		return true
	}
	return pkg == nil || pkg.FinalSubmissionDraft == nil || pkg.FinalSubmissionDraft.Confirmed
}

// HasSubmitImage reports whether package, draft, or preview payload data contains any submit image.
func HasSubmitImage(pkg *Package) bool {
	if pkg == nil {
		return false
	}
	if pkg.Images != nil && sheinmarketpub.SubmitImageURLsHaveImage(pkg.Images.MainImage, pkg.Images.WhiteBgImage) {
		return true
	}
	if pkg.DraftPayload != nil {
		if imageDraftHasImage(pkg.DraftPayload.ImageInfo) {
			return true
		}
		for _, skc := range pkg.DraftPayload.SKCList {
			if imageDraftHasImage(skc.ImageInfo) {
				return true
			}
			for _, sku := range skc.SKUList {
				if strings.TrimSpace(sku.MainImage) != "" {
					return true
				}
			}
		}
	}
	if pkg.PreviewPayload != nil {
		if productImageInfoHasImage(pkg.PreviewPayload.ImageInfo) {
			return true
		}
		for _, skc := range pkg.PreviewPayload.SKCList {
			if productImageInfoHasImage(&skc.ImageInfo) {
				return true
			}
			for _, sku := range skc.SKUS {
				if productImageInfoHasImage(sku.ImageInfo) {
					return true
				}
			}
		}
	}
	return false
}

// ImageDraftHasImage reports whether a request image draft contains any image URL.
func imageDraftHasImage(info *ImageDraft) bool {
	if info == nil {
		return false
	}
	return sheinmarketpub.SubmitImageDraftHasImage(sheinmarketpub.SubmitImageDraftInput{
		MainImage: info.MainImage,
		WhiteBg:   info.WhiteBg,
		Gallery:   append([]string(nil), info.Gallery...),
		Source:    append([]string(nil), info.Source...),
	})
}

// ProductImageInfoHasImage reports whether a SHEIN product image info contains any image URL.
func productImageInfoHasImage(info *sheinproduct.ImageInfo) bool {
	return sheinmarketpub.ProductImageInfoHasImage(info)
}

func uniqueNonEmptySubmitStrings(values []string) []string {
	return sheinmarketpub.UniqueNonEmptyImageURLs(values)
}

// VariantImageCoverageStatus returns the current blocked variant image coverage message.
func VariantImageCoverageStatus(pkg *Package) (string, bool) {
	if pkg == nil {
		return "", false
	}
	return sheinmarketpub.VariantImageCoverageStatus(pkg.Metadata)
}
