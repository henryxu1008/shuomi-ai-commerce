package goods

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/shein/model"
	"testing"
	"time"
)

func TestShelfEligibilityUsesOriginalAuditAndManagedPlan(t *testing.T) {
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	approved, off, on, rejected, appeal := 2, 0, 1, 3, 5
	for _, tc := range []struct {
		name         string
		mode         model.ApplicationMode
		plan         model.ProductSKC
		audit, shelf *int
		want         bool
	}{
		{"self", model.ModeSelfOperated, model.ProductSKC{}, &approved, &off, true},
		{"full", model.ModeFullyManaged, model.ProductSKC{ShelfWay: "1"}, &approved, &off, false},
		{"future", model.ModeSemiManaged, model.ProductSKC{ShelfWay: "2", HopeOnSaleDate: "2026-10-20 08:00:00"}, &approved, &off, false},
		{"due", model.ModeSemiManaged, model.ProductSKC{ShelfWay: "2", HopeOnSaleDate: "2026-10-19 08:00:00"}, &approved, &off, true},
		{"auto", model.ModeSemiManaged, model.ProductSKC{ShelfWay: "1"}, &approved, &off, true},
		{"unknown plan", model.ModeSemiManaged, model.ProductSKC{}, &approved, &off, false},
		{"rejected", model.ModeSelfOperated, model.ProductSKC{}, &rejected, &off, false},
		{"appeal", model.ModeSelfOperated, model.ProductSKC{}, &appeal, &off, false},
		{"missing audit", model.ModeSelfOperated, model.ProductSKC{}, nil, &off, false},
		{"missing shelf", model.ModeSelfOperated, model.ProductSKC{}, &approved, nil, false},
		{"already on", model.ModeSelfOperated, model.ProductSKC{}, &approved, &on, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := ShelfEligibility(tc.mode, tc.plan, tc.audit, tc.shelf, now)
			require.Equal(t, tc.want, got)
		})
	}
	require.Equal(t, "审核驳回", AuditLabel(&rejected))
	require.Equal(t, "申诉中", AuditLabel(&appeal))
	unknown := 99
	require.Equal(t, "审核状态未知", AuditLabel(&unknown))
}
