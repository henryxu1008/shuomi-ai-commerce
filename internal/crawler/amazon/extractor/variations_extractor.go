package extractor

import (
	"github.com/sirupsen/logrus"

	amazonresult "task-processor/internal/crawler/amazon/capturedata"
	"task-processor/internal/crawler/amazon/variations"

	"github.com/mxschmitt/playwright-go"
)

// VariationsExtractor 变体信息提取器（向后兼容）
type VariationsExtractor struct {
	extractor *variations.Extractor
}

// NewVariationsExtractor 创建变体信息提取器实例
func NewVariationsExtractor(logs func(string) *logrus.Entry) *VariationsExtractor {
	extractor := variations.NewExtractor(logs)
	return &VariationsExtractor{
		extractor: extractor,
	}
}

// Extract 提取产品变体信息
func (ve *VariationsExtractor) Extract(page playwright.Page, product *amazonresult.Product) error {
	// 提取变体数据
	variationsData, err := ve.extractor.ExtractFromPage(page)
	if err != nil {
		return err
	}

	// 转换为 VariationValue 格式
	var variationsValues []amazonresult.VariationValue
	for variantName, values := range variationsData.VariationsValues {
		if len(values) > 0 {
			variationsValues = append(variationsValues, amazonresult.VariationValue{
				VariantName: variantName,
				Values:      values,
			})
		}
	}
	product.VariationsValues = variationsValues

	// 构建变体列表
	varList := ve.extractor.BuildVariations(
		convertToVariationsValues(variationsValues),
		variationsData.ASINMapping,
		variationsData.PriceMapping,
		product.FinalPrice,
		"USD",
	)

	// 转换回 amazon.Variation 类型
	product.Variations = convertToAmazonVariations(varList)

	return nil
}

// 辅助函数：转换类型
func convertToVariationsValues(values []amazonresult.VariationValue) []variations.VariationValue {
	result := make([]variations.VariationValue, len(values))
	for i, v := range values {
		result[i] = variations.VariationValue{
			VariantName: v.VariantName,
			Values:      v.Values,
		}
	}
	return result
}

func convertToAmazonVariations(varList []variations.Variation) []amazonresult.Variation {
	result := make([]amazonresult.Variation, len(varList))
	for i, v := range varList {
		result[i] = amazonresult.Variation{
			Name: v.Name,
			Asin: v.Asin,
			//Price:      v.Price,
			//Currency:   v.Currency,
			Attributes: v.Attributes,
		}
	}
	return result
}
