// Package alibaba1688 提供1688平台处理器核心功能
package alibaba1688

import (
	"github.com/sirupsen/logrus"

	"context"
	"errors"
	"task-processor/internal/core/config"

	"task-processor/internal/crawler/alibaba1688/model"
	"time"
)

// Alibaba1688Processor 1688爬虫处理器
type Alibaba1688Processor struct {
	loggerDependency

	config          *config.Config
	singleProcessor *SingleProcessor
	urlHelper       *URLHelper
	productChecker  *ProductChecker
}

// NewAlibaba1688Processor 使用全局配置创建1688处理器
func NewAlibaba1688Processor(cfg *config.Config, logs func(string) *logrus.Entry) *Alibaba1688Processor {
	componentLogger(logs, "crawler/alibaba1688").Infof("创建1688处理器")

	// 创建辅助组件
	urlHelper := NewURLHelper(logs)
	productChecker := NewProductChecker(logs)
	singleProcessor := NewSingleProcessor(cfg, urlHelper, productChecker, logs)

	return &Alibaba1688Processor{
		loggerDependency: loggerDependency{logFor: logs},
		config:           cfg,
		singleProcessor:  singleProcessor,
		urlHelper:        urlHelper,
		productChecker:   productChecker,
	}
}

// Prepare provisions the public browser dependencies before a local-agent job
// is claimed, so setup time does not consume the execution lease.
func (ap *Alibaba1688Processor) Prepare(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if ap == nil || ap.singleProcessor == nil {
		return errors.New("1688 crawler processor is not configured")
	}
	return ap.singleProcessor.Prepare(ctx)
}

// Process 处理1688产品页面
func (ap *Alibaba1688Processor) Process(ctx context.Context, url string) (*model.Product1688, error) {
	startTime := time.Now()
	ap.componentLogger("crawler/alibaba1688").Infof("开始处理1688产品: %s", url)

	return ap.singleProcessor.ProcessWithSingleBrowser(ctx, url, startTime)
}

// Shutdown 关闭处理器
func (ap *Alibaba1688Processor) Shutdown() {
	ap.componentLogger("crawler/alibaba1688").Info("关闭1688处理器...")
}
