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
