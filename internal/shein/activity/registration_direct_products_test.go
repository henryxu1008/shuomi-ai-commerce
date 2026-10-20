package activity

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"task-processor/internal/listingruntime"
	"task-processor/internal/shein/api/marketing"

	"github.com/sirupsen/logrus"
)

func TestPromotionGoodsFromProductSnapshotsKeepsRetailOnlySKUForDiscountPricing(t *testing.T) {
	goods := promotionGoodsFromProductSnapshots([]marketing.SkcInfo{{
		Skc: "skc-retail-only", Stock: 1,
		SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 29.9, IsAvailable: true}},
		SkuPriceInfoList:  []marketing.SkuSitePriceInfo{{SkuCode: "sku-1"}},
	}}, "USD")
	if len(goods) != 1 || len(goods[0].SkuInfoList) != 1 {
		t.Fatalf("goods = %+v, want retail-only SKU retained for discount pricing", goods)
	}
}

func TestBuildCalculateRequestForPromotionProductsRequiresDirectSKUPrices(t *testing.T) {
	goods := []marketing.PromotionGoodsData{{
		Skc:           "skc-direct-prices",
		USSupplyPrice: 99,
		SkuInfoList: []marketing.PromotionSkuInfo{
			{Sku: "sku-complete"},
			{Sku: "sku-missing-cost"},
			{Sku: "sku-eur-retail"},
			{Sku: "sku-eur-cost"},
			{Sku: "sku-disabled-retail"},
		},
	}}
	products := []marketing.SkcInfo{{
		Skc: "skc-direct-prices", Stock: 1, SupplyPrice: 99,
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{
			{SkuCode: "sku-complete", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 100, IsAvailable: true}}},
			{SkuCode: "sku-missing-cost", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 80, IsAvailable: true}}},
			{SkuCode: "sku-eur-retail", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "EUR", SalePrice: 500, IsAvailable: true}}},
			{SkuCode: "sku-eur-cost", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 60, IsAvailable: true}}},
			{SkuCode: "sku-disabled-retail", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 70, IsAvailable: false}}},
		},
		SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
			{SkuCode: "sku-complete", CostPrice: 50, Currency: "USD"},
			{SkuCode: "sku-eur-retail", CostPrice: 5, Currency: "USD"},
			{SkuCode: "sku-eur-cost", CostPrice: 1, Currency: "EUR"},
			{SkuCode: "sku-disabled-retail", CostPrice: 10, Currency: "USD"},
		},
	}}

	for _, tc := range []struct {
		name          string
		mode          string
		wantSKUPrices map[string]float64
	}{
		{name: "discount", mode: "DISCOUNT", wantSKUPrices: map[string]float64{"sku-complete": 100, "sku-missing-cost": 80, "sku-eur-cost": 60}},
		{name: "profit", mode: "PROFIT", wantSKUPrices: map[string]float64{"sku-complete": 100}},
		{name: "breakeven", mode: "BREAKEVEN", wantSKUPrices: map[string]float64{"sku-complete": 100}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := buildCalculateRequestForPromotionProducts(activityRuleInputs{
				PriceMode: tc.mode, DiscountRate: 0.2,
			}, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
			if req == nil || len(req.SkcInfoList) != 1 || len(req.SkcInfoList[0].SkuInfoList) != len(tc.wantSKUPrices) {
				t.Fatalf("calculate request = %+v, want SKU prices %+v", req, tc.wantSKUPrices)
			}
			for _, sku := range req.SkcInfoList[0].SkuInfoList {
				wantPrice, ok := tc.wantSKUPrices[sku.SkuCode]
				if !ok || sku.ProductPrice != wantPrice || sku.DiscountValue <= 0 {
					t.Fatalf("calculated SKU = %+v, want direct USD retail price and positive activity price", sku)
				}
			}
		})
	}
}

func TestBuildCreateActivityRequestValidatesEverySKUDiscount(t *testing.T) {
	tests := []struct {
		name                   string
		secondSKUOriginalPrice float64
		secondSKUPrice         float64
		wantIncluded           bool
		wantReasonPrice        string
	}{
		{name: "equal to 95 percent", secondSKUOriginalPrice: 200, secondSKUPrice: 190, wantIncluded: false, wantReasonPrice: "190.00"},
		{name: "equal to 95 percent with decimal rounding", secondSKUOriginalPrice: 16.60, secondSKUPrice: 15.77, wantIncluded: false, wantReasonPrice: "15.77"},
		{name: "above 95 percent", secondSKUOriginalPrice: 200, secondSKUPrice: 191, wantIncluded: false, wantReasonPrice: "191.00"},
		{name: "strictly below 95 percent", secondSKUOriginalPrice: 200, secondSKUPrice: 189.99, wantIncluded: true},
		{name: "strictly below 95 percent with sub-cent precision", secondSKUOriginalPrice: 100, secondSKUPrice: 94.999, wantIncluded: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goods := []marketing.PromotionGoodsData{{
				Skc:              "sg-multi-sku-discount",
				InventoryNum:     100,
				USSupplyPrice:    100,
				MaxUSSupplyPrice: 100,
				SkuInfoList: []marketing.PromotionSkuInfo{
					{Sku: "sku-small", USSupplyPrice: promotionTestFloat64Ptr(100)},
					{Sku: "sku-large", USSupplyPrice: promotionTestFloat64Ptr(tt.secondSKUOriginalPrice)},
				},
			}}
			calcResp := &marketing.CalculateSupplyPriceResponse{Info: []marketing.SkcCalculationResult{{
				SkcName: "sg-multi-sku-discount",
				SkuInfoList: []marketing.SkuCalculationInfo{
					{SkuCode: "sku-small", PriceInfo: marketing.PriceInfo{ProductAmount: 100, PromotionAmount: 20}},
					{SkuCode: "sku-large", PriceInfo: marketing.PriceInfo{ProductAmount: tt.secondSKUOriginalPrice, PromotionAmount: tt.secondSKUOriginalPrice - tt.secondSKUPrice}},
				},
			}}}

			req, _, reasons := buildCreateActivityRequest(
				activityRuleInputs{}, marketing.CreateActivityRequest{}, []int{2},
				goods,
				nil,
				calcResp, logrus.NewEntry(logrus.New()),
			)

			if got := len(req.AddCostAndStockInfoList); (got == 1) != tt.wantIncluded {
				t.Fatalf("created goods count = %d, want included %t", got, tt.wantIncluded)
			}
			if tt.wantIncluded {
				if reason := reasons["sg-multi-sku-discount"]; reason != "" {
					t.Fatalf("filter reason = %q, want empty", reason)
				}
				return
			}
			reason := reasons["sg-multi-sku-discount"]
			for _, want := range []string{"sku-large", tt.wantReasonPrice, fmt.Sprintf("%.2f", tt.secondSKUOriginalPrice), "95%"} {
				if !strings.Contains(reason, want) {
					t.Fatalf("filter reason = %q, want %q", reason, want)
				}
			}
		})
	}
}

func TestBuildCreateActivityRequestRejectsMissingSKUActivityPrice(t *testing.T) {
	goods := []marketing.PromotionGoodsData{{
		Skc:              "sg-missing-sku-price",
		InventoryNum:     100,
		USSupplyPrice:    100,
		MaxUSSupplyPrice: 100,
		SkuInfoList: []marketing.PromotionSkuInfo{
			{Sku: "sku-priced", USSupplyPrice: promotionTestFloat64Ptr(100)},
			{Sku: "sku-missing", USSupplyPrice: promotionTestFloat64Ptr(200)},
		},
	}}
	calcResp := &marketing.CalculateSupplyPriceResponse{Info: []marketing.SkcCalculationResult{{
		SkcName: "sg-missing-sku-price",
		SkuInfoList: []marketing.SkuCalculationInfo{
			{SkuCode: "sku-priced", PriceInfo: marketing.PriceInfo{ProductAmount: 100, PromotionAmount: 20}},
		},
	}}}

	req, _, reasons := buildCreateActivityRequest(
		activityRuleInputs{}, marketing.CreateActivityRequest{}, []int{2},
		goods,
		nil,
		calcResp, logrus.NewEntry(logrus.New()),
	)

	if got := len(req.AddCostAndStockInfoList); got != 0 {
		t.Fatalf("created goods count = %d, want missing-price SKC excluded", got)
	}
	reason := reasons["sg-missing-sku-price"]
	for _, want := range []string{"sku-missing", "活动价 0.00", "原价 0.00"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("filter reason = %q, want %q", reason, want)
		}
	}
}

func TestBuildCalculateRequestForPromotionProductsUsesDirectSKUCosts(t *testing.T) {
	config := activityRuleInputs{
		PriceMode:            "PROFIT",
		MinProfitRate:        0,
		FixedPriceAdjustment: 0,
	}

	req := buildCalculateRequestForPromotionProducts(
		config, marketing.CalculateSupplyPriceRequest{Currency: "USD"},
		[]marketing.PromotionGoodsData{{
			Skc:           "sg-multi-sku-profit",
			USSupplyPrice: 25.16,
			SkuInfoList: []marketing.PromotionSkuInfo{
				{Sku: "sku-small", USSupplyPrice: promotionFloat64Ptr(25.16)},
				{Sku: "sku-medium", USSupplyPrice: promotionFloat64Ptr(35.34)},
				{Sku: "sku-large", USSupplyPrice: promotionFloat64Ptr(39.06)},
			},
		}},
		[]marketing.SkcInfo{{
			Skc: "sg-multi-sku-profit",
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-small", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 25.16, IsAvailable: true}}},
				{SkuCode: "sku-medium", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 35.34, IsAvailable: true}}},
				{SkuCode: "sku-large", SitePriceInfoList: []marketing.SitePriceInfo{{Currency: "USD", SalePrice: 39.06, IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "sku-small", CostPrice: 20.88, Currency: "USD"},
				{SkuCode: "sku-medium", CostPrice: 29.33, Currency: "USD"},
				{SkuCode: "sku-large", CostPrice: 32.42, Currency: "USD"},
			},
		}},
	)

	if req == nil || len(req.SkcInfoList) != 1 || len(req.SkcInfoList[0].SkuInfoList) != 3 {
		t.Fatalf("calculate request = %+v, want three SKU prices", req)
	}
	assertClose(t, req.SkcInfoList[0].SkuInfoList[0].DiscountValue, 20.88)
	assertClose(t, req.SkcInfoList[0].SkuInfoList[1].DiscountValue, 29.33)
	assertClose(t, req.SkcInfoList[0].SkuInfoList[2].DiscountValue, 32.42)
}

func assertClose(t *testing.T, got float64, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("value = %.4f, want %.4f", got, want)
	}
}

func promotionTestFloat64Ptr(value float64) *float64 {
	return &value
}

func promotionFloat64Ptr(value float64) *float64 { return &value }

// Fixtures and expected rates are retained from the former registration tests;
// only the platform calls, paging, opening and session assertions retire.
func TestPromotionConfigRulesFromSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name                                          string
		strategy                                      *listingruntime.OperationStrategy
		products                                      []marketing.SkcInfo
		wantRegular, wantLimited, wantStock, wantRows int
		wantTypes                                     []int
	}{
		{name: "UsesProvidedPriceAndStockWhenAttributesAreMissing", strategy: &listingruntime.OperationStrategy{
			StoreID:               870,
			ActivityPriceMode:     "PROFIT",
			ActivityMinProfitRate: 0.15,
			ActivityStockRatio:    0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg260603174864291057873",
			Stock:               999,
			SupplyPrice:         12,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   48,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 499, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "AllowsZeroMinProfitRateWithProvidedProductSnapshot", strategy: &listingruntime.OperationStrategy{
			StoreID:               870,
			ActivityPriceMode:     "PROFIT",
			ActivityMinProfitRate: 0,
			ActivityStockRatio:    0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-zero-profit",
			Stock:               10,
			SupplyPrice:         20,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   20,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 1, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "UsesBreakevenDropRateWithProvidedProductSnapshot", strategy: &listingruntime.OperationStrategy{
			StoreID:            870,
			ActivityPriceMode:  "BREAKEVEN",
			ActivityStockRatio: 0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-breakeven",
			Stock:               10,
			SupplyPrice:         18,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   30,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 40, wantLimited: 0, wantStock: 5, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "AcceptsPromotionMultiSkuDifferentPrices", strategy: &listingruntime.OperationStrategy{
			StoreID:            870,
			ActivityPriceMode:  "BREAKEVEN",
			ActivityStockRatio: 0.5,
		}, products: []marketing.SkcInfo{marketing.SkcInfo{
			Skc:         "sg-multi-price",
			Stock:       100,
			SupplyPrice: 30,
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   30,
				Currency:    "USD",
				IsAvailable: true,
			}},
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{
					SkuCode: "sku-small",
					SitePriceInfoList: []marketing.SitePriceInfo{{
						SalePrice:   29.9,
						Currency:    "USD",
						IsAvailable: true,
					}},
				},
				{
					SkuCode: "sku-large",
					SitePriceInfoList: []marketing.SitePriceInfo{{
						SalePrice:   34.9,
						Currency:    "USD",
						IsAvailable: true,
					}},
				},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "sku-small", CostPrice: 12.5, Currency: "USD"},
				{SkuCode: "sku-large", CostPrice: 20.5, Currency: "USD"},
			},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "AllowsPromotionMultiSkuSamePrices", strategy: &listingruntime.OperationStrategy{
			StoreID:               870,
			ActivityPriceMode:     "PROFIT",
			ActivityMinProfitRate: 0.1,
			ActivityStockRatio:    0.5,
		}, products: []marketing.SkcInfo{{
			Skc:         "sg-multi-same-price",
			Stock:       100,
			SupplyPrice: 30,
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   30,
				Currency:    "USD",
				IsAvailable: true,
			}},
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{
					SkuCode: "sku-small",
					SitePriceInfoList: []marketing.SitePriceInfo{{
						SalePrice:   29.9,
						Currency:    "USD",
						IsAvailable: true,
					}},
				},
				{
					SkuCode: "sku-large",
					SitePriceInfoList: []marketing.SitePriceInfo{{
						SalePrice:   29.9,
						Currency:    "USD",
						IsAvailable: true,
					}},
				},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "sku-small", CostPrice: 18, Currency: "USD"},
				{SkuCode: "sku-large", CostPrice: 18, Currency: "USD"},
			},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "KeepsLimitedBreakevenDropRateGreaterThanRegular", strategy: &listingruntime.OperationStrategy{
			StoreID:             870,
			ActivityPriceMode:   "BREAKEVEN",
			ActivityPartakeType: "BOTH",
			ActivityStockRatio:  0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-breakeven-both",
			Stock:               100,
			SupplyPrice:         18,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   30,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 39, wantLimited: 40, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "AllowsRegularPartakeWithoutStockRatio", strategy: &listingruntime.OperationStrategy{
			StoreID:               870,
			ActivityPriceMode:     "PROFIT",
			ActivityPartakeType:   "REGULAR",
			ActivityMinProfitRate: 0,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-regular-no-stock-ratio",
			Stock:               10,
			SupplyPrice:         20,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   20,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "UsesLimitedPartakeStrategy", strategy: &listingruntime.OperationStrategy{
			StoreID:               870,
			ActivityPriceMode:     "PROFIT",
			ActivityPartakeType:   "LIMITED",
			ActivityMinProfitRate: 0,
			ActivityStockRatio:    0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-enable-limited",
			Stock:               10,
			SupplyPrice:         20,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   20,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesBothPartakeStrategy", strategy: &listingruntime.OperationStrategy{
			StoreID:                      870,
			ActivityPriceMode:            "PROFIT",
			ActivityPartakeType:          "BOTH",
			ActivityMinProfitRate:        0.1,
			ActivityLimitedMinProfitRate: 0,
			ActivityStockRatio:           0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-enable-both",
			Stock:               10,
			SupplyPrice:         10,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   20,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesLimitedDiscountForBothPartake", strategy: &listingruntime.OperationStrategy{
			StoreID:                     870,
			ActivityPriceMode:           "DISCOUNT",
			ActivityPartakeType:         "BOTH",
			ActivityDiscountRate:        0.2,
			ActivityLimitedDiscountRate: 0.25,
			ActivityStockRatio:          0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-both-discount",
			Stock:               10,
			SupplyPrice:         10,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   20,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 20, wantLimited: 25, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesLimitedMinProfitForBothPartake", strategy: &listingruntime.OperationStrategy{
			StoreID:                      870,
			ActivityPriceMode:            "PROFIT",
			ActivityPartakeType:          "BOTH",
			ActivityMinProfitRate:        0.2,
			ActivityLimitedMinProfitRate: 0.1,
			ActivityStockRatio:           0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-both-profit",
			Stock:               10,
			SupplyPrice:         10,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   20,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 37, wantLimited: 44, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "KeepsLimitedProfitDropRateGreaterAfterRounding", strategy: &listingruntime.OperationStrategy{
			StoreID:                      870,
			ActivityPriceMode:            "PROFIT",
			ActivityPartakeType:          "BOTH",
			ActivityMinProfitRate:        0.01,
			ActivityLimitedMinProfitRate: 0,
			ActivityStockRatio:           0.5,
		}, products: []marketing.SkcInfo{{
			Skc:                 "sg-both-profit-rounded",
			Stock:               999,
			SupplyPrice:         16.88,
			SupplyPriceCurrency: "USD",
			SitePriceInfoList: []marketing.SitePriceInfo{{
				SalePrice:   30.08,
				Currency:    "USD",
				IsAvailable: true,
			}},
		}}, wantRegular: 43, wantLimited: 44, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesLowestSKUPriceAndHighestCostForBoth", strategy: &listingruntime.OperationStrategy{
			StoreID: 177, ActivityPriceMode: "BREAKEVEN", ActivityPartakeType: "BOTH", ActivityStockRatio: 0.5,
		}, products: []marketing.SkcInfo{marketing.SkcInfo{
			Skc:   "sh260625180761728097751",
			Stock: 10989,
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-min", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 40, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-mid", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 80, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-high", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 100, Currency: "USD", IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "SKU-MIN", CostPrice: 10, Currency: "USD"},
				{SkuCode: "SKU-MID", CostPrice: 15, Currency: "USD"},
				{SkuCode: "SKU-HIGH", CostPrice: 30, Currency: "USD"},
			},
		}}, wantRegular: 24, wantLimited: 25, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesSupplyPriceForBreakevenDropRate", strategy: &listingruntime.OperationStrategy{
			StoreID: 1043, ActivityPriceMode: "BREAKEVEN", ActivityPartakeType: "BOTH", ActivityStockRatio: 0.5,
		}, products: []marketing.SkcInfo{marketing.SkcInfo{
			Skc: "sz260708164727639531767", Stock: 7992, SupplyPrice: 82.16,
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-one", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 96.98, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-two", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 96.98, Currency: "USD", IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "sku-one", CostPrice: 26.12, Currency: "USD"},
				{SkuCode: "sku-two", CostPrice: 26.12, Currency: "USD"},
			},
		}}, wantRegular: 67, wantLimited: 68, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesSavedSKUPricesForCandidate", strategy: &listingruntime.OperationStrategy{
			StoreID: 687, ActivityPriceMode: "BREAKEVEN", ActivityPartakeType: "BOTH", ActivityStockRatio: 0.5,
		}, products: []marketing.SkcInfo{marketing.SkcInfo{
			Skc: "sq-live-prices", Stock: 100,
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-one", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 74.8, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-two", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 59.2, Currency: "USD", IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "SKU-ONE", CostPrice: 10, Currency: "USD"},
				{SkuCode: "SKU-TWO", CostPrice: 5, Currency: "USD"},
			},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular, marketing.AutoPartakeActivityTypeLimited}},
		{name: "UsesSavedSKUPricesWhenRealtimePricesAreMissing", strategy: &listingruntime.OperationStrategy{
			StoreID: 1043, ActivityPriceMode: "BREAKEVEN", ActivityPartakeType: "REGULAR",
		}, products: []marketing.SkcInfo{{
			Skc: "sq-common-live-price", Stock: 100,
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-one", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 96.98, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-two", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 96.98, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-three", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 96.98, Currency: "USD", IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "sku-one", CostPrice: 5, Currency: "USD"},
				{SkuCode: "sku-two", CostPrice: 10, Currency: "USD"},
				{SkuCode: "sku-three", CostPrice: 15, Currency: "USD"},
			},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "DoesNotFilterCandidateForMissingRealtimePrices", strategy: &listingruntime.OperationStrategy{
			StoreID: 687, ActivityPriceMode: "BREAKEVEN", ActivityPartakeType: "REGULAR",
		}, products: []marketing.SkcInfo{{
			Skc: "sq-live-prices-incomplete", Stock: 100,
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-one", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 74.8, Currency: "USD", IsAvailable: true}}},
				{SkuCode: "sku-two", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 59.2, Currency: "USD", IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
				{SkuCode: "sku-one", CostPrice: 10, Currency: "USD"},
				{SkuCode: "sku-two", CostPrice: 5, Currency: "USD"},
			},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 1, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
		{name: "RejectsPartialMultiSKUPrices", strategy: &listingruntime.OperationStrategy{
			StoreID: 177, ActivityPriceMode: "BREAKEVEN", ActivityPartakeType: "REGULAR", ActivityStockRatio: 0.5,
		}, products: []marketing.SkcInfo{{
			Skc: "skc-partial", Stock: 10,
			SkuPriceInfoList: []marketing.SkuSitePriceInfo{
				{SkuCode: "sku-one", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 30, IsAvailable: true}}},
				{SkuCode: "sku-two", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 40, IsAvailable: true}}},
			},
			SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{{SkuCode: "sku-one", CostPrice: 18}},
		}}, wantRegular: 0, wantLimited: 0, wantStock: 0, wantRows: 0, wantTypes: []int{marketing.AutoPartakeActivityTypeRegular}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logger := logrus.NewEntry(logrus.New())
			mode := autoPartakePriceModeFromStrategy(tc.strategy)
			if err := validateAutoPartakeDiscountsForStrategy(tc.strategy); err != nil {
				t.Fatal(err)
			}
			if got := autoPartakeActivityTypesFromStrategy(tc.strategy); !reflect.DeepEqual(got, tc.wantTypes) {
				t.Fatalf("types %v, want %v", got, tc.wantTypes)
			}
			configs := buildActivityConfigsFromProvidedProducts(tc.products, tc.strategy, mode, logger)
			if len(configs) != tc.wantRows {
				t.Fatalf("configs %+v, want %d rows", configs, tc.wantRows)
			}
			if tc.wantRows == 0 {
				return
			}
			if configs[0].DropRate <= 0 || configs[0].DropRate > 80 {
				t.Fatalf("invalid rate %+v", configs[0])
			}
			if tc.wantStock > 0 && (configs[0].ActStock != tc.wantStock || configs[0].ReservedActStock != tc.products[0].Stock) {
				t.Fatalf("stock %+v, want %d / %d", configs[0], tc.wantStock, tc.products[0].Stock)
			}
			regular := promotionConfigListForActivityType(tc.products, configs, marketing.AutoPartakeActivityTypeRegular, tc.strategy, mode, logger)
			limited := promotionConfigListForActivityType(tc.products, configs, marketing.AutoPartakeActivityTypeLimited, tc.strategy, mode, logger)
			if tc.wantRegular > 0 && regular[0].DropRate != tc.wantRegular {
				t.Fatalf("regular %+v, want %d", regular, tc.wantRegular)
			}
			if tc.wantLimited > 0 && limited[0].DropRate != tc.wantLimited {
				t.Fatalf("limited %+v, want %d", limited, tc.wantLimited)
			}
			if len(tc.wantTypes) == 2 && (limited[0].DropRate <= regular[0].DropRate || limited[0].DropRate > 80) {
				t.Fatalf("regular/limited rates %+v / %+v", regular, limited)
			}
		})
	}
}

func TestApplyPromotionCreateConfigKeepsFullStockRatioUnlimited(t *testing.T) {
	cfg := activityRuleInputs{StockPercent: 100}
	applyPromotionCreateConfig(&cfg, &listingruntime.OperationStrategy{ActivityStockRatio: 1})
	if cfg.StockLimit {
		t.Fatal("full stock ratio must remain unlimited")
	}
}

func TestPromotionRequestProjectionPreservesMetadataAndStockRules(t *testing.T) {
	products := []marketing.SkcInfo{{
		Skc: "metadata-skc", Stock: 100, SupplyPrice: 30,
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{{SkuCode: "sku-1", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 30, Currency: "USD", IsAvailable: true}}}},
	}}
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	goods[0].CheckStock = &marketing.CheckStock{MinStock: 15, MaxStock: 40}
	metadata := marketing.CalculateSupplyPriceRequest{
		Currency: "USD", RefToolID: 30, SceneID: 1, TimeZone: "America/Los_Angeles",
		ZoneStartTime: "2026-10-17 10:00:00", ZoneEndTime: "2026-10-31 10:00:00",
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.2}
	calculated := buildCalculateRequestForPromotionProducts(inputs, metadata, goods, products)
	copyWithoutPrices := *calculated
	copyWithoutPrices.SkcInfoList = nil
	if !reflect.DeepEqual(copyWithoutPrices, metadata) || metadata.SkcInfoList != nil {
		t.Fatalf("calculate metadata changed: %+v / %+v", copyWithoutPrices, metadata)
	}
	response := &marketing.CalculateSupplyPriceResponse{Info: []marketing.SkcCalculationResult{{SkcName: "metadata-skc", SkuInfoList: []marketing.SkuCalculationInfo{{SkuCode: "sku-1", PriceInfo: marketing.PriceInfo{ProductAmount: 30, PromotionAmount: 6}}}}}}
	createMetadata := marketing.CreateActivityRequest{ActivityBaseInfoRequest: marketing.ActivityBaseInfo{
		ActName: "provided-name", TimeZone: metadata.TimeZone, ZoneStartTime: metadata.ZoneStartTime,
		ZoneEndTime: metadata.ZoneEndTime, RefToolID: 30, SubTypeID: 2, NotifyFlag: 1,
	}, PricingType: 2}
	for _, tc := range []struct {
		name     string
		inputs   activityRuleInputs
		wantRows int
	}{
		{name: "full stock respects platform maximum", inputs: inputs, wantRows: 1},
		{name: "limited stock below platform minimum", inputs: activityRuleInputs{StockLimit: true, StockPercent: 10}},
		{name: "SKC whitelist", inputs: activityRuleInputs{FilterSkcList: []string{"another-skc"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, _, _ := buildCreateActivityRequest(tc.inputs, createMetadata, []int{2}, goods, calculated, response, logrus.NewEntry(logrus.New()))
			if req.ActivityBaseInfoRequest != createMetadata.ActivityBaseInfoRequest || req.PricingType != createMetadata.PricingType || createMetadata.AddCostAndStockInfoList != nil {
				t.Fatalf("create metadata changed: %+v", req)
			}
			if len(req.AddCostAndStockInfoList) != tc.wantRows {
				t.Fatalf("stock/whitelist projection = %+v, want %d rows", req, tc.wantRows)
			}
			if tc.wantRows > 0 {
				row := req.AddCostAndStockInfoList[0]
				if row.AttendNum != 40 || row.StockNum != 40 || !reflect.DeepEqual(row.CenterList, []int{2}) {
					t.Fatalf("stock/centers changed: %+v", row)
				}
			}
		})
	}
}

func TestExtractedPromotionRiskRules(t *testing.T) {
	for _, tc := range []struct {
		name     string
		risk     int
		warning  float64
		allow    bool
		wantRisk bool
	}{
		{name: "risk blocked", risk: 1, wantRisk: true},
		{name: "risk allowed", risk: 1, allow: true},
		{name: "warning still blocked", risk: 1, allow: true, warning: 1, wantRisk: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &marketing.CalculateSupplyPriceResponse{
				Info: []marketing.SkcCalculationResult{
					{SkuInfoList: []marketing.SkuCalculationInfo{{RiskTag: tc.risk, WarningValue: tc.warning}}},
				},
			}
			err := validatePriceRisk(response, activityRuleInputs{AllowRiskProducts: tc.allow}, logrus.NewEntry(logrus.New()))
			if (err == ErrProductPriceRisk) != tc.wantRisk {
				t.Fatalf("risk result %v, want risk %v", err, tc.wantRisk)
			}
		})
	}
}

func TestPromotionRequestRulesAcceptsPromotionMultiSkuDifferentPrices(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:            870,
		ActivityPriceMode:  "BREAKEVEN",
		ActivityStockRatio: 0.5,
	}
	products := []marketing.SkcInfo{marketing.SkcInfo{
		Skc:         "sg-multi-price",
		Stock:       100,
		SupplyPrice: 30,
		SitePriceInfoList: []marketing.SitePriceInfo{{
			SalePrice:   30,
			Currency:    "USD",
			IsAvailable: true,
		}},
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{
			{
				SkuCode: "sku-small",
				SitePriceInfoList: []marketing.SitePriceInfo{{
					SalePrice:   29.9,
					Currency:    "USD",
					IsAvailable: true,
				}},
			},
			{
				SkuCode: "sku-large",
				SitePriceInfoList: []marketing.SitePriceInfo{{
					SalePrice:   34.9,
					Currency:    "USD",
					IsAvailable: true,
				}},
			},
		},
		SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
			{SkuCode: "sku-small", CostPrice: 12.5, Currency: "USD"},
			{SkuCode: "sku-large", CostPrice: 20.5, Currency: "USD"},
		},
	}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{{
			SkcName: products[0].Skc,
			SkuInfoList: []marketing.SkuCalculationInfo{
				{SkuCode: "sku-small", PriceInfo: marketing.PriceInfo{ProductAmount: 29.9, PromotionAmount: 12.5}},
				{SkuCode: "sku-large", PriceInfo: marketing.PriceInfo{ProductAmount: 34.9, PromotionAmount: 20.5}},
			},
		}},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if calculated == nil || len(calculated.SkcInfoList) != 1 {
		t.Fatalf("calculated request = %+v, want one SKU-priced product", calculated)
	}
	skuPrices := calculated.SkcInfoList[0].SkuInfoList
	if len(skuPrices) != 2 {
		t.Fatalf("calculated SKU prices = %+v, want two entries", skuPrices)
	}
	if skuPrices[0].SkuCode != "sku-small" || skuPrices[0].ProductPrice != 29.9 || skuPrices[0].DiscountValue != 12.5 {
		t.Fatalf("first calculated SKU price = %+v, want sku-small price 29.9 and cost 12.5", skuPrices[0])
	}
	if skuPrices[1].SkuCode != "sku-large" || skuPrices[1].ProductPrice != 34.9 || skuPrices[1].DiscountValue != 20.5 {
		t.Fatalf("second calculated SKU price = %+v, want sku-large price 34.9 and cost 20.5", skuPrices[1])
	}
	if request == nil || len(request.AddCostAndStockInfoList) != 1 {
		t.Fatalf("created request = %+v, want one product", request)
	}
	createdProduct := request.AddCostAndStockInfoList[0]
	if createdProduct.IsSaleAttribute != 1 {
		t.Fatalf("is_sale_attribute = %d, want 1 for a multi-SKU activity", createdProduct.IsSaleAttribute)
	}
	if createdProduct.CostPrice != 0 || createdProduct.MaxProductActPrice != 0 || createdProduct.ProductActPrice != 0 {
		t.Fatalf("multi-SKU product-level prices = %+v, want zero so each add_sku_list price takes effect", createdProduct)
	}
	createdSKUs := createdProduct.AddSkuList
	if len(createdSKUs) != 2 {
		t.Fatalf("created SKU list = %+v, want two entries", createdSKUs)
	}
	if createdSKUs[0].Sku != "sku-small" || createdSKUs[0].CostPrice != 29.9 {
		t.Fatalf("first created SKU = %+v, want sku-small original 29.9 activity 12.5", createdSKUs[0])
	}
	assertClose(t, createdSKUs[0].ProductActPrice, 12.5)
	if createdSKUs[1].Sku != "sku-large" || createdSKUs[1].CostPrice != 34.9 {
		t.Fatalf("second created SKU = %+v, want sku-large original 34.9 activity 20.5", createdSKUs[1])
	}
	assertClose(t, createdSKUs[1].ProductActPrice, 20.5)
}

func TestPromotionRequestRulesUsesSkuPricesForSaleAttributeGoods(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:              870,
		ActivityPriceMode:    "DISCOUNT",
		ActivityDiscountRate: 0.2,
		ActivityStockRatio:   0.5,
	}
	products := []marketing.SkcInfo{{Skc: "sg-sale-attribute", Stock: 100, SupplyPrice: 35.04, SkuPriceInfoList: []marketing.SkuSitePriceInfo{
		{SkuCode: "sku-small", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 31.68, Currency: "USD", IsAvailable: true}}},
		{SkuCode: "sku-large", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 35.04, Currency: "USD", IsAvailable: true}}},
	}}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{
			{
				SkcName: "sg-sale-attribute",
				SkuInfoList: []marketing.SkuCalculationInfo{
					{
						SkuCode: "sku-small",
						PriceInfo: marketing.PriceInfo{
							ProductAmount:   31.68,
							PromotionAmount: 15.69,
						},
					},
					{
						SkuCode: "sku-large",
						PriceInfo: marketing.PriceInfo{
							ProductAmount:   35.04,
							PromotionAmount: 16.05,
						},
					},
				},
			},
		},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if calculated == nil || len(calculated.SkcInfoList) != 1 || len(calculated.SkcInfoList[0].SkuInfoList) != 2 {
		t.Fatalf("calculated request = %+v, want two sale-attribute SKU prices", calculated)
	}
	if got := calculated.SkcInfoList[0].SkuInfoList[0].ProductPrice; got != 31.68 {
		t.Fatalf("first SKU calculate product price = %.2f, want SKU price 31.68", got)
	}
	if got := calculated.SkcInfoList[0].SkuInfoList[1].ProductPrice; got != 35.04 {
		t.Fatalf("second SKU calculate product price = %.2f, want SKU price 35.04", got)
	}

	if request == nil || len(request.AddCostAndStockInfoList) != 1 {
		t.Fatalf("created request = %+v, want one sale-attribute SKC", request)
	}
	created := request.AddCostAndStockInfoList[0]
	if created.IsSaleAttribute != 1 || len(created.AddSkuList) != 2 {
		t.Fatalf("created sale attribute item = %+v, want two SKU rows", created)
	}
	if created.CostPrice != 0 || created.MaxProductActPrice != 0 || created.ProductActPrice != 0 {
		t.Fatalf("multi-SKU product-level prices = %+v, want zero", created)
	}
	if got := created.AddSkuList[0].CostPrice; got != 31.68 {
		t.Fatalf("first SKU create cost price = %.2f, want 31.68", got)
	}
	if got := created.AddSkuList[0].MaxProductActPrice; got != 31.68 {
		t.Fatalf("first SKU create max activity price = %.2f, want 31.68", got)
	}
	if got := created.AddSkuList[0].ProductActPrice; got != 15.99 {
		t.Fatalf("first SKU create activity price = %.2f, want 15.99", got)
	}
	if got := created.AddSkuList[1].CostPrice; got != 35.04 {
		t.Fatalf("second SKU create cost price = %.2f, want 35.04", got)
	}
	if got := created.AddSkuList[1].MaxProductActPrice; got != 35.04 {
		t.Fatalf("second SKU create max activity price = %.2f, want 35.04", got)
	}
	if got := created.AddSkuList[1].ProductActPrice; got != 18.99 {
		t.Fatalf("second SKU create activity price = %.2f, want 18.99", got)
	}
}

func TestPromotionRequestRulesUsesProvidedSkuPricesOverPromotionGoodsSkuPrices(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:              870,
		ActivityPriceMode:    "DISCOUNT",
		ActivityDiscountRate: 0.2,
		ActivityStockRatio:   0.5,
	}
	products := []marketing.SkcInfo{{
		Skc: "sg-snapshot-prices", Stock: 100, SupplyPrice: 35.04,
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{
			{
				SkuCode: "sku-small",
				SitePriceInfoList: []marketing.SitePriceInfo{{
					SalePrice:   31.68,
					Currency:    "USD",
					IsAvailable: true,
				}},
			},
			{
				SkuCode: "sku-large",
				SitePriceInfoList: []marketing.SitePriceInfo{{
					SalePrice:   35.04,
					Currency:    "USD",
					IsAvailable: true,
				}},
			},
		},
	}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{
			{
				SkcName: "sg-snapshot-prices",
				SkuInfoList: []marketing.SkuCalculationInfo{
					{
						SkuCode: "sku-small",
						PriceInfo: marketing.PriceInfo{
							ProductAmount:   31.68,
							PromotionAmount: 6.34,
						},
					},
					{
						SkuCode: "sku-large",
						PriceInfo: marketing.PriceInfo{
							ProductAmount:   35.04,
							PromotionAmount: 7.01,
						},
					},
				},
			},
		},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if calculated == nil || len(calculated.SkcInfoList) != 1 || len(calculated.SkcInfoList[0].SkuInfoList) != 2 {
		t.Fatalf("calculated request = %+v, want two SKU prices from provided snapshot", calculated)
	}
	if got := calculated.SkcInfoList[0].SkuInfoList[0].ProductPrice; got != 31.68 {
		t.Fatalf("first SKU calculate product price = %.2f, want snapshot price 31.68", got)
	}
	if got := calculated.SkcInfoList[0].SkuInfoList[1].ProductPrice; got != 35.04 {
		t.Fatalf("second SKU calculate product price = %.2f, want snapshot price 35.04", got)
	}
	if request == nil || len(request.AddCostAndStockInfoList) != 1 || len(request.AddCostAndStockInfoList[0].AddSkuList) != 2 {
		t.Fatalf("created request = %+v, want two SKU rows from snapshot prices", request)
	}
	if got := request.AddCostAndStockInfoList[0].AddSkuList[0].CostPrice; got != 31.68 {
		t.Fatalf("first SKU create cost price = %.2f, want snapshot price 31.68", got)
	}
	if got := request.AddCostAndStockInfoList[0].AddSkuList[1].CostPrice; got != 35.04 {
		t.Fatalf("second SKU create cost price = %.2f, want snapshot price 35.04", got)
	}
}

func TestPromotionRequestRulesUsesCandidateSalePriceOverPromotionGoodsPrice(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:              870,
		ActivityPriceMode:    "DISCOUNT",
		ActivityDiscountRate: 0.2,
		ActivityStockRatio:   0.5,
	}
	products := []marketing.SkcInfo{{
		Skc: "sg-missing-price", Stock: 100, SupplyPrice: 53.95,
		SitePriceInfoList: []marketing.SitePriceInfo{{
			SalePrice:   53.95,
			Currency:    "USD",
			IsAvailable: true,
		}},
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{{SkuCode: "sku-missing-price-1", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 53.95, Currency: "USD", IsAvailable: true}}}},
	}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{
			{
				SkcName: "sg-missing-price",
				SkuInfoList: []marketing.SkuCalculationInfo{
					{
						SkuCode: "sku-missing-price-1",
						PriceInfo: marketing.PriceInfo{
							ProductAmount:   53.95,
							PromotionAmount: 10.79,
						},
					},
				},
			},
		},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if calculated == nil || len(calculated.SkcInfoList) != 1 {
		t.Fatalf("calculated request = %+v, want one SKC from candidate sale price", calculated)
	}
	sku := calculated.SkcInfoList[0].SkuInfoList[0]
	assertClose(t, sku.ProductPrice, 53.95)
	assertClose(t, sku.DiscountValue, 43.16)
	if sku.ProductPrice == 62 {
		t.Fatalf("sku price = %+v, still using promotion goods price", sku)
	}
	if request == nil || len(request.AddCostAndStockInfoList) != 1 {
		t.Fatalf("created request = %+v, want one created SKC", request)
	}
	created := request.AddCostAndStockInfoList[0]
	assertClose(t, created.ProductActPrice, 43.16)
	assertClose(t, created.CostPrice, 53.95)
	assertClose(t, created.MaxProductActPrice, 53.95)
}

func TestPromotionRequestRulesUsesRequestedProfitModeSkuActivityPrices(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:               870,
		ActivityPriceMode:     "PROFIT",
		ActivityMinProfitRate: 0,
	}
	products := []marketing.SkcInfo{{
		Skc: "sg-profit-sku-prices", Stock: 999, SupplyPrice: 10.88,
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{{SkuCode: "sku-small", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 29.7, Currency: "USD", IsAvailable: true}}}, {SkuCode: "sku-large", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 31.68, Currency: "USD", IsAvailable: true}}}},
		SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
			{SkuCode: "sku-small", CostPrice: 10.88, Currency: "USD"},
			{SkuCode: "sku-large", CostPrice: 11.6053, Currency: "USD"},
		},
	}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{{
			SkcName: "sg-profit-sku-prices",
			SkuInfoList: []marketing.SkuCalculationInfo{
				{
					SkuCode: "sku-small",
					PriceInfo: marketing.PriceInfo{
						ProductAmount:   29.7,
						PromotionAmount: 18.82,
					},
				},
				{
					SkuCode: "sku-large",
					PriceInfo: marketing.PriceInfo{
						ProductAmount:   31.68,
						PromotionAmount: 20.8,
					},
				},
			},
		}},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if request == nil || len(request.AddCostAndStockInfoList) != 1 || len(request.AddCostAndStockInfoList[0].AddSkuList) != 2 {
		t.Fatalf("created request = %+v, want two SKU rows", request)
	}
	wantSmall := calculated.SkcInfoList[0].SkuInfoList[0].DiscountValue
	wantLarge := calculated.SkcInfoList[0].SkuInfoList[1].DiscountValue
	assertClose(t, wantSmall, 10.88)
	assertClose(t, wantLarge, 11.6053)
	assertClose(t, request.AddCostAndStockInfoList[0].AddSkuList[0].ProductActPrice, wantSmall)
	assertClose(t, request.AddCostAndStockInfoList[0].AddSkuList[1].ProductActPrice, wantLarge)
}

func TestPromotionRequestRulesUsesSKUCostsForProfitModeSkuActivityPrices(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:               870,
		ActivityPriceMode:     "PROFIT",
		ActivityMinProfitRate: 0,
	}
	products := []marketing.SkcInfo{{
		Skc: "sg-profit-sku-costs", Stock: 999, SupplyPrice: 10.88,
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{{SkuCode: "sku-small", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 29.7, Currency: "USD", IsAvailable: true}}}, {SkuCode: "sku-large", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 31.68, Currency: "USD", IsAvailable: true}}}},
		SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
			{SkuCode: "sku-small", CostPrice: 9.88, Currency: "USD"},
			{SkuCode: "sku-large", CostPrice: 10.88, Currency: "USD"},
		},
	}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{{
			SkcName: "sg-profit-sku-costs",
			SkuInfoList: []marketing.SkuCalculationInfo{
				{
					SkuCode: "sku-small",
					PriceInfo: marketing.PriceInfo{
						ProductAmount:   29.7,
						PromotionAmount: 19.82,
					},
				},
				{
					SkuCode: "sku-large",
					PriceInfo: marketing.PriceInfo{
						ProductAmount:   31.68,
						PromotionAmount: 20.8,
					},
				},
			},
		}},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if request == nil || len(request.AddCostAndStockInfoList) != 1 || len(request.AddCostAndStockInfoList[0].AddSkuList) != 2 {
		t.Fatalf("created request = %+v, want two SKU rows", request)
	}
	assertClose(t, calculated.SkcInfoList[0].SkuInfoList[0].DiscountValue, 9.88)
	assertClose(t, calculated.SkcInfoList[0].SkuInfoList[1].DiscountValue, 10.88)
	assertClose(t, request.AddCostAndStockInfoList[0].AddSkuList[0].ProductActPrice, 9.88)
	assertClose(t, request.AddCostAndStockInfoList[0].AddSkuList[1].ProductActPrice, 10.88)
}

func TestPromotionRequestRulesUsesSKUCostsForBreakevenModeSkuActivityPrices(t *testing.T) {
	strategy := &listingruntime.OperationStrategy{
		StoreID:           870,
		ActivityPriceMode: "BREAKEVEN",
	}
	products := []marketing.SkcInfo{{
		Skc: "sg-breakeven-sku-costs", Stock: 999, SupplyPrice: 10.88,
		SkuPriceInfoList: []marketing.SkuSitePriceInfo{{SkuCode: "sku-small", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 29.7, Currency: "USD", IsAvailable: true}}}, {SkuCode: "sku-large", SitePriceInfoList: []marketing.SitePriceInfo{{SalePrice: 31.68, Currency: "USD", IsAvailable: true}}}},
		SkuCostPriceInfoList: []marketing.SkuCostPriceInfo{
			{SkuCode: "sku-small", CostPrice: 9.88, Currency: "USD"},
			{SkuCode: "sku-large", CostPrice: 10.88, Currency: "USD"},
		},
	}}
	calcResponse := &marketing.CalculateSupplyPriceResponse{
		Code: "0",
		Msg:  "ok",
		Info: []marketing.SkcCalculationResult{{
			SkcName: "sg-breakeven-sku-costs",
			SkuInfoList: []marketing.SkuCalculationInfo{
				{
					SkuCode: "sku-small",
					PriceInfo: marketing.PriceInfo{
						ProductAmount:   29.7,
						PromotionAmount: 1.82,
					},
				},
				{
					SkuCode: "sku-large",
					PriceInfo: marketing.PriceInfo{
						ProductAmount:   31.68,
						PromotionAmount: 2.8,
					},
				},
			},
		}},
	}
	inputs := activityRuleInputs{PriceMode: "DISCOUNT", DiscountRate: 0.4, MinProfitRate: 0.15, StockPercent: 100, FixedPriceAdjustment: strategy.FixedPriceAdjustment}
	applyPromotionCreateConfig(&inputs, strategy)
	goods := promotionGoodsFromProductSnapshots(products, "USD")
	calculated := buildCalculateRequestForPromotionProducts(inputs, marketing.CalculateSupplyPriceRequest{Currency: "USD"}, goods, products)
	request, _, _ := buildCreateActivityRequest(inputs, marketing.CreateActivityRequest{}, []int{2}, goods, calculated, calcResponse, logrus.NewEntry(logrus.New()))

	if request == nil || len(request.AddCostAndStockInfoList) != 1 || len(request.AddCostAndStockInfoList[0].AddSkuList) != 2 {
		t.Fatalf("created request = %+v, want two SKU rows", request)
	}
	assertClose(t, calculated.SkcInfoList[0].SkuInfoList[0].DiscountValue, 9.88)
	assertClose(t, calculated.SkcInfoList[0].SkuInfoList[1].DiscountValue, 10.88)
	assertClose(t, request.AddCostAndStockInfoList[0].AddSkuList[0].ProductActPrice, 9.88)
	assertClose(t, request.AddCostAndStockInfoList[0].AddSkuList[1].ProductActPrice, 10.88)
}
