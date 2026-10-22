package prompt

import (
	"github.com/sirupsen/logrus"
	"io"
)

func diagnosticLogger(log *logrus.Entry) *logrus.Entry {
	if log != nil {
		return log
	}
	disabled := logrus.New()
	disabled.SetOutput(io.Discard)
	return logrus.NewEntry(disabled)
}
