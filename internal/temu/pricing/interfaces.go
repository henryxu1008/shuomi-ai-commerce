// package pricing 提供核价服务接口定义
package pricing

import (
	"task-processor/internal/listingadmin"
)

// PriceCalculator 价格计算器接口
type PriceCalculator interface {
	GetDefaultPricingRules(originCostPrice float64, rules *[]listingadmin.PricingRuleRespDTO) *listingadmin.PricingRuleRespDTO
	CalculateMinAcceptablePrice(originCostPrice float64, rule *listingadmin.PricingRuleRespDTO) float64
}
