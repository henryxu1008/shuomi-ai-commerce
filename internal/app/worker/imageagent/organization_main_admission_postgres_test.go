package imageagentworker

import (
	"os"
	"strings"
	"testing"

	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/imageagent"
	resourceadapter "task-processor/internal/integration/orgresource"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOrganizationMainUnpricedGenerationCreatesNoCommercialReservation(t *testing.T) {
	db := admissionPostgres(t)
	delegate := &recordingMainExecutor{}
	executor := organizationMainSlotExecutor{delegate: delegate}
	_, err := executor.GenerateQuotedSlot(mainAdmissionContext(), mainAdmissionInput(), mainGenerationQuote())
	require.ErrorIs(t, err, imageagent.ErrBudgetQuoteUnavailable)
	require.Equal(t, imageagent.ProviderNotDispatched, imageagent.ProviderDispatchStateOf(err))
	require.Zero(t, delegate.generateCalls, "generation stays closed without its own approved admission contract")
	var reservations int64
	require.NoError(t, db.Table("saas_organization_resource_reservations").Count(&reservations).Error)
	require.Zero(t, reservations, "denied admission cannot create a held reservation")
}

func admissionPostgres(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ISSUE487_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL ISSUE487_TEST_DSN")
	}
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	schema := "issue487_admission_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE SCHEMA "+schema).Error)
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		pool, _ := db.DB()
		_ = pool.Close()
		_ = root.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		rootPool, _ := root.DB()
		_ = rootPool.Close()
	})
	require.NoError(t, resourceadapter.AutoMigrate(db))
	require.NoError(t, aistore.AutoMigrateInvocationLedger(db))
	return db
}
