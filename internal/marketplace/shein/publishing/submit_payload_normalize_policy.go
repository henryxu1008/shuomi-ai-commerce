package publishing

import (
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
)

// NormalizeSubmitCollections ensures SHEIN submit payload collection fields serialize as empty arrays instead of null.
func NormalizeSubmitCollections(product *previewdata.Product) {
	if product == nil {
		return
	}
	if product.BrandSeriesList == nil {
		product.BrandSeriesList = []string{}
	}
	if product.MultiLanguageMakeupIngredientList == nil {
		product.MultiLanguageMakeupIngredientList = []any{}
	}
	if product.ProductVideoList == nil {
		product.ProductVideoList = []previewdata.ProductVideo{}
	}
	if product.PartInfoList == nil {
		product.PartInfoList = []any{}
	}
	if product.PLMPatternIDList == nil {
		product.PLMPatternIDList = []any{}
	}
	if product.SizeAttributeList == nil {
		product.SizeAttributeList = []previewdata.SizeAttribute{}
	}
	if product.BackSizeAttributeList == nil {
		product.BackSizeAttributeList = []any{}
	}
	if product.ProductCertificateList == nil {
		product.ProductCertificateList = []int{}
	}
	if product.CertificateList == nil {
		product.CertificateList = []int{}
	}
	if product.DelOtherCertificateSNList == nil {
		product.DelOtherCertificateSNList = []string{}
	}
	if product.CustomAttributeRelation == nil {
		product.CustomAttributeRelation = []previewdata.CustomAttributeRelation{}
	}
}

// NormalizeSubmitExtra ensures SHEIN submit extra fields use publish-page defaults.
func NormalizeSubmitExtra(product *previewdata.Product) {
	if product == nil {
		return
	}
	fromPageID := "product_publish"
	product.Extra.FromPageID = &fromPageID
	product.Extra.SwitchToSPUPic = false
	product.Extra.UseCVTransformImage = false
	product.Extra.TransformCVSizeImage = false
}

// FinalizeSubmitTransportFields ensures submit-only transport collections serialize as empty values.
func FinalizeSubmitTransportFields(product *previewdata.Product) {
	if product == nil {
		return
	}
	if product.Extra.SPUTag == nil {
		product.Extra.SPUTag = []string{}
	}
	if product.Extra.ConfirmVolumeSKU == nil {
		product.Extra.ConfirmVolumeSKU = []string{}
	}
	if product.Extra.ConfirmWeightSKU == nil {
		product.Extra.ConfirmWeightSKU = []string{}
	}
	if product.Extra.ControlPriceData == nil {
		product.Extra.ControlPriceData = map[string]string{}
	}
	for skcIndex := range product.SKCList {
		skc := &product.SKCList[skcIndex]
		if skc.SiteDetailImageInfoList == nil {
			skc.SiteDetailImageInfoList = []previewdata.SiteDetailImageInfo{}
		}
		if skc.SiteSpecImageInfoList == nil {
			skc.SiteSpecImageInfoList = []any{}
		}
		if skc.SKCScopeAttributeList == nil {
			skc.SKCScopeAttributeList = []any{}
		}
		if skc.ProofOfStockList == nil {
			skc.ProofOfStockList = []any{}
		}
		for skuIndex := range skc.SKUS {
			sku := &skc.SKUS[skuIndex]
			if sku.CompetingCostPriceImages == nil {
				sku.CompetingCostPriceImages = []any{}
			}
		}
	}
}
