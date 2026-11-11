package currentapplication

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"testing"
)

func TestModuleInspectionDoesNotStartRuntimeOrOpenMissingOwners(t *testing.T) {
	calls := 0
	got := InspectModules(context.Background(), &Config{}, func(context.Context, DatabaseConfig) (*gorm.DB, func(), error) {
		calls++
		t.Fatal("missing owner must not be opened")
		return nil, nil, nil
	})
	raw, e := json.Marshal(got)
	require.NoError(t, e)
	require.Contains(t, string(raw), "aiWorkbench")
	require.Zero(t, calls)
	for _, m := range got {
		require.False(t, m.Ready)
		require.NotEqual(t, "verified", m.Installed)
	}
}
