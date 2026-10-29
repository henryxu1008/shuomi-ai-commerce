package offlinepackage

import (
	"fmt"
	"strings"

	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	sheinmarketpub "task-processor/internal/marketplace/shein/publishing"
)

// PrepareProductForValidation applies the same offline defaults as new submit
// without creating a submission identity or reading randomness. It mutates only
// the supplied product; validators must pass an owned copy.
func PrepareProductForValidation(product *previewdata.Product) {
	normalizeProductForSubmit(product, payloadDefaults{
		Site:          "US",
		WarehouseCode: defaultSubmitWarehouseCode,
	})
}

func normalizeProductForSubmit(product *previewdata.Product, settings payloadDefaults) {
	if product == nil {
		return
	}
	// SHEIN generates spu_name for new products. Sending a display title here
	// makes the product API reject the draft/publish request.
	product.SPUName = ""
	product.SourceSystem = "listingkit"
	product.SupplierCode = deriveSubmitProductSupplierCode(product)
	normalizeSubmitCollections(product)
	ensureSubmitSites(product, settings)
	ensureSubmitSKUs(product, settings)
	normalizeSubmitImages(product)
	normalizeSubmitExtra(product)
	finalizeSubmitTransportFields(product)
}

func deriveSubmitProductSupplierCode(product *previewdata.Product) string {
	if product == nil {
		return ""
	}
	supplierSKUs := make([]string, 0)
	for _, skc := range product.SKCList {
		for _, sku := range skc.SKUS {
			supplierSKUs = append(supplierSKUs, sku.SupplierSKU)
		}
	}
	return sheinmarketpub.DeriveSubmitSupplierCode(product.SupplierCode, supplierSKUs)
}

// ValidatePreparedProductPublishPayload validates a submit product after submit normalization has run.
func validatePreparedProductPublishPayload(product *previewdata.Product) error {
	return sheinmarketpub.ValidatePreparedProductPublishPayload(product)
}

func ValidatePreparedProductPublishPayloadWithSizeChartAttributes(product *previewdata.Product, sizeChartAttributes []PendingAttributeCandidate) error {
	if err := validatePreparedProductPublishPayload(product); err != nil {
		return err
	}
	return validateRequiredSizeChartAttributesForSubmit(product, sizeChartAttributes)
}

func validateRequiredSizeChartAttributesForSubmit(product *previewdata.Product, candidates []PendingAttributeCandidate) error {
	if product == nil || len(candidates) == 0 {
		return nil
	}
	present := make(map[int]struct{}, len(product.SizeAttributeList))
	for _, attr := range product.SizeAttributeList {
		if attr.AttributeID <= 0 || strings.TrimSpace(attr.AttributeExtraValue) == "" {
			continue
		}
		present[attr.AttributeID] = struct{}{}
	}
	missing := make([]string, 0)
	seen := map[int]struct{}{}
	for _, candidate := range candidates {
		if !candidate.Required || candidate.AttributeID <= 0 {
			continue
		}
		if _, ok := seen[candidate.AttributeID]; ok {
			continue
		}
		seen[candidate.AttributeID] = struct{}{}
		if _, ok := present[candidate.AttributeID]; ok {
			continue
		}
		missing = append(missing, firstNonEmpty(candidate.AttributeName, candidate.AttributeNameEn, candidate.Name, fmt.Sprintf("attribute_id=%d", candidate.AttributeID)))
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("SHEIN publish blocked: missing required size chart attributes: %s", strings.Join(missing, ", "))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
