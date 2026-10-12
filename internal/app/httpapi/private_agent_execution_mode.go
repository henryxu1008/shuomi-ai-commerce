package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/agentcustomization"
	"task-processor/internal/commercetool"
)

type privateQualityMode struct {
	repository agentconfig.ModeRepository
	resolve    func(context.Context, agentconfig.ModeSubject, bool) (agent.ExecutionIdentity, agentconfig.ResolvedModeSubject, error)
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
	identity, subject, err := p.resolve(ctx, agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: delivery.ID}, false)
	if err != nil {
		return privateModeFailure(err)
	}
	if identity.OrganizationID != scope.OrganizationID || identity.ActorID != scope.ActorID {
		return agentcustomization.ErrForbidden
	}
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
