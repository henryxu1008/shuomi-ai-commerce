package productsourcing

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	planstore "task-processor/internal/integration/persistence/product/acquisitionplan"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/sourcing"
)

type capacityAcquisitionProvider struct {
	reject bool
	calls  int
}

func TestCollectorCapacityCommitLossDoesNotGrantUnknownDispatch(t *testing.T) {
	for _, stage := range []string{"rejection", "reclaim"} {
		t.Run(stage, func(t *testing.T) {
			owner := acquisitionDatabase(t)
			require.NoError(t, InstallAcquisitionSchema(owner))
			require.NoError(t, collectionstore.InstallSchema(owner))
			charges, _, _ := acquisitionRecoveryResource(t, owner)
			provider := &capacityAcquisitionProvider{reject: true}
			permissions, live := acquisitionPermissionDependencies(t)
			build := func(db *gorm.DB) *AcquisitionService {
				core, err := NewPublicAcquisition(context.Background(), db, live, permissions, provider, charges, WithCollectionPublication())
				require.NoError(t, err)
				return core
			}
			core := build(owner)
			ctx, key := acquisitionIdentity("org-charge", "actor"), uuid.NewString()
			if stage == "reclaim" {
				_, err := core.Acquire(ctx, key, "981645030344")
				require.ErrorIs(t, err, sourcing.ErrAcquisitionCapacity)
				provider.reject = false
			}
			// Reuse the existing real-COMMIT acknowledgement-loss driver. Both
			// marker and reclaim write failure_code; no new fault platform.
			var name string
			require.NoError(t, owner.Raw("SELECT current_database()").Scan(&name).Error)
			fault := &acquisitionAckDriver{target: "finish"}
			driverName := "capacity_ack_" + uuid.NewString()
			sql.Register(driverName, fault)
			pool, err := sql.Open(driverName, os.Getenv("ISSUE398_TEST_DSN")+" dbname="+name)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, pool.Close()) })
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
			require.NoError(t, err)
			_, err = build(db).Acquire(ctx, key, "981645030344")
			require.ErrorIs(t, err, sourcing.ErrAcquisitionUnknown)
			require.True(t, fault.lost.Load())
			require.Equal(t, 1, provider.calls, "lost reclaim acknowledgement grants no provider call")
			rebuilt := build(owner)
			op, err := rebuilt.operations.ByKey(ctx, sourcing.PublicationScope{OrganizationID: "org-charge", ActorID: "actor"}, key)
			require.NoError(t, err)
			require.Equal(t, sourcing.AcquisitionAcquiring, op.State)
			receipt, err := charges.(orgresource.ConsumerChargeLookupPort).Lookup(ctx, orgresource.ConsumerChargeIdentity{OrganizationID: "org-charge", Consumer: orgresource.ConsumerProductAcquisition, OperationID: op.ID})
			require.NoError(t, err)
			require.Equal(t, orgresource.ReservationReserved, receipt.State)
			provider.reject = false
			if stage == "rejection" {
				// A plan whose response was lost reads the existing owner marker
				// through recoverChild, then obtains authority from original Acquire.
				browser, err := NewBrowserAcquisitionService(rebuilt.operations, provider, rebuilt.publisher, rebuilt.reader, rebuilt.authorizer, 0, rebuilt.charges)
				require.NoError(t, err)
				service := &AcquisitionPlanService{child: browser}
				result, err := service.recoverChild(ctx, sourcing.AcquisitionPlanScope{OrganizationID: "org-charge", ActorID: "actor", MemberID: "membership:actor"}, sourcing.AcquisitionPlanItem{Source: op.Source.URL, ChildKey: key, State: "needs_verification"})
				require.NoError(t, err)
				require.Equal(t, op.ID, result.Operation.ID)
				require.Equal(t, sourcing.AcquisitionPublished, result.Operation.State)
				require.Equal(t, 2, provider.calls)
			} else {
				require.Empty(t, op.FailureCode)
				require.NoError(t, owner.Exec("UPDATE product_acquisition_operations SET lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=?", op.ID).Error)
				_, err = rebuilt.Acquire(ctx, key, "981645030344")
				require.ErrorIs(t, err, sourcing.ErrAcquisitionUnknown)
				require.Equal(t, 1, provider.calls, "a bound claim with lost response cannot be repeated")
			}
		})
	}
}

func (p *capacityAcquisitionProvider) Acquire(_ context.Context, source sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error) {
	p.calls++
	if p.reject {
		return sourcing.AcquisitionEvidence{}, sourcing.ErrAcquisitionCapacity
	}
	evidence := browserEvidence()
	evidence.SourceURL, evidence.OfferID = source.URL, source.OfferID
	return evidence, nil
}

func TestAcquisitionCollectorCapacityResumesOriginalReservation(t *testing.T) {
	for _, mode := range []string{"http", "browser", "plan"} {
		t.Run(mode, func(t *testing.T) {
			db := acquisitionDatabase(t)
			require.NoError(t, InstallAcquisitionSchema(db))
			require.NoError(t, collectionstore.InstallSchema(db))
			charges, positions, resourceDB := acquisitionRecoveryResource(t, db)
			ctx := acquisitionIdentity("org-charge", "actor")
			provider := &capacityAcquisitionProvider{reject: true}
			permissions, live := acquisitionPermissionDependencies(t)
			revoked := false
			currentLive := liveRolesFunc(func(ctx context.Context, org, actor string) ([]string, error) {
				if revoked {
					return nil, nil
				}
				return live.ResolveLiveRoles(ctx, org, actor)
			})
			core, err := NewPublicAcquisition(ctx, db, currentLive, permissions, provider, charges, WithCollectionPublication())
			require.NoError(t, err)
			browser, err := NewBrowserAcquisitionService(core.operations, provider, core.publisher, core.reader, core.authorizer, 0, core.charges)
			require.NoError(t, err)
			acquire := core.Acquire
			if mode != "http" {
				acquire = browser.Acquire
			}
			key, source := uuid.NewString(), "981645030344"
			var plan sourcing.AcquisitionPlan
			var service *AcquisitionPlanService
			if mode == "plan" {
				require.NoError(t, planstore.InstallSchema(db))
				repo, err := planstore.NewRepository(ctx, db)
				require.NoError(t, err)
				scope := sourcing.AcquisitionPlanScope{OrganizationID: "org-charge", ActorID: "actor", MemberID: "membership:actor"}
				service, err = NewAcquisitionPlanService(repo, nil, browser, nil, nil, planTestAuth{scope}, nil)
				require.NoError(t, err)
				plan, err = service.Create(ctx, uuid.NewString(), CreatePlanInput{Mode: "product", Inputs: []string{source, "981645030345"}})
				require.NoError(t, err)
				key = plan.Items[0].ChildKey
				plan, err = service.Advance(ctx, plan.ID, plan.Revision)
				require.NoError(t, err)
				require.Equal(t, "paused", plan.State)
			} else {
				_, err = acquire(ctx, key, source)
				require.ErrorIs(t, err, sourcing.ErrAcquisitionCapacity)
			}
			op, err := core.operations.ByKey(ctx, sourcing.PublicationScope{OrganizationID: "org-charge", ActorID: "actor"}, key)
			require.NoError(t, err)
			require.Equal(t, sourcing.AcquisitionAcquiring, op.State, "collector contention must not become a terminal source failure")
			require.Equal(t, "COLLECTOR_CAPACITY_REJECTED", op.FailureCode)
			receipt, err := charges.(orgresource.ConsumerChargeLookupPort).Lookup(ctx, orgresource.ConsumerChargeIdentity{OrganizationID: "org-charge", Consumer: orgresource.ConsumerProductAcquisition, OperationID: op.ID})
			require.NoError(t, err)
			require.Equal(t, orgresource.ReservationReserved, receipt.State)
			revoked = true
			_, err = acquire(ctx, key, source)
			require.ErrorIs(t, err, sourcing.ErrPublicationForbidden)
			require.Equal(t, 1, provider.calls, "revoked authorization cannot reclaim or dispatch the marked original")
			revoked = false
			old := op
			old.Fence--
			old.FailureCode = ""
			require.ErrorIs(t, core.operations.Finish(ctx, old, sourcing.AcquisitionFailed, "SOURCE_UNAVAILABLE"), sourcing.ErrAcquisitionFence)
			provider.reject = false
			products := 1
			if mode == "plan" {
				products = 2
				for range products {
					plan, err = service.Advance(ctx, plan.ID, plan.Revision)
					require.NoError(t, err)
				}
				require.Equal(t, "completed", plan.State)
				require.Equal(t, key, plan.Items[0].ChildKey)
				require.Equal(t, op.ID, plan.Items[0].OperationID)
			} else {
				result, err := acquire(ctx, key, source)
				require.NoError(t, err)
				require.Equal(t, op.ID, result.Operation.ID)
				require.Equal(t, sourcing.AcquisitionPublished, result.Operation.State)
			}
			_, err = acquire(ctx, key, source)
			require.NoError(t, err)
			require.Equal(t, products+1, provider.calls, "one capacity rejection, then one fetch per published product; replay performs no fetch")
			after, err := charges.(orgresource.ConsumerChargeLookupPort).Lookup(ctx, receipt.Intent.Identity)
			require.NoError(t, err)
			require.Equal(t, receipt.ReservationID, after.ReservationID)
			require.Equal(t, orgresource.ReservationCommitted, after.State)
			position, err := positions.ReadPosition(ctx, "org-charge", "membership:actor", orgresource.ResourceDataRow)
			require.NoError(t, err)
			require.EqualValues(t, products, position.Consumed)
			require.Zero(t, position.Reserved)
			var reservations int64
			require.NoError(t, resourceDB.Table("saas_organization_resource_reservations").Count(&reservations).Error)
			require.EqualValues(t, products, reservations)
		})
	}
}
