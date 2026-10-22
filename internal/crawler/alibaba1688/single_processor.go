// Package alibaba1688 提供1688单个产品处理器
package alibaba1688

import (
	"github.com/sirupsen/logrus"

	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"task-processor/internal/core/config"

	"task-processor/internal/crawler/alibaba1688/extractor"
	"task-processor/internal/crawler/alibaba1688/model"
	"time"
)

// SingleProcessor 单个产品处理器
type SingleProcessor struct {
	loggerDependency

	config         *config.Config
	urlHelper      *URLHelper
	productChecker *ProductChecker
	extractor      *extractor.ProductExtractor
	pageOperator   *PageOperator
}

// NewSingleProcessor 创建新的单个产品处理器
func NewSingleProcessor(cfg *config.Config, urlHelper *URLHelper, productChecker *ProductChecker, logs func(string) *logrus.Entry) *SingleProcessor {
	return &SingleProcessor{
		loggerDependency: loggerDependency{logFor: logs},
		config:           cfg,
		urlHelper:        urlHelper,
		productChecker:   productChecker,
		extractor:        extractor.NewProductExtractor(logs),
		pageOperator:     NewPageOperator(logs),
	}
}

// Prepare provisions the browser runtime without navigating to a product.
func (sp *SingleProcessor) Prepare(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if sp == nil || sp.config == nil {
		return errors.New("1688 single processor is not configured")
	}
	manager := sp.newPublicBrowserManager()
	if _, _, _, cleanup, err := manager.CreateBrowser(); err != nil {
		return err
	} else {
		defer closeOnContextDone(ctx, cleanup)()
	}
	return ctx.Err()
}

// ProcessWithSingleBrowser 使用单个浏览器处理产品
func (sp *SingleProcessor) ProcessWithSingleBrowser(ctx context.Context, url string, startTime time.Time) (*model.Product1688, error) {
	return sp.processWithBrowserManager(ctx, url, startTime, sp.newPublicBrowserManager())
}

func (sp *SingleProcessor) newPublicBrowserManager() *BrowserManager {
	return NewPublicBrowserManager(sp.config, sp.logFor)
}

func (sp *SingleProcessor) processWithBrowserManager(ctx context.Context, url string, startTime time.Time, browserManager *BrowserManager) (*model.Product1688, error) {
	sp.componentLogger("crawler/alibaba1688").Infof("使用单浏览器模式处理1688产品: %s", url)

	// 验证和标准化URL
	normalizedURL, err := sp.urlHelper.ValidateAndNormalizeURL(url)
	if err != nil {
		return nil, NewPublicAccessError(PublicAccessFailureInvalidURL, fmt.Errorf("URL验证失败: %w", err))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 创建浏览器实例
	_, _, page, cleanup, err := browserManager.CreateBrowser()
	if err != nil {
		return nil, NewPublicAccessError(PublicAccessFailureBrowser, err)
	}
	defer closeOnContextDone(ctx, cleanup)()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 导航到产品页面
	navErr := sp.pageOperator.NavigateToProductWithoutManualCaptcha(ctx, page, normalizedURL)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if navErr != nil {
		kind := PublicAccessFailureTransport
		if isChallengeError(navErr) {
			kind = PublicAccessFailureChallenge
		}
		return nil, NewPublicAccessError(kind, fmt.Errorf("导航到产品页面失败: %w", navErr))
	}

	// 提取产品信息
	product, err := sp.extractor.ExtractProductFromPage(ctx, page, normalizedURL)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, NewPublicAccessError(PublicAccessFailureMissingFields, fmt.Errorf("提取产品信息失败: %w", err))
	}

	// 验证产品信息
	if validateErr := sp.productChecker.ValidateProduct(product); validateErr != nil {
		kind := classifyProductValidationFailure(validateErr)
		return nil, NewPublicAccessError(kind, fmt.Errorf("产品信息验证失败: %w", validateErr))
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	duration := time.Since(startTime)
	sp.componentLogger("crawler/alibaba1688").Infof("单浏览器模式处理完成: %s, 耗时: %v", product.Title, duration)

	return product, nil
}

func closeOnContextDone(ctx context.Context, close func()) func() {
	closeOnce := sync.OnceFunc(close)
	stop := context.AfterFunc(ctx, closeOnce)
	return func() {
		stop()
		closeOnce()
	}
}

func classifyProductValidationFailure(err error) PublicAccessFailureKind {
	var requiredErr *requiredFieldsError
	if errors.As(err, &requiredErr) && requiredErr != nil {
		return PublicAccessFailureMissingFields
	}
	return PublicAccessFailureValidation
}

func isChallengeError(err error) bool {
	if err == nil {
		return false
	}
	var accessErr *PublicAccessError
	if errors.As(err, &accessErr) && accessErr != nil {
		return accessErr.Kind == PublicAccessFailureChallenge
	}
	message := strings.ToLower(err.Error())
	for _, marker := range []string{"captcha", "challenge", "验证码", "登录", "拦截"} {
		if strings.Contains(message, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}
