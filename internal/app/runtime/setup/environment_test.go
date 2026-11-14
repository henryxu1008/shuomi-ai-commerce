package setup

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestDockerChildEnvironmentCannotActivateProfilesOrOverrideInstallInput(t *testing.T) {
	t.Setenv("ACCOUNT_APPLICATION_PORT", "35444")
	t.Setenv("COMPOSE_PROFILES", "acceptance")
	t.Setenv("DOCKER_HOST", "tcp://foreign:2375")
	t.Setenv("UNRELATED_PROVIDER_SECRET", "private-test-value")
	env := strings.Join(dockerEnvironment(), "\n")
	for _, key := range []string{"ACCOUNT_APPLICATION_PORT=", "COMPOSE_PROFILES=", "DOCKER_HOST=", "UNRELATED_PROVIDER_SECRET="} {
		require.NotContains(t, env, key)
	}
	require.Contains(t, strings.ToUpper(env), "PATH=")
}
