package goods

import (
	"task-processor/internal/marketplace/shein/model"
	"time"
)

func AuditLabel(state *int) string {
	if state == nil {
		return "审核状态未知"
	}
	switch *state {
	case -1:
		return "受理失败"
	case 1:
		return "待审核"
	case 2:
		return "审核通过"
	case 3:
		return "审核驳回"
	case 4:
		return "已撤回"
	case 5:
		return "申诉中"
	default:
		return "审核状态未知"
	}
}
func ShelfEligibility(mode model.ApplicationMode, plan model.ProductSKC, audit, shelf *int, now time.Time) (bool, string) {
	if mode == model.ModeFullyManaged {
		return false, "等待平台安排；全托管不支持人工上架"
	}
	if !mode.Valid() {
		return false, "店铺经营模式未知"
	}
	if audit == nil || *audit != 2 {
		return false, "原版本尚未审核通过"
	}
	if shelf == nil {
		return false, "美国站当前在架状态未知"
	}
	if *shelf != 0 {
		if *shelf == 1 {
			return false, "当前已上架"
		}
		return false, "美国站当前在架状态未知"
	}
	if mode == model.ModeSemiManaged {
		switch plan.ShelfWay {
		case "1":
			if plan.HopeOnSaleDate != "" {
				return false, "原上架计划未知"
			}
		case "2":
			at, e := time.ParseInLocation("2006-01-02 15:04:05", plan.HopeOnSaleDate, time.FixedZone("Beijing", 8*3600))
			if e != nil || at.Format("2006-01-02 15:04:05") != plan.HopeOnSaleDate {
				return false, "原定时计划未知"
			}
			if now.Before(at) {
				return false, "计划时间未到，等待平台"
			}
		default:
			return false, "原上架计划未知"
		}
	}
	return true, "可明确确认首次上架；平台仍可能按当前政策拒绝"
}
