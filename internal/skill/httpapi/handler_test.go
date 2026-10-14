package skillhttp

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"task-processor/internal/authidentity"
	"task-processor/internal/httproute"
	d "task-processor/internal/skill"
	"testing"
)

type captureRepo struct {
	d.Repository
	command d.Command
	calls   int
}

func (r *captureRepo) Execute(_ context.Context, c d.Command, _ d.ResolvePrincipal) (d.Receipt, error) {
	r.command = c
	r.calls++
	return d.Receipt{ID: uuid.NewString(), Version: 1, Revision: 1, Operation: c.Operation}, nil
}
func TestPersonalCreationUsesVerifiedIdentityWithoutOrganization(t *testing.T) {
	repo := &captureRepo{}
	service, e := d.NewService(repo, func(ctx context.Context) (d.Principal, error) {
		id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok {
			return d.Principal{}, d.ErrForbidden
		}
		return d.Principal{UserID: id.UserID}, nil
	})
	require.NoError(t, e)
	h := &Handler{Service: service}
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	for _, route := range Routes(h) {
		engine.Handle(route.Method, route.Path, route.Handler)
	}
	body := `{"files":[{"path":"SKILL.md","text":"---\nname: product-title-normalization\ndescription: Normalize saved facts.\n---\nNo invented facts.\n"}],"allowCopy":true,"allowExport":true}`
	request := httptest.NewRequest(http.MethodPost, Base+"/personal", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	request.Header.Set("X-User-ID", "impostor")
	request = request.WithContext(authidentity.WithAuthenticatedIdentity(request.Context(), authidentity.AuthenticatedIdentity{UserID: "alice"}))
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, 200, response.Code)
	require.Equal(t, d.Owner{Kind: d.UserOwner, ID: "alice"}, repo.command.Owner)
	require.Equal(t, "alice", repo.command.Principal.UserID)
	bad := strings.TrimSuffix(body, "}") + `,"owner":"PLATFORM"}`
	request = httptest.NewRequest(http.MethodPost, Base+"/personal", strings.NewReader(bad))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	request = request.WithContext(authidentity.WithAuthenticatedIdentity(request.Context(), authidentity.AuthenticatedIdentity{UserID: "alice"}))
	response = httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	require.Equal(t, 400, response.Code)
	require.Equal(t, 1, repo.calls)
}
func TestSkillRoutesKeepTheirIdentityAndOrganizationBoundaries(t *testing.T) {
	for _, route := range Routes(nil) {
		require.NoError(t, ValidateDescriptor(route))
		changed := route
		changed.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		require.Error(t, ValidateDescriptor(changed))
	}
}
