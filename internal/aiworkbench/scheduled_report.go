package aiworkbench

import (
	"github.com/google/uuid"
	"time"
)

const DailyReportKind = "store.operating-daily-report"

type DailyReportSpec struct {
	StoreID      string `json:"storeId"`
	Version      int    `json:"version"`
	PeriodPolicy string `json:"periodPolicy"`
	BusinessZone string `json:"businessZone"`
}

func (s DailyReportSpec) Valid() bool {
	id, err := uuid.Parse(s.StoreID)
	return err == nil && id != uuid.Nil && id.String() == s.StoreID && s.Version == 1 && s.PeriodPolicy == "previous_natural_day" && s.BusinessZone == "Asia/Shanghai"
}
func (s DailyReportSpec) Period(slot time.Time) string {
	if !s.Valid() || slot.IsZero() {
		return ""
	}
	return slot.In(time.FixedZone("UTC+8", 8*3600)).AddDate(0, 0, -1).Format(time.DateOnly)
}
func (p ScheduledPlanDraft) Kind() string {
	if p.Report != nil {
		return DailyReportKind
	}
	return ContentTaskKind
}
func (p ScheduledPlanDraft) TotalUpperBound() (uint64, error) {
	if p.Report != nil && p.Budget.Unit == "none" && p.Budget.PerRunUpperBoundMicros == 0 {
		return 0, nil
	}
	return p.Budget.TotalUpperBoundMicros()
}
