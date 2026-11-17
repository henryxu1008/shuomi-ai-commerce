package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/authz"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type unifiedSummaryFixture struct {
	fail bool
	org  string
}

func (r *unifiedSummaryFixture) ReadServiceSummary(_ context.Context, org string, _ time.Time) (storecenter.ServiceSummary, error) {
	r.org = org
	if r.fail {
		return storecenter.ServiceSummary{}, errors.New("synthetic read failure")
	}
	return storecenter.ServiceSummary{Records: 3, Active: 2}, nil
}

func TestUnifiedOverviewUsesIndependentAvailabilityAndTrustedScope(t *testing.T) {
	resources := &resourceBalanceFixture{}
	stores := &unifiedSummaryFixture{fail: true}
	module := unifiedCommercialModule{resources: resources, stores: stores}
	routes := kernelmodule.NewRegistry()
	require.NoError(t, module.Register(routes))
	grants := &auditHTTPGrants{role: authz.EnterpriseRoleKey("B", 1)}
	server := buildCurrentApplicationHTTPServer(routes.Routes(), routeAuthDependencies{workbenchVerifier: applicationVerifier{}, organizationResolver: workbenchcontext.NewResolver(grants, "project", "v1", nil), authorizer: currentRoleTestAuthorizer()})
	get := func(path, org, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodGet, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer fixture")
		request.Header.Set("X-Requested-Organization-ID", org)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		return response
	}
	response := get("/api/v1/workbench/commercial/overview", "B", "")
	require.Equal(t, 200, response.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, "unified-base-prepaid-v1", payload["schema_version"])
	require.Equal(t, "B", payload["organization_id"])
	require.Equal(t, false, payload["base_plan"].(map[string]any)["subscription_required"])
	require.Equal(t, "available", payload["resources"].(map[string]any)["state"])
	require.Equal(t, "unavailable", payload["store_services"].(map[string]any)["state"])
	require.Equal(t, "B", stores.org)
	require.Contains(t, response.Header().Get("Cache-Control"), "no-store")
	resources.failure = true
	stores.fail = false
	response = get("/api/v1/workbench/commercial/overview", "B", "")
	require.Equal(t, 200, response.Code)
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, "unavailable", payload["resources"].(map[string]any)["state"])
	require.Equal(t, "available", payload["store_services"].(map[string]any)["state"])
	for _, test := range []struct {
		path, org, body string
		code            int
	}{{"/api/v1/workbench/commercial/overview?organization_id=A", "B", "", 400}, {"/api/v1/workbench/commercial/overview", "A", "", 403}, {"/api/v1/workbench/commercial/overview", "B", "{}", 400}} {
		before := resources.calls
		response = get(test.path, test.org, test.body)
		require.Equal(t, test.code, response.Code)
		require.Equal(t, before, resources.calls)
	}
}

type resourceEventFixture struct {
	query orgresource.EventQuery
	calls int
}

func (r *resourceEventFixture) ListEvents(_ context.Context, q orgresource.EventQuery) (orgresource.EventPage, error) {
	r.calls++
	r.query = q
	if q.Cursor == "invalid" {
		return orgresource.EventPage{}, orgresource.ErrInvalidInput
	}
	return orgresource.EventPage{OrganizationID: q.OrganizationID, Items: []orgresource.EventView{}}, nil
}
func TestResourceEventsUseBoundedFiltersAndLiveOrganization(t *testing.T) {
	reader := &resourceEventFixture{}
	grants := &auditHTTPGrants{role: authz.EnterpriseRoleKey("B", 1)}
	module := commercialResourcesModule{reader: &resourceBalanceFixture{}, events: reader}
	server := buildCurrentApplicationHTTPServer(module.routes(), routeAuthDependencies{workbenchVerifier: applicationVerifier{}, organizationResolver: workbenchcontext.NewResolver(grants, "project", "v1", nil), authorizer: currentRoleTestAuthorizer()})
	get := func(suffix, org, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("GET", commercialResourceEventsPath+suffix, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer fixture")
		request.Header.Set("X-Requested-Organization-ID", org)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		return response
	}
	response := get("?deductions_only=true&resource_type=ai_point&limit=2&from=2026-09-01T00:00:00Z&until=2026-10-01T00:00:00Z", "B", "")
	require.Equal(t, 200, response.Code)
	require.Equal(t, "B", reader.query.OrganizationID)
	require.Equal(t, 2, reader.query.Limit)
	require.Equal(t, orgresource.ResourceAIPoint, reader.query.ResourceType)
	require.True(t, reader.query.DeductionsOnly)
	require.Equal(t, 200, get("?deductions_only=false", "B", "").Code)
	require.False(t, reader.query.DeductionsOnly)
	for _, query := range []string{"?organization_id=A", "?limit=51", "?limit=2&limit=3", "?resource_type=token", "?deductions_only=True", "?deductions_only=true&deductions_only=false", "?from=bad", "?from=2026-10-01T00:00:00Z&until=2026-09-01T00:00:00Z", "?"} {
		before := reader.calls
		require.Equal(t, 400, get(query, "B", "").Code)
		require.Equal(t, before, reader.calls)
	}
	require.Equal(t, 400, get("?cursor=invalid", "B", "").Code)
	before := reader.calls
	require.Equal(t, 403, get("", "A", "").Code)
	require.Equal(t, 400, get("", "B", "{}").Code)
	grants.revoked = true
	require.Equal(t, 403, get("", "B", "").Code)
	require.Equal(t, before, reader.calls)
}

func TestUsageObjectRouteRejectsOpaqueTargetInput(t *testing.T) {
	grants := &auditHTTPGrants{role: authz.EnterpriseRoleKey("B", 1)}
	module := commercialResourcesModule{reader: &resourceBalanceFixture{}}
	server := buildCurrentApplicationHTTPServer(module.routes(), routeAuthDependencies{workbenchVerifier: applicationVerifier{}, organizationResolver: workbenchcontext.NewResolver(grants, "project", "v1", nil), authorizer: currentRoleTestAuthorizer()})
	for _, path := range []string{"/not-an-event/object", "/55555555-5555-4555-8555-555555555555/object?source_identity=some-store"} {
		request := httptest.NewRequest("GET", commercialResourceEventsPath+path, nil)
		request.Header.Set("Authorization", "Bearer fixture")
		request.Header.Set("X-Requested-Organization-ID", "B")
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		require.Equal(t, 400, response.Code, response.Body.String())
	}
}

func TestUsageObjectUsesOnlyOwnedEventSource(t *testing.T) {
	const eventID = "55555555-5555-4555-8555-555555555555"
	const sourceID = "66666666-6666-4666-8666-666666666666"
	grants := &auditHTTPGrants{role: authz.EnterpriseRoleKey("B", 1)}
	event := orgresource.EventSource{OrganizationID: "B", EventID: eventID, ResourceType: orgresource.ResourceStoreRenewalPeriod,
		SourceType: string(orgresource.ConsumerStoreService), SourceIdentity: sourceID}
	found := true
	reads, resolutions := 0, 0
	var resolvedPurpose string
	checkSource := func(org, source string) {
		resolutions++
		require.Equal(t, "B", org)
		require.Equal(t, sourceID, source)
	}
	module := commercialResourcesModule{reader: &resourceBalanceFixture{}, objects: &commercialUsageLookup{
		readEvent: func(_ context.Context, org, id string) (orgresource.EventSource, bool, error) {
			reads++
			require.Equal(t, "B", org)
			require.Equal(t, eventID, id)
			return event, found, nil
		},
		store: func(_ context.Context, org, source string) *commercialUsageObject {
			checkSource(org, source)
			return usageObject("store", "Fixture store", sourceID)
		},
		amazon: func(_ context.Context, org, source string) (string, error) {
			checkSource(org, source)
			return "Fixture product", nil
		},
		model: func(_ *gin.Context, org, source string) (*commercialUsageObject, string) {
			checkSource(org, source)
			return nil, resolvedPurpose
		},
	}}
	server := buildCurrentApplicationHTTPServer(module.routes(), routeAuthDependencies{workbenchVerifier: applicationVerifier{}, organizationResolver: workbenchcontext.NewResolver(grants, "project", "v1", nil), authorizer: currentRoleTestAuthorizer()})
	get := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, commercialResourceEventsPath+"/"+eventID+"/object", nil)
		req.Header.Set("Authorization", "Bearer fixture")
		req.Header.Set("X-Requested-Organization-ID", "B")
		w := httptest.NewRecorder()
		server.Handler.ServeHTTP(w, req)
		return w
	}
	response := get()
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, "private, no-store", response.Header().Get("Cache-Control"))
	require.JSONEq(t, `{"organization_id":"B","event_id":"`+eventID+`","object":{"kind":"store","name":"Fixture store","id":"`+sourceID+`"}}`, response.Body.String())
	event.ResourceType = orgresource.ResourceDataRow
	event.SourceType = string(orgresource.ConsumerAmazonData)
	response = get()
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `{"organization_id":"B","event_id":"`+eventID+`","object":{"kind":"amazon_product","name":"Fixture product"}}`, response.Body.String())
	event.ResourceType = orgresource.ResourceAIPoint
	event.SourceType = "model_invocation_v1"
	response = get()
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `{"organization_id":"B","event_id":"`+eventID+`","object":null}`, response.Body.String())
	resolvedPurpose = "chat_planning"
	response = get()
	require.Equal(t, 200, response.Code)
	require.JSONEq(t, `{"organization_id":"B","event_id":"`+eventID+`","object":null,"purpose":"chat_planning"}`, response.Body.String())
	require.Equal(t, 4, resolutions)
	event.ResourceType = orgresource.ResourceDataRow
	require.Contains(t, get().Body.String(), `"object":null`)
	event.OrganizationID = "A"
	require.Equal(t, 404, get().Code)
	found = false
	event.OrganizationID = "B"
	require.Equal(t, 404, get().Code)
	require.Equal(t, 4, resolutions)
	require.Equal(t, 7, reads)
	grants.revoked = true
	require.Equal(t, 403, get().Code)
	require.Equal(t, 7, reads)
}

func TestUsageObjectTaskRequiresExactRunRequest(t *testing.T) {
	const taskID = "77777777-7777-4777-8777-777777777777"
	scope := aiworkbench.Scope{OrganizationID: "B", ActorID: "actor"}
	request := agent.Request{Key: "execution", GoalSummary: "Original task"}
	raw, err := json.Marshal(request)
	require.NoError(t, err)
	task := aiworkbench.BusinessTask{ID: taskID, Scope: scope, Kind: aiworkbench.ContentTaskKind,
		ExecutionRequestKey: request.Key, ExecutionRequest: raw}
	require.True(t, usageTaskMatches(task, scope, aiworkbench.ContentTaskKind, "", request))
	require.True(t, usageTaskMatches(task, scope, aiworkbench.ContentTaskKind, taskID, request))
	require.False(t, usageTaskMatches(task, scope, aiworkbench.TitleTaskKind, "", request))
	require.False(t, usageTaskMatches(task, scope, aiworkbench.ContentTaskKind, "some-other-task", request))
	require.False(t, usageTaskMatches(task, aiworkbench.Scope{OrganizationID: "B", ActorID: "other"}, aiworkbench.ContentTaskKind, "", request))
	request.GoalSummary = "A different task"
	require.False(t, usageTaskMatches(task, scope, aiworkbench.ContentTaskKind, "", request))
}

func TestUsageInvocationPurposeUsesRecordedOperation(t *testing.T) {
	for _, entry := range []struct {
		operation aicapability.Operation
		taskID    string
		purpose   string
	}{
		{operation: aicapability.OperationAIWorkbenchChatPlan, purpose: "chat_planning"},
		{operation: aicapability.OperationOwnProductIntake, purpose: "own_product_intake"},
		{operation: aicapability.OperationOwnProductIntake, taskID: "bound-task"},
		{operation: aicapability.OperationProductContentDecision},
		{operation: aicapability.OperationAIWorkbenchStoreRoute},
	} {
		fact := aicapability.InvocationRecord{Operation: entry.operation, BusinessTaskID: entry.taskID}
		require.Equal(t, entry.purpose, usageInvocationPurpose(fact), "operation %s, task %s", entry.operation, entry.taskID)
	}
}
