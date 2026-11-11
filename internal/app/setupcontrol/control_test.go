package setupcontrol

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"sync"
	"testing"
)

type testDriver struct {
	mu       sync.Mutex
	calls    int
	revision string
	fail     bool
	stage    string
	failure  error
	release  chan struct{}
}

func (d *testDriver) Identity(context.Context, Request) (string, error) {
	return "stable-instance", nil
}
func (d *testDriver) Prepare(_ context.Context, r Request, _ string) (Prepared, error) {
	return Prepared{Request: r, InstanceID: "stable-instance", Before: d.revision, Candidate: "next", Services: []string{"listingkit-ui"}}, nil
}
func (d *testDriver) Fingerprint(context.Context, Prepared) (string, error) { return d.revision, nil }
func (d *testDriver) Apply(_ context.Context, _ Prepared, report Report) error {
	stage := d.stage
	if stage == "" {
		stage = "start"
	}
	if err := report(stage); err != nil {
		return err
	}
	d.mu.Lock()
	d.calls++
	d.mu.Unlock()
	if d.release != nil {
		<-d.release
	}
	if d.fail {
		if d.failure != nil {
			return d.failure
		}
		return errors.New("unknown")
	}
	d.revision = "next"
	return nil
}
func (d *testDriver) Verify(_ context.Context, _ Prepared, report Report) error {
	if err := report("verify"); err != nil {
		return err
	}
	if d.revision != "next" {
		return errors.New("not verified")
	}
	return nil
}

func TestApplyBindsOriginalPlanAndDriftBeforeDispatch(t *testing.T) {
	d := &testDriver{revision: "before"}
	c := New(t.TempDir(), d)
	p, err := c.Plan(context.Background(), Request{Mode: "existing", InstanceID: "stable-instance", Enabled: map[string]bool{"knowledge": true}})
	require.NoError(t, err)
	d.revision = "foreign"
	_, err = c.Apply(context.Background(), p.ID)
	require.ErrorIs(t, err, ErrDrift)
	require.Zero(t, d.calls)
}
func TestUnknownSurvivesRestartAndBlocksNewKey(t *testing.T) {
	root := t.TempDir()
	d := &testDriver{revision: "before", fail: true}
	c := New(root, d)
	p, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	got, err := c.Apply(context.Background(), p.ID)
	require.Error(t, err)
	require.Equal(t, "unknown", got.Status)
	next := New(root, d)
	_, err = next.Plan(context.Background(), Request{Mode: "existing"})
	require.ErrorIs(t, err, ErrUnresolved)
	_, err = next.Apply(context.Background(), p.ID)
	require.ErrorIs(t, err, ErrUnresolved)
	require.Equal(t, 1, d.calls)
	_, err = next.Reconcile(context.Background(), p.ID)
	require.Error(t, err)
	require.Equal(t, 1, d.calls)
	d.revision = "next"
	got, err = next.Reconcile(context.Background(), p.ID)
	require.NoError(t, err)
	require.Equal(t, "succeeded", got.Status)
}
func TestOriginalKeyReplayDoesNotDispatchAgain(t *testing.T) {
	root := t.TempDir()
	d := &testDriver{revision: "before"}
	c := New(root, d)
	p, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	_, err = c.Apply(context.Background(), p.ID)
	require.NoError(t, err)
	_, err = New(root, d).Apply(context.Background(), p.ID)
	require.NoError(t, err)
	require.Equal(t, 1, d.calls)
}
func TestCrossControllerLockAndPrivatePlanCannotBeReplaced(t *testing.T) {
	root := t.TempDir()
	d := &testDriver{revision: "before", release: make(chan struct{})}
	c := New(root, d)
	p, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	lock, err := lockInstance(filepath.Join(root, "stable-instance"))
	require.NoError(t, err)
	_, err = New(root, d).Apply(context.Background(), p.ID)
	require.ErrorIs(t, err, ErrBusy)
	require.Zero(t, d.calls)
	lock.Close()
}
