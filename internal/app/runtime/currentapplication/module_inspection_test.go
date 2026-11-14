package currentapplication

import (
	"context"
	"encoding/json"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
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
func TestProductInspectionConsumesCurrentPermissionOwnerAndRejectsBroadGrants(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	for i := 0; i < 3; i++ {
		mock.ExpectQuery("(?s).*has_table_privilege.*").WillReturnRows(sqlmock.NewRows([]string{"current_user", "required", "forbidden"}).AddRow("source_acquisition_runtime", true, true))
	}
	got := InspectModules(context.Background(), &Config{ProductCollections: true, ProductAcquisitionDatabase: &DatabaseConfig{}}, func(context.Context, DatabaseConfig) (*gorm.DB, func(), error) { return db, func() {}, nil })
	for _, m := range got {
		if m.ID == "productCollections" || m.ID == "acquisitionPlans" || m.ID == "supplyCompletionDrafts" {
			require.NotEqual(t, "verified", m.Installed)
		}
	}
	require.NoError(t, mock.ExpectationsWereMet())
}
