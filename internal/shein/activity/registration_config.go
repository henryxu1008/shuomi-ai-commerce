package activity

import (
	"fmt"
	"strings"

	"task-processor/internal/listingruntime"
	"task-processor/internal/shein/api/marketing"

	"github.com/sirupsen/logrus"
)

// activityRuleInputs contains only values read by stateless platform rules.
// Platform request metadata is supplied through the existing marketing DTOs.
type activityRuleInputs struct {
	PriceMode            string
	DiscountRate         float64
	MinProfitRate        float64
	FixedPriceAdjustment float64
	StockLimit           bool
	StockPercent         int
	FilterSkcList        []string
	AllowRiskProducts    bool
	MaxWarningValue      float64
}

func buildActivityConfigsFromProvidedProducts(
	products []marketing.SkcInfo,
	strategy *listingruntime.OperationStrategy,
	priceMode string,
	logger *logrus.Entry,
) []marketing.ActivityConfig {
	if strategy == nil || len(products) == 0 {
		return nil
	}

	configList := make([]marketing.ActivityConfig, 0, len(products))
	for _, product := range products {
		if product.IsConfigured || product.Skc == "" {
			continue
		}
		if product.Stock <= 0 {
			logger.Warnf("产品 [%s] 预留库存为0，跳过", product.Skc)
			continue
		}

		dropRate, ok := dropRateFromProvidedProduct(product, strategy, priceMode, logger)
		if !ok {
			continue
		}
		if priceMode == "BREAKEVEN" && strings.EqualFold(strategy.ActivityPartakeType, "BOTH") && dropRate <= 1 {
			logger.Warnf("产品 [%s] 保本降幅不足以同时配置常规和限量活动，跳过", product.Skc)
			continue
		}
		actStock := calculateActivityStock(product.Stock, autoPartakeStockRatioFromStrategy(strategy))
		if actStock <= 0 {
			logger.Warnf("产品 [%s] 活动库存为0，跳过", product.Skc)
			continue
		}

		configList = append(configList, marketing.ActivityConfig{
			Skc:               product.Skc,
			ActStock:          actStock,
			DropRate:          dropRate,
			ReservedActStock:  product.Stock,
			SitePriceInfoList: []marketing.ActivitySitePriceInfo{},
		})
	}
	if len(configList) > 0 {
		logger.Infof("使用候选商品快照构建活动配置完成 - 成功: %d", len(configList))
	}
	return configList
}

func dropRateFromProvidedProduct(
	product marketing.SkcInfo,
	strategy *listingruntime.OperationStrategy,
	priceMode string,
	logger *logrus.Entry,
) (int, bool) {
	if priceMode == "PROFIT" || priceMode == "BREAKEVEN" {
		minProfitRate := strategy.ActivityMinProfitRate
		if minProfitRate < 0 || minProfitRate >= 1 {
			minProfitRate = 0.15
		}
		if dropRate, ok, hasSKUData := minimumPromotionSKUDiscountRate(
			product, minProfitRate, strategy.FixedPriceAdjustment, priceMode,
		); hasSKUData {
			if !ok {
				logger.Warnf("产品 [%s] SKU价格或成本数据不完整，跳过", product.Skc)
			}
			return dropRate, ok
		}
	}

	salePrice := firstAvailablePromotionSalePrice(product.SitePriceInfoList)
	if salePrice <= 0 {
		logger.Warnf("产品 [%s] 没有站点价格信息，跳过", product.Skc)
		return 0, false
	}

	if priceMode == "PROFIT" {
		minProfitRate := strategy.ActivityMinProfitRate
		if minProfitRate < 0 || minProfitRate >= 1 {
			minProfitRate = 0.15
		}
		if product.SupplyPrice <= 0 {
			logger.Warnf("产品 [%s] 供货成本价为空，跳过", product.Skc)
			return 0, false
		}
		activityPrice := calculatePriceByProfit(salePrice, product.SupplyPrice, minProfitRate, strategy.FixedPriceAdjustment)
		if activityPrice <= 0 {
			minPrice := product.SupplyPrice/(1-minProfitRate) + strategy.FixedPriceAdjustment
			logger.Warnf("产品 [%s] 利润率不足 (原价: %.2f, 成本: %.2f, 最低售价: %.2f, 固定调整: %.2f, 要求利润率: %.2f%%)，跳过",
				product.Skc, salePrice, product.SupplyPrice, minPrice, strategy.FixedPriceAdjustment, minProfitRate*100)
			return 0, false
		}
		discountRate := (salePrice - activityPrice) / salePrice
		return ValidateDropRate(int(discountRate*100), discountRate, logger), true
	}
	if priceMode == "BREAKEVEN" {
		if product.SupplyPrice <= 0 {
			logger.Warnf("产品 [%s] 供货成本价为空，跳过", product.Skc)
			return 0, false
		}
		activityPrice := product.SupplyPrice + strategy.FixedPriceAdjustment
		if activityPrice >= salePrice {
			logger.Warnf("产品 [%s] 保本活动价不低于销售价 (原价: %.2f, 成本: %.2f, 固定调整: %.2f)，跳过",
				product.Skc, salePrice, product.SupplyPrice, strategy.FixedPriceAdjustment)
			return 0, false
		}
		discountRate := (salePrice - activityPrice) / salePrice
		return ValidateDropRate(int(discountRate*100), discountRate, logger), true
	}

	discountRate := strategy.ActivityDiscountRate
	dropRate := CalculateDropRateFromDiscount(discountRate, logger)
	if product.SupplyPrice > 0 {
		profitMargin := (salePrice - product.SupplyPrice) / product.SupplyPrice
		if profitMargin < discountRate {
			logger.WithField("skc", product.Skc).Info("产品利润率不足，无法支持降价，已过滤")
			return 0, false
		}
	}
	return dropRate, true
}

func calculateEquivalentDropRate(
	originalPrices []marketing.SitePriceInfo,
	activityPrices []marketing.ActivitySitePriceInfo,
	logger *logrus.Entry,
) int {
	if len(originalPrices) == 0 || len(activityPrices) == 0 {
		return 10 // 默认10%
	}

	// 使用第一个站点的价格计算等效折扣率
	originalPrice := originalPrices[0].SalePrice
	activityPrice := activityPrices[0].SalePrice

	if originalPrice <= 0 {
		return 10
	}

	// 计算折扣率
	discountRate := (originalPrice - activityPrice) / originalPrice
	dropRate := int(discountRate * 100)

	// 确保在合理范围内（SHEIN API要求1-99）
	dropRate = ValidateDropRate(dropRate, discountRate, logger)

	return dropRate
}

func minimumPromotionSKUDiscountRate(
	product marketing.SkcInfo,
	minProfitRate float64,
	fixedPriceAdjustment float64,
	priceMode string,
) (int, bool, bool) {
	if len(product.SkuPriceInfoList) == 0 && len(product.SkuCostPriceInfoList) == 0 {
		return 0, false, false
	}
	if len(product.SkuPriceInfoList) == 0 || len(product.SkuCostPriceInfoList) == 0 {
		return 0, false, true
	}

	costBySKU := make(map[string]float64, len(product.SkuCostPriceInfoList))
	for _, item := range product.SkuCostPriceInfoList {
		skuCode := normalizePromotionSKUCode(item.SkuCode)
		if skuCode == "" || item.CostPrice <= 0 {
			return 0, false, true
		}
		if _, exists := costBySKU[skuCode]; exists {
			return 0, false, true
		}
		costBySKU[skuCode] = item.CostPrice
	}
	if len(costBySKU) != len(product.SkuPriceInfoList) {
		return 0, false, true
	}

	// The activity drop rate is based on the synchronized SHEIN supply price.
	// SKU retail prices remain required for a complete multi-SKU snapshot, but
	// are only a legacy fallback when the supply price has not been synchronized.
	lowestOriginalPrice := product.SupplyPrice
	highestCostPrice := 0.0
	seenSKU := make(map[string]struct{}, len(product.SkuPriceInfoList))
	for _, item := range product.SkuPriceInfoList {
		skuCode := normalizePromotionSKUCode(item.SkuCode)
		originalPrice := firstAvailablePromotionSalePrice(item.SitePriceInfoList)
		costPrice, ok := costBySKU[skuCode]
		if skuCode == "" || originalPrice <= 0 || !ok || costPrice <= 0 {
			return 0, false, true
		}
		if _, exists := seenSKU[skuCode]; exists {
			return 0, false, true
		}
		seenSKU[skuCode] = struct{}{}
		if product.SupplyPrice <= 0 && (lowestOriginalPrice <= 0 || originalPrice < lowestOriginalPrice) {
			lowestOriginalPrice = originalPrice
		}
		if costPrice > highestCostPrice {
			highestCostPrice = costPrice
		}
	}

	activityPrice := calculatePriceByBreakeven(lowestOriginalPrice, highestCostPrice, fixedPriceAdjustment)
	if strings.EqualFold(priceMode, "PROFIT") {
		activityPrice = calculatePriceByProfit(lowestOriginalPrice, highestCostPrice, minProfitRate, fixedPriceAdjustment)
	}
	if activityPrice <= 0 || activityPrice >= lowestOriginalPrice {
		return 0, false, true
	}
	rate := (lowestOriginalPrice - activityPrice) / lowestOriginalPrice
	return ValidateDropRate(int(rate*100), rate, nil), true, true
}

func normalizePromotionSKUCode(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func firstAvailablePromotionSalePrice(items []marketing.SitePriceInfo) float64 {
	for _, item := range items {
		if item.SalePrice <= 0 {
			continue
		}
		if !item.IsAvailable {
			continue
		}
		return item.SalePrice
	}
	for _, item := range items {
		if item.SalePrice > 0 {
			return item.SalePrice
		}
	}
	return 0
}

func calculateActivityStock(totalStock int, stockRatio float64) int {
	// 确保库存比例在有效范围内
	if stockRatio <= 0 {
		stockRatio = 1.0 // 默认使用全部库存
	} else if stockRatio > 1.0 {
		stockRatio = 1.0
	}

	actStock := int(float64(totalStock) * stockRatio)

	// 至少保留1个库存用于活动
	if actStock < 1 && totalStock > 0 {
		actStock = 1
	}

	return actStock
}

func ensureLimitedPromotionDropRatesGreater(regularConfigs, limitedConfigs []marketing.ActivityConfig) []marketing.ActivityConfig {
	regularDropRateBySKC := make(map[string]int, len(regularConfigs))
	for _, config := range regularConfigs {
		if config.Skc == "" {
			continue
		}
		regularDropRateBySKC[config.Skc] = config.DropRate
	}
	if len(regularDropRateBySKC) == 0 {
		return limitedConfigs
	}

	out := append([]marketing.ActivityConfig(nil), limitedConfigs...)
	for i := range out {
		regularDropRate, ok := regularDropRateBySKC[out[i].Skc]
		if !ok || out[i].DropRate > regularDropRate || regularDropRate >= 99 {
			continue
		}
		out[i].DropRate = regularDropRate + 1
	}
	return out
}

func validateAutoPartakeDiscountsForStrategy(strategy *listingruntime.OperationStrategy) error {
	if strategy == nil {
		return nil
	}
	if strings.ToUpper(strings.TrimSpace(strategy.ActivityPartakeType)) != "BOTH" {
		return nil
	}
	switch autoPartakePriceModeFromStrategy(strategy) {
	case "DISCOUNT":
		if strategy.ActivityLimitedDiscountRate <= 0 || strategy.ActivityLimitedDiscountRate >= 1 {
			return fmt.Errorf("限量活动折扣率必须在 0 到 1 之间")
		}
		if strategy.ActivityLimitedDiscountRate <= strategy.ActivityDiscountRate {
			return fmt.Errorf("同时报名常规和限量活动时，限量活动折扣率必须大于常规活动折扣率")
		}
		return nil
	case "PROFIT":
		if strategy.ActivityLimitedMinProfitRate < 0 || strategy.ActivityLimitedMinProfitRate >= 1 {
			return fmt.Errorf("限量活动最低利润率必须在 0 到 1 之间")
		}
		if strategy.ActivityLimitedMinProfitRate >= strategy.ActivityMinProfitRate {
			return fmt.Errorf("同时报名常规和限量活动时，限量活动最低利润率必须小于常规活动最低利润率")
		}
	}
	return nil
}

func autoPartakePriceModeFromStrategy(strategy *listingruntime.OperationStrategy) string {
	if strategy == nil {
		return "DISCOUNT"
	}
	priceMode := strings.ToUpper(strings.TrimSpace(strategy.ActivityPriceMode))
	if priceMode == "" {
		return "DISCOUNT"
	}
	return priceMode
}

func autoPartakeActivityTypesFromStrategy(strategy *listingruntime.OperationStrategy) []int {
	if strategy == nil {
		return []int{marketing.AutoPartakeActivityTypeRegular}
	}
	switch strings.ToUpper(strings.TrimSpace(strategy.ActivityPartakeType)) {
	case "LIMITED":
		return []int{marketing.AutoPartakeActivityTypeLimited}
	case "BOTH":
		return []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}
	default:
		return []int{marketing.AutoPartakeActivityTypeRegular}
	}
}

func autoPartakeStockRatioFromStrategy(strategy *listingruntime.OperationStrategy) float64 {
	if strategy == nil {
		return 1
	}
	if !autoPartakeRequiresStockRatio(strategy) && strategy.ActivityStockRatio <= 0 {
		return 1
	}
	return strategy.ActivityStockRatio
}

func autoPartakeRequiresStockRatio(strategy *listingruntime.OperationStrategy) bool {
	if strategy == nil {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(strategy.ActivityPartakeType)) {
	case "LIMITED", "BOTH":
		return true
	default:
		return false
	}
}

func promotionConfigListForActivityType(products []marketing.SkcInfo, configList []marketing.ActivityConfig, activityType int, strategy *listingruntime.OperationStrategy, priceMode string, logger *logrus.Entry) []marketing.ActivityConfig {
	copied := append([]marketing.ActivityConfig(nil), configList...)
	if activityType == marketing.AutoPartakeActivityTypeRegular &&
		priceMode == "BREAKEVEN" &&
		strategy != nil &&
		strings.EqualFold(strategy.ActivityPartakeType, "BOTH") {
		for i := range copied {
			copied[i].DropRate--
		}
		return copied
	}
	if activityType != marketing.AutoPartakeActivityTypeLimited || strategy == nil {
		return copied
	}
	if priceMode == "PROFIT" && strategy.ActivityLimitedMinProfitRate >= 0 && strategy.ActivityLimitedMinProfitRate < 1 {
		limitedStrategy := *strategy
		limitedStrategy.ActivityMinProfitRate = strategy.ActivityLimitedMinProfitRate
		limitedConfigList := buildActivityConfigsFromProvidedProducts(products, &limitedStrategy, priceMode, logger)
		if len(limitedConfigList) > 0 {
			return ensureLimitedPromotionDropRatesGreater(copied, limitedConfigList)
		}
		return copied
	}
	if autoPartakePriceModeFromStrategy(strategy) != "DISCOUNT" || strategy.ActivityLimitedDiscountRate <= 0 {
		return copied
	}
	limitedDropRate := CalculateDropRateFromDiscount(strategy.ActivityLimitedDiscountRate, nil)
	for i := range copied {
		copied[i].DropRate = limitedDropRate
	}
	return copied
}
