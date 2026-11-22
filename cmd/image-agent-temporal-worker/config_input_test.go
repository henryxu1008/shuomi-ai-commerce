package main

import (
	"context"
	appruntime "github.com/henryxu1008/shuomi-ai-commerce/internal/app/runtime"
	imageagenttemporal "github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/temporal"
	"github.com/sirupsen/logrus"
	"strings"
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
