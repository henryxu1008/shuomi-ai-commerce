package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/httproute"
	kernel "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	rh "task-processor/internal/product/research/httpapi"
)

func TestSHEINTrialRoutesRequireExplicitAdmission(t *testing.T) {
	var routes []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		routes = append(routes, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	registry := kernel.NewRegistry()
	require.NoError(t, (researchModule{}).Register(registry))
	routes = append(routes, registry.Routes()...)
	routes = append(routes, marketResearchAnalysisRoutes(nil, nil)...)
	validate := func(rs []httproute.Descriptor, opts currentApplicationOptionalRoutes) error {
		return validateCurrentApplicationRoutesInternal(rs, false, false, false, false, false, false, false, opts)
	}
	var options currentApplicationOptions
	require.Nil(t, options.sheinTrial)
	withSHEINUSTrial(sheinTrialDependencies{})(&options)
	require.False(t, options.sheinTrial.complete())
	base := currentApplicationOptionalRoutes{MarketResearch: true}
	require.NoError(t, validate(routes, base))
	withTrial := append(routes, marketResearchSHEINTrialRoutes(nil)...)
	require.Error(t, validate(withTrial, base))
	require.Error(t, validate(withTrial, currentApplicationOptionalRoutes{SHEINTrial: true}))
	base.SHEINTrial = true
	require.Error(t, validate(routes, base))
	require.NoError(t, validate(withTrial, base))
	for _, weaken := range []func(*httproute.Descriptor){
		func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic },
		func(d *httproute.Descriptor) { d.RejectUnreadRequestBody = false },
		func(d *httproute.Descriptor) {
			d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		},
	} {
		changed := append([]httproute.Descriptor(nil), withTrial...)
		weaken(&changed[len(changed)-1])
		require.Error(t, validate(changed, base))
	}
}

type sheinTrialTestProjects struct{ marketDenied map[collection.Scope]bool }

func (sheinTrialTestProjects) Authorize(context.Context, collection.Scope, string, bool) error {
	return nil
}

func (a sheinTrialTestProjects) AuthorizeDataMarket(_ context.Context, s collection.Scope) error {
	if a.marketDenied[s] {
		return dataacquisition.ErrForbidden
	}
	return nil
}

type sheinTrialTestAuthority struct {
	sheinResearchTrialAuthority
	allow func(dataacquisition.Principal, dataacquisition.SHEINUSTrialAction) bool
}

func (a sheinTrialTestAuthority) AuthorizeSHEINUSTrial(_ context.Context, p dataacquisition.Principal, id string, action dataacquisition.SHEINUSTrialAction) (dataacquisition.SHEINUSTrialApproval, error) {
	if id != a.approval.ID || !a.allow(p, action) {
		return dataacquisition.SHEINUSTrialApproval{}, dataacquisition.ErrForbidden
	}
	return a.approval, nil
}

func sheinTrialTestHTTP(t *testing.T, ctx context.Context, module *dataservicesapp.Module, deps sheinTrialDependencies, projects sheinTrialProjects, members map[string]collection.Scope) http.Handler {
	t.Helper()
	bind := func(ctx context.Context, bearer string) (context.Context, collection.Scope, error) {
		if s, ok := members[strings.TrimPrefix(bearer, "Bearer ")]; ok {
			return ctx, s, nil
		}
		return ctx, collection.Scope{}, dataacquisition.ErrForbidden
	}
	missing := deps
	missing.lookup = nil
	_, err := composeMarketResearchSHEINTrial(ctx, module, missing, projects, bind)
	require.ErrorIs(t, err, dataacquisition.ErrUnavailable)
	app, err := composeMarketResearchSHEINTrial(ctx, module, deps, projects, bind)
	require.NoError(t, err)
	registry := kernel.NewRegistry()
	require.NoError(t, (researchModule{trial: app}).Register(registry))
	router := gin.New()
	for _, route := range registry.Routes() {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	return router
}

func sheinTrialTestPath(project, id string) string {
	return rh.Base + "/" + project + "/shein-us-trials/" + id
}

func sheinTrialTestRequest(t *testing.T, server http.Handler, method, path, who, body string, status int) string {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+who)
	receipt := httptest.NewRecorder()
	server.ServeHTTP(receipt, request)
	require.Equal(t, status, receipt.Code, receipt.Body.String())
	require.Equal(t, "private, no-store", receipt.Header().Get("Cache-Control"))
	return receipt.Body.String()
}

func newSHEINTrialTestApproval(access *sheinResearchAccess) dataacquisition.SHEINUSTrialApproval {
	return dataacquisition.SHEINUSTrialApproval{
		ID: uuid.NewString(), ProjectID: access.project, Scope: access.scope, AccountRef: uuid.NewString(),
		KeyRef: uuid.NewString(), Funding: orgresource.FundingMember, ExpiresAt: time.Now().Add(time.Minute),
	}
}

func verifySHEINTrialRecovery(t *testing.T, ctx context.Context, module *dataservicesapp.Module, access *sheinResearchAccess) {
	t.Helper()
	a := newSHEINTrialTestApproval(access)
	accountant := collection.Scope{OrganizationID: a.Scope.OrganizationID, ActorID: "fixture-accountant", MemberID: "fixture-accountant-member"}
	outsider := collection.Scope{OrganizationID: "fixture-outsider", ActorID: accountant.ActorID, MemberID: accountant.MemberID}
	auth := sheinTrialTestAuthority{sheinResearchTrialAuthority{a}, func(p dataacquisition.Principal, action dataacquisition.SHEINUSTrialAction) bool {
		if action == dataacquisition.SHEINUSTrialReconcile {
			return p.Scope == accountant || p.Scope == outsider
		}
		return p.Scope == a.Scope
	}}
	sends, lookups := 0, 0
	projects := sheinTrialTestProjects{marketDenied: make(map[collection.Scope]bool)}
	revokeDuringLookup := false
	const requestID = "fixture-original-shein-request"
	deps := sheinTrialDependencies{
		authority: auth,
		send: func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			sends++
			return dataacquisition.SHEINUSTrialSubmission{SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{
				ApprovalID: a.ID, Binding: a.Binding(), RequestID: requestID,
			}}, dataacquisition.ErrUnknown
		},
		lookup: func(_ context.Context, _ dataacquisition.SHEINUSTrialApproval, _ dataacquisition.SHEINUSTrialRequest, original dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
			lookups++
			require.Equal(t, requestID, original.RequestID)
			if revokeDuringLookup {
				projects.marketDenied[accountant] = true
			}
			credits := 7
			return dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding(), RequestID: requestID, Credits: &credits}, nil
		},
	}
	server := sheinTrialTestHTTP(t, ctx, module, deps, projects, map[string]collection.Scope{"researcher": a.Scope, "accountant": accountant, "outsider": outsider})
	path := sheinTrialTestPath(a.ProjectID, a.ID)
	wrongProject := sheinTrialTestPath(uuid.NewString(), a.ID)
	sheinTrialTestRequest(t, server, http.MethodPost, wrongProject+"/submit", "researcher", "", http.StatusForbidden)
	sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "unrecognized", "", http.StatusForbidden)
	sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit?url=x", "researcher", "", http.StatusBadRequest)
	sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", `{"credits":1}`, http.StatusBadRequest)
	require.Zero(t, sends)
	unknown := `{"approvalId":"` + a.ID + `","requestId":"` + requestID + `","credits":null,"reservedCredits":25,"state":"UNKNOWN"}`
	require.JSONEq(t, unknown, sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", "", http.StatusConflict))
	require.JSONEq(t, unknown, sheinTrialTestRequest(t, server, http.MethodGet, path, "researcher", "", http.StatusOK))
	require.JSONEq(t, unknown, sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", "", http.StatusConflict))
	require.Zero(t, lookups)
	forbidden := `{"code":"FORBIDDEN"}`
	projects.marketDenied[a.Scope] = true
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", "", http.StatusForbidden))
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodGet, path, "researcher", "", http.StatusForbidden))
	delete(projects.marketDenied, a.Scope)
	revokeDuringLookup = true
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodPost, path+"/reconcile", "accountant", "", http.StatusForbidden))
	require.JSONEq(t, unknown, sheinTrialTestRequest(t, server, http.MethodGet, path, "researcher", "", http.StatusOK))
	revokeDuringLookup = false
	delete(projects.marketDenied, accountant)
	access.denied = true
	defer func() { access.denied = false }()
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodGet, path, "researcher", "", http.StatusForbidden))
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodGet, path, "accountant", "", http.StatusForbidden))
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "accountant", "", http.StatusForbidden))
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodPost, wrongProject+"/reconcile", "accountant", "", http.StatusForbidden))
	require.JSONEq(t, forbidden, sheinTrialTestRequest(t, server, http.MethodPost, path+"/reconcile", "outsider", "", http.StatusForbidden))
	require.Equal(t, 1, lookups)
	settled := `{"approvalId":"` + a.ID + `","requestId":"` + requestID + `","credits":7,"reservedCredits":0,"state":"CONFIRMED"}`
	require.JSONEq(t, settled, sheinTrialTestRequest(t, server, http.MethodPost, path+"/reconcile", "accountant", "", http.StatusOK))
	require.JSONEq(t, settled, sheinTrialTestRequest(t, server, http.MethodPost, path+"/reconcile", "accountant", "", http.StatusOK))
	require.Equal(t, 1, sends)
	require.Equal(t, 2, lookups)
}

func verifySHEINTrialPublicationDenied(t *testing.T, ctx context.Context, module *dataservicesapp.Module, access *sheinResearchAccess, f sheinResearchPublication) {
	t.Helper()
	a := newSHEINTrialTestApproval(access)
	var original dataacquisition.SHEINUSObservation
	require.NoError(t, json.Unmarshal([]byte(f.ObservationUTF8), &original))
	checks, sends := 0, 0
	deps := sheinTrialDependencies{
		authority: sheinTrialTestAuthority{sheinResearchTrialAuthority{a}, func(_ dataacquisition.Principal, action dataacquisition.SHEINUSTrialAction) bool {
			if action != dataacquisition.SHEINUSTrialPublish {
				return true
			}
			checks++
			return checks == 1
		}},
		send: func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest) (dataacquisition.SHEINUSTrialSubmission, error) {
			sends++
			credits := 5
			return dataacquisition.SHEINUSTrialSubmission{
				SHEINUSTrialUsage: dataacquisition.SHEINUSTrialUsage{ApprovalID: a.ID, Binding: a.Binding(), RequestID: "fixture-no-disclosure", Credits: &credits},
				Cards:             []dataacquisition.SHEINUSTrialCard{{Position: 1, GoodsID: original.NativeGoodsID, Title: *original.Title, URL: original.SourceURL}},
			}, nil
		},
		lookup: func(context.Context, dataacquisition.SHEINUSTrialApproval, dataacquisition.SHEINUSTrialRequest, dataacquisition.SHEINUSTrialAttempt) (dataacquisition.SHEINUSTrialUsage, error) {
			t.Fatal("publish rejection must not request usage again")
			return dataacquisition.SHEINUSTrialUsage{}, dataacquisition.ErrUnavailable
		},
	}
	server := sheinTrialTestHTTP(t, ctx, module, deps, sheinResearchProject{access}, map[string]collection.Scope{"researcher": a.Scope})
	path := sheinTrialTestPath(a.ProjectID, a.ID)
	require.JSONEq(t, `{"code":"FORBIDDEN"}`, sheinTrialTestRequest(t, server, http.MethodPost, path+"/submit", "researcher", "", http.StatusForbidden))
	require.Equal(t, 2, checks)
	require.Equal(t, 1, sends)
	require.JSONEq(t, `{"approvalId":"`+a.ID+`","requestId":"fixture-no-disclosure","credits":5,"reservedCredits":0,"state":"CONFIRMED"}`, sheinTrialTestRequest(t, server, http.MethodGet, path, "researcher", "", http.StatusOK))
}
