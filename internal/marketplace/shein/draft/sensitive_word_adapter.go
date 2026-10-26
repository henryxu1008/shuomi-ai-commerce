package draft

import (
	"github.com/sirupsen/logrus"

	"task-processor/internal/shein/content"
)

type sheinSensitiveWordSanitizer interface {
	SanitizeDisplayTextWithContext(ctx *content.CleanupContext, text string) string
}

func newSheinSensitiveWordSanitizer(logs func(string) *logrus.Entry) sheinSensitiveWordSanitizer {
	return content.NewSensitiveWordServiceInMemory(logs)
}
