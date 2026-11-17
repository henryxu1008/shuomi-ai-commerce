package orgresourceadapter

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/orgresource"
)

func TestResourceEventReadIsScopedStableAndBoundToFilters(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	repo, err := NewGormRepository(db, TransactionConfig{})
	require.NoError(t, err)
	at := time.Now().UTC().Truncate(time.Microsecond)
	for _, org := range []string{"org-a", "org-b"} {
		for i := 0; i < 3; i++ {
			op := uuid.NewString()
			require.NoError(t, db.Create(&organizationResourceOperationRow{OrganizationID: org, OperationID: op, OperationType: "synthetic", RequestFingerprint: "synthetic", State: "succeeded", ImmutableResult: "{}"}).Error)
			require.NoError(t, db.Create(&organizationResourceEventRow{QuantityUnit: resourceQuantityUnit(string(orgresource.ResourceDataRow)), EventID: uuid.NewString(), OrganizationID: org, OperationID: op, ResourceType: string(orgresource.ResourceDataRow), Quantity: 1, AvailableAfter: 10, AllocatedAfter: 8, Reason: "synthetic", SourceType: "synthetic", SourceIdentity: op, CreatedAt: at}).Error)
		}
	}
	query := orgresource.EventQuery{OrganizationID: "org-a", ResourceType: orgresource.ResourceDataRow, Limit: 2}
	first, err := repo.ListEvents(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	require.NotNil(t, first.NextCursor)
	rawCursor, err := base64.RawURLEncoding.DecodeString(*first.NextCursor)
	require.NoError(t, err)
	var decoded resourceEventCursor
	require.NoError(t, json.Unmarshal(rawCursor, &decoded))
	oldScope := sha256.Sum256([]byte(`{"Organization":"org-a","Resource":"data_row","From":null,"Until":null}`))
	require.Equal(t, hex.EncodeToString(oldScope[:]), decoded.Scope, "all events keep the existing cursor scope")
	query.Cursor = *first.NextCursor
	second, err := repo.ListEvents(context.Background(), query)
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Nil(t, second.NextCursor)
	for _, item := range first.Items {
		require.NotEqual(t, item.EventID, second.Items[0].EventID)
		require.Equal(t, "8", item.AllocatedAfter)
	}
	query.OrganizationID = "org-b"
	_, err = repo.ListEvents(context.Background(), query)
	require.ErrorIs(t, err, orgresource.ErrInvalidInput)
	query.OrganizationID = "org-a"
	query.ResourceType = orgresource.ResourceAIPoint
	_, err = repo.ListEvents(context.Background(), query)
	require.ErrorIs(t, err, orgresource.ErrInvalidInput)
	query.Cursor = ""
	query.Limit = 51
	_, err = repo.ListEvents(context.Background(), query)
	require.ErrorIs(t, err, orgresource.ErrInvalidInput)
}

func TestReadResourceEventSourceKeepsQuantityUnitValidation(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	repo, err := NewGormRepository(db, TransactionConfig{})
	require.NoError(t, err)
	ctx := context.Background()
	const org = "org-a"
	for _, resource := range []orgresource.ResourceType{orgresource.ResourceAIPoint, orgresource.ResourceStoreRenewalPeriod, orgresource.ResourceDataRow} {
		t.Run(string(resource), func(t *testing.T) {
			op, eventID := uuid.NewString(), uuid.NewString()
			require.NoError(t, db.Create(&organizationResourceOperationRow{
				OrganizationID: org, OperationID: op, OperationType: "fixture",
				RequestFingerprint: "fixture", State: "succeeded", ImmutableResult: "{}",
			}).Error)
			require.NoError(t, db.Create(&organizationResourceEventRow{
				QuantityUnit: resourceQuantityUnit(string(resource)), EventID: eventID, OrganizationID: org,
				OperationID: op, ResourceType: string(resource), Quantity: 1, Reason: "fixture",
				SourceType: "fixture", SourceIdentity: op,
			}).Error)
			source, found, err := repo.ReadEventSource(ctx, org, eventID)
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, orgresource.EventSource{
				OrganizationID: org, EventID: eventID, ResourceType: resource,
				SourceType: "fixture", SourceIdentity: op,
			}, source)
			source, found, err = repo.ReadEventSource(ctx, "other-org", eventID)
			require.NoError(t, err)
			require.False(t, found)
			require.Empty(t, source)
		})
	}
}

func TestResourceEventsFilterDeductionsBeforePagination(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	repo, err := NewGormRepository(db, TransactionConfig{})
	require.NoError(t, err)
	ctx := context.Background()
	at := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	add := func(org string, seconds int, consumed int64) {
		op := uuid.NewString()
		require.NoError(t, db.Create(&organizationResourceOperationRow{
			OrganizationID: org, OperationID: op, OperationType: "fixture",
			RequestFingerprint: "fixture", State: "succeeded", ImmutableResult: "{}",
		}).Error)
		require.NoError(t, db.Create(&organizationResourceEventRow{
			QuantityUnit: resourceQuantityUnit(string(orgresource.ResourceAIPoint)), EventID: uuid.NewString(),
			OrganizationID: org, OperationID: op, ResourceType: string(orgresource.ResourceAIPoint),
			Quantity: 8, ConsumedDelta: consumed, Reason: "fixture", SourceType: "fixture", SourceIdentity: op,
			CreatedAt: at.Add(time.Duration(seconds) * time.Second),
		}).Error)
	}
	add("org-a", 0, 2)
	add("org-a", 1, -1)
	add("org-a", 2, 1)
	add("org-a", 3, 0)
	add("org-b", 4, 7)

	query := orgresource.EventQuery{OrganizationID: "org-a", ResourceType: orgresource.ResourceAIPoint, Limit: 1, DeductionsOnly: true}
	first, err := repo.ListEvents(ctx, query)
	require.NoError(t, err)
	require.Len(t, first.Items, 1)
	require.Equal(t, "0.000001", first.Items[0].ConsumedDelta)
	require.NotNil(t, first.NextCursor)

	query.Cursor = *first.NextCursor
	second, err := repo.ListEvents(ctx, query)
	require.NoError(t, err)
	require.Len(t, second.Items, 1)
	require.Equal(t, "0.000002", second.Items[0].ConsumedDelta)
	require.Nil(t, second.NextCursor)

	query.DeductionsOnly = false
	_, err = repo.ListEvents(ctx, query)
	require.ErrorIs(t, err, orgresource.ErrInvalidInput)
	query.Cursor = ""
	all, err := repo.ListEvents(ctx, query)
	require.NoError(t, err)
	require.Len(t, all.Items, 1)
	require.Equal(t, "0", all.Items[0].ConsumedDelta)
	require.NotNil(t, all.NextCursor)
	query.Cursor = *all.NextCursor
	query.DeductionsOnly = true
	_, err = repo.ListEvents(ctx, query)
	require.ErrorIs(t, err, orgresource.ErrInvalidInput)
}
