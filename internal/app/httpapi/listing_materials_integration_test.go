package httpapi

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	supplyapp "task-processor/internal/app/supplychain"
	"task-processor/internal/commercetool"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	sheinmodel "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

// The original acquisition/supply fixtures create a fresh synthetic database.
// Only the rules/model protocol are controlled loopback fixtures; all run,
// configuration, member, point, source and Target owners are actual adapters.
func TestMaterialsHTTPRunAndNativeDraftUseFourExistingModes(t *testing.T) {
	for _, mode := range []string{"default", "read_only", "auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "supply materials "+mode) })
	}
}

func TestMaterialsNewAdoptionRechecksOriginalModeConfigurationAndTemplate(t *testing.T) {
	for _, change := range []string{"mode changed", "disabled", "archived"} {
		t.Run(change, func(t *testing.T) { testProductAgentOwners(t, "supply materials default "+change) })
	}
}

func TestMaterialsAdoptionRejectsDifferentManualCommandUnderAgentKey(t *testing.T) {
	testProductAgentOwners(t, "supply materials default receipt collision")
}

func TestMaterialsQualifiedAutomaticSaveUsesNativeReceipt(t *testing.T) {
	for _, mode := range []string{"auto_approve", "full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, "supply materials "+mode+" qualified") })
	}
}

func materialsFixturePointer[T any](v T) *T { return &v }

type materialsQualifiedFixtureRules struct{ supplyAgentFixtureRules }

func (r materialsQualifiedFixtureRules) ReadTargetRules(context.Context, collection.Scope, string, goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	return r.merchant, goods.OfficialRuleSnapshot{ApplicationMode: sheinmodel.ModeSelfOperated, Warehouses: []sheinmodel.Warehouse{}, Categories: []sheinmodel.Category{{ID: 123, ProductTypeID: 456, Leaf: materialsFixturePointer(true)}}, Sites: []sheinmodel.MainSite{{ID: "shein", Sites: []sheinmodel.Site{{Abbreviation: "shein-us", Status: materialsFixturePointer(1), StoreType: materialsFixturePointer(2), Currency: "USD"}}}}, Fill: sheinmodel.FillStandards{DefaultLanguage: "en", DefaultTitleMaximum: materialsFixturePointer(150), SupplierCodeInSPU: materialsFixturePointer(false), Fields: []sheinmodel.FillRule{}, Pictures: []sheinmodel.PictureRule{{Field: "switch_spu_picture", Enabled: materialsFixturePointer(false)}, {Field: "sku_image_required", Enabled: materialsFixturePointer(false)}}}, Attributes: sheinmodel.AttributeTemplate{ProductTypeID: 456, MainAttributeStatus: materialsFixturePointer(1), Attributes: []sheinmodel.Attribute{{ID: 12, Name: "Default", Type: materialsFixturePointer(1), Show: materialsFixturePointer(1), MainLabel: materialsFixturePointer(1), Mode: materialsFixturePointer(2), Status: materialsFixturePointer(3), MaximumSelections: materialsFixturePointer(1), Options: []sheinmodel.AttributeOption{{ID: 34, Show: materialsFixturePointer(1), Name: "Default"}}}}}, Linked: []sheinmodel.LinkedRules{{GroupID: "product", Attributes: []sheinmodel.LinkedAttributeRule{}}, {GroupID: "sku-0-0", Attributes: []sheinmodel.LinkedAttributeRule{}}}, Brands: []sheinmodel.Brand{{Code: "fixture-brand", Name: "Fixture"}}}, nil
}
func materialsQualifiedDraft() goods.OfficialDraftInput {
	return goods.OfficialDraftInput{Product: sheinmodel.PublishProduct{CategoryID: 123, BrandCode: "fixture-brand", Descriptions: []sheinmodel.LanguageContent{{Language: "en", Name: "Human fixture description"}}, SupplierCode: "source-product", Names: []sheinmodel.LanguageContent{{Language: "en", Name: "Human fixture product"}}, Attributes: []sheinmodel.AttributeValue{}, SKCs: []sheinmodel.ProductSKC{{SupplierCode: "merchant-code", SaleAttribute: sheinmodel.AttributeValue{AttributeID: 12, AttributeValueID: materialsFixturePointer(int64(34))}, SKUs: []sheinmodel.ProductSKU{{SupplierSKU: "merchant-sku", Length: "10", Width: "10", Height: "10", Weight: materialsFixturePointer(10.0), MallState: 1, Prices: []sheinmodel.ProductPrice{{BasePrice: 12.5, Currency: "USD", SubSite: "shein-us"}}, Stock: []sheinmodel.ProductStock{{Quantity: 5}}, SaleAttributes: []sheinmodel.AttributeValue{}}}}}}, Images: []goods.OfficialImageSlot{{Group: "skc", AssetID: "qualified-main", Sort: 1, Type: 1}, {Group: "skc", AssetID: "qualified-detail", Sort: 2, Type: 2}, {Group: "skc", AssetID: "qualified-square", Sort: 3, Type: 5}}}
}

type materialsQualifiedImageProbe struct{}

func (materialsQualifiedImageProbe) Probe(_ context.Context, a asset.ApprovedAsset, typ int) (goods.OfficialImageObservation, error) {
	return goods.OfficialImageObservation{AssetID: a.ID, SourceURL: a.URL, Width: 900, Height: 900, Type: typ, Bytes: 1000, MediaType: "image/png", ContentHash: strings.Repeat("a", 64)}, nil
}
func installMaterialsQualifiedAssets(t *testing.T, f *acquisitionHTTPFixture, productKey string) {
	t.Helper()
	repo, err := assetstore.NewRepository(f.owner)
	require.NoError(t, err)
	var images []asset.ApprovedAsset
	for _, name := range []string{"main", "detail", "square"} {
		images = append(images, asset.ApprovedAsset{ID: "qualified-" + name, RunID: "fixture-images", PlanRevision: 1, SlotID: name, Attempt: 1, Role: asset.RoleGallery, URL: "https://fixture.example/" + name + ".png", Width: 900, Height: 900})
	}
	_, err = repo.CommitApproval(context.Background(), asset.ApprovalCommit{TenantID: "B", ProductKey: productKey, TargetPlatform: "shein", SourceSnapshotVersion: 1, ActionID: "qualified-materials-assets", Assets: images})
	require.NoError(t, err)
}

type materialsOwnerFixtureRules struct{ supplyAgentFixtureRules }

func (r materialsOwnerFixtureRules) ReadTargetRules(context.Context, collection.Scope, string, goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	return r.merchant, goods.OfficialRuleSnapshot{ApplicationMode: sheinmodel.ModeSelfOperated, Brands: []sheinmodel.Brand{{Code: "fixture-brand", Name: "Disclosed fixture brand"}}}, nil
}

func testMaterialsNativeOwners(t *testing.T, value string, parent *productAgentApplication, supply *supplyapp.Application, base record.TargetReceipt, actor context.Context, deps routeAuthDependencies, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	_, err := parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.MaterialsAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.MaterialsAgentID, Version: agentconfig.MaterialsAgentVersion}}
	mode, err := parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: subject, Key: uuid.NewString(), Mode: agent.ExecutionMode(strings.Fields(value)[0])})
	require.NoError(t, err)
	template, err := parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "create-template", Input: agentconfig.TemplateInput{Name: "固定平台资料测试", TargetPlatform: "shein"}})
	require.NoError(t, err)
	a, err := buildListingMaterialsApplication(ctx, parent, supply)
	require.NoError(t, err)
	require.NotNil(t, a.runtime)
	parent.config.Limits.Runtime = 2 * time.Minute
	parent.materials = a
	server := httptest.NewServer(buildIsolatedApplicationHTTPServer(listingMaterialsRoutes(a, a.bindRequest), deps, 2*time.Minute).Handler)
	defer server.Close()
	key := uuid.NewString()
	path := strings.Replace(listingMaterialsBase, ":record_id", base.Record.ID, 1)
	call := func(method, path, key string, body any, want int) []byte {
		t.Helper()
		text := ""
		if body != nil {
			text = string(mustMaterialsJSON(t, body))
		}
		code, raw, e := acquisitionHTTPRequest(server, method, path, "admin", "B", key, text)
		require.NoError(t, e)
		require.Equal(t, want, code, string(raw))
		return raw
	}
	raw := call("POST", path+"/prepare", "", materialsPrepareInput{RequestKey: key, ExpectedModeRevision: mode.Revision, GoalSummary: "核对品牌资料", Template: &agentconfig.TemplateRef{TemplateID: template.TemplateID, Revision: "1"}}, 200)
	var prepared struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	}
	require.NoError(t, json.Unmarshal(raw, &prepared))
	frozen, err := parent.configuration.LoadSnapshot(ctx, scope, prepared.ConfigurationSnapshotRef)
	require.NoError(t, err)
	require.LessOrEqual(t, frozen.Request.Limits.Runtime, 90*time.Second, "the model run must reserve route time for Target saving")
	require.True(t, materialsRefValid(prepared.ConfigurationSnapshotRef))
	start := materialsStartBody{ConfigurationSnapshotRef: prepared.ConfigurationSnapshotRef}
	raw = call("POST", path+"/runs", key, start, 200)
	var result materialsRunDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.Equal(t, agent.HumanReviewRequired, result.Phase, string(raw))
	require.Equal(t, agent.ExecutionMode(strings.Fields(value)[0]), result.ExecutionMode)
	require.NotNil(t, result.Proposal)
	require.NotNil(t, result.After)
	if strings.Contains(value, " qualified") {
		require.True(t, base.Record.Result.ReadyForUpload, "fixture must satisfy actual official builder, not a mocked readiness flag")
		require.NotNil(t, result.Adoption, "qualified automatic mode must save through original Target; unresolved=%v", result.Unresolved)
		require.Equal(t, int64(2), result.Adoption.Record.Revision)
		require.True(t, result.Adoption.Record.Result.ReadyForUpload)
		saved := result.Adoption.Record.ID
		call("POST", path+"/runs", key, start, 200)
		read := call("GET", path+"/runs/"+key+"/adoption", "", nil, 200)
		require.Contains(t, string(read), saved)
		require.Equal(t, int32(2), calls.Load(), "no model resend when saved head advances")
		return
	}
	require.Equal(t, "fixture-brand", *result.Proposal.BrandCode)
	require.Equal(t, base.Record.Input, result.Before)
	require.Equal(t, int32(2), calls.Load(), "one governed inspect/propose sequence")
	require.Nil(t, result.Adoption, "incomplete rules never automatically adopt, even full")
	adopt := materialsAdoptBody{ConfigurationSnapshotRef: prepared.ConfigurationSnapshotRef, CandidateHash: result.CandidateHash}
	if strings.Contains(value, "receipt collision") {
		manual := base.Record.Input
		manual.ExpectedRevision = base.Record.Revision
		manual.Draft.Product.BrandCode = "different-manual-brand"
		commandKey := collection.StableID("B", "admin", "listing-materials-adopt", result.RunID, result.CandidateHash)
		_, err := supply.Targets.Create(actor, commandKey, manual)
		require.NoError(t, err)
		call("GET", path+"/runs/"+key+"/adoption", "", nil, 409)
		adopt.HumanConfirmed = true
		call("POST", path+"/runs/"+key+"/adopt", "", adopt, 409)
		raw := call("GET", path+"/runs/"+key, "", nil, 200)
		var projected materialsRunDTO
		require.NoError(t, json.Unmarshal(raw, &projected))
		require.Nil(t, projected.Adoption, "a different manual command cannot be projected as this candidate's save")
		require.False(t, projected.CanRequestAdoption)
		require.LessOrEqual(t, len(projected.Unresolved), 4096, "collision history must remain within the existing BFF contract")
		require.Contains(t, projected.Unresolved, "原保存编号对应的草稿与本次建议不一致，请核对原记录。")
		require.Contains(t, projected.Unresolved[len(projected.Unresolved)-1], "省略")
		require.Equal(t, int32(2), calls.Load(), "reading bounded history does not call the model again")
		return
	}
	if value != "default" && strings.HasPrefix(value, "default ") {
		switch strings.TrimPrefix(value, "default ") {
		case "mode changed":
			_, err = parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: subject, Key: uuid.NewString(), ExpectedRevision: 1, Mode: agent.ModeReadOnly})
		case "disabled":
			_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "disable", Expected: 1})
		case "archived":
			_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "archive-template", TemplateID: template.TemplateID, Expected: 1})
		}
		require.NoError(t, err)
		adopt.HumanConfirmed = true
		call("POST", path+"/runs/"+key+"/adopt", "", adopt, 409)
		head, e := supply.ReadTarget(actor, base.Record.Source.ID, base.Record.Input.StoreID)
		require.NoError(t, e)
		require.Equal(t, base.Record.ID, head.ID)
		call("GET", path+"/runs/"+key, "", nil, 200)
		require.Equal(t, int32(2), calls.Load())
		return
	}
	status := 409
	if value == "read_only" {
		status = 403
	}
	raw = call("POST", path+"/runs/"+key+"/adopt", "", adopt, status)
	if value != "read_only" {
		require.Contains(t, string(raw), "HUMAN_CONFIRMATION_REQUIRED")
	}
	head, e := supply.ReadTarget(actor, base.Record.Source.ID, base.Record.Input.StoreID)
	require.NoError(t, e)
	require.Equal(t, base.Record.ID, head.ID, "denial cannot advance Target head")
	adopt.HumanConfirmed = true
	if value == "read_only" {
		call("POST", path+"/runs/"+key+"/adopt", "", adopt, 403)
		raw = call("GET", path+"/runs/"+key, "", nil, 200)
		require.NoError(t, json.Unmarshal(raw, &result))
		require.False(t, result.CanRequestAdoption)
		return
	}
	raw = call("POST", path+"/runs/"+key+"/adopt", "", adopt, 200)
	var saved record.TargetReceipt
	require.NoError(t, json.Unmarshal(raw, &saved))
	require.Equal(t, int64(2), saved.Record.Revision)
	require.Equal(t, "fixture-brand", saved.Record.Input.Draft.Product.BrandCode)
	// The Target owner alone persists a new record. Canonical content and all
	// protected commerce/source fields are byte-equivalent to the original.
	expected := base.Record.Input
	expected.ExpectedRevision = base.Record.Revision
	expected.Draft.Product.BrandCode = "fixture-brand"
	require.Equal(t, expected, saved.Record.Input)
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.MaterialsAgentID, Operation: "disable", Expected: 1})
	require.NoError(t, err)
	_, err = parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: subject, Key: uuid.NewString(), ExpectedRevision: 1, Mode: agent.ModeReadOnly})
	require.NoError(t, err)
	for _, method := range []string{"GET", "POST"} {
		endpoint := path + "/runs/" + key + "/adoption"
		var body any
		if method == "POST" {
			endpoint = path + "/runs/" + key + "/adopt"
			body = adopt
		}
		raw = call(method, endpoint, "", body, 200)
		var replay record.TargetReceipt
		require.NoError(t, json.Unmarshal(raw, &replay))
		require.Equal(t, saved.Record.ID, replay.Record.ID)
	}
	raw = call("GET", path+"/runs/"+key, "", nil, 200)
	require.NoError(t, json.Unmarshal(raw, &result))
	require.NotNil(t, result.Adoption)
	require.Equal(t, saved.Record.ID, result.Adoption.Record.ID)
	raw = call("POST", path+"/runs", key, start, 200)
	require.Contains(t, string(raw), saved.Record.ID, "old head/config/mode cannot hide committed history")
	require.Equal(t, int32(2), calls.Load(), "replay/read never invokes model again")
}
