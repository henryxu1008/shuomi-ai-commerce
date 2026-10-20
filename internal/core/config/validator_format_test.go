package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValidationErrors_GroupsByModule(t *testing.T) {
	lines := formatValidationErrors([]error{
		&ValidationError{Field: "openai.apiKey", Message: "missing", Hint: "set TASK_PROCESSOR_OPENAI_API_KEY"},
		&ValidationError{Field: "browser.poolSize", Message: "missing"},
		&ValidationError{Field: "openai.model", Message: "missing"},
	})

	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "[OpenAI]")
	assert.Contains(t, joined, "openai.apiKey: missing")
	assert.Contains(t, joined, "hint: set TASK_PROCESSOR_OPENAI_API_KEY")
	assert.Contains(t, joined, "openai.model: missing")
	assert.Contains(t, joined, "[Browser]")
	assert.Contains(t, joined, "browser.poolSize: missing")
	assert.Less(t, strings.Index(joined, "[OpenAI]"), strings.Index(joined, "[Browser]"))
}

func TestFormatValidationErrors_FallbacksToGeneral(t *testing.T) {
	lines := formatValidationErrors([]error{errors.New("plain failure")})
	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "[General]")
	assert.Contains(t, joined, "plain failure")
}

func TestValidateConfigWithError_UsesGroupedFormat(t *testing.T) {
	cfg := &Config{
		Browser: BrowserConfig{
			Enabled:  true,
			PoolSize: 0,
		},
	}

	err := ValidateConfigWithError(cfg)
	if assert.Error(t, err) {
		assert.NotContains(t, err.Error(), "[Management]")
		assert.Contains(t, err.Error(), "[Browser]")
	}
}
