package storecenter

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBrowserSchemaRejectsBothInvalidRelationsRatherThanTreatingMissing(t *testing.T) {
	for _, tc := range []struct {
		count            int
		shape, wantError bool
	}{{0, false, false}, {1, true, true}, {2, false, true}} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		mock.ExpectQuery(`SELECT count\(\*\),coalesce`).WillReturnRows(sqlmock.NewRows([]string{"count", "shape"}).AddRow(tc.count, tc.shape))
		installed, err := VerifyBrowserSchemaIfPresent(context.Background(), db)
		require.False(t, installed)
		require.Equal(t, tc.wantError, err != nil)
		require.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	}
}
func TestBrowserSchemaOwnerVerifiesRuntimeGrantsOnReadback(t *testing.T) {
	for _, valid := range []bool{true, false} {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		mock.ExpectQuery(`SELECT bool_and`).WillReturnRows(sqlmock.NewRows([]string{"valid"}).AddRow(valid))
		err = VerifyBrowserRuntimeGrants(context.Background(), db)
		require.Equal(t, !valid, err != nil)
		require.NoError(t, mock.ExpectationsWereMet())
		db.Close()
	}
}
