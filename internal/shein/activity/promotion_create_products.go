package activity

import (
	"strings"

	"task-processor/internal/shein/api/marketing"
)

func applyPromotionCreateConfig(config *activityRuleInputs, strategy *activityStrategy) {
	if config == nil || strategy == nil {
		return
	}
	priceMode := strings.ToUpper(strings.TrimSpace(strategy.ActivityPriceMode))
	if priceMode != "" {
		config.PriceMode = priceMode
	}
	if strategy.ActivityDiscountRate > 0 && strategy.ActivityDiscountRate < 1 {
		config.DiscountRate = strategy.ActivityDiscountRate
	}
	if strategy.ActivityMinProfitRate >= 0 && strategy.ActivityMinProfitRate < 1 {
		config.MinProfitRate = strategy.ActivityMinProfitRate
	}
	if strategy.ActivityStockRatio > 0 && strategy.ActivityStockRatio < 1 {
		config.StockLimit = true
		config.StockPercent = int(strategy.ActivityStockRatio * 100)
		if config.StockPercent < 1 {
			config.StockPercent = 1
		}
	}
}

func selectedPromotionSKCs(products []marketing.SkcInfo) []string {
	selected := make([]string, 0, len(products))
	seen := make(map[string]struct{}, len(products))
	for _, product := range products {
		skc := strings.TrimSpace(product.Skc)
		if skc == "" {
			continue
		}
		if _, ok := seen[skc]; ok {
			continue
		}
		seen[skc] = struct{}{}
		selected = append(selected, skc)
	}
	return selected
}

func filterPromotionGoodsBySKC(goods []marketing.PromotionGoodsData, selected []string) []marketing.PromotionGoodsData {
	if len(selected) == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(selected))
	for _, skc := range selected {
		allowed[skc] = struct{}{}
	}
	filtered := make([]marketing.PromotionGoodsData, 0, len(goods))
	for _, item := range goods {
		if _, ok := allowed[item.Skc]; ok {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func promotionGoodsFromProductSnapshots(products []marketing.SkcInfo, _ string) []marketing.PromotionGoodsData {
	goods := make([]marketing.PromotionGoodsData, 0, len(products))
	for _, product := range products {
		skc := strings.TrimSpace(product.Skc)
		if skc == "" || product.Stock <= 0 {
			continue
		}
		supplyPrice := product.SupplyPrice
		skuInfoList := make([]marketing.PromotionSkuInfo, 0, len(product.SkuPriceInfoList))
		for _, skuPrice := range product.SkuPriceInfoList {
			skuCode := strings.TrimSpace(skuPrice.SkuCode)
			if skuCode == "" {
				continue
			}
			skuInfoList = append(skuInfoList, marketing.PromotionSkuInfo{Sku: skuCode})
		}
		if len(skuInfoList) == 0 {
			continue
		}
		isSaleAttribute := 0
		if len(skuInfoList) > 1 {
			isSaleAttribute = 1
		}
		goods = append(goods, marketing.PromotionGoodsData{
			Skc:              skc,
			InventoryNum:     product.Stock,
			IsSaleAttribute:  isSaleAttribute,
			USSupplyPrice:    supplyPrice,
			MaxUSSupplyPrice: supplyPrice,
			SkuInfoList:      skuInfoList,
		})
	}
	return goods
}

func enrichPromotionGoodsFromProductSnapshots(
	goods []marketing.PromotionGoodsData,
	products []marketing.SkcInfo,
	currency string,
) []marketing.PromotionGoodsData {
	productBySKC := make(map[string]marketing.SkcInfo, len(products))
	for _, product := range products {
		if strings.TrimSpace(product.Skc) != "" {
			productBySKC[product.Skc] = product
		}
	}

	enriched := make([]marketing.PromotionGoodsData, 0, len(goods))
	for _, item := range goods {
		product, ok := productBySKC[item.Skc]
		if !ok {
			enriched = append(enriched, item)
			continue
		}
		if item.InventoryNum <= 0 && product.Stock > 0 {
			item.InventoryNum = product.Stock
		}
		enriched = append(enriched, item)
	}
	return enriched
}

func configCurrencyOrDefault(currency string) string {
	currency = strings.TrimSpace(strings.ToUpper(currency))
	if currency == "" {
		return "USD"
	}
	return currency
}

type promotionSKUPriceInput struct {
	SKU         string
	RetailPrice float64
	CostPrice   float64
	Currency    string
}

func promotionSKUPriceInputs(products []marketing.SkcInfo, currency string) map[string]map[string]promotionSKUPriceInput {
	targetCurrency := configCurrencyOrDefault(currency)
	inputsBySKC := make(map[string]map[string]promotionSKUPriceInput, len(products))
	for _, product := range products {
		skc := strings.TrimSpace(product.Skc)
		if skc == "" {
			continue
		}
		inputsBySKU := inputsBySKC[skc]
		if inputsBySKU == nil {
			inputsBySKU = make(map[string]promotionSKUPriceInput)
			inputsBySKC[skc] = inputsBySKU
		}
		for _, skuSitePrice := range product.SkuPriceInfoList {
			sku := normalizedPromotionSKUCode(skuSitePrice.SkuCode)
			if sku == "" {
				continue
			}
			input := inputsBySKU[sku]
			input.SKU = sku
			input.Currency = targetCurrency
			for _, sitePrice := range skuSitePrice.SitePriceInfoList {
				if input.RetailPrice == 0 && sitePrice.IsAvailable && sitePrice.SalePrice > 0 && strings.EqualFold(sitePrice.Currency, targetCurrency) {
					input.RetailPrice = sitePrice.SalePrice
				}
			}
			inputsBySKU[sku] = input
		}
		for _, skuCost := range product.SkuCostPriceInfoList {
			sku := normalizedPromotionSKUCode(skuCost.SkuCode)
			if sku == "" || skuCost.CostPrice <= 0 || !strings.EqualFold(skuCost.Currency, targetCurrency) {
				continue
			}
			input := inputsBySKU[sku]
			input.SKU = sku
			input.Currency = targetCurrency
			if input.CostPrice == 0 {
				input.CostPrice = skuCost.CostPrice
			}
			inputsBySKU[sku] = input
		}
	}
	return inputsBySKC
}

func normalizedPromotionSKUCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func buildCalculateRequestForPromotionProducts(
	config activityRuleInputs,
	request marketing.CalculateSupplyPriceRequest,
	goods []marketing.PromotionGoodsData,
	products []marketing.SkcInfo,
) *marketing.CalculateSupplyPriceRequest {
	mode := strings.ToUpper(strings.TrimSpace(config.PriceMode))
	priceInputsBySKC := promotionSKUPriceInputs(products, request.Currency)
	skcInfoList := make([]marketing.SkcPriceInfo, 0, len(goods))
	for _, item := range goods {
		priceInputsBySKU := priceInputsBySKC[strings.TrimSpace(item.Skc)]
		skuInfoList := make([]marketing.SkuPriceInfo, 0, len(item.SkuInfoList))
		for _, sku := range item.SkuInfoList {
			input, ok := priceInputsBySKU[normalizedPromotionSKUCode(sku.Sku)]
			if !ok || input.RetailPrice <= 0 {
				continue
			}
			productPrice := input.RetailPrice
			var skuActivityPrice float64
			switch mode {
			case "PROFIT":
				if input.CostPrice <= 0 {
					continue
				}
				skuActivityPrice = calculatePriceByProfit(productPrice, input.CostPrice, config.MinProfitRate, config.FixedPriceAdjustment)
			case "BREAKEVEN":
				if input.CostPrice <= 0 {
					continue
				}
				skuActivityPrice = calculatePriceByBreakeven(productPrice, input.CostPrice, config.FixedPriceAdjustment)
			default:
				skuActivityPrice = calculatePriceByDiscount(productPrice, config.DiscountRate)
			}
			if skuActivityPrice <= 0 {
				continue
			}
			skuInfoList = append(skuInfoList, marketing.SkuPriceInfo{
				SkuCode:       sku.Sku,
				ProductPrice:  productPrice,
				DiscountValue: skuActivityPrice,
			})
		}
		if len(skuInfoList) == 0 {
			continue
		}
		skcInfoList = append(skcInfoList, marketing.SkcPriceInfo{
			SkcName:     item.Skc,
			SkuInfoList: skuInfoList,
		})
	}

	request.SkcInfoList = skcInfoList
	return &request
}
