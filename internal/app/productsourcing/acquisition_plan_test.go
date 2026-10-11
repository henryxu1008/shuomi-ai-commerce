package productsourcing

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/sourcing"
)

type planTestAuth struct{ scope sourcing.AcquisitionPlanScope }

func (a planTestAuth) AuthorizePlan(context.Context, bool) (sourcing.AcquisitionPlanScope, error) {
	return a.scope, nil
}

type planCollections struct {
	item  collection.ItemDetail
	batch collection.Batch
}

func (c *planCollections) Mutate(context.Context, string, collection.Mutation) (collection.Receipt, error) {
	return collection.Receipt{}, nil
}
func (c *planCollections) ReadItem(context.Context, string) (collection.ItemDetail, error) {
	return c.item, nil
}
func (c *planCollections) ReadBatch(context.Context, string) (collection.Batch, error) {
	return c.batch, nil
}

type planPreparation struct{ calls int }

func (p *planPreparation) Transfer(context.Context, string, preparation.TransferInput) (preparation.TransferReceipt, error) {
	p.calls++
	return preparation.TransferReceipt{}, preparation.ErrConflict
}
func (p *planPreparation) ReadByKey(context.Context, string) (preparation.TransferReceipt, error) {
	return preparation.TransferReceipt{}, preparation.ErrNotFound
}

func TestPlanRefusesChangedSelectedCollectionBeforeFreezingTransfer(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("product", []string{"123"}, sourcing.AcquisitionRule{}, true)
	batchID, itemID, operationID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: 1, Revision: 1, State: "pending", Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], State: "published", OperationID: operationID, Evaluation: &sourcing.RuleEvaluation{State: sourcing.RuleMatched}, CollectionItemID: itemID, CollectionBatchID: batchID, CollectionRevision: 1}}}}
	collections := &planCollections{item: collection.ItemDetail{Item: collection.Item{ID: itemID, BatchID: batchID, Revision: 2}}, batch: collection.Batch{ID: batchID, Revision: 2}}
	preps := &planPreparation{}
	service, e := NewAcquisitionPlanService(r, nil, &planChild{}, collections, preps, planTestAuth{scope}, nil)
	require.NoError(t, e)
	_, e = service.Advance(context.Background(), r.plan.ID, 1)
	require.ErrorIs(t, e, sourcing.ErrAcquisitionConflict)
	require.Empty(t, r.plan.TransferInputJSON)
	require.Zero(t, preps.calls)
}

type planMemory struct {
	plan           sourcing.AcquisitionPlan
	losePrefixSave bool
}

func (r *planMemory) Create(_ context.Context, p sourcing.AcquisitionPlan, _ sourcing.PlanAuthorityCheck) (sourcing.AcquisitionPlan, error) {
	r.plan = p
	return p, nil
}
func (r *planMemory) ByKey(context.Context, sourcing.AcquisitionPlanScope, string) (sourcing.AcquisitionPlan, error) {
	return r.plan, nil
}
func (r *planMemory) Read(context.Context, sourcing.AcquisitionPlanScope, string) (sourcing.AcquisitionPlan, error) {
	return r.plan, nil
}
func (r *planMemory) List(context.Context, sourcing.AcquisitionPlanScope, string) ([]sourcing.AcquisitionPlan, error) {
	return []sourcing.AcquisitionPlan{r.plan}, nil
}
func (r *planMemory) Claim(_ context.Context, _ sourcing.AcquisitionPlanScope, _ string, rev int64, _ sourcing.PlanAuthorityCheck) (sourcing.AcquisitionPlan, error) {
	if rev != r.plan.Revision {
		return sourcing.AcquisitionPlan{}, sourcing.ErrAcquisitionConflict
	}
	r.plan.Fence++
	return r.plan, nil
}
func (r *planMemory) Save(_ context.Context, p sourcing.AcquisitionPlan, _ sourcing.PlanAuthorityCheck) (sourcing.AcquisitionPlan, error) {
	if r.losePrefixSave && len(p.PreparationIDs) > 0 {
		r.losePrefixSave = false
		return p, sourcing.ErrAcquisitionUnknown
	}
	p.Revision++
	r.plan = p
	return p, nil
}

type planChild struct {
	acquire, verify                  int
	result                           sourcing.AcquisitionResult
	err                              error
	readResult                       sourcing.AcquisitionResult
	readErr                          error
	readIDs, acquireKeys, verifyKeys []string
}

type failedDiscovery struct{}

func (failedDiscovery) Discover(context.Context, sourcing.DiscoveryInput) (sourcing.DiscoveryEvidence, error) {
	return sourcing.DiscoveryEvidence{}, sourcing.ErrAcquisitionFailed
}
func TestDiscoveryFailureReturnsEmptyArrayAndContinuesRemainingInputs(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput("keyword", []string{"cup", "bag"}, sourcing.AcquisitionRule{}, false)
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, Revision: 1, State: "pending", Items: []sourcing.AcquisitionPlanItem{}, DiscoveryFailures: []sourcing.DiscoveryFailure{}}}
	child := &planChild{}
	service, e := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, failedDiscovery{})
	require.NoError(t, e)
	p, e := service.Advance(context.Background(), r.plan.ID, 1)
	require.NoError(t, e)
	require.Equal(t, 1, p.DiscoveryIndex)
	require.Len(t, p.DiscoveryFailures, 1)
	require.Zero(t, child.acquire)
	raw, e := json.Marshal(p)
	require.NoError(t, e)
	require.Contains(t, string(raw), `"items":[]`)
}

func (c *planChild) Acquire(_ context.Context, key string, _ string) (sourcing.AcquisitionResult, error) {
	c.acquire++
	c.acquireKeys = append(c.acquireKeys, key)
	return c.result, c.err
}
func (c *planChild) Verify(_ context.Context, key string, _ string) (sourcing.AcquisitionResult, error) {
	c.verify++
	c.verifyKeys = append(c.verifyKeys, key)
	return c.result, c.err
}
func (c *planChild) Read(_ context.Context, id string) (sourcing.AcquisitionResult, error) {
	c.readIDs = append(c.readIDs, id)
	return c.readResult, c.readErr
}

func TestPlanUnknownKeepsOriginalChildAndDoesNotAcquireAgain(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput(sourcing.DiscoveryProduct, []string{"123"}, sourcing.AcquisitionRule{}, false)
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: len(input.Inputs), Revision: 1, State: "pending", Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], State: "pending"}}, CreatedAt: time.Now()}}
	child := &planChild{err: sourcing.ErrAcquisitionUnknown}
	service, err := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
	require.NoError(t, err)
	p, err := service.Advance(context.Background(), r.plan.ID, 1)
	require.NoError(t, err)
	require.Equal(t, "needs_verification", p.Items[0].State)
	original := p.Items[0].ChildKey
	source, _ := sourcing.Canonical1688Source(p.Items[0].Source)
	child.readResult.Operation = acquisitionOperation(sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, original, source)
	child.readResult.Operation.State = sourcing.AcquisitionPublishing
	p, err = service.Advance(context.Background(), r.plan.ID, p.Revision)
	require.NoError(t, err)
	require.Equal(t, 1, child.acquire)
	require.Equal(t, 1, child.verify)
	require.Equal(t, original, p.Items[0].ChildKey)
}

func TestPlanReadsSealedFactsAndNeverTransfersZeroMatches(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput(sourcing.DiscoveryProduct, []string{"123"}, sourcing.AcquisitionRule{MaxPrice: "5"}, true)
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, DiscoveryIndex: len(input.Inputs), Revision: 1, State: "pending", Items: []sourcing.AcquisitionPlanItem{{Source: input.Inputs[0], State: "pending"}}}}
	currency := "CNY"
	quantity := "2"
	raw, _ := json.Marshal(sourcing.AcquisitionPrice{Amount: "10", Currency: &currency, MinQuantity: &quantity})
	child := &planChild{result: sourcing.AcquisitionResult{Operation: sourcing.AcquisitionOperation{ID: uuid.NewString(), State: sourcing.AcquisitionPublished, Command: &sourcing.PublicationCommand{Envelope: sourcing.SourceEnvelope{SupplierOrCostFacts: sourcing.SupplierOrCostFacts{Facts: map[string]string{"price_fact.0": string(raw)}}}}}, Publication: &sourcing.PersistedPublication{}}}
	source, _ := sourcing.Canonical1688Source(input.Inputs[0])
	op := acquisitionOperation(sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, planChildKey(r.plan.ID, source.URL), source)
	op.State = sourcing.AcquisitionPublished
	op.Command = child.result.Operation.Command
	op.Command.PublicationID = "source-run:acquisition:" + op.ID
	op.Command.ProductKey = "1688:123"
	sealed, _ := json.Marshal(op.Command)
	hash := sha256.Sum256(sealed)
	op.CommandHash = hex.EncodeToString(hash[:])
	child.result.Operation = op
	child.result.Publication.Receipt.OrganizationID = scope.OrganizationID
	child.result.Publication.Receipt.ActorID = scope.ActorID
	child.result.Publication.Receipt.PublicationID = op.Command.PublicationID
	child.result.Publication.Receipt.ProductKey = op.Command.ProductKey
	child.result.Publication.Receipt.CatalogVersion = 1
	service, err := NewAcquisitionPlanService(r, nil, child, nil, nil, planTestAuth{scope}, nil)
	require.NoError(t, err)
	p, err := service.Advance(context.Background(), r.plan.ID, 1)
	require.NoError(t, err)
	require.Equal(t, sourcing.RuleNotMatched, p.Items[0].Evaluation.State)
	require.Equal(t, "completed", p.State, "zero selected members must not become Transfer(empty)=entire batch")
}

func TestPlanCreateReplayDoesNotReloadChangedOrArchivedTemplate(t *testing.T) {
	scope := sourcing.AcquisitionPlanScope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	input, _ := sourcing.NewAcquisitionPlanInput(sourcing.DiscoveryProduct, []string{"123"}, sourcing.AcquisitionRule{MaxPrice: "10"}, false)
	input.TemplateID = uuid.NewString()
	input.TemplateRevision = 1
	r := &planMemory{plan: sourcing.AcquisitionPlan{ID: uuid.NewString(), Key: uuid.NewString(), Scope: scope, Input: input, Revision: 2, State: "pending"}}
	service, err := NewAcquisitionPlanService(r, nil, &planChild{}, nil, nil, planTestAuth{scope}, nil)
	require.NoError(t, err)
	p, err := service.Create(context.Background(), r.plan.Key, CreatePlanInput{Mode: sourcing.DiscoveryProduct, Inputs: []string{"123"}, TemplateID: input.TemplateID, TemplateRevision: 1})
	require.NoError(t, err)
	require.Equal(t, r.plan.ID, p.ID)
	require.Equal(t, "10", p.Input.Rule.MaxPrice)
	_, err = service.Create(context.Background(), r.plan.Key, CreatePlanInput{Mode: sourcing.DiscoveryProduct, Inputs: []string{"456"}, TemplateID: input.TemplateID, TemplateRevision: 1})
	require.ErrorIs(t, err, sourcing.ErrAcquisitionConflict)
}
