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

// Metadata explains the exact original Agent. Only the injected live resolver
// can turn it into authority; no HTTP route accepts a policy approval field.
type ExecutionProvenance struct {
	Scope         agent.Scope                    `json:"scope"`
	RunID         string                         `json:"runId"`
	RequestKey    string                         `json:"requestKey"`
	Binding       agent.Binding                  `json:"binding"`
	Configuration agent.ConfigurationSnapshotRef `json:"configuration"`
	Mode          agent.ModeSnapshotRef          `json:"mode"`
}

func (p *ExecutionProvenance) Valid() bool {
	return p == nil || agent.ValidID(p.Scope.OrganizationID) && agent.ValidID(p.Scope.ActorID) && validID(p.RunID) && validID(p.RequestKey) && p.Binding.Valid() && !p.Configuration.Absent() && p.Configuration.ValidOrAbsent() && p.Mode.Valid()
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
	return p.Execution.Valid() && p.Identity.Valid() && p.Identity.OrganizationID == p.Execution.Scope.OrganizationID && p.Identity.ActorID == p.Execution.Scope.ActorID && (p.Mode == agent.ModeAutoApprove && !p.RequiresHuman || p.Mode == agent.ModeFullAccess) && err == nil && strconv.FormatUint(rev, 10) == p.ModeRevision && p.PolicyVersion == agent.ExecutionModePolicyVersion && p.ActionVersion == TitlePolicyAdoptionVersion && hashErr == nil && len(hash) == 32 && hex.EncodeToString(hash) == p.ProposalDigest
}

type ExecutionPolicyResolver func(context.Context, ExecutionProvenance, enrichment.Candidate, bool) (PolicyProvenance, error)

func (s *Service) WithExecutionPolicy(resolve ExecutionPolicyResolver) *Service {
	s.executionPolicy = resolve
	return s
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
func titleAdoptionAction() agent.ExecutionAction {
	return agent.ExecutionAction{Ref: commercetool.ToolRef{ID: "product.title.policy.adopt", Version: "v1.0.0"}, Class: agent.ActionLocalWrite, Risk: commercetool.RiskWrite, SideEffect: commercetool.SideEffectBusinessMutation, AutomaticApproval: true}
}
func proposalDigest(r Record) string {
	raw, _ := json.Marshal(struct {
		Input                       CreateInput
		Base, Before, Title, Policy string
		Original                    enrichment.Proposal
		Revision                    uint64
	}{r.Input, r.BasePublicationID, r.Before, r.Title, r.Policy, r.Original, r.Revision})
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
	if v, found, e := p.service.store.Preflight(ctx, op); e != nil || found {
		return v, agent.ExecutionPolicyDecision{Outcome: agent.PolicyAllow, Reason: "committed_receipt"}, e
	}
	record, err := p.service.store.Read(ctx, scope, in.ProposalID)
	if err != nil {
		return View{}, denied, err
	}
	if record.Execution == nil || *record.Execution != in.Execution || record.Revision != in.ExpectedRevision || record.State != "pending" || len(record.History) != 0 || len(record.Original.Changes) != 1 || record.Title != record.Original.Changes[0].Value {
		return View{}, denied, ErrConflict
	}
	proof, err := p.resolve(ctx, in.Execution, enrichment.Candidate{Changes: record.Original.Changes}, false)
	if err != nil {
		return record.View(), denied, err
	}
	if proof.Identity != (agent.ExecutionIdentity{OrganizationID: identity.OrganizationID, ActorID: identity.ActorID, MemberID: identity.MemberID}) || proof.Execution != in.Execution || proof.PolicyVersion != agent.ExecutionModePolicyVersion || proof.ActionVersion != TitlePolicyAdoptionVersion {
		return View{}, denied, ErrForbidden
	}
	rule := agent.RulePassed
	if proof.RequiresHuman || len(record.Original.Warnings) > 0 || len(record.Original.Rejections) > 0 {
		rule = agent.RuleNeedsHuman
	}
	decision := agent.EvaluateExecutionPolicy(proof.Mode, titleAdoptionAction(), agent.ExecutionPolicyChecks{OwnerAuthorized: true, ContentValid: record.Original.Validation.Valid, Automatic: rule})
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
		if r.State != "pending" || r.Revision != in.ExpectedRevision || r.Execution == nil || *r.Execution != in.Execution || proposalDigest(r) != proof.ProposalDigest {
			return View{}, ErrConflict
		}
		fresh, e := p.resolve(ctx, in.Execution, enrichment.Candidate{Changes: r.Original.Changes}, false)
		if e != nil {
			return View{}, e
		}
		fresh.ProposalDigest = proof.ProposalDigest
		if !reflect.DeepEqual(fresh, proof) {
			return View{}, ErrConflict
		}
		base, e := p.service.validatePatch(ctx, tx, &r, scope)
		if e != nil {
			return View{}, e
		}
		r.PolicyAdoption = &proof
		if e = publishTitle(ctx, tx, &r, scope, base); e != nil {
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

func publishTitle(ctx context.Context, tx Tx, r *Record, scope Scope, base catalog.PublishedSnapshot) error {
	snapshot, err := catalog.CloneProductSnapshot(base.Snapshot)
	if err != nil {
		return err
	}
	snapshot.Title = r.Title
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
