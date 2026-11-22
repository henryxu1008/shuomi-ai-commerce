package sourcingpersistence

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTransactionWriterCannotOwnCommitOrAcceptRootPool(t *testing.T) {
	pool, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer pool.Close()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	_, err = NewTransactionWriter(db, testCatalogBridgeFactory)
	require.ErrorIs(t, err, sourcing.ErrSourcePublicationUnavailable)
	mock.ExpectBegin()
	tx := db.Begin()
	require.NoError(t, tx.Error)
	store, err := NewTransactionWriter(tx, testCatalogBridgeFactory)
	require.NoError(t, err)
	actual := store.(*repository)
	called := false
	require.NoError(t, actual.writeTransaction(context.Background(), func(got *gorm.DB) error {
		called = true
		require.Equal(t, tx.Statement.ConnPool, got.Statement.ConnPool)
		return nil
	}))
	require.True(t, called)
	mock.ExpectRollback()
	require.NoError(t, tx.Rollback().Error)
	require.NoError(t, mock.ExpectationsWereMet())
}
