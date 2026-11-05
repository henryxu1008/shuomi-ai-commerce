// Package researchinspect binds the bounded candidate reader to the original
// Commerce Tool registry, fresh principal, authorization and durable audit.
package researchinspect

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
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research/analysis"
	"time"
)

func AgentDefinition() commercetool.AgentDefinition {
	return commercetool.AgentDefinition{ID: agentconfig.ResearchAgentID, Version: agentconfig.ResearchAgentVersion, AllowedTools: []commercetool.ToolRef{agentconfig.ResearchInspectToolRef()}}
}

type input struct {
	Binding agent.Binding `json:"binding"`
}

func Definition() commercetool.Definition {
	reflector := jsonschema.Reflector{Anonymous: true, ExpandedStruct: true}
	in, _ := json.Marshal(reflector.Reflect(input{}))
	return commercetool.Definition{Ref: agentconfig.ResearchInspectToolRef(), Capability: "product.research", Owner: "product.research", Description: "Read exact private saved candidate facts for advisory comparison; no purchase or product mutation.", InputSchema: in, OutputSchema: OutputSchema(), Risk: commercetool.RiskRead, Permission: commercetool.PermissionRequirement{Permission: authz.PermissionLocalAgentWrite}, SideEffects: commercetool.SideEffectPolicy{Mode: commercetool.SideEffectNone}, Idempotency: commercetool.IdempotencyPolicy{Mode: commercetool.IdempotencyDeterministic}, Timeout: commercetool.TimeoutPolicy{Duration: 15 * time.Second}, Retry: commercetool.RetryPolicy{Owner: commercetool.RetryOwnerCaller}, Usage: commercetool.UsagePolicy{Owner: commercetool.UsageOwnerUnmetered}}
}

type FreshPrincipalResolver interface {
	ResolveFreshPrincipal(context.Context) (commercetool.Principal, error)
}
type freshDelegate struct{ fresh FreshPrincipalResolver }

func (d freshDelegate) ResolvePrincipal(ctx context.Context) (commercetool.Principal, error) {
	return d.fresh.ResolveFreshPrincipal(ctx)
}

type Invoker struct{ tools *commercetool.BoundToolSet }

func NewInvoker(reader analysis.InspectionReader, fresh FreshPrincipalResolver, deps commercetool.InvocationDependencies) (*Invoker, error) {
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
	if i == nil || i.tools == nil || ref != agentconfig.ResearchInspectToolRef() || !binding.Valid() || binding.ContextKind != agent.ResearchSelectionContextKind {
		return commercetool.Result{}, readError(agent.ErrInvalid)
	}
	raw, err := json.Marshal(input{binding})
	if err != nil {
		return commercetool.Result{}, readError(err)
	}
	return i.tools.Invoke(ctx, commercetool.Call{Tool: ref, Metadata: metadata, Arguments: raw})
}

type executor struct{ reader analysis.InspectionReader }

func (e executor) Execute(ctx context.Context, envelope commercetool.ExecutionEnvelope, raw json.RawMessage) (commercetool.ExecutionResult, error) {
	var in input
	strict, err := sigjson.UnmarshalStrict(raw, &in)
	if err != nil || len(strict) > 0 || !in.Binding.Valid() || in.Binding.ContextKind != agent.ResearchSelectionContextKind {
		return commercetool.ExecutionResult{}, readError(agent.ErrInvalid)
	}
	out, err := e.reader.Inspect(ctx, in.Binding)
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if out.Scope.OrganizationID != envelope.Principal().TenantID || out.Scope.ActorID != envelope.Principal().UserID || !analysis.InspectionMatchesBinding(out, in.Binding) {
		return commercetool.ExecutionResult{}, readError(collection.ErrForbidden)
	}
	wire, err := json.Marshal(out)
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if _, err = analysis.ParseInspection(wire); err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	return commercetool.ExecutionResult{Output: wire}, nil
}
func readError(err error) error {
	code := commercetool.ErrorDependencyUnavailable
	switch {
	case errors.Is(err, collection.ErrForbidden):
		code = commercetool.ErrorPermissionDenied
	case errors.Is(err, collection.ErrNotFound):
		code = commercetool.ErrorNotFound
	case errors.Is(err, collection.ErrConflict), errors.Is(err, agent.ErrConflict):
		code = commercetool.ErrorFailedPrecondition
	case errors.Is(err, agent.ErrInvalid):
		code = commercetool.ErrorInvalidInput
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		code = commercetool.ErrorDeadlineExceeded
	}
	return commercetool.NewError(code, "selected candidate facts unavailable", err)
}

var _ agent.ToolGateway = (*Invoker)(nil)
