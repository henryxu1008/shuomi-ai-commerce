package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"task-processor/internal/agent"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/enrichment"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	reader            catalog.VersionedSnapshotReader
	sourceReader      SourcePublicationGateway
	store             Store
	proposer          enrichment.Proposer
	auth              Authorizer
	executionResolver func(context.Context) (CandidateExecutionScope, error)
	executionPolicy   ExecutionPolicyResolver
}

func NewService(reader catalog.VersionedSnapshotReader, sourceReader SourcePublicationGateway, store Store, proposer enrichment.Proposer, auth Authorizer) (*Service, error) {
	if reader == nil || sourceReader == nil || store == nil || proposer == nil || auth == nil {
		return nil, ErrUnavailable
	}
	return &Service{reader: reader, sourceReader: sourceReader, store: store, proposer: proposer, auth: auth}, nil
}

// NewCandidateService composes existing candidate intake and human review when
// no fixed generator is configured. Create explicitly remains unavailable.
func NewCandidateService(reader catalog.VersionedSnapshotReader, sourceReader SourcePublicationGateway, store Store, auth Authorizer) (*Service, error) {
	if reader == nil || sourceReader == nil || store == nil || auth == nil {
		return nil, ErrUnavailable
	}
	return &Service{reader: reader, sourceReader: sourceReader, store: store, auth: auth}, nil
}
func (s *Service) authorize(ctx context.Context, write, admin bool) (Scope, error) {
	if err := ctx.Err(); err != nil {
		return Scope{}, err
	}
	if s.executionResolver != nil {
		return s.executionScope(ctx, admin)
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !ValidKey(i.UserID) || !ValidKey(i.EffectiveOrganizationID) || i.TenantID != i.EffectiveOrganizationID || !time.Now().Before(i.TokenExpiresAt) {
		return Scope{}, ErrForbidden
	}
	// Scope roles are produced by the existing organization resolver. Configured
	// global user grants cannot substitute for a grant in the effective org.
	a := Scope{i.EffectiveOrganizationID, i.UserID, s.auth.IsTenantAdmin("", i.Roles)}
	allowed, err := authz.AuthorizeOrganization(ctx, s.auth, "", i.EffectiveOrganizationID, i.Roles, authz.PermissionLocalAgentWrite)
	if err != nil {
		return Scope{}, ErrUnavailable
	}
	if !allowed || admin && !a.Admin {
		return Scope{}, ErrForbidden
	}
	return a, nil
}
func operation(a Scope, key, kind, id string, input any) (Operation, error) {
	if !ValidKey(key) {
		return Operation{}, ErrInvalid
	}
	raw, err := json.Marshal([]any{kind, id, input})
	if err != nil {
		return Operation{}, ErrInvalid
	}
	sum := sha256.Sum256(raw)
	return Operation{a, key, hex.EncodeToString(sum[:])}, nil
}
func (s *Service) Create(ctx context.Context, key string, in CreateInput) (View, error) {
	if s == nil || s.proposer == nil {
		return View{}, ErrUnavailable
	}
	view, _, err := s.create(ctx, key, in, nil)
	return view, err
}

// CreateFromCandidate admits an already-generated candidate into the existing
// pending review state. It cannot regenerate, approve, Apply or publish it.
func (s *Service) CreateFromCandidate(ctx context.Context, key string, in CandidateInput) (View, error) {
	view, _, err := s.CreateFromCandidateWithTransactionAttempt(ctx, key, in)
	return view, err
}

// TransactionAttempt is true once Store.Run is called, including an ambiguous
// transaction result. Authorization, validation and read-only preflight never
// grant this signal. Callers must not infer a safe retry from a transaction error.
func (s *Service) CreateFromCandidateWithTransactionAttempt(ctx context.Context, key string, in CandidateInput) (View, bool, error) {
	if !in.ContextProvenance.Valid() {
		return View{}, false, ErrInvalid
	}
	if ctx == nil || !ValidKey(in.PublicationID) {
		return View{}, false, ErrInvalid
	}
	if _, err := candidatePolicy(in); err != nil {
		return View{}, false, err
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return View{}, false, ErrInvalid
	}
	if len(raw) > MaxRecordBytes {
		return View{}, false, ErrTooLarge
	}
	var isolated CandidateInput
	if err := json.Unmarshal(raw, &isolated); err != nil {
		return View{}, false, ErrInvalid
	}
	return s.create(ctx, key, isolated.Base, &isolated)
}

// ValidateCandidate uses the same exact source and fixed field policy as intake,
// without generating or saving a proposal. Intake always repeats this check.
func (s *Service) ValidateCandidate(ctx context.Context, in CandidateInput) (enrichment.Proposal, error) {
	if !in.ContextProvenance.Valid() {
		return enrichment.Proposal{}, ErrInvalid
	}
	if ctx == nil || !ValidKey(in.PublicationID) {
		return enrichment.Proposal{}, ErrInvalid
	}
	if _, err := candidatePolicy(in); err != nil {
		return enrichment.Proposal{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, true, false)
	if err != nil {
		return enrichment.Proposal{}, err
	}
	if !contentScopeMatches(in, scope) {
		return enrichment.Proposal{}, ErrForbidden
	}
	base, source, err := s.source(ctx, s.reader, s.sourceReader, scope, in.Base)
	if err != nil {
		return enrichment.Proposal{}, err
	}
	if base.PublicationID != in.PublicationID {
		return enrichment.Proposal{}, ErrConflict
	}
	return validateIntakeProposal(ctx, base.Snapshot, source, in)
}

func (s *Service) create(ctx context.Context, key string, in CreateInput, supplied *CandidateInput) (View, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	a, err := s.authorize(ctx, true, false)
	if err != nil {
		return View{}, false, err
	}
	if !ValidKey(in.ProductKey) || in.BaseVersion == 0 || in.BaseVersion > 1<<63-1 {
		return View{}, false, ErrInvalid
	}
	kind, payload := "create", any(in)
	if supplied != nil {
		kind, payload = "create_candidate", supplied
		if !contentScopeMatches(*supplied, a) {
			return View{}, false, ErrForbidden
		}
	}
	op, err := operation(a, key, kind, "", payload)
	if err != nil {
		return View{}, false, err
	}
	ctx, err = s.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return View{}, false, mapSourceReadError(err)
	}
	if v, found, e := s.store.Preflight(ctx, op); e != nil || found {
		return v, false, e
	}
	base, source, err := s.source(ctx, s.reader, s.sourceReader, a, in)
	if err != nil {
		return View{}, false, err
	}
	if supplied != nil && supplied.PublicationID != base.PublicationID {
		return View{}, false, ErrConflict
	}
	policy := enrichment.PolicySnapshot{Version: "title-review-v1", AllowedFields: []string{"title"}, RequiredFields: []string{"title"}}
	request := enrichment.Request{Snapshot: base.Snapshot, Source: source, Policy: policy}
	var proposal enrichment.Proposal
	if supplied != nil {
		policy, err = candidatePolicy(*supplied)
		if err != nil {
			return View{}, false, err
		}
		proposal, err = validateIntakeProposal(ctx, base.Snapshot, source, *supplied)
	} else {
		proposal, err = s.proposer.Propose(ctx, request)
	}
	if err != nil {
		return View{}, false, err
	}
	if !proposal.Validation.Valid || policy.Version == "title-review-v1" && (len(proposal.Changes) != 1 || proposal.Changes[0].Field != "title") {
		return View{}, false, ErrInvalid
	}
	if policy.Version == "title-review-v1" {
		if err = ValidateTitle(proposal.Changes[0].Value); err != nil {
			return View{}, false, err
		}
	}
	ctx, err = s.sourceReader.AuthorizeRead(ctx)
	if err != nil {
		return View{}, false, mapSourceReadError(err)
	}
	r := Record{ID: uuid.NewString(), Org: a.Org, Owner: a.Actor, Input: in, BasePublicationID: base.PublicationID, Policy: policy.Version, Before: base.Snapshot.Title, Title: proposal.Changes[0].Value, State: "pending", Revision: 1, Original: proposal}
	if policy.Version == enrichment.ContentPolicyVersion {
		r.Before, r.Title = "", ""
		r.Content, err = contentPayload(base.Snapshot, supplied.ContentSelection, proposal)
		if err != nil {
			return View{}, false, err
		}
	}
	if supplied != nil {
		r.ContextProvenance = supplied.ContextProvenance.clone()
		if !supplied.Execution.Valid() {
			return View{}, false, ErrInvalid
		}
		if supplied.Execution != nil {
			value := *supplied.Execution
			r.Execution = &value
		}
	}
	if err = r.ValidateStorage(); err != nil {
		return View{}, false, err
	}
	view, err := s.store.Run(ctx, op, func(tx Tx) (View, error) {
		if v, found, e := tx.Replay(); e != nil || found {
			return v, e
		}
		rechecked, freshSource, e := s.source(ctx, tx.Reader(), tx.SourceReader(), a, in, appliedLookup(tx))
		if e != nil {
			return View{}, e
		}
		if rechecked.Identity != base.Identity || rechecked.Version != base.Version ||
			rechecked.PublicationID != base.PublicationID || !reflect.DeepEqual(rechecked.Snapshot, base.Snapshot) {
			return View{}, ErrConflict
		}
		if supplied != nil {
			if _, e = validateIntakeProposal(ctx, rechecked.Snapshot, freshSource, *supplied); e != nil {
				return View{}, e
			}
		}
		if e := tx.Save(r); e != nil {
			return View{}, e
		}
		v := r.View()
		return v, tx.Complete(v)
	})
	return view, true, err
}
func validID(id string) bool {
	parsed, err := uuid.Parse(id)
	return err == nil && parsed.String() == id && parsed != uuid.Nil
}
func (s *Service) Get(ctx context.Context, id string) (View, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	a, err := s.authorize(ctx, false, false)
	if err != nil {
		return View{}, err
	}
	if !validID(id) {
		return View{}, ErrInvalid
	}
	r, err := s.store.Read(ctx, a, id)
	if err != nil {
		return View{}, err
	}
	return r.View(), nil
}

// FindAgentReview returns the current Review View for the exact Agent run when
// its review operation exists. It never treats a create-response snapshot as
// the current approval or Apply state.
func (s *Service) FindAgentReview(ctx context.Context, runID string) (View, bool, error) {
	if s == nil || ctx == nil || !ValidKey(runID) {
		return View{}, false, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, false, false)
	if err != nil {
		return View{}, false, err
	}
	record, found, err := s.findAgentReview(ctx, scope, runID)
	if err != nil || !found {
		return View{}, found, err
	}
	return record.View(), true, nil
}

// FindAgentTaskReviewState exposes only the current state of this actor's exact
// Agent review to the BusinessTask projection. It grants no Review View,
// decision, or Apply authority to a task reader.
func (s *Service) FindAgentTaskReviewState(ctx context.Context, runID string) (string, bool, error) {
	if s == nil || ctx == nil || !ValidKey(runID) {
		return "", false, ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !ValidKey(identity.UserID) || !ValidKey(identity.EffectiveOrganizationID) ||
		identity.TenantID != identity.EffectiveOrganizationID || !time.Now().Before(identity.TokenExpiresAt) ||
		!authz.AllowedOrganization(ctx, s.auth, identity.UserID, identity.EffectiveOrganizationID, identity.Roles, authz.PermissionWorkbenchTaskRead) {
		return "", false, ErrForbidden
	}
	scope := Scope{Org: identity.EffectiveOrganizationID, Actor: identity.UserID}
	record, found, err := s.findAgentReview(ctx, scope, runID)
	if err != nil || !found {
		return "", found, err
	}
	return record.State, true, nil
}

func (s *Service) findAgentReview(ctx context.Context, scope Scope, runID string) (Record, bool, error) {
	lookup, ok := s.store.(AgentReviewLookup)
	if !ok {
		return Record{}, false, ErrUnavailable
	}
	id, found, err := lookup.FindAgentReviewID(ctx, scope, runID)
	if err != nil || !found {
		return Record{}, found, err
	}
	record, err := s.store.Read(ctx, scope, id)
	if err != nil || record.Owner != scope.Actor {
		return Record{}, false, ErrUnavailable
	}
	return record, true, nil
}
func (s *Service) Decide(ctx context.Context, key, id string, in DecisionInput) (View, error) {
	return s.change(ctx, key, id, "decision", in, in.Action != "edit", in.Action != "reject", func(ctx context.Context, tx Tx, r *Record, a Scope) error {
		if err := r.Decide(a.Actor, in); err != nil {
			return err
		}
		if in.Action != "reject" {
			_, err := s.validatePatch(ctx, tx, r, a)
			return err
		}
		return nil
	})
}
func (s *Service) validatePatch(ctx context.Context, tx Tx, r *Record, a Scope) (catalog.PublishedSnapshot, error) {
	if r.Policy == enrichment.ContentPolicyVersion {
		base, source, err := s.source(ctx, tx.Reader(), tx.SourceReader(), Scope{Org: a.Org, Actor: r.Owner}, r.Input, appliedLookup(tx))
		if err != nil {
			return base, err
		}
		return base, validateContentPatch(ctx, base, source, *r)
	}
	if err := ValidateTitle(r.Title); err != nil {
		return catalog.PublishedSnapshot{}, err
	}
	base, source, err := s.source(ctx, tx.Reader(), tx.SourceReader(), Scope{Org: a.Org, Actor: r.Owner}, r.Input, appliedLookup(tx))
	if err != nil {
		return base, err
	}
	evidence, err := enrichment.CanonicalEvidenceID(source)
	if err != nil {
		return base, err
	}
	if base.PublicationID != r.BasePublicationID || base.Snapshot.Title != r.Before || r.Policy != "title-review-v1" || len(r.Original.Changes) != 1 || r.Original.Changes[0].Field != "title" || len(r.Original.Changes[0].EvidenceIDs) != 1 || r.Original.Changes[0].EvidenceIDs[0] != evidence {
		return base, ErrConflict
	}
	return base, nil
}
func (s *Service) Apply(ctx context.Context, key, id string, in ApplyInput) (View, error) {
	return s.change(ctx, key, id, "apply", in, true, true, func(ctx context.Context, tx Tx, r *Record, a Scope) error {
		if r.State != "accepted" || in.ExpectedRevision != r.Revision || in.ExpectedRevision == 0 {
			return ErrConflict
		}
		base, err := s.validatePatch(ctx, tx, r, a)
		if err != nil {
			return err
		}
		return publishReview(ctx, tx, r, a, base)
	})
}
func (s *Service) change(ctx context.Context, key, id, kind string, input any, admin, sourceRead bool, mutate func(context.Context, Tx, *Record, Scope) error) (View, error) {
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	a, err := s.authorize(ctx, true, admin)
	if err != nil {
		return View{}, err
	}
	if !validID(id) {
		return View{}, ErrInvalid
	}
	op, err := operation(a, key, kind, id, input)
	if err != nil {
		return View{}, err
	}
	if sourceRead {
		if s.sourceReader == nil {
			return View{}, ErrUnavailable
		}
		ctx, err = s.sourceReader.AuthorizeRead(ctx)
		if err != nil {
			return View{}, mapSourceReadError(err)
		}
	}
	if v, found, e := s.store.Preflight(ctx, op); e != nil || found {
		return v, e
	}
	return s.store.Run(ctx, op, func(tx Tx) (View, error) {
		if v, found, e := tx.Replay(); e != nil || found {
			return v, e
		}
		r, e := tx.Load(id)
		if e != nil {
			return View{}, e
		}
		if s.executionPolicy != nil {
			if r.Execution == nil {
				return View{}, ErrForbidden
			}
			proof, e := s.executionPolicy(ctx, *r.Execution, enrichment.Candidate{Changes: r.Original.Changes}, true)
			if e != nil {
				return View{}, e
			}
			if proof.Mode == agent.ModeReadOnly {
				return View{}, ErrForbidden
			}
		}
		if e = mutate(ctx, tx, &r, a); e != nil {
			return View{}, e
		}
		if e = tx.Save(r); e != nil {
			return View{}, e
		}
		v := r.View()
		return v, tx.Complete(v)
	})
}
