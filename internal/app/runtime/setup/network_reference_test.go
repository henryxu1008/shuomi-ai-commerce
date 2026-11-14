package setup

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestRunningNetworkReferenceUsesExactSameProjectContainer(t *testing.T) {
	doc := map[string]any{"services": map[string]any{"current-application": map[string]any{"network_mode": "service:runtime-network"}}}
	parent := container{ID: "exact-network-id"}
	parent.Config.Labels = map[string]string{"com.docker.compose.service": "runtime-network"}
	parent.State.Running = true
	app := container{}
	app.HostConfig.NetworkMode = "container:exact-network-id"
	require.True(t, resolveNetworkReference(doc, "current-application", app, []container{parent}))
	require.Equal(t, "container:exact-network-id", doc["services"].(map[string]any)["current-application"].(map[string]any)["network_mode"])
	doc["services"].(map[string]any)["current-application"].(map[string]any)["network_mode"] = "service:runtime-network"
	app.HostConfig.NetworkMode = "container:foreign-id"
	require.False(t, resolveNetworkReference(doc, "current-application", app, []container{parent}))
}
