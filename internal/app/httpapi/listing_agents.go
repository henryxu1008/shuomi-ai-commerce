package httpapi

import (
	"context"
	"errors"
	"strconv"

	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aiworkbench"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/commercetool"
	"task-processor/internal/integration/commercetoolauth"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	reviewstore "task-processor/internal/integration/persistence/product/review"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
	reviewproposal "task-processor/internal/product/review/proposal"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/product/sourcing/tools/sourceevidenceinspect"
)

type listingAgentExecutionKey struct{}
type listingAgentCommand struct {
	owner  collection.Scope
	record record.TargetRecord
	input  preparation.OperationInput
}

func listingAgentContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	_, ok := ctx.Value(listingAgentExecutionKey{}).(listingAgentCommand)
	return ok
}

type listingProductAgent struct {
	agent         *productAgentApplication
	app           *listingapp.Application
	authorization listingapp.OrganizationExecutionAuthorizer
}
type listingAgentPrincipal struct {
	identity agent.ExecutionIdentity
	roles    []string
	binding  agent.Binding
	original sourcing.PublicationExecutionScope
}

func (s *listingProductAgent) current(ctx context.Context) (listingAgentPrincipal, error) {
	var result listingAgentPrincipal
	command, ok := ctx.Value(listingAgentExecutionKey{}).(listingAgentCommand)
	if !ok || s == nil || s.agent == nil || s.app == nil || command.owner.Validate() != nil {
		return result, review.ErrForbidden
	}
	if _, authenticated := authidentity.AuthenticatedIdentityFromContext(ctx); authenticated {
		return result, review.ErrForbidden
	}
	roles, err := s.authorization.ResolveAgentExecution(ctx, command.owner)
	if err != nil {
		if errors.Is(err, collection.ErrUnavailable) {
			return result, review.ErrUnavailable
		}
		return result, review.ErrForbidden
	}
	allowed := false
	for _, org := range s.agent.config.AllowedOrganizationIDs {
		allowed = allowed || org == command.owner.OrganizationID
	}
	if !allowed || !s.agent.config.Enabled {
		return result, agent.ErrUnavailable
	}
	saved, err := s.app.Records.ReadTargetRecord(ctx, command.owner, command.record.ID)
	if err != nil || collection.Digest(saved) != collection.Digest(command.record) {
		return result, record.ErrConflict
	}
	head, err := s.app.Records.ReadTargetHead(ctx, command.owner, saved.TargetID)
	if err != nil || head.ID != saved.ID || head.Revision != saved.Revision {
		return result, record.ErrConflict
	}
	merchant, rules, err := s.app.Rules.ReadTargetRules(ctx, command.owner, saved.Input.StoreID, saved.Input.Draft)
	if err != nil {
		return result, err
	}
	if merchant != saved.Merchant || collection.Digest(rules) != saved.RulesHash {
		return result, record.ErrConflict
	}
	selected, err := s.app.Sources.SelectForExecution(ctx, command.owner, saved.Source.ID)
	if err != nil {
		return result, err
	}
	scope, source, original, err := selected.Read(ctx)
	if err != nil || scope != command.owner || source != saved.Source {
		return result, record.ErrConflict
	}
	effective, err := s.app.Products.ReadEffectiveTargetProduct(ctx, selected, saved.EffectiveVersion, saved.ApplyReceiptID)
	if err != nil || collection.Digest(effective.Snapshot) != saved.ProductHash {
		return result, record.ErrConflict
	}
	result.identity = agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}
	result.roles = roles
	result.binding = agent.Binding{ContextKind: "collection", ContextID: source.ID, ProductKey: source.Source.ProductKey, CatalogVersion: strconv.FormatUint(effective.Version, 10), PublicationID: effective.PublicationID, TargetPlatform: "shein"}
	result.original = sourcing.PublicationExecutionScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, PublicationID: original.PublicationID, ProductKey: original.Identity.ProductKey, CatalogVersion: original.Version}
	return result, nil
}
func (a *productAgentApplication) resolveExecutionIdentity(ctx context.Context) (agent.ExecutionIdentity, error) {
	if scheduledExecutionContext(ctx) {
		i, _, err := a.scheduledPrincipal(ctx)
		return i, err
	}
	if a == nil || a.listingExecution == nil {
		return agent.ExecutionIdentity{}, review.ErrForbidden
	}
	current, err := a.listingExecution.current(ctx)
	return current.identity, err
}
func (a *productAgentApplication) authorizeListingModel(ctx context.Context, org, actor, member string) error {
	identity, err := a.resolveExecutionIdentity(ctx)
	if err != nil {
		return err
	}
	if identity.OrganizationID != org || identity.ActorID != actor || identity.MemberID != member {
		return review.ErrForbidden
	}
	return nil
}
func (a *productAgentApplication) authorizeListingBinding(ctx context.Context, b agent.Binding) (agent.Scope, error) {
	if a == nil || a.listingExecution == nil || !listingAgentContext(ctx) {
		return agent.Scope{}, review.ErrForbidden
	}
	current, err := a.listingExecution.current(ctx)
	if err != nil {
		return agent.Scope{}, err
	}
	if current.binding != b {
		return agent.Scope{}, agent.ErrConflict
	}
	return agent.Scope{OrganizationID: current.identity.OrganizationID, ActorID: current.identity.ActorID}, nil
}
func (a *productAgentApplication) requestOrExecutionScope(ctx context.Context) (agent.Scope, error) {
	if scheduledExecutionContext(ctx) {
		i, _, err := a.scheduledPrincipal(ctx)
		return agent.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}, err
	}
	if listingAgentContext(ctx) {
		i, err := a.resolveExecutionIdentity(ctx)
		return agent.Scope{OrganizationID: i.OrganizationID, ActorID: i.ActorID}, err
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		return agent.Scope{}, review.ErrForbidden
	}
	return agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, nil
}

type productAgentPrincipal struct {
	application *productAgentApplication
	request     *commercetoolauth.FreshWorkbenchPrincipalResolver
}

func (r productAgentPrincipal) ResolvePrincipal(ctx context.Context) (commercetool.Principal, error) {
	return r.ResolveFreshPrincipal(ctx)
}
func (r productAgentPrincipal) ResolveFreshPrincipal(ctx context.Context) (commercetool.Principal, error) {
	if scheduledExecutionContext(ctx) {
		i, roles, err := r.application.scheduledPrincipal(ctx)
		return commercetool.Principal{TenantID: i.OrganizationID, UserID: i.ActorID, Roles: roles}, err
	}
	if listingAgentContext(ctx) {
		if r.application == nil || r.application.listingExecution == nil {
			return commercetool.Principal{}, review.ErrForbidden
		}
		current, err := r.application.listingExecution.current(ctx)
		if err != nil {
			return commercetool.Principal{}, err
		}
		return commercetool.Principal{TenantID: current.identity.OrganizationID, UserID: current.identity.ActorID, Roles: current.roles}, nil
	}
	return r.request.ResolveFreshPrincipal(ctx)
}

type productAgentSourceGateway struct {
	application *productAgentApplication
	request     review.SourcePublicationGateway
}

func (r productAgentSourceGateway) Read(ctx context.Context, id string) (sourcing.PersistedPublication, error) {
	if scheduledExecutionContext(ctx) {
		if r.application.scheduledSource == nil {
			return sourcing.PersistedPublication{}, review.ErrUnavailable
		}
		return r.application.scheduledSource.Read(ctx, id)
	}
	if listingAgentContext(ctx) {
		if r.application.executionSource == nil {
			return sourcing.PersistedPublication{}, review.ErrForbidden
		}
		return r.application.executionSource.Read(ctx, id)
	}
	return r.request.Read(ctx, id)
}
func (r productAgentSourceGateway) AuthorizeRead(ctx context.Context) (context.Context, error) {
	if scheduledExecutionContext(ctx) {
		if r.application.scheduledSource == nil {
			return ctx, review.ErrUnavailable
		}
		return r.application.scheduledSource.AuthorizeRead(ctx)
	}
	if listingAgentContext(ctx) {
		if r.application.executionSource == nil {
			return ctx, review.ErrForbidden
		}
		return r.application.executionSource.AuthorizeRead(ctx)
	}
	return r.request.AuthorizeRead(ctx)
}
func (r productAgentSourceGateway) AuthorizePublicationExecution(ctx context.Context, org, actor string) error {
	if scheduledExecutionContext(ctx) {
		if r.application == nil || r.application.scheduledSource == nil {
			return review.ErrUnavailable
		}
		return r.application.scheduledSource.AuthorizePublicationExecution(ctx, org, actor)
	}
	if !listingAgentContext(ctx) || r.application == nil || r.application.executionSource == nil {
		return review.ErrForbidden
	}
	return r.application.executionSource.AuthorizePublicationExecution(ctx, org, actor)
}
func (r productAgentSourceGateway) ReadEffectiveSource(ctx context.Context, principal commercetool.Principal, published catalog.PublishedSnapshot) (sourceevidenceinspect.EffectiveSourceObservation, error) {
	var out sourceevidenceinspect.EffectiveSourceObservation
	if r.application == nil || r.application.snapshots == nil || r.application.reviewLookup == nil {
		return out, sourcing.ErrPublicationForbidden
	}
	var binding agent.Binding
	var original sourcing.PublicationExecutionScope
	var source review.SourcePublicationGateway = r.request
	if scheduledExecutionContext(ctx) {
		source = r.application.scheduledSource
	}
	if listingAgentContext(ctx) {
		if r.application.listingExecution == nil || r.application.executionSource == nil {
			return out, sourcing.ErrPublicationForbidden
		}
		current, err := r.application.listingExecution.current(ctx)
		if err != nil {
			return out, err
		}
		if current.identity.OrganizationID != principal.TenantID || current.identity.ActorID != principal.UserID {
			return out, sourcing.ErrPublicationForbidden
		}
		binding, original, source = current.binding, current.original, r.application.executionSource
	} else {
		var ok bool
		binding, ok = ctx.Value(productAgentSourceBindingKey{}).(agent.Binding)
		if ok && binding.ContextKind == titlePilotContext {
			if published.Identity.TenantID != principal.TenantID || published.Identity.ProductKey != binding.ProductKey ||
				published.PublicationID != binding.PublicationID || strconv.FormatUint(published.Version, 10) != binding.CatalogVersion {
				return out, sourcing.ErrPublicationForbidden
			}
			publication, err := r.application.titlePilotPublicationForBinding(ctx, binding)
			if err != nil || publication.Receipt.ActorID != principal.UserID || publication.Receipt.OrganizationID != principal.TenantID {
				return out, sourcing.ErrPublicationForbidden
			}
			out.Requested = sourceevidenceinspect.CatalogReference{ProductKey: binding.ProductKey, Version: binding.CatalogVersion, PublicationID: binding.PublicationID}
			out.Original = publication
			return out, nil
		}
		if !ok || binding.ContextKind != "acquisition" && binding.ContextKind != "own_product" {
			return out, sourcing.ErrPublicationForbidden
		}
		identity, err := r.application.currentAgentIdentity(ctx)
		if err != nil || identity.TenantID != principal.TenantID || identity.UserID != principal.UserID {
			return out, sourcing.ErrPublicationForbidden
		}
		var base catalog.PublishedSnapshot
		if binding.ContextKind == "own_product" {
			if err = r.application.authorizeOwnBinding(ctx, identity, binding); err != nil {
				return out, err
			}
			own, ownErr := r.application.ownSource(ctx, identity, binding.ContextID, &ownSourceSelection{binding.CatalogVersion, binding.PublicationID})
			base, err = own.Original, ownErr
		} else {
			if err = r.application.authorizeAcquisitionBinding(ctx, identity, binding); err != nil {
				return out, err
			}
			base, err = r.application.acquisitionSnapshot(ctx, identity, binding.ContextID)
		}
		if err != nil {
			return out, err
		}
		original = sourcing.PublicationExecutionScope{OrganizationID: identity.TenantID, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID, PublicationID: base.PublicationID, ProductKey: base.Identity.ProductKey, CatalogVersion: base.Version}
	}
	if binding.ProductKey != published.Identity.ProductKey || binding.PublicationID != published.PublicationID || binding.CatalogVersion != strconv.FormatUint(published.Version, 10) || source == nil {
		return out, sourcing.ErrPublicationForbidden
	}
	ctx, err := source.AuthorizeRead(ctx)
	if err != nil {
		return out, err
	}
	lineage, err := review.ResolveAppliedSource(ctx, review.Scope{Org: principal.TenantID, Actor: principal.UserID}, published, r.application.snapshots, source, r.application.reviewLookup)
	if err != nil {
		return out, err
	}
	if lineage.Original.PublicationID != original.PublicationID || lineage.Original.Version != original.CatalogVersion {
		return out, sourcing.ErrSourcePublicationConflict
	}
	out.Requested = sourceevidenceinspect.CatalogReference{ProductKey: published.Identity.ProductKey, Version: strconv.FormatUint(published.Version, 10), PublicationID: published.PublicationID}
	out.Original = lineage.Source
	for _, applied := range lineage.Applied {
		out.Applied = append(out.Applied, sourceevidenceinspect.AppliedPublicationReference{AppliedTitleReference: sourceevidenceinspect.AppliedTitleReference{ProposalID: applied.ProposalID, BaseVersion: strconv.FormatUint(applied.BaseVersion, 10), BasePublicationID: applied.BasePublicationID, Version: strconv.FormatUint(applied.Version, 10), PublicationID: applied.PublicationID}, Policy: applied.Policy, ContentSelection: sourceevidenceinspect.ContentSelection(applied.ContentSelection)})
	}
	return out, nil
}
func (a *productAgentApplication) validateListingCandidate(ctx context.Context, b agent.Binding, policy string, candidate reviewproposal.Candidate, history []agent.Observation) (agent.Validation, error) {
	if a.executionReviews == nil {
		return agent.Validation{}, agent.ErrUnavailable
	}
	if !agentCandidateEvidenceObserved(b, candidate, history) {
		return agent.Validation{PolicyVersion: policy, Unresolved: []string{"candidate_evidence_not_observed"}}, nil
	}
	proposal, err := a.executionReviews.ValidateCandidate(ctx, agentReviewInput(b, policy, candidate))
	if err != nil {
		if errors.Is(err, reviewproposal.ErrEvidenceInsufficient) || errors.Is(err, reviewproposal.ErrOutputValidation) || errors.Is(err, reviewproposal.ErrPolicyRejected) || errors.Is(err, review.ErrInvalid) {
			return agent.Validation{PolicyVersion: policy, Unresolved: []string{"title_or_evidence_invalid"}}, nil
		}
		return agent.Validation{}, err
	}
	return agent.Validation{Valid: proposal.Validation.Valid, PolicyVersion: policy}, nil
}
func connectListingProductAgent(ctx context.Context, db *gorm.DB, application *listingapp.Application, a *productAgentApplication, authority listingapp.OrganizationExecutionAuthorizer) (*listingProductAgent, error) {
	if a == nil || a.config.ContentTitleOnly || a.config.ContentOnly {
		return nil, nil
	}
	bridge := &listingProductAgent{agent: a, app: application, authorization: authority}
	gateway, err := productsourcing.NewExecutionPublicationGateway(db, func(ctx context.Context) (sourcing.PublicationExecutionScope, error) {
		current, err := bridge.current(ctx)
		return current.original, err
	})
	if err != nil {
		return nil, err
	}
	reader, err := catalogstore.NewBoundedSnapshotReader(db, 1<<20)
	if err != nil {
		return nil, err
	}
	repo, err := reviewstore.NewRepository(db, func(tx *gorm.DB) (review.SourcePublicationReader, error) {
		return productsourcing.NewTransactionReader(tx)
	})
	if err != nil {
		return nil, err
	}
	candidate, err := review.NewExecutionCandidateService(reader, gateway, repo, a.authorizer, func(ctx context.Context) (review.CandidateExecutionScope, error) {
		current, err := bridge.current(ctx)
		return review.CandidateExecutionScope{OrganizationID: current.identity.OrganizationID, ActorID: current.identity.ActorID, MemberID: current.identity.MemberID}, err
	})
	if err != nil {
		return nil, err
	}
	a.listingExecution, a.executionSource, a.executionReviews = bridge, gateway, candidate
	candidate.WithExecutionPolicy(a.resolveTitlePolicy)
	a.executionReviewLookup = repo
	a.executionPolicyAdoption, err = review.NewPolicyAdoptionService(reader, gateway, repo, a.authorizer, a.titlePolicyScope, a.resolveTitlePolicy)
	if err != nil {
		return nil, err
	}
	if a.content != nil {
		a.content.listingExecution, a.content.executionSource, a.content.executionReviews = bridge, gateway, candidate
		a.content.executionReviewLookup = repo
		a.content.executionPolicyAdoption, err = review.NewPolicyAdoptionService(reader, gateway, repo, a.authorizer, a.content.titlePolicyScope, a.content.resolveTitlePolicy)
		if err != nil {
			return nil, err
		}
	}
	return bridge, nil
}
func (s *listingProductAgent) Optimize(ctx context.Context, op preparation.Operation, item preparation.OperationItem, key string) (string, error) {
	if s == nil || item.RecordID == "" || op.Input.TitleTemplateID == "" || op.Input.ImageTemplateID != "" {
		return "", record.ErrNotReady
	}
	proof, err := s.app.Operations.AuthorizeExecution(ctx, op.Owner.OrganizationID, op.ID)
	if err != nil {
		return "", err
	}
	durable, err := proof.Read(ctx)
	if err != nil {
		return "", err
	}
	if durable.Owner != op.Owner || collection.Digest(durable.Input) != collection.Digest(op.Input) || key != preparation.ItemCommandID(op.ID, item.SourceID, preparation.OperationOptimize) {
		return "", record.ErrConflict
	}
	fixed, err := s.app.Execution.Repository.BeginOperationItem(ctx, proof, item.SourceID)
	if err != nil {
		return "", err
	}
	if preparation.ItemTerminal(fixed.Status) {
		return fixed.ResultReference, nil
	}
	if fixed.RecordID != item.RecordID || fixed.RecordRevision != item.RecordRevision {
		return "", record.ErrConflict
	}
	saved, err := s.app.Records.ReadTargetRecord(ctx, op.Owner, fixed.RecordID)
	if err != nil {
		return "", err
	}
	if saved.Source.ID != item.SourceID || saved.Input.StoreID != op.Input.StoreID || saved.Revision != fixed.RecordRevision {
		return "", record.ErrConflict
	}
	executionCtx := context.WithValue(ctx, listingAgentExecutionKey{}, listingAgentCommand{owner: op.Owner, record: saved, input: op.Input})
	current, err := s.current(executionCtx)
	if err != nil {
		return "", err
	}
	selected := agentconfig.TemplateRef{TemplateID: op.Input.TitleTemplateID, Revision: op.Input.TitleTemplateRevision}
	profile, err := s.validateTitleSelection(executionCtx, agent.Scope{OrganizationID: op.Owner.OrganizationID, ActorID: op.Owner.ActorID}, selected, op.Input.TitleQuoteHash)
	if err != nil {
		return "", err
	}
	request, err := s.agent.startRequestWithProfile(executionCtx, current.binding, key, "", "", &selected, profile, "", op.Input.TitleModeRevision)
	if err != nil {
		return "", err
	}
	run, err := s.agent.runtime.Start(executionCtx, request)
	if err != nil {
		return "", err
	}
	if !agentRunReviewable(run.State) {
		if run.State.Phase == agent.Running || run.State.PendingInvocationID != "" || run.State.StopReason == agent.StopModelUnknown || run.State.StopReason == agent.StopUsageUnknown || run.State.StopReason == agent.StopExecutionOutcomeUnknown {
			return run.State.RunID, preparation.ErrUnknown
		}
		return "", record.ErrNotReady
	}
	input := agentReviewInput(current.binding, run.State.Request.PolicyVersion, run.State.Candidate)
	input.Execution = agentExecutionProvenance(run.State)
	input.ContextProvenance, err = agentContextProvenance(run.State)
	if err != nil {
		return "", err
	}
	view, err := s.agent.executionReviews.CreateFromCandidate(executionCtx, "agent:"+run.State.RunID, input)
	if err == nil {
		if adopted, e := s.agent.finishTitleCandidate(executionCtx, run); e != nil {
			return view.ID, e
		} else if adopted != nil {
			view = *adopted
		}
	}
	return view.ID, err
}

var _ listingapp.OperationOptimizer = (*listingProductAgent)(nil)

// Scheduled consumers reuse the source/review constructors at this admitted
// composition root, with the same original owner and narrow current scope.
func (a *aiWorkbenchApplication) connectScheduledExecution(ctx context.Context, db, sourceDB *gorm.DB) error {
	if a == nil || a.agent == nil || a.agent.originRoles == nil || db == nil || sourceDB == nil {
		return aiworkbench.ErrUnavailable
	}
	consumer := a.agent
	if err := consumer.connectScheduledSourceReceipts(ctx, sourceDB); err != nil {
		return err
	}
	source, err := productsourcing.NewExecutionPublicationGateway(sourceDB, func(ctx context.Context) (sourcing.PublicationExecutionScope, error) {
		i, _, err := consumer.scheduledPrincipal(ctx)
		if err != nil {
			return sourcing.PublicationExecutionScope{}, err
		}
		command := ctx.Value(scheduledExecutionKey{}).(scheduledExecutionCommand)
		plan, err := a.store.ReadScheduledExecutionPlan(ctx, command.planID)
		if err != nil {
			return sourcing.PublicationExecutionScope{}, err
		}
		identity, err := consumer.currentAgentIdentity(ctx)
		if err != nil {
			return sourcing.PublicationExecutionScope{}, err
		}
		if plan.Selection.Subject.Kind == "acquisition" {
			original, err := consumer.acquisitionSnapshot(ctx, identity, plan.Selection.Subject.ID)
			if err != nil {
				return sourcing.PublicationExecutionScope{}, err
			}
			return sourcing.PublicationExecutionScope{OrganizationID: i.OrganizationID, ActorID: i.ActorID, MemberID: i.MemberID, PublicationID: original.PublicationID, ProductKey: original.Identity.ProductKey, CatalogVersion: original.Version}, nil
		}
		if plan.Selection.Subject.Kind != "own_product" {
			return sourcing.PublicationExecutionScope{}, aiworkbench.ErrInvalid
		}
		own, err := consumer.ownSource(ctx, identity, plan.Selection.Subject.ID, &ownSourceSelection{plan.Selection.Subject.CatalogVersion, plan.Selection.Subject.PublicationID})
		if err != nil {
			return sourcing.PublicationExecutionScope{}, err
		}
		return sourcing.PublicationExecutionScope{OrganizationID: i.OrganizationID, ActorID: i.ActorID, MemberID: i.MemberID, PublicationID: own.Original.PublicationID, ProductKey: own.Original.Identity.ProductKey, CatalogVersion: own.Original.Version}, nil
	})
	if err != nil {
		return err
	}
	repository, err := reviewstore.NewRepository(db, func(tx *gorm.DB) (review.SourcePublicationReader, error) {
		return productsourcing.NewTransactionReader(tx)
	})
	if err != nil {
		return err
	}
	candidate, err := review.NewExecutionCandidateService(consumer.snapshots, source, repository, consumer.authorizer, func(ctx context.Context) (review.CandidateExecutionScope, error) {
		i, _, err := consumer.scheduledPrincipal(ctx)
		return review.CandidateExecutionScope{OrganizationID: i.OrganizationID, ActorID: i.ActorID, MemberID: i.MemberID}, err
	})
	if err != nil {
		return err
	}
	consumer.scheduledSource, consumer.scheduledReviews = source, candidate
	return nil
}
