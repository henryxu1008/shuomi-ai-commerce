package appenv

import (
	"testing"

	logging "task-processor/internal/platform/logging"
)

func TestCompositionFactoryResolvesCurrentManagerOnEveryCall(t *testing.T) {
	logs := ComponentLoggers()
	logging.InitGlobalLogger(&logging.LogConfig{Level: "warn", Format: "json"})
	first := logs("fixture")
	logging.InitGlobalLogger(&logging.LogConfig{Level: "debug", Format: "text"})
	second := logs("fixture")
	if first.Logger == second.Logger || second.Logger != logging.GetGlobalLogManager().GetRawLogger() || second.Data["component"] != "fixture" {
		t.Fatal("composition callback captured a stale logging manager")
	}
}
