package productsourcing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

func TestPlanQuotaPauseKeepsOriginalChildAndDoesNotProcessLaterProducts(t *testing.T) {
	for _, quotaError := range []error{orgresource.ErrInsufficientBalance, orgresource.ErrResourceDebtOutstanding, errors.Join(errors.New("wrapped admission"), orgresource.ErrResourceDebtOutstanding)} {
		t.Run(quotaError.Error(), func(t *testing.T) {
			scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
			input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123", "456"}, sourcing.AcquisitionRule{}, false)
			id := uuid.NewString()
			key := planChildKey(id, input.Inputs[0])
			r := &planMemory{plan: sourcing.AcquisitionPlan{ID: id, Scope: scope, Input: input, DiscoveryIndex: 2, Revision: 1, State: "pending", Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], State: "pending"}, {Source: input.Inputs[1], State: "pending"}}}}
			child := &planChild{err: quotaError}
			service, err := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
			require.NoError(t, err)
			for attempt := 0; attempt < 3; attempt++ {
				p, err := service.Advance(t.Context(), id, r.plan.Revision)
				require.NoError(t, err)
				require.Equal(t, "paused", p.State)
				require.Equal(t, "capacity_or_quota_unavailable", p.Items[0].FailureCode)
				require.Equal(t, "pending", p.Items[0].State)
				require.Equal(t, key, p.Items[0].ChildKey)
				require.Equal(t, "pending", p.Items[1].State)
				require.Empty(t, p.Items[1].ChildKey)
			}
			require.Equal(t, []string{key, key, key}, child.acquireKeys)
			require.Empty(t, child.verifyKeys)
		})
	}
}

func TestPlanRecoveredFailedQuotaChildRemainsPaused(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123", "456"}, sourcing.AcquisitionRule{}, false)
	id := uuid.NewString()
	key := planChildKey(id, input.Inputs[0])
	source, _ := sourcing.Canonical1688Source(input.Inputs[0])
	op := acquisitionOperation(sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, key, source)
	op.State, op.FailureCode = sourcing.AcquisitionFailed, "DATA_QUOTA_INSUFFICIENT"
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: id, Scope: scope, Input: input, DiscoveryIndex: 2, Revision: 1, State: "paused", Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], ChildKey: key, State: "needs_verification"}, {Source: input.Inputs[1], State: "pending"}}}}
	child := &planChild{readResult: sourcing.AcquisitionResult{Operation: op}, err: orgresource.ErrInsufficientBalance}
	service, err := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
	require.NoError(t, err)
	for attempt := 0; attempt < 3; attempt++ {
		p, err := service.Advance(t.Context(), id, r.plan.Revision)
		require.NoError(t, err)
		require.Equal(t, "paused", p.State)
		require.Equal(t, "capacity_or_quota_unavailable", p.Items[0].FailureCode)
		require.Equal(t, "pending", p.Items[0].State)
		require.Equal(t, op.ID, p.Items[0].OperationID)
		require.Equal(t, key, p.Items[0].ChildKey)
		require.Empty(t, p.Items[1].ChildKey)
	}
	require.Equal(t, []string{op.ID}, child.readIDs)
	require.Equal(t, []string{key, key}, child.acquireKeys)
	require.Empty(t, child.verifyKeys)
}

func TestPlanRecoveredSourceFailureStillAllowsNextProduct(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123", "456"}, sourcing.AcquisitionRule{}, false)
	id := uuid.NewString()
	key := planChildKey(id, input.Inputs[0])
	source, _ := sourcing.Canonical1688Source(input.Inputs[0])
	op := acquisitionOperation(sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, key, source)
	op.State, op.FailureCode = sourcing.AcquisitionFailed, "SOURCE_UNAVAILABLE"
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: id, Scope: scope, Input: input, DiscoveryIndex: 2, Revision: 1, State: "paused", Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], ChildKey: key, State: "needs_verification"}, {Source: input.Inputs[1], State: "pending"}}}}
	child := &planChild{readResult: sourcing.AcquisitionResult{Operation: op}, err: sourcing.ErrAcquisitionUnknown}
	service, err := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
	require.NoError(t, err)
	p, err := service.Advance(t.Context(), id, r.plan.Revision)
	require.NoError(t, err)
	require.Equal(t, "failed", p.Items[0].State)
	require.Equal(t, "source_unavailable", p.Items[0].FailureCode)
	_, err = service.Advance(t.Context(), id, p.Revision)
	require.NoError(t, err)
	require.Equal(t, []string{planChildKey(id, input.Inputs[1])}, child.acquireKeys)
}

type conflictPlanPreparations struct {
	err   error
	calls int
}

func (p *conflictPlanPreparations) ReadByKey(context.Context, string) (preparation.TransferReceipt, error) {
	return preparation.TransferReceipt{}, preparation.ErrNotFound
}
func (p *conflictPlanPreparations) Transfer(context.Context, string, preparation.TransferInput) (preparation.TransferReceipt, error) {
	p.calls++
	return preparation.TransferReceipt{}, p.err
}

func TestPlanStopsKnownFrozenTransferConflictWithoutUnknownRecovery(t *testing.T) {
	for _, conflict := range []error{preparation.ErrConflict, collection.ErrConflict, collection.ErrNotFound} {
		t.Run(conflict.Error(), func(t *testing.T) {
			scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
			input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123"}, sourcing.AcquisitionRule{}, true)
			planID, batchID, itemID := uuid.NewString(), uuid.NewString(), uuid.NewString()
			frozen, _ := json.Marshal([]planTransferIntent{{Key: collection.StableID(planID, batchID, "transfer"), Input: preparation.TransferInput{BatchID: batchID, ExpectedRevision: 3, ItemIDs: []string{itemID}}}})
			r := &planMemory{plan: sourcing.AcquisitionPlan{ID: planID, Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: 1, Revision: 1, State: "transfer_pending", TransferKey: collection.StableID(planID, "transfer"), TransferInputJSON: frozen, Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], State: "published"}}}}
			preps := &conflictPlanPreparations{err: errors.Join(errors.New("source changed"), conflict)}
			service, err := NewAcquisitionPlanService(r, nil, &planChild{}, nil, preps, planTestAuth{scope}, nil)
			require.NoError(t, err)
			p, err := service.Advance(t.Context(), planID, 1)
			require.NoError(t, err)
			require.Equal(t, "transfer_source_conflict", p.FailureCode)
			require.Equal(t, "transfer_pending", p.State)
			require.Equal(t, frozen, p.TransferInputJSON)
			require.Empty(t, p.PreparationIDs)
			_, err = service.Advance(t.Context(), planID, p.Revision)
			require.NoError(t, err)
			require.Equal(t, 1, preps.calls, "a definite stale intent cannot acquire a future matching receipt")
		})
	}
}

func TestPlanRecoveryUsesOriginalOwnerByDurableState(t *testing.T) {
	for _, tc := range []struct {
		name            string
		state           string
		readErr         error
		acquire, verify int
		foreign         bool
	}{
		{name: "acquiring", state: sourcing.AcquisitionAcquiring, acquire: 1},
		{name: "prepared", state: sourcing.AcquisitionPrepared, acquire: 1},
		{name: "publishing", state: sourcing.AcquisitionPublishing, verify: 1},
		{name: "published charge recovery", state: sourcing.AcquisitionPublished, verify: 1},
		{name: "not found", readErr: sourcing.ErrAcquisitionNotFound, acquire: 1},
		{name: "read unavailable", readErr: sourcing.ErrAcquisitionUnavailable},
		{name: "foreign operation", state: sourcing.AcquisitionPrepared, foreign: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
			input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123"}, sourcing.AcquisitionRule{}, false)
			id := uuid.NewString()
			key := planChildKey(id, input.Inputs[0])
			source, _ := sourcing.Canonical1688Source(input.Inputs[0])
			op := acquisitionOperation(sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, key, source)
			op.State = tc.state
			if tc.foreign {
				op.Scope.ActorID = "other"
			}
			r := &planMemory{plan: sourcing.AcquisitionPlan{ID: id, Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: 1, Revision: 1, State: "paused", Items: []sourcing.AcquisitionPlanItem{{Source: source.URL, ChildKey: key, State: "needs_verification"}}}}
			child := &planChild{readResult: sourcing.AcquisitionResult{Operation: op}, readErr: tc.readErr, err: sourcing.ErrAcquisitionUnknown}
			service, e := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
			require.NoError(t, e)
			p, e := service.Advance(context.Background(), id, 1)
			require.NoError(t, e)
			require.Equal(t, tc.acquire, child.acquire)
			require.Equal(t, tc.verify, child.verify)
			require.Equal(t, []string{op.ID}, child.readIDs)
			for _, actual := range append(child.acquireKeys, child.verifyKeys...) {
				require.Equal(t, key, actual)
			}
			require.Equal(t, "needs_verification", p.Items[0].State)
		})
	}
}

type planChargeRecoveryPublisher struct {
	receipt  sourcing.PublicationReceipt
	verifies int
}

func (p *planChargeRecoveryPublisher) Publish(context.Context, sourcing.PublicationCommand) (sourcing.PublicationReceipt, error) {
	panic("recovery cannot publish")
}
func (p *planChargeRecoveryPublisher) Verify(context.Context, sourcing.PublicationCommand) (sourcing.PublicationReceipt, error) {
	p.verifies++
	return p.receipt, nil
}
func (p *planChargeRecoveryPublisher) Read(context.Context, string) (sourcing.PersistedPublication, error) {
	return sourcing.PersistedPublication{Receipt: p.receipt}, nil
}

type planChargeRecovery struct {
	calls, consumed int
	operationID     string
}

func (*planChargeRecovery) BeforeDispatch(context.Context, sourcing.AcquisitionOperation) error {
	panic("recovery cannot acquire new admission")
}
func (c *planChargeRecovery) Reconcile(_ context.Context, op sourcing.AcquisitionOperation) error {
	c.calls++
	if c.calls == 1 {
		return sourcing.ErrAcquisitionUnknown
	}
	if c.operationID == "" {
		c.operationID = op.ID
		c.consumed++
	}
	return nil
}
func TestPlanPublishedRecoveryReconcilesOriginalChargeBeforeCompleting(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123"}, sourcing.AcquisitionRule{}, false)
	id := uuid.NewString()
	source, _ := sourcing.Canonical1688Source(input.Inputs[0])
	op := acquisitionOperation(sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, planChildKey(id, source.URL), source)
	op.State = sourcing.AcquisitionPublished
	op.Command = &sourcing.PublicationCommand{PublicationID: "source-run:acquisition:" + op.ID, ProductKey: "1688:123"}
	raw, _ := json.Marshal(op.Command)
	hash := sha256.Sum256(raw)
	op.CommandHash = hex.EncodeToString(hash[:])
	publisher := &planChargeRecoveryPublisher{receipt: sourcing.PublicationReceipt{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, PublicationID: op.Command.PublicationID, CatalogPublicationID: op.Command.PublicationID, ProductKey: op.Command.ProductKey, CatalogVersion: 1}}
	charges := &planChargeRecovery{}
	child, e := NewAcquisitionService(&acquisitionStoreSpy{op: op}, nil, publisher, nil, acquisitionAuthFunc(func(context.Context) (sourcing.PublicationScope, error) { return op.Scope, nil }), charges)
	require.NoError(t, e)
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: id, Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: 1, Revision: 1, State: "pending", Items: []sourcing.AcquisitionPlanItem{{Source: source.URL, State: "pending"}}}}
	service, e := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
	require.NoError(t, e)
	p, e := service.Advance(context.Background(), id, 1)
	require.NoError(t, e)
	require.Equal(t, "needs_verification", p.Items[0].State)
	require.Zero(t, charges.consumed)
	p, e = service.Advance(context.Background(), id, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "completed", p.State)
	require.Equal(t, 2, charges.calls)
	require.Equal(t, 1, charges.consumed)
	require.Equal(t, op.ID, charges.operationID)
	require.Equal(t, 2, publisher.verifies)
	_, e = service.Advance(context.Background(), id, p.Revision)
	require.NoError(t, e)
	require.Equal(t, 1, charges.consumed)
}

type groupedPlanCollections struct {
	items   map[string]collection.ItemDetail
	batches map[string]collection.Batch
}

func (c *groupedPlanCollections) Mutate(context.Context, string, collection.Mutation) (collection.Receipt, error) {
	panic("must not move or recreate selected members")
}
func (c *groupedPlanCollections) ReadItem(_ context.Context, id string) (collection.ItemDetail, error) {
	return c.items[id], nil
}
func (c *groupedPlanCollections) ReadBatch(_ context.Context, id string) (collection.Batch, error) {
	return c.batches[id], nil
}

type groupedPlanPreparations struct {
	scope      preparation.Scope
	receipts   map[string]preparation.TransferReceipt
	keys       []string
	inputs     []preparation.TransferInput
	loseSecond bool
}

func (p *groupedPlanPreparations) ReadByKey(_ context.Context, key string) (preparation.TransferReceipt, error) {
	if receipt, ok := p.receipts[key]; ok {
		return receipt, nil
	}
	return preparation.TransferReceipt{}, preparation.ErrNotFound
}
func (p *groupedPlanPreparations) Transfer(_ context.Context, key string, input preparation.TransferInput) (preparation.TransferReceipt, error) {
	p.keys = append(p.keys, key)
	p.inputs = append(p.inputs, input)
	receipt := preparation.TransferReceipt{Preparation: preparation.Preparation{ID: preparation.OperationID(p.scope, key), SourceBatchID: input.BatchID, SourceRevision: input.ExpectedRevision, Count: int64(len(input.ItemIDs))}}
	p.receipts[key] = receipt
	if p.loseSecond && len(p.keys) == 2 {
		return preparation.TransferReceipt{}, preparation.ErrUnknown
	}
	return receipt, nil
}
func TestPlanTransfersDifferentOriginalBatchesAndRecoversLostReceipt(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123", "456"}, sourcing.AcquisitionRule{}, true)
	collections := &groupedPlanCollections{items: map[string]collection.ItemDetail{}, batches: map[string]collection.Batch{}}
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: 2, Revision: 1, State: "pending"}}
	for i, source := range input.Inputs {
		batchID, itemID, opID := uuid.NewString(), uuid.NewString(), uuid.NewString()
		collections.batches[batchID] = collection.Batch{ID: batchID, Revision: int64(i + 3)}
		collections.items[itemID] = collection.ItemDetail{Item: collection.Item{ID: itemID, BatchID: batchID, Revision: 1, Source: collection.Source{OperationID: opID}}}
		r.plan.Items = append(r.plan.Items, sourcing.AcquisitionPlanItem{Source: source, ChildKey: planChildKey(r.plan.ID, source), State: "published", OperationID: opID, Evaluation: &sourcing.RuleEvaluation{State: sourcing.RuleMatched}, CollectionItemID: itemID, CollectionBatchID: batchID, CollectionRevision: 1})
	}
	preps := &groupedPlanPreparations{scope: preparation.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}, receipts: map[string]preparation.TransferReceipt{}, loseSecond: true}
	service, e := NewAcquisitionPlanService(r, nil, &planChild{}, collections, preps, planTestAuth{scope}, nil)
	require.NoError(t, e)
	p, e := service.Advance(context.Background(), r.plan.ID, 1)
	require.NoError(t, e)
	require.Equal(t, "transfer_pending", p.State)
	require.Empty(t, preps.keys)
	frozen := append([]byte{}, p.TransferInputJSON...)
	p, e = service.Advance(context.Background(), p.ID, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "transfer_pending", p.State)
	p, e = service.Advance(context.Background(), p.ID, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "transfer_pending", p.State)
	require.Equal(t, "transfer_needs_verification", p.FailureCode)
	p, e = service.Advance(context.Background(), p.ID, p.Revision)
	require.NoError(t, e)
	require.Equal(t, "completed", p.State)
	require.Equal(t, frozen, p.TransferInputJSON)
	require.Len(t, preps.keys, 2)
	for i, item := range p.Items {
		require.Equal(t, collection.StableID(p.ID, item.CollectionBatchID, "transfer"), preps.keys[i])
		require.Equal(t, []string{item.CollectionItemID}, preps.inputs[i].ItemIDs)
		require.Equal(t, item.CollectionBatchID, preps.inputs[i].BatchID)
	}
	raw, e := json.Marshal(p)
	require.NoError(t, e)
	var result struct {
		PreparationIDs []string `json:"preparationIds"`
	}
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Len(t, result.PreparationIDs, 2)
}

func TestPlanRestoresOriginalTransferAfterProjectionSaveLoss(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123"}, sourcing.AcquisitionRule{}, true)
	planID, batchID, itemID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	key := collection.StableID(planID, batchID, "transfer")
	frozen, _ := json.Marshal([]planTransferIntent{{Key: key, Input: preparation.TransferInput{BatchID: batchID, ExpectedRevision: 3, ItemIDs: []string{itemID}}}})
	r := &planMemory{losePrefixSave: true, plan: sourcing.AcquisitionPlan{ID: planID, Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: 1, Revision: 1, State: "transfer_pending", TransferKey: collection.StableID(planID, "transfer"), TransferInputJSON: frozen, Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], State: "published"}}}}
	preps := &groupedPlanPreparations{scope: preparation.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}, receipts: map[string]preparation.TransferReceipt{}}
	service, e := NewAcquisitionPlanService(r, nil, &planChild{}, nil, preps, planTestAuth{scope}, nil)
	require.NoError(t, e)
	_, e = service.Advance(context.Background(), planID, 1)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionUnknown)
	require.Empty(t, r.plan.PreparationIDs)
	p, e := service.Advance(context.Background(), planID, 1)
	require.NoError(t, e)
	require.Equal(t, "completed", p.State)
	require.Equal(t, []string{key}, preps.keys)
	require.Equal(t, frozen, p.TransferInputJSON)
	require.Equal(t, []string{preparation.OperationID(preps.scope, key)}, p.PreparationIDs)
}
