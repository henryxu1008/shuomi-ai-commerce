package activity

import (
	"math"

	"task-processor/internal/shein/api/marketing"
	"task-processor/internal/shein/productsync"
)

func calculateActivityPrice(
	config activityRuleInputs,
	originalPrice float64,
	costPrice float64,
) float64 {
	switch config.PriceMode {
	case "PROFIT":
		// 按最低利润率定价
		return calculatePriceByProfit(originalPrice, costPrice, config.MinProfitRate, config.FixedPriceAdjustment)
	case "BREAKEVEN":
		return calculatePriceByBreakeven(originalPrice, costPrice, config.FixedPriceAdjustment)
	case "DISCOUNT":
		// 按折扣率定价
		return calculatePriceByDiscount(originalPrice, config.DiscountRate)
	default:
		// 默认按折扣率定价
		return calculatePriceByProfit(originalPrice, costPrice, config.MinProfitRate, config.FixedPriceAdjustment)
	}
}

func calculatePriceByDiscount(originalPrice float64, discountRate float64) float64 {
	return capSHEINActivityPrice(originalPrice, originalPrice*(1-discountRate))
}

func calculatePriceByProfit(originalPrice float64, costPrice float64, minProfitRate float64, fixedAdjustment float64) float64 {
	// 计算最低售价 = 成本价 / (1 - 最低利润率) + 固定调整值
	minPrice := costPrice/(1-minProfitRate) + fixedAdjustment

	// 如果原价低于最低售价
	if originalPrice < minPrice {
		return 0
	}

	// 计算活动价格：取原价和最低售价中的较大值
	activityPrice := minPrice
	if activityPrice > originalPrice {
		activityPrice = originalPrice
	}

	return capSHEINActivityPrice(originalPrice, activityPrice)
}

func calculatePriceByBreakeven(originalPrice float64, costPrice float64, fixedAdjustment float64) float64 {
	activityPrice := costPrice + fixedAdjustment
	if costPrice <= 0 || originalPrice <= 0 {
		return 0
	}
	activityPrice = capSHEINActivityPrice(originalPrice, activityPrice)
	if activityPrice >= originalPrice {
		return 0
	}
	return activityPrice
}

func capSHEINActivityPrice(originalPrice float64, activityPrice float64) float64 {
	if originalPrice <= 0 || activityPrice <= 0 {
		return activityPrice
	}
	minimumPrice := math.Ceil(originalPrice*(1-sheinMaximumActivityDiscountRate)*100) / 100
	if activityPrice < minimumPrice {
		return minimumPrice
	}
	return activityPrice
}

func calculateProfitRate(salePrice float64, costPrice float64) float64 {
	if salePrice <= 0 {
		return 0
	}
	return (salePrice - costPrice) / salePrice
}

func promotionSKUUSSupplyPrice(sku marketing.PromotionSkuInfo, _ float64) float64 {
	if sku.USSupplyPrice != nil && *sku.USSupplyPrice > 0 {
		return *sku.USSupplyPrice
	}
	if sku.SupplyPrice != nil && *sku.SupplyPrice > 0 {
		return *sku.SupplyPrice
	}
	if sku.SupplyPriceInfo != nil && sku.SupplyPriceInfo.SupplyPrice > 0 {
		return sku.SupplyPriceInfo.SupplyPrice
	}
	return 0
}

func promotionSKUMaxUSSupplyPrice(sku marketing.PromotionSkuInfo, fallback float64) float64 {
	if sku.MaxUSSupplyPrice != nil && *sku.MaxUSSupplyPrice > 0 {
		return *sku.MaxUSSupplyPrice
	}
	if sku.MaxSupplyPrice != nil && *sku.MaxSupplyPrice > 0 {
		return *sku.MaxSupplyPrice
	}
	return fallback
}

func amazonCostBySKU(skcData *productsync.EnrichedSkcInfo) map[string]float64 {
	costs := make(map[string]float64)
	if skcData == nil {
		return costs
	}
	for _, sku := range skcData.SkuInfo {
		code := normalizePromotionSKUCode(sku.SkuCode)
		if code == "" || sku.AmazonMonitorData == nil || sku.AmazonMonitorData.Price <= 0 {
			continue
		}
		costs[code] = sku.AmazonMonitorData.Price
	}
	return costs
}
