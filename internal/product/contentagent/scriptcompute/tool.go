// Package scriptcompute binds fixed candidate postprocessing to the current
// Commerce registry. Neither scripts nor models register callable tools.
package scriptcompute

import (
	"context"
	"encoding/json"

	"task-processor/internal/agent"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
)

func Definition() commercetool.Definition {
	return commercetool.Definition{
		Ref: commercetool.ToolRef{ID: "skill.script.compute", Version: "v1.0.0"}, Capability: "skill.script.compute", Owner: "product.content", Description: "Fixed offline postprocessing of one authorized frozen title candidate.",
		InputSchema:  json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["inputDigest","packageDigest","descriptorDigest"],"properties":{"inputDigest":{"type":"string","pattern":"^[0-9a-f]{64}$"},"packageDigest":{"type":"string","pattern":"^[0-9a-f]{64}$"},"descriptorDigest":{"type":"string","pattern":"^[0-9a-f]{64}$"}}}`),
		OutputSchema: json.RawMessage(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","additionalProperties":false,"required":["title","receipt"],"properties":{"title":{"type":"string","maxLength":8192},"receipt":{"type":"object","additionalProperties":false,"required":["outcome","cleanup","reason","inputDigest","outputDigest","descriptorDigest"],"properties":{"outcome":{"type":"string","enum":["succeeded","failed","outcome_unknown","not_dispatched"]},"cleanup":{"type":"string","enum":["completed","pending","not_needed"]},"reason":{"type":"string","maxLength":64},"inputDigest":{"type":"string","pattern":"^[0-9a-f]{64}$"},"outputDigest":{"type":"string","pattern":"^(?:|[0-9a-f]{64})$"},"descriptorDigest":{"type":"string","pattern":"^[0-9a-f]{64}$"}}}}}`),
		Risk:         commercetool.RiskCompute, Permission: commercetool.PermissionRequirement{Permission: authz.PermissionLocalAgentWrite}, SideEffects: commercetool.SideEffectPolicy{Mode: commercetool.SideEffectNone}, Idempotency: commercetool.IdempotencyPolicy{Mode: commercetool.IdempotencyDeterministic}, Timeout: commercetool.TimeoutPolicy{Duration: agent.ScriptStepTimeout}, Retry: commercetool.RetryPolicy{Owner: commercetool.RetryOwnerCaller}, Usage: commercetool.UsagePolicy{Owner: commercetool.UsageOwnerUnmetered},
	}
}

// Invoke's task is supplied only by the authorized fixed node. Arguments carry
// hashes for the existing audit contract; code/input and credentials are never
// retained in audit. The fixed tool is bound to a request-local execution, not
// dynamically registered for each saved script.
func Invoke(ctx context.Context, definition commercetool.AgentDefinition, deps commercetool.InvocationDependencies, metadata commercetool.CallMetadata, task agent.ScriptTask, executor agent.ScriptExecutor) (commercetool.Result, error) {
	if executor == nil {
		return commercetool.Result{}, agent.ErrUnavailable
	}
	tool := Definition()
	allowed := 0
	for _, ref := range definition.AllowedTools {
		if ref == tool.Ref {
			allowed++
		}
	}
	if allowed != 1 {
		return commercetool.Result{}, agent.ErrInvalid
	}
	arguments, _ := json.Marshal(map[string]string{"inputDigest": agent.ScriptDigest(task.Input), "packageDigest": task.PackageDigest, "descriptorDigest": task.Descriptor.Digest()})
	registered, err := commercetool.NewRegistry(commercetool.Tool{Definition: tool, Executor: commercetool.ExecutorFunc(func(call context.Context, _ commercetool.ExecutionEnvelope, _ json.RawMessage) (commercetool.ExecutionResult, error) {
		result, err := executor.Execute(call, task)
		raw, marshalErr := json.Marshal(result)
		if err == nil {
			err = marshalErr
		}
		return commercetool.ExecutionResult{Output: raw}, err
	})})
	if err != nil {
		return commercetool.Result{}, err
	}
	definition.AllowedTools = []commercetool.ToolRef{tool.Ref}
	bound, err := registered.Bind(definition, deps)
	if err != nil {
		return commercetool.Result{}, err
	}
	return bound.Invoke(ctx, commercetool.Call{Tool: tool.Ref, Metadata: metadata, Arguments: arguments})
}
