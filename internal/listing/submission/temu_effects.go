package submission

import (
	"strconv"
	"task-processor/internal/authidentity"
	"task-processor/internal/storecenter"
)

const TemuPublishAction = "temu_create_product"
const TemuLocalImageAction = "temu_local_image_v2"

// Mutable site selection, app/token and record identities cannot establish a
// second creation effect for the same canonical product and verified merchant.
func TemuProductSubjectID(productKey string, binding storecenter.ProductMerchantBinding) (string, error) {
	if !authidentity.IsBoundedIdentifier(productKey) || !binding.ValidTemuApplication() || !isExecutionDigest(binding.SupplierIdentityHash) {
		return "", ErrExecutionInvalid
	}
	namespace := "temu-supplier-product-"
	if binding.MerchantMode == storecenter.MerchantLocal {
		namespace = "temu-local-goods-"
	}
	return namespace + digestExecutionParts(namespace, productKey, binding.SupplierIdentityHash), nil
}

func TemuLocalImageSubjectID(contentHash string, catID, usage int64, binding storecenter.ProductMerchantBinding) (string, error) {
	if !binding.ValidTemuApplication() || binding.MerchantMode != storecenter.MerchantLocal || !isExecutionDigest(binding.SupplierIdentityHash) || !isExecutionDigest(contentHash) || catID < 1 || usage < 1 || usage > 7 {
		return "", ErrExecutionInvalid
	}
	return "temu-local-image-v2-" + digestExecutionParts(contentHash, binding.SupplierIdentityHash, TemuLocalImageAction, strconv.FormatInt(catID, 10), strconv.FormatInt(usage, 10)), nil
}
