package sheinlogin

import (
	"io"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestAutomationUsesRuntimeLoggerWithoutPackageInitializationCapture(t *testing.T) {
	current := logrus.New()
	current.SetOutput(io.Discard)
	automation := NewPlaywrightAutomation(func(component string) *logrus.Entry {
		return current.WithFields(logrus.Fields{"component": component, "request_id": "synthetic-request"})
	})
	for i := 0; i < 2; i++ {
		entry := automation.componentLogger("sheinlogin_service")
		if entry.Logger != current || entry.Data["request_id"] != "synthetic-request" || entry.Data["component"] != "sheinlogin_service" {
			t.Fatal("automation captured a different logger or lost runtime context")
		}
		current = logrus.New()
		current.SetOutput(io.Discard)
	}
}
