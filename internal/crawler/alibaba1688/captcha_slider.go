// Package alibaba1688 提供1688滑动验证码处理功能
package alibaba1688

import (
	"context"
	"fmt"
	"strings"

	"time"

	"github.com/mxschmitt/playwright-go"
)

func (ch *CaptchaHandler) handleSliderCaptchaWithManualOption(ctx context.Context, page playwright.Page, allowManual bool) CaptchaResult {
	startTime := time.Now()
	attempts := 0
	maxRetries := ch.maxRetries

	sliderSelectors := []string{
		".nc_iconfont.btn_slide",
		"#nc_1_n1z",
		".nc-container .nc_iconfont",
		".slider-button",
		".captcha-slider-button",
		"[class*='slider'][class*='button']",
		".verify-slider-track .verify-slider-button",
		".slide-verify .slide-verify-slider-mask-item",
		"span.btn_slide",
		".geetest_slider_button",
		".geetest_slider",
		".captcha-slider-wrap .slider-btn",
	}

	for _, selector := range sliderSelectors {
		if err := ctx.Err(); err != nil {
			return canceledCaptchaResult(ctx, CaptchaTypeSlider)
		}
		sliderBtn, err := page.QuerySelector(selector)
		if err != nil || sliderBtn == nil {
			continue
		}

		isVisible, err := sliderBtn.IsVisible()
		if err != nil || !isVisible {
			continue
		}

		ch.componentLogger("crawler/alibaba1688").Info("检测到滑动验证码，使用人类行为策略滑动")

		for attempt := 1; attempt <= maxRetries; attempt++ {
			attempts++
			ch.componentLogger("crawler/alibaba1688").Infof("第 %d 次尝试人类行为滑动", attempt)

			if err := ch.performSliderAction(ctx, page, sliderBtn, "human"); err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					result := canceledCaptchaResult(ctx, CaptchaTypeSlider)
					result.Attempts = attempts
					return result
				}
				ch.componentLogger("crawler/alibaba1688").Warnf("人类行为滑动失败: %v", err)
			} else {
				if err := waitForContext(ctx, 2*time.Second); err != nil {
					result := canceledCaptchaResult(ctx, CaptchaTypeSlider)
					result.Attempts = attempts
					return result
				}
				if ch.checkSliderSuccess(page) {
					ch.componentLogger("crawler/alibaba1688").Info("人类行为策略滑动验证码成功")

					if err := ch.waitForPageRedirect(ctx, page); err != nil {
						if ctxErr := ctx.Err(); ctxErr != nil {
							result := canceledCaptchaResult(ctx, CaptchaTypeSlider)
							result.Attempts = attempts
							return result
						}
						ch.componentLogger("crawler/alibaba1688").Warnf("等待页面跳转失败: %v", err)
					}

					return CaptchaResult{
						Type:       CaptchaTypeSlider,
						Status:     CaptchaStatusSuccess,
						Attempts:   attempts,
						UsedMethod: "human_behavior",
						Duration:   time.Since(startTime),
					}
				}
			}

			if attempt < maxRetries {
				ch.componentLogger("crawler/alibaba1688").Infof("第 %d 次滑动失败，刷新页面重试", attempt)
				if _, err := page.Reload(); err != nil {
					ch.componentLogger("crawler/alibaba1688").Warnf("刷新页面失败: %v", err)
				} else {
					if err := waitForContext(ctx, 3*time.Second); err != nil {
						result := canceledCaptchaResult(ctx, CaptchaTypeSlider)
						result.Attempts = attempts
						return result
					}

					newSliderBtn, err := page.QuerySelector(selector)
					if err != nil || newSliderBtn == nil {
						ch.componentLogger("crawler/alibaba1688").Warn("刷新后未找到滑动按钮")
						continue
					}

					isVisible, err := newSliderBtn.IsVisible()
					if err != nil || !isVisible {
						ch.componentLogger("crawler/alibaba1688").Warn("刷新后滑动按钮不可见")
						continue
					}

					sliderBtn = newSliderBtn
					ch.componentLogger("crawler/alibaba1688").Info("页面刷新完成，准备重新尝试滑动")
				}
			}

			if err := waitForContext(ctx, time.Duration(1000+ch.randomDelay(1000))*time.Millisecond); err != nil {
				result := canceledCaptchaResult(ctx, CaptchaTypeSlider)
				result.Attempts = attempts
				return result
			}
		}

		if allowManual {
			ch.componentLogger("crawler/alibaba1688").Warn("人类行为滑动重试失败，等待用户手动操作")
		}
		result := ch.fallbackSliderCaptchaResult(ctx, page, allowManual)
		result.Attempts = attempts
		result.Type = CaptchaTypeSlider
		result.Duration = time.Since(startTime)
		return result
	}

	return CaptchaResult{
		Type:       CaptchaTypeSlider,
		Status:     CaptchaStatusFailed,
		Attempts:   attempts,
		UsedMethod: "none",
		Duration:   time.Since(startTime),
	}
}

func (ch *CaptchaHandler) fallbackSliderCaptchaResult(ctx context.Context, page playwright.Page, allowManual bool) CaptchaResult {
	if !allowManual {
		return CaptchaResult{
			Type:       CaptchaTypeSlider,
			Status:     CaptchaStatusFailed,
			Error:      fmt.Errorf("自动处理滑动验证码失败"),
			UsedMethod: "automatic_only",
		}
	}
	return ch.waitForManualSliderWithResult(ctx, page)
}

// performSliderAction 执行滑动操作
func (ch *CaptchaHandler) performSliderAction(ctx context.Context, page playwright.Page, sliderBtn playwright.ElementHandle, strategy string) error {
	box, err := sliderBtn.BoundingBox()
	if err != nil {
		return fmt.Errorf("获取滑动按钮位置失败: %w", err)
	}
	if box == nil {
		return fmt.Errorf("滑动按钮位置信息为空")
	}

	slideDistance, err := ch.calculateSlideDistance(page, box)
	if err != nil {
		ch.componentLogger("crawler/alibaba1688").Warnf("计算滑动距离失败，使用默认距离: %v", err)
		slideDistance = 260.0
	}

	ch.componentLogger("crawler/alibaba1688").Infof("开始滑动验证码，策略: %s, 滑动距离: %.2f", strategy, slideDistance)

	return ch.optimizedSlideWithHumanBehavior(ctx, page, box, slideDistance)
}

// calculateSlideDistance 计算滑动距离（优化版）
func (ch *CaptchaHandler) calculateSlideDistance(page playwright.Page, buttonBox *playwright.Rect) (float64, error) {
	if buttonBox == nil {
		return 0, fmt.Errorf("按钮位置信息为空")
	}

	trackSelectors := []string{
		".nc-lang-cnt",
		"#nc_1__scale_text",
		".nc_scale",
		".nc_wrapper",
		".slider-track",
		".captcha-slider-track",
		".slide-verify",
		"[class*='track']",
		"[class*='slider-container']",
		".geetest_slider_track",
		".geetest_canvas_bg",
	}

	for _, selector := range trackSelectors {
		track, err := page.QuerySelector(selector)
		if err != nil || track == nil {
			continue
		}

		trackBox, err := track.BoundingBox()
		if err != nil || trackBox == nil {
			continue
		}

		var distance float64
		if strings.Contains(selector, "nc-lang-cnt") || strings.Contains(selector, "nc_") {
			distance = trackBox.Width - buttonBox.Width + 5
		} else if strings.Contains(selector, "geetest") {
			distance = trackBox.Width - buttonBox.Width - 8
		} else {
			distance = trackBox.Width - buttonBox.Width - 15
		}

		ch.componentLogger("crawler/alibaba1688").Debugf("找到轨道: %s, 轨道宽度: %.2f, 按钮宽度: %.2f, 计算距离: %.2f",
			selector, trackBox.Width, buttonBox.Width, distance)

		if distance > 50 && distance < 400 {
			return distance, nil
		}
	}

	return 0, fmt.Errorf("未找到合适的滑动轨道")
}

// checkSliderSuccess 检查滑动验证是否成功（增强版）
func (ch *CaptchaHandler) checkSliderSuccess(page playwright.Page) bool {
	successSelectors := []string{
		".nc-lang-cnt[data-nc-lang='_ddddd']",
		".slider-success",
		".captcha-success",
		"[class*='success']",
		".verify-success",
		".geetest_success",
		".geetest_success_popup",
		".nc_iconfont.nc_success",
	}

	for _, selector := range successSelectors {
		element, err := page.QuerySelector(selector)
		if err == nil && element != nil {
			isVisible, _ := element.IsVisible()
			if isVisible {
				ch.componentLogger("crawler/alibaba1688").Info("检测到滑动验证成功标识")
				return true
			}
		}
	}

	captchaSelectors := []string{
		".nc_iconfont.btn_slide",
		".slider-button",
		".captcha-slider-button",
		".geetest_slider_button",
	}

	captchaExists := false
	for _, selector := range captchaSelectors {
		element, err := page.QuerySelector(selector)
		if err == nil && element != nil {
			isVisible, _ := element.IsVisible()
			if isVisible {
				captchaExists = true
				break
			}
		}
	}

	if !captchaExists {
		title, err := page.Title()
		if err == nil && isProductPageReadyAfterCaptcha(title, false) {
			ch.componentLogger("crawler/alibaba1688").Info("滑动验证码已消失，验证成功")
			return true
		}
	}

	return false
}

func isProductPageReadyAfterCaptcha(title string, hasPageData bool) bool {
	if hasPageData {
		return true
	}
	title = strings.TrimSpace(strings.ToLower(title))
	return title != "" && !strings.Contains(title, "captcha")
}

// waitForManualSliderWithResult 等待用户手动完成滑动验证
func (ch *CaptchaHandler) waitForManualSliderWithResult(ctx context.Context, page playwright.Page) CaptchaResult {
	ch.componentLogger("crawler/alibaba1688").Warn("自动滑动失败，请手动完成滑动验证码")
	ch.componentLogger("crawler/alibaba1688").Info("程序将等待您手动操作，请在浏览器中完成滑动验证...")

	timeout := manualCaptchaTimeout
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		if err := ctx.Err(); err != nil {
			return canceledCaptchaResult(ctx, CaptchaTypeSlider)
		}
		if ch.checkSliderSuccess(page) {
			ch.componentLogger("crawler/alibaba1688").Info("检测到验证码已完成，继续处理")
			return CaptchaResult{
				Type:       CaptchaTypeSlider,
				Status:     CaptchaStatusSuccess,
				UsedMethod: "manual",
			}
		}
		if err := waitForContext(ctx, time.Second); err != nil {
			return canceledCaptchaResult(ctx, CaptchaTypeSlider)
		}
	}

	return CaptchaResult{
		Type:       CaptchaTypeSlider,
		Status:     CaptchaStatusManualRequired,
		Error:      fmt.Errorf("等待用户手动操作超时"),
		UsedMethod: "manual_timeout",
	}
}

// waitForPageRedirect 等待页面跳转到商品页面
func (ch *CaptchaHandler) waitForPageRedirect(ctx context.Context, page playwright.Page) error {
	ch.componentLogger("crawler/alibaba1688").Info("等待页面跳转到商品页面...")

	timeout := 30 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		if err := ctx.Err(); err != nil {
			return err
		}
		title, err := page.Title()
		if err == nil && title != "Captcha Interception" && title != "" {
			ch.componentLogger("crawler/alibaba1688").Infof("页面已跳转，新标题: %s", title)

			if err := waitForContext(ctx, 3*time.Second); err != nil {
				return err
			}

			productSelectors := []string{
				"h1",
				".product-title",
				".offer-title",
				".main-content",
				".content",
			}

			for _, selector := range productSelectors {
				element, err := page.QuerySelector(selector)
				if err == nil && element != nil {
					isVisible, _ := element.IsVisible()
					if isVisible {
						ch.componentLogger("crawler/alibaba1688").Info("商品页面加载完成")
						return nil
					}
				}
			}
		}

		currentURL := page.URL()
		if strings.Contains(currentURL, "offer/") && strings.Contains(currentURL, ".html") {
			ch.componentLogger("crawler/alibaba1688").Infof("URL已跳转到商品页面: %s", currentURL)

			ready := false
			for i := 0; i < 10; i++ {
				if err := waitForContext(ctx, time.Second); err != nil {
					return err
				}

				title, _ := page.Title()
				if isProductPageReadyAfterCaptcha(title, false) {
					ch.componentLogger("crawler/alibaba1688").Infof("页面标题已更新: %s", title)
					ready = true
					break
				}

				hasData, err := page.Evaluate(`() => {
					return (typeof window.__INIT_DATA !== 'undefined' && window.__INIT_DATA !== null) ||
						   (typeof window.context !== 'undefined' && window.context !== null);
				}`)
				if err == nil && hasData == true {
					ch.componentLogger("crawler/alibaba1688").Info("检测到页面数据已加载")
					ready = true
					break
				}

				if i == 4 {
					ch.componentLogger("crawler/alibaba1688").Info("页面数据未加载，尝试刷新页面")
					page.Reload()
					if err := waitForContext(ctx, 3*time.Second); err != nil {
						return err
					}
				}

				ch.componentLogger("crawler/alibaba1688").Debugf("等待页面数据加载... (%d/10)", i+1)
			}

			if !ready {
				return fmt.Errorf("验证码未放行，页面仍停留在拦截状态")
			}

			if err := waitForContext(ctx, 2*time.Second); err != nil {
				return err
			}
			return nil
		}

		if err := waitForContext(ctx, time.Second); err != nil {
			return err
		}
	}

	return fmt.Errorf("等待页面跳转超时")
}

// optimizedSlideWithHumanBehavior 优化的人类行为滑动（使用真实轨迹算法）
func (ch *CaptchaHandler) optimizedSlideWithHumanBehavior(ctx context.Context, page playwright.Page, box *playwright.Rect, slideDistance float64) error {
	return ch.optimizedSlideWithRealTrack(ctx, page, box, slideDistance)
}
