package alibaba1688

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestOfferURLWithCaptchaTitleIsNotReadyAfterCaptcha(t *testing.T) {
	if isProductPageReadyAfterCaptcha("Captcha Interception", false) {
		t.Fatal("offer URL with captcha title and no page data should not be treated as ready")
	}
}

func TestOfferURLWithProductDataIsReadyAfterCaptcha(t *testing.T) {
	if !isProductPageReadyAfterCaptcha("Captcha Interception", true) {
		t.Fatal("offer URL with page data should be treated as ready")
	}
}

func TestOfferURLWithProductTitleIsReadyAfterCaptcha(t *testing.T) {
	if !isProductPageReadyAfterCaptcha("Wholesale Product", false) {
		t.Fatal("offer URL with non-captcha title should be treated as ready")
	}
}

func TestCaptchaTypeString(t *testing.T) {
	tests := []struct {
		captchaType CaptchaType
		expected    string
	}{
		{CaptchaTypeUnknown, "unknown"},
		{CaptchaTypeSlider, "slider"},
		{CaptchaTypeClick, "click"},
		{CaptchaTypeImage, "image"},
		{CaptchaTypeText, "text"},
		{CaptchaTypeMath, "math"},
	}

	for _, tt := range tests {
		if got := tt.captchaType.String(); got != tt.expected {
			t.Errorf("CaptchaType.String() for %v = %q, want %q", tt.captchaType, got, tt.expected)
		}
	}
}

func TestWaitForManualSliderReturnsCancellationPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	result := NewCaptchaHandler(nil).waitForManualSliderWithResult(ctx, nil)

	if !errors.Is(result.Error, context.Canceled) {
		t.Fatalf("manual slider error = %v, want context.Canceled", result.Error)
	}
	if elapsed := time.Since(started); elapsed >= 200*time.Millisecond {
		t.Fatalf("manual slider wait took %v after cancellation", elapsed)
	}
}

func TestCalculateMathExpression(t *testing.T) {
	handler := NewCaptchaHandler(nil)

	tests := []struct {
		expression string
		expected   string
	}{
		{"2 + 3", "5"},
		{"10-5", "5"},
		{"4 * 6", "24"},
		{"10 / 2", "5"},
		{"123+456", "579"},
		{"99 - 33", "66"},
	}

	for _, tt := range tests {
		if got := handler.calculateMathExpression(tt.expression); got != tt.expected {
			t.Errorf("calculateMathExpression(%q) = %q, want %q", tt.expression, got, tt.expected)
		}
	}
}

func TestStatisticsReset(t *testing.T) {
	handler := NewCaptchaHandler(nil)

	handler.recordSuccess(CaptchaTypeSlider)
	handler.recordFailure(CaptchaTypeImage)
	handler.recordManual(CaptchaTypeText)

	stats := handler.GetStatistics()
	if stats.TotalCount == 0 {
		t.Error("Expected TotalCount > 0 after recording events")
	}

	handler.ResetStatistics()
	stats = handler.GetStatistics()

	if stats.TotalCount != 0 || stats.SuccessCount != 0 || stats.FailedCount != 0 || stats.ManualCount != 0 {
		t.Error("Expected all statistics to be reset to 0")
	}
}
