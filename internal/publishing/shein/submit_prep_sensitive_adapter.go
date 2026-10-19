package shein

import (
	"context"

	"task-processor/internal/shein/content"
	"task-processor/internal/shein/submitprep"
)

type sheinSensitiveWordSanitizer interface {
	SanitizeDisplayTextWithContext(ctx *content.CleanupContext, text string) string
}

func newSheinSensitiveWordSanitizer(ctx context.Context) sheinSensitiveWordSanitizer {
	return submitprep.NewSensitiveWordServiceForContext(ctx)
}
