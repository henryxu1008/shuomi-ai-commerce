package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	confighttp "task-processor/internal/agentconfig/httpapi"
	"task-processor/internal/agentcustomization"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
)

type privateQualityMode struct {
	repository agentconfig.ModeRepository
	resolve    func(context.Context, agentconfig.ModeSubject, bool) (agent.ExecutionIdentity, agentconfig.ResolvedModeSubject, error)
	identity   func(context.Context) (agent.ExecutionIdentity, error)
}

func privateQualityExecutionIdentity(ctx context.Context, h *confighttp.Handler) (agent.ExecutionIdentity, error) {
	if h == nil || h.Authorize == nil {
		return agent.ExecutionIdentity{}, agentconfig.ErrUnavailable
	}
	if _, err := h.Authorize(ctx, authz.PermissionWorkbenchAgentUse); err != nil {
		return agent.ExecutionIdentity{}, err
	}
	return agentModeIdentity(ctx, h, false)
}

func (p privateQualityMode) ReadQualityMode(ctx context.Context, scope agentcustomization.Scope, run agentcustomization.QualityRun) (agent.ExecutionMode, string, error) {
	identity, subject, err := p.resolve(ctx, agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: run.DeliveryID}, false)
	if err != nil {
		return "", "", privateModeFailure(err)
	}
	if identity.OrganizationID != scope.OrganizationID || identity.ActorID != scope.ActorID || run.OrganizationID != scope.OrganizationID || run.ActorID != scope.ActorID || run.Draft == nil {
		return "", "", agentcustomization.ErrForbidden
	}
	snapshot, err := p.repository.LoadModeSnapshot(ctx, identity, run.ModeSnapshotRef)
	if err != nil {
		return "", "", privateModeFailure(err)
	}
	if snapshot.Subject != subject || snapshot.Binding.ContextKind != "private-draft-quality" || snapshot.Binding.ContextID != run.Draft.RecordID || snapshot.Binding.RequestKey != run.Key || snapshot.Subject.Definition.ID != run.Definition || snapshot.Subject.Definition.Version != run.Version {
		return "", "", agentcustomization.ErrForbidden
	}
	return snapshot.Mode, snapshot.Revision, nil
}

func privateModeFailure(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, agentconfig.ErrChanged), errors.Is(err, agentconfig.ErrRevision):
		return agentcustomization.ErrRevision
	case errors.Is(err, agentconfig.ErrInvalid):
		return agentcustomization.ErrInvalid
	case errors.Is(err, agentconfig.ErrForbidden):
		return agentcustomization.ErrForbidden
	case errors.Is(err, agentconfig.ErrNotFound):
		return agentcustomization.ErrNotFound
	default:
		return agentcustomization.ErrUnavailable
	}
}
func (p privateQualityMode) PrepareQualityMode(ctx context.Context, scope agentcustomization.Scope, key string, delivery agentcustomization.Delivery, input agentcustomization.DraftSelection, expected string) (agent.ModeSnapshotRef, error) {
	identity, subject, err := p.resolve(ctx, agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: delivery.ID}, false)
	if err != nil {
		return agent.ModeSnapshotRef{}, privateModeFailure(err)
	}
	if identity.OrganizationID != scope.OrganizationID || identity.ActorID != scope.ActorID || subject.Definition.ID != delivery.Definition || subject.Definition.Version != delivery.Version {
		return agent.ModeSnapshotRef{}, agentcustomization.ErrForbidden
	}
	revision, err := agentconfig.ParseModeRevision(expected)
	if err != nil {
		return agent.ModeSnapshotRef{}, agentcustomization.ErrInvalid
	}
	raw, _ := json.Marshal(struct {
		Scope    agentcustomization.Scope
		Delivery agentcustomization.Delivery
		Input    agentcustomization.DraftSelection
	}{scope, delivery, input})
	hash := sha256.Sum256(raw)
	snapshot, err := p.repository.PrepareMode(ctx, agentconfig.ModeStartCommand{Identity: identity, Subject: subject, Binding: agentconfig.ModeBinding{ContextKind: "private-draft-quality", ContextID: input.RecordID, RequestKey: key, IntentDigest: hex.EncodeToString(hash[:])}, ExpectedRevision: revision})
	return snapshot.Ref(), privateModeFailure(err)
}
func (p privateQualityMode) CheckQualityMode(ctx context.Context, scope agentcustomization.Scope, delivery agentcustomization.Delivery, ref agent.ModeSnapshotRef) error {
	// Delivery is the immutable fact already resolved by this Service before
	// its report transaction. Re-reading that owner here can exhaust its pool.
	if p.identity == nil || p.repository == nil {
		return agentcustomization.ErrUnavailable
	}
	identity, err := p.identity(ctx)
	if err != nil {
		return privateModeFailure(err)
	}
	if identity.OrganizationID != scope.OrganizationID || identity.ActorID != scope.ActorID || delivery.OrganizationID != scope.OrganizationID || delivery.Definition != agentcustomization.QualityDefinition || delivery.Version != agentcustomization.QualityVersion || !agentconfig.UUID(delivery.ID) {
		return agentcustomization.ErrForbidden
	}
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: delivery.ID}, Definition: commercetool.AgentRef{ID: delivery.Definition, Version: delivery.Version}}
	snapshot, err := p.repository.CheckModeSnapshot(ctx, identity, ref)
	if err != nil {
		return privateModeFailure(err)
	}
	if snapshot.Subject != subject || snapshot.Binding.ContextKind != "private-draft-quality" {
		return agentcustomization.ErrForbidden
	}
	action := agent.ExecutionAction{Ref: commercetool.ToolRef{ID: "product.quality.check", Version: "v2.0.0"}, Class: agent.ActionObservation, Risk: commercetool.RiskCompute, SideEffect: commercetool.SideEffectNone}
	if agent.EvaluateExecutionPolicy(snapshot.Mode, action, agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: true}).Outcome != agent.PolicyAllow {
		return agentcustomization.ErrForbidden
	}
	return nil
}
