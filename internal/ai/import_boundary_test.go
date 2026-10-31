package ai_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"task-processor/internal/ai"
	"task-processor/internal/integration/bailian"
)

var (
	_ ai.RouteBoundImageEditor = (*bailian.Client)(nil)
)

func TestProviderAdaptersDoNotImportOpenAIForSharedContracts(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Dir(currentFile)
	for _, relative := range []string{
		filepath.Join("..", "integration", "grsai", "client.go"),
	} {
		path := filepath.Join(root, relative)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if strings.Contains(string(content), "task-processor/internal/integration/openai") {
			t.Fatalf("provider adapter %s imports the OpenAI implementation package", path)
		}
	}
}
