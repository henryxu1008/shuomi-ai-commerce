package setupcontrol

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/platform/privatefile"
)

func TestFailureRecordsStageCategoryWithoutSavingRawSecretAndCheckDoesNotEraseIt(t *testing.T) {
	for _, tc := range []struct {
		stage, status string
		failure       error
		reason        string
	}{
		{"download", "failed", &CommandError{Code: "timeout"}, "等待超时"},
		{"build", "failed", &CommandError{Code: "command_failed", ExitCode: 42}, "命令未完成"},
		{"start", "unknown", errors.New("sk-fixture-private-password"), "尚未定位"},
	} {
		t.Run(tc.stage, func(t *testing.T) {
			d := &testDriver{revision: "before", fail: true, stage: tc.stage, failure: tc.failure}
			c := New(t.TempDir(), d)
			p, err := c.Plan(context.Background(), Request{Mode: "existing"})
			require.NoError(t, err)
			got, err := c.Apply(context.Background(), p.ID)
			require.Error(t, err)
			require.Equal(t, tc.status, got.Status)
			require.Equal(t, tc.stage, got.Stage)
			require.Contains(t, got.Reason, tc.reason)
			encoded, _ := json.Marshal(got)
			require.NotContains(t, string(encoded), "sk-fixture-private-password")
			dir, _ := c.directory(p.ID)
			raw, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
			require.NoError(t, err)
			require.NotContains(t, string(raw), "sk-fixture-private-password")
			_, _ = c.Reconcile(context.Background(), p.ID)
			after, err := c.Read(p.ID)
			require.NoError(t, err)
			require.Equal(t, got.Stage, after.Stage)
			require.Equal(t, got.Reason, after.Reason)
		})
	}
}

type recoveryFixture struct {
	testDriver
	audit RecoveryAudit
}

func (d *recoveryFixture) AuditRecovery(context.Context, Prepared, bool) (RecoveryAudit, error) {
	return d.audit, nil
}
func (d *recoveryFixture) PrepareRetry(_ context.Context, p Prepared, subnet, _ string) (Prepared, error) {
	p.Request.NetworkSubnet = subnet
	p.Before = d.audit.Fingerprint
	p.Candidate = "retry-candidate"
	return p, nil
}

func retainedUnknown(t *testing.T, d *recoveryFixture) (*Control, Plan, []byte) {
	t.Helper()
	c := New(t.TempDir(), d)
	id := "stable-instance." + uuid.NewString()
	p := Plan{ID: id, InstanceID: "stable-instance", Mode: "fresh", Status: "unknown"}
	r := record{Version: 1, Plan: p, Prepared: Prepared{InstanceID: p.InstanceID, Request: Request{Mode: "fresh", Project: "test-install", Profile: "base"}, Candidate: "original-candidate"}}
	dir, err := c.directory(id)
	require.NoError(t, err)
	b, err := json.Marshal(r)
	require.NoError(t, err)
	require.NoError(t, privatefile.Write(filepath.Join(dir, "receipt.json"), b))
	key, _ := json.Marshal(id)
	require.NoError(t, privatefile.Write(filepath.Join(c.root, p.InstanceID, "active.json"), key))
	return c, p, b
}

func TestArchiveKeepsOriginalBytesAndUnknownResultAndGuard(t *testing.T) {
	d := &recoveryFixture{audit: RecoveryAudit{Fingerprint: "removed", Removed: true}}
	c, p, original := retainedUnknown(t, d)
	preview, err := c.PrepareRecovery(context.Background(), p.ID, "archive", "")
	require.NoError(t, err)
	dir, _ := c.directory(p.ID)
	b, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
	require.NoError(t, err)
	require.Equal(t, original, b, "preview must not adopt or alter the original operation")
	got, err := c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.NoError(t, err)
	require.True(t, got.Archived)
	require.Equal(t, "unknown", got.Status)
	saved, err := os.ReadFile(filepath.Join(dir, "original-receipt.json"))
	require.NoError(t, err)
	require.Equal(t, original, saved)
	r, err := c.read(p.ID)
	require.NoError(t, err)
	require.Equal(t, 2, r.Version, "old tools must refuse the new record format")
	guard, err := c.active(p.InstanceID)
	require.NoError(t, err)
	require.Equal(t, p.ID, guard)
	pending, err := c.Outstanding()
	require.NoError(t, err)
	require.Empty(t, pending)
	history, err := c.History()
	require.NoError(t, err)
	require.Len(t, history, 1)
	_, err = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.NoError(t, err)
	require.Zero(t, d.calls)
}

func TestFailedV1InspectionDoesNotAdoptOrAlterOriginalRecord(t *testing.T) {
	d := &recoveryFixture{}
	c, p, original := retainedUnknown(t, d)
	dir, err := c.directory(p.ID)
	require.NoError(t, err)
	guard := filepath.Join(c.root, p.InstanceID, "active.json")
	before, err := os.ReadFile(guard)
	require.NoError(t, err)
	_, err = c.Reconcile(context.Background(), p.ID)
	require.ErrorIs(t, err, ErrUnresolved)
	after, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
	require.NoError(t, err)
	require.Equal(t, original, after)
	after, err = os.ReadFile(guard)
	require.NoError(t, err)
	require.Equal(t, before, after)
	_, err = os.Stat(filepath.Join(dir, "original-receipt.json"))
	require.True(t, os.IsNotExist(err), "an unsuccessful inspection must not adopt v1")
	d.revision = "next"
	got, err := c.Reconcile(context.Background(), p.ID)
	require.NoError(t, err)
	require.Equal(t, "succeeded", got.Status)
	after, err = os.ReadFile(filepath.Join(dir, "original-receipt.json"))
	require.NoError(t, err)
	require.Equal(t, original, after)
}

func TestRetryStaysOnOriginalOperationAndRepeatedConfirmationDispatchesOnce(t *testing.T) {
	d := &recoveryFixture{testDriver: testDriver{revision: "empty"}, audit: RecoveryAudit{Fingerprint: "empty", Empty: true}}
	c, p, original := retainedUnknown(t, d)
	preview, err := c.PrepareRecovery(context.Background(), p.ID, "retry", "10.251.128.0/24")
	require.NoError(t, err)
	got, err := c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.NoError(t, err)
	require.Equal(t, p.ID, got.ID)
	require.Equal(t, "succeeded", got.Status)
	require.Len(t, got.Attempts, 2)
	require.Equal(t, "unknown", got.Attempts[0].Status)
	require.Equal(t, "succeeded", got.Attempts[1].Status)
	_, err = New(c.root, d).ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.NoError(t, err)
	require.Equal(t, 1, d.calls)
	r, err := c.read(p.ID)
	require.NoError(t, err)
	require.Equal(t, "", r.Prepared.Request.NetworkSubnet, "original input must not be rewritten")
	dir, _ := c.directory(p.ID)
	saved, err := os.ReadFile(filepath.Join(dir, "original-receipt.json"))
	require.NoError(t, err)
	require.Equal(t, original, saved)
}

func TestRecoveryRejectsNonemptyDriftAndAnotherActiveOperation(t *testing.T) {
	for _, action := range []string{"archive", "retry"} {
		t.Run(action, func(t *testing.T) {
			d := &recoveryFixture{audit: RecoveryAudit{Fingerprint: "before"}}
			c, p, original := retainedUnknown(t, d)
			preview, err := c.PrepareRecovery(context.Background(), p.ID, action, "")
			require.NoError(t, err)
			require.NotEmpty(t, preview.Blocks)
			_, err = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
			require.ErrorIs(t, err, ErrInvalid)
			d.audit.Empty, d.audit.Removed = true, true
			if action == "retry" {
				d.audit.Removed = false
			}
			preview, err = c.PrepareRecovery(context.Background(), p.ID, action, "")
			require.NoError(t, err)
			d.audit.Fingerprint = "foreign"
			_, err = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
			require.ErrorIs(t, err, ErrDrift)
			dir, _ := c.directory(p.ID)
			b, err := os.ReadFile(filepath.Join(dir, "receipt.json"))
			require.NoError(t, err)
			require.Equal(t, original, b)
			require.Zero(t, d.calls)
		})
	}
}

func TestUnknownRemovedResourcesRequireArchiveRatherThanEmptyVolumeRetry(t *testing.T) {
	d := &recoveryFixture{audit: RecoveryAudit{Fingerprint: "removed", Removed: true, Empty: true}}
	c, p, original := retainedUnknown(t, d)
	preview, e := c.PrepareRecovery(context.Background(), p.ID, "retry", "")
	require.NoError(t, e)
	require.NotEmpty(t, preview.Blocks)
	_, e = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.ErrorIs(t, e, ErrInvalid)
	dir, _ := c.directory(p.ID)
	b, e := os.ReadFile(filepath.Join(dir, "receipt.json"))
	require.NoError(t, e)
	require.Equal(t, original, b)
}

func TestExpiredPreviewAndForeignGuardCannotAdoptOriginal(t *testing.T) {
	d := &recoveryFixture{audit: RecoveryAudit{Fingerprint: "empty", Empty: true}}
	c, p, original := retainedUnknown(t, d)
	preview, e := c.PrepareRecovery(context.Background(), p.ID, "retry", "")
	require.NoError(t, e)
	file, e := c.previewPath(p.ID, preview.ID)
	require.NoError(t, e)
	b, e := os.ReadFile(file)
	require.NoError(t, e)
	var seal sealedRecovery
	require.NoError(t, json.Unmarshal(b, &seal))
	seal.Preview.ExpiresAt = time.Now().Add(-time.Second)
	require.NoError(t, privatefile.Write(file, mustJSON(seal)))
	_, e = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.ErrorIs(t, e, ErrExpired)
	preview, e = c.PrepareRecovery(context.Background(), p.ID, "retry", "")
	require.NoError(t, e)
	require.NoError(t, privatefile.Write(filepath.Join(c.root, p.InstanceID, "active.json"), mustJSON("another-operation")))
	_, e = c.ConfirmRecovery(context.Background(), p.ID, preview.ID)
	require.ErrorIs(t, e, ErrUnresolved)
	dir, _ := c.directory(p.ID)
	b, e = os.ReadFile(filepath.Join(dir, "receipt.json"))
	require.NoError(t, e)
	require.Equal(t, original, b)
	require.Zero(t, d.calls)
}

func TestRestartAfterIntentRepairsOnlyGuardAndNeverDispatches(t *testing.T) {
	d := &testDriver{revision: "before"}
	c := New(t.TempDir(), d)
	p, e := c.Plan(context.Background(), Request{Mode: "existing"})
	require.NoError(t, e)
	r, e := c.read(p.ID)
	require.NoError(t, e)
	r.Attempts[0].Status = "running"
	require.NoError(t, c.save(&r))
	_, e = New(c.root, d).Reconcile(context.Background(), p.ID)
	require.ErrorIs(t, e, ErrUnresolved)
	got, e := c.Read(p.ID)
	require.NoError(t, e)
	require.Equal(t, "unknown", got.Status)
	guard, e := c.active(p.InstanceID)
	require.NoError(t, e)
	require.Equal(t, p.ID, guard)
	require.Zero(t, d.calls)
	_, e = c.Apply(context.Background(), p.ID)
	require.ErrorIs(t, e, ErrUnresolved)
}
