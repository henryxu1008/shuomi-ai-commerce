//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// Reuse the existing application/owner fixture with a task-owned database and
// loopback controlled text provider; no real provider or trial data is used.
func TestChatStoreQueryIsolatedOwners(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("issue398_owner"), tcpostgres.WithUsername("issue398_owner"), tcpostgres.WithPassword("isolated-store-query"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	t.Setenv("ISSUE398_TEST_DSN", fmt.Sprintf("host=127.0.0.1 port=%s user=issue398_owner password=isolated-store-query sslmode=disable", port.Port()))
	t.Setenv("CODEX_STORE_QUERY_FIXTURE", "1")
	t.Run("governed", func(t *testing.T) { testProductAgentOwners(t, "content flow chat") })
	t.Run("unavailable", func(t *testing.T) { testConversationOnlyScopedCRUDAndNoExecution(t, nil) })
	t.Run("selected product context", func(t *testing.T) {
		t.Setenv("CODEX_PRODUCT_CONTEXT_FIXTURE", "1")
		t.Run("acquisition", func(t *testing.T) { testProductAgentOwners(t, "content flow chat") })
		t.Run("own product", func(t *testing.T) { testProductAgentOwners(t, "own content chat") })
	})
}
