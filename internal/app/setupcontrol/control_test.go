package setupcontrol

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
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

type prepareFailDriver struct {
	testDriver
	wrongIdentity bool
}

func (d *prepareFailDriver) Prepare(_ context.Context, _ Request, dir string) (Prepared, error) {
	if err := os.WriteFile(filepath.Join(dir, "candidate"), []byte("temporary"), 0600); err != nil {
		return Prepared{}, err
	}
	if d.wrongIdentity {
		return Prepared{InstanceID: "another-instance"}, nil
	}
	return Prepared{}, ErrInvalid
}

func TestFailedPreviewDoesNotBreakHistory(t *testing.T) {
	for _, wrong := range []bool{false, true} {
		root := t.TempDir()
		c := New(root, &testDriver{revision: "before"})
		prior, err := c.Plan(context.Background(), Request{Mode: "existing"})
		require.NoError(t, err)
		payloadRoot := filepath.Join(root, "stable-instance", "payloads")
		priorDir := filepath.Join(payloadRoot, strings.Split(prior.ID, ".")[1])
		require.NoError(t, os.WriteFile(filepath.Join(priorDir, "keep"), []byte("preserved"), 0600))
		c.driver = &prepareFailDriver{wrongIdentity: wrong}
		_, err = c.Plan(context.Background(), Request{Mode: "existing"})
		if wrong {
			require.ErrorIs(t, err, ErrDrift)
		} else {
			require.ErrorIs(t, err, ErrInvalid)
		}
		entries, err := os.ReadDir(payloadRoot)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		contents, err := os.ReadFile(filepath.Join(priorDir, "keep"))
		require.NoError(t, err)
		require.Equal(t, "preserved", string(contents))
		_, err = c.History()
		require.NoError(t, err)
	}
}
func TestPublishedPlanWithMissingReceiptStillFailsClosed(t *testing.T) {
	c := New(t.TempDir(), &testDriver{revision: "before"})
	p, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	dir, err := c.directory(p.ID)
	require.NoError(t, err)
	require.NoError(t, os.Remove(filepath.Join(dir, "receipt.json")))
	_, err = c.History()
	require.ErrorIs(t, err, ErrUnavailable)
}
func TestPublicationFailurePreservesCandidate(t *testing.T) {
	root := t.TempDir()
	instance := filepath.Join(root, "stable-instance")
	require.NoError(t, os.MkdirAll(instance, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(instance, "pending-plans"), []byte("blocked"), 0600))
	c := New(root, &testDriver{revision: "before"})
	_, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.Error(t, err)
	entries, err := os.ReadDir(filepath.Join(instance, "payloads"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	_, err = c.History()
	require.NoError(t, err)
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
func TestExistingCheckFailureAllowsNewExplicitPlanButPreservesOriginal(t *testing.T) {
	d := &testDriver{revision: "before", fail: true, stage: "check"}
	c := New(t.TempDir(), d)
	original, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	failed, err := c.Apply(context.Background(), original.ID)
	require.Error(t, err)
	require.Equal(t, "failed", failed.Status)
	newPlan, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	require.NotEqual(t, original.ID, newPlan.ID)
	retained, err := c.Read(original.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", retained.Status)
	require.Equal(t, 1, d.calls, "preparing a new plan must not dispatch it")
}
func TestExistingFailureAfterStartRemainsBlocked(t *testing.T) {
	d := &testDriver{revision: "before", fail: true, stage: "start"}
	c := New(t.TempDir(), d)
	p, err := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, err)
	_, err = c.Apply(context.Background(), p.ID)
	require.Error(t, err)
	_, err = c.Plan(context.Background(), Request{Mode: "existing"})
	require.ErrorIs(t, err, ErrUnresolved)
	require.Equal(t, 1, d.calls)
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
