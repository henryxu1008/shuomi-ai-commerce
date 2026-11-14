package setupcontrol

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// An actual second OS process must use the original journal and OS lock. A
// same-handler mutex test cannot establish this crash/restart property.
func TestControlAcrossProcessLockAndRestart(t *testing.T) {
	root := t.TempDir()
	c := New(root, &testDriver{revision: "before"})
	p, e := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, e)
	child := func(action string) {
		cmd := exec.Command(os.Args[0], "-test.run=^TestControlProcessHelper$")
		cmd.Env = append(os.Environ(), "SETUP_TEST_ROOT="+root, "SETUP_TEST_PLAN="+p.ID, "SETUP_TEST_ACTION="+action)
		b, e := cmd.CombinedOutput()
		require.NoError(t, e, string(b))
	}
	held, e := lockInstance(filepath.Join(root, "stable-instance"))
	require.NoError(t, e)
	child("busy")
	require.NoError(t, held.Close())
	global, e := lockInstance(filepath.Join(root, "controller"))
	require.NoError(t, e)
	child("busy")
	require.NoError(t, global.Close())
	child("unknown")
	d := &testDriver{revision: "before"}
	restarted := New(root, d)
	_, e = restarted.Plan(context.Background(), Request{Mode: "existing"})
	require.ErrorIs(t, e, ErrUnresolved)
	_, e = restarted.Apply(context.Background(), p.ID)
	require.ErrorIs(t, e, ErrUnresolved)
	require.Zero(t, d.calls)
	d.revision = "next"
	_, e = restarted.Reconcile(context.Background(), p.ID)
	require.NoError(t, e)
	require.Zero(t, d.calls)
}
func TestControlProcessHelper(t *testing.T) {
	root := os.Getenv("SETUP_TEST_ROOT")
	if root == "" {
		t.Skip("subprocess helper")
	}
	d := &testDriver{revision: "before", fail: true}
	c := New(root, d)
	_, e := c.Apply(context.Background(), os.Getenv("SETUP_TEST_PLAN"))
	if os.Getenv("SETUP_TEST_ACTION") == "busy" {
		require.ErrorIs(t, e, ErrBusy)
		require.Zero(t, d.calls)
	} else {
		require.ErrorIs(t, e, ErrUnresolved)
		require.Equal(t, 1, d.calls)
	}
}
