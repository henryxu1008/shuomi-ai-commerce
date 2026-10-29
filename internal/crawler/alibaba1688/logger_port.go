package alibaba1688

import (
	"github.com/sirupsen/logrus"
	"io"
)

// loggerDependency carries only the current runtime's diagnostic dependency.
// Its zero value disables diagnostics without consulting global state.
type loggerDependency struct{ logFor func(string) *logrus.Entry }

func (d loggerDependency) componentLogger(component string) *logrus.Entry {
	return componentLogger(d.logFor, component)
}
func componentLogger(logs func(string) *logrus.Entry, component string) *logrus.Entry {
	if logs != nil {
		if entry := logs(component); entry != nil {
			return entry
		}
	}
	disabled := logrus.New()
	disabled.SetOutput(io.Discard)
	return disabled.WithField("component", component)
}
