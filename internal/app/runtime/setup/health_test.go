package setup

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFreshCompletionRequiresOriginalWholeComposition(t *testing.T) {
	app := container{}
	app.Config.Labels = map[string]string{"com.docker.compose.service": "current-application"}
	app.State.Running = true
	init := container{}
	init.Config.Labels = map[string]string{"com.docker.compose.service": "schema-init"}
	init.State.Status = "exited"
	init.State.ExitCode = 1
	required := map[string]string{"current-application": "running", "schema-init": "completed"}
	require.False(t, requiredHealthy([]container{app, init}, required))
	init.State.ExitCode = 0
	require.True(t, requiredHealthy([]container{app, init}, required))
	require.False(t, requiredHealthy([]container{app}, required))
	require.False(t, requiredHealthy(nil, nil))
}
