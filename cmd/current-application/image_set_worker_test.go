package main

import (
	"os"
	"path/filepath"
	"reflect"
	"task-processor/internal/app/runtime/currentapplication"
	"task-processor/internal/core/config"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFullImageWorkerConfigDoesNotRequireAnUnusedGlobalOpenAIKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "worker.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`database:
  host: 127.0.0.1
  port: 5432
  database: image
  user: image_agent_worker_runtime
  password: controlled-private-fixture
  max_connections: 4
  max_idle_connections: 1
`), 0600))
	expected := currentapplication.DatabaseConfig{Host: "127.0.0.1", Port: 5432, Database: "image"}
	cfg, err := loadImageSetWorkerConfig(path, expected)
	require.NoError(t, err, "full mode consumes its scoped organization credential, never a global OpenAI key")
	_, exposed := reflect.TypeOf(*cfg).FieldByName("OpenAI")
	require.False(t, exposed, "retired global OpenAI configuration must be absent")
	expected.Database = "another-owner"
	_, err = loadImageSetWorkerConfig(path, expected)
	require.Error(t, err)
}

func TestFullImageWorkerCannotUseTheHTTPRoleOrAnotherOwner(t *testing.T) {
	expected := currentapplication.DatabaseConfig{Host: "127.0.0.1", Port: 5432, Database: "image"}
	valid := config.DatabaseConfig{Host: expected.Host, Port: expected.Port, Database: expected.Database, User: "image_agent_worker_runtime", MaxConnections: 4}
	cfg := &config.Config{Database: &valid}
	require.NoError(t, validateImageSetWorkerDatabase(cfg, expected))
	for _, mutate := range []func(*config.DatabaseConfig){func(d *config.DatabaseConfig) { d.User = "image_agent_runtime" }, func(d *config.DatabaseConfig) { d.Database = "other" }, func(d *config.DatabaseConfig) { d.MaxConnections = 20 }} {
		changed := valid
		mutate(&changed)
		cfg.Database = &changed
		require.Error(t, validateImageSetWorkerDatabase(cfg, expected))
	}
}
