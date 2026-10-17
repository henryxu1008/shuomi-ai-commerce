package sku

import (
	"fmt"
	"strings"

	"task-processor/internal/model"
	models "task-processor/internal/temu/api/product"
	temutemplate "task-processor/internal/temu/api/template"

	"github.com/sirupsen/logrus"
)

// SkuSpecHandler SKU规格处理器
type SkuSpecHandler struct {
	logger *logrus.Entry
}

// NewSkuSpecHandler 创建新的规格处理器
func NewSkuSpecHandler(logger *logrus.Entry) *SkuSpecHandler {
	return &SkuSpecHandler{
		logger: logger,
	}
}

// IsSizeSpec 判断是否为尺寸规格（实现 SpecHandlerInterface 接口）
func (sh *SkuSpecHandler) IsSizeSpec(specName string) bool {
	sizeKeywords := []string{
		"size", "尺寸", "尺码", "大小", "码数",
		"length", "width", "height", "长度", "宽度", "高度",
		"dimension", "dimensions", "规格",
	}

	specNameLower := strings.ToLower(specName)
	for _, keyword := range sizeKeywords {
		if strings.Contains(specNameLower, keyword) {
			return true
		}
	}
	return false
}

// convertUserInputSpecsToGoodsSpecProperties 转换用户输入规格为商品规格属性
func (sh *SkuSpecHandler) convertUserInputSpecsToGoodsSpecProperties(userInputSpecs []temutemplate.UserInputParentSpec) []temutemplate.TemplateRespGoodsSpecProperty {
	var specProperties []temutemplate.TemplateRespGoodsSpecProperty
	for i, userSpec := range userInputSpecs {
		specProperty := temutemplate.TemplateRespGoodsSpecProperty{
			PID:               i + 1000, // 使用临时ID
			TemplateModuleID:  0,
			TemplatePID:       0,
			RefPID:            0,
			Name:              userSpec.ParentSpecName,
			PropertyValueType: 1, // 假设为选择类型
			ValueUnit:         []string{},
			Values:            []temutemplate.PropertyValue{}, // 用户输入规格通常没有预定义值
			MaxValue:          "",
			MinValue:          "",
			ValuePrecision:    0,
			Required:          false,
			IsSale:            true,
			ParentSpecID:      userSpec.ParentSpecID,
			MainSale:          true,
			Feature:           0,
			ControlType:       1,
		}
		specProperties = append(specProperties, specProperty)
	}
	return specProperties
}

// ValidateSpecs 验证规格是否有效
func (sh *SkuSpecHandler) ValidateSpecs(specs []models.SpecInfo) error {
	if len(specs) == 0 {
		return fmt.Errorf("规格列表为空")
	}

	// TEMU规则：销售规格最多只能有2种
	if len(specs) > 2 {
		return fmt.Errorf("规格数量超过限制：当前有%d个规格，但TEMU最多只允许2个销售规格", len(specs))
	}

	for i, spec := range specs {
		if spec.ParentSpecID == "" {
			return fmt.Errorf("规格[%d]的ParentSpecID为空", i)
		}
		if spec.SpecID == "" {
			return fmt.Errorf("规格[%d]的SpecID为空", i)
		}
	}

	return nil
}

// CreateDefaultSpec 创建默认规格（当没有规格时使用）
// 注意：不应该使用默认规格，必须从模板中选择规格
// 这个方法现在返回空切片并记录错误
func (sh *SkuSpecHandler) CreateDefaultSpec(variant *model.Product) []models.SpecInfo {
	sh.logger.Error("❌ 尝试创建默认规格！这是不允许的，必须使用TEMU模板中的规格")
	sh.logger.Error("❌ 请检查AI映射是否正确生成了规格，或者模板规格是否正确配置")

	// 返回空切片，让调用方知道出错了
	return []models.SpecInfo{}
}
