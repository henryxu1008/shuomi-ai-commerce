package selectioninspect

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/selection"
	"testing"
	"time"
)

type testReader struct {
	out     selection.Inspection
	binding agent.Binding
	calls   int
}

func (r *testReader) Inspect(_ context.Context, b agent.Binding) (selection.Inspection, error) {
	r.calls++
	if b != r.binding {
		return selection.Inspection{}, agent.ErrConflict
	}
	return r.out, nil
}

type testFresh struct {
	revoked bool
	org     string
}

func (f *testFresh) ResolveFreshPrincipal(context.Context) (commercetool.Principal, error) {
	if f.revoked {
		return commercetool.Principal{}, collection.ErrForbidden
	}
	return commercetool.Principal{TenantID: f.org, UserID: "actor", Roles: []string{"operator"}}, nil
}

type testAuthorize struct{}

func (testAuthorize) Authorize(context.Context, commercetool.Principal, commercetool.PermissionRequirement) error {
	return nil
}

type testAudit struct{ calls int }

func (a *testAudit) RecordToolCall(context.Context, commercetool.AuditRecord) error {
	a.calls++
	return nil
}

func TestSelectionToolTraversesOriginalRegistryAuditAndFreshScope(t *testing.T) {
	b := agent.Binding{ContextKind: agent.SelectionContextKind, ContextID: uuid.NewString(), TargetPlatform: "product"}
	for n := 0; n < 2; n++ {
		b.Selection[n] = agent.SelectionProductRef{ItemID: uuid.NewString(), ItemRevision: "1", ProductKey: string(rune('a' + n)), CatalogVersion: "1", PublicationID: "pub"}
	}
	if b.Selection[0].ItemID > b.Selection[1].ItemID {
		b.Selection[0], b.Selection[1] = b.Selection[1], b.Selection[0]
	}
	r := &testReader{binding: b, out: selection.Inspection{Scope: collection.Scope{"org", "actor", "member"}, Products: []selection.Product{{Ref: b.Selection[0], SourceKind: "own", Facts: []selection.Fact{}, Missing: []string{"price"}}, {Ref: b.Selection[1], SourceKind: "own", Facts: []selection.Fact{}, Missing: []string{"price"}}}}}
	fresh, audit := &testFresh{org: "org"}, &testAudit{}
	invoker, err := NewInvoker(r, fresh, commercetool.InvocationDependencies{Authorizer: testAuthorize{}, Recorder: audit, Tracer: otel.Tracer("selection-test"), Now: time.Now, AuditTimeout: time.Second})
	require.NoError(t, err)
	def := AgentDefinition()
	meta := commercetool.CallMetadata{CallID: "read", AgentID: def.ID, AgentVersion: def.Version, AgentRunID: "run", BusinessTaskID: b.ContextID}
	result, err := invoker.Invoke(context.Background(), Definition().Ref, meta, b)
	require.NoError(t, err)
	require.Equal(t, commercetool.AuditStatusRecorded, result.AuditStatus)
	require.Equal(t, 1, r.calls)
	require.Equal(t, 1, audit.calls)
	fresh.revoked = true
	_, err = invoker.Invoke(context.Background(), Definition().Ref, meta, b)
	require.Error(t, err)
	require.Equal(t, 1, r.calls)
	fresh.revoked = false
	fresh.org = "other"
	_, err = invoker.Invoke(context.Background(), Definition().Ref, meta, b)
	require.Error(t, err)
}
