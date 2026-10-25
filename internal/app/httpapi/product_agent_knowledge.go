package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/trace"

	sigjson "sigs.k8s.io/json"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/authidentity"
	"task-processor/internal/integration/commercetoolauth"
	"task-processor/internal/knowledge"
	"task-processor/internal/product/review"
	reviewproposal "task-processor/internal/product/review/proposal"
	skill "task-processor/internal/skill"
)

type productAgentRequestBody struct {
	ManualReviewOnly     bool                             `json:"manualReviewOnly,omitempty"`
	OwnSource            *ownSourceSelection              `json:"ownSource,omitempty"`
	ContentSelection     *reviewproposal.ContentSelection `json:"contentSelection,omitempty"`
	TemplateSelection    json.RawMessage                  `json:"templateSelection,omitempty"`
	Revision             string                           `json:"revision,omitempty"`
	Feedback             string                           `json:"feedback,omitempty"`
	TargetPlatform       string                           `json:"targetPlatform,omitempty"`
	KnowledgeSelection   json.RawMessage                  `json:"knowledgeSelection,omitempty"`
	SkillSelection       json.RawMessage                  `json:"skillSelection,omitempty"`
	ExpectedModeRevision string                           `json:"expectedModeRevision,omitempty"`
}

type productSkillSelection struct {
	SkillID string `json:"skillId"`
	Version uint64 `json:"version"`
	Digest  string `json:"digest"`
}

func (b productAgentRequestBody) skillSelection(action string) (*productSkillSelection, error) {
	if len(b.SkillSelection) == 0 {
		return nil, nil
	}
	if action != "start" {
		return nil, agent.ErrInvalid
	}
	var s productSkillSelection
	violations, e := sigjson.UnmarshalStrict(b.SkillSelection, &s)
	ref := agent.SkillVersionRef{Kind: "skill-version-v1", ID: s.SkillID, Version: s.Version, Digest: s.Digest, OwnerKind: "PLATFORM", OwnerID: "official", PolicyVersion: "skill-method-v1"}
	if e != nil || len(violations) > 0 || !ref.Valid() {
		return nil, agent.ErrInvalid
	}
	return &s, nil
}

func (b productAgentRequestBody) templateSelection(action string) (*agentconfig.TemplateRef, error) {
	if len(b.TemplateSelection) == 0 {
		return nil, nil
	}
	if action != "start" {
		return nil, agent.ErrInvalid
	}
	var ref agentconfig.TemplateRef
	violations, e := sigjson.UnmarshalStrict(b.TemplateSelection, &ref)
	n, ne := strconv.ParseUint(ref.Revision, 10, 63)
	if e != nil || len(violations) > 0 || !agentconfig.UUID(ref.TemplateID) || ne != nil || n == 0 || strconv.FormatUint(n, 10) != ref.Revision {
		return nil, agent.ErrInvalid
	}
	return &ref, nil
}

func (b productAgentRequestBody) knowledgeSelection(action string) (string, error) {
	if len(b.KnowledgeSelection) == 0 {
		return "", nil
	}
	if action != "start" {
		return "", agent.ErrInvalid
	}
	var selection struct {
		KnowledgeBaseID string `json:"knowledgeBaseId"`
	}
	violations, err := sigjson.UnmarshalStrict(b.KnowledgeSelection, &selection)
	if err != nil || len(violations) != 0 || !knowledge.ValidID(selection.KnowledgeBaseID) {
		return "", agent.ErrInvalid
	}
	return selection.KnowledgeBaseID, nil
}

// Reuse the existing verified, request-local bearer capability. No credential
// is copied into the Agent request, provenance, response or persistence.
func knowledgeRequestContext(ctx context.Context) (context.Context, error) {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	capability, bound := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability)
	if !ok || !bound || capability.actorID != identity.UserID || capability.effectiveOrganizationID != identity.TenantID ||
		identity.TenantID != identity.EffectiveOrganizationID {
		return ctx, knowledge.ErrForbidden
	}
	return commercetoolauth.WithOrganizationRequest(ctx, commercetoolauth.OrganizationRequest{Identity: identity, BearerToken: capability.bearerToken, RequestedOrganizationID: identity.TenantID}), nil
}

func (a *productAgentApplication) startRequest(ctx context.Context, binding agent.Binding, key, baseID string, template *agentconfig.TemplateRef, expectedModeRevision string) (agent.Request, error) {
	return a.startRequestWithProfile(ctx, binding, key, baseID, "", template, aicapability.ModelProfile{}, "", expectedModeRevision)
}

func (a *productAgentApplication) startRequestWithProfile(ctx context.Context, binding agent.Binding, key, baseID, expectedRevisionSet string, template *agentconfig.TemplateRef, expectedProfile aicapability.ModelProfile, goalSummary, expectedModeRevision string) (agent.Request, error) {
	return a.startTextRequest(ctx, binding, key, baseID, expectedRevisionSet, template, expectedProfile, goalSummary, expectedModeRevision, reviewproposal.ContentSelection{})
}

func (a *productAgentApplication) startTextRequest(ctx context.Context, binding agent.Binding, key, baseID, expectedRevisionSet string, template *agentconfig.TemplateRef, expectedProfile aicapability.ModelProfile, goalSummary, expectedModeRevision string, selection reviewproposal.ContentSelection) (agent.Request, error) {
	return a.startTextRequestWithSkill(ctx, binding, key, baseID, expectedRevisionSet, template, expectedProfile, goalSummary, expectedModeRevision, selection, nil)
}
func (a *productAgentApplication) startTextRequestWithSkill(ctx context.Context, binding agent.Binding, key, baseID, expectedRevisionSet string, template *agentconfig.TemplateRef, expectedProfile aicapability.ModelProfile, goalSummary, expectedModeRevision string, selection reviewproposal.ContentSelection, method *productSkillSelection, manualOnly ...bool) (agent.Request, error) {
	if len(manualOnly) > 1 || len(manualOnly) == 1 && manualOnly[0] && (a.definition.ID != agentconfig.ContentAgentID || goalSummary != "") {
		return agent.Request{}, agent.ErrInvalid
	}
	if err := a.checkContentTitleSelection(selection); err != nil {
		return agent.Request{}, err
	}
	if !agent.ValidGoalSummary(goalSummary) {
		return agent.Request{}, agent.ErrInvalid
	}
	request := agent.Request{Key: key, Binding: binding, GoalSummary: goalSummary, PolicyVersion: "title-review-v1", PromptVersion: "product-title-agent-v1", Limits: a.config.Limits}
	request.ManualReviewOnly = len(manualOnly) == 1 && manualOnly[0]
	content := a.definition.ID == agentconfig.ContentAgentID
	if method != nil && (!content || !selection.TitleOnly()) {
		return agent.Request{}, agent.ErrInvalid
	}
	if content {
		if !selection.Valid() {
			return agent.Request{}, agent.ErrInvalid
		}
		request.ContentSelection, request.PolicyVersion, request.PromptVersion = selection, reviewproposal.ContentPolicyVersion, agentconfig.ContentPromptVersion
	} else if !selection.Absent() {
		return agent.Request{}, agent.ErrInvalid
	}
	scope, err := a.requestOrExecutionScope(ctx)
	if err != nil {
		return agent.Request{}, err
	}
	if a.configuration == nil || a.store == nil {
		return agent.Request{}, agentconfig.ErrUnavailable
	}
	identity, err := a.currentModeIdentity(ctx)
	if err != nil {
		return agent.Request{}, err
	}
	if _, err = agentconfig.ParseModeRevision(expectedModeRevision); err != nil {
		return agent.Request{}, agentconfig.ErrPrecondition
	}
	if baseID != "" && !content {
		request.PromptVersion = "product-title-agent-knowledge-v1"
	}
	input := agentconfig.StartCommand{Scope: scope, MemberID: identity.MemberID, ExpectedModeRevision: expectedModeRevision, AgentID: a.definition.ID, AgentVersion: a.definition.Version, KnowledgeBaseID: baseID, Template: template, Request: request}
	existing, found, readErr := a.store.Lookup(ctx, input.Scope, binding, key)
	if readErr != nil {
		return agent.Request{}, readErr
	}
	if found {
		if method != nil {
			ref := existing.State.Request.SkillVersionRef
			if !ref.Valid() || ref.ID != method.SkillID || ref.Version != method.Version || ref.Digest != method.Digest {
				return agent.Request{}, agentconfig.ErrConflict
			}
			input.Request.SkillVersionRef = ref
		}
		frozen, err := a.configuration.Match(ctx, input, existing.State.Request.ConfigurationSnapshotRef)
		if err != nil {
			return agent.Request{}, err
		}
		if expectedProfile.Validate() == nil && frozen.ExecutionModelProfile != expectedProfile {
			return agent.Request{}, agentconfig.ErrConflict
		}
		if baseID != "" {
			if a.context == nil {
				return agent.Request{}, knowledge.ErrUnavailable
			}
			ref := existing.State.Request.ContextSnapshotRef
			command := knowledge.ContextRequest{Scope: knowledge.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, Binding: binding, Key: key, Selection: "knowledge-base:" + baseID, PolicyVersion: knowledge.ContextPolicyVersion, ExpectedRevisionSetDigest: expectedRevisionSet}
			if err := a.context.ValidateMaterializedRequest(ctx, command, knowledge.ContextSnapshotRef{Kind: ref.Kind, ID: ref.ID, Digest: ref.Digest}); err != nil {
				return agent.Request{}, err
			}
		}
		if err := a.checkManualReviewMode(ctx, identity, existing.State.Request); err != nil {
			return agent.Request{}, err
		}
		return existing.State.Request, nil
	}
	if method != nil {
		if a.skills == nil {
			return agent.Request{}, skill.ErrUnavailable
		}
		p, e := a.skillPrincipal(ctx, identity)
		if e != nil {
			return agent.Request{}, e
		}
		ref, e := a.skills.ResolveSelection(ctx, p, method.SkillID, method.Version, method.Digest)
		if e != nil {
			return agent.Request{}, e
		}
		input.Request.SkillVersionRef = ref
	}
	if a.selectTitleProfile == nil {
		return agent.Request{}, agent.ErrUnavailable
	}
	profile, err := a.selectTitleProfile(ctx, scope.OrganizationID)
	if err != nil || profile.Validate() != nil || (expectedProfile.Validate() == nil && profile != expectedProfile) {
		return agent.Request{}, agent.ErrUnavailable
	}
	input.ExecutionModelProfile = profile
	snapshot, err := a.configuration.Prepare(ctx, input)
	if err != nil {
		return agent.Request{}, err
	}
	request = snapshot.Request
	request.ConfigurationSnapshotRef = agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: snapshot.ID, Digest: snapshot.Digest}
	if err := a.checkManualReviewMode(ctx, identity, request); err != nil {
		return agent.Request{}, err
	}
	if baseID != "" {
		if a.context == nil {
			return agent.Request{}, knowledge.ErrUnavailable
		}
		command := knowledge.ContextRequest{
			Scope: knowledge.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, Binding: binding, Key: key,
			Selection: "knowledge-base:" + baseID, PolicyVersion: knowledge.ContextPolicyVersion, ExpectedRevisionSetDigest: expectedRevisionSet,
		}
		ref, err := a.context.Materialize(ctx, command)
		if err != nil {
			return agent.Request{}, err
		}
		request.ContextSnapshotRef = agent.ContextSnapshotRef{Kind: ref.Kind, ID: ref.ID, Digest: ref.Digest}
		if !content {
			request.PromptVersion = "product-title-agent-knowledge-v1"
		}
	}
	if a.model == nil {
		return agent.Request{}, agent.ErrUnavailable
	}
	// Reserve the real UUID width without creating a run. Trace and the empty
	// first state match execution; the quote is never reserved or dispatched.
	initial := agent.State{Request: request, RunID: strings.Repeat("0", 36)}
	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		initial.TraceID = span.TraceID().String()
	}
	if _, err := a.model.Quote(ctx, initial.ModelInput(a.definition)); err != nil {
		if errors.Is(err, aicapability.ErrTextEnvelope) {
			return agent.Request{}, knowledge.ErrContextTooLarge
		}
		return agent.Request{}, err
	}
	return request, nil
}

func (a *productAgentApplication) checkContentTitleSelection(selection reviewproposal.ContentSelection) error {
	if a.config.ContentTitleOnly && (a.definition.ID != agentconfig.ContentAgentID || !selection.TitleOnly()) {
		return agent.ErrInvalid
	}
	return nil
}

func agentContextProvenance(state agent.State) (*review.ContextProvenanceRef, error) {
	ref := state.Request.ContextSnapshotRef
	if !agent.ValidContextCitations(ref, state.ContextCitationRefs) {
		return nil, agent.ErrConflict
	}
	if ref.Absent() {
		return nil, nil
	}
	if ref.Kind != knowledge.ContextKind || !knowledge.ValidID(ref.ID) {
		return nil, agent.ErrConflict
	}
	provenance := &review.ContextProvenanceRef{Kind: ref.Kind, BundleID: ref.ID, BundleDigest: ref.Digest, OriginAgentRunID: state.RunID, CitationIDs: []string{}}
	for _, citation := range state.ContextCitationRefs {
		provenance.CitationIDs = append(provenance.CitationIDs, citation.ID)
	}
	if !provenance.Valid() {
		return nil, agent.ErrConflict
	}
	return provenance, nil
}
