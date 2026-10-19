package config

import (
	"testing"
)

// TestDefaultCostCalculationConfig 测试默认成本计算配置
func TestDefaultCostCalculationConfig(t *testing.T) {
	config := DefaultCostCalculationConfig()

	if config == nil {
		t.Fatal("DefaultCostCalculationConfig 返回 nil")
	}

	// 验证所有默认值都是0
	if config.FixedCostAmount != 0 {
		t.Errorf("FixedCostAmount = %v, 期望 0", config.FixedCostAmount)
	}

	if config.FixedCostPercent != 0 {
		t.Errorf("FixedCostPercent = %v, 期望 0", config.FixedCostPercent)
	}

	if config.ShippingCost != 0 {
		t.Errorf("ShippingCost = %v, 期望 0", config.ShippingCost)
	}

	if config.ProcessingFee != 0 {
		t.Errorf("ProcessingFee = %v, 期望 0", config.ProcessingFee)
	}

	if config.PlatformCommission != 0 {
		t.Errorf("PlatformCommission = %v, 期望 0", config.PlatformCommission)
	}
}

// TestCalculateTotalCost 测试总成本计算
func TestCalculateTotalCost(t *testing.T) {
	tests := []struct {
		name      string
		config    *CostCalculationConfig
		basePrice float64
		expected  float64
	}{
		{
			name:      "默认配置（无额外成本）",
			config:    DefaultCostCalculationConfig(),
			basePrice: 100.0,
			expected:  100.0,
		},
		{
			name: "仅固定金额成本",
			config: &CostCalculationConfig{
				FixedCostAmount: 5.0,
			},
			basePrice: 100.0,
			expected:  105.0,
		},
		{
			name: "仅固定百分比成本（10%）",
			config: &CostCalculationConfig{
				FixedCostPercent: 10.0,
			},
			basePrice: 100.0,
			expected:  110.0,
		},
		{
			name: "仅运费成本",
			config: &CostCalculationConfig{
				ShippingCost: 8.0,
			},
			basePrice: 100.0,
			expected:  108.0,
		},
		{
			name: "仅处理费用",
			config: &CostCalculationConfig{
				ProcessingFee: 3.0,
			},
			basePrice: 100.0,
			expected:  103.0,
		},
		{
			name: "仅平台佣金（5%）",
			config: &CostCalculationConfig{
				PlatformCommission: 5.0,
			},
			basePrice: 100.0,
			expected:  105.0,
		},
		{
			name: "所有成本组合",
			config: &CostCalculationConfig{
				FixedCostAmount:    5.0,  // +5
				FixedCostPercent:   10.0, // +10 (10% of 100)
				ShippingCost:       8.0,  // +8
				ProcessingFee:      3.0,  // +3
				PlatformCommission: 5.0,  // +5 (5% of 100)
			},
			basePrice: 100.0,
			expected:  131.0, // 100 + 5 + 10 + 8 + 3 + 5
		},
		{
			name: "零基础价格",
			config: &CostCalculationConfig{
				FixedCostAmount: 5.0,
				ShippingCost:    8.0,
			},
			basePrice: 0,
			expected:  13.0, // 0 + 5 + 8
		},
		{
			name: "小数基础价格",
			config: &CostCalculationConfig{
				FixedCostPercent: 10.0,
			},
			basePrice: 99.99,
			expected:  109.989, // 99.99 + 9.999
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.CalculateTotalCost(tt.basePrice)
			// 使用小的误差范围比较浮点数
			if diff := result - tt.expected; diff < -0.001 || diff > 0.001 {
				t.Errorf("CalculateTotalCost() = %v, 期望 %v", result, tt.expected)
			}
		})
	}
}

// BenchmarkCalculateTotalCost 性能基准测试
func BenchmarkCalculateTotalCost(b *testing.B) {
	config := &CostCalculationConfig{
		FixedCostAmount:    5.0,
		FixedCostPercent:   10.0,
		ShippingCost:       8.0,
		ProcessingFee:      3.0,
		PlatformCommission: 5.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config.CalculateTotalCost(100.0)
	}
}
