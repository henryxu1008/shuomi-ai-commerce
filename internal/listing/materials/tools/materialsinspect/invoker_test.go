package materialsinspect

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type reader struct {
	value materials.Inspection
	calls int
}

func (r *reader) ReadMaterialsInspection(_ context.Context, b agent.Binding) (materials.Inspection, error) {
	r.calls++
	if b != r.value.Binding {
		return materials.Inspection{}, materials.ErrInspectionChanged
	}
	return r.value, nil
}

func (r *reader) ReadCandidateRules(context.Context, agent.Binding) (materials.CandidateRulesDisclosure, error) {
	return materials.CandidateRulesDisclosure{}, materials.ErrInspectionChanged
}

type queryReader struct {
	*reader
	query      materials.CandidateRulesDisclosure
	queryCalls int
}

func (r *queryReader) ReadCandidateRules(_ context.Context, b agent.Binding) (materials.CandidateRulesDisclosure, error) {
	r.queryCalls++
	if b != r.query.Binding {
		return materials.CandidateRulesDisclosure{}, materials.ErrInspectionChanged
	}
	return r.query, nil
}

func TestCandidateRulesToolUsesOriginalRegistryAuditAndPurposeIsolation(t *testing.T) {
	_, r, f, a := inspector(t)
	base := r.value.Base
	base.ID, base.Input.StoreID = uuid.NewString(), uuid.NewString()
	base.Merchant.StoreID = base.Input.StoreID
	leaf := true
	q, err := materials.NewCandidateRulesDisclosure(base, r.value.Product, "123", goods.OfficialRuleSnapshot{Categories: []model.Category{{ID: 123, ProductTypeID: 456, Leaf: &leaf}}, Attributes: model.AttributeTemplate{ProductTypeID: 456}}, time.Now().UTC())
	require.NoError(t, err)
	qr := &queryReader{reader: r, query: q}
	i, err := NewInvoker(qr, f, AgentDefinition(), commercetool.InvocationDependencies{Authorizer: authorize{}, Recorder: a, Tracer: otel.Tracer("candidate-test"), Now: time.Now, AuditTimeout: time.Second})
	require.NoError(t, err)
	meta := commercetool.CallMetadata{CallID: "query-call", AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, AgentRunID: "run", BusinessTaskID: q.Binding.ContextID}
	result, err := i.Invoke(context.Background(), materials.RulesToolRef(), meta, q.Binding)
	require.NoError(t, err, "new rule read must traverse the actual Registry and record audit")
	require.Equal(t, commercetool.AuditStatusRecorded, result.AuditStatus)
	require.Equal(t, 1, qr.queryCalls)
	disclosed, err := materials.ParseCandidateRulesDisclosure(result.Output)
	require.NoError(t, err)
	require.Equal(t, meta.CallID, disclosed.Evidence.CallID)
	require.Empty(t, result.AIInvocationID)
	_, err = i.Invoke(context.Background(), materials.InspectToolRef(), meta, q.Binding)
	require.Error(t, err, "query binding cannot invoke the writable-material inspection path")
	f.organization = "other-org"
	_, err = i.Invoke(context.Background(), materials.RulesToolRef(), meta, q.Binding)
	require.Equal(t, commercetool.ErrorPermissionDenied, commercetool.CodeOf(err))
}

type fresh struct {
	revoked      bool
	organization string
	calls        int
}

func (f *fresh) ResolveFreshPrincipal(context.Context) (commercetool.Principal, error) {
	f.calls++
	if f.revoked {
		return commercetool.Principal{}, errors.New("revoked")
	}
	return commercetool.Principal{TenantID: f.organization, UserID: "actor", Roles: []string{"operator"}}, nil
}

type authorize struct{}

func (authorize) Authorize(context.Context, commercetool.Principal, commercetool.PermissionRequirement) error {
	return nil
}

type audit struct{ records int }

func (a *audit) RecordToolCall(context.Context, commercetool.AuditRecord) error {
	a.records++
	return nil
}

func inspector(t *testing.T) (*Invoker, *reader, *fresh, *audit) {
	t.Helper()
	product := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org", ProductKey: "product"}, Version: 1, PublicationID: "publication", Snapshot: catalog.ProductSnapshot{Title: "Original", Sources: []catalog.SourceRecord{{Type: "crawler", Metadata: map[string]string{"tenant_id": "private-source-metadata"}, Notes: []string{"disclosed source note"}}}}}
	rules := goods.OfficialRuleSnapshot{}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: "org", ProductKey: "product", TargetPlatform: "shein", SourceSnapshotVersion: 1}, Assets: []asset.ApprovedAsset{}}
	base := record.TargetRecord{ID: "record", TargetID: "target", Revision: 1, EffectiveVersion: 1, ProductHash: collection.Digest(product.Snapshot), RulesHash: collection.Digest(rules), InventoryHash: collection.Digest(inventory), Merchant: storecenter.ProductMerchantBinding{OrganizationID: "org", StoreID: "store", Site: "shein-us"}, Input: record.TargetInput{SourceID: "source", StoreID: "store", EffectiveVersion: 1}}
	base.Source.ID = "source"
	base.Source.Source = collection.Source{ProductKey: "product", PublicationID: "publication", Version: 1}
	i, err := materials.NewInspection(base, product, rules, inventory, time.Now().UTC(), time.Now().UTC())
	require.NoError(t, err, "%v", errors.Unwrap(err))
	r, f, a := &reader{value: i}, &fresh{organization: "org"}, &audit{}
	invoker, err := NewInvoker(r, f, AgentDefinition(), commercetool.InvocationDependencies{Authorizer: authorize{}, Recorder: a, Tracer: otel.Tracer("materials-test"), Now: time.Now, AuditTimeout: time.Second})
	require.NoError(t, err)
	return invoker, r, f, a
}

func invoke(i *Invoker, binding agent.Binding) (commercetool.Result, error) {
	return i.Invoke(context.Background(), Definition().Ref, commercetool.CallMetadata{CallID: "call", AgentID: materials.AgentID, AgentVersion: materials.AgentVersion, AgentRunID: "run", BusinessTaskID: binding.ContextID}, binding)
}

func TestInspectToolUsesFreshAuthorizationAndDisclosesNoTenantAuthority(t *testing.T) {
	i, r, f, a := inspector(t)
	result, err := invoke(i, r.value.Binding)
	require.NoError(t, err, "%v", errors.Unwrap(err))
	require.Equal(t, commercetool.AuditStatusRecorded, result.AuditStatus)
	require.Empty(t, result.AIInvocationID, "an unmetered read must not mint a model invocation")
	var disclosed materials.Disclosure
	require.NoError(t, json.Unmarshal(result.Output, &disclosed))
	require.Equal(t, r.value.EvidenceID, disclosed.EvidenceID)
	require.NotContains(t, string(result.Output), "tenant_id")
	require.NotContains(t, string(result.Output), "private-source-metadata")
	require.Contains(t, string(result.Output), "disclosed source note")
	require.Equal(t, "private-source-metadata", r.value.Product.Snapshot.Sources[0].Metadata["tenant_id"], "projection must not mutate/hash-remap canonical facts")
	require.Equal(t, 1, a.records)
	f.revoked = true
	_, err = invoke(i, r.value.Binding)
	require.Error(t, err)
	require.Equal(t, 2, f.calls)
	require.Equal(t, 1, r.calls)
}

func TestInspectToolRejectsWrongTenantBindingAndArbitraryTool(t *testing.T) {
	i, r, f, _ := inspector(t)
	f.organization = "other-org"
	_, err := invoke(i, r.value.Binding)
	require.Equal(t, commercetool.ErrorPermissionDenied, commercetool.CodeOf(err))
	f.organization = "org"
	b := r.value.Binding
	b.ProductKey = "other-product"
	_, err = invoke(i, b)
	require.Error(t, err)
	_, err = i.Invoke(context.Background(), commercetool.ToolRef{ID: "publish", Version: "v1.0.0"}, commercetool.CallMetadata{}, r.value.Binding)
	require.Equal(t, commercetool.ErrorToolNotAllowed, commercetool.CodeOf(err))
}
