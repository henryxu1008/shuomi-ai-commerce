package draft

import (
	"regexp"
	"strconv"
	"strings"

	"task-processor/internal/marketplace/shein/offlinepackage"
	"task-processor/internal/marketplace/shein/offlinepackage/previewdata"
)

func toProductAttributes(items []offlinepackage.Attribute) []previewdata.ProductAttribute {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.ProductAttribute, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.Value) == "" {
			continue
		}
		result = append(result, previewdata.ProductAttribute{AttributeExtraValue: item.Value})
	}
	return result
}

func toImageInfo(info *offlinepackage.ImageDraft) *previewdata.ImageInfo {
	if info == nil {
		return nil
	}
	images := make([]previewdata.ImageDetail, 0, 1+len(info.Gallery))
	if strings.TrimSpace(info.MainImage) != "" {
		images = append(images, previewdata.ImageDetail{ImageType: 1, ImageSort: 1, ImageURL: info.MainImage, MarketingMainImage: true})
	}
	for idx, imageURL := range info.Gallery {
		if strings.TrimSpace(imageURL) == "" {
			continue
		}
		images = append(images, previewdata.ImageDetail{ImageType: 2, ImageSort: idx + 2, ImageURL: imageURL})
	}
	if strings.TrimSpace(info.WhiteBg) != "" {
		images = append(images, previewdata.ImageDetail{ImageType: 2, ImageSort: len(images) + 1, ImageURL: info.WhiteBg})
	}
	if len(images) == 0 {
		return nil
	}
	return &previewdata.ImageInfo{ImageInfoList: images}
}

func toSiteInfo(items []offlinepackage.Site) []previewdata.SiteInfo {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.SiteInfo, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.MainSite) == "" {
			continue
		}
		result = append(result, previewdata.SiteInfo{MainSite: item.MainSite, SubSiteList: append([]string(nil), item.SubSites...)})
	}
	return result
}

func toSKCs(items []offlinepackage.SKCRequestDraft) []previewdata.SKC {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.SKC, 0, len(items))
	for _, item := range items {
		supplierCode := item.SupplierCode
		saleAttribute := previewdata.SaleAttribute{PreFillSpec: true}
		if item.SaleAttribute != nil && item.SaleAttribute.AttributeID > 0 && item.SaleAttribute.AttributeValueID != nil {
			saleAttribute = previewdata.SaleAttribute{
				AttributeID:      item.SaleAttribute.AttributeID,
				AttributeValueID: *item.SaleAttribute.AttributeValueID,
				PreFillSpec:      true,
			}
		}
		result = append(result, previewdata.SKC{
			SaleAttribute:         saleAttribute,
			SKUS:                  toSKUs(item.SKUList),
			ImageInfo:             derefImageInfo(toImageInfo(item.ImageInfo)),
			MultiLanguageName:     previewdata.LanguageContent{Language: "en", Name: item.SkcName},
			MultiLanguageNameList: toLanguageContents(item.MultiLanguageNameList),
			Sort:                  item.Sort,
			SupplierCode:          &supplierCode,
			IsFirstPublish:        true,
		})
	}
	return result
}

func toSKUs(items []offlinepackage.SKUDraft) []previewdata.SKU {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.SKU, 0, len(items))
	for _, item := range items {
		stockCount := item.StockCount
		result = append(result, previewdata.SKU{
			SaleAttributeList: toSaleAttributes(item.SaleAttributes),
			CostInfo:          &previewdata.CostInfo{CostPrice: item.CostPrice, Currency: item.Currency},
			Height:            item.Height,
			Length:            item.Length,
			Width:             item.Width,
			LengthUnit:        item.LengthUnit,
			Weight:            item.Weight,
			WeightUnit:        item.WeightUnit,
			StockCount:        &stockCount,
			SupplierSKU:       item.SupplierSKU,
			StopPurchase:      0,
			MallState:         1,
			PriceInfoList:     toPriceInfoList(item.SitePriceList),
			ImageInfo:         toSKUImageInfoFromDraft(item),
			StockInfoList:     toStockInfoList(item.StockInfoList),
			PackageType:       3,
		})
	}
	return result
}

func toSaleAttributes(items []offlinepackage.ResolvedSaleAttribute) []previewdata.SaleAttribute {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.SaleAttribute, 0, len(items))
	for _, item := range items {
		if item.AttributeID <= 0 || item.AttributeValueID == nil {
			continue
		}
		result = append(result, previewdata.SaleAttribute{
			AttributeID:      item.AttributeID,
			AttributeValueID: *item.AttributeValueID,
		})
	}
	return result
}

func toPriceInfoList(items []offlinepackage.SitePrice) []previewdata.PriceInfo {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.PriceInfo, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item.BasePrice) == "" {
			continue
		}
		result = append(result, previewdata.PriceInfo{
			SubSite:   item.SubSite,
			BasePrice: parseSourceFloatDefault(item.BasePrice),
			Currency:  item.Currency,
		})
	}
	return result
}

func toStockInfoList(items []offlinepackage.StockInfo) []previewdata.StockInfo {
	if len(items) == 0 {
		return nil
	}
	result := make([]previewdata.StockInfo, 0, len(items))
	for _, item := range items {
		result = append(result, previewdata.StockInfo{
			InventoryNum:          item.InventoryNum,
			MerchantWarehouseCode: item.WarehouseCode,
		})
	}
	return result
}

func toSKUImageInfo(image string) *previewdata.ImageInfo {
	if strings.TrimSpace(image) == "" {
		return nil
	}
	return &previewdata.ImageInfo{
		ImageInfoList: []previewdata.ImageDetail{{
			ImageType: 1,
			ImageSort: 1,
			ImageURL:  image,
		}},
	}
}

func toSKUImageInfoFromDraft(item offlinepackage.SKUDraft) *previewdata.ImageInfo {
	if item.ImageInfo == nil {
		return toSKUImageInfo(item.MainImage)
	}
	main := strings.TrimSpace(item.ImageInfo.MainImage)
	if main == "" {
		main = strings.TrimSpace(item.MainImage)
	}
	images := make([]previewdata.ImageDetail, 0, 1+len(item.ImageInfo.Gallery))
	if main != "" {
		images = append(images, previewdata.ImageDetail{ImageType: 1, ImageSort: 1, ImageURL: main})
	}
	for _, imageURL := range item.ImageInfo.Gallery {
		if strings.TrimSpace(imageURL) == "" {
			continue
		}
		images = append(images, previewdata.ImageDetail{ImageType: 2, ImageSort: len(images) + 1, ImageURL: imageURL})
	}
	if strings.TrimSpace(item.ImageInfo.WhiteBg) != "" {
		images = append(images, previewdata.ImageDetail{ImageType: 2, ImageSort: len(images) + 1, ImageURL: item.ImageInfo.WhiteBg})
	}
	if len(images) == 0 {
		return nil
	}
	return &previewdata.ImageInfo{ImageInfoList: images}
}

func derefImageInfo(info *previewdata.ImageInfo) previewdata.ImageInfo {
	if info == nil {
		return previewdata.ImageInfo{}
	}
	return *info
}

var numericTokenPattern = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)

var (
	compositionPercentPattern        = regexp.MustCompile(`(?i)([^,;+\n]+?)\s*[:：]?\s*(\d+(?:\.\d+)?)\s*%`)
	compositionLeadingPercentPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%\s*([^,;+\n]+)`)
)

type compositionItem struct {
	Name    string
	Percent float64
}

func parseCompositionItems(value string) []compositionItem {
	items := collectCompositionItems(value, compositionPercentPattern, false)
	if len(items) > 0 {
		return items
	}
	return collectCompositionItems(value, compositionLeadingPercentPattern, true)
}

func collectCompositionItems(value string, pattern *regexp.Regexp, percentFirst bool) []compositionItem {
	matches := pattern.FindAllStringSubmatch(value, -1)
	if len(matches) == 0 {
		return nil
	}
	items := make([]compositionItem, 0, len(matches))
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		var (
			nameRaw    string
			percentRaw string
		)
		if percentFirst {
			percentRaw = match[1]
			nameRaw = match[2]
		} else {
			nameRaw = match[1]
			percentRaw = match[2]
		}
		name := strings.TrimSpace(strings.Trim(nameRaw, ",;+/ "))
		percent, err := strconv.ParseFloat(strings.TrimSpace(percentRaw), 64)
		if err != nil || name == "" {
			continue
		}
		items = append(items, compositionItem{Name: name, Percent: percent})
	}
	return items
}
