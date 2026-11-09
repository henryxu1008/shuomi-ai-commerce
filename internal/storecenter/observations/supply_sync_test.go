package observations

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	"testing"
)

func TestSupplyCommitPreservesOriginalBatchAndCoverage(t *testing.T) {
	p := Checkpoint{Page: 1, SupplyOrders: &SupplyProgress{OrderIDs: []string{"PO-A", "PO-B"}, MissingIDs: []string{}}}
	current := Sync{Kind: SupplyOrders, Binding: Binding{Platform: storecenter.PlatformShein, ApplicationType: storecenter.ApplicationFullyManaged}, Progress: p}
	order := SupplyOrder{ID: "PO-A", Lines: []SupplyLine{}}
	next := copyCheckpoint(p)
	next.Page++
	next.Pages++
	next.Seen = 1
	next.SupplyOrders.Next = 2
	next.SupplyOrders.MissingIDs = []string{"PO-B"}
	next.Note("supply_ids_not_returned")
	rows := []Record{{ID: "PO-A", SupplyOrder: &order}}
	require.NoError(t, ValidateSupplyCommit(current, next, rows, "partial"))
	require.ErrorIs(t, ValidateSupplyCommit(current, next, rows, "completed"), ErrInvalid)
	changed := copyCheckpoint(next)
	changed.SupplyOrders.OrderIDs = []string{"PO-A", "PO-C"}
	require.ErrorIs(t, ValidateSupplyCommit(current, changed, rows, "partial"), ErrInvalid)
	changed = copyCheckpoint(next)
	changed.SupplyOrders.MissingIDs = []string{}
	require.ErrorIs(t, ValidateSupplyCommit(current, changed, rows, "partial"), ErrInvalid)
	require.ErrorIs(t, ValidateSupplyCommit(current, next, append(rows, rows...), "partial"), ErrInvalid)
	changed = copyCheckpoint(p)
	changed.Note("duration_limit")
	require.NoError(t, ValidateSupplyCommit(current, changed, nil, "partial"))
}
