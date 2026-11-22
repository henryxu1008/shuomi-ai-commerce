package httpapi

import (
	"context"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/commercetool"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/commercetoolauth"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/contentagent/scriptcompute"
	reviewproposal "github.com/henryxu1008/shuomi-ai-commerce/internal/product/review/proposal"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/skill"
)

func (a *productAgentApplication) checkScriptTrialIdentity(identity agent.ExecutionIdentity) error {
	if allowed := a.config.ScriptTrialIdentity; allowed != nil && identity != *allowed {
		return skill.ErrForbidden
	}
	return nil
}

func (a *productAgentApplication) acquireScript(ctx context.Context, metadata commercetool.CallMetadata, request agent.Request) (skill.Package, func(), error) {
	if a.script == nil || a.skills == nil || a.configuration == nil || a.store == nil || !request.ValidScript() || !request.ScriptSelected() || request.ScriptDescriptor != a.scriptDescriptor || a.definition.ID != agentconfig.ContentAgentID {
		return skill.Package{}, nil, agent.ErrUnavailable
	}
	identity, err := a.currentModeIdentity(ctx)
	if err != nil {
		return skill.Package{}, nil, err
	}
	if err := a.checkScriptTrialIdentity(identity); err != nil {
		return skill.Package{}, nil, err
	}
	scope, err := a.Authorize(ctx, request.Binding)
	if err != nil || scope != (agent.Scope{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID}) {
		return skill.Package{}, nil, agent.ErrInvalid
	}
	snapshot, err := a.configuration.LoadSnapshot(ctx, scope, request.ConfigurationSnapshotRef)
	if err != nil {
		return skill.Package{}, nil, err
	}
	command := request
	command.ContextSnapshotRef = agent.ContextSnapshotRef{}
	command.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{}
	if snapshot.Request != command || snapshot.MemberID != identity.MemberID || snapshot.AgentVersion != a.definition.Version {
		return skill.Package{}, nil, agent.ErrConflict
	}
	mode, err := a.configuration.CheckRegisteredTextExecutionMode(ctx, identity, request.ConfigurationSnapshotRef, request.ModeSnapshotRef)
	if err != nil {
		return skill.Package{}, nil, err
	}
	if mode.Mode != agent.ModeDefault && mode.Mode != agent.ModeReadOnly {
		return skill.Package{}, nil, agent.ErrManualReviewRequired
	}
	run, found, err := a.store.Lookup(ctx, scope, request.Binding, request.Key)
	if err != nil {
		return skill.Package{}, nil, err
	}
	if !found || run.State.RunID != metadata.AgentRunID || run.State.Request != request || run.State.Phase != agent.Running {
		return skill.Package{}, nil, agent.ErrConflict
	}
	principal, err := a.skillPrincipal(ctx, identity)
	if err != nil {
		return skill.Package{}, nil, err
	}
	return a.skills.AcquireUse(ctx, principal, request.SkillVersionRef)
}

func (a *productAgentApplication) InvokeScript(ctx context.Context, metadata commercetool.CallMetadata, request agent.Request, candidate reviewproposal.Candidate) (commercetool.Result, error) {
	input, err := agent.ScriptInputBytes(request, candidate)
	if err != nil {
		return commercetool.Result{}, err
	}
	pkg, release, err := a.acquireScript(ctx, metadata, request)
	if err != nil {
		return commercetool.Result{}, err
	}
	defer release()
	if pkg.Execution == nil || pkg.Execution.Profile != request.ScriptDescriptor.Profile || pkg.Digest != request.SkillVersionRef.Digest {
		return commercetool.Result{}, agent.ErrConflict
	}
	if !listingAgentContext(ctx) && !scheduledExecutionContext(ctx) {
		identity, err := a.freshIdentity(ctx)
		if err != nil {
			return commercetool.Result{}, err
		}
		subject := agent.ExecutionIdentity{OrganizationID: identity.EffectiveOrganizationID, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID}
		if err := a.checkScriptTrialIdentity(subject); err != nil {
			return commercetool.Result{}, err
		}
		capability, ok := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
		if !ok {
			return commercetool.Result{}, agent.ErrInvalid
		}
		ctx = authidentity.WithAuthenticatedIdentity(ctx, identity)
		ctx = commercetoolauth.WithOrganizationRequest(ctx, commercetoolauth.OrganizationRequest{Identity: identity, BearerToken: capability.bearerToken, RequestedOrganizationID: identity.TenantID})
	}
	task := agent.ScriptTask{Descriptor: request.ScriptDescriptor, PackageDigest: pkg.Digest, Entrypoint: pkg.Execution.Entrypoint, Input: input, RunID: metadata.AgentRunID, CallID: metadata.CallID}
	for _, file := range pkg.Files {
		task.Files = append(task.Files, agent.ScriptFile{Path: file.Path, Text: file.Text})
	}
	result, err := scriptcompute.Invoke(ctx, a.definition, a.scriptDependencies, metadata, task, a.script)
	release()
	if err != nil {
		return result, err
	}
	// Re-resolve live permissions/mode after remote execution, before any result
	// can enter validation/Review. Cleanup never substitutes for authorization.
	_, again, checkErr := a.acquireScript(ctx, metadata, request)
	if checkErr != nil {
		return result, checkErr
	}
	again()
	return result, nil
}
