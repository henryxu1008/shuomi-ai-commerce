package setupcontrol

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type processRecoveryDriver struct {
	recoveryFixture
	root string
	hold bool
}

func (d *processRecoveryDriver) Apply(_ context.Context, _ Prepared, report Report) error {
	if e := report("start"); e != nil {
		return e
	}
	f, e := os.OpenFile(filepath.Join(d.root, "dispatches"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if e != nil {
		return e
	}
	if _, e = f.WriteString("dispatch\n"); e != nil {
		_ = f.Close()
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if d.hold {
		if e = os.WriteFile(filepath.Join(d.root, "ready"), []byte("ready"), 0600); e != nil {
			return e
		}
		for {
			time.Sleep(time.Second)
		}
	}
	d.revision = "next"
	return nil
}
func TestRecoveryAcrossProcessesDoesNotRedispatchAfterHardExit(t *testing.T) {
	d := &recoveryFixture{audit: RecoveryAudit{Fingerprint: "empty", Empty: true}}
	c, p, _ := retainedUnknown(t, d)
	preview, e := c.PrepareRecovery(context.Background(), p.ID, "retry", "")
	require.NoError(t, e)
	child := func(mode string) *exec.Cmd {
		cmd := exec.Command(os.Args[0], "-test.run=^TestRecoveryProcessHelper$")
		cmd.Env = append(os.Environ(), "SETUP_RECOVERY_ROOT="+c.root, "SETUP_RECOVERY_PLAN="+p.ID, "SETUP_RECOVERY_PREVIEW="+preview.ID, "SETUP_RECOVERY_MODE="+mode)
		return cmd
	}
	running := child("hold")
	require.NoError(t, running.Start())
	t.Cleanup(func() { _ = running.Process.Kill() })
	require.Eventually(t, func() bool { _, e := os.Stat(filepath.Join(c.root, "ready")); return e == nil }, 10*time.Second, 10*time.Millisecond)
	b, e := child("busy").CombinedOutput()
	require.NoError(t, e, string(b))
	require.NoError(t, running.Process.Kill())
	_ = running.Wait()
	b, e = child("replay").CombinedOutput()
	require.NoError(t, e, string(b))
	log, e := os.ReadFile(filepath.Join(c.root, "dispatches"))
	require.NoError(t, e)
	require.Equal(t, 1, strings.Count(string(log), "dispatch"))
	_, e = c.Reconcile(context.Background(), p.ID)
	require.ErrorIs(t, e, ErrUnresolved)
	got, e := c.Read(p.ID)
	require.NoError(t, e)
	require.Equal(t, "unknown", got.Status)
	require.Len(t, got.Attempts, 2)
	_, e = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.NoError(t, e)
	log, e = os.ReadFile(filepath.Join(c.root, "dispatches"))
	require.NoError(t, e)
	require.Equal(t, 1, strings.Count(string(log), "dispatch"))
}
func TestRecoveryProcessHelper(t *testing.T) {
	root := os.Getenv("SETUP_RECOVERY_ROOT")
	if root == "" {
		t.Skip("isolated process helper")
	}
	d := &processRecoveryDriver{recoveryFixture: recoveryFixture{audit: RecoveryAudit{Fingerprint: "empty", Empty: true}}, root: root, hold: os.Getenv("SETUP_RECOVERY_MODE") == "hold"}
	c := New(root, d)
	p, e := c.ConfirmRecovery(context.Background(), os.Getenv("SETUP_RECOVERY_PLAN"), os.Getenv("SETUP_RECOVERY_PREVIEW"))
	if os.Getenv("SETUP_RECOVERY_MODE") == "busy" {
		require.ErrorIs(t, e, ErrBusy)
	} else {
		require.NoError(t, e)
		require.Equal(t, "running", p.Status)
	}
}
