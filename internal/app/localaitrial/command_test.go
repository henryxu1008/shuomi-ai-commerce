package localaitrial

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/orgresource"
)

const testConfigJSON = `{"organizationId":"local-org","operatorId":"trusted-operator","approvalEvidenceId":"human-decision","grantEnabled":false,"database":{"host":"127.0.0.1","port":5432,"user":"commercial_owner_runtime","password":"synthetic-db-secret","database":"resources"}}`

func privateTrialConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "trial.json")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	if runtime.GOOS == "windows" {
		identity, err := user.Current()
		require.NoError(t, err)
		output, err := exec.Command("icacls", path, "/inheritance:r", "/grant:r", "*"+identity.Uid+":F").CombinedOutput()
		require.NoError(t, err, "%s", output)
	}
	return path
}

func TestTrialConfigRejectsUnknownDuplicateTrailingAndOversizedInput(t *testing.T) {
	for _, data := range []string{
		strings.Replace(testConfigJSON, `"grantEnabled":false`, `"quantity":99999`, 1),
		strings.Replace(testConfigJSON, `"grantEnabled":false`, `"grantEnabled":false,"grantEnabled":true`, 1),
		testConfigJSON + ` {}`, strings.Repeat(" ", 16<<10) + testConfigJSON,
	} {
		_, err := LoadConfig(context.Background(), privateTrialConfig(t, data))
		require.Error(t, err)
		require.NotContains(t, err.Error(), "synthetic-db-secret")
	}
	_, err := LoadConfig(context.Background(), "relative.json")
	require.Error(t, err)
}

type trialCommandExecutor struct {
	reads, writes int
	err           error
}

func (e *trialCommandExecutor) ReplayTrialAIGrant(context.Context, orgresource.TrialAIGrantExecution) (orgresource.TrialAIGrantResult, bool, error) {
	e.reads++
	return orgresource.TrialAIGrantResult{}, false, e.err
}
func (e *trialCommandExecutor) ExecuteTrialAIGrant(context.Context, orgresource.TrialAIGrantExecution) (orgresource.TrialAIGrantResult, error) {
	e.writes++
	return orgresource.TrialAIGrantResult{}, e.err
}

func TestTrialCommandInspectCannotGrantAndDisabledGrantStopsBeforeDatabaseOpen(t *testing.T) {
	cfg, err := LoadConfig(context.Background(), privateTrialConfig(t, testConfigJSON))
	require.NoError(t, err)
	executor := &trialCommandExecutor{}
	result, err := execute(context.Background(), cfg, "original-operation", "inspect", executor)
	require.NoError(t, err)
	require.False(t, result.Found)
	require.Nil(t, result.Receipt)
	require.Equal(t, 1, executor.reads)
	require.Zero(t, executor.writes)
	_, err = Run(context.Background(), cfg, "original-operation", "grant")
	require.ErrorIs(t, err, orgresource.ErrForbidden)
	cfg.Database.User = "business_cluster_admin"
	_, err = Run(context.Background(), cfg, "original-operation", "inspect")
	require.ErrorIs(t, err, orgresource.ErrInvalidInput)
}

func TestTrialCommandRejectsCrossOrganizationAndSanitizesRepositoryFailure(t *testing.T) {
	cfg, err := LoadConfig(context.Background(), privateTrialConfig(t, testConfigJSON))
	require.NoError(t, err)
	_, err = (operatorAuthorizer{cfg}).AuthorizeTrialAIGrant(context.Background(), orgresource.Principal{ID: cfg.OperatorID, Kind: orgresource.PrincipalTrustedTrialOperator}, "another-org")
	require.ErrorIs(t, err, orgresource.ErrForbidden)
	executor := &trialCommandExecutor{err: errors.New("password=synthetic-db-secret")}
	_, err = execute(context.Background(), cfg, "original-operation", "inspect", executor)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "synthetic-db-secret")
	cfg.GrantEnabled = true
	_, err = execute(context.Background(), cfg, "original-operation", "grant", executor)
	require.ErrorIs(t, err, ErrOutcomeUnconfirmed)
	require.NotContains(t, err.Error(), "synthetic-db-secret")
}
