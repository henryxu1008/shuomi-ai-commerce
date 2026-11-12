package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"strconv"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"time"
)

const workbenchScheduleBase = "/api/v1/workbench/schedules"

func aiWorkbenchScheduleRoutes(a *aiWorkbenchApplication) []httproute.Descriptor {
	var routes []httproute.Descriptor
	for _, spec := range []struct{ method, path, action, permission string }{
		{http.MethodGet, workbenchScheduleBase + "/availability", "availability", authz.PermissionWorkbenchTaskRead},
		{http.MethodGet, workbenchScheduleBase, "list", authz.PermissionWorkbenchTaskRead},
		{http.MethodPost, workbenchScheduleBase + "/quote", "quote", authz.PermissionWorkbenchAgentUse},
		{http.MethodPost, workbenchScheduleBase, "create", authz.PermissionWorkbenchAgentUse},
		{http.MethodGet, workbenchScheduleBase + "/:schedule_id", "read", authz.PermissionWorkbenchTaskRead},
		{http.MethodGet, workbenchScheduleBase + "/:schedule_id/occurrences", "history", authz.PermissionWorkbenchTaskRead},
		{http.MethodPatch, workbenchScheduleBase + "/:schedule_id", "edit", authz.PermissionWorkbenchAgentUse},
		{http.MethodPost, workbenchScheduleBase + "/:schedule_id/pause", "pause", authz.PermissionWorkbenchAgentUse},
		{http.MethodPost, workbenchScheduleBase + "/:schedule_id/resume", "resume", authz.PermissionWorkbenchAgentUse},
		{http.MethodPost, workbenchScheduleBase + "/:schedule_id/close", "close", authz.PermissionWorkbenchAgentUse},
	} {
		spec := spec
		routes = append(routes, httproute.Descriptor{Method: spec.method, Path: spec.path, Module: "ai-workbench", Permission: spec.permission, AuthPolicy: httproute.AuthPolicyVerifiedIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 30 * time.Second, Handler: func(c *gin.Context) {
			if a == nil {
				writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
				return
			}
			ctx, owner, err := a.bind(c, spec.permission)
			if err != nil {
				writeAIWorkbenchError(c, err)
				return
			}
			if c.Request.Method == http.MethodGet && workbenchEmptyBody(c) != nil {
				writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
				return
			}
			if spec.action == "availability" {
				if c.Request.URL.RawQuery != "" {
					writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
					return
				}
				enabled := a.schedules != nil && a.agent != nil && !a.conversationOnly
				workbenchReply(c, 200, gin.H{"enabled": enabled, "ready": enabled && a.schedules.Ready()})
				return
			}
			if a.schedules == nil || a.agent == nil || a.conversationOnly {
				writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
				return
			}
			if spec.action == "list" {
				a.listScheduled(c, ctx, owner)
				return
			}
			if spec.action == "quote" || spec.action == "create" {
				if spec.action == "quote" && !a.schedules.Ready() {
					writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
					return
				}
				if c.Request.URL.RawQuery != "" {
					writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
					return
				}
				var input scheduledInput
				if err = workbenchJSON(c, &input); err != nil {
					writeAIWorkbenchError(c, err)
					return
				}
				if spec.action == "quote" {
					quote, _, _, err := a.quoteScheduled(ctx, owner, input)
					if err != nil {
						writeAIWorkbenchError(c, err)
						return
					}
					workbenchReply(c, 200, quote)
					return
				}
				a.createScheduled(c, ctx, owner, input)
				return
			}
			id := c.Param("schedule_id")
			if !acquisitionHTTPUUID(id) {
				writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
				return
			}
			plan, err := a.store.ReadScheduledPlan(ctx, owner, id)
			if err != nil {
				writeAIWorkbenchError(c, err)
				return
			}
			if spec.action == "history" {
				a.scheduledHistory(c, ctx, owner, id)
				return
			}
			if c.Request.URL.RawQuery != "" {
				writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
				return
			}
			if spec.action == "read" {
				workbenchReply(c, 200, gin.H{"plan": a.scheduledView(ctx, plan)})
				return
			}
			a.changeScheduled(c, ctx, owner, plan, spec.action)
		}})
	}
	return routes
}

func (a *aiWorkbenchApplication) createScheduled(c *gin.Context, ctx context.Context, owner aiworkbench.Scope, input scheduledInput) {
	key, err := workbenchKey(c)
	if err != nil || input.ScheduleID != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	id := uuid.NewSHA1(uuid.NameSpaceURL, []byte("workbench-scheduled-plan\x1f"+owner.OrganizationID+"\x1f"+owner.ActorID+"\x1f"+key)).String()
	fingerprint := scheduledHash(input)
	plan, err := a.store.ReadScheduledPlan(ctx, owner, id)
	if err == nil {
		if plan.CreationFingerprint != fingerprint {
			writeAIWorkbenchError(c, aiworkbench.ErrIdempotencyConflict)
			return
		}
		a.replyScheduledSave(c, ctx, plan)
		return
	}
	if !errors.Is(err, aiworkbench.ErrNotFound) {
		writeAIWorkbenchError(c, err)
		return
	}
	quote, p, budget, err := a.quoteScheduled(ctx, owner, input)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	if input.Confirmation == "" || input.Confirmation != quote.Confirmation {
		writeAIWorkbenchError(c, aiworkbench.ErrRevisionMismatch)
		return
	}
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	spec, err := json.Marshal(p)
	if err != nil {
		writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
		return
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	draft := aiworkbench.ScheduledPlanDraft{ID: id, Scope: owner, MemberID: i.EffectiveMemberID, Selection: *p.Selection(), GoalSummary: p.GoalSummary, Rule: input.Rule, Budget: budget, CreatedAt: now, RuleStartAt: now, ExecutionSpec: spec, CreationFingerprint: fingerprint}
	createErr := a.store.CreateScheduledPlan(ctx, draft)
	plan, err = a.store.ReadScheduledPlan(ctx, owner, id)
	if err != nil {
		writeAIWorkbenchError(c, errors.Join(createErr, err))
		return
	}
	if plan.CreationFingerprint != fingerprint {
		writeAIWorkbenchError(c, aiworkbench.ErrIdempotencyConflict)
		return
	}
	a.replyScheduledSave(c, ctx, plan)
}

func (a *aiWorkbenchApplication) replyScheduledSave(c *gin.Context, ctx context.Context, plan aiworkbench.ScheduledPlan) {
	if plan.RegistrationState != "MATCHED" {
		_ = a.synchronizeScheduledPlan(ctx, plan)
	}
	current, err := a.store.ReadScheduledPlan(ctx, plan.Scope, plan.ID)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	workbenchReply(c, 200, gin.H{"plan": a.scheduledView(ctx, current), "planSaved": true})
}

func (a *aiWorkbenchApplication) changeScheduled(c *gin.Context, ctx context.Context, owner aiworkbench.Scope, plan aiworkbench.ScheduledPlan, action string) {
	key, err := workbenchKey(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	var body struct {
		ExpectedRevision string                    `json:"expectedRevision"`
		Rule             *aiworkbench.ScheduleRule `json:"rule,omitempty"`
		Confirmation     string                    `json:"confirmation,omitempty"`
	}
	if err = workbenchJSON(c, &body); err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	revision, err := strconv.ParseInt(body.ExpectedRevision, 10, 63)
	if err != nil || revision < 1 || strconv.FormatInt(revision, 10) != body.ExpectedRevision || (action == "edit") != (body.Rule != nil) || action != "edit" && body.Confirmation != "" {
		writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
		return
	}
	if saved, found, err := a.store.ReadScheduledControlReceipt(ctx, owner, plan.ID, key, revision, action, body.Rule); err != nil {
		writeAIWorkbenchError(c, err)
		return
	} else if found {
		if action == "edit" {
			var p aiworkbench.ExecutionProposal
			if json.Unmarshal(saved.ExecutionSpec, &p) != nil {
				writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
				return
			}
			if body.Confirmation != scheduledQuoteConfirmation(p, *body.Rule, saved.Budget) {
				writeAIWorkbenchError(c, aiworkbench.ErrIdempotencyConflict)
				return
			}
		}
		a.replyScheduledSave(c, ctx, saved)
		return
	}
	if action == "resume" {
		var p aiworkbench.ExecutionProposal
		if json.Unmarshal(plan.ExecutionSpec, &p) != nil {
			writeAIWorkbenchError(c, aiworkbench.ErrUnavailable)
			return
		}
		if _, err = a.validateScheduledProposal(ctx, p); err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
	}
	if action == "edit" {
		quote, _, _, err := a.quoteScheduled(ctx, owner, scheduledInput{ScheduleID: plan.ID, Rule: *body.Rule, MaxOccurrences: strconv.FormatUint(plan.Budget.MaxOccurrences, 10)})
		if err != nil {
			writeAIWorkbenchError(c, err)
			return
		}
		if body.Confirmation != quote.Confirmation {
			writeAIWorkbenchError(c, aiworkbench.ErrRevisionMismatch)
			return
		}
	}
	changed, err := a.store.ChangeScheduledPlan(ctx, owner, plan.ID, key, revision, action, body.Rule, time.Now().UTC())
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	a.replyScheduledSave(c, ctx, changed)
}

func (a *aiWorkbenchApplication) listScheduled(c *gin.Context, ctx context.Context, owner aiworkbench.Scope) {
	for name, values := range c.Request.URL.Query() {
		if len(values) != 1 || (name != "limit" && name != "after") {
			writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
			return
		}
	}
	limit, err := workbenchSize(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	plans, next, err := a.store.ListScheduledPlans(ctx, owner, c.Query("after"), limit)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	views := make([]scheduledPlanView, 0, len(plans))
	for _, plan := range plans {
		views = append(views, a.scheduledView(ctx, plan))
	}
	workbenchReply(c, 200, gin.H{"plans": views, "next": next})
}

func (a *aiWorkbenchApplication) scheduledHistory(c *gin.Context, ctx context.Context, owner aiworkbench.Scope, id string) {
	for name, values := range c.Request.URL.Query() {
		if len(values) != 1 || (name != "limit" && name != "after") {
			writeAIWorkbenchError(c, aiworkbench.ErrInvalid)
			return
		}
	}
	limit, err := workbenchSize(c)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	items, next, err := a.store.ListScheduledOccurrences(ctx, owner, id, c.Query("after"), limit)
	if err != nil {
		writeAIWorkbenchError(c, err)
		return
	}
	type occurrenceView struct {
		ID             string    `json:"id"`
		ScheduledAt    time.Time `json:"scheduledAt"`
		PlanRevision   string    `json:"planRevision"`
		State          string    `json:"state"`
		Reason         string    `json:"reason,omitempty"`
		BusinessTaskID string    `json:"businessTaskId,omitempty"`
	}
	views := make([]occurrenceView, 0, len(items))
	for _, item := range items {
		views = append(views, occurrenceView{ID: item.ID, ScheduledAt: item.ScheduledAtUTC, PlanRevision: strconv.FormatInt(item.PlanRevision, 10), State: item.State, Reason: item.Reason, BusinessTaskID: item.BusinessTaskID})
	}
	workbenchReply(c, 200, gin.H{"occurrences": views, "next": next})
}
