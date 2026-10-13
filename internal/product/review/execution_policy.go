package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
	"task-processor/internal/agent"
	"task-processor/internal/commercetool"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
	"time"
)

const TitlePolicyAdoptionVersion = "product.title.policy-adopt-v1"
const ContentPolicyAdoptionVersion = "product.content.policy-adopt-v1"

// Metadata explains the exact original Agent. Only the injected live resolver
// can turn it into authority; no HTTP route accepts a policy approval field.
type ExecutionProvenance struct {
	ContentSelection enrichment.ContentSelection    `json:",omitzero"`
	Scope            agent.Scope                    `json:"scope"`
	RunID            string                         `json:"runId"`
	RequestKey       string                         `json:"requestKey"`
	Binding          agent.Binding                  `json:"binding"`
	Configuration    agent.ConfigurationSnapshotRef `json:"configuration"`
	Mode             agent.ModeSnapshotRef          `json:"mode"`
}

func (p *ExecutionProvenance) Valid() bool {
	return p == nil || (p.ContentSelection.Absent() || p.ContentSelection.Valid()) && agent.ValidID(p.Scope.OrganizationID) && agent.ValidID(p.Scope.ActorID) && validID(p.RunID) && validID(p.RequestKey) && p.Binding.Valid() && !p.Configuration.Absent() && p.Configuration.ValidOrAbsent() && p.Mode.Valid()
}

type PolicyProvenance struct {
	Execution      ExecutionProvenance     `json:"execution"`
	Identity       agent.ExecutionIdentity `json:"identity"`
	Mode           agent.ExecutionMode     `json:"mode"`
	ModeRevision   string                  `json:"modeRevision"`
	PolicyVersion  string                  `json:"policyVersion"`
	ActionVersion  string                  `json:"actionVersion"`
	ProposalDigest string                  `json:"proposalDigest"`
	RequiresHuman  bool                    `json:"requiresHuman"`
}

func (p *PolicyProvenance) Valid() bool {
	if p == nil {
		return true
	}
	rev, err := strconv.ParseUint(p.ModeRevision, 10, 63)
	hash, hashErr := hex.DecodeString(p.ProposalDigest)
	actionValid := p.ActionVersion == TitlePolicyAdoptionVersion && p.Execution.ContentSelection.Absent() || p.ActionVersion == ContentPolicyAdoptionVersion && p.Execution.ContentSelection.Valid()
	automaticValid := p.Mode != agent.ModeAutoApprove || p.Execution.ContentSelection.Absent() || p.Execution.ContentSelection.TitleOnly()
	return p.Execution.Valid() && actionValid && automaticValid && p.Identity.Valid() && p.Identity.OrganizationID == p.Execution.Scope.OrganizationID && p.Identity.ActorID == p.Execution.Scope.ActorID && (p.Mode == agent.ModeAutoApprove && !p.RequiresHuman || p.Mode == agent.ModeFullAccess) && err == nil && rev > 0 && strconv.FormatUint(rev, 10) == p.ModeRevision && p.PolicyVersion == agent.ExecutionModePolicyVersion && hashErr == nil && len(hash) == 32 && hex.EncodeToString(hash) == p.ProposalDigest
}

type ExecutionPolicyResolver func(context.Context, ExecutionProvenance, enrichment.Candidate, bool) (PolicyProvenance, error)

func (s *Service) WithExecutionPolicy(resolve ExecutionPolicyResolver) *Service {
	s.executionPolicy = resolve
	return s
}

// This is a current execution check, not a native adoption proof: default and
// read_only may persist a pending suggestion without publishing a Product.
func (s *Service) resolveRecordExecutionPolicy(ctx context.Context, r Record) (PolicyProvenance, error) {
	if r.Execution == nil || s.executionPolicy == nil {
		return PolicyProvenance{}, ErrForbidden
	}
	proof, err := s.executionPolicy(ctx, *r.Execution, enrichment.Candidate{Changes: r.Original.Changes}, true)
	if err != nil {
		return PolicyProvenance{}, err
	}
	action := TitlePolicyAdoptionVersion
	if r.Policy == enrichment.ContentPolicyVersion {
		action = ContentPolicyAdoptionVersion
	}
	if proof.Execution != *r.Execution || !proof.Identity.Valid() || proof.Identity.OrganizationID != r.Org || proof.Identity.ActorID != r.Owner || !proof.Mode.Valid() || proof.PolicyVersion != agent.ExecutionModePolicyVersion || proof.ActionVersion != action {
		return PolicyProvenance{}, ErrForbidden
	}
	return proof, nil
}

type PolicyAdoptionCommand struct {
	ProposalID       string
	ExpectedRevision uint64
	Execution        ExecutionProvenance
}
type PolicyAdoptionService struct {
	service *Service
	scope   func(context.Context) (CandidateExecutionScope, error)
	resolve ExecutionPolicyResolver
}

func NewPolicyAdoptionService(reader catalog.VersionedSnapshotReader, source SourcePublicationGateway, store Store, auth Authorizer, scope func(context.Context) (CandidateExecutionScope, error), resolve ExecutionPolicyResolver) (*PolicyAdoptionService, error) {
	if scope == nil || resolve == nil {
		return nil, ErrUnavailable
	}
	service, err := NewCandidateService(reader, source, store, auth)
	if err != nil {
		return nil, err
	}
	return &PolicyAdoptionService{service: service, scope: scope, resolve: resolve}, nil
}
func adoptionAction(r Record) (agent.ExecutionAction, string) {
	id, version := "product.title.policy.adopt", TitlePolicyAdoptionVersion
	if r.Policy == enrichment.ContentPolicyVersion {
		id, version = "product.content.policy.adopt", ContentPolicyAdoptionVersion
	}
	return agent.ExecutionAction{Ref: commercetool.ToolRef{ID: id, Version: "v1.0.0"}, Class: agent.ActionLocalWrite, Risk: commercetool.RiskWrite, SideEffect: commercetool.SideEffectBusinessMutation, AutomaticApproval: true}, version
}

func originalNativeCandidate(r Record) bool {
	if r.State != "pending" || len(r.History) != 0 {
		return false
	}
	if r.Policy != enrichment.ContentPolicyVersion {
		return r.Policy == "title-review-v1" && r.Content == nil && len(r.Original.Changes) == 1 && r.Original.Changes[0].Field == "title" && r.Title == r.Original.Changes[0].Value
	}
	if !r.validContentRecord() || len(r.Content.Decisions) != 0 || len(r.Original.Warnings) != 0 || len(r.Original.Rejections) != 0 {
		return false
	}
	ordered, err := originalContentValues(r)
	return err == nil && reflect.DeepEqual(ordered, r.Content.After)
}

func contentNeedsHuman(r Record) bool {
	return r.Content != nil && !r.Content.Selection.TitleOnly()
}
func proposalDigest(r Record) string {
	raw, _ := json.Marshal(struct {
		Input                       CreateInput
		Base, Before, Title, Policy string
		Original                    enrichment.Proposal
		Revision                    uint64
		Content                     *ContentPayload `json:",omitempty"`
	}{r.Input, r.BasePublicationID, r.Before, r.Title, r.Policy, r.Original, r.Revision, r.Content})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// Adopt shares the existing source validation, Review/Catalog transaction and
// receipt. It never calls Decide or manufactures a human decision/signal.
func (p *PolicyAdoptionService) Adopt(ctx context.Context, key string, in PolicyAdoptionCommand) (View, agent.ExecutionPolicyDecision, error) {
	denied := agent.ExecutionPolicyDecision{Outcome: agent.PolicyDeny, Reason: "owner_validation_failed"}
	if p == nil || !validID(in.ProposalID) || in.ExpectedRevision == 0 || !in.Execution.Valid() {
		return View{}, denied, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	identity, err := p.scope(ctx)
	if err != nil || !ValidKey(identity.OrganizationID) || !ValidKey(identity.ActorID) || !ValidKey(identity.MemberID) {
		return View{}, denied, ErrForbidden
	}
	scope := Scope{Org: identity.OrganizationID, Actor: identity.ActorID}
	op, err := operation(scope, key, "policy-adopt", in.ProposalID, in)
	if err != nil {
		return View{}, denied, err
	}
	ctx, err = p.service.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return View{}, denied, mapSourceReadError(err)
	}
	if v, found, e := p.service.store.Preflight(ctx, op); e != nil || found {
		return v, agent.ExecutionPolicyDecision{Outcome: agent.PolicyAllow, Reason: "committed_receipt"}, e
	}
	record, err := p.service.store.Read(ctx, scope, in.ProposalID)
	if err != nil {
		return View{}, denied, err
	}
	if record.Execution == nil || *record.Execution != in.Execution || record.Revision != in.ExpectedRevision || !originalNativeCandidate(record) {
		return View{}, denied, ErrConflict
	}
	proof, err := p.resolve(ctx, in.Execution, enrichment.Candidate{Changes: record.Original.Changes}, false)
	if err != nil {
		return record.View(), denied, err
	}
	action, actionVersion := adoptionAction(record)
	if proof.Identity != (agent.ExecutionIdentity{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID, MemberID: identity.MemberID}) || proof.Execution != in.Execution || proof.PolicyVersion != agent.ExecutionModePolicyVersion || proof.ActionVersion != actionVersion {
		return View{}, denied, ErrForbidden
	}
	proof.RequiresHuman = proof.RequiresHuman || contentNeedsHuman(record)
	rule := agent.RulePassed
	if proof.RequiresHuman || len(record.Original.Warnings) > 0 || len(record.Original.Rejections) > 0 {
		rule = agent.RuleNeedsHuman
	}
	decision := agent.EvaluateExecutionPolicy(proof.Mode, action, agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: record.Original.Validation.Valid, Automatic: rule})
	if decision.Outcome != agent.PolicyAllow {
		return record.View(), decision, nil
	}
	proof.ProposalDigest = proposalDigest(record)
	if !proof.Valid() {
		return View{}, denied, ErrForbidden
	}
	ctx, err = p.service.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return View{}, denied, mapSourceReadError(err)
	}
	view, err := p.service.store.Run(ctx, op, func(tx Tx) (View, error) {
		if v, found, e := tx.Replay(); e != nil || found {
			return v, e
		}
		r, e := tx.Load(in.ProposalID)
		if e != nil {
			return View{}, e
		}
		if !originalNativeCandidate(r) || r.Revision != in.ExpectedRevision || r.Execution == nil || *r.Execution != in.Execution || proposalDigest(r) != proof.ProposalDigest {
			return View{}, ErrConflict
		}
		fresh, e := p.resolve(ctx, in.Execution, enrichment.Candidate{Changes: r.Original.Changes}, false)
		if e != nil {
			return View{}, e
		}
		fresh.ProposalDigest = proof.ProposalDigest
		fresh.RequiresHuman = fresh.RequiresHuman || contentNeedsHuman(r)
		if !reflect.DeepEqual(fresh, proof) {
			return View{}, ErrConflict
		}
		base, e := p.service.validatePatch(ctx, tx, &r, scope)
		if e != nil {
			return View{}, e
		}
		r.PolicyAdoption = &proof
		if e = publishReview(ctx, tx, &r, scope, base); e != nil {
			return View{}, e
		}
		if e = tx.Save(r); e != nil {
			return View{}, e
		}
		v := r.View()
		return v, tx.Complete(v)
	})
	return view, decision, err
}

func publishReview(ctx context.Context, tx Tx, r *Record, scope Scope, base catalog.PublishedSnapshot) error {
	snapshot, err := catalog.CloneProductSnapshot(base.Snapshot)
	if err != nil {
		return err
	}
	if r.Policy == enrichment.ContentPolicyVersion {
		snapshot, err = applyContentPatch(base.Snapshot, r.Content.Selection, contentChanges(r.Content.After))
		if err != nil {
			return err
		}
	} else {
		snapshot.Title = r.Title
	}
	raw, _ := json.Marshal([]any{r.Org, r.ID, r.Revision})
	sum := sha256.Sum256(raw)
	published, err := tx.Publisher().Publish(ctx, catalog.PublishRequest{Identity: base.Identity, ExpectedBaseVersion: &base.Version, PublicationID: "review:" + hex.EncodeToString(sum[:]), Snapshot: snapshot})
	if err != nil {
		return err
	}
	r.State = "applied"
	r.Receipt = &Receipt{r.ID, r.Revision, published.Version, published.PublicationID, scope.Actor, time.Now().UTC()}
	return nil
}
