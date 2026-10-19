// Package config 提供平台通用配置类型
package config

// CostCalculationConfig 成本计算配置（平台通用）
// 用于TEMU、SHEIN等平台的成本计算
type CostCalculationConfig struct {
	FixedCostAmount    float64 `json:"fixed_cost_amount" yaml:"fixed_cost_amount"`     // 固定成本金额
	FixedCostPercent   float64 `json:"fixed_cost_percent" yaml:"fixed_cost_percent"`   // 固定成本百分比
	ShippingCost       float64 `json:"shipping_cost" yaml:"shipping_cost"`             // 运费成本
	ProcessingFee      float64 `json:"processing_fee" yaml:"processing_fee"`           // 处理费用
	PlatformCommission float64 `json:"platform_commission" yaml:"platform_commission"` // 平台佣金百分比
}

// DefaultCostCalculationConfig 返回默认成本计算配置
func DefaultCostCalculationConfig() *CostCalculationConfig {
	return &CostCalculationConfig{
		FixedCostAmount:    0,
		FixedCostPercent:   0,
		ShippingCost:       0,
		ProcessingFee:      0,
		PlatformCommission: 0,
	}
}

// CalculateTotalCost 计算总成本
func (c *CostCalculationConfig) CalculateTotalCost(basePrice float64) float64 {
	totalCost := basePrice

	// 添加固定金额成本
	totalCost += c.FixedCostAmount

	// 添加固定百分比成本
	if c.FixedCostPercent > 0 {
		totalCost += basePrice * (c.FixedCostPercent / 100.0)
	}

	// 添加运费成本
	totalCost += c.ShippingCost

	// 添加处理费用
	totalCost += c.ProcessingFee

	// 添加平台佣金
	if c.PlatformCommission > 0 {
		totalCost += basePrice * (c.PlatformCommission / 100.0)
	}

	return totalCost
}
