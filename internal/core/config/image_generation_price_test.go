package config

import "testing"

func TestImageGenerationAdmissionRequiresExplicitUnitAndCeiling(t *testing.T) {
	for _, c := range []ImageAgentGenerationConfig{{}, {Enabled: true}, {Enabled: true, UnitVersion: "whole-point", MaximumMicropoints: 12}, {Enabled: true, UnitVersion: "ai-micropoint-v1", MaximumMicropoints: -1}} {
		if c.Configured() {
			t.Fatalf("invalid admission: %+v", c)
		}
	}
	if !(ImageAgentGenerationConfig{Enabled: true, UnitVersion: "ai-micropoint-v1", MaximumMicropoints: 12}).Configured() {
		t.Fatal("explicit micro-unit ceiling rejected")
	}
}
