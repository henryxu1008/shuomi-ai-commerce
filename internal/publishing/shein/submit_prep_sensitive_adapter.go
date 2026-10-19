package shein

import (
	"task-processor/internal/shein/content"
)

type sheinSensitiveWordSanitizer interface {
	SanitizeDisplayTextWithContext(ctx *content.CleanupContext, text string) string
}

func newSheinSensitiveWordSanitizer() sheinSensitiveWordSanitizer {
	return content.NewSensitiveWordServiceInMemory()
}
