package extractor

import (
	"context"
	"fmt"

	"task-processor/internal/model"
	"task-processor/internal/pkg/goroutine"
	"time"

	"github.com/mxschmitt/playwright-go"
	"github.com/sirupsen/logrus"
)

// Extractor 提取器接口
type Extractor interface {
	Extract(page playwright.Page, product *model.Product) error
}

// CompositeExtractor 组合提取器
type CompositeExtractor struct {
	loggerDependency

	extractors    []Extractor
	errorDetector *ErrorDetector
	logger        *logrus.Entry
}

// NewCompositeExtractor 创建组合提取器
func NewCompositeExtractor(marketplace string, logs func(string) *logrus.Entry) *CompositeExtractor {
	return &CompositeExtractor{
		loggerDependency: loggerDependency{logFor: logs},
		extractors: []Extractor{
			&TitleExtractor{loggerDependency: loggerDependency{logFor: logs}},
			&AvailabilityExtractor{loggerDependency: loggerDependency{logFor: logs}}, // 先提取可用性，价格提取器需要依赖这个信息
			NewPriceExtractor(marketplace, logs),                                     // 使用构造函数正确初始化
			&BrandExtractor{},
			&RatingExtractor{loggerDependency: loggerDependency{logFor: logs}}, // 包含评分和评论数量提取
			&ImageExtractor{loggerDependency: loggerDependency{logFor: logs}},
			NewVideoExtractor(logs),       // 视频提取器
			&CategoriesExtractor{},        // 分类提取器
			NewParentAsinExtractor(logs),  // Parent ASIN提取器
			&SellerExtractor{},            // 卖家提取器
			&ShipsFromExtractor{},         // 发货地提取器
			&DeliveryExtractor{},          // 配送信息提取器
			NewDescriptionExtractor(logs), // 使用构造函数正确初始化
			NewSizeChartExtractor(),       // 尺码表提取器
			&ProductDetailsExtractor{},    // 产品详情提取器
			NewVariationsExtractor(logs),  // 变体提取器
			NewBestsellerExtractor(),      // 畅销排名提取器
			NewFeatureParserExtractor(),   // 特性解析提取器
			&FeaturesExtractor{},          // 基础特性提取器
		},
		errorDetector: NewErrorDetector(),
		logger:        componentLogger(logs, "CompositeExtractor"),
	}
}

// Extract 提取所有信息（使用ParallelProcessor优化）
func (ce *CompositeExtractor) Extract(page playwright.Page, product *model.Product) error {
	return ce.ExtractWithContext(context.Background(), page, product)
}

// ExtractWithContext 提取所有信息，支持 ctx 超时控制
func (ce *CompositeExtractor) ExtractWithContext(ctx context.Context, page playwright.Page, product *model.Product) error {
	// 第一阶段：必须串行执行的提取器（有依赖关系）
	serialExtractors := []Extractor{
		&TitleExtractor{loggerDependency: loggerDependency{logFor: ce.logFor}},
		&AvailabilityExtractor{loggerDependency: loggerDependency{logFor: ce.logFor}},
		ce.extractors[2], // PriceExtractor（依赖Availability）
	}

	for _, extractor := range serialExtractors {
		extractorName := getExtractorName(extractor)
		startedAt := time.Now()
		ce.componentLogger("crawler/amazon").Infof("提取器开始执行: %s", extractorName)
		if err := extractor.Extract(page, product); err != nil {
			ce.componentLogger("crawler/amazon").Infof("提取器执行失败 (%s): %v (耗时=%s)", extractorName, err, time.Since(startedAt).Round(time.Millisecond))
			if ce.errorDetector.IsCriticalError(err) {
				ce.componentLogger("crawler/amazon").Infof("检测到关键错误，停止后续提取器执行: %v", err)
				return err
			}
		} else {
			ce.componentLogger("crawler/amazon").Infof("提取器执行完成: %s (耗时=%s)", extractorName, time.Since(startedAt).Round(time.Millisecond))
		}
	}

	// 第二阶段：使用ParallelProcessor并行执行
	parallelExtractors := ce.extractors[3:] // 从BrandExtractor开始的所有提取器

	// 创建并行处理器（15个提取器，使用15个worker，每个提取器30秒超时）
	processor := goroutine.NewProcessor(len(parallelExtractors), 30*time.Second, ce.logger)

	// 创建任务
	tasks := make([]*goroutine.Task, len(parallelExtractors))
	for i, ext := range parallelExtractors {
		tasks[i] = &goroutine.Task{
			Index: i,
			ID:    getExtractorName(ext),
			Data:  ext,
		}
	}

	// 定义处理函数
	processFunc := func(ctx context.Context, task *goroutine.Task) (any, error) {
		extractor := task.Data.(Extractor)
		// startedAt := time.Now()
		// defer func() {
		// }()
		return nil, extractor.Extract(page, product)
	}

	// 并行执行，传入外层 ctx 确保超时时能及时取消
	results := processor.ProcessParallel(ctx, tasks, processFunc)

	// 检查是否有关键错误
	var criticalErr error
	for _, result := range results {
		if result.Error != nil {
			ce.componentLogger("crawler/amazon").Infof("提取器执行失败 (%s): %v", result.ID, result.Error)
			if ce.errorDetector.IsCriticalError(result.Error) && criticalErr == nil {
				criticalErr = result.Error
				ce.componentLogger("crawler/amazon").Infof("检测到关键错误: %v", result.Error)
			}
		}
	}

	return criticalErr
}

// getExtractorName 获取提取器名称
func getExtractorName(ext Extractor) string {
	return fmt.Sprintf("%T", ext)
}
