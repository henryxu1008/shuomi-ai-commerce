package target

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestLocalTrialBootstrapImportMappingKeepOneTargetAndProvenance(t *testing.T) {
	ctx, old, repo, c, r, _, sales := salesServiceFixture(t)
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	c.saved.Revision = 1
	c.saved.Content.Title = "R1 title"
	sales.CompletionDraft.Revision = 1
	r.rules.Authority = "local_trial"
	r.rules.Warehouses = []model.Warehouse{{Code: "warehouse", Type: 1, SaleCountries: []string{"US"}}}
	r.merchant.Platform = storecenter.PlatformShein
	r.merchant.ConnectionRevision = 0
	r.merchant.ApplicationID = ""
	r.merchant.ApplicationRevision = ""
	r.merchant.ServiceExpiresAt = time.Time{}
	policy := LocalTrialPolicy{Scope: scope, Binding: LocalTrial{ContractVersion: 1, TrialID: "issue1158-local-draft-v1", SourceID: repo.head.Source.ID, StoreID: r.merchant.StoreID, CompletionDraft: sales.CompletionDraft, FixtureVersion: "issue1158-local-draft-v1", FixtureHash: collection.Digest(r.rules)}, ProductKey: repo.head.Source.Source.ProductKey, ProductVersion: 1, Enabled: true}
	s, err := NewLocalTrialTargetService(old.dependencies, policy)
	require.NoError(t, err)
	repo.head = TargetRecord{}
	repo.byID = map[string]TargetRecord{}
	repo.values = map[string]TargetReceipt{}
	repo.hashes = map[string]string{}
	repo.writes = 0
	before := collection.Digest(c.saved)
	bootstrap, err := s.LocalTrialBootstrap(ctx, policy.Binding.SourceID, policy.Binding.StoreID)
	require.NoError(t, err)
	raw, _ := json.Marshal(bootstrap)
	for _, name := range []string{"remove marker", "add sku", "title", "hidden field", "wrong store"} {
		t.Run(name, func(t *testing.T) {
			var in TargetInput
			require.NoError(t, json.Unmarshal(raw, &in))
			switch name {
			case "remove marker":
				in.LocalTrial = nil
			case "add sku":
				in.Draft.Product.SKCs[0].SKUs = append(in.Draft.Product.SKCs[0].SKUs, in.Draft.Product.SKCs[0].SKUs[0])
			case "title":
				in.Draft.Product.Names[0].Name = "injected"
			case "hidden field":
				in.Draft.Product.BrandCode = "injected"
			case "wrong store":
				in.StoreID = uuid.NewString()
			}
			_, err := s.Create(ctx, uuid.NewString(), in)
			require.Error(t, err)
			require.Zero(t, repo.writes)
		})
	}
	key := uuid.NewString()
	t1, err := s.Create(ctx, key, bootstrap)
	require.NoError(t, err)
	require.True(t, ValidPlatformRecord(t1.Record))
	replay, err := s.Create(ctx, key, bootstrap)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, 1, repo.writes)
	request := SheinImportRequest{SourceID: policy.Binding.SourceID, BaseRecordID: t1.Record.ID, CompletionDraft: policy.Binding.CompletionDraft, Selection: SheinImportSelection{Title: true, SKUs: []SheinSKUMapping{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0, Stock: true, WarehouseID: "warehouse"}}}}
	preview, err := s.PreviewSheinImport(ctx, request)
	require.NoError(t, err)
	require.True(t, SameLocalTrial(t1.Record.LocalTrial, preview.Input.LocalTrial))
	t2, err := s.Create(ctx, uuid.NewString(), preview.Input)
	require.NoError(t, err)
	sales.BaseRecordID = t2.Record.ID
	sales.Selection.Images = nil
	mapped, err := s.PreviewSheinSalesMapping(ctx, sales)
	require.NoError(t, err)
	require.NotNil(t, mapped.Input, "%v", mapped.Issues)
	t3, err := s.Create(ctx, uuid.NewString(), *mapped.Input)
	require.NoError(t, err)
	require.Equal(t, t1.Record.TargetID, t3.Record.TargetID)
	require.EqualValues(t, 3, t3.Record.Revision)
	rows := t3.Record.Input.Draft.Product.SKCs
	require.Len(t, rows, 2)
	require.Len(t, rows[0].SKUs, 2)
	require.Len(t, rows[1].SKUs, 1)
	require.Equal(t, 0, rows[0].SKUs[0].Stock[0].Quantity)
	require.Empty(t, rows[0].SKUs[1].Stock)
	require.Empty(t, rows[1].SKUs[0].Stock)
	require.Empty(t, rows[0].SKUs[0].Prices)
	require.Equal(t, before, collection.Digest(c.saved))
	require.Len(t, repo.byID, 3)
	require.False(t, PublicationReady(t3.Record))
	require.Empty(t, t3.Record.Result.SubmissionPayload)
	_, err = s.CreateForExecution(context.Background(), scope, uuid.NewString(), *mapped.Input)
	require.ErrorIs(t, err, ErrForbidden)
	_, err = old.Create(ctx, uuid.NewString(), *mapped.Input)
	require.ErrorIs(t, err, ErrForbidden)
}
