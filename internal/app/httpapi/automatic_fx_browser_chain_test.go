package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	fxruntime "task-processor/internal/app/runtime/automaticfx"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	fxsource "task-processor/internal/integration/aicapability/automaticfx"
	"task-processor/internal/workbenchcontext"
)

type fxChainVerifier struct{}
type fxChainStatusWriter struct {
	http.ResponseWriter
	status int
}

func (w *fxChainStatusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *fxChainStatusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (fxChainVerifier) Verify(_ context.Context, token string) (authidentity.AuthenticatedIdentity, error) {
	if token != "synthetic-only" {
		return authidentity.AuthenticatedIdentity{}, errors.New("synthetic token required")
	}
	return authidentity.AuthenticatedIdentity{UserID: "maintainer", HomeOrganizationID: "fixture-org", TokenExpiresAt: time.Now().Add(time.Hour)}, nil
}

type fxChainGrants struct{}

func (fxChainGrants) ListOwnProjectAuthorizations(_ context.Context, _, subject, project string) ([]authidentity.OrganizationGrant, error) {
	if subject != "maintainer" {
		return nil, errors.New("unexpected synthetic identity")
	}
	return []authidentity.OrganizationGrant{{AuthorizationID: "fixture-member", OrganizationID: "fixture-org", OrganizationName: "synthetic", ProjectID: project, Roles: []string{"listingkit_admin"}}}, nil
}

// This is an opt-in isolated integration test, not a deployment runner. It
// exercises the actual browser, Auth.js session, Next BFF, HTTP auth middleware
// and FX Control against a dedicated Temporal. Identity and source are synthetic.
func TestIsolatedBrowserBFFTemporalFXChain(t *testing.T) {
	if os.Getenv("FX_FULL_CHAIN_ISOLATED") != "1" {
		t.Skip("requires explicitly isolated browser and Temporal fixtures")
	}
	require.Equal(t, "127.0.0.1:39212", os.Getenv("FX_FULL_CHAIN_TEMPORAL"))
	inspection, err := exec.Command("docker", "inspect", "codex-fx-chain-temporal-20261112").Output()
	require.NoError(t, err)
	var containers []struct {
		Image  string
		State  struct{ Running bool }
		Config struct {
			Cmd        []string
			Entrypoint []string
		}
		Mounts          []json.RawMessage
		HostConfig      struct{ NetworkMode string }
		NetworkSettings struct {
			Ports map[string][]struct {
				HostIP   string `json:"HostIp"`
				HostPort string
			}
		}
	}
	require.NoError(t, json.Unmarshal(inspection, &containers))
	require.Len(t, containers, 1)
	fixture := containers[0]
	require.True(t, fixture.State.Running)
	require.Equal(t, "sha256:cf86707827fac99e4d1c4a47dc11b105382d796199c7bd41fb3213fb0471628e", fixture.Image)
	require.Equal(t, []string{"temporal"}, fixture.Config.Entrypoint)
	require.Equal(t, []string{"server", "start-dev", "--ip", "0.0.0.0", "--port", "7233", "--headless"}, fixture.Config.Cmd)
	require.Empty(t, fixture.Mounts)
	require.Equal(t, "bridge", fixture.HostConfig.NetworkMode)
	require.Len(t, fixture.NetworkSettings.Ports["7233/tcp"], 1)
	require.Equal(t, "127.0.0.1", fixture.NetworkSettings.Ports["7233/tcp"][0].HostIP)
	require.Equal(t, "39212", fixture.NetworkSettings.Ports["7233/tcp"][0].HostPort)
	chrome, edge := os.Getenv("FX_CHROME_PATH"), os.Getenv("FX_EDGE_PATH")
	require.NotEmpty(t, chrome)
	require.NotEmpty(t, edge)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c, err := client.Dial(client.Options{HostPort: "127.0.0.1:39212", Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	var fetched atomic.Int32
	w := worker.New(c, fxruntime.TaskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(fxruntime.CoordinatorWorkflow, workflow.RegisterOptions{Name: fxruntime.CoordinatorWorkflowName})
	w.RegisterWorkflowWithOptions(fxruntime.TickWorkflow, workflow.RegisterOptions{Name: fxruntime.TickWorkflowName})
	w.RegisterActivityWithOptions(func(context.Context) (fxsource.Observation, error) {
		fetched.Add(1)
		return fxsource.Observation{}, errors.New("synthetic source unavailable")
	}, activity.RegisterOptions{Name: fxruntime.FetchActivity})
	require.NoError(t, w.Start())
	defer w.Stop()
	a, err := authz.NewListingKitAuthorizer([]string{"maintainer"}, nil)
	require.NoError(t, err)
	resolver := workbenchcontext.NewResolver(workbenchcontext.NewGrantResolver(fxChainGrants{}, workbenchcontext.NewGrantCache(nil)), "fixture-project", "fixture-contract", nil)
	log := logrus.New()
	log.SetOutput(io.Discard)
	auth := routeAuthDependencies{workbenchVerifier: fxChainVerifier{}, organizationResolver: resolver, authorizer: a, auditRecorder: workbenchcontext.NewStructuredAuditRecorder(log), auditNow: time.Now}
	control := &fxruntime.Control{Client: c, Namespace: "default"}
	module := modelRoutePricingModule{services: ModelRoutePricingDependencies{AutomaticFX: control}, authorizer: a}
	app := buildCurrentApplicationHTTPServer(module.routes(), auth)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := &fxChainStatusWriter{ResponseWriter: w, status: 200}
		app.Handler.ServeHTTP(out, r)
		t.Logf("synthetic backend %s %s status=%d", r.Method, r.URL.Path, out.status)
	}))
	defer backend.Close()
	webDir, err := filepath.Abs(filepath.Join("..", "..", "..", "web", "listingkit-ui"))
	require.NoError(t, err)
	node, err := exec.LookPath("node")
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:39211")
	require.NoError(t, err, "fixture port already in use")
	require.NoError(t, listener.Close())
	dev := exec.CommandContext(ctx, node, filepath.Join(webDir, "node_modules", "next", "dist", "bin", "next"), "dev", "--hostname", "127.0.0.1", "--port", "39211")
	dev.Dir = webDir
	secret := "synthetic-fx-chain-" + uuid.NewString()
	dev.Env = append(os.Environ(), "AUTH_SECRET="+secret, "ZITADEL_ISSUER_URL=https://fx-chain.invalid", "ZITADEL_CLIENT_ID=fx-chain-fixture", "ZITADEL_CLIENT_SECRET=synthetic-only", "AUTH_URL=http://127.0.0.1:39211", "LISTINGKIT_PUBLIC_BASE_URL=http://127.0.0.1:39211", "LISTINGKIT_SERVICE_API_BASE="+backend.URL+"/api/v1", "AUTH_TRUST_HOST=true", "NEXT_TELEMETRY_DISABLED=1")
	logFile, err := os.CreateTemp("", "fx-chain-next-*.log")
	require.NoError(t, err)
	defer os.Remove(logFile.Name())
	defer logFile.Close()
	dev.Stdout = logFile
	dev.Stderr = logFile
	require.NoError(t, dev.Start())
	devDone := make(chan error, 1)
	go func() { devDone <- dev.Wait() }()
	defer func() {
		_ = dev.Process.Kill()
		select {
		case <-devDone:
		case <-time.After(5 * time.Second):
		}
	}()
	require.Eventually(t, func() bool {
		conn, e := net.DialTimeout("tcp", "127.0.0.1:39211", 200*time.Millisecond)
		if e != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 60*time.Second, 200*time.Millisecond, "isolated Next server did not start")
	select {
	case e := <-devDone:
		t.Fatalf("isolated Next process exited before validation: %v", e)
	default:
	}
	playwright := exec.CommandContext(ctx, node, filepath.Join(webDir, "node_modules", "@playwright", "test", "cli.js"), "test", "--config=playwright.console.config.ts", "automatic-fx-real-chain.browser.ts", "--workers=1")
	playwright.Dir = webDir
	playwright.Env = append(os.Environ(), "AUTH_SECRET="+secret, "FX_CHAIN_SECRET="+secret, "FX_CHAIN_BROWSER=1", "FX_CHROME_PATH="+chrome, "FX_EDGE_PATH="+edge)
	output, err := playwright.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		_ = logFile.Sync()
		raw, _ := os.ReadFile(logFile.Name())
		t.Log(string(raw))
	}
	require.NoError(t, err)
	desc, err := c.DescribeWorkflowExecution(ctx, fxruntime.CoordinatorID, "")
	require.NoError(t, err)
	runID := desc.GetWorkflowExecutionInfo().GetExecution().GetRunId()
	require.NotEmpty(t, runID)
	require.Equal(t, enums.WORKFLOW_EXECUTION_STATUS_RUNNING, desc.GetWorkflowExecutionInfo().GetStatus())
	schedule, err := c.ScheduleClient().GetHandle(ctx, fxruntime.ScheduleID).Describe(ctx)
	require.NoError(t, err)
	require.False(t, schedule.Schedule.State.Paused)
	require.Eventually(t, func() bool { return fetched.Load() > 0 }, 5*time.Second, 100*time.Millisecond, "synthetic source was not invoked")
	require.Eventually(t, func() bool { state, e := control.Status(ctx); return e == nil && state.State == "SOURCE_UNAVAILABLE" }, 5*time.Second, 100*time.Millisecond, "synthetic source failure not reflected in the real coordinator")
	require.Eventually(t, func() bool {
		it := c.GetWorkflowHistory(ctx, fxruntime.CoordinatorID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		count := 0
		for it.HasNext() {
			event, e := it.Next()
			if e != nil {
				return false
			}
			if signal := event.GetWorkflowExecutionSignaledEventAttributes(); signal != nil && signal.SignalName == fxruntime.TickSignal {
				count++
			}
		}
		return count == 1
	}, 5*time.Second, 100*time.Millisecond, "the original coordinator must receive exactly one initial tick")
	fmt.Println("PASS: browsers -> actual Auth.js/BFF -> mounted Go authorization and FX Control -> isolated Temporal; identity and source synthetic")
}
