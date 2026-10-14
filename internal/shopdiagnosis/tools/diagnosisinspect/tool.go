// Package diagnosisinspect binds the bounded shop evidence reader to the original
// Commerce Tool registry, fresh principal, authorization and durable audit.
package diagnosisinspect

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/invopop/jsonschema"
	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	c "task-processor/internal/operationscockpit"
	"task-processor/internal/shopdiagnosis"
	"time"
)

func AgentDefinition() commercetool.AgentDefinition {
	return commercetool.AgentDefinition{ID: agentconfig.DiagnosisAgentID, Version: agentconfig.DiagnosisAgentVersion, AllowedTools: []commercetool.ToolRef{agentconfig.DiagnosisInspectToolRef()}}
}

type input struct {
	Binding agent.Binding `json:"binding"`
}

func Definition() commercetool.Definition {
	reflector := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	in, _ := json.Marshal(reflector.Reflect(input{}))
	return commercetool.Definition{Ref: agentconfig.DiagnosisInspectToolRef(), Capability: "shop.operations-diagnosis", Owner: "shop.operations-diagnosis", Description: "Read exact authorized saved shop financial and operating facts for advisory diagnosis; no business mutation.", InputSchema: in, OutputSchema: OutputSchema(), Risk: commercetool.RiskRead, Permission: commercetool.PermissionRequirement{Permission: authz.PermissionLocalAgentWrite}, SideEffects: commercetool.SideEffectPolicy{Mode: commercetool.SideEffectNone}, Idempotency: commercetool.IdempotencyPolicy{Mode: commercetool.IdempotencyDeterministic}, Timeout: commercetool.TimeoutPolicy{Duration: 15 * time.Second}, Retry: commercetool.RetryPolicy{Owner: commercetool.RetryOwnerCaller}, Usage: commercetool.UsagePolicy{Owner: commercetool.UsageOwnerUnmetered}}
}

func OutputSchema() json.RawMessage {
	reflector := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	raw, _ := json.Marshal(reflector.Reflect(shopdiagnosis.Inspection{}))
	return raw
}

type FreshPrincipalResolver interface {
	ResolveFreshPrincipal(context.Context) (commercetool.Principal, error)
}
type freshDelegate struct{ fresh FreshPrincipalResolver }

func (d freshDelegate) ResolvePrincipal(ctx context.Context) (commercetool.Principal, error) {
	return d.fresh.ResolveFreshPrincipal(ctx)
}

type Invoker struct{ tools *commercetool.BoundToolSet }

func NewInvoker(reader shopdiagnosis.InspectionReader, fresh FreshPrincipalResolver, deps commercetool.InvocationDependencies) (*Invoker, error) {
	if reader == nil || fresh == nil {
		return nil, agent.ErrUnavailable
	}
	registry, err := commercetool.NewRegistry(commercetool.Tool{Definition: Definition(), Executor: executor{reader}})
	if err != nil {
		return nil, err
	}
	deps.PrincipalResolver = freshDelegate{fresh}
	bound, err := registry.Bind(AgentDefinition(), deps)
	if err != nil {
		return nil, err
	}
	return &Invoker{bound}, nil
}
func (i *Invoker) Invoke(ctx context.Context, ref commercetool.ToolRef, metadata commercetool.CallMetadata, binding agent.Binding) (commercetool.Result, error) {
	if i == nil || i.tools == nil || ref != agentconfig.DiagnosisInspectToolRef() || !binding.Valid() || binding.ContextKind != agent.StoreDiagnosisContextKind {
		return commercetool.Result{}, readError(agent.ErrInvalid)
	}
	raw, err := json.Marshal(input{binding})
	if err != nil {
		return commercetool.Result{}, readError(err)
	}
	return i.tools.Invoke(ctx, commercetool.Call{Tool: ref, Metadata: metadata, Arguments: raw})
}

type executor struct {
	reader shopdiagnosis.InspectionReader
}

func (e executor) Execute(ctx context.Context, envelope commercetool.ExecutionEnvelope, raw json.RawMessage) (commercetool.ExecutionResult, error) {
	var in input
	strict, err := sigjson.UnmarshalStrict(raw, &in)
	if err != nil || len(strict) > 0 || !in.Binding.Valid() || in.Binding.ContextKind != agent.StoreDiagnosisContextKind {
		return commercetool.ExecutionResult{}, readError(agent.ErrInvalid)
	}
	out, err := e.reader.Inspect(ctx, in.Binding)
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if out.Scope.OrganizationID != envelope.Principal().TenantID || out.Scope.ActorID != envelope.Principal().UserID || !shopdiagnosis.InspectionMatchesBinding(out, in.Binding) {
		return commercetool.ExecutionResult{}, readError(c.ErrForbidden)
	}
	wire, err := json.Marshal(out)
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if _, err = shopdiagnosis.ParseInspection(wire); err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	return commercetool.ExecutionResult{Output: wire}, nil
}
func readError(err error) error {
	code := commercetool.ErrorDependencyUnavailable
	switch {
	case errors.Is(err, c.ErrForbidden):
		code = commercetool.ErrorPermissionDenied
	case errors.Is(err, c.ErrNotFound):
		code = commercetool.ErrorNotFound
	case errors.Is(err, c.ErrConflict), errors.Is(err, agent.ErrConflict):
		code = commercetool.ErrorFailedPrecondition
	case errors.Is(err, agent.ErrInvalid):
		code = commercetool.ErrorInvalidInput
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		code = commercetool.ErrorDeadlineExceeded
	}
	return commercetool.NewError(code, "shop operating evidence unavailable", err)
}

var _ agent.ToolGateway = (*Invoker)(nil)
