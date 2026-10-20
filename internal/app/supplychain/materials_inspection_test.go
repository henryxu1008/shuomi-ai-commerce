package supplychainapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	"task-processor/internal/listing/materials/tools/materialsinspect"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

func materialsReadContext() context.Context {
	return authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-a", EffectiveOrganizationID: "org-a", UserID: "actor-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(time.Minute)})
}

func materialsApplication(t *testing.T) (*Application, *uploadRecords, *uploadAuth) {
	t.Helper()
	upload, _, records, _, _, _, auth := uploadFixture(t)
	d := upload.dependencies
	return &Application{Sources: d.Sources.(*preparation.SourceSelector), Records: records, Products: d.Products, Rules: d.Rules, Assets: d.Assets, Authorization: auth}, records, auth
}

func TestMaterialsInspectionUsesExactOwnersWithoutWritingOrPublishing(t *testing.T) {
	a, records, _ := materialsApplication(t)
	i, err := a.InspectMaterialsRecord(materialsReadContext(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, records.saved.ID, i.Binding.ContextID)
	require.Equal(t, "listing-target-record", i.Binding.ContextKind)
	require.Equal(t, records.saved.Input.Draft, i.Base.Input.Draft)
	require.Equal(t, records.saved.ProductHash, collection.Digest(i.Product.Snapshot))
	require.Equal(t, records.saved.RulesHash, collection.Digest(i.Rules))
	require.Equal(t, records.saved.InventoryHash, collection.Digest(i.Inventory))
	invoker, err := materialsinspect.NewInvoker(a, materialsToolIdentity{}, materialsinspect.AgentDefinition(), commercetool.InvocationDependencies{Authorizer: materialsToolAuthorization{}, Recorder: materialsToolAudit{}, Tracer: otel.Tracer("materials-owner-test"), Now: time.Now, AuditTimeout: time.Second})
	require.NoError(t, err)
	result, err := invoker.Invoke(materialsReadContext(), materials.InspectToolRef(), commercetool.CallMetadata{CallID: "call", AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, AgentRunID: "run", BusinessTaskID: i.Binding.ContextID}, i.Binding)
	require.NoError(t, err, "actual saved SKU/fill/attribute/asset shapes must pass the strict shared Tool schema")
	require.Equal(t, commercetool.AuditStatusRecorded, result.AuditStatus)
	_, err = a.ReadMaterialsInspection(materialsReadContext(), i.Binding)
	require.NoError(t, err)
	wrong := i.Binding
	wrong.PublicationID = "another-publication"
	_, err = a.ReadMaterialsInspection(materialsReadContext(), wrong)
	require.ErrorIs(t, err, materials.ErrInspectionChanged)
}

type materialsToolIdentity struct{}

func (materialsToolIdentity) ResolveFreshPrincipal(context.Context) (commercetool.Principal, error) {
	return commercetool.Principal{TenantID: "org-a", UserID: "actor-a", Roles: []string{"operator"}}, nil
}

type materialsToolAuthorization struct{}

func (materialsToolAuthorization) Authorize(context.Context, commercetool.Principal, commercetool.PermissionRequirement) error {
	return nil
}

type materialsToolAudit struct{}

func (materialsToolAudit) RecordToolCall(context.Context, commercetool.AuditRecord) error { return nil }

func TestMaterialsInspectionRejectsHeadStoreRulesAssetAndSourceDrift(t *testing.T) {
	for _, change := range []func(*Application, *uploadRecords){
		func(_ *Application, r *uploadRecords) { r.saved.ID = "different-record" },
		func(_ *Application, r *uploadRecords) { r.saved.ProductHash = "changed" },
		func(_ *Application, r *uploadRecords) { r.saved.RulesHash = "changed" },
		func(_ *Application, r *uploadRecords) { r.saved.InventoryHash = "changed" },
		func(_ *Application, r *uploadRecords) { r.saved.Merchant.StoreVersion++ },
		func(_ *Application, r *uploadRecords) { r.saved.Source.Source.PublicationID = "another-source" },
	} {
		a, records, _ := materialsApplication(t)
		original := records.saved.ID
		change(a, records)
		_, err := a.InspectMaterialsRecord(materialsReadContext(), original)
		require.Error(t, err)
	}
	a, records, auth := materialsApplication(t)
	changed := auth.scope
	changed.MemberID = "other-member"
	a.Authorization = &materialsReadAuth{original: auth.scope, changed: changed}
	_, err := a.InspectMaterialsRecord(materialsReadContext(), records.saved.ID)
	require.Error(t, err)
}

type materialsReadAuth struct {
	original, changed collection.Scope
	calls             int
}

func (a *materialsReadAuth) Authorize(context.Context, string) (collection.Scope, error) {
	a.calls++
	if a.calls > 1 {
		return a.changed, nil
	}
	return a.original, nil
}

var _ materials.InspectionReader = (*Application)(nil)
