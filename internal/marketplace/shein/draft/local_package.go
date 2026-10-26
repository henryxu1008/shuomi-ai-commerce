package draft

import (
	"context"
	"io"
	"strings"

	"github.com/sirupsen/logrus"

	"task-processor/internal/marketplace/shein/offlinepackage"
	sheinmarketpub "task-processor/internal/marketplace/shein/publishing"
	"task-processor/internal/product/catalog/canonical"
	"task-processor/internal/publishing/common"
)

func buildLocalPackage(product *canonical.Product, country, language string, logs func(string) *logrus.Entry) *offlinepackage.Package {
	runtimeCtx := context.Background()
	if product == nil {
		return &offlinepackage.Package{ReviewNotes: []string{"canonical product is empty"}}
	}
	log := componentLogger(logs, "shein/assembler")

	images := common.BuildImages(product)
	spuName := common.WithBrandHint(product.Title, "")
	copy := buildLocalCopy(runtimeCtx, product, spuName, logs)
	brand := common.ResolveBrand("", product)
	authorizedBrandCode := ""
	authorizedBrandName := ""
	variants := common.BuildVariants(product)
	productAttributes := common.BuildAttributes(product.Attributes)
	siteList := common.DefaultSites(country)
	categoryName := common.LastCategory(product.CategoryPath)

	pkg := &offlinepackage.Package{
		SpuName:           spuName,
		BrandName:         brand,
		ProductNameEn:     copy.Title,
		ProductNameMulti:  copy.Title,
		TitleDiagnostics:  copy.TitleDiagnostics,
		CategoryName:      categoryName,
		CategoryPath:      append([]string(nil), product.CategoryPath...),
		Description:       copy.Description,
		SellingPoints:     append([]string(nil), product.SellingPoints...),
		Attributes:        common.FlattenAttributes(product.Attributes),
		ProductAttributes: productAttributes,
		SiteList:          siteList,
		Images:            images,
		DraftPayload: &offlinepackage.RequestDraft{
			SpuName:               spuName,
			MultiLanguageNameList: localizedEnglishText(language, copy.Title),
			MultiLanguageDescList: localizedEnglishText(language, copy.Description),
			ProductAttributeList:  productAttributes,
			ImageInfo:             buildImageDraft(images),
			SiteList:              siteList,
		},
		Metadata: map[string]string{
			"target_platform":       "shein",
			"country":               country,
			"language":              language,
			"category_name":         categoryName,
			"source_category_path":  strings.Join(product.CategoryPath, " > "),
			"authorized_brand_code": authorizedBrandCode,
			"authorized_brand_name": authorizedBrandName,
		},
	}
	if copy.DescriptionError != nil {
		pkg.ReviewNotes = append(pkg.ReviewNotes, "英文描述生成失败，无法进入最终提交")
		pkg.Metadata["english_description_generation"] = "failed"
	}
	if copy.TitleDiagnostics != nil {
		pkg.Metadata["title_source"] = copy.TitleDiagnostics.Source
		if copy.TitleDiagnostics.PromptContaminated {
			pkg.Metadata["title_prompt_contaminated"] = "true"
		}
		if note := strings.TrimSpace(copy.TitleDiagnostics.ResolutionNote); note != "" {
			pkg.Metadata["title_resolution_note"] = note
		}
		if base := strings.TrimSpace(copy.TitleDiagnostics.SKCBaseTitle); base != "" {
			pkg.Metadata["title_skc_base"] = base
		}
	}
	attachSourceFactReviewMetadata(pkg, product)
	normalizeLocalCopy(runtimeCtx, pkg, product, language, logs)
	groups := buildLocalVariantGroups(variants, images)
	pkg.SkcList = buildSKCs(groups)
	supplierCode := firstSupplierCode(pkg.SkcList)
	pkg.DraftPayload.SupplierCode = supplierCode
	pkg.DraftPayload.SKCList = buildRequestSKCs(groups, images, siteList, product)
	sanitizeLocalDraft(pkg, runtimeCtx, nil, logs)
	pkg.PreviewPayload = buildLocalPreview(pkg)
	log.WithFields(logrus.Fields{
		"category_id": pkg.CategoryID,
		"skc_count":   len(pkg.SkcList),
		"sku_count":   countPackageSKUs(pkg.SkcList),
	}).Info("built SHEIN preview package")
	return pkg
}

func attachSourceFactReviewMetadata(pkg *offlinepackage.Package, canonical *canonical.Product) {
	if pkg == nil || canonical == nil || !canonicalHas1688Source(canonical) {
		return
	}
	fields := sourceFactReviewFields(canonical)
	if len(fields) == 0 {
		return
	}
	if pkg.Metadata == nil {
		pkg.Metadata = map[string]string{}
	}
	pkg.Metadata["source_platform"] = "1688"
	pkg.Metadata["source_fact_review_required"] = "true"
	pkg.Metadata["source_fact_review_fields"] = strings.Join(fields, ",")
}

func canonicalHas1688Source(product *canonical.Product) bool {
	if product == nil {
		return false
	}
	for _, trace := range product.FieldTraces {
		for _, source := range trace.Sources {
			if source.Type == canonical.SourceProductURL && strings.Contains(strings.ToLower(source.Detail), "detail.1688.com") {
				return true
			}
		}
	}
	return false
}

func sourceFactReviewFields(product *canonical.Product) []string {
	if product == nil {
		return nil
	}
	criticalFields := []string{
		"title",
		"brand",
		"category_path",
		"description",
		"selling_points",
		"seo_keywords",
		"specifications",
	}
	fields := make([]string, 0, len(criticalFields))
	for _, field := range criticalFields {
		trace, ok := product.FieldTraces[field]
		if !ok || !trace.NeedsReview || !hasCanonicalSourceType(trace.Sources, canonical.SourceLLM) {
			continue
		}
		fields = append(fields, field)
	}
	return fields
}

func hasCanonicalSourceType(sources []canonical.Source, want canonical.SourceType) bool {
	for _, source := range sources {
		if source.Type == want {
			return true
		}
	}
	return false
}

func countPackageSKUs(skcs []offlinepackage.SKCPackage) int {
	total := 0
	for _, skc := range skcs {
		total += len(skc.SKUs)
	}
	return total
}

func buildSKCs(groups []variantGroup) []offlinepackage.SKCPackage {
	if len(groups) == 0 {
		return nil
	}
	result := make([]offlinepackage.SKCPackage, 0, len(groups))
	for _, group := range groups {
		result = append(result, offlinepackage.SKCPackage{
			SkcName:      group.skcName,
			SaleName:     group.saleName,
			SupplierCode: group.supplierCode,
			MainImageURL: group.mainImageURL,
			Attributes:   common.CloneMap(group.attributes),
			SKUs:         append([]common.Variant(nil), group.skus...),
		})
	}
	return result
}

func buildRequestSKCs(groups []variantGroup, images *common.ImageSet, siteList []common.Site, canonical *canonical.Product) []offlinepackage.SKCRequestDraft {
	if len(groups) == 0 {
		return nil
	}
	var imageMain string
	var imageGallery []string
	var whiteBackgroundImage string
	if images != nil {
		imageMain = images.MainImage
		imageGallery = images.Gallery
		whiteBackgroundImage = images.WhiteBgImage
	}
	result := make([]offlinepackage.SKCRequestDraft, 0, len(groups))
	for idx, group := range groups {
		skus := make([]offlinepackage.SKUDraft, 0, len(group.skus))
		for _, variant := range group.skus {
			skus = append(skus, buildSKUDraft(variant, canonical, common.FirstNonEmpty(variant.Image, group.mainImageURL, imageMain), siteList))
		}
		result = append(result, offlinepackage.SKCRequestDraft{
			SkcName:      group.skcName,
			SaleName:     group.saleName,
			SupplierCode: group.supplierCode,
			Sort:         idx + 1,
			MultiLanguageNameList: []offlinepackage.LocalizedText{
				{Language: "en", Name: group.skcName},
			},
			ImageInfo: buildImageDraft(&common.ImageSet{
				MainImage:    group.mainImageURL,
				Gallery:      append([]string(nil), imageGallery...),
				WhiteBgImage: whiteBackgroundImage,
			}),
			SKUList: skus,
		})
	}
	return result
}

func buildSKUDraft(variant common.Variant, canonical *canonical.Product, mainImage string, siteList []common.Site) offlinepackage.SKUDraft {
	draft := offlinepackage.SKUDraft{
		SupplierSKU: variant.SKU,
		Attributes:  common.CloneMap(variant.Attributes),
		StockCount:  variant.Stock,
		MainImage:   mainImage,
		Barcode:     variant.Barcode,
		IsDefault:   variant.IsDefault,
	}
	if price := (sheinmarketpub.PricingPolicy{}).Apply(variant.Price); price != nil {
		draft.Currency = price.Currency
		draft.CostPrice = common.FormatFloat(price.CostPrice)
		draft.BasePrice = common.FormatFloat(price.Amount)
		draft.SitePriceList = buildSitePrices(price, siteList)
	}
	if variant.Weight != nil {
		draft.Weight = variant.Weight.Value
		draft.WeightUnit = variant.Weight.Unit
	} else if canonical != nil && canonical.Specifications != nil && canonical.Specifications.Weight != nil {
		draft.Weight = canonical.Specifications.Weight.Value
		draft.WeightUnit = canonical.Specifications.Weight.Unit
	}
	if variant.Dimensions != nil {
		draft.Length = common.FormatFloat(variant.Dimensions.Length)
		draft.Width = common.FormatFloat(variant.Dimensions.Width)
		draft.Height = common.FormatFloat(variant.Dimensions.Height)
		draft.LengthUnit = variant.Dimensions.Unit
	} else if canonical != nil && canonical.Specifications != nil && canonical.Specifications.Dimensions != nil {
		draft.Length = common.FormatFloat(canonical.Specifications.Dimensions.Length)
		draft.Width = common.FormatFloat(canonical.Specifications.Dimensions.Width)
		draft.Height = common.FormatFloat(canonical.Specifications.Dimensions.Height)
		draft.LengthUnit = canonical.Specifications.Dimensions.Unit
	}
	draft.StockInfoList = []offlinepackage.StockInfo{{WarehouseCode: "DEFAULT", InventoryNum: variant.Stock}}
	return draft
}

func buildSitePrices(price *common.Price, siteList []common.Site) []offlinepackage.SitePrice {
	if price == nil {
		return nil
	}
	subSite := "US"
	if len(siteList) > 0 && len(siteList[0].SubSites) > 0 {
		subSite = siteList[0].SubSites[0]
	}
	return []offlinepackage.SitePrice{{SubSite: subSite, BasePrice: common.FormatFloat(price.Amount), Currency: price.Currency}}
}

func firstSupplierCode(skcs []offlinepackage.SKCPackage) string {
	if len(skcs) == 0 {
		return ""
	}
	return skcs[0].SupplierCode
}

func componentLogger(logs func(string) *logrus.Entry, component string) *logrus.Entry {
	if logs != nil {
		if entry := logs(component); entry != nil {
			return entry
		}
	}
	disabled := logrus.New()
	disabled.SetOutput(io.Discard)
	return disabled.WithField("component", component)
}

func buildImageDraft(images *common.ImageSet) *offlinepackage.ImageDraft {
	if images == nil {
		return nil
	}
	return &offlinepackage.ImageDraft{
		MainImage: images.MainImage,
		Gallery:   append([]string(nil), images.Gallery...),
		WhiteBg:   images.WhiteBgImage,
		Source:    append([]string(nil), images.SourceImages...),
	}
}
