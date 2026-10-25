package tests

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMaintainedConfigFilesDoNotAdvertiseRetiredGlobalInputs(t *testing.T) {
	for _, name := range []string{"config-dev.yaml", "config-test.yaml", "config-prod.yaml"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "config", name))
			if err != nil {
				t.Fatal(err)
			}
			var cfg map[string]any
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				t.Fatal(err)
			}
			if len(cfg) == 0 {
				t.Fatal("maintained configuration must not be empty")
			}
			for _, root := range []string{"logging", "redis", "prompts"} {
				if _, exists := cfg[root]; exists {
					t.Errorf("retired global configuration %s must be absent", root)
				}
			}
		})
	}
}
