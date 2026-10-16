package activity

import (
	"math"
	"reflect"
	"testing"

	"task-processor/internal/shein"
	"task-processor/internal/shein/api/marketing"
	"task-processor/internal/shein/api/product"
	"task-processor/internal/shein/productsync"
)

const floatTolerance = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < floatTolerance
}

func TestPromotionSKUUSSupplyPriceDoesNotUseSKCFallback(t *testing.T) {
	if got := promotionSKUUSSupplyPrice(marketing.PromotionSkuInfo{}, 10); got != 0 {
		t.Fatalf("price = %v, want zero without direct SKU supply price", got)
	}
}

func TestAmazonCostBySKUUsesNormalizedSKUAndSkipsMissingCosts(t *testing.T) {
	data := &productsync.EnrichedSkcInfo{SkuInfo: []productsync.EnrichedSkuInfo{
		{
			SkuInfo:           product.SkuInfo{SkuCode: "sku-small"},
			AmazonMonitorData: &shein.AmazonMonitorData{Price: 12.5},
		},
		{
			SkuInfo:           product.SkuInfo{SkuCode: "SKU-LARGE"},
			AmazonMonitorData: &shein.AmazonMonitorData{Price: 20.5},
		},
		{SkuInfo: product.SkuInfo{SkuCode: "sku-missing"}},
	}}

	got := amazonCostBySKU(data)
	want := map[string]float64{"SKU-SMALL": 12.5, "SKU-LARGE": 20.5}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cost map = %#v, want %#v", got, want)
	}
}

func TestCalculatePriceByDiscount(t *testing.T) {
	tests := []struct {
		name          string
		originalPrice float64
		discountRate  float64
		want          float64
	}{
		{"40%折扣", 100.0, 0.4, 60.0},
		{"零折扣", 100.0, 0.0, 100.0},
		{"全额折扣", 100.0, 1.0, 0.0},
		{"小数价格", 29.99, 0.2, 23.992},
		{"零原价", 0.0, 0.5, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePriceByDiscount(tt.originalPrice, tt.discountRate)
			if !almostEqual(got, tt.want) {
				t.Errorf("calculatePriceByDiscount(%v, %v) = %v, want %v",
					tt.originalPrice, tt.discountRate, got, tt.want)
			}
		})
	}
}

func TestCalculatePriceByProfit(t *testing.T) {
	tests := []struct {
		name            string
		originalPrice   float64
		costPrice       float64
		minProfitRate   float64
		fixedAdjustment float64
		want            float64
		description     string
	}{
		{
			name:            "原价高于最低售价，返回最低售价",
			originalPrice:   100.0,
			costPrice:       50.0,
			minProfitRate:   0.15,
			fixedAdjustment: 0,
			// minPrice = 50/(1-0.15) = 58.82...
			want:        50.0 / (1 - 0.15),
			description: "成本50，利润率15%，最低售价约58.82",
		},
		{
			name:            "原价低于最低售价，返回0",
			originalPrice:   50.0,
			costPrice:       50.0,
			minProfitRate:   0.15,
			fixedAdjustment: 0,
			want:            0,
			description:     "原价等于成本，利润率不足，返回0",
		},
		{
			name:            "带固定调整值",
			originalPrice:   100.0,
			costPrice:       50.0,
			minProfitRate:   0.15,
			fixedAdjustment: 5.0,
			// minPrice = 50/(1-0.15) + 5 = 63.82...
			want:        50.0/(1-0.15) + 5.0,
			description: "加固定调整值5",
		},
		{
			name:            "零成本价",
			originalPrice:   100.0,
			costPrice:       0.0,
			minProfitRate:   0.15,
			fixedAdjustment: 0,
			want:            0.0, // minPrice=0, activityPrice=0
			description:     "成本为0，最低售价为0，返回0",
		},
		{
			// 原价略高于最低售价（避免浮点精度问题）
			name:            "原价略高于最低售价",
			originalPrice:   60.0,
			costPrice:       50.0,
			minProfitRate:   0.15,
			fixedAdjustment: 0,
			// minPrice = 50/(1-0.15) ≈ 58.82，60 > 58.82，返回 minPrice
			want:        50.0 / (1 - 0.15),
			description: "原价60略高于最低售价58.82，返回最低售价",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePriceByProfit(tt.originalPrice, tt.costPrice, tt.minProfitRate, tt.fixedAdjustment)
			if !almostEqual(got, tt.want) {
				t.Errorf("calculatePriceByProfit(%v, %v, %v, %v) = %v, want %v (%s)",
					tt.originalPrice, tt.costPrice, tt.minProfitRate, tt.fixedAdjustment,
					got, tt.want, tt.description)
			}
		})
	}
}

func TestCalculatePriceByBreakevenCapsSHEINDiscountAtEightyPercent(t *testing.T) {
	got := calculatePriceByBreakeven(124.96, 24.88, 0)
	if !almostEqual(got, 25) {
		t.Fatalf("breakeven activity price = %v, want 25.00 to cap the discount at 80%%", got)
	}
}

func TestCalculateActivityPriceCapsSHEINDiscountAtEightyPercentInEveryMode(t *testing.T) {
	tests := []struct {
		name   string
		config activityRuleInputs
		cost   float64
	}{
		{name: "breakeven", config: activityRuleInputs{PriceMode: "BREAKEVEN"}, cost: 24.88},
		{name: "profit", config: activityRuleInputs{PriceMode: "PROFIT", MinProfitRate: 0.1}, cost: 10},
		{name: "discount", config: activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.9}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateActivityPrice(tt.config, 124.96, tt.cost)
			if !almostEqual(got, 25) {
				t.Fatalf("activity price = %v, want 25.00 to cap the discount at 80%%", got)
			}
		})
	}
}

func TestCalculateProfitRate(t *testing.T) {
	tests := []struct {
		name      string
		salePrice float64
		costPrice float64
		want      float64
	}{
		{"正常利润率", 100.0, 80.0, 0.2},
		{"零利润", 100.0, 100.0, 0.0},
		{"零售价返回0", 0.0, 50.0, 0.0},
		{"负售价返回0", -10.0, 50.0, 0.0},
		{"成本为零", 100.0, 0.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateProfitRate(tt.salePrice, tt.costPrice)
			if !almostEqual(got, tt.want) {
				t.Errorf("calculateProfitRate(%v, %v) = %v, want %v",
					tt.salePrice, tt.costPrice, got, tt.want)
			}
		})
	}
}

func TestCalculateActivityPrice(t *testing.T) {
	tests := []struct {
		name          string
		config        activityRuleInputs
		originalPrice float64
		costPrice     float64
		wantFn        func(float64) bool
		description   string
	}{
		{
			name: "DISCOUNT模式使用折扣率",
			config: activityRuleInputs{
				PriceMode:    "DISCOUNT",
				DiscountRate: 0.3,
			},
			originalPrice: 100.0,
			costPrice:     0,
			wantFn:        func(got float64) bool { return almostEqual(got, 70.0) },
			description:   "100 * (1-0.3) = 70",
		},
		{
			name: "PROFIT模式使用利润率",
			config: activityRuleInputs{
				PriceMode:     "PROFIT",
				MinProfitRate: 0.15,
			},
			originalPrice: 100.0,
			costPrice:     50.0,
			wantFn: func(got float64) bool {
				return almostEqual(got, 50.0/(1-0.15))
			},
			description: "按利润率计算最低售价",
		},
		{
			name: "未知模式默认使用利润率",
			config: activityRuleInputs{
				PriceMode:     "UNKNOWN",
				MinProfitRate: 0.15,
			},
			originalPrice: 100.0,
			costPrice:     50.0,
			wantFn: func(got float64) bool {
				return almostEqual(got, 50.0/(1-0.15))
			},
			description: "default 分支走 calculatePriceByProfit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateActivityPrice(tt.config, tt.originalPrice, tt.costPrice)
			if !tt.wantFn(got) {
				t.Errorf("calculateActivityPrice() = %v, description: %s", got, tt.description)
			}
		})
	}
}

func TestBreakevenPricesUseExtractedAmazonCosts(t *testing.T) {
	data := &productsync.EnrichedSkcInfo{SkuInfo: []productsync.EnrichedSkuInfo{
		{SkuInfo: product.SkuInfo{SkuCode: "sku-small"}, AmazonMonitorData: &shein.AmazonMonitorData{Price: 12.5}},
		{SkuInfo: product.SkuInfo{SkuCode: "sku-large"}, AmazonMonitorData: &shein.AmazonMonitorData{Price: 20.5}},
		{SkuInfo: product.SkuInfo{SkuCode: "sku-invalid"}, AmazonMonitorData: &shein.AmazonMonitorData{Price: 20}},
	}}
	costs := amazonCostBySKU(data)
	for _, tc := range []struct {
		sku            string
		original, want float64
	}{
		{"SKU-SMALL", 30, 13.5}, {"SKU-LARGE", 45, 21.5}, {"SKU-INVALID", 18, 0},
	} {
		if got := calculatePriceByBreakeven(tc.original, costs[tc.sku], 1); !almostEqual(got, tc.want) {
			t.Fatalf("%s: activity price %v, want %v", tc.sku, got, tc.want)
		}
	}
}
