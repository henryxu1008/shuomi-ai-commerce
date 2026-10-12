package storagecapacitystore

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
)

func TestNamespaceMustBeEmptyBeforeFirstMeteredWriteAdmission(t *testing.T) {
	r, _ := storageFixture(t)
	ctx := context.Background()
	namespace := storagecapacity.Digest("physical-bucket")
	called := 0
	require.Error(t, r.RequireNamespace(ctx, namespace, "knowledge/"))
	require.ErrorIs(t, r.EnsureNamespace(ctx, namespace, "knowledge/", func(context.Context) error { called++; return storagecapacity.ErrConflict }), storagecapacity.ErrConflict)
	require.Error(t, r.RequireNamespace(ctx, namespace, "knowledge/"))
	require.NoError(t, r.EnsureNamespace(ctx, namespace, "knowledge/", func(context.Context) error { called++; return nil }))
	require.NoError(t, r.RequireNamespace(ctx, namespace, "knowledge/"))
	require.NoError(t, r.EnsureNamespace(ctx, namespace, "knowledge/", func(context.Context) error {
		t.Fatal("restart must use the established namespace receipt, not require stored objects to be empty")
		return nil
	}))
	require.Equal(t, 2, called)
	s, e := r.ReadSummary(ctx, "fresh-org")
	require.NoError(t, e)
	require.Equal(t, "5368709120", s.QuotaBytes)
	var accounts int64
	require.NoError(t, r.db.Model(&accountRow{}).Count(&accounts).Error)
	require.Zero(t, accounts, "summary reads must not create a capacity account")
}

func TestDisabledGuardPreservesEveryHistoricalCapacityReceipt(t *testing.T) {
	for _, receipt := range []struct {
		name string
		row  any
	}{
		{"namespace", &namespaceRow{Identity: storagecapacity.Digest("identity"), Namespace: storagecapacity.Digest("namespace"), Prefix: "knowledge/"}},
		{"empty_account", &accountRow{OrganizationID: "org"}},
		{"closed_reservation", &reservationRow{ID: "reservation", OrganizationID: "org", Owner: "knowledge", OperationID: "original-command", Fingerprint: storagecapacity.Digest("original-command"), State: "ABORTED"}},
		{"object", &objectRow{Identity: storagecapacity.Digest("object"), OrganizationID: "org", Namespace: storagecapacity.Digest("namespace"), Key: "original", SHA256: storagecapacity.Digest("content"), SizeBytes: 4, Category: "knowledge_document"}},
		{"binding", &bindingRow{ReservationID: "reservation", ObjectIdentity: storagecapacity.Digest("object")}},
		{"prepared_lot", &lotRow{OrderID: "order", ItemID: "item", OrganizationID: "org", TermsFingerprint: storagecapacity.Digest("terms"), Bytes: storagecapacity.PackBytes, DurationSeconds: storagecapacity.PeriodSeconds}},
	} {
		t.Run(receipt.name, func(t *testing.T) {
			r, _ := storageFixture(t)
			require.NoError(t, VerifyDisabled(context.Background(), r.db))
			require.NoError(t, r.db.Create(receipt.row).Error)
			require.Error(t, VerifyDisabled(context.Background(), r.db))
			var count int64
			require.NoError(t, r.db.Model(receipt.row).Count(&count).Error)
			require.Equal(t, int64(1), count, "startup rejection must preserve the original receipt")
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, _ := storageFixture(t)
	require.Error(t, VerifyDisabled(ctx, r.db))
	require.Error(t, VerifyDisabled(context.Background(), nil))
}
