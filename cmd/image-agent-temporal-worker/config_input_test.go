package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"strings"
	appruntime "task-processor/internal/app/runtime"
	imageagenttemporal "task-processor/internal/imageagent/temporal"
	"testing"
)

func TestWorkerRequiresConfigurationBeforeResolving(t *testing.T) {
	for _, path := range []string{"", " \t"} {
		called := false
		err := runConfigured(context.Background(), path, func(string) dependencyResolver {
			called = true
			return func(_ imageagenttemporal.WorkerWireMode) (resolvedDependencies, error) { panic("must not resolve") }
		}, nil, appruntime.ImageAgentTemporalWorkerOptions{}, logrus.New())
		if called || err == nil || !strings.Contains(err.Error(), "-config is required") {
			t.Fatalf("called=%v err=%v", called, err)
		}
	}
}
