// Package alibaba1688 提供人类行为模拟功能
package alibaba1688

import (
	"context"
	"math"
	"time"

	"github.com/mxschmitt/playwright-go"
)

// randomDelay 生成随机延迟（毫秒）
func (ch *CaptchaHandler) randomDelay(maxMs int) int {
	if maxMs <= 0 {
		return 0
	}
	return int(time.Now().UnixNano()%int64(maxMs)) + 1
}

// 基于真实轨迹的人类滑动算法
// 完整版：包含预热、探索、滑动、结束四个阶段
// 真实轨迹特征分析：
// 1. 总时长: ~2000ms
// 2. 距离: ~262px
// 3. 启动延迟: ~250ms
// 4. 速度阶段: 慢启动(0-30%) -> 匀速(30-70%) -> 减速(70-90%) -> 终点停留(90-100%)
// 5. 垂直抖动: ±10px
// 6. 时间间隔: 7-56ms随机，不均匀

func (ch *CaptchaHandler) optimizedSlideWithRealTrack(ctx context.Context, page playwright.Page, sliderBox *playwright.Rect, slideDistance float64) error {
	startX := sliderBox.X + sliderBox.Width/2
	startY := sliderBox.Y + sliderBox.Height/2
	endX := startX + slideDistance
	endY := startY + 15 // 真实轨迹显示终点Y会下降约15px

	// === 第一阶段：预热行为 (模拟人类从远处移动到滑块附近) ===
	// 真人通常不会突然出现在滑块上，而是从附近慢慢靠近
	page.Mouse().Move(startX-100+float64(ch.randomDelay(50)), startY-50+float64(ch.randomDelay(30)))
	if err := waitForContext(ctx, 300+time.Duration(ch.randomDelay(400))*time.Millisecond); err != nil {
		return err
	}

	// 移动到滑块上方附近（但不是直接在上面）
	page.Mouse().Move(startX+float64(ch.randomDelay(40)-20), startY-30+float64(ch.randomDelay(20)))
	if err := waitForContext(ctx, 200+time.Duration(ch.randomDelay(300))*time.Millisecond); err != nil {
		return err
	}

	// === 第二阶段：探索行为 (在滑块附近徘徊，模拟人类确认目标) ===
	// 真人会在按下去之前确认一下位置
	for i := 0; i < 2+ch.randomDelay(2); i++ {
		offsetX := float64(ch.randomDelay(20) - 10)
		offsetY := float64(ch.randomDelay(20) - 10)
		page.Mouse().Move(startX+offsetX, startY+offsetY)
		if err := waitForContext(ctx, 100+time.Duration(ch.randomDelay(150))*time.Millisecond); err != nil {
			return err
		}
	}

	// 犹豫一下（真实人类按下前会有明显犹豫）
	if err := waitForContext(ctx, 300+time.Duration(ch.randomDelay(400))*time.Millisecond); err != nil {
		return err
	}

	// === 第三阶段：按下并滑动 ===
	// 移动到精确起点位置
	page.Mouse().Move(startX, startY)
	if err := waitForContext(ctx, 50+time.Duration(ch.randomDelay(50))*time.Millisecond); err != nil {
		return err
	}

	// 按下鼠标
	page.Mouse().Down()
	if err := waitForContext(ctx, 50+time.Duration(ch.randomDelay(50))*time.Millisecond); err != nil {
		return err
	}

	// 真实轨迹显示有~250ms的启动犹豫
	if err := waitForContext(ctx, 200+time.Duration(ch.randomDelay(150))*time.Millisecond); err != nil {
		return err
	}

	// 生成基于真实轨迹的滑动点
	points := ch.generateRealHumanTrackPoints(startX, startY, endX, endY, 60)

	// 回放轨迹
	for i, pt := range points {
		if err := ctx.Err(); err != nil {
			return err
		}
		page.Mouse().Move(pt.x, pt.y)

		// 真实轨迹的时间间隔是不均匀的：7-56ms随机
		if i < len(points)-1 {
			delay := 7 + ch.randomDelay(49) // 7-56ms
			if err := waitForContext(ctx, time.Duration(delay)*time.Millisecond); err != nil {
				return err
			}
		}

		// 在滑动过程中添加随机停顿（模拟人类"思考"）
		if ch.randomDelay(100) < 15 { // 15%概率停顿
			stopDuration := 30 + ch.randomDelay(80)
			if err := waitForContext(ctx, time.Duration(stopDuration)*time.Millisecond); err != nil {
				return err
			}
		}
	}

	// === 第四阶段：结束行为 ===
	// 终点停留（真实轨迹显示松开前有~500ms停留）
	if err := waitForContext(ctx, 400+time.Duration(ch.randomDelay(200))*time.Millisecond); err != nil {
		return err
	}

	// 松开鼠标
	page.Mouse().Up()

	// 滑动后可能有一些微小移动（真实人类特征）
	if err := waitForContext(ctx, 50+time.Duration(ch.randomDelay(100))*time.Millisecond); err != nil {
		return err
	}

	return nil
}

// generateRealHumanTrackPoints 生成基于真实轨迹特征的点
func (ch *CaptchaHandler) generateRealHumanTrackPoints(startX, startY, endX, endY float64, numPoints int) []point {
	points := make([]point, numPoints)

	baseY := startY
	verticalNoise := 12.0 // 稍微增加垂直抖动范围

	// 预先生成一些随机回退点的位置
	backtrackPositions := make(map[int]bool)
	backtrackCount := 2 + ch.randomDelay(3) // 2-5次回退
	for i := 0; i < backtrackCount; i++ {
		idx := 5 + ch.randomDelay(numPoints-10)
		backtrackPositions[idx] = true
	}

	// 预先生成随机停顿点（模拟人类思考）
	pausePositions := make(map[int]int)
	pauseCount := 3 + ch.randomDelay(5) // 3-8次停顿
	for i := 0; i < pauseCount; i++ {
		idx := 3 + ch.randomDelay(numPoints-6)
		pausePositions[idx] = 30 + ch.randomDelay(100) // 停顿时长30-130ms
	}

	for i := 0; i < numPoints; i++ {
		t := float64(i) / float64(numPoints-1)

		// 真实轨迹的速度曲线（基于采样分析）
		var progress float64
		if t < 0.15 {
			// 0-15%: 慢启动阶段（真实轨迹显示这个阶段速度很慢）
			progress = 0.3 * math.Pow(t/0.15, 2)
		} else if t < 0.50 {
			// 15-50%: 匀速阶段（真实轨迹显示这个阶段速度最快且均匀）
			progress = 0.3 + 0.5*(t-0.15)/0.35
		} else if t < 0.85 {
			// 50-85%: 减速阶段（真实轨迹显示接近终点时明显减速）
			progress = 0.8 + 0.15*math.Pow((t-0.50)/0.35, 0.5)
		} else {
			// 85-100%: 终点停留阶段（真实轨迹显示最后有明显停留）
			progress = 0.95 + 0.05*math.Pow((t-0.85)/0.15, 0.3)
		}

		// 水平位移（线性）
		x := startX + (endX-startX)*progress

		// 垂直位移（带抖动的正弦波 + 随机噪声）
		// 真实轨迹显示Y坐标在中间偏高（730-745），终点下降到724
		yOffset := math.Sin(t*math.Pi*4) * verticalNoise * 0.3 // 正弦波动
		yOffset += float64(ch.randomDelay(24) - 12)            // 随机噪声扩大范围
		yOffset += t * (endY - startY)                         // 终点Y下降趋势

		// 确保Y在合理范围内
		y := baseY + yOffset
		if y < endY-5 {
			y = endY - 5
		}
		if y > startY+18 {
			y = startY + 18
		}

		points[i] = point{x: x, y: y}
	}

	// 添加微小的回退点（真实轨迹显示有少量回退）
	for idx := range backtrackPositions {
		if idx < numPoints-2 && idx > 2 {
			backtrackAmount := float64(ch.randomDelay(8) + 2) // 2-10px回退
			points[idx].x -= backtrackAmount
		}
	}

	return points
}

type point struct {
	x, y float64
}
