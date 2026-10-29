package offlinepackage

import (
	"strings"

	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
	sheinmarketpub "task-processor/internal/marketplace/shein/publishing"
)

// NormalizeSubmitImages normalizes SHEIN submit product, SKC, and SKU image payload fields.
func normalizeSubmitImages(product *previewdata.Product) {
	if product == nil {
		return
	}
	if product.ImageInfo != nil {
		product.ImageInfo.ImageInfoList = normalizeSubmitSPUImages(product.ImageInfo.ImageInfoList)
		if product.ImageInfo.OriginalImageInfoList == nil {
			empty := []any{}
			product.ImageInfo.OriginalImageInfoList = &empty
		}
	}
	product.Extra.SwitchToSPUPic = false
	for skcIndex := range product.SKCList {
		skc := &product.SKCList[skcIndex]
		normalizeSubmitSKCImages(skc)
		normalizeSubmitSKUImages(skc)
	}
}

// NormalizeSubmitSPUImages normalizes top-level SPU gallery images for SHEIN submit.
func normalizeSubmitSPUImages(images []previewdata.ImageDetail) []previewdata.ImageDetail {
	normalized := normalizeSubmitGalleryImages(images, false)
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

// NormalizeSubmitSKUImages normalizes SKU image payloads and repairs missing SKU images from SKC fallback.
func normalizeSubmitSKUImages(skc *previewdata.SKC) {
	if skc == nil {
		return
	}
	var fallback previewdata.ImageDetail
	hasFallback := false
	if len(skc.ImageInfo.ImageInfoList) > 0 {
		fallback = skc.ImageInfo.ImageInfoList[0]
		hasFallback = strings.TrimSpace(fallback.ImageURL) != ""
	}
	for skuIndex := range skc.SKUS {
		info := skc.SKUS[skuIndex].ImageInfo
		if info == nil || len(info.ImageInfoList) == 0 {
			if !hasFallback {
				continue
			}
			skc.SKUS[skuIndex].ImageInfo = &previewdata.ImageInfo{
				ImageInfoList: []previewdata.ImageDetail{normalizeSubmitSKUImageDetail(fallback)},
			}
			empty := []any{}
			skc.SKUS[skuIndex].ImageInfo.OriginalImageInfoList = &empty
			continue
		}
		info.ImageInfoList = dedupeImagesByURL(info.ImageInfoList)
		if len(info.ImageInfoList) > 0 {
			info.ImageInfoList = []previewdata.ImageDetail{normalizeSubmitSKUImageDetail(info.ImageInfoList[0])}
		}
		if info.OriginalImageInfoList == nil {
			empty := []any{}
			info.OriginalImageInfoList = &empty
		}
	}
}

// NormalizeSubmitSKUImageDetail normalizes one SKU image detail for SHEIN submit.
func normalizeSubmitSKUImageDetail(image previewdata.ImageDetail) previewdata.ImageDetail {
	return sheinmarketpub.NormalizeSubmitSKUImageDetail(image)
}

// NormalizeSubmitSKCImages normalizes SKC gallery and site detail images for SHEIN submit.
func normalizeSubmitSKCImages(skc *previewdata.SKC) {
	if skc == nil {
		return
	}
	if len(skc.ImageInfo.ImageInfoList) == 0 {
		skc.SiteDetailImageInfoList = []previewdata.SiteDetailImageInfo{}
		return
	}
	skc.ImageInfo.ImageInfoList = normalizeSubmitGalleryImages(skc.ImageInfo.ImageInfoList, true)
	if skc.ImageInfo.OriginalImageInfoList == nil {
		empty := []any{}
		skc.ImageInfo.OriginalImageInfoList = &empty
	}
	skc.SiteDetailImageInfoList = buildSubmitSiteDetailImageInfoList(skc.ImageInfo.ImageInfoList)
}

// BuildSubmitSiteDetailImageInfoList builds SHEIN site detail image groups from normalized images.
func buildSubmitSiteDetailImageInfoList(images []previewdata.ImageDetail) []previewdata.SiteDetailImageInfo {
	return sheinmarketpub.BuildSubmitSiteDetailImageInfoList(images)
}

// NormalizeSubmitGalleryImages normalizes gallery image types, ordering, square image, and optional color block.
func normalizeSubmitGalleryImages(images []previewdata.ImageDetail, includeColorBlock bool) []previewdata.ImageDetail {
	return sheinmarketpub.NormalizeSubmitGalleryImages(images, includeColorBlock)
}

// DedupeImagesByURL keeps the first non-empty image for each image URL.
func dedupeImagesByURL(images []previewdata.ImageDetail) []previewdata.ImageDetail {
	return sheinmarketpub.DedupeImagesByURL(images)
}
