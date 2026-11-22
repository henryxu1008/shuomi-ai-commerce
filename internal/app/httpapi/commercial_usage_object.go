package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

const commercialResourceObjectPath = commercialResourceEventsPath + "/:event_id/object"

type commercialUsageObject struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

type commercialUsageLookup struct {
	readEvent     func(context.Context, string, string) (orgresource.EventSource, bool, error)
	store         func(context.Context, string, string) *commercialUsageObject
	amazon        func(context.Context, string, string) (string, error)
	model         func(*gin.Context, string, string) (*commercialUsageObject, string)
	modelPurposes func(*gin.Context, string, []string) map[string]string
	ownedModels   func(*gin.Context, string, string) (ownedModelUsagePage, error)
}

func (m commercialResourcesModule) readUsageObject(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	if !readCommercialEmptyBody(c) {
		return
	}
	eventID := c.Param("event_id")
	if !acquisitionHTTPUUID(eventID) || c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		writeWorkbenchProtocolError(c, 400, "INVALID_REQUEST", "Usage event is invalid")
		return
	}
	ctx := c.Request.Context()
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !authidentity.IsBoundedIdentifier(identity.UserID) {
		writeWorkbenchProtocolError(c, 401, "AUTHENTICATION_REQUIRED", "Authentication is required")
		return
	}
	org := identity.EffectiveOrganizationID
	if !authidentity.IsBoundedIdentifier(org) || identity.TenantID != org {
		writeWorkbenchProtocolError(c, 403, "ORGANIZATION_ACCESS_DENIED", "Organization access is denied")
		return
	}
	if m.objects == nil || m.objects.readEvent == nil {
		writeWorkbenchProtocolError(c, 503, "DEPENDENCY_UNAVAILABLE", "Usage details are unavailable")
		return
	}
	event, found, err := m.objects.readEvent(ctx, org, eventID)
	if err != nil {
		writeWorkbenchProtocolError(c, 503, "DEPENDENCY_UNAVAILABLE", "Usage details are unavailable")
		return
	}
	if !found || event.OrganizationID != org || event.EventID != eventID {
		writeWorkbenchProtocolError(c, 404, "NOT_FOUND", "Usage event was not found")
		return
	}
	var object *commercialUsageObject
	var purpose string
	switch {
	case event.ResourceType == orgresource.ResourceStoreRenewalPeriod && event.SourceType == string(orgresource.ConsumerStoreService):
		if m.objects.store != nil {
			object = m.objects.store(ctx, org, event.SourceIdentity)
		}
	case event.ResourceType == orgresource.ResourceDataRow && event.SourceType == string(orgresource.ConsumerAmazonData):
		if m.objects.amazon != nil {
			name, err := m.objects.amazon(ctx, org, event.SourceIdentity)
			if err == nil {
				object = usageObject("amazon_product", name, "")
			}
		}
	case isModelUsageEvent(event.ResourceType, event.SourceType):
		if m.objects.model != nil {
			object, purpose = m.objects.model(c, org, event.SourceIdentity)
		}
	}
	response := gin.H{"organization_id": org, "event_id": eventID, "object": object}
	if purpose != "" {
		response["purpose"] = purpose
	}
	c.JSON(http.StatusOK, response)
}

func usageObject(kind, name, id string) *commercialUsageObject {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsFunc(name, unicode.IsControl) {
		return nil
	}
	if chars := []rune(name); len(chars) > 240 {
		name = strings.TrimSpace(string(chars[:240]))
	}
	return &commercialUsageObject{Kind: kind, Name: name, ID: id}
}

func storeUsageObjectReader(stores *storecenter.MemberScopedStoreRepository) func(context.Context, string, string) *commercialUsageObject {
	return func(ctx context.Context, org, operationID string) *commercialUsageObject {
		if !acquisitionHTTPUUID(operationID) {
			return nil
		}
		intent, err := stores.ReadServiceChargeIntent(ctx, org, operationID)
		execution := intent.Execution
		if err != nil || execution.OrganizationID != org || execution.OperationID != operationID || !acquisitionHTTPUUID(execution.StoreID) {
			return nil
		}
		store, err := stores.Get(ctx, org, execution.StoreID)
		if err != nil || store == nil || store.ID() != execution.StoreID || store.OrganizationID() != org {
			return nil
		}
		return usageObject("store", store.Name(), store.ID())
	}
}

func (a *aiWorkbenchApplication) bindUsageModel(c *gin.Context, org string) (context.Context, authidentity.AuthenticatedIdentity, bool) {
	if a == nil || a.agent == nil || a.agent.config.Ledger == nil {
		return nil, authidentity.AuthenticatedIdentity{}, false
	}
	ctx, scope, err := a.bind(c, authz.PermissionWorkbenchTaskRead)
	if err != nil || scope.OrganizationID != org {
		return nil, authidentity.AuthenticatedIdentity{}, false
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	return ctx, identity, ok && identity.EffectiveMemberID != ""
}

func (a *aiWorkbenchApplication) usageModelDetails(c *gin.Context, org, invocationID string) (*commercialUsageObject, string) {
	ctx, identity, ok := a.bindUsageModel(c, org)
	if !ok {
		return nil, ""
	}
	fact, err := a.agent.config.Ledger.ReadModelInvocation(ctx, org, invocationID)
	if err != nil || fact.InvocationID != invocationID || !usageInvocationOwnedBy(fact, identity) {
		return nil, ""
	}
	if purpose := usageInvocationPurpose(fact); purpose != "" {
		return nil, purpose
	}
	scope := aiworkbench.Scope{OrganizationID: org, ActorID: identity.UserID}
	return a.usageTaskObject(ctx, scope, fact), ""
}

func usageInvocationOwnedBy(fact aicapability.InvocationRecord, identity authidentity.AuthenticatedIdentity) bool {
	return fact.TenantID == identity.EffectiveOrganizationID && fact.UserID == identity.UserID && fact.MemberID == identity.EffectiveMemberID
}

type modelPurposeInvocationReader interface {
	ReadModelPurposeInvocations(context.Context, string, string, string, []string) ([]aicapability.InvocationRecord, error)
}

func (a *aiWorkbenchApplication) usageModelPurposes(c *gin.Context, org string, ids []string) map[string]string {
	ctx, identity, ok := a.bindUsageModel(c, org)
	if !ok {
		return nil
	}
	reader, ok := a.agent.config.Ledger.(modelPurposeInvocationReader)
	if !ok {
		return nil
	}
	facts, err := reader.ReadModelPurposeInvocations(ctx, org, identity.UserID, identity.EffectiveMemberID, ids)
	if err != nil {
		return nil
	}
	purposes := make(map[string]string, len(facts))
	for _, fact := range facts {
		if !usageInvocationOwnedBy(fact, identity) {
			continue
		}
		if purpose := usageInvocationPurpose(fact); purpose != "" {
			purposes[fact.InvocationID] = purpose
		}
	}
	return purposes
}

func isModelUsageEvent(resource orgresource.ResourceType, source string) bool {
	return resource == orgresource.ResourceAIPoint && source == "model_invocation_v1"
}

func (lookup commercialUsageLookup) withModelPurposes(c *gin.Context, page orgresource.EventPage) any {
	ids := make([]string, 0, len(page.Items))
	seen := make(map[string]bool, len(page.Items))
	for _, event := range page.Items {
		id := event.SourceIdentity
		if isModelUsageEvent(event.ResourceType, event.SourceType) && agent.ValidID(id) && !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return page
	}
	purposes := lookup.modelPurposes(c, page.OrganizationID, ids)
	if len(purposes) == 0 {
		return page
	}
	type eventWithPurpose struct {
		orgresource.EventView
		Purpose string `json:"purpose,omitempty"`
	}
	items := make([]eventWithPurpose, 0, len(page.Items))
	for _, event := range page.Items {
		view := eventWithPurpose{EventView: event}
		if isModelUsageEvent(event.ResourceType, event.SourceType) && seen[event.SourceIdentity] {
			view.Purpose = purposes[event.SourceIdentity]
		}
		items = append(items, view)
	}
	return gin.H{"organization_id": page.OrganizationID, "items": items, "next_cursor": page.NextCursor}
}

func usageInvocationPurpose(fact aicapability.InvocationRecord) string {
	if fact.BusinessTaskID != "" {
		return ""
	}
	switch fact.Operation {
	case aicapability.OperationAIWorkbenchChatIntent:
		return "chat_intent"
	case aicapability.OperationAIWorkbenchChatPlan:
		return "chat_planning"
	case aicapability.OperationAIWorkbenchChatImages:
		return "chat_images"
	case aicapability.OperationAIWorkbenchSkillAuthoring:
		return "skill_authoring"
	case aicapability.OperationOwnProductIntake:
		return "own_product_intake"
	default:
		return ""
	}
}

func (a *aiWorkbenchApplication) usageTaskObject(ctx context.Context, scope aiworkbench.Scope, fact aicapability.InvocationRecord) *commercialUsageObject {
	if a.agent.store == nil || a.store == nil || fact.AgentRunID == "" {
		return nil
	}
	var kind string
	switch fact.Operation {
	case aicapability.OperationProductAgentDecision:
		kind = aiworkbench.TitleTaskKind
	case aicapability.OperationProductContentDecision:
		kind = aiworkbench.ContentTaskKind
	default:
		return nil
	}
	runScope := agent.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}
	run, found, err := a.agent.store.LookupRunID(ctx, runScope, fact.AgentRunID)
	if err != nil || !found || run.State.RunID != fact.AgentRunID || run.State.Scope != runScope || run.State.Request.Key == "" {
		return nil
	}
	task, err := a.store.ReadTaskByExecution(ctx, scope, run.State.Request.Key)
	if err != nil || !usageTaskMatches(task, scope, kind, fact.BusinessTaskID, run.State.Request) {
		return nil
	}
	view, err := a.taskView(ctx, scope, task, false)
	if err != nil || view.ID != task.ID || view.ScopeUnavailable || view.HistoryUnavailable {
		return nil
	}
	return usageObject("task", view.Title, view.ID)
}

func usageTaskMatches(task aiworkbench.BusinessTask, scope aiworkbench.Scope, kind, taskID string, original agent.Request) bool {
	if task.Scope != scope || task.Kind != kind || !acquisitionHTTPUUID(task.ID) ||
		(taskID != "" && taskID != task.ID) || task.ExecutionRequestKey != original.Key {
		return false
	}
	var request agent.Request
	return json.Unmarshal(task.ExecutionRequest, &request) == nil && request == original
}
