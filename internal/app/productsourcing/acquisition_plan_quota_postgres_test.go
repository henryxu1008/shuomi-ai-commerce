package productsourcing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	planstore "task-processor/internal/integration/persistence/product/acquisitionplan"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/sourcing"
)

type quotaPlanProvider struct{ calls int }

func (p *quotaPlanProvider) Acquire(_ context.Context, source sourcing.AcquisitionSource) (sourcing.AcquisitionEvidence, error) {
	p.calls++
	evidence := browserEvidence()
	evidence.OfferID, evidence.SourceURL = source.OfferID, source.URL
	return evidence, nil
}

func TestPlanActualQuotaRecoveryUsesOriginalChildAndChargesOnce(t *testing.T) {
	for _, debt := range []bool{false, true} {
		name := "member-balance"
		if debt {
			name = "outstanding-debt"
		}
		t.Run(name, func(t *testing.T) {
			db := acquisitionDatabase(t)
			require.NoError(t, InstallAcquisitionSchema(db))
			require.NoError(t, collectionstore.InstallSchema(db))
			require.NoError(t, planstore.InstallSchema(db))
			charges, positions, resourceDB := acquisitionRecoveryResource(t, db)
			ctx := acquisitionIdentity("org-charge", "actor")
			transfer := func(action orgresource.MemberResourceAction, quantity int64) {
				position, err := positions.ReadPosition(ctx, "org-charge", "membership:actor", orgresource.ResourceDataRow)
				require.NoError(t, err)
				_, err = positions.Transfer(ctx, orgresource.MemberResourceTransfer{OrganizationID: "org-charge", MemberID: "membership:actor", ActorID: "admin", OperationID: uuid.NewString(), ResourceType: orgresource.ResourceDataRow, Action: action, Quantity: quantity, ExpectedVersion: position.Version})
				require.NoError(t, err)
			}
			inputs := []string{"981645030344", "981645030345"}
			if debt {
				inputs = inputs[:1]
				require.NoError(t, resourceDB.Exec(`INSERT INTO saas_organization_resource_debts (organization_id,resource_type,amount,created_at,updated_at) VALUES ('org-charge','data_row',1,?,?)`, time.Now().UTC(), time.Now().UTC()).Error)
			} else {
				transfer(orgresource.MemberResourceReclaim, 2)
			}
			provider := &quotaPlanProvider{}
			permissions, live := acquisitionPermissionDependencies(t)
			core, err := NewPublicAcquisition(ctx, db, live, permissions, provider, charges, WithCollectionPublication())
			require.NoError(t, err)
			child, err := NewBrowserAcquisitionService(core.operations, provider, core.publisher, core.reader, core.authorizer, 0, core.charges)
			require.NoError(t, err)
			repo, err := planstore.NewRepository(ctx, db)
			require.NoError(t, err)
			scope := sourcing.AcquisitionPlanScope{OrganizationID: "org-charge", ActorID: "actor", MemberID: "membership:actor"}
			service, err := NewAcquisitionPlanService(repo, nil, child, nil, nil, planTestAuth{scope}, nil)
			require.NoError(t, err)
			plan, err := service.Create(ctx, uuid.NewString(), CreatePlanInput{Mode: "product", Inputs: inputs})
			require.NoError(t, err)
			originalID, key := plan.ID, plan.Items[0].ChildKey
			plan, err = service.Advance(ctx, plan.ID, plan.Revision)
			require.NoError(t, err)
			require.Equal(t, "paused", plan.State)
			require.Equal(t, "capacity_or_quota_unavailable", plan.Items[0].FailureCode)
			require.Zero(t, provider.calls)
			original, err := core.operations.ByKey(ctx, sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, key)
			require.NoError(t, err)
			var reservations int64
			require.NoError(t, resourceDB.Table("saas_organization_resource_reservations").Count(&reservations).Error)
			require.Zero(t, reservations)
			if debt {
				transfer(orgresource.MemberResourceReclaim, 1)
			} else {
				transfer(orgresource.MemberResourceAllocate, 2)
			}
			// Advance the existing lease in this isolated fixture; no new claim or reset.
			require.NoError(t, db.Exec("UPDATE public.product_acquisition_operations SET lease_until=clock_timestamp()-interval '1 second' WHERE operation_id=?", original.ID).Error)
			for range inputs {
				plan, err = service.Advance(ctx, plan.ID, plan.Revision)
				require.NoError(t, err)
			}
			require.Equal(t, "completed", plan.State, "real original acquisition must resume after quota is restored")
			require.Equal(t, originalID, plan.ID)
			require.Equal(t, key, plan.Items[0].ChildKey)
			require.Equal(t, original.ID, plan.Items[0].OperationID)
			require.Equal(t, len(inputs), provider.calls)
			_, err = service.Advance(ctx, plan.ID, plan.Revision)
			require.NoError(t, err)
			_, err = child.Acquire(ctx, key, inputs[0])
			require.NoError(t, err)
			require.Equal(t, len(inputs), provider.calls, "plan/child replay must not dispatch")
			position, err := positions.ReadPosition(ctx, "org-charge", "membership:actor", orgresource.ResourceDataRow)
			require.NoError(t, err)
			require.EqualValues(t, len(inputs), position.Consumed)
			require.Zero(t, position.Reserved)
			for _, table := range []string{"product_acquisition_operations", "product_acquisition_charge_intents", "product_source_publication_receipts", "product_collection_items"} {
				var count int64
				require.NoError(t, db.Table(table).Count(&count).Error)
				require.EqualValues(t, len(inputs), count, table)
			}
			require.NoError(t, resourceDB.Table("saas_organization_resource_reservations").Count(&reservations).Error)
			require.EqualValues(t, len(inputs), reservations)
		})
	}
}
