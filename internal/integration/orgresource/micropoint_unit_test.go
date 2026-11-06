package orgresourceadapter

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/orgresource"
	"testing"
)

func TestResourceSchemaRefusesMissingOrMixedAIUnitWithoutRepair(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	require.Error(t, db.Create(&organizationResourceBucketRow{OrganizationID: "missing", ResourceType: "ai_point", Available: 100}).Error)
	require.Error(t, db.Create(&organizationResourceBucketRow{OrganizationID: "old", ResourceType: "ai_point", QuantityUnit: "point", Available: 100}).Error)
	require.NoError(t, db.Create(&organizationResourceBucketRow{OrganizationID: "current", ResourceType: "ai_point", QuantityUnit: orgresource.AIPointQuantityUnitV1, Available: 100}).Error)
	require.NoError(t, db.Create(&organizationResourceBucketRow{OrganizationID: "data", ResourceType: "data_row", Available: 100}).Error)
	require.NoError(t, AutoMigrate(db), "current installation is verifiable without writes")
	require.NoError(t, db.Exec("DROP TABLE saas_member_ai_point_months").Error)
	require.NoError(t, db.Exec("CREATE TABLE saas_member_ai_point_months (organization_id text, member_id text, month_start datetime, reserved bigint, consumed bigint)").Error)
	require.Error(t, AutoMigrate(db), "an existing ledger cannot be upgraded or reinterpreted")
	require.False(t, db.Migrator().HasColumn(&memberAIPointMonthRow{}, "quantity_unit"))
}

func TestPurchasedAIGrantConvertsSourcePointsExactlyOnce(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	repo, err := NewGormRepository(db, TransactionConfig{})
	require.NoError(t, err)
	service, err := orgresource.NewPurchasedResourceGrantService(repo, TrustedCommercialGrantAuthorizer{})
	require.NoError(t, err)
	input := orgresource.PurchasedResourceGrantInput{OrganizationID: "org", OperationID: "grant", CommercialOrderID: "order", CommercialOrderItemID: "item", ResourceType: orgresource.ResourceAIPoint, Quantity: 10, Principal: orgresource.Principal{ID: "commercial-billing", Kind: orgresource.PrincipalTrustedCommercial}}
	first, err := service.GrantPurchasedResource(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "10", first.Snapshot.Quantity, "the commercial item stays in whole points")
	require.Equal(t, "10000000", first.Snapshot.LedgerQuantity)
	require.Equal(t, orgresource.AIPointQuantityUnitV1, first.Snapshot.UnitVersion)
	input.OperationID = "another-key"
	again, err := service.GrantPurchasedResource(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, first.Snapshot, again.Snapshot)
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Where("resource_type = ?", "ai_point").Take(&bucket).Error)
	require.EqualValues(t, 10000000, bucket.Available)
	balances, err := repo.ReadBalances(context.Background(), "org")
	require.NoError(t, err)
	for _, balance := range balances.Resources {
		if balance.ResourceType == orgresource.ResourceAIPoint {
			require.Equal(t, "10", *balance.Available)
			require.Equal(t, orgresource.AIPointQuantityUnitV1, balance.UnitVersion)
		}
	}
}
