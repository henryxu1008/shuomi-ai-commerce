//go:build integration

package storagecapacitystore

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
	"time"
)

func TestStorageCapacityPostgresConcurrentAdmissionAndAbortFence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	c, e := pg.Run(ctx, "postgres:16-alpine", pg.WithDatabase("storage_capacity"), pg.WithUsername("owner"), pg.WithPassword(uuid.NewString()), pg.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, e := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, e := db.DB()
	require.NoError(t, e)
	pool.SetMaxOpenConns(8)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, AutoMigrate(db))
	repo, e := New(db)
	require.NoError(t, e)
	var wg sync.WaitGroup
	result := make(chan error, 12)
	for i := range 12 {
		wg.Go(func() {
			_, e := repo.ReserveWrite(ctx, reserveInput("org", fmt.Sprintf("concurrent-%d", i), storagecapacity.GiB))
			result <- e
		})
	}
	wg.Wait()
	close(result)
	accepted := 0
	for e := range result {
		if e == nil {
			accepted++
		} else {
			require.ErrorIs(t, e, storagecapacity.ErrExceeded)
		}
	}
	require.Equal(t, 5, accepted)
	s, e := repo.ReadSummary(ctx, "org")
	require.NoError(t, e)
	require.Equal(t, "5368709120", s.ReservedBytes)
	require.Equal(t, "0", s.RemainingBytes)
	r, e := repo.ReserveWrite(ctx, reserveInput("other", "fence", 1))
	require.NoError(t, e)
	race := make(chan error, 2)
	wg.Go(func() { race <- repo.ClaimDispatch(ctx, "other", r.ID) })
	wg.Go(func() { race <- repo.AbortUnstarted(ctx, "other", r.ID) })
	wg.Wait()
	close(race)
	ok := 0
	for e := range race {
		if e == nil {
			ok++
		} else {
			require.True(t, errors.Is(e, storagecapacity.ErrConflict) || errors.Is(e, storagecapacity.ErrUnknown))
		}
	}
	require.Equal(t, 1, ok)
}
