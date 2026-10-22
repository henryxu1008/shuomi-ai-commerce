package sheinloginworker

import (
	"io"
	"os"
	"time"
)

const defaultShutdownTimeout = 30 * time.Second

type Options struct {
	ConfigPath                                                string
	Port                                                      int
	Action                                                    string
	OrganizationID, StoreID, TokenFile, RequestKey, AttemptID string
	ForceLogin                                                bool
	ExpectedGeneration                                        int64
	Input                                                     io.Reader
	Output                                                    io.Writer
	ShutdownSignal                                            chan os.Signal
	ShutdownTimeout                                           time.Duration
}

func (o Options) shutdownTimeout() time.Duration {
	if o.ShutdownTimeout > 0 {
		return o.ShutdownTimeout
	}
	return defaultShutdownTimeout
}
