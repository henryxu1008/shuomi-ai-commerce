package materials

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type inspectionFixture struct{ value Inspection }

func (f inspectionFixture) ReadMaterialsInspection(context.Context, agent.Binding) (Inspection, error) {
	return f.value, nil
}

func materialInspection(t *testing.T) Inspection {
	t.Helper()
	product := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org", ProductKey: "product"}, Version: 1, PublicationID: "publication", Snapshot: catalog.ProductSnapshot{Title: "Actual product", Brand: "Actual brand"}}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: "org", ProductKey: "product", TargetPlatform: "shein", SourceSnapshotVersion: 1}, Assets: []asset.ApprovedAsset{{ID: "approved", URL: "https://example.test/approved.png"}}}
	leaf, show := true, 1
	rules := goods.OfficialRuleSnapshot{Categories: []model.Category{{ID: 12, Leaf: &leaf}, {ID: 13, Leaf: &leaf}, {ID: 14}}, Brands: []model.Brand{{Code: "known-brand", Name: "Actual brand"}}, Attributes: model.AttributeTemplate{Attributes: []model.Attribute{{ID: 5, Show: &show, Options: []model.AttributeOption{{ID: 8, Show: &show}}}}}}
	base := record.TargetRecord{ID: "record", TargetID: "target", Revision: 1, EffectiveVersion: 1, ProductHash: collection.Digest(product.Snapshot), InventoryHash: collection.Digest(inventory), RulesHash: collection.Digest(rules), Merchant: storecenter.ProductMerchantBinding{OrganizationID: "org", StoreID: "store", Site: "shein-us"}, Input: record.TargetInput{SourceID: "source", StoreID: "store", EffectiveVersion: 1, Draft: goods.OfficialDraftInput{Product: model.PublishProduct{CategoryID: 12, SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{{SupplierSKU: "sku"}}}}}}}}
	base.Source.ID = "source"
	base.Source.Source = collection.Source{ProductKey: "product", PublicationID: "publication", Version: 1}
	value, err := NewInspection(base, product, rules, inventory)
	require.NoError(t, err)
	return value
}

func observedInspection(t *testing.T, value Inspection) []agent.Observation {
	t.Helper()
	disclosed, err := value.Disclose()
	require.NoError(t, err)
	raw, err := json.Marshal(disclosed)
	require.NoError(t, err)
	return []agent.Observation{{Tool: InspectToolRef(), CallID: "call", AuditStatus: commercetool.AuditStatusRecorded, Output: raw}}
}

func proposalFor(t *testing.T, value Inspection, fields string) agent.DomainProposal {
	t.Helper()
	return agent.DomainProposal{Kind: ProposalKind, Payload: json.RawMessage(`{` + fields + `,"evidenceIds":["` + value.EvidenceID + `"]}`)}
}

func TestMaterialsValidatorRequiresActuallyDisclosedExactEvidence(t *testing.T) {
	value := materialInspection(t)
	request := agent.Request{Binding: value.Binding, PolicyVersion: PolicyVersion, PromptVersion: PromptVersion}
	candidate := proposalFor(t, value, `"brandCode":"known-brand"`)
	validator := Validator{Reader: inspectionFixture{value}}
	for _, mutate := range []func([]agent.Observation) []agent.Observation{
		func(_ []agent.Observation) []agent.Observation { return nil },
		func(h []agent.Observation) []agent.Observation {
			h[0].AuditStatus = commercetool.AuditStatusRecordFailed
			return h
		},
		func(h []agent.Observation) []agent.Observation { h[0].Tool.ID = "product.readiness.inspect"; return h },
		func(h []agent.Observation) []agent.Observation {
			h[0].InvocationID = "unexpected-model-invocation"
			return h
		},
	} {
		result, err := validator.ValidateDomain(context.Background(), request, candidate, mutate(observedInspection(t, value)))
		require.NoError(t, err)
		require.False(t, result.Valid)
	}
	result, err := validator.ValidateDomain(context.Background(), request, candidate, observedInspection(t, value))
	require.NoError(t, err)
	require.True(t, result.Valid)
	require.NotEmpty(t, result.Unresolved, "an incomplete draft is not platform readiness")
	changed := value
	changed.Rules.Brands = append(changed.Rules.Brands, model.Brand{Code: "new"})
	validator.Reader = inspectionFixture{changed}
	_, err = validator.ValidateDomain(context.Background(), request, candidate, observedInspection(t, value))
	require.ErrorIs(t, err, ErrInspectionChanged)
}

func TestMaterialPreviewRejectsInventedRulesAssetsAndNonleafCategory(t *testing.T) {
	value := materialInspection(t)
	for _, fields := range []string{
		`"brandCode":"invented"`,
		`"categoryId":14`,
		`"categoryId":999`,
		`"attributes":[{"attribute_id":999,"attribute_value_id":8}]`,
		`"attributes":[{"attribute_id":5,"attribute_value_id":999}]`,
		`"attributes":[{"attribute_id":5,"custom_attribute_value":"invented"}]`,
		`"images":[{"group":"spu","skc":0,"sku":0,"asset_id":"unapproved","sort":1,"type":1}]`,
	} {
		p, err := ParseProposal(proposalFor(t, value, fields).Payload)
		require.NoError(t, err)
		_, err = Preview(value, p)
		require.ErrorIs(t, err, ErrInvalidProposal, fields)
	}
	p, err := ParseProposal(proposalFor(t, value, `"categoryId":13`).Payload)
	require.NoError(t, err)
	preview, err := Preview(value, p)
	require.NoError(t, err)
	require.True(t, preview.RequiresHuman)
	require.False(t, preview.Result.ReadyForUpload)
	require.Empty(t, preview.Result.SubmissionPayload)
	require.EqualValues(t, 13, preview.Input.Draft.Product.CategoryID)
}

func TestMaterialsInspectionRejectsDriftAndOversizedDisclosure(t *testing.T) {
	value := materialInspection(t)
	value.Inventory.Assets = nil
	_, err := NewInspection(value.Base, value.Product, value.Rules, value.Inventory)
	require.ErrorIs(t, err, ErrInspectionChanged)
	value = materialInspection(t)
	value.Product.Snapshot.Description = string(make([]byte, MaxInspectionBytes))
	value.Base.ProductHash = collection.Digest(value.Product.Snapshot)
	_, err = NewInspection(value.Base, value.Product, value.Rules, value.Inventory)
	require.ErrorIs(t, err, ErrInspectionTooLarge)
}
