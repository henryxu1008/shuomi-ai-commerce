package materialsinspect

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"

	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/listing/materials"

	sigjson "sigs.k8s.io/json"
)

type FreshPrincipalResolver interface {
	ResolveFreshPrincipal(context.Context) (commercetool.Principal, error)
}
type freshDelegate struct{ fresh FreshPrincipalResolver }

func (d freshDelegate) ResolvePrincipal(ctx context.Context) (commercetool.Principal, error) {
	return d.fresh.ResolveFreshPrincipal(ctx)
}

type Invoker struct{ tools *commercetool.BoundToolSet }
type input struct {
	Binding agent.Binding `json:"binding"`
}

func NewInvoker(reader materials.InspectionReader, fresh FreshPrincipalResolver, definition commercetool.AgentDefinition, deps commercetool.InvocationDependencies) (*Invoker, error) {
	if nilDependency(reader) || nilDependency(fresh) || definition.ID != materials.AgentID || definition.Version != materials.AgentVersion || len(definition.AllowedTools) != 1 || definition.AllowedTools[0] != Definition().Ref {
		return nil, errors.New("exact material reader, fresh identity and fixed agent definition are required")
	}
	registry, err := commercetool.NewRegistry(commercetool.Tool{Definition: Definition(), Executor: executor{reader}})
	if err != nil {
		return nil, err
	}
	deps.PrincipalResolver = freshDelegate{fresh}
	bound, err := registry.Bind(definition, deps)
	if err != nil {
		return nil, err
	}
	return &Invoker{tools: bound}, nil
}

// Invoke accepts only the binding minted in the immutable server request;
// model output cannot supply arguments or another Tool.
func (i *Invoker) Invoke(ctx context.Context, ref commercetool.ToolRef, metadata commercetool.CallMetadata, binding agent.Binding) (commercetool.Result, error) {
	if ref != Definition().Ref {
		return commercetool.Result{}, commercetool.NewError(commercetool.ErrorToolNotAllowed, "platform material tool is not allowed", nil)
	}
	if ctx == nil || i == nil || i.tools == nil {
		return commercetool.Result{}, readError(materials.ErrInspectionChanged)
	}
	ctx, cancel := context.WithTimeout(ctx, Definition().Timeout.Duration)
	defer cancel()
	raw, err := json.Marshal(input{binding})
	if err != nil {
		return commercetool.Result{}, readError(err)
	}
	return i.tools.Invoke(ctx, commercetool.Call{Tool: ref, Metadata: metadata, Arguments: raw})
}

type executor struct{ reader materials.InspectionReader }

func (e executor) Execute(ctx context.Context, envelope commercetool.ExecutionEnvelope, raw json.RawMessage) (commercetool.ExecutionResult, error) {
	var in input
	strict, err := sigjson.UnmarshalStrict(raw, &in)
	if err != nil || len(strict) != 0 || !in.Binding.Valid() || in.Binding.ContextKind != "listing-target-record" || in.Binding.TargetPlatform != "shein" {
		return commercetool.ExecutionResult{}, commercetool.NewError(commercetool.ErrorInvalidInput, "invalid exact platform material binding", nil)
	}
	if err = ctx.Err(); err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	i, err := e.reader.ReadMaterialsInspection(ctx, in.Binding)
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	if i.Binding != in.Binding || i.Product.Identity.TenantID != envelope.Principal().TenantID {
		return commercetool.ExecutionResult{}, commercetool.NewError(commercetool.ErrorPermissionDenied, "material owner differs from current organization", nil)
	}
	disclosed, err := i.Disclose()
	if err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	output, err := json.Marshal(disclosed)
	if err != nil || len(output) > materials.MaxInspectionBytes {
		return commercetool.ExecutionResult{}, readError(materials.ErrInspectionTooLarge)
	}
	if err = ctx.Err(); err != nil {
		return commercetool.ExecutionResult{}, readError(err)
	}
	return commercetool.ExecutionResult{Output: output}, nil
}

func readError(err error) error {
	code := commercetool.ErrorDependencyUnavailable
	if errors.Is(err, materials.ErrInspectionChanged) || errors.Is(err, materials.ErrInspectionTooLarge) {
		code = commercetool.ErrorFailedPrecondition
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		code = commercetool.ErrorDeadlineExceeded
	}
	return commercetool.NewError(code, "exact platform materials unavailable", err)
}
func nilDependency(v any) bool {
	if v == nil {
		return true
	}
	x := reflect.ValueOf(v)
	switch x.Kind() {
	case reflect.Pointer, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan:
		return x.IsNil()
	}
	return false
}

var _ agent.ToolGateway = (*Invoker)(nil)
