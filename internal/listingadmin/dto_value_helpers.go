package listingadmin

import (
	"time"

	"task-processor/internal/pkg/types"
)

func ptrFloat64(value float64) *float64 {
	return &value
}

func ptrString(value string) *string {
	return &value
}

func flexibleTimeValue(value *time.Time) types.FlexibleTime {
	if value == nil {
		return types.FlexibleTime{}
	}
	return types.FlexibleTime{Time: *value}
}
