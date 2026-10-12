package submission

import (
	"strconv"
	"task-processor/internal/authidentity"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/storecenter"
)

const TemuPublishAction = "temu_create_product"
const TemuLocalImageAction = "temu_local_image_v2"
const TemuSupplierImageAction = "temu_supplier_image_global"

func TemuImageAction(b storecenter.ProductMerchantBinding) string {
	if b.MerchantMode == storecenter.MerchantLocal {
		return TemuLocalImageAction
	}
	return TemuSupplierImageAction
}
func TemuSupplierImageBusinessType(usage int64) (int64, error) {
	if usage == 6 {
		return 1, nil
	}
	if usage == 2 || usage == 3 || usage == 5 {
		return 0, nil
	}
	return 0, ErrExecutionInvalid
}
func TemuSupplierImageSubjectID(contentHash string, catID, businessType int64, binding storecenter.ProductMerchantBinding) (string, error) {
	if !binding.ValidTemuApplication() || binding.MerchantMode == storecenter.MerchantLocal || !isExecutionDigest(binding.SupplierIdentityHash) || !isExecutionDigest(contentHash) || catID < 1 || businessType < 0 || businessType > 1 {
		return "", ErrExecutionInvalid
	}
	return "temu-supplier-image-" + digestExecutionParts(contentHash, binding.SupplierIdentityHash, TemuSupplierImageAction, strconv.FormatInt(catID, 10), strconv.FormatInt(businessType, 10)), nil
}
func TemuImageSubjectID(image temugoods.ImageObservation, binding storecenter.ProductMerchantBinding) (string, error) {
	if binding.MerchantMode == storecenter.MerchantLocal {
		return TemuLocalImageSubjectID(image.ContentHash, image.CatID, image.Usage, binding)
	}
	biz, err := TemuSupplierImageBusinessType(image.Usage)
	if err != nil {
		return "", err
	}
	return TemuSupplierImageSubjectID(image.ContentHash, image.CatID, biz, binding)
}

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
