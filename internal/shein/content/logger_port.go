package content

import (
	"github.com/sirupsen/logrus"
	"io"
)

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
