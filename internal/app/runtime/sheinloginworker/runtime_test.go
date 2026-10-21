package sheinloginworker

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"task-processor/internal/sheinlogin"
	"testing"
)

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
