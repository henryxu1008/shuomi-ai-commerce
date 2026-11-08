package orgresourceadapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/aicapability"
)

func rejectionPostgresDatabases(t *testing.T) (*gorm.DB, *gorm.DB) {
	t.Helper()
	dsn := os.Getenv("ISSUE398_TEST_DSN")
	if dsn == "" {
		t.Skip("requires task-owned loopback ISSUE398_TEST_DSN")
	}
	require.Contains(t, dsn, "host=127.0.0.1")
	require.Contains(t, dsn, "user=issue398_owner")
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	rootPool, err := root.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, rootPool.Close()) })
	var databases []*gorm.DB
	for range 2 {
		name := "unbilled_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		quoted := pgx.Identifier{name}.Sanitize()
		require.NoError(t, root.Exec("CREATE DATABASE "+quoted).Error)
		db, err := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		pool.SetMaxOpenConns(8)
		t.Cleanup(func() {
			require.NoError(t, pool.Close())
			require.NoError(t, root.Exec("DROP DATABASE "+quoted+" WITH (FORCE)").Error)
		})
		databases = append(databases, db)
	}
	return databases[0], databases[1]
}

type rejectionSettlementFailure struct {
	*GormModelInvocationRepository
	afterCommit bool
}

func (f rejectionSettlementFailure) SettleAIInvocationUsage(ctx context.Context, org, member, id string, total int64, at time.Time) error {
	if f.afterCommit {
		if err := f.GormModelInvocationRepository.SettleAIInvocationUsage(ctx, org, member, id, total, at); err != nil {
			return err
		}
	}
	return errors.New("controlled settlement acknowledgement loss")
}

func TestProviderRejectionPostgresConcurrentRecovery(t *testing.T) {
	resource, native := rejectionPostgresDatabases(t)
	recorder, repo, fact := rejectedInvocationFixture(t, resource, native)
	ctx := context.Background()
	start := make(chan struct{})
	results := make(chan error, 16)
	var group sync.WaitGroup
	for n := range 16 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			if n%4 == 0 {
				_, err := repo.RecoverDue(ctx)
				results <- err
				return
			}
			results <- recorder.ResolveDispatchedInvocation(ctx, fact)
		}()
	}
	close(start)
	group.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	_, err := repo.RecoverDue(ctx)
	require.NoError(t, err)
	assertRejectedReservationSettledOnce(t, resource)
	require.NoError(t, repo.ConfirmModelInvocationSettlement(ctx, "org", fact.InvocationID, fact.AgentRunID))
}

func TestProviderRejectionPostgresOriginalBindingAndLostAcknowledgement(t *testing.T) {
	for _, after := range []bool{false, true} {
		t.Run(fmt.Sprint("commercial commit ", after), func(t *testing.T) {
			resource, native := rejectionPostgresDatabases(t)
			recorder, repo, fact := rejectedInvocationFixture(t, resource, native)
			ctx := context.Background()
			wrong := fact
			copyProof := *fact.ProviderRejectionProof
			wrong.ProviderRejectionProof = &copyProof
			wrong.ProviderRejectionProof.ReservationID = "another-reservation"
			wrong.ProviderRejectionProof.Digest = aicapability.ProviderRejectionDigest(*wrong.ProviderRejectionProof)
			require.Error(t, recorder.ResolveDispatchedInvocation(ctx, wrong))
			original, found, err := recorder.FindInvocation(ctx, "org", "member", fact.InvocationID, "input")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, aicapability.InvocationDispatched, original.Outcome)
			recorder.SetUsageSettler(rejectionSettlementFailure{GormModelInvocationRepository: repo, afterCommit: after})
			require.Error(t, recorder.ResolveDispatchedInvocation(ctx, fact))
			stored, found, err := recorder.FindInvocation(ctx, "org", "member", fact.InvocationID, "input")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, aicapability.InvocationFailed, stored.Outcome)
			recorder.SetUsageSettler(repo)
			_, err = repo.RecoverDue(ctx)
			require.NoError(t, err)
			require.NoError(t, recorder.ResolveDispatchedInvocation(ctx, fact))
			assertRejectedReservationSettledOnce(t, resource)
		})
	}
}

func TestProviderRejectionPostgresCanonicalTimestampReplay(t *testing.T) {
	resource, native := rejectionPostgresDatabases(t)
	recorder, _, fact := rejectedInvocationFixture(t, resource, native)
	p := fact.ProviderRejectionProof
	p.Evidence.RejectedAt = p.Evidence.RejectedAt.Add(999 * time.Nanosecond).In(time.FixedZone("receipt", 8*60*60))
	p.ObservedAt = p.ObservedAt.In(time.FixedZone("operator", -5*60*60))
	p.Digest = aicapability.ProviderRejectionDigest(*p)
	fact.FinishedAt = p.Evidence.RejectedAt
	for range 2 {
		require.NoError(t, recorder.ResolveDispatchedInvocation(context.Background(), fact))
	}
	assertRejectedReservationSettledOnce(t, resource)
}

func TestProviderRejectionPostgresDirectRecordAndInvocationCommitAcknowledgementLoss(t *testing.T) {
	for _, loseAck := range []bool{false, true} {
		t.Run(fmt.Sprint("invocation commit ack lost ", loseAck), func(t *testing.T) {
			resource, native := rejectionPostgresDatabases(t)
			recorder, repo, fact := rejectedInvocationFixture(t, resource, native)
			var lost atomic.Bool
			if loseAck {
				require.NoError(t, native.Callback().Update().After("gorm:commit_or_rollback_transaction").Register("test:invocation_ack_loss", func(tx *gorm.DB) {
					if tx.Error == nil && tx.Statement.Table == "ai_invocations" && lost.CompareAndSwap(false, true) {
						tx.AddError(errors.New("controlled invocation commit acknowledgement loss"))
					}
				}))
			}
			err := recorder.RecordInvocation(context.Background(), fact)
			if loseAck {
				require.Error(t, err)
				require.True(t, lost.Load())
			} else {
				require.NoError(t, err)
			}
			stored, found, err := recorder.FindInvocation(context.Background(), "org", "member", fact.InvocationID, "input")
			require.NoError(t, err)
			require.True(t, found)
			require.Equal(t, fact.ProviderRejectionProof.Digest, stored.ProviderRejectionProof.Digest)
			_, err = repo.RecoverDue(context.Background())
			require.NoError(t, err)
			require.NoError(t, recorder.RecordInvocation(context.Background(), fact))
			assertRejectedReservationSettledOnce(t, resource)
		})
	}
}
