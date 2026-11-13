package httpapi

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/storecenter"
)

func TestStoreHistoryProjectionRechecksEveryReadAndFailsClosed(t *testing.T) {
	scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "user"}
	messages := []aiworkbench.Message{{Content: "店铺概况", Kind: "STORE_OVERVIEW_REF", StoreQueryStatus: "ACTIVE"}}
	calls := 0
	app := &aiWorkbenchApplication{}
	app.scopes.storeOverview = func(_ context.Context, got aiworkbench.Scope, status storecenter.OverviewScope) (storecenter.VisibleStoreOverview, error) {
		calls++
		require.Equal(t, scope, got)
		require.Equal(t, storecenter.OverviewAll, status)
		if calls > 1 {
			return storecenter.VisibleStoreOverview{}, errors.New("access revoked")
		}
		return storecenter.VisibleStoreOverview{Total: 3, Statuses: map[string]int64{string(storecenter.RecordStatusActive): 3}}, nil
	}
	app.scopes.storeOverviewAuthorize = func(context.Context, aiworkbench.Scope, storecenter.VisibleStoreOverview) error { return nil }
	first := app.projectStoreMessages(context.Background(), scope, messages)
	require.NotNil(t, first[0].StoreOverview)
	require.EqualValues(t, 3, first[0].StoreOverview.Total)
	second := app.projectStoreMessages(context.Background(), scope, messages)
	require.Nil(t, second[0].StoreOverview)
	require.True(t, second[0].StoreOverviewUnavailable)
	require.Equal(t, 2, calls)
}

func TestStoreHistoryProjectionFailsClosedForWholePage(t *testing.T) {
	scope := aiworkbench.Scope{OrganizationID: "org", ActorID: "user"}
	messages := []aiworkbench.Message{{Kind: "STORE_OVERVIEW_REF", StoreQueryStatus: "ACTIVE"}, {Kind: "STORE_OVERVIEW_REF", StoreQueryStatus: "ALL_NON_DELETED"}}
	app := &aiWorkbenchApplication{}
	calls := 0
	app.scopes.storeOverview = func(context.Context, aiworkbench.Scope, storecenter.OverviewScope) (storecenter.VisibleStoreOverview, error) {
		calls++
		return storecenter.VisibleStoreOverview{Total: 2, Statuses: map[string]int64{string(storecenter.RecordStatusActive): 1}, ActivePlatforms: map[string]int64{"SHEIN": 1}}, nil
	}
	app.scopes.storeOverviewAuthorize = func(context.Context, aiworkbench.Scope, storecenter.VisibleStoreOverview) error {
		return errors.New("revoked before response")
	}
	projected := app.projectStoreMessages(context.Background(), scope, messages)
	require.Equal(t, 1, calls, "all references must derive from one authorized snapshot")
	for _, item := range projected {
		require.Nil(t, item.StoreOverview)
		require.True(t, item.StoreOverviewUnavailable)
	}
	app.scopes.storeOverviewAuthorize = func(context.Context, aiworkbench.Scope, storecenter.VisibleStoreOverview) error { return nil }
	projected = app.projectStoreMessages(context.Background(), scope, messages)
	require.EqualValues(t, 1, projected[0].StoreOverview.Total)
	require.EqualValues(t, 2, projected[1].StoreOverview.Total)
}
