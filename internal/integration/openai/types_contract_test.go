package openai

import (
	"testing"

	"task-processor/internal/ai"
)

func TestOpenAIClientImplementsProviderNeutralContracts(t *testing.T) {
	var _ ai.ChatCompleter = (*Client)(nil)
	var _ ai.ImageGenerator = (*Client)(nil)
}
