package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/commercetool"
)

type modeRepoFixture struct {
	agentconfig.ModeRepository
	reads, writes int
	command       agentconfig.ModeCommand
}

func (r *modeRepoFixture) ReadExecutionMode(_ context.Context, identity agent.ExecutionIdentity, subject agentconfig.ResolvedModeSubject) (agentconfig.ExecutionModeRecord, error) {
	r.reads++
	return agentconfig.ExecutionModeRecord{Subject: subject.Subject, Mode: agent.ModeDefault, Revision: "0"}, nil
}

func (r *modeRepoFixture) ExecuteMode(ctx context.Context, c agentconfig.ModeCommand, guards ...func(context.Context) error) (agentconfig.ModeReceipt, error) {
	r.writes++
	r.command = c
	for _, guard := range guards {
		if err := guard(ctx); err != nil {
			return agentconfig.ModeReceipt{}, err
		}
	}
	return agentconfig.ModeReceipt{CommandID: uuid.NewString(), RequestKey: c.Key, Operation: "set-execution-mode", Subject: c.Subject.Subject, Mode: c.Mode, BeforeRevision: "0", Revision: "1"}, nil
}

func modeHarness(t *testing.T, repo *modeRepoFixture, configure bool, subjectErr error) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	h := &Handler{Repository: &repoFixture{}, Catalog: catalogFixture{}, ModeRepository: repo}
	h.Authorize = func(_ context.Context, permissions ...string) (agent.Scope, error) {
		return agent.Scope{OrganizationID: "B", ActorID: "actor"}, nil
	}
	h.ResolveModeSubject = func(_ context.Context, subject agentconfig.ModeSubject, write bool) (agent.ExecutionIdentity, agentconfig.ResolvedModeSubject, error) {
		if subjectErr != nil {
			return agent.ExecutionIdentity{}, agentconfig.ResolvedModeSubject{}, subjectErr
		}
		if write && !configure {
			return agent.ExecutionIdentity{}, agentconfig.ResolvedModeSubject{}, agentconfig.ErrForbidden
		}
		definition := commercetool.AgentRef{ID: subject.ID, Version: "v1.0.0"}
		if subject.Kind == agentconfig.PrivateDeliverySubject {
			definition = commercetool.AgentRef{ID: "product.quality.check", Version: "2.0.0"}
		}
		return agent.ExecutionIdentity{OrganizationID: "B", ActorID: "actor", MemberID: "member"}, agentconfig.ResolvedModeSubject{Subject: subject, Definition: definition}, nil
	}
	r := gin.New()
	for _, route := range Routes(h) {
		r.Handle(route.Method, route.Path, route.Handler)
	}
	return r
}

func TestExecutionModeAPIUsesPrivateOwnerAndVerifiedMember(t *testing.T) {
	repo := &modeRepoFixture{}
	r := modeHarness(t, repo, true, nil)
	path := "/execution-modes/private_delivery/" + uuid.NewString()
	get := call(r, http.MethodGet, path, "", nil)
	require.Equal(t, 200, get.Code)
	require.Contains(t, get.Body.String(), `"mode":"default"`)
	require.Contains(t, get.Body.String(), `"version":"2.0.0"`)
	put := call(r, http.MethodPut, path, `{"mode":"full_access","expectedRevision":"0"}`, map[string]string{"Idempotency-Key": uuid.NewString()})
	require.Equal(t, 200, put.Code)
	require.Equal(t, agent.ModeFullAccess, repo.command.Mode)
	require.Equal(t, agentconfig.PrivateDeliverySubject, repo.command.Subject.Subject.Kind)
	require.Equal(t, "member", repo.command.Identity.MemberID)
	require.Equal(t, "2.0.0", repo.command.Subject.Definition.Version)
	denied := modeHarness(t, repo, false, nil)
	require.Equal(t, 403, call(denied, http.MethodPut, path, `{"mode":"read_only","expectedRevision":"1"}`, map[string]string{"Idempotency-Key": uuid.NewString()}).Code)
	require.Equal(t, 1, repo.writes)
	foreign := modeHarness(t, repo, true, agentconfig.ErrNotFound)
	require.Equal(t, 404, call(foreign, http.MethodGet, path, "", nil).Code)
	require.Equal(t, 1, repo.reads)
}

func TestExecutionModeAPIRejectsAuthorityFieldsAndUnboundCommands(t *testing.T) {
	repo := &modeRepoFixture{}
	r := modeHarness(t, repo, true, nil)
	path := "/execution-modes/registered_agent/product.title.agent"
	for _, body := range []string{
		`{"mode":"admin","expectedRevision":"0"}`,
		`{"mode":"full_access","expectedRevision":"00"}`,
		`{"mode":"full_access","expectedRevision":0}`,
		`{"mode":"full_access","expectedRevision":"0","organizationId":"foreign"}`,
		`{"mode":"full_access","expectedRevision":"0","memberId":"foreign"}`,
		`{"mode":"full_access","expectedRevision":"0","permissions":["*"]}`,
		`{"mode":"full_access","mode":"read_only","expectedRevision":"0"}`,
	} {
		require.Equal(t, 400, call(r, http.MethodPut, path, body, map[string]string{"Idempotency-Key": uuid.NewString()}).Code, body)
	}
	require.Equal(t, 428, call(r, http.MethodPut, path, `{"mode":"full_access"}`, map[string]string{"Idempotency-Key": uuid.NewString()}).Code)
	require.Equal(t, 400, call(r, http.MethodPut, path, `{"mode":"full_access","expectedRevision":"0"}`, nil).Code)
	require.Equal(t, 400, call(r, http.MethodGet, path+"?organizationId=foreign", "", nil).Code)
	require.Zero(t, repo.writes)
}
