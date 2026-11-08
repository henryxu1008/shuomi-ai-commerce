package httpapi

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/httproute"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
	"testing"
)

func TestResearchRoutesKeepPrivateScopeAndRejectDescriptorWidening(t *testing.T) {
	gin.SetMode(gin.TestMode)
	routes := Routes(nil)
	require.Len(t, routes, 3)
	for _, d := range routes {
		require.Equal(t, httproute.AuthPolicyCurrentIdentity, d.AuthPolicy)
		require.Equal(t, httproute.OrganizationAccessPolicyLiveWrite, d.OrganizationAccessPolicy)
		require.NoError(t, ValidateDescriptor(d))
		d.AuthPolicy = httproute.AuthPolicyPublic
		require.Error(t, ValidateDescriptor(d))
	}
	router := gin.New()
	for _, d := range routes {
		router.Handle(d.Method, d.Path, d.Handler)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, Base+"/c7d67882-b306-44ed-9cbd-85844ebfb362/commands", strings.NewReader(`{"action":"link"}`))
	router.ServeHTTP(w, req)
	require.Equal(t, 503, w.Code)
}

type scopeProjects struct{ scope collection.Scope }

func (p scopeProjects) Authorize(_ context.Context, s collection.Scope, _ string, _ bool) error {
	if s != p.scope {
		return research.ErrForbidden
	}
	return nil
}

type scopeRepository struct {
	research.Repository
	scope collection.Scope
	calls int
}

func (p scopeProjects) AuthorizeAnalysis(ctx context.Context, s collection.Scope, id string, active bool) error {
	return p.Authorize(ctx, s, id, active)
}

func (r *scopeRepository) List(_ context.Context, s collection.Scope, _, _ string) ([]research.Candidate, string, error) {
	r.calls++
	if s != r.scope {
		return nil, "", research.ErrForbidden
	}
	return []research.Candidate{}, "", nil
}

type noSources struct{}

func (noSources) Observe(context.Context, collection.Scope, string, research.Selection) (research.Observation, error) {
	return research.Observation{}, research.ErrNotFound
}
func TestResearchHTTPRejectsForgedAndDuplicateInputBeforeDomainAccess(t *testing.T) {
	scope := collection.Scope{OrganizationID: "verified-org", ActorID: "verified-actor", MemberID: "verified-member"}
	repo := &scopeRepository{scope: scope}
	h := &Handler{Service: research.New(repo, scopeProjects{scope}, noSources{}), Bind: func(ctx context.Context, _ string) (context.Context, collection.Scope, error) { return ctx, scope, nil }}
	router := gin.New()
	for _, d := range Routes(h) {
		router.Handle(d.Method, d.Path, d.Handler)
	}
	id := "11111111-1111-4111-8111-111111111111"
	for _, body := range []string{`{"action":"link","actorId":"forged","selections":[]}`, `{"action":"link","action":"reset","selections":[]}`, `{"action":"link","selections":[]}`, strings.Repeat("x", (64<<10)+1)} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", Base+"/"+id+"/commands", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", id)
		router.ServeHTTP(w, req)
		require.Equal(t, 400, w.Code)
	}
	require.Zero(t, repo.calls)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", Base+"/"+id, nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, 1, repo.calls)
	require.Equal(t, "private, no-store", w.Header().Get("Cache-Control"))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest("GET", Base+"/"+id+"?actorId=forged", nil))
	require.Equal(t, 400, w.Code)
	require.Equal(t, 1, repo.calls)
}

func TestAssessmentRoutesAreBoundedAndStrict(t *testing.T) {
	for _, d := range researchRoutes(nil, false, true) {
		require.NoError(t, ValidateDescriptor(d))
		if strings.Contains(d.Path, "assessment-preview") {
			require.Equal(t, "workbench.project.manage", d.Permission)
		}
	}
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	h := &Handler{Service: research.New(&scopeRepository{scope: scope}, scopeProjects{scope}, noSources{}), Bind: func(ctx context.Context, _ string) (context.Context, collection.Scope, error) { return ctx, scope, nil }}
	router := gin.New()
	for _, d := range researchRoutes(h, false, true) {
		router.Handle(d.Method, d.Path, d.Handler)
	}
	id := "11111111-1111-4111-8111-111111111111"
	for _, body := range []string{`{"action":"save_assessment","profit":"100"}`, `{"action":"save_assessment","action":"decide"}`, `{"organizationId":"forged"}`} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", Base+"/"+id+"/candidates/"+id+"/assessment-preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		require.Equal(t, 400, w.Code)
	}
	for _, suffix := range []string{"/assessments?before=01", "/assessments?actorId=x", "/assessments/" + id + "?latest=true"} {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", Base+"/"+id+"/candidates/"+id+suffix, nil))
		require.Equal(t, 400, w.Code)
	}
}
