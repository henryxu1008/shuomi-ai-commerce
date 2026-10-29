// Package alibaba1688 提供1688验证码处理功能
package alibaba1688

import (
	"github.com/sirupsen/logrus"

	"time"
)

// CaptchaType 验证码类型
type CaptchaType int

const (
	CaptchaTypeUnknown CaptchaType = iota
	CaptchaTypeSlider
	CaptchaTypeClick
	CaptchaTypeImage
	CaptchaTypeText
	CaptchaTypeMath
)

func (ct CaptchaType) String() string {
	switch ct {
	case CaptchaTypeSlider:
		return "slider"
	case CaptchaTypeClick:
		return "click"
	case CaptchaTypeImage:
		return "image"
	case CaptchaTypeText:
		return "text"
	case CaptchaTypeMath:
		return "math"
	default:
		return "unknown"
	}
}

// CaptchaStatus 验证码状态
type CaptchaStatus int

const (
	CaptchaStatusPending CaptchaStatus = iota
	CaptchaStatusDetected
	CaptchaStatusProcessing
	CaptchaStatusSuccess
	CaptchaStatusFailed
	CaptchaStatusManualRequired
)

// CaptchaResult 验证码处理结果
type CaptchaResult struct {
	Type       CaptchaType
	Status     CaptchaStatus
	Attempts   int
	Error      error
	UsedMethod string
	Duration   time.Duration
}

// CaptchaHandler 验证码处理器
type CaptchaHandler struct {
	loggerDependency

	maxRetries int
}

// NewCaptchaHandler 创建新的验证码处理器
func NewCaptchaHandler(logs func(string) *logrus.Entry) *CaptchaHandler {
	return &CaptchaHandler{
		loggerDependency: loggerDependency{logFor: logs},
		maxRetries:       3,
	}
}
