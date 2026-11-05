package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"task-processor/internal/agentconfig"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/dataservice"
	"task-processor/internal/httproute"
	"task-processor/internal/integration/acquisition/zenrows"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	rh "task-processor/internal/product/research/httpapi"
)

const (
	sheinTrialModule = "market-research-shein-trial"
	sheinTrialBase   = rh.Base + "/:project_id/shein-us-trials/:approval_id"
)

type sheinTrialDependencies struct {
	authority dataacquisition.SHEINUSTrialAuthority
	send      dataacquisition.SHEINUSTrialSender
	lookup    dataacquisition.SHEINUSTrialLookup
}

type SHEINUSTrialInputs struct {
	Approval   dataacquisition.SHEINUSTrialApproval
	Reconciler collection.Scope
	Access     dataacquisition.SHEINUSTrialAccessCheck
	Budget     dataacquisition.SHEINUSTrialBudgetReader
	ReadKey    zenrows.ResolveKey
	ReadUsage  zenrows.ReadUsageEvidence
}

// WithSHEINUSTrial opts into the existing single-request Product composition.
// Preparing the option does not resolve the provider credential.
func WithSHEINUSTrial(in SHEINUSTrialInputs) (CurrentApplicationOption, error) {
	d, err := assembleSHEINTrialDependencies(in)
	if err != nil {
		return nil, err
	}
	return withSHEINUSTrial(d), nil
}

func assembleSHEINTrialDependencies(in SHEINUSTrialInputs) (sheinTrialDependencies, error) {
	authority, err := dataacquisition.NewFixedSHEINUSTrialAuthority(in.Approval, in.Reconciler, in.Access, in.Budget)
	if err != nil {
		return sheinTrialDependencies{}, err
	}
	key, err := zenrows.NewDedicatedKeyResolver(in.Approval, in.ReadKey)
	if err != nil {
		return sheinTrialDependencies{}, err
	}
	fetch, err := zenrows.NewFetch(key)
	if err != nil {
		return sheinTrialDependencies{}, err
	}
	lookup, err := zenrows.NewLookup(in.ReadUsage)
	if err != nil {
		return sheinTrialDependencies{}, err
	}
	return sheinTrialDependencies{authority: authority, send: fetch.Send, lookup: lookup}, nil
}

func withSHEINUSTrial(d sheinTrialDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.sheinTrial = &d; o.sheinTrialConfigs++ }
}

func (d sheinTrialDependencies) complete() bool {
	return d.authority != nil && d.send != nil && d.lookup != nil
}

type sheinTrialProjects interface {
	Authorize(context.Context, collection.Scope, string, bool) error
	AuthorizeDataMarket(context.Context, collection.Scope) error
}

type sheinTrialProjectKey struct{}

type projectSHEINTrialAuthority struct {
	dataacquisition.SHEINUSTrialAuthority
	projects sheinTrialProjects
}

func (a projectSHEINTrialAuthority) AuthorizeSHEINUSTrial(ctx context.Context, p dataacquisition.Principal, id string, action dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	project, _ := ctx.Value(sheinTrialProjectKey{}).(string)
	denied := dataacquisition.SHEINUSTrialApproval{}
	if !agentconfig.UUID(project) {
		return denied, dataacquisition.ErrForbidden
	}
	switch action {
	case dataacquisition.SHEINUSTrialView, dataacquisition.SHEINUSTrialSend, dataacquisition.SHEINUSTrialPublish:
		if a.projects.Authorize(ctx, p.Scope, project, action != dataacquisition.SHEINUSTrialView) != nil {
			return denied, dataacquisition.ErrForbidden
		}
	case dataacquisition.SHEINUSTrialReconcile:
		// The account authority may grant original-bill reconciliation to
		// a current member other than the now-revoked original submitter.
	default:
		return denied, dataacquisition.ErrForbidden
	}
	approved, err := a.SHEINUSTrialAuthority.AuthorizeSHEINUSTrial(ctx, p, id, action)
	if err != nil || approved.ProjectID != project {
		return denied, dataacquisition.ErrForbidden
	}
	if (action == dataacquisition.SHEINUSTrialView || action == dataacquisition.SHEINUSTrialReconcile) && a.projects.AuthorizeDataMarket(ctx, p.Scope) != nil {
		return denied, dataacquisition.ErrForbidden
	}
	return approved, nil
}

type marketResearchSHEINTrial struct {
	trial *dataservicesapp.SHEINUSPageTrial
	bind  func(context.Context, string) (context.Context, collection.Scope, error)
}

func composeMarketResearchSHEINTrial(ctx context.Context, data *dataservicesapp.Module, d sheinTrialDependencies, projects sheinTrialProjects, bind func(context.Context, string) (context.Context, collection.Scope, error)) (*marketResearchSHEINTrial, error) {
	if !d.complete() || projects == nil || bind == nil {
		return nil, dataacquisition.ErrUnavailable
	}
	trial, err := data.AssembleSHEINUSTrial(ctx, projectSHEINTrialAuthority{d.authority, projects}, d.send, d.lookup)
	if err != nil {
		return nil, err
	}
	return &marketResearchSHEINTrial{trial: trial, bind: bind}, nil
}

func marketResearchSHEINTrialRoutes(app *marketResearchSHEINTrial) []httproute.Descriptor {
	return []httproute.Descriptor{
		sheinTrialRoute(app, http.MethodGet, "", dataservice.PermissionMarket, 10*time.Second, dataacquisition.SHEINUSTrialView),
		sheinTrialRoute(app, http.MethodPost, "/submit", "workbench.project.manage", 40*time.Second, dataacquisition.SHEINUSTrialSend),
		sheinTrialRoute(app, http.MethodPost, "/reconcile", dataservice.PermissionMarket, 35*time.Second, dataacquisition.SHEINUSTrialReconcile),
	}
}

func sheinTrialRoute(app *marketResearchSHEINTrial, method, suffix, permission string, timeout time.Duration, action dataacquisition.SHEINUSTrialAction) httproute.Descriptor {
	return httproute.Descriptor{
		Method: method, Path: sheinTrialBase + suffix, Module: sheinTrialModule, Permission: permission,
		AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
		RequestTimeout: timeout, RejectUnreadRequestBody: true,
		Handler: func(c *gin.Context) { app.serve(c, action) },
	}
}

func validateMarketResearchSHEINTrialDescriptor(d httproute.Descriptor) error {
	for _, expected := range marketResearchSHEINTrialRoutes(nil) {
		if d.Method == expected.Method && d.Path == expected.Path && d.Module == expected.Module &&
			d.Permission == expected.Permission && d.AuthPolicy == expected.AuthPolicy &&
			d.OrganizationAccessPolicy == expected.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil &&
			d.RequestTimeout == expected.RequestTimeout && d.RejectUnreadRequestBody && d.Handler != nil {
			return nil
		}
	}
	return errors.New("SHEIN trial route loses its admitted request or authorization boundary")
}

func (a *marketResearchSHEINTrial) serve(c *gin.Context, action dataacquisition.SHEINUSTrialAction) {
	c.Header("Cache-Control", "private, no-store")
	project, id := c.Param("project_id"), c.Param("approval_id")
	r := c.Request
	if !agentconfig.UUID(project) || !agentconfig.UUID(id) || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.ContentLength != 0 || len(r.TransferEncoding) != 0 || r.Header.Get("Content-Encoding") != "" || len(r.Header.Values("Idempotency-Key")) != 0 {
		httproute.RejectUnreadRequestBody(c)
		writeSHEINTrialError(c, dataacquisition.ErrInvalid)
		return
	}
	if a == nil || a.trial == nil || a.bind == nil {
		writeSHEINTrialError(c, dataacquisition.ErrUnavailable)
		return
	}
	ctx, scope, err := a.bind(r.Context(), c.GetHeader("Authorization"))
	if err != nil || scope.Validate() != nil {
		writeSHEINTrialError(c, dataacquisition.ErrForbidden)
		return
	}
	ctx = context.WithValue(ctx, sheinTrialProjectKey{}, project)
	p := dataacquisition.Principal{Scope: scope}
	var attempt dataacquisition.SHEINUSTrialAttempt
	var published []collection.Source
	switch action {
	case dataacquisition.SHEINUSTrialView:
		attempt, err = a.trial.View(ctx, p, id)
	case dataacquisition.SHEINUSTrialSend:
		attempt, published, err = a.trial.Run(ctx, p, id)
	case dataacquisition.SHEINUSTrialReconcile:
		attempt, err = a.trial.Reconcile(ctx, p, id)
	default:
		err = dataacquisition.ErrInvalid
	}
	if err != nil && (!errors.Is(err, dataacquisition.ErrUnknown) || attempt.Credits != nil) {
		writeSHEINTrialError(c, err)
		return
	}
	if attempt.ApprovalID != id || attempt.ClaimedAt.IsZero() {
		writeSHEINTrialError(c, dataacquisition.ErrUnavailable)
		return
	}
	state, status := "CONFIRMED", http.StatusOK
	if attempt.Credits == nil {
		state = "UNKNOWN"
		if action != dataacquisition.SHEINUSTrialView {
			status = http.StatusConflict
		}
	} else if attempt.OverLimit() {
		state = "OVER_LIMIT"
	}
	response := gin.H{"approvalId": id, "requestId": attempt.RequestID, "credits": attempt.Credits, "reservedCredits": attempt.ReservedCredits(), "state": state}
	if action == dataacquisition.SHEINUSTrialSend && err == nil {
		response["publishedCount"] = len(published)
	}
	c.JSON(status, response)
}

func writeSHEINTrialError(c *gin.Context, err error) {
	status, code := http.StatusServiceUnavailable, "SHEIN_TRIAL_UNAVAILABLE"
	switch {
	case errors.Is(err, dataacquisition.ErrInvalid):
		status, code = http.StatusBadRequest, "INVALID_REQUEST"
	case errors.Is(err, dataacquisition.ErrForbidden):
		status, code = http.StatusForbidden, "FORBIDDEN"
	case errors.Is(err, dataacquisition.ErrNotFound):
		status, code = http.StatusNotFound, "NOT_FOUND"
	case errors.Is(err, dataacquisition.ErrConflict):
		status, code = http.StatusConflict, "SHEIN_TRIAL_CONFLICT"
	}
	c.JSON(status, gin.H{"code": code})
}
