package sheinloginworker

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"task-processor/internal/sheinlogin"
	"testing"
)

func TestHealthDependencyOutageDoesNotFailProcessLiveness(t *testing.T) {
	available := false
	calls := 0
	h := healthHandler(func(context.Context) bool { calls++; return available })
	for _, path := range []string{"/livez", "/healthz"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		want := http.StatusOK
		if path == "/healthz" {
			want = http.StatusServiceUnavailable
		}
		if w.Code != want {
			t.Fatalf("%s=%d, want %d", path, w.Code, want)
		}
	}
	if calls != 1 {
		t.Fatal("liveness called external dependencies")
	}
	available = true
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatal("recovered dependency did not restore readiness")
	}
}

func TestWorkerReadinessRequiresAllCurrentDependencies(t *testing.T) {
	for _, failed := range []string{"", "redis", "store", "authorization"} {
		t.Run(failed, func(t *testing.T) {
			redis := func(context.Context) bool { return failed != "redis" }
			store := func(context.Context) error {
				if failed == "store" {
					return errors.New("synthetic Store outage")
				}
				return nil
			}
			auth := func(context.Context) bool { return failed != "authorization" }
			if workerDependenciesReady(context.Background(), redis, store, auth) != (failed == "") {
				t.Fatal("unavailable dependency reported ready")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if workerDependenciesReady(ctx, redis, store, auth) {
				t.Fatal("expired health deadline reported ready")
			}
		})
	}
	if workerDependenciesReady(context.Background(), nil, nil, nil) {
		t.Fatal("unconfigured dependencies reported ready")
	}
}

type producerFixture struct {
	received sheinlogin.Command
	calls    int
}

func (f *producerFixture) Execute(_ context.Context, c sheinlogin.Command) (sheinlogin.Result, error) {
	f.received = c
	f.calls++
	return sheinlogin.Result{OrganizationID: c.OrganizationID, StoreID: c.StoreID, AttemptID: c.AttemptID, Status: "queued", SyncStatus: "not_requested"}, nil
}

type producerAuth struct{ denied bool }

func (f producerAuth) Authenticate(ctx context.Context, token, org string) (context.Context, string, string, error) {
	if f.denied || token != "synthetic-current-jwt" || org != "org-a" {
		return nil, "", "", sheinlogin.ErrInvalidInput
	}
	return ctx, "verified-actor", "verified-member", nil
}
func TestAuthorizedCLIProducerDispatchAndPrivateVerificationInput(t *testing.T) {
	token := filepath.Join(t.TempDir(), "token")
	if os.WriteFile(token, []byte("synthetic-current-jwt"), 0600) != nil {
		t.Fatal("fixture")
	}
	if goruntime.GOOS == "windows" {
		identity, e := user.Current()
		if e != nil {
			t.Fatal(e)
		}
		if e = exec.Command("icacls", token, "/inheritance:r", "/grant:r", "*"+identity.Uid+":F").Run(); e != nil {
			t.Fatal(e)
		}
	}
	for _, action := range []string{"login", "status", "verify", "cancel", "activate", "quiesce"} {
		t.Run(action, func(t *testing.T) {
			var output bytes.Buffer
			service := &producerFixture{}
			o := Options{Action: action, OrganizationID: "org-a", StoreID: "00000000-0000-4000-8000-000000000001", TokenFile: token, Input: strings.NewReader("123456\n"), Output: &output}
			if action == "login" || action == "activate" || action == "quiesce" {
				o.RequestKey = "00000000-0000-4000-8000-000000000002"
			} else {
				o.AttemptID = "00000000-0000-4000-8000-000000000002"
			}
			if e := execute(context.Background(), service, producerAuth{}, o); e != nil {
				t.Fatal(e)
			}
			if service.calls != 1 || service.received.ActorID != "verified-actor" || service.received.MemberID != "verified-member" {
				t.Fatal("flags replaced current authority")
			}
			if action == "verify" && service.received.Code != "123456" {
				t.Fatal("stdin verification not delivered")
			}
			for _, secret := range []string{"123456", "synthetic-current-jwt", "verified-member"} {
				if strings.Contains(output.String(), secret) {
					t.Fatal("private input leaked")
				}
			}
			denied := &producerFixture{}
			if e := execute(context.Background(), denied, producerAuth{true}, o); e == nil || denied.calls != 0 {
				t.Fatal("authorization bypass")
			}
		})
	}
}
